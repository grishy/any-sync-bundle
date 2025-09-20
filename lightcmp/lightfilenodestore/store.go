package lightfilenodestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anyproto/any-sync-filenode/store/s3store"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/commonfile/fileblockstore"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const CName = fileblockstore.CName

const (

	// GC configuration.
	defaultGCInterval    = 29 * time.Hour // Prime number to avoid alignment
	defaultMaxGCDuration = time.Minute
	defaultGCThreshold   = 0.5 // Reclaim files with >= 50% garbage

	// Number of concurrent block retrievals in GetMany, took from S3 implementation.
	getManyWorkers = 4

	// Prefix for index keys to separate them from block keys.
	indexKeyPrefix = "idx:"
)

var (
	// Type assertion.
	_ s3store.S3Store = (*LightFileNodeStore)(nil)

	log = logger.NewNamed(CName)
)

// IndexLog represents a single WAL entry for index changes.
type IndexLog struct {
	Idx  uint64
	Data []byte
}

type storeConfig struct {
	storePath     string
	gcInterval    time.Duration
	gcThreshold   float64
	maxGCDuration time.Duration
}

type LightFileNodeStore struct {
	cfg      storeConfig
	db       *badger.DB
	gcCancel context.CancelFunc
}

func New(storePath string) *LightFileNodeStore {
	return &LightFileNodeStore{
		cfg: storeConfig{
			storePath:     storePath,
			gcInterval:    defaultGCInterval,
			gcThreshold:   defaultGCThreshold,
			maxGCDuration: defaultMaxGCDuration,
		},
	}
}

func (s *LightFileNodeStore) Init(_ *app.App) error {
	log.Info("initializing light filenode store")
	return nil
}

func (s *LightFileNodeStore) Name() string {
	return CName
}

func (s *LightFileNodeStore) Run(ctx context.Context) error {
	opts := badger.DefaultOptions(s.cfg.storePath).
		WithLogger(badgerLogger{}).
		WithCompression(options.None).
		WithZSTDCompressionLevel(0)

	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to open badger db: %w", err)
	}

	s.db = db

	// Create a cancellable context for GC
	gcCtx, cancel := context.WithCancel(ctx)
	s.gcCancel = cancel
	go s.runGC(gcCtx)

	return nil
}

func (s *LightFileNodeStore) Close(_ context.Context) error {
	// Cancel the GC goroutine
	if s.gcCancel != nil {
		s.gcCancel()
	}
	return s.db.Close()
}

func (s *LightFileNodeStore) Get(_ context.Context, k cid.Cid) (blocks.Block, error) {
	start := time.Now()
	var val []byte

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(k.String()))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return fileblockstore.ErrCIDNotFound
			}
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return nil, err
	}

	log.Debug("badger get",
		zap.Duration("total", time.Since(start)),
		zap.Int("kbytes", len(val)/1024),
		zap.String("key", k.String()),
	)

	return blocks.NewBlockWithCid(val, k)
}

func (s *LightFileNodeStore) GetMany(ctx context.Context, ks []cid.Cid) <-chan blocks.Block {
	res := make(chan blocks.Block)

	go func() {
		defer close(res)

		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(getManyWorkers)

		for _, k := range ks {
			g.Go(func() error {
				var val []byte
				err := s.db.View(func(txn *badger.Txn) error {
					item, err := txn.Get([]byte(k.String()))
					if err != nil {
						if !errors.Is(err, badger.ErrKeyNotFound) {
							log.Warn("failed to get block", zap.Error(err), zap.String("key", k.String()))
						}
						return err
					}

					val, err = item.ValueCopy(nil)
					if err != nil {
						log.Warn("failed to copy block value", zap.Error(err), zap.String("key", k.String()))
						return err
					}
					return nil
				})
				if err != nil {
					return nil // Already logged above
				}

				bl, err := blocks.NewBlockWithCid(val, k)
				if err != nil {
					log.Warn("failed to create block", zap.Error(err), zap.String("key", k.String()))
					return nil
				}

				select {
				case res <- bl:
				case <-gctx.Done():
				}

				return nil
			})
		}

		_ = g.Wait() // Ignore error since individual errors are already logged
	}()

	return res
}

func (s *LightFileNodeStore) Add(_ context.Context, bs []blocks.Block) error {
	start := time.Now()
	wb := s.db.NewWriteBatch()
	defer wb.Cancel()

	// Usually one block, so no concurrent writes needed
	var dataLen int
	keys := make([]string, 0, 1)

	for _, bl := range bs {
		data := bl.RawData()
		dataLen += len(data)
		key := bl.Cid().String()
		keys = append(keys, key)

		if err := wb.Set([]byte(key), data); err != nil {
			return fmt.Errorf("failed to set key %s: %w", key, err)
		}
	}

	if err := wb.Flush(); err != nil {
		return fmt.Errorf("failed to flush write batch: %w", err)
	}

	log.Debug("badger put",
		zap.Duration("total", time.Since(start)),
		zap.Int("blocks", len(bs)),
		zap.Int("kbytes", dataLen/1024),
		zap.Strings("keys", keys),
	)

	return nil
}

func (s *LightFileNodeStore) Delete(_ context.Context, c cid.Cid) error {
	// TODO: Create an issue that no Delete call after clean up of Bin in Anytype.
	// Check before, that here is no deferred call to Delete

	start := time.Now()
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(c.String()))
	})

	log.Debug("badger delete",
		zap.Duration("total", time.Since(start)),
		zap.String("key", c.String()),
	)

	return err
}

func (s *LightFileNodeStore) DeleteMany(_ context.Context, toDelete []cid.Cid) error {
	start := time.Now()
	wb := s.db.NewWriteBatch()
	defer wb.Cancel()

	var keys []string
	for _, c := range toDelete {
		key := c.String()
		keys = append(keys, key)
		if err := wb.Delete([]byte(key)); err != nil {
			log.Warn("can't delete cid", zap.Error(err), zap.String("key", key))
		}
	}

	err := wb.Flush()

	log.Debug("badger delete many",
		zap.Duration("total", time.Since(start)),
		zap.Int("count", len(toDelete)),
		zap.Strings("keys", keys),
	)

	if err != nil {
		return fmt.Errorf("failed to flush write batch: %w", err)
	}

	return nil
}

func (s *LightFileNodeStore) IndexGet(_ context.Context, key string) (value []byte, err error) {
	indexKey := indexKeyPrefix + key

	err = s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(indexKey))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				log.Warn("index key not found", zap.String("key", indexKey))
				return nil
			}

			return fmt.Errorf("failed to get index key: %w", err)
		}

		value, err = item.ValueCopy(nil)

		log.Debug("badger index get", zap.String("key", indexKey))
		if err != nil {
			return fmt.Errorf("failed to copy index value: %w", err)
		}

		return nil
	})

	log.Debug("badger index get result", zap.String("key", indexKey))
	return value, err
}

// IndexPut stores a value in the index with the given key.
func (s *LightFileNodeStore) IndexPut(_ context.Context, key string, value []byte) error {
	indexKey := indexKeyPrefix + key

	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(indexKey), value)
	})

	log.Debug("badger index put", zap.String("key", indexKey))
	return err
}

// IndexDelete deletes a value from the index with the given key.
func (s *LightFileNodeStore) IndexDelete(_ context.Context, key string) error {
	indexKey := indexKeyPrefix + key

	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(indexKey))
	})

	log.Debug("badger index delete", zap.String("key", indexKey))
	return err
}

func (s *LightFileNodeStore) runGC(ctx context.Context) {
	log.Info("starting badger garbage collection routine")
	ticker := time.NewTicker(s.cfg.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping badger garbage collection routine")
			return
		case <-ticker.C:
			start := time.Now()
			gcCount := 0

			// Run GC until either we hit time limit or no more files need GC
			for time.Since(start) < s.cfg.maxGCDuration {
				if err := s.db.RunValueLogGC(s.cfg.gcThreshold); err != nil {
					if errors.Is(err, badger.ErrNoRewrite) {
						break // No more files need GC
					}
					log.Warn("badger gc failed", zap.Error(err))
					break
				}
				gcCount++
			}

			log.Info("badger garbage collection completed",
				zap.Duration("duration", time.Since(start)),
				zap.Int("filesGCed", gcCount))
		}
	}
}

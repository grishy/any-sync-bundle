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
)

const CName = fileblockstore.CName

const (
	// GC configuration.
	defaultGCInterval    = 29 * time.Hour // Prime number to avoid alignment
	defaultMaxGCDuration = time.Minute
	defaultGCThreshold   = 0.5 // Reclaim files with >= 50% garbage

	// Key prefixes.
	blockKeyPrefix = "f:b:"
	indexKeyPrefix = "f:i:"
)

var (
	// Type assertion.
	_ s3store.S3Store = (*LightFileNodeStore)(nil)

	log = logger.NewNamed(CName)
)

func blockKeyString(k cid.Cid) string { return blockKeyPrefix + k.String() }

func blockKeyBytes(k cid.Cid) []byte { return []byte(blockKeyString(k)) }

func indexKeyString(key string) string { return indexKeyPrefix + key }

func indexKeyBytes(key string) []byte { return []byte(indexKeyString(key)) }

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
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

func (s *LightFileNodeStore) Get(_ context.Context, k cid.Cid) (blocks.Block, error) {
	st := time.Now()
	bl, err := s.loadBlock(k)
	if err != nil {
		return nil, err
	}

	log.Debug("badger get",
		zap.Duration("total", time.Since(st)),
		zap.Int("kbytes", len(bl.RawData())/1024),
		zap.String("key", k.String()),
	)

	return bl, nil
}

func (s *LightFileNodeStore) GetMany(ctx context.Context, ks []cid.Cid) <-chan blocks.Block {
	st := time.Now()
	res := make(chan blocks.Block)

	go func() {
		defer close(res)

		// Upstream S3 store fans out requests across workers. Local Badger access is
		// already low latency, so we iterate sequentially and surface a single summary log.
		delivered := 0
		for _, k := range ks {
			select {
			case <-ctx.Done():
				log.Debug("badger get many",
					zap.Duration("total", time.Since(st)),
					zap.Int("requested", len(ks)),
					zap.Int("delivered", delivered),
					zap.Error(ctx.Err()),
				)
				return
			default:
			}

			bl, err := s.loadBlock(k)
			if err != nil {
				if errors.Is(err, fileblockstore.ErrCIDNotFound) {
					log.Debug("block not found", zap.String("cid", k.String()))
				} else {
					log.Warn("failed to load block", zap.Error(err), zap.String("cid", k.String()))
				}
				continue
			}

			select {
			case res <- bl:
				delivered++
			case <-ctx.Done():
				log.Debug("badger get many",
					zap.Duration("total", time.Since(st)),
					zap.Int("requested", len(ks)),
					zap.Int("delivered", delivered),
					zap.Error(ctx.Err()),
				)
				return
			}
		}

		log.Debug("badger get many",
			zap.Duration("total", time.Since(st)),
			zap.Int("requested", len(ks)),
			zap.Int("delivered", delivered),
		)
	}()

	return res
}

// loadBlock retrieves a single block from the database.
func (s *LightFileNodeStore) loadBlock(k cid.Cid) (blocks.Block, error) {
	var val []byte
	key := blockKeyBytes(k)
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return fileblockstore.ErrCIDNotFound
			}
			return err
		}

		val, err = item.ValueCopy(nil)
		if err != nil {
			log.Warn("failed to copy block value", zap.Error(err), zap.String("cid", k.String()))
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	bl, err := blocks.NewBlockWithCid(val, k)
	if err != nil {
		log.Warn("failed to create block", zap.Error(err), zap.String("cid", k.String()))
		return nil, err
	}

	return bl, nil
}

func (s *LightFileNodeStore) Add(_ context.Context, bs []blocks.Block) error {
	st := time.Now()
	wb := s.db.NewWriteBatch()
	defer wb.Cancel()

	// Usually one block (what I saw), so no concurrent writes needed
	var dataLen int

	for _, bl := range bs {
		data := bl.RawData()
		dataLen += len(data)

		key := blockKeyBytes(bl.Cid())

		if err := wb.Set(key, data); err != nil {
			return fmt.Errorf("failed to set key %s: %w", bl.Cid().String(), err)
		}
	}

	err := wb.Flush()

	log.Debug("badger put",
		zap.Duration("total", time.Since(st)),
		zap.Int("blocks", len(bs)),
		zap.Int("kbytes", dataLen/1024),
	)

	return err
}

func (s *LightFileNodeStore) Delete(_ context.Context, c cid.Cid) error {
	// TODO: Create an issue that no Delete call after clean up of Bin in Anytype.
	// Check before, that here is no deferred call to Delete

	st := time.Now()
	err := s.db.Update(func(txn *badger.Txn) error {
		deleteErr := txn.Delete(blockKeyBytes(c))
		if errors.Is(deleteErr, badger.ErrKeyNotFound) {
			return nil
		}
		return deleteErr
	})

	log.Debug("badger delete",
		zap.Error(err),
		zap.Duration("total", time.Since(st)),
		zap.String("cid", c.String()),
	)

	return err
}

func (s *LightFileNodeStore) DeleteMany(_ context.Context, toDelete []cid.Cid) error {
	st := time.Now()
	wb := s.db.NewWriteBatch()
	defer wb.Cancel()

	// S3 implementation deletes sequentially and logs each failure. We keep the behavior but
	// rely on Badger's batch API for efficiency and aggregate logging.
	for _, c := range toDelete {
		if err := wb.Delete(blockKeyBytes(c)); err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				continue
			}
			log.Warn("can't delete cid", zap.Error(err), zap.String("cid", c.String()))
		}
	}

	err := wb.Flush()

	log.Debug("badger delete many",
		zap.Error(err),
		zap.Duration("total", time.Since(st)),
		zap.Int("count", len(toDelete)),
	)

	// Original implementation newer return an error
	return nil
}

func (s *LightFileNodeStore) IndexGet(_ context.Context, key string) ([]byte, error) {
	st := time.Now()
	var value []byte

	// Unlike S3 store's remote round-trip, Badger lookups are local; we still mirror the
	// not-found semantics (return nil without error).
	err := s.db.View(func(txn *badger.Txn) error {
		item, getErr := txn.Get(indexKeyBytes(key))
		if getErr != nil {
			if errors.Is(getErr, badger.ErrKeyNotFound) {
				return nil
			}

			return fmt.Errorf("failed to get index key: %w", getErr)
		}

		var copyErr error
		value, copyErr = item.ValueCopy(nil)

		if copyErr != nil {
			return fmt.Errorf("failed to copy index value: %w", copyErr)
		}

		return nil
	})

	log.Debug("badger index get",
		zap.Error(err),
		zap.Duration("total", time.Since(st)),
		zap.String("key", key),
	)

	return value, err
}

func (s *LightFileNodeStore) IndexPut(_ context.Context, key string, value []byte) error {
	st := time.Now()
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(indexKeyBytes(key), value)
	})

	log.Debug("badger index put",
		zap.Error(err),
		zap.Duration("total", time.Since(st)),
		zap.String("key", key),
	)

	return err
}

func (s *LightFileNodeStore) IndexDelete(_ context.Context, key string) error {
	st := time.Now()
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(indexKeyBytes(key))
	})

	log.Debug("badger index delete",
		zap.Error(err),
		zap.Duration("total", time.Since(st)),
		zap.String("key", key),
	)

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
			gcCount, duration, err := s.gcOnce()
			if err != nil {
				log.Warn("badger gc failed", zap.Error(err))
			}

			log.Info("badger garbage collection completed",
				zap.Duration("duration", duration),
				zap.Int("filesGCed", gcCount))
		}
	}
}

// gcOnce performs a single garbage collection iteration and returns the number of value log files reclaimed.
func (s *LightFileNodeStore) gcOnce() (int, time.Duration, error) {
	start := time.Now()
	gcCount := 0

	for time.Since(start) < s.cfg.maxGCDuration {
		if err := s.db.RunValueLogGC(s.cfg.gcThreshold); err != nil {
			if errors.Is(err, badger.ErrNoRewrite) {
				return gcCount, time.Since(start), nil
			}
			return gcCount, time.Since(start), err
		}
		gcCount++
	}

	return gcCount, time.Since(start), nil
}

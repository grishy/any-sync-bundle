package storeBadger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anyproto/any-sync-filenode/store/s3store"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/commonfile/fileblockstore"
	"golang.org/x/sync/errgroup"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"
)

const (
	CName = fileblockstore.CName

	// Number of concurrent block retrievals in GetMany, took from S3 implementation
	getManyWorkers = 4

	// Prefix for index keys to separate them from block keys
	indexKeyPrefix = "idx:"
)

var (
	log = logger.NewNamed(CName)
	// Just to make sure that Badger implements the interface
	_ s3store.S3Store = (*Badger)(nil)
)

// Badger implements a block store using BadgerDB.
// It provides persistent storage of content-addressed blocks with indexing support.
// Was thinking about sqlite, but it may hard to backup and restore one big file.
// A few drawbacks - no concurrent writes, not sure that needed.
type Badger struct {
	path string
	db   *badger.DB
}

func New(path string) *Badger {
	return &Badger{path: path}
}

func (b *Badger) Get(ctx context.Context, k cid.Cid) (blocks.Block, error) {
	start := time.Now()
	var val []byte

	err := b.db.View(func(txn *badger.Txn) error {
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

func (b *Badger) GetMany(ctx context.Context, ks []cid.Cid) <-chan blocks.Block {
	res := make(chan blocks.Block)

	go func() {
		defer close(res)

		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(getManyWorkers)

		err := b.db.View(func(txn *badger.Txn) error {
			for _, k := range ks {
				// TODO: Check lateer, not needed in new Go 1.24?
				k := k // Capture loop variable

				g.Go(func() error {
					item, err := txn.Get([]byte(k.String()))
					if err != nil {
						if !errors.Is(err, badger.ErrKeyNotFound) {
							log.Info("failed to get block", zap.Error(err), zap.String("key", k.String()))
						}
						return nil
					}

					val, err := item.ValueCopy(nil)
					if err != nil {
						log.Info("failed to copy block value", zap.Error(err), zap.String("key", k.String()))
						return nil
					}

					bl, err := blocks.NewBlockWithCid(val, k)
					if err != nil {
						log.Info("failed to create block", zap.Error(err), zap.String("key", k.String()))
						return nil
					}

					select {
					case res <- bl:
					case <-gctx.Done():
					}

					return nil
				})
			}

			return nil
		})
		if err != nil {
			log.Info("badger view transaction failed", zap.Error(err))
		}

		_ = g.Wait() // Ignore error since individual errors are already logged, no way to return error
	}()

	return res
}

func (b *Badger) Add(ctx context.Context, bs []blocks.Block) error {
	start := time.Now()
	wb := b.db.NewWriteBatch()
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

func (b *Badger) Delete(ctx context.Context, c cid.Cid) error {
	// TODO: Create an issue that no Delete call after clean up of Bin in Anytype.
	// Check before, that here is no deferred call to Delete

	start := time.Now()
	err := b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(c.String()))
	})

	log.Debug("badger delete",
		zap.Duration("total", time.Since(start)),
		zap.String("key", c.String()),
	)

	return err
}

func (b *Badger) DeleteMany(ctx context.Context, toDelete []cid.Cid) error {
	start := time.Now()
	wb := b.db.NewWriteBatch()
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

func (b *Badger) IndexGet(ctx context.Context, key string) (value []byte, err error) {
	indexKey := indexKeyPrefix + key

	err = b.db.View(func(txn *badger.Txn) error {
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

	return
}

// IndexPut stores a value in the index with the given key.
func (b *Badger) IndexPut(ctx context.Context, key string, value []byte) error {
	indexKey := indexKeyPrefix + key

	err := b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(indexKey), value)
	})

	log.Debug("badger index put", zap.String("key", indexKey))
	return err
}

func (b *Badger) Init(a *app.App) error {
	return nil
}

func (b *Badger) Name() string {
	return CName
}

// badgerLogger implements badger.Logger interface to redirect BadgerDB logs
// to our application logger with appropriate prefixes
type badgerLogger struct{}

func (b badgerLogger) Errorf(s string, i ...interface{}) {
	log.Error("badger: " + fmt.Sprintf(s, i...))
}

func (b badgerLogger) Warningf(s string, i ...interface{}) {
	log.Warn("badger: " + fmt.Sprintf(s, i...))
}

func (b badgerLogger) Infof(s string, i ...interface{}) {
	log.Info("badger: " + fmt.Sprintf(s, i...))
}

func (b badgerLogger) Debugf(s string, i ...interface{}) {
	log.Debug("badger: " + fmt.Sprintf(s, i...))
}

func (b *Badger) Run(ctx context.Context) error {
	opts := badger.DefaultOptions(b.path).
		// Core settings
		WithLogger(badgerLogger{}).
		WithMetricsEnabled(false).  // Metrics are not used
		WithDetectConflicts(false). // No need for conflict detection in blob store
		WithNumGoroutines(1).       // We don't use Stream
		// Memory and cache settings
		WithNumMemtables(4).          // Reduced to save memory, total possible MemTableSize(64MB)*NumMemtables if they are all full
		WithBlockCacheSize(64 << 20). // Block cache, we don't have reading same block frequently
		// LSM tree settings
		WithBaseTableSize(8 << 20).     // 8MB SSTable size
		WithNumLevelZeroTables(2).      // Match NumMemtables
		WithNumLevelZeroTablesStall(8). // Stall threshold
		WithNumCompactors(2).           // 2 compactors to reduce CPU usage
		// Value log settings
		WithValueLogFileSize(512 << 20). // 512MB per value log file
		// Disable compression since data is already encrypted
		WithCompression(options.None).
		WithZSTDCompressionLevel(0)

	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to open badger db: %w", err)
	}

	b.db = db

	// Start garbage collection in background
	go b.runGC(ctx)

	return nil
}

func (b *Badger) Close(ctx context.Context) error {
	return b.db.Close()
}

// TODO: Read about garbage collection in badger, looks like we don't need it often
// because here not a lot of deletions
func (b *Badger) runGC(ctx context.Context) {
	log.Info("starting badger garbage collection routine")
	ticker := time.NewTicker(29 * time.Hour) // Prime number interval, so it won't overlap with other routines (hopefully)
	defer ticker.Stop()

	const maxGCDuration = time.Minute

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping badger garbage collection routine")
			return
		case <-ticker.C:
			log.Debug("running badger garbage collection")
			start := time.Now()
			gcCount := 0

			for time.Since(start) < maxGCDuration {
				// Try to reclaim space if at least 50% of a value log file can be garbage collected
				err := b.db.RunValueLogGC(0.5)
				if err != nil {
					if errors.Is(err, badger.ErrNoRewrite) {
						break // No more files to GC
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

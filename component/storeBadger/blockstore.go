package storeBadger

import (
	"context"
	"errors"
	"fmt"
	"sync"
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

var _ s3store.S3Store = (*Badger)(nil)

const CName = fileblockstore.CName

var log = logger.NewNamed(CName)

// Was thinking about sqlite, but it may hard to backup and restore one big file.
// Also a few drawbacks - no concurrent writes, not sure that needed.

type Badger struct {
	path string
	db   *badger.DB
}

func New(path string) *Badger {
	return &Badger{
		path: path,
	}
}

func (b *Badger) Get(ctx context.Context, k cid.Cid) (blocks.Block, error) {
	st := time.Now()
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
		zap.Duration("total", time.Since(st)),
		zap.Int("kbytes", len(val)/1024),
		zap.String("key", k.String()),
	)
	return blocks.NewBlockWithCid(val, k)
}

func (b *Badger) GetMany(ctx context.Context, ks []cid.Cid) <-chan blocks.Block {
	res := make(chan blocks.Block)
	go func() {
		defer close(res)
		var wg sync.WaitGroup
		getManyLimiter := make(chan struct{}, 4)

		err := b.db.View(func(txn *badger.Txn) error {
			for _, k := range ks {
				wg.Add(1)
				select {
				case getManyLimiter <- struct{}{}:
				case <-ctx.Done():
					return ctx.Err()
				}
				go func(k cid.Cid) {
					defer func() { <-getManyLimiter }()
					defer wg.Done()

					item, err := txn.Get([]byte(k.String()))
					if err != nil {
						if !errors.Is(err, badger.ErrKeyNotFound) {
							log.Info("get error", zap.Error(err), zap.String("key", k.String()))
						}
						return
					}

					val, err := item.ValueCopy(nil)
					if err != nil {
						log.Info("get error", zap.Error(err), zap.String("key", k.String()))
						return
					}

					bl, err := blocks.NewBlockWithCid(val, k)
					if err != nil {
						log.Info("get error", zap.Error(err), zap.String("key", k.String()))
						return
					}

					select {
					case res <- bl:
					case <-ctx.Done():
					}
				}(k)
			}
			return nil
		})
		if err != nil {
			log.Info("view transaction error", zap.Error(err))
		}

		wg.Wait()
	}()
	return res
}

func (b *Badger) Add(ctx context.Context, bs []blocks.Block) error {
	st := time.Now()
	wb := b.db.NewWriteBatch()
	defer wb.Cancel()

	var dataLen int
	var keys []string
	for _, bl := range bs {
		data := bl.RawData()
		dataLen += len(data)
		key := bl.Cid().String()
		keys = append(keys, key)
		if err := wb.Set([]byte(key), data); err != nil {
			return err
		}
	}

	if err := wb.Flush(); err != nil {
		return err
	}

	log.Debug("badger put",
		zap.Duration("total", time.Since(st)),
		zap.Int("blocks", len(bs)),
		zap.Int("kbytes", dataLen/1024),
		zap.Strings("keys", keys),
	)
	return nil
}

func (b *Badger) Delete(ctx context.Context, c cid.Cid) error {
	st := time.Now()
	err := b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(c.String()))
	})
	log.Debug("badger delete",
		zap.Duration("total", time.Since(st)),
		zap.String("key", c.String()),
	)
	return err
}

func (b *Badger) DeleteMany(ctx context.Context, toDelete []cid.Cid) error {
	st := time.Now()
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
		zap.Duration("total", time.Since(st)),
		zap.Int("count", len(toDelete)),
		zap.Strings("keys", keys),
	)
	return err
}

func (b *Badger) IndexGet(ctx context.Context, key string) (value []byte, err error) {
	err = b.db.View(func(txn *badger.Txn) error {
		indexKey := "idx:" + key
		item, errGet := txn.Get([]byte(indexKey))

		if errGet != nil {
			if errors.Is(errGet, badger.ErrKeyNotFound) {
				return nil
			}

			return fmt.Errorf("failed to get index key: %w", errGet)
		}

		value, err = item.ValueCopy(nil)
		log.Debug("badger index get", zap.String("key", indexKey))
		return err
	})
	return
}

func (b *Badger) IndexPut(ctx context.Context, key string, value []byte) (err error) {
	indexKey := "idx:" + key
	err = b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(indexKey), value)
	})
	log.Debug("badger index put", zap.String("key", indexKey))
	return
}

func (b *Badger) Init(a *app.App) (err error) {
	return nil
}

func (b *Badger) Name() (name string) {
	return CName
}

// badgerLogger implements badger.Logger interface to redirect logs to our logger
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

func (b *Badger) Run(ctx context.Context) (err error) {
	opts := badger.DefaultOptions(b.path).
		// Core settings
		WithLogger(badgerLogger{}).
		WithMetricsEnabled(false).  // Metrics are not used
		WithDetectConflicts(false). // No need for conflict detection in blob store
		WithNumGoroutines(1).       // We don't use Stream
		// Memory and cache settings
		WithMemTableSize(32 << 20). // 32MB memtable (reduced from 64MB default)
		WithNumMemtables(3).        // Reduced from 5 to save memory
		WithBlockCacheSize(0).      // 32MB block cache (reduced, because no compression and encryption)
		// LSM tree settings
		WithBaseTableSize(8 << 20).     // 8MB SSTable size
		WithNumLevelZeroTables(3).      // Match NumMemtables
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

func (b *Badger) Close(ctx context.Context) (err error) {
	return b.db.Close()
}

// TODO: Read about garbage collection in badger
func (b *Badger) runGC(ctx context.Context) {
	log.Info("starting badger garbage collection routine")
	ticker := time.NewTicker(29 * time.Hour) // Prime number interval, so it won't overlap with other routines (hopefully)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping badger garbage collection routine")
			return
		case <-ticker.C:
			log.Debug("running badger garbage collection")
			gcStart := time.Now()
			gcCount := 0

			maxDuration := 1 * time.Minute

			for {
				if time.Since(gcStart) > maxDuration {
					log.Warn("badger gc timeout", zap.Duration("duration", time.Since(gcStart)))
					break
				}

				err := b.db.RunValueLogGC(0.5) // Clean if 50% space can be reclaimed, as in docs
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
				zap.Duration("duration", time.Since(gcStart)),
				zap.Int("filesGCed", gcCount))
		}
	}
}

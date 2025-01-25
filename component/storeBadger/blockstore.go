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

	badger "github.com/dgraph-io/badger/v4"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"
)

var _ s3store.S3Store = (*Badger)(nil)

const CName = fileblockstore.CName

var log = logger.NewNamed(CName)

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
	)
	return blocks.NewBlockWithCid(val, k)
}

func (b *Badger) GetMany(ctx context.Context, ks []cid.Cid) <-chan blocks.Block {
	res := make(chan blocks.Block)
	go func() {
		defer close(res)
		var wg sync.WaitGroup
		getManyLimiter := make(chan struct{}, 4)
		for _, k := range ks {
			wg.Add(1)
			select {
			case getManyLimiter <- struct{}{}:
			case <-ctx.Done():
				return
			}
			go func(k cid.Cid) {
				defer func() { <-getManyLimiter }()
				defer wg.Done()
				bl, err := b.Get(ctx, k)
				if err == nil {
					select {
					case res <- bl:
					case <-ctx.Done():
					}
				} else {
					log.Info("get error", zap.Error(err))
				}
			}(k)
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
	for _, bl := range bs {
		data := bl.RawData()
		dataLen += len(data)
		if err := wb.Set([]byte(bl.Cid().String()), data); err != nil {
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
	)
	return err
}

func (b *Badger) DeleteMany(ctx context.Context, toDelete []cid.Cid) error {
	st := time.Now()
	wb := b.db.NewWriteBatch()
	defer wb.Cancel()

	for _, c := range toDelete {
		if err := wb.Delete([]byte(c.String())); err != nil {
			log.Warn("can't delete cid", zap.Error(err))
		}
	}

	err := wb.Flush()
	log.Debug("badger delete many",
		zap.Duration("total", time.Since(st)),
		zap.Int("count", len(toDelete)),
	)
	return err
}

func (b *Badger) IndexGet(ctx context.Context, key string) (value []byte, err error) {
	err = b.db.View(func(txn *badger.Txn) error {
		item, errGet := txn.Get([]byte("idx:" + key))

		if errGet != nil {
			if errors.Is(errGet, badger.ErrKeyNotFound) {
				return nil
			}

			return fmt.Errorf("failed to get index key: %w", errGet)
		}

		value, err = item.ValueCopy(nil)
		return err
	})
	return
}

func (b *Badger) IndexPut(ctx context.Context, key string, value []byte) (err error) {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("idx:"+key), value)
	})
}

func (b *Badger) Init(a *app.App) (err error) {
	return nil
}

func (b *Badger) Name() (name string) {
	return CName
}

func (b *Badger) Run(ctx context.Context) (err error) {
	opts := badger.DefaultOptions(b.path)
	// opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to open badger db: %w", err)
	}

	b.db = db

	return nil
}

func (b *Badger) Close(ctx context.Context) (err error) {
	return b.db.Close()
}

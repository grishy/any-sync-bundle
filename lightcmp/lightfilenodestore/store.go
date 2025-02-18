package lightfilenodestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/commonfile/fileproto/fileprotoerr"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"
)

const (
	CName = "light.filenode.store"

	kPrefixFileNode      = "fn"
	kSeparator           = ":"
	kPrefixBlock         = kPrefixFileNode + kSeparator + "b"
	kPrefixCid           = kPrefixFileNode + kSeparator + "c"
	kPrefixFile          = kPrefixFileNode + kSeparator + "f"
	kPrefixLinkFileBlock = kPrefixFileNode + kSeparator + "l"
)

var log = logger.NewNamed(CName)

type cfgSrv interface {
	GetFilenodeStoreDir() string
}

type StoreService interface {
	app.Component

	// GetBlock retrieves a block by CID.
	GetBlock(ctx context.Context, k cid.Cid, spaceId string, wait bool) ([]byte, error)

	// PushBlock stores a block.
	PushBlock(ctx context.Context, spaceId string, fileId string, block blocks.Block) error
}

// lightfilenodestore implements a block store using BadgerDB.
// It provides persistent storage of content-addressed blocks with indexing support.
// Was thinking about sqlite, but it may hard to backup and restore one big file.
// A few drawbacks - no concurrent writes, not sure that needed.
type lightfilenodestore struct {
	cfg      cfgSrv
	badgerDB *badger.DB
}

func New() StoreService {
	return &lightfilenodestore{}
}

//
// App Component
//

func (b *lightfilenodestore) Init(a *app.App) error {
	log.Info("call Init")

	b.cfg = app.MustComponent[cfgSrv](a)

	return nil
}

func (b *lightfilenodestore) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (b *lightfilenodestore) Run(ctx context.Context) error {
	storePath := b.cfg.GetFilenodeStoreDir()
	// storePath := "./data/filenode_test_store"

	// TODO: Add OnTableRead for block checskum
	opts := badger.DefaultOptions(storePath).
		// Core settings
		WithLogger(badgerLogger{}).
		// Disable compression since data is already encrypted
		WithCompression(options.None).
		WithZSTDCompressionLevel(0)

	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to open badger db: %w", err)
	}

	b.badgerDB = db

	// Start garbage collection in background
	go b.runGC(ctx)

	return nil
}

func (b *lightfilenodestore) Close(ctx context.Context) error {
	return b.badgerDB.Close()
}

//
// Component methods
//

func keyFile(spaceId string, fileId string) []byte {
	return []byte(kPrefixFile + spaceId + kSeparator + fileId)
}

func keyLinkFileBlock(spaceId string, fileId string, k cid.Cid) []byte {
	return []byte(kPrefixLinkFileBlock + spaceId + kSeparator + fileId + kSeparator + k.String())
}

func (r *lightfilenodestore) checkBlock(ctx context.Context, k cid.Cid) (bool, error) {
	return false, nil
}

// TODO: Read about garbage collection in badger, looks like we don't need it often
// because here not a lot of deletions
func (r *lightfilenodestore) runGC(ctx context.Context) {
	log.Info("starting badger garbage collection routine")
	ticker := time.NewTicker(29 * time.Second) // Prime number interval, so it won't overlap with other routines (hopefully)
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
				err := r.badgerDB.RunValueLogGC(0.5)
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

func (r *lightfilenodestore) GetBlock(ctx context.Context, k cid.Cid, spaceId string, wait bool) ([]byte, error) {
	if wait {
		// TODO: Implement simple for loop with sleep for waiting
		// NOTE: 2025-02-15: Wait is not used on client side...So for now just panic.
		panic("wait not supported")
	}

	var (
		blockKey = keyBlock(spaceId, k)
		data     []byte
	)

	err := r.badgerDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(blockKey)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return fileprotoerr.ErrCIDNotFound
			}

			return fmt.Errorf("failed to get block: %w", err)
		}

		var errCopy error
		data, errCopy = item.ValueCopy(nil)
		if errCopy != nil {
			return fmt.Errorf("failed to copy block value: %w", errCopy)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed transaction: %w", err)
	}

	return data, nil
}

func (r *lightfilenodestore) PushBlock(ctx context.Context, spaceId string, fileId string, block blocks.Block) error {
	// For counter update
	// r.badgerDB.GetMergeOperator() // for counter update
	// r.badgerDB.NewWriteBatch()    // ??? use it ???

	blockKey := keyBlock(spaceId, block.Cid())
	fileKey := keyFile(spaceId, fileId)
	fileBlockLinkKey := keyLinkFileBlock(spaceId, fileId, block.Cid())

	_ = blockKey
	_ = fileKey
	_ = fileBlockLinkKey

	// err := r.badgerDB.Update(func(txn *badger.Txn) error {
	// 	// Check link, if exists, return nil, already exists
	// 	_, err := txn.Get(fileBlockLinkKey)
	// 	if err != nil {
	// 		if !errors.Is(err, badger.ErrKeyNotFound) {
	// 			return fmt.Errorf("failed to get link: %w", err)
	// 		}
	// 	}
	// 	// Link not found, continue

	// 	// Check block, if exists, update counter otherwise add block
	// 	_, err = txn.Get(blockKey)
	// 	if err != nil {
	// 		if !errors.Is(err, badger.ErrKeyNotFound) {

	// 	// TODO: Use block bind that update all counters

	// 	return nil
	// })
	// if err != nil {
	// 	return fmt.Errorf("failed to push block: %w", err)
	// }

	return nil
}

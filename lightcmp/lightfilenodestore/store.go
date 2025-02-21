//go:generate moq -fmt gofumpt -out store_mock.go . configService StoreService

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

	kPrefixFileNode = "fn"
	kSeparator      = ":"
)

var log = logger.NewNamed(CName)

type configService interface {
	app.Component

	GetFilenodeStoreDir() string
}

type StoreService interface {
	app.Component

	// TxView executes a read-only function within a transaction.
	TxView(f func(txn *badger.Txn) error) error

	// TxUpdate executes a read-write function within a transaction.
	TxUpdate(f func(txn *badger.Txn) error) error

	// GetBlock retrieves a block by CID.
	GetBlock(txn *badger.Txn, k cid.Cid, spaceId string) (*BlockObj, error)

	// PushBlock stores a block.
	PushBlock(txn *badger.Txn, spaceId string, block blocks.Block) error
}

// lightFileNodeStore implements a block store using BadgerDB.
// It provides persistent storage of content-addressed blocks with indexing support.
// Was thinking about sqlite, but it may hard to backup and restore one big file.
// A few drawbacks - no concurrent writes, not sure that needed.
type lightFileNodeStore struct {
	cfgSrv   configService
	badgerDB *badger.DB
}

func New() StoreService {
	return &lightFileNodeStore{}
}

//
// App Component
//

func (s *lightFileNodeStore) Init(a *app.App) error {
	log.Info("call Init")

	s.cfgSrv = app.MustComponent[configService](a)

	return nil
}

func (s *lightFileNodeStore) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (s *lightFileNodeStore) Run(ctx context.Context) error {
	storePath := s.cfgSrv.GetFilenodeStoreDir()

	// TODO: Add OnTableRead for block checksum
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

	s.badgerDB = db

	// Start garbage collection in background
	go s.runGC(ctx)

	return nil
}

func (s *lightFileNodeStore) Close(ctx context.Context) error {
	return s.badgerDB.Close()
}

// runGC periodically runs BadgerDB garbage collection to reclaim space.
// It runs every 30 seconds and attempts to GC value log files that are at least 50% garbage.
func (s *lightFileNodeStore) runGC(ctx context.Context) {
	log.Info("starting badger garbage collection routine")
	ticker := time.NewTicker(29 * time.Second) // Prime number interval, so it won't overlap with other routines (hopefully)
	defer ticker.Stop()

	const (
		maxGCDuration = time.Minute
		gcThreshold   = 0.5 // Reclaim files with >= 50% garbage
	)

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping badger garbage collection routine")
			return
		case <-ticker.C:
			start := time.Now()
			gcCount := 0

			// Run GC until we hit the time limit or have no more files to collect
			for time.Since(start) < maxGCDuration {
				if err := s.badgerDB.RunValueLogGC(gcThreshold); err != nil {
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

//
// Component methods
//

// TxView executes a read-only function within a transaction.
func (s *lightFileNodeStore) TxView(f func(txn *badger.Txn) error) error {
	return s.badgerDB.View(f)
}

// TxUpdate executes a read-write function within a transaction.
func (s *lightFileNodeStore) TxUpdate(f func(txn *badger.Txn) error) error {
	return s.badgerDB.Update(f)
}

// GetBlock retrieves a block by CID. Read-only transaction is used.
func (s *lightFileNodeStore) GetBlock(txn *badger.Txn, k cid.Cid, spaceId string) (*BlockObj, error) {
	log.Info("GetBlock",
		zap.String("cid", k.String()),
		zap.String("spaceId", spaceId),
	)

	block := NewBlockObj(spaceId, k)
	if err := block.populateData(txn); err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, fileprotoerr.ErrCIDNotFound
		}
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	return block, nil
}

func (s *lightFileNodeStore) PushBlock(txn *badger.Txn, spaceId string, blk blocks.Block) error {
	log.Debug("PushBlock",
		zap.String("spaceId", spaceId),
		zap.String("cid", blk.Cid().String()),
	)

	blkObj := NewBlockObj(spaceId, blk.Cid()).WithData(blk.RawData())
	if err := blkObj.write(txn); err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}

	dataSize := uint32(len(blkObj.Data()))
	cidObj := NewCidObj(spaceId, blk.Cid()).
		WithRefCount(1).
		WithSizeByte(dataSize).
		WithCreatedAt(time.Now().Unix())

	if err := cidObj.write(txn); err != nil {
		return fmt.Errorf("failed to save CID: %w", err)
	}

	return nil
}

//go:generate moq -fmt gofumpt -rm -out store_mock.go . configService StoreService

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

	defaultLimitBytes = 1 << 40 // 1 TiB in bytes

	kPrefixFileNode = "fn"
	kSeparator      = ":"
)

var log = logger.NewNamed(CName)

type configService interface {
	app.Component

	GetFilenodeStoreDir() string
	GetFilenodeDefaultLimitBytes() uint64
}

type StoreService interface {
	app.Component

	// TxView executes a read-only function within a transaction.
	TxView(f func(txn *badger.Txn) error) error

	// TxUpdate executes a read-write function within a transaction.
	TxUpdate(f func(txn *badger.Txn) error) error

	// HadCID checks if the given CID exists in the store.
	HadCID(txn *badger.Txn, k cid.Cid) (bool, error)

	// HasCIDInSpace checks if the given CIDs exist in the store.
	HasCIDInSpace(txn *badger.Txn, spaceId string, k cid.Cid) (bool, error)

	// GetBlock retrieves a block by CID.
	GetBlock(txn *badger.Txn, k cid.Cid) (*BlockObj, error)

	// PushBlock stores a block.
	PushBlock(txn *badger.Txn, spaceId string, block blocks.Block) error

	// GetSpace retrieves a space by ID.
	GetSpace(txn *badger.Txn, spaceId string) (*SpaceObj, error)

	// GetGroup retrieves a group by ID.
	GetGroup(txn *badger.Txn, groupId string) (*GroupObj, error)

	// GetFile retrieves a file by ID.
	GetFile(txn *badger.Txn, spaceId, fileId string) (*FileObj, error)

	// CreateLinkFileBlock creates a link between a file and a block.
	CreateLinkFileBlock(txn *badger.Txn, spaceId, fileId string, blk blocks.Block) error

	// CreateLinkGroupSpace creates a link between a group and a space.
	CreateLinkGroupSpace(txn *badger.Txn, groupId, spaceId string) error
}

// lightFileNodeStore implements a block store using BadgerDB.
// It provides persistent storage of content-addressed blocks with indexing support.
// Was thinking about sqlite, but it may hard to backup and restore one big file.
// A few drawbacks - no concurrent writes, not sure that needed.
type lightFileNodeStore struct {
	cfgSrv   configService
	badgerDB *badger.DB

	// Used to set default limit for user/group
	defaultLimitBytes uint64
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
	s.defaultLimitBytes = s.cfgSrv.GetFilenodeDefaultLimitBytes()
	if s.defaultLimitBytes == 0 {
		s.defaultLimitBytes = defaultLimitBytes
	}

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
func (s *lightFileNodeStore) GetBlock(txn *badger.Txn, k cid.Cid) (*BlockObj, error) {
	log.Info("GetBlock",
		zap.String("cid", k.String()),
	)

	blkObj := NewBlockObj(k)
	if err := blkObj.populateData(txn); err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, fileprotoerr.ErrCIDNotFound
		}
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	return blkObj, nil
}

func (s *lightFileNodeStore) HasCIDInSpace(txn *badger.Txn, spaceId string, k cid.Cid) (bool, error) {
	log.Info("HasCIDInSpace",
		zap.String("cid", k.String()),
		zap.String("spaceId", spaceId),
	)

	blkObj := NewCidObj(spaceId, k)
	return blkObj.exists(txn)
}

func (s *lightFileNodeStore) HadCID(txn *badger.Txn, k cid.Cid) (bool, error) {
	log.Info("HadCID",
		zap.String("cid", k.String()),
	)

	blkObj := NewBlockObj(k)
	return blkObj.exists(txn)
}

// PushBlock stores a block. Read-write transaction is used.
// We store the block data and CID metadata in separate keys.
// Block data is stored only by CID separations and CID metadata is stored by spaceId and CID.
// This allows us to easily check is CID just exists or exists in the specific space.
func (s *lightFileNodeStore) PushBlock(txn *badger.Txn, spaceId string, blk blocks.Block) error {
	log.Debug("PushBlock",
		zap.String("spaceId", spaceId),
		zap.String("cid", blk.Cid().String()),
	)

	blkObj := NewBlockObj(blk.Cid()).WithData(blk.RawData())
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

func (s *lightFileNodeStore) GetSpace(txn *badger.Txn, spaceId string) (*SpaceObj, error) {
	log.Debug("GetSpace",
		zap.String("spaceId", spaceId),
	)

	spaceObj := NewSpaceObj(spaceId)
	if err := spaceObj.populateValue(txn); err != nil {
		// We don't return error if space not found, just return empty space object
		if errors.Is(err, badger.ErrKeyNotFound) {
			return spaceObj, nil
		}

		return nil, fmt.Errorf("failed to get space: %w", err)
	}

	return spaceObj, nil
}

func (s *lightFileNodeStore) GetGroup(txn *badger.Txn, groupId string) (*GroupObj, error) {
	log.Debug("GetGroup",
		zap.String("groupId", groupId),
	)

	groupObj := NewGroupObj(groupId)
	if err := groupObj.populateValue(txn); err != nil {
		// We don't return error if group not found, just return empty group object with default limit
		if errors.Is(err, badger.ErrKeyNotFound) {
			groupObj = groupObj.WithLimitBytes(s.defaultLimitBytes)
			return groupObj, nil
		}

		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	return groupObj, nil
}

func (s *lightFileNodeStore) GetFile(txn *badger.Txn, spaceId, fileId string) (*FileObj, error) {
	log.Debug("GetFile",
		zap.String("fileId", fileId),
	)

	fileObj := NewFileObj(spaceId, fileId)
	if err := fileObj.populateValue(txn); err != nil {
		// We don't return error if file not found, just return empty file object
		if errors.Is(err, badger.ErrKeyNotFound) {
			return fileObj, nil
		}

		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return fileObj, nil
}

func (s *lightFileNodeStore) CreateLinkFileBlock(txn *badger.Txn, spaceId, fileId string, blk blocks.Block) error {
	log.Debug("CreateLinkFileBlock",
		zap.String("spaceId", spaceId),
		zap.String("fileId", fileId),
		zap.String("cid", blk.Cid().String()),
	)

	linkObj := NewLinkFileBlockObj(spaceId, fileId, blk.Cid())
	if err := linkObj.write(txn); err != nil {
		return fmt.Errorf("failed to create link file block: %w", err)
	}

	return nil
}

func (s *lightFileNodeStore) CreateLinkGroupSpace(txn *badger.Txn, groupId, spaceId string) error {
	log.Debug("CreateLinkGroupSpace",
		zap.String("groupId", groupId),
		zap.String("spaceId", spaceId),
	)

	linkObj := NewLinkGroupSpaceObj(groupId, spaceId)
	if err := linkObj.write(txn); err != nil {
		return fmt.Errorf("failed to create link group space: %w", err)
	}

	return nil
}

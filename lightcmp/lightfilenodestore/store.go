//go:generate moq -fmt gofumpt -rm -out store_mock.go . configService StoreService

package lightfilenodestore

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"
)

const (
	CName = "light.filenode.store"

	// Storage key prefixes and separators
	prefixFileNode = "fn"
	separator      = ":"
	blockType      = "b"
	snapshotType   = "s"
	logType        = "l"

	currentSnapshotSuffix = "current"
)

// GC related constants
const (
	gcInterval    = 29 * time.Second // Prime number to avoid alignment with other timers
	maxGCDuration = time.Minute
	gcThreshold   = 0.5 // Reclaim files with >= 50% garbage
)

var (
	log              = logger.NewNamed(CName)
	ErrNoSnapshot    = errors.New("no index snapshot found")
	ErrBlockNotFound = errors.New("block not found")
)

type configService interface {
	app.Component
	GetFilenodeStoreDir() string
}

// IndexLog represents a single WAL entry for index changes
type IndexLog struct {
	Idx  uint64 // Log entry counter
	Data []byte
}

// StoreService defines operations for persistent storage
type StoreService interface {
	app.ComponentRunnable

	// Transaction operations
	TxView(f func(txn *badger.Txn) error) error
	TxUpdate(f func(txn *badger.Txn) error) error

	// Block operations
	GetBlock(txn *badger.Txn, k cid.Cid) ([]byte, error)
	PutBlock(txn *badger.Txn, block blocks.Block) error
	DeleteBlock(txn *badger.Txn, c cid.Cid) error

	// Index snapshot operations
	GetIndexSnapshot(txn *badger.Txn) ([]byte, error)
	SaveIndexSnapshot(txn *badger.Txn, data []byte) error

	// Index log operations
	GetIndexLogs(txn *badger.Txn) ([]IndexLog, error)
	DeleteIndexLogs(txn *badger.Txn, idxs []uint64) error
	PushIndexLog(txn *badger.Txn, logData []byte) (uint64, error)
}

type lightFileNodeStore struct {
	srvCfg configService
	db     *badger.DB
}

// New creates a new StoreService instance
func New() StoreService {
	return &lightFileNodeStore{}
}

func (s *lightFileNodeStore) Init(a *app.App) error {
	log.Info("initializing light filenode store")
	s.srvCfg = app.MustComponent[configService](a)
	return nil
}

func (s *lightFileNodeStore) Name() string {
	return CName
}

func (s *lightFileNodeStore) Run(ctx context.Context) error {
	storePath := s.srvCfg.GetFilenodeStoreDir()

	opts := badger.DefaultOptions(storePath).
		WithLogger(badgerLogger{}).
		WithCompression(options.None).
		WithZSTDCompressionLevel(0)

	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to open badger db: %w", err)
	}

	s.db = db

	// Start garbage collection in background
	go s.runGC(ctx)

	return nil
}

func (s *lightFileNodeStore) Close(ctx context.Context) error {
	return s.db.Close()
}

func (s *lightFileNodeStore) runGC(ctx context.Context) {
	log.Info("starting badger garbage collection routine")
	ticker := time.NewTicker(gcInterval)
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
			for time.Since(start) < maxGCDuration {
				if err := s.db.RunValueLogGC(gcThreshold); err != nil {
					if errors.Is(err, badger.ErrNoRewrite) {
						// No more files need GC
						break
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

// Transaction methods

func (s *lightFileNodeStore) TxView(f func(txn *badger.Txn) error) error {
	return s.db.View(f)
}

func (s *lightFileNodeStore) TxUpdate(f func(txn *badger.Txn) error) error {
	return s.db.Update(f)
}

// Key management functions

// buildKey creates a key with format: fn:type:suffix
func buildKey(keyType, suffix string) []byte {
	key := make([]byte, 0, len(prefixFileNode)+len(separator)*2+len(keyType)+len(suffix))
	key = append(key, prefixFileNode...)
	key = append(key, separator...)
	key = append(key, keyType...)
	key = append(key, separator...)
	key = append(key, suffix...)
	return key
}

// buildKeyPrefix creates a key prefix with format: fn:type:
func buildKeyPrefix(keyType string) []byte {
	prefix := make([]byte, 0, len(prefixFileNode)+len(separator)*2+len(keyType))
	prefix = append(prefix, prefixFileNode...)
	prefix = append(prefix, separator...)
	prefix = append(prefix, keyType...)
	prefix = append(prefix, separator...)
	return prefix
}

func blockKey(c cid.Cid) []byte {
	return buildKey(blockType, c.String())
}

func snapshotKey() []byte {
	return buildKey(snapshotType, currentSnapshotSuffix)
}

func logKey(idx uint64) []byte {
	return buildKey(logType, strconv.FormatUint(idx, 10))
}

func logKeyPrefix() []byte {
	return buildKeyPrefix(logType)
}

// parseIdxFromKey extracts the index from a key string
func parseIdxFromKey(key []byte) (uint64, error) {
	lastSep := -1
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == separator[0] {
			lastSep = i
			break
		}
	}

	if lastSep == -1 || lastSep >= len(key)-1 {
		return 0, fmt.Errorf("malformed key: %s", key)
	}

	return strconv.ParseUint(string(key[lastSep+1:]), 10, 64)
}

// Block methods

func (s *lightFileNodeStore) GetBlock(txn *badger.Txn, k cid.Cid) ([]byte, error) {
	item, err := txn.Get(blockKey(k))
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrBlockNotFound, k)
		}
		return nil, fmt.Errorf("failed to get block %s: %w", k, err)
	}
	return item.ValueCopy(nil)
}

func (s *lightFileNodeStore) PutBlock(txn *badger.Txn, block blocks.Block) error {
	return txn.Set(blockKey(block.Cid()), block.RawData())
}

func (s *lightFileNodeStore) DeleteBlock(txn *badger.Txn, c cid.Cid) error {
	return txn.Delete(blockKey(c))
}

// Index snapshot methods

func (s *lightFileNodeStore) GetIndexSnapshot(txn *badger.Txn) ([]byte, error) {
	item, err := txn.Get(snapshotKey())
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, ErrNoSnapshot
		}
		return nil, fmt.Errorf("failed to get index snapshot: %w", err)
	}

	return item.ValueCopy(nil)
}

func (s *lightFileNodeStore) SaveIndexSnapshot(txn *badger.Txn, data []byte) error {
	if err := txn.Set(snapshotKey(), data); err != nil {
		return fmt.Errorf("failed to store index snapshot: %w", err)
	}
	return nil
}

// Index log methods

func (s *lightFileNodeStore) GetIndexLogs(txn *badger.Txn) ([]IndexLog, error) {
	prefix := logKeyPrefix()
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = true

	it := txn.NewIterator(opts)
	defer it.Close()

	var logs []IndexLog

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		idx, err := parseIdxFromKey(item.Key())
		if err != nil {
			continue // Skip malformed keys
		}

		data, err := item.ValueCopy(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to read log data: %w", err)
		}

		logs = append(logs, IndexLog{Idx: idx, Data: data})
	}

	return logs, nil
}

func (s *lightFileNodeStore) DeleteIndexLogs(txn *badger.Txn, indices []uint64) error {
	if len(indices) == 0 {
		return nil
	}

	for _, idx := range indices {
		if err := txn.Delete(logKey(idx)); err != nil {
			return fmt.Errorf("failed to delete log with index %d: %w", idx, err)
		}
	}

	return nil
}

func (s *lightFileNodeStore) PushIndexLog(txn *badger.Txn, logData []byte) (uint64, error) {
	idx, err := s.getNextIdx(txn, logKeyPrefix())
	if err != nil {
		return 0, err
	}

	if err := txn.Set(logKey(idx), logData); err != nil {
		return 0, fmt.Errorf("failed to store index log: %w", err)
	}

	return idx, nil
}

// getNextIdx finds the next available index for items with the given prefix
func (s *lightFileNodeStore) getNextIdx(txn *badger.Txn, prefix []byte) (uint64, error) {
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Reverse = true

	it := txn.NewIterator(opts)
	defer it.Close()

	// Start with index 1 if no existing items
	idx := uint64(1)

	it.Seek(append(prefix, 0xFF))
	if it.ValidForPrefix(prefix) {
		keyIdx, err := parseIdxFromKey(it.Item().Key())
		if err == nil {
			idx = keyIdx + 1
		}
	}

	return idx, nil
}

//go:generate moq -fmt gofumpt -rm -out store_mock.go . configService StoreService

package lightfilenodestore

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"
)

const CName = "light.filenode.store"

const (
	// Key prefixes and structure.
	prefixFileNode        = "fn"
	separator             = ":"
	blockType             = "b"
	snapshotType          = "s"
	logType               = "l"
	currentSnapshotSuffix = "current"

	// GC configuration.
	defaultGCInterval    = 29 * time.Hour // Prime number to avoid alignment
	defaultMaxGCDuration = time.Minute
	defaultGCThreshold   = 0.5 // Reclaim files with >= 50% garbage
)

var (
	// Type assertion.
	_ StoreService = (*lightFileNodeStore)(nil)

	log = logger.NewNamed(CName)

	// Errors.
	ErrNoSnapshot    = errors.New("no index snapshot found")
	ErrBlockNotFound = errors.New("block not found")
	ErrInvalidKey    = errors.New("invalid key format")
)

type configService interface {
	app.Component
	GetFilenodeStoreDir() string
}

// StoreService defines operations for persistent storage.
// NOTE: Perfectly we should have a separate badger.Txn to interface.
type StoreService interface {
	app.ComponentRunnable

	// Transaction operations.
	TxView(f func(txn *badger.Txn) error) error
	TxUpdate(f func(txn *badger.Txn) error) error

	// Block operations.
	GetBlock(txn *badger.Txn, k cid.Cid) ([]byte, error)
	PutBlock(txn *badger.Txn, block blocks.Block) error
	DeleteBlock(txn *badger.Txn, c cid.Cid) error

	// Index snapshot operations.
	GetIndexSnapshot(txn *badger.Txn) ([]byte, error)
	SaveIndexSnapshot(txn *badger.Txn, data []byte) error

	// Index log operations.
	GetIndexLogs(txn *badger.Txn) ([]IndexLog, error)
	DeleteIndexLogs(txn *badger.Txn, idxs []uint64) error
	PushIndexLog(txn *badger.Txn, logData []byte) error
}

// IndexLog represents a single WAL entry for index changes.
type IndexLog struct {
	Idx  uint64
	Data []byte
}

type storeConfig struct {
	gcInterval    time.Duration
	maxGCDuration time.Duration
	gcThreshold   float64
}

type lightFileNodeStore struct {
	srvCfg configService
	cfg    storeConfig
	db     *badger.DB

	// TODO: Implement atomic counter for log index, do not calculate on every push.
	// currentIndex atomic.Uint64
}

func New() *lightFileNodeStore {
	return &lightFileNodeStore{
		cfg: storeConfig{
			gcInterval:    defaultGCInterval,
			maxGCDuration: defaultMaxGCDuration,
			gcThreshold:   defaultGCThreshold,
		},
	}
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

	go s.runGC(ctx)

	return nil
}

func (s *lightFileNodeStore) Close(ctx context.Context) error {
	return s.db.Close()
}

func (s *lightFileNodeStore) TxView(f func(txn *badger.Txn) error) error {
	return s.db.View(f)
}

func (s *lightFileNodeStore) TxUpdate(f func(txn *badger.Txn) error) error {
	return s.db.Update(f)
}

func (s *lightFileNodeStore) GetBlock(txn *badger.Txn, k cid.Cid) ([]byte, error) {
	item, err := txn.Get(buildKey(blockType, k.String()))
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrBlockNotFound, k)
		}
		return nil, fmt.Errorf("failed to get block %s: %w", k, err)
	}

	return item.ValueCopy(nil)
}

func (s *lightFileNodeStore) PutBlock(txn *badger.Txn, block blocks.Block) error {
	key := buildKey(blockType, block.Cid().String())
	return txn.Set(key, block.RawData())
}

func (s *lightFileNodeStore) DeleteBlock(txn *badger.Txn, c cid.Cid) error {
	return txn.Delete(buildKey(blockType, c.String()))
}

func (s *lightFileNodeStore) GetIndexSnapshot(txn *badger.Txn) ([]byte, error) {
	item, err := txn.Get(buildKey(snapshotType, currentSnapshotSuffix))
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, ErrNoSnapshot
		}
		return nil, fmt.Errorf("failed to get index snapshot: %w", err)
	}
	return item.ValueCopy(nil)
}

func (s *lightFileNodeStore) SaveIndexSnapshot(txn *badger.Txn, data []byte) error {
	key := buildKey(snapshotType, currentSnapshotSuffix)
	if err := txn.Set(key, data); err != nil {
		return fmt.Errorf("failed to store index snapshot: %w", err)
	}
	return nil
}

func (s *lightFileNodeStore) GetIndexLogs(txn *badger.Txn) ([]IndexLog, error) {
	prefix := buildKeyPrefix(logType)
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = true

	it := txn.NewIterator(opts)
	defer it.Close()

	var logs []IndexLog
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		idx, err := parseIdxFromKey(item.Key())
		if err != nil {
			return nil, fmt.Errorf("failed to parse log index key='%s': %w", item.Key(), err)
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
		key := buildKey(logType, strconv.FormatUint(idx, 10))
		if err := txn.Delete(key); err != nil {
			return fmt.Errorf("failed to delete log with index %d: %w", idx, err)
		}
	}
	return nil
}

func (s *lightFileNodeStore) PushIndexLog(txn *badger.Txn, logData []byte) error {
	idx, err := s.getNextLogIndex(txn)
	if err != nil {
		return fmt.Errorf("failed to get next log index: %w", err)
	}

	key := buildKey(logType, strconv.FormatUint(idx, 10))
	if err := txn.Set(key, logData); err != nil {
		return fmt.Errorf("failed to store index log: %w", err)
	}

	return nil
}

func buildKey(keyType string, suffix string) []byte {
	key := make([]byte, 0, len(prefixFileNode)+len(separator)*2+len(keyType)+len(suffix))
	key = append(key, prefixFileNode...)
	key = append(key, separator...)
	key = append(key, keyType...)
	key = append(key, separator...)
	key = append(key, suffix...)
	return key
}

func buildKeyPrefix(keyType string) []byte {
	prefix := make([]byte, 0, len(prefixFileNode)+len(separator)*2+len(keyType))
	prefix = append(prefix, prefixFileNode...)
	prefix = append(prefix, separator...)
	prefix = append(prefix, keyType...)
	prefix = append(prefix, separator...)
	return prefix
}

func parseIdxFromKey(key []byte) (uint64, error) {
	keyStr := string(key)

	lastSepIdx := strings.LastIndex(keyStr, separator)
	if lastSepIdx == -1 || lastSepIdx >= len(keyStr)-1 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidKey, keyStr)
	}

	idx, err := strconv.ParseUint(keyStr[lastSepIdx+1:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidKey, keyStr)
	}

	return idx, nil
}

func (s *lightFileNodeStore) getNextLogIndex(txn *badger.Txn) (uint64, error) {
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Reverse = true

	prefix := buildKeyPrefix(logType)
	it := txn.NewIterator(opts)
	defer it.Close()

	// Start with index 1 if no existing items.
	idx := uint64(1)

	it.Seek(append(prefix, 0xFF))
	if it.ValidForPrefix(prefix) {
		if keyIdx, err := parseIdxFromKey(it.Item().Key()); err == nil {
			idx = keyIdx + 1
		}
	}

	return idx, nil
}

func (s *lightFileNodeStore) runGC(ctx context.Context) {
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

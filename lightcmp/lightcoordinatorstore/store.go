//go:generate go tool moq -fmt gofumpt -rm -out store_moq_test.go . dbService
package lightcoordinatorstore

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/coordinator/coordinatorproto"
	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/grishy/any-sync-bundle/lightcmp/lightcoordinatorstore/storepb"
)

const (
	CName = "light.coordinator.store"

	// Key prefixes for BadgerDB
	prefixCoordinator = "coordinator"
	separator         = ":"
	deleteLog         = "del"
	aclEvent          = "acl"

	// Default limits
	defaultDeleteLogLimit = 1000
)

var (
	// Type assertion
	_ StoreService = (*lightcoordinatorstore)(nil)

	log = logger.NewNamed(CName)
)

type StoreService interface {
	app.ComponentRunnable

	// ACL events
	AclEventAddLog(txn *badger.Txn, event AclEventLogEntry) (err error)
	AclEventGetAfter(txn *badger.Txn, identity, afterId string, limit uint32) (records []AclEventLogEntry, hasMore bool, err error)

	// Delete logs
	DeleteLogAdd(txn *badger.Txn, spaceId, fileGroup string, status coordinatorproto.DeletionLogRecordStatus) (id string, err error)
	DeleteLogGetAfter(txn *badger.Txn, afterId string, limit uint32) (records []DeleteLogRecord, hasMore bool, err error)

	// Space status
	// TODO
}

type EventLogEntryType uint8

const (
	EntryTypeSpaceReceipt      EventLogEntryType = 0
	EntryTypeSpaceShared       EventLogEntryType = 1
	EntryTypeSpaceUnshared     EventLogEntryType = 2
	EntryTypeSpaceAclAddRecord EventLogEntryType = 3
)

type AclEventLogEntry struct {
	Id        string
	SpaceId   string
	PeerId    string
	Owner     string
	Timestamp int64

	EntryType EventLogEntryType
	// only for EntryTypeSpaceAclAddRecord
	AclChangeId string
}

type DeleteLogRecord struct {
	Id        string
	Timestamp int64
	SpaceId   string
	FileGroup string
	Status    coordinatorproto.DeletionLogRecordStatus
}

type lightcoordinatorstore struct {
	db dbService

	mu              sync.RWMutex
	deleteLastIndex uint64
}

type dbService interface {
	app.Component

	TxView(f func(txn *badger.Txn) error) error
	TxUpdate(f func(txn *badger.Txn) error) error
}

func New() *lightcoordinatorstore {
	return new(lightcoordinatorstore)
}

//
// App Component
//

func (r *lightcoordinatorstore) Init(a *app.App) error {
	log.Info("call Init")

	r.db = app.MustComponent[dbService](a)

	return nil
}

func (r *lightcoordinatorstore) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (r *lightcoordinatorstore) Run(ctx context.Context) error {
	log.Info("call Run")

	// Calculate the last index from the database
	if err := r.initializeLastIndices(); err != nil {
		return fmt.Errorf("failed to initialize indices: %w", err)
	}

	return nil
}

func (r *lightcoordinatorstore) Close(ctx context.Context) error {
	log.Info("call Close")
	return nil
}

//
// Component methods
//

func (r *lightcoordinatorstore) AclEventAddLog(txn *badger.Txn, event AclEventLogEntry) (err error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorstore) AclEventGetAfter(txn *badger.Txn, identity, afterId string, limit uint32) (records []AclEventLogEntry, hasMore bool, err error) {
	// TODO implement me
	panic("implement me")
}

func (r *lightcoordinatorstore) DeleteLogAdd(txn *badger.Txn, spaceId, fileGroup string, status coordinatorproto.DeletionLogRecordStatus) (id string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.deleteLastIndex++
	recordId := fmt.Sprintf("%d", r.deleteLastIndex)

	pbRecord := &storepb.DeleteLogRecord{}
	pbRecord.SetTimestamp(time.Now().UnixNano())
	pbRecord.SetId(recordId)
	pbRecord.SetFileGroup(fileGroup)
	pbRecord.SetSpaceId(spaceId)
	pbRecord.SetStatus(int32(status))

	data, err := proto.Marshal(pbRecord)
	if err != nil {
		return "", fmt.Errorf("failed to marshal deletion log record: %w", err)
	}

	var recordIdBytes [8]byte
	binary.BigEndian.PutUint64(recordIdBytes[:], r.deleteLastIndex)
	key := buildKey(deleteLog, recordIdBytes[:])

	if errTx := txn.Set(key, data); errTx != nil {
		return "", fmt.Errorf("failed to store deletion log record: %w", errTx)
	}

	return recordId, nil
}

func (r *lightcoordinatorstore) DeleteLogGetAfter(txn *badger.Txn, afterId string, limit uint32) (records []DeleteLogRecord, hasMore bool, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit == 0 || limit > defaultDeleteLogLimit {
		limit = defaultDeleteLogLimit
	}

	// Add 1 to detect if there are more records
	fetchLimit := limit + 1

	startKey, err := buildDeleteLogStartKey(afterId)
	if err != nil {
		return nil, false, err
	}

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = true

	it := txn.NewIterator(opts)
	defer it.Close()

	collectedRecords := make([]DeleteLogRecord, 0, fetchLimit)
	prefixKey := buildKey(deleteLog, nil)

	for it.Seek(startKey); it.Valid(); it.Next() {
		item := it.Item()
		key := item.Key()

		// Check if we're still within our prefix
		if !bytes.HasPrefix(key, prefixKey) {
			break
		}

		// We've reached our limit
		if len(collectedRecords) >= int(fetchLimit) {
			break
		}

		val, errCp := item.ValueCopy(nil)
		if errCp != nil {
			return nil, false, fmt.Errorf("failed to copy value: %w", errCp)
		}

		pbRecord := &storepb.DeleteLogRecord{}
		if errUnm := proto.Unmarshal(val, pbRecord); errUnm != nil {
			return nil, false, fmt.Errorf("failed to unmarshal deletion log record: %w", errUnm)
		}

		record := DeleteLogRecord{
			Id:        pbRecord.GetId(),
			Timestamp: pbRecord.GetTimestamp(),
			SpaceId:   pbRecord.GetSpaceId(),
			FileGroup: pbRecord.GetFileGroup(),
			Status:    coordinatorproto.DeletionLogRecordStatus(pbRecord.GetStatus()),
		}

		collectedRecords = append(collectedRecords, record)
	}

	size := len(collectedRecords)
	if size > int(limit) {
		return collectedRecords[:limit], true, nil
	}

	return collectedRecords, false, nil
}

func buildDeleteLogStartKey(afterId string) ([]byte, error) {
	if afterId == "" {
		// Start from the beginning
		return buildKey(deleteLog, []byte{}), nil
	}

	// Convert afterId to uint64
	afterIdInt, err := strconv.ParseUint(afterId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid afterId: %w", err)
	}

	// Add 1 to start after (>), not from (>=)
	afterIdInt++

	// Create key to start after this record
	var recordIdBytes [8]byte
	binary.BigEndian.PutUint64(recordIdBytes[:], afterIdInt)
	return buildKey(deleteLog, recordIdBytes[:]), nil
}

func buildKey(keyType string, suffix []byte) []byte {
	key := make([]byte, 0, len(prefixCoordinator)+len(separator)*2+len(keyType)+len(suffix))
	key = append(key, []byte(prefixCoordinator)...)
	key = append(key, separator...)
	key = append(key, keyType...)
	key = append(key, separator...)
	key = append(key, suffix...)
	return key
}

func (r *lightcoordinatorstore) initializeLastIndices() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.deleteLastIndex = 0

	return r.db.TxView(func(txn *badger.Txn) error {
		maxDeleteIndex, err := findMaxDeleteLogIndex(txn)
		if err != nil {
			return err
		}

		r.deleteLastIndex = maxDeleteIndex

		log.Info("initialized indices",
			zap.Uint64("deleteLastIndex", r.deleteLastIndex),
		)

		return nil
	})
}

func findMaxDeleteLogIndex(txn *badger.Txn) (uint64, error) {
	var maxIndex uint64 = 0

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = true
	opts.Reverse = true

	it := txn.NewIterator(opts)
	defer it.Close()

	prefixKey := buildKey(deleteLog, nil)

	// No loop, because we need only need the highest ID (first record),
	it.Seek(append(prefixKey, 0xFF))
	if !it.Valid() {
		// No delete log records found
		return 0, nil
	}

	item := it.Item()
	key := item.Key()

	if !bytes.HasPrefix(key, prefixKey) {
		return 0, fmt.Errorf("unexpected key: %s", key)
	}

	// Extract the numeric ID from the key format is prefixCoordinator:deleteLog:<8-byte-uint64>
	keyParts := bytes.Split(key, []byte(separator))
	if len(keyParts) < 3 {
		return 0, fmt.Errorf("malformed key: %s", key)
	}

	// Get the ID part (last part of the key)
	idBytes := keyParts[len(keyParts)-1]
	if len(idBytes) >= 8 {
		currentId := binary.BigEndian.Uint64(idBytes)
		if currentId > maxIndex {
			maxIndex = currentId
		}
	}

	return maxIndex, nil
}

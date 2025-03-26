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

	// TODO: Calculate the last index from the database
	// - delete log
	// - acl event log

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

	// Create protobuf record
	pbRecord := &storepb.DeleteLogRecord{}
	pbRecord.SetTimestamp(time.Now().UnixNano())
	pbRecord.SetId(recordId)
	pbRecord.SetFileGroup(fileGroup)
	pbRecord.SetSpaceId(spaceId)
	pbRecord.SetStatus(int32(status))

	// Marshal to bytes
	data, err := proto.Marshal(pbRecord)
	if err != nil {
		return "", fmt.Errorf("failed to marshal deletion log record: %w", err)
	}

	// Create a key for the record
	var recordIdBytes [8]byte
	binary.BigEndian.PutUint64(recordIdBytes[:], r.deleteLastIndex)
	key := buildKey(deleteLog, recordIdBytes[:])

	// Add to database
	if errTx := txn.Set(key, data); errTx != nil {
		return "", fmt.Errorf("failed to store deletion log record: %w", errTx)
	}

	return recordId, nil
}

func (r *lightcoordinatorstore) DeleteLogGetAfter(txn *badger.Txn, afterId string, limit uint32) (records []DeleteLogRecord, hasMore bool, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Apply default or max limit if needed
	if limit == 0 || limit > defaultDeleteLogLimit {
		limit = defaultDeleteLogLimit
	}

	// Add 1 to detect if there are more records
	fetchLimit := limit + 1

	// Determine start key based on afterId
	startKey, err := buildDeleteLogStartKey(afterId)
	if err != nil {
		return nil, false, err
	}

	// Prefetch the iterator values for better performance
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = true

	it := txn.NewIterator(opts)
	defer it.Close()

	// Pre-allocate the records slice with expected capacity
	collectedRecords := make([]DeleteLogRecord, 0, fetchLimit)
	prefixKey := buildKey(deleteLog, nil)

	// Collect records
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

		// Copy value since it's only valid within this iteration
		val, errCp := item.ValueCopy(nil)
		if errCp != nil {
			return nil, false, fmt.Errorf("failed to copy value: %w", errCp)
		}

		// Unmarshal protobuf
		pbRecord := &storepb.DeleteLogRecord{}
		if errUnm := proto.Unmarshal(val, pbRecord); errUnm != nil {
			return nil, false, fmt.Errorf("failed to unmarshal deletion log record: %w", errUnm)
		}

		// Convert to domain model
		record := DeleteLogRecord{
			Id:        pbRecord.GetId(),
			Timestamp: pbRecord.GetTimestamp(),
			SpaceId:   pbRecord.GetSpaceId(),
			FileGroup: pbRecord.GetFileGroup(),
			Status:    coordinatorproto.DeletionLogRecordStatus(pbRecord.GetStatus()),
		}

		collectedRecords = append(collectedRecords, record)
	}

	// Determine if there are more records than requested
	size := len(collectedRecords)
	if size > int(limit) {
		return collectedRecords[:limit], true, nil
	}

	return collectedRecords, false, nil
}

// buildDeleteLogStartKey creates the appropriate start key for DeleteLogGetAfter
// based on whether an afterId was provided
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

package lightcoordinatorstore

import (
	"context"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/coordinator/coordinatorproto"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/dgraph-io/badger/v4"
)

const (
	CName = "light.coordinator.store"
)

var log = logger.NewNamed(CName)

type StoreService interface {
	app.ComponentRunnable

	// Transaction operations
	TxView(f func(txn *badger.Txn) error) error
	TxUpdate(f func(txn *badger.Txn) error) error

	// ACL events
	AclEventAddLog(ctx context.Context, event AclEventLogEntry) (err error)
	AclEventGetAfter(ctx context.Context, identity, afterId string, limit uint32) (records []AclEventLogEntry, hasMore bool, err error)

	// Delete logs
	DeleteLogAdd(ctx context.Context, spaceId, fileGroup string, status DeleteLogStatus) (id string, err error)
	DeleteLogGetAfter(ctx context.Context, afterId string, limit uint32) (records []DeleteLogRecord, hasMore bool, err error)

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
	SpaceId   string
	FileGroup string
	Status    DeleteLogStatus
}

type DeleteLogStatus int32

const (
	StatusOk            = DeleteLogStatus(coordinatorproto.DeletionLogRecordStatus_Ok)
	StatusRemovePrepare = DeleteLogStatus(coordinatorproto.DeletionLogRecordStatus_RemovePrepare)
	StatusRemove        = DeleteLogStatus(coordinatorproto.DeletionLogRecordStatus_Remove)
)

type lightcoordinatorstore struct {
	srvCfg configService
}

type configService interface {
	app.Component
	GetNodeConf() nodeconf.Configuration
}

func New() *lightcoordinatorstore {
	return new(lightcoordinatorstore)
}

//
// App Component
//

func (r *lightcoordinatorstore) Init(a *app.App) error {
	log.Info("call Init")

	r.srvCfg = app.MustComponent[configService](a)

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

	return nil
}

func (r *lightcoordinatorstore) Close(ctx context.Context) error {
	log.Info("call Close")
	return nil
}

//
// Component methods
//

func (r *lightcoordinatorstore) TxUpdate() {
}

func (r *lightcoordinatorstore) TxView() {
}

func (r *lightcoordinatorstore) AclEventAdd() {
}

func (r *lightcoordinatorstore) AclEventAfter() {
}

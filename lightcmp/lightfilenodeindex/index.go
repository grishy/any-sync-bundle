package lightfilenodeindex

import (
	"context"
	"sync"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
)

// TODO: Implement index in memory and store in DB
// - blocks data
// - snapshot on index
// - log of request on top of index
//
// And have all index in memory, periodically dump index and remove old logs and snapshots

const (
	CName = "light.filenode.index"

	defaultLimitBytes = 1 << 40 // 1 TiB in bytes
)

var log = logger.NewNamed(CName)

type configService interface {
	app.Component

	GetFilenodeDefaultLimitBytes() uint64
}

type IndexService interface {
	app.ComponentRunnable

	GroupInfo(txn *badger.Txn, groupId string) (GroupInfo, error)
	SpaceInfo(txn *badger.Txn, key index.Key) (SpaceInfo, error)
}

// Interface types

type Key struct {
	GroupID string
	SpaceID string
}

type GroupInfo struct {
	GroupID           string
	UsageBytes        uint64
	CidsCount         uint64
	AccountLimitBytes uint64
	LimitBytes        uint64
	SpaceIds          []string
}

type SpaceInfo struct {
	SpaceID    string
	UsageBytes uint64
	CidsCount  uint64
	LimitBytes uint64
	FileCount  uint32
}

// Index

type Block struct {
	cid  cid.Cid // Unique content identifier
	size uint32  // Block size in bytes
	refs int32   // Number of files referencing this block
}

type File struct {
	cids   map[cid.Cid]*Block
	fileID string // Unique file identifier
}

type Space struct {
	spaceID string           // Unique identifier for the space
	files   map[string]*File // Files within this space (keyed by FileID)
}

type Group struct {
	groupID string            // Unique identifier for the group
	spaces  map[string]*Space // Spaces in the group (keyed by SpaceID)
}

// lightfileidex is the top-level structure representing the complete index.
type lightfileidex struct {
	cfgSrv configService

	sync.RWMutex
	defaultLimitBytes uint64
	groups            map[string]*Group // Maps GroupID to Group
}

func New() IndexService {
	idx := &lightfileidex{
		groups: make(map[string]*Group),
	}

	return idx
}

//
// App Component
//

func (i *lightfileidex) Init(a *app.App) error {
	log.Info("call Init")

	i.cfgSrv = app.MustComponent[configService](a)

	return nil
}

func (i *lightfileidex) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (i *lightfileidex) Run(ctx context.Context) error {
	i.defaultLimitBytes = i.cfgSrv.GetFilenodeDefaultLimitBytes()
	if i.defaultLimitBytes == 0 {
		i.defaultLimitBytes = defaultLimitBytes
	}

	return nil
}

func (i *lightfileidex) Close(_ context.Context) error {
	return nil
}

//
// Component methods
//

func (i *lightfileidex) Marshal() ([]byte, error) {
	i.RLock()
	defer i.RUnlock()

	return []byte{}, nil
}

func (i *lightfileidex) Unmarshal(data []byte) error {
	i.Lock()
	defer i.Unlock()

	return nil
}

func (i *lightfileidex) GroupInfo(txn *badger.Txn, groupId string) (GroupInfo, error) {
	// TODO implement me
	panic("implement me")
}

func (i *lightfileidex) SpaceInfo(txn *badger.Txn, key index.Key) (SpaceInfo, error) {
	// TODO implement me
	panic("implement me")
}

//go:generate moq -fmt gofumpt -rm -out index_mock.go . configService IndexService

package lightfilenodeindex

import (
	"context"
	"sync"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"

	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex/indexpb"
)

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

	GroupInfo(groupId string) GroupInfo
	SpaceInfo(key index.Key) SpaceInfo

	// TODO: Remove or replace
	HasCIDInSpace(spaceId string, k cid.Cid) bool
	HadCID(k cid.Cid) bool

	Modify(txn *badger.Txn, key index.Key, query *indexpb.Operation) error
}

// Interface types

type Key struct {
	GroupID string
	SpaceID string
}

type GroupInfo struct {
	UsageBytes        uint64
	CidsCount         uint64
	AccountLimitBytes uint64
	LimitBytes        uint64
	SpaceIds          []string
}

type SpaceInfo struct {
	UsageBytes uint64
	CidsCount  uint64
	LimitBytes uint64
	FileCount  uint32
}

type FileInfo struct {
	BytesUsage uint64
	CidsCount  uint64
}

// Index

type Block struct {
	cid  cid.Cid // Unique content identifier
	size uint32  // Block size in bytes
	refs int32   // Number of files referencing this block
}

type File struct {
	fileID string // Unique file identifier
	info   FileInfo
	cids   map[cid.Cid]*Block
}

type Space struct {
	spaceID string
	info    SpaceInfo
	files   map[string]*File // Files within this space (keyed by FileID)
}

type Group struct {
	groupID string
	info    GroupInfo
	spaces  map[string]*Space // Spaces in the group (keyed by SpaceID)
}

// lightfileidex is the top-level structure representing the complete index.
type lightfileidex struct {
	cfgSrv configService

	sync.RWMutex
	defaultLimitBytes uint64
	groups            map[string]*Group // Maps GroupID to Group
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

func (i *lightfileidex) GroupInfo(groupId string) GroupInfo {
	i.RLock()
	defer i.RUnlock()

	if group, ok := i.groups[groupId]; ok {
		return group.info
	}

	// Default group info
	defaultInfo := GroupInfo{
		UsageBytes: 0,
		CidsCount:  0,
		// TODO: Check in original code how they are computing AccountLimitBytes and LimitBytes
		AccountLimitBytes: i.defaultLimitBytes,
		LimitBytes:        i.defaultLimitBytes,
		SpaceIds:          []string{},
	}

	return defaultInfo
}

func (i *lightfileidex) SpaceInfo(key index.Key) SpaceInfo {
	i.RLock()
	defer i.RUnlock()

	// Default space info
	defaultInfo := SpaceInfo{
		UsageBytes: 0,
		CidsCount:  0,
		// TODO: Check in original code how they are computing LimitBytes
		LimitBytes: i.defaultLimitBytes,
		FileCount:  0,
	}

	group, ok := i.groups[key.GroupId]
	if !ok {
		return defaultInfo
	}

	space, ok := group.spaces[key.SpaceId]
	if ok {
		return space.info
	}

	return defaultInfo
}

func (i *lightfileidex) HasCIDInSpace(spaceId string, k cid.Cid) bool {
	// TODO implement me
	panic("implement me")
}

func (i *lightfileidex) HadCID(k cid.Cid) bool {
	// TODO implement me
	panic("implement me")
}

func New() IndexService {
	idx := &lightfileidex{
		groups: make(map[string]*Group),
	}

	return idx
}

func (i *lightfileidex) Modify(txn *badger.Txn, key index.Key, query *indexpb.Operation) error {
	i.Lock()
	defer i.Unlock()

	panic("implement me")
}

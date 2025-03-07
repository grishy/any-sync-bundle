//go:generate moq -fmt gofumpt -rm -out index_mock.go . configService IndexService

package lightfilenodeindex

import (
	"context"
	"sync"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/commonfile/fileproto"
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

	HasCIDInSpace(spaceId string, k cid.Cid) bool
	HadCID(k cid.Cid) bool

	GroupInfo(groupId string) GroupInfo

	SpaceInfo(key index.Key) SpaceInfo
	GetSpaceFiles(spaceId string) ([]string, error)

	GetFileInfo(spaceId string, fileIds ...string) ([]*fileproto.FileInfo, error)

	// Modify operation - the only method that can modify the index
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

func New() *lightfileidex {
	return &lightfileidex{}
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

	i.groups = make(map[string]*Group)

	return nil
}

func (i *lightfileidex) Close(_ context.Context) error {
	return nil
}

//
// Component methods - Read-only operations
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

// GetSpaceFiles returns all file IDs in a space
func (i *lightfileidex) GetSpaceFiles(spaceId string) ([]string, error) {
	i.RLock()
	defer i.RUnlock()

	// In a real implementation, this would query the index
	// For now, return a placeholder list of files
	return []string{"file1", "file2", "file3"}, nil
}

// GetFileInfo returns file information for the specified file IDs
func (i *lightfileidex) GetFileInfo(spaceId string, fileIds ...string) ([]*fileproto.FileInfo, error) {
	i.RLock()
	defer i.RUnlock()

	// Create response with file info for each requested file
	result := make([]*fileproto.FileInfo, 0, len(fileIds))

	for _, fileId := range fileIds {
		// In a real implementation, this would query the index for actual data
		// For now, return a placeholder fileinfo object
		info := &fileproto.FileInfo{
			FileId:     fileId,
			UsageBytes: 1000,
			CidsCount:  10,
		}
		result = append(result, info)
	}

	return result, nil
}

// HasCIDInSpace checks if a CID exists in the specified space
func (i *lightfileidex) HasCIDInSpace(spaceId string, k cid.Cid) (bool, error) {
	i.RLock()
	defer i.RUnlock()

	// In a real implementation, this would check if the CID exists in the space
	// For now, return false to indicate it doesn't exist
	return false, nil
}

// HadCID checks if a CID exists anywhere in the index
func (i *lightfileidex) HadCID(k cid.Cid) (bool, error) {
	i.RLock()
	defer i.RUnlock()

	// In a real implementation, this would check if the CID exists anywhere
	return false, nil
}

//
// The only method that can modify the index
//

func (i *lightfileidex) Modify(txn *badger.Txn, key index.Key, op *indexpb.Operation) error {
	i.Lock()
	defer i.Unlock()

	// TODO:

	return nil
}

//go:generate moq -fmt gofumpt -rm -out index_mock.go . configService IndexService

package lightfilenodeindex

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/commonfile/fileproto"
	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
	"google.golang.org/protobuf/proto"

	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex/indexpb"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

const (
	CName = "light.filenode.index"

	defaultLimitBytes = 1 << 40 // 1 TiB in bytes
)

var (
	// Type assertion
	_ IndexService = (*lightfileidex)(nil)

	log = logger.NewNamed(CName)

	ErrInvalidCID = errors.New("invalid cid")
)

type configService interface {
	app.Component

	GetFilenodeDefaultLimitBytes() uint64
}

// TODO: Avoid using badger.Txn directly
type IndexService interface {
	app.ComponentRunnable

	// Read-only operations

	HasCIDInSpace(key index.Key, k cid.Cid) bool
	HadCID(k cid.Cid) bool

	GroupInfo(groupId string) fileproto.AccountInfoResponse

	SpaceInfo(key index.Key) fileproto.SpaceInfoResponse
	SpaceFiles(key index.Key) []string

	FileInfo(key index.Key, fileIds ...string) []*fileproto.FileInfo

	// Modify operation - the only method that can modify the index
	Modify(txn *badger.Txn, key index.Key, query ...*indexpb.Operation) error
}

type block struct {
	size     uint32
	refCount int
}

type file struct {
	info   fileInfo
	blocks map[cid.Cid]struct{}
}

type fileInfo struct {
	BytesUsage uint64
	CidsCount  uint64
}

type space struct {
	info  spaceInfo
	files map[string]*file
}

type spaceInfo struct {
	UsageBytes uint64
	CidsCount  uint64
	LimitBytes uint64
	FileCount  uint32
}

type groupInfo struct {
	UsageBytes        uint64
	CidsCount         uint64
	AccountLimitBytes uint64
	LimitBytes        uint64
}

type group struct {
	info   groupInfo
	spaces map[string]*space
}

type lightfileidex struct {
	srvCfg   configService
	srvStore lightfilenodestore.StoreService

	defaultLimitBytes uint64

	sync.RWMutex
	groups     map[string]*group
	blocksLake map[cid.Cid]*block
}

func New() *lightfileidex {
	return &lightfileidex{}
}

//
// App Component
//

func (i *lightfileidex) Init(a *app.App) error {
	log.Info("call Init")

	i.srvCfg = app.MustComponent[configService](a)
	i.srvStore = app.MustComponent[lightfilenodestore.StoreService](a)

	return nil
}

func (i *lightfileidex) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (i *lightfileidex) Run(ctx context.Context) error {
	i.defaultLimitBytes = i.srvCfg.GetFilenodeDefaultLimitBytes()
	if i.defaultLimitBytes == 0 {
		i.defaultLimitBytes = defaultLimitBytes
	}

	i.groups = make(map[string]*group)
	i.blocksLake = make(map[cid.Cid]*block)

	return nil
}

func (i *lightfileidex) Close(_ context.Context) error {
	return nil
}

//
// Component methods
//

// ensureGroupAndSpace creates group and space if they don't exist
func (i *lightfileidex) ensureGroupAndSpace(groupId, spaceId string) (*group, *space) {
	grp, ok := i.groups[groupId]
	if !ok {
		grp = &group{
			info: groupInfo{
				UsageBytes:        0,
				CidsCount:         0,
				AccountLimitBytes: i.defaultLimitBytes,
				LimitBytes:        i.defaultLimitBytes,
			},
			spaces: make(map[string]*space),
		}
		i.groups[groupId] = grp
	}

	sp, ok := grp.spaces[spaceId]
	if !ok {
		sp = &space{
			info: spaceInfo{
				UsageBytes: 0,
				CidsCount:  0,
				LimitBytes: i.defaultLimitBytes,
				FileCount:  0,
			},
			files: make(map[string]*file),
		}
		grp.spaces[spaceId] = sp
	}

	return grp, sp
}

func (i *lightfileidex) GroupInfo(groupId string) fileproto.AccountInfoResponse {
	i.RLock()
	defer i.RUnlock()

	// Set default response
	response := fileproto.AccountInfoResponse{
		LimitBytes:        i.defaultLimitBytes,
		AccountLimitBytes: i.defaultLimitBytes,
		Spaces:            []*fileproto.SpaceInfoResponse{},
	}

	group, ok := i.groups[groupId]
	if !ok {
		return response
	}

	// Populate response from group info
	response.LimitBytes = group.info.LimitBytes
	response.AccountLimitBytes = group.info.AccountLimitBytes
	response.TotalUsageBytes = group.info.UsageBytes
	response.TotalCidsCount = group.info.CidsCount

	// Add spaces info
	for spaceId, sp := range group.spaces {
		spaceInfo := &fileproto.SpaceInfoResponse{
			SpaceId:         spaceId,
			LimitBytes:      sp.info.LimitBytes,
			TotalUsageBytes: sp.info.UsageBytes,
			CidsCount:       uint64(sp.info.CidsCount),
			FilesCount:      uint64(sp.info.FileCount),
			SpaceUsageBytes: sp.info.UsageBytes,
		}
		response.Spaces = append(response.Spaces, spaceInfo)
	}

	return response
}

// SpaceInfo returns information about a space
func (i *lightfileidex) SpaceInfo(key index.Key) fileproto.SpaceInfoResponse {
	i.RLock()
	defer i.RUnlock()

	// Default response
	response := fileproto.SpaceInfoResponse{
		LimitBytes: i.defaultLimitBytes,
		SpaceId:    key.SpaceId,
	}

	grp, ok := i.groups[key.GroupId]
	if !ok {
		return response
	}

	sp, ok := grp.spaces[key.SpaceId]
	if !ok {
		return response
	}

	// Populate from space info
	response.LimitBytes = sp.info.LimitBytes
	response.TotalUsageBytes = sp.info.UsageBytes
	response.SpaceUsageBytes = sp.info.UsageBytes
	response.CidsCount = uint64(sp.info.CidsCount)
	response.FilesCount = uint64(sp.info.FileCount)

	return response
}

// SpaceFiles returns all file IDs in a space
func (i *lightfileidex) SpaceFiles(key index.Key) []string {
	i.RLock()
	defer i.RUnlock()

	grp, ok := i.groups[key.GroupId]
	if !ok {
		return []string{}
	}

	sp, ok := grp.spaces[key.SpaceId]
	if !ok {
		return []string{}
	}

	fileIds := make([]string, 0, len(sp.files))
	for fileId := range sp.files {
		fileIds = append(fileIds, fileId)
	}

	return fileIds
}

// FileInfo returns information about specified files
func (i *lightfileidex) FileInfo(key index.Key, fileIds ...string) []*fileproto.FileInfo {
	i.RLock()
	defer i.RUnlock()

	group, ok := i.groups[key.GroupId]
	if !ok {
		return []*fileproto.FileInfo{}
	}

	space, ok := group.spaces[key.SpaceId]
	if !ok {
		return []*fileproto.FileInfo{}
	}

	result := make([]*fileproto.FileInfo, 0, len(fileIds))
	for _, fileId := range fileIds {
		file, ok := space.files[fileId]
		if !ok {
			result = append(result, &fileproto.FileInfo{
				FileId:     fileId,
				UsageBytes: 0,
				CidsCount:  0,
			})
			continue
		}

		info := &fileproto.FileInfo{
			FileId:     fileId,
			UsageBytes: file.info.BytesUsage,
			CidsCount:  uint32(file.info.CidsCount),
		}
		result = append(result, info)
	}

	return result
}

// HasCIDInSpace checks if a CID exists in a specific space
func (i *lightfileidex) HasCIDInSpace(key index.Key, k cid.Cid) bool {
	i.RLock()
	defer i.RUnlock()

	group, ok := i.groups[key.GroupId]
	if !ok {
		return false
	}

	space, ok := group.spaces[key.SpaceId]
	if !ok {
		return false
	}

	cidStr := k.String()
	return i.findBlock(space, cidStr) != nil
}

// HadCID checks if a CID exists anywhere in the index
func (i *lightfileidex) HadCID(k cid.Cid) bool {
	i.RLock()
	defer i.RUnlock()

	cidStr := k.String()

	// Check all groups and spaces for the CID
	for _, group := range i.groups {
		for _, space := range group.spaces {
			if i.findBlock(space, cidStr) != nil {
				return true
			}
		}
	}

	return false
}

// Modify applies operations to modify the index
func (i *lightfileidex) Modify(txn *badger.Txn, key index.Key, operations ...*indexpb.Operation) error {
	if len(operations) == 0 {
		return nil
	}

	i.Lock()
	defer i.Unlock()

	if err := i.recordOperations(txn, key, operations); err != nil {
		return fmt.Errorf("failed to record operations: %w", err)
	}

	// Ensure group and space exist
	group, space := i.ensureGroupAndSpace(key.GroupId, key.SpaceId)

	for _, op := range operations {
		if err := i.processOperation(group, space, op); err != nil {
			return fmt.Errorf("failed to process operation: %w", err)
		}
	}

	i.updateSpaceStats(space)
	i.updateGroupStats(group)

	return nil
}

// processOperation handles a single operation
func (i *lightfileidex) processOperation(group *group, space *space, op *indexpb.Operation) error {
	switch {
	case op.GetBindFile() != nil:
		return i.handleBindFile(group, space, op.GetBindFile())
	case op.GetDeleteFile() != nil:
		return i.handleDeleteFile(group, space, op.GetDeleteFile())
	case op.GetAccountLimitSet() != nil:
		return i.handleAccountLimitSet(group, op.GetAccountLimitSet())
	case op.GetSpaceLimitSet() != nil:
		return i.handleSpaceLimitSet(space, op.GetSpaceLimitSet())
	case op.GetCidAdd() != nil:
		return i.handleCIDAdd(space, op.GetCidAdd())
	default:
		return errors.New("unsupported operation")
	}
}

// handleBindFile associates CIDs with a file
func (i *lightfileidex) handleBindFile(group *group, space *space, op *indexpb.FileBindOperation) error {
	fileId := op.GetFileId()

	// Create file if it doesn't exist
	f, ok := space.files[fileId]
	if !ok {
		f = &file{
			info: fileInfo{
				BytesUsage: 0,
				CidsCount:  0,
			},
			blocks: make(map[cid.Cid]struct{}),
		}
		space.files[fileId] = f
		space.info.FileCount++
	}

	// Process CIDs
	for _, cidStr := range op.GetCids() {
		c, err := cid.Parse(cidStr)
		if err != nil {
			// log.Warn("Invalid CID in bind operation:", err)
			return ErrInvalidCID
		}

		// Check if this file already has this block
		if _, hasBlock := f.blocks[c]; hasBlock {
			continue // Skip if file already has this block
		}

		// Check if block already exists in any file in the space
		existingBlock := i.findBlock(space, cidStr)

		var b *block
		if existingBlock != nil {
			// Reuse existing block and increment its reference count
			b = existingBlock
			b.refCount++
		} else {
			// Create new block with initial refCount of 1
			b = &block{
				size:     0, // Will be updated by CidAddOperation if needed
				refCount: 1,
			}
		}

		// Add block to file
		f.blocks[c] = struct{}{}
		// f.info.CidsCount++
		// f.info.BytesUsage += uint64(b.size)
	}

	return nil
}

// handleDeleteFile removes files from a space
func (i *lightfileidex) handleDeleteFile(group *group, space *space, op *indexpb.FileDeleteOperation) error {
	for _, fileId := range op.GetFileIds() {
		file, ok := space.files[fileId]
		if !ok {
			continue // File not found, nothing to delete
		}

		// Remove file's usage from space totals
		space.info.UsageBytes -= file.info.BytesUsage

		// Collect blocks for reference count check and decrement their refCounts
		for cidStr := range file.blocks {
			c, err := cid.Parse(cidStr)
			if err != nil {
				// log.Warn("Invalid CID in bind operation:", err)
				return ErrInvalidCID
			}
			block := i.blocksLake[c]
			block.refCount--
		}

		// Remove the file
		delete(space.files, fileId)
	}

	return nil
}

// handleAccountLimitSet sets the account limit for a group
func (i *lightfileidex) handleAccountLimitSet(group *group, op *indexpb.AccountLimitSetOperation) error {
	group.info.AccountLimitBytes = op.GetLimit()
	group.info.LimitBytes = op.GetLimit()

	return nil
}

// handleSpaceLimitSet sets the space limit
func (i *lightfileidex) handleSpaceLimitSet(space *space, op *indexpb.SpaceLimitSetOperation) error {
	space.info.LimitBytes = op.GetLimit()

	return nil
}

// handleCIDAdd adds or updates a CID with size information
func (i *lightfileidex) handleCIDAdd(space *space, op *indexpb.CidAddOperation) error {
	cidStr := op.GetCid()
	c, err := cid.Parse(cidStr)
	if err != nil {
		return ErrInvalidCID
	}

	size := uint32(op.GetDataSize())

	// Find all files that reference this CID and update their blocks
	existingBlock := i.findBlock(space, cidStr)

	if existingBlock != nil {
		// Update size if the new size is larger
		if existingBlock.size < size {
			// Calculate size difference
			sizeDiff := uint64(size - existingBlock.size)

			// Update size in all files that contain this block
			for _, file := range space.files {
				if _, hasBlock := file.blocks[c]; hasBlock {
					file.info.BytesUsage += sizeDiff
				}
			}

			// Update the block size
			existingBlock.size = size
		}
	} else {
		// Create new block but don't add it to any file yet
		// It will be associated with files through BindFile operations
		log.Debug("CID not referenced by any file yet")
	}

	return nil
}

// updateSpaceStats recalculates space statistics
func (i *lightfileidex) updateSpaceStats(space *space) {
	// Reset space usage stats
	space.info.UsageBytes = 0

	// Count unique blocks across all files
	uniqueBlocks := i.countUniqueBlocks(space)
	space.info.CidsCount = uint64(uniqueBlocks)

	// Sum file usages
	for _, file := range space.files {
		space.info.UsageBytes += file.info.BytesUsage
	}
}

// updateGroupStats aggregates statistics from spaces to group
func (i *lightfileidex) updateGroupStats(group *group) {
	// Reset group usage stats
	group.info.UsageBytes = 0
	group.info.CidsCount = 0

	// Aggregate from all spaces
	for _, space := range group.spaces {
		group.info.UsageBytes += space.info.UsageBytes
		group.info.CidsCount += space.info.CidsCount
	}
}

// findBlock searches for a block by CID string in a space
// Returns the block if found, nil otherwise
func (i *lightfileidex) findBlock(space *space, cidStr string) *block {
	// for _, file := range space.files {
	// 	if block, found := file.blocks[cidStr]; found {
	// 		return block
	// 	}
	// }
	return nil
}

// countUniqueBlocks returns a map of all unique blocks in a space
func (i *lightfileidex) countUniqueBlocks(space *space) int {
	uniqueBlocks := make(map[cid.Cid]struct{})
	for _, file := range space.files {
		for c := range file.blocks {
			uniqueBlocks[c] = struct{}{}
		}
	}
	return len(uniqueBlocks)
}

// recordOperations persists an operation to the transaction log
func (i *lightfileidex) recordOperations(txn *badger.Txn, key index.Key, ops []*indexpb.Operation) error {
	keyBuilder := indexpb.Key_builder{
		GroupId: &key.GroupId,
		SpaceId: &key.SpaceId,
	}
	protoKey := keyBuilder.Build()

	timestamp := time.Now().Unix()
	walRecordBuilder := indexpb.WALRecord_builder{
		Timestamp: &timestamp,
		Key:       protoKey,
		Ops:       ops,
	}
	walRecord := walRecordBuilder.Build()

	data, err := proto.Marshal(walRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal WAL record: %w", err)
	}

	if errPush := i.srvStore.PushIndexLog(txn, data); errPush != nil {
		return fmt.Errorf("failed to push WAL record: %w", errPush)
	}

	return nil
}

//go:generate moq -fmt gofumpt -rm -out index_mock.go . configService IndexService

// Only public methods should lock the mutex
// All private methods assume the mutex is already locked
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
	bytesUsage uint64
	cidsCount  uint32
}

type space struct {
	info  spaceInfo
	files map[string]*file
}

type spaceInfo struct {
	usageBytes uint64
	cidsCount  uint64
	limitBytes uint64
	fileCount  uint64
}

type groupInfo struct {
	usageBytes        uint64
	cidsCount         uint64
	accountLimitBytes uint64
	limitBytes        uint64
}

type group struct {
	info   groupInfo
	spaces map[string]*space
}

// lightfileidex is the top-level structure representing the complete index
type lightfileidex struct {
	srvCfg   configService
	srvStore lightfilenodestore.StoreService

	// TODO: Store all objects without limit and add in runtime to by dymanic if we will change the limit later
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

	for _, file := range space.files {
		if _, hasBlock := file.blocks[k]; hasBlock {
			return true
		}
	}

	return false
}

// HadCID checks if a CID exists anywhere in the index
func (i *lightfileidex) HadCID(k cid.Cid) bool {
	i.RLock()
	defer i.RUnlock()

	_, exist := i.blocksLake[k]
	return exist
}

func (i *lightfileidex) GroupInfo(groupId string) fileproto.AccountInfoResponse {
	i.RLock()
	defer i.RUnlock()

	return i.getGroupInfo(groupId)
}

func (i *lightfileidex) getGroupInfo(groupId string) fileproto.AccountInfoResponse {
	group := i.getGroupEntry(groupId, false)

	resp := fileproto.AccountInfoResponse{
		LimitBytes:        group.info.limitBytes,
		TotalUsageBytes:   group.info.usageBytes,
		TotalCidsCount:    group.info.cidsCount,
		AccountLimitBytes: group.info.accountLimitBytes,
	}

	if resp.AccountLimitBytes == 0 {
		resp.AccountLimitBytes = i.defaultLimitBytes
		resp.LimitBytes = i.defaultLimitBytes
	}

	resp.Spaces = make([]*fileproto.SpaceInfoResponse, 0, len(group.spaces))
	for spaceId := range group.spaces {
		spInfo := i.getSpaceInfo(group, spaceId)
		resp.Spaces = append(resp.Spaces, &spInfo)
	}

	return resp
}

func (i *lightfileidex) getGroupEntry(groupId string, saveCreated bool) *group {
	g := i.groups[groupId]
	if g == nil {
		g = &group{
			info: groupInfo{
				usageBytes:        0,
				cidsCount:         0,
				accountLimitBytes: 0,
				limitBytes:        0,
			},
			spaces: make(map[string]*space),
		}

		if saveCreated {
			i.groups[groupId] = g
		}
	}

	return g
}

func (i *lightfileidex) SpaceInfo(key index.Key) fileproto.SpaceInfoResponse {
	i.RLock()
	defer i.RUnlock()

	grp := i.getGroupEntry(key.GroupId, false)
	spInfo := i.getSpaceInfo(grp, key.SpaceId)

	return spInfo
}

func (i *lightfileidex) getSpaceInfo(group *group, spaceId string) fileproto.SpaceInfoResponse {
	s := i.getSpaceEntry(group, spaceId, false)

	resp := fileproto.SpaceInfoResponse{
		SpaceId:         spaceId,
		TotalUsageBytes: s.info.usageBytes,
		LimitBytes:      s.info.limitBytes,
		CidsCount:       s.info.cidsCount,
		FilesCount:      s.info.fileCount,
		SpaceUsageBytes: s.info.usageBytes,
	}

	if resp.LimitBytes == 0 {
		resp.TotalUsageBytes = group.info.usageBytes
		resp.LimitBytes = group.info.limitBytes
	}

	return resp
}

func (i *lightfileidex) getSpaceEntry(group *group, spaceId string, saveCreated bool) *space {
	s := group.spaces[spaceId]
	if s == nil {
		s = &space{
			info: spaceInfo{
				limitBytes: 0,
				usageBytes: 0,
				cidsCount:  0,
				fileCount:  0,
			},
			files: make(map[string]*file),
		}

		if saveCreated {
			group.spaces[spaceId] = s
		}
	}

	return s
}

func (i *lightfileidex) SpaceFiles(key index.Key) []string {
	i.RLock()
	defer i.RUnlock()

	grp := i.getGroupEntry(key.GroupId, false)
	sp := i.getSpaceEntry(grp, key.SpaceId, false)

	fileIds := make([]string, 0, len(sp.files))
	for fileId := range sp.files {
		fileIds = append(fileIds, fileId)
	}

	return fileIds
}

func (i *lightfileidex) FileInfo(key index.Key, fileIds ...string) []*fileproto.FileInfo {
	i.RLock()
	defer i.RUnlock()

	grp := i.getGroupEntry(key.GroupId, false)
	sp := i.getSpaceEntry(grp, key.SpaceId, false)

	result := make([]*fileproto.FileInfo, 0, len(fileIds))
	for _, fileId := range fileIds {
		f := i.getFileEntry(sp, fileId, false)
		result = append(result, &fileproto.FileInfo{
			FileId:     fileId,
			UsageBytes: f.info.bytesUsage,
			CidsCount:  f.info.cidsCount,
		})
	}

	return result
}

func (i *lightfileidex) getFileEntry(space *space, fileId string, saveCreated bool) *file {
	f := space.files[fileId]
	if f == nil {
		f = &file{
			info: fileInfo{
				bytesUsage: 0,
				cidsCount:  0,
			},
			blocks: make(map[cid.Cid]struct{}),
		}

		if saveCreated {
			space.files[fileId] = f
		}
	}

	return f
}

// Modify applies operations to modify the index
func (i *lightfileidex) Modify(txn *badger.Txn, key index.Key, operations ...*indexpb.Operation) (err error) {
	if len(operations) == 0 {
		return nil
	}

	i.Lock()
	defer func() {
		if errSave := i.recordOperations(txn, key, operations); errSave != nil {
			err = fmt.Errorf("failed to record operations: %w", errSave)
		}

		i.Unlock()
	}()

	group := i.getGroupEntry(key.GroupId, true)
	space := i.getSpaceEntry(group, key.SpaceId, true)

	for _, op := range operations {
		if err := i.processOperation(group, space, op); err != nil {
			return fmt.Errorf("failed to process operation: %w", err)
		}
	}

	// TODO: Later, better to calculate this in place to avoid double iteration
	i.updateSpaceStats(space)
	i.updateGroupStats(group)

	return nil
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

	data, errMarshal := proto.Marshal(walRecord)
	if errMarshal != nil {
		return fmt.Errorf("failed to marshal WAL record: %w", errMarshal)
	}

	if errPush := i.srvStore.PushIndexLog(txn, data); errPush != nil {
		return fmt.Errorf("failed to push WAL record: %w", errPush)
	}

	return nil
}

// processOperation handles a single operation
func (i *lightfileidex) processOperation(group *group, space *space, op *indexpb.Operation) error {
	// TODO: Avoid error from handler
	switch {
	case op.GetCidAdd() != nil:
		return i.handleCIDAdd(space, op.GetCidAdd())
	case op.GetBindFile() != nil:
		return i.handleBindFile(space, op.GetBindFile())
	case op.GetDeleteFile() != nil:
		return i.handleDeleteFile(space, op.GetDeleteFile())
	case op.GetAccountLimitSet() != nil:
		return i.handleAccountLimitSet(group, op.GetAccountLimitSet())
	case op.GetSpaceLimitSet() != nil:
		return i.handleSpaceLimitSet(space, op.GetSpaceLimitSet())
	default:
		return errors.New("unsupported operation")
	}
}

// handleCIDAdd adds or updates a CID with size information
func (i *lightfileidex) handleCIDAdd(space *space, op *indexpb.CidAddOperation) error {
	c, err := cid.Parse(op.GetCid())
	if err != nil {
		return ErrInvalidCID
	}

	size := uint32(op.GetDataSize())

	// Find all files that reference this CID and update their blocks
	existingBlock := i.findBlock(space, c)

	if existingBlock != nil {
		// Update size if the new size is larger
		if existingBlock.size < size {
			// Calculate size difference
			sizeDiff := uint64(size - existingBlock.size)

			// Update size in all files that contain this block
			for _, file := range space.files {
				if _, hasBlock := file.blocks[c]; hasBlock {
					file.info.bytesUsage += sizeDiff
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

// handleBindFile associates CIDs with a file
func (i *lightfileidex) handleBindFile(space *space, op *indexpb.FileBindOperation) error {
	fileId := op.GetFileId()

	// Create file if it doesn't exist
	f, ok := space.files[fileId]
	if !ok {
		f = &file{
			info: fileInfo{
				bytesUsage: 0,
				cidsCount:  0,
			},
			blocks: make(map[cid.Cid]struct{}),
		}
		space.files[fileId] = f
		space.info.fileCount++
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
		existingBlock := i.findBlock(space, c)

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
func (i *lightfileidex) handleDeleteFile(space *space, op *indexpb.FileDeleteOperation) error {
	for _, fileId := range op.GetFileIds() {
		f, ok := space.files[fileId]
		if !ok {
			continue // File not found, nothing to delete
		}

		// Remove file's usage from space totals
		space.info.usageBytes -= f.info.bytesUsage

		// Collect blocks for reference count check and decrement their refCounts
		for c := range f.blocks {
			block := i.blocksLake[c]
			if block != nil {
				block.refCount--
			}
		}

		// Remove the file
		delete(space.files, fileId)
		space.info.fileCount--
	}

	return nil
}

// handleAccountLimitSet sets the account limit for a group
func (i *lightfileidex) handleAccountLimitSet(group *group, op *indexpb.AccountLimitSetOperation) error {
	group.info.accountLimitBytes = op.GetLimit()
	group.info.limitBytes = op.GetLimit()

	return nil
}

// handleSpaceLimitSet sets the space limit
func (i *lightfileidex) handleSpaceLimitSet(space *space, op *indexpb.SpaceLimitSetOperation) error {
	space.info.limitBytes = op.GetLimit()

	return nil
}

// updateSpaceStats recalculates space statistics
func (i *lightfileidex) updateSpaceStats(space *space) {
	// Reset space usage stats
	space.info.usageBytes = 0

	// Count unique blocks across all files
	uniqueBlocks := i.countUniqueBlocks(space)
	space.info.cidsCount = uint64(uniqueBlocks)

	// Sum file usages
	for _, file := range space.files {
		space.info.usageBytes += file.info.bytesUsage
	}
}

// updateGroupStats aggregates statistics from spaces to group
func (i *lightfileidex) updateGroupStats(group *group) {
	// Reset group usage stats
	group.info.usageBytes = 0
	group.info.cidsCount = 0

	// Aggregate from all spaces
	for _, space := range group.spaces {
		group.info.usageBytes += space.info.usageBytes
		group.info.cidsCount += space.info.cidsCount
	}
}

func (i *lightfileidex) findBlock(space *space, k cid.Cid) *block {
	for _, file := range space.files {
		if _, found := file.blocks[k]; found {
			return i.blocksLake[k]
		}
	}

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

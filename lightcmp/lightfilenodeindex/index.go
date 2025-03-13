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
	"go.uber.org/zap"
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
	_ IndexService = (*lightfileindex)(nil)

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
	createTime time.Time
	updateTime time.Time
	size       uint32
	refCount   int
}

type file struct {
	info   fileInfo
	blocks map[cid.Cid]struct{}
}

type fileInfo struct {
	createTime time.Time
	bytesUsage uint64
	cidsCount  uint32
}

type space struct {
	info  spaceInfo
	files map[string]*file
}

type spaceInfo struct {
	createTime time.Time
	usageBytes uint64
	cidsCount  uint64
	limitBytes uint64
	fileCount  uint64
}

type groupInfo struct {
	createTime        time.Time
	usageBytes        uint64
	cidsCount         uint64
	accountLimitBytes uint64
	limitBytes        uint64
}

type group struct {
	info   groupInfo
	spaces map[string]*space
}

type lightfileindex struct {
	srvCfg   configService
	srvStore lightfilenodestore.StoreService

	// TODO: Store all objects without limit and add in runtime to by dynamic if we will change the limit later
	defaultLimitBytes uint64

	sync.RWMutex
	groups     map[string]*group
	blocksLake map[cid.Cid]*block
}

func New() *lightfileindex {
	return &lightfileindex{}
}

//
// App Component
//

func (i *lightfileindex) Init(a *app.App) error {
	log.Info("call Init")

	i.srvCfg = app.MustComponent[configService](a)
	i.srvStore = app.MustComponent[lightfilenodestore.StoreService](a)

	return nil
}

func (i *lightfileindex) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (i *lightfileindex) Run(ctx context.Context) error {
	i.defaultLimitBytes = i.srvCfg.GetFilenodeDefaultLimitBytes()
	if i.defaultLimitBytes == 0 {
		i.defaultLimitBytes = defaultLimitBytes
	}

	i.groups = make(map[string]*group)
	i.blocksLake = make(map[cid.Cid]*block)

	return nil
}

func (i *lightfileindex) Close(_ context.Context) error {
	return nil
}

//
// Component methods
//

// HasCIDInSpace checks if a CID exists in a specific space
func (i *lightfileindex) HasCIDInSpace(key index.Key, k cid.Cid) bool {
	i.RLock()
	defer i.RUnlock()

	grp, ok := i.groups[key.GroupId]
	if !ok {
		return false
	}

	sps, ok := grp.spaces[key.SpaceId]
	if !ok {
		return false
	}

	for _, f := range sps.files {
		if _, hasBlock := f.blocks[k]; hasBlock {
			return true
		}
	}

	return false
}

// HadCID checks if a CID exists anywhere in the index
func (i *lightfileindex) HadCID(k cid.Cid) bool {
	i.RLock()
	defer i.RUnlock()

	_, exist := i.blocksLake[k]
	return exist
}

func (i *lightfileindex) GroupInfo(groupId string) fileproto.AccountInfoResponse {
	i.RLock()
	defer i.RUnlock()

	return i.getGroupInfo(groupId)
}

func (i *lightfileindex) getGroupInfo(groupId string) fileproto.AccountInfoResponse {
	grp, _ := i.getGroupEntry(groupId, false)

	resp := fileproto.AccountInfoResponse{
		LimitBytes:        grp.info.limitBytes,
		TotalUsageBytes:   grp.info.usageBytes,
		TotalCidsCount:    grp.info.cidsCount,
		AccountLimitBytes: grp.info.accountLimitBytes,
	}

	if resp.AccountLimitBytes == 0 {
		resp.AccountLimitBytes = i.defaultLimitBytes
		resp.LimitBytes = i.defaultLimitBytes
	}

	resp.Spaces = make([]*fileproto.SpaceInfoResponse, 0, len(grp.spaces))
	for spaceId := range grp.spaces {
		spInfo := i.getSpaceInfo(grp, spaceId)
		resp.Spaces = append(resp.Spaces, &spInfo)
	}

	return resp
}

func (i *lightfileindex) getGroupEntry(groupId string, saveCreated bool) (g *group, created bool) {
	g = i.groups[groupId]
	if g == nil {
		created = true
		g = &group{
			info: groupInfo{
				createTime:        time.Now(),
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

	return
}

func (i *lightfileindex) SpaceInfo(key index.Key) fileproto.SpaceInfoResponse {
	i.RLock()
	defer i.RUnlock()

	grp, _ := i.getGroupEntry(key.GroupId, false)
	spInfo := i.getSpaceInfo(grp, key.SpaceId)

	return spInfo
}

func (i *lightfileindex) getSpaceInfo(group *group, spaceId string) fileproto.SpaceInfoResponse {
	s, _ := i.getSpaceEntry(group, spaceId, false)

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

func (i *lightfileindex) getSpaceEntry(group *group, spaceId string, saveCreated bool) (s *space, created bool) {
	s = group.spaces[spaceId]
	if s == nil {
		created = true
		s = &space{
			info: spaceInfo{
				createTime: time.Now(),
				usageBytes: 0,
				cidsCount:  0,
				limitBytes: 0,
				fileCount:  0,
			},
			files: make(map[string]*file),
		}

		if saveCreated {
			group.spaces[spaceId] = s
		}
	}

	return
}

func (i *lightfileindex) SpaceFiles(key index.Key) []string {
	i.RLock()
	defer i.RUnlock()

	grp, _ := i.getGroupEntry(key.GroupId, false)
	sp, _ := i.getSpaceEntry(grp, key.SpaceId, false)

	fileIds := make([]string, 0, len(sp.files))
	for fileId := range sp.files {
		fileIds = append(fileIds, fileId)
	}

	return fileIds
}

func (i *lightfileindex) FileInfo(key index.Key, fileIds ...string) []*fileproto.FileInfo {
	i.RLock()
	defer i.RUnlock()

	grp, _ := i.getGroupEntry(key.GroupId, false)
	sp, _ := i.getSpaceEntry(grp, key.SpaceId, false)

	result := make([]*fileproto.FileInfo, 0, len(fileIds))
	for _, fileId := range fileIds {
		f, _ := i.getFileEntry(sp, fileId, false)
		result = append(result, &fileproto.FileInfo{
			FileId:     fileId,
			UsageBytes: f.info.bytesUsage,
			CidsCount:  f.info.cidsCount,
		})
	}

	return result
}

func (i *lightfileindex) getFileEntry(space *space, fileId string, saveCreated bool) (f *file, created bool) {
	f = space.files[fileId]
	if f == nil {
		created = true
		f = &file{
			info: fileInfo{
				createTime: time.Now(),
				bytesUsage: 0,
				cidsCount:  0,
			},
			blocks: make(map[cid.Cid]struct{}),
		}

		if saveCreated {
			space.files[fileId] = f
		}
	}

	return
}

// Modify applies operations to modify the index
func (i *lightfileindex) Modify(txn *badger.Txn, key index.Key, operations ...*indexpb.Operation) (err error) {
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

	grp, _ := i.getGroupEntry(key.GroupId, true)
	sps, _ := i.getSpaceEntry(grp, key.SpaceId, true)

	for _, op := range operations {
		if errProc := i.processOperation(grp, sps, op); errProc != nil {
			return fmt.Errorf("failed to process operation: %w", errProc)
		}
	}

	// TODO: Later, better to calculate this in place to avoid double iteration
	i.updateSpaceStats(sps)
	i.updateGroupStats(grp)

	return nil
}

// recordOperations persists an operation to the transaction log
func (i *lightfileindex) recordOperations(txn *badger.Txn, key index.Key, ops []*indexpb.Operation) error {
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

func (i *lightfileindex) processOperation(group *group, space *space, op *indexpb.Operation) error {
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

func (i *lightfileindex) handleCIDAdd(space *space, op *indexpb.CidAddOperation) error {
	c, err := cid.Parse(op.GetCid())
	if err != nil {
		return ErrInvalidCID
	}

	size := uint32(op.GetDataSize())
	b, exists := i.blocksLake[c]
	if !exists {
		b = &block{
			createTime: time.Now(),
			updateTime: time.Now(),
			size:       size,
			refCount:   0,
		}
		i.blocksLake[c] = b
	} else if b.size != size {
		log.Warn("block size mismatch",
			zap.String("cid", c.String()),
			zap.Uint32("expected", size),
			zap.Uint32("actual", b.size),
		)
		return fmt.Errorf("block size mismatch: %d != %d", b.size, size)
	}

	fileId := op.GetFileId()
	fileEntry, created := i.getFileEntry(space, fileId, true)

	if created {
		space.info.fileCount++
	}

	_, hadBlock := fileEntry.blocks[c]
	if !hadBlock {
		fileEntry.blocks[c] = struct{}{}
		fileEntry.info.cidsCount++
		fileEntry.info.bytesUsage += uint64(b.size)
		b.refCount++
	}

	return nil
}

// handleBindFile associates CIDs with a file
func (i *lightfileindex) handleBindFile(space *space, op *indexpb.FileBindOperation) error {
	fileId := op.GetFileId()

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

	for _, cidStr := range op.GetCids() {
		c, err := cid.Parse(cidStr)
		if err != nil {
			return ErrInvalidCID
		}

		if _, hasBlock := f.blocks[c]; hasBlock {
			continue
		}

		b, exists := i.blocksLake[c]
		if !exists {
			b = &block{
				size:     0,
				refCount: 1,
			}
			i.blocksLake[c] = b
		} else {
			b.refCount++
		}

		f.blocks[c] = struct{}{}
		f.info.cidsCount++
		f.info.bytesUsage += uint64(b.size)
	}

	return nil
}

// handleDeleteFile removes files from a space
func (i *lightfileindex) handleDeleteFile(space *space, op *indexpb.FileDeleteOperation) error {
	for _, fileId := range op.GetFileIds() {
		f, ok := space.files[fileId]
		if !ok {
			continue
		}

		space.info.usageBytes -= f.info.bytesUsage
		for c := range f.blocks {
			blk := i.blocksLake[c]
			if blk != nil {
				blk.refCount--
			}
		}

		delete(space.files, fileId)
		space.info.fileCount--
	}

	return nil
}

// handleAccountLimitSet sets the account limit for a group
func (i *lightfileindex) handleAccountLimitSet(group *group, op *indexpb.AccountLimitSetOperation) error {
	group.info.accountLimitBytes = op.GetLimit()
	group.info.limitBytes = op.GetLimit()

	return nil
}

// handleSpaceLimitSet sets the space limit
func (i *lightfileindex) handleSpaceLimitSet(space *space, op *indexpb.SpaceLimitSetOperation) error {
	space.info.limitBytes = op.GetLimit()

	return nil
}

// updateSpaceStats recalculates space statistics
func (i *lightfileindex) updateSpaceStats(space *space) {
	space.info.usageBytes = 0
	space.info.cidsCount = 0

	uniqueBlocks := make(map[cid.Cid]struct{})

	for _, f := range space.files {
		f.info.bytesUsage = 0
		f.info.cidsCount = uint32(len(f.blocks))

		for c := range f.blocks {
			if b, ok := i.blocksLake[c]; ok {
				uniqueBlocks[c] = struct{}{}
				f.info.bytesUsage += uint64(b.size)
			}
		}

		space.info.usageBytes += f.info.bytesUsage
	}

	space.info.cidsCount = uint64(len(uniqueBlocks))
}

// updateGroupStats aggregates statistics from spaces to group
func (i *lightfileindex) updateGroupStats(grp *group) {
	grp.info.usageBytes = 0
	grp.info.cidsCount = 0

	for _, sps := range grp.spaces {
		grp.info.usageBytes += sps.info.usageBytes
		grp.info.cidsCount += sps.info.cidsCount
	}
}

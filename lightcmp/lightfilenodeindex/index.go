//go:generate moq -fmt gofumpt -rm -out index_mock.go . configService IndexService
package lightfilenodeindex

// Only public methods should lock the mutex
// All private methods assume the mutex is already locked

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

// TODO: Implement restore on start from badger
// TODO: Implement log compaction

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

type cidBlock struct {
	size     uint32
	refCount int
}

type file struct {
	sizeBytes uint64
	cids      map[cid.Cid]struct{}
}

type space struct {
	sizeBytes uint64
	files     map[string]*file
	cids      map[cid.Cid]struct{}
}

func (s *space) putCid(c cid.Cid, blockSize uint32) {
	if _, ok := s.cids[c]; ok {
		return
	}

	s.cids[c] = struct{}{}
	s.sizeBytes += uint64(blockSize)
}

type group struct {
	sizeBytes uint64
	spaces    map[string]*space
	cids      map[cid.Cid]struct{}
}

func (g *group) putCid(c cid.Cid, blockSize uint32) {
	if _, ok := g.cids[c]; ok {
		return
	}

	g.cids[c] = struct{}{}
	g.sizeBytes += uint64(blockSize)
}

type lightfileindex struct {
	srvCfg   configService
	srvStore lightfilenodestore.StoreService

	defaultLimitBytes uint64

	sync.RWMutex
	groups     map[string]*group
	blocksLake map[cid.Cid]*cidBlock
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
	i.blocksLake = make(map[cid.Cid]*cidBlock)

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
		if _, hasBlock := f.cids[k]; hasBlock {
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
	grp := i.getGroupEntry(groupId, false)

	resp := fileproto.AccountInfoResponse{
		LimitBytes:        i.defaultLimitBytes,
		AccountLimitBytes: i.defaultLimitBytes,
		TotalUsageBytes:   grp.sizeBytes,
		TotalCidsCount:    uint64(len(grp.cids)),
	}

	resp.Spaces = make([]*fileproto.SpaceInfoResponse, 0, len(grp.spaces))
	for spaceId := range grp.spaces {
		spInfo := i.getSpaceInfo(grp, spaceId)
		resp.Spaces = append(resp.Spaces, &spInfo)
	}

	return resp
}

func (i *lightfileindex) getGroupEntry(groupId string, saveCreated bool) (g *group) {
	g = i.groups[groupId]
	if g == nil {
		g = &group{
			sizeBytes: 0,
			spaces:    make(map[string]*space),
			cids:      make(map[cid.Cid]struct{}),
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

	grp := i.getGroupEntry(key.GroupId, false)
	spInfo := i.getSpaceInfo(grp, key.SpaceId)

	return spInfo
}

func (i *lightfileindex) getSpaceInfo(group *group, spaceId string) fileproto.SpaceInfoResponse {
	s := i.getSpaceEntry(group, spaceId, false)

	resp := fileproto.SpaceInfoResponse{
		SpaceId:         spaceId,
		TotalUsageBytes: group.sizeBytes,
		LimitBytes:      i.defaultLimitBytes,
		CidsCount:       uint64(len(s.cids)),
		FilesCount:      uint64(len(s.files)),
		SpaceUsageBytes: s.sizeBytes,
	}

	return resp
}

func (i *lightfileindex) getSpaceEntry(group *group, spaceId string, saveCreated bool) (s *space) {
	s = group.spaces[spaceId]
	if s == nil {
		s = &space{
			sizeBytes: 0,
			files:     make(map[string]*file),
			cids:      make(map[cid.Cid]struct{}),
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

	grp := i.getGroupEntry(key.GroupId, false)
	sp := i.getSpaceEntry(grp, key.SpaceId, false)

	fileIds := make([]string, 0, len(sp.files))
	for fileId := range sp.files {
		fileIds = append(fileIds, fileId)
	}

	return fileIds
}

func (i *lightfileindex) FileInfo(key index.Key, fileIds ...string) []*fileproto.FileInfo {
	i.RLock()
	defer i.RUnlock()

	grp := i.getGroupEntry(key.GroupId, false)
	sp := i.getSpaceEntry(grp, key.SpaceId, false)

	result := make([]*fileproto.FileInfo, 0, len(fileIds))
	for _, fileId := range fileIds {
		f := i.getFileEntry(sp, fileId, false)
		result = append(result, &fileproto.FileInfo{
			FileId:     fileId,
			UsageBytes: f.sizeBytes,
			CidsCount:  uint32(len(f.cids)),
		})
	}

	return result
}

func (i *lightfileindex) getFileEntry(space *space, fileId string, saveCreated bool) (f *file) {
	f = space.files[fileId]
	if f == nil {
		f = &file{
			sizeBytes: 0,
			cids:      make(map[cid.Cid]struct{}),
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

	grp := i.getGroupEntry(key.GroupId, true)
	sps := i.getSpaceEntry(grp, key.SpaceId, true)

	for _, op := range operations {
		if errProc := i.processOperation(grp, sps, op); errProc != nil {
			return fmt.Errorf("failed to process operation: %w", errProc)
		}
	}

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
		return i.handleCIDAdd(group, space, op.GetCidAdd())
	case op.GetBindFile() != nil:
		return i.handleBindFile(group, space, op.GetBindFile())
	case op.GetDeleteFile() != nil:
		return i.handleDeleteFile(group, space, op.GetDeleteFile())
	default:
		return errors.New("unsupported operation")
	}
}

func (i *lightfileindex) handleCIDAdd(group *group, space *space, op *indexpb.CidAddOperation) error {
	c, err := cid.Parse(op.GetCid())
	if err != nil {
		return ErrInvalidCID
	}

	size := uint32(op.GetDataSize())
	b, exists := i.blocksLake[c]
	if !exists {
		b = &cidBlock{
			size:     size,
			refCount: 0,
		}
		i.blocksLake[c] = b
	} else if b.size != size {
		log.Warn("cidBlock size mismatch",
			zap.String("cid", c.String()),
			zap.Uint32("expected", size),
			zap.Uint32("actual", b.size),
		)
		return fmt.Errorf("cidBlock size mismatch: %d != %d", b.size, size)
	}

	fileId := op.GetFileId()
	fileEntry := i.getFileEntry(space, fileId, true)

	_, isFileHadBlock := fileEntry.cids[c]
	if !isFileHadBlock {
		fileEntry.cids[c] = struct{}{}
		fileEntry.sizeBytes += uint64(b.size)
		b.refCount++
	}

	space.putCid(c, size)
	group.putCid(c, size)

	return nil
}

func (i *lightfileindex) handleBindFile(group *group, space *space, op *indexpb.FileBindOperation) error {
	fileId := op.GetFileId()
	cids := op.GetCids()

	// Get or create the file entry
	fileEntry := i.getFileEntry(space, fileId, true)

	// Process each CID in the bind operation
	for _, cidStr := range cids {
		c, err := cid.Parse(cidStr)
		if err != nil {
			return ErrInvalidCID
		}

		// Check if the cidBlock exists in the global blocks lake
		blk, exists := i.blocksLake[c]
		if !exists {
			log.Warn("attempted to bind non-existent CID",
				zap.String("cid", cidStr),
				zap.String("fileId", fileId),
			)
			continue
		}

		if _, isFileHadBlock := fileEntry.cids[c]; !isFileHadBlock {
			fileEntry.cids[c] = struct{}{}
			fileEntry.sizeBytes += uint64(blk.size)
			blk.refCount++
		}

		space.putCid(c, blk.size)
		group.putCid(c, blk.size)
	}

	return nil
}

// handleDeleteFile removes files from a space
func (i *lightfileindex) handleDeleteFile(group *group, space *space, op *indexpb.FileDeleteOperation) error {
	for _, fileId := range op.GetFileIds() {
		f, ok := space.files[fileId]
		if !ok {
			continue
		}

		delete(space.files, fileId)

		for c := range f.cids {
			blk := i.blocksLake[c]
			if blk != nil {
				blk.refCount--

				// Check if this CID is still used by other files in this space
				isStillUsedInSpace := false
				for _, otherFile := range space.files {
					if _, used := otherFile.cids[c]; used {
						isStillUsedInSpace = true
						break
					}
				}

				// Only remove from space if no other files use this CID
				if !isStillUsedInSpace {
					delete(space.cids, c)
					space.sizeBytes -= uint64(blk.size)
				}

				// Check if this CID is still used in any space in this group
				isStillUsedInGroup := false
				if !isStillUsedInSpace {
					// Only check other spaces if it's not used in the current space
					for _, sp := range group.spaces {
						if sp != space { // We already checked the current space
							for _, otherFile := range sp.files {
								if _, used := otherFile.cids[c]; used {
									isStillUsedInGroup = true
									break
								}
							}

							if isStillUsedInGroup {
								break
							}
						}
					}
				} else {
					// If it's still used in this space, it's definitely still used in the group
					isStillUsedInGroup = true
				}

				// Only remove from group if no files in any space use this CID
				if !isStillUsedInGroup {
					delete(group.cids, c)
					group.sizeBytes -= uint64(blk.size)
				}
			}
		}
	}

	return nil
}

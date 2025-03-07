package lightfilenoderpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anyproto/any-sync/acl"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/commonfile/fileproto"
	"github.com/anyproto/any-sync/commonfile/fileproto/fileprotoerr"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/dgraph-io/badger/v4"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"

	"github.com/anyproto/any-sync-filenode/index"

	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex/indexpb"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

const (
	CName             = "light.filenode.rpc"
	cidSizeLimit      = 2 << 20                // 2 Mb
	blockGetRetryTime = 100 * time.Millisecond // How often to retry block get if it's not available
)

var log = logger.NewNamed(CName)

var (
	ErrWrongHash = fmt.Errorf("block data hash mismatch")
)

type lightFileNodeRpc struct {
	srvAcl   acl.AclService
	srvDRPC  server.DRPCServer
	srvIndex lightfilenodeindex.IndexService
	srvStore lightfilenodestore.StoreService
}

func New() app.ComponentRunnable {
	return new(lightFileNodeRpc)
}

//
// App Component
//

//          RPC (DRPC)
//           ├──────┐
//           │      │ [Read/Write]
// [BlockGet]│      ▼
//           │    Index (In-memory)
//           │      │ [Write-log and Snapshots]
//           ├──────┘
//           ▼
//         Store (BadgerDB)

func (r *lightFileNodeRpc) Init(a *app.App) error {
	log.Info("call Init")

	r.srvAcl = app.MustComponent[acl.AclService](a)
	r.srvDRPC = app.MustComponent[server.DRPCServer](a)
	r.srvIndex = app.MustComponent[lightfilenodeindex.IndexService](a)
	r.srvStore = app.MustComponent[lightfilenodestore.StoreService](a)

	return nil
}

func (r *lightFileNodeRpc) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (r *lightFileNodeRpc) Run(ctx context.Context) error {
	return fileproto.DRPCRegisterFile(r.srvDRPC, r)
}

func (r *lightFileNodeRpc) Close(ctx context.Context) error {
	return nil
}

//
// Component methods
//

// BlockGet returns block data by CID
// It does not check permissions, just returns the block data, if it exists
func (r *lightFileNodeRpc) BlockGet(ctx context.Context, req *fileproto.BlockGetRequest) (*fileproto.BlockGetResponse, error) {
	log.InfoCtx(ctx, "BlockGet",
		zap.String("spaceId", req.SpaceId),
		zap.Strings("cid", cidsToStrings(req.Cid)),
		zap.Bool("wait", req.Wait),
	)

	c, err := cid.Cast(req.Cid)
	if err != nil {
		return nil, fmt.Errorf("failed to cast CID='%s': %w", string(req.Cid), err)
	}

	var blockData []byte
	for {
		errTx := r.srvStore.TxView(func(txn *badger.Txn) error {
			var errGet error
			blockData, errGet = r.srvStore.GetBlock(txn, c)
			return errGet
		})

		if errTx == nil {
			break
		}

		hasRetry := req.Wait && errors.Is(errTx, lightfilenodestore.ErrBlockNotFound)
		if !hasRetry {
			return nil, fmt.Errorf("transaction view failed: %w", errTx)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(blockGetRetryTime):
			continue
		}
	}

	resp := &fileproto.BlockGetResponse{
		Cid:  req.Cid,
		Data: blockData,
	}

	return resp, nil
}

// BlockPush stores a CID and data in the datastore.
// It checks permissions and space limits before storing the data.
func (r *lightFileNodeRpc) BlockPush(ctx context.Context, req *fileproto.BlockPushRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "BlockPush",
		zap.String("spaceId", req.SpaceId),
		zap.String("fileId", req.FileId),
		zap.Strings("cid", cidsToStrings(req.Cid)),
		zap.Int("dataSize", len(req.Data)),
	)

	dataSize := len(req.Data)

	// Check that CID is valid for the data
	c, err := cid.Cast(req.Cid)
	if err != nil {
		return nil, fmt.Errorf("failed to cast CID: %w", err)
	}

	blk, err := blocks.NewBlockWithCid(req.Data, c)
	if err != nil {
		return nil, fmt.Errorf("failed to create block: %w", err)
	}

	if len(req.Data) > cidSizeLimit {
		return nil, fileprotoerr.ErrQuerySizeExceeded
	}

	// Check that the block data checksum matches the provided checksum
	chkc, err := c.Prefix().Sum(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate block data checksum: %w", err)
	}

	if !chkc.Equals(c) {
		return nil, ErrWrongHash
	}

	// Finally, save the block and update all necessary counters
	errTx := r.srvStore.TxUpdate(func(txn *badger.Txn) error {
		storageKey, errPerm := r.canWrite(ctx, req.SpaceId)
		if errPerm != nil {
			return errPerm
		}

		// Check if CID exists before storing to avoid duplicate storage
		hadCid := r.srvIndex.HadCID(c)

		// If block doesn't exist, store it
		if !hadCid {
			if errPush := r.srvStore.PutBlock(txn, blk); errPush != nil {
				return fmt.Errorf("failed to push block: %w", errPush)
			}

			// TODO: Create CID in Index

			bindOp := &indexpb.FileBindOperation{}
			bindOp.SetFileId(req.FileId)
			bindOp.SetBlockCids([]string{c.String()})
			bindOp.SetDataSizes([]int32{int32(dataSize)})

			op := &indexpb.Operation{}
			op.SetBindFile(bindOp)

			if errModify := r.srvIndex.Modify(txn, storageKey, op); errModify != nil {
				return fmt.Errorf("failed to bind block through Modify: %w", errModify)
			}
		}

		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return &fileproto.Ok{}, nil
}

func (r *lightFileNodeRpc) BlocksCheck(ctx context.Context, request *fileproto.BlocksCheckRequest) (*fileproto.BlocksCheckResponse, error) {
	log.InfoCtx(ctx, "BlocksCheck",
		zap.String("spaceId", request.SpaceId),
		zap.Strings("cids", cidsToStrings(request.Cids...)),
	)

	availability := make([]*fileproto.BlockAvailability, 0, len(request.Cids))
	seen := make(map[cid.Cid]struct{}, len(request.Cids))

	for _, rawCid := range request.Cids {
		c, err := cid.Cast(rawCid)
		if err != nil {
			return nil, fmt.Errorf("failed to cast CID='%s': %w", string(rawCid), err)
		}

		// Skip duplicates
		if _, exists := seen[c]; exists {
			continue
		}
		seen[c] = struct{}{}

		status := &fileproto.BlockAvailability{
			Cid:    rawCid,
			Status: r.checkCIDExists(request.SpaceId, c),
		}

		availability = append(availability, status)
	}

	return &fileproto.BlocksCheckResponse{
		BlocksAvailability: availability,
	}, nil
}

func (r *lightFileNodeRpc) BlocksBind(ctx context.Context, req *fileproto.BlocksBindRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "BlocksBind",
		zap.String("spaceId", req.SpaceId),
		zap.String("fileId", req.FileId),
		zap.Strings("cids", cidsToStrings(req.Cids...)),
	)

	// Convert binary CIDs to actual CID objects
	cids := r.convertCids(req.Cids)

	// Convert to string representation for the proto message
	cidStrings := make([]string, 0, len(cids))
	for _, c := range cids {
		cidStrings = append(cidStrings, c.String())
	}

	errTx := r.srvStore.TxUpdate(func(txn *badger.Txn) error {
		storageKey, errPerm := r.canWrite(ctx, req.SpaceId)
		if errPerm != nil {
			return errPerm
		}

		bindOp := &indexpb.FileBindOperation{}
		bindOp.SetFileId(req.FileId)
		bindOp.SetBlockCids(cidStrings)

		op := &indexpb.Operation{}
		op.SetBindFile(bindOp)

		if errModify := r.srvIndex.Modify(txn, storageKey, op); errModify != nil {
			return fmt.Errorf("failed to bind blocks through Modify: %w", errModify)
		}

		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return &fileproto.Ok{}, nil
}

func (r *lightFileNodeRpc) FilesInfo(ctx context.Context, request *fileproto.FilesInfoRequest) (*fileproto.FilesInfoResponse, error) {
	log.InfoCtx(ctx, "FilesInfo",
		zap.String("spaceId", request.SpaceId),
		zap.Strings("fileIds", request.FileIds),
	)

	if _, errPerm := r.canRead(ctx, request.SpaceId); errPerm != nil {
		return nil, errPerm
	}

	fileInfos, err := r.srvIndex.GetFileInfo(request.SpaceId, request.FileIds...)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &fileproto.FilesInfoResponse{
		FilesInfo: fileInfos,
	}, nil
}

func (r *lightFileNodeRpc) FilesGet(request *fileproto.FilesGetRequest, stream fileproto.DRPCFile_FilesGetStream) error {
	ctx := stream.Context()
	log.InfoCtx(ctx, "FilesGet", zap.String("spaceId", request.SpaceId))

	if _, err := r.canRead(ctx, request.SpaceId); err != nil {
		return err
	}

	files, err := r.srvIndex.GetSpaceFiles(request.SpaceId)
	if err != nil {
		return err
	}

	// Stream files to client
	for _, fileID := range files {
		select {
		case <-ctx.Done():
			log.WarnCtx(ctx, "FilesGet stream cancelled", zap.Error(ctx.Err()))
			return ctx.Err()
		default:
			if errSend := stream.Send(&fileproto.FilesGetResponse{FileId: fileID}); errSend != nil {
				log.ErrorCtx(ctx, "FilesGet failed to send response", zap.Error(errSend))
				return errSend
			}
		}
	}

	return nil
}

func (r *lightFileNodeRpc) FilesDelete(ctx context.Context, request *fileproto.FilesDeleteRequest) (*fileproto.FilesDeleteResponse, error) {
	log.InfoCtx(ctx, "FilesDelete",
		zap.String("spaceId", request.SpaceId),
		zap.Strings("fileIds", request.FileIds),
	)

	errTx := r.srvStore.TxUpdate(func(txn *badger.Txn) error {
		storageKey, errPerm := r.canWrite(ctx, request.SpaceId)
		if errPerm != nil {
			return errPerm
		}

		unbindOp := &indexpb.FileUnbindOperation{}
		unbindOp.SetFileIds(request.FileIds)

		op := &indexpb.Operation{}
		op.SetUnbindFile(unbindOp)

		if errModify := r.srvIndex.Modify(txn, storageKey, op); errModify != nil {
			return fmt.Errorf("failed to unbind files through Modify: %w", errModify)
		}

		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return &fileproto.FilesDeleteResponse{}, nil
}

func (r *lightFileNodeRpc) Check(ctx context.Context, request *fileproto.CheckRequest) (*fileproto.CheckResponse, error) {
	log.InfoCtx(ctx, "Check")

	// In a real implementation, this would return the list of spaces the user has access to
	// and whether they have write permissions
	// For now, return a simple response allowing write access
	return &fileproto.CheckResponse{
		SpaceIds:   []string{"space1", "space2"},
		AllowWrite: true,
	}, nil
}

func (r *lightFileNodeRpc) SpaceInfo(ctx context.Context, request *fileproto.SpaceInfoRequest) (*fileproto.SpaceInfoResponse, error) {
	log.InfoCtx(ctx, "SpaceInfo",
		zap.String("spaceId", request.SpaceId),
	)

	storageKey, err := r.canRead(ctx, request.SpaceId)
	if err != nil {
		return nil, err
	}

	spaceInfo := r.srvIndex.SpaceInfo(storageKey)

	return &fileproto.SpaceInfoResponse{
		SpaceId:         request.SpaceId,
		LimitBytes:      spaceInfo.LimitBytes,
		TotalUsageBytes: spaceInfo.UsageBytes,
		CidsCount:       spaceInfo.CidsCount,
		FilesCount:      uint64(spaceInfo.FileCount),
		SpaceUsageBytes: spaceInfo.UsageBytes,
	}, nil
}

func (r *lightFileNodeRpc) AccountInfo(ctx context.Context, request *fileproto.AccountInfoRequest) (*fileproto.AccountInfoResponse, error) {
	log.InfoCtx(ctx, "AccountInfo")

	// Get group info for the default group
	// In a real implementation, this would return information about the user's account
	groupInfo := r.srvIndex.GroupInfo("default-group")

	// Get space info for all spaces in the group
	spaces := make([]*fileproto.SpaceInfoResponse, 0, len(groupInfo.SpaceIds))
	for _, spaceId := range groupInfo.SpaceIds {
		spaceInfo := r.srvIndex.SpaceInfo(index.Key{GroupId: "default-group", SpaceId: spaceId})
		spaces = append(spaces, &fileproto.SpaceInfoResponse{
			SpaceId:         spaceId,
			LimitBytes:      spaceInfo.LimitBytes,
			TotalUsageBytes: spaceInfo.UsageBytes,
			CidsCount:       spaceInfo.CidsCount,
			FilesCount:      uint64(spaceInfo.FileCount),
			SpaceUsageBytes: spaceInfo.UsageBytes,
		})
	}

	return &fileproto.AccountInfoResponse{
		LimitBytes:        groupInfo.LimitBytes,
		TotalUsageBytes:   groupInfo.UsageBytes,
		TotalCidsCount:    groupInfo.CidsCount,
		Spaces:            spaces,
		AccountLimitBytes: groupInfo.AccountLimitBytes,
	}, nil
}

func (r *lightFileNodeRpc) AccountLimitSet(ctx context.Context, request *fileproto.AccountLimitSetRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "AccountLimitSet",
		zap.String("identity", request.Identity),
		zap.Uint64("limit", request.Limit),
	)

	errTx := r.srvStore.TxUpdate(func(txn *badger.Txn) error {
		setLimitOp := &indexpb.AccountLimitSetOperation{}
		setLimitOp.SetLimit(request.Limit)

		op := &indexpb.Operation{}
		op.SetAccountLimitSet(setLimitOp)

		return r.srvIndex.Modify(txn, index.Key{GroupId: request.Identity}, op)
	})

	if errTx != nil {
		return nil, errTx
	}

	return &fileproto.Ok{}, nil
}

func (r *lightFileNodeRpc) SpaceLimitSet(ctx context.Context, request *fileproto.SpaceLimitSetRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "SpaceLimitSet",
		zap.String("spaceId", request.SpaceId),
		zap.Uint64("limit", request.Limit),
	)

	errTx := r.srvStore.TxUpdate(func(txn *badger.Txn) error {
		storageKey, errPerm := r.canWrite(ctx, request.SpaceId)
		if errPerm != nil {
			return fmt.Errorf("failed to check permissions: %w", errPerm)
		}

		setLimitOp := &indexpb.SpaceLimitSetOperation{}
		setLimitOp.SetLimit(request.Limit)

		op := &indexpb.Operation{}
		op.SetSpaceLimitSet(setLimitOp)

		return r.srvIndex.Modify(txn, storageKey, op)
	})

	if errTx != nil {
		return nil, errTx
	}

	return &fileproto.Ok{}, nil
}

package lightfilenoderpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync/acl"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/commonfile/fileproto"
	"github.com/anyproto/any-sync/commonfile/fileproto/fileprotoerr"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/dgraph-io/badger/v4"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"

	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex/indexpb"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

const (
	CName = "light.filenode.rpc"
)

const (
	cidSizeLimit      = 2 << 20                // 2 MiB in bytes
	fileInfoReqLimit  = 1000                   // Maximum number of file info requests
	blockGetRetryTime = 100 * time.Millisecond // How often to retry block get if it's not available
)

var (
	// Type assertion.
	_ fileproto.DRPCFileServer = (*lightfilenoderpc)(nil)

	log = logger.NewNamed(CName)

	ErrWrongChecksum = fmt.Errorf("block data checksum mismatch")
)

//          RPC (DRPC)
//           ├──────┐
//           │      │ [Read/Write]
// [BlockGet]│      ▼
//           │    Index (In-memory)
//           │      │ [Write-log and Snapshots]
//           ├──────┘
//           ▼
//         Store (BadgerDB)

type lightfilenoderpc struct {
	srvAcl   acl.AclService
	srvDRPC  server.DRPCServer
	srvIndex lightfilenodeindex.IndexService
	srvStore lightfilenodestore.StoreService
}

func New() *lightfilenoderpc {
	return new(lightfilenoderpc)
}

//
// App Component.
//

func (r *lightfilenoderpc) Init(a *app.App) error {
	log.Info("call Init")

	r.srvAcl = app.MustComponent[acl.AclService](a)
	r.srvDRPC = app.MustComponent[server.DRPCServer](a)
	r.srvIndex = app.MustComponent[lightfilenodeindex.IndexService](a)
	r.srvStore = app.MustComponent[lightfilenodestore.StoreService](a)

	return nil
}

func (r *lightfilenoderpc) Name() (name string) {
	return CName
}

//
// App Component Runnable.
//

func (r *lightfilenoderpc) Run(ctx context.Context) error {
	return fileproto.DRPCRegisterFile(r.srvDRPC, r)
}

func (r *lightfilenoderpc) Close(ctx context.Context) error {
	return nil
}

//
// Component methods.
//

// BlockGet returns block data by CID
// It does not check permissions, just returns the block data, if it exists.
func (r *lightfilenoderpc) BlockGet(
	ctx context.Context,
	req *fileproto.BlockGetRequest,
) (*fileproto.BlockGetResponse, error) {
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
		} else if !errors.Is(errTx, lightfilenodestore.ErrBlockNotFound) {
			return nil, fmt.Errorf("failed to get block: %w", errTx)
		}

		if !req.Wait {
			return nil, fileprotoerr.ErrCIDNotFound
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
func (r *lightfilenoderpc) BlockPush(ctx context.Context, req *fileproto.BlockPushRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "BlockPush",
		zap.String("spaceId", req.SpaceId),
		zap.String("fileId", req.FileId),
		zap.Strings("cid", cidsToStrings(req.Cid)),
		zap.Int("dataSize", len(req.Data)),
	)

	dataSize := len(req.Data)

	// Check that CID is valid for the data.
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

	// Check that the block data checksum matches the provided checksum.
	chkc, err := c.Prefix().Sum(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate block data checksum: %w", err)
	}

	if !chkc.Equals(c) {
		return nil, ErrWrongChecksum
	}

	storageKey, err := r.canWrite(ctx, req.SpaceId)
	if err != nil {
		return nil, err
	}

	cidString := c.String()

	errTx := r.srvStore.TxUpdate(func(txn *badger.Txn) error {
		// Check if CID exists before storing to avoid duplicate storage.
		hadCid := r.srvIndex.HadCID(c)
		if !hadCid {
			if errPush := r.srvStore.PutBlock(txn, blk); errPush != nil {
				return fmt.Errorf("failed to push block: %w", errPush)
			}
		}

		cidOp := &indexpb.CidAddOperation{}
		cidOp.SetFileId(req.FileId)
		cidOp.SetCid(cidString)
		cidOp.SetDataSize(uint64(dataSize))

		op := &indexpb.Operation{}
		op.SetCidAdd(cidOp)

		if errModify := r.srvIndex.Modify(txn, storageKey, op); errModify != nil {
			return fmt.Errorf("failed to modify index: %w", errModify)
		}

		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return &fileproto.Ok{}, nil
}

// BlocksCheck checks if the given CIDs exist in the space or in general in storage.
func (r *lightfilenoderpc) BlocksCheck(
	ctx context.Context,
	request *fileproto.BlocksCheckRequest,
) (*fileproto.BlocksCheckResponse, error) {
	log.InfoCtx(ctx, "BlocksCheck",
		zap.String("spaceId", request.SpaceId),
		zap.Strings("cids", cidsToStrings(request.Cids...)),
	)

	storageKey, err := r.canRead(ctx, request.SpaceId)
	if err != nil {
		return nil, err
	}

	availability := make([]*fileproto.BlockAvailability, 0, len(request.Cids))
	seen := make(map[cid.Cid]struct{}, len(request.Cids))

	for _, rawCid := range request.Cids {
		c, errCast := cid.Cast(rawCid)
		if errCast != nil {
			return nil, fmt.Errorf("failed to cast CID='%s': %w", string(rawCid), errCast)
		}

		// Skip duplicates.
		if _, exists := seen[c]; exists {
			continue
		}
		seen[c] = struct{}{}

		status := &fileproto.BlockAvailability{
			Cid:    rawCid,
			Status: r.checkCIDExists(storageKey, c),
		}

		availability = append(availability, status)
	}

	return &fileproto.BlocksCheckResponse{
		BlocksAvailability: availability,
	}, nil
}

// checkCIDExists if the CID exists in space or in general in storage or not
func (r *lightfilenoderpc) checkCIDExists(storeKey index.Key, k cid.Cid) (status fileproto.AvailabilityStatus) {
	if storeKey.SpaceId != "" {
		exist := r.srvIndex.HasCIDInSpace(storeKey, k)
		if exist {
			return fileproto.AvailabilityStatus_ExistsInSpace
		}
	}

	exist := r.srvIndex.HadCID(k)
	if exist {
		return fileproto.AvailabilityStatus_Exists
	}

	return fileproto.AvailabilityStatus_NotExists
}

// BlocksBind connect a list of CIDs to a file in the space.
func (r *lightfilenoderpc) BlocksBind(ctx context.Context, req *fileproto.BlocksBindRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "BlocksBind",
		zap.String("spaceId", req.SpaceId),
		zap.String("fileId", req.FileId),
		zap.Strings("cids", cidsToStrings(req.Cids...)),
	)

	storageKey, err := r.canWrite(ctx, req.SpaceId)
	if err != nil {
		return nil, err
	}

	cids, err := convertCids(req.Cids)
	if err != nil {
		return nil, err
	}

	cidStrings := make([]string, 0, len(cids))
	for _, c := range cids {
		cidStrings = append(cidStrings, c.String())
	}

	bindOp := &indexpb.FileBindOperation{}
	bindOp.SetFileId(req.FileId)
	bindOp.SetCids(cidStrings)

	op := &indexpb.Operation{}
	op.SetBindFile(bindOp)

	errTx := r.srvStore.TxUpdate(func(txn *badger.Txn) error {
		return r.srvIndex.Modify(txn, storageKey, op)
	})

	if errTx != nil {
		return nil, errTx
	}

	return &fileproto.Ok{}, nil
}

// FilesInfo returns information about a file in the space.
func (r *lightfilenoderpc) FilesInfo(
	ctx context.Context,
	req *fileproto.FilesInfoRequest,
) (*fileproto.FilesInfoResponse, error) {
	log.InfoCtx(ctx, "FilesInfo",
		zap.String("spaceId", req.SpaceId),
		zap.Strings("fileIds", req.FileIds),
	)

	if len(req.FileIds) > fileInfoReqLimit {
		return nil, fileprotoerr.ErrQuerySizeExceeded
	}

	storeKey, err := r.canRead(ctx, req.SpaceId)
	if err != nil {
		return nil, err
	}

	fileInfos := r.srvIndex.FileInfo(storeKey, req.FileIds...)

	return &fileproto.FilesInfoResponse{
		FilesInfo: fileInfos,
	}, nil
}

// FilesGet returns a stream of file IDs in the space.
func (r *lightfilenoderpc) FilesGet(
	request *fileproto.FilesGetRequest,
	stream fileproto.DRPCFile_FilesGetStream,
) error {
	ctx := stream.Context()
	log.InfoCtx(ctx, "FilesGet", zap.String("spaceId", request.SpaceId))

	storeKey, err := r.canRead(ctx, request.SpaceId)
	if err != nil {
		return err
	}

	files := r.srvIndex.SpaceFiles(storeKey)

	for _, fileID := range files {
		fileProto := &fileproto.FilesGetResponse{FileId: fileID}

		if errSend := stream.Send(fileProto); errSend != nil {
			log.ErrorCtx(ctx, "FilesGet failed to send response", zap.Error(errSend))
			return errSend
		}
	}

	return nil
}

// FilesDelete removes a list of files from the space.
func (r *lightfilenoderpc) FilesDelete(
	ctx context.Context,
	request *fileproto.FilesDeleteRequest,
) (*fileproto.FilesDeleteResponse, error) {
	log.InfoCtx(ctx, "FilesDelete",
		zap.String("spaceId", request.SpaceId),
		zap.Strings("fileIds", request.FileIds),
	)

	storageKey, err := r.canWrite(ctx, request.SpaceId)
	if err != nil {
		return nil, err
	}

	deleteOp := &indexpb.FileDeleteOperation{}
	deleteOp.SetFileIds(request.FileIds)

	op := &indexpb.Operation{}
	op.SetDeleteFile(deleteOp)

	errTx := r.srvStore.TxUpdate(func(txn *badger.Txn) error {
		return r.srvIndex.Modify(txn, storageKey, op)
	})

	if errTx != nil {
		return nil, errTx
	}

	return &fileproto.FilesDeleteResponse{}, nil
}

// Check is just simple health check.
func (r *lightfilenoderpc) Check(ctx context.Context, _ *fileproto.CheckRequest) (*fileproto.CheckResponse, error) {
	log.InfoCtx(ctx, "Check")

	return &fileproto.CheckResponse{
		SpaceIds:   nil,
		AllowWrite: true,
	}, nil
}

// SpaceInfo returns information about the space.
func (r *lightfilenoderpc) SpaceInfo(
	ctx context.Context,
	request *fileproto.SpaceInfoRequest,
) (*fileproto.SpaceInfoResponse, error) {
	log.InfoCtx(ctx, "SpaceInfo",
		zap.String("spaceId", request.SpaceId),
	)

	storageKey, err := r.canRead(ctx, request.SpaceId)
	if err != nil {
		return nil, err
	}

	spaceInfo := r.srvIndex.SpaceInfo(storageKey)
	return &spaceInfo, nil
}

// AccountInfo returns information about the account/group.
func (r *lightfilenoderpc) AccountInfo(
	ctx context.Context,
	req *fileproto.AccountInfoRequest,
) (*fileproto.AccountInfoResponse, error) {
	log.InfoCtx(ctx, "AccountInfo")

	identity, err := peer.CtxPubKey(ctx)
	if err != nil {
		log.WarnCtx(ctx, "AccountInfo failed to get identity", zap.Error(err))
		return nil, fileprotoerr.ErrForbidden
	}
	groupId := identity.Account()

	groupInfo := r.srvIndex.GroupInfo(groupId)
	return &groupInfo, nil
}

// AccountLimitSet sets the account/group storage limit.
// NOTE: Logic is changed for self-hosted version.
func (r *lightfilenoderpc) AccountLimitSet(
	ctx context.Context,
	request *fileproto.AccountLimitSetRequest,
) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "AccountLimitSet",
		zap.String("identity", request.Identity),
		zap.Uint64("limit", request.Limit),
	)

	// Because we are using self-hosted version, we simplify the logic here.
	// We don't have payment system, so we can't set account limit,
	// and we don't have any option from Anytype to set account limit.
	// It may be implemented in the future, but for now we just return error.

	return nil, fmt.Errorf("you can't set account limit in this implementation: %w", fileprotoerr.ErrForbidden)
}

// SpaceLimitSet sets the space storage limit.
func (r *lightfilenoderpc) SpaceLimitSet(
	ctx context.Context,
	request *fileproto.SpaceLimitSetRequest,
) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "SpaceLimitSet",
		zap.String("spaceId", request.SpaceId),
		zap.Uint64("limit", request.Limit),
	)

	// Because we are using self-hosted version, we simplify the logic here.
	// There in no option from Anytype to set space limit.
	// It may be implemented in the future, but for now we just return error.

	return nil, fmt.Errorf("you can't set space limit in this implementation: %w", fileprotoerr.ErrForbidden)
}

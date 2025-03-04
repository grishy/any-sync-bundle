package lightfilenoderpc

import (
	"context"
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

	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

const (
	CName             = "light.filenode.rpc"
	cidSizeLimit      = 2 << 20                // 2 Mb
	blockGetRetryTime = 100 * time.Millisecond // How often to retry block get if it's not available
)

var log = logger.NewNamed(CName)

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

		if !req.Wait {
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

	// Check that the block data checksum matches the provided checksum as implemented in debug mode of blocks pkg.
	chkc, err := c.Prefix().Sum(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate block data checksum: %w", err)
	}

	if !chkc.Equals(c) {
		return nil, fmt.Errorf("block data checksum mismatch: %s != %s", chkc, c)
	}

	// Finally, save the block and update all necessary counters
	errTx := r.srvStore.TxUpdate(func(txn *badger.Txn) error {
		storageKey, errPerm := r.canWrite(ctx, txn, req.SpaceId)
		if errPerm != nil {
			return errPerm
		}

		_ = dataSize
		_ = blk
		_ = storageKey

		// // Check existing block before, to avoid overwriting
		// isBlkExist, errBlock := r.index.HadCID(txn, c)
		// if errBlock != nil {
		// 	return fmt.Errorf("failed to check if CID exists: %w", errBlock)
		// }
		//
		// if !isBlkExist {
		// 	if errPush := r.store.PutBlock(txn, blk); errPush != nil {
		// 		return fmt.Errorf("failed to push block: %w", errPush)
		// 	}
		// }
		//
		// errBind := r.bindBlock(txn, storageKey, req.SpaceId, req.FileId, c, dataSize)
		// if errBind != nil {
		// 	return fmt.Errorf("failed to bind block: %w", errBind)
		// }

		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return &fileproto.Ok{}, nil
}

// func (r *lightFileNodeRpc) bindBlock(txn *badger.Txn, storageKey index.Key, spaceId, fileId string, c cid.Cid, dataSize int) error {
// 	// Update file info, if link exists we don't need to update counters
// 	isLinkExist, errLink := r.store.HasLinkFileBlock(txn, spaceId, fileId, c)
// 	if errLink != nil {
// 		return fmt.Errorf("failed to check if link exists: %w", errLink)
// 	}
//
// 	if isLinkExist {
// 		// If the link exists, we don't need to update counters
// 		return nil
// 	}
//
// 	// TODO: Update ref counter on CID from file
//
// 	if errLink := r.store.CreateLinkFileBlock(txn, spaceId, fileId, c); errLink != nil {
// 		return fmt.Errorf("failed to create link: %w", errLink)
// 	}
//
// 	fileObj, errGetFile := r.store.GetFile(txn, spaceId, fileId)
// 	if errGetFile != nil {
// 		return fmt.Errorf("failed to get file: %w", errGetFile)
// 	}
//
// 	// Check if this is a new file (no CIDs yet)
// 	isNewFile := fileObj.CidsCount() == 0
//
// 	// Update file info
// 	fileObj.IncCidsCount()
// 	fileObj.IncUsageBytes(uint64(dataSize))
//
// 	if errWrite := r.store.WriteFile(txn, fileObj); errWrite != nil {
// 		return fmt.Errorf("failed to write file: %w", errWrite)
// 	}
//
// 	// Update space info
// 	spaceObj, errGetSpace := r.store.GetSpace(txn, spaceId)
// 	if errGetSpace != nil {
// 		return fmt.Errorf("failed to get space: %w", errGetSpace)
// 	}
//
// 	spaceObj.IncCidsCount()
// 	if isNewFile {
// 		spaceObj.IncFilesCount() // Only increment for new files
// 	}
// 	spaceObj.IncUsageBytes(uint64(dataSize))
//
// 	if errWrite := r.store.WriteSpace(txn, spaceObj); errWrite != nil {
// 		return fmt.Errorf("failed to write space: %w", errWrite)
// 	}
//
// 	// Update group info
// 	groupObj, errGetGroup := r.store.GetGroup(txn, storageKey.GroupId)
// 	if errGetGroup != nil {
// 		return fmt.Errorf("failed to get group: %w", errGetGroup)
// 	}
//
// 	groupObj.IncCidsCount()
// 	groupObj.IncUsageBytes(uint64(dataSize))
//
// 	if errWrite := r.store.WriteGroup(txn, groupObj); errWrite != nil {
// 		return fmt.Errorf("failed to write group: %w", errWrite)
// 	}
//
// 	return nil
// }

func (r *lightFileNodeRpc) BlocksCheck(ctx context.Context, request *fileproto.BlocksCheckRequest) (*fileproto.BlocksCheckResponse, error) {
	log.InfoCtx(ctx, "BlocksCheck",
		zap.String("spaceId", request.SpaceId),
		zap.Strings("cids", cidsToStrings(request.Cids...)),
	)

	// TODO: Check permissions?
	// Related issue: https://github.com/anyproto/any-sync-filenode/issues/140

	availability := make([]*fileproto.BlockAvailability, 0, len(request.Cids))
	seen := make(map[cid.Cid]struct{}, len(request.Cids))

	errTx := r.srvStore.TxView(func(txn *badger.Txn) error {
		for _, rawCid := range request.Cids {
			c, err := cid.Cast(rawCid)
			if err != nil {
				return fmt.Errorf("failed to cast CID='%s': %w", string(rawCid), err)
			}

			// Skip duplicates
			if _, exists := seen[c]; exists {
				continue
			}
			seen[c] = struct{}{}

			availabilityStatus, err := r.checkCIDExists(txn, request.SpaceId, c)
			if err != nil {
				return fmt.Errorf("failed to check CID='%s': %w", c.String(), err)
			}

			status := &fileproto.BlockAvailability{
				Cid:    rawCid,
				Status: availabilityStatus,
			}

			availability = append(availability, status)
		}
		return nil
	})

	if errTx != nil {
		return nil, errTx
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

	errTx := r.srvStore.TxUpdate(func(txn *badger.Txn) error {
		storageKey, errPerm := r.canWrite(ctx, txn, req.SpaceId)
		if errPerm != nil {
			return errPerm
		}

		for _, rawCid := range req.Cids {
			c, err := cid.Cast(rawCid)
			if err != nil {
				return fmt.Errorf("failed to cast CID='%s': %w", string(rawCid), err)
			}

			_ = c
			_ = storageKey

			// if errBind := r.bindBlock(txn, storageKey, req.SpaceId, req.FileId, c, 0); errBind != nil {
			// 	return fmt.Errorf("failed to bind block: %w", errBind)
			// }
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

	infoResp := &fileproto.FilesInfoResponse{
		FilesInfo: make([]*fileproto.FileInfo, 0, len(request.FileIds)),
	}

	errTx := r.srvStore.TxView(func(txn *badger.Txn) error {
		if _, errPerm := r.canRead(ctx, request.SpaceId); errPerm != nil {
			return errPerm
		}

		// for _, fileID := range request.FileIds {
		// 	fileObj, errGetFile := r.store.GetFile(txn, request.SpaceId, fileID)
		// 	if errGetFile != nil {
		// 		return fmt.Errorf("failed to get file: %w", errGetFile)
		// 	}
		//
		// 	infoResp.FilesInfo = append(infoResp.FilesInfo, &fileproto.FileInfo{
		// 		FileId:     fileID,
		// 		UsageBytes: fileObj.UsageBytes(),
		// 		CidsCount:  fileObj.CidsCount(),
		// 	})
		// }

		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return infoResp, nil
}

func (r *lightFileNodeRpc) FilesGet(request *fileproto.FilesGetRequest, stream fileproto.DRPCFile_FilesGetStream) error {
	ctx := stream.Context()
	log.InfoCtx(ctx, "FilesGet", zap.String("spaceId", request.SpaceId))

	var files []string
	errTx := r.srvStore.TxView(func(txn *badger.Txn) error {
		if _, err := r.canRead(ctx, request.SpaceId); err != nil {
			return err
		}

		// TODO: Read all files in space
		files = make([]string, 10)
		for i := 0; i < 10; i++ {
			files[i] = "test"
		}

		return nil
	})
	if errTx != nil {
		return errTx
	}

	// Stream files to client after transaction
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
		if _, errPerm := r.canWrite(ctx, txn, request.SpaceId); errPerm != nil {
			return errPerm
		}

		return nil
	})

	if errTx != nil {
		return nil, errTx
	}
	// TODO: Implement logic to bind blocks to the file in the datastore

	// r.badgerDB.NewWriteBatch()

	// err := r.badgerDB.Update(func(txn *badger.Txn) error {
	// 	// For each file, decrement reference counter and delete the key
	// 	prefix := []byte(fmt.Sprintf("l:%s.%s:", request.SpaceId, request.FileIds[0]))
	// 	opts := badger.IteratorOptions{PrefetchSize: 1000}

	// 	it := txn.NewIterator(opts)
	// 	defer it.Close()

	// 	batch := txn.NewBatch()
	// 	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
	// 		cid := extractCID(it.Item().Key())

	// 		batch.Merge(rcKey(cid), uint64ToBytes(^uint64(0))) // Decrement
	// 		batch.Delete(it.Item().Key())

	// 		if batch.Len() >= 1000 {
	// 			if err := batch.Flush(); err != nil {
	// 				return err
	// 			}
	// 			batch = txn.NewBatch()
	// 		}
	// 	}
	// 	return batch.Flush()
	// })

	return &fileproto.FilesDeleteResponse{}, nil
}

func (r *lightFileNodeRpc) Check(ctx context.Context, request *fileproto.CheckRequest) (*fileproto.CheckResponse, error) {
	log.InfoCtx(ctx, "Check")

	// TODO: Implement logic to bind blocks to the file in the datastore

	return &fileproto.CheckResponse{
		SpaceIds:   []string{"space1", "space2"},
		AllowWrite: true,
	}, nil
}

func (r *lightFileNodeRpc) SpaceInfo(ctx context.Context, request *fileproto.SpaceInfoRequest) (*fileproto.SpaceInfoResponse, error) {
	log.InfoCtx(ctx, "SpaceInfo",
		zap.String("spaceId", request.SpaceId),
	)

	infoResp := &fileproto.SpaceInfoResponse{
		SpaceId:         request.SpaceId,
		LimitBytes:      0,
		TotalUsageBytes: 0,
		CidsCount:       0,
		FilesCount:      0,
		SpaceUsageBytes: 0,
	}

	errTx := r.srvStore.TxView(func(txn *badger.Txn) error {
		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	// TODO: Implement logic to bind blocks to the file in the datastore

	// Use file approach for counter that updated on each block push

	return infoResp, nil
}

func (r *lightFileNodeRpc) AccountInfo(ctx context.Context, request *fileproto.AccountInfoRequest) (*fileproto.AccountInfoResponse, error) {
	log.InfoCtx(ctx, "AccountInfo")

	// TODO: Implement logic to bind blocks to the file in the datastore

	// Use file approach for counter that updated on each block push

	return &fileproto.AccountInfoResponse{
		LimitBytes:      100,
		TotalUsageBytes: 30,
		TotalCidsCount:  100,
		Spaces: []*fileproto.SpaceInfoResponse{
			{
				LimitBytes:      100,
				TotalUsageBytes: 222,
				CidsCount:       34,
				FilesCount:      123,
				SpaceUsageBytes: 555,
				SpaceId:         "space1",
			},
		},
		AccountLimitBytes: 29,
	}, nil
}

func (r *lightFileNodeRpc) AccountLimitSet(ctx context.Context, request *fileproto.AccountLimitSetRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "AccountLimitSet",
		zap.String("identity", request.Identity),
		zap.Uint64("limit", request.Limit),
	)

	// TODO: Implement logic to bind blocks to the file in the datastore

	return &fileproto.Ok{}, nil
}

func (r *lightFileNodeRpc) SpaceLimitSet(ctx context.Context, request *fileproto.SpaceLimitSetRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "SpaceLimitSet",
		zap.String("spaceId", request.SpaceId),
		zap.Uint64("limit", request.Limit),
	)

	// TODO: Implement logic to bind blocks to the file in the datastore

	return &fileproto.Ok{}, nil
}

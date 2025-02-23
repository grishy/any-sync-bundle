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

	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

const (
	CName        = "light.filenode.rpc"
	cidSizeLimit = 2 << 20 // 2 Mb
)

var log = logger.NewNamed(CName)

type lightFileNodeRpc struct {
	acl   acl.AclService
	dRPC  server.DRPCServer
	store lightfilenodestore.StoreService
}

func New() Service {
	return new(lightFileNodeRpc)
}

type Service interface {
	app.Component
}

//
// App Component
//

func (r *lightFileNodeRpc) Init(a *app.App) error {
	log.Info("call Init")

	r.dRPC = app.MustComponent[server.DRPCServer](a)
	r.acl = app.MustComponent[acl.AclService](a)
	r.store = app.MustComponent[lightfilenodestore.StoreService](a)

	return nil
}

func (r *lightFileNodeRpc) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (r *lightFileNodeRpc) Run(ctx context.Context) error {
	return fileproto.DRPCRegisterFile(r.dRPC, r)
}

func (r *lightFileNodeRpc) Close(ctx context.Context) error {
	return nil
}

//
// Component methods
//

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

	var blockObj *lightfilenodestore.BlockObj
	for {
		errTx := r.store.TxView(func(txn *badger.Txn) error {
			var errGet error
			blockObj, errGet = r.store.GetBlock(txn, c)
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
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}

	resp := &fileproto.BlockGetResponse{
		Cid:  req.Cid,
		Data: blockObj.Data(),
	}

	return resp, nil
}

func (r *lightFileNodeRpc) BlockPush(ctx context.Context, req *fileproto.BlockPushRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "BlockPush",
		zap.String("spaceId", req.SpaceId),
		zap.String("fileId", req.FileId),
		zap.Strings("cid", cidsToStrings(req.Cid)),
		zap.Int("dataSize", len(req.Data)),
	)

	// Check that CID is valid for the data
	c, err := cid.Cast(req.Cid)
	if err != nil {
		return nil, fmt.Errorf("failed to cast CID: %w", err)
	}

	b, err := blocks.NewBlockWithCid(req.Data, c)
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
	errTx := r.store.TxUpdate(func(txn *badger.Txn) error {
		// Verify that we have access to store and enough space
		if errPerm := r.canWriteWithLimit(ctx, txn, req.SpaceId); errPerm != nil {
			return errPerm
		}

		if errPush := r.store.PushBlock(txn, req.SpaceId, b); errPush != nil {
			return fmt.Errorf("failed to push block: %w", errPush)
		}

		// And connect the block to the file
		// TODO: Implement logic to connect the block to the file in the datastore

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

	// TODO: Check permissions?
	// Related issue: https://github.com/anyproto/any-sync-filenode/issues/140

	availability := make([]*fileproto.BlockAvailability, 0, len(request.Cids))
	seen := make(map[cid.Cid]struct{}, len(request.Cids))

	errTx := r.store.TxView(func(txn *badger.Txn) error {
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

			availabilityStatus, err := r.checkCIDFn(txn, request.SpaceId, c)
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

func (r *lightFileNodeRpc) BlocksBind(ctx context.Context, request *fileproto.BlocksBindRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "BlocksBind",
		zap.String("spaceId", request.SpaceId),
		zap.String("fileId", request.FileId),
		zap.Strings("cids", cidsToStrings(request.Cids...)),
	)

	// TODO: Implement logic to bind blocks to the file in the datastore

	return &fileproto.Ok{}, nil
}

func (r *lightFileNodeRpc) FilesDelete(ctx context.Context, request *fileproto.FilesDeleteRequest) (*fileproto.FilesDeleteResponse, error) {
	log.InfoCtx(ctx, "FilesDelete",
		zap.String("spaceId", request.SpaceId),
		zap.Strings("fileIds", request.FileIds),
	)

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

func (r *lightFileNodeRpc) FilesInfo(ctx context.Context, request *fileproto.FilesInfoRequest) (*fileproto.FilesInfoResponse, error) {
	log.InfoCtx(ctx, "FilesInfo",
		zap.String("spaceId", request.SpaceId),
		zap.Strings("fileIds", request.FileIds),
	)

	// TODO: Implement logic to bind blocks to the file in the datastore

	// Store usage and CID count in the same key
	// Because we always update both values together
	// Key format: f:<spaceId>.<fileId> -> [<usageBytes> uint64][<cidsCount> uint32}

	return &fileproto.FilesInfoResponse{
		FilesInfo: []*fileproto.FileInfo{
			{
				FileId:     request.FileIds[0],
				UsageBytes: 123,
				CidsCount:  100,
			},
		},
	}, nil
}

func (r *lightFileNodeRpc) FilesGet(request *fileproto.FilesGetRequest, stream fileproto.DRPCFile_FilesGetStream) error {
	log.InfoCtx(context.Background(), "FilesGet",
		zap.String("spaceId", request.SpaceId),
	)

	// TODO: Implement logic to bind blocks to the file in the datastore

	for i := 0; i < 10; i++ {
		err := stream.Send(&fileproto.FilesGetResponse{
			FileId: "test",
		})
		if err != nil {
			log.ErrorCtx(context.Background(), "FilesGet stream send failed",
				zap.Error(err))
			return err
		}
	}

	return nil
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

	// TODO: Implement logic to bind blocks to the file in the datastore

	// Use file approach for counter that updated on each block push

	return &fileproto.SpaceInfoResponse{
		LimitBytes:      100,
		TotalUsageBytes: 40,
		CidsCount:       10,
		FilesCount:      2,
		SpaceUsageBytes: 100,
		SpaceId:         request.SpaceId,
	}, nil
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

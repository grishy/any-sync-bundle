package lightfilenoderpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/commonfile/fileproto"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

func byteSlicesToStrings(byteSlices [][]byte) []string {
	strs := make([]string, len(byteSlices))
	for i, b := range byteSlices {
		strs[i] = string(b)
	}
	return strs
}

const (
	CName = "light.filenode.rpc"
)

var log = logger.NewNamed(CName)

type cfgSrv interface {
	GetFilenodeStoreDir() string
}

type lightfilenoderpc struct {
	cfg  cfgSrv
	dRPC server.DRPCServer

	badgerDB *badger.DB
}

func New() *lightfilenoderpc {
	return &lightfilenoderpc{}
}

//
// App Component
//

func (r *lightfilenoderpc) Init(a *app.App) error {
	log.Info("call Init")

	r.cfg = app.MustComponent[cfgSrv](a)
	r.dRPC = app.MustComponent[server.DRPCServer](a)

	return nil
}

func (r *lightfilenoderpc) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (r *lightfilenoderpc) Run(ctx context.Context) error {
	log.Info("call Run")

	storePath := r.cfg.GetFilenodeStoreDir()

	// TODO: Add OnTableRead for block checskum
	// TODO: Fine-tune BadgerDB options also base on anytype config
	opts := badger.DefaultOptions(storePath).
		// Core settings
		WithLogger(badgerLogger{}).
		WithMetricsEnabled(false). // Metrics are not used
		WithNumGoroutines(1).      // We don't use Stream
		// Memory and cache settings
		WithNumMemtables(4).          // Reduced to save memory, total possible MemTableSize(64MB)*NumMemtables if they are all full
		WithBlockCacheSize(64 << 20). // Block cache, we don't have reading same block frequently
		// LSM tree settings
		WithBaseTableSize(8 << 20).     // 8MB SSTable size
		WithNumLevelZeroTables(2).      // Match NumMemtables
		WithNumLevelZeroTablesStall(8). // Stall threshold
		WithNumCompactors(2).           // 2 compactors to reduce CPU usage
		// Value log settings
		WithValueLogFileSize(512 << 20). // 512MB per value log file
		// Disable compression since data is already encrypted
		WithCompression(options.None).
		WithZSTDCompressionLevel(0)

	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to open badger db: %w", err)
	}

	r.badgerDB = db

	// Start garbage collection in background
	go r.runGC(ctx)
	return fileproto.DRPCRegisterFile(r.dRPC, r)
}

func (r *lightfilenoderpc) Close(ctx context.Context) error {
	return r.badgerDB.Close()
}

// badgerLogger implements badger.Logger interface to redirect BadgerDB logs
// to our application logger with appropriate prefixes
type badgerLogger struct{}

func (b badgerLogger) Errorf(s string, i ...interface{}) {
	log.Error("badger: " + fmt.Sprintf(s, i...))
}

func (b badgerLogger) Warningf(s string, i ...interface{}) {
	log.Warn("badger: " + fmt.Sprintf(s, i...))
}

func (b badgerLogger) Infof(s string, i ...interface{}) {
	log.Info("badger: " + fmt.Sprintf(s, i...))
}

func (b badgerLogger) Debugf(s string, i ...interface{}) {
	log.Debug("badger: " + fmt.Sprintf(s, i...))
}

// TODO: Read about garbage collection in badger, looks like we don't need it often
// because here not a lot of deletions
func (r *lightfilenoderpc) runGC(ctx context.Context) {
	log.Info("starting badger garbage collection routine")
	ticker := time.NewTicker(29 * time.Minute) // Prime number interval, so it won't overlap with other routines (hopefully)
	defer ticker.Stop()

	const maxGCDuration = time.Minute

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping badger garbage collection routine")
			return
		case <-ticker.C:
			log.Debug("running badger garbage collection")
			start := time.Now()
			gcCount := 0

			for time.Since(start) < maxGCDuration {
				// Try to reclaim space if at least 50% of a value log file can be garbage collected
				err := r.badgerDB.RunValueLogGC(0.5)
				if err != nil {
					if errors.Is(err, badger.ErrNoRewrite) {
						break // No more files to GC
					}

					log.Warn("badger gc failed", zap.Error(err))
					break
				}
				gcCount++
			}

			log.Info("badger garbage collection completed",
				zap.Duration("duration", time.Since(start)),
				zap.Int("filesGCed", gcCount))
		}
	}
}

//
// Component methods
//

func (r *lightfilenoderpc) BlockGet(ctx context.Context, req *fileproto.BlockGetRequest) (*fileproto.BlockGetResponse, error) {
	log.InfoCtx(ctx, "BlockGet",
		zap.String("spaceId", req.SpaceId),
		zap.String("cid", string(req.Cid)),
		zap.Bool("wait", req.Wait),
	)

	c, err := cid.Cast(req.Cid)
	if err != nil {
		return nil, err
	}

	b, err := r.getBlock(ctx, c, req.Wait)
	if err != nil {
		return nil, err
	}

	resp := &fileproto.BlockGetResponse{
		Cid:  req.Cid,
		Data: b.data,
	}

	return resp, nil
}

func (r *lightfilenoderpc) BlockPush(ctx context.Context, request *fileproto.BlockPushRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "BlockPush",
		zap.String("spaceId", request.SpaceId),
		zap.String("fileId", request.FileId),
		zap.String("cid", string(request.Cid)),
		zap.Int("dataSize", len(request.Data)),
	)

	// For counter update
	r.badgerDB.GetMergeOperator() // for counter update
	r.badgerDB.NewWriteBatch()    // ??? use it ???

	return &fileproto.Ok{}, nil
}

func (r *lightfilenoderpc) BlocksCheck(ctx context.Context, request *fileproto.BlocksCheckRequest) (*fileproto.BlocksCheckResponse, error) {
	log.InfoCtx(ctx, "BlocksCheck",
		zap.String("spaceId", request.SpaceId),
		zap.Strings("cids", byteSlicesToStrings(request.Cids)),
	)

	return &fileproto.BlocksCheckResponse{
		BlocksAvailability: []*fileproto.BlockAvailability{
			{
				Cid:    request.Cids[0],
				Status: fileproto.AvailabilityStatus_NotExists,
			},
		},
	}, nil
}

func (r *lightfilenoderpc) BlocksBind(ctx context.Context, request *fileproto.BlocksBindRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "BlocksBind",
		zap.String("spaceId", request.SpaceId),
		zap.String("fileId", request.FileId),
		zap.Strings("cids", byteSlicesToStrings(request.Cids)),
	)

	return &fileproto.Ok{}, nil
}

func (r *lightfilenoderpc) FilesDelete(ctx context.Context, request *fileproto.FilesDeleteRequest) (*fileproto.FilesDeleteResponse, error) {
	log.InfoCtx(ctx, "FilesDelete",
		zap.String("spaceId", request.SpaceId),
		zap.Strings("fileIds", request.FileIds),
	)

	r.badgerDB.NewWriteBatch()

	err := r.badgerDB.Update(func(txn *badger.Txn) error {
		// For each file, decrement reference counter and delete the key
		prefix := []byte(fmt.Sprintf("l:%s.%s:", request.SpaceId, request.FileIds[0]))
		opts := badger.IteratorOptions{PrefetchSize: 1000}

		it := txn.NewIterator(opts)
		defer it.Close()

		batch := txn.NewBatch()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			cid := extractCID(it.Item().Key())

			batch.Merge(rcKey(cid), uint64ToBytes(^uint64(0))) // Decrement
			batch.Delete(it.Item().Key())

			if batch.Len() >= 1000 {
				if err := batch.Flush(); err != nil {
					return err
				}
				batch = txn.NewBatch()
			}
		}
		return batch.Flush()
	})

	return &fileproto.FilesDeleteResponse{}, nil
}

func (r *lightfilenoderpc) FilesInfo(ctx context.Context, request *fileproto.FilesInfoRequest) (*fileproto.FilesInfoResponse, error) {
	log.InfoCtx(ctx, "FilesInfo",
		zap.String("spaceId", request.SpaceId),
		zap.Strings("fileIds", request.FileIds),
	)

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

func (r *lightfilenoderpc) FilesGet(request *fileproto.FilesGetRequest, stream fileproto.DRPCFile_FilesGetStream) error {
	log.InfoCtx(context.Background(), "FilesGet",
		zap.String("spaceId", request.SpaceId),
	)
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

func (r *lightfilenoderpc) Check(ctx context.Context, request *fileproto.CheckRequest) (*fileproto.CheckResponse, error) {
	log.InfoCtx(ctx, "Check")

	return &fileproto.CheckResponse{
		SpaceIds:   []string{"space1", "space2"},
		AllowWrite: true,
	}, nil
}

func (r *lightfilenoderpc) SpaceInfo(ctx context.Context, request *fileproto.SpaceInfoRequest) (*fileproto.SpaceInfoResponse, error) {
	log.InfoCtx(ctx, "SpaceInfo",
		zap.String("spaceId", request.SpaceId),
	)

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

func (r *lightfilenoderpc) AccountInfo(ctx context.Context, request *fileproto.AccountInfoRequest) (*fileproto.AccountInfoResponse, error) {
	log.InfoCtx(ctx, "AccountInfo")

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

func (r *lightfilenoderpc) AccountLimitSet(ctx context.Context, request *fileproto.AccountLimitSetRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "AccountLimitSet",
		zap.String("identity", request.Identity),
		zap.Uint64("limit", request.Limit),
	)

	return &fileproto.Ok{}, nil
}

func (r *lightfilenoderpc) SpaceLimitSet(ctx context.Context, request *fileproto.SpaceLimitSetRequest) (*fileproto.Ok, error) {
	log.InfoCtx(ctx, "SpaceLimitSet",
		zap.String("spaceId", request.SpaceId),
		zap.Uint64("limit", request.Limit),
	)

	return &fileproto.Ok{}, nil
}

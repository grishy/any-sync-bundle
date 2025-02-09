package rpc

import (
	"context"
	"time"

	consensus "github.com/anyproto/any-sync-consensusnode"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/consensus/consensusproto"
	"github.com/anyproto/any-sync/consensus/consensusproto/consensuserr"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/any-sync/util/cidutil"

	"go.uber.org/zap"

	lightDb "github.com/grishy/any-sync-bundle/light/consensus/db"
)

const CName = "light.consensus.consensusrpc"

// TODO: Maybe use original RPC and just implement Database Interface
var (
	logRpc = logger.NewNamed(CName)
	okResp = &consensusproto.Ok{}
)

type dbSrv interface {
	// AddLog adds new log db
	AddLog(ctx context.Context, log consensus.Log) (err error)
	// DeleteLog deletes the log
	DeleteLog(ctx context.Context, logId string) error
	// AddRecord adds new record to existing log
	// returns consensuserr.ErrConflict if record didn't match or log not found
	AddRecord(ctx context.Context, logId string, record consensus.Record) (err error)
	// FetchLog gets log by id
	FetchLog(ctx context.Context, logId string) (log consensus.Log, err error)
}

type lightRpc struct {
	db       dbSrv
	nodeconf nodeconf.Service
	dRPC     server.DRPCServer
}

func New() *lightRpc {
	return &lightRpc{}
}

//
// App Component
//

func (c *lightRpc) Init(a *app.App) error {
	logRpc.Info("call Init")

	c.db = a.MustComponent(lightDb.CName).(dbSrv)
	c.nodeconf = a.MustComponent(nodeconf.CName).(nodeconf.Service)
	c.dRPC = a.MustComponent(server.CName).(server.DRPCServer)

	return nil
}

func (c *lightRpc) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (c *lightRpc) Run(_ context.Context) error {
	logRpc.Info("call Run")

	// TODO: Create a ticket to move call of this in original code to the Run stage, to avoid call any logic in Init
	return consensusproto.DRPCRegisterConsensus(c.dRPC, c)
}

func (c *lightRpc) Close(ctx context.Context) error {
	logRpc.Info("call Close")

	return nil
}

//
// ConsensusRPC
//

func (c *lightRpc) LogAdd(ctx context.Context, req *consensusproto.LogAddRequest) (*consensusproto.Ok, error) {
	logRpc.InfoCtx(ctx, "call LogAdd", zap.String("request", req.String()))

	if err := c.checkWrite(ctx); err != nil {
		return nil, err
	}

	if req.GetRecord() == nil || !cidutil.VerifyCid(req.Record.Payload, req.Record.Id) {
		return nil, consensuserr.ErrInvalidPayload
	}

	// we don't sign the first record because it affects the id, but we sign the following records as a confirmation that the chain is valid and the record added from a valid source
	l := consensus.Log{
		Id: req.LogId,
		Records: []consensus.Record{
			{
				Id:      req.Record.Id,
				Payload: req.Record.Payload,
				Created: time.Now(),
			},
		},
	}

	_ = l

	return okResp, nil
}

func (c *lightRpc) RecordAdd(ctx context.Context, request *consensusproto.RecordAddRequest) (*consensusproto.RawRecordWithId, error) {
	logRpc.InfoCtx(ctx, "call RecordAdd", zap.String("request", request.String()))

	// TODO implement me
	panic("implement me")
}

func (c *lightRpc) LogWatch(stream consensusproto.DRPCConsensus_LogWatchStream) error {
	logRpc.Info("call LogWatch")

	// TODO implement me
	panic("implement me")
}

func (c *lightRpc) LogDelete(ctx context.Context, request *consensusproto.LogDeleteRequest) (*consensusproto.Ok, error) {
	logRpc.InfoCtx(ctx, "call LogDelete", zap.String("request", request.String()))

	// TODO implement me
	panic("implement me")
}

//
// Helper from original code
//

func (c *lightRpc) checkRead(ctx context.Context) (err error) {
	peerId, err := peer.CtxPeerId(ctx)
	if err != nil {
		return consensuserr.ErrForbidden
	}

	nodeTypes := c.nodeconf.NodeTypes(peerId)
	for _, nodeType := range nodeTypes {
		switch nodeType {
		case nodeconf.NodeTypeCoordinator,
			nodeconf.NodeTypeTree,
			nodeconf.NodeTypeFile:
			return nil
		}
	}

	return consensuserr.ErrForbidden
}

func (c *lightRpc) checkWrite(ctx context.Context) (err error) {
	peerId, err := peer.CtxPeerId(ctx)
	if err != nil {
		return consensuserr.ErrForbidden
	}

	nodeTypes := c.nodeconf.NodeTypes(peerId)
	for _, nodeType := range nodeTypes {
		switch nodeType {
		case nodeconf.NodeTypeCoordinator,
			nodeconf.NodeTypeTree:
			return nil
		}
	}

	return consensuserr.ErrForbidden
}

func recordsToProto(recs []consensus.Record) []*consensusproto.RawRecordWithId {
	res := make([]*consensusproto.RawRecordWithId, len(recs))
	for i, rec := range recs {
		res[i] = &consensusproto.RawRecordWithId{
			Payload: rec.Payload,
			Id:      rec.Id,
		}
	}

	return res
}

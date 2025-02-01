package consensus

import (
	"context"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/consensus/consensusproto"
	"github.com/anyproto/any-sync/net/rpc/server"
	"go.uber.org/zap"
)

var logRpc = logger.NewNamed("bundle.consensus.consensusrpc")

type bundleRpc struct {
	dRPC server.DRPCServer
}

//
// App.Run
//

func (c *bundleRpc) Init(a *app.App) error {
	logRpc.Info("call Init")

	c.dRPC = a.MustComponent(server.CName).(server.DRPCServer)

	return nil
}

func (c *bundleRpc) Name() (name string) {
	return "bundle.consensus.consensusrpc"
}

func (c *bundleRpc) Run(_ context.Context) error {
	logRpc.Info("call Run")

	// TODO: Create a ticket to move call of this in original code to the Run stage, to avoid call any logic in Init
	return consensusproto.DRPCRegisterConsensus(c.dRPC, c)
}

func (c *bundleRpc) Close(ctx context.Context) error {
	logRpc.Info("call Close")

	return nil
}

//
// ConsensusRPC
//

func (c *bundleRpc) LogAdd(ctx context.Context, request *consensusproto.LogAddRequest) (*consensusproto.Ok, error) {
	logRpc.InfoCtx(ctx, "call LogAdd", zap.String("request", request.String()))

	// TODO implement me
	panic("implement me")
}

func (c *bundleRpc) RecordAdd(ctx context.Context, request *consensusproto.RecordAddRequest) (*consensusproto.RawRecordWithId, error) {
	logRpc.InfoCtx(ctx, "call RecordAdd", zap.String("request", request.String()))

	// TODO implement me
	panic("implement me")
}

func (c *bundleRpc) LogWatch(stream consensusproto.DRPCConsensus_LogWatchStream) error {
	logRpc.Info("call LogWatch")

	// TODO implement me
	panic("implement me")
}

func (c *bundleRpc) LogDelete(ctx context.Context, request *consensusproto.LogDeleteRequest) (*consensusproto.Ok, error) {
	logRpc.InfoCtx(ctx, "call LogDelete", zap.String("request", request.String()))

	// TODO implement me
	panic("implement me")
}

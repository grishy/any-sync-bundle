package app

import (
	"os"

	"github.com/anyproto/any-sync-consensusnode/account"
	"github.com/anyproto/any-sync-consensusnode/config"
	"github.com/anyproto/any-sync-consensusnode/consensusrpc"
	"github.com/anyproto/any-sync-consensusnode/db"
	"github.com/anyproto/any-sync-consensusnode/stream"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/coordinator/coordinatorclient"
	"github.com/anyproto/any-sync/coordinator/nodeconfsource"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/any-sync/nodeconf/nodeconfstore"
	"go.uber.org/zap"
)

func NewConsensusApp(log logger.CtxLogger, cfg *config.Config) *app.App {
	// TODO: Remove when merged https://github.com/anyproto/any-sync/pull/374
	if err := os.MkdirAll(cfg.NetworkStorePath, 0o775); err != nil {
		log.Panic("can't create directory network store", zap.Error(err))
	}

	a := new(app.App).
		Register(cfg).
		Register(metric.New()).
		Register(account.New()).
		Register(nodeconf.New()).
		Register(nodeconfstore.New()).
		Register(nodeconfsource.New()).
		Register(coordinatorclient.New()).
		Register(pool.New()).
		Register(peerservice.New()).
		Register(yamux.New()).
		Register(quic.New()).
		Register(secureservice.New()).
		Register(server.New()).
		Register(db.New()).
		Register(stream.New()).
		Register(consensusrpc.New())

	return a
}

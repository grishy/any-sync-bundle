package node

import (
	"github.com/anyproto/any-sync-consensusnode/account"
	"github.com/anyproto/any-sync-consensusnode/config"
	"github.com/anyproto/any-sync-consensusnode/consensusrpc"
	"github.com/anyproto/any-sync-consensusnode/db"
	"github.com/anyproto/any-sync-consensusnode/stream"

	"github.com/anyproto/any-sync/app"
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
)

func NewConsensusApp(cfg *config.Config) *app.App {
	MustMkdirAll(cfg.NetworkStorePath)

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

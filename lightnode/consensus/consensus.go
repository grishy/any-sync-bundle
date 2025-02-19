package consensus

import (
	"github.com/anyproto/any-sync-consensusnode/account"
	"github.com/anyproto/any-sync-consensusnode/config"
	"github.com/anyproto/any-sync-consensusnode/consensusrpc"
	"github.com/anyproto/any-sync-consensusnode/db"
	"github.com/anyproto/any-sync-consensusnode/stream"
	"github.com/anyproto/any-sync/coordinator/nodeconfsource"
	"github.com/anyproto/any-sync/metric"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/coordinator/coordinatorclient"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/any-sync/nodeconf/nodeconfstore"

	"github.com/grishy/any-sync-bundle/lightcmp/lightconfig"
	"github.com/grishy/any-sync-bundle/lightcmp/lightconsensusdb"
	"github.com/grishy/any-sync-bundle/lightcmp/lightconsensusrpc"
	"github.com/grishy/any-sync-bundle/lightcmp/lightnodeconf"
	"github.com/grishy/any-sync-bundle/lightnode"
)

func NewApp(cfg *config.Config) *app.App {
	// TODO: Add limiter to server?

	lCfg := &lightconfig.LightConfig{
		Account:         cfg.Account,
		Network:         cfg.Network,
		ListenTCPAddr:   cfg.Yamux.ListenAddrs,
		ListenUDPAddr:   cfg.Quic.ListenAddrs,
		ConsensusDBPath: "consensus.db",
	}

	a := new(app.App).
		Register(lCfg).
		Register(lightnodeconf.New()).
		Register(lightconsensusdb.New()).
		Register(lightconsensusrpc.New()).
		// TODO: Remove pool and peerservice, when will have all nodes on one port?
		Register(account.New()).       // Original, to sign messages
		Register(pool.New()).          // Original, provide pool of peers for 'peerservice'
		Register(peerservice.New()).   // Original, provide accepter for 'yamux' and 'quic'
		Register(yamux.New()).         // Original, TCP transport
		Register(quic.New()).          // Original, UDP transport
		Register(secureservice.New()). // Original, secure service on top of 'yamux' and 'quic'
		Register(server.New())         // Original, allow to register RPC services

	return a
}

func NewConsensusApp(cfg *config.Config) *app.App {
	lightnode.MustMkdirAll(cfg.NetworkStorePath)

	a := new(app.App).
		Register(cfg).
		Register(account.New()).
		Register(db.New()).
		Register(metric.New()).
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
		Register(stream.New()).
		Register(consensusrpc.New())

	return a
}

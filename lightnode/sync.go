package lightnode

import (
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/debugstat"
	"github.com/anyproto/any-sync/commonspace"
	"github.com/anyproto/any-sync/commonspace/credentialprovider"
	"github.com/anyproto/any-sync/consensus/consensusclient"
	"github.com/anyproto/any-sync/coordinator/coordinatorclient"
	"github.com/anyproto/any-sync/coordinator/nodeconfsource"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc/debugserver"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/streampool"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/node/nodeclient"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/any-sync/nodeconf/nodeconfstore"
	"github.com/anyproto/any-sync/util/syncqueues"

	"github.com/anyproto/any-sync-node/account"
	"github.com/anyproto/any-sync-node/config"
	"github.com/anyproto/any-sync-node/debug/nodedebugrpc"
	"github.com/anyproto/any-sync-node/nodehead"
	"github.com/anyproto/any-sync-node/nodespace"
	"github.com/anyproto/any-sync-node/nodespace/nodecache"
	"github.com/anyproto/any-sync-node/nodespace/peermanager"
	"github.com/anyproto/any-sync-node/nodespace/spacedeleter"
	"github.com/anyproto/any-sync-node/nodestorage"
	"github.com/anyproto/any-sync-node/nodesync"
	"github.com/anyproto/any-sync-node/nodesync/coldsync"
	"github.com/anyproto/any-sync-node/nodesync/hotsync"
)

func NewSyncNode(cfg *config.Config) *app.App {
	MustMkdirAll(cfg.NetworkStorePath)

	a := new(app.App).
		Register(cfg).
		Register(account.New()).
		Register(metric.New()).
		Register(debugstat.New()).
		Register(credentialprovider.NewNoOp()).
		Register(coordinatorclient.New()).
		Register(nodeconfstore.New()).
		Register(nodeconfsource.New()).
		Register(nodeconf.New()).
		Register(nodestorage.New()).
		Register(syncqueues.New()).
		Register(server.New()).
		Register(peerservice.New()).
		Register(pool.New()).
		Register(nodeclient.New()).
		Register(consensusclient.New()).
		Register(nodespace.NewStreamOpener()).
		Register(streampool.New()).
		Register(nodehead.New()).
		Register(nodecache.New(200)).
		Register(hotsync.New()).
		Register(coldsync.New()).
		Register(nodesync.New()).
		Register(secureservice.New()).
		Register(commonspace.New()).
		Register(nodespace.New()).
		Register(spacedeleter.New()).
		Register(peermanager.New()).
		Register(debugserver.New()).
		Register(nodedebugrpc.New()).
		Register(quic.New()).
		Register(yamux.New())

	return a
}

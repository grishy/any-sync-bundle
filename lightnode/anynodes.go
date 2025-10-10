package lightnode

import (
	consensusnodeConfig "github.com/anyproto/any-sync-consensusnode/config"
	"github.com/anyproto/any-sync-consensusnode/consensusrpc"
	consensusnodeDB "github.com/anyproto/any-sync-consensusnode/db"
	"github.com/anyproto/any-sync-consensusnode/stream"

	coordinatorAccount "github.com/anyproto/any-sync-coordinator/account"
	"github.com/anyproto/any-sync-coordinator/accountlimit"
	"github.com/anyproto/any-sync-coordinator/acleventlog"
	coordinatorConfig "github.com/anyproto/any-sync-coordinator/config"
	"github.com/anyproto/any-sync-coordinator/coordinator"
	"github.com/anyproto/any-sync-coordinator/coordinatorlog"
	"github.com/anyproto/any-sync-coordinator/db"
	"github.com/anyproto/any-sync-coordinator/deletionlog"
	"github.com/anyproto/any-sync-coordinator/identityrepo"
	coordinatorNodeconfsource "github.com/anyproto/any-sync-coordinator/nodeconfsource"
	"github.com/anyproto/any-sync-coordinator/spacestatus"

	filenodeConfig "github.com/anyproto/any-sync-filenode/config"
	"github.com/anyproto/any-sync-filenode/deletelog"
	"github.com/anyproto/any-sync-filenode/filenode"
	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/redisprovider"
	filenodeStat "github.com/anyproto/any-sync-filenode/stat"

	"github.com/anyproto/any-sync/acl"
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

	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

// newCoordinatorApp creates a coordinator application instance.
// This is the primary app that creates the full network stack.
func newCoordinatorApp(cfg *coordinatorConfig.Config) *app.App {
	a := new(app.App).
		Register(cfg).
		Register(db.New()).
		Register(metric.New()).
		Register(coordinatorAccount.New()).

		// Configuration
		Register(nodeconfstore.New()).
		Register(nodeconf.New()).
		Register(coordinatorNodeconfsource.New()).

		// Data
		Register(deletionlog.New()).

		// Security & Transport
		Register(secureservice.New()).
		Register(yamux.New()).
		Register(quic.New()).

		// Network Services
		Register(peerservice.New()).
		Register(pool.New()).
		Register(server.New()).

		// Logging & Monitoring
		Register(coordinatorlog.New()).
		Register(acleventlog.New()).
		Register(spacestatus.New()).

		// Service Logic
		Register(consensusclient.New()).
		Register(acl.New()).
		Register(accountlimit.New()).
		Register(identityrepo.New()).
		Register(coordinator.New())

	return a
}

// newSyncApp creates a sync node application instance with shared network.
// Only modern nodestorage (any-store format) is included; legacy oldstorage and migrator are omitted.
func newSyncApp(cfg *config.Config, net *sharedNetwork) *app.App {
	return new(app.App).
		Register(cfg).
		Register(net.Account).
		Register(net.Metric).
		Register(debugstat.New()).
		Register(credentialprovider.NewNoOp()).
		Register(nodeconfsource.New()).

		// Shared network components
		Register(net.NodeConfStore).
		Register(net.NodeConf).
		Register(net.SecureService).
		Register(net.Quic).
		Register(net.Yamux).
		Register(net.Server).
		Register(net.PeerService).
		Register(net.Pool).

		// Network clients
		Register(coordinatorclient.New()).
		Register(nodeclient.New()).
		Register(consensusclient.New()).

		// Storage (depends on config)
		// oldstorage.New() - SKIPPED: Not needed for new installations (legacy BadgerDB format)
		Register(nodestorage.New()).
		// migrator.New() - SKIPPED: Not needed for new installations (migrates oldstorage → nodestorage)
		Register(syncqueues.New()).

		// Space Sync
		Register(nodespace.NewStreamOpener()).
		Register(streampool.New()).
		Register(nodehead.New()).
		Register(nodecache.New(200)).
		Register(hotsync.New()).
		Register(coldsync.New()).
		Register(nodesync.New()).

		// Space Services
		Register(commonspace.New()).
		Register(nodespace.New()).
		Register(spacedeleter.New()).
		Register(peermanager.New()).

		// Debug
		Register(debugserver.New()).
		Register(nodedebugrpc.New())
}

// newFileNodeApp creates a filenode application instance with shared network.
func newFileNodeApp(cfg *filenodeConfig.Config, fileDir string, net *sharedNetwork) *app.App {
	return new(app.App).
		Register(cfg).
		Register(net.Account).
		Register(filenodeStat.New()).
		Register(nodeconfsource.New()).

		// Shared network components
		Register(net.NodeConfStore).
		Register(net.NodeConf).
		Register(net.SecureService).
		Register(net.Yamux).
		Register(net.Quic).
		Register(net.PeerService).
		Register(net.Pool).
		Register(net.Server).
		Register(net.Metric).

		// Network clients
		Register(coordinatorclient.New()).
		Register(consensusclient.New()).
		Register(acl.New()).

		// File Storage
		// store() - REPLACED: lightfilenodestore.New() uses BadgerDB instead of original S3/MinIO store
		Register(lightfilenodestore.New(fileDir)).
		Register(redisprovider.New()).
		Register(index.New()).

		// Service Logic
		Register(filenode.New()).
		Register(deletelog.New())
}

// newConsensusApp creates a consensus node application instance with shared network.
func newConsensusApp(cfg *consensusnodeConfig.Config, net *sharedNetwork) *app.App {
	return new(app.App).
		Register(cfg).
		Register(net.Account).
		Register(nodeconfsource.New()).

		// Shared network components
		Register(net.NodeConfStore).
		Register(net.NodeConf).
		Register(net.SecureService).
		Register(net.Yamux).
		Register(net.Quic).
		Register(net.Server).
		Register(net.Pool).
		Register(net.PeerService).
		Register(net.Metric).

		// Network clients
		Register(coordinatorclient.New()).

		// Storage & Service Logic
		Register(consensusnodeDB.New()).
		Register(stream.New()).
		Register(consensusrpc.New())
}

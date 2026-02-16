package lightnode

import (
	consensusnodeConfig "github.com/anyproto/any-sync-consensusnode/config"
	"github.com/anyproto/any-sync-consensusnode/consensusrpc"
	consensusnodeDB "github.com/anyproto/any-sync-consensusnode/db"
	consensusdeletelog "github.com/anyproto/any-sync-consensusnode/deletelog"
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
	"github.com/anyproto/any-sync-coordinator/inbox"
	coordinatorNodeconfsource "github.com/anyproto/any-sync-coordinator/nodeconfsource"
	"github.com/anyproto/any-sync-coordinator/spacestatus"
	"github.com/anyproto/any-sync-coordinator/subscribe"

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

	"github.com/anyproto/any-sync-node/archive"
	"github.com/anyproto/any-sync-node/archive/archivestore"
	"github.com/anyproto/any-sync-node/config"
	"github.com/anyproto/any-sync-node/debug/nodedebugrpc"
	"github.com/anyproto/any-sync-node/debug/spacechecker"
	"github.com/anyproto/any-sync-node/nodehead"
	"github.com/anyproto/any-sync-node/nodespace"
	"github.com/anyproto/any-sync-node/nodespace/nodecache"
	"github.com/anyproto/any-sync-node/nodespace/peermanager"
	"github.com/anyproto/any-sync-node/nodespace/spacedeleter"
	"github.com/anyproto/any-sync-node/nodestorage"
	"github.com/anyproto/any-sync-node/nodesync"
	"github.com/anyproto/any-sync-node/nodesync/coldsync"
	"github.com/anyproto/any-sync-node/nodesync/hotsync"

	"github.com/anyproto/any-sync-filenode/store/s3store"
	"github.com/anyproto/any-sync/app/logger"
	"go.uber.org/zap"

	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

var log = logger.NewNamed("lightnode")

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
		Register(subscribe.New()).
		Register(inbox.New()).
		Register(accountlimit.New()).
		Register(identityrepo.New()).
		Register(coordinator.New())

	return a
}

// newSyncApp creates a sync node application instance with shared components.
// Only modern nodestorage (any-store format) is included; legacy oldstorage and migrator are omitted.
func newSyncApp(cfg *config.Config, net *sharedCmp) *app.App {
	return new(app.App).
		Register(cfg).
		Register(net.Account).
		Register(net.Metric).
		Register(debugstat.New()).
		Register(credentialprovider.NewNoOp()).
		Register(nodeconfsource.New()).

		// Shared components
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
		Register(spacechecker.New()).
		Register(nodedebugrpc.New()).

		// Archive
		Register(archivestore.New()).
		Register(archive.New())
}

// selectFileStore returns S3 or BadgerDB storage based on configuration.
func selectFileStore(cfg *filenodeConfig.Config, fileDir string) app.Component {
	if cfg.S3Store.Bucket != "" {
		log.Info("using S3 storage backend",
			zap.String("bucket", cfg.S3Store.Bucket),
			zap.String("endpoint", cfg.S3Store.Endpoint))
		return s3store.New()
	}
	log.Info("using BadgerDB storage backend", zap.String("path", fileDir))
	return lightfilenodestore.New(fileDir)
}

// newFileNodeApp creates a filenode application instance with shared components.
func newFileNodeApp(cfg *filenodeConfig.Config, fileDir string, net *sharedCmp) *app.App {
	return new(app.App).
		Register(cfg).
		Register(net.Account).
		Register(filenodeStat.New()).
		Register(nodeconfsource.New()).

		// Shared components
		Register(net.NodeConfStore).
		Register(net.NodeConf).
		Register(net.SecureService).
		Register(net.Yamux).
		Register(net.Quic).
		Register(net.PeerService).
		Register(net.Pool).
		Register(net.Server).
		Register(net.Metric).
		Register(net.ACL).

		// Network clients
		Register(coordinatorclient.New()).
		Register(consensusclient.New()).

		// File Storage - S3 (if configured) or BadgerDB (default)
		Register(selectFileStore(cfg, fileDir)).
		Register(redisprovider.New()).
		Register(index.New()).

		// Service Logic
		Register(filenode.New()).
		Register(deletelog.New())
}

// newConsensusApp creates a consensus node application instance with shared components.
func newConsensusApp(cfg *consensusnodeConfig.Config, net *sharedCmp) *app.App {
	return new(app.App).
		Register(cfg).
		Register(net.Account).
		Register(nodeconfsource.New()).

		// Shared components
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
		Register(consensusrpc.New()).

		// Deletion Log
		Register(consensusdeletelog.New())
}

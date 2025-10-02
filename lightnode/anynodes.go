package lightnode

import (
	consensusnodeAccount "github.com/anyproto/any-sync-consensusnode/account"
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

	filenodeAccount "github.com/anyproto/any-sync-filenode/account"
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

	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

// NewSyncApp creates a sync node application instance.
//
// Storage architecture: We ONLY include nodestorage, which uses AnyStorePath (modern any-store format).
// The oldstorage component (uses Path field for legacy BadgerDB format) is NOT included - new installations only.
// The migrator component is also NOT included - it migrates from oldstorage to nodestorage, unnecessary here.
// Result: config.Storage.Path field is unused, only config.Storage.AnyStorePath is accessed.
//
// Component registration order is based on actual initialization dependencies.
// Order is CRITICAL - components can only access previously registered components in Init().
func NewSyncApp(cfg *config.Config) *app.App {
	a := new(app.App).
		Register(cfg).

		// Foundation (no dependencies)
		Register(account.New()).
		Register(metric.New()).
		Register(debugstat.New()).
		Register(credentialprovider.NewNoOp()).

		// Configuration (depends on foundation)
		Register(coordinatorclient.New()).
		Register(nodeconfstore.New()).
		Register(nodeconfsource.New()).
		Register(nodeconf.New()).

		// Storage (depends on config)
		// oldstorage.New() - SKIPPED: Not needed for new installations (legacy BadgerDB format)
		Register(nodestorage.New()).
		// migrator.New() - SKIPPED: Not needed for new installations (migrates oldstorage → nodestorage)
		Register(syncqueues.New()).

		// Security & Transport (secureservice MUST be before yamux/quic - they require it in Init)
		Register(secureservice.New()).
		Register(quic.New()).
		Register(yamux.New()).

		// Network Services (depend on transport)
		Register(server.New()).
		Register(peerservice.New()).
		Register(pool.New()).
		Register(nodeclient.New()).
		Register(consensusclient.New()).

		// Space Sync (depends on network)
		Register(nodespace.NewStreamOpener()).
		Register(streampool.New()).
		Register(nodehead.New()).
		Register(nodecache.New(200)).
		Register(hotsync.New()).
		Register(coldsync.New()).
		Register(nodesync.New()).

		// Space Services (depend on sync)
		Register(commonspace.New()).
		Register(nodespace.New()).
		Register(spacedeleter.New()).
		Register(peermanager.New()).

		// Debug (can be last)
		Register(debugserver.New()).
		Register(nodedebugrpc.New())

	return a
}

// NewFileNodeApp creates a filenode application instance.
//
// Component registration order is based on actual initialization dependencies.
// Order is CRITICAL - components can only access previously registered components in Init().
func NewFileNodeApp(cfg *filenodeConfig.Config, fileDir string) *app.App {
	a := new(app.App).
		Register(cfg).

		// Foundation (no dependencies)
		Register(filenodeAccount.New()).
		Register(filenodeStat.New()).
		Register(metric.New()).

		// Configuration (depends on foundation)
		Register(nodeconfsource.New()).
		Register(nodeconfstore.New()).
		Register(nodeconf.New()).

		// Security & Transport (secureservice MUST be before yamux/quic - they require it in Init)
		Register(secureservice.New()).
		Register(yamux.New()).
		Register(quic.New()).

		// Network Services (depend on transport)
		Register(peerservice.New()).
		Register(pool.New()).
		Register(coordinatorclient.New()).
		Register(consensusclient.New()).
		Register(acl.New()).

		// File Storage (depends on network)
		// store() - REPLACED: lightfilenodestore.New() uses BadgerDB instead of original S3/MinIO store
		Register(lightfilenodestore.New(fileDir)).
		Register(redisprovider.New()).
		Register(index.New()).

		// Service Logic (depends on storage)
		Register(server.New()).
		Register(filenode.New()).
		Register(deletelog.New())

	return a
}

// NewCoordinatorApp creates a coordinator application instance.
//
// Note: No bootstrap needed - network config passed directly via YAML and client-config.yml file.
// Component registration order is based on actual initialization dependencies.
// Order is CRITICAL - components can only access previously registered components in Init().
func NewCoordinatorApp(cfg *coordinatorConfig.Config) *app.App {
	a := new(app.App).
		Register(cfg).

		// Foundation (db early for MongoDB, metric for telemetry)
		Register(db.New()).
		Register(metric.New()).
		Register(coordinatorAccount.New()).

		// Configuration (depends on foundation)
		Register(nodeconfstore.New()).
		Register(nodeconf.New()).
		Register(coordinatorNodeconfsource.New()).

		// Data (depends on config)
		Register(deletionlog.New()).

		// Security & Transport (secureservice MUST be before yamux/quic - they require it in Init)
		Register(secureservice.New()).
		Register(yamux.New()).
		Register(quic.New()).

		// Network Services (depend on transport)
		Register(peerservice.New()).
		Register(pool.New()).
		Register(server.New()).

		// Logging & Monitoring (depend on network)
		Register(coordinatorlog.New()).
		Register(acleventlog.New()).
		Register(spacestatus.New()).

		// Service Logic (depends on all infrastructure)
		Register(consensusclient.New()).
		Register(acl.New()).
		Register(accountlimit.New()).
		Register(identityrepo.New()).
		Register(coordinator.New())

	return a
}

// NewConsensusApp creates a consensus node application instance.
//
// Component registration order is based on actual initialization dependencies.
// Order is CRITICAL - components can only access previously registered components in Init().
func NewConsensusApp(cfg *consensusnodeConfig.Config) *app.App {
	a := new(app.App).
		Register(cfg).

		// Foundation (no dependencies)
		Register(metric.New()).
		Register(consensusnodeAccount.New()).

		// Configuration (depends on foundation)
		Register(nodeconf.New()).
		Register(nodeconfstore.New()).
		Register(nodeconfsource.New()).

		// Security & Transport (secureservice MUST be before yamux/quic - they require it in Init)
		Register(secureservice.New()).
		Register(yamux.New()).
		Register(quic.New()).

		// Network Services (depend on transport)
		Register(server.New()).
		Register(pool.New()).
		Register(peerservice.New()).

		// Clients (depend on network)
		Register(coordinatorclient.New()).

		// Storage & Service Logic (depend on all infrastructure)
		Register(consensusnodeDB.New()).
		Register(stream.New()).
		Register(consensusrpc.New())

	return a
}

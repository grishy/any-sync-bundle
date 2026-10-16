package lightnode

import (
	"context"

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
	"github.com/anyproto/any-sync-coordinator/fileusage"
	"github.com/anyproto/any-sync-coordinator/identityrepo"
	"github.com/anyproto/any-sync-coordinator/inbox"
	"github.com/anyproto/any-sync-coordinator/invitestore"
	coordinatorNodeconfsource "github.com/anyproto/any-sync-coordinator/nodeconfsource"
	"github.com/anyproto/any-sync-coordinator/spacestatus"
	"github.com/anyproto/any-sync-coordinator/subscribe"

	filenodeConfig "github.com/anyproto/any-sync-filenode/config"
	"github.com/anyproto/any-sync-filenode/deletelog"
	"github.com/anyproto/any-sync-filenode/filenode"
	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/redisprovider"
	filenodeStat "github.com/anyproto/any-sync-filenode/stat"
	"github.com/anyproto/any-sync-filenode/store/s3store"

	"github.com/anyproto/any-sync/acl"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/debugstat"
	"github.com/anyproto/any-sync/commonspace"
	"github.com/anyproto/any-sync/commonspace/credentialprovider"
	"github.com/anyproto/any-sync/consensus/consensusclient"
	"github.com/anyproto/any-sync/coordinator/coordinatorclient"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc/debugserver"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/streampool"
	"github.com/anyproto/any-sync/net/transport"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/node/nodeclient"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/any-sync/nodeconf/nodeconfstore"
	"github.com/anyproto/any-sync/util/syncqueues"

	"github.com/anyproto/any-sync-node/archive"
	"github.com/anyproto/any-sync-node/archive/adopter"
	"github.com/anyproto/any-sync-node/archive/archivestore"
	syncConfig "github.com/anyproto/any-sync-node/config"
	"github.com/anyproto/any-sync-node/debug/nodedebugrpc"
	"github.com/anyproto/any-sync-node/debug/spacechecker"
	"github.com/anyproto/any-sync-node/nodehead"
	"github.com/anyproto/any-sync-node/nodespace"
	"github.com/anyproto/any-sync-node/nodespace/nodecache"
	"github.com/anyproto/any-sync-node/nodespace/peermanager"
	"github.com/anyproto/any-sync-node/nodespace/pubsubrelay"
	"github.com/anyproto/any-sync-node/nodespace/spacedeleter"
	"github.com/anyproto/any-sync-node/nodestorage"
	"github.com/anyproto/any-sync-node/nodesync"
	"github.com/anyproto/any-sync-node/nodesync/coldsync"
	"github.com/anyproto/any-sync-node/nodesync/hotsync"
	"github.com/anyproto/any-sync-node/repairer"
	"github.com/anyproto/any-sync-node/resharder"

	"github.com/anyproto/any-sync/app/logger"
	"go.uber.org/zap"

	"github.com/grishy/any-sync-bundle/config"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

var log = logger.NewNamed("lightnode")

// Service is one upstream application of the bundle.
type Service struct {
	Name string
	App  *app.App
}

// Bundle is the four upstream applications running as one node.
//
// The coordinator owns the network: account, connection pool, DRPC server and
// transports. The other services are its child apps, so they resolve those
// components from the coordinator and start only their own. See
// https://pkg.go.dev/github.com/anyproto/any-sync@v0.13.8/app#App.ChildApp
//
// Transports accept requests as soon as they run, and a request that reaches a
// service before it runs can crash it, as upstream consensus found:
// https://github.com/anyproto/any-sync-consensusnode/commit/9dc0cb6862d13a879c87fb8088e156e26862c782
// So the coordinator only initializes them, and Listeners run after every
// service and close before any.
type Bundle struct {
	Services  []Service // start in order, close in reverse
	Listeners []app.ComponentRunnable
}

func NewBundle(cfgs *config.NodeConfigs) *Bundle {
	tcp, udp := yamux.New(), quic.New()
	coordinatorApp := newCoordinatorApp(cfgs.Coordinator, tcp, udp)

	return &Bundle{
		Services: []Service{
			{Name: "coordinator", App: coordinatorApp},
			{Name: "consensus", App: newConsensusApp(coordinatorApp.ChildApp(), cfgs.Consensus)},
			{Name: "filenode", App: newFileNodeApp(coordinatorApp.ChildApp(), cfgs.Filenode, cfgs.FilenodeStorePath)},
			{Name: "sync", App: newSyncApp(coordinatorApp.ChildApp(), cfgs.Sync)},
		},
		Listeners: []app.ComponentRunnable{tcp, udp},
	}
}

type transportComponent interface {
	transport.Transport
	app.ComponentRunnable
}

// deferredTransport lets the coordinator initialize a transport without
// listening; the bundle runs and closes it as a listener.
type deferredTransport struct{ transportComponent }

func (deferredTransport) Run(context.Context) error   { return nil }
func (deferredTransport) Close(context.Context) error { return nil }

// Component lists copy each upstream Bootstrap without the network components
// that the coordinator shares.

// https://github.com/anyproto/any-sync-coordinator/blob/v0.13.2/cmd/coordinator/coordinator.go
func newCoordinatorApp(cfg *coordinatorConfig.Config, tcp, udp transportComponent) *app.App {
	return new(app.App).
		Register(cfg).
		Register(db.New()).
		Register(metric.New()).
		Register(coordinatorAccount.New()).
		Register(nodeconfstore.New()).
		Register(nodeconf.New()).
		Register(coordinatorNodeconfsource.New()).
		Register(deletionlog.New()).
		Register(peerservice.New()).
		Register(pool.New()).
		Register(secureservice.New()).
		Register(server.New()).
		Register(coordinatorlog.New()).
		Register(acleventlog.New()).
		Register(spacestatus.New()).
		Register(consensusclient.New()).
		Register(acl.New()).
		Register(subscribe.New()).
		Register(inbox.New()).
		Register(accountlimit.New()).
		Register(fileusage.New()).
		Register(identityrepo.New()).
		Register(invitestore.New()).
		Register(coordinator.New()).
		Register(deferredTransport{tcp}).
		Register(deferredTransport{udp})
}

// https://github.com/anyproto/any-sync-consensusnode/blob/v0.13.3/cmd/consensusnode.go
func newConsensusApp(a *app.App, cfg *consensusnodeConfig.Config) *app.App {
	return a.
		Register(cfg).
		Register(consensusnodeDB.New()).
		Register(coordinatorclient.New()).
		Register(stream.New()).
		Register(consensusrpc.New()).
		Register(consensusdeletelog.New())
}

// https://github.com/anyproto/any-sync-filenode/blob/v0.13.2/cmd/filenode.go
//
// The filenode uses the coordinator's ACL service, so it needs no consensus
// client of its own.
func newFileNodeApp(a *app.App, cfg *filenodeConfig.Config, fileDir string) *app.App {
	return a.
		Register(cfg).
		Register(filenodeStat.New()).
		Register(coordinatorclient.New()).
		Register(selectFileStore(cfg, fileDir)).
		Register(redisprovider.New()).
		Register(index.New()).
		Register(filenode.New()).
		Register(deletelog.New())
}

// selectFileStore returns S3 or BadgerDB storage based on configuration.
func selectFileStore(cfg *filenodeConfig.Config, fileDir string) s3store.S3Store {
	if cfg.S3Store.Bucket != "" {
		log.Info("using S3 storage backend",
			zap.String("event", "filenode_storage_backend_s3"),
			zap.String("bucket", cfg.S3Store.Bucket),
			zap.String("endpoint", cfg.S3Store.Endpoint))
		return s3store.New()
	}
	log.Info("using BadgerDB storage backend",
		zap.String("event", "filenode_storage_backend_badger"),
		zap.String("path", fileDir))
	return lightfilenodestore.New(fileDir)
}

// https://github.com/anyproto/any-sync-node/blob/v0.13.4/cmd/any-sync-node.go
//
// The bundle has always stored sync data in the any-store format, so the
// upstream legacy storage and its migrator are omitted.
func newSyncApp(a *app.App, cfg *syncConfig.Config) *app.App {
	return a.
		Register(cfg).
		Register(debugstat.New()).
		Register(credentialprovider.NewNoOp()).
		Register(coordinatorclient.New()).
		Register(nodestorage.New()).
		Register(syncqueues.New()).
		Register(nodeclient.New()).
		Register(consensusclient.New()).
		Register(nodespace.NewStreamOpener()).
		Register(streampool.New()).
		Register(nodehead.New()).
		Register(nodecache.New(200)).
		Register(hotsync.New()).
		Register(coldsync.New()).
		Register(nodesync.New()).
		Register(commonspace.New()).
		Register(nodespace.New()).
		Register(pubsubrelay.New()).
		Register(spacedeleter.New()).
		Register(peermanager.New()).
		Register(debugserver.New()).
		Register(spacechecker.New()).
		Register(nodedebugrpc.New()).
		Register(archivestore.New()).
		Register(archive.New()).
		Register(adopter.New()).
		Register(resharder.New()).
		Register(repairer.New())
}

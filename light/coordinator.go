package light

import (
	"github.com/anyproto/any-sync-coordinator/account"
	"github.com/anyproto/any-sync-coordinator/accountlimit"
	"github.com/anyproto/any-sync-coordinator/acleventlog"
	"github.com/anyproto/any-sync-coordinator/config"
	"github.com/anyproto/any-sync-coordinator/coordinator"
	"github.com/anyproto/any-sync-coordinator/coordinatorlog"
	"github.com/anyproto/any-sync-coordinator/db"
	"github.com/anyproto/any-sync-coordinator/deletionlog"
	"github.com/anyproto/any-sync-coordinator/identityrepo"
	"github.com/anyproto/any-sync-coordinator/nodeconfsource"
	"github.com/anyproto/any-sync-coordinator/spacestatus"

	"github.com/anyproto/any-sync/acl"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/consensus/consensusclient"
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

func NewCoordinatorApp(cfg *config.Config) *app.App {
	MustMkdirAll(cfg.NetworkStorePath)

	a := new(app.App).
		Register(cfg).
		Register(db.New()).
		Register(metric.New()).
		Register(account.New()).
		Register(nodeconfstore.New()).
		Register(nodeconf.New()).
		Register(nodeconfsource.New()).
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
		Register(accountlimit.New()).
		Register(identityrepo.New()).
		Register(coordinator.New()).
		Register(yamux.New()).
		Register(quic.New())

	return a
}

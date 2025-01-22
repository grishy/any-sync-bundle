package app

import (
	"os"

	"go.uber.org/zap"

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
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/net/rpc"

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

	"any-sync-bundle/services/metricmock"
)

func NewCoordinatorApp(log logger.CtxLogger) *app.App {
	yamixCfg := yamux.Config{
		ListenAddrs: []string{
			"0.0.0.0:15000",
		},
		WriteTimeoutSec:    10,
		DialTimeoutSec:     10,
		KeepAlivePeriodSec: 0,
	}

	quicCfg := quic.Config{
		ListenAddrs: []string{
			"0.0.0.0:15010",
		},
		WriteTimeoutSec:    0,
		DialTimeoutSec:     0,
		MaxStreams:         0,
		KeepAlivePeriodSec: 0,
	}

	metricCfg := metric.Config{}

	drpcCfg := rpc.Config{
		Stream: rpc.StreamConfig{
			MaxMsgSizeMb: 256,
		},
	}

	// TODO: Remove when merged https://github.com/anyproto/any-sync/pull/374
	netStorePath := "./data/networkStore/coordinator"
	if err := os.MkdirAll(netStorePath, 0o775); err != nil {
		log.Panic("can't create directory for", zap.Error(err))
	}

	cfg := &config.Config{
		Account:                  confAcc,
		Drpc:                     drpcCfg,
		Metric:                   metricCfg,
		Network:                  confNetwork,
		NetworkStorePath:         netStorePath,
		NetworkUpdateIntervalSec: 0,
		Mongo: db.Mongo{
			Connect:  "mongodb://localhost:27017",
			Database: "coordinator",
		},
		SpaceStatus: spacestatus.Config{
			RunSeconds:         5,
			DeletionPeriodDays: 0,
			SpaceLimit:         0,
		},
		Yamux: yamixCfg,
		Quic:  quicCfg,
		AccountLimits: accountlimit.SpaceLimits{
			SpaceMembersRead:  1000,
			SpaceMembersWrite: 1000,
			SharedSpacesLimit: 1000,
		},
	}

	a := new(app.App)

	a.
		Register(cfg).
		Register(db.New()).
		Register(metricmock.New()). // Changed
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

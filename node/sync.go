package app

import (
	"os"

	"github.com/anyproto/any-sync/app/debugstat"
	anyConfig "github.com/anyproto/any-sync/commonspace/config"
	"github.com/anyproto/any-sync/commonspace/credentialprovider"
	"github.com/anyproto/any-sync/consensus/consensusclient"
	"github.com/anyproto/any-sync/coordinator/coordinatorclient"
	"github.com/anyproto/any-sync/coordinator/nodeconfsource"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc"
	"github.com/anyproto/any-sync/net/rpc/debugserver"
	"github.com/anyproto/any-sync/net/rpc/limiter"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/streampool"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/node/nodeclient"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/any-sync/nodeconf/nodeconfstore"
	"github.com/anyproto/any-sync/util/syncqueues"

	"github.com/anyproto/any-sync-node/nodehead"
	"github.com/anyproto/any-sync-node/nodespace/peermanager"
	"github.com/anyproto/any-sync-node/nodespace/spacedeleter"
	"github.com/anyproto/any-sync-node/nodesync"
	"github.com/anyproto/any-sync-node/nodesync/coldsync"
	"github.com/anyproto/any-sync-node/nodesync/hotsync"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/commonspace"

	"github.com/anyproto/any-sync/net/secureservice"

	"github.com/anyproto/any-sync-node/account"
	"github.com/anyproto/any-sync-node/config"
	"github.com/anyproto/any-sync-node/debug/nodedebugrpc"
	"github.com/anyproto/any-sync-node/nodespace"
	"github.com/anyproto/any-sync-node/nodespace/nodecache"
	"github.com/anyproto/any-sync-node/nodestorage"

	"go.uber.org/zap"
)

func NewSyncApp(log logger.CtxLogger) *app.App {
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
	netStorePath := "./data/networkStore/sync"
	if err := os.MkdirAll(netStorePath, 0o775); err != nil {
		log.Panic("can't create directory for sync", zap.Error(err))
	}

	storagePath := "./data/sync-storage"
	if err := os.MkdirAll(netStorePath, 0o775); err != nil {
		log.Panic("can't create directory for sync", zap.Error(err))
	}

	cfg := &config.Config{
		Drpc:    drpcCfg,
		Account: confAcc,
		APIServer: debugserver.Config{
			ListenAddr: "0.0.0.0:18080",
		},
		Network:                  confNetwork,
		NetworkStorePath:         netStorePath,
		NetworkUpdateIntervalSec: 0,
		Space: anyConfig.Config{
			GCTTL:      60,
			SyncPeriod: 600,
		},
		Storage: nodestorage.Config{
			Path: storagePath,
		},
		Metric: metricCfg,
		Log: logger.Config{
			Production: false,
		},
		NodeSync: nodesync.Config{
			SyncOnStart:       false,
			PeriodicSyncHours: 0,
			HotSync: hotsync.Config{
				SimultaneousRequests: 0,
			},
		},
		Yamux: yamixCfg,
		Limiter: limiter.Config{
			DefaultTokens: limiter.Tokens{
				TokensPerSecond: 0,
				MaxTokens:       0,
			},
			ResponseTokens: nil,
		},
		Quic: quicCfg,
	}

	a := new(app.App)

	a.
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

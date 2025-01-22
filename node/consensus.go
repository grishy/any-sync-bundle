package app

import (
	"os"

	"github.com/anyproto/any-sync-consensusnode/account"
	"github.com/anyproto/any-sync-consensusnode/config"
	"github.com/anyproto/any-sync-consensusnode/consensusrpc"
	"github.com/anyproto/any-sync-consensusnode/db"
	"github.com/anyproto/any-sync-consensusnode/stream"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/coordinator/coordinatorclient"
	"github.com/anyproto/any-sync/coordinator/nodeconfsource"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/any-sync/nodeconf/nodeconfstore"
	"go.uber.org/zap"

	"any-sync-bundle/services/metricmock"
)

func NewConsensusApp(log logger.CtxLogger) *app.App {
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
	netStorePath := "./data/networkStore/consensus"
	if err := os.MkdirAll(netStorePath, 0o775); err != nil {
		log.Panic("can't create directory for consensus", zap.Error(err))
	}

	cfg := &config.Config{
		Drpc:                     drpcCfg,
		Account:                  confAcc,
		Network:                  confNetwork,
		NetworkStorePath:         netStorePath,
		NetworkUpdateIntervalSec: 0,
		Mongo: config.Mongo{
			Connect:       "mongodb://lab_anytype_mongo:27017/?w=majority",
			Database:      "consensus",
			LogCollection: "log",
		},
		Metric: metricCfg,
		Log: logger.Config{
			Production: false,
		},
		Yamux: yamixCfg,
		Quic:  quicCfg,
	}

	a := new(app.App)

	a.
		Register(cfg).
		Register(metricmock.New()). // Changed
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

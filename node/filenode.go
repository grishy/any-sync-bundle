package app

import (
	"os"

	"go.uber.org/zap"

	"github.com/anyproto/any-sync-filenode/account"
	"github.com/anyproto/any-sync-filenode/config"
	"github.com/anyproto/any-sync-filenode/deletelog"
	"github.com/anyproto/any-sync-filenode/filenode"
	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/redisprovider"

	"github.com/anyproto/any-sync/acl"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/consensus/consensusclient"
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

	"any-sync-bundle/services/filenodesqlite"
	"any-sync-bundle/services/metricmock"
)

func NewFileNodeApp(log logger.CtxLogger) *app.App {
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

	metricCfg := metric.Config{
		Addr: "0.0.0.0:18080",
	}

	drpcCfg := rpc.Config{
		Stream: rpc.StreamConfig{
			MaxMsgSizeMb: 256,
		},
	}

	// TODO: Create ticket about mkdirall
	// https://github.com/anyproto/any-sync/pull/297
	netStorePath := "./data/networkStore/coordinator"
	if err := os.MkdirAll(netStorePath, 0o775); err != nil {
		log.Panic("can't create directory for coordinator", zap.Error(err))
	}

	cfgFileNode := &config.Config{
		Account: confAcc,
		Drpc:    drpcCfg,
		Yamux:   yamixCfg,
		Quic:    quicCfg,
		Metric:  metricCfg,
		Redis: redisprovider.Config{
			IsCluster: false,
			Url:       "",
		},
		Network:          confNetwork,
		NetworkStorePath: netStorePath,
		DefaultLimit:     1099511627776, // 1 TB
	}

	a := new(app.App)
	a.Register(cfgFileNode).
		Register(metricmock.New()). // Changed
		Register(account.New()).
		Register(nodeconfsource.New()).
		Register(nodeconfstore.New()).
		Register(nodeconf.New()).
		Register(peerservice.New()).
		Register(secureservice.New()).
		Register(pool.New()).
		Register(coordinatorclient.New()).
		Register(consensusclient.New()).
		Register(acl.New()).
		Register(filenodesqlite.NewSqlStorage()). // Changed: Replacement for S3 store
		Register(redisprovider.New()).
		Register(index.New()).
		Register(server.New()).
		Register(filenode.New()).
		Register(deletelog.New()).
		Register(yamux.New()).
		Register(quic.New())

	return a
}

package main

import (
	"context"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"any-sync-bundle/bundlefilenode"

	"github.com/anyproto/any-sync-filenode/account"
	configFilenode "github.com/anyproto/any-sync-filenode/config"
	"github.com/anyproto/any-sync-filenode/deletelog"
	"github.com/anyproto/any-sync-filenode/filenode"
	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/redisprovider"

	"github.com/anyproto/any-sync/accountservice"
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
	"go.uber.org/zap"
)

var log = logger.NewNamed("main")

func main() {
	// TODO: Replace it on new build-in version of it in Go
	app.AppName = "any-sync-bundle"

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	// Common configs
	yamixCfg := yamux.Config{
		ListenAddrs: []string{
			"0.0.0.0:5000",
		},
		WriteTimeoutSec:    10,
		DialTimeoutSec:     10,
		KeepAlivePeriodSec: 0,
	}

	quicCfg := quic.Config{
		ListenAddrs: []string{
			"0.0.0.0:5010",
		},
		WriteTimeoutSec:    0,
		DialTimeoutSec:     0,
		MaxStreams:         0,
		KeepAlivePeriodSec: 0,
	}

	metricCfg := metric.Config{
		Addr: "0.0.0.0:8080",
	}

	drpcCfg := rpc.Config{
		Stream: rpc.StreamConfig{
			MaxMsgSizeMb: 256,
		},
	}

	// TODO: Create ticket about mkdirall
	// https://github.com/anyproto/any-sync/pull/297
	netStorePath := "./data/networkStore/filenode"
	if err := os.MkdirAll(netStorePath, 0o775); err != nil {
		log.Panic("can't create directory for filenode")
	}

	cfgFileNode := &configFilenode.Config{
		Account: accountservice.Config{
			PeerId:     "12D3KooWGDY4mGz1xR3yjeLLQv2Umcjz48gAGzA48eFacpiK1mF4",
			PeerKey:    "9uneWpf+EkW0UJbqPbmh331bmrWcwiDFBqgN2xRRSDhfFautpQs0kAe0Y+eCxDnwW3LMw6qNAPI73GGTQf0lXw==",
			SigningKey: "42NuMqiLioOREOpqZZqJoxtLiXbIwonncJy8kyc/22jShI0uFDdr27ULth0uioPcm9h2o381sQHYbU3SDTG5GQ==",
		},
		Drpc:   drpcCfg,
		Yamux:  yamixCfg,
		Quic:   quicCfg,
		Metric: metricCfg,
		Redis: redisprovider.Config{
			IsCluster: false,
			Url:       "",
		},
		Network: nodeconf.Configuration{
			Id:        "",
			NetworkId: "",
			Nodes:     nil,
		},
		NetworkStorePath: netStorePath,
		DefaultLimit:     1099511627776, // 1 TB
	}

	fnApp := newFileNode(cfgFileNode)

	// start app
	if err := fnApp.Start(ctx); err != nil {
		log.Fatal("can't start app", zap.Error(err))
	}
	log.Info("app started")

	// wait exit signal
	<-ctx.Done()

	// close app
	ctxClose, cancelClose := context.WithTimeout(ctx, time.Minute)
	defer cancelClose()
	if err := fnApp.Close(ctxClose); err != nil {
		log.Fatal("close error", zap.Error(err))
	} else {
		log.Info("goodbye!")
	}

	time.Sleep(time.Second / 3)
}

func newFileNode(cfg *configFilenode.Config) *app.App {
	a := new(app.App)
	a.Register(cfg).
		Register(account.New()).
		Register(metric.New()).
		Register(nodeconfsource.New()).
		Register(nodeconfstore.New()).
		Register(nodeconf.New()).
		Register(peerservice.New()).
		Register(secureservice.New()).
		Register(pool.New()).
		Register(coordinatorclient.New()).
		Register(consensusclient.New()).
		Register(acl.New()).
		Register(bundlefilenode.NewSqlStorage()). // Replacement for S3 store
		Register(redisprovider.New()).            // TODO: Replace?
		Register(index.New()).
		Register(server.New()).
		Register(filenode.New()).
		Register(deletelog.New()).
		Register(yamux.New()).
		Register(quic.New())

	return a
}

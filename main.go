package main

import (
	"context"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"any-sync-bundle/bundlefilenode"

	"go.uber.org/zap"

	syncAccountservice "github.com/anyproto/any-sync/accountservice"
	syncAcl "github.com/anyproto/any-sync/acl"
	syncApp "github.com/anyproto/any-sync/app"
	syncLogger "github.com/anyproto/any-sync/app/logger"
	syncConsensusclient "github.com/anyproto/any-sync/consensus/consensusclient"
	syncCoordinatorclient "github.com/anyproto/any-sync/coordinator/coordinatorclient"
	syncNodeconfsource "github.com/anyproto/any-sync/coordinator/nodeconfsource"
	syncMetric "github.com/anyproto/any-sync/metric"
	syncPeerservice "github.com/anyproto/any-sync/net/peerservice"
	syncPool "github.com/anyproto/any-sync/net/pool"
	syncRpc "github.com/anyproto/any-sync/net/rpc"
	syncServer "github.com/anyproto/any-sync/net/rpc/server"
	syncSecureservice "github.com/anyproto/any-sync/net/secureservice"
	syncQuic "github.com/anyproto/any-sync/net/transport/quic"
	syncYamux "github.com/anyproto/any-sync/net/transport/yamux"
	syncNodeconf "github.com/anyproto/any-sync/nodeconf"
	syncNodeconfstore "github.com/anyproto/any-sync/nodeconf/nodeconfstore"

	fileNodeAccount "github.com/anyproto/any-sync-filenode/account"
	fileNodeConfig "github.com/anyproto/any-sync-filenode/config"
	fileNodeDeletelog "github.com/anyproto/any-sync-filenode/deletelog"
	fileNodeFilenode "github.com/anyproto/any-sync-filenode/filenode"
	fileNodeIndex "github.com/anyproto/any-sync-filenode/index"
	fileNodeRedis "github.com/anyproto/any-sync-filenode/redisprovider"
)

var log = syncLogger.NewNamed("main")

func main() {
	// TODO: Replace it on new build-in version of it in Go
	syncApp.AppName = "any-sync-bundle"

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	// Common configs
	apps := []*syncApp.App{
		newFileNodeApp(),
	}

	// start apps
	for _, app := range apps {
		if err := app.Start(ctx); err != nil {
			log.Fatal("can't start app", zap.Error(err))
		}
	}
	log.Info("apps started")

	// wait exit signal
	<-ctx.Done()

	// close apps
	for _, app := range apps {
		ctxClose, cancelClose := context.WithTimeout(context.Background(), time.Minute)
		if err := app.Close(ctxClose); err != nil {
			log.Fatal("close error", zap.Error(err))
		}
		cancelClose()
	}
	log.Info("goodbye!")

	time.Sleep(time.Second / 3)
}

func newFileNodeApp() *syncApp.App {
	yamixCfg := syncYamux.Config{
		ListenAddrs: []string{
			"0.0.0.0:15000",
		},
		WriteTimeoutSec:    10,
		DialTimeoutSec:     10,
		KeepAlivePeriodSec: 0,
	}

	quicCfg := syncQuic.Config{
		ListenAddrs: []string{
			"0.0.0.0:15010",
		},
		WriteTimeoutSec:    0,
		DialTimeoutSec:     0,
		MaxStreams:         0,
		KeepAlivePeriodSec: 0,
	}

	metricCfg := syncMetric.Config{
		Addr: "0.0.0.0:18080",
	}

	drpcCfg := syncRpc.Config{
		Stream: syncRpc.StreamConfig{
			MaxMsgSizeMb: 256,
		},
	}

	// TODO: Create ticket about mkdirall
	// https://github.com/anyproto/any-sync/pull/297
	netStorePath := "./data/networkStore/filenode"
	if err := os.MkdirAll(netStorePath, 0o775); err != nil {
		log.Panic("can't create directory for filenode")
	}

	cfgFileNode := &fileNodeConfig.Config{
		Account: syncAccountservice.Config{
			PeerId:     "12D3KooWGDY4mGz1xR3yjeLLQv2Umcjz48gAGzA48eFacpiK1mF4",
			PeerKey:    "9uneWpf+EkW0UJbqPbmh331bmrWcwiDFBqgN2xRRSDhfFautpQs0kAe0Y+eCxDnwW3LMw6qNAPI73GGTQf0lXw==",
			SigningKey: "42NuMqiLioOREOpqZZqJoxtLiXbIwonncJy8kyc/22jShI0uFDdr27ULth0uioPcm9h2o381sQHYbU3SDTG5GQ==",
		},
		Drpc:   drpcCfg,
		Yamux:  yamixCfg,
		Quic:   quicCfg,
		Metric: metricCfg,
		Redis: fileNodeRedis.Config{
			IsCluster: false,
			Url:       "",
		},
		Network: syncNodeconf.Configuration{
			Id:        "",
			NetworkId: "",
			Nodes:     nil,
		},
		NetworkStorePath: netStorePath,
		DefaultLimit:     1099511627776, // 1 TB
	}

	a := new(syncApp.App)
	a.Register(cfgFileNode).
		Register(fileNodeAccount.New()).
		Register(syncMetric.New()).
		Register(syncNodeconfsource.New()).
		Register(syncNodeconfstore.New()).
		Register(syncNodeconf.New()).
		Register(syncPeerservice.New()).
		Register(syncSecureservice.New()).
		Register(syncPool.New()).
		Register(syncCoordinatorclient.New()).
		Register(syncConsensusclient.New()).
		Register(syncAcl.New()).
		Register(bundlefilenode.NewSqlStorage()). // Replacement for S3 store
		Register(fileNodeRedis.New()).            // TODO: Replace?
		Register(fileNodeIndex.New()).
		Register(syncServer.New()).
		Register(fileNodeFilenode.New()).
		Register(fileNodeDeletelog.New()).
		Register(syncYamux.New()).
		Register(syncQuic.New())

	return a
}

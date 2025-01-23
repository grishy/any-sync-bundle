package main

import (
	"context"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"go.uber.org/zap"

	bundleCfg "any-sync-bundle/config"
	bundleNode "any-sync-bundle/node"
)

var log = logger.NewNamed("main")

const (
	configBundlePath = "./data/cfg/priv_bundle.yaml"
	configClientPath = "./data/cfg/pub_client.yaml"
)

func main() {
	// TODO: Replace it on new build-in version of it in Go
	app.AppName = "any-sync-bundle"

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	// TODO: Create commands to only generate conf and allow to provide external addrs
	// TODO: Cread configs also from args or env
	cfgBundle := bundleCfg.BundleCfg(configBundlePath)
	cfgNodes := cfgBundle.NodeConfigs()

	// Common configs
	apps := []*app.App{
		bundleNode.NewCoordinatorApp(logger.NewNamed("coordinator"), cfgNodes.Coordinator),
		// bundleNode.NewConsensusApp(logger.NewNamed("consensus"), cfgNodes.Consensus),
		// bundleNode.NewFileNodeApp(logger.NewNamed("filenode"), cfgNodes.Filenode),
		// bundleNode.NewSyncApp(logger.NewNamed("sync"), cfgNodes.Sync),
	}

	// start apps
	for _, a := range apps {
		if err := a.Start(ctx); err != nil {
			log.Fatal("can't start app", zap.Error(err))
		}
	}
	log.Info("apps started")

	// wait exit signal
	<-ctx.Done()

	// TODO: Stop in reverse order
	// close apps
	for _, a := range apps {
		ctxClose, cancelClose := context.WithTimeout(context.Background(), time.Minute)
		if err := a.Close(ctxClose); err != nil {
			log.Fatal("close error", zap.Error(err))
		}
		cancelClose()
	}
	log.Info("goodbye!")

	time.Sleep(time.Second / 3)
}

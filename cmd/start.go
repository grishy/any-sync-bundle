package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	bundleCfg "github.com/grishy/any-sync-bundle/config"
	bundleNode "github.com/grishy/any-sync-bundle/node"
)

type node struct {
	name string
	app  *app.App
}

func cmdStart(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start bundle services",
		Flags: []cli.Flag{},
		Action: func(cCtx *cli.Context) error {
			log := logger.NewNamed("main")

			initExternalAddrs := cCtx.StringSlice(fGlobalInitExternalAddrs)
			initMongoURI := cCtx.String(fGlobalInitMongoURI)
			initRedisURI := cCtx.String(fGlobalInitRedisURI)
			cfgPath := cCtx.String(fGlobalBundleConfigPath)
			storagePath := cCtx.String(fGlobalStoragePath)

			// TODO: AppName global, AppName not working properly on app instance

			printWelcome()

			// TODO: Create commands to only generate conf and allow to provide external addrs
			// TODO: Cread configs also from args or env?

			var cfgBundle *bundleCfg.Config
			log.Info("loading config")
			if _, err := os.Stat(cfgPath); err == nil {
				log.Info("loaded existing config")
				cfgBundle = bundleCfg.Read(cfgPath)
			}

			log.Info("file not found, created new config")
			// TODO Create if not exist
			cfgBundle = bundleCfg.CreateWrite(&bundleCfg.CreateOptions{
				CfgPath:       cfgPath,
				StorePath:     storagePath,
				MongoURI:      initMongoURI,
				RedisURI:      initRedisURI,
				ExternalAddrs: initExternalAddrs,
			})

			cfgNodes := cfgBundle.NodeConfigs()

			// TODO
			// mongoInit(ctx, cfgBundle.Nodes.Coordinator.MongoConnect)

			// Dump client config
			// TODO if no exist create
			// cfgBundle.ClientConfig(configClientPath)

			fileStore := filepath.Join(cfgBundle.StoragePath, "storage-file")

			// Common configs
			apps := []node{
				{
					name: "coordinator",
					app:  bundleNode.NewCoordinatorApp(logger.NewNamed("coordinator"), cfgNodes.Coordinator),
				},
				{
					name: "consensus",
					app:  bundleNode.NewConsensusApp(logger.NewNamed("consensus"), cfgNodes.Consensus),
				},
				{
					name: "filenode",
					app:  bundleNode.NewFileNodeApp(logger.NewNamed("filenode"), cfgNodes.Filenode, fileStore),
				},
				{
					name: "sync",
					app:  bundleNode.NewSyncApp(logger.NewNamed("sync"), cfgNodes.Sync),
				},
			}

			// Start all services
			log.Info("Initiating service startup", zap.Int("count", len(apps)))

			for _, a := range apps {
				log.Info("▶ Starting service", zap.String("name", a.name))
				if err := a.app.Start(ctx); err != nil {
					log.Panic("✗ Service startup failed",
						zap.String("name", a.name),
						zap.Error(err))
				}

				log.Info("✓ Service started successfully", zap.String("name", a.name))
			}

			log.Info("↑ Service startup complete")

			// wait exit signal
			<-ctx.Done()

			// Stop apps in reverse order
			log.Info("⚡ Initiating service shutdown", zap.Int("count", len(apps)))
			for _, a := range slices.Backward(apps) {
				log.Info("▶ Stopping service", zap.String("name", a.name))
				ctxClose, cancelClose := context.WithTimeout(context.Background(), 30*time.Second)
				if err := a.app.Close(ctxClose); err != nil {
					log.Error("✗ Service shutdown failed", zap.String("name", a.name), zap.Error(err))
				} else {
					log.Info("✓ Service stopped successfully", zap.String("name", a.name))
				}

				cancelClose()
			}

			log.Info("→ Goodbye!")
			return nil
		},
	}
}

func printWelcome() {
	fmt.Printf(`
┌───────────────────────────────────────────────────────────────────┐

                 Welcome to the AnySync Bundle!
           https://github.com/grishy/any-sync-bundle                   

    Version: %s
    Built:   %s
    Commit:  %s

`, version, commit, date)

	fmt.Println(" Based on these components:")
	info, ok := debug.ReadBuildInfo()
	if !ok {
		log.Panic("failed to read build info")
		return
	}

	for _, mod := range info.Deps {
		if strings.HasPrefix(mod.Path, "github.com/anyproto/any-sync") {
			fmt.Printf(" ‣ %s (%s)\n", mod.Path, mod.Version)
		}
	}
	fmt.Print(`
└───────────────────────────────────────────────────────────────────┘
`)
}

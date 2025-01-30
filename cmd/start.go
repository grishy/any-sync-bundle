package cmd

import (
	"context"
	"fmt"
	"os"
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

const serviceShutdownTimeout = 30 * time.Second

func cmdStart(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:   "start",
		Usage:  "Start bundle services",
		Action: startAction(ctx),
	}
}

func startAction(ctx context.Context) cli.ActionFunc {
	return func(cCtx *cli.Context) error {
		printWelcome()

		// Load or create bundle configuration
		cfgBundle := loadOrCreateConfig(cCtx, log)

		// TODO: Write client config if not exists

		// Initialize service instances
		cfgNodes := cfgBundle.NodeConfigs()

		apps := []node{
			{name: "coordinator", app: bundleNode.NewCoordinatorApp(cfgNodes.Coordinator)},
			{name: "consensus", app: bundleNode.NewConsensusApp(cfgNodes.Consensus)},
			{name: "filenode", app: bundleNode.NewFileNodeApp(cfgNodes.Filenode, cfgNodes.FilenodeStorePath)},
			{name: "sync", app: bundleNode.NewSyncApp(cfgNodes.Sync)},
		}

		if err := startServices(ctx, apps); err != nil {
			return err
		}

		// Wait for shutdown signal
		<-ctx.Done()

		shutdownServices(apps)

		log.Info("→ Goodbye!")
		return nil
	}
}

func loadOrCreateConfig(cCtx *cli.Context, log logger.CtxLogger) *bundleCfg.Config {
	cfgPath := cCtx.String(fGlobalBundleConfigPath)
	log.Info("loading config")

	if _, err := os.Stat(cfgPath); err == nil {
		log.Info("loaded existing config")
		return bundleCfg.Load(cfgPath)
	}

	log.Info("creating new config")
	return bundleCfg.CreateWrite(&bundleCfg.CreateOptions{
		CfgPath:       cfgPath,
		StorePath:     cCtx.String(fGlobalStoragePath),
		MongoURI:      cCtx.String(fGlobalInitMongoURI),
		RedisURI:      cCtx.String(fGlobalInitRedisURI),
		ExternalAddrs: cCtx.StringSlice(fGlobalInitExternalAddrs),
	})
}

func startServices(ctx context.Context, apps []node) error {
	log.Info("initiating service startup", zap.Int("count", len(apps)))

	for _, a := range apps {
		log.Info("▶ starting service", zap.String("name", a.name))
		if err := a.app.Start(ctx); err != nil {
			return fmt.Errorf("service startup failed: %w", err)
		}

		log.Info("✓ service started successfully", zap.String("name", a.name))
	}

	log.Info("↑ service startup complete")
	return nil
}

func shutdownServices(apps []node) {
	log.Info("⚡ initiating service shutdown", zap.Int("count", len(apps)))

	for _, a := range slices.Backward(apps) {
		log.Info("▶ stopping service", zap.String("name", a.name))

		ctx, cancel := context.WithTimeout(context.Background(), serviceShutdownTimeout)

		if err := a.app.Close(ctx); err != nil {
			log.Error("✗ service shutdown failed", zap.String("name", a.name), zap.Error(err))
		} else {
			log.Info("✓ service stopped successfully", zap.String("name", a.name))
		}

		cancel()
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

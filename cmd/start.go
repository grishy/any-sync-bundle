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

	bundleConfig "github.com/grishy/any-sync-bundle/config"
	bundleNode "github.com/grishy/any-sync-bundle/node"
)

const (
	fIsInitMongoRs = "init-mongo-rs"
)

type node struct {
	name string
	app  *app.App
}

const serviceShutdownTimeout = 30 * time.Second

func cmdStart(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start bundle services",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    fIsInitMongoRs,
				Usage:   "Initialize MongoDB replica set",
				EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_MONGO_RS"},
			},
		},
		Action: startAction(ctx),
	}
}

func startAction(ctx context.Context) cli.ActionFunc {
	return func(cCtx *cli.Context) error {
		clientCfgPath := cCtx.String(fGlobalClientConfigPath)
		isInitMongoRs := cCtx.Bool(fIsInitMongoRs)

		printWelcomeMsg()

		// Load or create bundle configuration
		bundleCfg := loadOrCreateConfig(cCtx, log)

		// Create client configuration if not exists
		if _, err := os.Stat(clientCfgPath); err != nil {
			log.Warn("client configuration not found, creating new one")
			yamlData, err := bundleCfg.YamlClientConfig()
			if err != nil {
				return fmt.Errorf("failed to generate client config: %w", err)
			}

			if err := os.WriteFile(clientCfgPath, yamlData, configFileMode); err != nil {
				return fmt.Errorf("failed to write client config: %w", err)
			}

			log.Info("client configuration written", zap.String("path", clientCfgPath))
		}

		if isInitMongoRs {
			log.Info("initializing MongoDB replica set")

			if err := initReplicaSetAction(ctx, defaultMongoReplica, defaultMongoURI); err != nil {
				return fmt.Errorf("failed to initialize MongoDB replica set: %w", err)
			}
		}

		// Initialize nodes' instances
		cfgNodes := bundleCfg.NodeConfigs()

		apps := []node{
			{name: "coordinator", app: bundleNode.NewCoordinatorApp(cfgNodes.Coordinator)},
			{name: "consensus", app: bundleNode.NewConsensusApp(cfgNodes.Consensus)},
			{name: "filenode", app: bundleNode.NewFileNodeApp(cfgNodes.Filenode, cfgNodes.FilenodeStorePath)},
			{name: "sync", app: bundleNode.NewSyncApp(cfgNodes.Sync)},
		}

		if err := startServices(ctx, apps); err != nil {
			return err
		}

		printStartupMsg()

		// Wait for shutdown signal
		<-ctx.Done()

		shutdownServices(apps)
		printShutdownMsg()

		log.Info("→ Goodbye!")
		return nil
	}
}

func loadOrCreateConfig(cCtx *cli.Context, log logger.CtxLogger) *bundleConfig.Config {
	cfgPath := cCtx.String(fGlobalBundleConfigPath)
	log.Info("loading config")

	if _, err := os.Stat(cfgPath); err == nil {
		log.Info("loaded existing config")
		return bundleConfig.Load(cfgPath)
	}

	log.Info("creating new config")
	return bundleConfig.CreateWrite(&bundleConfig.CreateOptions{
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

func printWelcomeMsg() {
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

func printStartupMsg() {
	fmt.Printf(`
┌───────────────────────────────────────────────────────────────────┐

                      AnySync Bundle is ready!
                      All services are running.
                   Press Ctrl+C to stop services.

└───────────────────────────────────────────────────────────────────┘
`)
}

func printShutdownMsg() {
	fmt.Printf(`
┌───────────────────────────────────────────────────────────────────┐

                 AnySync Bundle shutdown complete!
                     All services are stopped.

└───────────────────────────────────────────────────────────────────┘
`)
}

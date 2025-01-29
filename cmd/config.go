package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	bundleCfg "github.com/grishy/any-sync-bundle/config"
)

const (
	fForce = "force"
)

func cmdConfig(_ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Generates a new bundle configuration file with secure default settings.",
		Subcommands: []*cli.Command{
			cmdConfigBundle(),
			cmdConfigClient(),
		},
	}
}

func cmdConfigBundle() *cli.Command {
	return &cli.Command{
		Name:  "bundle",
		Usage: "Generates a new bundle configuration file with secure default settings.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    fForce,
				Aliases: []string{"f"},
				Value:   false,
				Usage:   "Force overwrite if configuration file already exists",
				EnvVars: []string{"ANY_SYNC_BUNDLE_FORCE"},
			},
		},
		Action: func(cCtx *cli.Context) error {
			initExternalAddrs := cCtx.StringSlice(fGlobalInitExternalAddrs)
			initMongoURI := cCtx.String(fGlobalInitMongoURI)
			initRedisURI := cCtx.String(fGlobalInitRedisURI)
			cfgPath := cCtx.String(fGlobalBundleConfigPath)
			storagePath := cCtx.String(fGlobalStoragePath)
			force := cCtx.Bool(fForce)

			log.Info("write new bundle configuration",
				zap.String("path", cfgPath),
				zap.Bool("force", force),
				zap.String("initial-mongo-uri", initMongoURI),
				zap.String("initial-redis-uri", initRedisURI),
				zap.Strings("initial-external-addrs", initExternalAddrs),
			)

			// Check if file exists
			if _, err := os.Stat(cfgPath); err == nil {
				if !force {
					return fmt.Errorf("configuration file already exists at '%s', use --%s to overwrite", cfgPath, fForce)
				}
				log.Warn("overwriting existing configuration file")
			}

			log.Info("generating new bundle configuration")

			cfg := bundleCfg.CreateWrite(&bundleCfg.CreateOptions{
				CfgPath:       cfgPath,
				StorePath:     storagePath,
				MongoURI:      initMongoURI,
				RedisURI:      initRedisURI,
				ExternalAddrs: initExternalAddrs,
			})
			_ = cfg

			log.Info("bundle configuration written")

			return nil
		},
	}
}

func cmdConfigClient() *cli.Command {
	return &cli.Command{
		Name:  "client",
		Usage: "Generates a new bundle configuration file with secure default settings.",
		Action: func(cCtx *cli.Context) error {
			cfgPath := cCtx.String(fGlobalBundleConfigPath)
			clientCfgPath := cCtx.String(fGlobalClientConfigPath)

			log.Info("generating new client configuration based on bundle configuration",
				zap.String("path", cfgPath),
				zap.String("clientPath", clientCfgPath),
			)

			cfg := bundleCfg.Read(cfgPath)
			log.Info("bundle configuration read")

			clientCfgData, err := cfg.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to generate client configuration: %w", err)
			}

			if err := os.WriteFile(clientCfgPath, clientCfgData, 0o644); err != nil {
				return fmt.Errorf("failed to write client configuration: %w", err)
			}

			log.Info("client configuration written")

			return nil
		},
	}
}

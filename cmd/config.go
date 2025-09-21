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
	fForce         = "force"
	configFileMode = 0o644
)

func cmdConfig(_ context.Context) *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage bundle and client configurations",
		Subcommands: []*cli.Command{
			cmdConfigBundle(),
			cmdConfigClient(),
		},
	}
}

func cmdConfigBundle() *cli.Command {
	return &cli.Command{
		Name:  "bundle",
		Usage: "Generate a new bundle configuration file",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    fForce,
				Aliases: []string{"f"},
				Usage:   "Force overwrite if configuration file already exists",
				EnvVars: []string{"ANY_SYNC_BUNDLE_FORCE"},
			},
		},
		Action: func(cCtx *cli.Context) error {
			cfgPath := cCtx.String(fGlobalBundleConfigPath)
			storagePath := cCtx.String(fGlobalStoragePath)
			mongoURI := cCtx.String(fGlobalInitMongoURI)
			redisURI := cCtx.String(fGlobalInitRedisURI)
			externalAddrs := cCtx.StringSlice(fGlobalInitExternalAddrs)
			forceOverwrite := cCtx.Bool(fForce)

			// Prevent accidental config overwrite.
			if !forceOverwrite {
				if _, err := os.Stat(cfgPath); err == nil {
					return fmt.Errorf(
						"configuration file already exists at '%s', use --%s to overwrite",
						cfgPath,
						fForce,
					)
				}
			}

			log.Info("generating new bundle configuration",
				zap.String("path", cfgPath),
				zap.String("mongo_uri", mongoURI),
				zap.String("redis_uri", redisURI),
				zap.Strings("external_addrs", externalAddrs),
			)

			// Create and write bundle configuration.
			cfg := bundleCfg.CreateWrite(&bundleCfg.CreateOptions{
				CfgPath:       cfgPath,
				StorePath:     storagePath,
				MongoURI:      mongoURI,
				RedisURI:      redisURI,
				ExternalAddrs: externalAddrs,
			})
			_ = cfg

			log.Info("bundle configuration written successfully")
			return nil
		},
	}
}

func cmdConfigClient() *cli.Command {
	return &cli.Command{
		Name:  "client",
		Usage: "Generate a client configuration from bundle config",
		Action: func(cCtx *cli.Context) error {
			bundleCfgPath := cCtx.String(fGlobalBundleConfigPath)
			clientCfgPath := cCtx.String(fGlobalClientConfigPath)

			log.Info("generating client configuration",
				zap.String("bundle_config", bundleCfgPath),
				zap.String("client_config", clientCfgPath),
			)

			// Generate and write client configuration.
			bundleConfig := bundleCfg.Load(bundleCfgPath)
			clientCfgData, err := bundleConfig.YamlClientConfig()
			if err != nil {
				return fmt.Errorf("failed to generate client configuration: %w", err)
			}

			if errWrite := os.WriteFile(clientCfgPath, clientCfgData, configFileMode); errWrite != nil {
				return fmt.Errorf("failed to write client configuration: %w", errWrite)
			}

			log.Info("client configuration written successfully")
			return nil
		},
	}
}

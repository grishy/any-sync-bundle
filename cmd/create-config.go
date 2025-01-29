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
	flagForce = "force"
)

func createConfig(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "create-config",
		Usage: "Generates a new bundle configuration file with secure default settings.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    flagForce,
				Value:   false,
				Usage:   "Force overwrite if configuration file already exists",
				EnvVars: []string{"ANY_SYNC_BUNDLE_FORCE"},
			},
		},
		Action: func(cCtx *cli.Context) error {
			cfgPath := cCtx.String(flagConfig)

			log := log.With(zap.String("path", cfgPath))

			// Check if file exists
			if _, err := os.Stat(cfgPath); err == nil {
				if !cCtx.Bool(flagForce) {
					return fmt.Errorf("configuration file already exists at %s. Use --%s to overwrite", cfgPath, flagForce)
				}
				log.Warn("overwriting existing configuration file", zap.String("path", cfgPath))
			}

			log.Info("generating new bundle configuration")

			cfg := bundleCfg.CreateWrite(cfgPath)

			log.Info("bundle configuration generated successfully",
				zap.String("path", cfgPath),
				zap.String("config_id", cfg.ConfigID),
				zap.String("network_id", cfg.NetworkID))

			return nil
		},
	}
}

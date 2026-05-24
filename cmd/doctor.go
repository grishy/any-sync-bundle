package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/grishy/any-sync-bundle/doctor"
)

func cmdDoctor(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:        "doctor",
		Usage:       "Run experimental diagnostics against the already running bundle process",
		Description: "EXPERIMENTAL: command output and JSON report schema may change between releases.",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:    flagStartBundleConfigPath,
				Aliases: []string{"c"},
				Value:   "./data/bundle-config.yml",
				EnvVars: []string{"ANY_SYNC_BUNDLE_CONFIG"},
				Usage:   "Path to the bundle configuration YAML file, used to locate the doctor socket",
			},
		},
		Action: func(cCtx *cli.Context) error {
			bundleConfigPath, err := filepath.Abs(cCtx.String(flagStartBundleConfigPath))
			if err != nil {
				return fmt.Errorf("resolve bundle config path: %w", err)
			}
			return doctor.RunClient(ctx, doctor.SocketPath(bundleConfigPath), cCtx.App.Writer)
		},
	}
}

package cmd

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/anyproto/any-sync/app/logger"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

const (
	appName = "any-sync-bundle"

	// Flag names
	flagDebug   = "debug"
	flagConfig  = "config"
	flagDataDir = "data-dir"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func Root(ctx context.Context) *cli.App {
	cli.VersionPrinter = versionPrinter

	cliApp := &cli.App{
		Name:    appName,
		Usage:   "A TODO",
		Version: version,
		Description: `
		TODO
		`,
		Authors: []*cli.Author{{
			Name:  "Sergei G.",
			Email: "mail@grishy.dev",
		}},
		// Global flags, before any subcommand
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    flagDebug,
				Value:   false,
				Usage:   "Enable debug mode with detailed logging",
				EnvVars: []string{"ANY_SYNC_BUNDLE_DEBUG"},
			},
			&cli.PathFlag{
				Name:    flagConfig,
				Aliases: []string{"c"},
				Value:   "./data/cfg/priv_bundle.yml",
				EnvVars: []string{"ANY_SYNC_BUNDLE_CONFIG"},
				Usage:   "Path to the configuration YAML file (must be readable)",
			},
			&cli.PathFlag{
				Name:    flagDataDir,
				Aliases: []string{"d"},
				Value:   "./data/",
				EnvVars: []string{"ANY_SYNC_BUNDLE_DATA"},
				Usage:   "Path to the data directory (must be writable)",
			},
		},
		Before: setupLogger,
		Commands: []*cli.Command{
			GenerateCfg(ctx),
			MongoInitReplica(ctx),
			StartBundle(ctx),
			// TODO: Remove this debug command
			{
				Name: "log",
				Action: func(cCtx *cli.Context) error {
					log := logger.NewNamed("debug-test")

					log.Debug("This is a debug message")
					log.Info("This is an info message")
					log.Warn("This is a warning message")
					log.Error("This is an error message")

					log.Debug("Debug message with fields",
						zap.String("field1", "value1"),
						zap.Int("field2", 42))

					log.Info("Info message with fields",
						zap.Bool("enabled", true),
						zap.Duration("elapsed", time.Second*5))

					return nil
				},
			},
		},
	}

	return cliApp
}

// setupLogger configures the global logger with appropriate settings
func setupLogger(c *cli.Context) error {
	anyLogCfg := logger.Config{
		Production:   false,
		DefaultLevel: "",
		Format:       logger.PlaintextOutput,
	}

	if c.Bool(flagDebug) {
		anyLogCfg.DefaultLevel = "debug"
		anyLogCfg.Format = logger.ColorizedOutput
	}

	anyLogCfg.ApplyGlobal()

	return nil
}

func versionPrinter(c *cli.Context) {
	fmt.Println(c.App.Name)
	fmt.Printf("Version:   %s\n", version)
	fmt.Printf("Commit:    %s\n", commit)
	fmt.Printf("Date:      %s\n", date)
	fmt.Printf("GoVersion: %s\n", runtime.Version())
	fmt.Printf("Platform:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

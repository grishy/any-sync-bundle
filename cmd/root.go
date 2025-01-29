package cmd

import (
	"context"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/urfave/cli/v2"
)

// Version information, set during build
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const (
	appName = "any-sync-bundle"

	// CLI global flags
	fGlobalIsDbg             = "debug"
	fGlobalStoragePath       = "storage"
	fGlobalBundleConfigPath  = "bundle-config"
	fGlobalClientConfigPath  = "client-config"
	fGlobalInitExternalAddrs = "initial-external-addrs"
	fGlobalInitMongoURI      = "initial-mongo-uri"
	fGlobalInitRedisURI      = "initial-redis-uri"
)

var log = logger.NewNamed("cli")

func Root(ctx context.Context) *cli.App {
	cli.VersionPrinter = versionPrinter

	// Any-sync package, used in network communication but just for info
	// Yes, this is global between all instances of the app...
	// TODO: Create task to avoid it, use app instance.
	app.AppName = appName
	app.GitSummary = version
	app.GitCommit = commit
	app.BuildDate = date

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
				Name:    fGlobalIsDbg,
				Value:   false,
				Usage:   "Enable debug mode with detailed logging",
				EnvVars: []string{"ANY_SYNC_BUNDLE_DEBUG"},
			},
			&cli.PathFlag{
				Name:    fGlobalBundleConfigPath,
				Aliases: []string{"c"},
				Value:   "./data/bundle-config.yml",
				EnvVars: []string{"ANY_SYNC_BUNDLE_CONFIG"},
				Usage:   "Path to the bundle configuration YAML file",
			},
			&cli.PathFlag{
				Name:    fGlobalClientConfigPath,
				Aliases: []string{"cc"},
				// NOTE: Anytype support only yml, but not yaml
				// Fixed: https://github.com/anyproto/anytype-ts/pull/1186
				Value:   "./data/client-config.yml",
				EnvVars: []string{"ANY_SYNC_BUNDLE_CLIENT_CONFIG"},
				Usage:   "Path where write to the Anytype client configuration YAML file if needed",
			},
			&cli.PathFlag{
				Name:    fGlobalStoragePath,
				Value:   "./data/storage/",
				EnvVars: []string{"ANY_SYNC_BUNDLE_STORAGE"},
				Usage:   "Path to the bundle data directory (must be writable)",
			},
			&cli.StringSliceFlag{
				Name:    fGlobalInitExternalAddrs,
				Value:   cli.NewStringSlice("192.168.0.10", "example.local"),
				EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS"},
				Usage:   "Initial external addresses for the bundle",
			},
			&cli.StringFlag{
				Name:    fGlobalInitMongoURI,
				Value:   "mongodb://127.0.0.1:27017",
				EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_MONGO_URI"},
				Usage:   "Initial MongoDB URI for the bundle",
			},
			&cli.StringFlag{
				Name:    fGlobalInitRedisURI,
				Value:   "redis://127.0.0.1:6379",
				EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_REDIS_URI"},
				Usage:   "Initial Redis URI for the bundle",
			},
		},
		Before: setupLogger,
		Commands: []*cli.Command{
			cmdConfig(ctx),
			cmdMongo(ctx),
			cmdStart(ctx),
		},
	}

	return cliApp
}

// setupLogger configures the global logger with appropriate settings
func setupLogger(c *cli.Context) error {
	anyLogCfg := logger.Config{
		DefaultLevel: "",
		Format:       logger.PlaintextOutput,
	}

	if c.Bool(fGlobalIsDbg) {
		anyLogCfg.DefaultLevel = "debug"
		anyLogCfg.Format = logger.ColorizedOutput
	}

	anyLogCfg.ApplyGlobal()

	return nil
}

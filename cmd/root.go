package cmd

import (
	"context"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/urfave/cli/v2"
)

// Build-time version information
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const (
	appName = "any-sync-bundle"
)

// Global CLI flags
const (
	fGlobalIsDbg             = "debug"
	fGlobalStoragePath       = "storage"
	fGlobalBundleConfigPath  = "bundle-config"
	fGlobalClientConfigPath  = "client-config"
	fGlobalInitExternalAddrs = "initial-external-addrs"
	fGlobalInitMongoURI      = "initial-mongo-uri"
	fGlobalInitRedisURI      = "initial-redis-uri"
)

var log = logger.NewNamed("cli")

// Root returns the main CLI application with all commands and flags configured
func Root(ctx context.Context) *cli.App {
	cli.VersionPrinter = versionPrinter

	// Any-sync package, used in network communication but just for info
	// Yes, this is global between all instances of the app...
	// TODO: Create issue to avoid global app and use app instance instead
	app.AppName = appName
	app.GitSummary = version
	app.GitCommit = commit
	app.BuildDate = date

	return &cli.App{
		Name:    appName,
		Usage:   "📦 Anytype Self-Hosting: All-in-One Prepared for You",
		Version: version,
		Authors: []*cli.Author{{
			Name:  "Sergei G.",
			Email: "mail@grishy.dev",
		}},
		Flags:  buildGlobalFlags(),
		Before: setupLogger,
		Commands: []*cli.Command{
			cmdConfig(ctx),
			cmdMongo(ctx),
			cmdStart(ctx),
		},
	}
}

func buildGlobalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    fGlobalIsDbg,
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
			Value:   cli.NewStringSlice("192.168.8.214", "example.local"),
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS"},
			Usage:   "Initial external addresses for the bundle",
		},
		&cli.StringFlag{
			Name:    fGlobalInitMongoURI,
			Value:   "mongodb://127.0.0.1:27017/",
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_MONGO_URI"},
			Usage:   "Initial MongoDB URI for the bundle",
		},
		&cli.StringFlag{
			Name:    fGlobalInitRedisURI,
			Value:   "redis://127.0.0.1:6379/",
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_REDIS_URI"},
			Usage:   "Initial Redis URI for the bundle",
		},
	}
}

func setupLogger(c *cli.Context) error {
	cfg := logger.Config{
		Format: logger.PlaintextOutput,
	}

	if c.Bool(fGlobalIsDbg) {
		cfg.DefaultLevel = "debug"
		cfg.Format = logger.ColorizedOutput
	}

	cfg.ApplyGlobal()
	return nil
}

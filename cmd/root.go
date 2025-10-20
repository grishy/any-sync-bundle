package cmd

import (
	"context"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/urfave/cli/v2"
)

// Build-time version information.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const (
	appName = "any-sync-bundle"
)

// Global CLI flags.
const (
	flagDebug     = "debug"
	flagLogLevel  = "log-level"
	flagPprof     = "pprof"
	flagPprofAddr = "pprof-addr"
)

// Command-scoped flags for start commands.
const (
	flagStartBundleConfigPath = "bundle-config"
	flagStartClientConfigPath = "client-config"
	flagStartStoragePath      = "initial-storage"
	flagStartExternalAddrs    = "initial-external-addrs"
	flagStartMongoURI         = "initial-mongo-uri"
	flagStartRedisURI         = "initial-redis-uri"

	// S3 storage flags (optional - if not set, BadgerDB is used).
	flagStartS3Region         = "initial-s3-region"
	flagStartS3BlockBucket    = "initial-s3-block-bucket"
	flagStartS3IndexBucket    = "initial-s3-index-bucket"
	flagStartS3Endpoint       = "initial-s3-endpoint"
	flagStartS3Profile        = "initial-s3-profile"
	flagStartS3AccessKey      = "initial-s3-access-key"
	flagStartS3SecretKey      = "initial-s3-secret-key" //nolint:gosec // This is a flag name, not actual credentials
	flagStartS3ForcePathStyle = "initial-s3-force-path-style"
)

var log = logger.NewNamed("cli")

// Root returns the main CLI application with all commands and flags configured.
func Root(ctx context.Context) *cli.App {
	cli.VersionPrinter = versionPrinter

	// Any-sync package, used in network communication but just for info.
	// Yes, this is global between all instances of the app...
	// TODO: Create issue to avoid global app and use app instance instead.
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
			cmdStartAllInOne(ctx),
			cmdStartBundle(ctx),
		},
	}
}

func buildGlobalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    flagDebug,
			Usage:   "Enable debug mode with detailed logging",
			EnvVars: []string{"ANY_SYNC_BUNDLE_DEBUG"},
		},
		&cli.StringFlag{
			Name:    flagLogLevel,
			Usage:   "Log level (debug, info, warn, error, fatal)",
			EnvVars: []string{"ANY_SYNC_BUNDLE_LOG_LEVEL"},
		},
		&cli.BoolFlag{
			Name:    flagPprof,
			Usage:   "Enable pprof HTTP server for profiling",
			EnvVars: []string{"ANY_SYNC_BUNDLE_PPROF"},
		},
		&cli.StringFlag{
			Name:    flagPprofAddr,
			Usage:   "Address for pprof HTTP server (only used when --pprof is enabled)",
			Value:   "localhost:6060",
			EnvVars: []string{"ANY_SYNC_BUNDLE_PPROF_ADDR"},
		},
	}
}

func buildStartFlags() []cli.Flag {
	return []cli.Flag{
		&cli.PathFlag{
			Name:    flagStartBundleConfigPath,
			Aliases: []string{"c"},
			Value:   "./data/bundle-config.yml",
			EnvVars: []string{"ANY_SYNC_BUNDLE_CONFIG"},
			Usage:   "Path to the bundle configuration YAML file",
		},
		&cli.PathFlag{
			Name:    flagStartClientConfigPath,
			Aliases: []string{"cc"},
			Value:   "./data/client-config.yml",
			EnvVars: []string{"ANY_SYNC_BUNDLE_CLIENT_CONFIG"},
			Usage:   "Path where write to the Anytype client configuration YAML file if needed",
		},
		&cli.PathFlag{
			Name:    flagStartStoragePath,
			Value:   "./data/storage/",
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_STORAGE"},
			Usage:   "Initial path to the bundle data directory (must be writable)",
		},
		&cli.StringSliceFlag{
			Name:    flagStartExternalAddrs,
			Value:   cli.NewStringSlice("192.168.8.214", "example.local"),
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS"},
			Usage:   "Initial external addresses for the bundle",
		},
		&cli.StringFlag{
			Name:    flagStartMongoURI,
			Value:   "mongodb://127.0.0.1:27017/",
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_MONGO_URI"},
			Usage:   "Initial MongoDB URI for the bundle",
		},
		&cli.StringFlag{
			Name:    flagStartRedisURI,
			Value:   "redis://127.0.0.1:6379/",
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_REDIS_URI"},
			Usage:   "Initial Redis URI for the bundle",
		},

		// S3 Storage Flags (optional - if not provided, BadgerDB is used)
		&cli.StringFlag{
			Name:    flagStartS3Region,
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_S3_REGION"},
			Usage:   "S3 region (e.g., us-east-1). Required if using S3 storage.",
		},
		&cli.StringFlag{
			Name:    flagStartS3BlockBucket,
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_S3_BLOCK_BUCKET"},
			Usage:   "S3 bucket for file blocks. Required if using S3 storage.",
		},
		&cli.StringFlag{
			Name:    flagStartS3IndexBucket,
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_S3_INDEX_BUCKET"},
			Usage:   "S3 bucket for metadata index. Required if using S3 storage.",
		},
		&cli.StringFlag{
			Name:    flagStartS3Endpoint,
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_S3_ENDPOINT"},
			Usage:   "Custom S3 endpoint for S3-compatible services (e.g., MinIO)",
		},
		&cli.StringFlag{
			Name:    flagStartS3Profile,
			Value:   "default",
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_S3_PROFILE"},
			Usage:   "AWS profile name from ~/.aws/credentials",
		},
		&cli.StringFlag{
			Name:    flagStartS3AccessKey,
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_S3_ACCESS_KEY"},
			Usage:   "S3 access key (for static credentials)",
		},
		&cli.StringFlag{
			Name:    flagStartS3SecretKey,
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_S3_SECRET_KEY"},
			Usage:   "S3 secret key (for static credentials)",
		},
		&cli.BoolFlag{
			Name:    flagStartS3ForcePathStyle,
			EnvVars: []string{"ANY_SYNC_BUNDLE_INIT_S3_FORCE_PATH_STYLE"},
			Usage:   "Use path-style S3 URLs (required for MinIO and some S3-compatible services)",
		},
	}
}

func setupLogger(c *cli.Context) error {
	cfg := logger.Config{
		Format:       logger.PlaintextOutput,
		DefaultLevel: "info",
	}

	// --log-level flag
	if logLevel := c.String(flagLogLevel); logLevel != "" {
		cfg.DefaultLevel = logLevel
	}

	// --debug flag (overrides everything)
	if c.Bool(flagDebug) {
		cfg.DefaultLevel = "debug"
		cfg.Format = logger.ColorizedOutput
	}

	cfg.ApplyGlobal()
	return nil
}

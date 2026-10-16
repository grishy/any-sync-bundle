package cmd

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/grishy/any-sync-bundle/config"
	"github.com/grishy/any-sync-bundle/lightnode"
)

const (
	// Bundle services share one shutdown deadline.
	servicesShutdownTimeout = 30 * time.Second

	// WaitDelay is the SIGTERM grace period before os/exec sends SIGKILL.
	infraProcessWaitDelay = 60 * time.Second
	// Allow cmd.Wait to publish the exit after a forced kill.
	infraProcessReapMargin = 5 * time.Second
	infraShutdownTimeout   = infraProcessWaitDelay + infraProcessReapMargin

	// Keep the process watchdog outside all cleanup deadlines.
	shutdownWatchdogMargin = 10 * time.Second
	// ShutdownTimeout bounds process shutdown after root cancellation. Container
	// stop grace periods must be longer so the application owns forced exit.
	ShutdownTimeout = servicesShutdownTimeout + infraShutdownTimeout + shutdownWatchdogMargin

	// Lifecycle events.
	bundleReadyEvent            = "bundle_ready"
	bundleShutdownCompleteEvent = "bundle_shutdown_complete"
)

func cmdStartAllInOne(ctx context.Context, stopRoot context.CancelCauseFunc) *cli.Command {
	return &cli.Command{
		Name:  "start-all-in-one",
		Usage: "Start bundle together with embedded MongoDB and Redis",
		Flags: buildStartFlags(),
		Action: func(c *cli.Context) error {
			if err := assertContainerRuntime(); err != nil {
				return err
			}
			printWelcomeMsg()
			bundleCfg := loadBundleConfig(c)
			applyAllInOneDefaults(bundleCfg)
			startPprofServer(ctx, c)

			infra := newInfraSuite(ctx, stopRoot, infraProcessWaitDelay)
			err := startAllInOneInfra(ctx, infra)
			switch {
			case err == nil:
				err = runBundleServices(ctx, stopRoot, bundleCfg)
			case stoppedByOperator(ctx):
				log.Info("startup interrupted", zap.Error(err))
				err = nil
			default:
				stopRoot(err)
			}

			shutdownCtx, cancelShutdown := context.WithTimeout(context.WithoutCancel(ctx), infraShutdownTimeout)
			defer cancelShutdown()
			if err = errors.Join(err, infra.stop(shutdownCtx)); err != nil {
				return err
			}
			reportShutdownComplete()
			return nil
		},
	}
}

func cmdStartBundle(ctx context.Context, stopRoot context.CancelCauseFunc) *cli.Command {
	return &cli.Command{
		Name:  "start-bundle",
		Usage: "Start bundle services and use external MongoDB/Redis",
		Flags: buildStartFlags(),
		Action: func(c *cli.Context) error {
			printWelcomeMsg()
			bundleCfg := loadBundleConfig(c)
			startPprofServer(ctx, c)

			if err := runBundleServices(ctx, stopRoot, bundleCfg); err != nil {
				return err
			}
			reportShutdownComplete()
			return nil
		},
	}
}

func runBundleServices(ctx context.Context, stopRoot context.CancelCauseFunc, bundleCfg *config.Config) error {
	printConfigurationInfo(bundleCfg)
	return runServices(ctx, stopRoot, lightnode.NewBundle(bundleCfg.NodeConfigs()))
}

// loadBundleConfig loads the bundle config, creating it from the start flags on
// first start, and regenerates the client config on every start.
func loadBundleConfig(c *cli.Context) *config.Config {
	bundleCfg := config.LoadOrCreate(&config.CreateOptions{
		CfgPath:       c.String(flagStartBundleConfigPath),
		StorePath:     c.String(flagStartStoragePath),
		MongoURI:      c.String(flagStartMongoURI),
		RedisURI:      c.String(flagStartRedisURI),
		ExternalAddrs: c.StringSlice(flagStartExternalAddrs),

		// S3 configuration (optional) - credentials via AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY env vars
		S3Bucket:         c.String(flagStartS3Bucket),
		S3Endpoint:       c.String(flagStartS3Endpoint),
		S3Region:         c.String(flagStartS3Region),
		S3ForcePathStyle: c.Bool(flagStartS3ForcePathStyle),

		// Filenode configuration
		FilenodeDefaultLimit: c.Uint64(flagStartFilenodeDefaultLimit),
	})
	bundleCfg.WriteClientConfig(c.String(flagStartClientConfigPath))
	return bundleCfg
}

func assertContainerRuntime() error {
	// Docker creates /.dockerenv, Podman creates /run/.containerenv.
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return nil
	}
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return nil
	}

	return errors.New(
		"start-all-in-one is only supported inside the official container image; please run the all-in-one container or use start-bundle with external MongoDB/Redis",
	)
}

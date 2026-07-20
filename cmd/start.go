package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/anyproto/any-sync/app/logger"
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

func cmdStartAllInOne(ctx context.Context, cancelRoot context.CancelFunc) *cli.Command {
	return &cli.Command{
		Name:  "start-all-in-one",
		Usage: "Start bundle together with embedded MongoDB and Redis",
		Flags: buildStartFlags(),
		Action: func(c *cli.Context) error {
			if err := assertContainerRuntime(); err != nil {
				return err
			}

			printWelcomeMsg()

			bundleCfg, err := prepareBundleConfig(c)
			if err != nil {
				return err
			}

			applyAllInOneDefaults(bundleCfg)
			startPprofServer(ctx, c)

			infra := newInfraSuite(ctx, cancelRoot, infraProcessWaitDelay)

			startErr := startAllInOneInfra(ctx, infra)
			var bundleErr error
			if startErr != nil {
				rootInterrupted := isRootInterruption(ctx, startErr)
				cancelRoot()
				if rootInterrupted {
					startErr = nil
				}
			} else {
				bundleErr = runBundleServices(ctx, cancelRoot, bundleCfg)
			}

			shutdownCtx, cancelShutdown := context.WithTimeout(
				context.WithoutCancel(ctx),
				infraShutdownTimeout,
			)
			infraErr := infra.stop(shutdownCtx)
			cancelShutdown()

			resultErr := errors.Join(startErr, bundleErr, infraErr)
			if resultErr != nil {
				return resultErr
			}
			reportShutdownComplete()
			return nil
		},
	}
}

func cmdStartBundle(ctx context.Context, cancelRoot context.CancelFunc) *cli.Command {
	return &cli.Command{
		Name:  "start-bundle",
		Usage: "Start bundle services and use external MongoDB/Redis",
		Flags: buildStartFlags(),
		Action: func(c *cli.Context) error {
			printWelcomeMsg()

			bundleCfg, err := prepareBundleConfig(c)
			if err != nil {
				return err
			}

			startPprofServer(ctx, c)

			err = runBundleServices(ctx, cancelRoot, bundleCfg)
			if err != nil {
				return err
			}
			reportShutdownComplete()
			return nil
		},
	}
}

func runBundleServices(
	ctx context.Context,
	cancelRoot context.CancelFunc,
	bundleCfg *config.Config,
) error {
	printConfigurationInfo(bundleCfg)

	nodeCfgs := bundleCfg.NodeConfigs()
	bundle := lightnode.NewBundle(nodeCfgs)

	services := []bundleService{
		{name: "coordinator", app: bundle.Coordinator},
		{name: "consensus", app: bundle.Consensus},
		{name: "filenode", app: bundle.FileNode},
		{name: "sync", app: bundle.Sync},
	}

	if err := startServices(ctx, cancelRoot, services, bundleCfg); err != nil {
		if isRootInterruption(ctx, err) {
			return nil
		}
		return err
	}

	select {
	case <-ctx.Done():
	default:
		emitBundleEvent(bundleReadyEvent)
		printStartupMsg()
		<-ctx.Done()
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(
		context.WithoutCancel(ctx),
		servicesShutdownTimeout,
	)
	shutdownErr := shutdownServices(shutdownCtx, services)
	cancelShutdown()
	return shutdownErr
}

// A wrapped or joined interruption may contain another failure, so only the
// exact root error represents an operator-requested stop.
func isRootInterruption(ctx context.Context, err error) bool {
	rootErr := ctx.Err()
	return rootErr != nil && err == rootErr //nolint:errorlint // Error traversal would weaken the invariant.
}

func prepareBundleConfig(c *cli.Context) (*config.Config, error) {
	bundleCfg := loadOrCreateConfig(c, log)
	clientCfgPath := c.String(flagStartClientConfigPath)

	if err := writeClientConfig(bundleCfg, clientCfgPath); err != nil {
		return nil, err
	}

	return bundleCfg, nil
}

func loadOrCreateConfig(c *cli.Context, log logger.CtxLogger) *config.Config {
	cfgPath := c.String(flagStartBundleConfigPath)
	log.Info("loading config")

	if _, err := os.Stat(cfgPath); err == nil {
		log.Info("loaded existing config")
		return config.Load(cfgPath)
	}

	log.Info("creating new config")
	return config.CreateWrite(&config.CreateOptions{
		CfgPath:       cfgPath,
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
}

func writeClientConfig(cfg *config.Config, path string) error {
	const clientConfigMode = 0o644

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("failed to create client config directory: %w", err)
	}

	yamlData, err := cfg.YamlClientConfig()
	if err != nil {
		return fmt.Errorf("failed to generate client config: %w", err)
	}

	if err = os.WriteFile(path, yamlData, clientConfigMode); err != nil {
		return fmt.Errorf("failed to write client config: %w", err)
	}

	log.Info("client configuration written", zap.String("path", path))
	return nil
}

func emitBundleEvent(event string, fields ...zap.Field) {
	fields = append(fields, zap.String("event", event))
	log.Info("bundle lifecycle event", fields...)
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

func reportShutdownComplete() {
	emitBundleEvent(bundleShutdownCompleteEvent)
	fmt.Printf(`
┌───────────────────────────────────────────────────────────────────┐

                 AnySync Bundle shutdown complete!
                     All services are stopped.

└───────────────────────────────────────────────────────────────────┘
`)
	log.Info("→ Goodbye!")
}

func printConfigurationInfo(cfg *config.Config) {
	log.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Info("Configuration Summary")
	log.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Info("→ Network Configuration",
		zap.String("tcp_listen", cfg.Network.ListenTCPAddr),
		zap.String("udp_listen", cfg.Network.ListenUDPAddr))
	log.Info("→ MongoDB Configuration",
		zap.String("coordinator_uri", cfg.Coordinator.MongoConnect),
		zap.String("coordinator_db", cfg.Coordinator.MongoDatabase),
		zap.String("consensus_uri", cfg.Consensus.MongoConnect),
		zap.String("consensus_db", cfg.Consensus.MongoDatabase))
	log.Info("→ Redis Configuration",
		zap.String("filenode_uri", cfg.FileNode.RedisConnect))
	log.Info("→ External Addresses",
		zap.Strings("addresses", cfg.ExternalAddr))
	log.Info("→ Node Identity",
		zap.String("peer_id", cfg.Account.PeerId))
	log.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
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

func startPprofServer(ctx context.Context, c *cli.Context) {
	if !c.Bool(flagPprof) {
		return
	}

	addr := c.String(flagPprofAddr)
	log.Info("🔍 starting pprof HTTP server",
		zap.String("addr", addr),
		zap.String("url", "http://"+addr+"/debug/pprof/"))

	// A private mux avoids exposing pprof through the process-wide default mux.
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("pprof server failed", zap.Error(err))
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancelShutdown := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancelShutdown()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Warn("pprof server shutdown failed", zap.Error(err))
		}
	}()

	log.Info("✓ pprof server started - use 'go tool pprof' to analyze",
		zap.String("cpu_profile", "go tool pprof http://"+addr+"/debug/pprof/profile?seconds=30"),
		zap.String("heap_profile", "go tool pprof http://"+addr+"/debug/pprof/heap"),
		zap.String("goroutine_profile", "go tool pprof http://"+addr+"/debug/pprof/goroutine"))
}

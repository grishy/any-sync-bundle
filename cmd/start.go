package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	bundleConfig "github.com/grishy/any-sync-bundle/config"
	"github.com/grishy/any-sync-bundle/lightnode"
)

type node struct {
	name string
	app  *app.App
}

const (
	serviceShutdownTimeout = 10 * time.Second
	clientConfigMode       = 0o644

	dockerMongoPort        = "27017"
	dockerRedisPort        = "6379"
	dockerMongoURI         = "mongodb://127.0.0.1:27017/"
	dockerMongoMajorityURI = "mongodb://127.0.0.1:27017/?w=majority"
	dockerRedisURI         = "redis://127.0.0.1:6379/"
	dockerMongoDataDir     = "/data/mongo"
	dockerRedisDataDir     = "/data/redis"
)

func cmdStartAllInOne(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "start-all-in-one",
		Usage: "Start bundle together with embedded MongoDB and Redis",
		Flags: buildStartFlags(),
		Action: func(cCtx *cli.Context) error {
			if err := assertContainerRuntime(); err != nil {
				return err
			}

			printWelcomeMsg()

			bundleCfg, err := prepareBundleConfig(cCtx)
			if err != nil {
				return err
			}

			applyAllInOneDefaults(bundleCfg)

			// Start pprof server if enabled
			startPprofServer(ctx, cCtx)

			infra, err := startAllInOneInfra(ctx)
			if err != nil {
				return err
			}
			defer infra.stop()

			return runBundleServices(ctx, bundleCfg)
		},
	}
}

func cmdStartBundle(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "start-bundle",
		Usage: "Start bundle services and use external MongoDB/Redis",
		Flags: buildStartFlags(),
		Action: func(cCtx *cli.Context) error {
			printWelcomeMsg()

			bundleCfg, err := prepareBundleConfig(cCtx)
			if err != nil {
				return err
			}

			// Start pprof server if enabled
			startPprofServer(ctx, cCtx)

			return runBundleServices(ctx, bundleCfg)
		},
	}
}

func runBundleServices(ctx context.Context, bundleCfg *bundleConfig.Config) error {
	printConfigurationInfo(bundleCfg)

	cfgNodes := bundleCfg.NodeConfigs()
	bundle := lightnode.NewBundle(cfgNodes)

	apps := []node{
		{name: "coordinator", app: bundle.Coordinator},
		{name: "consensus", app: bundle.Consensus},
		{name: "filenode", app: bundle.FileNode},
		{name: "sync", app: bundle.Sync},
	}

	if err := startServices(ctx, apps, bundleCfg); err != nil {
		return err
	}

	printStartupMsg()

	<-ctx.Done()

	shutdownServices(apps)
	printShutdownMsg()

	log.Info("→ Goodbye!")
	return nil
}

func prepareBundleConfig(cCtx *cli.Context) (*bundleConfig.Config, error) {
	bundleCfg := loadOrCreateConfig(cCtx, log)
	clientCfgPath := cCtx.String(flagStartClientConfigPath)

	if err := writeClientConfig(bundleCfg, clientCfgPath); err != nil {
		return nil, err
	}

	return bundleCfg, nil
}

func loadOrCreateConfig(cCtx *cli.Context, log logger.CtxLogger) *bundleConfig.Config {
	cfgPath := cCtx.String(flagStartBundleConfigPath)
	log.Info("loading config")

	if _, err := os.Stat(cfgPath); err == nil {
		log.Info("loaded existing config")
		return bundleConfig.Load(cfgPath)
	}

	log.Info("creating new config")
	return bundleConfig.CreateWrite(&bundleConfig.CreateOptions{
		CfgPath:       cfgPath,
		StorePath:     cCtx.String(flagStartStoragePath),
		MongoURI:      cCtx.String(flagStartMongoURI),
		RedisURI:      cCtx.String(flagStartRedisURI),
		ExternalAddrs: cCtx.StringSlice(flagStartExternalAddrs),

		// S3 configuration (optional) - credentials via AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY env vars
		S3Bucket:         cCtx.String(flagStartS3Bucket),
		S3Endpoint:       cCtx.String(flagStartS3Endpoint),
		S3Region:         cCtx.String(flagStartS3Region),
		S3ForcePathStyle: cCtx.Bool(flagStartS3ForcePathStyle),

		// Filenode configuration
		FilenodeDefaultLimit: cCtx.Uint64(flagStartFilenodeDefaultLimit),
	})
}

func writeClientConfig(cfg *bundleConfig.Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("failed to create client config directory: %w", err)
	}

	yamlData, err := cfg.YamlClientConfig()
	if err != nil {
		return fmt.Errorf("failed to generate client config: %w", err)
	}

	if writeErr := os.WriteFile(path, yamlData, clientConfigMode); writeErr != nil {
		return fmt.Errorf("failed to write client config: %w", writeErr)
	}

	log.Info("client configuration written", zap.String("path", path))
	return nil
}

func startAllInOneInfra(ctx context.Context) (*infraSuite, error) {
	// Create required data directories with proper permissions
	if err := os.MkdirAll(dockerMongoDataDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create mongo data dir: %w", err)
	}
	if err := os.MkdirAll(dockerRedisDataDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create redis data dir: %w", err)
	}

	log.Info("data directories prepared",
		zap.String("mongo", dockerMongoDataDir),
		zap.String("redis", dockerRedisDataDir))

	mongoArgs := []string{
		"--port", dockerMongoPort,
		"--dbpath", dockerMongoDataDir,
		"--replSet", defaultMongoReplica,
		"--bind_ip", "127.0.0.1",
	}

	log.Info("starting embedded MongoDB",
		zap.String("addr", "127.0.0.1:"+dockerMongoPort),
		zap.String("dbpath", dockerMongoDataDir))

	mongoProc, mongoErr := newInfraProcess(ctx, "mongo", "mongod", mongoArgs...)
	if mongoErr != nil {
		return nil, fmt.Errorf("start mongod: %w", mongoErr)
	}

	redisArgs := []string{
		"--port", dockerRedisPort,
		"--dir", dockerRedisDataDir,
		"--appendonly", "yes",
		"--maxmemory", "256mb",
		"--maxmemory-policy", "noeviction",
		"--protected-mode", "no",
		"--bind", "127.0.0.1",
		"--loadmodule", "/opt/redis-stack/lib/redisbloom.so",
	}

	log.Info("starting embedded Redis",
		zap.String("addr", "127.0.0.1:"+dockerRedisPort),
		zap.String("dir", dockerRedisDataDir))

	redisProc, redisErr := newInfraProcess(ctx, "redis", "redis-server", redisArgs...)
	if redisErr != nil {
		mongoProc.stop()
		_ = mongoProc.wait()
		return nil, fmt.Errorf("start redis-server: %w", redisErr)
	}

	suite := &infraSuite{
		processes: []*infraProcess{mongoProc, redisProc},
	}

	// Wait for MongoDB TCP ready (or process death)
	mongoAddr := net.JoinHostPort("127.0.0.1", dockerMongoPort)
	if err := waitForTCPOrExit(mongoAddr, 180*time.Second, mongoProc); err != nil {
		suite.stop()
		if isIllegalInstruction(err) {
			printMongoAVXError()
			return nil, &MongoAVXError{Cause: err}
		}
		return nil, fmt.Errorf("mongodb not ready: %w", err)
	}

	if initErr := initReplicaSetAction(ctx, defaultMongoReplica, dockerMongoURI); initErr != nil {
		suite.stop()
		return nil, fmt.Errorf("init replica set: %w", initErr)
	}

	// Wait for Redis TCP ready (or process death)
	redisAddr := net.JoinHostPort("127.0.0.1", dockerRedisPort)
	if err := waitForTCPOrExit(redisAddr, 30*time.Second, redisProc); err != nil {
		suite.stop()
		return nil, fmt.Errorf("redis not ready: %w", err)
	}

	return suite, nil
}

func applyAllInOneDefaults(cfg *bundleConfig.Config) {
	cfg.Coordinator.MongoConnect = dockerMongoURI
	cfg.Consensus.MongoConnect = dockerMongoMajorityURI
	cfg.FileNode.RedisConnect = dockerRedisURI
}

type infraProcess struct {
	name    string
	cmd     *exec.Cmd
	done    chan struct{} // Closed when process exits
	exitErr error         // Set when process exits
}

func newInfraProcess(ctx context.Context, name, bin string, args ...string) (*infraProcess, error) {
	cmd := exec.CommandContext(ctx, bin, args...)

	stdout, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		return nil, fmt.Errorf("failed to capture stdout for %s: %w", name, pipeErr)
	}

	stderr, errPipe := cmd.StderrPipe()
	if errPipe != nil {
		return nil, fmt.Errorf("failed to capture stderr for %s: %w", name, errPipe)
	}

	if startErr := cmd.Start(); startErr != nil {
		return nil, fmt.Errorf("failed to start %s: %w", name, startErr)
	}

	p := &infraProcess{
		name: name,
		cmd:  cmd,
		done: make(chan struct{}),
	}

	go func() {
		p.exitErr = cmd.Wait()
		close(p.done)
	}()

	go streamPipe(name, stdout)
	go streamPipe(name, stderr)

	return p, nil
}

func (p *infraProcess) stop() {
	if p == nil || p.cmd.Process == nil {
		return
	}

	if p.cmd.ProcessState != nil && p.cmd.ProcessState.Exited() {
		return
	}

	if err := p.cmd.Process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
		log.Warn("failed to interrupt process",
			zap.String("process", p.name),
			zap.Error(err))
	}
}

func (p *infraProcess) wait() error {
	if p == nil {
		return nil
	}

	<-p.done
	return p.exitErr
}

type infraSuite struct {
	processes []*infraProcess
}

func (s *infraSuite) stop() {
	if s == nil {
		return
	}

	for _, p := range s.processes {
		p.stop()
	}

	for _, p := range s.processes {
		if err := p.wait(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, os.ErrProcessDone) {
			log.Debug("process terminated with error",
				zap.String("process", p.name),
				zap.Error(err))
		}
	}
}

func streamPipe(name string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		fmt.Printf("[%s] %s\n", name, scanner.Text())
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		log.Warn("log stream error",
			zap.String("process", name),
			zap.Error(err))
	}
}

// isIllegalInstruction checks if an error indicates SIGILL.
// This typically means the CPU lacks required instructions (e.g., AVX for MongoDB 5.0+).
func isIllegalInstruction(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "illegal instruction")
}

// MongoAVXError indicates MongoDB failed due to missing AVX CPU support.
type MongoAVXError struct {
	Cause error
}

func (e *MongoAVXError) Error() string {
	return fmt.Sprintf("mongodb requires AVX CPU support: %v", e.Cause)
}

func (e *MongoAVXError) Unwrap() error {
	return e.Cause
}

// printMongoAVXError displays a user-friendly error message for AVX failures.
func printMongoAVXError() {
	const msg = `
┌─────────────────────────────────────────────────────────────────────┐
│  MongoDB failed to start: CPU does not support AVX instructions     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  MongoDB 5.0+ requires AVX CPU instructions, but your processor     │
│  does not support them. The process was terminated by the kernel    │
│  with SIGILL (Illegal Instruction).                                 │
│                                                                     │
│  Solutions:                                                         │
│    • Use external MongoDB 4.4 with the start-bundle command         │
│    • See compose.external.yml for example setup                     │
│                                                                     │
│  More info: https://github.com/grishy/any-sync-bundle/pull/39       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
`
	fmt.Fprint(os.Stderr, msg)
}

// waitForTCPReady polls the address until a TCP connection succeeds or timeout is reached.
func waitForTCPReady(addr string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dialer := &net.Dialer{
		Timeout: 100 * time.Millisecond,
	}

	attempt := 0
	startTime := time.Now()

	for {
		attempt++
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			elapsed := time.Since(startTime)
			log.Info("TCP listener ready",
				zap.String("addr", addr),
				zap.Int("attempts", attempt),
				zap.Duration("elapsed", elapsed))
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("TCP listener not ready after %v (attempts: %d): %w", timeout, attempt, ctx.Err())
		default:
		}

		if attempt%5 == 0 {
			log.Debug("waiting for TCP listener",
				zap.String("addr", addr),
				zap.Int("attempts", attempt),
				zap.Duration("elapsed", time.Since(startTime)))
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// waitForTCPOrExit polls the address until TCP connects, process exits, or timeout.
// Returns nil if TCP is ready.
// Returns process exit error if process dies.
// Returns timeout error if deadline reached.
func waitForTCPOrExit(addr string, timeout time.Duration, proc *infraProcess) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dialer := &net.Dialer{
		Timeout: 100 * time.Millisecond,
	}

	attempt := 0
	startTime := time.Now()

	for {
		attempt++

		// Check if process died
		select {
		case <-proc.done:
			return proc.exitErr
		default:
		}

		// Try TCP connect
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			log.Info("TCP listener ready",
				zap.String("addr", addr),
				zap.Int("attempts", attempt),
				zap.Duration("elapsed", time.Since(startTime)))
			return nil
		}

		// Check for timeout
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout after %v (attempts: %d)", timeout, attempt)
		default:
		}

		if attempt%5 == 0 {
			log.Debug("waiting for TCP listener",
				zap.String("addr", addr),
				zap.Int("attempts", attempt),
				zap.Duration("elapsed", time.Since(startTime)))
		}

		// Wait before retry, watching for process exit
		select {
		case <-proc.done:
			return proc.exitErr
		case <-ctx.Done():
			return fmt.Errorf("timeout after %v (attempts: %d)", timeout, attempt)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// startServices initializes and runs all bundle services using a custom two-phase approach.
//
// Why we can't use app.Start() directly:
// The bundle architecture has 4 separate apps (coordinator, consensus, filenode, sync) that
// share a single DRPC multiplexer from the coordinator's server component. If we call
// app.Start() sequentially on each service, a race condition occurs:
//
//  1. coordinator.Start() = Init (registers handlers) + Run (starts network listeners)
//  2. Network is now accepting connections and calling mux.HandleRPC()
//  3. consensus.Start() = Init tries to register handlers on the same mux
//  4. RACE: goroutine reads mux map (HandleRPC) while another writes to it (register)
func startServices(ctx context.Context, apps []node, cfg *bundleConfig.Config) error {
	log.Info("initiating service startup", zap.Int("count", len(apps)))
	log.Info("━━━ Phase 1: Initializing all services ━━━")

	initialized := []node{}
	for _, app := range apps {
		if err := initOneApp(app); err != nil {
			shutdownServices(initialized)
			return err
		}
		initialized = append(initialized, app)
	}
	log.Info("✓ all services initialized, all DRPC handlers registered")

	// Phase 2: Run all services
	// Track which services have been successfully Run() to avoid closing
	// components that were Init'd but never Run'd (they may have nil pointers).
	log.Info("━━━ Phase 2: Running all services ━━━")
	running := []node{}
	for _, app := range initialized {
		if err := runOneApp(ctx, app, cfg); err != nil {
			shutdownServices(running)
			return err
		}
		running = append(running, app)
	}
	log.Info("✓ all services running")

	return nil
}

// initOneApp initializes all components for a single app.
func initOneApp(n node) error {
	log.Info("▶ initializing service", zap.String("name", n.name))

	var firstError error
	n.app.IterateComponents(func(c app.Component) {
		if firstError != nil {
			return
		}
		if err := c.Init(n.app); err != nil {
			firstError = fmt.Errorf("component '%s': %w", c.Name(), err)
			log.Error("component init failed",
				zap.String("service", n.name),
				zap.String("component", c.Name()),
				zap.Error(err))
		}
	})

	if firstError != nil {
		return fmt.Errorf("service '%s' init failed: %w", n.name, firstError)
	}

	log.Info("✓ service initialized", zap.String("name", n.name))
	return nil
}

// runOneApp runs all runnable components for a single app.
func runOneApp(ctx context.Context, n node, cfg *bundleConfig.Config) error {
	log.Info("▶ running service", zap.String("name", n.name))

	var firstError error
	n.app.IterateComponents(func(c app.Component) {
		if firstError != nil {
			return
		}
		if runnable, ok := c.(app.ComponentRunnable); ok {
			if err := runnable.Run(ctx); err != nil {
				firstError = fmt.Errorf("component '%s': %w", runnable.Name(), err)
				log.Error("component run failed",
					zap.String("service", n.name),
					zap.String("component", runnable.Name()),
					zap.Error(err))
			}
		}
	})

	if firstError != nil {
		return fmt.Errorf("service '%s' run failed: %w", n.name, firstError)
	}

	// Coordinator-specific: wait for network to be ready
	if n.name == "coordinator" {
		addr := cfg.Network.ListenTCPAddr
		log.Info("waiting for coordinator TCP listener", zap.String("addr", addr))

		if err := waitForTCPReady(addr, 5*time.Second); err != nil {
			return fmt.Errorf("coordinator network not ready: %w", err)
		}

		log.Info("coordinator network ready")
	}

	log.Info("✓ service running", zap.String("name", n.name))
	return nil
}

func shutdownServices(apps []node) {
	log.Info("⚡ initiating service shutdown", zap.Int("count", len(apps)))

	for _, a := range slices.Backward(apps) {
		log.Info("▶ stopping service", zap.String("name", a.name))

		ctx, cancel := context.WithTimeout(context.Background(), serviceShutdownTimeout)

		if err := a.app.Close(ctx); err != nil {
			log.Error("✗ service shutdown failed", zap.String("name", a.name), zap.Error(err))
		} else {
			log.Info("✓ service stopped successfully", zap.String("name", a.name))
		}

		cancel()
	}
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

func printShutdownMsg() {
	fmt.Printf(`
┌───────────────────────────────────────────────────────────────────┐

                 AnySync Bundle shutdown complete!
                     All services are stopped.

└───────────────────────────────────────────────────────────────────┘
`)
}

func printConfigurationInfo(cfg *bundleConfig.Config) {
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

func startPprofServer(ctx context.Context, cCtx *cli.Context) {
	if !cCtx.Bool(flagPprof) {
		return
	}

	addr := cCtx.String(flagPprofAddr)
	log.Info("🔍 starting pprof HTTP server",
		zap.String("addr", addr),
		zap.String("url", "http://"+addr+"/debug/pprof/"))

	// Create a custom mux and manually register pprof handlers
	// This avoids gosec G108 warning and is more secure than using the default mux
	mux := http.NewServeMux()

	// Register pprof handlers
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Register additional profile types
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Warn("pprof server shutdown failed", zap.Error(err))
		}
	}()

	log.Info("✓ pprof server started - use 'go tool pprof' to analyze",
		zap.String("cpu_profile", "go tool pprof http://"+addr+"/debug/pprof/profile?seconds=30"),
		zap.String("heap_profile", "go tool pprof http://"+addr+"/debug/pprof/heap"),
		zap.String("goroutine_profile", "go tool pprof http://"+addr+"/debug/pprof/goroutine"))
}

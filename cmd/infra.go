package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/grishy/any-sync-bundle/config"
)

const (
	dockerMongoPort        = "27017"
	dockerRedisPort        = "6379"
	dockerMongoURI         = "mongodb://127.0.0.1:27017/"
	dockerMongoMajorityURI = "mongodb://127.0.0.1:27017/?w=majority"
	dockerRedisURI         = "redis://127.0.0.1:6379/"
	dockerMongoDataDir     = "/data/mongo"
	dockerRedisDataDir     = "/data/redis"
)

func startAllInOneInfra(ctx context.Context, infra *infraSuite) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Create required data directories with proper permissions
	if err := os.MkdirAll(dockerMongoDataDir, 0o750); err != nil {
		return fmt.Errorf("failed to create mongo data dir: %w", err)
	}
	if err := os.MkdirAll(dockerRedisDataDir, 0o750); err != nil {
		return fmt.Errorf("failed to create redis data dir: %w", err)
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

	mongoProc, err := infra.start("mongo", "mongod", mongoArgs...)
	if err != nil {
		if isRootInterruption(ctx, err) {
			return err
		}
		return fmt.Errorf("start mongod: %w", err)
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

	redisProc, err := infra.start("redis", "redis-server", redisArgs...)
	if err != nil {
		if isRootInterruption(ctx, err) {
			return err
		}
		return fmt.Errorf("start redis-server: %w", err)
	}

	// Wait for MongoDB TCP ready (or process death)
	mongoAddr := net.JoinHostPort("127.0.0.1", dockerMongoPort)
	if err = waitForTCPOrExit(ctx, mongoAddr, 180*time.Second, mongoProc); err != nil {
		if isRootInterruption(ctx, err) {
			return err
		}
		if isIllegalInstruction(err) {
			printMongoAVXError()
			return &MongoAVXError{Cause: err}
		}
		return fmt.Errorf("mongodb not ready: %w", err)
	}

	if err = initReplicaSetAction(ctx, defaultMongoReplica, dockerMongoURI); err != nil {
		if isRootInterruption(ctx, err) {
			return err
		}
		return fmt.Errorf("init replica set: %w", err)
	}

	// Wait for Redis TCP ready (or process death)
	redisAddr := net.JoinHostPort("127.0.0.1", dockerRedisPort)
	if err = waitForTCPOrExit(ctx, redisAddr, 30*time.Second, redisProc); err != nil {
		if isRootInterruption(ctx, err) {
			return err
		}
		return fmt.Errorf("redis not ready: %w", err)
	}

	return nil
}

func applyAllInOneDefaults(cfg *config.Config) {
	cfg.Coordinator.MongoConnect = dockerMongoURI
	cfg.Consensus.MongoConnect = dockerMongoMajorityURI
	cfg.FileNode.RedisConnect = dockerRedisURI
}

type infraExitError struct {
	name string
	err  error
}

func (e infraExitError) Error() string {
	if e.err == nil {
		return fmt.Sprintf("%s exited unexpectedly", e.name)
	}
	return fmt.Sprintf("%s exited unexpectedly: %v", e.name, e.err)
}

func (e infraExitError) Unwrap() error {
	return e.err
}

type infraProcess struct {
	name    string
	process *os.Process
	done    chan struct{}
	waitErr error
}

func (p *infraProcess) unexpectedExit() infraExitError {
	return infraExitError{name: p.name, err: p.waitErr}
}

type infraSuite struct {
	rootCtx        context.Context
	childrenCtx    context.Context
	cancelChildren context.CancelFunc
	cancelRoot     context.CancelFunc
	waitDelay      time.Duration
	mu             sync.Mutex
	stopping       bool
	processes      []*infraProcess
}

func newInfraSuite(
	rootCtx context.Context,
	cancelRoot context.CancelFunc,
	waitDelay time.Duration,
) *infraSuite {
	childrenCtx, cancelChildren := context.WithCancel(context.WithoutCancel(rootCtx))
	return &infraSuite{
		rootCtx:        rootCtx,
		childrenCtx:    childrenCtx,
		cancelChildren: cancelChildren,
		cancelRoot:     cancelRoot,
		waitDelay:      waitDelay,
	}
}

func (s *infraSuite) start(name, bin string, args ...string) (*infraProcess, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.rootCtx.Err(); err != nil {
		return nil, err
	}
	if s.stopping {
		return nil, fmt.Errorf("start %s after embedded process shutdown", name)
	}

	// Production callers provide fixed binaries; tests intentionally provide
	// the test binary.
	//nolint:gosec // Production callers fix the command specification.
	cmd := exec.CommandContext(s.childrenCtx, bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = s.waitDelay

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", name, err)
	}

	process := &infraProcess{
		name:    name,
		process: cmd.Process,
		done:    make(chan struct{}),
	}
	s.processes = append(s.processes, process)

	go func() {
		waitErr := cmd.Wait()
		s.mu.Lock()
		process.waitErr = waitErr
		if !s.stopping {
			s.cancelRoot()
		}
		close(process.done)
		s.mu.Unlock()
	}()

	return process, nil
}

func (p *infraProcess) shutdownError() error {
	err := p.waitErr
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return nil
	}

	shutdownErr := fmt.Errorf("%s shutdown failed: %w", p.name, err)
	exitErr, ok := errors.AsType[*exec.ExitError](err)
	if !ok {
		return shutdownErr
	}
	waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return shutdownErr
	}
	if !waitStatus.Signaled() {
		return shutdownErr
	}
	if waitStatus.Signal() == syscall.SIGTERM {
		return nil
	}
	if waitStatus.Signal() == syscall.SIGKILL {
		return fmt.Errorf("%s required forced shutdown: %w", p.name, err)
	}

	return shutdownErr
}

func (s *infraSuite) stop(ctx context.Context) error {
	// Closing done is the Wait owner's publication point. A result published
	// before this handoff remains unexpected. Every later result belongs to
	// intentional shutdown.
	s.mu.Lock()
	s.stopping = true
	running := make([]*infraProcess, 0, len(s.processes))
	unexpected := make([]*infraProcess, 0, len(s.processes))
	for _, process := range s.processes {
		select {
		case <-process.done:
			unexpected = append(unexpected, process)
		default:
			running = append(running, process)
		}
	}
	s.mu.Unlock()

	// Every command watches this same context, so all children receive SIGTERM
	// before this function waits for any one of them.
	s.cancelChildren()

waitForProcesses:
	for _, process := range running {
		select {
		case <-process.done:
		case <-ctx.Done():
			break waitForProcesses
		}
	}

	errs := make([]error, 0, len(unexpected)+len(running))
	for _, process := range unexpected {
		errs = append(errs, process.unexpectedExit())
	}

	// The caller's deadline is a bound on graceful shutdown, not permission to
	// abandon an owned child. Kill every survivor, then let the sole Wait owner
	// publish its result. The process-level watchdog remains the final bound if
	// the operating system cannot reap a killed process.
	if shutdownErr := ctx.Err(); shutdownErr != nil {
		for _, process := range running {
			select {
			case <-process.done:
				continue
			default:
			}

			errs = append(errs, fmt.Errorf(
				"%s exceeded shutdown deadline: %w",
				process.name,
				shutdownErr,
			))
			if err := process.process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				errs = append(errs, fmt.Errorf(
					"force %s shutdown: %w",
					process.name,
					err,
				))
			}
		}
	}

	for _, process := range running {
		<-process.done
		errs = append(errs, process.shutdownError())
	}

	return errors.Join(errs...)
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
func waitForTCPReady(parent context.Context, addr string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	dialer := &net.Dialer{
		Timeout: 100 * time.Millisecond,
	}

	attempts := 0
	startTime := time.Now()

	for {
		attempts++
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			elapsed := time.Since(startTime)
			log.Info("TCP listener ready",
				zap.String("addr", addr),
				zap.Int("attempts", attempts),
				zap.Duration("elapsed", elapsed))
			return nil
		}

		if attempts%5 == 0 {
			log.Debug("waiting for TCP listener",
				zap.String("addr", addr),
				zap.Int("attempts", attempts),
				zap.Duration("elapsed", time.Since(startTime)))
		}

		select {
		case <-ctx.Done():
			waitErr := ctx.Err()
			if isRootInterruption(parent, waitErr) {
				return parent.Err()
			}
			return fmt.Errorf("TCP listener %s not ready (limit: %v, attempts: %d): %w",
				addr, timeout, attempts, waitErr)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// waitForTCPOrExit polls the address until TCP connects, process exits, or timeout.
// Returns nil if TCP is ready.
// Returns process exit error if process dies.
// Returns timeout error if deadline reached.
func waitForTCPOrExit(
	parent context.Context,
	addr string,
	timeout time.Duration,
	process *infraProcess,
) error {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	dialer := &net.Dialer{
		Timeout: 100 * time.Millisecond,
	}

	attempts := 0
	startTime := time.Now()

	for {
		attempts++

		// Check if process died
		select {
		case <-process.done:
			return process.unexpectedExit()
		default:
		}

		// Try TCP connect
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			log.Info("TCP listener ready",
				zap.String("addr", addr),
				zap.Int("attempts", attempts),
				zap.Duration("elapsed", time.Since(startTime)))
			return nil
		}

		// Check for timeout
		select {
		case <-ctx.Done():
			waitErr := ctx.Err()
			if isRootInterruption(parent, waitErr) {
				return parent.Err()
			}
			return fmt.Errorf("TCP listener %s not ready (limit: %v, attempts: %d): %w",
				addr, timeout, attempts, waitErr)
		default:
		}

		if attempts%5 == 0 {
			log.Debug("waiting for TCP listener",
				zap.String("addr", addr),
				zap.Int("attempts", attempts),
				zap.Duration("elapsed", time.Since(startTime)))
		}

		// Wait before retry, watching for process exit
		select {
		case <-process.done:
			return process.unexpectedExit()
		case <-ctx.Done():
			waitErr := ctx.Err()
			if isRootInterruption(parent, waitErr) {
				return parent.Err()
			}
			return fmt.Errorf("TCP listener %s not ready (limit: %v, attempts: %d): %w",
				addr, timeout, attempts, waitErr)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

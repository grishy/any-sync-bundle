//go:build !windows

package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestInfraProcessHelper runs as a child test binary. The ready file is
// written only after signal handling is installed, which makes the parent
// tests independent of scheduler timing.
//
//nolint:gocognit // One process fixture keeps every signal mode in the same executable boundary.
func TestInfraProcessHelper(t *testing.T) {
	var args []string
	for idx, arg := range os.Args {
		if arg == "--" {
			args = os.Args[idx+1:]
			break
		}
	}
	if len(args) == 0 {
		return
	}

	mode := args[0]
	if mode == "exit" {
		return
	}
	if len(args) < 2 {
		t.Fatal("helper requires a ready-file path")
	}

	readyPath := args[1]
	if mode == "stubborn" {
		signal.Ignore(syscall.SIGTERM)
		if err := os.WriteFile(readyPath, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		select {}
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	t.Cleanup(func() { signal.Stop(signals) })
	if err := os.WriteFile(readyPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	<-signals
	if mode == "nonzero" {
		signal.Stop(signals)
		os.Exit(23)
	}
	if mode != "graceful" {
		t.Fatalf("unknown helper mode %q", mode)
	}

	if len(args) >= 3 {
		signalPath := args[2]
		if signalPath != "" {
			if err := os.WriteFile(signalPath, nil, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(args) >= 4 {
		peerSignalPath := args[3]
		if peerSignalPath != "" {
			for {
				if _, err := os.Stat(peerSignalPath); err == nil {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
		}
	}
}

func newInfraTestSuite(
	t *testing.T,
	waitDelay time.Duration,
) (*infraSuite, context.CancelFunc) {
	t.Helper()
	rootCtx, cancelRoot := context.WithCancel(t.Context())
	t.Cleanup(cancelRoot)
	return newInfraSuite(rootCtx, cancelRoot, waitDelay), cancelRoot
}

func startInfraTestProcess(
	t *testing.T,
	suite *infraSuite,
	name string,
	mode string,
	readyPath string,
	coordinationPaths ...string,
) *infraProcess {
	t.Helper()
	// Race-instrumented test binaries otherwise wait one second after printing
	// PASS. That artificial exit delay would exercise WaitDelay instead of the
	// helper's signal behavior when these tests use short deadlines.
	t.Setenv("GORACE", strings.TrimSpace(os.Getenv("GORACE")+" atexit_sleep_ms=0"))
	args := []string{"-test.run=^TestInfraProcessHelper$", "--", mode, readyPath}
	args = append(args, coordinationPaths...)
	process, err := suite.start(name, os.Args[0], args...)
	if err != nil {
		t.Fatalf("start %s helper: %v", name, err)
	}
	t.Cleanup(func() {
		select {
		case <-process.done:
			return
		default:
		}

		// A failed assertion can bypass infraSuite.stop, so the fixture remains
		// the final owner responsible for killing and reaping its child.
		_ = process.process.Kill()
		<-process.done
	})
	if readyPath != "" {
		waitForInfraTestFile(t, readyPath)
	}
	return process
}

func waitForInfraTestFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for helper file %s", path)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Test cleanup is the final process owner when an assertion aborts before
// infraSuite.stop can run.
func TestStartInfraTestProcessCleansUpAbandonedProcess(t *testing.T) {
	var process *infraProcess
	t.Run("owner", func(t *testing.T) {
		suite, _ := newInfraTestSuite(t, 50*time.Millisecond)
		readyPath := filepath.Join(t.TempDir(), "ready")
		process = startInfraTestProcess(t, suite, "mongo", "stubborn", readyPath)
	})

	select {
	case <-process.done:
	default:
		_ = process.process.Kill()
		<-process.done
		t.Fatal("test cleanup returned before killing and reaping the child")
	}
}

// Start and running supervision.
func TestInfraSuiteDoesNotStartAfterRootCancellation(t *testing.T) {
	suite, cancelRoot := newInfraTestSuite(t, time.Second)
	cancelRoot()

	_, err := suite.start("redis", os.Args[0], "-test.run=^TestInfraProcessHelper$")
	if !isRootInterruption(suite.rootCtx, err) {
		t.Fatalf("expected exact root cancellation, got %v", err)
	}
}

// A child waiter both broadcasts cancellation and retains its concrete error.
// Context owns lifetime; the returned shutdown error owns the command outcome.
func TestInfraSuiteCancelsRootAndReportsUnexpectedExit(t *testing.T) {
	suite, _ := newInfraTestSuite(t, time.Second)
	startInfraTestProcess(t, suite, "redis", "exit", "")
	select {
	case <-suite.rootCtx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("unexpected exit did not cancel the root context")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelShutdown()
	err := suite.stop(shutdownCtx)
	if err == nil || !strings.Contains(err.Error(), "redis exited unexpectedly") {
		t.Fatalf("unexpected child result was not returned: %v", err)
	}
}

// Root cancellation begins service shutdown, not child shutdown. A dependency
// that exits in that interval is still unexpected and must make the final
// result fail when ownership is handed to stop.
func TestInfraSuiteStopReportsProcessThatExitedBeforeHandoff(t *testing.T) {
	suite, cancelRoot := newInfraTestSuite(t, time.Second)
	process := startInfraTestProcess(t, suite, "redis", "exit", "")

	select {
	case <-process.done:
	case <-time.After(5 * time.Second):
		t.Fatal("process did not exit")
	}
	cancelRoot()

	shutdownCtx, cancelShutdown := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelShutdown()
	err := suite.stop(shutdownCtx)
	if err == nil {
		t.Fatal("expected pre-handoff exit to fail shutdown")
	}
	if !strings.Contains(err.Error(), "redis exited unexpectedly") {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
}

// Graceful shutdown.

// Root cancellation hands ownership to stop before child cancellation, so a
// graceful SIGTERM exit is expected and Wait must complete before stop returns.
func TestInfraSuiteStopGracefullyReapsProcess(t *testing.T) {
	suite, cancelRoot := newInfraTestSuite(t, time.Second)
	readyPath := filepath.Join(t.TempDir(), "ready")
	process := startInfraTestProcess(t, suite, "mongo", "graceful", readyPath)
	cancelRoot()

	shutdownCtx, cancelShutdown := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelShutdown()
	if err := suite.stop(shutdownCtx); err != nil {
		t.Fatalf("graceful stop failed: %v", err)
	}

	select {
	case <-process.done:
	default:
		t.Fatal("stop returned before Wait reaped the process")
	}
}

// Both children wait for proof that the other received SIGTERM. A supervisor
// that signals one child and immediately waits for it deadlocks this test and
// is forced to kill the first child.
func TestInfraSuiteStopSignalsEveryProcessBeforeWaiting(t *testing.T) {
	suite, cancelRoot := newInfraTestSuite(t, 200*time.Millisecond)
	dir := t.TempDir()
	mongoReady := filepath.Join(dir, "mongo-ready")
	redisReady := filepath.Join(dir, "redis-ready")
	mongoSignal := filepath.Join(dir, "mongo-signal")
	redisSignal := filepath.Join(dir, "redis-signal")

	startInfraTestProcess(t, suite, "mongo", "graceful", mongoReady, mongoSignal, redisSignal)
	startInfraTestProcess(t, suite, "redis", "graceful", redisReady, redisSignal, mongoSignal)
	cancelRoot()

	shutdownCtx, cancelShutdown := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelShutdown()
	if err := suite.stop(shutdownCtx); err != nil {
		t.Fatalf("children were not signaled together: %v", err)
	}
	waitForInfraTestFile(t, mongoSignal)
	waitForInfraTestFile(t, redisSignal)
}

func TestInfraSuiteStopReportsNonZeroGracefulExit(t *testing.T) {
	suite, cancelRoot := newInfraTestSuite(t, time.Second)
	readyPath := filepath.Join(t.TempDir(), "ready")
	startInfraTestProcess(t, suite, "redis", "nonzero", readyPath)
	cancelRoot()

	shutdownCtx, cancelShutdown := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelShutdown()
	err := suite.stop(shutdownCtx)
	if err == nil {
		t.Fatal("expected non-zero shutdown exit to fail")
	}
	if !strings.Contains(err.Error(), "redis") {
		t.Fatalf("shutdown error does not identify Redis: %v", err)
	}
	if _, ok := errors.AsType[*exec.ExitError](err); !ok {
		t.Fatalf("shutdown error lost the process result: %v", err)
	}
}

// Forced shutdown.

func TestInfraSuiteStopKillsAndReapsSurvivor(t *testing.T) {
	suite, cancelRoot := newInfraTestSuite(t, 50*time.Millisecond)
	readyPath := filepath.Join(t.TempDir(), "ready")
	process := startInfraTestProcess(t, suite, "mongo", "stubborn", readyPath)
	cancelRoot()

	shutdownCtx, cancelShutdown := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelShutdown()
	err := suite.stop(shutdownCtx)
	if err == nil {
		t.Fatal("expected forced MongoDB shutdown to fail")
	}

	select {
	case <-process.done:
	default:
		t.Fatal("forced process was not reaped")
	}

	exitErr, ok := errors.AsType[*exec.ExitError](process.waitErr)
	if !ok {
		t.Fatalf("expected process exit error, got %v", process.waitErr)
	}
	waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !waitStatus.Signaled() || waitStatus.Signal() != syscall.SIGKILL {
		t.Fatalf("expected SIGKILL, got %v", exitErr)
	}
}

// A caller deadline may be shorter than Cmd.WaitDelay. Stop must force the
// child and wait for its sole waiter instead of returning an owned zombie.
func TestInfraSuiteStopReapsProcessAfterCallerDeadline(t *testing.T) {
	suite, cancelRoot := newInfraTestSuite(t, 200*time.Millisecond)
	readyPath := filepath.Join(t.TempDir(), "ready")
	process := startInfraTestProcess(t, suite, "mongo", "stubborn", readyPath)
	cancelRoot()

	shutdownCtx, cancelShutdown := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancelShutdown()
	err := suite.stop(shutdownCtx)

	select {
	case <-process.done:
	default:
		t.Fatal("stop returned before reaping the process")
	}
	if err == nil {
		t.Fatal("expected forced shutdown to fail")
	}
}

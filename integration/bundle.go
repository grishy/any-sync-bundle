//go:build integration

package integration

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	bundleReadyEvent            = "bundle_ready"
	bundleShutdownCompleteEvent = "bundle_shutdown_complete"
	filenodeStorageBackendS3    = "filenode_storage_backend_s3"
)

// BundleProcess manages the any-sync-bundle process.
type BundleProcess struct {
	cmd         *exec.Cmd
	output      *strings.Builder
	mu          sync.Mutex
	waitDone    chan struct{}
	waitErr     error
	tmpDir      string
	projectRoot string
}

// BundleConfig configures the bundle process.
type BundleConfig struct {
	MongoURI    string
	RedisURI    string
	S3Bucket    string
	S3Endpoint  string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
}

// StartBundle builds and starts the bundle process.
func StartBundle(ctx context.Context, cfg BundleConfig) (*BundleProcess, error) {
	// Get the project root (go up from integration directory)
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// If we're in the integration directory, go up one level
	projectRoot := wd
	if strings.HasSuffix(wd, "/integration") {
		projectRoot = strings.TrimSuffix(wd, "/integration")
	}

	// Build binary from project root
	binaryPath := filepath.Join(projectRoot, "test-bundle")
	build := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".")
	build.Dir = projectRoot
	buildOutput, buildErr := build.CombinedOutput()
	if buildErr != nil {
		return nil, fmt.Errorf("build failed: %s: %w", buildOutput, buildErr)
	}

	tmpDir, err := os.MkdirTemp("", "bundle-test-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	args := []string{
		"start-bundle",
		"--bundle-config", filepath.Join(tmpDir, "bundle.yml"),
		"--client-config", filepath.Join(tmpDir, "client.yml"),
		"--initial-storage", filepath.Join(tmpDir, "storage"),
		"--initial-mongo-uri", cfg.MongoURI,
		"--initial-redis-uri", cfg.RedisURI,
		"--initial-external-addrs", "127.0.0.1",
	}

	// Add S3 config if provided
	if cfg.S3Bucket != "" {
		args = append(args,
			"--initial-s3-bucket", cfg.S3Bucket,
			"--initial-s3-endpoint", cfg.S3Endpoint,
			"--initial-s3-force-path-style",
		)
		if cfg.S3Region != "" {
			args = append(args, "--initial-s3-region", cfg.S3Region)
		}
	}

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Dir = projectRoot

	// Set S3 credentials
	if cfg.S3AccessKey != "" {
		cmd.Env = append(os.Environ(),
			"AWS_ACCESS_KEY_ID="+cfg.S3AccessKey,
			"AWS_SECRET_ACCESS_KEY="+cfg.S3SecretKey,
		)
	}

	output := &strings.Builder{}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	bp := &BundleProcess{
		cmd:         cmd,
		output:      output,
		waitDone:    make(chan struct{}),
		tmpDir:      tmpDir,
		projectRoot: projectRoot,
	}

	if startErr := cmd.Start(); startErr != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to start bundle: %w", startErr)
	}

	// Capture output in background
	go bp.captureOutput(stdout)
	go bp.captureOutput(stderr)
	go bp.waitForExit()

	return bp, nil
}

func (bp *BundleProcess) captureOutput(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		bp.mu.Lock()
		bp.output.WriteString(scanner.Text() + "\n")
		bp.mu.Unlock()
	}
	if err := scanner.Err(); err != nil {
		bp.mu.Lock()
		bp.output.WriteString(fmt.Sprintf("output capture error: %v\n", err))
		bp.mu.Unlock()
	}
}

func (bp *BundleProcess) waitForExit() {
	waitErr := bp.cmd.Wait()
	bp.mu.Lock()
	bp.waitErr = waitErr
	bp.mu.Unlock()
	close(bp.waitDone)
}

// Output returns captured stdout/stderr.
func (bp *BundleProcess) Output() string {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.output.String()
}

// WaitReady waits for the machine-readable ready event.
func (bp *BundleProcess) WaitReady(timeout time.Duration) error {
	return bp.waitForOutputMarker(timeout, bundleReadyEvent, "ready")
}

// WaitForS3Backend verifies S3 storage backend was selected.
func (bp *BundleProcess) WaitForS3Backend(timeout time.Duration) error {
	return bp.waitForOutputMarker(timeout, filenodeStorageBackendS3, "S3 backend")
}

// VerifyPort checks if TCP port is listening.
func (bp *BundleProcess) VerifyPort(port string) error {
	dialer := &net.Dialer{Timeout: time.Second}
	conn, err := dialer.Dial("tcp", net.JoinHostPort("localhost", port))
	if err != nil {
		return fmt.Errorf("port %s not listening: %w", port, err)
	}
	_ = conn.Close()
	return nil
}

// Stop gracefully stops the bundle process.
func (bp *BundleProcess) Stop() error {
	if bp.cmd.Process == nil {
		return nil
	}

	if bp.hasExited() {
		return bp.shutdownResult()
	}

	_ = bp.cmd.Process.Signal(os.Interrupt)

	select {
	case <-bp.waitDone:
		return bp.shutdownResult()
	case <-time.After(30 * time.Second):
		_ = bp.cmd.Process.Kill()
		return errors.New("timeout during shutdown, killed process")
	}
}

// Cleanup removes temporary files and binary.
func (bp *BundleProcess) Cleanup() {
	_ = os.RemoveAll(bp.tmpDir)
	_ = os.Remove(filepath.Join(bp.projectRoot, "test-bundle"))
}

func (bp *BundleProcess) hasExited() bool {
	select {
	case <-bp.waitDone:
		return true
	default:
		return false
	}
}

func (bp *BundleProcess) waitForOutputMarker(
	timeout time.Duration,
	marker string,
	description string,
) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(bp.Output(), marker) {
			return nil
		}
		if bp.hasExited() {
			return bp.exitedBeforeMarkerError(description)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if bp.hasExited() {
		return bp.exitedBeforeMarkerError(description)
	}
	return fmt.Errorf("timeout waiting for %s marker %q\n%s",
		description, marker, bp.Output())
}

func (bp *BundleProcess) shutdownResult() error {
	output := bp.Output()
	if !strings.Contains(output, bundleShutdownCompleteEvent) {
		if err := bp.exitErr(); err != nil {
			return fmt.Errorf("bundle exited without clean shutdown: %w\n%s", err, output)
		}
		return fmt.Errorf("bundle exited without shutdown marker %q\n%s",
			bundleShutdownCompleteEvent, output)
	}
	if err := bp.exitErr(); err != nil {
		return fmt.Errorf("bundle exited during shutdown: %w\n%s", err, output)
	}
	return nil
}

func (bp *BundleProcess) exitedBeforeMarkerError(description string) error {
	output := bp.Output()
	if err := bp.exitErr(); err != nil {
		return fmt.Errorf("bundle process exited before %s: %w\n%s",
			description, err, output)
	}
	return fmt.Errorf("bundle process exited before %s\n%s", description, output)
}

func (bp *BundleProcess) exitErr() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.waitErr
}

//go:build !windows

package doctor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestServerStreamsDoctorOutput(t *testing.T) {
	socketPath := shortSocketPath(t)
	generatedAt := time.Date(2026, 5, 22, 14, 33, 10, 0, time.UTC)
	runner := runnerFunc(func(_ context.Context, out io.Writer) (*Report, error) {
		_, _ = io.WriteString(out, "[1/7] Config\n  status: ok\n")
		return &Report{
			GeneratedAt: generatedAt,
			Verdict:     VerdictProblemsFound,
			ReportPath:  "/data/doctor/doctor_2026-05-22T14-33-10Z.json",
		}, nil
	})
	server := NewServer(ServerConfig{
		SocketPath: socketPath,
		Runner:     runner,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer server.Close(context.Background())

	var out bytes.Buffer
	err := RunClient(ctx, socketPath, &out)
	if err != nil {
		t.Fatalf("RunClient() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Connecting to running bundle") {
		t.Fatalf("client output = %q, want client connection text", got)
	}
	if !strings.Contains(got, "  socket: "+socketPath) {
		t.Fatalf("client output = %q, want socket path", got)
	}
	if !strings.Contains(got, "  status: connected") {
		t.Fatalf("client output = %q, want connected status", got)
	}
	if !strings.Contains(got, "[1/7] Config") {
		t.Fatalf("client output = %q, want streamed phase", got)
	}
}

func TestServerAllowsOnlyOneScanAtATime(t *testing.T) {
	socketPath := shortSocketPath(t)
	started := make(chan struct{})
	release := make(chan struct{})
	var closeStarted sync.Once
	runner := runnerFunc(func(ctx context.Context, _ io.Writer) (*Report, error) {
		closeStarted.Do(func() { close(started) })
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release:
			return &Report{Verdict: VerdictHealthy}, nil
		}
	})
	server := NewServer(ServerConfig{
		SocketPath: socketPath,
		Runner:     runner,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer server.Close(context.Background())

	firstDone := make(chan error, 1)
	go func() {
		var out bytes.Buffer
		err := RunClient(ctx, socketPath, &out)
		firstDone <- err
	}()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("first doctor run did not start")
	}

	client := unixHTTPClient(socketPath)
	response, err := client.Post("http://doctor/doctor/run", "text/plain", nil)
	if err != nil {
		t.Fatalf("second request error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		t.Fatalf("second status = %d, want %d", response.StatusCode, http.StatusConflict)
	}

	close(release)
	if firstErr := <-firstDone; firstErr != nil {
		t.Fatalf("first RunClient() error = %v", firstErr)
	}
}

func TestRunClientReturnsErrorWhenSocketIsMissing(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "missing.sock")

	var out bytes.Buffer
	err := RunClient(context.Background(), socketPath, &out)

	if err == nil {
		t.Fatal("RunClient() error = nil, want error")
	}
}

func TestRunClientReturnsNilWhenStartedScanFailsInStream(t *testing.T) {
	socketPath := shortSocketPath(t)
	runner := runnerFunc(func(_ context.Context, out io.Writer) (*Report, error) {
		_, _ = io.WriteString(out, "[1/7] Config\n")
		return nil, errors.New("boom")
	})
	server := NewServer(ServerConfig{
		SocketPath: socketPath,
		Runner:     runner,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer server.Close(context.Background())

	var out bytes.Buffer
	err := RunClient(ctx, socketPath, &out)
	if err != nil {
		t.Fatalf("RunClient() error = %v, want nil after stream started", err)
	}
	if !strings.Contains(out.String(), "Doctor failed: boom") {
		t.Fatalf("client output does not contain streamed failure:\n%s", out.String())
	}
}

func TestServerStartRefusesToRemoveRegularFileSocketPath(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "bundle.sock")
	if err := os.WriteFile(socketPath, []byte("not a socket"), 0o600); err != nil {
		t.Fatalf("write regular file: %v", err)
	}
	server := NewServer(ServerConfig{
		SocketPath: socketPath,
		Runner: runnerFunc(func(_ context.Context, _ io.Writer) (*Report, error) {
			return &Report{Verdict: VerdictHealthy}, nil
		}),
	})

	err := server.Start(context.Background())
	if err == nil {
		t.Fatal("Start() error = nil, want non-socket refusal")
	}
	if !strings.Contains(err.Error(), "refusing to remove non-socket") {
		t.Fatalf("Start() error = %v", err)
	}
	raw, readErr := os.ReadFile(socketPath)
	if readErr != nil {
		t.Fatalf("regular file was removed: %v", readErr)
	}
	if string(raw) != "not a socket" {
		t.Fatalf("regular file content = %q", string(raw))
	}
}

func TestServerCloseIsIdempotent(t *testing.T) {
	socketPath := shortSocketPath(t)
	server := NewServer(ServerConfig{
		SocketPath: socketPath,
		Runner: runnerFunc(func(_ context.Context, _ io.Writer) (*Report, error) {
			return &Report{Verdict: VerdictHealthy}, nil
		}),
	})
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := server.Close(context.Background()); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := server.Close(context.Background()); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket still exists after Close(): %v", err)
	}
}

func shortSocketPath(t *testing.T) string {
	t.Helper()

	// Keep the socket path short enough for Unix socket limits on macOS.
	//nolint:usetesting // t.TempDir can be too long for Unix socket paths on macOS.
	dir, err := os.MkdirTemp("/tmp", "doctor-test-")
	if err != nil {
		t.Fatalf("create short temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return filepath.Join(dir, "bundle.sock")
}

type runnerFunc func(ctx context.Context, out io.Writer) (*Report, error)

func (f runnerFunc) RunDoctor(ctx context.Context, out io.Writer) (*Report, error) {
	return f(ctx, out)
}

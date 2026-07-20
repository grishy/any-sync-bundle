package cmd

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestIsIllegalInstruction(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "mixed-case illegal instruction",
			err:  errors.New("Signal: Illegal Instruction"),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("connection refused"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIllegalInstruction(tt.err)
			if got != tt.want {
				t.Errorf("isIllegalInstruction(%v) = %v, want %v",
					tt.err, got, tt.want)
			}
		})
	}
}

func TestWaitForTCPOrExit_ProcessDies(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		process := &infraProcess{
			done: make(chan struct{}),
		}

		expectedErr := errors.New("signal: illegal instruction")

		go func() {
			time.Sleep(10 * time.Millisecond)
			process.name = "mongo"
			process.waitErr = expectedErr
			close(process.done)
		}()

		err := waitForTCPOrExit(
			context.Background(),
			"127.0.0.1:59999",
			5*time.Second,
			process,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestWaitForTCPOrExit_TCPReady(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	process := &infraProcess{
		done: make(chan struct{}),
	}

	err = waitForTCPOrExit(
		context.Background(),
		listener.Addr().String(),
		5*time.Second,
		process,
	)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestWaitForTCPOrExit_Timeout(t *testing.T) {
	process := &infraProcess{
		done: make(chan struct{}),
	}

	err := waitForTCPOrExit(
		context.Background(),
		"127.0.0.1:59999",
		200*time.Millisecond,
		process,
	)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// A process that exits successfully before opening its listener is still a
// failed dependency. Treating a nil Wait result as readiness would leave the
// bundle running without the database it owns.
func TestWaitForTCPOrExitReportsCleanProcessExit(t *testing.T) {
	process := &infraProcess{
		name: "redis",
		done: make(chan struct{}),
	}
	close(process.done)

	err := waitForTCPOrExit(
		context.Background(),
		"127.0.0.1:59999",
		time.Minute,
		process,
	)
	if err == nil {
		t.Fatal("expected a clean early exit to be reported")
	}
	if !strings.Contains(err.Error(), "redis exited unexpectedly") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Readiness is part of the root startup scope. Cancellation must interrupt the
// wait immediately rather than leave shutdown blocked behind its own timeout.
func TestWaitForTCPOrExitStopsOnParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	process := &infraProcess{done: make(chan struct{})}
	err := waitForTCPOrExit(ctx, "127.0.0.1:59999", time.Minute, process)
	if !isRootInterruption(ctx, err) {
		t.Fatalf("expected exact parent cancellation, got %v", err)
	}
}

func TestWaitForTCPReadyStopsOnParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitForTCPReady(ctx, "127.0.0.1:59999", time.Minute)
	if !isRootInterruption(ctx, err) {
		t.Fatalf("expected exact parent cancellation, got %v", err)
	}
}

func TestStartAllInOneInfraStopsBeforeAcquiringResources(t *testing.T) {
	ctx, cancelRoot := context.WithCancel(t.Context())
	cancelRoot()
	infra := newInfraSuite(ctx, cancelRoot, time.Second)

	err := startAllInOneInfra(ctx, infra)
	if !isRootInterruption(ctx, err) {
		t.Fatalf("expected exact root cancellation, got %v", err)
	}
}

func TestMongoAVXError(t *testing.T) {
	cause := errors.New("signal: illegal instruction")
	err := &MongoAVXError{Cause: cause}

	want := "mongodb requires AVX CPU support: signal: illegal instruction"
	if got := err.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("MongoAVXError lost its cause: %v", err)
	}
}

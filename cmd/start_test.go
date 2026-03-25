package cmd

import (
	"context"
	"errors"
	"net"
	"testing"
	"testing/synctest"
	"time"

	"github.com/anyproto/any-sync/app"
)

func TestIsIllegalInstruction(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "sigill lowercase",
			err:  errors.New("signal: illegal instruction (core dumped)"),
			want: true,
		},
		{
			name: "sigill uppercase",
			err:  errors.New("SIGNAL: ILLEGAL INSTRUCTION"),
			want: true,
		},
		{
			name: "sigill mixed case",
			err:  errors.New("Signal: Illegal Instruction"),
			want: true,
		},
		{
			name: "connection refused",
			err:  errors.New("connection refused"),
			want: false,
		},
		{
			name: "exit status 1",
			err:  errors.New("exit status 1"),
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
		// Create a mock process that dies
		proc := &infraProcess{
			done: make(chan struct{}),
		}

		expectedErr := errors.New("signal: illegal instruction")

		// Simulate process dying after a short time
		go func() {
			time.Sleep(10 * time.Millisecond)
			proc.exitErr = expectedErr
			close(proc.done)
		}()

		// Use a non-existent address so TCP never connects
		// With synctest, time advances automatically when blocked
		err := waitForTCPOrExit("127.0.0.1:59999", 5*time.Second, proc)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestWaitForTCPOrExit_TCPReady(t *testing.T) {
	// Start a TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	// Create a mock process that stays alive
	proc := &infraProcess{
		done: make(chan struct{}),
	}

	// Wait for TCP (should succeed quickly)
	err = waitForTCPOrExit(listener.Addr().String(), 5*time.Second, proc)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestWaitForTCPOrExit_Timeout(t *testing.T) {
	// Create a mock process that stays alive
	proc := &infraProcess{
		done: make(chan struct{}),
	}

	// Use a non-existent address and short timeout
	err := waitForTCPOrExit("127.0.0.1:59999", 200*time.Millisecond, proc)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestMongoAVXError(t *testing.T) {
	cause := errors.New("signal: illegal instruction")
	err := &MongoAVXError{Cause: cause}

	t.Run("error message", func(t *testing.T) {
		want := "mongodb requires AVX CPU support: signal: illegal instruction"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		got := errors.Unwrap(err)
		if !errors.Is(got, cause) {
			t.Errorf("Unwrap() = %v, want %v", got, cause)
		}
	})

	t.Run("errors.As", func(t *testing.T) {
		var target *MongoAVXError
		if !errors.As(err, &target) {
			t.Error("errors.As() should match MongoAVXError")
		}
	})

	t.Run("errors.Is with wrapped", func(t *testing.T) {
		wrapped := errors.New("signal: illegal instruction")
		avxErr := &MongoAVXError{Cause: wrapped}
		if !errors.Is(avxErr, wrapped) {
			t.Error("errors.Is() should find wrapped cause")
		}
	})
}

type lifecycleTestRunnable struct {
	name    string
	events  *[]string
	initErr error
	runErr  error
}

func (r *lifecycleTestRunnable) Init(*app.App) error {
	*r.events = append(*r.events, "init:"+r.name)
	return r.initErr
}

func (r *lifecycleTestRunnable) Name() string {
	return r.name
}

func (r *lifecycleTestRunnable) Run(context.Context) error {
	*r.events = append(*r.events, "run:"+r.name)
	return r.runErr
}

func (r *lifecycleTestRunnable) Close(context.Context) error {
	*r.events = append(*r.events, "close:"+r.name)
	return nil
}

func TestInitOneApp_PartialFailureClosesCurrentService(t *testing.T) {
	events := []string{}
	first := &lifecycleTestRunnable{name: "first", events: &events}
	second := &lifecycleTestRunnable{
		name:    "second",
		events:  &events,
		initErr: errors.New("boom"),
	}
	third := &lifecycleTestRunnable{name: "third", events: &events}

	testApp := new(app.App).
		Register(first).
		Register(second).
		Register(third)

	err := initOneApp(node{name: "test", app: testApp})
	if err == nil {
		t.Fatal("expected init error, got nil")
	}

	want := []string{
		"init:first",
		"init:second",
		"close:second",
		"close:first",
	}
	if len(events) != len(want) {
		t.Fatalf("unexpected event count: got %v want %v", events, want)
	}
	for idx := range want {
		if events[idx] != want[idx] {
			t.Fatalf("unexpected events: got %v want %v", events, want)
		}
	}
}

func TestStartServices_RunFailureClosesCurrentAndPreviousServices(t *testing.T) {
	events := []string{}
	firstService := &lifecycleTestRunnable{name: "one", events: &events}
	secondServiceFirst := &lifecycleTestRunnable{name: "two-a", events: &events}
	secondServiceSecond := &lifecycleTestRunnable{
		name:   "two-b",
		events: &events,
		runErr: errors.New("boom"),
	}

	appOne := new(app.App).Register(firstService)
	appTwo := new(app.App).
		Register(secondServiceFirst).
		Register(secondServiceSecond)

	err := startServices(context.Background(), []node{
		{name: "svc-one", app: appOne},
		{name: "svc-two", app: appTwo},
	}, nil)
	if err == nil {
		t.Fatal("expected run error, got nil")
	}

	want := []string{
		"init:one",
		"init:two-a",
		"init:two-b",
		"run:one",
		"run:two-a",
		"run:two-b",
		"close:two-b",
		"close:two-a",
		"close:one",
	}
	if len(events) != len(want) {
		t.Fatalf("unexpected event count: got %v want %v", events, want)
	}
	for idx := range want {
		if events[idx] != want[idx] {
			t.Fatalf("unexpected events: got %v want %v", events, want)
		}
	}
}

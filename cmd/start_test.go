package cmd

import (
	"errors"
	"testing"
	"time"
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

func TestInfraProcessWaitWarmup(t *testing.T) {
	tests := []struct {
		name      string
		exitAfter time.Duration // 0 means don't exit during test
		exitErr   error
		timeout   time.Duration
		wantErr   bool
	}{
		{
			name:    "process survives warmup",
			timeout: 50 * time.Millisecond,
			wantErr: false,
		},
		{
			name:      "process dies during warmup",
			exitAfter: 10 * time.Millisecond,
			exitErr:   errors.New("signal: illegal instruction"),
			timeout:   100 * time.Millisecond,
			wantErr:   true,
		},
		{
			name:      "process dies after warmup timeout",
			exitAfter: 100 * time.Millisecond,
			exitErr:   errors.New("some error"),
			timeout:   20 * time.Millisecond,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &infraProcess{
				done: make(chan struct{}),
			}

			if tt.exitAfter > 0 {
				go func() {
					time.Sleep(tt.exitAfter)
					p.exitErr = tt.exitErr
					close(p.done)
				}()
			}

			err := p.waitWarmup(tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("waitWarmup() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, tt.exitErr) {
				t.Errorf("waitWarmup() error = %v, want %v", err, tt.exitErr)
			}
		})
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

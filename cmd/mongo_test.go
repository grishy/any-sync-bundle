package cmd

import (
	"context"
	"errors"
	"net"
	"testing"
	"testing/synctest"
	"time"

	"go.mongodb.org/mongo-driver/mongo/options"
)

const mongoURIWithInvalidOptions = "mongodb://localhost/?directConnection=invalid"

// The lifecycle distinguishes an exact root cancellation from operational
// failures. A driver error produced after that cancellation must not turn an
// operator-requested stop into a non-zero exit.
func TestTryInitReplicaSetReturnsExactCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})
	if err = listener.(*net.TCPListener).SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set listener deadline: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)
	clientOpts := options.Client().
		ApplyURI("mongodb://" + listener.Addr().String() + "/").
		SetDirect(true)
	go func() {
		result <- tryInitReplicaSet(ctx, clientOpts, defaultMongoReplica)
	}()

	connection, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept MongoDB connection: %v", err)
	}
	cancel()
	_ = connection.Close()

	select {
	case err = <-result:
	case <-time.After(10 * time.Second):
		t.Fatal("replica-set attempt ignored cancellation")
	}
	if err != context.Canceled { //nolint:errorlint // The process boundary requires exact cancellation identity.
		t.Fatalf("expected exact cancellation, got %v", err)
	}
}

// Cancellation owns the replica-set retry loop as well as each MongoDB call.
// A signal during backoff must not wait for the next retry delay to expire.
func TestInitReplicaSetActionCancellationInterruptsRetryDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const cancelAfter = 100 * time.Millisecond
		ctx, cancel := context.WithCancel(t.Context())
		go func() {
			time.Sleep(cancelAfter)
			cancel()
		}()

		started := time.Now()
		err := initReplicaSetAction(ctx, defaultMongoReplica, mongoURIWithInvalidOptions)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation, got %v", err)
		}
		if elapsed := time.Since(started); elapsed != cancelAfter {
			t.Fatalf("retry delay ignored cancellation: got %v want %v", elapsed, cancelAfter)
		}
	})
}

// Ten attempts need nine separating delays. A delay after the final failed
// attempt consumes shutdown budget without making another attempt possible.
func TestInitReplicaSetActionDoesNotWaitAfterFinalAttempt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const retryDelayTotal = 142 * time.Second

		started := time.Now()
		err := initReplicaSetAction(
			t.Context(),
			defaultMongoReplica,
			mongoURIWithInvalidOptions,
		)
		if err == nil {
			t.Fatal("expected replica-set initialization to fail")
		}
		if errors.Unwrap(err) == nil {
			t.Fatal("final retry error does not preserve the last attempt failure")
		}
		if elapsed := time.Since(started); elapsed != retryDelayTotal {
			t.Fatalf("unexpected retry delay total: got %v want %v", elapsed, retryDelayTotal)
		}
	})
}

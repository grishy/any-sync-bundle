package cmd

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

const mongoURIWithInvalidOptions = "mongodb://localhost/?directConnection=invalid"

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

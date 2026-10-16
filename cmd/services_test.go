package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/anyproto/any-sync/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grishy/any-sync-bundle/lightnode"
)

type lifecycleTestRunnable struct {
	name     string
	events   *[]string
	onRun    func()
	runErr   error
	closeErr error
	onClose  func(context.Context)
}

func (r *lifecycleTestRunnable) Name() string { return r.name }

func (r *lifecycleTestRunnable) Init(*app.App) error {
	*r.events = append(*r.events, "init:"+r.name)
	return nil
}

func (r *lifecycleTestRunnable) Run(context.Context) error {
	*r.events = append(*r.events, "run:"+r.name)
	if r.onRun != nil {
		r.onRun()
	}
	return r.runErr
}

func (r *lifecycleTestRunnable) Close(ctx context.Context) error {
	*r.events = append(*r.events, "close:"+r.name)
	if r.onClose != nil {
		r.onClose(ctx)
	}
	return r.closeErr
}

func testService(name string, components ...*lifecycleTestRunnable) lightnode.Service {
	a := new(app.App)
	for _, c := range components {
		a.Register(c)
	}
	return lightnode.Service{Name: name, App: a}
}

func testBundle(listener *lifecycleTestRunnable, services ...lightnode.Service) *lightnode.Bundle {
	return &lightnode.Bundle{Services: services, Listeners: []app.ComponentRunnable{listener}}
}

func TestRunServicesOpensListenersLastAndClosesThemFirst(t *testing.T) {
	ctx, stopRoot := context.WithCancelCause(t.Context())
	var events []string
	var closeHasDeadline bool
	var closeCtxErr error
	onClose := func(ctx context.Context) {
		_, closeHasDeadline = ctx.Deadline()
		closeCtxErr = ctx.Err()
	}
	bundle := testBundle(
		&lifecycleTestRunnable{name: "listener", events: &events, onRun: func() { stopRoot(nil) }},
		testService("first", &lifecycleTestRunnable{name: "a", events: &events, onClose: onClose}),
		testService("second", &lifecycleTestRunnable{name: "b", events: &events}),
	)

	require.NoError(t, runServices(ctx, stopRoot, bundle))

	assert.Equal(t, []string{
		"init:a", "run:a", "init:b", "run:b", "run:listener",
		"close:listener", "close:b", "close:a",
	}, events)
	assert.True(t, closeHasDeadline, "shutdown must be bounded")
	assert.NoError(t, closeCtxErr, "shutdown must not inherit root cancellation")
}

func TestRunServicesStartFailureStopsEverythingWithItsCause(t *testing.T) {
	ctx, stopRoot := context.WithCancelCause(t.Context())
	var events []string
	runErr := errors.New("run failed")
	bundle := testBundle(
		&lifecycleTestRunnable{name: "listener", events: &events},
		testService("first", &lifecycleTestRunnable{name: "a", events: &events}),
		testService("second",
			&lifecycleTestRunnable{name: "b", events: &events},
			&lifecycleTestRunnable{name: "c", events: &events, runErr: runErr}),
		testService("third", &lifecycleTestRunnable{name: "d", events: &events}),
	)

	err := runServices(ctx, stopRoot, bundle)

	require.ErrorIs(t, err, runErr)
	require.ErrorContains(t, err, "start second")
	require.ErrorIs(t, context.Cause(ctx), runErr, "the failure must be the root stop cause")
	assert.Equal(t, []string{
		"init:a", "run:a",
		"init:b", "init:c", "run:b", "run:c", "close:c", "close:b",
		"close:listener", "close:a",
	}, events)
}

func TestRunServicesOperatorStopDuringStartupIsCleanUnlessShutdownFails(t *testing.T) {
	tests := []struct {
		name     string
		closeErr error
	}{
		{name: "clean stop"},
		{name: "shutdown failure", closeErr: errors.New("close failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, stopRoot := context.WithCancelCause(t.Context())
			var events []string
			bundle := testBundle(
				&lifecycleTestRunnable{name: "listener", events: &events},
				testService("first", &lifecycleTestRunnable{name: "a", events: &events, closeErr: tt.closeErr}),
				testService("second", &lifecycleTestRunnable{
					name: "b", events: &events, onRun: func() { stopRoot(nil) }, runErr: context.Canceled,
				}),
				testService("third", &lifecycleTestRunnable{name: "c", events: &events}),
			)

			err := runServices(ctx, stopRoot, bundle)

			if tt.closeErr == nil {
				require.NoError(t, err)
			} else {
				// Upstream app.Close reports component errors as text.
				require.ErrorContains(t, err, tt.closeErr.Error())
				require.NotErrorIs(t, err, context.Canceled)
			}
			assert.NotContains(t, events, "init:c", "nothing starts after the stop")
			assert.NotContains(t, events, "run:listener", "listeners never open during startup")
		})
	}
}

// Another failure, such as an embedded process exit, stops the root context
// with its own cause. A startup error it causes is still a failure.
func TestRunServicesStartupErrorAfterFailureStopIsReturned(t *testing.T) {
	ctx, stopRoot := context.WithCancelCause(t.Context())
	var events []string
	bundle := testBundle(
		&lifecycleTestRunnable{name: "listener", events: &events},
		testService("first", &lifecycleTestRunnable{
			name: "a", events: &events,
			onRun:  func() { stopRoot(errors.New("mongo exited unexpectedly")) },
			runErr: context.Canceled,
		}),
	)

	err := runServices(ctx, stopRoot, bundle)

	require.ErrorContains(t, err, "start first")
}

func TestRunServicesReturnsShutdownFailure(t *testing.T) {
	ctx, stopRoot := context.WithCancelCause(t.Context())
	var events []string
	closeErr := errors.New("close failed")
	bundle := testBundle(
		&lifecycleTestRunnable{name: "listener", events: &events, onRun: func() { stopRoot(nil) }},
		testService("first", &lifecycleTestRunnable{name: "a", events: &events, closeErr: closeErr}),
	)

	err := runServices(ctx, stopRoot, bundle)

	require.ErrorContains(t, err, "stop first")
	require.ErrorContains(t, err, closeErr.Error())
}

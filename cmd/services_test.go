package cmd

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/anyproto/any-sync/app"
)

type lifecycleTestRunnable struct {
	name          string
	events        *[]string
	closeContexts *[]context.Context
	onClose       func()
	onInit        func()
	onRun         func()
	initErr       error
	runErr        error
	closeErr      error
}

func (r *lifecycleTestRunnable) Init(*app.App) error {
	*r.events = append(*r.events, "init:"+r.name)
	if r.onInit != nil {
		r.onInit()
	}
	return r.initErr
}

func (r *lifecycleTestRunnable) Name() string {
	return r.name
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
		r.onClose()
	}
	if r.closeContexts != nil {
		*r.closeContexts = append(*r.closeContexts, ctx)
	}
	return r.closeErr
}

func TestStartServicesRootCancellationIsCleanButCleanupFailureIsNot(t *testing.T) {
	tests := []struct {
		name     string
		closeErr error
	}{
		{name: "clean interruption"},
		{name: "cleanup failure", closeErr: errors.New("close failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancelRoot := context.WithCancel(t.Context())
			events := []string{}
			err := startServices(
				ctx,
				cancelRoot,
				[]bundleService{{
					name: "test",
					app: new(app.App).Register(&lifecycleTestRunnable{
						name:     "component",
						events:   &events,
						onRun:    cancelRoot,
						runErr:   context.Canceled,
						closeErr: tt.closeErr,
					}),
				}},
				nil,
			)

			if tt.closeErr == nil {
				if !isRootInterruption(ctx, err) {
					t.Fatalf("clean interruption lost its exact identity: %v", err)
				}
			} else {
				if isRootInterruption(ctx, err) || !errors.Is(err, tt.closeErr) {
					t.Fatalf("cleanup failure was lost: %v", err)
				}
			}
		})
	}
}

func TestStartServicesStopsInitializingAfterRootCancellation(t *testing.T) {
	ctx, cancelRoot := context.WithCancel(t.Context())
	events := []string{}
	first := &lifecycleTestRunnable{
		name:   "first",
		events: &events,
		onInit: cancelRoot,
	}
	second := &lifecycleTestRunnable{
		name:   "second",
		events: &events,
	}

	err := startServices(
		ctx,
		cancelRoot,
		[]bundleService{{
			name: "test",
			app:  new(app.App).Register(first).Register(second),
		}},
		nil,
	)

	if !isRootInterruption(ctx, err) {
		t.Fatalf("root cancellation lost its exact identity: %v", err)
	}
	want := []string{"init:first", "close:first"}
	if !slices.Equal(events, want) {
		t.Fatalf("startup acquired resources after cancellation: got %v want %v", events, want)
	}
}

func TestStartServicesStopsRunningAfterRootCancellation(t *testing.T) {
	ctx, cancelRoot := context.WithCancel(t.Context())
	events := []string{}
	first := &lifecycleTestRunnable{
		name:   "first",
		events: &events,
		onRun:  cancelRoot,
	}
	second := &lifecycleTestRunnable{
		name:   "second",
		events: &events,
	}

	err := startServices(
		ctx,
		cancelRoot,
		[]bundleService{{
			name: "test",
			app:  new(app.App).Register(first).Register(second),
		}},
		nil,
	)

	if !isRootInterruption(ctx, err) {
		t.Fatalf("root cancellation lost its exact identity: %v", err)
	}
	want := []string{
		"init:first",
		"init:second",
		"run:first",
		"close:first",
	}
	if !slices.Equal(events, want) {
		t.Fatalf("startup acquired resources after cancellation: got %v want %v", events, want)
	}
}

func TestStartServicesPreservesIndependentComponentCancellation(t *testing.T) {
	ctx, cancelRoot := context.WithCancel(t.Context())
	events := []string{}
	err := startServices(
		ctx,
		cancelRoot,
		[]bundleService{{
			name: "test",
			app: new(app.App).Register(&lifecycleTestRunnable{
				name:   "component",
				events: &events,
				runErr: context.Canceled,
			}),
		}},
		nil,
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("independent component cancellation was lost: %v", err)
	}
	if isRootInterruption(ctx, err) {
		t.Fatal("independent component cancellation became a clean root interruption")
	}
}

func TestStartServicesPreservesFailureJoinedWithRootCancellation(t *testing.T) {
	ctx, cancelRoot := context.WithCancel(t.Context())
	persistenceErr := errors.New("persistence failed")
	events := []string{}
	err := startServices(
		ctx,
		cancelRoot,
		[]bundleService{{
			name: "test",
			app: new(app.App).Register(&lifecycleTestRunnable{
				name:   "component",
				events: &events,
				onRun:  cancelRoot,
				runErr: errors.Join(context.Canceled, persistenceErr),
			}),
		}},
		nil,
	)

	if isRootInterruption(ctx, err) || !errors.Is(err, persistenceErr) {
		t.Fatalf("joined component failure was lost: %v", err)
	}
}

func TestStartServicesCancelsRootBeforeRollback(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	initErr := errors.New("init failed")
	events := []string{}
	var rootErrAtClose error
	runnable := &lifecycleTestRunnable{
		name:    "failing",
		events:  &events,
		initErr: initErr,
		onClose: func() {
			rootErrAtClose = ctx.Err()
		},
	}

	err := startServices(ctx, cancel, []bundleService{{
		name: "test",
		app:  new(app.App).Register(runnable),
	}}, nil)
	if err == nil {
		t.Fatal("expected startup error")
	}
	if !errors.Is(rootErrAtClose, context.Canceled) {
		t.Fatalf("rollback began before root cancellation: %v", rootErrAtClose)
	}
}

// The bundle apps depend on earlier apps, so shutdown uses one deadline in
// reverse order and still attempts every close after a failure.
func TestShutdownServicesClosesEverythingInReverseUnderOneDeadline(t *testing.T) {
	events := []string{}
	closeContexts := []context.Context{}
	first := &lifecycleTestRunnable{
		name:          "first",
		events:        &events,
		closeContexts: &closeContexts,
		closeErr:      errors.New("first close failed"),
	}
	second := &lifecycleTestRunnable{
		name:          "second",
		events:        &events,
		closeContexts: &closeContexts,
		closeErr:      errors.New("second close failed"),
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(t.Context(), time.Minute)
	defer cancelShutdown()
	shutdownDeadline, _ := shutdownCtx.Deadline()
	err := shutdownServices(shutdownCtx, []bundleService{
		{name: "one", app: new(app.App).Register(first)},
		{name: "two", app: new(app.App).Register(second)},
	})
	if err == nil {
		t.Fatal("expected shutdown errors")
	}
	for _, message := range []string{"one", "first close failed", "two", "second close failed"} {
		if !strings.Contains(err.Error(), message) {
			t.Fatalf("shutdown error does not contain %q: %v", message, err)
		}
	}
	want := []string{"close:second", "close:first"}
	if !slices.Equal(events, want) {
		t.Fatalf("unexpected close order: got %v want %v", events, want)
	}
	if len(closeContexts) != len(want) {
		t.Fatalf("unexpected context count: got %d want %d", len(closeContexts), len(want))
	}
	for _, closeCtx := range closeContexts {
		closeDeadline, ok := closeCtx.Deadline()
		if !ok || !closeDeadline.Equal(shutdownDeadline) {
			t.Fatalf("service received a different shutdown deadline: %v", closeDeadline)
		}
	}
}

func TestStartServicesInitFailureClosesInitializedComponentsAndReturnsEveryError(t *testing.T) {
	events := []string{}
	first := &lifecycleTestRunnable{
		name:     "first",
		events:   &events,
		closeErr: errors.New("first close failed"),
	}
	second := &lifecycleTestRunnable{
		name:     "second",
		events:   &events,
		initErr:  errors.New("init failed"),
		closeErr: errors.New("second close failed"),
	}

	err := startServices(
		context.Background(),
		func() {},
		[]bundleService{{
			name: "test",
			app:  new(app.App).Register(first).Register(second),
		}},
		nil,
	)
	if err == nil {
		t.Fatal("expected init and cleanup errors")
	}
	for _, message := range []string{"init failed", "first close failed", "second close failed"} {
		if !strings.Contains(err.Error(), message) {
			t.Fatalf("error does not contain %q: %v", message, err)
		}
	}
	want := []string{
		"init:first",
		"init:second",
		"close:second",
		"close:first",
	}
	if !slices.Equal(events, want) {
		t.Fatalf("unexpected lifecycle: got %v want %v", events, want)
	}
}

func TestStartServices_RunFailureClosesCurrentAndPreviousServices(t *testing.T) {
	events := []string{}
	closeContexts := []context.Context{}
	firstService := &lifecycleTestRunnable{
		name:          "one",
		events:        &events,
		closeContexts: &closeContexts,
	}
	secondServiceFirst := &lifecycleTestRunnable{
		name:          "two-a",
		events:        &events,
		closeContexts: &closeContexts,
	}
	secondServiceSecond := &lifecycleTestRunnable{
		name:          "two-b",
		events:        &events,
		closeContexts: &closeContexts,
		runErr:        errors.New("boom"),
	}

	appOne := new(app.App).Register(firstService)
	appTwo := new(app.App).
		Register(secondServiceFirst).
		Register(secondServiceSecond)

	err := startServices(
		context.Background(),
		func() {},
		[]bundleService{
			{name: "svc-one", app: appOne},
			{name: "svc-two", app: appTwo},
		},
		nil,
	)
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
	if len(closeContexts) != 3 {
		t.Fatalf("unexpected close context count: %d", len(closeContexts))
	}
	for _, closeCtx := range closeContexts[1:] {
		if closeCtx != closeContexts[0] {
			t.Fatal("startup rollback used more than one shutdown context")
		}
	}
}

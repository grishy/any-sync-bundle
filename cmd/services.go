package cmd

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"go.uber.org/zap"

	"github.com/grishy/any-sync-bundle/lightnode"
)

// runServices starts the bundle, waits for the root context, and stops the
// bundle. A startup failure stops the root context with that failure.
func runServices(ctx context.Context, stopRoot context.CancelCauseFunc, bundle *lightnode.Bundle) error {
	err := startServices(ctx, bundle)
	switch {
	case err == nil:
		if ctx.Err() == nil {
			emitBundleEvent(bundleReadyEvent)
			printStartupMsg()
		}
		<-ctx.Done()
	case stoppedByOperator(ctx):
		// The stop most likely caused this error, so it is not a failure.
		log.Info("startup interrupted", zap.Error(err))
		err = nil
	default:
		stopRoot(err)
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.WithoutCancel(ctx), servicesShutdownTimeout)
	defer cancelShutdown()
	return errors.Join(err, stopServices(shutdownCtx, bundle))
}

// startServices starts each service with upstream app.Start, which closes the
// service's own components when one fails. The listeners open last, so no
// request reads the shared DRPC mux before every handler is registered.
func startServices(ctx context.Context, bundle *lightnode.Bundle) error {
	for _, s := range bundle.Services {
		if err := ctx.Err(); err != nil {
			return err
		}
		log.Info("▶ starting service", zap.String("name", s.Name))
		if err := s.App.Start(ctx); err != nil {
			return fmt.Errorf("start %s: %w", s.Name, err)
		}
		log.Info("✓ service started", zap.String("name", s.Name))
	}

	for _, l := range bundle.Listeners {
		if err := l.Run(ctx); err != nil {
			return fmt.Errorf("open listener %s: %w", l.Name(), err)
		}
	}
	log.Info("✓ listeners open")
	return nil
}

// stopServices closes everything, including what never started: transports
// close safely without Run, and app.Close closes only components whose Init
// was reached.
func stopServices(ctx context.Context, bundle *lightnode.Bundle) error {
	var errs []error
	for _, l := range bundle.Listeners {
		if err := l.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("close listener %s: %w", l.Name(), err))
		}
	}

	for _, s := range slices.Backward(bundle.Services) {
		log.Info("▶ stopping service", zap.String("name", s.Name))
		if err := s.App.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop %s: %w", s.Name, err))
			continue
		}
		log.Info("✓ service stopped", zap.String("name", s.Name))
	}
	return errors.Join(errs...)
}

// stoppedByOperator reports whether a signal, rather than a failure, stopped the
// root context. Failures stop it with their own error as the cause, while
// os/signal.NotifyContext uses a cause that matches context.Canceled:
// https://pkg.go.dev/os/signal@go1.27.2#NotifyContext
func stoppedByOperator(ctx context.Context) bool {
	return ctx.Err() != nil && errors.Is(context.Cause(ctx), context.Canceled)
}

package cmd

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/anyproto/any-sync/app"
	"go.uber.org/zap"

	"github.com/grishy/any-sync-bundle/config"
)

type bundleService struct {
	name string
	app  *app.App
}

// startServices initializes and runs all bundle services using a custom two-phase approach.
//
// Why we can't use app.Start() directly:
// The bundle architecture has 4 separate apps (coordinator, consensus, filenode, sync) that
// share a single DRPC multiplexer from the coordinator's server component. If we call
// app.Start() sequentially on each service, a race condition occurs:
//
//  1. coordinator.Start() = Init (registers handlers) + Run (starts network listeners)
//  2. Network is now accepting connections and calling mux.HandleRPC()
//  3. consensus.Start() = Init tries to register handlers on the same mux
//  4. RACE: goroutine reads mux map (HandleRPC) while another writes to it (register)
func startServices(
	ctx context.Context,
	cancelRoot context.CancelFunc,
	services []bundleService,
	cfg *config.Config,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	log.Info("initiating service startup", zap.Int("count", len(services)))
	log.Info("━━━ Phase 1: Initializing all services ━━━")

	rollbackCtx := context.WithoutCancel(ctx)
	var initialized []bundleService
	for _, service := range services {
		partial, err := initOneApp(ctx, service)
		if err != nil {
			rootErr := ctx.Err()
			cancelRoot()
			shutdownCtx, cancelShutdown := context.WithTimeout(rollbackCtx, servicesShutdownTimeout)
			partialErr := shutdownRunnables(shutdownCtx, service.name, partial)
			shutdownErr := shutdownServices(shutdownCtx, initialized)
			cancelShutdown()
			cleanupErr := errors.Join(partialErr, shutdownErr)
			return startupResult(rootErr, err, cleanupErr)
		}
		initialized = append(initialized, service)
	}
	log.Info("✓ all services initialized, all DRPC handlers registered")

	// Phase 2: Run all services
	// Track which services have been successfully Run() to avoid closing
	// components that were Init'd but never Run'd (they may have nil pointers).
	log.Info("━━━ Phase 2: Running all services ━━━")
	var running []bundleService
	for _, service := range initialized {
		partial, err := runOneApp(ctx, service, cfg)
		if err != nil {
			rootErr := ctx.Err()
			cancelRoot()
			shutdownCtx, cancelShutdown := context.WithTimeout(rollbackCtx, servicesShutdownTimeout)
			partialErr := shutdownRunnables(shutdownCtx, service.name, partial)
			shutdownErr := shutdownServices(shutdownCtx, running)
			cancelShutdown()
			cleanupErr := errors.Join(partialErr, shutdownErr)
			return startupResult(rootErr, err, cleanupErr)
		}
		running = append(running, service)
	}
	log.Info("✓ all services running")

	return nil
}

// initOneApp initializes all components for a single app.
func initOneApp(
	ctx context.Context,
	service bundleService,
) ([]app.ComponentRunnable, error) {
	log.Info("▶ initializing service", zap.String("name", service.name))

	var firstError error
	var initialized []app.ComponentRunnable
	var failedRunnable app.ComponentRunnable

	service.app.IterateComponents(func(component app.Component) {
		if firstError != nil {
			return
		}
		if err := ctx.Err(); err != nil {
			firstError = err
			return
		}
		if err := component.Init(service.app); err != nil {
			firstError = fmt.Errorf("component '%s': %w", component.Name(), err)
			if runnable, ok := component.(app.ComponentRunnable); ok {
				failedRunnable = runnable
			}
			return
		}

		if runnable, ok := component.(app.ComponentRunnable); ok {
			initialized = append(initialized, runnable)
		}
	})

	if firstError == nil {
		firstError = ctx.Err()
	}
	if firstError != nil {
		if isRootInterruption(ctx, firstError) {
			return initialized, firstError
		}
		if failedRunnable != nil {
			initialized = append(initialized, failedRunnable)
		}
		return initialized, fmt.Errorf("service '%s' init failed: %w", service.name, firstError)
	}

	log.Info("✓ service initialized", zap.String("name", service.name))
	return nil, nil
}

// runOneApp runs all runnable components for a single app.
func runOneApp(
	ctx context.Context,
	service bundleService,
	cfg *config.Config,
) ([]app.ComponentRunnable, error) {
	log.Info("▶ running service", zap.String("name", service.name))

	var firstError error
	var running []app.ComponentRunnable
	var failedRunnable app.ComponentRunnable

	service.app.IterateComponents(func(component app.Component) {
		if firstError != nil {
			return
		}
		runnable, ok := component.(app.ComponentRunnable)
		if !ok {
			return
		}
		if err := ctx.Err(); err != nil {
			firstError = err
			return
		}
		if err := runnable.Run(ctx); err != nil {
			if isRootInterruption(ctx, err) {
				firstError = err
				failedRunnable = runnable
				return
			}
			firstError = fmt.Errorf("component '%s': %w", runnable.Name(), err)
			failedRunnable = runnable
			return
		}
		running = append(running, runnable)
	})

	if firstError != nil {
		if failedRunnable != nil {
			running = append(running, failedRunnable)
		}
		if isRootInterruption(ctx, firstError) {
			return running, firstError
		}
		return running, fmt.Errorf("service '%s' run failed: %w", service.name, firstError)
	}

	if service.name != "coordinator" {
		log.Info("✓ service running", zap.String("name", service.name))
		return nil, nil
	}

	addr := cfg.Network.ListenTCPAddr
	log.Info("waiting for coordinator TCP listener", zap.String("addr", addr))
	if err := waitForTCPReady(ctx, addr, 5*time.Second); err != nil {
		if isRootInterruption(ctx, err) {
			return running, err
		}
		return running, fmt.Errorf("coordinator network not ready: %w", err)
	}

	log.Info("coordinator network ready")
	log.Info("✓ service running", zap.String("name", service.name))
	return nil, nil
}

func shutdownRunnables(
	ctx context.Context,
	serviceName string,
	runnables []app.ComponentRunnable,
) error {
	if len(runnables) == 0 {
		return nil
	}

	log.Info("⚡ cleaning up partially started service",
		zap.String("name", serviceName),
		zap.Int("components", len(runnables)))

	var errs []error
	for _, runnable := range slices.Backward(runnables) {
		log.Info("▶ stopping component",
			zap.String("service", serviceName),
			zap.String("component", runnable.Name()))

		if err := runnable.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf(
				"service %s component %s cleanup failed: %w",
				serviceName,
				runnable.Name(),
				err,
			))
			continue
		}

		log.Info("✓ component cleaned up",
			zap.String("service", serviceName),
			zap.String("component", runnable.Name()))
	}

	return errors.Join(errs...)
}

func shutdownServices(ctx context.Context, services []bundleService) error {
	log.Info("⚡ initiating service shutdown", zap.Int("count", len(services)))

	var errs []error
	for _, service := range slices.Backward(services) {
		log.Info("▶ stopping service", zap.String("name", service.name))

		if err := service.app.Close(ctx); err != nil {
			errs = append(errs,
				fmt.Errorf("service %s shutdown failed: %w", service.name, err))
		} else {
			log.Info("✓ service stopped successfully", zap.String("name", service.name))
		}
	}

	return errors.Join(errs...)
}

// A root interruption remains the startup result only when cleanup adds no
// failure. Every other combination retains all concrete errors.
func startupResult(rootErr, startupErr, cleanupErr error) error {
	if rootErr != nil && startupErr == rootErr { //nolint:errorlint // Error traversal would weaken the invariant.
		if cleanupErr != nil {
			return cleanupErr
		}
		return rootErr
	}
	return errors.Join(startupErr, cleanupErr)
}

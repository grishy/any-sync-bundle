package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grishy/any-sync-bundle/cmd"
)

func main() {
	// The root context is the process lifetime. A failure stops it with its
	// error as the cause; a signal stops it with a cause matching context.Canceled.
	rootCtx, stopRoot := context.WithCancelCause(context.Background())
	ctx, stopSignals := signal.NotifyContext(
		rootCtx,
		// Match Kubernetes' cross-platform termination signal set.
		syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT,
	)
	defer stopSignals()

	go func() {
		<-ctx.Done()
		time.Sleep(cmd.ShutdownTimeout)
		fmt.Fprintln(os.Stderr, "\nForced exit by timeout")
		os.Exit(1)
	}()

	if err := cmd.Root(ctx, stopRoot).Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "\nError:\n > %+v\n", err)
		os.Exit(1) //nolint:gocritic // need to exit with error code
	}
}

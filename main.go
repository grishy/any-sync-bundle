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
	ctx, cancelRoot := signal.NotifyContext(
		context.Background(),
		// Match Kubernetes' cross-platform termination signal set.
		syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT,
	)
	defer cancelRoot()

	cliRoot := cmd.Root(ctx, cancelRoot)

	go func() {
		<-ctx.Done()
		time.Sleep(cmd.ShutdownTimeout)
		fmt.Println("\nForced exit by timeout")
		os.Exit(1)
	}()

	if err := cliRoot.Run(os.Args); err != nil {
		cancelRoot()
		fmt.Println("\nError:")
		fmt.Printf(" > %+v\n", err)
		os.Exit(1) //nolint:gocritic // need to exit with error code
	}
}

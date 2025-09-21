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
	// terminationSignals are signals that cause the program to exit in the supported platforms.
	// List from kubectl project.
	terminationSignals := []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}

	ctx, cancel := signal.NotifyContext(context.Background(), terminationSignals...)
	defer cancel()

	cliRoot := cmd.Root(ctx)

	go func() {
		<-ctx.Done()
		fmt.Println("Context done, waiting before forced exit")
		time.Sleep(30 * time.Second)
		fmt.Println("Forced exit by timeout")
		os.Exit(1)
	}()

	if err := cliRoot.Run(os.Args); err != nil {
		fmt.Println("Error:")
		fmt.Printf(" > %+v\n", err)
		cancel()
		os.Exit(1) //nolint:gocritic // need to exit with error code
	}
}

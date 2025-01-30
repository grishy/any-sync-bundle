package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/grishy/any-sync-bundle/cmd"
)

// terminationSignals are signals that cause the program to exit in the supported platforms.
// List from kubectl project.
var terminationSignals = []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), terminationSignals...)
	defer cancel()

	cliRoot := cmd.Root(ctx)
	if err := cliRoot.Run(os.Args); err != nil {
		fmt.Println("Error:")
		fmt.Printf(" > %+v\n", err)
		os.Exit(1)
	}
}

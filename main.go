package main

import (
	"context"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"go.uber.org/zap"

	bundleNode "any-sync-bundle/node"
)

var log = logger.NewNamed("main")

func main() {
	// TODO: Replace it on new build-in version of it in Go
	app.AppName = "any-sync-bundle"

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	// Common configs
	apps := []*app.App{
		bundleNode.NewFileNodeApp(logger.NewNamed("filenode")),
		bundleNode.NewCoordinatorApp(logger.NewNamed("coordinator")),
	}

	// start apps
	for _, app := range apps {
		if err := app.Start(ctx); err != nil {
			log.Fatal("can't start app", zap.Error(err))
		}
	}
	log.Info("apps started")

	// wait exit signal
	<-ctx.Done()

	// close apps
	for _, app := range apps {
		ctxClose, cancelClose := context.WithTimeout(context.Background(), time.Minute)
		if err := app.Close(ctxClose); err != nil {
			log.Fatal("close error", zap.Error(err))
		}
		cancelClose()
	}
	log.Info("goodbye!")

	time.Sleep(time.Second / 3)
}

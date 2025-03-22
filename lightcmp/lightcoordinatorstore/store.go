package lightcoordinatorstore

import (
	"context"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/nodeconf"
)

const (
	CName = "light.coordinator.store"
)

var log = logger.NewNamed(CName)

type lightcoordinatorstore struct{}

type configService interface {
	app.Component
	GetNodeConf() nodeconf.Configuration
}

func New() *lightcoordinatorstore {
	return new(lightcoordinatorstore)
}

//
// App Component
//

func (r *lightcoordinatorstore) Init(a *app.App) error {
	log.Info("call Init")

	// r.srvCfg = app.MustComponent[configService](a)

	return nil
}

func (r *lightcoordinatorstore) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (r *lightcoordinatorstore) Run(ctx context.Context) error {
	log.Info("call Run")

	return nil
}

func (r *lightcoordinatorstore) Close(ctx context.Context) error {
	log.Info("call Close")
	return nil
}

//
// Component methods
//

package app

import (
	"os"

	"github.com/anyproto/any-sync-filenode/account"
	"github.com/anyproto/any-sync-filenode/config"
	"github.com/anyproto/any-sync-filenode/deletelog"
	"github.com/anyproto/any-sync-filenode/filenode"
	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/redisprovider"

	"github.com/anyproto/any-sync/acl"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/consensus/consensusclient"
	"github.com/anyproto/any-sync/coordinator/coordinatorclient"
	"github.com/anyproto/any-sync/coordinator/nodeconfsource"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/any-sync/nodeconf/nodeconfstore"

	"go.uber.org/zap"

	"github.com/grishy/any-sync-bundle/component/storeBadger"
)

func NewFileNodeApp(cfg *config.Config, fileDir string) *app.App {
	log := logger.NewNamed("filenode")

	// TODO: Remove when merged https://github.com/anyproto/any-sync/pull/374
	if err := os.MkdirAll(cfg.NetworkStorePath, 0o775); err != nil {
		log.Panic("can't create directory network store", zap.Error(err))
	}

	a := new(app.App).
		Register(cfg).
		Register(metric.New()).
		Register(account.New()).
		Register(nodeconfsource.New()).
		Register(nodeconfstore.New()).
		Register(nodeconf.New()).
		Register(peerservice.New()).
		Register(secureservice.New()).
		Register(pool.New()).
		Register(coordinatorclient.New()).
		Register(consensusclient.New()).
		Register(acl.New()).
		// Register(filedevstore.New()).
		Register(storeBadger.New(fileDir)).
		Register(redisprovider.New()).
		Register(index.New()).
		Register(server.New()).
		Register(filenode.New()).
		Register(deletelog.New()).
		Register(yamux.New()).
		Register(quic.New())

	return a
}

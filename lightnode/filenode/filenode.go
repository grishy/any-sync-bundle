package filenode

import (
	"github.com/anyproto/any-sync-filenode/account"
	"github.com/anyproto/any-sync-filenode/config"
	"github.com/anyproto/any-sync-filenode/deletelog"
	"github.com/anyproto/any-sync-filenode/filenode"
	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/redisprovider"

	"github.com/anyproto/any-sync/acl"
	"github.com/anyproto/any-sync/app"
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

	"github.com/grishy/any-sync-bundle/lightcmp/lightconfig"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenoderpc"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
	"github.com/grishy/any-sync-bundle/lightcmp/lightnodeconf"
	"github.com/grishy/any-sync-bundle/lightnode"
)

func NewApp(cfg *config.Config, fileDir string) *app.App {
	lCfg := &lightconfig.LightConfig{
		Account:          cfg.Account,
		Network:          cfg.Network,
		ListenTCPAddr:    cfg.Yamux.ListenAddrs,
		ListenUDPAddr:    cfg.Quic.ListenAddrs,
		FilenodeStoreDir: "./data/filenode_store",
	}

	a := new(app.App).
		Register(lCfg).
		Register(lightnodeconf.New()).
		Register(lightfilenodeindex.New()).
		Register(lightfilenodestore.New()).
		Register(lightfilenoderpc.New()).
		// Original components
		Register(account.New()).
		// TODO: Use direct call for all clients
		Register(coordinatorclient.New()).
		Register(consensusclient.New()).
		Register(acl.New()).
		Register(peerservice.New()).
		Register(secureservice.New()).
		Register(pool.New()).
		Register(server.New()).
		Register(yamux.New()).
		Register(quic.New())

	return a
}

func NewFileNodeApp(cfg *config.Config, fileDir string) *app.App {
	lightnode.MustMkdirAll(cfg.NetworkStorePath)

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
		// Register(store()). // Original component replaced with storeBadger
		// TODO: Path is not working
		Register(lightfilenodestore.New()). // Bundle component
		Register(redisprovider.New()).
		Register(index.New()).
		Register(server.New()).
		Register(filenode.New()).
		Register(deletelog.New()).
		Register(yamux.New()).
		Register(quic.New())

	return a
}

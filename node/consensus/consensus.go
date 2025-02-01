package consensus

import (
	"github.com/anyproto/any-sync-consensusnode/account"
	"github.com/anyproto/any-sync-consensusnode/config"
	"github.com/anyproto/any-sync-consensusnode/consensusrpc"
	"github.com/anyproto/any-sync-consensusnode/db"
	"github.com/anyproto/any-sync-consensusnode/stream"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/coordinator/coordinatorclient"
	"github.com/anyproto/any-sync/coordinator/nodeconfsource"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/any-sync/nodeconf/nodeconfstore"

	"github.com/grishy/any-sync-bundle/node"
)

func NewLightConsensusApp(cfg *config.Config) *app.App {
	// TODO: Add limiter to server?

	// a := new(app.App).
	// 	Register(nodeconfstore.New()).     // Remove, static config
	// 	Register(nodeconfsource.New()).    // Remove, static config
	// 	Register(coordinatorclient.New()). // Replace, direct call of other service
	// 	Register(nodeconf.New()).          // Remove, provide node network configuration for other Original components
	// 	Register(pool.New()).              // Remove, provide pool of peers for other Original components
	// 	Register(peerservice.New()).       // Remove, provide peer service for pool
	// 	Register(cfg).                     // Provide config for original components
	// 	Register(account.New()).           // Original, to sign messages
	// 	Register(yamux.New()).             // Original, TCP transport
	// 	Register(quic.New()).              // Original, UDP transport
	// 	Register(secureservice.New()).     // Original, secure service on top of transport
	// 	Register(server.New()).            // Original, RPC server on top of secure service
	// 	Register(consensusrpc.New())       // gRPC API on top of RPC server

	lightCfg := &bundleConfig{
		Account:       cfg.Account,
		Network:       cfg.Network,
		ListenTCPAddr: cfg.Yamux.ListenAddrs[0],
		ListenUDPAddr: cfg.Quic.ListenAddrs[0],
	}

	lightNodeconf := &bundleNodeconf{}

	lightRpc := &bundleRpc{}

	a := new(app.App).
		Register(lightCfg).
		Register(lightNodeconf).
		Register(account.New()).       // Original, to sign messages
		Register(secureservice.New()). // Original, secure service on top of 'yamux' and 'quic'
		Register(yamux.New()).         // Original, TCP transport
		Register(quic.New()).          // Original, UDP transport
		Register(server.New()).        // Original, allow to register RPC services
		Register(peerservice.New()).   // Original, provide accepter for 'yamux' and 'quic'
		Register(pool.New()).          // Original, provide pool of peers for 'peerservice'
		Register(lightRpc)             // Bundle, registrated at the end, because call dRPC on init

	return a
}

func NewConsensusApp(cfg *config.Config) *app.App {
	node.MustMkdirAll(cfg.NetworkStorePath)

	a := new(app.App).
		Register(cfg).
		Register(account.New()).
		Register(nodeconf.New()).
		Register(nodeconfstore.New()).
		Register(nodeconfsource.New()).
		Register(coordinatorclient.New()).
		Register(pool.New()).
		Register(peerservice.New()).
		Register(yamux.New()).
		Register(quic.New()).
		Register(secureservice.New()).
		Register(server.New()).
		Register(db.New()).
		Register(stream.New()).
		Register(consensusrpc.New())

	return a
}

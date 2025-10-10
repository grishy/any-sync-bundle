package lightnode

import (
	"context"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/transport"
)

// noOpComponent provides no-op lifecycle methods for shared components.
type noOpComponent struct {
	name string
}

func (n *noOpComponent) Name() string                  { return n.name }
func (n *noOpComponent) Init(_ *app.App) error         { return nil }
func (n *noOpComponent) Run(_ context.Context) error   { return nil }
func (n *noOpComponent) Close(_ context.Context) error { return nil }

// Pool.
type sharedPool struct {
	noOpComponent
	pool.Pool
}

// Server.
type sharedServer struct {
	noOpComponent
	server.DRPCServer
}

func (w *sharedServer) Name() string { return w.noOpComponent.Name() }

func (w *sharedServer) Init(_ *app.App) error { return nil }

// Transport.
type sharedTransport struct {
	noOpComponent
	transport.Transport
}

// PeerService.
type sharedPeerService struct {
	noOpComponent
	peerservice.PeerService
}

func (w *sharedPeerService) Name() string { return w.noOpComponent.Name() }

func (w *sharedPeerService) Init(_ *app.App) error { return nil }

// SecureService.
type sharedSecureService struct {
	noOpComponent
	secureservice.SecureService
}

func (w *sharedSecureService) Name() string { return w.noOpComponent.Name() }

func (w *sharedSecureService) Init(_ *app.App) error { return nil }

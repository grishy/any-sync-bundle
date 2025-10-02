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
// When embedded, it prevents re-initialization while satisfying app.Component interface.
type noOpComponent struct {
	name string
}

func (n *noOpComponent) Name() string                  { return n.name }
func (n *noOpComponent) Init(_ *app.App) error         { return nil }
func (n *noOpComponent) Run(_ context.Context) error   { return nil }
func (n *noOpComponent) Close(_ context.Context) error { return nil }

// sharedPool wraps pool.Pool to prevent re-initialization.
// Methods from pool.Pool are automatically promoted via interface embedding.
type sharedPool struct {
	noOpComponent
	pool.Pool
}

// sharedServer wraps server.DRPCServer to prevent re-initialization.
// Methods from server.DRPCServer are automatically promoted via interface embedding.
type sharedServer struct {
	noOpComponent
	server.DRPCServer
}

// Name overrides to resolve ambiguity between noOpComponent and DRPCServer.
func (w *sharedServer) Name() string { return w.noOpComponent.Name() }

// Init overrides the embedded Init to prevent re-initialization.
func (w *sharedServer) Init(_ *app.App) error { return nil }

// sharedTransport wraps transport.Transport to prevent re-initialization.
// Methods from transport.Transport are automatically promoted via interface embedding.
type sharedTransport struct {
	noOpComponent
	transport.Transport
}

// sharedPeerService wraps peerservice.PeerService to prevent re-initialization.
// Methods from peerservice.PeerService are automatically promoted via interface embedding.
type sharedPeerService struct {
	noOpComponent
	peerservice.PeerService
}

// Name overrides to resolve ambiguity between noOpComponent and PeerService.
func (w *sharedPeerService) Name() string { return w.noOpComponent.Name() }

// Init overrides the embedded Init to prevent re-initialization.
func (w *sharedPeerService) Init(_ *app.App) error { return nil }

// sharedSecureService wraps secureservice.SecureService to prevent re-initialization.
// Methods from secureservice.SecureService are automatically promoted via interface embedding.
type sharedSecureService struct {
	noOpComponent
	secureservice.SecureService
}

// Name overrides to resolve ambiguity between noOpComponent and SecureService.
func (w *sharedSecureService) Name() string { return w.noOpComponent.Name() }

// Init overrides the embedded Init to prevent re-initialization.
func (w *sharedSecureService) Init(_ *app.App) error { return nil }

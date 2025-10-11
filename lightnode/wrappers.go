package lightnode

import (
	"context"
	"fmt"
	"reflect"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/transport"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
)

// extractComponent extracts a component from the coordinator and performs type assertion.
func extractComponent[T any](coordinator *app.App, componentName string) T {
	comp := coordinator.MustComponent(componentName)
	typed, ok := comp.(T)
	if !ok {
		expectedType := reflect.TypeFor[T]()
		panic(fmt.Sprintf(
			"component %s has unexpected type %T, expected %s",
			componentName,
			comp,
			expectedType.String(),
		))
	}
	return typed
}

// noOpComponent provides no-op lifecycle methods for shared components.
// When embedded in a wrapper, it prevents re-initialization of already-running components.
type noOpComponent struct {
	name string
}

func (n *noOpComponent) Name() string                  { return n.name }
func (n *noOpComponent) Init(_ *app.App) error         { return nil }
func (n *noOpComponent) Run(_ context.Context) error   { return nil }
func (n *noOpComponent) Close(_ context.Context) error { return nil }

// sharedPoolComponent wraps pool.Pool with no-op lifecycle methods.
type sharedPoolComponent struct {
	noOpComponent
	pool.Pool
}

func newSharedPoolComponent(coordinator *app.App) *sharedPoolComponent {
	return &sharedPoolComponent{
		noOpComponent: noOpComponent{name: pool.CName},
		Pool:          extractComponent[pool.Pool](coordinator, pool.CName),
	}
}

func (s *sharedPoolComponent) Name() string                  { return s.noOpComponent.Name() }
func (s *sharedPoolComponent) Init(_ *app.App) error         { return nil }
func (s *sharedPoolComponent) Run(_ context.Context) error   { return nil }
func (s *sharedPoolComponent) Close(_ context.Context) error { return nil }

// sharedServerComponent wraps server.DRPCServer with no-op lifecycle methods.
type sharedServerComponent struct {
	noOpComponent
	server.DRPCServer
}

func newSharedServerComponent(coordinator *app.App) *sharedServerComponent {
	return &sharedServerComponent{
		noOpComponent: noOpComponent{name: server.CName},
		DRPCServer:    extractComponent[server.DRPCServer](coordinator, server.CName),
	}
}

func (s *sharedServerComponent) Name() string                  { return s.noOpComponent.Name() }
func (s *sharedServerComponent) Init(_ *app.App) error         { return nil }
func (s *sharedServerComponent) Run(_ context.Context) error   { return nil }
func (s *sharedServerComponent) Close(_ context.Context) error { return nil }

// sharedPeerServiceComponent wraps peerservice.PeerService with no-op lifecycle methods.
type sharedPeerServiceComponent struct {
	noOpComponent
	peerservice.PeerService
}

func newSharedPeerServiceComponent(coordinator *app.App) *sharedPeerServiceComponent {
	return &sharedPeerServiceComponent{
		noOpComponent: noOpComponent{name: peerservice.CName},
		PeerService:   extractComponent[peerservice.PeerService](coordinator, peerservice.CName),
	}
}

func (s *sharedPeerServiceComponent) Name() string                  { return s.noOpComponent.Name() }
func (s *sharedPeerServiceComponent) Init(_ *app.App) error         { return nil }
func (s *sharedPeerServiceComponent) Run(_ context.Context) error   { return nil }
func (s *sharedPeerServiceComponent) Close(_ context.Context) error { return nil }

// sharedSecureServiceComponent wraps secureservice.SecureService with no-op lifecycle methods.
type sharedSecureServiceComponent struct {
	noOpComponent
	secureservice.SecureService
}

func newSharedSecureServiceComponent(coordinator *app.App) *sharedSecureServiceComponent {
	return &sharedSecureServiceComponent{
		noOpComponent: noOpComponent{name: secureservice.CName},
		SecureService: extractComponent[secureservice.SecureService](coordinator, secureservice.CName),
	}
}

func (s *sharedSecureServiceComponent) Name() string                  { return s.noOpComponent.Name() }
func (s *sharedSecureServiceComponent) Init(_ *app.App) error         { return nil }
func (s *sharedSecureServiceComponent) Run(_ context.Context) error   { return nil }
func (s *sharedSecureServiceComponent) Close(_ context.Context) error { return nil }

// sharedTransportComponent wraps transport.Transport with no-op lifecycle methods.
type sharedTransportComponent struct {
	noOpComponent
	transport.Transport
}

func newSharedYamuxComponent(coordinator *app.App) *sharedTransportComponent {
	return &sharedTransportComponent{
		noOpComponent: noOpComponent{name: yamux.CName},
		Transport:     extractComponent[transport.Transport](coordinator, yamux.CName),
	}
}

func newSharedQuicComponent(coordinator *app.App) *sharedTransportComponent {
	return &sharedTransportComponent{
		noOpComponent: noOpComponent{name: quic.CName},
		Transport:     extractComponent[transport.Transport](coordinator, quic.CName),
	}
}

func (s *sharedTransportComponent) Name() string                  { return s.noOpComponent.Name() }
func (s *sharedTransportComponent) Init(_ *app.App) error         { return nil }
func (s *sharedTransportComponent) Run(_ context.Context) error   { return nil }
func (s *sharedTransportComponent) Close(_ context.Context) error { return nil }

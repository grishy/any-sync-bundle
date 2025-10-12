package lightnode

import (
	"context"
	"fmt"
	"reflect"

	"github.com/anyproto/any-sync/accountservice"
	"github.com/anyproto/any-sync/acl"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/transport"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"
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

// sharedMetricComponent wraps metric.Metric with no-op lifecycle methods.
type sharedMetricComponent struct {
	noOpComponent
	metric.Metric
}

func newSharedMetricComponent(coordinator *app.App) *sharedMetricComponent {
	return &sharedMetricComponent{
		noOpComponent: noOpComponent{name: metric.CName},
		Metric:        extractComponent[metric.Metric](coordinator, metric.CName),
	}
}

func (s *sharedMetricComponent) Name() string                  { return s.noOpComponent.Name() }
func (s *sharedMetricComponent) Init(_ *app.App) error         { return nil }
func (s *sharedMetricComponent) Run(_ context.Context) error   { return nil }
func (s *sharedMetricComponent) Close(_ context.Context) error { return nil }

// sharedACLComponent wraps acl.AclService with no-op lifecycle methods.
// For future myself: ACL (Access Control List) manages space permissions
// by providing a caching layer over the consensus database.
// It answers: "Who owns space X? Can user Y write to it?"
//
// Usage Pattern in Bundle:
//   - Coordinator: writes new permissions (AclAddRecord) + reads for validation
//   - Filenode: reads only (OwnerPubKey, Permissions) for authorization checks
//   - Consensus: stores raw ACL records (doesn't use ACL component)
//   - Sync: doesn't use ACL
type sharedACLComponent struct {
	noOpComponent
	acl.AclService
}

func newSharedACLComponent(coordinator *app.App) *sharedACLComponent {
	return &sharedACLComponent{
		noOpComponent: noOpComponent{name: acl.CName},
		AclService:    extractComponent[acl.AclService](coordinator, acl.CName),
	}
}

func (s *sharedACLComponent) Name() string                  { return s.noOpComponent.Name() }
func (s *sharedACLComponent) Init(_ *app.App) error         { return nil }
func (s *sharedACLComponent) Run(_ context.Context) error   { return nil }
func (s *sharedACLComponent) Close(_ context.Context) error { return nil }

// sharedNodeConfComponent wraps nodeconf.Service with no-op lifecycle methods.
//
// NodeConf manages network configuration (node addresses, peer discovery, consensus peers).
// It periodically fetches updated configuration from the coordinator via periodicsync.
type sharedNodeConfComponent struct {
	noOpComponent
	nodeconf.Service
}

func newSharedNodeConfComponent(coordinator *app.App) *sharedNodeConfComponent {
	return &sharedNodeConfComponent{
		noOpComponent: noOpComponent{name: nodeconf.CName},
		Service:       extractComponent[nodeconf.Service](coordinator, nodeconf.CName),
	}
}

func (s *sharedNodeConfComponent) Name() string                  { return s.noOpComponent.Name() }
func (s *sharedNodeConfComponent) Init(_ *app.App) error         { return nil }
func (s *sharedNodeConfComponent) Run(_ context.Context) error   { return nil }
func (s *sharedNodeConfComponent) Close(_ context.Context) error { return nil }

// sharedNodeConfStoreComponent wraps nodeconf.Store with no-op lifecycle methods.
//
// NodeConfStore is a simple file-based storage for network configuration.
// It has no Run() or Close() methods, only Init() which creates the storage directory.
type sharedNodeConfStoreComponent struct {
	noOpComponent
	nodeconf.Store
}

func newSharedNodeConfStoreComponent(coordinator *app.App) *sharedNodeConfStoreComponent {
	return &sharedNodeConfStoreComponent{
		noOpComponent: noOpComponent{name: nodeconf.CNameStore},
		Store:         extractComponent[nodeconf.Store](coordinator, nodeconf.CNameStore),
	}
}

func (s *sharedNodeConfStoreComponent) Name() string          { return s.noOpComponent.Name() }
func (s *sharedNodeConfStoreComponent) Init(_ *app.App) error { return nil }

// sharedAccountComponent wraps accountservice.Service with no-op lifecycle methods.
//
// Account holds the node's cryptographic identity (peer key + signing key).
// All services in the bundle share the same identity - they are one logical node.
//
// Why Wrap:
//  1. Consistency: All shared components should use the wrapper pattern
//  2. Clarity: Makes it explicit all services share one identity
type sharedAccountComponent struct {
	noOpComponent
	accountservice.Service
}

func newSharedAccountComponent(coordinator *app.App) *sharedAccountComponent {
	return &sharedAccountComponent{
		noOpComponent: noOpComponent{name: accountservice.CName},
		Service:       extractComponent[accountservice.Service](coordinator, accountservice.CName),
	}
}

func (s *sharedAccountComponent) Name() string          { return s.noOpComponent.Name() }
func (s *sharedAccountComponent) Init(_ *app.App) error { return nil }

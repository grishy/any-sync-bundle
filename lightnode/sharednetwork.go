package lightnode

import (
	"github.com/anyproto/any-sync/accountservice"
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

// sharedNetwork holds network components extracted from coordinator.
// Lifecycle components are wrapped to prevent re-initialization.
type sharedNetwork struct {
	Account       app.Component
	Pool          *sharedComponent[pool.Pool]
	Server        *sharedComponent[server.DRPCServer]
	Yamux         *sharedComponent[transport.Transport]
	Quic          *sharedComponent[transport.Transport]
	PeerService   *sharedComponent[peerservice.PeerService]
	SecureService *sharedComponent[secureservice.SecureService]
	NodeConf      app.Component
	NodeConfStore app.Component
	Metric        app.Component
}

// extractSharedNetwork extracts network components from coordinator.
// Wrapped components have no-op Init/Run/Close to prevent re-initialization.
func extractSharedNetwork(coordinator *app.App) *sharedNetwork {
	return &sharedNetwork{
		// Shared account (peer ID) - all services use coordinator's identity
		Account: coordinator.MustComponent(accountservice.CName),

		// Wrap lifecycle-aware components to prevent re-initialization
		Pool: newSharedComponent(
			pool.CName,
			coordinator.MustComponent(pool.CName).(pool.Pool),
		), //nolint:errcheck // MustComponent panics, doesn't return error
		Server: newSharedComponent(
			server.CName,
			coordinator.MustComponent(server.CName).(server.DRPCServer),
		), //nolint:errcheck // MustComponent panics, doesn't return error
		Yamux: newSharedComponent(
			yamux.CName,
			coordinator.MustComponent(yamux.CName).(transport.Transport),
		), //nolint:errcheck // MustComponent panics, doesn't return error
		Quic: newSharedComponent(
			quic.CName,
			coordinator.MustComponent(quic.CName).(transport.Transport),
		), //nolint:errcheck // MustComponent panics, doesn't return error
		PeerService: newSharedComponent(
			peerservice.CName,
			coordinator.MustComponent(peerservice.CName).(peerservice.PeerService),
		), //nolint:errcheck // MustComponent panics, doesn't return error
		SecureService: newSharedComponent(
			secureservice.CName,
			coordinator.MustComponent(secureservice.CName).(secureservice.SecureService),
		), //nolint:errcheck // MustComponent panics, doesn't return error

		// Pass through (pure data, no lifecycle side effects)
		NodeConf:      coordinator.MustComponent(nodeconf.CName),
		NodeConfStore: coordinator.MustComponent(nodeconf.CNameStore),
		Metric:        coordinator.MustComponent(metric.CName),
	}
}

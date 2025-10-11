package lightnode

import (
	"github.com/anyproto/any-sync/accountservice"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/nodeconf"
)

// sharedNetwork holds network components extracted from coordinator.
// Lifecycle components are wrapped to prevent re-initialization.
type sharedNetwork struct {
	Account       app.Component
	Pool          *sharedPoolComponent
	Server        *sharedServerComponent
	Yamux         *sharedTransportComponent
	Quic          *sharedTransportComponent
	PeerService   *sharedPeerServiceComponent
	SecureService *sharedSecureServiceComponent
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
		Pool:          newSharedPoolComponent(coordinator),
		Server:        newSharedServerComponent(coordinator),
		Yamux:         newSharedYamuxComponent(coordinator),
		Quic:          newSharedQuicComponent(coordinator),
		PeerService:   newSharedPeerServiceComponent(coordinator),
		SecureService: newSharedSecureServiceComponent(coordinator),

		// Pass through (pure data, no lifecycle side effects)
		NodeConf:      coordinator.MustComponent(nodeconf.CName),
		NodeConfStore: coordinator.MustComponent(nodeconf.CNameStore),
		Metric:        coordinator.MustComponent(metric.CName),
	}
}

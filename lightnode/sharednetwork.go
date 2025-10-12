package lightnode

import (
	"github.com/anyproto/any-sync/app"
)

// sharedNetwork holds network components extracted from coordinator.
// Lifecycle components are wrapped to prevent re-initialization.
type sharedNetwork struct {
	Account       *sharedAccountComponent
	Pool          *sharedPoolComponent
	Server        *sharedServerComponent
	Yamux         *sharedTransportComponent
	Quic          *sharedTransportComponent
	PeerService   *sharedPeerServiceComponent
	SecureService *sharedSecureServiceComponent
	NodeConf      *sharedNodeConfComponent
	NodeConfStore *sharedNodeConfStoreComponent
	Metric        *sharedMetricComponent
	ACL           *sharedACLComponent
}

// extractSharedNetwork extracts network components from coordinator.
// All components are wrapped to prevent re-initialization and ensure consistency.
func extractSharedNetwork(coordinator *app.App) *sharedNetwork {
	return &sharedNetwork{
		// All shared components are wrapped for consistency and safety
		Account:       newSharedAccountComponent(coordinator),
		Pool:          newSharedPoolComponent(coordinator),
		Server:        newSharedServerComponent(coordinator),
		Yamux:         newSharedYamuxComponent(coordinator),
		Quic:          newSharedQuicComponent(coordinator),
		PeerService:   newSharedPeerServiceComponent(coordinator),
		SecureService: newSharedSecureServiceComponent(coordinator),
		NodeConf:      newSharedNodeConfComponent(coordinator),
		NodeConfStore: newSharedNodeConfStoreComponent(coordinator),
		Metric:        newSharedMetricComponent(coordinator),
		ACL:           newSharedACLComponent(coordinator),
	}
}

package lightnode

import (
	"github.com/anyproto/any-sync/app"
)

// sharedCmp holds components extracted from coordinator.
// Lifecycle components are wrapped to prevent re-initialization.
type sharedCmp struct {
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

// extractSharedCmp extracts components from coordinator.
func extractSharedCmp(coordinator *app.App) *sharedCmp {
	return &sharedCmp{
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

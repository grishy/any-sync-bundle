package lightnode

import (
	"github.com/anyproto/any-sync/app"

	"github.com/grishy/any-sync-bundle/adminui"
	"github.com/grishy/any-sync-bundle/config"
)

// Bundle holds all any-sync service applications with shared network infrastructure.
//
// Architecture:
// - Coordinator creates the network stack (TCP 33010, UDP 33020, DRPC mux, connection pool).
// - Other services (Consensus, FileNode, Sync) reuse coordinator's network components.
// - All services register their RPC handlers to the shared DRPC multiplexer:
//   - Coordinator: /CoordinatorService/*.
//   - Consensus: /ConsensusService/*.
//   - FileNode: /FileService/*.
//   - SyncNode: /SpaceSyncService/*.
//
// - AdminUI provides web-based administration interface on port 8888.
//
// Benefits:
// - Single port pair instead of 8 ports.
// - Shared connection pool for all services.
// - Reduced memory footprint.
// - True service bundling with admin interface.
type Bundle struct {
	Coordinator *app.App
	Consensus   *app.App
	FileNode    *app.App
	Sync        *app.App
	AdminUI     *app.App // Web-based admin interface
}

// NewBundle creates all services with shared network infrastructure.
//
// The coordinator is created first with a full network stack.
// Network components are extracted once and reused by all other services.
// AdminUI is created with access to coordinator and filenode for admin operations.
//
// Usage:
//
//	bundle := lightnode.NewBundle(configs)
//	Access apps: bundle.Coordinator, bundle.Consensus, bundle.FileNode, bundle.Sync, bundle.AdminUI.
func NewBundle(configs *config.NodeConfigs) *Bundle {
	coordinator := newCoordinatorApp(configs.Coordinator)
	net := extractSharedNetwork(coordinator)

	consensus := newConsensusApp(configs.Consensus, net)
	filenode := newFileNodeApp(configs.Filenode, configs.FilenodeStorePath, net)
	sync := newSyncApp(configs.Sync, net)

	// Create AdminUI with direct access to all apps
	adminUI := newAdminUIApp(configs.AdminUI, coordinator, consensus, filenode, sync)

	return &Bundle{
		Coordinator: coordinator,
		Consensus:   consensus,
		FileNode:    filenode,
		Sync:        sync,
		AdminUI:     adminUI,
	}
}

// newAdminUIApp creates the admin UI application with direct app references.
func newAdminUIApp(cfg adminui.Config, coordinator, consensus, filenode, sync *app.App) *app.App {
	return new(app.App).
		Register(adminui.New(cfg, coordinator, consensus, filenode, sync))
}

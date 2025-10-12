package lightnode

import (
	"github.com/anyproto/any-sync/app"

	"github.com/grishy/any-sync-bundle/config"
)

// Bundle holds all any-sync service applications with shared network infrastructure.
//
// Architecture:
// - Coordinator creates the network stack (TCP 33010, UDP 33020, DRPC mux, connection pool, etc.)
// - Other services (Consensus, FileNode, Sync) reuse coordinator's components.
type Bundle struct {
	Coordinator *app.App
	Consensus   *app.App
	FileNode    *app.App
	Sync        *app.App
}

// NewBundle creates all services with shared  infrastructure.
func NewBundle(configs *config.NodeConfigs) *Bundle {
	coordinator := newCoordinatorApp(configs.Coordinator)
	net := extractSharedCmp(coordinator)

	return &Bundle{
		Coordinator: coordinator,
		Consensus:   newConsensusApp(configs.Consensus, net),
		FileNode:    newFileNodeApp(configs.Filenode, configs.FilenodeStorePath, net),
		Sync:        newSyncApp(configs.Sync, net),
	}
}

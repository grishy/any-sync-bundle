package lightnode

import (
	"github.com/anyproto/any-sync/app"

	"github.com/grishy/any-sync-bundle/config"
)

// Bundle holds all any-sync service applications with shared network infrastructure.
//
// Architecture:
// - Coordinator creates the network stack (TCP 33010, UDP 33020, DRPC mux, connection pool)
// - Other services (Consensus, FileNode, Sync) reuse coordinator's network components
// - All services register their RPC handlers to the shared DRPC multiplexer:
//   - Coordinator: /CoordinatorService/*
//   - Consensus: /ConsensusService/*
//   - FileNode: /FileService/*
//   - SyncNode: /SpaceSyncService/*
//
// Benefits:
// - Single port pair instead of 8 ports
// - Shared connection pool for all services
// - Reduced memory footprint
// - True service bundling.
type Bundle struct {
	Coordinator *app.App
	Consensus   *app.App
	FileNode    *app.App
	Sync        *app.App
}

// NewBundle creates all services with shared network infrastructure.
//
// The coordinator is created first with a full network stack.
// Network components are extracted once and reused by all other services.
//
// Usage:
//
//	bundle := lightnode.NewBundle(configs)
//	// Access apps: bundle.Coordinator, bundle.Consensus, bundle.FileNode, bundle.Sync
func NewBundle(configs *config.NodeConfigs) *Bundle {
	coordinator := newCoordinatorApp(configs.Coordinator)
	net := extractSharedNetwork(coordinator)

	return &Bundle{
		Coordinator: coordinator,
		Consensus:   newConsensusApp(configs.Consensus, net),
		FileNode:    newFileNodeApp(configs.Filenode, configs.FilenodeStorePath, net),
		Sync:        newSyncApp(configs.Sync, net),
	}
}

package config

import (
	"net"
	"path/filepath"
	"time"

	consensusconfig "github.com/anyproto/any-sync-consensusnode/config"
	coordinatorconfig "github.com/anyproto/any-sync-coordinator/config"
	filenodeconfig "github.com/anyproto/any-sync-filenode/config"
	syncconfig "github.com/anyproto/any-sync-node/config"

	"github.com/anyproto/any-sync-coordinator/accountlimit"
	"github.com/anyproto/any-sync-coordinator/db"
	"github.com/anyproto/any-sync-coordinator/spacestatus"
	"github.com/anyproto/any-sync-filenode/redisprovider"
	"github.com/anyproto/any-sync-node/nodestorage"
	"github.com/anyproto/any-sync-node/nodesync"
	"github.com/anyproto/any-sync-node/nodesync/hotsync"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/commonspace/config"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/rpc"
	"github.com/anyproto/any-sync/net/rpc/limiter"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"

	"go.uber.org/zap"
)

// NodeConfigs holds configuration for all node types in the system.
type NodeConfigs struct {
	Coordinator *coordinatorconfig.Config
	Consensus   *consensusconfig.Config
	Filenode    *filenodeconfig.Config
	Sync        *syncconfig.Config

	// Used for our component and we can't add this into existing configs.
	FilenodeStorePath string
}

type nodeConfigOpts struct {
	pathNetworkStoreCoordinator string
	pathNetworkStoreConsensus   string
	pathNetworkStoreFilenode    string
	pathNetworkStoreSync        string
	pathStorageSync             string
	pathStorageFilenode         string

	networkCfg nodeconf.Configuration
	metricCfg  metric.Config
}

// NodeConfigs generates configurations for all node types based on the base config.
func (bc *Config) NodeConfigs() *NodeConfigs {
	opts := &nodeConfigOpts{
		pathNetworkStoreCoordinator: filepath.Join(bc.StoragePath, "network-store/coordinator"),
		pathNetworkStoreConsensus:   filepath.Join(bc.StoragePath, "network-store/consensus"),
		pathNetworkStoreFilenode:    filepath.Join(bc.StoragePath, "network-store/filenode"),
		pathNetworkStoreSync:        filepath.Join(bc.StoragePath, "network-store/sync"),
		pathStorageSync:             filepath.Join(bc.StoragePath, "storage-sync"),
		pathStorageFilenode:         filepath.Join(bc.StoragePath, "storage-file"),
		networkCfg:                  bc.networkCfg(),
		// TODO: Don't turn on metrics.
		// https://github.com/anyproto/any-sync/issues/373
		metricCfg: metric.Config{},
	}

	return &NodeConfigs{
		Coordinator: bc.coordinatorConfig(opts),
		Consensus:   bc.consensusConfig(opts),
		Filenode:    bc.filenodeConfig(opts),
		Sync:        bc.syncConfig(opts),

		FilenodeStorePath: opts.pathStorageFilenode,
	}
}

func (bc *Config) coordinatorConfig(opts *nodeConfigOpts) *coordinatorconfig.Config {
	return &coordinatorconfig.Config{
		Account: bc.Accounts.Coordinator,
		Drpc: rpc.Config{
			Stream: rpc.StreamConfig{MaxMsgSizeMb: 256},
		},
		Metric:                   opts.metricCfg,
		Network:                  opts.networkCfg,
		NetworkStorePath:         opts.pathNetworkStoreCoordinator,
		NetworkUpdateIntervalSec: 0,
		Mongo: db.Mongo{
			Connect:  bc.Nodes.Coordinator.MongoConnect,
			Database: bc.Nodes.Coordinator.MongoDatabase,
		},
		SpaceStatus: spacestatus.Config{
			RunSeconds:         5,
			DeletionPeriodDays: 0,
			SpaceLimit:         0,
		},
		Yamux: bc.yamuxConfig(bc.Nodes.Coordinator.ListenTCPAddr),
		Quic:  bc.quicConfig(bc.Nodes.Coordinator.ListenUDPAddr),
		AccountLimits: accountlimit.SpaceLimits{
			SpaceMembersRead:  1000,
			SpaceMembersWrite: 1000,
			SharedSpacesLimit: 1000,
		},
	}
}

func (bc *Config) consensusConfig(opts *nodeConfigOpts) *consensusconfig.Config {
	return &consensusconfig.Config{
		Drpc: rpc.Config{
			Stream: rpc.StreamConfig{MaxMsgSizeMb: 256},
		},
		Account:                  bc.Accounts.Consensus,
		Network:                  opts.networkCfg,
		NetworkStorePath:         opts.pathNetworkStoreConsensus,
		NetworkUpdateIntervalSec: 0,
		Mongo: consensusconfig.Mongo{
			Connect:       bc.Nodes.Consensus.MongoConnect,
			Database:      bc.Nodes.Consensus.MongoDatabase,
			LogCollection: "log",
		},
		Metric: opts.metricCfg,
		Log:    logger.Config{Production: false},
		Yamux:  bc.yamuxConfig(bc.Nodes.Consensus.ListenTCPAddr),
		Quic:   bc.quicConfig(bc.Nodes.Consensus.ListenUDPAddr),
	}
}

func (bc *Config) filenodeConfig(opts *nodeConfigOpts) *filenodeconfig.Config {
	const oneTerabyte = 1024 * 1024 * 1024 * 1024 // 1 TB in bytes

	return &filenodeconfig.Config{
		Account: bc.Accounts.File,
		Drpc: rpc.Config{
			Stream: rpc.StreamConfig{MaxMsgSizeMb: 256},
		},
		Yamux:                    bc.yamuxConfig(bc.Nodes.File.ListenTCPAddr),
		Quic:                     bc.quicConfig(bc.Nodes.File.ListenUDPAddr),
		Metric:                   opts.metricCfg,
		Redis:                    redisprovider.Config{Url: bc.Nodes.File.RedisConnect},
		Network:                  opts.networkCfg,
		NetworkStorePath:         opts.pathNetworkStoreFilenode,
		NetworkUpdateIntervalSec: 0,
		DefaultLimit:             oneTerabyte,
	}
}

func (bc *Config) syncConfig(opts *nodeConfigOpts) *syncconfig.Config {
	return &syncconfig.Config{
		Drpc: rpc.Config{
			Stream: rpc.StreamConfig{MaxMsgSizeMb: 256},
		},
		Account:                  bc.Accounts.Tree,
		Network:                  opts.networkCfg,
		NetworkStorePath:         opts.pathNetworkStoreSync,
		NetworkUpdateIntervalSec: 0,
		Space:                    config.Config{GCTTL: 60, SyncPeriod: 600},
		Storage:                  nodestorage.Config{Path: opts.pathStorageSync},
		Metric:                   opts.metricCfg,
		Log:                      logger.Config{Production: false},
		NodeSync:                 nodesync.Config{HotSync: hotsync.Config{}},
		Yamux:                    bc.yamuxConfig(bc.Nodes.Tree.ListenTCPAddr),
		Quic:                     bc.quicConfig(bc.Nodes.Tree.ListenUDPAddr),
		Limiter:                  limiter.Config{},
	}
}

func (bc *Config) yamuxConfig(listenAddr string) yamux.Config {
	return yamux.Config{
		ListenAddrs:     []string{listenAddr},
		WriteTimeoutSec: 10,
		DialTimeoutSec:  10,
	}
}

func (bc *Config) quicConfig(listenAddr string) quic.Config {
	return quic.Config{
		ListenAddrs:     []string{listenAddr},
		WriteTimeoutSec: 10,
		DialTimeoutSec:  10,
	}
}

func (bc *Config) networkCfg() nodeconf.Configuration {
	return nodeconf.Configuration{
		Id:        bc.ConfigID,
		NetworkId: bc.NetworkID,
		Nodes: []nodeconf.Node{
			{
				PeerId:    bc.Accounts.Coordinator.PeerId,
				Addresses: convertListenToConnect(bc.Nodes.Coordinator.NodeShared),
				Types:     []nodeconf.NodeType{nodeconf.NodeTypeCoordinator},
			},
			{
				PeerId:    bc.Accounts.Consensus.PeerId,
				Addresses: convertListenToConnect(bc.Nodes.Consensus.NodeShared),
				Types:     []nodeconf.NodeType{nodeconf.NodeTypeConsensus},
			},
			{
				PeerId:    bc.Accounts.Tree.PeerId,
				Addresses: convertListenToConnect(bc.Nodes.Tree.NodeShared),
				Types:     []nodeconf.NodeType{nodeconf.NodeTypeTree},
			},
			{
				PeerId:    bc.Accounts.File.PeerId,
				Addresses: convertListenToConnect(bc.Nodes.File.NodeShared),
				Types:     []nodeconf.NodeType{nodeconf.NodeTypeFile},
			},
		},
		CreationTime: time.Now(),
	}
}

// convertListenToConnect converts listen addresses to connection addresses,
// replacing 0.0.0.0 with 127.0.0.1 for local connections
func convertListenToConnect(listen NodeShared) []string {
	hostTCP, portTCP, err := net.SplitHostPort(listen.ListenTCPAddr)
	if err != nil {
		log.Panic("failed to split TCP listen addr", zap.Error(err))
	}
	if hostTCP == "0.0.0.0" {
		hostTCP = "127.0.0.1"
	}

	hostUDP, portUDP, err := net.SplitHostPort(listen.ListenUDPAddr)
	if err != nil {
		log.Panic("failed to split UDP listen addr", zap.Error(err))
	}
	if hostUDP == "0.0.0.0" {
		hostUDP = "127.0.0.1"
	}

	return []string{
		"quic://" + hostUDP + ":" + portUDP,
		hostTCP + ":" + portTCP,
	}
}

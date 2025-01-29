package config

import (
	"net"
	"path/filepath"
	"time"

	consensusconfig "github.com/anyproto/any-sync-consensusnode/config"
	coordinatorconfig "github.com/anyproto/any-sync-coordinator/config"
	filenodeconfig "github.com/anyproto/any-sync-filenode/config"
	syncconfig "github.com/anyproto/any-sync-node/config"
	"go.uber.org/zap"

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
)

type NodeConfigs struct {
	Coordinator *coordinatorconfig.Config
	Consensus   *consensusconfig.Config
	Filenode    *filenodeconfig.Config
	Sync        *syncconfig.Config
}

func (bc *Config) NodeConfigs() *NodeConfigs {
	networkCfg := bc.networkCfg()

	// TODO: Don't use metrics
	// https://github.com/anyproto/any-sync/issues/373
	metricCfg := metric.Config{}

	// TODO: Check all fields

	// Coordinator
	cfgCoord := &coordinatorconfig.Config{
		Account: bc.Accounts.Coordinator,
		Drpc: rpc.Config{
			Stream: rpc.StreamConfig{
				MaxMsgSizeMb: 256,
				// 	TODO: Issue that `timeoutMilliseconds: 1000` is not exist in the config
			},
		},
		Metric:                   metricCfg,
		Network:                  networkCfg,
		NetworkStorePath:         filepath.Join(bc.StoragePath, "network-store/coordinator"),
		NetworkUpdateIntervalSec: 0,
		Mongo: db.Mongo{
			Connect:  bc.Nodes.Coordinator.MongoConnect,
			Database: bc.Nodes.Coordinator.MongoDatabase,
			// TODO: Issue that `log: log` and `spaces: spaces` is not exist in the config
		},
		SpaceStatus: spacestatus.Config{
			RunSeconds:         5,
			DeletionPeriodDays: 0,
			SpaceLimit:         0,
		},
		Yamux: yamux.Config{
			ListenAddrs: []string{
				bc.Nodes.Coordinator.ListenTCPAddr,
			},
			WriteTimeoutSec: 10,
			DialTimeoutSec:  10,
		},
		Quic: quic.Config{
			ListenAddrs: []string{
				bc.Nodes.Coordinator.ListenUDPAddr,
			},
			WriteTimeoutSec: 10,
			DialTimeoutSec:  10,
		},
		AccountLimits: accountlimit.SpaceLimits{
			SpaceMembersRead:  1000,
			SpaceMembersWrite: 1000,
			SharedSpacesLimit: 1000,
		},
	}

	// Consensus
	cfgCons := &consensusconfig.Config{
		Drpc: rpc.Config{
			Stream: rpc.StreamConfig{
				MaxMsgSizeMb: 256,
			},
		},
		Account:                  bc.Accounts.Consensus,
		Network:                  networkCfg,
		NetworkStorePath:         filepath.Join(bc.StoragePath, "network-store/consensus"),
		NetworkUpdateIntervalSec: 0,
		Mongo: consensusconfig.Mongo{
			Connect:       bc.Nodes.Consensus.MongoConnect,
			Database:      bc.Nodes.Consensus.MongoDatabase,
			LogCollection: "log",
		},
		Metric: metricCfg,
		Log: logger.Config{
			Production: false,
		},
		Yamux: yamux.Config{
			ListenAddrs: []string{
				bc.Nodes.Consensus.ListenTCPAddr,
			},
			WriteTimeoutSec: 10,
			DialTimeoutSec:  10,
		},
		Quic: quic.Config{
			ListenAddrs: []string{
				bc.Nodes.Consensus.ListenUDPAddr,
			},
			WriteTimeoutSec: 10,
			DialTimeoutSec:  10,
		},
	}

	// Filenode
	cfgFileNode := &filenodeconfig.Config{
		Account: bc.Accounts.File,
		Drpc: rpc.Config{
			Stream: rpc.StreamConfig{
				MaxMsgSizeMb: 256,
			},
		},
		Yamux: yamux.Config{
			ListenAddrs: []string{
				bc.Nodes.File.ListenTCPAddr,
			},
			WriteTimeoutSec: 10,
			DialTimeoutSec:  10,
		},
		Quic: quic.Config{
			ListenAddrs: []string{
				bc.Nodes.File.ListenUDPAddr,
			},
			WriteTimeoutSec: 10,
			DialTimeoutSec:  10,
		},
		Metric: metricCfg,
		Redis: redisprovider.Config{
			IsCluster: false,
			Url:       bc.Nodes.File.RedisConnect,
		},
		Network:                  networkCfg,
		NetworkStorePath:         filepath.Join(bc.StoragePath, "network-store/filenode"),
		NetworkUpdateIntervalSec: 0,
		CafeMigrateKey:           "",
		DefaultLimit:             1099511627776, // 1 TB
		PersistTtl:               0,
	}

	// Sync
	cfgSync := &syncconfig.Config{
		Drpc: rpc.Config{
			Stream: rpc.StreamConfig{
				MaxMsgSizeMb: 256,
			},
		},
		Account:                  bc.Accounts.Tree,
		Network:                  networkCfg,
		NetworkStorePath:         filepath.Join(bc.StoragePath, "network-store/sync"),
		NetworkUpdateIntervalSec: 0,
		Space: config.Config{
			GCTTL:      60,
			SyncPeriod: 600,
		},
		Storage: nodestorage.Config{
			Path: filepath.Join(bc.StoragePath, "storage-sync"),
		},
		Metric: metricCfg,
		Log: logger.Config{
			Production: false,
		},
		NodeSync: nodesync.Config{
			SyncOnStart:       false,
			PeriodicSyncHours: 0,
			HotSync: hotsync.Config{
				SimultaneousRequests: 0,
			},
		},
		Yamux: yamux.Config{
			ListenAddrs: []string{
				bc.Nodes.Tree.ListenTCPAddr,
			},
			WriteTimeoutSec: 10,
			DialTimeoutSec:  10,
		},
		Limiter: limiter.Config{
			DefaultTokens: limiter.Tokens{
				TokensPerSecond: 0,
				MaxTokens:       0,
			},
			ResponseTokens: nil,
		},
		Quic: quic.Config{
			ListenAddrs: []string{
				bc.Nodes.Tree.ListenUDPAddr,
			},
			WriteTimeoutSec: 10,
			DialTimeoutSec:  10,
		},
	}

	return &NodeConfigs{
		Coordinator: cfgCoord,
		Consensus:   cfgCons,
		Filenode:    cfgFileNode,
		Sync:        cfgSync,
	}
}

// TODO: Support IPv6
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

func (bc *Config) networkCfg() nodeconf.Configuration {
	network := nodeconf.Configuration{
		Id:        bc.ConfigID,
		NetworkId: bc.NetworkID,
		Nodes: []nodeconf.Node{
			{
				PeerId:    bc.Accounts.Coordinator.PeerId,
				Addresses: convertListenToConnect(bc.Nodes.Coordinator.NodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeCoordinator,
				},
			},
			{
				PeerId:    bc.Accounts.Consensus.PeerId,
				Addresses: convertListenToConnect(bc.Nodes.Consensus.NodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeConsensus,
				},
			},
			{
				PeerId:    bc.Accounts.Tree.PeerId,
				Addresses: convertListenToConnect(bc.Nodes.Tree.NodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeTree,
				},
			},
			{
				PeerId:    bc.Accounts.File.PeerId,
				Addresses: convertListenToConnect(bc.Nodes.File.NodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeFile,
				},
			},
		},
		CreationTime: time.Now(),
	}
	return network
}

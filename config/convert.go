package config

import (
	"path/filepath"
	"time"

	consensusconfig "github.com/anyproto/any-sync-consensusnode/config"
	consensusdb "github.com/anyproto/any-sync-consensusnode/db"
	consensusdeletelog "github.com/anyproto/any-sync-consensusnode/deletelog"
	coordinatorconfig "github.com/anyproto/any-sync-coordinator/config"
	filenodeconfig "github.com/anyproto/any-sync-filenode/config"
	syncconfig "github.com/anyproto/any-sync-node/config"

	"github.com/anyproto/any-sync-coordinator/accountlimit"
	"github.com/anyproto/any-sync-coordinator/db"
	"github.com/anyproto/any-sync-coordinator/spacestatus"
	"github.com/anyproto/any-sync-filenode/redisprovider"
	"github.com/anyproto/any-sync-filenode/store/s3store"
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

const (
	// Sync node configuration defaults.
	defaultSyncOnStart       = true
	defaultPeriodicSyncHours = 2
	defaultSpaceGCTTL        = 60  // Seconds
	defaultSpaceSyncPeriod   = 600 // Seconds
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
		metricCfg:                   metric.Config{}, // TODO: Enable metrics (https://github.com/anyproto/any-sync/issues/373)
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
		Account: bc.Account,
		Drpc: rpc.Config{
			Stream: rpc.StreamConfig{MaxMsgSizeMb: 256},
			Snappy: true,
		},
		Metric:                   opts.metricCfg,
		Network:                  opts.networkCfg,
		NetworkStorePath:         opts.pathNetworkStoreCoordinator,
		NetworkUpdateIntervalSec: 0,
		Mongo: db.Mongo{
			Connect:  bc.Coordinator.MongoConnect,
			Database: bc.Coordinator.MongoDatabase,
		},
		SpaceStatus: spacestatus.Config{
			RunSeconds:         5,
			DeletionPeriodDays: 0,
			SpaceLimit:         0,
		},
		Yamux: bc.yamuxConfig(),
		Quic:  bc.quicConfig(),
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
			Snappy: true,
		},
		Account:                  bc.Account,
		Network:                  opts.networkCfg,
		NetworkStorePath:         opts.pathNetworkStoreConsensus,
		NetworkUpdateIntervalSec: 0,
		Mongo: consensusdb.Config{
			Connect:       bc.Consensus.MongoConnect,
			Database:      bc.Consensus.MongoDatabase,
			LogCollection: "log",
		},
		Metric: opts.metricCfg,
		Log:    logger.Config{Production: false},
		Yamux:  bc.yamuxConfig(),
		Quic:   bc.quicConfig(),
		Deletion: consensusdeletelog.Config{
			Enable: true,
		},
	}
}

func (bc *Config) filenodeConfig(opts *nodeConfigOpts) *filenodeconfig.Config {
	const oneTerabyte = 1024 * 1024 * 1024 * 1024 // 1 TiB in bytes

	cfg := &filenodeconfig.Config{
		Account: bc.Account,
		Drpc: rpc.Config{
			Stream: rpc.StreamConfig{MaxMsgSizeMb: 256},
		},
		Yamux:  bc.yamuxConfig(),
		Quic:   bc.quicConfig(),
		Metric: opts.metricCfg,
		Redis: redisprovider.Config{
			IsCluster: false,
			Url:       bc.FileNode.RedisConnect,
		},
		Network:                  opts.networkCfg,
		NetworkStorePath:         opts.pathNetworkStoreFilenode,
		NetworkUpdateIntervalSec: 0,
		DefaultLimit:             oneTerabyte,
	}

	// Configure S3 storage if S3 config is present
	if bc.FileNode.S3 != nil {
		cfg.S3Store = bc.convertS3Config()
	}

	return cfg
}

// convertS3Config converts bundle S3Config to upstream s3store.Config format.
func (bc *Config) convertS3Config() s3store.Config {
	s3 := bc.FileNode.S3

	cfg := s3store.Config{
		Region:         s3.Region,
		Bucket:         s3.BlockBucket,
		IndexBucket:    s3.IndexBucket,
		Endpoint:       s3.Endpoint,
		Profile:        s3.Profile,
		MaxThreads:     16,
		ForcePathStyle: s3.ForcePathStyle,
	}

	// Set default profile if not specified
	if cfg.Profile == "" {
		cfg.Profile = "default"
	}

	// Copy credentials if present
	if s3.Credentials != nil {
		cfg.Credentials = s3store.Credentials{
			AccessKey: s3.Credentials.AccessKey,
			SecretKey: s3.Credentials.SecretKey,
		}
	}

	return cfg
}

func (bc *Config) syncConfig(opts *nodeConfigOpts) *syncconfig.Config {
	return &syncconfig.Config{
		// APIServer omitted - disabled by default.
		Drpc: rpc.Config{
			Stream: rpc.StreamConfig{MaxMsgSizeMb: 256},
			Snappy: true,
		},
		Account:                  bc.Account,
		Network:                  opts.networkCfg,
		NetworkStorePath:         opts.pathNetworkStoreSync,
		NetworkUpdateIntervalSec: 0,
		Space:                    config.Config{GCTTL: defaultSpaceGCTTL, SyncPeriod: defaultSpaceSyncPeriod},
		// Storage paths: nodestorage uses ONLY AnyStorePath (see nodestorage/storageservice.go:284).
		// Path is for oldstorage (NOT included). Set to invalid path as fuse - app will fail if ever accessed.
		Storage: nodestorage.Config{
			Path:         "/dev/null/oldstorage-not-used", // Fuse: fail immediately if accessed
			AnyStorePath: opts.pathStorageSync,            // Actually used by nodestorage
		},
		Metric: opts.metricCfg,
		Log:    logger.Config{Production: false},
		NodeSync: nodesync.Config{
			SyncOnStart:       defaultSyncOnStart,
			PeriodicSyncHours: defaultPeriodicSyncHours,
			HotSync:           hotsync.Config{},
		},
		Yamux:   bc.yamuxConfig(),
		Quic:    bc.quicConfig(),
		Limiter: limiter.Config{},
	}
}

func (bc *Config) yamuxConfig() yamux.Config {
	return yamux.Config{
		ListenAddrs:     []string{bc.Network.ListenTCPAddr},
		WriteTimeoutSec: 10,
		DialTimeoutSec:  10,
	}
}

func (bc *Config) quicConfig() quic.Config {
	return quic.Config{
		ListenAddrs:     []string{bc.Network.ListenUDPAddr},
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
				PeerId:    bc.Account.PeerId,
				Addresses: bc.convertListenToConnect(),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeCoordinator,
					nodeconf.NodeTypeConsensus,
					nodeconf.NodeTypeTree,
					nodeconf.NodeTypeFile,
				},
			},
		},
		CreationTime: time.Now(),
	}
}

// convertListenToConnect replaces 0.0.0.0 with 127.0.0.1 for local connections.
func (bc *Config) convertListenToConnect() []string {
	endpoints := bc.listenEndpoints()

	hostTCP := endpoints.tcpHost
	if hostTCP == "0.0.0.0" {
		hostTCP = "127.0.0.1"
	}

	hostUDP := endpoints.udpHost
	if hostUDP == "0.0.0.0" {
		hostUDP = "127.0.0.1"
	}

	return []string{
		"quic://" + hostUDP + ":" + endpoints.udpPort,
		hostTCP + ":" + endpoints.tcpPort,
	}
}

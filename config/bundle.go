// Package config build on top of github.com/anyproto/any-sync-tools/anyconf
package config

import (
	"net/url"
	"os"
	"path/filepath"

	"github.com/anyproto/any-sync/accountservice"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/util/crypto"

	"go.uber.org/zap"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/yaml.v3"
)

var log = logger.NewNamed("bundle-config")

type Config struct {
	BundleVersion string   `yaml:"bundleVersion"`
	BundleFormat  int      `yaml:"bundleFormat"`
	ExternalAddr  []string `yaml:"externalAddr"`
	ConfigID      string   `yaml:"configId"`
	NetworkID     string   `yaml:"networkId"`
	StoragePath   string   `yaml:"storagePath"`
	Accounts      Accounts `yaml:"accounts"`
	Nodes         Nodes    `yaml:"nodes"`
}

type Accounts struct {
	Coordinator accountservice.Config `yaml:"coordinator"`
	Consensus   accountservice.Config `yaml:"consensus"`
	Tree        accountservice.Config `yaml:"tree"`
	File        accountservice.Config `yaml:"file"`
}

type Nodes struct {
	Coordinator NodeCoordinator `yaml:"coordinator"`
	Consensus   NodeConsensus   `yaml:"consensus"`
	Tree        Tree            `yaml:"tree"`
	File        NodeFile        `yaml:"file"`
}

type NodeShared struct {
	ListenTCPAddr string `yaml:"localTCPAddr"`
	ListenUDPAddr string `yaml:"localUDPAddr"`
}

type NodeCoordinator struct {
	NodeShared    `yaml:",inline"`
	MongoConnect  string `yaml:"mongoConnect"`
	MongoDatabase string `yaml:"mongoDatabase"`
}

type NodeConsensus struct {
	NodeShared    `yaml:",inline"`
	MongoConnect  string `yaml:"mongoConnect"`
	MongoDatabase string `yaml:"mongoDatabase"`
}

type Tree struct {
	NodeShared `yaml:",inline"`
}

type NodeFile struct {
	NodeShared   `yaml:",inline"`
	RedisConnect string `yaml:"redisConnect"`
}

func Read(cfgPath string) *Config {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Panic("can't read config file", zap.Error(err))
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Panic("can't unmarshal config", zap.Error(err))
	}

	return &cfg
}

type CreateOptions struct {
	CfgPath       string
	StorePath     string
	MongoURI      string
	RedisURI      string
	ExternalAddrs []string
}

func CreateWrite(cfg *CreateOptions) *Config {
	createdCfg := newBundleConfig(cfg)

	createCfgYaml, err := yaml.Marshal(createdCfg)
	if err != nil {
		log.Panic("can't marshal config", zap.Error(err))
	}

	if err := os.MkdirAll(filepath.Dir(cfg.CfgPath), 0o755); err != nil {
		log.Panic("can't create config directory", zap.Error(err))
	}

	if err := os.WriteFile(cfg.CfgPath, createCfgYaml, 0o644); err != nil {
		log.Panic("can't write config file", zap.Error(err))
	}

	return createdCfg
}

// newBundleConfig creates a new configuration for any-bundle that contain all info base of which internal services are created
// Base on https://tech.anytype.io/any-sync/configuration?id=common-nodes-configuration-options
// But docs above are not accurate, so I used also source code as reference...
func newBundleConfig(cfg *CreateOptions) *Config {
	cfgId := bson.NewObjectId().Hex()

	netKey, _, err := crypto.GenerateRandomEd25519KeyPair()
	if err != nil {
		log.Panic("can't generate ed25519 key for network", zap.Error(err))
	}

	netId := netKey.GetPublic().Network()

	// Parse MongoDB URI and add w=majority if not already present
	// Base on Anytype dockercompose version
	mongoConsensusURI, err := url.Parse(cfg.MongoURI)
	if err != nil {
		log.Panic("invalid mongo URI", zap.Error(err))
	}

	query := mongoConsensusURI.Query()
	if query.Get("w") == "" {
		query.Set("w", "majority")
		mongoConsensusURI.RawQuery = query.Encode()
	}

	defaultCfg := &Config{
		BundleFormat:  1,
		BundleVersion: app.Version(),
		ExternalAddr:  cfg.ExternalAddrs,
		ConfigID:      cfgId,
		NetworkID:     netId,
		StoragePath:   cfg.StorePath,
		Accounts: Accounts{
			Coordinator: newAcc(),
			Consensus:   newAcc(),
			Tree:        newAcc(),
			File:        newAcc(),
		},
		Nodes: Nodes{
			Coordinator: NodeCoordinator{
				NodeShared: NodeShared{
					ListenTCPAddr: "0.0.0.0:33010",
					ListenUDPAddr: "0.0.0.0:33020",
				},
				MongoConnect:  cfg.MongoURI,
				MongoDatabase: "coordinator",
			},
			Consensus: NodeConsensus{
				NodeShared: NodeShared{
					ListenTCPAddr: "0.0.0.0:33011",
					ListenUDPAddr: "0.0.0.0:33021",
				},
				MongoConnect:  mongoConsensusURI.String(),
				MongoDatabase: "consensus",
			},
			Tree: Tree{
				NodeShared{
					ListenTCPAddr: "0.0.0.0:33012",
					ListenUDPAddr: "0.0.0.0:33022",
				},
			},
			File: NodeFile{
				NodeShared: NodeShared{
					ListenTCPAddr: "0.0.0.0:33013",
					ListenUDPAddr: "0.0.0.0:33023",
				},
				RedisConnect: cfg.RedisURI,
			},
		},
	}

	// Base on docs https://tech.anytype.io/any-sync/configuration?id=common-nodes-configuration-options
	// "Signing key of coordinator is private key of the network and sync and file nodes use their peerKey"
	privNetKey, err := crypto.EncodeKeyToString(netKey)
	if err != nil {
		log.Panic("can't encode network key to string", zap.Error(err))
	}
	defaultCfg.Accounts.Coordinator.SigningKey = privNetKey

	return defaultCfg
}

func newAcc() accountservice.Config {
	signKey, _, err := crypto.GenerateRandomEd25519KeyPair()
	if err != nil {
		log.Panic("can't generate ed25519 key for account", zap.Error(err))
	}

	encPeerSignKey, err := crypto.EncodeKeyToString(signKey) // encSignKey
	if err != nil {
		log.Panic("can't encode key to string", zap.Error(err))
	}

	return accountservice.Config{
		PeerId:     signKey.GetPublic().PeerId(), // public key
		PeerKey:    encPeerSignKey,               // private key
		SigningKey: encPeerSignKey,               // private key
	}
}

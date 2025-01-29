// Package config build on top of github.com/anyproto/any-sync-tools/anyconf
package config

import (
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

func CreateWrite(cfgPath string) *Config {
	createdCfg := newBundleConfig()

	log.Info("writing new config", zap.String("path", cfgPath))

	createCfgYaml, err := yaml.Marshal(createdCfg)
	if err != nil {
		log.Panic("can't marshal config", zap.Error(err))
	}

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		log.Panic("can't create config directory", zap.Error(err))
	}

	if err := os.WriteFile(cfgPath, createCfgYaml, 0o644); err != nil {
		log.Panic("can't write config file", zap.Error(err))
	}

	return createdCfg
}

// newBundleConfig creates a new configuration for any-bundle that contain all info base of which internal services are created
// Base on https://tech.anytype.io/any-sync/configuration?id=common-nodes-configuration-options
// But docs above are not accurate, so I used also source code as reference...
func newBundleConfig() *Config {
	cfgId := bson.NewObjectId().Hex()

	netKey, _, err := crypto.GenerateRandomEd25519KeyPair()
	if err != nil {
		log.Panic("can't generate ed25519 key for network", zap.Error(err))
	}

	netId := netKey.GetPublic().Network()

	cfg := &Config{
		BundleFormat:  1,
		BundleVersion: app.Version(),
		// TODO: Read from flag values
		ExternalAddr: []string{
			"192.168.100.9",
		},
		ConfigID:  cfgId,
		NetworkID: netId,
		// TODO: Read from flag
		StoragePath: "./data/storage",
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
				MongoConnect:  "mongodb://127.0.0.1:27017/",
				MongoDatabase: "coordinator",
			},
			Consensus: NodeConsensus{
				NodeShared: NodeShared{
					ListenTCPAddr: "0.0.0.0:33011",
					ListenUDPAddr: "0.0.0.0:33021",
				},
				MongoConnect:  "mongodb://127.0.0.1:27017/?w=majority",
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
				RedisConnect: "redis://127.0.0.1:6379?dial_timeout=3&read_timeout=6s",
			},
		},
	}

	// Base on docs https://tech.anytype.io/any-sync/configuration?id=common-nodes-configuration-options
	// "Signing key of coordinator is private key of the network and sync and file nodes use their peerKey"
	privNetKey, err := crypto.EncodeKeyToString(netKey)
	if err != nil {
		log.Panic("can't encode network key to string", zap.Error(err))
	}
	cfg.Accounts.Coordinator.SigningKey = privNetKey

	return cfg
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

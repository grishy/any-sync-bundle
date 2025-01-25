// Package config build on top of github.com/anyproto/any-sync-tools/anyconf
package config

import (
	"os"
	"path/filepath"

	"github.com/anyproto/any-sync/accountservice"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/util/crypto"

	"go.uber.org/zap"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/yaml.v3"
)

var log = logger.NewNamed("bundle-config")

type BundleConfig struct {
	Version            int                  `yaml:"version"`
	ExternalListenAddr []string             `yaml:"externalListenAddr"`
	ConfigID           string               `yaml:"configId"`
	NetworkID          string               `yaml:"networkId"`
	StoragePath        string               `yaml:"storagePath"`
	Accounts           BundleConfigAccounts `yaml:"accounts"`
	Nodes              BundleConfigNodes    `yaml:"nodes"`
}

type BundleConfigAccounts struct {
	Coordinator accountservice.Config `yaml:"coordinator"`
	Consensus   accountservice.Config `yaml:"consensus"`
	Tree        accountservice.Config `yaml:"tree"`
	File        accountservice.Config `yaml:"file"`
}

type BundleConfigNodes struct {
	Coordinator BundleConfigNodeCoordinator `yaml:"coordinator"`
	Consensus   BundleConfigNodeConsensus   `yaml:"consensus"`
	Tree        BundleConfigNodeTree        `yaml:"tree"`
	File        BundleConfigNodeFile        `yaml:"file"`
}

type BundleConfigNodeShared struct {
	ListenTCPAddr string `yaml:"localTcpAddr"`
	ListenUDPAddr string `yaml:"localUdpAddr"`
}

type BundleConfigNodeCoordinator struct {
	BundleConfigNodeShared `yaml:",inline"`
	MongoConnect           string `yaml:"mongoConnect"`
	MongoDatabase          string `yaml:"mongoDatabase"`
}

type BundleConfigNodeConsensus struct {
	BundleConfigNodeShared `yaml:",inline"`
	MongoConnect           string `yaml:"mongoConnect"`
	MongoDatabase          string `yaml:"mongoDatabase"`
	MongoLogCollection     string `yaml:"mongoLogCollection"`
}

type BundleConfigNodeTree struct {
	BundleConfigNodeShared `yaml:",inline"`
}

type BundleConfigNodeFile struct {
	BundleConfigNodeShared `yaml:",inline"`
	RedisURL               string `yaml:"redisUrl"`
}

func BundleCfg(cfgPath string) *BundleConfig {
	if _, err := os.Stat(cfgPath); err == nil {
		log.Info("loaded existing config")
		return loadConfig(cfgPath)
	}

	createdCfg := createAndWriteConfig(cfgPath)
	log.Info("created new config")

	return createdCfg
}

func loadConfig(cfgPath string) *BundleConfig {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Panic("can't read config file", zap.Error(err))
	}

	var cfg BundleConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Panic("can't unmarshal config", zap.Error(err))
	}

	return &cfg
}

func createAndWriteConfig(cfgPath string) *BundleConfig {
	createdCfg := createBundleConfig()

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

// createConfig creates a new configuration for any-bundle that contain all info base of which internal services are created
// Base on https://tech.anytype.io/any-sync/configuration?id=common-nodes-configuration-options
func createBundleConfig() *BundleConfig {
	cfgId := bson.NewObjectId().Hex()

	netKey, _, err := crypto.GenerateRandomEd25519KeyPair()
	if err != nil {
		log.Panic("can't generate ed25519 key", zap.Error(err))
	}

	netId := netKey.GetPublic().Network()

	cfg := &BundleConfig{
		ExternalListenAddr: []string{
			"192.168.100.3",
		},
		Version:     1,
		ConfigID:    cfgId,
		NetworkID:   netId,
		StoragePath: "./data/storage",
		Accounts: BundleConfigAccounts{
			Coordinator: generateAccCfg(),
			Consensus:   generateAccCfg(),
			Tree:        generateAccCfg(),
			File:        generateAccCfg(),
		},
		Nodes: BundleConfigNodes{
			Coordinator: BundleConfigNodeCoordinator{
				BundleConfigNodeShared: BundleConfigNodeShared{
					ListenTCPAddr: "0.0.0.0:33010",
					ListenUDPAddr: "0.0.0.0:33011",
				},
				MongoConnect:  "mongodb://127.0.0.1:27017/",
				MongoDatabase: "coordinator",
			},
			Consensus: BundleConfigNodeConsensus{
				BundleConfigNodeShared: BundleConfigNodeShared{
					ListenTCPAddr: "0.0.0.0:33020",
					ListenUDPAddr: "0.0.0.0:33021",
				},
				MongoConnect:       "mongodb://127.0.0.1:27017/?w=majority",
				MongoDatabase:      "consensus",
				MongoLogCollection: "log",
			},
			Tree: BundleConfigNodeTree{
				BundleConfigNodeShared{
					ListenTCPAddr: "0.0.0.0:33030",
					ListenUDPAddr: "0.0.0.0:33031",
				},
			},
			File: BundleConfigNodeFile{
				BundleConfigNodeShared: BundleConfigNodeShared{
					ListenTCPAddr: "0.0.0.0:33040",
					ListenUDPAddr: "0.0.0.0:33041",
				},
				RedisURL: "redis://127.0.0.1:6379?dial_timeout=3&read_timeout=6s",
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

func generateAccCfg() accountservice.Config {
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

// Package config build on top of https://github.com/anyproto/any-sync-tools/tree/72b131eaf4d6dc299ecf87dad60648e68054b35a/anyconf.
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

const (
	// MinSupportedBundleFormat is the oldest config format version this binary can load.
	MinSupportedBundleFormat = 1
	// CurrentBundleFormat is the config format version this binary creates.
	CurrentBundleFormat = 1
)

type Config struct {
	BundleVersion string                `yaml:"bundleVersion"`
	BundleFormat  int                   `yaml:"bundleFormat"`
	ExternalAddr  []string              `yaml:"externalAddr"`
	ConfigID      string                `yaml:"configId"`
	NetworkID     string                `yaml:"networkId"`
	StoragePath   string                `yaml:"storagePath"`
	Account       accountservice.Config `yaml:"account"`
	Network       NetworkConfig         `yaml:"network"`
	Coordinator   CoordinatorConfig     `yaml:"coordinator"`
	Consensus     ConsensusConfig       `yaml:"consensus"`
	FileNode      FileNodeConfig        `yaml:"filenode"`
}

type NetworkConfig struct {
	ListenTCPAddr string `yaml:"listenTCPAddr"`
	ListenUDPAddr string `yaml:"listenUDPAddr"`
}

type CoordinatorConfig struct {
	MongoConnect  string `yaml:"mongoConnect"`
	MongoDatabase string `yaml:"mongoDatabase"`
}

type ConsensusConfig struct {
	MongoConnect  string `yaml:"mongoConnect"`
	MongoDatabase string `yaml:"mongoDatabase"`
}

type FileNodeConfig struct {
	RedisConnect string    `yaml:"redisConnect"`
	S3           *S3Config `yaml:"s3,omitempty"` // Optional: if present, use S3 storage instead of BadgerDB
}

// S3Config configures S3 storage backend for the filenode.
// When this config is present in FileNodeConfig, S3 storage is automatically enabled.
type S3Config struct {
	Region         string         `yaml:"region"`                   // AWS region (e.g., "us-east-1")
	BlockBucket    string         `yaml:"blockBucket"`              // S3 bucket for file blocks/data
	IndexBucket    string         `yaml:"indexBucket"`              // S3 bucket for metadata index
	Endpoint       string         `yaml:"endpoint,omitempty"`       // Optional: Custom endpoint for S3-compatible services (MinIO, etc.)
	Profile        string         `yaml:"profile,omitempty"`        // Optional: AWS profile name from ~/.aws/credentials
	ForcePathStyle bool           `yaml:"forcePathStyle,omitempty"` // Optional: Use path-style URLs for S3-compatible services
	Credentials    *S3Credentials `yaml:"credentials,omitempty"`    // Optional: Static credentials (not recommended for production)
}

// S3Credentials holds static AWS credentials.
type S3Credentials struct {
	AccessKey string `yaml:"accessKey"` // AWS Access Key ID
	SecretKey string `yaml:"secretKey"` // AWS Secret Access Key
}

func Load(cfgPath string) *Config {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Panic("can't read config file", zap.Error(err))
	}

	var cfg Config
	if errUnmarshal := yaml.Unmarshal(data, &cfg); errUnmarshal != nil {
		log.Panic("can't unmarshal config", zap.Error(errUnmarshal))
	}

	// Validate bundleFormat version
	if cfg.BundleFormat < MinSupportedBundleFormat {
		log.Panic("config format too old, please migrate your configuration",
			zap.Int("format", cfg.BundleFormat),
			zap.Int("min_supported", MinSupportedBundleFormat),
			zap.String("path", cfgPath))
	}
	if cfg.BundleFormat > CurrentBundleFormat {
		log.Panic("config format too new, please upgrade the binary",
			zap.Int("format", cfg.BundleFormat),
			zap.Int("current", CurrentBundleFormat),
			zap.String("path", cfgPath))
	}

	return &cfg
}

type CreateOptions struct {
	CfgPath       string
	StorePath     string
	MongoURI      string
	RedisURI      string
	ExternalAddrs []string

	// S3 storage (optional - if not set, BadgerDB is used)
	S3Region         string
	S3BlockBucket    string
	S3IndexBucket    string
	S3Endpoint       string
	S3Profile        string
	S3AccessKey      string
	S3SecretKey      string
	S3ForcePathStyle bool
}

func CreateWrite(cfg *CreateOptions) *Config {
	createdCfg := newBundleConfig(cfg)

	createCfgYaml, err := yaml.Marshal(createdCfg)
	if err != nil {
		log.Panic("can't marshal config", zap.Error(err))
	}

	if errMkdir := os.MkdirAll(filepath.Dir(cfg.CfgPath), 0o750); errMkdir != nil {
		log.Panic("can't create config directory", zap.Error(errMkdir))
	}

	if errWrite := os.WriteFile(cfg.CfgPath, createCfgYaml, 0o600); errWrite != nil {
		log.Panic("can't write config file", zap.Error(errWrite))
	}

	return createdCfg
}

// newBundleConfig creates a new configuration for any-bundle that contain all info base of which internal services are created
// Base on https://tech.anytype.io/any-sync/configuration?id=common-nodes-configuration-options.
// But docs above are not accurate, so I used also source code as reference...
func newBundleConfig(cfg *CreateOptions) *Config {
	cfgID := bson.NewObjectId().Hex()

	netKey, _, err := crypto.GenerateRandomEd25519KeyPair()
	if err != nil {
		log.Panic("can't generate ed25519 key for network", zap.Error(err))
	}

	netID := netKey.GetPublic().Network()

	// Parse MongoDB URI and add w=majority if not already present.
	// Base on Anytype dockercompose version.
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
		ConfigID:      cfgID,
		NetworkID:     netID,
		StoragePath:   cfg.StorePath,
		Account:       newAcc(netKey),
		Network: NetworkConfig{
			ListenTCPAddr: "0.0.0.0:33010",
			ListenUDPAddr: "0.0.0.0:33020",
		},
		Coordinator: CoordinatorConfig{
			MongoConnect:  cfg.MongoURI,
			MongoDatabase: "coordinator",
		},
		Consensus: ConsensusConfig{
			MongoConnect:  mongoConsensusURI.String(),
			MongoDatabase: "consensus",
		},
		FileNode: FileNodeConfig{
			RedisConnect: cfg.RedisURI,
		},
	}

	// Configure S3 storage if S3 flags are provided
	if cfg.S3BlockBucket != "" || cfg.S3Region != "" || cfg.S3IndexBucket != "" {
		// Validate: all three required S3 fields must be present
		if cfg.S3Region == "" || cfg.S3BlockBucket == "" || cfg.S3IndexBucket == "" {
			log.Panic(
				"S3 storage requires all three fields: --initial-s3-region, --initial-s3-block-bucket, and --initial-s3-index-bucket",
			)
		}

		defaultCfg.FileNode.S3 = &S3Config{
			Region:         cfg.S3Region,
			BlockBucket:    cfg.S3BlockBucket,
			IndexBucket:    cfg.S3IndexBucket,
			Endpoint:       cfg.S3Endpoint,
			Profile:        cfg.S3Profile,
			ForcePathStyle: cfg.S3ForcePathStyle,
		}

		// Add static credentials if provided
		if cfg.S3AccessKey != "" && cfg.S3SecretKey != "" {
			defaultCfg.FileNode.S3.Credentials = &S3Credentials{
				AccessKey: cfg.S3AccessKey,
				SecretKey: cfg.S3SecretKey,
			}
		}

		log.Info("S3 storage configured",
			zap.String("region", cfg.S3Region),
			zap.String("blockBucket", cfg.S3BlockBucket),
			zap.String("indexBucket", cfg.S3IndexBucket))
	}

	return defaultCfg
}

func newAcc(netKey crypto.PrivKey) accountservice.Config {
	signKey, _, err := crypto.GenerateRandomEd25519KeyPair()
	if err != nil {
		log.Panic("can't generate ed25519 key for account", zap.Error(err))
	}

	encPeerSignKey, err := crypto.EncodeKeyToString(signKey)
	if err != nil {
		log.Panic("can't encode key to string", zap.Error(err))
	}

	// Base on docs https://tech.anytype.io/any-sync/configuration?id=common-nodes-configuration-options.
	// "Signing key of coordinator is private key of the network and sync and file nodes use their peerKey"
	// Because we create account for bundle, we reuse only logic here of configurator.
	privNetKey, err := crypto.EncodeKeyToString(netKey)
	if err != nil {
		log.Panic("can't encode network key to string", zap.Error(err))
	}

	return accountservice.Config{
		PeerId:     signKey.GetPublic().PeerId(), // public key
		PeerKey:    encPeerSignKey,               // private key
		SigningKey: privNetKey,                   // private key
	}
}

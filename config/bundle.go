// Package config build on top of https://github.com/anyproto/any-sync-tools/tree/72b131eaf4d6dc299ecf87dad60648e68054b35a/anyconf.
package config

import (
	"errors"
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

// S3Config configures S3-compatible storage backend for the filenode.
// Supports AWS S3, MinIO, Cloudflare R2, Backblaze B2, etc.
// Credentials are provided via environment variables: AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY.
//
// TODO(bundleFormat:v2): Make Region a required field instead of defaulting to "us-east-1".
// This will force explicit configuration and avoid silent auth failures with non-default regions.
type S3Config struct {
	Bucket         string `yaml:"bucket"`                   // S3 bucket name (required)
	Endpoint       string `yaml:"endpoint"`                 // S3 endpoint URL (required, e.g., "https://s3.us-east-1.amazonaws.com")
	Region         string `yaml:"region,omitempty"`         // S3 region for SigV4 signing (default: "us-east-1")
	ForcePathStyle bool   `yaml:"forcePathStyle,omitempty"` // Use path-style URLs (required for MinIO)
}

// S3 configuration errors.
var (
	ErrS3BucketRequired   = errors.New("S3 bucket name is required (--initial-s3-bucket)")
	ErrS3EndpointRequired = errors.New("S3 endpoint URL is required (--initial-s3-endpoint)")
)

// validateS3Config validates S3 configuration and returns the S3Config if valid.
// It checks that both bucket and endpoint are provided, and warns if credentials are missing.
// Region is optional and defaults to "us-east-1" at conversion time if not provided.
func validateS3Config(bucket, endpoint, region string, forcePathStyle bool) (*S3Config, error) {
	if bucket == "" {
		return nil, ErrS3BucketRequired
	}
	if endpoint == "" {
		return nil, ErrS3EndpointRequired
	}

	// Check for credentials and warn if missing
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if accessKey == "" || secretKey == "" {
		log.Warn("S3 credentials not set - authentication may fail at runtime",
			zap.Bool("AWS_ACCESS_KEY_ID_set", accessKey != ""),
			zap.Bool("AWS_SECRET_ACCESS_KEY_set", secretKey != ""))
	}

	return &S3Config{
		Bucket:         bucket,
		Endpoint:       endpoint,
		Region:         region,
		ForcePathStyle: forcePathStyle,
	}, nil
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
	// Credentials via env vars: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
	S3Bucket         string
	S3Endpoint       string
	S3Region         string // Optional, defaults to "us-east-1" if empty
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
	if cfg.S3Bucket != "" || cfg.S3Endpoint != "" {
		s3Cfg, s3Err := validateS3Config(cfg.S3Bucket, cfg.S3Endpoint, cfg.S3Region, cfg.S3ForcePathStyle)
		if s3Err != nil {
			log.Panic("invalid S3 configuration", zap.Error(s3Err))
		}
		defaultCfg.FileNode.S3 = s3Cfg

		log.Info("S3 storage configured",
			zap.String("bucket", cfg.S3Bucket),
			zap.String("endpoint", cfg.S3Endpoint),
			zap.String("region", cfg.S3Region))
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

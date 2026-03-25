package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anyproto/any-sync/accountservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ValidFormat(t *testing.T) {
	// Create a temporary valid config with bundleFormat=1
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "valid-config.yml")

	validConfig := `bundleVersion: "0.13.0"
bundleFormat: 1
externalAddr:
  - "192.168.1.100"
configId: "test-config-id"
networkId: "test-network-id"
storagePath: "./data/storage"
account:
  peerId: "test-peer-id"
  peerKey: "test-peer-key"
  signingKey: "test-signing-key"
network:
  listenTCPAddr: "0.0.0.0:33010"
  listenUDPAddr: "0.0.0.0:33020"
coordinator:
  mongoConnect: "mongodb://localhost:27017/"
  mongoDatabase: "coordinator"
consensus:
  mongoConnect: "mongodb://localhost:27017/?w=majority"
  mongoDatabase: "consensus"
filenode:
  redisConnect: "redis://localhost:6379/"
`

	err := os.WriteFile(cfgPath, []byte(validConfig), 0o600)
	require.NoError(t, err)

	// Should load successfully
	cfg := Load(cfgPath)
	assert.NotNil(t, cfg)
	assert.Equal(t, 1, cfg.BundleFormat)
	assert.Equal(t, "0.13.0", cfg.BundleVersion)
}

func TestLoad_MissingFormat(t *testing.T) {
	// Create a config without bundleFormat field (defaults to 0)
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "missing-format.yml")

	configWithoutFormat := `bundleVersion: "0.13.0"
externalAddr:
  - "192.168.1.100"
configId: "test-config-id"
networkId: "test-network-id"
storagePath: "./data/storage"
account:
  peerId: "test-peer-id"
  peerKey: "test-peer-key"
  signingKey: "test-signing-key"
network:
  listenTCPAddr: "0.0.0.0:33010"
  listenUDPAddr: "0.0.0.0:33020"
coordinator:
  mongoConnect: "mongodb://localhost:27017/"
  mongoDatabase: "coordinator"
consensus:
  mongoConnect: "mongodb://localhost:27017/?w=majority"
  mongoDatabase: "consensus"
filenode:
  redisConnect: "redis://localhost:6379/"
`

	err := os.WriteFile(cfgPath, []byte(configWithoutFormat), 0o600)
	require.NoError(t, err)

	// Should panic with "config format too old"
	assert.Panics(t, func() {
		Load(cfgPath)
	})
}

func TestLoad_FormatZero(t *testing.T) {
	// Create a config with explicit bundleFormat=0
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "format-zero.yml")

	configFormatZero := `bundleVersion: "0.12.0"
bundleFormat: 0
externalAddr:
  - "192.168.1.100"
configId: "test-config-id"
networkId: "test-network-id"
storagePath: "./data/storage"
account:
  peerId: "test-peer-id"
  peerKey: "test-peer-key"
  signingKey: "test-signing-key"
network:
  listenTCPAddr: "0.0.0.0:33010"
  listenUDPAddr: "0.0.0.0:33020"
coordinator:
  mongoConnect: "mongodb://localhost:27017/"
  mongoDatabase: "coordinator"
consensus:
  mongoConnect: "mongodb://localhost:27017/?w=majority"
  mongoDatabase: "consensus"
filenode:
  redisConnect: "redis://localhost:6379/"
`

	err := os.WriteFile(cfgPath, []byte(configFormatZero), 0o600)
	require.NoError(t, err)

	// Should panic with "config format too old"
	assert.Panics(t, func() {
		Load(cfgPath)
	})
}

func TestLoad_FutureFormat(t *testing.T) {
	// Create a config with bundleFormat=2 (future version)
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "future-format.yml")

	futureConfig := `bundleVersion: "1.0.0"
bundleFormat: 2
externalAddr:
  - "192.168.1.100"
configId: "test-config-id"
networkId: "test-network-id"
storagePath: "./data/storage"
account:
  peerId: "test-peer-id"
  peerKey: "test-peer-key"
  signingKey: "test-signing-key"
network:
  listenTCPAddr: "0.0.0.0:33010"
  listenUDPAddr: "0.0.0.0:33020"
coordinator:
  mongoConnect: "mongodb://localhost:27017/"
  mongoDatabase: "coordinator"
consensus:
  mongoConnect: "mongodb://localhost:27017/?w=majority"
  mongoDatabase: "consensus"
filenode:
  redisConnect: "redis://localhost:6379/"
`

	err := os.WriteFile(cfgPath, []byte(futureConfig), 0o600)
	require.NoError(t, err)

	// Should panic with "config format too new"
	assert.Panics(t, func() {
		Load(cfgPath)
	})
}

func TestLoad_NegativeFormat(t *testing.T) {
	// Create a config with bundleFormat=-1 (invalid)
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "negative-format.yml")

	negativeConfig := `bundleVersion: "0.13.0"
bundleFormat: -1
externalAddr:
  - "192.168.1.100"
configId: "test-config-id"
networkId: "test-network-id"
storagePath: "./data/storage"
account:
  peerId: "test-peer-id"
  peerKey: "test-peer-key"
  signingKey: "test-signing-key"
network:
  listenTCPAddr: "0.0.0.0:33010"
  listenUDPAddr: "0.0.0.0:33020"
coordinator:
  mongoConnect: "mongodb://localhost:27017/"
  mongoDatabase: "coordinator"
consensus:
  mongoConnect: "mongodb://localhost:27017/?w=majority"
  mongoDatabase: "consensus"
filenode:
  redisConnect: "redis://localhost:6379/"
`

	err := os.WriteFile(cfgPath, []byte(negativeConfig), 0o600)
	require.NoError(t, err)

	// Should panic with "config format too old"
	assert.Panics(t, func() {
		Load(cfgPath)
	})
}

func TestCreateWrite_SetsBundleFormat(t *testing.T) {
	// Verify that CreateWrite sets bundleFormat to CurrentBundleFormat
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "created-config.yml")

	opts := &CreateOptions{
		CfgPath:       cfgPath,
		StorePath:     filepath.Join(tmpDir, "storage"),
		MongoURI:      "mongodb://localhost:27017/",
		RedisURI:      "redis://localhost:6379/",
		ExternalAddrs: []string{"192.168.1.100"},
	}

	cfg := CreateWrite(opts)
	assert.Equal(t, CurrentBundleFormat, cfg.BundleFormat)

	// Verify the written file can be loaded back
	loadedCfg := Load(cfgPath)
	assert.Equal(t, CurrentBundleFormat, loadedCfg.BundleFormat)
}

// S3 Configuration Tests

func TestValidateS3Config_Valid(t *testing.T) {
	// Set credentials for this test
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	cfg, err := validateS3Config("my-bucket", "https://s3.amazonaws.com", "", false)
	require.NoError(t, err)
	assert.Equal(t, "my-bucket", cfg.Bucket)
	assert.Equal(t, "https://s3.amazonaws.com", cfg.Endpoint)
	assert.Empty(t, cfg.Region, "Region should be empty when not provided")
	assert.False(t, cfg.ForcePathStyle)
}

func TestValidateS3Config_WithForcePathStyle(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	cfg, err := validateS3Config("my-bucket", "http://minio:9000", "", true)
	require.NoError(t, err)
	assert.True(t, cfg.ForcePathStyle)
}

func TestValidateS3Config_WithRegion(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	cfg, err := validateS3Config("my-bucket", "http://minio:9000", "sz-hq", true)
	require.NoError(t, err)
	assert.Equal(t, "sz-hq", cfg.Region)
}

func TestValidateS3Config_MissingBucket(t *testing.T) {
	cfg, err := validateS3Config("", "https://s3.amazonaws.com", "", false)
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, ErrS3BucketRequired)
}

func TestValidateS3Config_MissingEndpoint(t *testing.T) {
	cfg, err := validateS3Config("my-bucket", "", "", false)
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, ErrS3EndpointRequired)
}

func TestValidateS3Config_MissingBoth(t *testing.T) {
	// When both are missing, bucket error should come first
	cfg, err := validateS3Config("", "", "", false)
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, ErrS3BucketRequired)
}

func TestValidateS3Config_MissingCredentials(t *testing.T) {
	// Ensure credentials are not set
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	// Should still succeed but with a warning (tested via logs)
	cfg, err := validateS3Config("my-bucket", "https://s3.amazonaws.com", "", false)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestValidateS3Config_PartialCredentials(t *testing.T) {
	// Only access key set
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	// Should still succeed but with a warning
	cfg, err := validateS3Config("my-bucket", "https://s3.amazonaws.com", "", false)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestCreateWrite_WithS3Config(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "s3-config.yml")

	opts := &CreateOptions{
		CfgPath:          cfgPath,
		StorePath:        filepath.Join(tmpDir, "storage"),
		MongoURI:         "mongodb://localhost:27017/",
		RedisURI:         "redis://localhost:6379/",
		ExternalAddrs:    []string{"192.168.1.100"},
		S3Bucket:         "test-bucket",
		S3Endpoint:       "https://s3.amazonaws.com",
		S3ForcePathStyle: true,
	}

	cfg := CreateWrite(opts)
	require.NotNil(t, cfg.FileNode.S3)
	assert.Equal(t, "test-bucket", cfg.FileNode.S3.Bucket)
	assert.Equal(t, "https://s3.amazonaws.com", cfg.FileNode.S3.Endpoint)
	assert.True(t, cfg.FileNode.S3.ForcePathStyle)

	// Verify the config can be loaded back with S3 settings
	loadedCfg := Load(cfgPath)
	require.NotNil(t, loadedCfg.FileNode.S3)
	assert.Equal(t, "test-bucket", loadedCfg.FileNode.S3.Bucket)
}

func TestCreateWrite_WithoutS3Config(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "no-s3-config.yml")

	opts := &CreateOptions{
		CfgPath:       cfgPath,
		StorePath:     filepath.Join(tmpDir, "storage"),
		MongoURI:      "mongodb://localhost:27017/",
		RedisURI:      "redis://localhost:6379/",
		ExternalAddrs: []string{"192.168.1.100"},
		// No S3 options
	}

	cfg := CreateWrite(opts)
	assert.Nil(t, cfg.FileNode.S3, "S3 config should be nil when not configured")
}

func TestCreateWrite_S3MissingEndpoint_Panics(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "invalid-s3-config.yml")

	opts := &CreateOptions{
		CfgPath:       cfgPath,
		StorePath:     filepath.Join(tmpDir, "storage"),
		MongoURI:      "mongodb://localhost:27017/",
		RedisURI:      "redis://localhost:6379/",
		ExternalAddrs: []string{"192.168.1.100"},
		S3Bucket:      "test-bucket",
		// Missing S3Endpoint
	}

	assert.Panics(t, func() {
		CreateWrite(opts)
	})
}

func TestCreateWrite_S3MissingBucket_Panics(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "invalid-s3-config.yml")

	opts := &CreateOptions{
		CfgPath:       cfgPath,
		StorePath:     filepath.Join(tmpDir, "storage"),
		MongoURI:      "mongodb://localhost:27017/",
		RedisURI:      "redis://localhost:6379/",
		ExternalAddrs: []string{"192.168.1.100"},
		S3Endpoint:    "https://s3.amazonaws.com",
		// Missing S3Bucket
	}

	assert.Panics(t, func() {
		CreateWrite(opts)
	})
}

func TestLoad_WithS3Config(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "s3-config.yml")

	configWithS3 := `bundleVersion: "1.0.0"
bundleFormat: 1
externalAddr:
  - "192.168.1.100"
configId: "test-config-id"
networkId: "test-network-id"
storagePath: "./data/storage"
account:
  peerId: "test-peer-id"
  peerKey: "test-peer-key"
  signingKey: "test-signing-key"
network:
  listenTCPAddr: "0.0.0.0:33010"
  listenUDPAddr: "0.0.0.0:33020"
coordinator:
  mongoConnect: "mongodb://localhost:27017/"
  mongoDatabase: "coordinator"
consensus:
  mongoConnect: "mongodb://localhost:27017/?w=majority"
  mongoDatabase: "consensus"
filenode:
  redisConnect: "redis://localhost:6379/"
  s3:
    bucket: "my-bucket"
    endpoint: "https://s3.amazonaws.com"
    forcePathStyle: true
`

	err := os.WriteFile(cfgPath, []byte(configWithS3), 0o600)
	require.NoError(t, err)

	cfg := Load(cfgPath)
	require.NotNil(t, cfg.FileNode.S3)
	assert.Equal(t, "my-bucket", cfg.FileNode.S3.Bucket)
	assert.Equal(t, "https://s3.amazonaws.com", cfg.FileNode.S3.Endpoint)
	assert.True(t, cfg.FileNode.S3.ForcePathStyle)
}

// Filenode Default Limit Tests

func TestCreateWrite_WithFilenodeDefaultLimit(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "limit-config.yml")

	const tenGiB = 10 * 1024 * 1024 * 1024 // 10 GiB

	opts := &CreateOptions{
		CfgPath:              cfgPath,
		StorePath:            filepath.Join(tmpDir, "storage"),
		MongoURI:             "mongodb://localhost:27017/",
		RedisURI:             "redis://localhost:6379/",
		ExternalAddrs:        []string{"192.168.1.100"},
		FilenodeDefaultLimit: tenGiB,
	}

	cfg := CreateWrite(opts)
	assert.Equal(t, uint64(tenGiB), cfg.FileNode.DefaultLimit)

	// Verify it persists and loads back correctly
	loadedCfg := Load(cfgPath)
	assert.Equal(t, uint64(tenGiB), loadedCfg.FileNode.DefaultLimit)
}

func TestCreateWrite_WithoutFilenodeDefaultLimit(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "no-limit-config.yml")

	const oneTiB = 1024 * 1024 * 1024 * 1024 // 1 TiB

	opts := &CreateOptions{
		CfgPath:       cfgPath,
		StorePath:     filepath.Join(tmpDir, "storage"),
		MongoURI:      "mongodb://localhost:27017/",
		RedisURI:      "redis://localhost:6379/",
		ExternalAddrs: []string{"192.168.1.100"},
		// FilenodeDefaultLimit not set (zero value)
	}

	cfg := CreateWrite(opts)
	assert.Equal(t, uint64(oneTiB), cfg.FileNode.DefaultLimit,
		"DefaultLimit should be 1 TiB when not configured (written to config file)")
}

func TestLoad_WithFilenodeDefaultLimit(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "limit-config.yml")

	configWithLimit := `bundleVersion: "1.0.0"
bundleFormat: 1
externalAddr:
  - "192.168.1.100"
configId: "test-config-id"
networkId: "test-network-id"
storagePath: "./data/storage"
account:
  peerId: "test-peer-id"
  peerKey: "test-peer-key"
  signingKey: "test-signing-key"
network:
  listenTCPAddr: "0.0.0.0:33010"
  listenUDPAddr: "0.0.0.0:33020"
coordinator:
  mongoConnect: "mongodb://localhost:27017/"
  mongoDatabase: "coordinator"
consensus:
  mongoConnect: "mongodb://localhost:27017/?w=majority"
  mongoDatabase: "consensus"
filenode:
  redisConnect: "redis://localhost:6379/"
  defaultLimit: 10737418240
`

	err := os.WriteFile(cfgPath, []byte(configWithLimit), 0o600)
	require.NoError(t, err)

	cfg := Load(cfgPath)
	assert.Equal(t, uint64(10737418240), cfg.FileNode.DefaultLimit)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(cfg *Config)
		wantErr string
	}{
		{
			name: "valid config",
		},
		{
			name: "missing external address",
			mutate: func(cfg *Config) {
				cfg.ExternalAddr = nil
			},
			wantErr: "externalAddr must contain at least one address",
		},
		{
			name: "blank external address",
			mutate: func(cfg *Config) {
				cfg.ExternalAddr = []string{" "}
			},
			wantErr: "externalAddr[0] is required",
		},
		{
			name: "invalid tcp listen address",
			mutate: func(cfg *Config) {
				cfg.Network.ListenTCPAddr = "33010"
			},
			wantErr: "network.listenTCPAddr must be in host:port format",
		},
		{
			name: "invalid redis uri",
			mutate: func(cfg *Config) {
				cfg.FileNode.RedisConnect = "localhost:6379"
			},
			wantErr: "filenode.redisConnect must include a host",
		},
		{
			name: "invalid s3 endpoint",
			mutate: func(cfg *Config) {
				cfg.FileNode.S3 = &S3Config{
					Bucket:   "bucket",
					Endpoint: "minio:9000",
				}
			},
			wantErr: "filenode.s3.endpoint must include a host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validTestConfig()
			if tt.mutate != nil {
				tt.mutate(cfg)
			}

			err := cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func validTestConfig() *Config {
	return &Config{
		BundleVersion: "1.0.0",
		BundleFormat:  1,
		ExternalAddr:  []string{"example.local"},
		ConfigID:      "test-config-id",
		NetworkID:     "test-network-id",
		StoragePath:   "./data/storage",
		Account: accountservice.Config{
			PeerId:     "test-peer-id",
			PeerKey:    "test-peer-key",
			SigningKey: "test-signing-key",
		},
		Network: NetworkConfig{
			ListenTCPAddr: "0.0.0.0:33010",
			ListenUDPAddr: "0.0.0.0:33020",
		},
		Coordinator: CoordinatorConfig{
			MongoConnect:  "mongodb://localhost:27017/",
			MongoDatabase: "coordinator",
		},
		Consensus: ConsensusConfig{
			MongoConnect:  "mongodb://localhost:27017/?w=majority",
			MongoDatabase: "consensus",
		},
		FileNode: FileNodeConfig{
			RedisConnect: "redis://localhost:6379/",
		},
	}
}

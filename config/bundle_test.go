package config

import (
	"os"
	"path/filepath"
	"testing"

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

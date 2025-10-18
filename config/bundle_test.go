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
  listenUDPAddr: "0.0.0.0:33010"
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
  listenUDPAddr: "0.0.0.0:33010"
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
  listenUDPAddr: "0.0.0.0:33010"
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
  listenUDPAddr: "0.0.0.0:33010"
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
  listenUDPAddr: "0.0.0.0:33010"
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

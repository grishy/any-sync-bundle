package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anyproto/any-sync/accountservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Loading has three format states: supported, older than supported, and newer
// than this binary. Values within either rejected range have identical meaning.
func TestLoadBundleFormat(t *testing.T) {
	tests := []struct {
		name        string
		format      int
		shouldPanic bool
	}{
		{name: "current", format: CurrentBundleFormat},
		{name: "below minimum", format: MinSupportedBundleFormat - 1, shouldPanic: true},
		{name: "newer than binary", format: CurrentBundleFormat + 1, shouldPanic: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validTestConfig()
			cfg.BundleFormat = test.format

			data, err := yaml.Marshal(cfg)
			require.NoError(t, err)

			cfgPath := filepath.Join(t.TempDir(), "bundle.yml")
			require.NoError(t, os.WriteFile(cfgPath, data, 0o600))

			if test.shouldPanic {
				assert.Panics(t, func() {
					Load(cfgPath)
				})
				return
			}

			loaded := Load(cfgPath)
			assert.Equal(t, CurrentBundleFormat, loaded.BundleFormat)
		})
	}
}

// Release fixtures exercise the reader independently of the current writer.
// Together with the creation round trips below, they cover every 1.x config
// shape that changed while bundle format 1 remained supported.
func TestLoadV1Compatibility(t *testing.T) {
	t.Run("v1.0 local storage", func(t *testing.T) {
		cfg := Load("testdata/bundle-v1.0.yml")

		assert.Equal(t, 1, cfg.BundleFormat)
		assert.Equal(t, "1.0.0", cfg.BundleVersion)
		assert.Equal(t, []string{"192.168.1.100"}, cfg.ExternalAddr)
		assert.Equal(t, "test-config-id", cfg.ConfigID)
		assert.Equal(t, "test-network-id", cfg.NetworkID)
		assert.Equal(t, "./data/storage", cfg.StoragePath)
		assert.Equal(t, "test-peer-id", cfg.Account.PeerId)
		assert.Equal(t, "test-peer-key", cfg.Account.PeerKey)
		assert.Equal(t, "test-signing-key", cfg.Account.SigningKey)
		assert.Equal(t, "0.0.0.0:33010", cfg.Network.ListenTCPAddr)
		assert.Equal(t, "0.0.0.0:33020", cfg.Network.ListenUDPAddr)
		assert.Equal(t, "mongodb://localhost:27017/", cfg.Coordinator.MongoConnect)
		assert.Equal(t, "coordinator", cfg.Coordinator.MongoDatabase)
		assert.Equal(t, "mongodb://localhost:27017/?w=majority", cfg.Consensus.MongoConnect)
		assert.Equal(t, "consensus", cfg.Consensus.MongoDatabase)
		assert.Equal(t, "redis://localhost:6379/", cfg.FileNode.RedisConnect)
		assert.Nil(t, cfg.FileNode.S3)
		assert.Equal(t, uint64(oneTiB), cfg.NodeConfigs().Filenode.DefaultLimit)
	})

	t.Run("v1.2 S3 storage", func(t *testing.T) {
		cfg := Load("testdata/bundle-v1.2-s3.yml")

		assert.Equal(t, 1, cfg.BundleFormat)
		assert.Equal(t, "1.2.0", cfg.BundleVersion)
		require.NotNil(t, cfg.FileNode.S3)
		assert.Equal(t, "my-bucket", cfg.FileNode.S3.Bucket)
		assert.Equal(t, "https://s3.amazonaws.com", cfg.FileNode.S3.Endpoint)
		assert.True(t, cfg.FileNode.S3.ForcePathStyle)

		filenodeCfg := cfg.NodeConfigs().Filenode
		assert.Equal(t, uint64(oneTiB), filenodeCfg.DefaultLimit)
		assert.Equal(t, "us-east-1", filenodeCfg.S3Store.Region)
	})

	t.Run("v1.3 explicit storage settings", func(t *testing.T) {
		const tenGiB = 10 * 1024 * 1024 * 1024

		cfg := Load("testdata/bundle-v1.3-s3.yml")

		assert.Equal(t, 1, cfg.BundleFormat)
		assert.Equal(t, "1.3.0", cfg.BundleVersion)
		assert.Equal(t, uint64(tenGiB), cfg.FileNode.DefaultLimit)
		require.NotNil(t, cfg.FileNode.S3)
		assert.Equal(t, "eu-central-1", cfg.FileNode.S3.Region)

		filenodeCfg := cfg.NodeConfigs().Filenode
		assert.Equal(t, uint64(tenGiB), filenodeCfg.DefaultLimit)
		assert.Equal(t, "eu-central-1", filenodeCfg.S3Store.Region)
	})
}

// Loading treats the persisted configuration as operator-owned input. Invalid
// MongoDB syntax must stop startup instead of becoming different runtime state.
func TestLoadRejectsInvalidMongoURI(t *testing.T) {
	cfg := validTestConfig()
	cfg.Consensus.MongoConnect = "mongodb://localhost:27017?w=majority"

	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)

	cfgPath := filepath.Join(t.TempDir(), "bundle.yml")
	require.NoError(t, os.WriteFile(cfgPath, data, 0o600))

	assert.Panics(t, func() {
		Load(cfgPath)
	})
}

// Generated defaults are persisted, not reconstructed only in memory, so the
// configuration remains explicit and stable across restarts.
func TestCreateWriteRoundTrip(t *testing.T) {
	options := validCreateOptions(t)

	created := CreateWrite(options)
	loaded := Load(options.CfgPath)

	assert.Equal(t, CurrentBundleFormat, created.BundleFormat)
	assert.Equal(t, CurrentBundleFormat, loaded.BundleFormat)
	assert.Equal(t, uint64(oneTiB), created.FileNode.DefaultLimit)
	assert.Equal(t, uint64(oneTiB), loaded.FileNode.DefaultLimit)
	assert.Nil(t, created.FileNode.S3)
	assert.Nil(t, loaded.FileNode.S3)
}

// The coordinator receives the valid operator-provided URI unchanged. The
// bundle owns the derived consensus URI and therefore its required separator.
func TestCreateWritePreservesMongoURI(t *testing.T) {
	options := validCreateOptions(t)
	options.MongoURI = "mongodb://localhost:27017"

	cfg := CreateWrite(options)

	assert.Equal(t, options.MongoURI, cfg.Coordinator.MongoConnect)
	assert.Equal(t, "mongodb://localhost:27017/?w=majority", cfg.Consensus.MongoConnect)
}

func TestCreateWriteRejectsInvalidMongoURI(t *testing.T) {
	options := validCreateOptions(t)
	options.MongoURI = "mongodb://localhost:27017?replicaSet=rs0"

	assert.Panics(t, func() {
		CreateWrite(options)
	})

	_, err := os.Stat(options.CfgPath)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestValidateS3Config(t *testing.T) {
	tests := []struct {
		name            string
		bucket          string
		endpoint        string
		withCredentials bool
		wantErr         error
	}{
		{
			name:            "valid",
			bucket:          "my-bucket",
			endpoint:        "http://minio:9000",
			withCredentials: true,
		},
		{
			name:     "missing bucket",
			endpoint: "http://minio:9000",
			wantErr:  ErrS3BucketRequired,
		},
		{
			name:    "missing endpoint",
			bucket:  "my-bucket",
			wantErr: ErrS3EndpointRequired,
		},
		{
			name:     "credentials may come from another AWS provider",
			bucket:   "my-bucket",
			endpoint: "https://s3.amazonaws.com",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.withCredentials {
				t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
				t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
			} else {
				t.Setenv("AWS_ACCESS_KEY_ID", "")
				t.Setenv("AWS_SECRET_ACCESS_KEY", "")
			}

			cfg, err := validateS3Config(
				test.bucket,
				test.endpoint,
				"custom-region",
				true,
			)
			if test.wantErr != nil {
				assert.Nil(t, cfg)
				assert.ErrorIs(t, err, test.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)
			assert.Equal(t, test.bucket, cfg.Bucket)
			assert.Equal(t, test.endpoint, cfg.Endpoint)
			assert.Equal(t, "custom-region", cfg.Region)
			assert.True(t, cfg.ForcePathStyle)
		})
	}
}

func TestCreateWriteS3RoundTrip(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	options := validCreateOptions(t)
	options.S3Bucket = "test-bucket"
	options.S3Endpoint = "http://minio:9000"
	options.S3Region = "custom-region"
	options.S3ForcePathStyle = true

	created := CreateWrite(options)
	loaded := Load(options.CfgPath)

	for _, cfg := range []*Config{created, loaded} {
		require.NotNil(t, cfg.FileNode.S3)
		assert.Equal(t, "test-bucket", cfg.FileNode.S3.Bucket)
		assert.Equal(t, "http://minio:9000", cfg.FileNode.S3.Endpoint)
		assert.Equal(t, "custom-region", cfg.FileNode.S3.Region)
		assert.True(t, cfg.FileNode.S3.ForcePathStyle)
	}
}

func TestCreateWriteExplicitFilenodeLimit(t *testing.T) {
	const tenGiB = 10 * 1024 * 1024 * 1024

	options := validCreateOptions(t)
	options.FilenodeDefaultLimit = tenGiB

	created := CreateWrite(options)
	loaded := Load(options.CfgPath)

	assert.Equal(t, uint64(tenGiB), created.FileNode.DefaultLimit)
	assert.Equal(t, uint64(tenGiB), loaded.FileNode.DefaultLimit)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(cfg *Config)
		wantErr string
	}{
		{name: "valid config"},
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
			name: "invalid MongoDB URI",
			mutate: func(cfg *Config) {
				cfg.Consensus.MongoConnect = "mongodb://localhost:27017?w=majority"
			},
			wantErr: "consensus.mongoConnect must be a valid MongoDB URI",
		},
		{
			name: "MongoDB URI with surrounding whitespace",
			mutate: func(cfg *Config) {
				cfg.Consensus.MongoConnect = " mongodb://localhost:27017/?w=majority "
			},
			wantErr: "consensus.mongoConnect must be a valid MongoDB URI",
		},
		{
			name: "invalid redis URI",
			mutate: func(cfg *Config) {
				cfg.FileNode.RedisConnect = "localhost:6379"
			},
			wantErr: "filenode.redisConnect must include a host",
		},
		{
			name: "invalid S3 endpoint",
			mutate: func(cfg *Config) {
				cfg.FileNode.S3 = &S3Config{
					Bucket:   "bucket",
					Endpoint: "minio:9000",
				}
			},
			wantErr: "filenode.s3.endpoint must include a host",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validTestConfig()
			if test.mutate != nil {
				test.mutate(cfg)
			}

			err := cfg.Validate()
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.ErrorContains(t, err, test.wantErr)
		})
	}
}

func validCreateOptions(t *testing.T) *CreateOptions {
	t.Helper()

	dir := t.TempDir()
	return &CreateOptions{
		CfgPath:       filepath.Join(dir, "bundle.yml"),
		StorePath:     filepath.Join(dir, "storage"),
		MongoURI:      "mongodb://localhost:27017/",
		RedisURI:      "redis://localhost:6379/",
		ExternalAddrs: []string{"192.168.1.100"},
	}
}

func validTestConfig() *Config {
	return &Config{
		BundleVersion: "1.0.0",
		BundleFormat:  CurrentBundleFormat,
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

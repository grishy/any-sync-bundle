package config

import (
	"testing"

	"github.com/anyproto/any-sync/accountservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertS3Config(t *testing.T) {
	t.Run("all configured fields", func(t *testing.T) {
		t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")

		cfg := newTestConfig()
		cfg.FileNode.S3 = &S3Config{
			Bucket:         "my-bucket",
			Endpoint:       "http://minio:9000",
			Region:         "custom-region",
			ForcePathStyle: true,
		}

		s3Cfg := cfg.convertS3Config()

		assert.Equal(t, "custom-region", s3Cfg.Region)
		assert.Equal(t, "my-bucket", s3Cfg.Bucket)
		assert.Equal(t, "my-bucket", s3Cfg.IndexBucket)
		assert.Equal(t, "http://minio:9000", s3Cfg.Endpoint)
		assert.Equal(t, "default", s3Cfg.Profile)
		assert.Equal(t, 16, s3Cfg.MaxThreads)
		assert.True(t, s3Cfg.ForcePathStyle)
		assert.Equal(t, "test-access-key", s3Cfg.Credentials.AccessKey)
		assert.Equal(t, "test-secret-key", s3Cfg.Credentials.SecretKey)
	})

	t.Run("empty region preserves backwards-compatible default", func(t *testing.T) {
		cfg := newTestConfig()
		cfg.FileNode.S3 = &S3Config{
			Bucket:   "my-bucket",
			Endpoint: "https://s3.amazonaws.com",
		}

		assert.Equal(t, "us-east-1", cfg.convertS3Config().Region)
	})
}

func TestFilenodeConfig(t *testing.T) {
	t.Run("S3 configured", func(t *testing.T) {
		t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

		cfg := newTestConfig()
		cfg.FileNode.S3 = &S3Config{
			Bucket:         "test-bucket",
			Endpoint:       "http://minio:9000",
			Region:         "custom-region",
			ForcePathStyle: true,
		}

		filenode := cfg.NodeConfigs().Filenode

		require.NotNil(t, filenode)
		assert.Equal(t, "test-bucket", filenode.S3Store.Bucket)
		assert.Equal(t, "test-bucket", filenode.S3Store.IndexBucket)
		assert.Equal(t, "http://minio:9000", filenode.S3Store.Endpoint)
		assert.Equal(t, "custom-region", filenode.S3Store.Region)
		assert.True(t, filenode.S3Store.ForcePathStyle)
	})

	t.Run("S3 absent", func(t *testing.T) {
		filenode := newTestConfig().NodeConfigs().Filenode

		require.NotNil(t, filenode)
		assert.Empty(t, filenode.S3Store.Bucket)
	})

	t.Run("explicit storage limit", func(t *testing.T) {
		const tenGiB = 10 * 1024 * 1024 * 1024

		cfg := newTestConfig()
		cfg.FileNode.DefaultLimit = tenGiB

		assert.Equal(t, uint64(tenGiB), cfg.NodeConfigs().Filenode.DefaultLimit)
	})

	t.Run("zero storage limit uses compatibility default", func(t *testing.T) {
		cfg := newTestConfig()
		cfg.FileNode.DefaultLimit = 0

		assert.Equal(t, uint64(oneTiB), cfg.NodeConfigs().Filenode.DefaultLimit)
	})
}

func newTestConfig() *Config {
	return &Config{
		ConfigID:    "test-config-id",
		NetworkID:   "test-network-id",
		StoragePath: "/tmp/storage",
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
			MongoConnect:  "mongodb://localhost:27017/",
			MongoDatabase: "consensus",
		},
		FileNode: FileNodeConfig{
			RedisConnect: "redis://localhost:6379/",
		},
		ExternalAddr: []string{"192.168.1.100"},
	}
}

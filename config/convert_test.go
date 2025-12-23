package config

import (
	"testing"

	"github.com/anyproto/any-sync/accountservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertS3Config_AllFields(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")

	cfg := &Config{
		FileNode: FileNodeConfig{
			RedisConnect: "redis://localhost:6379/",
			S3: &S3Config{
				Bucket:         "my-bucket",
				Endpoint:       "https://s3.amazonaws.com",
				ForcePathStyle: false,
			},
		},
	}

	s3Cfg := cfg.convertS3Config()

	assert.Equal(t, "my-bucket", s3Cfg.Bucket)
	assert.Equal(t, "my-bucket", s3Cfg.IndexBucket, "IndexBucket should match Bucket")
	assert.Equal(t, "https://s3.amazonaws.com", s3Cfg.Endpoint)
	assert.Equal(t, "us-east-1", s3Cfg.Region)
	assert.Equal(t, "default", s3Cfg.Profile)
	assert.Equal(t, 16, s3Cfg.MaxThreads)
	assert.False(t, s3Cfg.ForcePathStyle)
	assert.Equal(t, "test-access-key", s3Cfg.Credentials.AccessKey)
	assert.Equal(t, "test-secret-key", s3Cfg.Credentials.SecretKey)
}

func TestConvertS3Config_ForcePathStyle(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "minio-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "minio-secret")

	cfg := &Config{
		FileNode: FileNodeConfig{
			S3: &S3Config{
				Bucket:         "local-bucket",
				Endpoint:       "http://minio:9000",
				ForcePathStyle: true,
			},
		},
	}

	s3Cfg := cfg.convertS3Config()

	assert.True(t, s3Cfg.ForcePathStyle)
	assert.Equal(t, "http://minio:9000", s3Cfg.Endpoint)
}

func TestConvertS3Config_MissingCredentials(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	cfg := &Config{
		FileNode: FileNodeConfig{
			S3: &S3Config{
				Bucket:   "my-bucket",
				Endpoint: "https://s3.amazonaws.com",
			},
		},
	}

	s3Cfg := cfg.convertS3Config()

	// Should still create config, but with empty credentials
	assert.Empty(t, s3Cfg.Credentials.AccessKey)
	assert.Empty(t, s3Cfg.Credentials.SecretKey)
}

// newTestConfig creates a minimal valid Config for testing.
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

func TestFilenodeConfig_WithS3(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	cfg := newTestConfig()
	cfg.FileNode.S3 = &S3Config{
		Bucket:         "test-bucket",
		Endpoint:       "https://s3.amazonaws.com",
		ForcePathStyle: false,
	}

	nodeCfgs := cfg.NodeConfigs()

	require.NotNil(t, nodeCfgs.Filenode)
	assert.Equal(t, "test-bucket", nodeCfgs.Filenode.S3Store.Bucket)
	assert.Equal(t, "test-bucket", nodeCfgs.Filenode.S3Store.IndexBucket)
	assert.Equal(t, "https://s3.amazonaws.com", nodeCfgs.Filenode.S3Store.Endpoint)
}

func TestFilenodeConfig_WithoutS3(t *testing.T) {
	cfg := newTestConfig()
	// No S3 config - default

	nodeCfgs := cfg.NodeConfigs()

	require.NotNil(t, nodeCfgs.Filenode)
	// S3Store should be empty (zero value)
	assert.Empty(t, nodeCfgs.Filenode.S3Store.Bucket)
}

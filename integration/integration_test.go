//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBundleFreshInstall(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Start MongoDB
	mongo, err := StartMongo(ctx)
	require.NoError(t, err, "Failed to start MongoDB")
	defer mongo.Terminate(ctx)
	t.Logf("MongoDB URI: %s", mongo.URI)

	// Start Redis
	redis, err := StartRedis(ctx)
	require.NoError(t, err, "Failed to start Redis")
	defer redis.Terminate(ctx)
	t.Logf("Redis URI: %s", redis.URI)

	// Start bundle
	bundle, err := StartBundle(ctx, BundleConfig{
		MongoURI: mongo.URI,
		RedisURI: redis.URI,
	})
	require.NoError(t, err, "Failed to start bundle")
	defer bundle.Cleanup()
	defer bundle.Stop()

	// Wait for ready
	err = bundle.WaitReady(90 * time.Second)
	require.NoError(t, err, "Bundle should become ready")
	t.Log("Bundle is ready")

	// Verify TCP port
	err = bundle.VerifyPort("33010")
	require.NoError(t, err, "Port 33010 should be listening")
	t.Log("Port 33010 is listening")

	// Graceful shutdown
	err = bundle.Stop()
	require.NoError(t, err, "Bundle should shutdown cleanly")
	t.Log("Bundle shutdown complete")
}

func TestBundleWithS3Storage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Start MongoDB
	mongo, err := StartMongo(ctx)
	require.NoError(t, err, "Failed to start MongoDB")
	defer mongo.Terminate(ctx)
	t.Logf("MongoDB URI: %s", mongo.URI)

	// Start Redis
	redis, err := StartRedis(ctx)
	require.NoError(t, err, "Failed to start Redis")
	defer redis.Terminate(ctx)
	t.Logf("Redis URI: %s", redis.URI)

	// Start MinIO
	minio, err := StartMinIO(ctx)
	require.NoError(t, err, "Failed to start MinIO")
	defer minio.Terminate(ctx)
	t.Logf("MinIO Endpoint: %s", minio.Endpoint)

	// Start bundle with S3
	bundle, err := StartBundle(ctx, BundleConfig{
		MongoURI:    mongo.URI,
		RedisURI:    redis.URI,
		S3Bucket:    "anytype-data",
		S3Endpoint:  minio.Endpoint,
		S3AccessKey: minio.AccessKey,
		S3SecretKey: minio.SecretKey,
	})
	require.NoError(t, err, "Failed to start bundle")
	defer bundle.Cleanup()
	defer bundle.Stop()

	// Wait for ready
	err = bundle.WaitReady(90 * time.Second)
	require.NoError(t, err, "Bundle should become ready")
	t.Log("Bundle is ready")

	// Verify S3 backend selected
	err = bundle.WaitForS3Backend(5 * time.Second)
	require.NoError(t, err, "S3 storage backend should be selected")
	t.Log("S3 storage backend confirmed")

	// Verify TCP port
	err = bundle.VerifyPort("33010")
	require.NoError(t, err, "Port 33010 should be listening")
	t.Log("Port 33010 is listening")

	// Graceful shutdown
	err = bundle.Stop()
	require.NoError(t, err, "Bundle should shutdown cleanly")
	t.Log("Bundle shutdown complete")
}

// TestBundleWithS3CustomRegion verifies that the bundle works with MinIO
// configured with a custom region. This tests the fix for GitHub issue #47.
func TestBundleWithS3CustomRegion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	customRegion := "custom-test-region"

	// Start MongoDB
	mongo, err := StartMongo(ctx)
	require.NoError(t, err, "Failed to start MongoDB")
	defer mongo.Terminate(ctx)
	t.Logf("MongoDB URI: %s", mongo.URI)

	// Start Redis
	redis, err := StartRedis(ctx)
	require.NoError(t, err, "Failed to start Redis")
	defer redis.Terminate(ctx)
	t.Logf("Redis URI: %s", redis.URI)

	// Start MinIO with custom region
	minio, err := StartMinIOWithRegion(ctx, customRegion)
	require.NoError(t, err, "Failed to start MinIO with custom region")
	defer minio.Terminate(ctx)
	t.Logf("MinIO Endpoint: %s, Region: %s", minio.Endpoint, minio.Region)

	// Start bundle with S3 and matching region
	bundle, err := StartBundle(ctx, BundleConfig{
		MongoURI:    mongo.URI,
		RedisURI:    redis.URI,
		S3Bucket:    "anytype-data",
		S3Endpoint:  minio.Endpoint,
		S3Region:    customRegion,
		S3AccessKey: minio.AccessKey,
		S3SecretKey: minio.SecretKey,
	})
	require.NoError(t, err, "Failed to start bundle")
	defer bundle.Cleanup()
	defer bundle.Stop()

	// Wait for ready
	err = bundle.WaitReady(90 * time.Second)
	require.NoError(t, err, "Bundle should become ready with custom S3 region")
	t.Log("Bundle is ready")

	// Verify S3 backend selected
	err = bundle.WaitForS3Backend(5 * time.Second)
	require.NoError(t, err, "S3 storage backend should be selected")
	t.Log("S3 storage backend confirmed")

	// Verify TCP port
	err = bundle.VerifyPort("33010")
	require.NoError(t, err, "Port 33010 should be listening")
	t.Log("Port 33010 is listening")

	// Graceful shutdown
	err = bundle.Stop()
	require.NoError(t, err, "Bundle should shutdown cleanly")
	t.Log("Bundle shutdown complete")
}

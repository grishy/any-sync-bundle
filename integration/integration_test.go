//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/anyproto/any-sync-filenode/store/s3store"
	"github.com/anyproto/any-sync/app"
	blocks "github.com/ipfs/go-block-format"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	bundleconfig "github.com/grishy/any-sync-bundle/config"
)

func TestBundleFreshInstall(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	mongo, err := StartMongo(ctx)
	require.NoError(t, err, "start MongoDB")
	defer mongo.Terminate(ctx)

	redis, err := StartRedis(ctx)
	require.NoError(t, err, "start Redis")
	defer redis.Terminate(ctx)

	bundle, err := StartBundle(ctx, BundleConfig{
		MongoURI: mongo.URI,
		RedisURI: redis.URI,
	})
	require.NoError(t, err, "start bundle")
	defer bundle.Cleanup()
	defer bundle.Stop()

	require.NoError(t, bundle.WaitReady(90*time.Second))
	require.NoError(t, bundle.VerifyPort("33010"))
	require.NoError(t, bundle.Stop(), "bundle should shut down cleanly")
}

// This test deliberately crosses the configuration and S3 request boundaries.
// Observing startup alone cannot detect invalid SigV4 credentials or region.
func TestS3StorageCustomRegionRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	const minioRegion = "custom-test-region"
	minio, err := StartMinIOWithRegion(ctx, minioRegion)
	require.NoError(t, err, "start MinIO")
	defer minio.Terminate(ctx)

	t.Setenv("AWS_ACCESS_KEY_ID", minio.AccessKey)
	t.Setenv("AWS_SECRET_ACCESS_KEY", minio.SecretKey)

	cfg := &bundleconfig.Config{
		ConfigID:    "s3-integration",
		NetworkID:   "s3-integration",
		StoragePath: t.TempDir(),
		Network: bundleconfig.NetworkConfig{
			ListenTCPAddr: "127.0.0.1:33010",
			ListenUDPAddr: "127.0.0.1:33020",
		},
		FileNode: bundleconfig.FileNodeConfig{
			S3: &bundleconfig.S3Config{
				Bucket:         "anytype-data",
				Endpoint:       minio.Endpoint,
				Region:         minioRegion,
				ForcePathStyle: true,
			},
		},
	}

	filenodeCfg := cfg.NodeConfigs().Filenode
	store := s3store.New()
	require.NoError(t, store.Init(new(app.App).Register(filenodeCfg)))
	require.NoError(t, store.Run(ctx))
	defer store.Close(context.Background())

	expected := blocks.NewBlock([]byte("custom-region-round-trip"))
	require.NoError(t, store.Add(ctx, []blocks.Block{expected}))

	actual, err := store.Get(ctx, expected.Cid())
	require.NoError(t, err)
	require.True(t, bytes.Equal(expected.RawData(), actual.RawData()))
}

// This is the container-boundary proof for the embedded process supervisor.
// A clean exit means services stopped first, MongoDB and Redis received their
// grace period, every child was reaped, and the final marker was published.
func TestAllInOneContainerShutsDownCleanly(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
	defer cancel()

	// All Dockerfile bases are public. An empty auth config keeps the build
	// independent of unrelated or expired credentials in the operator's store.
	t.Setenv("DOCKER_AUTH_CONFIG", "{}")

	image := fmt.Sprintf("any-sync-bundle-integration:%d", os.Getpid())
	buildImage := exec.CommandContext(
		ctx,
		"docker",
		"build",
		"--quiet",
		"--target",
		"stage-release-all-in-one",
		"--tag",
		image,
		"..",
	)
	buildOutput, err := buildImage.CombinedOutput()
	require.NoError(t, err, "build all-in-one image:\n%s", buildOutput)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			time.Minute,
		)
		defer cleanupCancel()
		_ = exec.CommandContext(
			cleanupCtx,
			"docker",
			"image",
			"rm",
			"--force",
			image,
		).Run()
	})

	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image: image,
				Env: map[string]string{
					"ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS": "127.0.0.1",
				},
				WaitingFor: wait.ForLog(bundleReadyEvent).
					WithStartupTimeout(7 * time.Minute),
			},
			Started: true,
		},
	)
	require.NoError(t, err, "start all-in-one container")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			time.Minute,
		)
		defer cleanupCancel()
		_ = container.Terminate(cleanupCtx)
	})

	stopGracePeriod := 2 * time.Minute
	require.NoError(t, container.Stop(ctx, &stopGracePeriod))

	logs, err := container.Logs(ctx)
	require.NoError(t, err)
	output, err := io.ReadAll(logs)
	require.NoError(t, err)
	require.NoError(t, logs.Close())
	require.Contains(t, string(output), bundleShutdownCompleteEvent)

	state, err := container.State(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, state.ExitCode)
}

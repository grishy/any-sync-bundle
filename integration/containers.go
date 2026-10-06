//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	mongoImage = "mongo:8.0.4"
	redisImage = "redis/redis-stack-server:7.4.0-v7"
	// Official images were removed. Upstream Compose uses this community build;
	// it includes both minio and mc. Pin the multi-platform manifest for both.
	minioImage = "docker.io/pgsty/minio:RELEASE.2026-08-04T00-00-00Z@sha256:b6bfe7239bfc83fb90d31612d9704d86039dd714f7904b3f1ad68f211e602372"
)

// MongoContainer wraps a MongoDB container with replica set support.
type MongoContainer struct {
	*mongodb.MongoDBContainer

	URI string
}

// StartMongo starts MongoDB with replica set using the official module.
// Uses API-based health checks instead of log parsing.
func StartMongo(ctx context.Context) (*MongoContainer, error) {
	container, err := mongodb.Run(ctx, mongoImage, mongodb.WithReplicaSet("rs0"))
	if err != nil {
		return nil, fmt.Errorf("failed to start mongo: %w", err)
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get mongo uri: %w", err)
	}

	return &MongoContainer{
		MongoDBContainer: container,
		URI:              uri + "&directConnection=true",
	}, nil
}

// RedisContainer wraps a Redis Stack container.
type RedisContainer struct {
	*tcredis.RedisContainer

	URI string
}

// StartRedis starts Redis Stack using the official module.
// Uses API-based health checks instead of log parsing.
func StartRedis(ctx context.Context) (*RedisContainer, error) {
	container, err := tcredis.Run(ctx, redisImage)
	if err != nil {
		return nil, fmt.Errorf("failed to start redis: %w", err)
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get redis uri: %w", err)
	}

	return &RedisContainer{
		RedisContainer: container,
		URI:            uri,
	}, nil
}

// MinIOContainer wraps a MinIO container.
type MinIOContainer struct {
	testcontainers.Container

	Endpoint  string
	AccessKey string
	SecretKey string
	Region    string
	network   *testcontainers.DockerNetwork
}

// StartMinIO starts MinIO for S3 testing with default region (us-east-1).
// Uses HTTP health endpoint for readiness check.
func StartMinIO(ctx context.Context) (*MinIOContainer, error) {
	return StartMinIOWithRegion(ctx, "")
}

// StartMinIOWithRegion starts MinIO with a custom region for S3 testing.
// If region is empty, MinIO uses its default (us-east-1).
func StartMinIOWithRegion(ctx context.Context, region string) (*MinIOContainer, error) {
	accessKey := "minioadmin"
	secretKey := "minioadmin"

	// Create a network for MinIO and mc to communicate
	dockerNet, err := network.New(ctx, network.WithDriver("bridge"))
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}
	networkName := dockerNet.Name

	env := map[string]string{
		"MINIO_ROOT_USER":     accessKey,
		"MINIO_ROOT_PASSWORD": secretKey,
	}
	if region != "" {
		env["MINIO_REGION"] = region
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        minioImage,
			ExposedPorts: []string{"9000/tcp"},
			Env:          env,
			Cmd:          []string{"server", "/data"},
			WaitingFor:   wait.ForHTTP("/minio/health/live").WithPort("9000/tcp"),
			Networks:     []string{networkName},
			NetworkAliases: map[string][]string{
				networkName: {"minio"},
			},
		},
		Started: true,
	})
	if err != nil {
		_ = dockerNet.Remove(ctx)
		return nil, fmt.Errorf("failed to start minio container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		_ = dockerNet.Remove(ctx)
		return nil, fmt.Errorf("failed to get minio host: %w", err)
	}

	port, err := container.MappedPort(ctx, "9000")
	if err != nil {
		_ = container.Terminate(ctx)
		_ = dockerNet.Remove(ctx)
		return nil, fmt.Errorf("failed to get minio port: %w", err)
	}

	// External endpoint (for the bundle to connect)
	externalEndpoint := "http://" + net.JoinHostPort(host, port.Port())

	// Create bucket using mc (use internal network address)
	if bucketErr := createMinioBucket(
		ctx,
		networkName,
		"http://minio:9000",
		accessKey,
		secretKey,
		"anytype-data",
	); bucketErr != nil {
		_ = container.Terminate(ctx)
		_ = dockerNet.Remove(ctx)
		return nil, fmt.Errorf("failed to create bucket: %w", bucketErr)
	}

	return &MinIOContainer{
		Container: container,
		Endpoint:  externalEndpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Region:    region,
		network:   dockerNet,
	}, nil
}

// Terminate stops the container and removes the network.
func (m *MinIOContainer) Terminate(ctx context.Context) error {
	err := m.Container.Terminate(ctx)
	if m.network != nil {
		_ = m.network.Remove(ctx)
	}
	return err
}

func createMinioBucket(ctx context.Context, networkName, endpoint, accessKey, secretKey, bucket string) error {
	mc, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:      minioImage,
			Entrypoint: []string{"/bin/sh"},
			Cmd: []string{"-c", fmt.Sprintf(
				"mc alias set minio %s %s %s && mc mb minio/%s --ignore-existing",
				endpoint, accessKey, secretKey, bucket,
			)},
			Networks:   []string{networkName},
			WaitingFor: wait.ForExit(),
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("failed to start mc container: %w", err)
	}
	defer func() { _ = mc.Terminate(ctx) }()

	state, err := mc.State(ctx)
	if err != nil {
		return fmt.Errorf("failed to get mc state: %w", err)
	}
	if state.ExitCode != 0 {
		logs, logsErr := mc.Logs(ctx)
		if logsErr == nil && logs != nil {
			logBytes, readErr := io.ReadAll(logs)
			if readErr == nil {
				return fmt.Errorf("mc exited with code %d: %s", state.ExitCode, string(logBytes))
			}
		}
		return errors.New("mc exited with non-zero code")
	}
	return nil
}

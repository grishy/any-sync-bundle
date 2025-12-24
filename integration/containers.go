//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoImage = "mongo:8.0.4"
	redisImage = "redis/redis-stack-server:7.4.0-v7"
	minioImage = "minio/minio:RELEASE.2025-09-07T16-13-09Z"
	mcImage    = "minio/mc:RELEASE.2025-08-13T08-35-41Z"

	defaultReplica = "rs0"
)

// MongoContainer wraps a MongoDB container with replica set support.
type MongoContainer struct {
	testcontainers.Container

	URI string
}

// StartMongo starts MongoDB with replica set, same as compose.dev.yml.
func StartMongo(ctx context.Context) (*MongoContainer, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        mongoImage,
			ExposedPorts: []string{"27017/tcp"},
			Cmd:          []string{"--replSet", defaultReplica, "--bind_ip", "0.0.0.0"},
			WaitingFor:   wait.ForLog("Waiting for connections").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start mongo container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get mongo host: %w", err)
	}

	port, err := container.MappedPort(ctx, "27017")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get mongo port: %w", err)
	}

	externalAddr := net.JoinHostPort(host, port.Port())
	baseURI := "mongodb://" + externalAddr + "/"

	// Initialize replica set with internal address (MongoDB must recognize itself)
	if initErr := initReplicaSet(ctx, baseURI); initErr != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to init replica set: %w", initErr)
	}

	// Use directConnection=true to bypass replica set discovery
	// This works because we're connecting to the primary directly
	return &MongoContainer{
		Container: container,
		URI:       baseURI + "?directConnection=true",
	}, nil
}

// initReplicaSet initializes MongoDB replica set.
// Adapted from cmd/mongo.go:initReplicaSetAction.
func initReplicaSet(ctx context.Context, uri string) error {
	clientOpts := options.Client().ApplyURI(uri).SetDirect(true)

	var client *mongo.Client
	var err error

	// Retry connection (MongoDB may not be ready immediately)
	for range 30 {
		client, err = mongo.Connect(ctx, clientOpts)
		if err == nil {
			if pingErr := client.Ping(ctx, nil); pingErr == nil {
				break
			}
			_ = client.Disconnect(ctx)
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to mongo: %w", err)
	}
	defer func() { _ = client.Disconnect(ctx) }()

	// Use localhost:27017 for replica set config since that's what MongoDB sees internally.
	// Clients will use directConnection=true to bypass replica set discovery.
	internalHost := "localhost:27017"

	// Initialize replica set
	cmd := bson.D{
		{Key: "replSetInitiate", Value: bson.D{
			{Key: "_id", Value: defaultReplica},
			{Key: "members", Value: bson.A{
				bson.D{
					{Key: "_id", Value: 0},
					{Key: "host", Value: internalHost},
				},
			}},
		}},
	}

	err = client.Database("admin").RunCommand(ctx, cmd).Err()
	if err != nil {
		// Check if already initialized
		var status bson.M
		statusErr := client.Database("admin").
			RunCommand(ctx, bson.D{{Key: "replSetGetStatus", Value: 1}}).
			Decode(&status)
		if statusErr != nil {
			return fmt.Errorf("init failed and status check failed: %w", err)
		}
		if ok, _ := status["ok"].(float64); ok != 1 {
			return fmt.Errorf("replica set not OK: %w", err)
		}
	}

	// Wait for replica set to stabilize
	time.Sleep(5 * time.Second)
	return nil
}

// RedisContainer wraps a Redis Stack container.
type RedisContainer struct {
	testcontainers.Container

	URI string
}

// StartRedis starts Redis Stack with bloom module, same as compose.dev.yml.
func StartRedis(ctx context.Context) (*RedisContainer, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        redisImage,
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start redis container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get redis host: %w", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get redis port: %w", err)
	}

	return &RedisContainer{
		Container: container,
		URI:       "redis://" + net.JoinHostPort(host, port.Port()) + "/",
	}, nil
}

// MinIOContainer wraps a MinIO container.
type MinIOContainer struct {
	testcontainers.Container

	Endpoint  string
	AccessKey string
	SecretKey string
	network   *testcontainers.DockerNetwork
}

// StartMinIO starts MinIO for S3 testing.
func StartMinIO(ctx context.Context) (*MinIOContainer, error) {
	accessKey := "minioadmin"
	secretKey := "minioadmin"

	// Create a network for MinIO and mc to communicate
	dockerNet, err := network.New(ctx, network.WithDriver("bridge"))
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}
	networkName := dockerNet.Name

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        minioImage,
			ExposedPorts: []string{"9000/tcp"},
			Env: map[string]string{
				"MINIO_ROOT_USER":     accessKey,
				"MINIO_ROOT_PASSWORD": secretKey,
			},
			Cmd:        []string{"server", "/data"},
			WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000/tcp").WithStartupTimeout(30 * time.Second),
			Networks:   []string{networkName},
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
	if bucketErr := createMinioBucket(ctx, networkName, "http://minio:9000", accessKey, secretKey, "anytype-data"); bucketErr != nil {
		_ = container.Terminate(ctx)
		_ = dockerNet.Remove(ctx)
		return nil, fmt.Errorf("failed to create bucket: %w", bucketErr)
	}

	return &MinIOContainer{
		Container: container,
		Endpoint:  externalEndpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
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
			Image:      mcImage,
			Entrypoint: []string{"/bin/sh"},
			Cmd: []string{"-c", fmt.Sprintf(
				"mc alias set minio %s %s %s && mc mb minio/%s --ignore-existing",
				endpoint, accessKey, secretKey, bucket,
			)},
			Networks:   []string{networkName},
			WaitingFor: wait.ForExit().WithExitTimeout(30 * time.Second),
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

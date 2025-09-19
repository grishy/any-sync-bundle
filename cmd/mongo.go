package cmd

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	// Timeouts for MongoDB operations.
	mongoConnectTimeout    = 10 * time.Second
	mongoCommandTimeout    = 5 * time.Second
	mongoStabilizeWaitTime = 5 * time.Second

	// Default MongoDB parameters.
	defaultMongoReplica = "rs0"
	defaultMongoURI     = "mongodb://127.0.0.1:27017/"
)

func cmdMongo(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:        "mongo",
		Usage:       "MongoDB management commands",
		Subcommands: []*cli.Command{cmdMongoInit(ctx)},
	}
}

func cmdMongoInit(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize MongoDB replica set",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "replica",
				Aliases: []string{"r"},
				Value:   defaultMongoReplica,
				Usage:   "Name of the replica set",
				EnvVars: []string{"ANY_SYNC_BUNDLE_MONGO_REPLICA"},
			},
			&cli.StringFlag{
				Name:    "uri",
				Aliases: []string{"u"},
				Value:   defaultMongoURI,
				Usage:   "MongoDB connection URI",
				EnvVars: []string{"ANY_SYNC_BUNDLE_MONGO_URI"},
			},
		},
		Action: func(cCtx *cli.Context) error {
			replica := cCtx.String("replica")
			mongoURI := cCtx.String("uri")

			return initReplicaSetAction(ctx, replica, mongoURI)
		},
	}
}

func initReplicaSetAction(ctx context.Context, replica, mongoURI string) error {
	// For exponential backoff, limit number of attempts.
	retryDelays := []int{1, 2, 3, 5, 8, 13, 21, 34, 55, 89}

	log.Info("initializing mongo replica set",
		zap.String("uri", mongoURI),
		zap.String("replica", replica))

	// Direct - before we have a replica set, we need it.
	clientOpts := options.Client().ApplyURI(mongoURI).SetDirect(true)

	for _, delay := range retryDelays {
		err := tryInitReplicaSet(ctx, clientOpts, replica)
		if err == nil {
			log.Info("successfully initialized mongo replica set")
			return nil
		} else if ctx.Err() != nil {
			log.Error("context canceled while initializing mongo replica set", zap.Error(ctx.Err()))
			return ctx.Err()
		}

		log.Warn("failed to initialize mongo replica set, retrying...",
			zap.Error(err),
			zap.Int("delay_seconds", delay))

		time.Sleep(time.Duration(delay) * time.Second)
	}

	return fmt.Errorf("failed to initialize mongo replica set after all retries")
}

func tryInitReplicaSet(ctx context.Context, clientOpts *options.ClientOptions, replica string) error {
	ctxConn, cancel := context.WithTimeout(ctx, mongoConnectTimeout)
	defer cancel()

	log.Debug("connecting to mongo", zap.String("uri", clientOpts.GetURI()))

	client, err := mongo.Connect(ctxConn, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to mongo: %w", err)
	}

	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Error("failed to disconnect from mongo", zap.Error(err))
		}
	}()

	if err := initNewReplicaSet(ctx, client, replica, clientOpts.GetURI()); err == nil {
		log.Info("successfully initialized new replica set, waiting for stabilization...")
		time.Sleep(mongoStabilizeWaitTime)
		return nil
	} else {
		log.Warn("failed to initialize new replica set", zap.Error(err))
	}

	return checkReplicaSetStatus(ctx, client)
}

func initNewReplicaSet(ctx context.Context, client *mongo.Client, replica, uri string) error {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("failed to parse mongodb uri: %w", err)
	}

	cmd := bson.D{
		{Key: "replSetInitiate", Value: bson.D{
			{Key: "_id", Value: replica},
			{Key: "members", Value: bson.A{
				bson.D{
					{Key: "_id", Value: 0},
					{Key: "host", Value: parsedURL.Host},
				},
			}},
		}},
	}

	ctxCmd, cancel := context.WithTimeout(ctx, mongoCommandTimeout)
	defer cancel()

	log.Debug("initializing new replica set")
	return client.Database("admin").RunCommand(ctxCmd, cmd).Err()
}

func checkReplicaSetStatus(ctx context.Context, client *mongo.Client) error {
	ctxCmd, cancel := context.WithTimeout(ctx, mongoCommandTimeout)
	defer cancel()

	log.Info("checking replica set status")

	var result bson.M
	err := client.Database("admin").
		RunCommand(ctxCmd, bson.D{{Key: "replSetGetStatus", Value: 1}}).
		Decode(&result)
	if err != nil {
		log.Warn("replica set status check failed", zap.Error(err))
		return fmt.Errorf("failed to get replica set status: %w", err)
	}

	if ok, _ := result["ok"].(float64); ok == 1 {
		log.Info("replica set is already initialized and OK")
		return nil
	}

	return fmt.Errorf("replica set is not properly initialized")
}

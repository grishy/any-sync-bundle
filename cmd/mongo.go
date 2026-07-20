package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	// Timeouts for MongoDB operations.
	mongoConnectTimeout     = 10 * time.Second
	mongoCommandTimeout     = 5 * time.Second
	mongoStabilizationDelay = 5 * time.Second

	// Default MongoDB parameters.
	defaultMongoReplica = "rs0"
)

func initReplicaSetAction(ctx context.Context, replica, mongoURI string) error {
	// The leading zero makes the first attempt immediate and a final delay impossible.
	attemptDelaysSec := [...]int{0, 1, 2, 3, 5, 8, 13, 21, 34, 55}

	log.Info("initializing mongo replica set",
		zap.String("uri", mongoURI),
		zap.String("replica", replica))

	// Direct - before we have a replica set, we need it.
	clientOpts := options.Client().ApplyURI(mongoURI).SetDirect(true)

	var lastErr error
	for _, attemptDelaySec := range attemptDelaysSec {
		attemptDelay := time.Duration(attemptDelaySec) * time.Second
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(attemptDelay):
		}

		lastErr = tryInitReplicaSet(ctx, clientOpts, replica)
		if lastErr == nil {
			log.Info("successfully initialized mongo replica set")
			return nil
		}
		if ctx.Err() != nil {
			return lastErr
		}
	}

	return fmt.Errorf("failed to initialize mongo replica set after all retries: %w", lastErr)
}

func tryInitReplicaSet(ctx context.Context, clientOpts *options.ClientOptions, replica string) error {
	connCtx, cancel := context.WithTimeout(ctx, mongoConnectTimeout)
	defer cancel()

	log.Debug("connecting to mongo", zap.String("uri", clientOpts.GetURI()))

	client, err := mongo.Connect(connCtx, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to mongo: %w", err)
	}

	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			log.Error("failed to disconnect from mongo", zap.Error(disconnectErr))
		}
	}()

	initErr := initNewReplicaSet(ctx, client, replica, clientOpts.GetURI())
	if initErr == nil {
		log.Info("successfully initialized new replica set, waiting for stabilization...")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(mongoStabilizationDelay):
			return nil
		}
	}
	log.Warn("failed to initialize new replica set", zap.Error(initErr))

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

	cmdCtx, cancel := context.WithTimeout(ctx, mongoCommandTimeout)
	defer cancel()

	log.Debug("initializing new replica set")
	return client.Database("admin").RunCommand(cmdCtx, cmd).Err()
}

func checkReplicaSetStatus(ctx context.Context, client *mongo.Client) error {
	cmdCtx, cancel := context.WithTimeout(ctx, mongoCommandTimeout)
	defer cancel()

	log.Info("checking replica set status")

	var result bson.M
	err := client.Database("admin").
		RunCommand(cmdCtx, bson.D{{Key: "replSetGetStatus", Value: 1}}).
		Decode(&result)
	if err != nil {
		log.Warn("replica set status check failed", zap.Error(err))
		return fmt.Errorf("failed to get replica set status: %w", err)
	}

	if ok, _ := result["ok"].(float64); ok == 1 {
		log.Info("replica set is already initialized and OK")
		return nil
	}

	return errors.New("replica set is not properly initialized")
}

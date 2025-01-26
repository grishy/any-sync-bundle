package main

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	bundleCfg "any-sync-bundle/config"
	bundleNode "any-sync-bundle/node"
)

var log = logger.NewNamed("main")

const (
	configBundlePath = "./data/cfg/priv_bundle.yml"
	configClientPath = "./data/cfg/pub_client.yml"
)

func main() {
	// TODO: Replace it on new build-in version of it in Go
	app.AppName = "any-sync-bundle"

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	// TODO: Create commands to only generate conf and allow to provide external addrs
	// TODO: Cread configs also from args or env
	cfgBundle := bundleCfg.BundleCfg(configBundlePath)
	cfgNodes := cfgBundle.NodeConfigs()

	mongoInit(ctx, cfgBundle.Nodes.Coordinator.MongoConnect)

	// Dump client config
	cfgBundle.DumpClientConfig(configClientPath)

	fileStore := filepath.Join(cfgBundle.StoragePath, "storage-file")

	// Common configs
	apps := []*app.App{
		bundleNode.NewCoordinatorApp(logger.NewNamed("coordinator"), cfgNodes.Coordinator),
		bundleNode.NewConsensusApp(logger.NewNamed("consensus"), cfgNodes.Consensus),
		bundleNode.NewFileNodeApp(logger.NewNamed("filenode"), cfgNodes.Filenode, fileStore),
		bundleNode.NewSyncApp(logger.NewNamed("sync"), cfgNodes.Sync),
	}

	// start apps
	for _, a := range apps {
		if err := a.Start(ctx); err != nil {
			log.Fatal("can't start app", zap.Error(err))
		}
	}
	log.Info("apps started")

	// wait exit signal
	<-ctx.Done()

	// TODO: Stop in reverse order
	// close apps
	for _, a := range apps {
		ctxClose, cancelClose := context.WithTimeout(context.Background(), time.Minute)
		if err := a.Close(ctxClose); err != nil {
			log.Fatal("close error", zap.Error(err))
		}
		cancelClose()
	}
	log.Info("goodbye!")

	time.Sleep(time.Second / 3)
}

func mongoInit(ctx context.Context, mongoURI string) {
	log.Info("initializing mongo replica set", zap.String("uri", mongoURI))

	sleepsSec := []int{1, 2, 3, 5, 8, 13, 21, 34, 55, 89}

	clientOpts := options.Client().ApplyURI(mongoURI).
		SetDirect(true).
		SetTimeout(time.Second * 10)

	initRs := func() error {
		// Use shorter timeout for initial connection and remove replicaSet from URI for initial connection
		ctxConn, cancelConn := context.WithTimeout(ctx, time.Second*5)
		defer cancelConn()

		log.Info("connecting to mongo", zap.String("uri", clientOpts.GetURI()))
		client, err := mongo.Connect(ctxConn, clientOpts)
		if err != nil {
			return fmt.Errorf("failed to connect to mongo: %w", err)
		}
		defer func() {
			if err := client.Disconnect(ctx); err != nil {
				log.Error("failed to disconnect from mongo", zap.Error(err))
			}
		}()

		// Try to initialize replica set first
		cmd := bson.D{
			{Key: "replSetInitiate", Value: bson.D{
				{Key: "_id", Value: "rs0"},
				{Key: "members", Value: bson.A{
					bson.D{
						{Key: "_id", Value: 0},
						{Key: "host", Value: "127.0.0.1:27017"},
					},
				}},
			}},
		}

		ctxCmd, cancelCmd := context.WithTimeout(ctx, time.Second*5)
		defer cancelCmd()

		log.Info("initializing new replica set")
		err = client.Database("admin").RunCommand(ctxCmd, cmd).Err()
		if err == nil {
			log.Info("successfully initialized new replica set, waiting for it to stabilize...")
			time.Sleep(time.Second * 5)
			return nil
		}

		log.Info("replSetInitiate failed, checking if already initialized", zap.Error(err))

		// Check replica set status
		ctxCmd, cancelCmd = context.WithTimeout(ctx, time.Second*5)
		defer cancelCmd()

		var result bson.M
		err = client.Database("admin").RunCommand(ctxCmd, bson.D{{Key: "replSetGetStatus", Value: 1}}).Decode(&result)
		if err != nil {
			log.Warn("replica set status check failed",
				zap.Error(err))
			return fmt.Errorf("failed to get replica set status: %w", err)
		}

		if ok, _ := result["ok"].(float64); ok == 1 {
			log.Info("replica set is already initialized and OK")
			return nil
		}

		return fmt.Errorf("replica set is not properly initialized")
	}

	for _, sec := range sleepsSec {
		err := initRs()
		if err == nil {
			log.Info("successfully initialized mongo replica set")
			return
		}

		if ctx.Err() != nil {
			log.Panic("context canceled while initializing mongo replica set", zap.Error(ctx.Err()))
		}

		log.Warn("failed to initialize mongo replica set, retrying...", zap.Error(err), zap.Int("sec", sec))
		time.Sleep(time.Second * time.Duration(sec))
	}

	log.Panic("failed to initialize mongo replica set after all retries")
}

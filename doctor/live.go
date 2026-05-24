package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anyproto/any-sync-filenode/redisprovider"
	filenodestore "github.com/anyproto/any-sync-filenode/store"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/commonfile/fileblockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	bundleconfig "github.com/grishy/any-sync-bundle/config"
)

type LiveRuntimeConfig struct {
	BundleConfig     *bundleconfig.Config
	BundleConfigPath string
	ClientConfigPath string
	Build            BuildInfo
	FileNode         *app.App
}

func NewLiveRuntimeRunner(cfg LiveRuntimeConfig) (*RuntimeRunner, error) {
	redisClient, err := redisClientFromFileNode(cfg.FileNode)
	if err != nil {
		return nil, err
	}
	blockStore, err := blockStoreFromFileNode(cfg.FileNode)
	if err != nil {
		return nil, err
	}

	return NewRuntimeRunner(RuntimeRunnerConfig{
		BundleConfig:     cfg.BundleConfig,
		BundleConfigPath: cfg.BundleConfigPath,
		ClientConfigPath: cfg.ClientConfigPath,
		Build:            cfg.Build,
		Now:              func() time.Time { return time.Now().UTC() },
		LoadIndexSnapshot: func(ctx context.Context) (IndexSnapshot, error) {
			return LoadIndexSnapshotFromRedis(ctx, redisClient)
		},
		ProbeBlocks: func(ctx context.Context, inventory Inventory) (BlockProbeResult, error) {
			return ProbeBlocksWithStore(ctx, inventory, blockStore)
		},
		ProbeRuntime: func(ctx context.Context) RuntimeProbeResult {
			return ProbeRuntime(ctx, cfg.BundleConfig, redisClient)
		},
	}), nil
}

func LoadIndexSnapshotFromRedis(ctx context.Context, client redis.UniversalClient) (IndexSnapshot, error) {
	if client == nil {
		return IndexSnapshot{}, errors.New("redis client is required")
	}

	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{},
		Values: map[string]string{},
	}
	for _, pattern := range []string{
		redisIndexScanPattern(redisIndexGroupPrefix),
		redisIndexScanPattern(redisIndexSpacePrefix),
	} {
		keys, err := scanRedisKeys(ctx, client, pattern)
		if err != nil {
			return IndexSnapshot{}, err
		}
		for _, key := range keys {
			fields, fieldsErr := client.HGetAll(ctx, key).Result()
			if fieldsErr != nil {
				return IndexSnapshot{}, fmt.Errorf("read redis hash %s: %w", key, fieldsErr)
			}
			snapshot.Hashes[key] = fields
		}
	}

	keys, err := scanRedisKeys(ctx, client, redisIndexScanPattern(redisIndexCIDPrefix))
	if err != nil {
		return IndexSnapshot{}, err
	}
	for _, key := range keys {
		value, valueErr := client.Get(ctx, key).Result()
		if valueErr != nil {
			if errors.Is(valueErr, redis.Nil) {
				continue
			}
			return IndexSnapshot{}, fmt.Errorf("read redis value %s: %w", key, valueErr)
		}
		snapshot.Values[key] = value
	}

	return snapshot, nil
}

func ProbeBlocksWithStore(
	ctx context.Context,
	inventory Inventory,
	blockStore filenodestore.Store,
) (BlockProbeResult, error) {
	if blockStore == nil {
		return BlockProbeResult{}, errors.New("filenode block store is required")
	}

	result := BlockProbeResult{
		MissingByFile: map[string][]string{},
		CorruptByFile: map[string][]string{},
	}
	for _, file := range inventory.Files {
		for _, cidString := range file.CIDs {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			decodedCID, err := cid.Decode(cidString)
			if err != nil {
				result.Problems = append(result.Problems, Problem{
					Scope: problemScopeFile,
					ID:    file.ID,
					Issue: fmt.Sprintf("CID %s cannot be decoded: %v", cidString, err),
				})
				continue
			}

			blockCtx := fileblockstore.CtxWithSpaceId(ctx, file.SpaceID)
			blockCtx = fileblockstore.CtxWithFileId(blockCtx, file.ID)
			block, err := blockStore.Get(blockCtx, decodedCID)
			result.Checked++
			if err == nil {
				if blockProblems := inspectBlock(file, cidString, decodedCID, block); len(blockProblems) > 0 {
					key := fileReportKey(file.SpaceID, file.ID)
					result.CorruptByFile[key] = append(result.CorruptByFile[key], cidString)
					result.Problems = append(result.Problems, blockProblems...)
				}
				continue
			}
			if errors.Is(err, fileblockstore.ErrCIDNotFound) {
				key := fileReportKey(file.SpaceID, file.ID)
				result.MissingByFile[key] = append(result.MissingByFile[key], cidString)
				continue
			}
			result.Problems = append(result.Problems, Problem{
				Scope: problemScopeFile,
				ID:    file.ID,
				Issue: fmt.Sprintf("block %s cannot be read: %v", cidString, err),
			})
		}
	}
	return result, nil
}

func inspectBlock(file FileReport, cidString string, expectedCID cid.Cid, block blocks.Block) []Problem {
	if block == nil {
		return []Problem{{
			Scope: problemScopeFile,
			ID:    file.ID,
			Issue: fmt.Sprintf("block %s returned an empty block", cidString),
		}}
	}

	problems := []Problem{}
	if !block.Cid().Equals(expectedCID) {
		problems = append(problems, Problem{
			Scope: problemScopeFile,
			ID:    file.ID,
			Issue: fmt.Sprintf("block %s returned CID %s", cidString, block.Cid().String()),
		})
	}

	data := block.RawData()
	actualCID, err := expectedCID.Prefix().Sum(data)
	if err != nil {
		problems = append(problems, Problem{
			Scope: problemScopeFile,
			ID:    file.ID,
			Issue: fmt.Sprintf("block %s content cannot be hashed: %v", cidString, err),
		})
	} else if !actualCID.Equals(expectedCID) {
		problems = append(problems, Problem{
			Scope: problemScopeFile,
			ID:    file.ID,
			Issue: fmt.Sprintf("block %s content does not match its CID", cidString),
		})
	}

	expectedSize, ok := file.CIDSizes[cidString]
	if !ok {
		if len(file.CIDs) == 1 {
			expectedSize = file.Size
			ok = expectedSize > 0
		}
	}
	if ok {
		actualSize := uint64(len(data))
		if actualSize != expectedSize {
			problems = append(problems, Problem{
				Scope: problemScopeFile,
				ID:    file.ID,
				Issue: fmt.Sprintf("block %s size mismatch: index=%d actual=%d",
					cidString, expectedSize, actualSize),
			})
		}
	}

	return problems
}

func ProbeRuntime(ctx context.Context, cfg *bundleconfig.Config, redisClient redis.UniversalClient) RuntimeProbeResult {
	result := RuntimeProbeResult{
		Mongo:      StatusOK,
		Redis:      StatusOK,
		RedisBloom: StatusOK,
		Storage:    StatusOK,
	}
	if cfg == nil {
		result.Mongo = StatusProblem
		result.Redis = StatusProblem
		result.RedisBloom = StatusProblem
		result.Storage = StatusProblem
		result.Problems = append(result.Problems, Problem{
			Scope: problemScopeRuntime,
			Issue: "bundle config is not available for runtime probes",
		})
		return result
	}
	if storagePathErr := checkStoragePath(cfg.StoragePath); storagePathErr != nil {
		result.Storage = StatusProblem
		result.Problems = append(result.Problems, Problem{
			Scope: problemScopeRuntime,
			Issue: fmt.Sprintf("storage path check failed: %v", storagePathErr),
		})
	} else if storageLayoutErr := checkStorageLayout(cfg); storageLayoutErr != nil {
		result.Storage = StatusProblem
		result.Problems = append(result.Problems, Problem{
			Scope: problemScopeRuntime,
			Issue: fmt.Sprintf("storage layout check failed: %v", storageLayoutErr),
		})
	}

	if err := pingMongo(ctx, cfg.Coordinator.MongoConnect); err != nil {
		result.Mongo = StatusProblem
		result.Problems = append(result.Problems, Problem{
			Scope: problemScopeRuntime,
			Issue: fmt.Sprintf("coordinator mongo ping failed: %v", err),
		})
	}
	if cfg.Consensus.MongoConnect != cfg.Coordinator.MongoConnect {
		if err := pingMongo(ctx, cfg.Consensus.MongoConnect); err != nil {
			result.Mongo = StatusProblem
			result.Problems = append(result.Problems, Problem{
				Scope: problemScopeRuntime,
				Issue: fmt.Sprintf("consensus mongo ping failed: %v", err),
			})
		}
	}
	if redisClient == nil {
		result.Redis = StatusProblem
		result.RedisBloom = StatusProblem
		result.Problems = append(result.Problems, Problem{
			Scope: problemScopeRuntime,
			Issue: "redis client is not available",
		})
		return result
	}
	if err := redisClient.Ping(ctx).Err(); err != nil {
		result.Redis = StatusProblem
		result.Problems = append(result.Problems, Problem{
			Scope: problemScopeRuntime,
			Issue: fmt.Sprintf("redis ping failed: %v", err),
		})
	}
	if err := redisClient.Do(ctx, "BF.EXISTS", "_doctor_bloom_probe", "probe").
		Err(); err != nil &&
		!errors.Is(err, redis.Nil) {
		result.RedisBloom = StatusProblem
		result.Problems = append(result.Problems, Problem{
			Scope: problemScopeRuntime,
			Issue: fmt.Sprintf("redis bloom probe failed: %v", err),
		})
	}
	return result
}

func checkStoragePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	tmp, err := os.CreateTemp(path, ".doctor-write-test-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if closeErr := tmp.Close(); closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if removeErr := os.Remove(tmpPath); removeErr != nil {
		return fmt.Errorf("remove write test file %s: %w", filepath.Base(tmpPath), removeErr)
	}
	return nil
}

func checkStorageLayout(cfg *bundleconfig.Config) error {
	requiredDirs := []string{
		filepath.Join(cfg.StoragePath, "network-store", "coordinator"),
		filepath.Join(cfg.StoragePath, "network-store", "consensus"),
		filepath.Join(cfg.StoragePath, "network-store", "filenode"),
		filepath.Join(cfg.StoragePath, "network-store", "sync"),
		filepath.Join(cfg.StoragePath, "storage-sync"),
	}
	if cfg.FileNode.S3 == nil {
		requiredDirs = append(requiredDirs, filepath.Join(cfg.StoragePath, "storage-file"))
	}

	for _, path := range requiredDirs {
		if err := checkReadableDirectory(path); err != nil {
			rel, relErr := filepath.Rel(cfg.StoragePath, path)
			if relErr != nil {
				rel = path
			}
			return fmt.Errorf("%s: %w", rel, err)
		}
	}
	return nil
}

func checkReadableDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("not a directory")
	}
	if _, readErr := os.ReadDir(path); readErr != nil {
		return readErr
	}
	return nil
}

func scanRedisKeys(ctx context.Context, client redis.UniversalClient, pattern string) ([]string, error) {
	var cursor uint64
	keys := []string{}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch, nextCursor, err := client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return nil, fmt.Errorf("scan redis keys %s: %w", pattern, err)
		}
		keys = append(keys, batch...)
		cursor = nextCursor
		if cursor == 0 {
			return keys, nil
		}
	}
}

func redisClientFromFileNode(fileNode *app.App) (redis.UniversalClient, error) {
	if fileNode == nil {
		return nil, errors.New("filenode app is not available")
	}
	component := fileNode.Component(redisprovider.CName)
	provider, ok := component.(redisprovider.RedisProvider)
	if !ok {
		return nil, errors.New("filenode redis provider component is not available")
	}
	return provider.Redis(), nil
}

func blockStoreFromFileNode(fileNode *app.App) (filenodestore.Store, error) {
	if fileNode == nil {
		return nil, errors.New("filenode app is not available")
	}
	component := fileNode.Component(fileblockstore.CName)
	blockStore, ok := component.(filenodestore.Store)
	if !ok {
		return nil, errors.New("filenode block store component is not available")
	}
	return blockStore, nil
}

func pingMongo(ctx context.Context, uri string) error {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(pingCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	defer func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer disconnectCancel()
		_ = client.Disconnect(disconnectCtx)
	}()

	return client.Ping(pingCtx, readpref.Primary())
}

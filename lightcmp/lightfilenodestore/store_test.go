package lightfilenodestore

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/commonfile/fileblockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers

func setupTestStore(t *testing.T) (*LightFileNodeStore, func()) {
	tmpDir := t.TempDir()
	store := New(tmpDir)

	// Initialize store
	err := store.Init(&app.App{})
	require.NoError(t, err)

	// Run store
	ctx := context.Background()
	err = store.Run(ctx)
	require.NoError(t, err)

	cleanup := func() {
		closeErr := store.Close(context.Background())
		if closeErr != nil {
			t.Logf("Failed to close store: %v", closeErr)
		}
	}

	return store, cleanup
}

func createTestBlock(t *testing.T, data []byte) blocks.Block {
	mh, err := multihash.Sum(data, multihash.SHA2_256, -1)
	if err != nil {
		t.Fatalf("failed to create multihash: %v", err)
	}

	c := cid.NewCidV1(cid.Raw, mh)
	block, err := blocks.NewBlockWithCid(data, c)
	if err != nil {
		t.Fatalf("failed to create block: %v", err)
	}

	return block
}

func createTestBlocks(t *testing.T, count int) []blocks.Block {
	blocks := make([]blocks.Block, count)
	for i := range count {
		data := fmt.Appendf(nil, "test-block-%d", i)
		blocks[i] = createTestBlock(t, data)
	}
	return blocks
}

// Basic CRUD Tests

func TestLightFileNodeStore_Init(t *testing.T) {
	tmpDir := t.TempDir()
	store := New(tmpDir)

	err := store.Init(&app.App{})
	require.NoError(t, err)
	assert.Equal(t, CName, store.Name())
}

func TestLightFileNodeStore_Run_UnwritablePath(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.Chmod(baseDir, 0o555))
	defer os.Chmod(baseDir, 0o755)

	store := New(filepath.Join(baseDir, "store"))
	require.NoError(t, store.Init(&app.App{}))

	err := store.Run(context.Background())
	assert.Error(t, err)
}

func TestLightFileNodeStore_AddGetVariants(t *testing.T) {
	t.Helper()

	cases := []struct {
		name       string
		makeBlocks func(*testing.T) []blocks.Block
	}{
		{
			name: "single",
			makeBlocks: func(t *testing.T) []blocks.Block {
				return []blocks.Block{createTestBlock(t, []byte("single-block"))}
			},
		},
		{
			name: "multiple",
			makeBlocks: func(t *testing.T) []blocks.Block {
				return createTestBlocks(t, 10)
			},
		},
		{
			name: "empty",
			makeBlocks: func(t *testing.T) []blocks.Block {
				return []blocks.Block{createTestBlock(t, []byte{})}
			},
		},
		{
			name: "large",
			makeBlocks: func(t *testing.T) []blocks.Block {
				data := make([]byte, 2*1024*1024)
				for i := range data {
					data[i] = byte(i % 256)
				}
				return []blocks.Block{createTestBlock(t, data)}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			store, cleanup := setupTestStore(t)
			defer cleanup()

			ctx := context.Background()
			blocks := tc.makeBlocks(t)

			require.NoError(t, store.Add(ctx, blocks))

			for _, block := range blocks {
				retrieved, err := store.Get(ctx, block.Cid())
				require.NoError(t, err)
				require.True(
					t,
					bytes.Equal(block.RawData(), retrieved.RawData()),
					"expected cid %s to match",
					block.Cid(),
				)
			}
		})
	}
}

func TestLightFileNodeStore_GetMany(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	testBlocks := createTestBlocks(t, 20)

	// Add blocks
	err := store.Add(ctx, testBlocks)
	require.NoError(t, err)

	// Prepare CIDs
	cids := make([]cid.Cid, len(testBlocks))
	for i, block := range testBlocks {
		cids[i] = block.Cid()
	}

	// Get many blocks
	resultChan := store.GetMany(ctx, cids)

	// Collect results
	results := make(map[string]blocks.Block)
	for block := range resultChan {
		results[block.Cid().String()] = block
	}

	// Verify all blocks retrieved
	assert.Len(t, results, len(testBlocks))
	for _, original := range testBlocks {
		retrieved, ok := results[original.Cid().String()]
		require.True(t, ok)
		assert.Equal(t, original.RawData(), retrieved.RawData())
	}
}

func TestLightFileNodeStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	block := createTestBlock(t, []byte("to be deleted"))

	// Add block
	err := store.Add(ctx, []blocks.Block{block})
	require.NoError(t, err)

	// Verify it exists
	_, err = store.Get(ctx, block.Cid())
	require.NoError(t, err)

	// Delete block
	err = store.Delete(ctx, block.Cid())
	require.NoError(t, err)

	// Verify it's gone
	_, err = store.Get(ctx, block.Cid())
	assert.ErrorIs(t, err, fileblockstore.ErrCIDNotFound)
}

func TestLightFileNodeStore_DeleteMany(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	testBlocks := createTestBlocks(t, 5)

	// Add blocks
	err := store.Add(ctx, testBlocks)
	require.NoError(t, err)

	// Delete all blocks
	cids := make([]cid.Cid, len(testBlocks))
	for i, block := range testBlocks {
		cids[i] = block.Cid()
	}
	err = store.DeleteMany(ctx, cids)
	require.NoError(t, err)

	// Verify all are gone
	for _, c := range cids {
		_, err = store.Get(ctx, c)
		assert.ErrorIs(t, err, fileblockstore.ErrCIDNotFound)
	}
}

func TestLightFileNodeStore_Delete_Idempotent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	block := createTestBlock(t, []byte("idempotent-delete"))
	require.NoError(t, store.Add(ctx, []blocks.Block{block}))

	// First delete removes the block.
	require.NoError(t, store.Delete(ctx, block.Cid()))

	// Subsequent deletes should be no-ops.
	require.NoError(t, store.Delete(ctx, block.Cid()))

	// Deleting a CID that never existed should also succeed.
	missing := createTestBlock(t, []byte("missing-delete"))
	require.NoError(t, store.Delete(ctx, missing.Cid()))
}

func TestLightFileNodeStore_DeleteMany_MissingEntries(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	blocks := createTestBlocks(t, 3)
	require.NoError(t, store.Add(ctx, blocks))

	missing := createTestBlocks(t, 2)
	var toDelete []cid.Cid
	for _, block := range blocks {
		toDelete = append(toDelete, block.Cid())
	}
	for _, block := range missing {
		toDelete = append(toDelete, block.Cid())
	}

	require.NoError(t, store.DeleteMany(ctx, toDelete))

	for _, block := range blocks {
		_, err := store.Get(ctx, block.Cid())
		assert.ErrorIs(t, err, fileblockstore.ErrCIDNotFound)
	}
}

// Error Handling Tests

func TestLightFileNodeStore_Get_NonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	block := createTestBlock(t, []byte("non-existent"))

	_, err := store.Get(ctx, block.Cid())
	assert.ErrorIs(t, err, fileblockstore.ErrCIDNotFound)
}

func TestLightFileNodeStore_GetMany_WithMissingBlocks(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	existingBlocks := createTestBlocks(t, 3)
	missingBlocks := createTestBlocks(t, 2)

	// Add only existing blocks
	err := store.Add(ctx, existingBlocks)
	require.NoError(t, err)

	// Request both existing and missing
	allCids := make([]cid.Cid, 0, 5)
	for _, block := range existingBlocks {
		allCids = append(allCids, block.Cid())
	}
	for _, block := range missingBlocks {
		allCids = append(allCids, block.Cid())
	}

	// Get many should return only existing blocks
	resultChan := store.GetMany(ctx, allCids)
	resultsMap := make(map[string]bool)
	for block := range resultChan {
		resultsMap[block.Cid().String()] = true
	}

	assert.Len(t, resultsMap, 3)
}

// Index Operations Tests

func TestLightFileNodeStore_IndexPut_Get(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	key := "test-index-key"
	value := []byte("test-index-value")

	// Put index value
	err := store.IndexPut(ctx, key, value)
	require.NoError(t, err)

	// Get index value
	retrieved, err := store.IndexGet(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestLightFileNodeStore_IndexGet_NonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	key := "non-existent-key"

	// Get non-existent index value
	value, err := store.IndexGet(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, value)
}

func TestLightFileNodeStore_IndexDelete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	key := "delete-me"
	value := []byte("to be deleted")

	// Put index value
	err := store.IndexPut(ctx, key, value)
	require.NoError(t, err)

	// Verify it exists
	retrieved, err := store.IndexGet(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Delete index value
	err = store.IndexDelete(ctx, key)
	require.NoError(t, err)

	// Verify it's gone
	retrieved, err = store.IndexGet(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestLightFileNodeStore_IndexDelete_NonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	err := store.IndexDelete(ctx, "missing-index")
	require.NoError(t, err)
}

func TestLightFileNodeStore_IndexKeyPrefix_Isolation(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a block with a CID that could collide with index prefix
	blockData := []byte("block data")
	block := createTestBlock(t, blockData)

	// Add block
	err := store.Add(ctx, []blocks.Block{block})
	require.NoError(t, err)

	// Create an index with a key that's the same as the block CID
	indexKey := block.Cid().String()
	indexValue := []byte("index value")

	// Put index value
	err = store.IndexPut(ctx, indexKey, indexValue)
	require.NoError(t, err)

	// Both should coexist without collision
	retrievedBlock, err := store.Get(ctx, block.Cid())
	require.NoError(t, err)
	assert.Equal(t, blockData, retrievedBlock.RawData())

	retrievedIndex, err := store.IndexGet(ctx, indexKey)
	require.NoError(t, err)
	assert.Equal(t, indexValue, retrievedIndex)
}

// Concurrent Operations Tests

func TestLightFileNodeStore_ConcurrentWrites(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	numGoroutines := 10
	blocksPerGoroutine := 5

	blockSets := make([][]blocks.Block, numGoroutines)
	for i := range blockSets {
		blockSets[i] = make([]blocks.Block, blocksPerGoroutine)
		for j := range blockSets[i] {
			data := fmt.Appendf(nil, "goroutine-%d-block-%d", i, j)
			blockSets[i][j] = createTestBlock(t, data)
		}
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	cidCh := make(chan cid.Cid, numGoroutines*blocksPerGoroutine)
	errCh := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(batch []blocks.Block) {
			defer wg.Done()
			for _, block := range batch {
				if err := store.Add(ctx, []blocks.Block{block}); err != nil {
					errCh <- err
					return
				}
				cidCh <- block.Cid()
			}
		}(blockSets[i])
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	close(cidCh)
	retrieved := 0
	for c := range cidCh {
		_, err := store.Get(ctx, c)
		require.NoError(t, err)
		retrieved++
	}

	assert.Equal(t, numGoroutines*blocksPerGoroutine, retrieved)
}

func TestLightFileNodeStore_ConcurrentReads(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	testBlocks := createTestBlocks(t, 10)

	// Add blocks
	err := store.Add(ctx, testBlocks)
	require.NoError(t, err)

	numGoroutines := 20
	readsPerGoroutine := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	start := make(chan struct{})
	errCh := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			<-start
			for j := 0; j < readsPerGoroutine; j++ {
				block := testBlocks[(id+j)%len(testBlocks)]
				retrieved, err := store.Get(ctx, block.Cid())
				if err != nil {
					errCh <- err
					return
				}
				if !assert.ObjectsAreEqual(block.RawData(), retrieved.RawData()) {
					errCh <- fmt.Errorf("data mismatch for %s", block.Cid())
					return
				}
			}
		}(i)
	}

	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
}

func TestLightFileNodeStore_ConcurrentMixedOperations(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	testBlocks := createTestBlocks(t, 20)

	// Add initial blocks
	err := store.Add(ctx, testBlocks[:10])
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(3)
	start := make(chan struct{})
	readerErr := make(chan error, 1)
	writerErr := make(chan error, 1)
	getManyErr := make(chan error, 1)
	var getManyCount atomic.Int32

	// Reader goroutine
	go func() {
		defer wg.Done()
		<-start
		for _, block := range testBlocks[:10] {
			retrieved, err := store.Get(ctx, block.Cid())
			if err != nil {
				readerErr <- err
				return
			}
			if !assert.ObjectsAreEqual(block.RawData(), retrieved.RawData()) {
				readerErr <- fmt.Errorf("reader mismatch for %s", block.Cid())
				return
			}
		}
		readerErr <- nil
	}()

	// Writer goroutine
	go func() {
		defer wg.Done()
		<-start
		for _, block := range testBlocks[10:] {
			if err := store.Add(ctx, []blocks.Block{block}); err != nil {
				writerErr <- err
				return
			}
		}
		writerErr <- nil
	}()

	// GetMany goroutine
	go func() {
		defer wg.Done()
		<-start
		cids := make([]cid.Cid, 10)
		for i := range 10 {
			cids[i] = testBlocks[i].Cid()
		}
		for i := 0; i < 5; i++ {
			resultChan := store.GetMany(ctx, cids)
			// Consume results to avoid blocking
			for range resultChan {
				getManyCount.Add(1)
			}
		}
		getManyErr <- nil
	}()

	close(start)
	wg.Wait()
	close(readerErr)
	for err := range readerErr {
		require.NoError(t, err)
	}
	close(writerErr)
	for err := range writerErr {
		require.NoError(t, err)
	}
	close(getManyErr)
	for err := range getManyErr {
		require.NoError(t, err)
	}

	// Ensure all blocks (original + new) are available after concurrent operations.
	for _, block := range testBlocks {
		retrieved, err := store.Get(ctx, block.Cid())
		require.NoError(t, err)
		assert.Equal(t, block.RawData(), retrieved.RawData())
	}

	assert.GreaterOrEqual(t, getManyCount.Load(), int32(10)) // Expect at least one full batch to be read.
}

// Edge Cases Tests

func TestLightFileNodeStore_DuplicateAdd(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	block := createTestBlock(t, []byte("duplicate"))

	// Add block multiple times
	for range 3 {
		err := store.Add(ctx, []blocks.Block{block})
		require.NoError(t, err)
	}

	// Should still be able to retrieve
	retrieved, err := store.Get(ctx, block.Cid())
	require.NoError(t, err)
	assert.Equal(t, block.RawData(), retrieved.RawData())
}

// Integration Tests

func TestLightFileNodeStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// First store instance
	store1 := New(tmpDir)
	err := store1.Init(&app.App{})
	require.NoError(t, err)
	err = store1.Run(ctx)
	require.NoError(t, err)

	// Add data
	block := createTestBlock(t, []byte("persistent data"))
	err = store1.Add(ctx, []blocks.Block{block})
	require.NoError(t, err)

	// Add index data
	err = store1.IndexPut(ctx, "persistent-key", []byte("persistent-value"))
	require.NoError(t, err)

	// Close store
	err = store1.Close(ctx)
	require.NoError(t, err)

	// Second store instance with same path
	store2 := New(tmpDir)
	err = store2.Init(&app.App{})
	require.NoError(t, err)
	err = store2.Run(ctx)
	require.NoError(t, err)
	defer store2.Close(ctx)

	// Verify block persisted
	retrieved, err := store2.Get(ctx, block.Cid())
	require.NoError(t, err)
	assert.Equal(t, block.RawData(), retrieved.RawData())

	// Verify index persisted
	indexValue, err := store2.IndexGet(ctx, "persistent-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("persistent-value"), indexValue)
}

func TestLightFileNodeStore_ContextCancellation(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	testBlocks := createTestBlocks(t, 100)
	err := store.Add(ctx, testBlocks)
	require.NoError(t, err)

	// Prepare CIDs for GetMany
	cids := make([]cid.Cid, len(testBlocks))
	for i, block := range testBlocks {
		cids[i] = block.Cid()
	}

	// Cancel context immediately
	cancel()

	// GetMany should handle cancellation gracefully
	resultChan := store.GetMany(ctx, cids)

	count := 0
	for range resultChan {
		count++
	}

	// Should get fewer results due to cancellation
	assert.Less(t, count, len(testBlocks))
}

// Benchmarks

func BenchmarkLightFileNodeStore_Add(b *testing.B) {
	store, cleanup := setupTestStore(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	data := []byte("benchmark data")

	for i := 0; b.Loop(); i++ {
		block := createTestBlock(&testing.T{}, append(data, byte(i)))
		addErr := store.Add(ctx, []blocks.Block{block})
		if addErr != nil {
			b.Fatal(addErr)
		}
	}
}

func BenchmarkLightFileNodeStore_Get(b *testing.B) {
	store, cleanup := setupTestStore(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	blocks := createTestBlocks(&testing.T{}, 100)
	err := store.Add(ctx, blocks)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; b.Loop(); i++ {
		block := blocks[i%len(blocks)]
		_, getErr := store.Get(ctx, block.Cid())
		if getErr != nil {
			b.Fatal(getErr)
		}
	}
}

func BenchmarkLightFileNodeStore_GetMany(b *testing.B) {
	store, cleanup := setupTestStore(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	blocks := createTestBlocks(&testing.T{}, 100)
	err := store.Add(ctx, blocks)
	if err != nil {
		b.Fatal(err)
	}

	cids := make([]cid.Cid, len(blocks))
	for i, block := range blocks {
		cids[i] = block.Cid()
	}

	for b.Loop() {
		resultChan := store.GetMany(ctx, cids)
		// Consume results to measure full operation
		for range resultChan {
			_ = struct{}{}
		}
	}
}

// Test for Garbage Collection (harder to test deterministically)

func TestLightFileNodeStore_GarbageCollection(t *testing.T) {
	tmpDir := t.TempDir()
	store := New(tmpDir)
	store.cfg.gcInterval = time.Millisecond
	store.cfg.maxGCDuration = 100 * time.Millisecond

	require.NoError(t, store.Init(&app.App{}))

	ctx := context.Background()
	require.NoError(t, store.Run(ctx))
	defer store.Close(context.Background())

	// Add and delete blocks to create garbage
	for i := 0; i < 10; i++ {
		block := createTestBlock(t, fmt.Appendf(nil, "gc-test-%d", i))
		require.NoError(t, store.Add(ctx, []blocks.Block{block}))
		require.NoError(t, store.Delete(ctx, block.Cid()))
	}

	_, _, err := store.gcOnce()
	require.NoError(t, err)

	// Store remains functional after GC iteration.
	testBlock := createTestBlock(t, []byte("after-gc"))
	require.NoError(t, store.Add(ctx, []blocks.Block{testBlock}))

	retrieved, err := store.Get(ctx, testBlock.Cid())
	require.NoError(t, err)
	assert.Equal(t, testBlock.RawData(), retrieved.RawData())
}

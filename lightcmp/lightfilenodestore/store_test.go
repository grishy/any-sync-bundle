package lightfilenodestore

import (
	"context"
	"fmt"
	"sync"
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

func TestLightFileNodeStore_Add_Get_Single(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	block := createTestBlock(t, []byte("test data"))

	// Add block
	err := store.Add(ctx, []blocks.Block{block})
	require.NoError(t, err)

	// Get block
	retrieved, err := store.Get(ctx, block.Cid())
	require.NoError(t, err)
	assert.Equal(t, block.Cid(), retrieved.Cid())
	assert.Equal(t, block.RawData(), retrieved.RawData())
}

func TestLightFileNodeStore_Add_Get_Multiple(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	testBlocks := createTestBlocks(t, 10)

	// Add multiple blocks
	err := store.Add(ctx, testBlocks)
	require.NoError(t, err)

	// Get each block individually
	for _, block := range testBlocks {
		retrieved, getErr := store.Get(ctx, block.Cid())
		require.NoError(t, getErr)
		assert.Equal(t, block.Cid(), retrieved.Cid())
		assert.Equal(t, block.RawData(), retrieved.RawData())
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

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for j := range blocksPerGoroutine {
				data := fmt.Appendf(nil, "goroutine-%d-block-%d", id, j)
				block := createTestBlock(t, data)
				err := store.Add(ctx, []blocks.Block{block})
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all blocks were written
	// Note: We can't easily verify the exact count without tracking all CIDs
	// but the test should complete without panics or errors
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

	for range numGoroutines {
		go func() {
			defer wg.Done()
			for j := range readsPerGoroutine {
				// Read random block
				block := testBlocks[j%len(testBlocks)]
				retrieved, getErr := store.Get(ctx, block.Cid())
				assert.NoError(t, getErr)
				assert.Equal(t, block.RawData(), retrieved.RawData())
			}
		}()
	}

	wg.Wait()
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

	// Reader goroutine
	go func() {
		defer wg.Done()
		for i := range 20 {
			if i < 10 {
				_, _ = store.Get(ctx, testBlocks[i].Cid())
			}
			time.Sleep(time.Millisecond)
		}
	}()

	// Writer goroutine
	go func() {
		defer wg.Done()
		for i := 10; i < 20; i++ {
			_ = store.Add(ctx, []blocks.Block{testBlocks[i]})
			time.Sleep(time.Millisecond)
		}
	}()

	// GetMany goroutine
	go func() {
		defer wg.Done()
		cids := make([]cid.Cid, 10)
		for i := range 10 {
			cids[i] = testBlocks[i].Cid()
		}
		for range 5 {
			resultChan := store.GetMany(ctx, cids)
			// Consume results to avoid blocking
			for range resultChan {
				_ = struct{}{}
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	wg.Wait()
}

// Edge Cases Tests

func TestLightFileNodeStore_EmptyBlock(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	block := createTestBlock(t, []byte{})

	// Add empty block
	err := store.Add(ctx, []blocks.Block{block})
	require.NoError(t, err)

	// Retrieve empty block
	retrieved, err := store.Get(ctx, block.Cid())
	require.NoError(t, err)
	// Empty blocks may return nil or empty slice, both are valid
	assert.Empty(t, retrieved.RawData())
}

func TestLightFileNodeStore_LargeBlock(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	// Create a 2MB block
	largeData := make([]byte, 2*1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	block := createTestBlock(t, largeData)

	// Add large block
	err := store.Add(ctx, []blocks.Block{block})
	require.NoError(t, err)

	// Retrieve large block
	retrieved, err := store.Get(ctx, block.Cid())
	require.NoError(t, err)
	assert.Equal(t, largeData, retrieved.RawData())
}

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
	store := &LightFileNodeStore{
		cfg: storeConfig{
			storePath:     tmpDir,
			gcInterval:    100 * time.Millisecond, // Fast GC for testing
			gcThreshold:   0.5,
			maxGCDuration: 10 * time.Second,
		},
	}

	err := store.Init(&app.App{})
	require.NoError(t, err)

	ctx := t.Context()

	err = store.Run(ctx)
	require.NoError(t, err)
	defer store.Close(context.Background())

	// Add and delete blocks to create garbage
	for i := range 10 {
		block := createTestBlock(t, fmt.Appendf(nil, "gc-test-%d", i))
		addErr := store.Add(ctx, []blocks.Block{block})
		require.NoError(t, addErr)
		delErr := store.Delete(ctx, block.Cid())
		require.NoError(t, delErr)
	}

	// Wait for GC to run
	time.Sleep(200 * time.Millisecond)

	// Test passes if no panic and store still functional
	testBlock := createTestBlock(t, []byte("after-gc"))
	err = store.Add(ctx, []blocks.Block{testBlock})
	require.NoError(t, err)

	retrieved, err := store.Get(ctx, testBlock.Cid())
	require.NoError(t, err)
	assert.Equal(t, testBlock.RawData(), retrieved.RawData())
}

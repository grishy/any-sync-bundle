package lightfilenodestore

import (
	"bytes"
	"context"
	"fmt"
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

func setupTestStore(t *testing.T) *LightFileNodeStore {
	t.Helper()

	store := New(t.TempDir())
	require.NoError(t, store.Init(&app.App{}))
	require.NoError(t, store.Run(t.Context()))
	t.Cleanup(func() {
		require.NoError(t, store.Close(context.Background()))
	})
	return store
}

func createTestBlock(t *testing.T, data []byte) blocks.Block {
	t.Helper()

	hash, err := multihash.Sum(data, multihash.SHA2_256, -1)
	require.NoError(t, err)

	block, err := blocks.NewBlockWithCid(data, cid.NewCidV1(cid.Raw, hash))
	require.NoError(t, err)
	return block
}

func createTestBlocks(t *testing.T, prefix string, count int) []blocks.Block {
	t.Helper()

	result := make([]blocks.Block, count)
	for index := range count {
		result[index] = createTestBlock(
			t,
			fmt.Appendf(nil, "%s-%d", prefix, index),
		)
	}
	return result
}

func TestLightFileNodeStoreBlockRoundTrip(t *testing.T) {
	store := setupTestStore(t)
	ctx := t.Context()
	expected := createTestBlocks(t, "round-trip", 3)

	require.NoError(t, store.Add(ctx, expected))

	for _, block := range expected {
		actual, err := store.Get(ctx, block.Cid())
		require.NoError(t, err)
		assert.Equal(t, block.RawData(), actual.RawData())
	}
}

func TestLightFileNodeStoreMissingBlock(t *testing.T) {
	store := setupTestStore(t)
	missing := createTestBlock(t, []byte("missing"))

	_, err := store.Get(t.Context(), missing.Cid())

	assert.ErrorIs(t, err, fileblockstore.ErrCIDNotFound)
}

// GetMany owns two semantics not supplied by Badger: it streams blocks over a
// channel and omits missing CIDs without failing the remaining request.
func TestLightFileNodeStoreGetMany(t *testing.T) {
	store := setupTestStore(t)
	ctx := t.Context()
	existing := createTestBlocks(t, "existing", 3)
	missing := createTestBlocks(t, "missing", 2)
	require.NoError(t, store.Add(ctx, existing))

	requested := make([]cid.Cid, 0, len(existing)+len(missing))
	for _, block := range existing {
		requested = append(requested, block.Cid())
	}
	for _, block := range missing {
		requested = append(requested, block.Cid())
	}

	actual := make(map[string]blocks.Block)
	for block := range store.GetMany(ctx, requested) {
		actual[block.Cid().String()] = block
	}

	assert.Len(t, actual, len(existing))
	for _, block := range existing {
		delivered, ok := actual[block.Cid().String()]
		require.True(t, ok, "missing expected CID %s", block.Cid())
		assert.Equal(t, block.RawData(), delivered.RawData())
	}
	for _, block := range missing {
		assert.NotContains(t, actual, block.Cid().String())
	}
}

func TestLightFileNodeStoreGetManyStopsOnCancellation(t *testing.T) {
	store := setupTestStore(t)
	ctx, cancel := context.WithCancel(t.Context())
	expected := createTestBlocks(t, "cancelled", 3)
	require.NoError(t, store.Add(ctx, expected))

	requested := make([]cid.Cid, len(expected))
	for index, block := range expected {
		requested[index] = block.Cid()
	}
	cancel()

	delivered := 0
	for range store.GetMany(ctx, requested) {
		delivered++
	}
	assert.Zero(t, delivered)
}

func TestLightFileNodeStoreDelete(t *testing.T) {
	t.Run("single block", func(t *testing.T) {
		store := setupTestStore(t)
		ctx := t.Context()
		block := createTestBlock(t, []byte("single-delete"))
		require.NoError(t, store.Add(ctx, []blocks.Block{block}))

		require.NoError(t, store.Delete(ctx, block.Cid()))

		_, err := store.Get(ctx, block.Cid())
		assert.ErrorIs(t, err, fileblockstore.ErrCIDNotFound)
	})

	t.Run("block batch", func(t *testing.T) {
		store := setupTestStore(t)
		ctx := t.Context()
		batch := createTestBlocks(t, "batch-delete", 3)
		require.NoError(t, store.Add(ctx, batch))

		cids := make([]cid.Cid, len(batch))
		for index, block := range batch {
			cids[index] = block.Cid()
		}
		require.NoError(t, store.DeleteMany(ctx, cids))

		for _, blockCID := range cids {
			_, err := store.Get(ctx, blockCID)
			assert.ErrorIs(t, err, fileblockstore.ErrCIDNotFound)
		}
	})
}

func TestLightFileNodeStoreIndexLifecycle(t *testing.T) {
	store := setupTestStore(t)
	ctx := t.Context()
	const key = "index-key"
	value := []byte("index-value")

	actual, err := store.IndexGet(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, actual)

	require.NoError(t, store.IndexPut(ctx, key, value))
	actual, err = store.IndexGet(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, actual)

	require.NoError(t, store.IndexDelete(ctx, key))
	actual, err = store.IndexGet(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, actual)
}

func TestLightFileNodeStoreIndexKeyPrefixIsolation(t *testing.T) {
	store := setupTestStore(t)
	ctx := t.Context()
	block := createTestBlock(t, []byte("block-value"))
	indexKey := block.Cid().String()
	indexValue := []byte("index-value")

	require.NoError(t, store.Add(ctx, []blocks.Block{block}))
	require.NoError(t, store.IndexPut(ctx, indexKey, indexValue))

	actualBlock, err := store.Get(ctx, block.Cid())
	require.NoError(t, err)
	assert.Equal(t, block.RawData(), actualBlock.RawData())

	actualIndex, err := store.IndexGet(ctx, indexKey)
	require.NoError(t, err)
	assert.Equal(t, indexValue, actualIndex)
}

// Reopening the same directory verifies the adapter's durability boundary,
// including both independently prefixed data classes.
func TestLightFileNodeStorePersistence(t *testing.T) {
	dir := t.TempDir()
	ctx := t.Context()
	block := createTestBlock(t, []byte("persistent-block"))
	const indexKey = "persistent-index"
	indexValue := []byte("persistent-value")

	first := New(dir)
	require.NoError(t, first.Init(&app.App{}))
	require.NoError(t, first.Run(ctx))
	require.NoError(t, first.Add(ctx, []blocks.Block{block}))
	require.NoError(t, first.IndexPut(ctx, indexKey, indexValue))
	require.NoError(t, first.Close(ctx))

	second := New(dir)
	require.NoError(t, second.Init(&app.App{}))
	require.NoError(t, second.Run(ctx))
	t.Cleanup(func() {
		require.NoError(t, second.Close(context.Background()))
	})

	actualBlock, err := second.Get(ctx, block.Cid())
	require.NoError(t, err)
	assert.Equal(t, block.RawData(), actualBlock.RawData())

	actualIndex, err := second.IndexGet(ctx, indexKey)
	require.NoError(t, err)
	assert.Equal(t, indexValue, actualIndex)
}

// GC is dependency-owned, but invoking our bounded loop once protects against
// adapter configuration that could corrupt or close the live store.
func TestLightFileNodeStoreGarbageCollection(t *testing.T) {
	store := New(t.TempDir())
	store.cfg.gcInterval = 24 * time.Hour
	store.cfg.maxGCDuration = 100 * time.Millisecond
	ctx := t.Context()
	require.NoError(t, store.Init(&app.App{}))
	require.NoError(t, store.Run(ctx))
	t.Cleanup(func() {
		require.NoError(t, store.Close(context.Background()))
	})

	for index := range 10 {
		block := createTestBlock(t, fmt.Appendf(nil, "garbage-%d", index))
		require.NoError(t, store.Add(ctx, []blocks.Block{block}))
		require.NoError(t, store.Delete(ctx, block.Cid()))
	}

	_, _, err := store.gcOnce()
	require.NoError(t, err)

	expected := createTestBlock(t, []byte("after-gc"))
	require.NoError(t, store.Add(ctx, []blocks.Block{expected}))
	actual, err := store.Get(ctx, expected.Cid())
	require.NoError(t, err)
	assert.True(t, bytes.Equal(expected.RawData(), actual.RawData()))
}

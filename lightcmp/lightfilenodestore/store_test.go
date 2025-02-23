package lightfilenodestore

import (
	"context"
	"math/rand/v2"
	"testing"

	"github.com/anyproto/any-sync-filenode/testutil"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/commonfile/fileproto/fileprotoerr"
	"github.com/dgraph-io/badger/v4"
	blocks "github.com/ipfs/go-block-format"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestBlockPushAndGet(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	spaceId := testutil.NewRandSpaceId()
	blk := testutil.NewRandBlock(1024)

	// Push block
	err := fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		return fx.storeSrv.PushBlock(txn, spaceId, blk)
	})
	require.NoError(t, err)

	// Get block and verify
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		// Check block data
		block, err := fx.storeSrv.GetBlock(txn, blk.Cid())
		require.NoError(t, err)
		require.Equal(t, blk.RawData(), block.Data())

		// Check CID metadata
		cid := NewCidObj(spaceId, blk.Cid())
		err = cid.populateValue(txn)
		require.NoError(t, err)
		require.Equal(t, uint32(1), cid.refCount)
		require.Equal(t, uint32(len(blk.RawData())), cid.sizeByte)
		return nil
	})
	require.NoError(t, err)

	// Try to get non-existent block
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		_, err := fx.storeSrv.GetBlock(txn, testutil.NewRandCid())
		require.ErrorIs(t, err, fileprotoerr.ErrCIDNotFound)
		return nil
	})
	require.NoError(t, err)
}

func TestBlockPushTwice(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	spaceId := testutil.NewRandSpaceId()
	blk1 := testutil.NewRandBlock(1024)
	// Only to get a different block data
	blk2Data := testutil.NewRandBlock(2048)
	blk2, err := blocks.NewBlockWithCid(blk2Data.RawData(), blk1.Cid())
	require.Nil(t, err)

	// First push
	err = fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		return fx.storeSrv.PushBlock(txn, spaceId, blk1)
	})
	require.NoError(t, err)

	// Verify block exists and CID metadata
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		// Check block data
		block, err := fx.storeSrv.GetBlock(txn, blk1.Cid())
		require.NoError(t, err)
		require.Equal(t, blk1.RawData(), block.Data())

		// Check CID metadata
		cid := NewCidObj(spaceId, blk1.Cid())
		err = cid.populateValue(txn)
		require.NoError(t, err)
		require.Equal(t, uint32(1), cid.refCount)
		require.Equal(t, uint32(len(blk1.RawData())), cid.sizeByte)
		return nil
	})
	require.NoError(t, err)

	// Push same block again
	err = fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		return fx.storeSrv.PushBlock(txn, spaceId, blk2)
	})
	require.NoError(t, err)

	// Verify block data and CID metadata remain unchanged
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		// Check block data
		block, err := fx.storeSrv.GetBlock(txn, blk2.Cid())
		require.NoError(t, err)
		require.Equal(t, blk2.RawData(), block.Data())

		// Check CID metadata
		cid := NewCidObj(spaceId, blk2.Cid())
		err = cid.populateValue(txn)
		require.NoError(t, err)
		require.Equal(t, uint32(1), cid.refCount)
		require.Equal(t, uint32(len(blk2.RawData())), cid.sizeByte)
		return nil
	})
	require.NoError(t, err)
}

func TestBlockPushParallel(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	spaceId := testutil.NewRandSpaceId()

	// Generate different blocks
	const numBlocks = 100
	blks := make([]blocks.Block, numBlocks)
	for i := 0; i < numBlocks; i++ {
		blks[i] = testutil.NewRandBlock(rand.IntN(1024) + 1)
	}

	// Push blocks concurrently
	var g errgroup.Group

	for i := 0; i < numBlocks; i++ {
		blk := blks[i] // Capture block for goroutine
		g.Go(func() error {
			return fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
				return fx.storeSrv.PushBlock(txn, spaceId, blk)
			})
		})
	}

	require.NoError(t, g.Wait())

	// Verify all blocks exist and have correct data
	err := fx.storeSrv.TxView(func(txn *badger.Txn) error {
		for _, blk := range blks {
			block, err := fx.storeSrv.GetBlock(txn, blk.Cid())
			require.NoError(t, err)
			require.Equal(t, blk.RawData(), block.Data())
		}
		return nil
	})
	require.NoError(t, err)
}

func TestLightFileNodeStore_HasCIDInSpace(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	spaceId := testutil.NewRandSpaceId()
	blk := testutil.NewRandBlock(1024)

	// Initially CID should not exist
	err := fx.storeSrv.TxView(func(txn *badger.Txn) error {
		exists, err := fx.storeSrv.HasCIDInSpace(txn, spaceId, blk.Cid())
		require.NoError(t, err)
		require.False(t, exists)
		return nil
	})
	require.NoError(t, err)

	// Push block
	err = fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		return fx.storeSrv.PushBlock(txn, spaceId, blk)
	})
	require.NoError(t, err)

	// Now CID should exist
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		exists, err := fx.storeSrv.HasCIDInSpace(txn, spaceId, blk.Cid())
		require.NoError(t, err)
		require.True(t, exists)
		return nil
	})
	require.NoError(t, err)

	// Check CID doesn't exist in different space
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		exists, err := fx.storeSrv.HasCIDInSpace(txn, testutil.NewRandSpaceId(), blk.Cid())
		require.NoError(t, err)
		require.False(t, exists)
		return nil
	})
	require.NoError(t, err)

	// Check non-existent CID
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		exists, err := fx.storeSrv.HasCIDInSpace(txn, spaceId, testutil.NewRandCid())
		require.NoError(t, err)
		require.False(t, exists)
		return nil
	})
	require.NoError(t, err)
}

func TestLightFileNodeStore_HadCID(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	spaceId := testutil.NewRandSpaceId()
	blk := testutil.NewRandBlock(1024)

	// Initially CID should not exist
	err := fx.storeSrv.TxView(func(txn *badger.Txn) error {
		exists, err := fx.storeSrv.HadCID(txn, blk.Cid())
		require.NoError(t, err)
		require.False(t, exists)
		return nil
	})
	require.NoError(t, err)

	// Push block
	err = fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		return fx.storeSrv.PushBlock(txn, spaceId, blk)
	})
	require.NoError(t, err)

	// Now CID should exist
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		exists, err := fx.storeSrv.HadCID(txn, blk.Cid())
		require.NoError(t, err)
		require.True(t, exists)
		return nil
	})
	require.NoError(t, err)

	// Check non-existent CID
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		exists, err := fx.storeSrv.HadCID(txn, testutil.NewRandCid())
		require.NoError(t, err)
		require.False(t, exists)
		return nil
	})
	require.NoError(t, err)
}

func TestLightFileNodeStore_GetSpace(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	spaceId := testutil.NewRandSpaceId()

	// Initially space should be empty
	err := fx.storeSrv.TxView(func(txn *badger.Txn) error {
		space, err := fx.storeSrv.GetSpace(txn, spaceId)
		require.NoError(t, err)
		require.Equal(t, uint64(0), space.LimitBytes())
		require.Equal(t, uint64(0), space.CidsCount())
		require.Equal(t, uint64(0), space.FilesCount())
		require.Equal(t, uint64(0), space.SpaceUsageBytes())
		return nil
	})
	require.NoError(t, err)

	// Create space with some values
	err = fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		space := NewSpaceObj(spaceId).
			WithLimitBytes(1024).
			WithSpaceUsageBytes(256).
			WithCidsCount(10).
			WithFilesCount(5)
		return space.write(txn)
	})
	require.NoError(t, err)

	// Verify space values
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		space, err := fx.storeSrv.GetSpace(txn, spaceId)
		require.NoError(t, err)
		require.Equal(t, uint64(1024), space.LimitBytes())
		require.Equal(t, uint64(256), space.SpaceUsageBytes())
		require.Equal(t, uint64(10), space.CidsCount())
		require.Equal(t, uint64(5), space.FilesCount())
		return nil
	})
	require.NoError(t, err)

	// Check non-existent space returns empty object
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		space, err := fx.storeSrv.GetSpace(txn, testutil.NewRandSpaceId())
		require.NoError(t, err)
		require.Equal(t, uint64(0), space.LimitBytes())
		require.Equal(t, uint64(0), space.SpaceUsageBytes())
		require.Equal(t, uint64(0), space.CidsCount())
		require.Equal(t, uint64(0), space.FilesCount())
		return nil
	})
	require.NoError(t, err)
}

func TestLightFileNodeStore_GetGroup(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	groupId := testutil.NewRandSpaceId()

	// Initially group should have default limit and zero counters
	err := fx.storeSrv.TxView(func(txn *badger.Txn) error {
		group, err := fx.storeSrv.GetGroup(txn, groupId)
		require.NoError(t, err)
		require.Equal(t, uint64(defaultLimitBytes), group.LimitBytes())
		require.Equal(t, uint64(0), group.TotalUsageBytes())
		require.Equal(t, uint64(0), group.TotalCidsCount())
		return nil
	})
	require.NoError(t, err)

	// Create group with some values
	err = fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		group := NewGroupObj(groupId).
			WithLimitBytes(1024).
			WithTotalUsageBytes(512).
			WithTotalCidsCount(10)
		return group.write(txn)
	})
	require.NoError(t, err)

	// Verify group values
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		group, err := fx.storeSrv.GetGroup(txn, groupId)
		require.NoError(t, err)
		require.Equal(t, uint64(1024), group.LimitBytes())
		require.Equal(t, uint64(512), group.TotalUsageBytes())
		require.Equal(t, uint64(10), group.TotalCidsCount())
		return nil
	})
	require.NoError(t, err)

	// Check non-existent group returns object with default limit
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		group, err := fx.storeSrv.GetGroup(txn, testutil.NewRandSpaceId())
		require.NoError(t, err)
		require.Equal(t, uint64(defaultLimitBytes), group.LimitBytes())
		require.Equal(t, uint64(0), group.TotalUsageBytes())
		require.Equal(t, uint64(0), group.TotalCidsCount())
		return nil
	})
	require.NoError(t, err)
}

func TestLightFileNodeStore_GetFile(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	spaceId := testutil.NewRandSpaceId()
	fileId := testutil.NewRandSpaceId()

	// Initially file should have zero counters
	err := fx.storeSrv.TxView(func(txn *badger.Txn) error {
		file, err := fx.storeSrv.GetFile(txn, spaceId, fileId)
		require.NoError(t, err)
		require.Equal(t, uint64(0), file.UsageBytes())
		require.Equal(t, uint32(0), file.CidsCount())
		return nil
	})
	require.NoError(t, err)

	// Create file with some values
	err = fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		file := NewFileObj(spaceId, fileId).
			WithUsageBytes(512).
			WithCidsCount(10)
		return file.write(txn)
	})
	require.NoError(t, err)

	// Verify file values
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		file, err := fx.storeSrv.GetFile(txn, spaceId, fileId)
		require.NoError(t, err)
		require.Equal(t, uint64(512), file.UsageBytes())
		require.Equal(t, uint32(10), file.CidsCount())
		return nil
	})
	require.NoError(t, err)

	// Check non-existent file returns empty object
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		file, err := fx.storeSrv.GetFile(txn, spaceId, testutil.NewRandSpaceId())
		require.NoError(t, err)
		require.Equal(t, uint64(0), file.UsageBytes())
		require.Equal(t, uint32(0), file.CidsCount())
		return nil
	})
	require.NoError(t, err)
}

func TestLightFileNodeStore_CreateLinks(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	spaceId := testutil.NewRandSpaceId()
	fileId := testutil.NewRandSpaceId()
	groupId := testutil.NewRandSpaceId()
	blk := testutil.NewRandBlock(1024)

	// Initially links should not exist
	err := fx.storeSrv.TxView(func(txn *badger.Txn) error {
		// Check file-block link
		linkFB := NewLinkFileBlockObj(spaceId, fileId, blk.Cid())
		exists, err := linkFB.exists(txn)
		require.NoError(t, err)
		require.False(t, exists)

		// Check group-space link
		linkGS := NewLinkGroupSpaceObj(groupId, spaceId)
		exists, err = linkGS.exists(txn)
		require.NoError(t, err)
		require.False(t, exists)
		return nil
	})
	require.NoError(t, err)

	// Create file-block link
	err = fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		return fx.storeSrv.CreateLinkFileBlock(txn, spaceId, fileId, blk)
	})
	require.NoError(t, err)

	// Create group-space link
	err = fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		return fx.storeSrv.CreateLinkGroupSpace(txn, groupId, spaceId)
	})
	require.NoError(t, err)

	// Verify links exist
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		// Check file-block link
		linkFB := NewLinkFileBlockObj(spaceId, fileId, blk.Cid())
		exists, err := linkFB.exists(txn)
		require.NoError(t, err)
		require.True(t, exists)

		// Check group-space link
		linkGS := NewLinkGroupSpaceObj(groupId, spaceId)
		exists, err = linkGS.exists(txn)
		require.NoError(t, err)
		require.True(t, exists)
		return nil
	})
	require.NoError(t, err)
}

//
// Fixtures
//

type fixture struct {
	a        *app.App
	cfgSrc   *configServiceMock
	storeSrv *lightFileNodeStore
}

func (fx *fixture) Finish(t *testing.T) {
	require.NoError(t, fx.a.Close(context.Background()))
}

func newFixture(t *testing.T) *fixture {
	tempDir := t.TempDir()

	fx := &fixture{
		a: new(app.App),
		cfgSrc: &configServiceMock{
			InitFunc: func(a *app.App) error {
				return nil
			},
			NameFunc: func() string {
				return "config"
			},
			GetFilenodeStoreDirFunc: func() string {
				return tempDir
			},
			GetFilenodeDefaultLimitBytesFunc: func() uint64 {
				return defaultLimitBytes
			},
		},
		storeSrv: New().(*lightFileNodeStore),
	}

	fx.a.
		Register(fx.cfgSrc).
		Register(fx.storeSrv)

	t.Logf("start app with temp dir: %s", tempDir)
	require.NoError(t, fx.a.Start(context.Background()))
	return fx
}

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

func TestBlockPushGet(t *testing.T) {
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
		block, err := fx.storeSrv.GetBlock(txn, blk.Cid(), spaceId, false)
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
		_, err := fx.storeSrv.GetBlock(txn, testutil.NewRandCid(), spaceId, false)
		require.ErrorIs(t, err, fileprotoerr.ErrCIDNotFound)
		return nil
	})
	require.NoError(t, err)

	// Try to get block with wait=true should panic
	require.Panics(t, func() {
		_ = fx.storeSrv.TxView(func(txn *badger.Txn) error {
			_, err := fx.storeSrv.GetBlock(txn, blk.Cid(), spaceId, true)
			return err
		})
	})
}

func TestBlockPushTwice(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	spaceId := testutil.NewRandSpaceId()
	blk := testutil.NewRandBlock(1024)

	// First push
	err := fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		return fx.storeSrv.PushBlock(txn, spaceId, blk)
	})
	require.NoError(t, err)

	// Verify block exists and CID metadata
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		// Check block data
		block, err := fx.storeSrv.GetBlock(txn, blk.Cid(), spaceId, false)
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

	// Push same block again
	err = fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		return fx.storeSrv.PushBlock(txn, spaceId, blk)
	})
	require.NoError(t, err)

	// Verify block data and CID metadata remain unchanged
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		// Check block data
		block, err := fx.storeSrv.GetBlock(txn, blk.Cid(), spaceId, false)
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
}

func TestBlockPushParallel(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	spaceId := testutil.NewRandSpaceId()

	// Generate different blocks
	const numBlocks = 500
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
			block, err := fx.storeSrv.GetBlock(txn, blk.Cid(), spaceId, false)
			require.NoError(t, err)
			require.Equal(t, blk.RawData(), block.Data())
		}
		return nil
	})
	require.NoError(t, err)
}

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

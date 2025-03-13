package lightfilenodeindex

import (
	"context"
	"testing"
	"time"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/testutil"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/util/crypto"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"

	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex/indexpb"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

type testFixture struct {
	app       *app.App
	srvConfig *configServiceMock
	srvStore  *lightfilenodestore.StoreServiceMock
	srvIndex  IndexService
}

// cleanup will be called automatically when the test ends
func (f *testFixture) cleanup(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, f.app.Close(ctx))
}

func newTestFixture(t *testing.T) *testFixture {
	t.Helper()

	f := &testFixture{
		app: new(app.App),
		srvConfig: &configServiceMock{
			GetFilenodeDefaultLimitBytesFunc: func() uint64 { return 1024 },
			InitFunc:                         func(a *app.App) error { return nil },
			NameFunc:                         func() string { return "config" },
		},
		srvStore: &lightfilenodestore.StoreServiceMock{
			InitFunc: func(a *app.App) error {
				return nil
			},
			NameFunc: func() string {
				return lightfilenodestore.CName
			},
			RunFunc: func(ctx context.Context) error {
				return nil
			},
			CloseFunc: func(ctx context.Context) error {
				return nil
			},
			// Default for tests
			TxViewFunc: func(f func(txn *badger.Txn) error) error {
				return f(nil)
			},
			TxUpdateFunc: func(f func(txn *badger.Txn) error) error {
				return f(nil)
			},
		},
		srvIndex: New(),
	}

	f.app.
		Register(f.srvConfig).
		Register(f.srvStore).
		Register(f.srvIndex)

	require.NoError(t, f.app.Start(context.Background()))

	t.Cleanup(func() { f.cleanup(t) })
	return f
}

func newRandKey() index.Key {
	_, pubKey, _ := crypto.GenerateRandomEd25519KeyPair()
	return index.Key{
		SpaceId: testutil.NewRandSpaceId(),
		GroupId: pubKey.Account(),
	}
}

func TestIndexServiceCID(t *testing.T) {
	t.Run("check existence after add", func(t *testing.T) {
		var (
			key      = newRandKey()
			otherKey = newRandKey()
			b        = testutil.NewRandBlock(1024)
			c        = b.Cid()
		)

		f := newTestFixture(t)

		f.srvStore.PushIndexLogFunc = func(txn *badger.Txn, logData []byte) error {
			require.NotEmpty(t, logData)
			return nil
		}

		// Before
		require.False(t, f.srvIndex.HadCID(c))
		require.False(t, f.srvIndex.HasCIDInSpace(key, c))
		require.False(t, f.srvIndex.HasCIDInSpace(otherKey, c))

		// Add
		cidOp := &indexpb.CidAddOperation{}
		cidOp.SetCid(c.String())
		cidOp.SetDataSize(uint64(len(b.RawData())))

		op := &indexpb.Operation{}
		op.SetCidAdd(cidOp)

		require.NoError(t, f.srvIndex.Modify(nil, key, op))

		// After
		require.True(t, f.srvIndex.HadCID(c))
		require.True(t, f.srvIndex.HasCIDInSpace(key, c))
		require.False(t, f.srvIndex.HasCIDInSpace(otherKey, c))
	})
}

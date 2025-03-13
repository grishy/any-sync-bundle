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
			PushIndexLogFunc: func(txn *badger.Txn, logData []byte) error {
				require.NotEmpty(t, logData)
				return nil
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

func TestCidAddOperation(t *testing.T) {
	t.Run("basic CID add", func(t *testing.T) {
		var (
			key    = newRandKey()
			b      = testutil.NewRandBlock(1024)
			c      = b.Cid()
			fileId = "test-file-id"
		)

		f := newTestFixture(t)

		// Before
		require.False(t, f.srvIndex.HadCID(c))
		require.False(t, f.srvIndex.HasCIDInSpace(key, c))

		// Operation
		cidOp := &indexpb.CidAddOperation{}
		cidOp.SetCid(c.String())
		cidOp.SetFileId(fileId)
		cidOp.SetDataSize(uint64(len(b.RawData())))

		op := &indexpb.Operation{}
		op.SetCidAdd(cidOp)

		require.NoError(t, f.srvIndex.Modify(nil, key, op))

		// After
		require.True(t, f.srvIndex.HadCID(c), "CID should exist after add")
		require.True(t, f.srvIndex.HasCIDInSpace(key, c), "CID should exist in space after add")

		// Verify file info was updated
		fileInfos := f.srvIndex.FileInfo(key, fileId)
		require.Len(t, fileInfos, 1)
		require.Equal(t, fileId, fileInfos[0].FileId)
		require.Equal(t, uint32(1), fileInfos[0].CidsCount)
		require.Equal(t, uint64(len(b.RawData())), fileInfos[0].UsageBytes, "File usage bytes should match block size")

		// Verify space info was updated
		spaceInfo := f.srvIndex.SpaceInfo(key)
		require.Equal(t, uint64(len(b.RawData())), spaceInfo.TotalUsageBytes, "Space usage bytes should match block size")
		require.Equal(t, uint64(1), spaceInfo.CidsCount)
		require.Equal(t, uint64(1), spaceInfo.FilesCount)

		// Verify group statistics were updated
		groupInfo := f.srvIndex.GroupInfo(key.GroupId)
		require.Equal(t, uint64(len(b.RawData())), groupInfo.TotalUsageBytes, "Group usage bytes should match block size")
		require.Equal(t, uint64(1), groupInfo.TotalCidsCount)
	})

	t.Run("add same CID to multiple files", func(t *testing.T) {
		var (
			key     = newRandKey()
			b       = testutil.NewRandBlock(1024)
			c       = b.Cid()
			fileId1 = "test-file-id-1"
			fileId2 = "test-file-id-2"
		)

		f := newTestFixture(t)

		// Add CID to first file
		cidOp1 := &indexpb.CidAddOperation{}
		cidOp1.SetCid(c.String())
		cidOp1.SetFileId(fileId1)
		cidOp1.SetDataSize(uint64(len(b.RawData())))

		op1 := &indexpb.Operation{}
		op1.SetCidAdd(cidOp1)

		require.NoError(t, f.srvIndex.Modify(nil, key, op1))

		// Add same CID to second file
		cidOp2 := &indexpb.CidAddOperation{}
		cidOp2.SetCid(c.String())
		cidOp2.SetFileId(fileId2)
		cidOp2.SetDataSize(uint64(len(b.RawData())))

		op2 := &indexpb.Operation{}
		op2.SetCidAdd(cidOp2)

		require.NoError(t, f.srvIndex.Modify(nil, key, op2))

		// Check that both files have the CID
		fileInfos := f.srvIndex.FileInfo(key, fileId1, fileId2)
		require.Len(t, fileInfos, 2)

		// Each file should have 1 CID
		require.Equal(t, uint32(1), fileInfos[0].CidsCount)
		require.Equal(t, uint32(1), fileInfos[1].CidsCount)

		// Each file should have the block size worth of usage
		require.Equal(t, uint64(len(b.RawData())), fileInfos[0].UsageBytes)
		require.Equal(t, uint64(len(b.RawData())), fileInfos[1].UsageBytes)

		// Verify space info
		spaceInfo := f.srvIndex.SpaceInfo(key)
		require.Equal(t, uint64(len(b.RawData())*2), spaceInfo.TotalUsageBytes, "Space should count usage from both files")
		require.Equal(t, uint64(1), spaceInfo.CidsCount)
		require.Equal(t, uint64(2), spaceInfo.FilesCount)
	})

	t.Run("attempt to add CID with incorrect size", func(t *testing.T) {
		var (
			key    = newRandKey()
			b      = testutil.NewRandBlock(1024)
			c      = b.Cid()
			fileId = "test-file-id"
		)

		f := newTestFixture(t)

		// Add CID with correct size first
		cidOp1 := &indexpb.CidAddOperation{}
		cidOp1.SetCid(c.String())
		cidOp1.SetFileId(fileId)
		cidOp1.SetDataSize(uint64(len(b.RawData())))

		op1 := &indexpb.Operation{}
		op1.SetCidAdd(cidOp1)

		require.NoError(t, f.srvIndex.Modify(nil, key, op1))

		// Try to add the same CID but with different size
		cidOp2 := &indexpb.CidAddOperation{}
		cidOp2.SetCid(c.String())
		cidOp2.SetFileId("another-file")
		cidOp2.SetDataSize(uint64(len(b.RawData()) + 100)) // Different size

		op2 := &indexpb.Operation{}
		op2.SetCidAdd(cidOp2)

		err := f.srvIndex.Modify(nil, key, op2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "block size mismatch", "Error should mention size mismatch")

		// Verify original file was not affected
		fileInfos := f.srvIndex.FileInfo(key, fileId)
		require.Len(t, fileInfos, 1)
		require.Equal(t, uint32(1), fileInfos[0].CidsCount)
	})

	t.Run("add CID with invalid CID string", func(t *testing.T) {
		var (
			key    = newRandKey()
			fileId = "test-file-id"
		)

		f := newTestFixture(t)

		// Try with invalid CID
		cidOp := &indexpb.CidAddOperation{}
		cidOp.SetCid("not-a-valid-cid")
		cidOp.SetFileId(fileId)
		cidOp.SetDataSize(1024)

		op := &indexpb.Operation{}
		op.SetCidAdd(cidOp)

		err := f.srvIndex.Modify(nil, key, op)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidCID)
	})

	t.Run("add multiple CIDs to one file", func(t *testing.T) {
		var (
			key    = newRandKey()
			b1     = testutil.NewRandBlock(1024)
			b2     = testutil.NewRandBlock(2048)
			c1     = b1.Cid()
			c2     = b2.Cid()
			fileId = "test-file-id"
		)

		f := newTestFixture(t)

		// Add first CID
		cidOp1 := &indexpb.CidAddOperation{}
		cidOp1.SetCid(c1.String())
		cidOp1.SetFileId(fileId)
		cidOp1.SetDataSize(uint64(len(b1.RawData())))

		op1 := &indexpb.Operation{}
		op1.SetCidAdd(cidOp1)

		require.NoError(t, f.srvIndex.Modify(nil, key, op1))

		// Add second CID to same file
		cidOp2 := &indexpb.CidAddOperation{}
		cidOp2.SetCid(c2.String())
		cidOp2.SetFileId(fileId)
		cidOp2.SetDataSize(uint64(len(b2.RawData())))

		op2 := &indexpb.Operation{}
		op2.SetCidAdd(cidOp2)

		require.NoError(t, f.srvIndex.Modify(nil, key, op2))

		// Verify file has both CIDs
		fileInfos := f.srvIndex.FileInfo(key, fileId)
		require.Len(t, fileInfos, 1)
		require.Equal(t, uint32(2), fileInfos[0].CidsCount)
		require.Equal(t, uint64(len(b1.RawData())+len(b2.RawData())), fileInfos[0].UsageBytes, "File usage should be sum of both blocks")

		// Verify space info shows correct counts
		spaceInfo := f.srvIndex.SpaceInfo(key)
		require.Equal(t, uint64(len(b1.RawData())+len(b2.RawData())), spaceInfo.TotalUsageBytes, "Space should have correct total usage")
		require.Equal(t, uint64(2), spaceInfo.CidsCount)
		require.Equal(t, uint64(1), spaceInfo.FilesCount)

		// Verify group info
		groupInfo := f.srvIndex.GroupInfo(key.GroupId)
		require.Equal(t, uint64(len(b1.RawData())+len(b2.RawData())), groupInfo.TotalUsageBytes, "Group should have correct total usage")
	})
}

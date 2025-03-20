package lightfilenodeindex

import (
	"context"
	"testing"
	"time"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/util/crypto"
	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex/indexpb"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
	"github.com/grishy/any-sync-bundle/testutil"
)

type testFixture struct {
	app       *app.App
	srvConfig *configServiceMock
	srvStore  *lightfilenodestore.StoreServiceMock
	srvIndex  *lightfileindex
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
		require.Equal(t, 1, int(fileInfos[0].CidsCount))
		require.Equal(t, len(b.RawData()), int(fileInfos[0].UsageBytes), "File usage bytes should match cidBlock size")

		// Verify space info was updated
		spaceInfo := f.srvIndex.SpaceInfo(key)
		require.Equal(t, len(b.RawData()), int(spaceInfo.TotalUsageBytes), "Space usage bytes should match cidBlock size")
		require.Equal(t, 1, int(spaceInfo.CidsCount))
		require.Equal(t, 1, int(spaceInfo.FilesCount))

		// Verify group statistics were updated
		groupInfo := f.srvIndex.GroupInfo(key.GroupId)
		require.Equal(t, len(b.RawData()), int(groupInfo.TotalUsageBytes), "Group usage bytes should match cidBlock size")
		require.Equal(t, 1, int(groupInfo.TotalCidsCount))
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
		require.Equal(t, 1, int(fileInfos[0].CidsCount))
		require.Equal(t, 1, int(fileInfos[1].CidsCount))

		// Each file should have the cidBlock size worth of usage
		require.Equal(t, len(b.RawData()), int(fileInfos[0].UsageBytes))
		require.Equal(t, len(b.RawData()), int(fileInfos[1].UsageBytes))

		// Verify space info
		spaceInfo := f.srvIndex.SpaceInfo(key)
		require.Equal(t, len(b.RawData()), int(spaceInfo.TotalUsageBytes), "Space should count only CID directly")
		require.Equal(t, 1, int(spaceInfo.CidsCount))
		require.Equal(t, 2, int(spaceInfo.FilesCount))
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
		require.Contains(t, err.Error(), "cidBlock size mismatch", "Error should mention size mismatch")

		// Verify original file was not affected
		fileInfos := f.srvIndex.FileInfo(key, fileId)
		require.Len(t, fileInfos, 1)
		require.Equal(t, 1, int(fileInfos[0].CidsCount))
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
		require.Equal(t, 2, int(fileInfos[0].CidsCount))
		require.Equal(t, len(b1.RawData())+len(b2.RawData()), int(fileInfos[0].UsageBytes), "File usage should be sum of both blocks")

		// Verify space info shows correct counts
		spaceInfo := f.srvIndex.SpaceInfo(key)
		require.Equal(t, len(b1.RawData())+len(b2.RawData()), int(spaceInfo.TotalUsageBytes), "Space should have correct total usage")
		require.Equal(t, 2, int(spaceInfo.CidsCount))
		require.Equal(t, 1, int(spaceInfo.FilesCount))

		// Verify group info
		groupInfo := f.srvIndex.GroupInfo(key.GroupId)
		require.Equal(t, len(b1.RawData())+len(b2.RawData()), int(groupInfo.TotalUsageBytes), "Group should have correct total usage")
	})
}

func TestBindFileOperation(t *testing.T) {
	t.Run("bind CIDs to a file", func(t *testing.T) {
		var (
			key    = newRandKey()
			b1     = testutil.NewRandBlock(1024)
			b2     = testutil.NewRandBlock(2048)
			c1     = b1.Cid()
			c2     = b2.Cid()
			fileId = "test-bind-file"
		)

		f := newTestFixture(t)

		// First, add the CIDs to the blocksLake using CidAdd operations
		cidOp1 := &indexpb.CidAddOperation{}
		cidOp1.SetCid(c1.String())
		cidOp1.SetFileId("temp-file-1") // Different file ID
		cidOp1.SetDataSize(uint64(len(b1.RawData())))

		op1 := &indexpb.Operation{}
		op1.SetCidAdd(cidOp1)

		require.NoError(t, f.srvIndex.Modify(nil, key, op1))

		cidOp2 := &indexpb.CidAddOperation{}
		cidOp2.SetCid(c2.String())
		cidOp2.SetFileId("temp-file-2") // Different file ID
		cidOp2.SetDataSize(uint64(len(b2.RawData())))

		op2 := &indexpb.Operation{}
		op2.SetCidAdd(cidOp2)

		require.NoError(t, f.srvIndex.Modify(nil, key, op2))

		// Now bind both CIDs to the target fileId using BindFile operation
		bindOp := &indexpb.FileBindOperation{}
		bindOp.SetFileId(fileId)
		bindOp.SetCids([]string{c1.String(), c2.String()})

		bindFileOp := &indexpb.Operation{}
		bindFileOp.SetBindFile(bindOp)

		require.NoError(t, f.srvIndex.Modify(nil, key, bindFileOp))

		// Verify file now has both CIDs
		fileInfos := f.srvIndex.FileInfo(key, fileId)
		require.Len(t, fileInfos, 1)
		require.Equal(t, 2, int(fileInfos[0].CidsCount))
		require.Equal(t, len(b1.RawData())+len(b2.RawData()), int(fileInfos[0].UsageBytes))

		// Verify CIDs are properly associated with the space
		require.True(t, f.srvIndex.HasCIDInSpace(key, c1))
		require.True(t, f.srvIndex.HasCIDInSpace(key, c2))

		// Check space statistics were updated
		spaceInfo := f.srvIndex.SpaceInfo(key)
		require.Equal(t, 3, int(spaceInfo.FilesCount)) // temp-file-1, temp-file-2, and our bind target
	})

	t.Run("bind with non-existent CID", func(t *testing.T) {
		var (
			key    = newRandKey()
			b1     = testutil.NewRandBlock(1024)
			c1     = b1.Cid()
			b2     = testutil.NewRandBlock(1024)
			c2     = b2.Cid()
			fileId = "test-bind-file"
		)

		f := newTestFixture(t)

		// Add only the first CID to the blocksLake
		cidOp := &indexpb.CidAddOperation{}
		cidOp.SetCid(c1.String())
		cidOp.SetFileId("temp-file")
		cidOp.SetDataSize(uint64(len(b1.RawData())))

		op := &indexpb.Operation{}
		op.SetCidAdd(cidOp)

		require.NoError(t, f.srvIndex.Modify(nil, key, op))

		// Now try to bind both CIDs (including c2 which doesn't exist in blocksLake)
		bindOp := &indexpb.FileBindOperation{}
		bindOp.SetFileId(fileId)
		bindOp.SetCids([]string{c1.String(), c2.String()})

		bindFileOp := &indexpb.Operation{}
		bindFileOp.SetBindFile(bindOp)

		require.NoError(t, f.srvIndex.Modify(nil, key, bindFileOp)) // Should not error, just log a warning

		// Verify only c1 was bound to the file
		fileInfos := f.srvIndex.FileInfo(key, fileId)
		require.Len(t, fileInfos, 1)
		require.Equal(t, 1, int(fileInfos[0].CidsCount))
		require.Equal(t, len(b1.RawData()), int(fileInfos[0].UsageBytes))

		// Verify only c1 is associated with the space
		require.True(t, f.srvIndex.HasCIDInSpace(key, c1))
		require.False(t, f.srvIndex.HasCIDInSpace(key, c2))
	})

	t.Run("bind with invalid CID string", func(t *testing.T) {
		var (
			key    = newRandKey()
			fileId = "test-bind-file"
		)

		f := newTestFixture(t)

		// Try to bind with an invalid CID
		bindOp := &indexpb.FileBindOperation{}
		bindOp.SetFileId(fileId)
		bindOp.SetCids([]string{"not-a-valid-cid"})

		bindFileOp := &indexpb.Operation{}
		bindFileOp.SetBindFile(bindOp)

		err := f.srvIndex.Modify(nil, key, bindFileOp)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidCID)
	})

	t.Run("bind same CIDs to multiple files", func(t *testing.T) {
		var (
			key     = newRandKey()
			b1      = testutil.NewRandBlock(1024)
			b2      = testutil.NewRandBlock(2048)
			c1      = b1.Cid()
			c2      = b2.Cid()
			fileId1 = "test-bind-file-1"
			fileId2 = "test-bind-file-2"
		)

		f := newTestFixture(t)

		// First, add the CIDs to the blocksLake
		cidOp1 := &indexpb.CidAddOperation{}
		cidOp1.SetCid(c1.String())
		cidOp1.SetFileId("temp-file")
		cidOp1.SetDataSize(uint64(len(b1.RawData())))

		op1 := &indexpb.Operation{}
		op1.SetCidAdd(cidOp1)

		require.NoError(t, f.srvIndex.Modify(nil, key, op1))

		cidOp2 := &indexpb.CidAddOperation{}
		cidOp2.SetCid(c2.String())
		cidOp2.SetFileId("temp-file")
		cidOp2.SetDataSize(uint64(len(b2.RawData())))

		op2 := &indexpb.Operation{}
		op2.SetCidAdd(cidOp2)

		require.NoError(t, f.srvIndex.Modify(nil, key, op2))

		// Now bind both CIDs to first file
		bindOp1 := &indexpb.FileBindOperation{}
		bindOp1.SetFileId(fileId1)
		bindOp1.SetCids([]string{c1.String(), c2.String()})

		bindFileOp1 := &indexpb.Operation{}
		bindFileOp1.SetBindFile(bindOp1)

		require.NoError(t, f.srvIndex.Modify(nil, key, bindFileOp1))

		// Now bind both CIDs to second file
		bindOp2 := &indexpb.FileBindOperation{}
		bindOp2.SetFileId(fileId2)
		bindOp2.SetCids([]string{c1.String(), c2.String()})

		bindFileOp2 := &indexpb.Operation{}
		bindFileOp2.SetBindFile(bindOp2)

		require.NoError(t, f.srvIndex.Modify(nil, key, bindFileOp2))

		// Verify both files have both CIDs
		fileInfos := f.srvIndex.FileInfo(key, fileId1, fileId2)
		require.Len(t, fileInfos, 2)
		require.Equal(t, 2, int(fileInfos[0].CidsCount))
		require.Equal(t, 2, int(fileInfos[1].CidsCount))

		// Each file should have the sum of cidBlock sizes
		totalSize := len(b1.RawData()) + len(b2.RawData())
		require.Equal(t, totalSize, int(fileInfos[0].UsageBytes))
		require.Equal(t, totalSize, int(fileInfos[1].UsageBytes))

		// Check space statistics
		spaceInfo := f.srvIndex.SpaceInfo(key)
		require.Equal(t, 3, int(spaceInfo.FilesCount))              // temp-file + 2 bind targets
		require.Equal(t, 2, int(spaceInfo.CidsCount))               // 2 unique CIDs
		require.Equal(t, totalSize, int(spaceInfo.TotalUsageBytes)) // Both files have both CIDs

		// Check reference count indirectly by checking if CIDs are in spaces
		require.True(t, f.srvIndex.HasCIDInSpace(key, c1))
		require.True(t, f.srvIndex.HasCIDInSpace(key, c2))

		// We can also check that the files have the CIDs via FileInfo
		fileInfos1 := f.srvIndex.FileInfo(key, fileId1)
		fileInfos2 := f.srvIndex.FileInfo(key, fileId2)
		require.Equal(t, 2, int(fileInfos1[0].CidsCount))
		require.Equal(t, 2, int(fileInfos2[0].CidsCount))
	})

	t.Run("bind CIDs to file multiple times (idempotence)", func(t *testing.T) {
		var (
			key    = newRandKey()
			b1     = testutil.NewRandBlock(1024)
			b2     = testutil.NewRandBlock(2048)
			c1     = b1.Cid()
			c2     = b2.Cid()
			fileId = "test-bind-file"
		)

		f := newTestFixture(t)

		// First, add the CIDs to the blocksLake
		cidOp1 := &indexpb.CidAddOperation{}
		cidOp1.SetCid(c1.String())
		cidOp1.SetFileId("temp-file")
		cidOp1.SetDataSize(uint64(len(b1.RawData())))

		op1 := &indexpb.Operation{}
		op1.SetCidAdd(cidOp1)

		require.NoError(t, f.srvIndex.Modify(nil, key, op1))

		cidOp2 := &indexpb.CidAddOperation{}
		cidOp2.SetCid(c2.String())
		cidOp2.SetFileId("temp-file")
		cidOp2.SetDataSize(uint64(len(b2.RawData())))

		op2 := &indexpb.Operation{}
		op2.SetCidAdd(cidOp2)

		require.NoError(t, f.srvIndex.Modify(nil, key, op2))

		// Bind CIDs to file the first time
		bindOp1 := &indexpb.FileBindOperation{}
		bindOp1.SetFileId(fileId)
		bindOp1.SetCids([]string{c1.String(), c2.String()})

		bindFileOp1 := &indexpb.Operation{}
		bindFileOp1.SetBindFile(bindOp1)

		require.NoError(t, f.srvIndex.Modify(nil, key, bindFileOp1))

		// Get first file info
		infoBeforeBind := f.srvIndex.FileInfo(key, fileId)
		require.Len(t, infoBeforeBind, 1)

		// Bind the same CIDs to the same file again (should be idempotent)
		bindOp2 := &indexpb.FileBindOperation{}
		bindOp2.SetFileId(fileId)
		bindOp2.SetCids([]string{c1.String(), c2.String()})

		bindFileOp2 := &indexpb.Operation{}
		bindFileOp2.SetBindFile(bindOp2)

		require.NoError(t, f.srvIndex.Modify(nil, key, bindFileOp2))

		// Verify file still has the same stats
		infoAfterBind := f.srvIndex.FileInfo(key, fileId)
		require.Len(t, infoAfterBind, 1)
		require.Equal(t, infoBeforeBind[0].CidsCount, infoAfterBind[0].CidsCount)
		require.Equal(t, infoBeforeBind[0].UsageBytes, infoAfterBind[0].UsageBytes)

		totalSize := len(b1.RawData()) + len(b2.RawData())
		require.Equal(t, totalSize, int(infoAfterBind[0].UsageBytes))

		// Verify space statistics are correct
		spaceInfo := f.srvIndex.SpaceInfo(key)
		require.Equal(t, 2, int(spaceInfo.FilesCount)) // temp-file + bind target
		require.Equal(t, 2, int(spaceInfo.CidsCount))  // 2 unique CIDs
	})

	t.Run("bind empty CIDs list", func(t *testing.T) {
		var (
			key    = newRandKey()
			fileId = "test-bind-file"
		)

		f := newTestFixture(t)

		// Try to bind with an empty CIDs list
		bindOp := &indexpb.FileBindOperation{}
		bindOp.SetFileId(fileId)
		bindOp.SetCids([]string{})

		bindFileOp := &indexpb.Operation{}
		bindFileOp.SetBindFile(bindOp)

		// Should not error
		require.NoError(t, f.srvIndex.Modify(nil, key, bindFileOp))

		// File should exist but have no CIDs
		fileInfos := f.srvIndex.FileInfo(key, fileId)
		require.Len(t, fileInfos, 1)
		require.Equal(t, 0, int(fileInfos[0].CidsCount))
		require.Equal(t, 0, int(fileInfos[0].UsageBytes))

		// Space should have one file but no CIDs or usage
		spaceInfo := f.srvIndex.SpaceInfo(key)
		require.Equal(t, 1, int(spaceInfo.FilesCount))
		require.Equal(t, 0, int(spaceInfo.CidsCount))
		require.Equal(t, 0, int(spaceInfo.TotalUsageBytes))
	})
}

func TestReadMethodsAfterBind(t *testing.T) {
	var (
		key     = newRandKey()
		b1      = testutil.NewRandBlock(1024)
		b2      = testutil.NewRandBlock(2048)
		c1      = b1.Cid()
		c2      = b2.Cid()
		fileId1 = "test-file-1"
		fileId2 = "test-file-2"
	)

	f := newTestFixture(t)

	// First, add the CIDs to the blocksLake
	cidOp1 := &indexpb.CidAddOperation{}
	cidOp1.SetCid(c1.String())
	cidOp1.SetFileId("temp-file")
	cidOp1.SetDataSize(uint64(len(b1.RawData())))

	op1 := &indexpb.Operation{}
	op1.SetCidAdd(cidOp1)

	require.NoError(t, f.srvIndex.Modify(nil, key, op1))

	cidOp2 := &indexpb.CidAddOperation{}
	cidOp2.SetCid(c2.String())
	cidOp2.SetFileId("temp-file")
	cidOp2.SetDataSize(uint64(len(b2.RawData())))

	op2 := &indexpb.Operation{}
	op2.SetCidAdd(cidOp2)

	require.NoError(t, f.srvIndex.Modify(nil, key, op2))

	// Now bind c1 to first file
	bindOp1 := &indexpb.FileBindOperation{}
	bindOp1.SetFileId(fileId1)
	bindOp1.SetCids([]string{c1.String()})

	bindFileOp1 := &indexpb.Operation{}
	bindFileOp1.SetBindFile(bindOp1)

	require.NoError(t, f.srvIndex.Modify(nil, key, bindFileOp1))

	// Bind c2 to second file
	bindOp2 := &indexpb.FileBindOperation{}
	bindOp2.SetFileId(fileId2)
	bindOp2.SetCids([]string{c2.String()})

	bindFileOp2 := &indexpb.Operation{}
	bindFileOp2.SetBindFile(bindOp2)

	require.NoError(t, f.srvIndex.Modify(nil, key, bindFileOp2))

	// 1. Test HasCIDInSpace method
	require.True(t, f.srvIndex.HasCIDInSpace(key, c1), "c1 should exist in the space")
	require.True(t, f.srvIndex.HasCIDInSpace(key, c2), "c2 should exist in the space")

	randomCid, _ := cid.Parse("bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi")
	require.False(t, f.srvIndex.HasCIDInSpace(key, randomCid), "random CID should not exist in the space")

	// 2. Test HadCID method
	require.True(t, f.srvIndex.HadCID(c1), "c1 should exist in the index")
	require.True(t, f.srvIndex.HadCID(c2), "c2 should exist in the index")
	require.False(t, f.srvIndex.HadCID(randomCid), "random CID should not exist in the index")

	// 3. Test GroupInfo method
	groupInfo := f.srvIndex.GroupInfo(key.GroupId)
	require.Equal(t, len(b1.RawData())+len(b2.RawData()), int(groupInfo.TotalUsageBytes), "Group usage count only unique CIDs")
	require.Equal(t, 2, int(groupInfo.TotalCidsCount), "Group should have 2 unique CIDs")
	require.Equal(t, key.SpaceId, groupInfo.Spaces[0].SpaceId, "Group should include the correct space ID")

	// 4. Test SpaceInfo method
	spaceInfo := f.srvIndex.SpaceInfo(key)
	require.Equal(t, len(b1.RawData())+len(b2.RawData()), int(spaceInfo.TotalUsageBytes), "Space should count only unique CIDs")
	require.Equal(t, 2, int(spaceInfo.CidsCount), "Space should have 2 unique CIDs")
	require.Equal(t, 3, int(spaceInfo.FilesCount), "Space should have 3 files (temp-file + 2 bind targets)")

	// 5. Test SpaceFiles method
	files := f.srvIndex.SpaceFiles(key)
	require.Contains(t, files, "temp-file", "Space files should include the temp file")
	require.Contains(t, files, fileId1, "Space files should include the first file")
	require.Contains(t, files, fileId2, "Space files should include the second file")
	require.Equal(t, 3, len(files), "Should have exactly 3 files")

	// 6. Test FileInfo method
	fileInfos := f.srvIndex.FileInfo(key, fileId1, fileId2)
	require.Len(t, fileInfos, 2, "Should return info for two files")

	// First file should have c1
	fileInfo1 := fileInfos[0]
	require.Equal(t, fileId1, fileInfo1.FileId)
	require.Equal(t, 1, int(fileInfo1.CidsCount))
	require.Equal(t, len(b1.RawData()), int(fileInfo1.UsageBytes))

	// Second file should have c2
	fileInfo2 := fileInfos[1]
	require.Equal(t, fileId2, fileInfo2.FileId)
	require.Equal(t, 1, int(fileInfo2.CidsCount))
	require.Equal(t, len(b2.RawData()), int(fileInfo2.UsageBytes))
}

func TestReadMethodsAfterMultiBindAndDelete(t *testing.T) {
	var (
		key     = newRandKey()
		b1      = testutil.NewRandBlock(1024)
		b2      = testutil.NewRandBlock(2048)
		c1      = b1.Cid()
		c2      = b2.Cid()
		fileId0 = "test-file-cid-add"
		fileId1 = "test-file-1-bind"
		fileId2 = "test-file-2-bind"
	)

	f := newTestFixture(t)

	// First, create file by CIDs add
	cidOp1 := &indexpb.CidAddOperation{}
	cidOp1.SetCid(c1.String())
	cidOp1.SetFileId(fileId0)
	cidOp1.SetDataSize(uint64(len(b1.RawData())))

	op1 := &indexpb.Operation{}
	op1.SetCidAdd(cidOp1)

	require.NoError(t, f.srvIndex.Modify(nil, key, op1))

	cidOp2 := &indexpb.CidAddOperation{}
	cidOp2.SetCid(c2.String())
	cidOp2.SetFileId(fileId0)
	cidOp2.SetDataSize(uint64(len(b2.RawData())))

	op2 := &indexpb.Operation{}
	op2.SetCidAdd(cidOp2)

	require.NoError(t, f.srvIndex.Modify(nil, key, op2))

	// Bind both CIDs to both files
	bindOp1 := &indexpb.FileBindOperation{}
	bindOp1.SetFileId(fileId1)
	bindOp1.SetCids([]string{c1.String(), c2.String()})

	bindFileOp1 := &indexpb.Operation{}
	bindFileOp1.SetBindFile(bindOp1)

	require.NoError(t, f.srvIndex.Modify(nil, key, bindFileOp1))

	bindOp2 := &indexpb.FileBindOperation{}
	bindOp2.SetFileId(fileId2)
	bindOp2.SetCids([]string{c1.String(), c2.String()})

	bindFileOp2 := &indexpb.Operation{}
	bindFileOp2.SetBindFile(bindOp2)

	require.NoError(t, f.srvIndex.Modify(nil, key, bindFileOp2))

	// Check statistics before deletion
	beforeSpaceInfo := f.srvIndex.SpaceInfo(key)
	beforeGroupInfo := f.srvIndex.GroupInfo(key.GroupId)

	// Calculate expected total size
	totalSize := len(b1.RawData()) + len(b2.RawData())

	require.Equal(t, 3, int(beforeSpaceInfo.FilesCount))
	require.Equal(t, 2, int(beforeSpaceInfo.CidsCount))
	require.Equal(t, totalSize, int(beforeSpaceInfo.TotalUsageBytes))

	require.Equal(t, 2, int(beforeGroupInfo.TotalCidsCount))
	require.Equal(t, totalSize, int(beforeGroupInfo.TotalUsageBytes))

	// Instead of checking reference counts directly, verify CID presence in files
	fileInfosBefore := f.srvIndex.FileInfo(key, fileId1, fileId2)
	require.Len(t, fileInfosBefore, 2)
	require.Equal(t, 2, int(fileInfosBefore[0].CidsCount))
	require.Equal(t, 2, int(fileInfosBefore[1].CidsCount))

	// Now delete the first file
	deleteOp := &indexpb.FileDeleteOperation{}
	deleteOp.SetFileIds([]string{fileId1})

	deleteFileOp := &indexpb.Operation{}
	deleteFileOp.SetDeleteFile(deleteOp)

	require.NoError(t, f.srvIndex.Modify(nil, key, deleteFileOp))

	// Check statistics after deletion
	afterSpaceInfo := f.srvIndex.SpaceInfo(key)
	afterGroupInfo := f.srvIndex.GroupInfo(key.GroupId)

	require.Equal(t, 2, int(afterSpaceInfo.FilesCount), "Space should have 2 files after deletion")
	require.Equal(t, 2, int(afterSpaceInfo.CidsCount), "Space should still have 2 unique CIDs")
	require.Equal(t, totalSize, int(afterSpaceInfo.TotalUsageBytes), "Space should be same, because CIDs used in other file")

	require.Equal(t, 2, int(afterGroupInfo.TotalCidsCount), "Group should still have 2 unique CIDs")
	require.Equal(t, totalSize, int(afterGroupInfo.TotalUsageBytes), "Group usage should be same, because CIDs used in other file")

	// Check files in space
	files := f.srvIndex.SpaceFiles(key)
	require.NotContains(t, files, fileId1, "Deleted file should no longer be in space files")
	require.Contains(t, files, fileId2, "Second file should still be in space files")
	require.Contains(t, files, fileId0, "File, created by CID add, should still be in space files")
	require.Equal(t, 2, len(files), "Should have exactly 2 files")

	// Verify file info after deletion - fileId1 should be gone
	fileInfosAfter := f.srvIndex.FileInfo(key, fileId1, fileId2)
	require.Len(t, fileInfosAfter, 2)
	require.Equal(t, fileId1, fileInfosAfter[0].FileId, "Temporary file should be first")
	require.Equal(t, fileId2, fileInfosAfter[1].FileId, "The remaining file should be fileId2")

	// Verify HasCIDInSpace still returns true for both CIDs since they're still in other files
	require.True(t, f.srvIndex.HasCIDInSpace(key, c1), "c1 should still exist in the space")
	require.True(t, f.srvIndex.HasCIDInSpace(key, c2), "c2 should still exist in the space")
}

func TestFileDeleteWithReferenceTracking(t *testing.T) {
	t.Run("delete file with CIDs still used by other files", func(t *testing.T) {
		var (
			key     = newRandKey()
			b1      = testutil.NewRandBlock(1024)
			b2      = testutil.NewRandBlock(2048)
			c1      = b1.Cid()
			c2      = b2.Cid()
			fileId1 = "test-file-1"
			fileId2 = "test-file-2"
			fileId3 = "test-file-3"
		)

		f := newTestFixture(t)

		// Add CID c1 to first file and second file
		cidOp1 := &indexpb.CidAddOperation{}
		cidOp1.SetCid(c1.String())
		cidOp1.SetFileId(fileId1)
		cidOp1.SetDataSize(uint64(len(b1.RawData())))
		op1 := &indexpb.Operation{}
		op1.SetCidAdd(cidOp1)
		require.NoError(t, f.srvIndex.Modify(nil, key, op1))

		cidOp2 := &indexpb.CidAddOperation{}
		cidOp2.SetCid(c1.String())
		cidOp2.SetFileId(fileId2)
		cidOp2.SetDataSize(uint64(len(b1.RawData())))
		op2 := &indexpb.Operation{}
		op2.SetCidAdd(cidOp2)
		require.NoError(t, f.srvIndex.Modify(nil, key, op2))

		// Add CID c2 to third file only
		cidOp3 := &indexpb.CidAddOperation{}
		cidOp3.SetCid(c2.String())
		cidOp3.SetFileId(fileId3)
		cidOp3.SetDataSize(uint64(len(b2.RawData())))
		op3 := &indexpb.Operation{}
		op3.SetCidAdd(cidOp3)
		require.NoError(t, f.srvIndex.Modify(nil, key, op3))

		// Check initial state
		initialSpaceInfo := f.srvIndex.SpaceInfo(key)
		require.Equal(t, 3, int(initialSpaceInfo.FilesCount))
		require.Equal(t, 2, int(initialSpaceInfo.CidsCount)) // c1 and c2

		// Now delete file 1
		deleteOp := &indexpb.FileDeleteOperation{}
		deleteOp.SetFileIds([]string{fileId1})
		deleteFileOp := &indexpb.Operation{}
		deleteFileOp.SetDeleteFile(deleteOp)
		require.NoError(t, f.srvIndex.Modify(nil, key, deleteFileOp))

		// Check state after first delete
		// File count should decrease, but CID count should stay the same as both CIDs are still used
		afterDelete1 := f.srvIndex.SpaceInfo(key)
		require.Equal(t, 2, int(afterDelete1.FilesCount)) // file2 and file3 remain
		require.Equal(t, 2, int(afterDelete1.CidsCount))  // Both c1 and c2 are still in use

		// c1 should still be in the space since file2 uses it
		require.True(t, f.srvIndex.HasCIDInSpace(key, c1))

		// Now delete file 3, which exclusively holds c2
		deleteOp2 := &indexpb.FileDeleteOperation{}
		deleteOp2.SetFileIds([]string{fileId3})
		deleteFileOp2 := &indexpb.Operation{}
		deleteFileOp2.SetDeleteFile(deleteOp2)
		require.NoError(t, f.srvIndex.Modify(nil, key, deleteFileOp2))

		// Check final state
		finalState := f.srvIndex.SpaceInfo(key)
		require.Equal(t, 1, int(finalState.FilesCount)) // only file2 remains
		require.Equal(t, 1, int(finalState.CidsCount))  // only c1 remains

		// c1 should still exist in space
		require.True(t, f.srvIndex.HasCIDInSpace(key, c1))

		// c2 should be gone since no file references it
		require.False(t, f.srvIndex.HasCIDInSpace(key, c2))

		// But c2 should still exist in the global blocks lake
		require.True(t, f.srvIndex.HadCID(c2))
	})

	t.Run("deleting files across multiple spaces with shared CIDs", func(t *testing.T) {
		var (
			key1 = newRandKey() // First space
			key2 = index.Key{   // Second space in same group
				SpaceId: testutil.NewRandSpaceId(),
				GroupId: key1.GroupId, // Same group ID
			}
			b1      = testutil.NewRandBlock(1024)
			c1      = b1.Cid()
			fileId1 = "space1-file"
			fileId2 = "space2-file"
		)

		f := newTestFixture(t)

		// Add same CID to files in different spaces
		cidOp1 := &indexpb.CidAddOperation{}
		cidOp1.SetCid(c1.String())
		cidOp1.SetFileId(fileId1)
		cidOp1.SetDataSize(uint64(len(b1.RawData())))
		op1 := &indexpb.Operation{}
		op1.SetCidAdd(cidOp1)
		require.NoError(t, f.srvIndex.Modify(nil, key1, op1))

		cidOp2 := &indexpb.CidAddOperation{}
		cidOp2.SetCid(c1.String())
		cidOp2.SetFileId(fileId2)
		cidOp2.SetDataSize(uint64(len(b1.RawData())))
		op2 := &indexpb.Operation{}
		op2.SetCidAdd(cidOp2)
		require.NoError(t, f.srvIndex.Modify(nil, key2, op2))

		// Check initial state
		initialGroupInfo := f.srvIndex.GroupInfo(key1.GroupId)
		require.Equal(t, 2, len(initialGroupInfo.Spaces))         // Both spaces
		require.Equal(t, 1, int(initialGroupInfo.TotalCidsCount)) // One unique CID

		// Delete file in first space
		deleteOp := &indexpb.FileDeleteOperation{}
		deleteOp.SetFileIds([]string{fileId1})
		deleteFileOp := &indexpb.Operation{}
		deleteFileOp.SetDeleteFile(deleteOp)
		require.NoError(t, f.srvIndex.Modify(nil, key1, deleteFileOp))

		// CID should no longer be in first space
		require.False(t, f.srvIndex.HasCIDInSpace(key1, c1))

		// But it should still be in second space
		require.True(t, f.srvIndex.HasCIDInSpace(key2, c1))

		// Group should still have the CID
		groupInfoAfter := f.srvIndex.GroupInfo(key1.GroupId)
		require.Equal(t, 1, int(groupInfoAfter.TotalCidsCount))

		// Finally delete from second space
		deleteOp2 := &indexpb.FileDeleteOperation{}
		deleteOp2.SetFileIds([]string{fileId2})
		deleteFileOp2 := &indexpb.Operation{}
		deleteFileOp2.SetDeleteFile(deleteOp2)
		require.NoError(t, f.srvIndex.Modify(nil, key2, deleteFileOp2))

		// CID should now be gone from group
		finalGroupInfo := f.srvIndex.GroupInfo(key1.GroupId)
		require.Equal(t, 0, int(finalGroupInfo.TotalCidsCount))
		require.Equal(t, uint64(0), finalGroupInfo.TotalUsageBytes)
	})
}

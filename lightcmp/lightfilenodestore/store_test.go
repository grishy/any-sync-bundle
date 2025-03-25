package lightfilenodestore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/dgraph-io/badger/v4"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grishy/any-sync-bundle/lightcmp/lightdb"
)

type testFixture struct {
	app      *app.App
	srvCfg   *configServiceMock
	srvDB    lightdb.DBService
	srvStore *lightFileNodeStore
}

func newTestFixture(t *testing.T) *testFixture {
	t.Helper()

	f := &testFixture{
		app: new(app.App),
		srvCfg: &configServiceMock{
			GetDBDirFunc: func() string {
				return t.TempDir()
			},
			InitFunc: func(a *app.App) error {
				return nil
			},
			NameFunc: func() string {
				return "config"
			},
		},
		srvDB:    lightdb.New(),
		srvStore: New(),
	}

	f.app.
		Register(f.srvCfg).
		Register(f.srvDB).
		Register(f.srvStore)

	require.NoError(t, f.app.Start(context.Background()))

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		require.NoError(t, f.app.Close(ctx))
	})

	return f
}

// createTestBlock generates a test block with provided data and returns the block and its CID
func createTestBlock(t *testing.T, data []byte) (blocks.Block, cid.Cid) {
	t.Helper()
	mhash, err := mh.Sum(data, mh.SHA2_256, -1)
	require.NoError(t, err)

	c := cid.NewCidV1(cid.Raw, mhash)
	block, err := blocks.NewBlockWithCid(data, c)
	require.NoError(t, err)

	return block, c
}

func TestBlockOperations(t *testing.T) {
	fx := newTestFixture(t)

	t.Run("put and get block", func(t *testing.T) {
		testData := []byte("test block data")
		testBlock, testCID := createTestBlock(t, testData)

		// Put block
		err := fx.srvDB.TxUpdate(func(txn *badger.Txn) error {
			return fx.srvStore.PutBlock(txn, testBlock)
		})
		require.NoError(t, err)

		// Get block and verify
		var retrievedData []byte
		err = fx.srvDB.TxView(func(txn *badger.Txn) error {
			var err error
			retrievedData, err = fx.srvStore.GetBlock(txn, testCID)
			return err
		})
		require.NoError(t, err)
		assert.Equal(t, testData, retrievedData)
	})

	t.Run("delete block", func(t *testing.T) {
		testData := []byte("block to delete")
		testBlock, testCID := createTestBlock(t, testData)

		// Put block
		err := fx.srvDB.TxUpdate(func(txn *badger.Txn) error {
			return fx.srvStore.PutBlock(txn, testBlock)
		})
		require.NoError(t, err)

		// Delete block
		err = fx.srvDB.TxUpdate(func(txn *badger.Txn) error {
			return fx.srvStore.DeleteBlock(txn, testCID)
		})
		require.NoError(t, err)

		// Verify block is gone
		err = fx.srvDB.TxView(func(txn *badger.Txn) error {
			_, err := fx.srvStore.GetBlock(txn, testCID)
			return err
		})
		assert.ErrorIs(t, err, ErrBlockNotFound)
	})

	t.Run("get non-existent block", func(t *testing.T) {
		_, nonExistentCID := createTestBlock(t, []byte("non-existent"))

		err := fx.srvDB.TxView(func(txn *badger.Txn) error {
			_, err := fx.srvStore.GetBlock(txn, nonExistentCID)
			return err
		})
		assert.ErrorIs(t, err, ErrBlockNotFound)
	})
}

func TestIndexSnapshotOperations(t *testing.T) {
	fx := newTestFixture(t)

	t.Run("no initial snapshot", func(t *testing.T) {
		err := fx.srvDB.TxView(func(txn *badger.Txn) error {
			_, err := fx.srvStore.GetIndexSnapshot(txn)
			return err
		})
		assert.ErrorIs(t, err, ErrNoSnapshot)
	})

	t.Run("save and get snapshot", func(t *testing.T) {
		snapshotData := []byte("test snapshot data")

		// Save snapshot
		err := fx.srvDB.TxUpdate(func(txn *badger.Txn) error {
			return fx.srvStore.SaveIndexSnapshot(txn, snapshotData)
		})
		require.NoError(t, err)

		// Retrieve and verify
		var retrieved []byte
		err = fx.srvDB.TxView(func(txn *badger.Txn) error {
			var err error
			retrieved, err = fx.srvStore.GetIndexSnapshot(txn)
			return err
		})
		require.NoError(t, err)
		assert.Equal(t, snapshotData, retrieved)
	})
}

func TestIndexLogOperations(t *testing.T) {
	fx := newTestFixture(t)

	// Helper to retrieve index logs
	getIndexLogs := func() []IndexLog {
		var logs []IndexLog
		err := fx.srvDB.TxView(func(txn *badger.Txn) error {
			var err error
			logs, err = fx.srvStore.GetIndexLogs(txn)
			return err
		})
		require.NoError(t, err)
		return logs
	}

	t.Run("initial state", func(t *testing.T) {
		logs := getIndexLogs()
		assert.Empty(t, logs)
	})

	t.Run("push and get logs", func(t *testing.T) {
		testLogs := []struct {
			data        []byte
			expectedIdx uint64
		}{
			{[]byte("log 1"), 1},
			{[]byte("log 2"), 2},
			{[]byte("log 3"), 3},
		}

		// Push logs
		for _, log := range testLogs {
			err := fx.srvDB.TxUpdate(func(txn *badger.Txn) error {
				return fx.srvStore.PushIndexLog(txn, log.data)
			})
			require.NoError(t, err)
		}

		// Verify logs
		logs := getIndexLogs()
		require.Len(t, logs, len(testLogs))

		for i, expected := range testLogs {
			assert.Equal(t, expected.expectedIdx, logs[i].Idx)
			assert.Equal(t, expected.data, logs[i].Data)
		}
	})

	t.Run("delete logs", func(t *testing.T) {
		// Delete middle log
		err := fx.srvDB.TxUpdate(func(txn *badger.Txn) error {
			return fx.srvStore.DeleteIndexLogs(txn, []uint64{2})
		})
		require.NoError(t, err)

		// Verify log was deleted
		logs := getIndexLogs()
		require.Len(t, logs, 2)
		assert.Equal(t, uint64(1), logs[0].Idx)
		assert.Equal(t, uint64(3), logs[1].Idx)

		// Delete empty list should not error
		err = fx.srvDB.TxUpdate(func(txn *badger.Txn) error {
			return fx.srvStore.DeleteIndexLogs(txn, nil)
		})
		require.NoError(t, err)

		// Verify no changes
		logs = getIndexLogs()
		assert.Len(t, logs, 2)
	})
}

func TestTransactionOperations(t *testing.T) {
	fx := newTestFixture(t)
	expectedErr := errors.New("test error")

	testCases := []struct {
		name    string
		op      func() error
		wantErr error
	}{
		{name: "view success", op: func() error { return fx.srvDB.TxView(func(*badger.Txn) error { return nil }) }},
		{name: "view error", op: func() error { return fx.srvDB.TxView(func(*badger.Txn) error { return expectedErr }) }, wantErr: expectedErr},
		{name: "update success", op: func() error { return fx.srvDB.TxUpdate(func(*badger.Txn) error { return nil }) }},
		{name: "update error", op: func() error { return fx.srvDB.TxUpdate(func(*badger.Txn) error { return expectedErr }) }, wantErr: expectedErr},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.op()
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKeyManagement(t *testing.T) {
	testCases := []struct {
		name string
		test func(*testing.T)
	}{
		{
			name: "buildKey block",
			test: func(t *testing.T) {
				_, testCID := createTestBlock(t, []byte("test"))
				key := buildKey(blockType, testCID.String())
				assert.Equal(t, prefixFileNode+separator+blockType+separator+testCID.String(), string(key))
			},
		},
		{
			name: "buildKey snapshot",
			test: func(t *testing.T) {
				key := buildKey(snapshotType, currentSnapshotSuffix)
				assert.Equal(t, prefixFileNode+separator+snapshotType+separator+currentSnapshotSuffix, string(key))
			},
		},
		{
			name: "buildKeyPrefix",
			test: func(t *testing.T) {
				prefix := buildKeyPrefix(logType)
				assert.Equal(t, prefixFileNode+separator+logType+separator, string(prefix))
			},
		},
		{
			name: "parseIdxFromKey valid",
			test: func(t *testing.T) {
				key := buildKey(logType, "123")
				idx, err := parseIdxFromKey(key)
				require.NoError(t, err)
				assert.Equal(t, uint64(123), idx)
			},
		},
		{
			name: "parseIdxFromKey invalid",
			test: func(t *testing.T) {
				invalidKeys := []string{
					"invalid",
					"invalid:key",
					prefixFileNode + separator,
					prefixFileNode + separator + logType,
				}

				for _, key := range invalidKeys {
					_, err := parseIdxFromKey([]byte(key))
					assert.ErrorIs(t, err, ErrInvalidKey)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

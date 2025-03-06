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
)

type testFixture struct {
	app           *app.App
	configService *configServiceMock
	store         *lightFileNodeStore
	cleanupDone   bool // Track if cleanup was already performed
}

func (f *testFixture) cleanup(t *testing.T) {
	t.Helper()
	if f.cleanupDone {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, f.app.Close(ctx))
	f.cleanupDone = true
}

func newTestFixture(t *testing.T) *testFixture {
	t.Helper()
	tempDir := t.TempDir()

	f := &testFixture{
		app: new(app.App),
		configService: &configServiceMock{
			GetFilenodeStoreDirFunc: func() string { return tempDir },
			InitFunc:                func(a *app.App) error { return nil },
			NameFunc:                func() string { return "config" },
		},
	}
	f.store = New().(*lightFileNodeStore)

	f.app.Register(f.configService).Register(f.store)
	require.NoError(t, f.app.Start(context.Background()))

	t.Cleanup(func() { f.cleanup(t) })
	return f
}

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

		err := fx.store.TxUpdate(func(txn *badger.Txn) error {
			return fx.store.PutBlock(txn, testBlock)
		})
		require.NoError(t, err)

		var retrievedData []byte
		err = fx.store.TxView(func(txn *badger.Txn) error {
			var err error
			retrievedData, err = fx.store.GetBlock(txn, testCID)
			return err
		})
		require.NoError(t, err)
		assert.Equal(t, testData, retrievedData)
	})

	t.Run("delete block", func(t *testing.T) {
		testData := []byte("block to delete")
		testBlock, testCID := createTestBlock(t, testData)

		err := fx.store.TxUpdate(func(txn *badger.Txn) error {
			return fx.store.PutBlock(txn, testBlock)
		})
		require.NoError(t, err)

		err = fx.store.TxUpdate(func(txn *badger.Txn) error {
			return fx.store.DeleteBlock(txn, testCID)
		})
		require.NoError(t, err)

		err = fx.store.TxView(func(txn *badger.Txn) error {
			_, err := fx.store.GetBlock(txn, testCID)
			return err
		})
		assert.ErrorIs(t, err, ErrBlockNotFound)
	})

	t.Run("get non-existent block", func(t *testing.T) {
		_, nonExistentCID := createTestBlock(t, []byte("non-existent"))

		err := fx.store.TxView(func(txn *badger.Txn) error {
			_, err := fx.store.GetBlock(txn, nonExistentCID)
			return err
		})
		assert.ErrorIs(t, err, ErrBlockNotFound)
	})
}

func TestIndexSnapshotOperations(t *testing.T) {
	fx := newTestFixture(t)

	t.Run("no initial snapshot", func(t *testing.T) {
		err := fx.store.TxView(func(txn *badger.Txn) error {
			_, err := fx.store.GetIndexSnapshot(txn)
			return err
		})
		assert.ErrorIs(t, err, ErrNoSnapshot)
	})

	t.Run("save and get snapshot", func(t *testing.T) {
		snapshotData := []byte("test snapshot data")

		err := fx.store.TxUpdate(func(txn *badger.Txn) error {
			return fx.store.SaveIndexSnapshot(txn, snapshotData)
		})
		require.NoError(t, err)

		var retrieved []byte
		err = fx.store.TxView(func(txn *badger.Txn) error {
			var err error
			retrieved, err = fx.store.GetIndexSnapshot(txn)
			return err
		})
		require.NoError(t, err)
		assert.Equal(t, snapshotData, retrieved)
	})
}

func TestIndexLogOperations(t *testing.T) {
	fx := newTestFixture(t)

	getIndexLogs := func(t *testing.T) []IndexLog {
		t.Helper()
		var logs []IndexLog
		err := fx.store.TxView(func(txn *badger.Txn) error {
			var err error
			logs, err = fx.store.GetIndexLogs(txn)
			return err
		})
		require.NoError(t, err)
		return logs
	}

	t.Run("initial state", func(t *testing.T) {
		logs := getIndexLogs(t)
		assert.Empty(t, logs)
	})

	t.Run("push and get logs", func(t *testing.T) {
		testData := []struct {
			data        []byte
			expectedIdx uint64
		}{
			{[]byte("log 1"), 1},
			{[]byte("log 2"), 2},
			{[]byte("log 3"), 3},
		}

		for _, td := range testData {
			var idx uint64
			err := fx.store.TxUpdate(func(txn *badger.Txn) error {
				var err error
				idx, err = fx.store.PushIndexLog(txn, td.data)
				return err
			})
			require.NoError(t, err)
			assert.Equal(t, td.expectedIdx, idx)
		}

		logs := getIndexLogs(t)
		require.Len(t, logs, len(testData))

		for i, td := range testData {
			assert.Equal(t, td.expectedIdx, logs[i].Idx)
			assert.Equal(t, td.data, logs[i].Data)
		}
	})

	t.Run("delete logs", func(t *testing.T) {
		// Delete middle log
		err := fx.store.TxUpdate(func(txn *badger.Txn) error {
			return fx.store.DeleteIndexLogs(txn, []uint64{2})
		})
		require.NoError(t, err)

		logs := getIndexLogs(t)
		require.Len(t, logs, 2)
		assert.Equal(t, uint64(1), logs[0].Idx)
		assert.Equal(t, uint64(3), logs[1].Idx)

		// Delete empty list should not error
		err = fx.store.TxUpdate(func(txn *badger.Txn) error {
			return fx.store.DeleteIndexLogs(txn, nil)
		})
		require.NoError(t, err)

		logs = getIndexLogs(t)
		require.Len(t, logs, 2)
	})
}

func TestTransactionOperations(t *testing.T) {
	fx := newTestFixture(t)

	testCases := []struct {
		name    string
		op      func() error
		wantErr error
	}{
		{
			name: "view success",
			op: func() error {
				return fx.store.TxView(func(txn *badger.Txn) error {
					return nil
				})
			},
		},
		{
			name: "view error",
			op: func() error {
				return fx.store.TxView(func(txn *badger.Txn) error {
					return errors.New("test error")
				})
			},
			wantErr: errors.New("test error"),
		},
		{
			name: "update success",
			op: func() error {
				return fx.store.TxUpdate(func(txn *badger.Txn) error {
					return nil
				})
			},
		},
		{
			name: "update error",
			op: func() error {
				return fx.store.TxUpdate(func(txn *badger.Txn) error {
					return errors.New("test error")
				})
			},
			wantErr: errors.New("test error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.op()
			if tc.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.wantErr.Error(), err.Error())
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
				testCases := []string{
					"invalid",
					"invalid:key",
					prefixFileNode + separator,
					prefixFileNode + separator + logType,
				}

				for _, tc := range testCases {
					_, err := parseIdxFromKey([]byte(tc))
					assert.ErrorIs(t, err, ErrInvalidKey)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

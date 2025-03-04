package lightfilenodestore

import (
	"bytes"
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

func TestLightFileNodeStore_BlockOperations(t *testing.T) {
	fx := newFixture(t)
	defer fx.cleanup(t)

	testData := []byte("test block data")
	testBlock, testCID := createTestBlock(t, testData)

	// Test block storage and retrieval
	err := fx.store.TxUpdate(func(txn *badger.Txn) error {
		return fx.store.PutBlock(txn, testBlock)
	})
	require.NoError(t, err)

	var retrievedData []byte
	err = fx.store.TxView(func(txn *badger.Txn) error {
		var readErr error
		retrievedData, readErr = fx.store.GetBlock(txn, testCID)
		return readErr
	})
	require.NoError(t, err)
	assert.Equal(t, testData, retrievedData)

	// Test block deletion
	err = fx.store.TxUpdate(func(txn *badger.Txn) error {
		return fx.store.DeleteBlock(txn, testCID)
	})
	require.NoError(t, err)

	err = fx.store.TxView(func(txn *badger.Txn) error {
		_, err = fx.store.GetBlock(txn, testCID)
		return err
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBlockNotFound)
}

func TestLightFileNodeStore_IndexSnapshotOperations(t *testing.T) {
	fx := newFixture(t)
	defer fx.cleanup(t)

	// Verify no snapshot initially exists
	var snapshot []byte
	err := fx.store.TxView(func(txn *badger.Txn) error {
		var readErr error
		snapshot, readErr = fx.store.GetIndexSnapshot(txn)
		return readErr
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoSnapshot)

	// Test storing and retrieving snapshot
	snapshotData := []byte("index snapshot data")
	err = fx.store.TxUpdate(func(txn *badger.Txn) error {
		return fx.store.SaveIndexSnapshot(txn, snapshotData)
	})
	require.NoError(t, err)

	err = fx.store.TxView(func(txn *badger.Txn) error {
		var readErr error
		snapshot, readErr = fx.store.GetIndexSnapshot(txn)
		return readErr
	})
	require.NoError(t, err)
	assert.Equal(t, snapshotData, snapshot)
}

func TestLightFileNodeStore_IndexLogOperations(t *testing.T) {
	fx := newFixture(t)
	defer fx.cleanup(t)

	// Verify no logs initially exist
	logs, err := getIndexLogs(t, fx.store)
	require.NoError(t, err)
	assert.Empty(t, logs)

	// Test adding logs
	logData1 := []byte("log entry 1")
	logData2 := []byte("log entry 2")
	var idx1, idx2 uint64

	err = fx.store.TxUpdate(func(txn *badger.Txn) error {
		var err error
		idx1, err = fx.store.PushIndexLog(txn, logData1)
		if err != nil {
			return err
		}
		idx2, err = fx.store.PushIndexLog(txn, logData2)
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(1), idx1)
	assert.Equal(t, uint64(2), idx2)

	// Verify logs were stored correctly
	logs, err = getIndexLogs(t, fx.store)
	require.NoError(t, err)
	require.Len(t, logs, 2)
	assert.Equal(t, uint64(1), logs[0].Idx)
	assert.Equal(t, uint64(2), logs[1].Idx)
	assert.Equal(t, logData1, logs[0].Data)
	assert.Equal(t, logData2, logs[1].Data)

	// Test log deletion
	err = fx.store.TxUpdate(func(txn *badger.Txn) error {
		return fx.store.DeleteIndexLogs(txn, []uint64{idx1})
	})
	require.NoError(t, err)

	logs, err = getIndexLogs(t, fx.store)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, uint64(2), logs[0].Idx)
	assert.Equal(t, logData2, logs[0].Data)

	// Verify empty deletion doesn't affect anything
	err = fx.store.TxUpdate(func(txn *badger.Txn) error {
		return fx.store.DeleteIndexLogs(txn, []uint64{})
	})
	require.NoError(t, err)

	logs, err = getIndexLogs(t, fx.store)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
}

func TestLightFileNodeStore_KeyManagement(t *testing.T) {
	testCases := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "blockKey",
			testFunc: func(t *testing.T) {
				testData := []byte("test block data")
				_, testCID := createTestBlock(t, testData)

				key := blockKey(testCID)
				assert.True(t, bytes.HasPrefix(key, []byte(prefixFileNode+separator+blockType+separator)))
				assert.Contains(t, string(key), testCID.String())
			},
		},
		{
			name: "snapshotKey",
			testFunc: func(t *testing.T) {
				key := snapshotKey()
				assert.Equal(t, prefixFileNode+separator+snapshotType+separator+currentSnapshotSuffix, string(key))
			},
		},
		{
			name: "logKey",
			testFunc: func(t *testing.T) {
				key := logKey(123)
				assert.Equal(t, prefixFileNode+separator+logType+separator+"123", string(key))
			},
		},
		{
			name: "parseIdxFromKey",
			testFunc: func(t *testing.T) {
				key := logKey(456)
				idx, err := parseIdxFromKey(key)
				require.NoError(t, err)
				assert.Equal(t, uint64(456), idx)

				_, err = parseIdxFromKey([]byte("malformed:key"))
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestLightFileNodeStore_TransactionOperations(t *testing.T) {
	fx := newFixture(t)
	defer fx.cleanup(t)

	t.Run("TxView success", func(t *testing.T) {
		viewCalled := false
		err := fx.store.TxView(func(txn *badger.Txn) error {
			viewCalled = true
			return nil
		})
		require.NoError(t, err)
		assert.True(t, viewCalled)
	})

	t.Run("TxView error", func(t *testing.T) {
		expectedErr := errors.New("view error")
		err := fx.store.TxView(func(txn *badger.Txn) error {
			return expectedErr
		})
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("TxUpdate success", func(t *testing.T) {
		updateCalled := false
		err := fx.store.TxUpdate(func(txn *badger.Txn) error {
			updateCalled = true
			return nil
		})
		require.NoError(t, err)
		assert.True(t, updateCalled)
	})

	t.Run("TxUpdate error", func(t *testing.T) {
		expectedErr := errors.New("update error")
		err := fx.store.TxUpdate(func(txn *badger.Txn) error {
			return expectedErr
		})
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestLightFileNodeStore_GetNextIdx(t *testing.T) {
	fx := newFixture(t)
	defer fx.cleanup(t)

	// Test initial index
	var idx uint64
	err := fx.store.TxUpdate(func(txn *badger.Txn) error {
		var err error
		idx, err = fx.store.getNextIdx(txn, logKeyPrefix())
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(1), idx)

	// Test index increment after adding an entry
	err = fx.store.TxUpdate(func(txn *badger.Txn) error {
		_, err := fx.store.PushIndexLog(txn, []byte("test log"))
		return err
	})
	require.NoError(t, err)

	err = fx.store.TxUpdate(func(txn *badger.Txn) error {
		var err error
		idx, err = fx.store.getNextIdx(txn, logKeyPrefix())
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(2), idx)
}

// Helper functions

func createTestBlock(t *testing.T, data []byte) (blocks.Block, cid.Cid) {
	t.Helper()
	mhash, err := mh.Sum(data, mh.SHA2_256, -1)
	require.NoError(t, err)

	c := cid.NewCidV1(cid.Raw, mhash)
	block, err := blocks.NewBlockWithCid(data, c)
	require.NoError(t, err)

	return block, c
}

func getIndexLogs(t *testing.T, store *lightFileNodeStore) ([]IndexLog, error) {
	t.Helper()
	var logs []IndexLog
	err := store.TxView(func(txn *badger.Txn) error {
		var readErr error
		logs, readErr = store.GetIndexLogs(txn)
		return readErr
	})
	return logs, err
}

type fixture struct {
	app           *app.App
	configService *configServiceMock
	store         *lightFileNodeStore
}

func (f *fixture) cleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, f.app.Close(ctx))
}

func newFixture(t *testing.T) *fixture {
	tempDir := t.TempDir()

	f := &fixture{
		app: new(app.App),
		configService: &configServiceMock{
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
		store: New().(*lightFileNodeStore),
	}

	f.app.
		Register(f.configService).
		Register(f.store)

	t.Logf("Starting test app with temp dir: %s", tempDir)
	require.NoError(t, f.app.Start(context.Background()))
	return f
}

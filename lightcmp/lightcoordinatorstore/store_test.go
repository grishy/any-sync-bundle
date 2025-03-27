package lightcoordinatorstore

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/coordinator/coordinatorproto"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grishy/any-sync-bundle/lightcmp/lightdb"
)

type testFixture struct {
	app      *app.App
	srvCfg   *configServiceMock
	srvDB    dbService
	srvStore *lightcoordinatorstore
}

func newTestFixture(t *testing.T) *testFixture {
	t.Helper()

	f := &testFixture{
		app: new(app.App),
		srvCfg: &configServiceMock{
			dbStore: t.TempDir(),
		},
		srvDB:    lightdb.New(),
		srvStore: New(),
	}

	f.app.
		Register(f.srvCfg).
		Register(f.srvDB).
		Register(f.srvStore)

	require.NoError(t, f.app.Start(context.TODO()))

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
		defer cancel()

		require.NoError(t, f.app.Close(ctx))
	})

	return f
}

// reusableFixture creates a new fixture that reuses the database from an existing fixture.
// This simulates restarting the app with the same underlying database.
func reusableFixture(t *testing.T, sourceFixture *testFixture) *testFixture {
	t.Helper()

	newFixture := &testFixture{
		app: new(app.App),
		srvCfg: &configServiceMock{
			dbStore: sourceFixture.srvCfg.dbStore, // Reuse the same DB directory
		},
		srvDB:    lightdb.New(),
		srvStore: New(),
	}

	newFixture.app.
		Register(newFixture.srvCfg).
		Register(newFixture.srvDB).
		Register(newFixture.srvStore)

	require.NoError(t, newFixture.app.Start(context.TODO()))

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
		defer cancel()
		require.NoError(t, newFixture.app.Close(ctx))
	})

	return newFixture
}

type configServiceMock struct {
	dbStore string
}

func (m *configServiceMock) Init(a *app.App) error {
	return nil
}

func (m *configServiceMock) Name() string {
	return "config"
}

func (m *configServiceMock) GetDBDir() string {
	return m.dbStore
}

func TestDeleteLogAdd(t *testing.T) {
	fx := newTestFixture(t)

	t.Run("add single record", func(t *testing.T) {
		var id string
		var err error

		err = fx.srvDB.TxUpdate(func(txn *badger.Txn) error {
			id, err = fx.srvStore.DeleteLogAdd(txn, "spaceId1", "fileGroup1", coordinatorproto.DeletionLogRecordStatus_Ok)
			return err
		})

		require.NoError(t, err)
		assert.NotEmpty(t, id)
	})

	t.Run("add multiple records", func(t *testing.T) {
		// Add several records
		for i := 0; i < 3; i++ {
			spaceId := fmt.Sprintf("space%d", i)
			fileGroup := fmt.Sprintf("fileGroup%d", i)
			status := coordinatorproto.DeletionLogRecordStatus_RemovePrepare

			err := fx.srvDB.TxUpdate(func(txn *badger.Txn) error {
				id, err := fx.srvStore.DeleteLogAdd(txn, spaceId, fileGroup, status)
				if err != nil {
					return err
				}
				assert.NotEmpty(t, id)
				return nil
			})

			require.NoError(t, err)
		}
	})

	t.Run("add in parallel", func(t *testing.T) {
		const numParallel = 50

		// Channel to collect results
		results := make(chan string, numParallel)
		var wg sync.WaitGroup
		wg.Add(numParallel)

		// Launch goroutines to add records in parallel
		for i := 0; i < numParallel; i++ {
			go func(idx int) {
				defer wg.Done()

				spaceId := fmt.Sprintf("parallelSpace%d", idx)
				fileGroup := fmt.Sprintf("parallelFileGroup%d", idx)
				status := coordinatorproto.DeletionLogRecordStatus_Ok

				var id string
				err := fx.srvDB.TxUpdate(func(txn *badger.Txn) error {
					var err error
					id, err = fx.srvStore.DeleteLogAdd(txn, spaceId, fileGroup, status)
					return err
				})

				if err == nil && id != "" {
					results <- id
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(results)

		// Collect the IDs
		ids := make(map[string]bool)
		for id := range results {
			ids[id] = true
		}

		// Verify we got the expected number of successful additions
		assert.Equal(t, numParallel, len(ids), "Expected all parallel operations to succeed with unique IDs")
	})
}

func TestDeleteLogGetAfter(t *testing.T) {
	fx := newTestFixture(t)

	// Setup: Add a known set of records to test against
	recordIds := make([]string, 10)
	for i := 0; i < 10; i++ {
		spaceId := fmt.Sprintf("testSpace%d", i)
		fileGroup := fmt.Sprintf("testFileGroup%d", i)
		status := coordinatorproto.DeletionLogRecordStatus_Ok

		err := fx.srvDB.TxUpdate(func(txn *badger.Txn) error {
			id, err := fx.srvStore.DeleteLogAdd(txn, spaceId, fileGroup, status)
			if err != nil {
				return err
			}
			recordIds[i] = id
			return nil
		})
		require.NoError(t, err)
	}

	t.Run("get all records", func(t *testing.T) {
		var records []DeleteLogRecord
		var hasMore bool
		var err error

		err = fx.srvDB.TxView(func(txn *badger.Txn) error {
			// Get all records (using default limit)
			records, hasMore, err = fx.srvStore.DeleteLogGetAfter(txn, "", 0)
			return err
		})
		require.NoError(t, err)

		// Should return defaultDeleteLogLimit records or fewer
		assert.LessOrEqual(t, len(records), defaultDeleteLogLimit)
		// Since we only inserted 10 records, hasMore should be false
		assert.False(t, hasMore)

		// Verify records are returned in ascending order by ID
		for i := 1; i < len(records); i++ {
			prevId, _ := strconv.ParseUint(records[i-1].Id, 10, 64)
			curId, _ := strconv.ParseUint(records[i].Id, 10, 64)
			assert.Less(t, prevId, curId, "Records should be returned in ascending order by ID")
		}
	})

	t.Run("get with limit", func(t *testing.T) {
		var records []DeleteLogRecord
		var hasMore bool
		var err error

		const limit = 5
		err = fx.srvDB.TxView(func(txn *badger.Txn) error {
			records, hasMore, err = fx.srvStore.DeleteLogGetAfter(txn, "", limit)
			return err
		})
		require.NoError(t, err)

		// Should return exactly 'limit' records
		assert.Equal(t, limit, len(records))
		// Since we have more records than the limit, hasMore should be true
		assert.True(t, hasMore)
	})

	t.Run("get after specific ID", func(t *testing.T) {
		var firstBatch []DeleteLogRecord
		var secondBatch []DeleteLogRecord
		var hasMore bool
		var err error

		// First get some records
		const limit = 4
		err = fx.srvDB.TxView(func(txn *badger.Txn) error {
			firstBatch, hasMore, err = fx.srvStore.DeleteLogGetAfter(txn, "", limit)
			return err
		})
		require.NoError(t, err)
		assert.Len(t, firstBatch, limit)
		assert.True(t, hasMore)

		// Now get the next batch using the last ID from first batch
		lastId := firstBatch[len(firstBatch)-1].Id
		err = fx.srvDB.TxView(func(txn *badger.Txn) error {
			secondBatch, hasMore, err = fx.srvStore.DeleteLogGetAfter(txn, lastId, limit)
			return err
		})
		require.NoError(t, err)
		assert.Len(t, secondBatch, limit)
		assert.True(t, hasMore)

		// Verify the second batch starts after the last ID from first batch
		for _, record := range secondBatch {
			recordId, _ := strconv.ParseUint(record.Id, 10, 64)
			lastIdInt, _ := strconv.ParseUint(lastId, 10, 64)
			assert.Greater(t, recordId, lastIdInt, "Second batch records should have IDs greater than the afterId")
		}
	})

	t.Run("invalid afterId", func(t *testing.T) {
		var err error

		err = fx.srvDB.TxView(func(txn *badger.Txn) error {
			_, _, err = fx.srvStore.DeleteLogGetAfter(txn, "not-a-number", 5)
			return err
		})

		// Should return an error for invalid afterId
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid afterId")
	})

	t.Run("get with large limit", func(t *testing.T) {
		var records []DeleteLogRecord
		var err error

		err = fx.srvDB.TxView(func(txn *badger.Txn) error {
			// Use a limit larger than defualt limit
			records, _, err = fx.srvStore.DeleteLogGetAfter(txn, "", defaultDeleteLogLimit*2)
			return err
		})
		require.NoError(t, err)

		// Should respect defaultDeleteLogLimit
		assert.LessOrEqual(t, len(records), defaultDeleteLogLimit)
	})
}

func TestDeleteLogIndexOnStart(t *testing.T) {
	// Create first fixture with initial data
	fx1 := newTestFixture(t)
	const recordCount = 5

	// Add records in a single transaction for efficiency
	recordIds := make([]string, recordCount)
	err := fx1.srvDB.TxUpdate(func(txn *badger.Txn) error {
		for i := 0; i < recordCount; i++ {
			id, err := fx1.srvStore.DeleteLogAdd(
				txn,
				fmt.Sprintf("indexSpace%d", i),
				fmt.Sprintf("indexFileGroup%d", i),
				coordinatorproto.DeletionLogRecordStatus_Ok,
			)
			if err != nil {
				return err
			}
			recordIds[i] = id
		}
		return nil
	})
	require.NoError(t, err)

	// Get highest record ID to verify continuity after restart
	lastId, err := strconv.ParseUint(recordIds[recordCount-1], 10, 64)
	require.NoError(t, err)

	// Close first fixture to simulate app shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	require.NoError(t, fx1.app.Close(ctx))
	cancel()

	// Create second fixture with same database to simulate restart
	fx2 := reusableFixture(t, fx1)

	// Test index recovery: add new record and verify ID continuity
	var newId string
	err = fx2.srvDB.TxUpdate(func(txn *badger.Txn) error {
		newId, err = fx2.srvStore.DeleteLogAdd(
			txn,
			"newSpace",
			"newFileGroup",
			coordinatorproto.DeletionLogRecordStatus_RemovePrepare,
		)
		return err
	})
	require.NoError(t, err)

	// Verify new ID continues from previous highest ID
	newIdNum, err := strconv.ParseUint(newId, 10, 64)
	require.NoError(t, err)
	assert.Greater(t, newIdNum, lastId, "New record ID should continue from previous highest ID")

	// Verify all records are retrievable and correctly sorted
	var records []DeleteLogRecord
	err = fx2.srvDB.TxView(func(txn *badger.Txn) error {
		records, _, err = fx2.srvStore.DeleteLogGetAfter(txn, "", 0)
		return err
	})
	require.NoError(t, err)
	assert.Len(t, records, recordCount+1, "Should contain all original records plus the new one")

	// Check records are sorted by ID
	for i := 1; i < len(records); i++ {
		prevId, _ := strconv.ParseUint(records[i-1].Id, 10, 64)
		curId, _ := strconv.ParseUint(records[i].Id, 10, 64)
		assert.Less(t, prevId, curId, "Records should be in ascending ID order")
	}
}

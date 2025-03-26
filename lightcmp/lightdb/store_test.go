package lightdb

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testFixture struct {
	app           *app.App
	configService *configServiceMock
	store         *lightdb
	tempDir       string
}

func (f *testFixture) cleanup(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, f.app.Close(ctx))
}

func newTestFixture(t *testing.T) *testFixture {
	t.Helper()
	tempDir := t.TempDir()

	f := &testFixture{
		app:     new(app.App),
		tempDir: tempDir,
		configService: &configServiceMock{
			GetDBDirFunc: func() string { return tempDir },
			InitFunc:     func(a *app.App) error { return nil },
			NameFunc:     func() string { return "config" },
		},
	}
	f.store = New()

	f.app.Register(f.configService).Register(f.store)
	require.NoError(t, f.app.Start(context.Background()))

	t.Cleanup(func() { f.cleanup(t) })
	return f
}

func TestLightStoreInit(t *testing.T) {
	store := New()
	a := new(app.App)
	configService := &configServiceMock{
		GetDBDirFunc: func() string { return t.TempDir() },
		InitFunc:     func(a *app.App) error { return nil },
		NameFunc:     func() string { return "config" },
	}

	a.Register(configService)
	err := store.Init(a)
	assert.NoError(t, err)
	assert.Equal(t, configService, store.srvCfg)
	assert.Equal(t, CName, store.Name())
}

func TestLightStoreRun(t *testing.T) {
	fx := newTestFixture(t)

	// Verify the store directory was created
	_, err := os.Stat(fx.tempDir)
	assert.NoError(t, err)

	// Verify DB is operational by performing a simple transaction
	err = fx.store.TxUpdate(func(txn *badger.Txn) error {
		return txn.Set([]byte("test-key"), []byte("test-value"))
	})
	assert.NoError(t, err)

	var value []byte
	err = fx.store.TxView(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("test-key"))
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	assert.NoError(t, err)
	assert.Equal(t, []byte("test-value"), value)
}

func TestTransactionOperations(t *testing.T) {
	fx := newTestFixture(t)

	t.Run("TxView success", func(t *testing.T) {
		err := fx.store.TxView(func(txn *badger.Txn) error {
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("TxView error", func(t *testing.T) {
		expectedErr := errors.New("test view error")
		err := fx.store.TxView(func(txn *badger.Txn) error {
			return expectedErr
		})
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("TxUpdate success", func(t *testing.T) {
		err := fx.store.TxUpdate(func(txn *badger.Txn) error {
			return txn.Set([]byte("update-key"), []byte("update-value"))
		})
		assert.NoError(t, err)

		// Verify the update worked
		var value []byte
		err = fx.store.TxView(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte("update-key"))
			if err != nil {
				return err
			}
			value, err = item.ValueCopy(nil)
			return err
		})
		assert.NoError(t, err)
		assert.Equal(t, []byte("update-value"), value)
	})

	t.Run("TxUpdate error", func(t *testing.T) {
		expectedErr := errors.New("test update error")
		err := fx.store.TxUpdate(func(txn *badger.Txn) error {
			return expectedErr
		})
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestLightStoreClose(t *testing.T) {
	fx := newTestFixture(t)

	ctx := context.Background()
	err := fx.store.Close(ctx)
	assert.NoError(t, err)

	// Verify DB is closed by trying to use it - this should fail
	err = fx.store.TxView(func(txn *badger.Txn) error {
		return nil
	})
	assert.Error(t, err)
}

func TestLightStoreRunOpenError(t *testing.T) {
	// // TODO: Fix file handle cleanup issue on Windows where the temporary file remains in use
	// // causing "The process cannot access the file because it is being used by another process" errors
	// if runtime.GOOS == "windows" {
	// 	t.Skip("Skipping test on Windows due to file handle cleanup issues")
	// }

	store := New()
	a := new(app.App)

	// Set up a mock config that points to an invalid path that will cause BadgerDB to fail
	configService := &configServiceMock{
		GetDBDirFunc: func() string {
			// Create a directory and a file, so we can't create the DB in file
			dbFilepath := filepath.Join(t.TempDir(), "not-a-directory")
			file, err := os.Create(dbFilepath)
			require.NoError(t, err)
			require.NoError(t, file.Close())

			return dbFilepath
		},
		InitFunc: func(a *app.App) error { return nil },
		NameFunc: func() string { return "config" },
	}

	// Register the config service
	a.Register(configService)
	require.NoError(t, store.Init(a))

	// Try to run the store, which should fail when opening the database
	err := store.Run(context.Background())

	// Verify that an error was returned
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open badger db")
}

func TestStartRunGC(t *testing.T) {
	// Create a mock DB that implements the dbGC interface
	mockDB := newMockBadgerDB(t, []error{
		nil,                 // First call succeeds
		nil,                 // Second call succeeds
		nil,                 // Fourth call succeeds
		badger.ErrNoRewrite, // Fifth call indicates no more files to GC
		nil,                 // Sixth call succeeds (shouldn't be reached)
	})

	// Create a configuration for testing with short durations
	cfg := storeConfig{
		gcInterval:    10 * time.Millisecond, // Short interval for testing
		maxGCDuration: 1000 * time.Millisecond,
		gcThreshold:   0.5,
	}

	// Run the GC logic with a context we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startRunGC(ctx, cfg, mockDB)
	}()

	// Give it time to process at least one GC cycle
	time.Sleep(cfg.gcInterval + cfg.gcInterval/2)
	cancel()
	wg.Wait()

	assert.Equal(t, 4, mockDB.gcCalls)
}

func TestStartRunGCError(t *testing.T) {
	// Create a mock DB that implements the dbGC interface
	mockDB := newMockBadgerDB(t, []error{
		nil,                    // First call succeeds
		nil,                    // Second call succeeds
		errors.New("gc error"), // Third call fails
		nil,                    // Fourth call succeeds (but shouldn't be reached if we stop on error)
		badger.ErrNoRewrite,    // Fifth call indicates no more files to GC (shouldn't be reached)
	})

	// Create a configuration for testing with short durations
	cfg := storeConfig{
		gcInterval:    10 * time.Millisecond, // Short interval for testing
		maxGCDuration: 1000 * time.Millisecond,
		gcThreshold:   0.5,
	}

	// Run the GC logic with a context we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startRunGC(ctx, cfg, mockDB)
	}()

	// Give it time to process at least one GC cycle
	time.Sleep(cfg.gcInterval + cfg.gcInterval/2)
	cancel()
	wg.Wait()

	assert.Equal(t, 3, mockDB.gcCalls)
}

// mockBadgerDB implements the dbGC interface for testing
type mockBadgerDB struct {
	t         *testing.T
	gcCalls   int
	gcResults []error
	mu        sync.Mutex
}

func newMockBadgerDB(t *testing.T, gcResults []error) *mockBadgerDB {
	return &mockBadgerDB{
		t:         t,
		gcResults: gcResults,
	}
}

func (m *mockBadgerDB) RunValueLogGC(threshold float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.t.Logf("GC calls: %d, threshold: %f", m.gcCalls, threshold)

	var err error
	if m.gcCalls < len(m.gcResults) {
		err = m.gcResults[m.gcCalls]
	} else {
		err = errors.New("unexpected gc call")
	}
	m.gcCalls++
	return err
}

func TestBadgerLogger(t *testing.T) {
	logger := badgerLogger{}

	// These assertions just verify that the methods don't panic
	// since we can't easily capture the log output
	assert.NotPanics(t, func() {
		logger.Debugf("test debug message %s", "arg")
		logger.Infof("test info message %s", "arg")
		logger.Warningf("test warning message %s", "arg")
		logger.Errorf("test error message %s", "arg")
	})
}

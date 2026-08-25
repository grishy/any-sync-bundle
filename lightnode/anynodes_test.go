package lightnode

import (
	"reflect"
	"testing"

	coordinatorConfig "github.com/anyproto/any-sync-coordinator/config"
	"github.com/anyproto/any-sync-coordinator/coordinator"
	"github.com/anyproto/any-sync-coordinator/invitestore"
	filenodeConfig "github.com/anyproto/any-sync-filenode/config"
	"github.com/anyproto/any-sync-filenode/store/s3store"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_ func(*filenodeConfig.Config, string) s3store.S3Store = selectFileStore
	_ nodeconf.HistoryStore                                = (*sharedNodeConfStoreComponent)(nil)
)

// The upstream coordinator owns its component list. This focused check keeps
// the wrapper aligned when a new required component is added there.
func TestNewCoordinatorAppRegistersInviteStore(t *testing.T) {
	coordinatorApp := newCoordinatorApp(&coordinatorConfig.Config{})

	assert.NotNil(t, coordinatorApp.Component(invitestore.CName))
}

// Transports open the shared listeners. Starting them last prevents requests
// from reaching coordinator components that have not run yet, and reversed
// shutdown closes the listeners before their dependencies.
func TestNewCoordinatorAppStartsTransportsLast(t *testing.T) {
	coordinatorApp := newCoordinatorApp(&coordinatorConfig.Config{})
	names := coordinatorApp.ComponentNames()
	require.GreaterOrEqual(t, len(names), 3)

	assert.Equal(t, []string{
		coordinator.CName,
		yamux.CName,
		quic.CName,
	}, names[len(names)-3:])
}

func TestSelectFileStore_S3(t *testing.T) {
	cfg := &filenodeConfig.Config{
		S3Store: s3store.Config{
			Bucket:   "my-bucket",
			Endpoint: "https://s3.amazonaws.com",
		},
	}

	store := selectFileStore(cfg, "/tmp/filestore")

	// Check type name since s3store.New() returns an interface
	typeName := reflect.TypeOf(store).String()
	assert.Contains(t, typeName, "s3store", "should return S3 store when bucket is configured")
}

func TestSelectFileStore_BadgerDB(t *testing.T) {
	cfg := &filenodeConfig.Config{
		S3Store: s3store.Config{
			// Empty bucket means no S3
			Bucket: "",
		},
	}

	store := selectFileStore(cfg, "/tmp/filestore")

	// Check type name
	typeName := reflect.TypeOf(store).String()
	assert.Contains(t, typeName, "LightFileNodeStore", "should return BadgerDB store when bucket is empty")
}

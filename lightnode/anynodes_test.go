package lightnode

import (
	"reflect"
	"testing"

	consensusnodeConfig "github.com/anyproto/any-sync-consensusnode/config"
	consensusnodeDB "github.com/anyproto/any-sync-consensusnode/db"
	coordinatorConfig "github.com/anyproto/any-sync-coordinator/config"
	"github.com/anyproto/any-sync-coordinator/invitestore"
	filenodeConfig "github.com/anyproto/any-sync-filenode/config"
	"github.com/anyproto/any-sync-filenode/store/s3store"
	syncConfig "github.com/anyproto/any-sync-node/config"
	"github.com/anyproto/any-sync/accountservice"
	"github.com/anyproto/any-sync/acl"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/coordinator/coordinatorclient"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/peerservice"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/transport"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grishy/any-sync-bundle/config"
)

func newTestBundle() *Bundle {
	return NewBundle(&config.NodeConfigs{
		Coordinator: &coordinatorConfig.Config{},
		Consensus:   &consensusnodeConfig.Config{},
		Filenode:    &filenodeConfig.Config{},
		Sync:        &syncConfig.Config{},
	})
}

func ownComponentNames(a *app.App) []string {
	var names []string
	a.IterateComponents(func(c app.Component) { names = append(names, c.Name()) })
	return names
}

func TestNewBundleStartOrder(t *testing.T) {
	var names []string
	for _, s := range newTestBundle().Services {
		names = append(names, s.Name)
	}
	assert.Equal(t, []string{"coordinator", "consensus", "filenode", "sync"}, names)
}

// Transports accept requests as soon as they run. Every service must be running
// before then, and reverse shutdown must stop them before any service closes.
func TestNewBundleOpensListenersLast(t *testing.T) {
	bundle := newTestBundle()
	var listeners []string
	for _, l := range bundle.Listeners {
		listeners = append(listeners, l.Name())
	}
	assert.Equal(t, []string{yamux.CName, quic.CName}, listeners)

	for _, s := range bundle.Services {
		s.App.IterateComponents(func(c app.Component) {
			if _, ok := c.(transport.Transport); !ok {
				return
			}
			// An uninitialized upstream transport fails to run; only a
			// deferred one succeeds without listening.
			runnable, ok := c.(app.ComponentRunnable)
			require.True(t, ok, "%s/%s", s.Name, c.Name())
			assert.NoError(t, runnable.Run(t.Context()), "%s runs %s", s.Name, c.Name())
		})
	}
}

// Services inherit the coordinator's network through app.ChildApp. Registering
// a network component again would give that service a second, unstarted copy.
func TestServicesInheritCoordinatorNetwork(t *testing.T) {
	shared := []string{
		accountservice.CName, metric.CName, nodeconf.CName, nodeconf.CNameStore,
		peerservice.CName, pool.CName, secureservice.CName, server.CName, acl.CName,
	}
	for _, s := range newTestBundle().Services[1:] {
		own := ownComponentNames(s.App)
		for _, name := range shared {
			assert.NotContains(t, own, name, "%s registers its own %s", s.Name, name)
			assert.NotNil(t, s.App.Component(name), "%s cannot reach %s", s.Name, name)
		}
	}
}

// Upstream consensus runs the db before anything that dials, so no request
// finds the db before it has connected to MongoDB.
func TestConsensusRunsDBBeforeDialers(t *testing.T) {
	names := ownComponentNames(newTestBundle().Services[1].App)
	db := assertIndex(t, names, consensusnodeDB.CName)
	assert.Less(t, db, assertIndex(t, names, coordinatorclient.CName))
}

func assertIndex(t *testing.T, names []string, name string) int {
	t.Helper()
	for i, n := range names {
		if n == name {
			return i
		}
	}
	t.Fatalf("%s is not registered in %v", name, names)
	return -1
}

// The upstream coordinator owns its component list. This focused check keeps
// the copy aligned when a new required component is added there.
func TestNewCoordinatorAppRegistersInviteStore(t *testing.T) {
	assert.NotNil(t, newTestBundle().Services[0].App.Component(invitestore.CName))
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

package lightnodeconf

import (
	"context"
	"testing"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testFixture struct {
	app     *app.App
	cfgMock *cfgSrvMock
	service *lightNodeconf
}

func newTestFixture(t *testing.T) *testFixture {
	t.Helper()

	// Create mock node configuration.
	nodes := []nodeconf.Node{
		{
			PeerId:    "peer1",
			Types:     []nodeconf.NodeType{nodeconf.NodeTypeConsensus, nodeconf.NodeTypeFile},
			Addresses: []string{"addr1-1", "addr1-2"},
		},
		{
			PeerId:    "peer2",
			Types:     []nodeconf.NodeType{nodeconf.NodeTypeCoordinator},
			Addresses: []string{"addr2"},
		},
		{
			PeerId:    "peer3",
			Types:     []nodeconf.NodeType{nodeconf.NodeTypeFile},
			Addresses: []string{"addr3-1", "addr3-2", "addr3-3"},
		},
	}

	mockConfig := nodeconf.Configuration{
		Nodes: nodes,
	}

	// Create config service mock.
	cfgMock := &cfgSrvMock{
		GetNodeConfFunc: func() nodeconf.Configuration {
			return mockConfig
		},
		InitFunc: func(a *app.App) error {
			return nil
		},
		NameFunc: func() string {
			return "config"
		},
	}

	// Create and init app.
	a := new(app.App)
	service := New().(*lightNodeconf)

	a.Register(cfgMock).Register(service)
	require.NoError(t, a.Start(context.Background()))

	return &testFixture{
		app:     a,
		cfgMock: cfgMock,
		service: service,
	}
}

func (f *testFixture) cleanup(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, f.app.Close(ctx))
}

func TestComponentMethods(t *testing.T) {
	fx := newTestFixture(t)
	defer fx.cleanup(t)

	t.Run("Name returns correct component name", func(t *testing.T) {
		assert.Equal(t, nodeconf.CName, fx.service.Name())
	})

	t.Run("Run and Close return no errors", func(t *testing.T) {
		// These methods are already called during setup and cleanup,
		// but we test them explicitly here
		assert.NoError(t, fx.service.Run(context.Background()))
		assert.NoError(t, fx.service.Close(context.Background()))
	})
}

func TestNodeTypes(t *testing.T) {
	fx := newTestFixture(t)
	defer fx.cleanup(t)

	t.Run("returns correct types for existing node", func(t *testing.T) {
		types := fx.service.NodeTypes("peer1")
		require.Len(t, types, 2)
		assert.Contains(t, types, nodeconf.NodeTypeConsensus)
		assert.Contains(t, types, nodeconf.NodeTypeFile)
	})

	t.Run("returns nil for non-existent node", func(t *testing.T) {
		types := fx.service.NodeTypes("non-existent")
		assert.Nil(t, types)
	})
}

func TestPeerAddresses(t *testing.T) {
	fx := newTestFixture(t)
	defer fx.cleanup(t)

	t.Run("returns correct addresses for existing peer", func(t *testing.T) {
		addrs, ok := fx.service.PeerAddresses("peer3")
		assert.True(t, ok)
		require.Len(t, addrs, 3)
		assert.Equal(t, []string{"addr3-1", "addr3-2", "addr3-3"}, addrs)
	})

	t.Run("returns false for non-existent peer", func(t *testing.T) {
		addrs, ok := fx.service.PeerAddresses("non-existent")
		assert.False(t, ok)
		assert.Nil(t, addrs)
	})
}

func TestPeersByType(t *testing.T) {
	fx := newTestFixture(t)
	defer fx.cleanup(t)

	t.Run("ConsensusPeers returns correct peers", func(t *testing.T) {
		peers := fx.service.ConsensusPeers()
		require.Len(t, peers, 1)
		assert.Equal(t, "peer1", peers[0])
	})

	t.Run("CoordinatorPeers returns correct peers", func(t *testing.T) {
		peers := fx.service.CoordinatorPeers()
		require.Len(t, peers, 1)
		assert.Equal(t, "peer2", peers[0])
	})

	t.Run("FilePeers returns correct peers", func(t *testing.T) {
		peers := fx.service.FilePeers()
		require.Len(t, peers, 2)
		assert.Contains(t, peers, "peer1")
		assert.Contains(t, peers, "peer3")
	})
}

func TestUnimplementedMethods(t *testing.T) {
	fx := newTestFixture(t)
	defer fx.cleanup(t)

	// Test that unimplemented methods panic as expected.
	assert.Panics(t, func() { fx.service.CHash() })
	assert.Panics(t, func() { fx.service.Configuration() })
	assert.Panics(t, func() { fx.service.Id() })
	assert.Panics(t, func() { fx.service.IsResponsible("space") })
	assert.Panics(t, func() { fx.service.NamingNodePeers() })
	assert.Panics(t, func() { fx.service.NetworkCompatibilityStatus() })
	assert.Panics(t, func() { fx.service.NodeIds("space") })
	assert.Panics(t, func() { fx.service.Partition("space") })
	assert.Panics(t, func() { fx.service.PaymentProcessingNodePeers() })
}

// Mock implementation of the cfgSrv interface.
type cfgSrvMock struct {
	GetNodeConfFunc func() nodeconf.Configuration
	InitFunc        func(a *app.App) error
	NameFunc        func() string
}

func (m *cfgSrvMock) GetNodeConf() nodeconf.Configuration {
	return m.GetNodeConfFunc()
}

func (m *cfgSrvMock) Init(a *app.App) error {
	return m.InitFunc(a)
}

func (m *cfgSrvMock) Name() string {
	return m.NameFunc()
}

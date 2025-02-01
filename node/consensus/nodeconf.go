package consensus

import (
	"context"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/nodeconf"
	"go.uber.org/zap"
)

var logNodeconf = logger.NewNamed("bundle.consensus.nodeconf")

type bundleNodeconf struct {
	Network nodeconf.Configuration

	// Just implement all and panic on wrong call
	nodeconf.Service
}

func (s *bundleNodeconf) Init(a *app.App) error {
	logNodeconf.Info("call Init")

	return nil
}

func (s *bundleNodeconf) Name() (name string) {
	return nodeconf.CName
}

func (s *bundleNodeconf) Run(_ context.Context) error {
	logNodeconf.Info("call Run")

	return nil
}

// NodeTypes returns list of known nodeTypes by nodeId, if node not registered in configuration will return empty list
// Implemented for secureservice
func (s *bundleNodeconf) NodeTypes(nodeId string) []nodeconf.NodeType {
	logNodeconf.Info("call NodeTypes", zap.String("nodeId", nodeId))

	for _, m := range s.Network.Nodes {
		if m.PeerId == nodeId {
			return m.Types
		}
	}

	return nil
}

// PeerAddresses returns peer addresses by peer id
// Implemented for peerservice
func (s *bundleNodeconf) PeerAddresses(peerId string) (addrs []string, ok bool) {
	logNodeconf.Info("call PeerAddresses", zap.String("peerId", peerId))

	for _, m := range s.Network.Nodes {
		if m.PeerId == peerId {
			return m.Addresses, true
		}
	}

	return nil, false
}

func (s *bundleNodeconf) Close(ctx context.Context) error {
	logNodeconf.Info("call Close")

	return nil
}

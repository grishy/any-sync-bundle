package consensus

import (
	"context"

	"github.com/anyproto/any-sync-consensusnode/config"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/go-chash"

	"go.uber.org/zap"
)

var logNodeconf = logger.NewNamed("light.consensus." + config.CName)

type cfgSrv interface {
	GetNodeConf() nodeconf.Configuration
}

type lightNodeconf struct {
	network nodeconf.Configuration
}

func NewLightNodeconf() nodeconf.Service {
	return &lightNodeconf{}
}

//
// App Component
//

func (s *lightNodeconf) Init(a *app.App) error {
	logNodeconf.Info("call Init")

	s.network = a.MustComponent(config.CName).(cfgSrv).GetNodeConf()

	return nil
}

func (s *lightNodeconf) Name() (name string) {
	return nodeconf.CName
}

//
// App Component Runnable
//

func (s *lightNodeconf) Run(_ context.Context) error {
	logNodeconf.Info("call Run")

	return nil
}

func (s *lightNodeconf) Close(ctx context.Context) error {
	logNodeconf.Info("call Close")

	return nil
}

//
// Component
//

// NodeTypes returns list of known nodeTypes by nodeId, if node not registered in configuration will return empty list
// Implemented for secureservice
func (s *lightNodeconf) NodeTypes(nodeId string) []nodeconf.NodeType {
	logNodeconf.Info("call NodeTypes", zap.String("nodeId", nodeId))

	for _, m := range s.network.Nodes {
		if m.PeerId == nodeId {
			return m.Types
		}
	}

	return nil
}

// PeerAddresses returns peer addresses by peer id
// Implemented for peerservice
func (s *lightNodeconf) PeerAddresses(peerId string) (addrs []string, ok bool) {
	logNodeconf.Info("call PeerAddresses", zap.String("peerId", peerId))

	for _, m := range s.network.Nodes {
		if m.PeerId == peerId {
			return m.Addresses, true
		}
	}

	return nil, false
}

// TODO: Check that this is not called
// Auto generated code to panic if called

// CHash implements nodeconf.Service.
func (s *lightNodeconf) CHash() chash.CHash {
	panic("unimplemented")
}

// Configuration implements nodeconf.Service.
func (s *lightNodeconf) Configuration() nodeconf.Configuration {
	panic("unimplemented")
}

// ConsensusPeers implements nodeconf.Service.
func (s *lightNodeconf) ConsensusPeers() []string {
	panic("unimplemented")
}

// CoordinatorPeers implements nodeconf.Service.
func (s *lightNodeconf) CoordinatorPeers() []string {
	panic("unimplemented")
}

// FilePeers implements nodeconf.Service.
func (s *lightNodeconf) FilePeers() []string {
	panic("unimplemented")
}

// Id implements nodeconf.Service.
func (s *lightNodeconf) Id() string {
	panic("unimplemented")
}

// IsResponsible implements nodeconf.Service.
func (s *lightNodeconf) IsResponsible(spaceId string) bool {
	panic("unimplemented")
}

// NamingNodePeers implements nodeconf.Service.
func (s *lightNodeconf) NamingNodePeers() []string {
	panic("unimplemented")
}

// NetworkCompatibilityStatus implements nodeconf.Service.
func (s *lightNodeconf) NetworkCompatibilityStatus() nodeconf.NetworkCompatibilityStatus {
	panic("unimplemented")
}

// NodeIds implements nodeconf.Service.
func (s *lightNodeconf) NodeIds(spaceId string) []string {
	panic("unimplemented")
}

// Partition implements nodeconf.Service.
func (s *lightNodeconf) Partition(spaceId string) (part int) {
	panic("unimplemented")
}

// PaymentProcessingNodePeers implements nodeconf.Service.
func (s *lightNodeconf) PaymentProcessingNodePeers() []string {
	panic("unimplemented")
}

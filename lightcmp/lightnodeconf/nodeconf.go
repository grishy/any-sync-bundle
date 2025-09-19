package lightnodeconf

import (
	"context"

	"github.com/anyproto/any-sync-consensusnode/config"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/go-chash"

	"go.uber.org/zap"
)

var log = logger.NewNamed("light.consensus." + config.CName)

type cfgSrv interface {
	GetNodeConf() nodeconf.Configuration
}

type lightNodeconf struct {
	network nodeconf.Configuration
}

func New() nodeconf.Service {
	return &lightNodeconf{}
}

//
// App Component.
//

func (nc *lightNodeconf) Init(a *app.App) error {
	log.Info("call Init")

	nc.network = app.MustComponent[cfgSrv](a).GetNodeConf()

	return nil
}

func (nc *lightNodeconf) Name() (name string) {
	return nodeconf.CName
}

//
// App Component Runnable.
//

func (nc *lightNodeconf) Run(_ context.Context) error {
	log.Info("call Run")

	return nil
}

func (nc *lightNodeconf) Close(ctx context.Context) error {
	log.Info("call Close")

	return nil
}

//
// Component.
//

// NodeTypes returns list of known nodeTypes by nodeId, if node not registered in configuration will return empty list.
// Implemented for secureservice.
func (nc *lightNodeconf) NodeTypes(nodeId string) []nodeconf.NodeType {
	log.Info("call NodeTypes", zap.String("nodeId", nodeId))

	for _, m := range nc.network.Nodes {
		if m.PeerId == nodeId {
			return m.Types
		}
	}

	return nil
}

// PeerAddresses returns peer addresses by peer id.
// Implemented for peerservice.
func (nc *lightNodeconf) PeerAddresses(peerId string) (addrs []string, ok bool) {
	log.Info("call PeerAddresses", zap.String("peerId", peerId))

	for _, m := range nc.network.Nodes {
		if m.PeerId == peerId {
			return m.Addresses, true
		}
	}

	return nil, false
}

// TODO: Check that this is not called.
// Auto generated code to panic if called.

// CHash implements nodeconf.Service.
func (nc *lightNodeconf) CHash() chash.CHash {
	panic("unimplemented")
}

// Configuration implements nodeconf.Service.
func (nc *lightNodeconf) Configuration() nodeconf.Configuration {
	panic("unimplemented")
}

// ConsensusPeers implements nodeconf.Service.
func (nc *lightNodeconf) ConsensusPeers() []string {
	return nc.findPeer(nodeconf.NodeTypeConsensus)
}

// CoordinatorPeers implements nodeconf.Service.
func (nc *lightNodeconf) CoordinatorPeers() []string {
	return nc.findPeer(nodeconf.NodeTypeCoordinator)
}

// FilePeers implements nodeconf.Service.
func (nc *lightNodeconf) FilePeers() []string {
	return nc.findPeer(nodeconf.NodeTypeFile)
}

// Id implements nodeconf.Service.
func (nc *lightNodeconf) Id() string {
	panic("unimplemented")
}

// IsResponsible implements nodeconf.Service.
func (nc *lightNodeconf) IsResponsible(spaceId string) bool {
	panic("unimplemented")
}

// NamingNodePeers implements nodeconf.Service.
func (nc *lightNodeconf) NamingNodePeers() []string {
	panic("unimplemented")
}

// NetworkCompatibilityStatus implements nodeconf.Service.
func (nc *lightNodeconf) NetworkCompatibilityStatus() nodeconf.NetworkCompatibilityStatus {
	panic("unimplemented")
}

// NodeIds implements nodeconf.Service.
func (nc *lightNodeconf) NodeIds(spaceId string) []string {
	panic("unimplemented")
}

// Partition implements nodeconf.Service.
func (nc *lightNodeconf) Partition(spaceId string) (part int) {
	panic("unimplemented")
}

// PaymentProcessingNodePeers implements nodeconf.Service.
func (nc *lightNodeconf) PaymentProcessingNodePeers() []string {
	panic("unimplemented")
}

func (nc *lightNodeconf) findPeer(nodeType nodeconf.NodeType) (peerIDs []string) {
	for _, n := range nc.network.Nodes {
		if n.HasType(nodeType) {
			peerIDs = append(peerIDs, n.PeerId)
		}
	}

	return peerIDs
}

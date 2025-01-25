package config

import (
	"net"
	"os"
	"time"

	"github.com/anyproto/any-sync/nodeconf"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func (bc *BundleConfig) convertExternalListenAddr(listen BundleConfigNodeShared) []string {
	if len(bc.ExternalListenAddr) == 0 {
		return []string{}
	}

	addrs := make([]string, 0, len(bc.ExternalListenAddr)*2)
	for _, addr := range bc.ExternalListenAddr {
		_, portTCP, err := net.SplitHostPort(listen.ListenTCPAddr)
		if err != nil {
			log.Panic("failed to split external listen addr", zap.Error(err))
		}

		_, portUDP, err := net.SplitHostPort(listen.ListenUDPAddr)
		if err != nil {
			log.Panic("failed to split external listen addr", zap.Error(err))
		}

		addrs = append(addrs, "quic://"+addr+":"+portUDP)
		addrs = append(addrs, addr+":"+portTCP)
	}

	return addrs
}

func (bc *BundleConfig) DumpClientConfig(cfgPath string) {
	network := nodeconf.Configuration{
		Id:        bc.ConfigID,
		NetworkId: bc.NetworkID,
		Nodes: []nodeconf.Node{
			{
				PeerId:    bc.Accounts.Coordinator.PeerId,
				Addresses: bc.convertExternalListenAddr(bc.Nodes.Coordinator.BundleConfigNodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeCoordinator,
				},
			},
			{
				PeerId:    bc.Accounts.Consensus.PeerId,
				Addresses: bc.convertExternalListenAddr(bc.Nodes.Consensus.BundleConfigNodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeConsensus,
				},
			},
			{
				PeerId:    bc.Accounts.Tree.PeerId,
				Addresses: bc.convertExternalListenAddr(bc.Nodes.Tree.BundleConfigNodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeTree,
				},
			},
			{
				PeerId:    bc.Accounts.File.PeerId,
				Addresses: bc.convertExternalListenAddr(bc.Nodes.File.BundleConfigNodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeFile,
				},
			},
		},
		CreationTime: time.Now(),
	}

	yamlData, err := yaml.Marshal(network)
	if err != nil {
		log.Panic("failed to marshal network configuration", zap.Error(err))
	}

	if err := os.WriteFile(cfgPath, yamlData, 0644); err != nil {
		log.Panic("failed to write network configuration", zap.Error(err))
	}
}

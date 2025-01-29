package config

import (
	"fmt"
	"net"
	"time"

	"github.com/anyproto/any-sync/nodeconf"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func (bc *Config) convertExternalListenAddr(listen NodeShared) []string {
	if len(bc.ExternalAddr) == 0 {
		return []string{}
	}

	addrs := make([]string, 0, len(bc.ExternalAddr)*2)
	for _, addr := range bc.ExternalAddr {
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

func (bc *Config) ClientConfig() ([]byte, error) {
	network := nodeconf.Configuration{
		Id:        bc.ConfigID,
		NetworkId: bc.NetworkID,
		Nodes: []nodeconf.Node{
			{
				PeerId:    bc.Accounts.Coordinator.PeerId,
				Addresses: bc.convertExternalListenAddr(bc.Nodes.Coordinator.NodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeCoordinator,
				},
			},
			{
				PeerId:    bc.Accounts.Consensus.PeerId,
				Addresses: bc.convertExternalListenAddr(bc.Nodes.Consensus.NodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeConsensus,
				},
			},
			{
				PeerId:    bc.Accounts.Tree.PeerId,
				Addresses: bc.convertExternalListenAddr(bc.Nodes.Tree.NodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeTree,
				},
			},
			{
				PeerId:    bc.Accounts.File.PeerId,
				Addresses: bc.convertExternalListenAddr(bc.Nodes.File.NodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeFile,
				},
			},
		},
		CreationTime: time.Now(),
	}

	yamlData, err := yaml.Marshal(network)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal network configuration: %w", err)
	}
	return yamlData, nil
}

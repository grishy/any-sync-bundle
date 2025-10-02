package config

import (
	"fmt"
	"net"
	"time"

	"github.com/anyproto/any-sync/nodeconf"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func (bc *Config) convertExternalAddrs(listen NodeShared) []string {
	// TCP and UDP
	addrs := make([]string, 0, len(bc.ExternalAddr)*2)

	for _, externalAddr := range bc.ExternalAddr {
		_, tcpPort, err := net.SplitHostPort(listen.ListenTCPAddr)
		if err != nil {
			log.With(
				zap.Error(err),
				zap.String("addr", listen.ListenTCPAddr),
			).Panic("invalid TCP listen address")
		}

		_, udpPort, err := net.SplitHostPort(listen.ListenUDPAddr)
		if err != nil {
			log.With(
				zap.Error(err),
				zap.String("addr", listen.ListenUDPAddr),
			).Panic("invalid UDP listen address")
		}

		addrs = append(addrs,
			"quic://"+externalAddr+":"+udpPort,
			externalAddr+":"+tcpPort,
		)
	}

	return addrs
}

func (bc *Config) YamlClientConfig() ([]byte, error) {
	network := nodeconf.Configuration{
		Id:        bc.ConfigID,
		NetworkId: bc.NetworkID,
		Nodes: []nodeconf.Node{
			{
				PeerId:    bc.Accounts.Coordinator.PeerId,
				Addresses: bc.convertExternalAddrs(bc.Nodes.Coordinator.NodeShared),
				Types: []nodeconf.NodeType{
					nodeconf.NodeTypeCoordinator,
					nodeconf.NodeTypeConsensus,
					nodeconf.NodeTypeTree,
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

package config

import (
	"fmt"
	"net"
	"time"

	"github.com/anyproto/any-sync/nodeconf"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func (bc *Config) convertExternalAddrs() []string {
	// TCP and UDP
	addrs := make([]string, 0, len(bc.ExternalAddr)*2)

	for _, externalAddr := range bc.ExternalAddr {
		_, tcpPort, err := net.SplitHostPort(bc.Network.ListenTCPAddr)
		if err != nil {
			log.With(
				zap.Error(err),
				zap.String("addr", bc.Network.ListenTCPAddr),
			).Panic("invalid TCP listen address")
		}

		_, udpPort, err := net.SplitHostPort(bc.Network.ListenUDPAddr)
		if err != nil {
			log.With(
				zap.Error(err),
				zap.String("addr", bc.Network.ListenUDPAddr),
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
				PeerId:    bc.Account.PeerId,
				Addresses: bc.convertExternalAddrs(),
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

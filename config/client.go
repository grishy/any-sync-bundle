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
	endpoints := bc.listenEndpoints()

	for _, externalAddr := range bc.ExternalAddr {
		addrs = append(addrs,
			"quic://"+externalAddr+":"+endpoints.udpPort,
			externalAddr+":"+endpoints.tcpPort,
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

type listenEndpoints struct {
	tcpHost string
	tcpPort string
	udpHost string
	udpPort string
}

func (bc *Config) listenEndpoints() listenEndpoints {
	tcpHost, tcpPort, err := net.SplitHostPort(bc.Network.ListenTCPAddr)
	if err != nil {
		log.With(
			zap.Error(err),
			zap.String("addr", bc.Network.ListenTCPAddr),
		).Panic("invalid TCP listen address")
	}

	udpHost, udpPort, err := net.SplitHostPort(bc.Network.ListenUDPAddr)
	if err != nil {
		log.With(
			zap.Error(err),
			zap.String("addr", bc.Network.ListenUDPAddr),
		).Panic("invalid UDP listen address")
	}

	return listenEndpoints{
		tcpHost: tcpHost,
		tcpPort: tcpPort,
		udpHost: udpHost,
		udpPort: udpPort,
	}
}

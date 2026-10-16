package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
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

// WriteClientConfig writes the network config that Anytype clients import.
// It is regenerated on every start, so it always matches the bundle config.
func (bc *Config) WriteClientConfig(path string) {
	yamlData, err := bc.YamlClientConfig()
	if err != nil {
		log.Panic("can't generate client config", zap.Error(err))
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		log.Panic("can't create client config directory", zap.Error(err), zap.String("path", path))
	}
	// Clients read this file and it holds no secrets, unlike the bundle config.
	const clientConfigMode = 0o644
	if err = os.WriteFile(path, yamlData, clientConfigMode); err != nil {
		log.Panic("can't write client config", zap.Error(err), zap.String("path", path))
	}
	log.Info("client configuration written", zap.String("path", path))
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

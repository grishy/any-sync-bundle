package consensus

import (
	"github.com/anyproto/any-sync-consensusnode/config"
	commonaccount "github.com/anyproto/any-sync/accountservice"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/rpc"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"
)

var logCfg = logger.NewNamed("bundle.consensus.config")

type bundleConfig struct {
	Account       commonaccount.Config
	Network       nodeconf.Configuration
	ListenTCPAddr string
	ListenUDPAddr string
}

func (c *bundleConfig) Init(a *app.App) error {
	logCfg.Info("call Init")

	return nil
}

func (c *bundleConfig) Name() (name string) {
	return config.CName
}

func (c *bundleConfig) GetMongo() config.Mongo {
	panic("no mongo config")
}

func (c *bundleConfig) GetDrpc() rpc.Config {
	logCfg.Info("call GetDrpc")

	return rpc.Config{
		Stream: rpc.StreamConfig{
			MaxMsgSizeMb: 256,
		},
	}
}

func (c *bundleConfig) GetAccount() commonaccount.Config {
	logCfg.Info("call GetAccount")

	return c.Account
}

func (c *bundleConfig) GetMetric() metric.Config {
	logCfg.Info("call GetMetric")

	return metric.Config{}
}

func (c *bundleConfig) GetNodeConf() nodeconf.Configuration {
	logCfg.Info("call GetNodeConf")

	return c.Network
}

func (c *bundleConfig) GetNodeConfStorePath() string {
	panic("no node conf store path")
}

func (c *bundleConfig) GetNodeConfUpdateInterval() int {
	panic("no node conf update interval")
}

func (c *bundleConfig) GetYamux() yamux.Config {
	logCfg.Info("call GetYamux")

	return yamux.Config{
		ListenAddrs: []string{c.ListenTCPAddr},
	}
}

func (c *bundleConfig) GetQuic() quic.Config {
	logCfg.Info("call GetQuic")

	return quic.Config{
		ListenAddrs: []string{c.ListenUDPAddr},
	}
}

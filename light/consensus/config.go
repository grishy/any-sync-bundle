package consensus

import (
	"github.com/anyproto/any-sync-consensusnode/config"

	commonaccount "github.com/anyproto/any-sync/accountservice"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/net/rpc"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"
)

var logCfg = logger.NewNamed("light.consensus." + config.CName)

type lightConfig struct {
	Account       commonaccount.Config
	Network       nodeconf.Configuration
	ListenTCPAddr string
	ListenUDPAddr string
	DBPath        string
}

//
// App Component
//

func (c *lightConfig) Init(a *app.App) error {
	logCfg.Info("call Init")

	return nil
}

func (c *lightConfig) Name() (name string) {
	return config.CName
}

//
// Component
//

func (c *lightConfig) GetDrpc() rpc.Config {
	logCfg.Info("call GetDrpc")

	return rpc.Config{
		Stream: rpc.StreamConfig{
			MaxMsgSizeMb: 256,
		},
	}
}

func (c *lightConfig) GetAccount() commonaccount.Config {
	logCfg.Info("call GetAccount")

	return c.Account
}

func (c *lightConfig) GetNodeConf() nodeconf.Configuration {
	logCfg.Info("call GetNodeConf")

	return c.Network
}

func (c *lightConfig) GetYamux() yamux.Config {
	logCfg.Info("call GetYamux")

	return yamux.Config{
		ListenAddrs: []string{c.ListenTCPAddr},
	}
}

func (c *lightConfig) GetQuic() quic.Config {
	logCfg.Info("call GetQuic")

	return quic.Config{
		ListenAddrs: []string{c.ListenUDPAddr},
	}
}

func (c *lightConfig) GetDBPath() string {
	logCfg.Info("call GetDBPath")

	return c.DBPath
}

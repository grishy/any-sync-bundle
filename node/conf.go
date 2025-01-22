package app

import (
	syncAccount "github.com/anyproto/any-sync/accountservice"
	syncNodeconf "github.com/anyproto/any-sync/nodeconf"
)

var confAcc = syncAccount.Config{
	PeerId:     "12D3KooWRt4bWqSE9Lz8KjVv6LMfYBpGxGj9GX7qwsSksMqGvHu7",
	PeerKey:    "7kpWxYf+DmR1VKcpQbnh442cnrXdwiEGCqgM3xRSTEhgGbutpRs1kBe1Y+fCxEnwX3KMw7qNBPI84GGUQf2lXw==",
	SigningKey: "35NuMqiKioNREOpqYZqKoxtKiXbJwonmcJy9kyc/23jThI1uFDds27UKth1uioPdm9h3o382sQHZbU4SDTG6GQ==",
}

var confNetwork = syncNodeconf.Configuration{
	Id:        "89692d43854a2eff14a6d789",
	NetworkId: "N8rUqvmknuNK4hDbFVLpKWgBuEykgMSwDY2JPwaHm5c4fKEC",
	Nodes: []syncNodeconf.Node{
		{
			PeerId:    "12D3KooWKkGLLdSQ2m5S6PSBN4bZLN6Ya2p1zDUiJQEvr3aQc2k3",
			Addresses: []string{},
			Types: []syncNodeconf.NodeType{
				syncNodeconf.NodeTypeTree,
			},
		},
		{
			PeerId:    "12D3KooWGvc8RjztFburDyW4TMYTtkea5e8mUh8ecm9kmEqp6bhD",
			Addresses: []string{},
			Types: []syncNodeconf.NodeType{
				syncNodeconf.NodeTypeCoordinator,
			},
		},
		{
			PeerId:    "12D3KooWNUeMCpyAuseBtMVkF2eRLNz8wP9itnirH4vPUvAA26bh",
			Addresses: []string{},
			Types: []syncNodeconf.NodeType{
				syncNodeconf.NodeTypeFile,
			},
		},
		{
			PeerId:    "12D3KooWDfRifdLqTWRrBCqSFEviWwSoFU58D8qPk7KbYnQTUj5Q",
			Addresses: []string{},
			Types: []syncNodeconf.NodeType{
				syncNodeconf.NodeTypeConsensus,
			},
		},
	},
}

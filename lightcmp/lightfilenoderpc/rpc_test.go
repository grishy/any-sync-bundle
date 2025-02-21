package lightfilenoderpc

import (
	"context"
	"testing"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/testutil"
	"github.com/anyproto/any-sync/acl"
	"github.com/anyproto/any-sync/acl/mock_acl"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/commonfile/fileproto"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/util/crypto"
	"github.com/dgraph-io/badger/v4"
	blocks "github.com/ipfs/go-block-format"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/grishy/any-sync-bundle/lightcmp/lightconfig"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

func TestFileNode_Add(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileId        = testutil.NewRandCid().String()
			b             = testutil.NewRandBlock(1024)
		)

		fx.aclService.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.storeService.PushBlockFunc = func(txn *badger.Txn, spaceId string, b blocks.Block) error {
			require.Equal(t, storeKey.SpaceId, spaceId)
			require.Equal(t, b.Cid().Bytes(), b.Cid().Bytes())
			require.Equal(t, b.RawData(), b.RawData())
			return nil
		}

		resp, err := fx.rpcService.BlockPush(ctx, &fileproto.BlockPushRequest{
			SpaceId: storeKey.SpaceId,
			FileId:  fileId,
			Cid:     b.Cid().Bytes(),
			Data:    b.RawData(),
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
	})

}

func newFixture(t *testing.T) *fixture {
	ctrl := gomock.NewController(t)

	fx := &fixture{
		ctrl:       ctrl,
		a:          new(app.App),
		drpcServer: server.New(),
		aclService: mock_acl.NewMockAclService(ctrl),
		rpcService: New().(*lightFileNodeRpc),
		storeService: &lightfilenodestore.StoreServiceMock{
			InitFunc: func(a *app.App) error {
				return nil
			},
			NameFunc: func() string {
				return lightfilenodestore.CName
			},
			// Default for tests
			TxViewFunc: func(f func(txn *badger.Txn) error) error {
				return f(nil)
			},
			TxUpdateFunc: func(f func(txn *badger.Txn) error) error {
				return f(nil)
			},
			// Implement the test
			GetBlockFunc:  nil,
			PushBlockFunc: nil,
		},
	}

	fx.aclService.EXPECT().Name().Return(acl.CName).AnyTimes()
	fx.aclService.EXPECT().Init(gomock.Any()).AnyTimes()
	fx.aclService.EXPECT().Run(gomock.Any()).AnyTimes()
	fx.aclService.EXPECT().Close(gomock.Any()).AnyTimes()

	fx.a.Register(&lightconfig.LightConfig{}).
		Register(fx.drpcServer).
		Register(fx.aclService).
		Register(fx.storeService).
		Register(fx.rpcService)

	require.NoError(t, fx.a.Start(context.Background()))
	return fx
}

type fixture struct {
	ctrl         *gomock.Controller
	a            *app.App
	drpcServer   server.DRPCServer
	aclService   *mock_acl.MockAclService
	rpcService   *lightFileNodeRpc
	storeService *lightfilenodestore.StoreServiceMock
}

func (fx *fixture) Finish(t *testing.T) {
	fx.ctrl.Finish()
	require.NoError(t, fx.a.Close(context.Background()))
}

func newRandKey() (context.Context, index.Key) {
	_, pubKey, _ := crypto.GenerateRandomEd25519KeyPair()
	pubKeyRaw, _ := pubKey.Marshall()
	return peer.CtxWithIdentity(context.Background(), pubKeyRaw), index.Key{
		SpaceId: testutil.NewRandSpaceId(),
		GroupId: pubKey.Account(),
	}
}

func mustPubKey(ctx context.Context) crypto.PubKey {
	pubKey, err := peer.CtxPubKey(ctx)
	if err != nil {
		panic(err)
	}
	return pubKey
}

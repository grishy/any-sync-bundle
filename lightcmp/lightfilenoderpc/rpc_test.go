package lightfilenoderpc

import (
	"context"
	"testing"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/testutil"
	"github.com/anyproto/any-sync/acl"
	"github.com/anyproto/any-sync/acl/mock_acl"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/util/crypto"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/grishy/any-sync-bundle/lightcmp/lightconfig"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

// TODO: Write test with real database

// func TestLightFileNodeRpc_BlockGet(t *testing.T) {
// 	t.Run("not found", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		var (
// 			ctx, storeKey = newRandKey()
// 			b             = testutil.NewRandBlock(1024)
// 		)

// 		fx.storeService.GetBlockFunc = func(txn *badger.Txn, k cid.Cid) (*lightfilenodestore.BlockObj, error) {
// 			return nil, fileprotoerr.ErrCIDNotFound
// 		}

// 		resp, err := fx.rpcService.BlockGet(ctx, &fileproto.BlockGetRequest{
// 			SpaceId: storeKey.SpaceId,
// 			Cid:     b.Cid().Bytes(),
// 			Wait:    false,
// 		})

// 		require.Error(t, err)
// 		require.ErrorIs(t, err, fileprotoerr.ErrCIDNotFound)
// 		require.Nil(t, resp)
// 	})

// 	t.Run("found", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		var (
// 			ctx, storeKey = newRandKey()
// 			b             = testutil.NewRandBlock(1024)
// 		)

// 		fx.storeService.GetBlockFunc = func(txn *badger.Txn, k cid.Cid) (*lightfilenodestore.BlockObj, error) {
// 			blkObj := lightfilenodestore.NewBlockObj(b.Cid()).WithData(b.RawData())
// 			return blkObj, nil
// 		}

// 		resp, err := fx.rpcService.BlockGet(ctx, &fileproto.BlockGetRequest{
// 			SpaceId: storeKey.SpaceId,
// 			Cid:     b.Cid().Bytes(),
// 			Wait:    false,
// 		})

// 		require.NoError(t, err)
// 		require.NotNil(t, resp)
// 		require.Equal(t, b.RawData(), resp.Data)
// 	})

// 	t.Run("wait", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		var (
// 			ctx, storeKey = newRandKey()
// 			b             = testutil.NewRandBlock(1024)
// 			attempts      = 0
// 		)

// 		fx.storeService.GetBlockFunc = func(txn *badger.Txn, k cid.Cid) (*lightfilenodestore.BlockObj, error) {
// 			attempts++
// 			if attempts < 3 {
// 				return nil, fileprotoerr.ErrCIDNotFound
// 			}

// 			blkObj := lightfilenodestore.NewBlockObj(b.Cid()).WithData(b.RawData())
// 			return blkObj, nil
// 		}

// 		resp, err := fx.rpcService.BlockGet(ctx, &fileproto.BlockGetRequest{
// 			SpaceId: storeKey.SpaceId,
// 			Cid:     b.Cid().Bytes(),
// 			Wait:    true,
// 		})

// 		require.NoError(t, err)
// 		require.NotNil(t, resp)
// 		require.Equal(t, b.RawData(), resp.Data)
// 		require.Equal(t, 3, attempts)
// 	})

// 	t.Run("wait timeout", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		var (
// 			ctx, storeKey = newRandKey()
// 			b             = testutil.NewRandBlock(1024)
// 		)

// 		fx.storeService.GetBlockFunc = func(txn *badger.Txn, k cid.Cid) (*lightfilenodestore.BlockObj, error) {
// 			return nil, fileprotoerr.ErrCIDNotFound
// 		}

// 		ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
// 		defer cancel()

// 		resp, err := fx.rpcService.BlockGet(ctx, &fileproto.BlockGetRequest{
// 			SpaceId: storeKey.SpaceId,
// 			Cid:     b.Cid().Bytes(),
// 			Wait:    true,
// 		})

// 		require.Error(t, err)
// 		require.Contains(t, err.Error(), "context deadline exceeded")
// 		require.Nil(t, resp)
// 	})

// 	t.Run("invalid cid", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		ctx, storeKey := newRandKey()

// 		resp, err := fx.rpcService.BlockGet(ctx, &fileproto.BlockGetRequest{
// 			SpaceId: storeKey.SpaceId,
// 			Cid:     []byte("invalid"),
// 			Wait:    true,
// 		})

// 		require.Error(t, err)
// 		require.Contains(t, err.Error(), "failed to cast CID")
// 		require.Nil(t, resp)
// 	})
// }

// func TestLightFileNodeRpc_BlockPush(t *testing.T) {
// 	t.Run("success", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		var (
// 			ctx, storeKey = newRandKey()
// 			fileId        = testutil.NewRandCid().String()
// 			b             = testutil.NewRandBlock(1024)
// 		)

// 		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

// 		fx.storeService.HadCIDFunc = func(txn *badger.Txn, k cid.Cid) (bool, error) {
// 			require.Equal(t, b.Cid(), k)
// 			return false, nil
// 		}

// 		fx.storeService.PushBlockFunc = func(txn *badger.Txn, spaceId string, blk blocks.Block) error {
// 			require.Equal(t, storeKey.SpaceId, spaceId)
// 			require.Equal(t, b.Cid(), blk.Cid())
// 			require.Equal(t, b.RawData(), blk.RawData())
// 			return nil
// 		}

// 		fx.storeService.HasLinkFileBlockFunc = func(txn *badger.Txn, spaceId, fileId string, k cid.Cid) (bool, error) {
// 			require.Equal(t, storeKey.SpaceId, spaceId)
// 			require.Equal(t, fileId, fileId)
// 			require.Equal(t, b.Cid(), k)
// 			return false, nil
// 		}

// 		fx.storeService.GetFileFunc = func(txn *badger.Txn, spaceId, fileId string) (*lightfilenodestore.FileObj, error) {
// 			require.Equal(t, storeKey.SpaceId, spaceId)
// 			require.Equal(t, fileId, fileId)
// 			return lightfilenodestore.NewFileObj(spaceId, fileId), nil
// 		}

// 		fx.storeService.CreateLinkFileBlockFunc = func(txn *badger.Txn, spaceId, fileId string, k cid.Cid) error {
// 			require.Equal(t, storeKey.SpaceId, spaceId)
// 			require.Equal(t, fileId, fileId)
// 			require.Equal(t, b.Cid(), k)
// 			return nil
// 		}

// 		fx.storeService.WriteFileFunc = func(txn *badger.Txn, f *lightfilenodestore.FileObj) error {
// 			require.NotNil(t, f)
// 			require.Equal(t, storeKey.SpaceId, f.SpaceID())
// 			require.Equal(t, fileId, f.FileID())
// 			require.Equal(t, uint64(1024), f.UsageBytes()) // Block size
// 			require.Equal(t, uint32(1), f.CidsCount())     // One CID added
// 			return nil
// 		}

// 		fx.storeService.GetSpaceFunc = func(txn *badger.Txn, spaceId string) (*lightfilenodestore.SpaceObj, error) {
// 			require.Equal(t, storeKey.SpaceId, spaceId)
// 			spaceObj := lightfilenodestore.NewSpaceObj(storeKey.SpaceId)
// 			return spaceObj, nil
// 		}

// 		fx.storeService.WriteSpaceFunc = func(txn *badger.Txn, spaceObj *lightfilenodestore.SpaceObj) error {
// 			require.NotNil(t, spaceObj)
// 			require.Equal(t, storeKey.SpaceId, spaceObj.SpaceID())
// 			require.Equal(t, uint64(1024), spaceObj.SpaceUsageBytes()) // Block size
// 			require.Equal(t, uint64(1), spaceObj.CidsCount())          // One CID added
// 			return nil
// 		}

// 		fx.storeService.GetGroupFunc = func(txn *badger.Txn, groupId string) (*lightfilenodestore.GroupObj, error) {
// 			require.Equal(t, storeKey.GroupId, groupId)
// 			groupObj := lightfilenodestore.NewGroupObj(storeKey.GroupId).
// 				WithLimitBytes(1 << 30) // 1GB - default limit for group
// 			return groupObj, nil
// 		}

// 		fx.storeService.WriteGroupFunc = func(txn *badger.Txn, groupObj *lightfilenodestore.GroupObj) error {
// 			require.NotNil(t, groupObj)
// 			require.Equal(t, storeKey.GroupId, groupObj.GroupID())
// 			require.Equal(t, uint64(1024), groupObj.TotalUsageBytes()) // Block size
// 			return nil
// 		}

// 		resp, err := fx.rpcService.BlockPush(ctx, &fileproto.BlockPushRequest{
// 			SpaceId: storeKey.SpaceId,
// 			FileId:  fileId,
// 			Cid:     b.Cid().Bytes(),
// 			Data:    b.RawData(),
// 		})

// 		require.NoError(t, err)
// 		require.NotNil(t, resp)
// 	})

// 	t.Run("invalid cid", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		var (
// 			ctx, storeKey = newRandKey()
// 			fileId        = testutil.NewRandCid().String()
// 			b             = testutil.NewRandBlock(1024)
// 		)

// 		fx.storeService.GetSpaceFunc = func(txn *badger.Txn, spaceId string) (*lightfilenodestore.SpaceObj, error) {
// 			spaceObj := lightfilenodestore.NewSpaceObj(storeKey.SpaceId)
// 			return spaceObj, nil
// 		}

// 		fx.storeService.GetGroupFunc = func(txn *badger.Txn, groupId string) (*lightfilenodestore.GroupObj, error) {
// 			groupObj := lightfilenodestore.NewGroupObj(storeKey.GroupId).
// 				WithLimitBytes(1 << 30) // 1GB - default limit for group

// 			return groupObj, nil
// 		}

// 		resp, err := fx.rpcService.BlockPush(ctx, &fileproto.BlockPushRequest{
// 			SpaceId: storeKey.SpaceId,
// 			FileId:  fileId,
// 			Cid:     []byte("invalid"),
// 			Data:    b.RawData(),
// 		})

// 		require.Error(t, err)
// 		require.Contains(t, err.Error(), "failed to cast CID")
// 		require.Nil(t, resp)
// 	})

// 	t.Run("invalid cid checksum to data", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		var (
// 			ctx, storeKey = newRandKey()
// 			fileId        = testutil.NewRandCid().String()
// 			b             = testutil.NewRandBlock(1024)
// 			b2            = testutil.NewRandBlock(10)
// 		)

// 		fx.storeService.GetSpaceFunc = func(txn *badger.Txn, spaceId string) (*lightfilenodestore.SpaceObj, error) {
// 			spaceObj := lightfilenodestore.NewSpaceObj(storeKey.SpaceId)
// 			return spaceObj, nil
// 		}

// 		fx.storeService.GetGroupFunc = func(txn *badger.Txn, groupId string) (*lightfilenodestore.GroupObj, error) {
// 			groupObj := lightfilenodestore.NewGroupObj(storeKey.GroupId).
// 				WithLimitBytes(1 << 30) // 1GB - default limit for group

// 			return groupObj, nil
// 		}

// 		resp, err := fx.rpcService.BlockPush(ctx, &fileproto.BlockPushRequest{
// 			SpaceId: storeKey.SpaceId,
// 			FileId:  fileId,
// 			Cid:     b2.Cid().Bytes(),
// 			Data:    b.RawData(),
// 		})

// 		require.Error(t, err)
// 		require.Contains(t, err.Error(), "block data checksum mismatch")
// 		require.Nil(t, resp)
// 	})

// 	t.Run("data too big", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)

// 		var (
// 			ctx, key = newRandKey()
// 			spaceId  = key.SpaceId
// 			fileId   = testutil.NewRandCid().String()
// 			b        = testutil.NewRandBlock(3 << 20)
// 		)

// 		fx.storeService.GetSpaceFunc = func(txn *badger.Txn, spaceId string) (*lightfilenodestore.SpaceObj, error) {
// 			spaceObj := lightfilenodestore.NewSpaceObj(spaceId)
// 			return spaceObj, nil
// 		}

// 		fx.storeService.GetGroupFunc = func(txn *badger.Txn, groupId string) (*lightfilenodestore.GroupObj, error) {
// 			groupObj := lightfilenodestore.NewGroupObj(key.GroupId).
// 				WithLimitBytes(1 << 20) // 1MB - limit for group

// 			return groupObj, nil
// 		}

// 		resp, err := fx.rpcService.BlockPush(ctx, &fileproto.BlockPushRequest{
// 			SpaceId: spaceId,
// 			FileId:  fileId,
// 			Cid:     b.Cid().Bytes(),
// 			Data:    b.RawData(),
// 		})
// 		require.EqualError(t, err, fileprotoerr.ErrQuerySizeExceeded.Error())
// 		require.Nil(t, resp)
// 	})

// 	t.Run("store error", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		var (
// 			ctx, storeKey = newRandKey()
// 			fileId        = testutil.NewRandCid().String()
// 			b             = testutil.NewRandBlock(1024)
// 		)

// 		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

// 		fx.storeService.HadCIDFunc = func(txn *badger.Txn, k cid.Cid) (bool, error) {
// 			return false, nil
// 		}

// 		fx.storeService.GetSpaceFunc = func(txn *badger.Txn, spaceId string) (*lightfilenodestore.SpaceObj, error) {
// 			spaceObj := lightfilenodestore.NewSpaceObj(storeKey.SpaceId)
// 			return spaceObj, nil
// 		}

// 		fx.storeService.GetGroupFunc = func(txn *badger.Txn, groupId string) (*lightfilenodestore.GroupObj, error) {
// 			groupObj := lightfilenodestore.NewGroupObj(storeKey.GroupId).
// 				WithLimitBytes(1 << 30) // 1GB - default limit for group

// 			return groupObj, nil
// 		}

// 		fx.storeService.PushBlockFunc = func(txn *badger.Txn, spaceId string, b blocks.Block) error {
// 			return fmt.Errorf("store error")
// 		}

// 		resp, err := fx.rpcService.BlockPush(ctx, &fileproto.BlockPushRequest{
// 			SpaceId: storeKey.SpaceId,
// 			FileId:  fileId,
// 			Cid:     b.Cid().Bytes(),
// 			Data:    b.RawData(),
// 		})

// 		require.Error(t, err)
// 		require.Contains(t, err.Error(), "failed to push block: store error")
// 		require.Nil(t, resp)
// 	})
// }

// func TestLightFileNodeRpc_BlocksCheck(t *testing.T) {
// 	t.Run("success", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		var (
// 			ctx, storeKey = newRandKey()
// 			bs            = testutil.NewRandBlocks(3)
// 		)

// 		fx.storeService.HasCIDInSpaceFunc = func(txn *badger.Txn, spaceId string, k cid.Cid) (bool, error) {
// 			require.Equal(t, storeKey.SpaceId, spaceId)
// 			if k.Equals(bs[0].Cid()) {
// 				return true, nil
// 			}
// 			return false, nil
// 		}

// 		fx.storeService.HadCIDFunc = func(txn *badger.Txn, k cid.Cid) (bool, error) {
// 			if k.Equals(bs[1].Cid()) {
// 				return true, nil
// 			}
// 			return false, nil
// 		}

// 		cids := make([][]byte, len(bs))
// 		for i, b := range bs {
// 			cids[i] = b.Cid().Bytes()
// 		}

// 		resp, err := fx.rpcService.BlocksCheck(ctx, &fileproto.BlocksCheckRequest{
// 			SpaceId: storeKey.SpaceId,
// 			Cids:    cids,
// 		})

// 		require.NoError(t, err)
// 		require.NotNil(t, resp)
// 		require.Len(t, resp.BlocksAvailability, len(bs))

// 		// First block exists in space
// 		require.Equal(t, fileproto.AvailabilityStatus_ExistsInSpace, resp.BlocksAvailability[0].Status)
// 		require.Equal(t, bs[0].Cid().Bytes(), resp.BlocksAvailability[0].Cid)

// 		// Second block exists but not in space
// 		require.Equal(t, fileproto.AvailabilityStatus_Exists, resp.BlocksAvailability[1].Status)
// 		require.Equal(t, bs[1].Cid().Bytes(), resp.BlocksAvailability[1].Cid)

// 		// Third block does not exist
// 		require.Equal(t, fileproto.AvailabilityStatus_NotExists, resp.BlocksAvailability[2].Status)
// 		require.Equal(t, bs[2].Cid().Bytes(), resp.BlocksAvailability[2].Cid)
// 	})

// 	t.Run("invalid cid", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		ctx, storeKey := newRandKey()

// 		resp, err := fx.rpcService.BlocksCheck(ctx, &fileproto.BlocksCheckRequest{
// 			SpaceId: storeKey.SpaceId,
// 			Cids:    [][]byte{[]byte("invalid")},
// 		})

// 		require.Error(t, err)
// 		require.Contains(t, err.Error(), "failed to cast CID")
// 		require.Nil(t, resp)
// 	})

// 	t.Run("store error in space", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		var (
// 			ctx, storeKey = newRandKey()
// 			b             = testutil.NewRandBlock(1024)
// 		)

// 		storeErr := errors.New("HasCIDInSpaceFunc: store error")
// 		fx.storeService.HasCIDInSpaceFunc = func(txn *badger.Txn, spaceId string, k cid.Cid) (bool, error) {
// 			return false, storeErr
// 		}

// 		fx.storeService.HadCIDFunc = func(txn *badger.Txn, k cid.Cid) (bool, error) {
// 			return false, nil
// 		}

// 		resp, err := fx.rpcService.BlocksCheck(ctx, &fileproto.BlocksCheckRequest{
// 			SpaceId: storeKey.SpaceId,
// 			Cids:    [][]byte{b.Cid().Bytes()},
// 		})

// 		require.Error(t, err)
// 		require.ErrorIs(t, err, storeErr)
// 		require.Nil(t, resp)
// 	})

// 	t.Run("store error not in space", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		var (
// 			ctx, storeKey = newRandKey()
// 			b             = testutil.NewRandBlock(1024)
// 		)

// 		fx.storeService.HasCIDInSpaceFunc = func(txn *badger.Txn, spaceId string, k cid.Cid) (bool, error) {
// 			return false, nil
// 		}

// 		storeErr := errors.New("HadCIDFunc: store error")
// 		fx.storeService.HadCIDFunc = func(txn *badger.Txn, k cid.Cid) (bool, error) {
// 			return false, storeErr
// 		}

// 		resp, err := fx.rpcService.BlocksCheck(ctx, &fileproto.BlocksCheckRequest{
// 			SpaceId: storeKey.SpaceId,
// 			Cids:    [][]byte{b.Cid().Bytes()},
// 		})

// 		require.Error(t, err)
// 		require.ErrorIs(t, err, storeErr)
// 		require.Nil(t, resp)
// 	})

// 	t.Run("skip duplicate cids", func(t *testing.T) {
// 		fx := newFixture(t)
// 		defer fx.Finish(t)
// 		ctx, storeKey := newRandKey()
// 		b := testutil.NewRandBlock(1024)

// 		// Mock store service to return exists for the block
// 		fx.storeService.HasCIDInSpaceFunc = func(txn *badger.Txn, spaceId string, k cid.Cid) (bool, error) {
// 			return true, nil
// 		}

// 		fx.storeService.HadCIDFunc = func(txn *badger.Txn, k cid.Cid) (bool, error) {
// 			return true, nil
// 		}

// 		resp, err := fx.rpcService.BlocksCheck(ctx, &fileproto.BlocksCheckRequest{
// 			SpaceId: storeKey.SpaceId,
// 			// Send same CID twice
// 			Cids: [][]byte{b.Cid().Bytes(), b.Cid().Bytes()},
// 		})

// 		require.NoError(t, err)
// 		require.NotNil(t, resp)
// 		require.Len(t, resp.BlocksAvailability, 1) // Should only return one result
// 		require.Equal(t, fileproto.AvailabilityStatus_ExistsInSpace, resp.BlocksAvailability[0].Status)
// 		require.Equal(t, b.Cid().Bytes(), resp.BlocksAvailability[0].Cid)
// 	})
// }

func newFixture(t *testing.T) *fixture {
	ctrl := gomock.NewController(t)

	fx := &fixture{
		ctrl:    ctrl,
		a:       new(app.App),
		drpcSrv: server.New(),
		aclSrv:  mock_acl.NewMockAclService(ctrl),
		rpcSrv:  New().(*lightfilenoderpc),
		storeSrv: &lightfilenodestore.StoreServiceMock{
			InitFunc: func(a *app.App) error {
				return nil
			},
			NameFunc: func() string {
				return lightfilenodestore.CName
			},
			RunFunc: func(ctx context.Context) error {
				return nil
			},
			CloseFunc: func(ctx context.Context) error {
				return nil
			},
			// Default for tests
			TxViewFunc: func(f func(txn *badger.Txn) error) error {
				return f(nil)
			},
			TxUpdateFunc: func(f func(txn *badger.Txn) error) error {
				return f(nil)
			},
		},
		indexSrv: &lightfilenodeindex.IndexServiceMock{
			InitFunc: func(a *app.App) error {
				return nil
			},
			NameFunc: func() string {
				return lightfilenodeindex.CName
			},
			RunFunc: func(ctx context.Context) error {
				return nil
			},
			CloseFunc: func(ctx context.Context) error {
				return nil
			},
		},
	}

	fx.aclSrv.EXPECT().Name().Return(acl.CName).AnyTimes()
	fx.aclSrv.EXPECT().Init(gomock.Any()).AnyTimes()
	fx.aclSrv.EXPECT().Run(gomock.Any()).AnyTimes()
	fx.aclSrv.EXPECT().Close(gomock.Any()).AnyTimes()

	fx.a.Register(&lightconfig.LightConfig{}).
		Register(fx.drpcSrv).
		Register(fx.aclSrv).
		Register(fx.storeSrv).
		Register(fx.indexSrv).
		Register(fx.rpcSrv)

	require.NoError(t, fx.a.Start(context.Background()))
	return fx
}

type fixture struct {
	ctrl *gomock.Controller
	a    *app.App

	drpcSrv  server.DRPCServer
	aclSrv   *mock_acl.MockAclService
	rpcSrv   *lightfilenoderpc
	storeSrv *lightfilenodestore.StoreServiceMock
	indexSrv *lightfilenodeindex.IndexServiceMock
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

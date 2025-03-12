package lightfilenoderpc

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/testutil"
	"github.com/anyproto/any-sync/commonfile/fileproto"
	"github.com/anyproto/any-sync/commonfile/fileproto/fileprotoerr"
	"github.com/anyproto/any-sync/commonspace/object/acl/list"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/util/crypto"
	"github.com/stretchr/testify/require"
)

func TestLightFileNodeRpc_resolveStoreKey(t *testing.T) {
	t.Run("empty space id", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		key, err := fx.rpcSrv.resolveStoreKey(context.Background(), "")
		require.ErrorIs(t, err, fileprotoerr.ErrForbidden)
		require.Empty(t, key)
	})

	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		ctx, storeKey := newRandKey()
		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		key, err := fx.rpcSrv.resolveStoreKey(ctx, storeKey.SpaceId)
		require.NoError(t, err)
		require.Equal(t, storeKey, key)
	})
}

func TestLightFileNodeRpc_canRead(t *testing.T) {
	t.Run("owner can read", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		ctx, storeKey := newRandKey()
		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		key, err := fx.rpcSrv.canRead(ctx, storeKey.SpaceId)
		require.NoError(t, err)
		require.Equal(t, storeKey, key)
	})

	t.Run("reader can read", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		_, pubKey, err := crypto.GenerateEd25519Key(rand.Reader)
		require.NoError(t, err)
		protoPub, err := pubKey.Marshall()
		require.NoError(t, err)
		ctx := peer.CtxWithIdentity(context.Background(), protoPub)
		spaceId := testutil.NewRandSpaceId()

		// Get the actual public key from context that will be used in canRead
		ctxPubKey, err := peer.CtxPubKey(ctx)
		require.NoError(t, err)

		_, ownerPubKey, err := crypto.GenerateRandomEd25519KeyPair()
		require.NoError(t, err)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, spaceId).Return(ownerPubKey, nil)
		fx.aclSrv.EXPECT().Permissions(ctx, ctxPubKey, spaceId).Return(list.AclPermissionsReader, nil)

		key, err := fx.rpcSrv.canRead(ctx, spaceId)
		require.NoError(t, err)
		require.Equal(t, ownerPubKey.Account(), key.GroupId)
		require.Equal(t, spaceId, key.SpaceId)
	})

	t.Run("no permission", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		_, pubKey, err := crypto.GenerateEd25519Key(rand.Reader)
		require.NoError(t, err)
		protoPub, err := pubKey.Marshall()
		require.NoError(t, err)
		ctx := peer.CtxWithIdentity(context.Background(), protoPub)
		spaceId := testutil.NewRandSpaceId()

		// Get the actual public key from context that will be used in canRead
		ctxPubKey, err := peer.CtxPubKey(ctx)
		require.NoError(t, err)

		_, ownerPubKey, err := crypto.GenerateRandomEd25519KeyPair()
		require.NoError(t, err)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, spaceId).Return(ownerPubKey, nil)
		fx.aclSrv.EXPECT().Permissions(ctx, ctxPubKey, spaceId).Return(list.AclPermissionsNone, nil)

		_, err = fx.rpcSrv.canRead(ctx, spaceId)
		require.ErrorIs(t, err, fileprotoerr.ErrForbidden)
	})
}

func TestLightFileNodeRpc_canWrite(t *testing.T) {
	t.Run("owner can write", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		ctx, storeKey := newRandKey()
		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				LimitBytes: 1 << 30,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		key, err := fx.rpcSrv.canWrite(ctx, storeKey.SpaceId)
		require.NoError(t, err)
		require.Equal(t, storeKey, key)
	})

	t.Run("writer can write", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		_, pubKey, err := crypto.GenerateEd25519Key(rand.Reader)
		require.NoError(t, err)
		protoPub, err := pubKey.Marshall()
		require.NoError(t, err)
		ctx := peer.CtxWithIdentity(context.Background(), protoPub)
		spaceId := testutil.NewRandSpaceId()

		// Get the actual public key from context that will be used in canRead
		ctxPubKey, err := peer.CtxPubKey(ctx)
		require.NoError(t, err)

		_, ownerPubKey, err := crypto.GenerateRandomEd25519KeyPair()
		require.NoError(t, err)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, spaceId).Return(ownerPubKey, nil)
		fx.aclSrv.EXPECT().Permissions(ctx, ctxPubKey, spaceId).Return(list.AclPermissionsWriter, nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				LimitBytes: 1 << 30,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		key, err := fx.rpcSrv.canWrite(ctx, spaceId)
		require.NoError(t, err)
		require.Equal(t, ownerPubKey.Account(), key.GroupId)
		require.Equal(t, spaceId, key.SpaceId)
	})

	t.Run("reader cannot write", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		_, pubKey, err := crypto.GenerateEd25519Key(rand.Reader)
		require.NoError(t, err)
		protoPub, err := pubKey.Marshall()
		require.NoError(t, err)
		ctx := peer.CtxWithIdentity(context.Background(), protoPub)
		spaceId := testutil.NewRandSpaceId()

		// Get the actual public key from context that will be used in canRead
		ctxPubKey, err := peer.CtxPubKey(ctx)
		require.NoError(t, err)

		_, ownerPubKey, err := crypto.GenerateRandomEd25519KeyPair()
		require.NoError(t, err)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, spaceId).Return(ownerPubKey, nil)
		fx.aclSrv.EXPECT().Permissions(ctx, ctxPubKey, spaceId).Return(list.AclPermissionsReader, nil)

		_, err = fx.rpcSrv.canWrite(ctx, spaceId)
		require.ErrorIs(t, err, fileprotoerr.ErrForbidden)
	})
}

func TestLightFileNodeRpc_hasEnoughSpace(t *testing.T) {
	t.Run("space limit exceeded", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		storeKey := index.Key{
			GroupId: testutil.NewRandSpaceId(),
			SpaceId: testutil.NewRandSpaceId(),
		}

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				TotalUsageBytes: 1025,
				LimitBytes:      2048,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{
				TotalUsageBytes: 0,
				LimitBytes:      1024,
			}
		}

		err := fx.rpcSrv.hasEnoughSpace(storeKey)
		require.ErrorIs(t, err, fileprotoerr.ErrSpaceLimitExceeded)
	})

	t.Run("group limit exceeded", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		storeKey := index.Key{
			GroupId: testutil.NewRandSpaceId(),
			SpaceId: testutil.NewRandSpaceId(),
		}

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				TotalUsageBytes: 1025,
				LimitBytes:      1024,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		err := fx.rpcSrv.hasEnoughSpace(storeKey)
		require.ErrorIs(t, err, fileprotoerr.ErrSpaceLimitExceeded)
	})

	t.Run("has enough space", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		storeKey := index.Key{
			GroupId: testutil.NewRandSpaceId(),
			SpaceId: testutil.NewRandSpaceId(),
		}

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				TotalUsageBytes: 1024,
				LimitBytes:      4096,
			}
		}
		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{
				LimitBytes: 2048,
			}
		}

		err := fx.rpcSrv.hasEnoughSpace(storeKey)
		require.NoError(t, err)
	})
}

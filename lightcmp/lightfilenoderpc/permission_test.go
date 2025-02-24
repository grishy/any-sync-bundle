package lightfilenoderpc

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/testutil"
	"github.com/anyproto/any-sync/commonfile/fileproto/fileprotoerr"
	"github.com/anyproto/any-sync/commonspace/object/acl/list"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/util/crypto"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"

	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

func TestLightFileNodeRpc_resolveStoreKey(t *testing.T) {
	t.Run("empty space id", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		key, err := fx.rpcService.resolveStoreKey(context.Background(), "")
		require.ErrorIs(t, err, fileprotoerr.ErrForbidden)
		require.Empty(t, key)
	})

	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		ctx, storeKey := newRandKey()
		fx.aclService.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		key, err := fx.rpcService.resolveStoreKey(ctx, storeKey.SpaceId)
		require.NoError(t, err)
		require.Equal(t, storeKey, key)
	})
}

func TestLightFileNodeRpc_canRead(t *testing.T) {
	t.Run("owner can read", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		ctx, storeKey := newRandKey()
		fx.aclService.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		key, err := fx.rpcService.canRead(ctx, storeKey.SpaceId)
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

		fx.aclService.EXPECT().OwnerPubKey(ctx, spaceId).Return(ownerPubKey, nil)
		fx.aclService.EXPECT().Permissions(ctx, ctxPubKey, spaceId).Return(list.AclPermissionsReader, nil)

		key, err := fx.rpcService.canRead(ctx, spaceId)
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

		fx.aclService.EXPECT().OwnerPubKey(ctx, spaceId).Return(ownerPubKey, nil)
		fx.aclService.EXPECT().Permissions(ctx, ctxPubKey, spaceId).Return(list.AclPermissionsNone, nil)

		_, err = fx.rpcService.canRead(ctx, spaceId)
		require.ErrorIs(t, err, fileprotoerr.ErrForbidden)
	})
}

func TestLightFileNodeRpc_canWrite(t *testing.T) {
	t.Run("owner can write", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		ctx, storeKey := newRandKey()
		fx.aclService.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.storeService.GetSpaceFunc = func(txn *badger.Txn, spaceId string) (*lightfilenodestore.SpaceObj, error) {
			return lightfilenodestore.NewSpaceObj(spaceId), nil
		}

		fx.storeService.GetGroupFunc = func(txn *badger.Txn, groupId string) (*lightfilenodestore.GroupObj, error) {
			return lightfilenodestore.NewGroupObj(groupId).WithLimitBytes(1 << 30), nil
		}

		key, err := fx.rpcService.canWrite(ctx, nil, storeKey.SpaceId)
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

		fx.aclService.EXPECT().OwnerPubKey(ctx, spaceId).Return(ownerPubKey, nil)
		fx.aclService.EXPECT().Permissions(ctx, ctxPubKey, spaceId).Return(list.AclPermissionsWriter, nil)

		fx.storeService.GetSpaceFunc = func(txn *badger.Txn, spaceId string) (*lightfilenodestore.SpaceObj, error) {
			return lightfilenodestore.NewSpaceObj(spaceId), nil
		}

		fx.storeService.GetGroupFunc = func(txn *badger.Txn, groupId string) (*lightfilenodestore.GroupObj, error) {
			return lightfilenodestore.NewGroupObj(groupId).WithLimitBytes(1 << 30), nil
		}

		key, err := fx.rpcService.canWrite(ctx, nil, spaceId)
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

		fx.aclService.EXPECT().OwnerPubKey(ctx, spaceId).Return(ownerPubKey, nil)
		fx.aclService.EXPECT().Permissions(ctx, ctxPubKey, spaceId).Return(list.AclPermissionsReader, nil)

		_, err = fx.rpcService.canWrite(ctx, nil, spaceId)
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

		fx.storeService.GetSpaceFunc = func(txn *badger.Txn, spaceId string) (*lightfilenodestore.SpaceObj, error) {
			return lightfilenodestore.NewSpaceObj(spaceId).WithLimitBytes(1024), nil
		}

		fx.storeService.GetGroupFunc = func(txn *badger.Txn, groupId string) (*lightfilenodestore.GroupObj, error) {
			return lightfilenodestore.NewGroupObj(groupId).
				WithLimitBytes(2048).
				WithTotalUsageBytes(1025), nil
		}

		err := fx.rpcService.hasEnoughSpace(nil, &storeKey)
		require.ErrorIs(t, err, fileprotoerr.ErrSpaceLimitExceeded)
	})

	t.Run("group limit exceeded", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		storeKey := index.Key{
			GroupId: testutil.NewRandSpaceId(),
			SpaceId: testutil.NewRandSpaceId(),
		}

		fx.storeService.GetSpaceFunc = func(txn *badger.Txn, spaceId string) (*lightfilenodestore.SpaceObj, error) {
			return lightfilenodestore.NewSpaceObj(spaceId), nil // No space limit
		}

		fx.storeService.GetGroupFunc = func(txn *badger.Txn, groupId string) (*lightfilenodestore.GroupObj, error) {
			return lightfilenodestore.NewGroupObj(groupId).
				WithLimitBytes(1024).
				WithTotalUsageBytes(1025), nil
		}

		err := fx.rpcService.hasEnoughSpace(nil, &storeKey)
		require.ErrorIs(t, err, fileprotoerr.ErrSpaceLimitExceeded)
	})

	t.Run("has enough space", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		storeKey := index.Key{
			GroupId: testutil.NewRandSpaceId(),
			SpaceId: testutil.NewRandSpaceId(),
		}

		fx.storeService.GetSpaceFunc = func(txn *badger.Txn, spaceId string) (*lightfilenodestore.SpaceObj, error) {
			return lightfilenodestore.NewSpaceObj(spaceId).WithLimitBytes(2048), nil
		}

		fx.storeService.GetGroupFunc = func(txn *badger.Txn, groupId string) (*lightfilenodestore.GroupObj, error) {
			return lightfilenodestore.NewGroupObj(groupId).
				WithLimitBytes(4096).
				WithTotalUsageBytes(1024), nil
		}

		err := fx.rpcService.hasEnoughSpace(nil, &storeKey)
		require.NoError(t, err)
	})
}

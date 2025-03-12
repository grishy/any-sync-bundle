package lightfilenoderpc

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/testutil"
	"github.com/anyproto/any-sync/acl"
	"github.com/anyproto/any-sync/acl/mock_acl"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/commonfile/fileproto"
	"github.com/anyproto/any-sync/commonfile/fileproto/fileprotoerr"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/util/cidutil"
	"github.com/anyproto/any-sync/util/crypto"
	"github.com/dgraph-io/badger/v4"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"storj.io/drpc"

	"github.com/grishy/any-sync-bundle/lightcmp/lightconfig"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodeindex/indexpb"
	"github.com/grishy/any-sync-bundle/lightcmp/lightfilenodestore"
)

// TODO: Write test with real database

func TestLightFileNodeRpc_BlockGet(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			b             = testutil.NewRandBlock(1024)
		)

		fx.storeSrv.GetBlockFunc = func(txn *badger.Txn, k cid.Cid) ([]byte, error) {
			return nil, lightfilenodestore.ErrBlockNotFound
		}

		resp, err := fx.rpcSrv.BlockGet(ctx, &fileproto.BlockGetRequest{
			SpaceId: storeKey.SpaceId,
			Cid:     b.Cid().Bytes(),
			Wait:    false,
		})

		require.Error(t, err)
		require.ErrorIs(t, err, fileprotoerr.ErrCIDNotFound)
		require.Nil(t, resp)
	})

	t.Run("found", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			b             = testutil.NewRandBlock(1024)
		)

		fx.storeSrv.GetBlockFunc = func(txn *badger.Txn, k cid.Cid) ([]byte, error) {
			return b.RawData(), nil
		}

		resp, err := fx.rpcSrv.BlockGet(ctx, &fileproto.BlockGetRequest{
			SpaceId: storeKey.SpaceId,
			Cid:     b.Cid().Bytes(),
			Wait:    false,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, b.RawData(), resp.Data)
	})

	t.Run("wait", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			b             = testutil.NewRandBlock(1024)
			attempts      = 0
		)

		fx.storeSrv.GetBlockFunc = func(txn *badger.Txn, k cid.Cid) ([]byte, error) {
			t.Logf("GetBlockFunc: %d", attempts)
			attempts++
			if attempts < 3 {
				return nil, lightfilenodestore.ErrBlockNotFound
			}

			return b.RawData(), nil
		}

		resp, err := fx.rpcSrv.BlockGet(ctx, &fileproto.BlockGetRequest{
			SpaceId: storeKey.SpaceId,
			Cid:     b.Cid().Bytes(),
			Wait:    true,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, b.RawData(), resp.Data)
		require.Equal(t, 3, attempts)
	})

	t.Run("wait timeout", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			b             = testutil.NewRandBlock(1024)
		)

		fx.storeSrv.GetBlockFunc = func(txn *badger.Txn, k cid.Cid) ([]byte, error) {
			return nil, lightfilenodestore.ErrBlockNotFound
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		defer cancel()

		resp, err := fx.rpcSrv.BlockGet(timeoutCtx, &fileproto.BlockGetRequest{
			SpaceId: storeKey.SpaceId,
			Cid:     b.Cid().Bytes(),
			Wait:    true,
		})

		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Nil(t, resp)
	})

	t.Run("invalid cid", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx, storeKey := newRandKey()

		resp, err := fx.rpcSrv.BlockGet(ctx, &fileproto.BlockGetRequest{
			SpaceId: storeKey.SpaceId,
			Cid:     []byte("invalid"),
			Wait:    true,
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to cast CID")
		require.Nil(t, resp)
	})

	t.Run("unknown error", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			b             = testutil.NewRandBlock(1024)
		)

		errUnknown := errors.New("unknown error")
		fx.storeSrv.GetBlockFunc = func(txn *badger.Txn, k cid.Cid) ([]byte, error) {
			return nil, errUnknown
		}

		resp, err := fx.rpcSrv.BlockGet(ctx, &fileproto.BlockGetRequest{
			SpaceId: storeKey.SpaceId,
			Cid:     b.Cid().Bytes(),
			Wait:    false,
		})

		require.Error(t, err)
		require.ErrorIs(t, err, errUnknown)
		require.Nil(t, resp)
	})
}

func TestLightFileNodeRpc_BlockPush(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileId        = testutil.NewRandCid().String()
			b             = testutil.NewRandBlock(1024)
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				LimitBytes: 1 << 30,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		fx.indexSrv.HadCIDFunc = func(k cid.Cid) bool {
			require.Equal(t, b.Cid(), k)
			return false
		}

		fx.storeSrv.PutBlockFunc = func(txn *badger.Txn, blk blocks.Block) error {
			require.Equal(t, b.Cid(), blk.Cid())
			require.Equal(t, b.RawData(), blk.RawData())
			return nil
		}

		capturedOps := make([]*indexpb.Operation, 0, 2)

		fx.indexSrv.ModifyFunc = func(txn *badger.Txn, key index.Key, operations ...*indexpb.Operation) error {
			require.Equal(t, storeKey, key)

			capturedOps = append(capturedOps, operations...)
			return nil
		}

		resp, err := fx.rpcSrv.BlockPush(ctx, &fileproto.BlockPushRequest{
			SpaceId: storeKey.SpaceId,
			FileId:  fileId,
			Cid:     b.Cid().Bytes(),
			Data:    b.RawData(),
		})

		require.NoError(t, err)
		require.NotNil(t, resp)

		require.Equal(t, 2, len(capturedOps), "Should have two operations: CidAdd and FileBindOperation")

		cidAddOp := capturedOps[0].GetCidAdd()
		require.NotNil(t, cidAddOp)
		require.Equal(t, b.Cid().String(), cidAddOp.GetCid())
		require.Equal(t, int64(len(b.RawData())), cidAddOp.GetDataSize())

		bindOp := capturedOps[1].GetBindFile()
		require.NotNil(t, bindOp)
		require.Equal(t, fileId, bindOp.GetFileId())
		require.Equal(t, 1, len(bindOp.GetCids()))
		require.Equal(t, b.Cid().String(), bindOp.GetCids()[0])
	})

	t.Run("invalid cid", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileId        = testutil.NewRandCid().String()
			b             = testutil.NewRandBlock(1024)
		)

		resp, err := fx.rpcSrv.BlockPush(ctx, &fileproto.BlockPushRequest{
			SpaceId: storeKey.SpaceId,
			FileId:  fileId,
			Cid:     []byte("invalid"),
			Data:    b.RawData(),
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to cast CID")
		require.Nil(t, resp)
	})

	t.Run("invalid cid checksum to data", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileId        = testutil.NewRandCid().String()
			b1            = testutil.NewRandBlock(1024)
			b2            = testutil.NewRandBlock(10)
		)

		resp, err := fx.rpcSrv.BlockPush(ctx, &fileproto.BlockPushRequest{
			SpaceId: storeKey.SpaceId,
			FileId:  fileId,
			Cid:     b2.Cid().Bytes(),
			Data:    b1.RawData(),
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "block data checksum mismatch")
		require.Nil(t, resp)
	})

	t.Run("data too big", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		var (
			ctx, key = newRandKey()
			spaceId  = key.SpaceId
			fileId   = testutil.NewRandCid().String()
			b        = testutil.NewRandBlock(3 << 20) // 3 MiB, larger than the 2 MiB limit
		)

		resp, err := fx.rpcSrv.BlockPush(ctx, &fileproto.BlockPushRequest{
			SpaceId: spaceId,
			FileId:  fileId,
			Cid:     b.Cid().Bytes(),
			Data:    b.RawData(),
		})
		require.EqualError(t, err, fileprotoerr.ErrQuerySizeExceeded.Error())
		require.Nil(t, resp)
	})

	t.Run("store error", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileId        = testutil.NewRandCid().String()
			b             = testutil.NewRandBlock(1024)
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				LimitBytes: 1 << 30,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		fx.indexSrv.HadCIDFunc = func(k cid.Cid) bool {
			require.Equal(t, b.Cid(), k)
			return false
		}

		storeErr := errors.New("store error")
		fx.storeSrv.PutBlockFunc = func(txn *badger.Txn, blk blocks.Block) error {
			return storeErr
		}

		resp, err := fx.rpcSrv.BlockPush(ctx, &fileproto.BlockPushRequest{
			SpaceId: storeKey.SpaceId,
			FileId:  fileId,
			Cid:     b.Cid().Bytes(),
			Data:    b.RawData(),
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to push block: store error")
		require.Nil(t, resp)
	})

	t.Run("forbidden", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileId        = testutil.NewRandCid().String()
			b             = testutil.NewRandBlock(1024)
		)

		// Simulate permission denied
		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(nil, fileprotoerr.ErrForbidden)

		resp, err := fx.rpcSrv.BlockPush(ctx, &fileproto.BlockPushRequest{
			SpaceId: storeKey.SpaceId,
			FileId:  fileId,
			Cid:     b.Cid().Bytes(),
			Data:    b.RawData(),
		})

		require.Error(t, err)
		require.Equal(t, fileprotoerr.ErrForbidden, err)
		require.Nil(t, resp)
	})
}

func TestLightFileNodeRpc_BlocksCheck(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			bs            = randBlocks(t, 3)
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.HasCIDInSpaceFunc = func(key index.Key, k cid.Cid) bool {
			require.Equal(t, storeKey, key)
			return k.Equals(bs[0].Cid())
		}

		fx.indexSrv.HadCIDFunc = func(k cid.Cid) bool {
			return k.Equals(bs[1].Cid())
		}

		cids := make([][]byte, len(bs))
		for i, b := range bs {
			cids[i] = b.Cid().Bytes()
		}

		resp, err := fx.rpcSrv.BlocksCheck(ctx, &fileproto.BlocksCheckRequest{
			SpaceId: storeKey.SpaceId,
			Cids:    cids,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.BlocksAvailability, len(bs))

		// First block exists in space
		require.Equal(t, fileproto.AvailabilityStatus_ExistsInSpace, resp.BlocksAvailability[0].Status)
		require.Equal(t, bs[0].Cid().Bytes(), resp.BlocksAvailability[0].Cid)

		// Second block exists but not in space
		require.Equal(t, fileproto.AvailabilityStatus_Exists, resp.BlocksAvailability[1].Status)
		require.Equal(t, bs[1].Cid().Bytes(), resp.BlocksAvailability[1].Cid)

		// Third block does not exist
		require.Equal(t, fileproto.AvailabilityStatus_NotExists, resp.BlocksAvailability[2].Status)
		require.Equal(t, bs[2].Cid().Bytes(), resp.BlocksAvailability[2].Cid)
	})

	t.Run("invalid cid", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx, storeKey := newRandKey()

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		resp, err := fx.rpcSrv.BlocksCheck(ctx, &fileproto.BlocksCheckRequest{
			SpaceId: storeKey.SpaceId,
			Cids:    [][]byte{[]byte("invalid")},
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to cast CID")
		require.Nil(t, resp)
	})

	t.Run("forbidden", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			b             = testutil.NewRandBlock(1024)
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(nil, fileprotoerr.ErrForbidden)

		resp, err := fx.rpcSrv.BlocksCheck(ctx, &fileproto.BlocksCheckRequest{
			SpaceId: storeKey.SpaceId,
			Cids:    [][]byte{b.Cid().Bytes()},
		})

		require.Error(t, err)
		require.Equal(t, fileprotoerr.ErrForbidden, err)
		require.Nil(t, resp)
	})

	t.Run("skip duplicate cids", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx, storeKey := newRandKey()
		b := testutil.NewRandBlock(1024)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.HasCIDInSpaceFunc = func(key index.Key, k cid.Cid) bool {
			require.Equal(t, storeKey, key)
			return true
		}

		fx.indexSrv.HadCIDFunc = func(k cid.Cid) bool {
			return false
		}

		resp, err := fx.rpcSrv.BlocksCheck(ctx, &fileproto.BlocksCheckRequest{
			SpaceId: storeKey.SpaceId,
			// Send same CID twice
			Cids: [][]byte{b.Cid().Bytes(), b.Cid().Bytes()},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.BlocksAvailability, 1) // Should only return one result
		require.Equal(t, fileproto.AvailabilityStatus_ExistsInSpace, resp.BlocksAvailability[0].Status)
		require.Equal(t, b.Cid().Bytes(), resp.BlocksAvailability[0].Cid)
	})
}

func TestLightFileNodeRpc_BlocksBind(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileId        = testutil.NewRandCid().String()
			bs            = randBlocks(t, 3)
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				LimitBytes: 1 << 30,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		// Prepare CIDs for request
		cids := make([][]byte, 0, len(bs))
		cidStrings := make([]string, 0, len(bs))
		for _, b := range bs {
			t.Logf("Block CID: %s", b.Cid().String())

			cids = append(cids, b.Cid().Bytes())
			cidStrings = append(cidStrings, b.Cid().String())
		}

		var capturedOp *indexpb.Operation

		fx.indexSrv.ModifyFunc = func(txn *badger.Txn, key index.Key, operations ...*indexpb.Operation) error {
			require.Equal(t, storeKey, key)
			require.Equal(t, 1, len(operations))
			capturedOp = operations[0]
			return nil
		}

		resp, err := fx.rpcSrv.BlocksBind(ctx, &fileproto.BlocksBindRequest{
			SpaceId: storeKey.SpaceId,
			FileId:  fileId,
			Cids:    cids,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify the operation was correct
		bindOp := capturedOp.GetBindFile()
		require.NotNil(t, bindOp)
		require.Equal(t, fileId, bindOp.GetFileId())
		require.Equal(t, len(bs), len(bindOp.GetCids()))

		// Verify all CIDs were included in the operation
		cidMap := make(map[string]bool)
		for _, cidStr := range cidStrings {
			cidMap[cidStr] = true
		}

		for _, opCid := range bindOp.GetCids() {
			require.True(t, cidMap[opCid], "Operation contains unexpected CID: %s", opCid)
		}
	})

	t.Run("deduplicates cids", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileId        = testutil.NewRandCid().String()
			b             = testutil.NewRandBlock(1024)
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				LimitBytes: 1 << 30,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		cids := [][]byte{b.Cid().Bytes(), b.Cid().Bytes()}
		var capturedOp *indexpb.Operation

		fx.indexSrv.ModifyFunc = func(txn *badger.Txn, key index.Key, operations ...*indexpb.Operation) error {
			require.Equal(t, storeKey, key)
			require.Equal(t, 1, len(operations))
			capturedOp = operations[0]
			return nil
		}

		resp, err := fx.rpcSrv.BlocksBind(ctx, &fileproto.BlocksBindRequest{
			SpaceId: storeKey.SpaceId,
			FileId:  fileId,
			Cids:    cids,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)

		bindOp := capturedOp.GetBindFile()
		require.NotNil(t, bindOp)
		require.Equal(t, fileId, bindOp.GetFileId())
		require.Equal(t, 1, len(bindOp.GetCids()), "Should have deduplicated the CIDs")
		require.Equal(t, b.Cid().String(), bindOp.GetCids()[0])
	})

	t.Run("invalid cid", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileId        = testutil.NewRandCid().String()
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				LimitBytes: 1 << 30,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		resp, err := fx.rpcSrv.BlocksBind(ctx, &fileproto.BlocksBindRequest{
			SpaceId: storeKey.SpaceId,
			FileId:  fileId,
			Cids:    [][]byte{[]byte("invalid")},
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to cast CID")
		require.Nil(t, resp)
	})

	t.Run("forbidden", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileId        = testutil.NewRandCid().String()
			b             = testutil.NewRandBlock(1024)
		)

		// Simulate permission denied
		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(nil, fileprotoerr.ErrForbidden)

		resp, err := fx.rpcSrv.BlocksBind(ctx, &fileproto.BlocksBindRequest{
			SpaceId: storeKey.SpaceId,
			FileId:  fileId,
			Cids:    [][]byte{b.Cid().Bytes()},
		})

		require.Error(t, err)
		require.Equal(t, fileprotoerr.ErrForbidden, err)
		require.Nil(t, resp)
	})

	t.Run("space limit exceeded", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileId        = testutil.NewRandCid().String()
			b             = testutil.NewRandBlock(1024)
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			limit := uint64(1024)
			return fileproto.AccountInfoResponse{
				TotalUsageBytes: limit + 1,
				LimitBytes:      limit,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		resp, err := fx.rpcSrv.BlocksBind(ctx, &fileproto.BlocksBindRequest{
			SpaceId: storeKey.SpaceId,
			FileId:  fileId,
			Cids:    [][]byte{b.Cid().Bytes()},
		})

		require.Error(t, err)
		require.Equal(t, fileprotoerr.ErrSpaceLimitExceeded, err)
		require.Nil(t, resp)
	})

	t.Run("modify error", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileId        = testutil.NewRandCid().String()
			b             = testutil.NewRandBlock(1024)
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				LimitBytes: 1 << 30,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		modifyErr := errors.New("modify error")
		fx.indexSrv.ModifyFunc = func(txn *badger.Txn, key index.Key, operations ...*indexpb.Operation) error {
			return modifyErr
		}

		resp, err := fx.rpcSrv.BlocksBind(ctx, &fileproto.BlocksBindRequest{
			SpaceId: storeKey.SpaceId,
			FileId:  fileId,
			Cids:    [][]byte{b.Cid().Bytes()},
		})

		require.Error(t, err)
		require.Equal(t, modifyErr, err)
		require.Nil(t, resp)
	})
}

func TestLightFileNodeRpc_FilesInfo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileIds       = []string{
				testutil.NewRandCid().String(),
				testutil.NewRandCid().String(),
			}
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		// Mock file info response
		expectedInfos := make([]*fileproto.FileInfo, len(fileIds))
		for i, fileId := range fileIds {
			expectedInfos[i] = &fileproto.FileInfo{
				FileId:     fileId,
				UsageBytes: uint64(1024 * (i + 1)),
				CidsCount:  uint32(i + 1),
			}
		}

		fx.indexSrv.FileInfoFunc = func(key index.Key, ids ...string) []*fileproto.FileInfo {
			require.Equal(t, storeKey, key)
			require.Equal(t, fileIds, ids)
			return expectedInfos
		}

		resp, err := fx.rpcSrv.FilesInfo(ctx, &fileproto.FilesInfoRequest{
			SpaceId: storeKey.SpaceId,
			FileIds: fileIds,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, expectedInfos, resp.FilesInfo)
	})

	t.Run("empty request", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx, storeKey := newRandKey()

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.FileInfoFunc = func(key index.Key, ids ...string) []*fileproto.FileInfo {
			require.Equal(t, storeKey, key)
			require.Empty(t, ids)
			return []*fileproto.FileInfo{}
		}

		resp, err := fx.rpcSrv.FilesInfo(ctx, &fileproto.FilesInfoRequest{
			SpaceId: storeKey.SpaceId,
			FileIds: []string{},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Empty(t, resp.FilesInfo)
	})

	t.Run("query size exceeded", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx, storeKey := newRandKey()

		// Create a slice of file IDs that exceeds the limit
		fileIds := make([]string, 2000)
		for i := range fileIds {
			fileIds[i] = testutil.NewRandCid().String()
		}

		resp, err := fx.rpcSrv.FilesInfo(ctx, &fileproto.FilesInfoRequest{
			SpaceId: storeKey.SpaceId,
			FileIds: fileIds,
		})

		require.Error(t, err)
		require.Equal(t, fileprotoerr.ErrQuerySizeExceeded, err)
		require.Nil(t, resp)
	})

	t.Run("forbidden", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileIds       = []string{
				testutil.NewRandCid().String(),
			}
		)

		// Simulate permission denied
		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(nil, fileprotoerr.ErrForbidden)

		resp, err := fx.rpcSrv.FilesInfo(ctx, &fileproto.FilesInfoRequest{
			SpaceId: storeKey.SpaceId,
			FileIds: fileIds,
		})

		require.Error(t, err)
		require.Equal(t, fileprotoerr.ErrForbidden, err)
		require.Nil(t, resp)
	})
}

func TestLightFileNodeRpc_FilesGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileIds       = []string{
				testutil.NewRandCid().String(),
				testutil.NewRandCid().String(),
				testutil.NewRandCid().String(),
			}
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.SpaceFilesFunc = func(key index.Key) []string {
			require.Equal(t, storeKey, key)
			return fileIds
		}

		stream := &mockFilesGetStream{
			ctx:      ctx,
			received: make([]*fileproto.FilesGetResponse, 0),
			t:        t,
		}

		err := fx.rpcSrv.FilesGet(&fileproto.FilesGetRequest{
			SpaceId: storeKey.SpaceId,
		}, stream)

		require.NoError(t, err)
		require.Equal(t, len(fileIds), len(stream.received))

		for i, resp := range stream.received {
			require.Equal(t, fileIds[i], resp.FileId)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx, storeKey := newRandKey()

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.SpaceFilesFunc = func(key index.Key) []string {
			require.Equal(t, storeKey, key)
			return []string{}
		}

		// Create a mock stream for testing
		stream := &mockFilesGetStream{
			ctx:      ctx,
			received: make([]*fileproto.FilesGetResponse, 0),
			t:        t,
		}

		err := fx.rpcSrv.FilesGet(&fileproto.FilesGetRequest{
			SpaceId: storeKey.SpaceId,
		}, stream)

		require.NoError(t, err)
		require.Empty(t, stream.received)
	})

	t.Run("forbidden", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx, storeKey := newRandKey()

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(nil, fileprotoerr.ErrForbidden)

		stream := &mockFilesGetStream{
			ctx:      ctx,
			received: make([]*fileproto.FilesGetResponse, 0),
			t:        t,
		}

		err := fx.rpcSrv.FilesGet(&fileproto.FilesGetRequest{
			SpaceId: storeKey.SpaceId,
		}, stream)

		require.Error(t, err)
		require.Equal(t, fileprotoerr.ErrForbidden, err)
		require.Empty(t, stream.received)
	})

	t.Run("send error", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileIds       = []string{
				testutil.NewRandCid().String(),
				testutil.NewRandCid().String(),
			}
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.SpaceFilesFunc = func(key index.Key) []string {
			require.Equal(t, storeKey, key)
			return fileIds
		}

		sendErr := errors.New("stream send error")
		stream := &mockFilesGetStream{
			ctx:     ctx,
			sendErr: sendErr,
			t:       t,
		}

		err := fx.rpcSrv.FilesGet(&fileproto.FilesGetRequest{
			SpaceId: storeKey.SpaceId,
		}, stream)

		require.Error(t, err)
		require.Equal(t, sendErr, err)
	})
}

func TestLightFileNodeRpc_FilesDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileIds       = []string{
				testutil.NewRandCid().String(),
				testutil.NewRandCid().String(),
			}
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				LimitBytes: 1 << 30,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		var capturedOp *indexpb.Operation

		fx.indexSrv.ModifyFunc = func(txn *badger.Txn, key index.Key, operations ...*indexpb.Operation) error {
			require.Equal(t, storeKey, key)
			require.Equal(t, 1, len(operations))
			capturedOp = operations[0]
			return nil
		}

		resp, err := fx.rpcSrv.FilesDelete(ctx, &fileproto.FilesDeleteRequest{
			SpaceId: storeKey.SpaceId,
			FileIds: fileIds,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)

		deleteOp := capturedOp.GetDeleteFile()
		require.NotNil(t, deleteOp)
		require.Equal(t, fileIds, deleteOp.GetFileIds())
	})

	t.Run("empty request", func(t *testing.T) {
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

		var capturedOp *indexpb.Operation

		fx.indexSrv.ModifyFunc = func(txn *badger.Txn, key index.Key, operations ...*indexpb.Operation) error {
			require.Equal(t, storeKey, key)
			require.Equal(t, 1, len(operations))
			capturedOp = operations[0]
			return nil
		}

		resp, err := fx.rpcSrv.FilesDelete(ctx, &fileproto.FilesDeleteRequest{
			SpaceId: storeKey.SpaceId,
			FileIds: []string{},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)

		deleteOp := capturedOp.GetDeleteFile()
		require.NotNil(t, deleteOp)
		require.Empty(t, deleteOp.GetFileIds())
	})

	t.Run("forbidden", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileIds       = []string{
				testutil.NewRandCid().String(),
			}
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(nil, fileprotoerr.ErrForbidden)

		resp, err := fx.rpcSrv.FilesDelete(ctx, &fileproto.FilesDeleteRequest{
			SpaceId: storeKey.SpaceId,
			FileIds: fileIds,
		})

		require.Error(t, err)
		require.Equal(t, fileprotoerr.ErrForbidden, err)
		require.Nil(t, resp)
	})

	t.Run("modify error", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			fileIds       = []string{
				testutil.NewRandCid().String(),
			}
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				LimitBytes: 1 << 30,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		modifyErr := errors.New("modify error")
		fx.indexSrv.ModifyFunc = func(txn *badger.Txn, key index.Key, operations ...*indexpb.Operation) error {
			return modifyErr
		}

		resp, err := fx.rpcSrv.FilesDelete(ctx, &fileproto.FilesDeleteRequest{
			SpaceId: storeKey.SpaceId,
			FileIds: fileIds,
		})

		require.Error(t, err)
		require.Equal(t, modifyErr, err)
		require.Nil(t, resp)
	})
}

func TestLightFileNodeRpc_Check(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx := context.Background()

		resp, err := fx.rpcSrv.Check(ctx, &fileproto.CheckRequest{})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.AllowWrite, "AllowWrite should be true in this implementation")
		require.Nil(t, resp.SpaceIds, "SpaceIds should be nil in this implementation")
	})
}

func TestLightFileNodeRpc_SpaceInfo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx, storeKey := newRandKey()

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		expectedInfo := fileproto.SpaceInfoResponse{
			TotalUsageBytes: 1024,
			FilesCount:      5,
			LimitBytes:      10240,
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			require.Equal(t, storeKey, key)
			return expectedInfo
		}

		resp, err := fx.rpcSrv.SpaceInfo(ctx, &fileproto.SpaceInfoRequest{
			SpaceId: storeKey.SpaceId,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, expectedInfo.TotalUsageBytes, resp.TotalUsageBytes)
		require.Equal(t, expectedInfo.FilesCount, resp.FilesCount)
		require.Equal(t, expectedInfo.LimitBytes, resp.LimitBytes)
	})

	t.Run("forbidden", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx, storeKey := newRandKey()

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(nil, fileprotoerr.ErrForbidden)

		resp, err := fx.rpcSrv.SpaceInfo(ctx, &fileproto.SpaceInfoRequest{
			SpaceId: storeKey.SpaceId,
		})

		require.Error(t, err)
		require.Equal(t, fileprotoerr.ErrForbidden, err)
		require.Nil(t, resp)
	})

	t.Run("empty space id", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx := context.Background()

		resp, err := fx.rpcSrv.SpaceInfo(ctx, &fileproto.SpaceInfoRequest{
			SpaceId: "",
		})

		require.Error(t, err)
		require.Equal(t, fileprotoerr.ErrForbidden, err)
		require.Nil(t, resp)
	})
}

func TestLightFileNodeRpc_AccountInfo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx, _ := newRandKey()

		// Get the actual public key from context
		ctxPubKey, err := peer.CtxPubKey(ctx)
		require.NoError(t, err)
		groupId := ctxPubKey.Account()

		expectedInfo := fileproto.AccountInfoResponse{
			TotalUsageBytes: 2048,
			LimitBytes:      102400,
		}

		fx.indexSrv.GroupInfoFunc = func(id string) fileproto.AccountInfoResponse {
			require.Equal(t, groupId, id)
			return expectedInfo
		}

		resp, err := fx.rpcSrv.AccountInfo(ctx, &fileproto.AccountInfoRequest{})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, expectedInfo.TotalUsageBytes, resp.TotalUsageBytes)
		require.Equal(t, expectedInfo.LimitBytes, resp.LimitBytes)
	})

	t.Run("no identity in context", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)

		ctx := context.Background()

		resp, err := fx.rpcSrv.AccountInfo(ctx, &fileproto.AccountInfoRequest{})

		require.Error(t, err)
		require.Equal(t, fileprotoerr.ErrForbidden, err)
		require.Nil(t, resp)
	})
}

func TestLightFileNodeRpc_SpaceLimitSet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			limit         = uint64(100 << 20) // 100 MiB
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				LimitBytes: 1 << 30,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		var capturedOp *indexpb.Operation

		fx.indexSrv.ModifyFunc = func(txn *badger.Txn, key index.Key, operations ...*indexpb.Operation) error {
			require.Equal(t, storeKey, key)
			require.Equal(t, 1, len(operations))
			capturedOp = operations[0]
			return nil
		}

		resp, err := fx.rpcSrv.SpaceLimitSet(ctx, &fileproto.SpaceLimitSetRequest{
			SpaceId: storeKey.SpaceId,
			Limit:   limit,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)

		setLimitOp := capturedOp.GetSpaceLimitSet()
		require.NotNil(t, setLimitOp)
		require.Equal(t, limit, setLimitOp.GetLimit())
	})

	t.Run("forbidden", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			limit         = uint64(100 << 20) // 100 MiB
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(nil, fileprotoerr.ErrForbidden)

		resp, err := fx.rpcSrv.SpaceLimitSet(ctx, &fileproto.SpaceLimitSetRequest{
			SpaceId: storeKey.SpaceId,
			Limit:   limit,
		})

		require.Error(t, err)
		require.Equal(t, fileprotoerr.ErrForbidden, err)
		require.Nil(t, resp)
	})

	t.Run("modify error", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		var (
			ctx, storeKey = newRandKey()
			limit         = uint64(100 << 20) // 100 MiB
		)

		fx.aclSrv.EXPECT().OwnerPubKey(ctx, storeKey.SpaceId).Return(mustPubKey(ctx), nil)

		fx.indexSrv.GroupInfoFunc = func(groupId string) fileproto.AccountInfoResponse {
			return fileproto.AccountInfoResponse{
				LimitBytes: 1 << 30,
			}
		}

		fx.indexSrv.SpaceInfoFunc = func(key index.Key) fileproto.SpaceInfoResponse {
			return fileproto.SpaceInfoResponse{}
		}

		modifyErr := errors.New("modify error")
		fx.indexSrv.ModifyFunc = func(txn *badger.Txn, key index.Key, operations ...*indexpb.Operation) error {
			return modifyErr
		}

		resp, err := fx.rpcSrv.SpaceLimitSet(ctx, &fileproto.SpaceLimitSetRequest{
			SpaceId: storeKey.SpaceId,
			Limit:   limit,
		})

		require.Error(t, err)
		require.Equal(t, modifyErr, err)
		require.Nil(t, resp)
	})
}

func TestLightFileNodeRpc_AccountLimitSet(t *testing.T) {
	t.Run("not implemented", func(t *testing.T) {
		fx := newFixture(t)
		defer fx.Finish(t)
		ctx := context.Background()

		resp, err := fx.rpcSrv.AccountLimitSet(ctx, &fileproto.AccountLimitSetRequest{
			Identity: "some-identity",
			Limit:    100 << 20, // 100 MiB
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "you can't set account limit in this implementation")
		require.Contains(t, err.Error(), fileprotoerr.ErrForbidden.Error())
		require.Nil(t, resp)
	})
}

// mockFilesGetStream is a mock implementation of fileproto.DRPCFile_FilesGetStream
type mockFilesGetStream struct {
	ctx      context.Context
	received []*fileproto.FilesGetResponse
	sendErr  error
	t        *testing.T
}

func (m *mockFilesGetStream) MsgSend(msg drpc.Message, enc drpc.Encoding) error {
	panic("should not be called")
}

func (m *mockFilesGetStream) MsgRecv(msg drpc.Message, enc drpc.Encoding) error {
	panic("should not be called")
}

func (m *mockFilesGetStream) Context() context.Context {
	return m.ctx
}

func (m *mockFilesGetStream) Send(resp *fileproto.FilesGetResponse) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.received = append(m.received, resp)
	return nil
}

func (m *mockFilesGetStream) Close() error {
	return nil
}

func (m *mockFilesGetStream) CloseSend() error {
	return nil
}

func newFixture(t *testing.T) *fixture {
	ctrl := gomock.NewController(t)

	fx := &fixture{
		ctrl:    ctrl,
		a:       new(app.App),
		drpcSrv: server.New(),
		aclSrv:  mock_acl.NewMockAclService(ctrl),
		rpcSrv:  New(),
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

// TODO: Replace with testutil.NewRandBlock when will be fixed
// https://github.com/anyproto/any-sync-filenode/issues/142
func randBlocks(t *testing.T, l int) []blocks.Block {
	newBlock := func(size int) blocks.Block {
		p := make([]byte, size)
		_, err := io.ReadFull(rand.Reader, p)
		require.NoError(t, err)

		c, err := cidutil.NewCidFromBytes(p)
		require.NoError(t, err)

		b, err := blocks.NewBlockWithCid(p, cid.MustParse(c))
		require.NoError(t, err)

		return b
	}

	bs := make([]blocks.Block, l)
	for i := range bs {
		bs[i] = newBlock(10 * l)
	}
	return bs
}

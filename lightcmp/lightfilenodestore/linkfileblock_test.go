package lightfilenodestore

import (
	"testing"

	"github.com/anyproto/any-sync-filenode/testutil"
	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
)

func TestKeyLinkFileBlock(t *testing.T) {
	spaceId := testutil.NewRandSpaceId()
	fileId := testutil.NewRandSpaceId()
	c, err := cid.Decode("QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
	require.NoError(t, err)

	expected := kPrefixLinkFileBlock + kSeparator + spaceId + kSeparator + fileId + kSeparator + c.String()
	key := keyLinkFileBlock(spaceId, fileId, c)
	require.Equal(t, expected, string(key))
}

func TestNewLinkFileBlockObj(t *testing.T) {
	spaceId := testutil.NewRandSpaceId()
	fileId := testutil.NewRandSpaceId()
	c, err := cid.Decode("QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
	require.NoError(t, err)

	link := NewLinkFileBlockObj(spaceId, fileId, c)
	require.Equal(t, spaceId, link.spaceID)
	require.Equal(t, fileId, link.fileID)
	require.Equal(t, c, link.cid)

	expectedKey := kPrefixLinkFileBlock + kSeparator + spaceId + kSeparator + fileId + kSeparator + c.String()
	require.Equal(t, expectedKey, string(link.key))
}

func TestLinkFileBlockObj_ExistsAndWrite(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	spaceId := testutil.NewRandSpaceId()
	fileId := testutil.NewRandSpaceId()
	c, err := cid.Decode("QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
	require.NoError(t, err)

	link := NewLinkFileBlockObj(spaceId, fileId, c)

	// Initially link should not exist
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		exists, err := link.exists(txn)
		require.NoError(t, err)
		require.False(t, exists)
		return nil
	})
	require.NoError(t, err)

	// Write link
	err = fx.storeSrv.TxUpdate(func(txn *badger.Txn) error {
		return link.write(txn)
	})
	require.NoError(t, err)

	// Verify link exists
	err = fx.storeSrv.TxView(func(txn *badger.Txn) error {
		exists, err := link.exists(txn)
		require.NoError(t, err)
		require.True(t, exists)
		return nil
	})
	require.NoError(t, err)
}

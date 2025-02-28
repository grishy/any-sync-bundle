package lightfilenodestore

import (
	"testing"

	"github.com/anyproto/any-sync-filenode/testutil"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

func TestKeyLinkGroupSpace(t *testing.T) {
	groupId := "group1"
	spaceId := "space1"
	expected := kPrefixLinkGroupSpace + kSeparator + groupId + kSeparator + spaceId
	key := keyLinkGroupSpace(groupId, spaceId)
	require.Equal(t, expected, string(key))
}

func TestNewLinkGroupSpaceObj(t *testing.T) {
	groupId := testutil.NewRandSpaceId()
	spaceId := testutil.NewRandSpaceId()

	link := NewLinkGroupSpaceObj(groupId, spaceId)
	require.Equal(t, groupId, link.groupID)
	require.Equal(t, spaceId, link.spaceID)

	expectedKey := kPrefixLinkGroupSpace + kSeparator + groupId + kSeparator + spaceId
	require.Equal(t, expectedKey, string(link.key))
}

func TestLinkGroupSpaceObj_ExistsAndWrite(t *testing.T) {
	fx := newFixture(t)
	defer fx.Finish(t)

	groupId := testutil.NewRandSpaceId()
	spaceId := testutil.NewRandSpaceId()

	link := NewLinkGroupSpaceObj(groupId, spaceId)

	// Initially link should not exist
	err := fx.storeSrv.TxView(func(txn *badger.Txn) error {
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

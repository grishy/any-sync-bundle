package lightfilenodestore

import (
	"testing"

	"github.com/anyproto/any-sync-filenode/testutil"
	"github.com/stretchr/testify/require"
)

func TestKeyBlock(t *testing.T) {
	cid := testutil.NewRandCid()
	expected := kPrefixBlock + kSeparator + cid.String()
	block := NewBlockObj(cid)
	require.Equal(t, expected, string(block.key))
}

func TestBlockDataAccessors(t *testing.T) {
	cid := testutil.NewRandCid()
	data := []byte("test data")

	block := NewBlockObj(cid)
	require.Nil(t, block.Data())

	block = block.WithData(data)
	require.Equal(t, data, block.Data())
}

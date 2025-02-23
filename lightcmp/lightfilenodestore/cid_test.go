package lightfilenodestore

import (
	"testing"

	"github.com/anyproto/any-sync-filenode/testutil"
	"github.com/stretchr/testify/require"
)

func TestKeyCid(t *testing.T) {
	spaceId := "space1"
	cid := testutil.NewRandCid()
	expected := kPrefixCid + kSeparator + spaceId + kSeparator + string(cid.Bytes())
	key := keyCid(spaceId, cid)
	require.Equal(t, expected, string(key))
}

func TestCidMarshalAndUnmarshalValue(t *testing.T) {
	spaceId := "space1"
	cid := testutil.NewRandCid()
	refCount := uint32(5)
	sizeByte := uint32(1024)
	createdAt := int64(1234567890)

	// Create and marshal CID object
	cidObj := NewCidObj(spaceId, cid).
		WithRefCount(refCount).
		WithSizeByte(sizeByte).
		WithCreatedAt(createdAt)
	marshaled := cidObj.marshalValue()

	// Verify marshaled length
	require.Equal(t, valueCidSize, len(marshaled))

	// Unmarshal into new object and verify values
	newCidObj := NewCidObj(spaceId, cid)
	require.NoError(t, newCidObj.unmarshalValue(marshaled))
	require.Equal(t, refCount, newCidObj.refCount)
	require.Equal(t, sizeByte, newCidObj.sizeByte)
	require.Equal(t, createdAt, newCidObj.createdAt)
}

func TestCidUnmarshalValueShortInput(t *testing.T) {
	spaceId := "space1"
	cid := testutil.NewRandCid()

	cidObj := NewCidObj(spaceId, cid)
	shortValue := make([]byte, valueCidSize-1)
	err := cidObj.unmarshalValue(shortValue)
	require.Error(t, err)
}

package lightfilenodestore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeySpace(t *testing.T) {
	spaceId := "space1"
	expected := kPrefixSpace + kSeparator + spaceId
	key := keySpace(spaceId)
	require.Equal(t, expected, string(key))
}

func TestSpaceMarshalAndUnmarshalValue(t *testing.T) {
	spaceId := "space1"
	limitBytes := uint64(123456789)
	cidsCount := uint64(42)
	filesCount := uint64(10)
	spaceUsageBytes := uint64(1024)

	// Create and marshal space object
	space := NewSpaceObj(spaceId).
		WithLimitBytes(limitBytes).
		WithSpaceUsageBytes(spaceUsageBytes).
		WithCidsCount(cidsCount).
		WithFilesCount(filesCount)

	marshaled := space.marshalValue()

	// Verify marshaled length
	require.Equal(t, valueSpaceSize, len(marshaled))

	// Unmarshal into new object and verify values
	newSpace := NewSpaceObj(spaceId)
	require.NoError(t, newSpace.unmarshalValue(marshaled))
	require.Equal(t, limitBytes, newSpace.LimitBytes())
	require.Equal(t, spaceUsageBytes, newSpace.SpaceUsageBytes())
	require.Equal(t, cidsCount, newSpace.CidsCount())
	require.Equal(t, filesCount, newSpace.FilesCount())
}

func TestSpaceUnmarshalValueShortInput(t *testing.T) {
	spaceId := "space1"

	space := NewSpaceObj(spaceId)
	shortValue := make([]byte, valueSpaceSize-1)
	err := space.unmarshalValue(shortValue)
	require.Error(t, err)
}

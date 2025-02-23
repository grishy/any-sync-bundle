package lightfilenodestore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeyFile(t *testing.T) {
	spaceId := "space1"
	fileId := "file1"
	expected := kPrefixFile + kSeparator + spaceId + kSeparator + fileId
	key := keyFile(spaceId, fileId)
	require.Equal(t, expected, string(key))
}

func TestMarshalAndUnmarshalValue(t *testing.T) {
	spaceId := "space1"
	fileId := "file1"
	usageBytes := uint64(123456789)
	cidsCount := uint32(42)

	// Create and marshal file object
	file := NewFileObj(spaceId, fileId).
		WithUsageBytes(usageBytes).
		WithCidsCount(cidsCount)
	marshaled := file.marshalValue()

	// Verify marshaled length
	require.Equal(t, valueFileSize, len(marshaled))

	// Unmarshal into new object and verify values
	newFile := NewFileObj(spaceId, fileId)
	require.NoError(t, newFile.unmarshalValue(marshaled))
	require.Equal(t, usageBytes, newFile.UsageBytes())
	require.Equal(t, cidsCount, newFile.CidsCount())
}

func TestUnmarshalValueShortInput(t *testing.T) {
	spaceId := "space1"
	fileId := "file1"

	file := NewFileObj(spaceId, fileId)
	shortValue := make([]byte, valueFileSize-1)
	err := file.unmarshalValue(shortValue)
	require.Error(t, err)
}

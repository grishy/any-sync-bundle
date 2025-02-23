package lightfilenodestore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeyGroup(t *testing.T) {
	groupId := "group1"
	expected := kPrefixGroup + kSeparator + groupId
	key := keyGroup(groupId)
	require.Equal(t, expected, string(key))
}

func TestGroupMarshalAndUnmarshalValue(t *testing.T) {
	groupId := "group1"
	limitBytes := uint64(123456789)
	totalUsageBytes := uint64(987654321)
	totalCidsCount := uint64(42)

	// Create and marshal group object
	group := NewGroupObj(groupId).
		WithLimitBytes(limitBytes).
		WithTotalUsageBytes(totalUsageBytes).
		WithTotalCidsCount(totalCidsCount)
	marshaled := group.marshalValue()

	// Verify marshaled length
	require.Equal(t, valueGroupSize, len(marshaled))

	// Unmarshal into new object and verify values
	newGroup := NewGroupObj(groupId)
	require.NoError(t, newGroup.unmarshalValue(marshaled))
	require.Equal(t, limitBytes, newGroup.LimitBytes())
	require.Equal(t, totalUsageBytes, newGroup.TotalUsageBytes())
	require.Equal(t, totalCidsCount, newGroup.TotalCidsCount())
}

func TestGroupUnmarshalValueShortInput(t *testing.T) {
	groupId := "group1"

	group := NewGroupObj(groupId)
	shortValue := make([]byte, valueGroupSize-1)
	err := group.unmarshalValue(shortValue)
	require.Error(t, err)
}

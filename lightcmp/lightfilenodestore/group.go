package lightfilenodestore

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

const (
	kPrefixGroup = kPrefixFileNode + kSeparator + "g"

	valueGroupSize = 24 // 8 bytes for limit + 16 bytes for counters
)

func keyGroup(groupID string) []byte {
	var buf bytes.Buffer
	buf.Grow(len(kPrefixGroup) + len(kSeparator) + len(groupID))

	buf.WriteString(kPrefixGroup)
	buf.WriteString(kSeparator)
	buf.WriteString(groupID)

	return buf.Bytes()
}

type GroupObj struct {
	key []byte

	// Key parts
	groupID string

	// Value parts
	limitBytes      uint64 // Account storage limit in bytes
	totalUsageBytes uint64 // Total account usage in bytes
	totalCidsCount  uint64 // Total number of CIDs in account
}

//
// Public methods
//

func NewGroupObj(groupID string) *GroupObj {
	return &GroupObj{
		key:     keyGroup(groupID),
		groupID: groupID,
	}
}

func (g *GroupObj) LimitBytes() uint64 {
	return g.limitBytes
}

func (g *GroupObj) TotalUsageBytes() uint64 {
	return g.totalUsageBytes
}

func (g *GroupObj) TotalCidsCount() uint64 {
	return g.totalCidsCount
}

func (g *GroupObj) WithLimitBytes(limitBytes uint64) *GroupObj {
	g.limitBytes = limitBytes
	return g
}

func (g *GroupObj) WithTotalUsageBytes(totalUsageBytes uint64) *GroupObj {
	g.totalUsageBytes = totalUsageBytes
	return g
}

func (g *GroupObj) WithTotalCidsCount(totalCidsCount uint64) *GroupObj {
	g.totalCidsCount = totalCidsCount
	return g
}

//
// Private methods - Value operations
//

func (g *GroupObj) marshalValue() []byte {
	buf := make([]byte, valueGroupSize)
	binary.LittleEndian.PutUint64(buf[0:8], g.limitBytes)
	binary.LittleEndian.PutUint64(buf[8:16], g.totalUsageBytes)
	binary.LittleEndian.PutUint64(buf[16:24], g.totalCidsCount)
	return buf
}

func (g *GroupObj) unmarshalValue(val []byte) error {
	if len(val) < valueGroupSize {
		return fmt.Errorf("value too short, expected at least %d bytes, got %d", valueGroupSize, len(val))
	}
	g.limitBytes = binary.LittleEndian.Uint64(val[0:8])
	g.totalUsageBytes = binary.LittleEndian.Uint64(val[8:16])
	g.totalCidsCount = binary.LittleEndian.Uint64(val[16:24])
	return nil
}

func (g *GroupObj) write(txn *badger.Txn) error {
	if err := txn.Set(g.key, g.marshalValue()); err != nil {
		return fmt.Errorf("failed to write value: %w", err)
	}
	return nil
}

func (g *GroupObj) populateValue(txn *badger.Txn) error {
	item, err := txn.Get(g.key)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	if err := item.Value(func(val []byte) error {
		return g.unmarshalValue(val)
	}); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}
	return nil
}

package lightfilenodestore

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

const (
	kPrefixGroup = kPrefixFileNode + kSeparator + "g"

	valueGroupMetaSize    = 8  // 8 bytes for limit
	valueGroupCounterSize = 16 // 8 + 8 bytes for counters
)

func keyGroupMeta(identity string) []byte {
	const groupName = "meta"
	var buf bytes.Buffer
	buf.Grow(len(kPrefixGroup) + len(kSeparator) + len(identity) + len(kSeparator) + len(groupName))

	buf.WriteString(kPrefixGroup)
	buf.WriteString(kSeparator)
	buf.WriteString(identity)
	buf.WriteString(kSeparator)
	buf.WriteString(groupName)

	return buf.Bytes()
}

func keyGroupCounter(identity string) []byte {
	const groupName = "counter"
	var buf bytes.Buffer
	buf.Grow(len(kPrefixGroup) + len(kSeparator) + len(identity) + len(kSeparator) + len(groupName))

	buf.WriteString(kPrefixGroup)
	buf.WriteString(kSeparator)
	buf.WriteString(identity)
	buf.WriteString(kSeparator)
	buf.WriteString(groupName)

	return buf.Bytes()
}

type GroupObj struct {
	keyMeta     []byte
	keyCounters []byte

	// Key parts
	identity string

	// Meta value parts
	limitBytes uint64 // Account storage limit in bytes

	// Counter value parts
	totalUsageBytes uint64 // Total account usage in bytes
	totalCidsCount  uint64 // Total number of CIDs in account
}

//
// Public methods
//

func NewGroupObj(identity string) *GroupObj {
	return &GroupObj{
		keyMeta:     keyGroupMeta(identity),
		keyCounters: keyGroupCounter(identity),
		identity:    identity,
	}
}

//
// Private methods - Meta operations
//

func (g *GroupObj) marshalMetaValue() []byte {
	buf := make([]byte, valueGroupMetaSize)
	binary.LittleEndian.PutUint64(buf[0:8], g.limitBytes)
	return buf
}

func (g *GroupObj) unmarshalMetaValue(val []byte) error {
	if len(val) < valueGroupMetaSize {
		return fmt.Errorf("meta value too short, expected at least %d bytes, got %d", valueGroupMetaSize, len(val))
	}
	g.limitBytes = binary.LittleEndian.Uint64(val[0:8])
	return nil
}

func (g *GroupObj) writeMeta(txn *badger.Txn) error {
	if err := txn.Set(g.keyMeta, g.marshalMetaValue()); err != nil {
		return fmt.Errorf("failed to write meta: %w", err)
	}
	return nil
}

func (g *GroupObj) existsMeta(txn *badger.Txn) (bool, error) {
	_, err := txn.Get(g.keyMeta)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get meta item: %w", err)
	}
	return true, nil
}

func (g *GroupObj) populateMeta(txn *badger.Txn) error {
	metaItem, err := txn.Get(g.keyMeta)
	if err != nil {
		return fmt.Errorf("failed to get meta item: %w", err)
	}

	if err := metaItem.Value(func(val []byte) error {
		return g.unmarshalMetaValue(val)
	}); err != nil {
		return fmt.Errorf("failed to unmarshal meta value: %w", err)
	}
	return nil
}

//
// Private methods - Counter operations
//

func (g *GroupObj) marshalCounterValue() []byte {
	buf := make([]byte, valueGroupCounterSize)
	binary.LittleEndian.PutUint64(buf[0:8], g.totalUsageBytes)
	binary.LittleEndian.PutUint64(buf[8:16], g.totalCidsCount)
	return buf
}

func (g *GroupObj) unmarshalCounterValue(val []byte) error {
	if len(val) < valueGroupCounterSize {
		return fmt.Errorf("counter value too short, expected at least %d bytes, got %d", valueGroupCounterSize, len(val))
	}
	g.totalUsageBytes = binary.LittleEndian.Uint64(val[0:8])
	g.totalCidsCount = binary.LittleEndian.Uint64(val[8:16])
	return nil
}

func (g *GroupObj) writeCounter(txn *badger.Txn) error {
	if err := txn.Set(g.keyCounters, g.marshalCounterValue()); err != nil {
		return fmt.Errorf("failed to write counters: %w", err)
	}
	return nil
}

func (g *GroupObj) existsCounter(txn *badger.Txn) (bool, error) {
	_, err := txn.Get(g.keyCounters)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get counter item: %w", err)
	}
	return true, nil
}

func (g *GroupObj) populateCounter(txn *badger.Txn) error {
	counterItem, err := txn.Get(g.keyCounters)
	if err != nil {
		return fmt.Errorf("failed to get counter item: %w", err)
	}

	if err := counterItem.Value(func(val []byte) error {
		return g.unmarshalCounterValue(val)
	}); err != nil {
		return fmt.Errorf("failed to unmarshal counter value: %w", err)
	}
	return nil
}

//
// Private methods - Combined operations
//

func (g *GroupObj) write(txn *badger.Txn) error {
	if err := g.writeMeta(txn); err != nil {
		return err
	}
	return g.writeCounter(txn)
}

func (g *GroupObj) exists(txn *badger.Txn) (bool, error) {
	existsMeta, err := g.existsMeta(txn)
	if err != nil {
		return false, err
	}
	existsCounter, err := g.existsCounter(txn)
	if err != nil {
		return false, err
	}
	return existsMeta && existsCounter, nil
}

func (g *GroupObj) populateValue(txn *badger.Txn) error {
	if err := g.populateMeta(txn); err != nil {
		return err
	}
	return g.populateCounter(txn)
}

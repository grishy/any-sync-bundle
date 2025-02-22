package lightfilenodestore

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

const (
	kPrefixSpace = kPrefixFileNode + kSeparator + "s"

	valueSpaceMetaSize    = 8  // 8 bytes for limit
	valueSpaceCounterSize = 32 // 8 + 8 + 8 + 8 bytes for counters
)

func keySpaceMeta(spaceID string) []byte {
	const groupName = "meta"
	var buf bytes.Buffer
	buf.Grow(len(kPrefixSpace) + len(kSeparator) + len(spaceID) + len(kSeparator) + len(groupName))

	buf.WriteString(kPrefixSpace)
	buf.WriteString(kSeparator)
	buf.WriteString(spaceID)
	buf.WriteString(kSeparator)
	buf.WriteString(groupName)

	return buf.Bytes()
}

func keySpaceCounter(spaceID string) []byte {
	const groupName = "counter"
	var buf bytes.Buffer
	buf.Grow(len(kPrefixSpace) + len(kSeparator) + len(spaceID) + len(kSeparator) + len(groupName))

	buf.WriteString(kPrefixSpace)
	buf.WriteString(kSeparator)
	buf.WriteString(spaceID)
	buf.WriteString(kSeparator)
	buf.WriteString(groupName)

	return buf.Bytes()
}

type SpaceObj struct {
	keyMeta     []byte
	keyCounters []byte

	// Key parts
	spaceID string

	// Meta value parts
	limitBytes uint64 // Space storage limit in bytes

	// Counter value parts
	totalUsageBytes uint64 // Total space usage in bytes
	cidsCount       uint64 // Number of CIDs in space
	filesCount      uint64 // Number of files in space
	spaceUsageBytes uint64 // Space usage in bytes
}

//
// Public methods
//

func NewSpaceObj(spaceID string) *SpaceObj {
	return &SpaceObj{
		keyMeta:     keySpaceMeta(spaceID),
		keyCounters: keySpaceCounter(spaceID),
		spaceID:     spaceID,
	}
}

//
// Private methods - Meta operations
//

func (s *SpaceObj) marshalMetaValue() []byte {
	buf := make([]byte, valueSpaceMetaSize)
	binary.LittleEndian.PutUint64(buf[0:8], s.limitBytes)
	return buf
}

func (s *SpaceObj) unmarshalMetaValue(val []byte) error {
	if len(val) < valueSpaceMetaSize {
		return fmt.Errorf("meta value too short, expected at least %d bytes, got %d", valueSpaceMetaSize, len(val))
	}
	s.limitBytes = binary.LittleEndian.Uint64(val[0:8])
	return nil
}

func (s *SpaceObj) writeMeta(txn *badger.Txn) error {
	if err := txn.Set(s.keyMeta, s.marshalMetaValue()); err != nil {
		return fmt.Errorf("failed to write meta: %w", err)
	}
	return nil
}

func (s *SpaceObj) existsMeta(txn *badger.Txn) (bool, error) {
	_, err := txn.Get(s.keyMeta)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get meta item: %w", err)
	}
	return true, nil
}

func (s *SpaceObj) populateMeta(txn *badger.Txn) error {
	metaItem, err := txn.Get(s.keyMeta)
	if err != nil {
		return fmt.Errorf("failed to get meta item: %w", err)
	}

	if err := metaItem.Value(func(val []byte) error {
		return s.unmarshalMetaValue(val)
	}); err != nil {
		return fmt.Errorf("failed to unmarshal meta value: %w", err)
	}
	return nil
}

//
// Private methods - Counter operations
//

func (s *SpaceObj) marshalCounterValue() []byte {
	buf := make([]byte, valueSpaceCounterSize)
	binary.LittleEndian.PutUint64(buf[0:8], s.totalUsageBytes)
	binary.LittleEndian.PutUint64(buf[8:16], s.cidsCount)
	binary.LittleEndian.PutUint64(buf[16:24], s.filesCount)
	binary.LittleEndian.PutUint64(buf[24:32], s.spaceUsageBytes)
	return buf
}

func (s *SpaceObj) unmarshalCounterValue(val []byte) error {
	if len(val) < valueSpaceCounterSize {
		return fmt.Errorf("counter value too short, expected at least %d bytes, got %d", valueSpaceCounterSize, len(val))
	}
	s.totalUsageBytes = binary.LittleEndian.Uint64(val[0:8])
	s.cidsCount = binary.LittleEndian.Uint64(val[8:16])
	s.filesCount = binary.LittleEndian.Uint64(val[16:24])
	s.spaceUsageBytes = binary.LittleEndian.Uint64(val[24:32])
	return nil
}

func (s *SpaceObj) writeCounter(txn *badger.Txn) error {
	if err := txn.Set(s.keyCounters, s.marshalCounterValue()); err != nil {
		return fmt.Errorf("failed to write counters: %w", err)
	}
	return nil
}

func (s *SpaceObj) existsCounter(txn *badger.Txn) (bool, error) {
	_, err := txn.Get(s.keyCounters)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get counter item: %w", err)
	}
	return true, nil
}

func (s *SpaceObj) populateCounter(txn *badger.Txn) error {
	counterItem, err := txn.Get(s.keyCounters)
	if err != nil {
		return fmt.Errorf("failed to get counter item: %w", err)
	}

	if err := counterItem.Value(func(val []byte) error {
		return s.unmarshalCounterValue(val)
	}); err != nil {
		return fmt.Errorf("failed to unmarshal counter value: %w", err)
	}
	return nil
}

//
// Private methods - Combined operations
//

func (s *SpaceObj) write(txn *badger.Txn) error {
	if err := s.writeMeta(txn); err != nil {
		return err
	}
	return s.writeCounter(txn)
}

func (s *SpaceObj) exists(txn *badger.Txn) (bool, error) {
	existsMeta, err := s.existsMeta(txn)
	if err != nil {
		return false, err
	}
	existsCounter, err := s.existsCounter(txn)
	if err != nil {
		return false, err
	}
	return existsMeta && existsCounter, nil
}

func (s *SpaceObj) populateValue(txn *badger.Txn) error {
	if err := s.populateMeta(txn); err != nil {
		return err
	}
	return s.populateCounter(txn)
}

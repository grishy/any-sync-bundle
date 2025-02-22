package lightfilenodestore

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

const (
	kPrefixSpace = kPrefixFileNode + kSeparator + "s"

	valueSpaceSize = 40 // 8 bytes for limit + 32 bytes for counters
)

func keySpace(spaceID string) []byte {
	var buf bytes.Buffer
	buf.Grow(len(kPrefixSpace) + len(kSeparator) + len(spaceID))

	buf.WriteString(kPrefixSpace)
	buf.WriteString(kSeparator)
	buf.WriteString(spaceID)

	return buf.Bytes()
}

type SpaceObj struct {
	key []byte

	// Key parts
	spaceID string

	// Value parts
	limitBytes      uint64 // Space storage limit in bytes
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
		key:     keySpace(spaceID),
		spaceID: spaceID,
	}
}

func (s *SpaceObj) LimitBytes() uint64 {
	return s.limitBytes
}

func (s *SpaceObj) TotalUsageBytes() uint64 {
	return s.totalUsageBytes
}

func (s *SpaceObj) CidsCount() uint64 {
	return s.cidsCount
}

func (s *SpaceObj) FilesCount() uint64 {
	return s.filesCount
}

func (s *SpaceObj) SpaceUsageBytes() uint64 {
	return s.spaceUsageBytes
}

func (s *SpaceObj) WithLimitBytes(limitBytes uint64) *SpaceObj {
	s.limitBytes = limitBytes
	return s
}

func (s *SpaceObj) WithTotalUsageBytes(totalUsageBytes uint64) *SpaceObj {
	s.totalUsageBytes = totalUsageBytes
	return s
}

func (s *SpaceObj) WithCidsCount(cidsCount uint64) *SpaceObj {
	s.cidsCount = cidsCount
	return s
}

func (s *SpaceObj) WithFilesCount(filesCount uint64) *SpaceObj {
	s.filesCount = filesCount
	return s
}

func (s *SpaceObj) WithSpaceUsageBytes(spaceUsageBytes uint64) *SpaceObj {
	s.spaceUsageBytes = spaceUsageBytes
	return s
}

//
// Private methods - Value operations
//

func (s *SpaceObj) marshalValue() []byte {
	buf := make([]byte, valueSpaceSize)
	binary.LittleEndian.PutUint64(buf[0:8], s.limitBytes)
	binary.LittleEndian.PutUint64(buf[8:16], s.totalUsageBytes)
	binary.LittleEndian.PutUint64(buf[16:24], s.cidsCount)
	binary.LittleEndian.PutUint64(buf[24:32], s.filesCount)
	binary.LittleEndian.PutUint64(buf[32:40], s.spaceUsageBytes)
	return buf
}

func (s *SpaceObj) unmarshalValue(val []byte) error {
	if len(val) < valueSpaceSize {
		return fmt.Errorf("value too short, expected at least %d bytes, got %d", valueSpaceSize, len(val))
	}
	s.limitBytes = binary.LittleEndian.Uint64(val[0:8])
	s.totalUsageBytes = binary.LittleEndian.Uint64(val[8:16])
	s.cidsCount = binary.LittleEndian.Uint64(val[16:24])
	s.filesCount = binary.LittleEndian.Uint64(val[24:32])
	s.spaceUsageBytes = binary.LittleEndian.Uint64(val[32:40])
	return nil
}

func (s *SpaceObj) write(txn *badger.Txn) error {
	if err := txn.Set(s.key, s.marshalValue()); err != nil {
		return fmt.Errorf("failed to write value: %w", err)
	}
	return nil
}

func (s *SpaceObj) populateValue(txn *badger.Txn) error {
	item, err := txn.Get(s.key)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	if err := item.Value(func(val []byte) error {
		return s.unmarshalValue(val)
	}); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}
	return nil
}

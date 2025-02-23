package lightfilenodestore

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

const (
	kPrefixFile   = kPrefixFileNode + kSeparator + "f"
	valueFileSize = 12 // 8 bytes for usageBytes + 4 bytes for cidsCount
)

func keyFile(spaceID, fileID string) []byte {
	var buf bytes.Buffer
	buf.Grow(len(kPrefixFile) + len(kSeparator) + len(spaceID) + len(kSeparator) + len(fileID))

	buf.WriteString(kPrefixFile)
	buf.WriteString(kSeparator)
	buf.WriteString(spaceID)
	buf.WriteString(kSeparator)
	buf.WriteString(fileID)

	return buf.Bytes()
}

type FileObj struct {
	key []byte

	// Key parts
	spaceID string
	fileID  string

	// Value parts
	usageBytes uint64 // File usage in bytes
	cidsCount  uint32 // Number of CIDs in file
}

func NewFileObj(spaceID, fileID string) *FileObj {
	return &FileObj{
		key:     keyFile(spaceID, fileID),
		spaceID: spaceID,
		fileID:  fileID,
	}
}

func (f *FileObj) UsageBytes() uint64 {
	return f.usageBytes
}

func (f *FileObj) CidsCount() uint32 {
	return f.cidsCount
}

func (f *FileObj) WithUsageBytes(usageBytes uint64) *FileObj {
	f.usageBytes = usageBytes
	return f
}

func (f *FileObj) WithCidsCount(cidsCount uint32) *FileObj {
	f.cidsCount = cidsCount
	return f
}

func (f *FileObj) marshalValue() []byte {
	buf := make([]byte, valueFileSize)
	binary.LittleEndian.PutUint64(buf[0:8], f.usageBytes)
	binary.LittleEndian.PutUint32(buf[8:12], f.cidsCount)
	return buf
}

func (f *FileObj) unmarshalValue(val []byte) error {
	if len(val) < valueFileSize {
		return fmt.Errorf("value too short, expected at least %d bytes, got %d", valueFileSize, len(val))
	}
	f.usageBytes = binary.LittleEndian.Uint64(val[0:8])
	f.cidsCount = binary.LittleEndian.Uint32(val[8:12])
	return nil
}

func (f *FileObj) write(txn *badger.Txn) error {
	if err := txn.Set(f.key, f.marshalValue()); err != nil {
		return fmt.Errorf("failed to write value: %w", err)
	}
	return nil
}

func (f *FileObj) populateValue(txn *badger.Txn) error {
	item, err := txn.Get(f.key)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	if err := item.Value(func(val []byte) error {
		return f.unmarshalValue(val)
	}); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}
	return nil
}

package lightfilenodestore

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
)

// More about CID: https://docs.ipfs.tech/concepts/content-addressing/

const (
	kPrefixCid   = kPrefixFileNode + kSeparator + "c"
	valueCidSize = 16 // 4 + 4 + 8 bytes for uint32 + uint32 + int64
)

func keyCid(spaceID string, k cid.Cid) []byte {
	var buf bytes.Buffer
	buf.Grow(len(kPrefixCid) + len(kSeparator) + len(spaceID) + len(kSeparator) + len(k.Bytes()))

	buf.WriteString(kPrefixCid)
	buf.WriteString(kSeparator)
	buf.WriteString(spaceID)
	buf.WriteString(kSeparator)
	buf.Write(k.Bytes())

	return buf.Bytes()
}

type CidObj struct {
	key []byte

	// Key parts
	cid cid.Cid
	// Value
	refCount  uint32
	sizeByte  uint32
	createdAt int64
}

//
// Public methods
//

func NewCidObj(spaceID string, k cid.Cid) *CidObj {
	return &CidObj{
		key:       keyCid(spaceID, k),
		cid:       k,
		refCount:  0,
		sizeByte:  0,
		createdAt: 0,
	}
}

func (c *CidObj) WithRefCount(refCount uint32) *CidObj {
	c.refCount = refCount
	return c
}

func (c *CidObj) WithSizeByte(sizeByte uint32) *CidObj {
	c.sizeByte = sizeByte
	return c
}

func (c *CidObj) WithCreatedAt(createdAt int64) *CidObj {
	c.createdAt = createdAt
	return c
}

//
// Private methods
//

func (c *CidObj) marshalValue() []byte {
	buf := make([]byte, valueCidSize)
	binary.LittleEndian.PutUint32(buf[0:4], c.refCount)
	binary.LittleEndian.PutUint32(buf[4:8], c.sizeByte)
	binary.LittleEndian.PutUint64(buf[8:16], uint64(c.createdAt))
	return buf
}

func (c *CidObj) unmarshalValue(val []byte) error {
	if len(val) < valueCidSize {
		return fmt.Errorf("value too short, expected at least %d bytes, got %d", valueCidSize, len(val))
	}
	c.refCount = binary.LittleEndian.Uint32(val[0:4])
	c.sizeByte = binary.LittleEndian.Uint32(val[4:8])
	c.createdAt = int64(binary.LittleEndian.Uint64(val[8:16]))
	return nil
}

func (c *CidObj) write(txn *badger.Txn) error {
	return txn.Set(c.key, c.marshalValue())
}

func (c *CidObj) exists(txn *badger.Txn) (bool, error) {
	_, err := txn.Get(c.key)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get item: %w", err)
	}

	return true, nil
}

func (c *CidObj) populateValue(txn *badger.Txn) error {
	item, err := txn.Get(c.key)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	return item.Value(func(val []byte) error {
		return c.unmarshalValue(val)
	})
}

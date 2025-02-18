package lightfilenodestore

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
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
	// Key
	cid cid.Cid
	// Value
	refCount  uint32
	sizeByte  uint32
	createdAt int64
}

func (c *CidObj) marshalValue() ([]byte, error) {
	return c.cid.Bytes(), nil
}

func (c *CidObj) unmarshalValue(data []byte) error {
	var err error
	c.cid, err = cid.Cast(data)
	return err
}

func fetchStoreCid(txn *badger.Txn, spaceID string, k cid.Cid) (*CidObj, error) {
	key := keyCid(spaceID, k)

	obj := &CidObj{
		cid: k,
	}

	item, err := txn.Get(key)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return obj, nil
		} else {
			return nil, fmt.Errorf("failed to get item: %w", err)
		}
	}

	err = item.Value(func(val []byte) error {
		return obj.unmarshalValue(val)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return obj, nil
}

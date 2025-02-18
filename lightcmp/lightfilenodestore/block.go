package lightfilenodestore

import (
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
)

func keyBlock(spaceId string, k cid.Cid) []byte {
	return []byte(kPrefixBlock + spaceId + kSeparator + k.String())
}

type BlockObj struct {
	// Key
	cid cid.Cid
	// Value
	data []byte
}

func readDbBlock(txn *badger.Txn, spaceID string, k cid.Cid) (*BlockObj, error) {
	key := keyBlock(spaceID, k)

	obj := &BlockObj{
		cid:  k,
		data: nil,
	}

	item, err := txn.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	err = item.Value(func(val []byte) error {
		obj.data = val
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get value: %w", err)
	}

	return obj, nil
}

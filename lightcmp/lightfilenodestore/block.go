package lightfilenodestore

import (
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
)

const kPrefixBlock = kPrefixFileNode + kSeparator + "b"

type BlockObj struct {
	key []byte

	// Key parts
	cid cid.Cid
	// Value
	data []byte
}

//
// Public methods
//

func NewBlockObj(k cid.Cid) *BlockObj {
	return &BlockObj{
		key:  []byte(kPrefixBlock + kSeparator + k.String()),
		cid:  k,
		data: nil,
	}
}

func (b *BlockObj) WithData(data []byte) *BlockObj {
	b.data = data
	return b
}

func (b *BlockObj) Data() []byte {
	return b.data
}

//
// Private methods
//

func (b *BlockObj) populateData(txn *badger.Txn) error {
	item, err := txn.Get(b.key)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	b.data, err = item.ValueCopy(nil)
	if err != nil {
		return fmt.Errorf("failed to copy value: %w", err)
	}

	return nil
}

func (b *BlockObj) exists(txn *badger.Txn) (bool, error) {
	_, err := txn.Get(b.key)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get item: %w", err)
	}

	return true, nil
}

func (b *BlockObj) write(txn *badger.Txn) error {
	return txn.Set(b.key, b.data)
}

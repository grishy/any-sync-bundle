package lightfilenodestore

import (
	"bytes"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
)

const kPrefixLinkFileBlock = kPrefixFileNode + kSeparator + "lb"

func keyLinkFileBlock(spaceId string, fileId string, k cid.Cid) []byte {
	var buf bytes.Buffer
	buf.Grow(len(kPrefixLinkFileBlock) + len(kSeparator) + len(spaceId) + len(kSeparator) + len(fileId) + len(kSeparator) + len(k.String()))

	buf.WriteString(kPrefixLinkFileBlock)
	buf.WriteString(kSeparator)
	buf.WriteString(spaceId)
	buf.WriteString(kSeparator)
	buf.WriteString(fileId)
	buf.WriteString(kSeparator)
	buf.WriteString(k.String())

	return buf.Bytes()
}

type LinkFileBlockObj struct {
	key []byte

	// Key parts
	spaceID string
	fileID  string
	cid     cid.Cid
}

func NewLinkFileBlockObj(spaceID, fileID string, k cid.Cid) *LinkFileBlockObj {
	return &LinkFileBlockObj{
		key:     keyLinkFileBlock(spaceID, fileID, k),
		spaceID: spaceID,
		fileID:  fileID,
		cid:     k,
	}
}

func (l *LinkFileBlockObj) exists(txn *badger.Txn) (bool, error) {
	_, err := txn.Get(l.key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to get item: %w", err)
	}
	return true, nil
}

func (l *LinkFileBlockObj) write(txn *badger.Txn) error {
	return txn.Set(l.key, nil)
}

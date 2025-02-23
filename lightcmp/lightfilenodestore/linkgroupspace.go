package lightfilenodestore

import (
	"bytes"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

const kPrefixLinkGroupSpace = kPrefixFileNode + kSeparator + "gs"

func keyLinkGroupSpace(groupId string, spaceId string) []byte {
	var buf bytes.Buffer
	buf.Grow(len(kPrefixLinkGroupSpace) + len(kSeparator) + len(groupId) + len(kSeparator) + len(spaceId))

	buf.WriteString(kPrefixLinkGroupSpace)
	buf.WriteString(kSeparator)
	buf.WriteString(groupId)
	buf.WriteString(kSeparator)
	buf.WriteString(spaceId)

	return buf.Bytes()
}

type LinkGroupSpaceObj struct {
	key []byte

	// Key parts
	groupID string
	spaceID string
}

func NewLinkGroupSpaceObj(groupID, spaceID string) *LinkGroupSpaceObj {
	return &LinkGroupSpaceObj{
		key:     keyLinkGroupSpace(groupID, spaceID),
		groupID: groupID,
		spaceID: spaceID,
	}
}

func (l *LinkGroupSpaceObj) exists(txn *badger.Txn) (bool, error) {
	_, err := txn.Get(l.key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to get item: %w", err)
	}
	return true, nil
}

func (l *LinkGroupSpaceObj) write(txn *badger.Txn) error {
	return txn.Set(l.key, nil)
}

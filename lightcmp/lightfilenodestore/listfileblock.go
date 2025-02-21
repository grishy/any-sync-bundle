package lightfilenodestore

import "github.com/ipfs/go-cid"

const kPrefixLinkFileBlock = kPrefixFileNode + kSeparator + "l"

func keyLinkFileBlock(spaceId string, fileId string, k cid.Cid) []byte {
	return []byte(kPrefixLinkFileBlock + spaceId + kSeparator + fileId + kSeparator + k.String())
}

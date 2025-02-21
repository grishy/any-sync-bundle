package lightfilenodestore

const kPrefixFile = kPrefixFileNode + kSeparator + "f"

func keyFile(spaceId string, fileId string) []byte {
	return []byte(kPrefixFile + spaceId + kSeparator + fileId)
}

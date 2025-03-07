package lightfilenoderpc

import (
	"fmt"

	"github.com/ipfs/go-cid"
)

// cidsToStrings converts a slice of CID byte arrays to readable strings for logging
func cidsToStrings(cids ...[]byte) []string {
	if len(cids) == 0 {
		return nil
	}
	strs := make([]string, 0, len(cids))
	for _, b := range cids {
		c, err := cid.Cast(b)
		if err != nil {
			strs = append(strs, fmt.Sprintf("invalid-cid-%x", b))
			continue
		}
		strs = append(strs, c.String())
	}
	return strs
}

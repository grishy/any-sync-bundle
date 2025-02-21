package lightfilenoderpc

import (
	"github.com/ipfs/go-cid"
)

// convertCids converts byte slices to CIDs, deduplicating if multiple CIDs are provided
func convertCids(bCids [][]byte) []cid.Cid {
	cids := make([]cid.Cid, 0, len(bCids))

	// Only create dedup map if we have multiple CIDs
	var seen map[string]struct{}
	if len(bCids) > 1 {
		seen = make(map[string]struct{})
	}

	for _, cidBytes := range bCids {
		c, err := cid.Cast(cidBytes)
		if err != nil {
			continue
		}

		// Skip if we've seen this CID before
		if seen != nil {
			key := c.KeyString()
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
		}

		cids = append(cids, c)
	}

	return cids
}

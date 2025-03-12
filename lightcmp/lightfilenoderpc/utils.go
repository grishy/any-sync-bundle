package lightfilenoderpc

import (
	"fmt"

	"github.com/ipfs/go-cid"
)

func convertCids(bCids [][]byte) (cids []cid.Cid, err error) {
	cids = make([]cid.Cid, 0, len(bCids))
	var uniqMap map[string]struct{}

	if len(bCids) > 1 {
		uniqMap = make(map[string]struct{})
	}

	for _, cd := range bCids {
		c, errCast := cid.Cast(cd)
		if errCast != nil {
			return nil, fmt.Errorf("failed to cast CID='%s': %w", string(cd), errCast)
		}

		if uniqMap != nil {
			if _, ok := uniqMap[c.KeyString()]; ok {
				continue
			} else {
				uniqMap[c.KeyString()] = struct{}{}
			}
		}
		cids = append(cids, c)
	}

	return
}

package lightfilenoderpc

import (
	"github.com/ipfs/go-cid"
)

func convertCids(bCids [][]byte) (cids []cid.Cid) {
	cids = make([]cid.Cid, 0, len(bCids))
	var uniqMap map[string]struct{}

	if len(bCids) > 1 {
		uniqMap = make(map[string]struct{})
	}

	for _, cd := range bCids {
		c, err := cid.Cast(cd)
		if err == nil {
			if uniqMap != nil {
				if _, ok := uniqMap[c.KeyString()]; ok {
					continue
				} else {
					uniqMap[c.KeyString()] = struct{}{}
				}
			}
			cids = append(cids, c)
		}
	}
	return
}

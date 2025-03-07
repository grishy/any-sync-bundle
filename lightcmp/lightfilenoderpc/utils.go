package lightfilenoderpc

import (
	"github.com/anyproto/any-sync/commonfile/fileproto"
	"github.com/ipfs/go-cid"
)

// convertCids converts [][]byte CIDs to []cid.Cid, filtering out duplicates
func (r *lightFileNodeRpc) convertCids(bCids [][]byte) (cids []cid.Cid) {
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

// checkCIDExists if the CID exists in space or in general in storage or not
func (r *lightFileNodeRpc) checkCIDExists(spaceId string, k cid.Cid) (status fileproto.AvailabilityStatus) {
	// Check if the CID exists in space
	if spaceId != "" {
		exist := r.srvIndex.HasCIDInSpace(spaceId, k)
		if exist {
			return fileproto.AvailabilityStatus_ExistsInSpace
		}
	}

	// Or check if the CID exists in storage at all
	// Create HasBlock method in storage that returns true if block exists
	exist := r.srvIndex.HadCID(k)
	if exist {
		return fileproto.AvailabilityStatus_Exists
	}

	return fileproto.AvailabilityStatus_NotExists
}

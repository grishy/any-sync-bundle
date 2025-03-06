package lightfilenoderpc

import (
	"github.com/anyproto/any-sync/commonfile/fileproto"
	"github.com/ipfs/go-cid"
)

// checkCIDExists if the CID exists in space or in general in storage or not
func (r *lightFileNodeRpc) checkCIDExists(spaceId string, k cid.Cid) (status fileproto.AvailabilityStatus, err error) {
	// Check if the CID exists in space
	exist := r.srvIndex.HasCIDInSpace(spaceId, k)
	if exist {
		return fileproto.AvailabilityStatus_ExistsInSpace, nil
	}

	// Or check if the CID exists in storage at all
	exist = r.srvIndex.HadCID(k)
	if exist {
		return fileproto.AvailabilityStatus_Exists, nil
	}

	return fileproto.AvailabilityStatus_NotExists, nil
}

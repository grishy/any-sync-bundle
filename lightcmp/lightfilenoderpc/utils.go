package lightfilenoderpc

import (
	"fmt"

	"github.com/anyproto/any-sync/commonfile/fileproto"
	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
)

// checkCIDExists if the CID exists in space or in general in storage or not
func (r *lightFileNodeRpc) checkCIDExists(txn *badger.Txn, spaceId string, k cid.Cid) (status fileproto.AvailabilityStatus, err error) {
	// Check if the CID exists in space
	exist, err := r.store.HasCIDInSpace(txn, spaceId, k)
	if err != nil {
		return status, fmt.Errorf("failed to check existence in space CID='%s': %w", spaceId, err)
	}

	if exist {
		return fileproto.AvailabilityStatus_ExistsInSpace, nil
	}

	// Or check if the CID exists in storage at all
	exist, err = r.store.HadCID(txn, k)
	if err != nil {
		return status, fmt.Errorf("failed to check CID existence: %w", err)
	}

	if exist {
		return fileproto.AvailabilityStatus_Exists, nil
	}

	return fileproto.AvailabilityStatus_NotExists, nil
}

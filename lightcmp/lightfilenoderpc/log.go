package lightfilenoderpc

import (
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"
)

// TODO: Remove this?
func cidsToStrings(cids ...[]byte) []string {
	strs := make([]string, len(cids))
	for i, b := range cids {
		c, err := cid.Cast(b)
		if err != nil {
			log.Warn("failed to cast cid", zap.Error(err))
			continue
		}

		strs[i] = c.String()
	}
	return strs
}

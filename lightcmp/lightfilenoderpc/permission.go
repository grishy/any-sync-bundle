package lightfilenoderpc

import (
	"context"
	"fmt"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync/commonfile/fileproto/fileprotoerr"
	"github.com/anyproto/any-sync/net/peer"
	"go.uber.org/zap"
)

func (r *lightfilenoderpc) getStoreKey(ctx context.Context, spaceId string) (storageKey index.Key, err error) {
	if spaceId == "" {
		return storageKey, fileprotoerr.ErrForbidden
	}

	ownerPubKey, err := r.acl.OwnerPubKey(ctx, spaceId)
	if err != nil {
		log.WarnCtx(ctx, "acl ownerPubKey error", zap.Error(err))
		return storageKey, fileprotoerr.ErrForbidden
	}

	storageKey = index.Key{
		GroupId: ownerPubKey.Account(),
		SpaceId: spaceId,
	}

	return
}

func (r *lightfilenoderpc) canWrite(ctx context.Context, spaceId string) (bool, error) {
	storageKey, err := r.getStoreKey(ctx, spaceId)
	if err != nil {
		return false, err
	}

	canWrite, err := r.canWritePerm(ctx, &storageKey)
	if err != nil {
		return false, err
	}

	if !canWrite {
		return false, fileprotoerr.ErrForbidden
	}

	canWrite, err = r.checkLimits(ctx, &storageKey)
	if err != nil {
		return false, err
	}

	if !canWrite {
		return false, fileprotoerr.ErrSpaceLimitExceeded
	}

	return canWrite, nil
}

func (r *lightfilenoderpc) canWritePerm(ctx context.Context, storageKey *index.Key) (bool, error) {
	identity, err := peer.CtxPubKey(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get identity: %w", err)
	}

	// if not owner, check permissions
	if identity.Account() != storageKey.GroupId {
		permissions, err := r.acl.Permissions(ctx, identity, storageKey.SpaceId)
		if err != nil {
			return false, fmt.Errorf("failed to get permissions: %w", err)
		}

		if !permissions.CanWrite() {
			return false, nil
		}
	}

	return true, nil
}

func (r *lightfilenoderpc) checkLimits(ctx context.Context, storageKey *index.Key) (bool, error) {
	// TODO: Implement this

	// 	if err = r.index.CheckLimits(ctx, storageKey); err != nil {
	// 		if errors.Is(err, index.ErrLimitExceed) {
	// 			return storageKey, fileprotoerr.ErrSpaceLimitExceeded
	// 		} else {
	// 			log.WarnCtx(ctx, "check limit error", zap.Error(err))
	// 			return storageKey, fileprotoerr.ErrUnexpected
	// 		}
	// 	}

	// // isolated space
	// if entry.space.Limit != 0 {
	// 	if entry.space.Size_ >= entry.space.Limit {
	// 		return ErrLimitExceed
	// 	}
	// 	return
	// }

	// // group limit
	// if entry.group.Size_ >= entry.group.Limit {
	// 	return ErrLimitExceed
	// }
	return false, nil
}

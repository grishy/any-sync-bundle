package lightfilenoderpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync/commonfile/fileproto/fileprotoerr"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

func (r *lightFileNodeRpc) resolveStoreKey(ctx context.Context, spaceID string) (key index.Key, err error) {
	if spaceID == "" {
		return key, fileprotoerr.ErrForbidden
	}

	ownerPubKey, err := r.acl.OwnerPubKey(ctx, spaceID)
	if err != nil {
		log.WarnCtx(ctx, "failed to get owner public key", zap.Error(err))
		return key, fileprotoerr.ErrForbidden
	}

	key = index.Key{
		GroupId: ownerPubKey.Account(),
		SpaceId: spaceID,
	}
	return key, nil
}

func (r *lightFileNodeRpc) canWrite(ctx context.Context, spaceID string) (bool, error) {
	storageKey, err := r.resolveStoreKey(ctx, spaceID)
	if err != nil {
		return false, err
	}

	hasPermission, err := r.canWritePerm(ctx, &storageKey)
	if err != nil {
		return false, err
	}
	if !hasPermission {
		return false, fileprotoerr.ErrForbidden
	}

	hasSpace, err := r.hasEnoughSpace(ctx, &storageKey)
	if err != nil {
		return false, err
	}
	if !hasSpace {
		return false, fileprotoerr.ErrSpaceLimitExceeded
	}

	return true, nil
}

func (r *lightFileNodeRpc) canWritePerm(ctx context.Context, storageKey *index.Key) (bool, error) {
	identity, err := peer.CtxPubKey(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get identity: %w", err)
	}

	// Owner has full permissions
	if identity.Account() == storageKey.GroupId {
		return true, nil
	}

	permissions, err := r.acl.Permissions(ctx, identity, storageKey.SpaceId)
	if err != nil {
		return false, fmt.Errorf("failed to get permissions: %w", err)
	}

	return permissions.CanWrite(), nil
}

func (r *lightFileNodeRpc) hasEnoughSpace(ctx context.Context, storageKey *index.Key) (bool, error) {
	errLimitExceeded := errors.New("storage limit exceeded")

	err := r.store.TxView(func(txn *badger.Txn) error {
		space, err := r.store.GetSpace(txn, storageKey.SpaceId)
		if err != nil {
			return fmt.Errorf("failed to get space: %w", err)
		}

		group, err := r.store.GetGroup(txn, storageKey.GroupId)
		if err != nil {
			return fmt.Errorf("failed to get group: %w", err)
		}

		// Check space-specific limit first (non-isolated space)
		if space.LimitBytes() != 0 {
			if space.TotalUsageBytes() >= space.LimitBytes() {
				return errLimitExceeded
			}
			return nil
		}

		// And also check group limit isolated spaces
		// Isolated space - space with limit
		if group.TotalUsageBytes() >= group.LimitBytes() {
			return errLimitExceeded
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, errLimitExceeded) {
			return false, fileprotoerr.ErrSpaceLimitExceeded
		}
		log.WarnCtx(ctx, "failed to check storage limits", zap.Error(err))
		return false, fileprotoerr.ErrUnexpected
	}

	return true, nil
}

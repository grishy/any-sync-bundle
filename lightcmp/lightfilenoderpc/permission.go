package lightfilenoderpc

import (
	"context"
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
		log.Warn("failed to get owner public key", zap.Error(err))
		return key, fileprotoerr.ErrForbidden
	}

	key = index.Key{
		GroupId: ownerPubKey.Account(),
		SpaceId: spaceID,
	}

	return key, nil
}

func (r *lightFileNodeRpc) canWriteWithLimit(ctx context.Context, txn *badger.Txn, spaceID string) error {
	storageKey, err := r.resolveStoreKey(ctx, spaceID)
	if err != nil {
		return err
	}

	if errPerm := r.hasWritePerm(ctx, &storageKey); errPerm != nil {
		return errPerm
	}

	if errSpace := r.hasEnoughSpace(txn, &storageKey); errSpace != nil {
		return errSpace
	}

	return nil
}

func (r *lightFileNodeRpc) hasWritePerm(ctx context.Context, storageKey *index.Key) error {
	identity, err := peer.CtxPubKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to get identity: %w", err)
	}

	// Owner has full permissions
	if identity.Account() == storageKey.GroupId {
		return nil
	}

	permissions, err := r.acl.Permissions(ctx, identity, storageKey.SpaceId)
	if err != nil {
		return fmt.Errorf("failed to get permissions: %w", err)
	}

	if !permissions.CanWrite() {
		return fileprotoerr.ErrForbidden
	}

	return nil
}

func (r *lightFileNodeRpc) hasEnoughSpace(txn *badger.Txn, storageKey *index.Key) error {
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
			return fileprotoerr.ErrSpaceLimitExceeded
		}
		return nil
	}

	// And also check group limit isolated spaces
	// Isolated space - space with limit
	if group.TotalUsageBytes() >= group.LimitBytes() {
		return fileprotoerr.ErrSpaceLimitExceeded
	}

	return nil
}

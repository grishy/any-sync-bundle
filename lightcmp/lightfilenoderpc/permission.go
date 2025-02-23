package lightfilenoderpc

import (
	"context"
	"fmt"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync/commonfile/fileproto/fileprotoerr"
	"github.com/anyproto/any-sync/commonspace/object/acl/aclrecordproto"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

func (r *lightFileNodeRpc) resolveStoreKey(ctx context.Context, spaceID string) (index.Key, error) {
	if spaceID == "" {
		return index.Key{}, fileprotoerr.ErrForbidden
	}

	ownerPubKey, err := r.acl.OwnerPubKey(ctx, spaceID)
	if err != nil {
		log.Warn("failed to get owner public key", zap.Error(err))
		return index.Key{}, fileprotoerr.ErrForbidden
	}

	return index.Key{
		GroupId: ownerPubKey.Account(),
		SpaceId: spaceID,
	}, nil
}

func (r *lightFileNodeRpc) canRead(ctx context.Context, spaceID string) error {
	storageKey, err := r.resolveStoreKey(ctx, spaceID)
	if err != nil {
		return err
	}

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

	// TODO: Create a PR to add CanRead to any-sync
	switch aclrecordproto.AclUserPermissions(permissions) {
	case aclrecordproto.AclUserPermissions_Reader,
		aclrecordproto.AclUserPermissions_Writer,
		aclrecordproto.AclUserPermissions_Admin,
		aclrecordproto.AclUserPermissions_Owner:
		return nil
	default:
		return fileprotoerr.ErrForbidden
	}
}

func (r *lightFileNodeRpc) canWrite(ctx context.Context, txn *badger.Txn, spaceID string) error {
	storageKey, err := r.resolveStoreKey(ctx, spaceID)
	if err != nil {
		return err
	}

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

	return r.hasEnoughSpace(txn, &storageKey)
}

func (r *lightFileNodeRpc) hasEnoughSpace(txn *badger.Txn, storageKey *index.Key) error {
	space, err := r.store.GetSpace(txn, storageKey.SpaceId)
	if err != nil {
		return fmt.Errorf("failed to get space: %w", err)
	}

	// For non-isolated spaces, check space-specific limit first
	if spaceLimit := space.LimitBytes(); spaceLimit > 0 {
		if space.TotalUsageBytes() >= spaceLimit {
			return fileprotoerr.ErrSpaceLimitExceeded
		}
		return nil
	}

	// For isolated spaces, check group limit
	group, err := r.store.GetGroup(txn, storageKey.GroupId)
	if err != nil {
		return fmt.Errorf("failed to get group: %w", err)
	}

	if group.TotalUsageBytes() >= group.LimitBytes() {
		return fileprotoerr.ErrSpaceLimitExceeded
	}

	return nil
}

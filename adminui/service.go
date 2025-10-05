package adminui

import (
	"context"
	"fmt"
	"time"

	"github.com/anyproto/any-sync-coordinator/accountlimit"
	"github.com/anyproto/any-sync-coordinator/acleventlog"
	"github.com/anyproto/any-sync-coordinator/db"
	"github.com/anyproto/any-sync-coordinator/spacestatus"
	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/nodeconf"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/grishy/any-sync-bundle/adminui/admintypes"
)

// Service provides business logic for admin UI.
type Service struct {
	coordinatorDB db.Database
	accountLimit  accountlimit.AccountLimit
	spaceStatus   spacestatus.SpaceStatus
	filenodeIndex index.Index
	aclEventLog   acleventlog.AclEventLog
	nodeConf      nodeconf.NodeConf
	metric        metric.Metric
}

// newService creates a new service with direct dependencies.
func newService(
	coordinatorDB db.Database,
	accountLimit accountlimit.AccountLimit,
	spaceStatus spacestatus.SpaceStatus,
	filenodeIndex index.Index,
	aclEventLog acleventlog.AclEventLog,
	nodeConf nodeconf.NodeConf,
	m metric.Metric,
) *Service {
	return &Service{
		coordinatorDB: coordinatorDB,
		accountLimit:  accountLimit,
		spaceStatus:   spaceStatus,
		filenodeIndex: filenodeIndex,
		aclEventLog:   aclEventLog,
		nodeConf:      nodeConf,
		metric:        m,
	}
}

// GetUserInfo retrieves comprehensive user information.
func (s *Service) GetUserInfo(ctx context.Context, identity string) (*admintypes.UserInfo, error) {
	// Get account limits from coordinator
	limits, err := s.accountLimit.GetLimits(ctx, identity)
	if err != nil {
		return nil, fmt.Errorf("get limits: %w", err)
	}

	// Get usage info from filenode
	groupInfo, err := s.filenodeIndex.GroupInfo(ctx, identity)
	if err != nil {
		return nil, fmt.Errorf("get group info: %w", err)
	}

	// Get spaces from MongoDB
	cursor, err := s.coordinatorDB.Db().Collection("spaces").Find(ctx, bson.M{
		"identity": identity,
		"status":   admintypes.SpaceStatusCreated,
	})
	if err != nil {
		return nil, fmt.Errorf("find spaces: %w", err)
	}
	defer cursor.Close(ctx)

	var spaces []admintypes.SpaceInfo
	for cursor.Next(ctx) {
		var space admintypes.SpaceInfo
		if decodeErr := cursor.Decode(&space); decodeErr != nil {
			return nil, fmt.Errorf("decode space: %w", decodeErr)
		}

		// Enrich with usage data from filenode
		spaceKey := index.Key{
			GroupId: identity,
			SpaceId: space.SpaceID,
		}
		if spaceInfo, spaceErr := s.filenodeIndex.SpaceInfo(ctx, spaceKey); spaceErr == nil {
			space.BytesUsage = spaceInfo.BytesUsage
			space.CidsCount = spaceInfo.CidsCount
			space.FileCount = spaceInfo.FileCount
		}

		spaces = append(spaces, space)
	}

	// Calculate usage percentage
	usagePercent := admintypes.CalculateUsagePercent(groupInfo.BytesUsage, limits.FileStorageBytes)

	return &admintypes.UserInfo{
		Identity:     identity,
		Limits:       limits,
		GroupInfo:    groupInfo,
		Spaces:       spaces,
		SpaceCount:   len(spaces),
		UsagePercent: usagePercent,
	}, nil
}

// SetUserQuota updates user storage quota.
func (s *Service) SetUserQuota(ctx context.Context, req admintypes.QuotaUpdateRequest) error {
	limits := accountlimit.Limits{
		Identity:          req.Identity,
		FileStorageBytes:  req.StorageBytes,
		SpaceMembersRead:  req.MembersRead,
		SpaceMembersWrite: req.MembersWrite,
		SharedSpacesLimit: req.SharedSpaces,
		Reason:            req.Reason,
	}

	return s.accountLimit.SetLimits(ctx, limits)
}

// GetSystemStats retrieves overall system statistics.
func (s *Service) GetSystemStats(ctx context.Context) (*admintypes.SystemStats, error) {
	stats := &admintypes.SystemStats{}

	// Count total unique users
	userIdentities, err := s.coordinatorDB.Db().Collection("spaces").
		Distinct(ctx, "identity", bson.M{"status": admintypes.SpaceStatusCreated})
	if err == nil {
		stats.TotalUsers = len(userIdentities)
	}

	// Count total active spaces
	activeCount, err := s.coordinatorDB.Db().Collection("spaces").
		CountDocuments(ctx, bson.M{"status": admintypes.SpaceStatusCreated})
	if err == nil {
		stats.ActiveSpaces = int(activeCount)
	}

	// Count total spaces (all statuses)
	totalCount, err := s.coordinatorDB.Db().Collection("spaces").
		CountDocuments(ctx, bson.M{})
	if err == nil {
		stats.TotalSpaces = int(totalCount)
	}

	// Count pending deletions
	pendingCount, err := s.coordinatorDB.Db().Collection("spaces").
		CountDocuments(ctx, bson.M{"status": admintypes.SpaceStatusDeletionPending})
	if err == nil {
		stats.PendingDelete = int(pendingCount)
	}

	return stats, nil
}

// GetDeletionLog retrieves recent deletion entries.
func (s *Service) GetDeletionLog(ctx context.Context) ([]admintypes.DeletionEntry, error) {
	cursor, err := s.coordinatorDB.Db().Collection("deletionLog").
		Find(ctx, bson.M{}, options.Find().
			SetSort(bson.D{{Key: "_id", Value: -1}}).
			SetLimit(int64(admintypes.PageSize)))
	if err != nil {
		return nil, fmt.Errorf("query deletion log: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []admintypes.DeletionEntry
	for cursor.Next(ctx) {
		var entry admintypes.DeletionEntry
		if decodeErr := cursor.Decode(&entry); decodeErr != nil {
			return nil, fmt.Errorf("decode deletion entry: %w", decodeErr)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetAllUsers retrieves all users with pagination using efficient MongoDB aggregation.
func (s *Service) GetAllUsers(ctx context.Context, page int) ([]admintypes.UserSummary, int, error) {
	if page < 1 {
		page = 1
	}

	// Build aggregation pipeline for efficient pagination
	pipeline := bson.A{
		// Step 1: Match only active spaces
		bson.M{"$match": bson.M{"status": admintypes.SpaceStatusCreated}},

		// Step 2: Group by identity and count spaces
		bson.M{"$group": bson.M{
			"_id":        "$identity",
			"spaceCount": bson.M{"$sum": 1},
		}},

		// Step 3: Sort for consistent pagination
		bson.M{"$sort": bson.M{"_id": 1}},

		// Step 4: Use $facet to get total count and paginated data in one query
		bson.M{"$facet": bson.M{
			"metadata": bson.A{
				bson.M{"$count": "total"},
			},
			"data": bson.A{
				bson.M{"$skip": (page - 1) * admintypes.PageSize},
				bson.M{"$limit": admintypes.PageSize},
			},
		}},
	}

	cursor, err := s.coordinatorDB.Db().Collection("spaces").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, fmt.Errorf("aggregate users: %w", err)
	}
	defer cursor.Close(ctx)

	// Parse aggregation result
	var results []bson.M
	if decodeErr := cursor.All(ctx, &results); decodeErr != nil {
		return nil, 0, fmt.Errorf("decode aggregation: %w", decodeErr)
	}

	if len(results) == 0 {
		return []admintypes.UserSummary{}, 0, nil
	}

	result := results[0]

	// Extract total count
	total := 0
	if metadata, metaOk := result["metadata"].(bson.A); metaOk && len(metadata) > 0 {
		if metaDoc, docOk := metadata[0].(bson.M); docOk {
			if totalVal, valOk := metaDoc["total"].(int32); valOk {
				total = int(totalVal)
			}
		}
	}

	// Extract paginated user data
	data, ok := result["data"].(bson.A)
	if !ok {
		return []admintypes.UserSummary{}, total, nil
	}

	users := make([]admintypes.UserSummary, 0, admintypes.PageSize)
	for _, item := range data {
		doc, itemOk := item.(bson.M)
		if !itemOk {
			continue
		}

		identity, _ := doc["_id"].(string)
		spaceCount := 0
		if sc, scOk := doc["spaceCount"].(int32); scOk {
			spaceCount = int(sc)
		}

		// Enrich with limits and storage info
		limits, _ := s.accountLimit.GetLimits(ctx, identity)
		groupInfo, _ := s.filenodeIndex.GroupInfo(ctx, identity)

		users = append(users, admintypes.UserSummary{
			Identity:     identity,
			StorageUsed:  groupInfo.BytesUsage,
			StorageLimit: limits.FileStorageBytes,
			UsagePercent: admintypes.CalculateUsagePercent(groupInfo.BytesUsage, limits.FileStorageBytes),
			SpaceCount:   spaceCount,
			LastUpdated:  formatTime(limits.UpdatedTime),
		})
	}

	return users, total, nil
}

// GetSpacesByIdentity retrieves all spaces for a given identity.
func (s *Service) GetSpacesByIdentity(ctx context.Context, identity string) ([]admintypes.SpaceInfo, error) {
	cursor, err := s.coordinatorDB.Db().Collection("spaces").Find(ctx, bson.M{
		"identity": identity,
	})
	if err != nil {
		return nil, fmt.Errorf("find spaces: %w", err)
	}
	defer cursor.Close(ctx)

	var spaces []admintypes.SpaceInfo
	for cursor.Next(ctx) {
		var space admintypes.SpaceInfo
		if decodeErr := cursor.Decode(&space); decodeErr != nil {
			return nil, fmt.Errorf("decode space: %w", decodeErr)
		}

		// Enrich with usage data
		spaceKey := index.Key{
			GroupId: identity,
			SpaceId: space.SpaceID,
		}
		if spaceInfo, spaceErr := s.filenodeIndex.SpaceInfo(ctx, spaceKey); spaceErr == nil {
			space.BytesUsage = spaceInfo.BytesUsage
			space.CidsCount = spaceInfo.CidsCount
			space.FileCount = spaceInfo.FileCount
		}

		spaces = append(spaces, space)
	}

	return spaces, nil
}

// GetAllSpacesWithPagination retrieves all spaces with pagination and filtering.
func (s *Service) GetAllSpacesWithPagination(
	ctx context.Context,
	page int,
	filterType, filterStatus int,
) ([]admintypes.SpaceInfo, int, error) {
	if page < 1 {
		page = 1
	}

	// Build filter
	filter := bson.M{}
	if filterType >= 0 {
		filter["type"] = filterType
	}
	if filterStatus >= 0 {
		filter["status"] = filterStatus
	}

	// Count total
	total, err := s.coordinatorDB.Db().Collection("spaces").CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count spaces: %w", err)
	}

	// Fetch spaces with pagination
	opts := options.Find().
		SetSkip(int64((page - 1) * admintypes.PageSize)).
		SetLimit(int64(admintypes.PageSize))

	cursor, err := s.coordinatorDB.Db().Collection("spaces").Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("find spaces: %w", err)
	}
	defer cursor.Close(ctx)

	var spaces []admintypes.SpaceInfo
	for cursor.Next(ctx) {
		var space admintypes.SpaceInfo
		if decodeErr := cursor.Decode(&space); decodeErr != nil {
			return nil, 0, fmt.Errorf("decode space: %w", decodeErr)
		}

		// Enrich with usage data
		spaceKey := index.Key{
			GroupId: space.Identity,
			SpaceId: space.SpaceID,
		}
		if spaceInfo, spaceErr := s.filenodeIndex.SpaceInfo(ctx, spaceKey); spaceErr == nil {
			space.BytesUsage = spaceInfo.BytesUsage
			space.CidsCount = spaceInfo.CidsCount
			space.FileCount = spaceInfo.FileCount
		}

		spaces = append(spaces, space)
	}

	return spaces, int(total), nil
}

// ToggleSpaceShareability toggles space shareability.
func (s *Service) ToggleSpaceShareability(ctx context.Context, req admintypes.ShareabilityRequest) error {
	// Get current user limits to check shared space limit
	limits, err := s.accountLimit.GetLimits(ctx, req.Identity)
	if err != nil {
		return fmt.Errorf("get limits: %w", err)
	}

	if req.MakeSharable {
		// Get the space status to determine its type
		status, statusErr := s.spaceStatus.Status(ctx, req.SpaceID)
		if statusErr != nil {
			return fmt.Errorf("get space status: %w", statusErr)
		}

		return s.spaceStatus.MakeShareable(ctx, req.SpaceID, status.Type, limits.SharedSpacesLimit)
	}
	return s.spaceStatus.MakeUnshareable(ctx, req.SpaceID)
}

// GetACLEvents retrieves ACL event log entries for a user.
func (s *Service) GetACLEvents(
	ctx context.Context,
	identity string,
	page int,
) ([]admintypes.ACLEventEntry, int, error) {
	filter := bson.M{"owner": identity}
	return s.getACLEventsWithFilter(ctx, filter, page)
}

// GetNetworkConfig retrieves the network configuration from the nodeconf component.
func (s *Service) GetNetworkConfig(_ context.Context) (*admintypes.NetworkConfig, error) {
	// Get the configuration from the nodeconf component (loaded from YAML, not MongoDB)
	conf := s.nodeConf.Configuration()

	// Convert from nodeconf.Configuration to admintypes.NetworkConfig
	nodes := make([]admintypes.NetworkNode, len(conf.Nodes))
	for i, node := range conf.Nodes {
		// Convert NodeType slice to string slice
		types := make([]string, len(node.Types))
		for j, t := range node.Types {
			types[j] = string(t)
		}

		nodes[i] = admintypes.NetworkNode{
			PeerID:    node.PeerId,
			Addresses: node.Addresses,
			Types:     types,
		}
	}

	return &admintypes.NetworkConfig{
		NetworkID: conf.NetworkId,
		Nodes:     nodes,
	}, nil
}

// GetAllACLEvents retrieves all ACL events with pagination (not filtered by user).
func (s *Service) GetAllACLEvents(ctx context.Context, page int) ([]admintypes.ACLEventEntry, int, error) {
	return s.getACLEventsWithFilter(ctx, bson.M{}, page)
}

// GetACLEventsBySpace retrieves ACL events for a specific space.
func (s *Service) GetACLEventsBySpace(
	ctx context.Context,
	spaceID string,
	page int,
) ([]admintypes.ACLEventEntry, int, error) {
	filter := bson.M{"spaceId": spaceID}
	return s.getACLEventsWithFilter(ctx, filter, page)
}

// getACLEventsWithFilter is a helper function to retrieve ACL events with a filter.
func (s *Service) getACLEventsWithFilter(
	ctx context.Context,
	filter bson.M,
	page int,
) ([]admintypes.ACLEventEntry, int, error) {
	if page < 1 {
		page = 1
	}

	// Count total
	total, err := s.coordinatorDB.Db().Collection("aclEventLog").CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count acl events: %w", err)
	}

	// Fetch events with pagination
	opts := options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}}). // Sort by ID descending (newest first)
		SetSkip(int64((page - 1) * admintypes.PageSize)).
		SetLimit(int64(admintypes.PageSize))

	cursor, err := s.coordinatorDB.Db().Collection("aclEventLog").Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("find acl events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []admintypes.ACLEventEntry
	for cursor.Next(ctx) {
		var event admintypes.ACLEventEntry
		if decodeErr := cursor.Decode(&event); decodeErr != nil {
			return nil, 0, fmt.Errorf("decode acl event: %w", decodeErr)
		}

		// Convert timestamp to time
		event.Time = time.Unix(event.Timestamp, 0)

		events = append(events, event)
	}

	return events, int(total), nil
}

// GetStorageStats retrieves system-wide storage statistics from filenode.
func (s *Service) GetStorageStats(ctx context.Context) (*admintypes.StorageStats, error) {
	stats := &admintypes.StorageStats{}

	// Get all unique identities from spaces
	identities, err := s.coordinatorDB.Db().Collection("spaces").
		Distinct(ctx, "identity", bson.M{"status": admintypes.SpaceStatusCreated})
	if err != nil {
		return nil, fmt.Errorf("get identities: %w", err)
	}

	stats.TotalUsers = len(identities)

	// Count total active spaces
	spaceCount, err := s.coordinatorDB.Db().Collection("spaces").
		CountDocuments(ctx, bson.M{"status": admintypes.SpaceStatusCreated})
	if err != nil {
		return nil, fmt.Errorf("count spaces: %w", err)
	}
	stats.TotalSpaces = int(spaceCount)

	// Aggregate storage stats from all users
	for _, identity := range identities {
		identityStr, ok := identity.(string)
		if !ok {
			continue
		}

		groupInfo, groupErr := s.filenodeIndex.GroupInfo(ctx, identityStr)
		if groupErr != nil {
			continue
		}

		stats.TotalBytesUsed += groupInfo.BytesUsage
		stats.TotalCidsCount += groupInfo.CidsCount
	}

	// Get total file count from all spaces
	cursor, err := s.coordinatorDB.Db().Collection("spaces").Find(ctx, bson.M{
		"status": admintypes.SpaceStatusCreated,
	})
	if err != nil {
		return stats, nil // Return partial stats
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var space admintypes.SpaceInfo
		if decodeErr := cursor.Decode(&space); decodeErr != nil {
			continue
		}

		spaceKey := index.Key{
			GroupId: space.Identity,
			SpaceId: space.SpaceID,
		}
		if spaceInfo, spaceErr := s.filenodeIndex.SpaceInfo(ctx, spaceKey); spaceErr == nil {
			stats.TotalFileCount += spaceInfo.FileCount
		}
	}

	return stats, nil
}

// formatTime formats time for display.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}

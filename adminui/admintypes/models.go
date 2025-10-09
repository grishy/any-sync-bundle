package admintypes

import (
	"fmt"
	"time"

	"github.com/anyproto/any-sync-coordinator/accountlimit"
	"github.com/anyproto/any-sync-filenode/index"
)

// PageSize is the fixed page size for all listings.
const PageSize = 50

// UserInfo represents comprehensive user information.
type UserInfo struct {
	Identity     string
	Limits       accountlimit.Limits
	GroupInfo    index.GroupInfo
	Spaces       []SpaceInfo
	SpaceCount   int
	UsagePercent float64
}

// SpaceInfo represents information about a single space.
type SpaceInfo struct {
	SpaceID     string `bson:"spaceId"`
	Identity    string
	Type        int // SpaceType: 0=Personal, 1=Tech, 2=Regular, 3=Chat
	Status      int // 0=Created, 1=DeletionPending, 2=DeletionStarted, 3=Deleted
	IsShareable bool
	BytesUsage  uint64
	CidsCount   uint64
	FileCount   uint32
}

// SystemStats represents overall system statistics.
type SystemStats struct {
	TotalUsers    int
	TotalSpaces   int
	TotalStorage  uint64
	ActiveSpaces  int
	PendingDelete int
}

// QuotaUpdateRequest represents a quota update request.
type QuotaUpdateRequest struct {
	Identity     string
	StorageBytes uint64
	MembersRead  uint32
	MembersWrite uint32
	SharedSpaces uint32
	Reason       string
}

// DeletionEntry represents an entry in the deletion log.
type DeletionEntry struct {
	ID        string    `bson:"_id"`
	SpaceID   string    `bson:"spaceId"`
	FileGroup string    `bson:"fileGroup"`
	Status    int       // 0=Ok, 1=RemovePrepare, 2=Remove
	Timestamp time.Time `bson:"timestamp"`
}

// UserSummary represents a summary of user information for the users list.
type UserSummary struct {
	Identity     string
	StorageUsed  uint64
	StorageLimit uint64
	UsagePercent float64
	SpaceCount   int
	LastUpdated  string
}

// Space status constants.
const (
	SpaceStatusCreated = iota
	SpaceStatusDeletionPending
	SpaceStatusDeletionStarted
	SpaceStatusDeleted
)

// Space type constants.
const (
	SpaceTypePersonal = iota
	SpaceTypeTech
	SpaceTypeRegular
	SpaceTypeChat
)

// Helper methods.

// SpaceTypeString returns human-readable space type.
func SpaceTypeString(t int) string {
	switch t {
	case SpaceTypePersonal:
		return "Personal"
	case SpaceTypeTech:
		return "Tech"
	case SpaceTypeRegular:
		return "Regular"
	case SpaceTypeChat:
		return "Chat"
	default:
		return unknownTypeString
	}
}

// SpaceStatusString returns human-readable space status.
func SpaceStatusString(s int) string {
	switch s {
	case SpaceStatusCreated:
		return "Active"
	case SpaceStatusDeletionPending:
		return "Deletion Pending"
	case SpaceStatusDeletionStarted:
		return "Deletion Started"
	case SpaceStatusDeleted:
		return "Deleted"
	default:
		return unknownTypeString
	}
}

// FormatBytes formats bytes into human-readable string.
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// CalculateUsagePercent calculates usage percentage.
func CalculateUsagePercent(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(used) / float64(total) * 100
}

const (
	unknownTypeString = "Unknown"
)

// ACLEventEntry represents an ACL event log entry.
type ACLEventEntry struct {
	ID          string    `bson:"_id"`
	SpaceID     string    `bson:"spaceId"`
	PeerID      string    `bson:"peerId"`
	Owner       string    `bson:"owner"`
	Timestamp   int64     `bson:"timestamp"`
	EntryType   int       `bson:"entryType"`
	ACLChangeID string    `bson:"aclChangeId,omitempty"`
	Time        time.Time // Computed from Timestamp
}

// ACL event type constants.
const (
	ACLEventTypeSpaceReceipt = iota
	ACLEventTypeSpaceShared
	ACLEventTypeSpaceUnshared
	ACLEventTypeSpaceACLAddRecord
)

// ACLEventTypeString returns human-readable ACL event type.
func ACLEventTypeString(t int) string {
	switch t {
	case ACLEventTypeSpaceReceipt:
		return "Space Receipt"
	case ACLEventTypeSpaceShared:
		return "Space Shared"
	case ACLEventTypeSpaceUnshared:
		return "Space Unshared"
	case ACLEventTypeSpaceACLAddRecord:
		return "ACL Record Added"
	default:
		return unknownTypeString
	}
}

// NetworkNode represents a node in the network topology.
type NetworkNode struct {
	PeerID    string   `yaml:"peerId"    bson:"peerId"`
	Addresses []string `yaml:"addresses" bson:"addresses"`
	Types     []string `yaml:"types"     bson:"types"`
}

// NetworkConfig represents the network configuration.
type NetworkConfig struct {
	NetworkID string        `yaml:"id"    bson:"id"`
	Nodes     []NetworkNode `yaml:"nodes" bson:"nodes"`
}

// ShareabilityRequest represents a space shareability toggle request.
type ShareabilityRequest struct {
	SpaceID      string
	Identity     string
	MakeSharable bool
}

// StorageStats represents system-wide storage statistics.
type StorageStats struct {
	TotalBytesUsed uint64
	TotalCidsCount uint64
	TotalFileCount uint32
	TotalSpaces    int
	TotalUsers     int
}

// SyncSpaceStats represents sync node statistics for a single space.
type SyncSpaceStats struct {
	SpaceID             string
	ObjectsCount        int
	DeletedObjectsCount int
	ChangesCount        int
	ACLReaders          int
	ACLWriters          int
}

// SyncStats represents aggregated sync node statistics.
type SyncStats struct {
	Spaces              []SyncSpaceStats
	TotalSpaces         int
	TotalObjects        int
	TotalDeletedObjects int
	TotalChanges        int
}

// SystemHealthStatus represents the health status of system components.
type SystemHealthStatus struct {
	RedisStatus      string
	RedisInfo        map[string]string
	MongoDBStatus    string
	ComponentsStatus map[string]string
}

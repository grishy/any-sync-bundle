package doctor

import (
	"context"
	"io"
	"time"
)

const currentReportSchema = 1

type Verdict string

const (
	VerdictHealthy                    Verdict = "bundle_healthy"
	VerdictProblemsFound              Verdict = "problems_found"
	VerdictFilesRequireClientReupload Verdict = "some_files_require_client_reupload"
)

type Summary struct {
	Groups uint64 `json:"groups"`
	Spaces uint64 `json:"spaces"`
	Files  uint64 `json:"files"`
	CIDs   uint64 `json:"cids"`
	Bytes  uint64 `json:"bytes"`
}

type HealthStatus string

const (
	StatusOK      HealthStatus = "ok"
	StatusProblem HealthStatus = "problem"
	StatusEmpty   HealthStatus = "empty"
	StatusSkipped HealthStatus = "skipped"
)

const (
	problemScopeCID     = "cid"
	problemScopeConfig  = "config"
	problemScopeFile    = "file"
	problemScopeGroup   = "group"
	problemScopeNetwork = "network"
	problemScopeRuntime = "runtime"
	problemScopeSpace   = "space"
)

type Report struct {
	GeneratedAt         time.Time     `json:"generatedAt"`
	Experimental        bool          `json:"experimental"`
	ReportSchema        int           `json:"reportSchema"`
	Build               BuildInfo     `json:"build,omitempty"`
	BundleConfigPath    string        `json:"bundleConfigPath,omitempty"`
	ClientConfigPath    string        `json:"clientConfigPath,omitempty"`
	ReportPath          string        `json:"reportPath,omitempty"`
	Config              ConfigReport  `json:"config,omitempty"`
	Runtime             RuntimeReport `json:"runtime,omitempty"`
	Network             NetworkReport `json:"network,omitempty"`
	Verdict             Verdict       `json:"verdict"`
	SuggestedNextAction string        `json:"suggestedNextAction,omitempty"`
	Summary             Summary       `json:"summary"`
	Inventory           Inventory     `json:"inventory,omitempty"`
	Problems            []Problem     `json:"problems,omitempty"`
}

type BuildInfo struct {
	Version string `json:"version,omitempty"`
	Commit  string `json:"commit,omitempty"`
	Date    string `json:"date,omitempty"`
}

type ConfigReport struct {
	ConfigID            string    `json:"configId,omitempty"`
	NetworkID           string    `json:"networkId,omitempty"`
	PeerID              string    `json:"peerId,omitempty"`
	MongoCoordinatorURI string    `json:"mongoCoordinatorUri,omitempty"`
	MongoConsensusURI   string    `json:"mongoConsensusUri,omitempty"`
	RedisURI            string    `json:"redisUri,omitempty"`
	S3                  *S3Report `json:"s3,omitempty"`
	StoragePath         string    `json:"storagePath,omitempty"`
	AdvertisedAddresses []string  `json:"advertisedAddresses,omitempty"`
}

type S3Report struct {
	Bucket         string `json:"bucket,omitempty"`
	Endpoint       string `json:"endpoint,omitempty"`
	Region         string `json:"region,omitempty"`
	ForcePathStyle bool   `json:"forcePathStyle,omitempty"`
}

type RuntimeReport struct {
	Mongo      HealthStatus `json:"mongo,omitempty"`
	Redis      HealthStatus `json:"redis,omitempty"`
	RedisBloom HealthStatus `json:"redisBloom,omitempty"`
	Storage    HealthStatus `json:"storage,omitempty"`
}

type NetworkReport struct {
	ListenTCPAddr       string       `json:"listenTcpAddr,omitempty"`
	ListenUDPAddr       string       `json:"listenUdpAddr,omitempty"`
	AdvertisedAddresses []string     `json:"advertisedAddresses,omitempty"`
	TCPListen           HealthStatus `json:"tcpListen,omitempty"`
	UDPListen           HealthStatus `json:"udpListen,omitempty"`
	Advertised          HealthStatus `json:"advertised,omitempty"`
}

type Problem struct {
	Scope       string `json:"scope"`
	ID          string `json:"id,omitempty"`
	Issue       string `json:"issue"`
	Recoverable string `json:"recoverable,omitempty"`
}

type Inventory struct {
	Groups []GroupReport `json:"groups,omitempty"`
	Spaces []SpaceReport `json:"spaces,omitempty"`
	Files  []FileReport  `json:"files,omitempty"`
}

type GroupReport struct {
	ID            string       `json:"id"`
	Spaces        uint64       `json:"spaces"`
	Files         uint64       `json:"files"`
	CIDs          uint64       `json:"cids"`
	Bytes         uint64       `json:"bytes"`
	Limit         uint64       `json:"limit,omitempty"`
	AccountLimit  uint64       `json:"accountLimit,omitempty"`
	IndexProblems uint64       `json:"indexProblems,omitempty"`
	Status        HealthStatus `json:"status"`
}

type SpaceReport struct {
	ID              string       `json:"id"`
	GroupID         string       `json:"groupId,omitempty"`
	Files           uint64       `json:"files"`
	CIDs            uint64       `json:"cids"`
	Bytes           uint64       `json:"bytes"`
	Limit           uint64       `json:"limit,omitempty"`
	MissingCIDIndex uint64       `json:"missingCidIndex,omitempty"`
	MissingBlocks   uint64       `json:"missingBlocks,omitempty"`
	CorruptBlocks   uint64       `json:"corruptBlocks,omitempty"`
	IndexProblems   uint64       `json:"indexProblems,omitempty"`
	Status          HealthStatus `json:"status"`
}

type FileReport struct {
	ID              string            `json:"id"`
	SpaceID         string            `json:"spaceId"`
	GroupID         string            `json:"groupId,omitempty"`
	Size            uint64            `json:"size"`
	CIDs            []string          `json:"cids,omitempty"`
	CIDSizes        map[string]uint64 `json:"-"`
	MissingCIDIndex []string          `json:"missingCidIndex,omitempty"`
	MissingBlocks   []string          `json:"missingBlocks,omitempty"`
	CorruptBlocks   []string          `json:"corruptBlocks,omitempty"`
}

type Runner interface {
	RunDoctor(ctx context.Context, out io.Writer) (*Report, error)
}

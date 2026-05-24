package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/index/indexproto"
	"github.com/anyproto/any-sync/accountservice"

	bundleconfig "github.com/grishy/any-sync-bundle/config"
)

func TestRuntimeRunnerWritesReportAndReturnsProblemVerdict(t *testing.T) {
	dir := t.TempDir()
	bundleConfigPath := filepath.Join(dir, "bundle-config.yml")
	clientConfigPath := filepath.Join(dir, "client-config.yml")
	generatedAt := time.Date(2026, 5, 22, 14, 33, 10, 0, time.UTC)
	cfg := validRuntimeTestConfig(t)
	writeRuntimeTestClientConfig(t, cfg, clientConfigPath)
	groupID := "account1"
	spaceID := "space1"
	fileID := "file1"
	cidMissing := "bafy-missing"
	key := index.Key{GroupId: groupID, SpaceId: spaceID}
	runner := NewRuntimeRunner(RuntimeRunnerConfig{
		BundleConfig:     cfg,
		BundleConfigPath: bundleConfigPath,
		ClientConfigPath: clientConfigPath,
		Build: BuildInfo{
			Version: "test-version",
			Commit:  "test-commit",
			Date:    "2026-05-22",
		},
		Now: func() time.Time { return generatedAt },
		LoadIndexSnapshot: func(_ context.Context) (IndexSnapshot, error) {
			return IndexSnapshot{
				Hashes: map[string]map[string]string{
					index.GroupKey(key): {
						redisIndexInfoField: mustMarshalGroupEntry(
							t,
							&indexproto.GroupEntry{GroupId: groupID, SpaceIds: []string{spaceID}},
						),
					},
					index.SpaceKey(key): {
						redisIndexInfoField: mustMarshalSpaceEntry(
							t,
							&indexproto.SpaceEntry{GroupId: groupID, FileCount: 1, CidCount: 1, Size: 10},
						),
						index.FileKey(fileID): mustMarshalFileEntry(
							t,
							&indexproto.FileEntry{Cids: []string{cidMissing}, Size: 10},
						),
					},
				},
			}, nil
		},
		ProbeBlocks: func(_ context.Context, _ Inventory) (BlockProbeResult, error) {
			return BlockProbeResult{Checked: 1}, nil
		},
		ProbeRuntime: healthyRuntimeProbe,
	})

	var out bytes.Buffer
	report, err := runner.RunDoctor(context.Background(), &out)
	if err != nil {
		t.Fatalf("RunDoctor() error = %v", err)
	}

	if report.Verdict != VerdictProblemsFound {
		t.Fatalf("verdict = %q, want %q", report.Verdict, VerdictProblemsFound)
	}
	if report.ReportPath != filepath.Join(dir, "doctor", "doctor_2026-05-22T14-33-10Z.json") {
		t.Fatalf("report path = %q", report.ReportPath)
	}

	output := out.String()
	for _, want := range []string{
		"Experimental:",
		"doctor output and JSON report schema may change between releases",
		"[7/7] Report",
		"[1/7] Config",
		"[4/7] Inventory",
		"space space1",
		"Problems:",
		"Verdict:",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}

	raw, err := os.ReadFile(report.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var written Report
	if unmarshalErr := json.Unmarshal(raw, &written); unmarshalErr != nil {
		t.Fatalf("unmarshal report: %v", unmarshalErr)
	}
	if written.Summary.Spaces != 1 {
		t.Fatalf("written spaces = %d, want 1", written.Summary.Spaces)
	}
	if written.Build.Version != "test-version" {
		t.Fatalf("written build version = %q, want test-version", written.Build.Version)
	}
	if !written.Experimental {
		t.Fatal("written experimental = false, want true")
	}
	if written.ReportSchema != currentReportSchema {
		t.Fatalf("written report schema = %d, want %d", written.ReportSchema, currentReportSchema)
	}
	if len(written.Problems) == 0 {
		t.Fatal("written problems = 0, want at least one problem")
	}
	if !problemIssuesContain(written.Problems, "global index entry is missing") {
		t.Fatalf("written problems do not include missing global index entry: %v", written.Problems)
	}
}

func TestRuntimeRunnerWritesRedactedConfigAndRuntimeDetails(t *testing.T) {
	dir := t.TempDir()
	bundleConfigPath := filepath.Join(dir, "bundle-config.yml")
	clientConfigPath := filepath.Join(dir, "client-config.yml")
	cfg := validRuntimeTestConfig(t)
	cfg.ConfigID = "config-id"
	cfg.NetworkID = "network-id"
	cfg.Account.PeerId = "peer-id"
	cfg.Account.PeerKey = "peer-secret"
	cfg.Account.SigningKey = "sign-secret"
	cfg.ExternalAddr = []string{"sync.example.com"}
	cfg.Coordinator.MongoConnect = "mongodb://mongo-user:mongo-secret@127.0.0.1:27017/?authSource=admin"
	cfg.Consensus.MongoConnect = "mongodb://consensus-user:consensus-secret@127.0.0.1:27017/?w=majority"
	cfg.FileNode.RedisConnect = "redis://:redis-secret@127.0.0.1:6379/1"
	cfg.FileNode.S3 = &bundleconfig.S3Config{
		Bucket:         "anytype-data",
		Endpoint:       "https://s3-user:s3-secret@s3.example.com/bucket?token=s3-token#frag",
		Region:         "eu-central-1",
		ForcePathStyle: true,
	}
	writeRuntimeTestClientConfig(t, cfg, clientConfigPath)
	runner := NewRuntimeRunner(RuntimeRunnerConfig{
		BundleConfig:     cfg,
		BundleConfigPath: bundleConfigPath,
		ClientConfigPath: clientConfigPath,
		Now:              fixedRuntimeTestNow,
		LoadIndexSnapshot: func(_ context.Context) (IndexSnapshot, error) {
			return IndexSnapshot{}, nil
		},
		ProbeBlocks: func(_ context.Context, _ Inventory) (BlockProbeResult, error) {
			return BlockProbeResult{}, nil
		},
		ProbeRuntime: healthyRuntimeProbe,
	})

	var out bytes.Buffer
	report, err := runner.RunDoctor(context.Background(), &out)
	if err != nil {
		t.Fatalf("RunDoctor() error = %v", err)
	}

	raw, err := os.ReadFile(report.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	for _, secret := range []string{
		"mongo-secret",
		"consensus-secret",
		"redis-secret",
		"peer-secret",
		"sign-secret",
		"s3-secret",
		"s3-token",
	} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatalf("report contains secret %q:\n%s", secret, string(raw))
		}
	}

	var written Report
	if unmarshalErr := json.Unmarshal(raw, &written); unmarshalErr != nil {
		t.Fatalf("unmarshal report: %v", unmarshalErr)
	}
	if written.Config.ConfigID != "config-id" {
		t.Fatalf("config ID = %q, want config-id", written.Config.ConfigID)
	}
	if written.Config.NetworkID != "network-id" {
		t.Fatalf("network ID = %q, want network-id", written.Config.NetworkID)
	}
	if written.Config.PeerID != "peer-id" {
		t.Fatalf("peer ID = %q, want peer-id", written.Config.PeerID)
	}
	if written.Config.MongoCoordinatorURI != "mongodb://127.0.0.1:27017/" {
		t.Fatalf("coordinator URI = %q", written.Config.MongoCoordinatorURI)
	}
	if written.Config.RedisURI != "redis://127.0.0.1:6379/1" {
		t.Fatalf("redis URI = %q", written.Config.RedisURI)
	}
	if written.Config.S3 == nil {
		t.Fatal("s3 config is nil")
	}
	if written.Config.S3.Bucket != "anytype-data" {
		t.Fatalf("s3 bucket = %q, want anytype-data", written.Config.S3.Bucket)
	}
	if written.Config.S3.Endpoint != "https://s3.example.com/bucket" {
		t.Fatalf("s3 endpoint = %q", written.Config.S3.Endpoint)
	}
	if written.Network.ListenTCPAddr != cfg.Network.ListenTCPAddr {
		t.Fatalf("tcp listen = %q, want %q", written.Network.ListenTCPAddr, cfg.Network.ListenTCPAddr)
	}
	if written.Runtime.Storage != StatusOK {
		t.Fatalf("storage status = %q, want ok", written.Runtime.Storage)
	}
}

func TestRuntimeRunnerReportsStaleClientConfig(t *testing.T) {
	dir := t.TempDir()
	bundleConfigPath := filepath.Join(dir, "bundle-config.yml")
	clientConfigPath := filepath.Join(dir, "client-config.yml")
	cfg := validRuntimeTestConfig(t)
	cfg.NetworkID = "network-id"
	staleClientConfig := `id: config-id
networkId: stale-network-id
nodes:
  - peerId: peer-id
    addresses: []
    types: []
`
	if err := os.WriteFile(clientConfigPath, []byte(staleClientConfig), 0o644); err != nil {
		t.Fatalf("write stale client config: %v", err)
	}
	runner := NewRuntimeRunner(RuntimeRunnerConfig{
		BundleConfig:     cfg,
		BundleConfigPath: bundleConfigPath,
		ClientConfigPath: clientConfigPath,
		Now:              fixedRuntimeTestNow,
		LoadIndexSnapshot: func(_ context.Context) (IndexSnapshot, error) {
			return IndexSnapshot{}, nil
		},
		ProbeBlocks: func(_ context.Context, _ Inventory) (BlockProbeResult, error) {
			return BlockProbeResult{}, nil
		},
		ProbeRuntime: healthyRuntimeProbe,
	})

	var out bytes.Buffer
	report, err := runner.RunDoctor(context.Background(), &out)
	if err != nil {
		t.Fatalf("RunDoctor() error = %v", err)
	}

	if !problemIssuesContain(report.Problems, "client config does not match current bundle config") {
		t.Fatalf("problems do not include stale client config: %v", report.Problems)
	}
}

func TestRuntimeRunnerStopsBeforeInventoryWhenRedisIsUnavailable(t *testing.T) {
	dir := t.TempDir()
	bundleConfigPath := filepath.Join(dir, "bundle-config.yml")
	clientConfigPath := filepath.Join(dir, "client-config.yml")
	cfg := validRuntimeTestConfig(t)
	writeRuntimeTestClientConfig(t, cfg, clientConfigPath)
	runner := NewRuntimeRunner(RuntimeRunnerConfig{
		BundleConfig:     cfg,
		BundleConfigPath: bundleConfigPath,
		ClientConfigPath: clientConfigPath,
		Now:              fixedRuntimeTestNow,
		LoadIndexSnapshot: func(_ context.Context) (IndexSnapshot, error) {
			t.Fatal("LoadIndexSnapshot was called even though Redis is unavailable")
			return IndexSnapshot{}, nil
		},
		ProbeBlocks: func(_ context.Context, _ Inventory) (BlockProbeResult, error) {
			t.Fatal("ProbeBlocks was called even though Redis is unavailable")
			return BlockProbeResult{}, nil
		},
		ProbeRuntime: func(_ context.Context) RuntimeProbeResult {
			return RuntimeProbeResult{
				Mongo:      StatusOK,
				Redis:      StatusProblem,
				RedisBloom: StatusProblem,
				Storage:    StatusOK,
				Problems: []Problem{{
					Scope: problemScopeRuntime,
					Issue: "redis ping failed: connection refused",
				}},
			}
		},
	})

	var out bytes.Buffer
	report, err := runner.RunDoctor(context.Background(), &out)
	if err == nil {
		t.Fatal("RunDoctor() error = nil, want Redis prerequisite error")
	}
	if report != nil {
		t.Fatalf("report = %#v, want nil", report)
	}
	if !strings.Contains(err.Error(), "redis is required for filenode inventory scan") {
		t.Fatalf("RunDoctor() error = %v", err)
	}
	if strings.Contains(out.String(), "[3/7] Network") {
		t.Fatalf("doctor continued after Redis prerequisite failure:\n%s", out.String())
	}
	if strings.Contains(out.String(), "Report:") {
		t.Fatalf("doctor printed report path even though no report was written:\n%s", out.String())
	}
	if _, statErr := os.Stat(filepath.Join(dir, "doctor")); !os.IsNotExist(statErr) {
		t.Fatalf("doctor report directory exists after prerequisite failure: %v", statErr)
	}
}

func TestRuntimeRunnerDoesNotPrintReportPathWhenInventoryFails(t *testing.T) {
	dir := t.TempDir()
	bundleConfigPath := filepath.Join(dir, "bundle-config.yml")
	clientConfigPath := filepath.Join(dir, "client-config.yml")
	cfg := validRuntimeTestConfig(t)
	writeRuntimeTestClientConfig(t, cfg, clientConfigPath)
	runner := NewRuntimeRunner(RuntimeRunnerConfig{
		BundleConfig:     cfg,
		BundleConfigPath: bundleConfigPath,
		ClientConfigPath: clientConfigPath,
		Now:              fixedRuntimeTestNow,
		LoadIndexSnapshot: func(_ context.Context) (IndexSnapshot, error) {
			return IndexSnapshot{}, errors.New("redis scan interrupted")
		},
		ProbeBlocks: func(_ context.Context, _ Inventory) (BlockProbeResult, error) {
			t.Fatal("ProbeBlocks was called even though inventory failed")
			return BlockProbeResult{}, nil
		},
		ProbeRuntime: healthyRuntimeProbe,
	})

	var out bytes.Buffer
	_, err := runner.RunDoctor(context.Background(), &out)
	if err == nil {
		t.Fatal("RunDoctor() error = nil, want inventory error")
	}
	if strings.Contains(out.String(), "/doctor/doctor_") {
		t.Fatalf("doctor printed report path even though no report was written:\n%s", out.String())
	}
}

func TestRuntimeRunnerAddsMissingBlockProblems(t *testing.T) {
	dir := t.TempDir()
	bundleConfigPath := filepath.Join(dir, "bundle-config.yml")
	clientConfigPath := filepath.Join(dir, "client-config.yml")
	cfg := validRuntimeTestConfig(t)
	writeRuntimeTestClientConfig(t, cfg, clientConfigPath)
	groupID := "account1"
	spaceID := "space1"
	fileID := "file1"
	cidMissingBlock := "bafy-missing-block"
	key := index.Key{GroupId: groupID, SpaceId: spaceID}
	runner := NewRuntimeRunner(RuntimeRunnerConfig{
		BundleConfig:     cfg,
		BundleConfigPath: bundleConfigPath,
		ClientConfigPath: clientConfigPath,
		Now:              fixedRuntimeTestNow,
		LoadIndexSnapshot: func(_ context.Context) (IndexSnapshot, error) {
			return IndexSnapshot{
				Hashes: map[string]map[string]string{
					index.GroupKey(key): {
						redisIndexInfoField: mustMarshalGroupEntry(
							t,
							&indexproto.GroupEntry{GroupId: groupID, SpaceIds: []string{spaceID}},
						),
					},
					index.SpaceKey(key): {
						redisIndexInfoField: mustMarshalSpaceEntry(
							t,
							&indexproto.SpaceEntry{GroupId: groupID, FileCount: 1, CidCount: 1, Size: 10},
						),
						index.FileKey(fileID): mustMarshalFileEntry(
							t,
							&indexproto.FileEntry{Cids: []string{cidMissingBlock}, Size: 10},
						),
					},
				},
				Values: map[string]string{
					redisIndexCIDKey(cidMissingBlock): mustMarshalCIDEntry(t, &indexproto.CidEntry{Size: 10, Refs: 1}),
				},
			}, nil
		},
		ProbeBlocks: func(_ context.Context, _ Inventory) (BlockProbeResult, error) {
			return BlockProbeResult{
				Checked: 1,
				MissingByFile: map[string][]string{
					fileReportKey(spaceID, fileID): {cidMissingBlock},
				},
			}, nil
		},
		ProbeRuntime: healthyRuntimeProbe,
	})

	var out bytes.Buffer
	report, err := runner.RunDoctor(context.Background(), &out)
	if err != nil {
		t.Fatalf("RunDoctor() error = %v", err)
	}

	if report.Summary.Spaces != 1 {
		t.Fatalf("spaces = %d, want 1", report.Summary.Spaces)
	}
	if report.Inventory.Spaces[0].MissingBlocks != 1 {
		t.Fatalf("missing blocks = %d, want 1", report.Inventory.Spaces[0].MissingBlocks)
	}
	if report.Verdict != VerdictFilesRequireClientReupload {
		t.Fatalf("verdict = %q, want %q", report.Verdict, VerdictFilesRequireClientReupload)
	}
	if !strings.Contains(report.SuggestedNextAction, "original client/cache") {
		t.Fatalf("suggested next action = %q", report.SuggestedNextAction)
	}
	if !strings.Contains(out.String(), "missing blocks: 1") {
		t.Fatalf("output does not mention missing block:\n%s", out.String())
	}
}

func TestPrintSpacesShowsIndexProblemsWithoutMissingGlobalCID(t *testing.T) {
	inventory := Inventory{
		Spaces: []SpaceReport{{
			ID:            "space1",
			Files:         1,
			CIDs:          1,
			IndexProblems: 2,
			Status:        StatusProblem,
		}},
	}

	var out bytes.Buffer
	printSpaces(&out, inventory)

	if !strings.Contains(out.String(), "index: problems 2") {
		t.Fatalf("output does not show index problems:\n%s", out.String())
	}
}

func validRuntimeTestConfig(t *testing.T) *bundleconfig.Config {
	t.Helper()

	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	t.Cleanup(func() {
		_ = tcpListener.Close()
	})
	udpListener, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	t.Cleanup(func() {
		_ = udpListener.Close()
	})

	return &bundleconfig.Config{
		ExternalAddr: []string{"127.0.0.1"},
		ConfigID:     "config-id",
		NetworkID:    "network-id",
		StoragePath:  "./data/storage",
		Account: accountservice.Config{
			PeerId: "peer-id",
		},
		Network: bundleconfig.NetworkConfig{
			ListenTCPAddr: tcpListener.Addr().String(),
			ListenUDPAddr: udpListener.LocalAddr().String(),
		},
		Coordinator: bundleconfig.CoordinatorConfig{
			MongoConnect:  "mongodb://127.0.0.1:27017/",
			MongoDatabase: "coordinator",
		},
		Consensus: bundleconfig.ConsensusConfig{
			MongoConnect:  "mongodb://127.0.0.1:27017/?w=majority",
			MongoDatabase: "consensus",
		},
		FileNode: bundleconfig.FileNodeConfig{
			RedisConnect: "redis://127.0.0.1:6379/",
			DefaultLimit: 1,
		},
	}
}

func writeRuntimeTestClientConfig(t *testing.T, cfg *bundleconfig.Config, path string) {
	t.Helper()

	data, err := cfg.YamlClientConfig()
	if err != nil {
		t.Fatalf("client config yaml: %v", err)
	}
	if writeErr := os.WriteFile(path, data, 0o644); writeErr != nil {
		t.Fatalf("write client config: %v", writeErr)
	}
}

func healthyRuntimeProbe(_ context.Context) RuntimeProbeResult {
	return RuntimeProbeResult{
		Mongo:      StatusOK,
		Redis:      StatusOK,
		RedisBloom: StatusOK,
		Storage:    StatusOK,
	}
}

func fixedRuntimeTestNow() time.Time {
	return time.Date(2026, 5, 22, 14, 33, 10, 0, time.UTC)
}

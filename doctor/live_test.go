package doctor

import (
	"context"
	"path/filepath"
	"testing"

	filenodestore "github.com/anyproto/any-sync-filenode/store"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/commonfile/fileblockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"

	bundleconfig "github.com/grishy/any-sync-bundle/config"
)

func TestProbeRuntimeReportsMissingStoragePath(t *testing.T) {
	cfg := &bundleconfig.Config{
		StoragePath: filepath.Join(t.TempDir(), "missing-storage"),
		Coordinator: bundleconfig.CoordinatorConfig{
			MongoConnect: "://bad-mongo-uri",
		},
		Consensus: bundleconfig.ConsensusConfig{
			MongoConnect: "://bad-mongo-uri",
		},
	}

	result := ProbeRuntime(context.Background(), cfg, nil)

	if result.Storage != StatusProblem {
		t.Fatalf("storage status = %q, want %q", result.Storage, StatusProblem)
	}
	if !problemIssuesContain(result.Problems, "storage path check failed") {
		t.Fatalf("problems do not include storage failure: %v", result.Problems)
	}
}

func TestProbeRuntimeReportsMissingStorageLayout(t *testing.T) {
	storagePath := t.TempDir()
	cfg := &bundleconfig.Config{
		StoragePath: storagePath,
		Coordinator: bundleconfig.CoordinatorConfig{
			MongoConnect: "://bad-mongo-uri",
		},
		Consensus: bundleconfig.ConsensusConfig{
			MongoConnect: "://bad-mongo-uri",
		},
	}

	result := ProbeRuntime(context.Background(), cfg, nil)

	if result.Storage != StatusProblem {
		t.Fatalf("storage status = %q, want %q", result.Storage, StatusProblem)
	}
	if !problemIssuesContain(result.Problems, "storage layout check failed") {
		t.Fatalf("problems do not include storage layout failure: %v", result.Problems)
	}
	if !problemIssuesContain(result.Problems, "network-store/coordinator") {
		t.Fatalf("problems do not mention missing coordinator network store: %v", result.Problems)
	}
}

func TestProbeBlocksWithStoreReportsCorruptBlockData(t *testing.T) {
	expectedBlock := blocks.NewBlock([]byte("expected-data"))
	returnedBlock, err := blocks.NewBlockWithCid([]byte("other-data"), expectedBlock.Cid())
	if err != nil {
		t.Fatalf("make returned block: %v", err)
	}
	store := &doctorTestStore{
		block: returnedBlock,
	}
	inventory := Inventory{
		Files: []FileReport{{
			ID:       "file1",
			SpaceID:  "space1",
			Size:     uint64(len(expectedBlock.RawData())),
			CIDs:     []string{expectedBlock.Cid().String()},
			CIDSizes: map[string]uint64{expectedBlock.Cid().String(): uint64(len(expectedBlock.RawData()))},
		}},
	}

	result, err := ProbeBlocksWithStore(context.Background(), inventory, store)
	if err != nil {
		t.Fatalf("ProbeBlocksWithStore() error = %v", err)
	}

	key := fileReportKey("space1", "file1")
	if len(result.CorruptByFile[key]) != 1 {
		t.Fatalf("corrupt blocks for file = %v, want one block", result.CorruptByFile[key])
	}
	if !problemIssuesContain(result.Problems, "block "+expectedBlock.Cid().String()+" content does not match its CID") {
		t.Fatalf("problems do not include corrupt block content: %v", result.Problems)
	}
}

var _ filenodestore.Store = (*doctorTestStore)(nil)

type doctorTestStore struct {
	block blocks.Block
}

func (s *doctorTestStore) Init(_ *app.App) error {
	return nil
}

func (s *doctorTestStore) Name() string {
	return "doctor.test.store"
}

func (s *doctorTestStore) Get(_ context.Context, _ cid.Cid) (blocks.Block, error) {
	return s.block, nil
}

func (s *doctorTestStore) GetMany(_ context.Context, _ []cid.Cid) <-chan blocks.Block {
	ch := make(chan blocks.Block)
	close(ch)
	return ch
}

func (s *doctorTestStore) Add(_ context.Context, _ []blocks.Block) error {
	return nil
}

func (s *doctorTestStore) Delete(_ context.Context, _ cid.Cid) error {
	return nil
}

func (s *doctorTestStore) DeleteMany(_ context.Context, _ []cid.Cid) error {
	return nil
}

func (s *doctorTestStore) IndexGet(_ context.Context, _ string) ([]byte, error) {
	return nil, fileblockstore.ErrCIDNotFound
}

func (s *doctorTestStore) IndexPut(_ context.Context, _ string, _ []byte) error {
	return nil
}

func (s *doctorTestStore) IndexDelete(_ context.Context, _ string) error {
	return nil
}

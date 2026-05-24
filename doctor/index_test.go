package doctor

import (
	"strings"
	"testing"

	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync-filenode/index/indexproto"
)

func TestInspectIndexSnapshotBuildsInventoryAndFindsMissingCIDIndex(t *testing.T) {
	groupID := "account1"
	spaceID := "space1"
	fileID := "file1"
	cidOK := "bafy-ok"
	cidMissing := "bafy-missing"
	key := index.Key{GroupId: groupID, SpaceId: spaceID}
	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{
			index.GroupKey(key): {
				redisIndexInfoField: mustMarshalGroupEntry(
					t,
					&indexproto.GroupEntry{GroupId: groupID, SpaceIds: []string{spaceID}},
				),
				redisIndexCIDKey(cidOK): "1",
			},
			index.SpaceKey(key): {
				redisIndexInfoField: mustMarshalSpaceEntry(
					t,
					&indexproto.SpaceEntry{GroupId: groupID, FileCount: 1, CidCount: 2, Size: 10},
				),
				index.FileKey(fileID): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidOK, cidMissing}, Size: 10},
				),
				redisIndexCIDKey(cidOK):      "1",
				redisIndexCIDKey(cidMissing): "1",
			},
		},
		Values: map[string]string{
			redisIndexCIDKey(cidOK): mustMarshalCIDEntry(t, &indexproto.CidEntry{Size: 5, Refs: 1}),
		},
	}

	inventory, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}

	if len(inventory.Groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(inventory.Groups))
	}
	if inventory.Groups[0].ID != groupID {
		t.Fatalf("group ID = %q, want %q", inventory.Groups[0].ID, groupID)
	}
	if inventory.Groups[0].Spaces != 1 {
		t.Fatalf("group spaces = %d, want 1", inventory.Groups[0].Spaces)
	}
	if inventory.Groups[0].Files != 1 {
		t.Fatalf("group files = %d, want 1", inventory.Groups[0].Files)
	}
	if inventory.Groups[0].CIDs != 2 {
		t.Fatalf("group cids = %d, want 2", inventory.Groups[0].CIDs)
	}
	if inventory.Groups[0].Bytes != 5 {
		t.Fatalf("group bytes = %d, want 5", inventory.Groups[0].Bytes)
	}
	if len(inventory.Spaces) != 1 {
		t.Fatalf("spaces = %d, want 1", len(inventory.Spaces))
	}

	space := inventory.Spaces[0]
	if space.ID != spaceID {
		t.Fatalf("space ID = %q, want %q", space.ID, spaceID)
	}
	if space.GroupID != groupID {
		t.Fatalf("space group = %q, want %q", space.GroupID, groupID)
	}
	if space.Files != 1 {
		t.Fatalf("space files = %d, want 1", space.Files)
	}
	if space.CIDs != 2 {
		t.Fatalf("space cids = %d, want 2", space.CIDs)
	}
	if space.MissingCIDIndex != 1 {
		t.Fatalf("missing CID index = %d, want 1", space.MissingCIDIndex)
	}
	if space.Status != StatusProblem {
		t.Fatalf("space status = %q, want %q", space.Status, StatusProblem)
	}

	if len(problems) == 0 {
		t.Fatal("problems = 0, want at least one missing CID problem")
	}
	if !strings.Contains(problems[0].Issue, cidMissing) {
		t.Fatalf("problem issue = %q, want missing CID", problems[0].Issue)
	}
}

func TestInspectIndexSnapshotKeepsEmptySpacesVisible(t *testing.T) {
	groupID := "account1"
	spaceID := "empty-space"
	key := index.Key{GroupId: groupID, SpaceId: spaceID}
	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{
			index.GroupKey(key): {
				redisIndexInfoField: mustMarshalGroupEntry(
					t,
					&indexproto.GroupEntry{GroupId: groupID, SpaceIds: []string{spaceID}},
				),
			},
			index.SpaceKey(key): {
				redisIndexInfoField: mustMarshalSpaceEntry(t, &indexproto.SpaceEntry{GroupId: groupID}),
			},
		},
	}

	inventory, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}

	if len(problems) != 0 {
		t.Fatalf("problems = %d, want 0", len(problems))
	}
	if len(inventory.Spaces) != 1 {
		t.Fatalf("spaces = %d, want 1", len(inventory.Spaces))
	}
	if inventory.Spaces[0].Status != StatusEmpty {
		t.Fatalf("space status = %q, want %q", inventory.Spaces[0].Status, StatusEmpty)
	}
}

func TestInspectIndexSnapshotFindsCIDRefAndCounterProblems(t *testing.T) {
	groupID := "account1"
	spaceID := "space1"
	fileID := "file1"
	cidOK := "bafy-ok"
	cidStale := "bafy-stale"
	key := index.Key{GroupId: groupID, SpaceId: spaceID}
	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{
			index.GroupKey(key): {
				redisIndexInfoField: mustMarshalGroupEntry(
					t,
					&indexproto.GroupEntry{GroupId: groupID, SpaceIds: []string{spaceID}, CidCount: 2, Size: 20},
				),
				redisIndexCIDKey(cidStale): "1",
			},
			index.SpaceKey(key): {
				redisIndexInfoField: mustMarshalSpaceEntry(
					t,
					&indexproto.SpaceEntry{GroupId: groupID, FileCount: 2, CidCount: 2, Size: 20},
				),
				index.FileKey(fileID): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidOK}, Size: 10},
				),
				redisIndexCIDKey(cidStale): "1",
			},
		},
		Values: map[string]string{
			redisIndexCIDKey(cidOK): mustMarshalCIDEntry(t, &indexproto.CidEntry{Size: 10, Refs: 1}),
		},
	}

	inventory, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}

	if len(inventory.Spaces) != 1 {
		t.Fatalf("spaces = %d, want 1", len(inventory.Spaces))
	}
	space := inventory.Spaces[0]
	if space.Status != StatusProblem {
		t.Fatalf("space status = %q, want %q", space.Status, StatusProblem)
	}
	if space.IndexProblems < 5 {
		t.Fatalf("space index problems = %d, want at least 5", space.IndexProblems)
	}

	wantIssues := []string{
		"space is missing CID ref " + cidOK,
		"space has stale CID ref " + cidStale,
		"group is missing CID ref " + cidOK,
		"group has stale CID ref " + cidStale,
		"space file count mismatch",
		"space CID count mismatch",
		"space byte count mismatch",
		"group CID count mismatch",
		"group byte count mismatch",
	}
	for _, want := range wantIssues {
		if !problemIssuesContain(problems, want) {
			t.Fatalf("problems do not contain %q:\n%v", want, problems)
		}
	}
}

func TestInspectIndexSnapshotDeduplicatesGroupCIDCounts(t *testing.T) {
	groupID := "account1"
	spaceIDOne := "space1"
	spaceIDTwo := "space2"
	cidShared := "bafy-shared"
	keyOne := index.Key{GroupId: groupID, SpaceId: spaceIDOne}
	keyTwo := index.Key{GroupId: groupID, SpaceId: spaceIDTwo}
	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{
			index.GroupKey(keyOne): {
				redisIndexInfoField: mustMarshalGroupEntry(
					t,
					&indexproto.GroupEntry{
						GroupId:  groupID,
						SpaceIds: []string{spaceIDOne, spaceIDTwo},
						CidCount: 1,
						Size:     10,
					},
				),
				redisIndexCIDKey(cidShared): "2",
			},
			index.SpaceKey(keyOne): {
				redisIndexInfoField: mustMarshalSpaceEntry(
					t,
					&indexproto.SpaceEntry{GroupId: groupID, FileCount: 1, CidCount: 1, Size: 10},
				),
				index.FileKey("file1"): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidShared}, Size: 10},
				),
				redisIndexCIDKey(cidShared): "1",
			},
			index.SpaceKey(keyTwo): {
				redisIndexInfoField: mustMarshalSpaceEntry(
					t,
					&indexproto.SpaceEntry{GroupId: groupID, FileCount: 1, CidCount: 1, Size: 10},
				),
				index.FileKey("file2"): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidShared}, Size: 10},
				),
				redisIndexCIDKey(cidShared): "1",
			},
		},
		Values: map[string]string{
			redisIndexCIDKey(cidShared): mustMarshalCIDEntry(t, &indexproto.CidEntry{Size: 10, Refs: 2}),
		},
	}

	inventory, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("problems = %d, want 0: %v", len(problems), problems)
	}
	if len(inventory.Groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(inventory.Groups))
	}
	if inventory.Groups[0].CIDs != 1 {
		t.Fatalf("group cids = %d, want 1", inventory.Groups[0].CIDs)
	}
	if inventory.Groups[0].Files != 2 {
		t.Fatalf("group files = %d, want 2", inventory.Groups[0].Files)
	}
}

func TestInspectIndexSnapshotFindsCIDRefCountMismatches(t *testing.T) {
	groupID := "account1"
	spaceID := "space1"
	cidShared := "bafy-shared"
	key := index.Key{GroupId: groupID, SpaceId: spaceID}
	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{
			index.GroupKey(key): {
				redisIndexInfoField: mustMarshalGroupEntry(
					t,
					&indexproto.GroupEntry{GroupId: groupID, SpaceIds: []string{spaceID}, CidCount: 1, Size: 10},
				),
				redisIndexCIDKey(cidShared): "1",
			},
			index.SpaceKey(key): {
				redisIndexInfoField: mustMarshalSpaceEntry(
					t,
					&indexproto.SpaceEntry{GroupId: groupID, FileCount: 2, CidCount: 1, Size: 10},
				),
				index.FileKey("file1"): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidShared}, Size: 10},
				),
				index.FileKey("file2"): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidShared}, Size: 10},
				),
				redisIndexCIDKey(cidShared): "1",
			},
		},
		Values: map[string]string{
			redisIndexCIDKey(cidShared): mustMarshalCIDEntry(t, &indexproto.CidEntry{Size: 10, Refs: 1}),
		},
	}

	_, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}

	for _, want := range []string{
		"space CID ref mismatch for " + cidShared + ": index=1 actual=2",
		"group CID ref mismatch for " + cidShared + ": index=1 actual=2",
	} {
		if !problemIssuesContain(problems, want) {
			t.Fatalf("problems do not contain %q:\n%v", want, problems)
		}
	}
}

func TestInspectIndexSnapshotComputesBytesFromCIDEntries(t *testing.T) {
	groupID := "account1"
	spaceID := "space1"
	cidShared := "bafy-shared"
	key := index.Key{GroupId: groupID, SpaceId: spaceID}
	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{
			index.GroupKey(key): {
				redisIndexInfoField: mustMarshalGroupEntry(
					t,
					&indexproto.GroupEntry{GroupId: groupID, SpaceIds: []string{spaceID}, CidCount: 1, Size: 10},
				),
				redisIndexCIDKey(cidShared): "2",
			},
			index.SpaceKey(key): {
				redisIndexInfoField: mustMarshalSpaceEntry(
					t,
					&indexproto.SpaceEntry{GroupId: groupID, FileCount: 2, CidCount: 1, Size: 10},
				),
				index.FileKey("file1"): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidShared}, Size: 10},
				),
				index.FileKey("file2"): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidShared}, Size: 10},
				),
				redisIndexCIDKey(cidShared): "2",
			},
		},
		Values: map[string]string{
			redisIndexCIDKey(cidShared): mustMarshalCIDEntry(t, &indexproto.CidEntry{Size: 10, Refs: 1}),
		},
	}

	inventory, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}

	if len(problems) != 0 {
		t.Fatalf("problems = %d, want 0: %v", len(problems), problems)
	}
	if inventory.Spaces[0].Bytes != 10 {
		t.Fatalf("space bytes = %d, want 10", inventory.Spaces[0].Bytes)
	}
	if inventory.Groups[0].Bytes != 10 {
		t.Fatalf("group bytes = %d, want 10", inventory.Groups[0].Bytes)
	}
}

func TestInspectIndexSnapshotFindsCorruptGlobalCIDEntry(t *testing.T) {
	groupID := "account1"
	spaceID := "space1"
	cidCorrupt := "bafy-corrupt"
	key := index.Key{GroupId: groupID, SpaceId: spaceID}
	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{
			index.GroupKey(key): {
				redisIndexInfoField: mustMarshalGroupEntry(
					t,
					&indexproto.GroupEntry{GroupId: groupID, SpaceIds: []string{spaceID}, CidCount: 1, Size: 10},
				),
				redisIndexCIDKey(cidCorrupt): "1",
			},
			index.SpaceKey(key): {
				redisIndexInfoField: mustMarshalSpaceEntry(
					t,
					&indexproto.SpaceEntry{GroupId: groupID, FileCount: 1, CidCount: 1, Size: 10},
				),
				index.FileKey("file1"): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidCorrupt}, Size: 10},
				),
				redisIndexCIDKey(cidCorrupt): "1",
			},
		},
		Values: map[string]string{
			redisIndexCIDKey(cidCorrupt): "not-protobuf",
		},
	}

	_, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}
	if !problemIssuesContain(problems, "global CID entry "+cidCorrupt+" cannot be decoded") {
		t.Fatalf("problems do not include corrupt global CID entry: %v", problems)
	}
}

func TestInspectIndexSnapshotFindsFileSizeMismatchFromCIDEntries(t *testing.T) {
	groupID := "account1"
	spaceID := "space1"
	cidOne := "bafy-one"
	cidTwo := "bafy-two"
	key := index.Key{GroupId: groupID, SpaceId: spaceID}
	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{
			index.GroupKey(key): {
				redisIndexInfoField: mustMarshalGroupEntry(
					t,
					&indexproto.GroupEntry{GroupId: groupID, SpaceIds: []string{spaceID}, CidCount: 2, Size: 10},
				),
				redisIndexCIDKey(cidOne): "1",
				redisIndexCIDKey(cidTwo): "1",
			},
			index.SpaceKey(key): {
				redisIndexInfoField: mustMarshalSpaceEntry(
					t,
					&indexproto.SpaceEntry{GroupId: groupID, FileCount: 1, CidCount: 2, Size: 10},
				),
				index.FileKey("file1"): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidOne, cidTwo}, Size: 99},
				),
				redisIndexCIDKey(cidOne): "1",
				redisIndexCIDKey(cidTwo): "1",
			},
		},
		Values: map[string]string{
			redisIndexCIDKey(cidOne): mustMarshalCIDEntry(t, &indexproto.CidEntry{Size: 4, Refs: 1}),
			redisIndexCIDKey(cidTwo): mustMarshalCIDEntry(t, &indexproto.CidEntry{Size: 6, Refs: 1}),
		},
	}

	_, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}
	if !problemIssuesContain(problems, "file size mismatch in space space1 file file1: index=99 actual=10") {
		t.Fatalf("problems do not include file size mismatch: %v", problems)
	}
}

func TestInspectIndexSnapshotAssignsSpaceGroupFromGroupEntrySpaceIDs(t *testing.T) {
	groupIDOne := "account1"
	groupIDTwo := "account2"
	spaceID := "space1"
	cidOne := "bafy-one"
	keyOne := index.Key{GroupId: groupIDOne, SpaceId: spaceID}
	keyTwo := index.Key{GroupId: groupIDTwo, SpaceId: "other-space"}
	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{
			index.GroupKey(keyOne): {
				redisIndexInfoField: mustMarshalGroupEntry(
					t,
					&indexproto.GroupEntry{GroupId: groupIDOne, SpaceIds: []string{spaceID}, CidCount: 1, Size: 10},
				),
				redisIndexCIDKey(cidOne): "1",
			},
			index.GroupKey(keyTwo): {
				redisIndexInfoField: mustMarshalGroupEntry(t, &indexproto.GroupEntry{GroupId: groupIDTwo}),
			},
			index.SpaceKey(keyOne): {
				redisIndexInfoField: mustMarshalSpaceEntry(
					t,
					&indexproto.SpaceEntry{FileCount: 1, CidCount: 1, Size: 10},
				),
				index.FileKey("file1"): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidOne}, Size: 10},
				),
				redisIndexCIDKey(cidOne): "1",
			},
		},
		Values: map[string]string{
			redisIndexCIDKey(cidOne): mustMarshalCIDEntry(t, &indexproto.CidEntry{Size: 10, Refs: 1}),
		},
	}

	inventory, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("problems = %d, want 0: %v", len(problems), problems)
	}
	if inventory.Spaces[0].GroupID != groupIDOne {
		t.Fatalf("space group = %q, want %q", inventory.Spaces[0].GroupID, groupIDOne)
	}
}

func TestInspectIndexSnapshotReportsUnassignedSpace(t *testing.T) {
	groupID := "account1"
	spaceID := "space1"
	cidOne := "bafy-one"
	key := index.Key{GroupId: groupID, SpaceId: spaceID}
	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{
			index.GroupKey(key): {
				redisIndexInfoField: mustMarshalGroupEntry(
					t,
					&indexproto.GroupEntry{GroupId: groupID, SpaceIds: []string{"other-space"}},
				),
			},
			index.SpaceKey(key): {
				redisIndexInfoField: mustMarshalSpaceEntry(
					t,
					&indexproto.SpaceEntry{FileCount: 1, CidCount: 1, Size: 10},
				),
				index.FileKey("file1"): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidOne}, Size: 10},
				),
				redisIndexCIDKey(cidOne): "1",
			},
		},
		Values: map[string]string{
			redisIndexCIDKey(cidOne): mustMarshalCIDEntry(t, &indexproto.CidEntry{Size: 10, Refs: 1}),
		},
	}

	inventory, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}
	if inventory.Spaces[0].GroupID != "" {
		t.Fatalf("space group = %q, want empty", inventory.Spaces[0].GroupID)
	}
	if !problemIssuesContain(problems, "space "+spaceID+" has no group assignment") {
		t.Fatalf("problems do not include unassigned space: %v", problems)
	}
}

func TestInspectIndexSnapshotReportsRequiredInfoAndOwnershipProblems(t *testing.T) {
	groupID := "account1"
	otherGroupID := "account2"
	spaceID := "space1"
	missingSpaceID := "missing-space"
	key := index.Key{GroupId: groupID, SpaceId: spaceID}
	otherKey := index.Key{GroupId: otherGroupID, SpaceId: spaceID}
	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{
			index.GroupKey(key): {
				redisIndexInfoField: mustMarshalGroupEntry(t, &indexproto.GroupEntry{
					GroupId:  groupID,
					SpaceIds: []string{missingSpaceID},
				}),
			},
			index.GroupKey(otherKey): {},
			index.SpaceKey(key): {
				redisIndexInfoField: mustMarshalSpaceEntry(t, &indexproto.SpaceEntry{
					GroupId: otherGroupID,
				}),
			},
		},
	}

	_, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}

	for _, want := range []string{
		"group info is missing",
		"space " + missingSpaceID + " is listed in group " + groupID + " but has no space entry",
		"space " + spaceID + " belongs to group " + otherGroupID + " but no group lists it",
	} {
		if !problemIssuesContain(problems, want) {
			t.Fatalf("problems do not contain %q:\n%v", want, problems)
		}
	}
}

func TestInspectIndexSnapshotReportsGlobalCIDEntriesWithoutFileRefs(t *testing.T) {
	cidOrphan := "bafy-orphan"
	cidCorrupt := "bafy-corrupt-orphan"
	snapshot := IndexSnapshot{
		Values: map[string]string{
			redisIndexCIDKey(cidOrphan):  mustMarshalCIDEntry(t, &indexproto.CidEntry{Size: 10, Refs: 1}),
			redisIndexCIDKey(cidCorrupt): "not-protobuf",
		},
	}

	_, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}

	for _, want := range []string{
		"global CID entry " + cidCorrupt + " cannot be decoded",
		"global CID entry " + cidOrphan + " is not referenced by any file",
	} {
		if !problemIssuesContain(problems, want) {
			t.Fatalf("problems do not contain %q:\n%v", want, problems)
		}
	}
}

func TestInspectIndexSnapshotReportsLimitProblems(t *testing.T) {
	groupID := "account1"
	spaceID := "space1"
	cidOne := "bafy-one"
	key := index.Key{GroupId: groupID, SpaceId: spaceID}
	snapshot := IndexSnapshot{
		Hashes: map[string]map[string]string{
			index.GroupKey(key): {
				redisIndexInfoField: mustMarshalGroupEntry(t, &indexproto.GroupEntry{
					GroupId:      groupID,
					SpaceIds:     []string{spaceID},
					CidCount:     1,
					Size:         10,
					Limit:        5,
					AccountLimit: 7,
				}),
				redisIndexCIDKey(cidOne): "1",
			},
			index.SpaceKey(key): {
				redisIndexInfoField: mustMarshalSpaceEntry(t, &indexproto.SpaceEntry{
					GroupId:   groupID,
					FileCount: 1,
					CidCount:  1,
					Size:      10,
					Limit:     5,
				}),
				index.FileKey("file1"): mustMarshalFileEntry(
					t,
					&indexproto.FileEntry{Cids: []string{cidOne}, Size: 10},
				),
				redisIndexCIDKey(cidOne): "1",
			},
		},
		Values: map[string]string{
			redisIndexCIDKey(cidOne): mustMarshalCIDEntry(t, &indexproto.CidEntry{Size: 10, Refs: 1}),
		},
	}

	inventory, problems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		t.Fatalf("InspectIndexSnapshot() error = %v", err)
	}

	if inventory.Groups[0].Limit != 5 {
		t.Fatalf("group limit = %d, want 5", inventory.Groups[0].Limit)
	}
	if inventory.Spaces[0].Limit != 5 {
		t.Fatalf("space limit = %d, want 5", inventory.Spaces[0].Limit)
	}
	for _, want := range []string{
		"group byte size exceeds limit: size=10 limit=5",
		"group byte size exceeds account limit: size=10 accountLimit=7",
		"space byte size exceeds limit: size=10 limit=5",
	} {
		if !problemIssuesContain(problems, want) {
			t.Fatalf("problems do not contain %q:\n%v", want, problems)
		}
	}
}

func mustMarshalGroupEntry(t *testing.T, entry *indexproto.GroupEntry) string {
	t.Helper()
	data, err := entry.MarshalVT()
	if err != nil {
		t.Fatalf("marshal group entry: %v", err)
	}
	return string(data)
}

func mustMarshalSpaceEntry(t *testing.T, entry *indexproto.SpaceEntry) string {
	t.Helper()
	data, err := entry.MarshalVT()
	if err != nil {
		t.Fatalf("marshal space entry: %v", err)
	}
	return string(data)
}

func mustMarshalFileEntry(t *testing.T, entry *indexproto.FileEntry) string {
	t.Helper()
	data, err := entry.MarshalVT()
	if err != nil {
		t.Fatalf("marshal file entry: %v", err)
	}
	return string(data)
}

func mustMarshalCIDEntry(t *testing.T, entry *indexproto.CidEntry) string {
	t.Helper()
	data, err := entry.MarshalVT()
	if err != nil {
		t.Fatalf("marshal CID entry: %v", err)
	}
	return string(data)
}

func problemIssuesContain(problems []Problem, want string) bool {
	for _, problem := range problems {
		if strings.Contains(problem.Issue, want) {
			return true
		}
	}
	return false
}

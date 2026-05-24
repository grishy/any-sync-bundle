package doctor

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/anyproto/any-sync-filenode/index/indexproto"
)

const (
	redisIndexInfoField   = "info"
	redisIndexGroupPrefix = "g:"
	redisIndexSpacePrefix = "s:"
	redisIndexCIDPrefix   = "c:"
	redisIndexFilePrefix  = "f:"
)

type IndexSnapshot struct {
	Hashes map[string]map[string]string
	Values map[string]string
}

type CIDRefs map[string]uint64

type cidEntryState struct {
	entry      *indexproto.CidEntry
	problem    Problem
	hasProblem bool
	missing    bool
}

type spaceInspection struct {
	space          SpaceReport
	files          []FileReport
	expectedRefs   CIDRefs
	uniqueCIDs     map[string]struct{}
	uniqueCIDBytes map[string]uint64
	problems       []Problem
}

//nolint:cyclop,funlen,gocognit,gocyclo,nestif // The Redis index scan is one coherent ownership/ref/counter pass.
func InspectIndexSnapshot(snapshot IndexSnapshot) (Inventory, []Problem, error) {
	inventory := Inventory{}
	problems := []Problem{}
	groupsByID := map[string]*GroupReport{}
	groupInfosByID := map[string]*indexproto.GroupEntry{}
	groupRefsByID := map[string]CIDRefs{}
	groupExpectedRefsByID := map[string]CIDRefs{}
	groupCIDBytesByID := map[string]map[string]uint64{}
	spaceGroupByID := map[string]string{}
	listedSpaceIDsByGroupID := map[string]map[string]struct{}{}
	seenSpaceIDs := map[string]struct{}{}
	cidEntryCache := map[string]cidEntryState{}
	cidProblemReported := map[string]struct{}{}
	globalSpacesByCID := map[string]map[string]struct{}{}

	hashKeys := sortedMapKeys(snapshot.Hashes)
	for _, key := range hashKeys {
		if !strings.HasPrefix(key, redisIndexGroupPrefix) {
			continue
		}
		groupID := parseIndexHashID(key, redisIndexGroupPrefix)
		report := &GroupReport{
			ID:     groupID,
			Status: StatusOK,
		}
		fields := snapshot.Hashes[key]
		if encodedInfo, ok := fields[redisIndexInfoField]; ok {
			entry := &indexproto.GroupEntry{}
			if err := entry.UnmarshalVT([]byte(encodedInfo)); err != nil {
				report.Status = StatusProblem
				report.IndexProblems++
				problems = append(problems, Problem{
					Scope: problemScopeGroup,
					ID:    groupID,
					Issue: fmt.Sprintf("group info cannot be decoded: %v", err),
				})
			} else {
				report.ID = entry.GetGroupId()
				if report.ID == "" {
					report.ID = groupID
				}
				report.Limit = entry.GetLimit()
				report.AccountLimit = entry.GetAccountLimit()
				groupInfosByID[report.ID] = entry
				listedSpaces := listedSpaceIDsByGroupID[report.ID]
				if listedSpaces == nil {
					listedSpaces = map[string]struct{}{}
					listedSpaceIDsByGroupID[report.ID] = listedSpaces
				}
				for _, spaceID := range entry.GetSpaceIds() {
					listedSpaces[spaceID] = struct{}{}
					if existingGroupID, exists := spaceGroupByID[spaceID]; exists && existingGroupID != report.ID {
						report.Status = StatusProblem
						report.IndexProblems++
						problems = append(problems, Problem{
							Scope: problemScopeSpace,
							ID:    spaceID,
							Issue: fmt.Sprintf("space %s is listed in multiple groups: %s and %s",
								spaceID, existingGroupID, report.ID),
						})
					} else {
						spaceGroupByID[spaceID] = report.ID
					}
				}
			}
		} else {
			report.Status = StatusProblem
			report.IndexProblems++
			problems = append(problems, Problem{
				Scope: problemScopeGroup,
				ID:    groupID,
				Issue: "group info is missing",
			})
		}
		refs, refProblems := cidRefsFromFields("group", report.ID, fields)
		if len(refProblems) > 0 {
			report.Status = StatusProblem
			report.IndexProblems += uint64(len(refProblems))
			problems = append(problems, refProblems...)
		}
		groupRefsByID[report.ID] = refs
		groupsByID[report.ID] = report
	}

	for _, key := range sortedMapKeys(snapshot.Values) {
		if !strings.HasPrefix(key, redisIndexCIDPrefix) {
			continue
		}
		cid := strings.TrimPrefix(key, redisIndexCIDPrefix)
		loadCIDEntry(cid, snapshot.Values, cidEntryCache)
	}

	for _, key := range hashKeys {
		if !strings.HasPrefix(key, redisIndexSpacePrefix) {
			continue
		}
		inspection := inspectSpaceHash(
			key,
			snapshot.Hashes[key],
			snapshot.Values,
			spaceGroupByID,
			cidEntryCache,
			cidProblemReported,
		)
		problems = append(problems, inspection.problems...)
		seenSpaceIDs[inspection.space.ID] = struct{}{}
		inventory.Spaces = append(inventory.Spaces, inspection.space)
		inventory.Files = append(inventory.Files, inspection.files...)

		for cid := range inspection.uniqueCIDs {
			spaces := globalSpacesByCID[cid]
			if spaces == nil {
				spaces = map[string]struct{}{}
				globalSpacesByCID[cid] = spaces
			}
			spaces[inspection.space.ID] = struct{}{}
		}

		if inspection.space.GroupID == "" {
			continue
		}
		group := groupsByID[inspection.space.GroupID]
		if group == nil {
			group = &GroupReport{ID: inspection.space.GroupID, Status: StatusOK}
			groupsByID[inspection.space.GroupID] = group
		}
		group.Spaces++
		group.Files += inspection.space.Files
		expectedRefs := groupExpectedRefsByID[inspection.space.GroupID]
		if expectedRefs == nil {
			expectedRefs = CIDRefs{}
			groupExpectedRefsByID[inspection.space.GroupID] = expectedRefs
		}
		for cid, ref := range inspection.expectedRefs {
			expectedRefs[cid] += ref
		}
		cidBytes := groupCIDBytesByID[inspection.space.GroupID]
		if cidBytes == nil {
			cidBytes = map[string]uint64{}
			groupCIDBytesByID[inspection.space.GroupID] = cidBytes
		}
		for cid, size := range inspection.uniqueCIDBytes {
			cidBytes[cid] = size
		}
		if inspection.space.Status == StatusProblem {
			group.Status = StatusProblem
			group.IndexProblems += inspection.space.IndexProblems
		}
	}

	for groupID, listedSpaces := range listedSpaceIDsByGroupID {
		group := groupsByID[groupID]
		if group == nil {
			continue
		}
		for _, spaceID := range sortedMapKeys(listedSpaces) {
			if _, ok := seenSpaceIDs[spaceID]; ok {
				continue
			}
			group.Status = StatusProblem
			group.IndexProblems++
			problems = append(problems, Problem{
				Scope: problemScopeGroup,
				ID:    groupID,
				Issue: fmt.Sprintf("space %s is listed in group %s but has no space entry", spaceID, groupID),
			})
		}
	}

	for groupID, group := range groupsByID {
		expectedRefs := groupExpectedRefsByID[groupID]
		group.CIDs = uint64(len(expectedRefs))
		group.Bytes = sumCIDBytes(groupCIDBytesByID[groupID])
		groupProblems := compareCIDRefs("group", groupID, expectedRefs, groupRefsByID[groupID])
		groupProblems = append(groupProblems, compareGroupCounters(groupID, groupInfosByID[groupID], group)...)
		if len(groupProblems) > 0 {
			group.Status = StatusProblem
			group.IndexProblems += uint64(len(groupProblems))
			problems = append(problems, groupProblems...)
		}
	}

	globalProblems := inspectGlobalCIDRefs(cidEntryCache, globalSpacesByCID, cidProblemReported)
	problems = append(problems, globalProblems...)

	groupIDs := sortedMapKeys(groupsByID)
	for _, groupID := range groupIDs {
		group := *groupsByID[groupID]
		if group.Files == 0 && group.CIDs == 0 && group.Status == StatusOK {
			group.Status = StatusEmpty
		}
		inventory.Groups = append(inventory.Groups, group)
	}

	sort.Slice(inventory.Spaces, func(i, j int) bool {
		return inventory.Spaces[i].ID < inventory.Spaces[j].ID
	})
	sort.Slice(inventory.Files, func(i, j int) bool {
		if inventory.Files[i].SpaceID == inventory.Files[j].SpaceID {
			return inventory.Files[i].ID < inventory.Files[j].ID
		}
		return inventory.Files[i].SpaceID < inventory.Files[j].SpaceID
	})

	return inventory, problems, nil
}

//nolint:funlen,gocognit,nestif // Space inspection mirrors one Redis hash and keeps derived values together.
func inspectSpaceHash(
	key string,
	fields map[string]string,
	values map[string]string,
	spaceGroupByID map[string]string,
	cidEntryCache map[string]cidEntryState,
	cidProblemReported map[string]struct{},
) spaceInspection {
	spaceID := parseIndexHashID(key, redisIndexSpacePrefix)
	inspection := spaceInspection{
		space: SpaceReport{
			ID:     spaceID,
			Status: StatusOK,
		},
		expectedRefs:   CIDRefs{},
		uniqueCIDs:     map[string]struct{}{},
		uniqueCIDBytes: map[string]uint64{},
	}
	declaredFileCount := uint64(0)
	declaredCIDCount := uint64(0)
	declaredBytes := uint64(0)
	hasSpaceInfo := false

	if encodedInfo, ok := fields[redisIndexInfoField]; ok {
		entry := &indexproto.SpaceEntry{}
		if err := entry.UnmarshalVT([]byte(encodedInfo)); err != nil {
			markSpaceProblem(&inspection.space, &inspection.problems, Problem{
				Scope: problemScopeSpace,
				ID:    spaceID,
				Issue: fmt.Sprintf("space info cannot be decoded: %v", err),
			})
		} else {
			hasSpaceInfo = true
			inspection.space.GroupID = entry.GetGroupId()
			inspection.space.Limit = entry.GetLimit()
			declaredFileCount = uint64(entry.GetFileCount())
			declaredCIDCount = entry.GetCidCount()
			declaredBytes = entry.GetSize()
		}
	} else {
		markSpaceProblem(&inspection.space, &inspection.problems, Problem{
			Scope: problemScopeSpace,
			ID:    spaceID,
			Issue: "space info is missing",
		})
	}

	listedGroupID, listedInGroup := spaceGroupByID[spaceID]
	if inspection.space.GroupID == "" {
		if listedInGroup {
			inspection.space.GroupID = listedGroupID
		}
	} else {
		if listedInGroup {
			if listedGroupID != inspection.space.GroupID {
				markSpaceProblem(&inspection.space, &inspection.problems, Problem{
					Scope: problemScopeSpace,
					ID:    spaceID,
					Issue: fmt.Sprintf("space %s group mismatch: spaceInfo=%s groupEntry=%s",
						spaceID, inspection.space.GroupID, listedGroupID),
				})
			}
		} else {
			markSpaceProblem(&inspection.space, &inspection.problems, Problem{
				Scope: problemScopeSpace,
				ID:    spaceID,
				Issue: fmt.Sprintf("space %s belongs to group %s but no group lists it",
					spaceID, inspection.space.GroupID),
			})
		}
	}
	if inspection.space.GroupID == "" {
		markSpaceProblem(&inspection.space, &inspection.problems, Problem{
			Scope: problemScopeSpace,
			ID:    spaceID,
			Issue: fmt.Sprintf("space %s has no group assignment", spaceID),
		})
	}

	spaceRefs, refProblems := cidRefsFromFields("space", spaceID, fields)
	for _, problem := range refProblems {
		markSpaceProblem(&inspection.space, &inspection.problems, problem)
	}

	fieldKeys := sortedMapKeys(fields)
	for _, field := range fieldKeys {
		if !strings.HasPrefix(field, redisIndexFilePrefix) {
			continue
		}
		fileID := strings.TrimPrefix(field, redisIndexFilePrefix)
		entry := &indexproto.FileEntry{}
		if err := entry.UnmarshalVT([]byte(fields[field])); err != nil {
			markSpaceProblem(&inspection.space, &inspection.problems, Problem{
				Scope: problemScopeFile,
				ID:    fileID,
				Issue: fmt.Sprintf("file entry in space %s cannot be decoded: %v", spaceID, err),
			})
			continue
		}

		file := FileReport{
			ID:       fileID,
			SpaceID:  spaceID,
			GroupID:  inspection.space.GroupID,
			Size:     entry.GetSize(),
			CIDs:     append([]string(nil), entry.GetCids()...),
			CIDSizes: map[string]uint64{},
		}
		fileSize := uint64(0)
		fileSizeKnown := true
		for _, cid := range file.CIDs {
			inspection.expectedRefs[cid]++
			inspection.uniqueCIDs[cid] = struct{}{}
			state := loadCIDEntry(cid, values, cidEntryCache)
			if state.hasProblem {
				if _, ok := cidProblemReported[cid]; !ok {
					markSpaceProblem(&inspection.space, &inspection.problems, state.problem)
					cidProblemReported[cid] = struct{}{}
				} else {
					inspection.space.Status = StatusProblem
					inspection.space.IndexProblems++
				}
				if state.missing {
					file.MissingCIDIndex = append(file.MissingCIDIndex, cid)
					inspection.space.MissingCIDIndex++
				}
				fileSizeKnown = false
				continue
			}
			fileSize += state.entry.GetSize()
			file.CIDSizes[cid] = state.entry.GetSize()
			inspection.uniqueCIDBytes[cid] = state.entry.GetSize()
		}
		if fileSizeKnown {
			file.Size = fileSize
			if entry.GetSize() != fileSize {
				markSpaceProblem(&inspection.space, &inspection.problems, Problem{
					Scope: problemScopeFile,
					ID:    fileID,
					Issue: fmt.Sprintf("file size mismatch in space %s file %s: index=%d actual=%d",
						spaceID, fileID, entry.GetSize(), fileSize),
				})
			}
		}
		inspection.files = append(inspection.files, file)
	}

	inspection.space.Files = uint64(len(inspection.files))
	inspection.space.CIDs = uint64(len(inspection.uniqueCIDs))
	inspection.space.Bytes = sumCIDBytes(inspection.uniqueCIDBytes)

	for _, problem := range compareCIDRefs("space", spaceID, inspection.expectedRefs, spaceRefs) {
		markSpaceProblem(&inspection.space, &inspection.problems, problem)
	}
	if hasSpaceInfo {
		for _, problem := range compareSpaceCounters(
			spaceID,
			declaredFileCount,
			declaredCIDCount,
			declaredBytes,
			inspection.space,
		) {
			markSpaceProblem(&inspection.space, &inspection.problems, problem)
		}
	}
	if inspection.space.Files == 0 && inspection.space.CIDs == 0 && inspection.space.Status == StatusOK {
		inspection.space.Status = StatusEmpty
	}

	return inspection
}

func compareCIDRefs(scope string, id string, expected CIDRefs, actual CIDRefs) []Problem {
	problems := []Problem{}
	for _, cid := range sortedMapKeys(expected) {
		expectedRef := expected[cid]
		actualRef, ok := actual[cid]
		if !ok {
			problems = append(problems, Problem{
				Scope: scope,
				ID:    id,
				Issue: fmt.Sprintf("%s is missing CID ref %s", scope, cid),
			})
			continue
		}
		if actualRef != expectedRef {
			problems = append(problems, Problem{
				Scope: scope,
				ID:    id,
				Issue: fmt.Sprintf("%s CID ref mismatch for %s: index=%d actual=%d",
					scope, cid, actualRef, expectedRef),
			})
		}
	}
	for _, cid := range sortedMapKeys(actual) {
		if _, ok := expected[cid]; ok {
			continue
		}
		problems = append(problems, Problem{
			Scope: scope,
			ID:    id,
			Issue: fmt.Sprintf("%s has stale CID ref %s", scope, cid),
		})
	}
	return problems
}

func compareGroupCounters(groupID string, info *indexproto.GroupEntry, group *GroupReport) []Problem {
	problems := []Problem{}
	if info != nil {
		if info.GetCidCount() != group.CIDs {
			problems = append(problems, Problem{
				Scope: problemScopeGroup,
				ID:    groupID,
				Issue: fmt.Sprintf("group CID count mismatch: index=%d actual=%d", info.GetCidCount(), group.CIDs),
			})
		}
		if info.GetSize() != group.Bytes {
			problems = append(problems, Problem{
				Scope: problemScopeGroup,
				ID:    groupID,
				Issue: fmt.Sprintf("group byte count mismatch: index=%d actual=%d", info.GetSize(), group.Bytes),
			})
		}
	}
	if group.Limit > 0 && group.Bytes > group.Limit {
		problems = append(problems, Problem{
			Scope: problemScopeGroup,
			ID:    groupID,
			Issue: fmt.Sprintf("group byte size exceeds limit: size=%d limit=%d", group.Bytes, group.Limit),
		})
	}
	if group.AccountLimit > 0 && group.Bytes > group.AccountLimit {
		problems = append(problems, Problem{
			Scope: problemScopeGroup,
			ID:    groupID,
			Issue: fmt.Sprintf("group byte size exceeds account limit: size=%d accountLimit=%d",
				group.Bytes, group.AccountLimit),
		})
	}
	return problems
}

func compareSpaceCounters(
	spaceID string,
	declaredFileCount uint64,
	declaredCIDCount uint64,
	declaredBytes uint64,
	space SpaceReport,
) []Problem {
	problems := []Problem{}
	if declaredFileCount != space.Files {
		problems = append(problems, Problem{
			Scope: problemScopeSpace,
			ID:    spaceID,
			Issue: fmt.Sprintf("space file count mismatch: index=%d actual=%d", declaredFileCount, space.Files),
		})
	}
	if declaredCIDCount != space.CIDs {
		problems = append(problems, Problem{
			Scope: problemScopeSpace,
			ID:    spaceID,
			Issue: fmt.Sprintf("space CID count mismatch: index=%d actual=%d", declaredCIDCount, space.CIDs),
		})
	}
	if declaredBytes != space.Bytes {
		problems = append(problems, Problem{
			Scope: problemScopeSpace,
			ID:    spaceID,
			Issue: fmt.Sprintf("space byte count mismatch: index=%d actual=%d", declaredBytes, space.Bytes),
		})
	}
	if space.Limit > 0 && space.Bytes > space.Limit {
		problems = append(problems, Problem{
			Scope: problemScopeSpace,
			ID:    spaceID,
			Issue: fmt.Sprintf("space byte size exceeds limit: size=%d limit=%d", space.Bytes, space.Limit),
		})
	}
	return problems
}

func inspectGlobalCIDRefs(
	cidEntryCache map[string]cidEntryState,
	globalSpacesByCID map[string]map[string]struct{},
	cidProblemReported map[string]struct{},
) []Problem {
	problems := []Problem{}
	for _, cid := range sortedMapKeys(cidEntryCache) {
		state := cidEntryCache[cid]
		if state.hasProblem {
			if _, ok := cidProblemReported[cid]; !ok {
				problems = append(problems, state.problem)
			}
			continue
		}
		if len(globalSpacesByCID[cid]) == 0 {
			problems = append(problems, Problem{
				Scope: problemScopeCID,
				ID:    cid,
				Issue: fmt.Sprintf("global CID entry %s is not referenced by any file", cid),
			})
			continue
		}
		expectedRefs := int64(len(globalSpacesByCID[cid]))
		if int64(state.entry.GetRefs()) != expectedRefs {
			problems = append(problems, Problem{
				Scope: problemScopeCID,
				ID:    cid,
				Issue: fmt.Sprintf("global CID refs mismatch: index=%d actual=%d",
					state.entry.GetRefs(), expectedRefs),
			})
		}
	}
	return problems
}

func loadCIDEntry(cid string, values map[string]string, cache map[string]cidEntryState) cidEntryState {
	if state, ok := cache[cid]; ok {
		return state
	}
	encoded, ok := values[redisIndexCIDKey(cid)]
	if !ok {
		state := cidEntryState{
			hasProblem: true,
			missing:    true,
			problem: Problem{
				Scope: problemScopeCID,
				ID:    cid,
				Issue: fmt.Sprintf("file references CID %s but global index entry is missing", cid),
			},
		}
		cache[cid] = state
		return state
	}
	entry := &indexproto.CidEntry{}
	if err := entry.UnmarshalVT([]byte(encoded)); err != nil {
		state := cidEntryState{
			hasProblem: true,
			problem: Problem{
				Scope: problemScopeCID,
				ID:    cid,
				Issue: fmt.Sprintf("global CID entry %s cannot be decoded: %v", cid, err),
			},
		}
		cache[cid] = state
		return state
	}
	state := cidEntryState{entry: entry}
	cache[cid] = state
	return state
}

func cidRefsFromFields(scope string, id string, fields map[string]string) (CIDRefs, []Problem) {
	cids := CIDRefs{}
	problems := []Problem{}
	for field, value := range fields {
		if !strings.HasPrefix(field, redisIndexCIDPrefix) {
			continue
		}
		cid := strings.TrimPrefix(field, redisIndexCIDPrefix)
		ref, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			problems = append(problems, Problem{
				Scope: scope,
				ID:    id,
				Issue: fmt.Sprintf("%s CID ref %s cannot be decoded: %v", scope, cid, err),
			})
			continue
		}
		cids[cid] = ref
	}
	return cids, problems
}

func redisIndexCIDKey(cid string) string {
	return redisIndexCIDPrefix + cid
}

func redisIndexScanPattern(prefix string) string {
	return prefix + "*"
}

func markSpaceProblem(space *SpaceReport, problems *[]Problem, problem Problem) {
	space.Status = StatusProblem
	space.IndexProblems++
	*problems = append(*problems, problem)
}

func sumCIDBytes(cids map[string]uint64) uint64 {
	var bytes uint64
	for _, size := range cids {
		bytes += size
	}
	return bytes
}

func parseIndexHashID(key string, prefix string) string {
	value := strings.TrimPrefix(key, prefix)
	if idx := strings.Index(value, ".{"); idx >= 0 {
		return value[:idx]
	}
	return value
}

func sortedMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

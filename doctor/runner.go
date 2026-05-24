package doctor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"slices"
	"time"

	"github.com/anyproto/any-sync/nodeconf"
	"gopkg.in/yaml.v3"

	bundleconfig "github.com/grishy/any-sync-bundle/config"
)

type RuntimeRunnerConfig struct {
	BundleConfig     *bundleconfig.Config
	BundleConfigPath string
	ClientConfigPath string
	Build            BuildInfo
	Now              func() time.Time

	LoadIndexSnapshot func(ctx context.Context) (IndexSnapshot, error)
	ProbeBlocks       func(ctx context.Context, inventory Inventory) (BlockProbeResult, error)
	ProbeRuntime      func(ctx context.Context) RuntimeProbeResult
}

type RuntimeRunner struct {
	cfg RuntimeRunnerConfig
}

type RuntimeProbeResult struct {
	Mongo      HealthStatus
	Redis      HealthStatus
	RedisBloom HealthStatus
	Storage    HealthStatus
	Problems   []Problem
}

type BlockProbeResult struct {
	Checked       uint64
	MissingByFile map[string][]string
	CorruptByFile map[string][]string
	Problems      []Problem
}

func NewRuntimeRunner(cfg RuntimeRunnerConfig) *RuntimeRunner {
	return &RuntimeRunner{cfg: cfg}
}

//nolint:funlen // The output is a fixed seven-phase diagnostic transcript; splitting would hide ordering.
func (r *RuntimeRunner) RunDoctor(ctx context.Context, out io.Writer) (*Report, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}

	generatedAt := r.cfg.Now().UTC()
	report := &Report{
		GeneratedAt:      generatedAt,
		Experimental:     true,
		ReportSchema:     currentReportSchema,
		Build:            r.cfg.Build,
		BundleConfigPath: r.cfg.BundleConfigPath,
		ClientConfigPath: r.cfg.ClientConfigPath,
		Config:           buildConfigReport(r.cfg.BundleConfig),
		Verdict:          VerdictHealthy,
	}
	problems := []Problem{}

	fmt.Fprintln(out, "Experimental:")
	fmt.Fprintln(out, "  doctor output and JSON report schema may change between releases.")
	fmt.Fprintln(out)

	fmt.Fprintln(out, "[1/7] Config")
	configProblems := r.checkConfig()
	if len(configProblems) == 0 {
		fmt.Fprintln(out, "  status: ok")
	} else {
		fmt.Fprintln(out, "  status: problem")
	}
	for _, problem := range configProblems {
		fmt.Fprintf(out, "  - %s\n", problem.Issue)
	}
	problems = append(problems, configProblems...)
	fmt.Fprintln(out)

	fmt.Fprintln(out, "[2/7] Runtime")
	runtimeResult := r.cfg.ProbeRuntime(ctx)
	report.Runtime = RuntimeReport{
		Mongo:      runtimeResult.Mongo,
		Redis:      runtimeResult.Redis,
		RedisBloom: runtimeResult.RedisBloom,
		Storage:    runtimeResult.Storage,
	}
	fmt.Fprintf(out, "  mongo: %s\n", runtimeResult.Mongo)
	fmt.Fprintf(out, "  redis: %s\n", runtimeResult.Redis)
	fmt.Fprintf(out, "  redis bloom: %s\n", runtimeResult.RedisBloom)
	fmt.Fprintf(out, "  storage: %s\n\n", runtimeResult.Storage)
	problems = append(problems, runtimeResult.Problems...)
	if err := runtimePrerequisiteError(runtimeResult); err != nil {
		return nil, err
	}

	fmt.Fprintln(out, "[3/7] Network")
	networkReport, networkProblems := r.checkNetwork(ctx)
	report.Network = networkReport
	fmt.Fprintf(out, "  tcp listen: %s\n", networkReport.TCPListen)
	fmt.Fprintf(out, "  udp listen: %s\n", networkReport.UDPListen)
	fmt.Fprintf(out, "  advertised addresses: %s\n", networkReport.Advertised)
	for _, problem := range networkProblems {
		fmt.Fprintf(out, "  - %s\n", problem.Issue)
	}
	problems = append(problems, networkProblems...)
	fmt.Fprintln(out)

	fmt.Fprintln(out, "[4/7] Inventory")
	snapshot, err := r.cfg.LoadIndexSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("load filenode index snapshot: %w", err)
	}
	inventory, indexProblems, err := InspectIndexSnapshot(snapshot)
	if err != nil {
		return nil, fmt.Errorf("inspect filenode index snapshot: %w", err)
	}
	problems = append(problems, indexProblems...)
	printInventory(out, inventory)
	fmt.Fprintln(out)

	fmt.Fprintln(out, "[5/7] Spaces and index")
	printSpaces(out, inventory)
	fmt.Fprintf(out, "  checked files: %d\n", len(inventory.Files))
	fmt.Fprintf(out, "  checked cid refs: %d\n", countFileCIDRefs(inventory.Files))
	fmt.Fprintf(out, "  problems: %d\n\n", len(indexProblems))

	fmt.Fprintln(out, "[6/7] Blocks")
	blockResult, err := r.cfg.ProbeBlocks(ctx, inventory)
	if err != nil {
		return nil, fmt.Errorf("probe filenode blocks: %w", err)
	}
	blockProblems := applyBlockProbeResult(&inventory, blockResult)
	blockProblems = append(blockProblems, blockResult.Problems...)
	problems = append(problems, blockProblems...)
	fmt.Fprintf(out, "  checked blocks: %d\n", blockResult.Checked)
	fmt.Fprintf(out, "  missing blocks: %d\n", countMissingBlocks(blockResult))
	fmt.Fprintf(out, "  corrupt blocks: %d\n\n", countCorruptBlocks(blockResult))

	report.Inventory = inventory
	report.Summary = summarizeInventory(inventory)
	report.Problems = problems
	report.Verdict, report.SuggestedNextAction = classifyVerdict(inventory, problems)

	fmt.Fprintln(out, "[7/7] Report")
	path, err := WriteReportAtomic(r.cfg.BundleConfigPath, generatedAt, *report)
	if err != nil {
		return nil, err
	}
	report.ReportPath = path
	fmt.Fprintf(out, "  written: %s\n\n", path)
	printProblems(out, problems)
	fmt.Fprintln(out, "Verdict:")
	fmt.Fprintf(out, "  %s\n\n", report.Verdict)
	if report.SuggestedNextAction != "" {
		fmt.Fprintln(out, "Suggested next action:")
		fmt.Fprintf(out, "  %s\n\n", report.SuggestedNextAction)
	}
	fmt.Fprintln(out, "Report written:")
	fmt.Fprintf(out, "  %s\n", path)

	return report, nil
}

func (r *RuntimeRunner) validate() error {
	if r.cfg.BundleConfig == nil {
		return errors.New("doctor bundle config is required")
	}
	if r.cfg.BundleConfigPath == "" {
		return errors.New("doctor bundle config path is required")
	}
	if r.cfg.ClientConfigPath == "" {
		return errors.New("doctor client config path is required")
	}
	if r.cfg.Now == nil {
		return errors.New("doctor clock is required")
	}
	if r.cfg.ProbeRuntime == nil {
		return errors.New("doctor runtime probe is required")
	}
	if r.cfg.LoadIndexSnapshot == nil {
		return errors.New("doctor index snapshot loader is required")
	}
	if r.cfg.ProbeBlocks == nil {
		return errors.New("doctor block probe is required")
	}
	return nil
}

func runtimePrerequisiteError(result RuntimeProbeResult) error {
	if result.Redis == StatusProblem {
		return errors.New("redis is required for filenode inventory scan; fix Redis and run doctor again")
	}
	return nil
}

func classifyVerdict(inventory Inventory, problems []Problem) (Verdict, string) {
	if len(problems) == 0 {
		return VerdictHealthy, "No action needed."
	}
	for _, file := range inventory.Files {
		if len(file.MissingBlocks) > 0 {
			return VerdictFilesRequireClientReupload,
				"Missing blocks cannot be recreated by the server; re-upload from the original client/cache."
		}
		if len(file.CorruptBlocks) > 0 {
			return VerdictFilesRequireClientReupload,
				"Corrupt blocks cannot be trusted by the server; re-upload from the original client/cache."
		}
	}
	return VerdictProblemsFound, "Review the problem list and JSON report before changing data."
}

func (r *RuntimeRunner) checkConfig() []Problem {
	problems := []Problem{}
	if err := r.cfg.BundleConfig.Validate(); err != nil {
		problems = append(problems, Problem{
			Scope: problemScopeConfig,
			Issue: fmt.Sprintf("bundle config is invalid: %v", err),
		})
	}
	// #nosec G703 -- Doctor reads the local client config path selected by the running bundle.
	if _, statErr := os.Stat(r.cfg.ClientConfigPath); statErr != nil {
		problems = append(problems, Problem{
			Scope: problemScopeConfig,
			Issue: fmt.Sprintf("client config cannot be read: %v", statErr),
		})
	} else {
		matchErr := checkClientConfigMatches(r.cfg.BundleConfig, r.cfg.ClientConfigPath)
		if matchErr != nil {
			problems = append(problems, Problem{
				Scope: problemScopeConfig,
				Issue: fmt.Sprintf("client config does not match current bundle config: %v", matchErr),
			})
		}
	}
	return problems
}

func checkClientConfigMatches(cfg *bundleconfig.Config, path string) error {
	expectedData, err := cfg.YamlClientConfig()
	if err != nil {
		return fmt.Errorf("generate expected client config: %w", err)
	}
	// #nosec G703 -- Doctor compares the local client config path selected by the running bundle.
	actualData, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var expected nodeconf.Configuration
	if decodeExpectedErr := yaml.Unmarshal(expectedData, &expected); decodeExpectedErr != nil {
		return fmt.Errorf("decode expected client config: %w", decodeExpectedErr)
	}
	var actual nodeconf.Configuration
	if decodeActualErr := yaml.Unmarshal(actualData, &actual); decodeActualErr != nil {
		return fmt.Errorf("decode current client config: %w", decodeActualErr)
	}

	if actual.Id != expected.Id {
		return fmt.Errorf("config id: current=%q expected=%q", actual.Id, expected.Id)
	}
	if actual.NetworkId != expected.NetworkId {
		return fmt.Errorf("network id: current=%q expected=%q", actual.NetworkId, expected.NetworkId)
	}
	if len(actual.Nodes) != len(expected.Nodes) {
		return fmt.Errorf("node count: current=%d expected=%d", len(actual.Nodes), len(expected.Nodes))
	}
	for idx := range expected.Nodes {
		actualNode := actual.Nodes[idx]
		expectedNode := expected.Nodes[idx]
		if actualNode.PeerId != expectedNode.PeerId {
			return fmt.Errorf("node[%d] peer id: current=%q expected=%q", idx, actualNode.PeerId, expectedNode.PeerId)
		}
		if !slices.Equal(actualNode.Addresses, expectedNode.Addresses) {
			return fmt.Errorf(
				"node[%d] addresses: current=%v expected=%v",
				idx,
				actualNode.Addresses,
				expectedNode.Addresses,
			)
		}
		if !slices.Equal(actualNode.Types, expectedNode.Types) {
			return fmt.Errorf("node[%d] types: current=%v expected=%v", idx, actualNode.Types, expectedNode.Types)
		}
	}
	return nil
}

func (r *RuntimeRunner) checkNetwork(ctx context.Context) (NetworkReport, []Problem) {
	report := buildNetworkReport(r.cfg.BundleConfig)
	problems := []Problem{}
	if err := checkTCPListener(ctx, r.cfg.BundleConfig.Network.ListenTCPAddr); err != nil {
		report.TCPListen = StatusProblem
		problems = append(problems, Problem{
			Scope: problemScopeNetwork,
			Issue: fmt.Sprintf("tcp listener check failed: %v", err),
		})
	} else {
		report.TCPListen = StatusOK
	}
	udpStatus, err := checkUDPListener(r.cfg.BundleConfig.Network.ListenUDPAddr)
	if err != nil {
		report.UDPListen = StatusProblem
		problems = append(problems, Problem{
			Scope: problemScopeNetwork,
			Issue: fmt.Sprintf("udp listener check failed: %v", err),
		})
	} else {
		report.UDPListen = udpStatus
	}
	if len(r.cfg.BundleConfig.ExternalAddr) == 0 {
		report.Advertised = StatusProblem
		problems = append(problems, Problem{
			Scope: problemScopeNetwork,
			Issue: "no advertised addresses configured",
		})
	} else {
		report.Advertised = StatusOK
	}
	return report, problems
}

func buildConfigReport(cfg *bundleconfig.Config) ConfigReport {
	report := ConfigReport{
		ConfigID:            cfg.ConfigID,
		NetworkID:           cfg.NetworkID,
		PeerID:              cfg.Account.PeerId,
		MongoCoordinatorURI: redactedURI(cfg.Coordinator.MongoConnect),
		MongoConsensusURI:   redactedURI(cfg.Consensus.MongoConnect),
		RedisURI:            redactedURI(cfg.FileNode.RedisConnect),
		StoragePath:         cfg.StoragePath,
		AdvertisedAddresses: append([]string(nil), cfg.ExternalAddr...),
	}
	if cfg.FileNode.S3 != nil {
		report.S3 = &S3Report{
			Bucket:         cfg.FileNode.S3.Bucket,
			Endpoint:       redactedURI(cfg.FileNode.S3.Endpoint),
			Region:         cfg.FileNode.S3.Region,
			ForcePathStyle: cfg.FileNode.S3.ForcePathStyle,
		}
	}
	return report
}

func buildNetworkReport(cfg *bundleconfig.Config) NetworkReport {
	return NetworkReport{
		ListenTCPAddr:       cfg.Network.ListenTCPAddr,
		ListenUDPAddr:       cfg.Network.ListenUDPAddr,
		AdvertisedAddresses: append([]string(nil), cfg.ExternalAddr...),
	}
}

func redactedURI(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "<invalid>"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func printInventory(out io.Writer, inventory Inventory) {
	fmt.Fprintf(out, "  accounts/groups: %d\n", len(inventory.Groups))
	for _, group := range inventory.Groups {
		fmt.Fprintf(out, "    - %s spaces=%d files=%d cids=%d bytes=%d status=%s\n",
			group.ID, group.Spaces, group.Files, group.CIDs, group.Bytes, group.Status)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  spaces: %d\n", len(inventory.Spaces))
	for _, space := range inventory.Spaces {
		fmt.Fprintf(out, "    - %s account=%s files=%d cids=%d bytes=%d status=%s\n",
			space.ID, space.GroupID, space.Files, space.CIDs, space.Bytes, space.Status)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  files: %d\n", len(inventory.Files))
	fmt.Fprintf(out, "  cids: %d\n", countUniqueFileCIDs(inventory.Files))
}

func printSpaces(out io.Writer, inventory Inventory) {
	for _, space := range inventory.Spaces {
		fmt.Fprintf(out, "  space %s\n", space.ID)
		fmt.Fprintf(out, "    files: %d\n", space.Files)
		fmt.Fprintf(out, "    cids: %d\n", space.CIDs)
		if space.MissingCIDIndex > 0 {
			fmt.Fprintf(out, "    index: missing %d cid entries\n", space.MissingCIDIndex)
		} else {
			if space.IndexProblems > 0 {
				fmt.Fprintf(out, "    index: problems %d\n", space.IndexProblems)
			} else {
				fmt.Fprintln(out, "    index: ok")
			}
		}
		if space.MissingBlocks == 0 {
			if space.CIDs == 0 {
				fmt.Fprintln(out, "    blocks: none")
			} else {
				fmt.Fprintln(out, "    blocks: ok")
			}
		} else {
			fmt.Fprintf(out, "    blocks: missing %d\n", space.MissingBlocks)
		}
		fmt.Fprintln(out)
	}
}

func printProblems(out io.Writer, problems []Problem) {
	if len(problems) == 0 {
		fmt.Fprintln(out, "Problems:")
		fmt.Fprintln(out, "  none")
		fmt.Fprintln(out)
		return
	}
	fmt.Fprintln(out, "Problems:")
	for idx, problem := range problems {
		fmt.Fprintf(out, "  %d. %s", idx+1, problem.Scope)
		if problem.ID != "" {
			fmt.Fprintf(out, " %s", problem.ID)
		}
		fmt.Fprintln(out)
		fmt.Fprintf(out, "     issue: %s\n", problem.Issue)
		if problem.Recoverable != "" {
			fmt.Fprintf(out, "     recoverable: %s\n", problem.Recoverable)
		}
	}
	fmt.Fprintln(out)
}

func applyBlockProbeResult(inventory *Inventory, result BlockProbeResult) []Problem {
	if len(result.MissingByFile) == 0 && len(result.CorruptByFile) == 0 {
		return nil
	}
	filesByKey := map[string]*FileReport{}
	spacesByID := map[string]*SpaceReport{}
	for idx := range inventory.Files {
		file := &inventory.Files[idx]
		filesByKey[fileReportKey(file.SpaceID, file.ID)] = file
	}
	for idx := range inventory.Spaces {
		space := &inventory.Spaces[idx]
		spacesByID[space.ID] = space
	}

	problems := []Problem{}
	for key, cids := range result.MissingByFile {
		file := filesByKey[key]
		if file == nil {
			continue
		}
		file.MissingBlocks = append(file.MissingBlocks, cids...)
		space := spacesByID[file.SpaceID]
		if space != nil {
			space.MissingBlocks += uint64(len(cids))
			space.Status = StatusProblem
		}
		problems = append(problems, Problem{
			Scope:       "file",
			ID:          file.ID,
			Issue:       fmt.Sprintf("%d referenced blocks are missing", len(cids)),
			Recoverable: "requires original client/cache",
		})
	}
	for key, cids := range result.CorruptByFile {
		file := filesByKey[key]
		if file == nil {
			continue
		}
		file.CorruptBlocks = append(file.CorruptBlocks, cids...)
		space := spacesByID[file.SpaceID]
		if space != nil {
			space.CorruptBlocks += uint64(len(cids))
			space.Status = StatusProblem
		}
		problems = append(problems, Problem{
			Scope:       "file",
			ID:          file.ID,
			Issue:       fmt.Sprintf("%d referenced blocks are corrupt", len(cids)),
			Recoverable: "requires original client/cache",
		})
	}
	return problems
}

func summarizeInventory(inventory Inventory) Summary {
	var bytes uint64
	var cids uint64
	for _, space := range inventory.Spaces {
		bytes += space.Bytes
		cids += space.CIDs
	}
	return Summary{
		Groups: uint64(len(inventory.Groups)),
		Spaces: uint64(len(inventory.Spaces)),
		Files:  uint64(len(inventory.Files)),
		CIDs:   cids,
		Bytes:  bytes,
	}
}

func countFileCIDRefs(files []FileReport) uint64 {
	var count uint64
	for _, file := range files {
		count += uint64(len(file.CIDs))
	}
	return count
}

func countUniqueFileCIDs(files []FileReport) uint64 {
	seen := map[string]struct{}{}
	for _, file := range files {
		for _, cid := range file.CIDs {
			seen[cid] = struct{}{}
		}
	}
	return uint64(len(seen))
}

func countMissingBlocks(result BlockProbeResult) uint64 {
	var count uint64
	for _, cids := range result.MissingByFile {
		count += uint64(len(cids))
	}
	return count
}

func countCorruptBlocks(result BlockProbeResult) uint64 {
	var count uint64
	for _, cids := range result.CorruptByFile {
		count += uint64(len(cids))
	}
	return count
}

func fileReportKey(spaceID string, fileID string) string {
	return spaceID + "\x00" + fileID
}

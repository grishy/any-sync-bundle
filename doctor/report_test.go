package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReportPathUsesDoctorDirectoryNextToBundleConfig(t *testing.T) {
	generatedAt := time.Date(2026, 5, 22, 14, 33, 10, 0, time.UTC)

	got := ReportPath("/data/bundle-config.yml", generatedAt)
	want := "/data/doctor/doctor_2026-05-22T14-33-10Z.json"

	if got != want {
		t.Fatalf("ReportPath() = %q, want %q", got, want)
	}
}

func TestSocketPathUsesBundleConfigDirectory(t *testing.T) {
	got := SocketPath("/data/bundle-config.yml")
	want := "/data/bundle.sock"

	if got != want {
		t.Fatalf("SocketPath() = %q, want %q", got, want)
	}
}

func TestWriteReportAtomicCreatesDoctorDirectoryAndWritesJSON(t *testing.T) {
	dir := t.TempDir()
	bundleConfigPath := filepath.Join(dir, "bundle-config.yml")
	generatedAt := time.Date(2026, 5, 22, 14, 33, 10, 0, time.UTC)
	report := &Report{
		GeneratedAt:      generatedAt,
		BundleConfigPath: bundleConfigPath,
		Verdict:          VerdictHealthy,
		Summary: Summary{
			Groups: 2,
			Spaces: 13,
			Files:  452,
			CIDs:   12934,
		},
	}

	path, err := WriteReportAtomic(bundleConfigPath, generatedAt, *report)
	if err != nil {
		t.Fatalf("WriteReportAtomic() error = %v", err)
	}

	wantPath := filepath.Join(dir, "doctor", "doctor_2026-05-22T14-33-10Z.json")
	if path != wantPath {
		t.Fatalf("WriteReportAtomic() path = %q, want %q", path, wantPath)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}

	var got Report
	if unmarshalErr := json.Unmarshal(raw, &got); unmarshalErr != nil {
		t.Fatalf("unmarshal report: %v", unmarshalErr)
	}
	if got.ReportPath != path {
		t.Fatalf("report path in JSON = %q, want %q", got.ReportPath, path)
	}
	if got.Verdict != VerdictHealthy {
		t.Fatalf("report verdict = %q, want %q", got.Verdict, VerdictHealthy)
	}
	if got.Summary.Spaces != 13 {
		t.Fatalf("report spaces = %d, want 13", got.Summary.Spaces)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat report: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("report mode = %o, want 600", info.Mode().Perm())
	}

	tmpMatches, err := filepath.Glob(filepath.Join(dir, "doctor", "*.tmp"))
	if err != nil {
		t.Fatalf("glob tmp files: %v", err)
	}
	if len(tmpMatches) != 0 {
		t.Fatalf("unexpected tmp files left behind: %v", tmpMatches)
	}
}

func TestWriteReportAtomicDoesNotOverwriteSameTimestampReport(t *testing.T) {
	dir := t.TempDir()
	bundleConfigPath := filepath.Join(dir, "bundle-config.yml")
	generatedAt := time.Date(2026, 5, 22, 14, 33, 10, 0, time.UTC)
	report := &Report{
		GeneratedAt: generatedAt,
		Verdict:     VerdictHealthy,
	}

	firstPath, err := WriteReportAtomic(bundleConfigPath, generatedAt, *report)
	if err != nil {
		t.Fatalf("first WriteReportAtomic() error = %v", err)
	}
	secondPath, err := WriteReportAtomic(bundleConfigPath, generatedAt, *report)
	if err != nil {
		t.Fatalf("second WriteReportAtomic() error = %v", err)
	}

	if firstPath == secondPath {
		t.Fatalf("second report overwrote first path %q", firstPath)
	}
	for _, path := range []string{firstPath, secondPath} {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("report %s does not exist: %v", path, statErr)
		}
	}
}

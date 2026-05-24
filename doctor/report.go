package doctor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	reportDirectoryName = "doctor"
	reportFileMode      = 0o600
	reportDirMode       = 0o750
)

func ReportPath(bundleConfigPath string, generatedAt time.Time) string {
	name := fmt.Sprintf("doctor_%s.json", generatedAt.UTC().Format("2006-01-02T15-04-05Z"))
	return filepath.Join(filepath.Dir(bundleConfigPath), reportDirectoryName, name)
}

func SocketPath(bundleConfigPath string) string {
	return filepath.Join(filepath.Dir(bundleConfigPath), "bundle.sock")
}

func WriteReportAtomic(bundleConfigPath string, generatedAt time.Time, report Report) (string, error) {
	path, err := nextReportPath(ReportPath(bundleConfigPath, generatedAt))
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if mkdirErr := os.MkdirAll(dir, reportDirMode); mkdirErr != nil {
		return "", fmt.Errorf("create doctor report directory: %w", mkdirErr)
	}

	tmpPath := filepath.Join(dir, "."+filepath.Base(path)+".tmp")
	tmp, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, reportFileMode)
	if err != nil {
		return "", fmt.Errorf("create temporary report file: %w", err)
	}
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	report.ReportPath = path
	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if encodeErr := encoder.Encode(report); encodeErr != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("encode doctor report: %w", encodeErr)
	}
	if chmodErr := tmp.Chmod(reportFileMode); chmodErr != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("chmod doctor report: %w", chmodErr)
	}
	if closeErr := tmp.Close(); closeErr != nil {
		return "", fmt.Errorf("close doctor report: %w", closeErr)
	}
	if renameErr := os.Rename(tmpPath, path); renameErr != nil {
		return "", fmt.Errorf("publish doctor report: %w", renameErr)
	}

	removeTmp = false
	return path, nil
}

func nextReportPath(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return path, nil
		}
		return "", fmt.Errorf("check doctor report path: %w", err)
	}

	extension := filepath.Ext(path)
	stem := strings.TrimSuffix(path, extension)
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s_%d%s", stem, suffix, extension)
		if _, err := os.Stat(candidate); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return candidate, nil
			}
			return "", fmt.Errorf("check doctor report path: %w", err)
		}
	}
}

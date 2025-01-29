package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// getHostOS returns detailed information about the operating system
func getHostOS() (string, error) {
	switch runtime.GOOS {
	case "linux":
		// Content example:
		// PRETTY_NAME="Ubuntu 23.04"
		// NAME="Ubuntu"
		// VERSION_ID="23.04"
		data, err := os.ReadFile("/etc/os-release")
		if err != nil {
			return "", err
		}

		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\""), nil
			}
		}

		return "Linux", nil
	case "darwin":
		if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
			return fmt.Sprintf("macOS %s", strings.TrimSpace(string(out))), nil
		}

		return "macOS", nil
	default:
		// I will skip Windows for god sake, if I will see someone using Windows then I will add it
		return runtime.GOOS, nil
	}
}

// getHostMem returns the total system memory
func getHostMem() (string, error) {
	var totalBytes uint64

	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return "", err
		}

		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) == 3 && fields[2] == "kB" {
					if kb, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
						totalBytes = kb * 1024
					}
				}
				break
			}
		}

	case "darwin":
		if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
			if bytes, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64); err == nil {
				totalBytes = bytes
			}
		}
	}

	if totalBytes > 0 {
		return fmt.Sprintf("%d MB", totalBytes/1024/1024), nil
	}

	return "", errors.New("unable to determine memory")
}

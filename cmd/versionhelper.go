package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

// versionPrinter prints the application version and build, host information to attached later to an issue.
func versionPrinter(c *cli.Context) {
	valueOrError := func(value string, err error) string {
		if err != nil {
			return fmt.Sprintf("unknown (%s)", err)
		}
		return value
	}

	hostname := valueOrError(os.Hostname())
	osInfo := valueOrError(getHostOS())
	hostMemory := valueOrError(getHostMem())

	fmt.Println(c.App.Name)
	fmt.Printf("Version:   %s\n", version)
	fmt.Printf("Commit:    %s\n", commit)
	fmt.Printf("Date:      %s\n", date)
	fmt.Printf("Hostname:  %s\n", hostname)
	fmt.Printf("OS:        %s\n", osInfo)
	fmt.Printf("GoVersion: %s\n", runtime.Version())
	fmt.Printf("Platform:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("NumCPU:    %d\n", runtime.NumCPU())
	fmt.Printf("Memory:    %s\n", hostMemory)
}

// getHostOS returns detailed information about the operating system.
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

		for line := range strings.SplitSeq(string(data), "\n") {
			if after, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
				return strings.Trim(after, "\""), nil
			}
		}

		return "Linux", nil
	case "darwin":
		if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
			return fmt.Sprintf("macOS %s", strings.TrimSpace(string(out))), nil
		}

		return "macOS", nil
	default:
		// I will skip Windows for god sake, if I will see someone using Windows then I will add it.
		return runtime.GOOS, nil
	}
}

// getHostMem returns the total system memory.
func getHostMem() (string, error) {
	var totalBytes uint64

	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return "", err
		}

		for line := range strings.SplitSeq(string(data), "\n") {
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

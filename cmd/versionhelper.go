package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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

	fmt.Println(c.App.Name)
	fmt.Printf("Version:   %s\n", version)
	fmt.Printf("Commit:    %s\n", commit)
	fmt.Printf("Date:      %s\n", date)
	fmt.Printf("Hostname:  %s\n", hostname)
	fmt.Printf("OS:        %s\n", osInfo)
	fmt.Printf("GoVersion: %s\n", runtime.Version())
	fmt.Printf("Platform:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("NumCPU:    %d\n", runtime.NumCPU())
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
		if out, err := exec.CommandContext(context.Background(), "sw_vers", "-productVersion").Output(); err == nil {
			return fmt.Sprintf("macOS %s", strings.TrimSpace(string(out))), nil
		}

		return "macOS", nil
	default:
		// I will skip Windows for god sake, if I will see someone using Windows then I will add it.
		return runtime.GOOS, nil
	}
}

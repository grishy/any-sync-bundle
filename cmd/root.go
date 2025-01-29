package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/urfave/cli/v2"
)

const (
	appName = "any-sync-bundle"

	// CLI global flags
	flagDebug        = "debug"
	flagDataDir      = "data-dir"
	flagConfig       = "config"
	flagClientConfig = "client-config"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var log = logger.NewNamed("cli")

func Root(ctx context.Context) *cli.App {
	cli.VersionPrinter = versionPrinter

	// For any-sync package, used in network communication but just for info
	// Yes, this is global
	// TODO: Create task to avoid it, use app instance.
	app.AppName = appName
	app.GitSummary = version
	app.GitCommit = commit
	app.BuildDate = date

	cliApp := &cli.App{
		Name:    appName,
		Usage:   "A TODO",
		Version: version,
		Description: `
		TODO
		`,
		Authors: []*cli.Author{{
			Name:  "Sergei G.",
			Email: "mail@grishy.dev",
		}},
		// Global flags, before any subcommand
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    flagDebug,
				Value:   false,
				Usage:   "Enable debug mode with detailed logging",
				EnvVars: []string{"ANY_SYNC_BUNDLE_DEBUG"},
			},
			&cli.PathFlag{
				Name:    flagConfig,
				Aliases: []string{"c"},
				Value:   "./data/config.yml",
				EnvVars: []string{"ANY_SYNC_BUNDLE_CONFIG"},
				Usage:   "Path to the bundle configuration YAML file (must be readable)",
			},
			&cli.PathFlag{
				Name:    flagClientConfig,
				Aliases: []string{"cc"},
				// TODO: Anytype support only yml, but not yaml
				Value:   "./data/client-config.yml",
				EnvVars: []string{"ANY_SYNC_BUNDLE_CLIENT_CONFIG"},
				Usage:   "Path where wtite to the Anytype client configuration YAML file (must be readable)",
			},
			&cli.PathFlag{
				Name:    flagDataDir,
				Aliases: []string{"d"},
				Value:   "./data/store/",
				EnvVars: []string{"ANY_SYNC_BUNDLE_DATA"},
				Usage:   "Path to the bundle data directory (must be writable)",
			},
		},
		Before: setupLogger,
		Commands: []*cli.Command{
			createConfig(ctx),
			initMongoReplica(ctx),
			start(ctx),
		},
	}

	return cliApp
}

// setupLogger configures the global logger with appropriate settings
func setupLogger(c *cli.Context) error {
	anyLogCfg := logger.Config{
		Production:   false,
		DefaultLevel: "",
		Format:       logger.PlaintextOutput,
	}

	if c.Bool(flagDebug) {
		anyLogCfg.DefaultLevel = "debug"
		anyLogCfg.Format = logger.ColorizedOutput
	}

	anyLogCfg.ApplyGlobal()

	return nil
}

// versionPrinter prints the application version and build, host information to attached later to ticket.
func versionPrinter(c *cli.Context) {
	valueOrError := func(value string, err error) string {
		if err != nil {
			return fmt.Sprintf("unknown (%s)", err)
		}
		return value
	}

	hostname := valueOrError(os.Hostname())
	osInfo := valueOrError(getOSInfo())
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

// getOSInfo returns detailed information about the operating system.
func getOSInfo() (string, error) {
	switch runtime.GOOS {
	case "linux":
		// Content example:
		// PRETTY_NAME="Ubuntu 23.04"
		// NAME="Ubuntu"
		// VERSION_ID="23.04"
		data, err := os.ReadFile("/etc/os-release")
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\""), nil
				}
			}
		}
		return "Linux", nil

	case "darwin":
		if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
			return fmt.Sprintf("macOS %s", strings.TrimSpace(string(out))), nil
		}
		return "macOS", nil

	case "windows":
		// NOTE: NOT verified on real machine
		if out, err := exec.Command("ver").Output(); err == nil {
			return strings.TrimSpace(string(out)), nil
		}
		return "Windows", nil

	default:
		return runtime.GOOS, nil
	}
}

// getHostMem returns the total amount of memory available on the host system.
func getHostMem() (string, error) {
	var totalBytes uint64

	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return "unknown", fmt.Errorf("failed to read /proc/meminfo: %w", err)
		}

		for _, line := range strings.Split(string(data), "\n") {
			if !strings.HasPrefix(line, "MemTotal:") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) != 3 || fields[2] != "kB" {
				return "unknown", fmt.Errorf("unexpected meminfo format, got: %q", line)
			}
			kb, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return "unknown", fmt.Errorf("failed to parse memory value %q: %w", fields[1], err)
			}
			totalBytes = kb * 1024 // Convert KB to bytes
			break
		}

	case "darwin":
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err != nil {
			return "unknown", fmt.Errorf("failed to execute sysctl command: %w", err)
		}
		totalBytes, err = strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
		if err != nil {
			return "unknown", fmt.Errorf("failed to parse sysctl output %q: %w", string(out), err)
		}

	default:
		return "unknown", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	mb := totalBytes / 1024 / 1024
	return fmt.Sprintf("%d MB", mb), nil
}

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/grishy/any-sync-bundle/config"
)

// Lifecycle events are log fields that tests and operators wait for.
func emitBundleEvent(event string, fields ...zap.Field) {
	fields = append(fields, zap.String("event", event))
	log.Info("bundle lifecycle event", fields...)
}

func printWelcomeMsg() {
	fmt.Printf(`
┌───────────────────────────────────────────────────────────────────┐

                 Welcome to the AnySync Bundle!
           https://github.com/grishy/any-sync-bundle

    Version: %s
    Commit:  %s
    Built:   %s

`, version, commit, date)

	fmt.Println(" Based on these components:")
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, mod := range info.Deps {
			if strings.HasPrefix(mod.Path, "github.com/anyproto/any-sync") {
				fmt.Printf(" ‣ %s (%s)\n", mod.Path, mod.Version)
			}
		}
	}
	fmt.Print(`
└───────────────────────────────────────────────────────────────────┘
`)
}

func printStartupMsg() {
	fmt.Printf(`
┌───────────────────────────────────────────────────────────────────┐

                      AnySync Bundle is ready!
                      All services are running.
                   Press Ctrl+C to stop services.

└───────────────────────────────────────────────────────────────────┘
`)
}

func reportShutdownComplete() {
	emitBundleEvent(bundleShutdownCompleteEvent)
	fmt.Printf(`
┌───────────────────────────────────────────────────────────────────┐

                 AnySync Bundle shutdown complete!
                     All services are stopped.

└───────────────────────────────────────────────────────────────────┘
`)
	log.Info("→ Goodbye!")
}

func printConfigurationInfo(cfg *config.Config) {
	log.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Info("Configuration Summary")
	log.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Info("→ Network Configuration",
		zap.String("tcp_listen", cfg.Network.ListenTCPAddr),
		zap.String("udp_listen", cfg.Network.ListenUDPAddr))
	log.Info("→ MongoDB Configuration",
		zap.String("coordinator_uri", cfg.Coordinator.MongoConnect),
		zap.String("coordinator_db", cfg.Coordinator.MongoDatabase),
		zap.String("consensus_uri", cfg.Consensus.MongoConnect),
		zap.String("consensus_db", cfg.Consensus.MongoDatabase))
	log.Info("→ Redis Configuration",
		zap.String("filenode_uri", cfg.FileNode.RedisConnect))
	log.Info("→ External Addresses",
		zap.Strings("addresses", cfg.ExternalAddr))
	log.Info("→ Node Identity",
		zap.String("peer_id", cfg.Account.PeerId))
	log.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

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

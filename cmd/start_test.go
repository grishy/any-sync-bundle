package cmd

import (
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// Every documented container must outlive the process watchdog so the
// application, rather than Docker, owns forced shutdown and reports the
// resulting failure.
func TestShutdownTimeoutFitsContainerStopGracePeriod(t *testing.T) {
	composePaths := []string{
		"../compose.aio.yml",
		"../compose.external.yml",
		"../compose.s3.yml",
		"../compose.traefik.yml",
	}

	for _, composePath := range composePaths {
		data, err := os.ReadFile(composePath)
		if err != nil {
			t.Fatalf("read %s: %v", composePath, err)
		}

		var compose struct {
			Services map[string]struct {
				StopGracePeriod string `yaml:"stop_grace_period"`
			} `yaml:"services"`
		}
		if unmarshalErr := yaml.Unmarshal(data, &compose); unmarshalErr != nil {
			t.Fatalf("parse %s: %v", composePath, unmarshalErr)
		}

		bundle, ok := compose.Services["any-sync-bundle"]
		if !ok {
			t.Fatalf("%s does not define the any-sync-bundle service", composePath)
		}
		if bundle.StopGracePeriod == "" {
			t.Fatalf("%s does not bound the bundle stop grace period", composePath)
		}

		containerStopGracePeriod, err := time.ParseDuration(bundle.StopGracePeriod)
		if err != nil {
			t.Fatalf("parse stop_grace_period in %s: %v", composePath, err)
		}
		if ShutdownTimeout >= containerStopGracePeriod {
			t.Fatalf(
				"%s stop grace period %v must exceed shutdown timeout %v",
				composePath,
				containerStopGracePeriod,
				ShutdownTimeout,
			)
		}
	}
}

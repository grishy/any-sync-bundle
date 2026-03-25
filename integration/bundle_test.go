//go:build integration

package integration

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBundleWaitReady_ProcessDiesBeforeReady(t *testing.T) {
	cmd := exec.Command("sh", "-c", "echo 'bundle crashed early'; exit 1")

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)

	bp := &BundleProcess{
		cmd:      cmd,
		output:   &strings.Builder{},
		waitDone: make(chan struct{}),
	}

	require.NoError(t, cmd.Start())

	go bp.captureOutput(stdout)
	go bp.captureOutput(stderr)
	go bp.waitForExit()

	err = bp.WaitReady(3 * time.Second)
	require.Error(t, err)
	require.ErrorContains(t, err, "exited before ready")
	require.ErrorContains(t, err, "bundle crashed early")
}

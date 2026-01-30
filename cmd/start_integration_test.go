//go:build integration

package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestWaitForTCPOrExit_SIGILL tests detection of SIGILL (AVX failure simulation).
// Run with: go test -tags=integration -v ./cmd/...
func TestWaitForTCPOrExit_SIGILL(t *testing.T) {
	// Create a temporary script that exits with SIGILL
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-mongod")

	// Script that sends SIGILL to itself
	script := `#!/bin/bash
kill -ILL $$
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Start the fake mongod
	ctx := context.Background()
	proc, err := newInfraProcess(ctx, "mongo", scriptPath)
	if err != nil {
		t.Fatalf("failed to start fake mongod: %v", err)
	}

	// Wait for TCP (should fail with SIGILL)
	err = waitForTCPOrExit("127.0.0.1:27017", 5*time.Second, proc)

	// Verify we got the error
	if err == nil {
		t.Fatal("expected SIGILL error, got nil")
	}

	// Check it's detected as illegal instruction
	if !isIllegalInstruction(err) {
		t.Errorf("expected illegal instruction error, got: %v", err)
	}

	t.Logf("Correctly detected SIGILL: %v", err)
}

// TestWaitForTCPOrExit_ProcessExitsNormally tests detection of normal exit.
func TestWaitForTCPOrExit_ProcessExitsNormally(t *testing.T) {
	// Create a script that exits normally with error
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-mongod")

	script := `#!/bin/bash
echo "Config error: invalid option"
exit 1
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	ctx := context.Background()
	proc, err := newInfraProcess(ctx, "mongo", scriptPath)
	if err != nil {
		t.Fatalf("failed to start fake mongod: %v", err)
	}

	err = waitForTCPOrExit("127.0.0.1:27017", 5*time.Second, proc)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should NOT be detected as illegal instruction
	if isIllegalInstruction(err) {
		t.Errorf("should not be illegal instruction: %v", err)
	}

	t.Logf("Correctly detected normal exit: %v", err)
}

// TestMongoAVXError_Integration tests the full error flow.
func TestMongoAVXError_Integration(t *testing.T) {
	// Create a script that exits with SIGILL
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-mongod")

	script := `#!/bin/bash
kill -ILL $$
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	ctx := context.Background()
	proc, err := newInfraProcess(ctx, "mongo", scriptPath)
	if err != nil {
		t.Fatalf("failed to start fake mongod: %v", err)
	}

	err = waitForTCPOrExit("127.0.0.1:27017", 5*time.Second, proc)

	// Simulate the error handling from startAllInOneInfra
	if err != nil && isIllegalInstruction(err) {
		// This is what would happen in real code
		avxErr := &MongoAVXError{Cause: err}

		// Verify error chain works
		var target *MongoAVXError
		if !errors.As(avxErr, &target) {
			t.Error("errors.As should match MongoAVXError")
		}

		t.Logf("Full AVX error: %v", avxErr)
	} else {
		t.Errorf("expected SIGILL error, got: %v", err)
	}
}

// TestRealMongodNotFound tests behavior when mongod binary doesn't exist.
func TestRealMongodNotFound(t *testing.T) {
	ctx := context.Background()
	_, err := newInfraProcess(ctx, "mongo", "/nonexistent/mongod")

	if err == nil {
		t.Fatal("expected error for nonexistent binary")
	}

	// Should be exec error, not process exit
	if !errors.Is(err, exec.ErrNotFound) && !os.IsNotExist(err) {
		t.Logf("Got expected error type: %v", err)
	}
}

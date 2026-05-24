package cmd

import (
	"context"
	"flag"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestRootIncludesDoctorCommand(t *testing.T) {
	app := Root(context.Background())

	if app.Command("doctor") == nil {
		t.Fatal("Root() does not include doctor command")
	}
}

func TestPrepareBundleConfigKeepsRuntimePaths(t *testing.T) {
	dir := t.TempDir()
	bundleConfigPath := filepath.Join(dir, "bundle-config.yml")
	clientConfigPath := filepath.Join(dir, "client-config.yml")
	storagePath := filepath.Join(dir, "storage")
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, cliFlag := range buildStartFlags() {
		if err := cliFlag.Apply(flagSet); err != nil {
			t.Fatalf("apply flag: %v", err)
		}
	}
	if err := flagSet.Set(flagStartBundleConfigPath, bundleConfigPath); err != nil {
		t.Fatalf("set bundle config path: %v", err)
	}
	if err := flagSet.Set(flagStartClientConfigPath, clientConfigPath); err != nil {
		t.Fatalf("set client config path: %v", err)
	}
	if err := flagSet.Set(flagStartStoragePath, storagePath); err != nil {
		t.Fatalf("set storage path: %v", err)
	}
	cCtx := cli.NewContext(cli.NewApp(), flagSet, nil)

	prepared, err := prepareBundleConfig(cCtx)
	if err != nil {
		t.Fatalf("prepareBundleConfig() error = %v", err)
	}

	if prepared.Config == nil {
		t.Fatal("prepared Config is nil")
	}
	if prepared.BundleConfigPath != bundleConfigPath {
		t.Fatalf("bundle config path = %q, want %q", prepared.BundleConfigPath, bundleConfigPath)
	}
	if prepared.ClientConfigPath != clientConfigPath {
		t.Fatalf("client config path = %q, want %q", prepared.ClientConfigPath, clientConfigPath)
	}
}

func TestPrepareBundleConfigStoresAbsoluteRuntimePaths(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, cliFlag := range buildStartFlags() {
		if err := cliFlag.Apply(flagSet); err != nil {
			t.Fatalf("apply flag: %v", err)
		}
	}
	if err := flagSet.Set(flagStartBundleConfigPath, "data/bundle-config.yml"); err != nil {
		t.Fatalf("set bundle config path: %v", err)
	}
	if err := flagSet.Set(flagStartClientConfigPath, "data/client-config.yml"); err != nil {
		t.Fatalf("set client config path: %v", err)
	}
	cCtx := cli.NewContext(cli.NewApp(), flagSet, nil)

	prepared, err := prepareBundleConfig(cCtx)
	if err != nil {
		t.Fatalf("prepareBundleConfig() error = %v", err)
	}

	wantBundleConfigPath := filepath.Join(dir, "data", "bundle-config.yml")
	if prepared.BundleConfigPath != wantBundleConfigPath {
		t.Fatalf("bundle config path = %q, want %q",
			prepared.BundleConfigPath, wantBundleConfigPath)
	}
	wantClientConfigPath := filepath.Join(dir, "data", "client-config.yml")
	if prepared.ClientConfigPath != wantClientConfigPath {
		t.Fatalf("client config path = %q, want %q",
			prepared.ClientConfigPath, wantClientConfigPath)
	}
}

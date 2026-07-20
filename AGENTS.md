# any-sync-bundle repository guide

## Mission and ownership

`any-sync-bundle` is a light process wrapper around the upstream Anytype
coordinator, consensus, filenode, and sync applications.

This repository owns orchestration, configuration conversion, shared-network
wiring, filenode store selection, and the local BadgerDB adapter. Preserve
upstream application behavior and lifecycle contracts. Do not copy or replace
an upstream service when a narrow adapter or upstream change is enough.

MinIO is an S3-compatible integration dependency, not another storage backend.
Configured S3 storage uses upstream `s3store`; local storage uses BadgerDB.

## Repository map

- `main.go` owns process signals and the final shutdown watchdog.
- `cmd/` owns the CLI, bundle lifecycle, embedded-process supervision, and
  MongoDB replica-set initialization.
- `config/` owns YAML boundaries and conversion to upstream node configs.
- `lightnode/anynodes.go` composes the four upstream applications, their shared
  network components, and the filenode store.
- `lightcmp/lightfilenodestore/` implements the local BadgerDB store.
- `integration/` and `compose.*.yml` prove and document Docker-backed system
  boundaries.
- `README.md` and `CONTRIBUTING.md` are the operator and developer workflow
  sources; keep their commands aligned with CI.

## Runtime invariants

- All services share the coordinator's PeerID, network stack, and DRPC mux on
  TCP 33010 and QUIC/UDP 33020.
- Service order is coordinator, consensus, filenode, then sync. Initialize every
  application before running any application so the network cannot read the
  shared DRPC mux while handlers are still being registered. Shut down in
  reverse order.
- The root context represents process lifetime. A process signal, startup
  failure, service run failure, or unexpected embedded-process exit cancels it.
  Preserve independent startup, runtime, and cleanup errors; an operator stop is
  successful only when cleanup adds no failure.
- All-in-one mode owns MongoDB and Redis for their full lifetime. An unexpected
  exit fails the bundle. Intentional shutdown sends SIGTERM to every child
  before waiting, then forces and reaps deadline survivors.
- `cmd.ShutdownTimeout` is the application-owned aggregate shutdown bound.
  Service, infrastructure, integration, watchdog, and bundle Compose timing must
  derive from or be checked against that policy rather than duplicate it.

## Configuration and data

Configuration bootstrap has one order:

1. Load an existing bundle YAML when present.
2. Otherwise create it with `config.CreateWrite` from flags and environment.
3. Regenerate the client configuration on every start.

Validate persisted configuration at this boundary without rewriting
operator-owned values. Validate MongoDB URIs with the driver, not only
`net/url`. When creating a config, preserve the coordinator URI and add a path
separator only to the derived consensus URI when the bundle adds query options.

With default flags, durable paths are `./data/bundle-config.yml`,
`./data/storage/network-store/`, `./data/storage/storage-sync/`, and
`./data/storage/storage-file/`. The generated client config is
`./data/client-config.yml`. All-in-one infrastructure uses `/data/mongo` and
`/data/redis`. Treat the bundle config as sensitive because it contains
credentials and keys.

Complete MongoDB and Redis URI logging is an explicit current project decision.
Do not change it as incidental cleanup; revisit it only for an explicit
security or privacy requirement.

## Change discipline

- Read the active diff before editing. Preserve unrelated staged, unstaged, and
  untracked work.
- Keep changes inside the repository-owned boundaries. Keep lifecycle order and
  resource ownership visible instead of introducing a generic framework around
  upstream applications.
- Test repository-owned contracts, not dependency internals. Prefer a few
  boundary and failure-path tests over broad cross-products.
- Treat `go.mod` as the Go-version source of truth. When changing it, keep the
  Dockerfile and GitHub workflows aligned and verify that Nix provides the same
  toolchain.

## Verification

Use focused tests while iterating. Run `gofmt -w` on each changed Go file.
Before presenting a Go change as ready, run these commands from the repository
root:

```bash
golangci-lint run ./...
go test -count=1 -race -shuffle=on -vet=all -failfast ./...
go build -o /tmp/any-sync-bundle .
git diff --check
```

Do not use `golangci-lint --fix` as a verification command because it mutates
the reviewed source.

For configuration, startup, shutdown, storage, Docker, or integration changes,
also run the Docker-dependent integration suite:

```bash
go test -count=1 -tags=integration -timeout=10m ./integration/...
```

For Go toolchain, Nix, dependency, or release-build changes, also run:

```bash
nix flake check --print-build-logs
nix build -L .#default
```

Documentation-only changes need `git diff --check` plus a direct review of every
changed command, path, link, and behavioral claim. Run broader gates when the
documentation changes an executable contract.

A change is ready only when the narrow regression proof and every applicable
broader gate pass from the final source state. Report the commands actually run
and any verification that could not be completed.

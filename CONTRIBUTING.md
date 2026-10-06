# Contributing

Thanks for your interest in improving any-sync-bundle! This document explains how to build, test, and propose changes.

## Development Setup

### Prerequisites

- Go 1.27.1 or later
- Docker (optional, for testing with containers)
- golangci-lint (for linting)

### Build and Test

```sh
# Build
go build -o any-sync-bundle .
./any-sync-bundle --version
./any-sync-bundle --help

# Run linter
golangci-lint run ./...

# Run tests
go test -count=1 -race -shuffle=on -vet=all -failfast ./...
git diff --check
```

### Local Dependencies

For a quick local stack, see `compose.dev.yml` (MongoDB replica set + Redis Stack):

```sh
docker compose -f compose.dev.yml up -d
```

### With Nix

[Nix](https://nixos.org/) provides reproducible builds and a complete development environment.

```sh
nix flake check --print-build-logs
nix build -L .#default
./result/bin/any-sync-bundle --version
```

### Dependency Updates

Use the latest [compatible service set](https://puppetdoc.anytype.io/api/v1/prod-any-sync-compatible-versions/),
not independently selected `@latest` service versions. Update only the service
modules already required by `go.mod`; other matrix entries are not instructions
to add services to the bundle. Review the shared `any-sync` version required by
those services, including lifecycle, configuration, and storage changes.

1. Update the service requirements and the `// Timestamp:` comment in `go.mod`
   to the selected compatibility entry.
2. Run `go mod tidy` and `go mod verify`. Recalculate `vendorHash` in `flake.nix`
   for the resulting dependencies; keep the Go toolchain aligned if it changes.
3. Run every command in **Build and Test** and **With Nix** above. Lint without
   `--fix` so verification does not change the reviewed source.
4. Run the Docker integration suite (required for dependency/lifecycle updates):

   ```sh
   go test -count=1 -tags=integration -timeout=10m ./integration/...
   ```

Before release, also exercise an existing-data upgrade with a current client:
object/file sync, permissions, restart, and reconnect. The startup and direct S3
integration checks do not establish client compatibility. Use a stopped data
snapshot for upgrade/rollback testing, not the only copy of production data.
Update the release references below using the selected timestamp's UTC date.

## Running Locally

- **All-in-one container** (bundled MongoDB/Redis): see `compose.aio.yml`
- **Minimal container** (external MongoDB/Redis): see `compose.external.yml`
- **Binary** (no container): use `start-bundle` and supply your own MongoDB/Redis

## Pull Requests

- Keep PRs small and focused
- Include a brief description of what changed and why
- Update README/docs when behavior or flags/env variables change
- Run linter and tests before submitting

By submitting a PR, you agree that your contribution is licensed under the repository's MIT license.

## Release Process

1. Update the version in README, Compose image tags, and `flake.nix`. The
   compatibility date is derived in UTC from the `go.mod` timestamp selected
   during the dependency update, not from a new compatibility-matrix response.

   Generate the version from the repository root (bash on macOS):

   ```bash
   VERSION=v1.6.1
   ANYTYPE_UNIX_TIMESTAMP=$(awk -F '"' '/Timestamp:/ {print $2}' go.mod)
   ANYTYPE_FORMATTED=$(date -u -r "$ANYTYPE_UNIX_TIMESTAMP" +%Y-%m-%d)
   FINAL_VERSION="$VERSION-$ANYTYPE_FORMATTED"
   echo "$FINAL_VERSION"
   # v1.6.1-2026-10-05
   ```

2. Run the repository checks above, then check release packaging locally:

   ```sh
   goreleaser check
   goreleaser release --snapshot --clean
   ```

3. Commit the reviewed changes and wait for CI to pass on that commit. Create
   and push an annotated tag using the version generated above:

   ```bash
   git tag -a "$FINAL_VERSION" -m "Release $FINAL_VERSION"
   git push origin "refs/tags/$FINAL_VERSION"
   ```

The workflow creates a GitHub draft with GoReleaser's generated changelog,
then builds both versioned Docker images. After both builds succeed, it updates
`minimal` and `latest` and publishes the GitHub release. The two alias updates
are separate operations, not an atomic switch.

### Version Format

`v[bundle-version]-[anytype-compatibility-date]`

- `v1.6.1` – Bundle's semantic version (SemVer)
- `YYYY-MM-DD` – UTC date of the compatibility timestamp recorded in `go.mod`.
  The [upstream matrix](https://puppetdoc.anytype.io/api/v1/prod-any-sync-compatible-versions/)
  is consulted during dependency updates, not again during publication.

---

Good luck and have fun!

# Contributing

Thanks for your interest in improving any-sync-bundle! This document explains how to build, test, and propose changes.

## Development Setup

### Prerequisites

- Go 1.25.2 or later
- Docker (optional, for testing with containers)
- golangci-lint (for linting)

### Build and Test

```sh
# Build
go build -o any-sync-bundle .
./any-sync-bundle --version

# Run linter
golangci-lint run --fix

# Run tests
go test -race -shuffle=on -vet=all -failfast ./...
```

### Local Dependencies

For a quick local stack, see `compose.dev.yml` (MongoDB replica set + Redis Stack):

```sh
docker compose -f compose.dev.yml up -d
```

### With Nix

[Nix](https://nixos.org/) provides reproducible builds and a complete development environment.

```sh
nix build
./result/bin/any-sync-bundle --version
nix flake check
```

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

For maintainers releasing a new version.

### 1. Check locally

```sh
goreleaser release --snapshot --clean
```

### 2. Create and push tag

```sh
# Set variables (fish shell)
set VERSION v1.3.0
set ANYTYPE_UNIX_TIMESTAMP 1769866558
set ANYTYPE_FORMATTED (date -r $ANYTYPE_UNIX_TIMESTAMP +'%Y-%m-%d')
set FINAL_VERSION $VERSION-$ANYTYPE_FORMATTED

# Create tag and push
git tag -a $FINAL_VERSION -m "Release $FINAL_VERSION"
git push origin tag $FINAL_VERSION
```

### Version Format

`v[bundle-version]-[anytype-compatibility-date]`

- `v1.3.0` – Bundle's semantic version (SemVer)
- `2026-01-31` – Anytype any-sync compatibility version from [anytype.io](https://puppetdoc.anytype.io/api/v1/prod-any-sync-compatible-versions/)

---

Good luck and have fun!

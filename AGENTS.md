## Project Overview

Any-Sync-Bundle is a self-hosting solution for Anytype server that bundles multiple Anytype sync services into a single binary. It merges any-sync-consensusnode, any-sync-coordinator, any-sync-filenode, and any-sync-node into one deployable unit, simplifying deployment compared to the official multi-container setup.

## Build and Development Commands

### Build

```bash
go build -o any-sync-bundle .
```

### Test

```bash
go test ./...
# Run specific test
go test ./lightcmp/lightfilenodestore -v
```

### Lint

```bash
golangci-lint run
# Auto-fix formatting issues
golangci-lint run --fix
```

### Run

```bash
# Start all services
./any-sync-bundle start

# Generate configurations
./any-sync-bundle config bundle
./any-sync-bundle config client

# MongoDB management
./any-sync-bundle mongo start
./any-sync-bundle mongo stop
```

## Architecture

The bundle integrates four Anytype sync services:

- **Consensus Node** (`lightnode/consensus.go`): Provides consensus service functionality
- **Coordinator** (`lightnode/coordinator.go`): Coordinates sync operations
- **File Node** (`lightnode/filenode.go`): Handles file storage and retrieval
- **Sync Node** (`lightnode/sync.go`): Core sync functionality

### Key Components

1. **CLI Structure** (`cmd/`):

   - `root.go`: Main CLI setup using urfave/cli
   - `start.go`: Service startup logic with graceful shutdown
   - `config.go`: Configuration generation for bundle and client
   - `mongo.go`: MongoDB lifecycle management
   - `versionhelper.go`: Version tracking with Anytype compatibility dates

2. **Configuration** (`config/`):

   - `bundle.go`: Bundle configuration model and loader
   - Uses YAML format for both bundle and client configurations
   - Supports environment variables with `ANY_SYNC_BUNDLE_` prefix

3. **Light Components** (`lightcmp/`):
   - `lightfilenodestore/`: BadgerDB-based file storage implementation
   - Replaces MinIO for lightweight deployments

## Service Port Mapping

- TCP ports: 33010-33013 (various sync services)
- UDP ports: 33020-33023 (QUIC transport)

## Important Configuration Patterns

When modifying configurations:

1. Bundle config is at `./data/bundle-config.yml`
2. Client config is generated at `./data/client-config.yml`
3. Storage directory is at `./data/storage/`
4. External addresses must be set via `ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS`

## Linting Configuration

The project uses golangci-lint v2.4.0 with extensive checks enabled (see `.golangci.yml`). Key requirements:

- Max line length: 120 characters
- Cyclomatic complexity limit: 30
- Function length limit: 100 lines, 50 statements
- All imports must be grouped with local packages (`github.com/grishy/any-sync-bundle`) separate
- Strict error checking and exhaustive switch/map checks enabled

## Testing Approach

- Unit tests exist primarily for `lightcmp/lightfilenodestore`
- Tests use standard Go testing package
- Mock generation available via `moq` tool (configured in go.mod)

## Version Management

Bundle versions follow format: `vX.Y.Z+YYYY-MM-DD`

- `vX.Y.Z`: Bundle semver
- `YYYY-MM-DD`: Anytype compatibility date from puppetdoc.anytype.io

## Development Notes

- The project replaces the official multi-container setup with a single binary
- Uses embedded BadgerDB instead of external MongoDB/Redis for light mode
- Supports both all-in-one container and minimal deployments
- Main branch is under active development; use releases for stability

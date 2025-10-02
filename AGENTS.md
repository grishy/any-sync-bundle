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

The bundle integrates four Anytype sync services into a single binary:

- **Coordinator** (`lightnode/anynodes.go:NewCoordinatorApp`): Network coordination, MUST start first
- **Consensus Node** (`lightnode/anynodes.go:NewConsensusApp`): Consensus service functionality
- **File Node** (`lightnode/anynodes.go:NewFileNodeApp`): File storage using BadgerDB (not S3/MinIO)
- **Sync Node** (`lightnode/anynodes.go:NewSyncApp`): Core sync functionality

### Critical: Service Startup Order

**ORDER MATTERS!** Services must start in this exact order:
1. **Coordinator** - MUST start first (provides shared network stack for all services)
2. **Consensus, File, Sync** - start after coordinator (reuse its network stack)

The coordinator creates the network infrastructure (TCP/UDP listeners, DRPC mux, connection pool).
Other services extract and reuse these components via `SharedNetworkManager`.

### Shared Network Architecture

**Key Innovation**: Instead of each service creating its own network stack, all services share ONE network infrastructure:

```
Coordinator App (Primary)
├── Creates network stack
│   ├── yamux (TCP listener on port 33010)
│   ├── quic (UDP listener on port 33020)
│   ├── server (DRPC multiplexer)
│   ├── pool (connection pool)
│   └── peerservice (peer management)
└── Registers CoordinatorService RPC handlers

SharedNetworkManager
├── Extracts network components from coordinator
└── Provides them to secondary services

Secondary Apps (Consensus, FileNode, Sync)
├── Receive shared network via SharedNetworkManager
├── Register their own RPC handlers to shared DRPC mux
└── Service-specific components only
```

**Benefits**:
- Single port pair (33010 TCP, 33020 UDP) instead of 8 ports
- Shared connection pool - all services reuse peer connections
- Reduced memory usage - one network stack instead of four
- True service bundling

**Implementation** (`lightnode/sharednetwork.go`):
- `ExtractSharedNetwork(app)` - extracts network components from coordinator
- `RegisterToApp(app)` - injects shared components into secondary services

**Startup Flow** (`cmd/start.go`):
1. Create coordinator app (full network stack)
2. Start coordinator
3. Extract shared network via `ExtractSharedNetwork(coordinatorApp)`
4. Create secondary apps with `sharedNet` parameter
5. Start secondary apps (they reuse coordinator's network)

**Backward Compatibility**: Services can still run standalone by passing `nil` for `sharedNet` parameter.

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

**New Shared Network Architecture** (default):
- TCP port: 33010 (ALL services share ONE listener)
- UDP port: 33020 (ALL services share ONE listener)

All services (coordinator, consensus, filenode, sync) register their RPC handlers to a single DRPC multiplexer:
- Coordinator: `/CoordinatorService/*`
- Consensus: `/ConsensusService/*`
- FileNode: `/FileService/*`
- SyncNode: `/SpaceSyncService/*`

These method namespaces don't conflict, so they coexist in one mux.

## Configuration Patterns

### File Locations
1. Bundle config: `./data/bundle-config.yml` (generated via `config bundle`)
2. Client config: `./data/client-config.yml` (auto-generated on first start)
3. Storage directories:
   - Sync: `./data/storage-sync/` (uses `AnyStorePath`)
   - File: `./data/storage-file/` (BadgerDB storage)
   - Network stores: `./data/network-store/{coordinator,consensus,filenode,sync}/`

### Environment Variables
- `ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS` - External addresses for client connections
- `ANY_SYNC_BUNDLE_INIT_MONGO_URI` - MongoDB connection string
- `ANY_SYNC_BUNDLE_INIT_REDIS_URI` - Redis connection string
- All follow pattern: `ANY_SYNC_BUNDLE_*`

### Config Generation Flow
1. User runs `./any-sync-bundle config bundle` (or `start` auto-generates)
2. Creates `bundle-config.yml` with network keys, accounts, listen addresses
3. On `start`, generates `client-config.yml` from bundle config
4. All services receive config via `NodeConfigs()` converter

## Linting Configuration

The project uses golangci-lint v2.4.0 with extensive checks enabled (see `.golangci.yml`). Key requirements:

- Max line length: 120 characters
- Cyclomatic complexity limit: 30
- Function length limit: 100 lines, 50 statements
- All imports must be grouped with local packages (`github.com/grishy/any-sync-bundle`) separate
- Strict error checking and exhaustive switch/map checks enabled

## Testing & Quality Assurance

### Test Commands
```bash
# Full test suite with race detector
go generate ./... && \
  golangci-lint fmt ./... && \
  golangci-lint run ./... && \
  go test -race -shuffle=on -vet=all -failfast ./...
```

### Test Coverage
- Unit tests: `lightcmp/lightfilenodestore` (BadgerDB storage)
- Integration tests: None yet (services tested via manual start)
- Mock generation: Available via `moq` tool

### Quality Checks
- **Linting:** golangci-lint v2.4.0 (extensive checks enabled)
- **Formatting:** gofumpt, goimports, golines (auto-fix via `--fix`)
- **Race detection:** Always run tests with `-race` flag
- **Vet checks:** Enabled via `-vet=all`

## Version Management

Bundle versions follow format: `vX.Y.Z+YYYY-MM-DD`

- `vX.Y.Z`: Bundle semver
- `YYYY-MM-DD`: Anytype compatibility date from puppetdoc.anytype.io

## Common Pitfalls & Debugging

### Issue: Sync Node Hangs on Startup
**Cause:** Missing `AnyStorePath` in storage config or wrong startup order
**Fix:** Ensure `AnyStorePath` is set and coordinator starts first

### Issue: Services Can't Connect to Coordinator
**Cause:** Coordinator started after other services
**Fix:** Check startup order in `cmd/start.go` - coordinator MUST be first

### Issue: File Storage Not Working
**Cause:** Missing `stat.New()` component in filenode
**Fix:** Verify `lightnode/anynodes.go:NewFileNodeApp` includes `filenodeStat.New()`

### Issue: Unexpected Storage Path Access
**Cause:** Code accidentally using `Storage.Path` instead of `Storage.AnyStorePath`
**Fix:** Will fail immediately with fuse path `/dev/null/oldstorage-not-used` - this is intentional!

## Reference: Service Dependencies

```
External Dependencies:
├── MongoDB (coordinator, consensus)
└── Redis (filenode)

Internal Service Dependencies:
Coordinator (no deps)
├── Consensus Node → needs Coordinator
├── File Node → needs Coordinator + Redis
└── Sync Node → needs Coordinator
```

## Comparison: Bundle vs Docker-Compose

| Feature | Docker-Compose | Bundle |
|---------|---------------|--------|
| Deployment | Multiple containers | Single binary |
| File Storage | MinIO (S3) | BadgerDB |
| Config Discovery | MongoDB (dynamic) | YAML files (static) |
| Bootstrap | `any-sync-confapply` | Not needed |
| Migration | Supported | Not supported |
| Use Case | Production clusters | Self-hosting, development |

## Key Architectural Decisions

### 1. **No Legacy Migration Support**

**Components NOT included** (present in original any-sync-node):
- `oldstorage.New()` - Legacy BadgerDB storage format
- `migrator.New()` - Migrates from old to new storage format

**Why:** Bundle is for NEW installations only. No backward compatibility needed.

**Impact:** Config field `Storage.Path` is unused (fuse set to `/dev/null/oldstorage-not-used`)

### 2. **Storage Configuration**

```go
Storage: nodestorage.Config{
    Path:         "/dev/null/oldstorage-not-used", // FUSE - will fail if accessed
    AnyStorePath: "./data/storage-sync",           // Actually used
}
```

- `nodestorage` uses ONLY `AnyStorePath` (see `nodestorage/storageservice.go:284`)
- `oldstorage` uses ONLY `Path` (NOT included in bundle)
- `Path` set to invalid fuse path - will fail immediately if accidentally accessed
- This is defensive programming - catches bugs early

### 3. **No Bootstrap Step**

**What docker-compose does:** Runs `any-sync-confapply` to save network config to MongoDB

**What bundle does:** Passes network config directly via YAML to all services + generates `client-config.yml`

**Why:** Simpler architecture, no dynamic discovery needed. All services get config at startup.

**Impact:** `coordinatorNodeconfsource` component registered but MongoDB collection remains empty

### 4. **Component Registration Order is CRITICAL**

**How any-sync framework initializes components:**
1. All `Register()` calls happen in order
2. Each component's `Init(a *app.App)` is called sequentially
3. During `Init()`, components call `a.MustComponent("name")` to get dependencies
4. `MustComponent()` PANICS if the dependency wasn't registered earlier

**This means:** Components can only access previously registered components!

**Common dependency pattern across all services:**
```
Foundation → Configuration → Security & Transport → Network Services → Storage & Logic
```

**Key constraint:** `secureservice` MUST be registered BEFORE `yamux`/`quic`
- Both yamux and quic call `a.MustComponent("secureservice")` in their Init()
- If secureservice isn't registered first, application will PANIC on startup

See `lightnode/anynodes.go` for detailed comments explaining each service's dependency chain.

### 5. **Component Registration Patterns**

Each service has specific component requirements:
- **Sync Node:** Needs `nodestorage.New()` but NOT `oldstorage` or `migrator`
- **File Node:** Needs `stat.New()` for statistics tracking
- **Coordinator:** Needs `coordinatorNodeconfsource.New()` (framework requirement)
- **All Services:** Need `yamux.New()` and `quic.New()` for transport (AFTER secureservice!)

### 6. **Storage Implementation**

- **File Node:** Uses `lightfilenodestore` (BadgerDB) instead of `s3store` (S3/MinIO)
- **Sync Node:** Uses `nodestorage` (any-store format) directly
- **Coordinator/Consensus:** Use MongoDB (external dependency)

## Development Notes

- **Single binary** replaces multi-container docker-compose setup
- **New installations only** - no migration from existing deployments
- **External dependencies:** MongoDB + Redis (required), no MinIO needed
- **Network config:** Embedded in YAML, not dynamically fetched
- **Main branch:** Active development, use releases for stability

## Project Overview

Any-Sync-Bundle is a self-hosting solution that bundles four Anytype sync services (coordinator, consensus, filenode, sync) into a single binary. This simplifies deployment from the official 6+ container setup to one binary + MongoDB + Redis.

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

- **Coordinator** (`lightnode/anynodes.go:newCoordinatorApp`): Network coordination, MUST start first
- **Consensus Node** (`lightnode/anynodes.go:newConsensusApp`): Consensus service functionality
- **File Node** (`lightnode/anynodes.go:newFileNodeApp`): File storage using BadgerDB (not S3/MinIO)
- **Sync Node** (`lightnode/anynodes.go:newSyncApp`): Core sync functionality
- **Admin UI** (`adminui/`): Web interface for user and space management (port 8080)

### Shared Network and Account

**Key Innovation**: All services share ONE network stack and ONE account (peer ID):

```
Coordinator (Primary)
├── Creates network stack (TCP 33010, UDP 33020, DRPC mux, connection pool)
├── Creates peer identity (coordinator-peer-id)
└── Registers CoordinatorService RPC handlers

Secondary Services (Consensus, FileNode, Sync)
├── Extract shared components via extractSharedNetwork()
├── Reuse coordinator's peer identity
└── Register their RPC handlers to shared DRPC mux
```

**Why share everything?**
1. **Physical constraint**: One TCP/UDP listener = one TLS identity
2. **DRPC routing**: Routes by method path (`/ConsensusService/*`), not peer ID
3. **Framework support**: Single node can have multiple NodeTypes

**Network topology**:
```yaml
nodes:
  - peerId: coordinator-peer-id
    addresses: [127.0.0.1:33010, quic://127.0.0.1:33020]
    types: [Coordinator, Consensus, Tree, File]
```

**Startup sequence** (`cmd/start.go`):
1. Create coordinator app (full network + identity)
2. **Start coordinator**
3. **Wait 2 seconds** (coordinator initialization)
4. Extract shared network via `extractSharedNetwork(coordinatorApp)`
5. Create secondary apps with shared network
6. Start secondary apps

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

4. **Admin UI** (`adminui/`):
   - Web-based administration interface using templ
   - User management, quota control, space monitoring
   - Fixed pagination with 50 items per page
   - Efficient MongoDB aggregation for scalable queries

## Network Ports and RPC Routes

**Ports** (all services share):
- TCP: 33010
- UDP: 33020

**Admin UI**:
- HTTP: 8080 (web interface)

**DRPC method namespaces** (multiplexed on shared network):
- `/CoordinatorService/*` → coordinator
- `/ConsensusService/*` → consensus
- `/FileService/*` → filenode
- `/SpaceSyncService/*` → sync

## Admin UI

The Admin UI provides a web interface for managing the any-sync-bundle deployment:

### Features
- User search and management
- Storage quota administration
- Space monitoring and filtering
- Deletion log tracking
- System statistics dashboard

### Performance Optimizations
- **Fixed pagination**: All listings use 50 items per page
- **Efficient aggregation**: User listings use MongoDB `$facet` to reduce queries from O(n) to O(1)
- **Database-level pagination**: Uses `$skip` and `$limit` in MongoDB instead of in-memory filtering

### Security Considerations
⚠️ **WARNING**: The current AdminUI implementation lacks authentication. In production:
- Run behind a reverse proxy with authentication
- Restrict network access to trusted IPs only
- Consider implementing OAuth2 or basic auth

### Access
```bash
# After starting the bundle
open http://localhost:8080/admin
```

## Configuration

### Default File Locations
```
./data/
├── bundle-config.yml       # Generated via 'config bundle' or auto-generated on first 'start'
├── client-config.yml       # Auto-generated on first 'start'
├── storage-sync/           # Sync node data (AnyStorePath)
├── storage-file/           # File node data (BadgerDB)
└── network-store/          # Network state for each service
    ├── coordinator/
    ├── consensus/
    ├── filenode/
    └── sync/
```

### Environment Variables
All follow `ANY_SYNC_BUNDLE_*` pattern:
- `ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS` - External addresses for clients
- `ANY_SYNC_BUNDLE_INIT_MONGO_URI` - MongoDB connection string
- `ANY_SYNC_BUNDLE_INIT_REDIS_URI` - Redis connection string

### Config Flow
1. `./any-sync-bundle start` checks for `bundle-config.yml`
2. If missing, creates it with generated keys/accounts
3. Generates `client-config.yml` for Anytype client
4. Converts to service-specific configs via `NodeConfigs()`

## Code Quality

### Linting (golangci-lint v2.4.0)
- Max line: 120 chars
- Max complexity: 30
- Max function: 100 lines/50 statements
- Imports grouped: stdlib → external → local

### Testing
```bash
# Full test suite
go generate ./... && golangci-lint fmt ./... && golangci-lint run ./... && go test -race -shuffle=on -vet=all -failfast ./...
```

**Current coverage**:
- ✅ Unit tests: `lightcmp/lightfilenodestore`
- ❌ Integration tests: Manual only (TODO)

## Version Management

Bundle versions follow format: `vX.Y.Z+YYYY-MM-DD`

- `vX.Y.Z`: Bundle semver
- `YYYY-MM-DD`: Anytype compatibility date from puppetdoc.anytype.io

### Automated Version Tracking

Compatible versions are tracked in `go.mod` (lines 7-17) from https://puppetdoc.anytype.io/api/v1/prod-any-sync-compatible-versions/

**CI Workflow:** `.github/workflows/version-check.yml` runs weekly to detect new Anytype releases and auto-creates GitHub issues with update instructions when available.

## Troubleshooting

### Sync Node Hangs on Startup

**Symptoms**: Bundle starts coordinator successfully but hangs when starting sync node

**Root causes**:
1. ❌ Coordinator not fully initialized before sync node starts
2. ❌ Missing `NodeSync.SyncOnStart=true` or `PeriodicSyncHours=0` in config
3. ❌ `Storage.AnyStorePath` not accessible or writable

**Fix**:
```bash
# 1. Check startup includes delay after coordinator (should see this log):
# "waiting for coordinator to fully initialize before starting dependent services"

# 2. Verify config has sync settings (config/convert.go sets these by default):
# NodeSync.SyncOnStart: true
# NodeSync.PeriodicSyncHours: 2

# 3. Check storage path exists and is writable:
ls -la ./data/storage-sync/

# 4. Enable debug logging to see where it hangs:
./any-sync-bundle start --debug
```

The 2-second delay after coordinator startup (added in `cmd/start.go:151`) prevents this issue.

### AdminUI Performance Issues

**Symptoms**: Slow page loads when viewing all users with many identities

**Root cause**: Inefficient pagination fetching all identities then paginating in memory

**Fix**: The AdminUI now uses MongoDB aggregation pipeline with `$facet` for efficient pagination:
- Before: 51 queries (1 Distinct + 50 individual)
- After: 1 aggregation query
- Result: O(1) query complexity regardless of user count

### Services Can't Connect to Each Other

**Cause**: Wrong startup order - coordinator MUST start first

**Fix**: Check `cmd/start.go` - services array must be `[coordinator, consensus, filenode, sync]`

### Storage Path Errors

**Expected behavior**: If you see `/dev/null/oldstorage-not-used` in errors, this is a **defensive feature**
- Bundle doesn't support legacy storage migration
- `Storage.Path` field is deliberately set to invalid path
- Only `Storage.AnyStorePath` should be used

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

## Key Design Decisions

### 1. New Installations Only

**Excluded components** (present in original any-sync-node):
- `oldstorage.New()` - legacy storage format
- `migrator.New()` - migration tool

**Reason**: Simpler codebase, no backward compatibility burden

**Impact**: `Storage.Path` unused (set to `/dev/null/oldstorage-not-used` as defensive fuse)

### 2. No Bootstrap Step

- **Docker-compose**: Runs `any-sync-confapply` to save network config to MongoDB
- **Bundle**: Passes config directly via YAML, generates `client-config.yml`

**Benefit**: Simpler, no dynamic discovery needed

### 3. Component Registration Order Matters

The any-sync framework initializes components sequentially:
1. `Register()` adds components in order
2. `Init()` called on each component
3. Component can only access previously registered dependencies via `MustComponent()`

**Critical**: `secureservice` MUST come before `yamux`/`quic` or app panics

See `lightnode/anynodes.go` for full dependency chains.

### 4. Storage Choices

- **File Node**: BadgerDB (`lightfilenodestore`) instead of S3/MinIO
- **Sync Node**: any-store format (`nodestorage`)
- **Coordinator/Consensus**: MongoDB

## Development Notes

- **Single binary** replaces multi-container docker-compose setup
- **New installations only** - no migration from existing deployments
- **External dependencies:** MongoDB + Redis (required), no MinIO needed
- **Network config:** Embedded in YAML, not dynamically fetched
- **Main branch:** Active development, use releases for stability

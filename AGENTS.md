## Overview

- `any-sync-bundle` wraps the Anytype coordinator, consensus, filenode, and sync services into one binary (`lightnode/anynodes.go`).
- All services share the coordinator's network stack: TCP 33010, QUIC/UDP 33020, one PeerID, one DRPC mux.
- The filenode supports two storage backends (auto-selected based on configuration):
  - **BadgerDB** (default): Local embedded storage via `lightcmp/lightfilenodestore`
  - **S3** (optional): Cloud storage via upstream `s3store` implementation
- External dependencies: MongoDB for coordinator/consensus, Redis for filenode cache. Sync node persists to AnyStore on disk.

Config bootstrap (cmd/start.go):

1. Load existing bundle YAML if present.
2. Otherwise create one via `config.CreateWrite`, injecting values from env/flags.
3. Always write the client config (`YamlClientConfig`) to the target path.

## Architecture Notes

- Coordinator starts first, then consensus, filenode, sync (`runBundleServices`).
- `extractSharedNetwork` copies network components from the coordinator into other apps.
- DRPC routes by method prefix (`/CoordinatorService`, `/ConsensusService`, `/FileService`, `/SpaceSyncService`).
- Data layout (default `./data`):

## Development

### Compose files

- `compose.dev.yml` – development dependencies (MongoDB replica set + Redis Stack).
- `compose.aio.yml` – bundle image with embedded MongoDB/Redis.
- `compose.external.yml` – bundle image plus external MongoDB and Redis containers.

```bash
go build -o any-sync-bundle .
golangci-lint run --fix
go test -race -shuffle=on -vet=all -failfast ./...
```

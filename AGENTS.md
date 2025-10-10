## Overview

- `any-sync-bundle` wraps the Anytype coordinator, consensus, filenode, and sync services into one binary (`lightnode/anynodes.go`).
- All services share the coordinator’s network stack: TCP 33010, QUIC/UDP 33020, one PeerID, one DRPC mux.
- The filenode uses `lightcmp/lightfilenodestore` (BadgerDB) instead of the upstream S3 implementation.
- External dependencies: MongoDB for coordinator/consensus, Redis for filenode. Sync node persists to AnyStore on disk.

## CLI (cmd/root.go)

| Command | Notes |
| ------- | ----- |
| `start-all-in-one` | Launch bundle plus embedded `mongod` + `redis-server` (see `cmd/start.go`). |
| `start-bundle` | Launch bundle only; uses URIs from config or env. |

Global flags / env vars:

- `--bundle-config` (`ANY_SYNC_BUNDLE_CONFIG`, default `./data/bundle-config.yml`)
- `--client-config` (`ANY_SYNC_BUNDLE_CLIENT_CONFIG`, default `./data/client-config.yml`)
- `--storage` (`ANY_SYNC_BUNDLE_STORAGE`, default `./data/storage/`)
- `--initial-external-addrs` (`ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS`)
- `--initial-mongo-uri` (`ANY_SYNC_BUNDLE_INIT_MONGO_URI`)
- `--initial-redis-uri` (`ANY_SYNC_BUNDLE_INIT_REDIS_URI`)
- `--debug`, `--log-level`

Config bootstrap (cmd/start.go):

1. Load existing bundle YAML if present.
2. Otherwise create one via `config.CreateWrite`, injecting values from env/flags.
3. Always write the client config (`YamlClientConfig`) to the target path.

## Architecture Notes

- Coordinator starts first, then consensus, filenode, sync (`runBundleServices`).
- `extractSharedNetwork` copies network components from the coordinator into other apps.
- DRPC routes by method prefix (`/CoordinatorService`, `/ConsensusService`, `/FileService`, `/SpaceSyncService`).
- Data layout (default `./data`):

```
bundle-config.yml
client-config.yml
network-store/{coordinator,consensus,filenode,sync}/
storage-file/    # Badger
storage-sync/    # AnyStore
```

## Development

### Compose files

- `compose.dev.yml` – development dependencies (MongoDB replica set + Redis Stack).
- `compose.aio.yml` – bundle image with embedded MongoDB/Redis.
- `compose.external.yml` – bundle image plus external MongoDB and Redis containers.

```bash
go build -o any-sync-bundle .
go test ./...
golangci-lint run
```

Focused store tests: `go test ./lightcmp/lightfilenodestore -v`

The store tests cover CRUD, idempotent deletes, concurrent readers/writers, `GetMany` cancellation, and GC via `LightFileNodeStore.gcOnce`.

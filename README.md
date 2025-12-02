<p align="center">
  <img src="./docs/logo.png" width="550">
</p>

<p align="center">
  <b>Self-host Anytype in 60 seconds. One command. Zero complexity.</b>
</p>

<p align="center">
  <a href="https://github.com/grishy/any-sync-bundle/tags"><img src="https://img.shields.io/github/v/tag/grishy/any-sync-bundle" alt="Version"></a>
  &nbsp;
  <a href="https://github.com/grishy/any-sync-bundle/actions"><img src="https://github.com/grishy/any-sync-bundle/actions/workflows/release.yml/badge.svg" alt="Build"></a>
  &nbsp;
  <img src="https://goreportcard.com/badge/github.com/grishy/gopkgview" alt="Go Report Card">
</p>

--- 

**any-sync-bundle** is a prepackaged, all-in-one self-hosted server solution designed for Anytype, a local-first, peer-to-peer note-taking and knowledge management application. It is based on the original modules used in the official Anytype server but merges them into a single binary for simplified deployment and zero-configuration setup.

> 💡 **New to Anytype?** It's a local-first, privacy-focused alternative to Notion. [Learn more →](https://anytype.io/)
  
## TL;DR – 60-Second Setup

This is a zero-config version of the official Anytype server. It uses the same upstream modules Anytype ships, but compacts them into a single binary - think of it as "K3s for Any Sync".

**Replace `192.168.100.9` with:**

- Your server's **local IP** for LAN-only access (e.g., `192.168.1.100`)
- Your **public domain** for remote access (e.g., `sync.example.com`)
- **Both** (comma-separated) for flexibility: `sync.example.com,192.168.1.100`

```sh
docker run \
    -e ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS="192.168.100.9" \
    -p 33010:33010 \
    -p 33020:33020/udp \
    -v $(pwd)/data:/data \
  ghcr.io/grishy/any-sync-bundle:1.1.3-2025-12-01
```

After the first run, point Anytype desktop/mobile apps at the generated client config in `./data/client-config.yml`. This is test start, check below more real configuration.

> How to configure Anytype apps to use your self-hosted server? See [Client setup →](https://doc.anytype.io/anytype-docs/advanced/data-and-security/self-hosting/self-hosted#how-to-switch-to-a-self-hosted-network)

## Available variants

- **✅ Bundle (all-in-one container)**: Bundled with MongoDB and Redis built in.
- **✅ Bundle (solo bundle / container)**: A variant without MongoDB and Redis. You can use your own instances.
- **🧶 Custom Light version[\*](#light-version-not-in-development)**: Not in development currently.

## Key features

- **Easy to start**: A single command to launch the server
- **All-in-one option**: All services in a single container or in separate binaries
- **Zero-config**: Sensible defaults, configurable when needed
- **Lightweight**: No MinIO, and no duplicate logical services
- **Only 2 open ports**: TCP 33010 (DRPC protocol) and UDP 33020 (QUIC protocol), configurable

### Who is this for?

- ✅ **Self-hosters** who value simplicity over complexity
- ✅ **Small teams** (5-50 users) who don't need horizontal scaling
- ✅ **Low resource** Homelab setups and Raspberry Pi deployments

### Not for you if

- ❌ You need high-availability clustering across multiple nodes
- ❌ You require horizontal scaling beyond a single server
- ❌ You want to use the official Anytype architecture as-is

## Architecture

![Comparison with original deployment](./docs/arch.svg)

## Version

The project version combines the bundle version and the original Anytype version.
Current version: **`v1.1.3-2025-12-01`**

**Version format:** `v[bundle-version]-[anytype-compatibility-date]`

- `v1.1.2` – Bundle's semantic version (SemVer)
- `2025-10-24` – The Anytype any-sync compatibility version from [anytype.io](https://puppetdoc.anytype.io/api/v1/prod-any-sync-compatible-versions/)

> Compatibility: 0.x (e.g., 0.5) is not supported. From 1.x onward we follow SemVer; 1.x upgrades are non‑breaking.

## How to start

### Container

Pick one of the published tags, for example `v1.1.3-2025-12-01` (see [Packages](https://github.com/grishy/any-sync-bundle/pkgs/container/any-sync-bundle)).

Latest tags are also available (`ghcr.io/grishy/any-sync-bundle:latest`, `:minimal`), but using an explicit release tag keeps upgrades deliberate (my recommendation).

- `ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS` multiple addresses can be added, separated by commas.
- `ANY_SYNC_BUNDLE_INIT_*` variables seed the initial configuration on first start; their values are persisted to `bundle-config.yml` afterward.

1. Container (all-in-one with embedded MongoDB/Redis)

   ```sh
   docker run -d \
       -e ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS="192.168.100.9" \
       -p 33010:33010 \
       -p 33020:33020/udp \
       -v $(pwd)/data:/data \
       --restart unless-stopped \
       --name any-sync-bundle-aio \
     ghcr.io/grishy/any-sync-bundle:1.1.3-2025-12-01
   ```

2. Container (solo bundle, external MongoDB/Redis)
   ```sh
   docker run -d \
       -e ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS="192.168.100.9" \
       -e ANY_SYNC_BUNDLE_INIT_MONGO_URI="mongodb://user:pass@mongo:27017/" \
       -e ANY_SYNC_BUNDLE_INIT_REDIS_URI="redis://redis:6379/" \
       -p 33010:33010 \
       -p 33020:33020/udp \
       -v $(pwd)/data:/data \
       --restart unless-stopped \
       --name any-sync-bundle \
     ghcr.io/grishy/any-sync-bundle:1.1.3-2025-12-01-minimal
   ```

### Docker Compose

- All-in-one image only:
  ```sh
  docker compose -f compose.aio.yml up -d
  ```
- Bundle + external MongoDB + Redis:
  ```sh
  docker compose -f compose.external.yml up -d
  ```
- With Traefik reverse proxy (TCP 33010 + UDP 33020):
  ```sh
  docker compose -f compose.traefik.yml up -d
  ```
  Edit `ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS` in the compose file before starting.

### Without container (binary)

1. Download the binary from the [Release page](https://github.com/grishy/any-sync-bundle/releases)
2. Start as below (replace IP and URIs as needed):

   ```sh
   ./any-sync-bundle start-bundle \
     --initial-external-addrs "192.168.100.9" \
     --initial-mongo-uri "mongodb://127.0.0.1:27017/" \
     --initial-redis-uri "redis://127.0.0.1:6379/" \
     --initial-storage ./data/storage
   ```

   systemd example:

   ```ini
   [Unit]
   Description=Any Sync Bundle
   After=network-online.target
   Wants=network-online.target

   [Service]
   WorkingDirectory=/opt/any-sync-bundle
   ExecStart=/opt/any-sync-bundle/any-sync-bundle start-bundle \
     --initial-external-addrs "example.local,192.168.100.9" \
     --initial-mongo-uri "mongodb://127.0.0.1:27017/?replicaSet=rs0" \
     --initial-redis-uri "redis://127.0.0.1:6379/" \
     --initial-storage /opt/any-sync-bundle/data/storage
   Restart=on-failure
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   ```

## Building from Source

### Traditional Go Build

**Prerequisites:**

- Go 1.25.2 or later
- Docker (optional, for testing with containers)

**Build:**

```sh
go build -o any-sync-bundle .
./any-sync-bundle --version
```

**Run tests:**

```sh
go test -race -shuffle=on -vet=all ./...
```

**Run linter:**

```sh
golangci-lint run
```

### With Nix

[Nix](https://nixos.org/) provides reproducible builds and a complete development environment with one command.

**Prerequisites:** Nix; direnv and [hook it into your shell](https://direnv.net/docs/hook.html)

**Build the binary:**

```sh
nix build
./result/bin/any-sync-bundle --version
```

**Run directly:**

```sh
nix run . -- --help
```

**Development environment:**

```sh
# Enter development shell with all tools (Go, golangci-lint, goreleaser, etc.)
nix develop

# Or use direnv for automatic environment activation
echo "use flake" > .envrc
direnv allow
```

**Check flake:**

```sh
nix flake check
```

## Configuration files

There are two configuration files under `./data` by default:

- `bundle-config.yml` — Private/important. Created on the first run. Contains service configuration and private keys. Back this up.
- `client-config.yml` — Regenerated on each start. Import this into Anytype apps to connect to your server.

### Client setup

After the first start, point Anytype desktop/mobile apps to `./data/client-config.yml`.

Read more about it in the official docs:

- [Anytype Docs -> Self-hosted](https://doc.anytype.io/anytype-docs/advanced/data-and-security/self-hosting/self-hosted#how-to-switch-to-a-self-hosted-network)
- [Anytype Tech -> Self-hosting](https://tech.anytype.io/how-to/self-hosting)

## Parameters

All parameters are available in two ways: binary flags or container environment variables. See `./any-sync-bundle --help` for details.

Important: "initial-\_" options (for example `--initial-external-addrs` or `ANY_SYNC_BUNDLE_INIT*`) are used only on the first run to create `bundle-config.yml`. Subsequent starts read from the persisted `bundle-config.yml`.

### Global parameters

| Flag              | Description                                                                                                                                                  |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `--debug`         | Enable debug mode with detailed logging <br> ‣ Default: `false` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_DEBUG`                                         |
| `--log-level`     | Log level (debug, info, warn, error, fatal) <br> ‣ Default: `info` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_LOG_LEVEL`                                  |
| `--pprof`         | Enable pprof HTTP server for profiling <br> ‣ Default: `false` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_PPROF`                                          |
| `--pprof-addr`    | Address for pprof HTTP server (only used when --pprof is enabled) <br> ‣ Default: `localhost:6060` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_PPROF_ADDR` |
| `--help`, `-h`    | show help                                                                                                                                                    |
| `--version`, `-v` | print the version, use it if you wanna create an issue.                                                                                                      |

### Commands

- `help` — Show help for the binary.
- `start-bundle` — Start all services, using external MongoDB and Redis. Ensure MongoDB has a replica set initialized before starting.
- `start-all-in-one` — Used inside the official all‑in‑one container. Starts Redis and MongoDB inside the container and initializes the MongoDB replica set automatically.

Flags for `start-bundle` and `start-all-in-one`:

| Flag                       | Description                                                                                                                                                                      |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--bundle-config`          | Path to the bundle configuration YAML file <br> ‣ Default: `./data/bundle-config.yml` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_CONFIG`                                      |
| `--client-config`          | Path where write to the Anytype client configuration YAML file if needed <br> ‣ Default: `./data/client-config.yml` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_CLIENT_CONFIG` |
| `--initial-storage`        | Initial path to the bundle data directory (must be writable) <br> ‣ Default: `./data/storage/` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_INIT_STORAGE`                       |
| `--initial-external-addrs` | Initial external addresses for the bundle <br> ‣ Default: `192.168.8.214,example.local` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS`                       |
| `--initial-mongo-uri`      | Initial MongoDB URI for the bundle <br> ‣ Default: `mongodb://127.0.0.1:27017/` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_INIT_MONGO_URI`                                    |
| `--initial-redis-uri`      | Initial Redis URI for the bundle <br> ‣ Default: `redis://127.0.0.1:6379/` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_INIT_REDIS_URI`                                         |

## Light version (not in development)

I explored a "light" Any Sync variant without MongoDB and Redis, using a single BadgerDB instance for all logical services (touching filenode, consensus, and coordinator). I decided not to continue due to long‑term maintenance cost.
Currently, only the filenode is slightly modified to remove the MinIO dependency.

The light version exists as [a draft PR](https://github.com/grishy/any-sync-bundle/pull/19) and is not planned for active development.

## Data & Backups

⚠️ **Always backup before upgrades**

### What Gets Backed Up

| Path                       | Critical | Contents                 |
| -------------------------- | -------- | ------------------------ |
| `./data/bundle-config.yml` | 🔴 Yes   | Config + private keys    |
| `./data/storage/`          | 🔴 Yes   | User sync data           |
| `./data/mongo/` (AIO only) | 🔴 Yes   | Coordinator/consensus DB |
| `./data/redis/` (AIO only) | 🔴 Yes   | Filenode cache           |
| `./data/client-config.yml` | 🟢 No    | Regenerated on start     |

### Backup Process

1. Stop your service
2. Backup the `./data/` directory
3. Restart service

**Core backup command:**

```bash
tar -czf backup-$(date +%Y%m%d-%H%M%S).tar.gz ./data/
```

### Restore Process

1. Stop your service
2. Remove current data directory
3. Extract backup
4. Restart service

**Core restore command:**

```bash
rm -rf ./data && tar -xzf backup-YYYYMMDD-HHMMSS.tar.gz
```

**If update fails to start:**

1. Stop failed service
2. Remove data: `rm -rf ./data`
3. Restore backup: `tar -xzf backup-YYYYMMDD-HHMMSS.tar.gz`
4. Restart with previous version/image

## Troubleshooting

- MongoDB replica set is not initiated (external DB):
  - Initialize manually once: `mongosh --host <mongo:27017> --eval "rs.initiate({_id:'rs0', members:[{_id:0, host:'localhost:27017'}]})"`
  - Replace `localhost` with the actual hostname or IP of your MongoDB server that will be used by the bundle later, if needed.
- Embedded MongoDB/Redis in AIO does not start:
  - Check logs for "starting embedded MongoDB/Redis". If the data directories are corrupted, stop the container and **if you no need data**, remove `/data/mongo` or `/data/redis` before restarting.
- QUIC/UDP blocked:
  - Open UDP 33020 on firewalls/NAT. Some environments block UDP by default.
  - Advertise both hostname and IP in `ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS` for clients behind NAT. Anytype will select one of the addresses to connect to that works.
- Wrong external address after first run:
  - Edit `./data/bundle-config.yml` → `externalAddr:` list, then restart the server. The new `client-config.yml` will be regenerated.

## Acknowledgments

> "Because I stand on the shoulders of giants, I can see further than they can."

This project wouldn't exist without:

- **[Anytype](https://anytype.io/)** – For creating an amazing local-first, privacy-focused tool
- The **any-sync** team – For open-sourcing the sync infrastructure
- The **self-hosting community** – For testing, feedback, and support
- Parcel.js for their awesome box logo!

## Release

Reminder for releasing a new version.

```sh
# 1. Check locally
goreleaser release --snapshot --clean
```

```sh
# Set variables (fish shell)
set VERSION v1.1.2
set ANYTYPE_UNIX_TIMESTAMP 1761316841
set ANYTYPE_FORMATTED (date -r $ANYTYPE_UNIX_TIMESTAMP +'%Y-%m-%d')
set FINAL_VERSION $VERSION-$ANYTYPE_FORMATTED

# Create tag and push
git tag -a $FINAL_VERSION -m "Release $FINAL_VERSION"
git push origin tag $FINAL_VERSION
```

> "Perfection is achieved, not when there is nothing more to add, but when there is nothing left to take away" – Antoine de Saint-Exupéry

## License

© 2025 [Sergei G.](https://github.com/grishy)
Licensed under [MIT](./LICENSE).

<p align="center">
  <sub>Built with ❤️ for data ownership, local-first, and open-source</sub>
</p>

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

**any-sync-bundle** is a prepackaged, all-in-one self-hosted server for [Anytype](https://anytype.io/) – a local-first, privacy-focused alternative to Notion. It merges all official Anytype sync modules into a single binary for simplified deployment. Think of it as "K3s for Any Sync".

> 💡 **New to Anytype?** It's a local-first, privacy-focused alternative to Notion. [Learn more →](https://anytype.io/)

## TL;DR

**Replace `192.168.100.9` with:**

- Your server's **local IP** for LAN-only access (e.g., `192.168.1.100`)
- Your **public domain** for remote access (e.g., `sync.example.com`)
- **Both** (comma-separated) for flexibility: `sync.example.com,192.168.1.100`

```sh
docker run -d \
    -e ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS="192.168.100.9" \
    -p 33010:33010 \
    -p 33020:33020/udp \
    -v $(pwd)/data:/data \
    --restart unless-stopped \
  ghcr.io/grishy/any-sync-bundle:1.2.1-2025-12-10
```

After the first run, import `./data/client-config.yml` into Anytype apps.

> How to configure Anytype apps to use your self-hosted server? [Client setup →](https://doc.anytype.io/anytype-docs/advanced/data-and-security/self-hosting/self-hosted#how-to-switch-to-a-self-hosted-network)

## Overview

### Key Features

- **Easy to start**: A single command to launch the server
- **All-in-one option**: All services in a single container or in separate binaries
- **Zero-config**: Sensible defaults, configurable when needed
- **Lightweight**: No MinIO option, and no duplicate logical services
- **Only 2 open ports**: TCP 33010 (DRPC protocol) and UDP 33020 (QUIC protocol)

### Who is this for?

- ✅ **Self-hosters** who value simplicity over complexity
- ✅ **Low resource** Homelab setups and Raspberry Pi deployments

### Not for you if

- ❌ You need high-availability clustering across multiple nodes
- ❌ You require horizontal scaling beyond a single server
- ❌ You want to use the official Anytype architecture as-is

### Architecture

![Comparison with original deployment](./docs/arch.svg)

### Version

Current version: **`v1.2.1-2025-12-10`**

Format: `v[bundle-version]-[anytype-compatibility-date]`

- `v1.1.3` – Bundle's semantic version (SemVer)
- `2025-12-01` – Anytype any-sync compatibility date from [anytype.io](https://puppetdoc.anytype.io/api/v1/prod-any-sync-compatible-versions/)

> Compatibility: From 1.x onward we follow SemVer; 1.x upgrades are non‑breaking.

## Installation

### Available Images

| Image Tag                                                 | Description                         |
| --------------------------------------------------------- | ----------------------------------- |
| `ghcr.io/grishy/any-sync-bundle:1.2.1-2025-12-10`         | All-in-one (embedded MongoDB/Redis) |
| `ghcr.io/grishy/any-sync-bundle:1.2.1-2025-12-10-minimal` | Minimal (external MongoDB/Redis)    |

Latest tags (`:latest`, `:minimal`) are available, but explicit version tags are recommended.

### Docker Compose (Recommended)

| File                   | Description                                  |
| ---------------------- | -------------------------------------------- |
| `compose.aio.yml`      | All-in-one with embedded MongoDB/Redis       |
| `compose.external.yml` | Bundle + external MongoDB + Redis containers |
| `compose.s3.yml`       | Bundle + MinIO for S3 storage                |
| `compose.traefik.yml`  | With Traefik reverse proxy                   |

```sh
# Pick one as example:
docker compose -f compose.aio.yml up -d
docker compose -f compose.external.yml up -d
docker compose -f compose.s3.yml up -d
```

Edit `ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS` in the compose file before starting.

### Binary

1. Download from the [Release page](https://github.com/grishy/any-sync-bundle/releases)
2. Run:

```sh
./any-sync-bundle start-bundle \
  --initial-external-addrs "192.168.100.9" \
  --initial-mongo-uri "mongodb://127.0.0.1:27017/" \
  --initial-redis-uri "redis://127.0.0.1:6379/" \
  --initial-storage ./data/storage
```

## Configuration

### Quick Reference

| Variable                              | Purpose              | Required |
| ------------------------------------- | -------------------- | -------- |
| `ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS` | Advertised addresses | Yes      |
| `ANY_SYNC_BUNDLE_INIT_MONGO_URI`      | MongoDB connection   | No       |
| `ANY_SYNC_BUNDLE_INIT_REDIS_URI`      | Redis connection     | No       |
| `ANY_SYNC_BUNDLE_INIT_S3_BUCKET`      | S3 bucket name       | No       |
| `ANY_SYNC_BUNDLE_INIT_S3_ENDPOINT`    | S3 endpoint URL      | No       |
| `AWS_ACCESS_KEY_ID`                   | S3 credentials       | No       |
| `AWS_SECRET_ACCESS_KEY`               | S3 credentials       | No       |

### Configuration Files

| File                       | Purpose                                   | Backup? |
| -------------------------- | ----------------------------------------- | ------- |
| `./data/bundle-config.yml` | Service config + private keys             | 🔴 Yes  |
| `./data/client-config.yml` | Client config (regenerated on each start) | 🟢 No   |

### Storage Options

#### Local Storage (Default)

By default, the bundle uses embedded BadgerDB. No configuration needed.

```
+ Zero configuration
+ No external dependencies
+ Lower latency
- Limited by local disk space
```

#### S3 Storage (Optional)

Uses the **original Anytype upstream S3 implementation** from any-sync-filenode.

```
+ Supports AWS S3, MinIO, DigitalOcean Spaces, Cloudflare R2, Backblaze B2
- Network latency
- Requires configuration
```

**Enable S3 with:**

```sh
export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."

./any-sync-bundle start-bundle \
  --initial-s3-bucket "my-bucket" \
  --initial-s3-endpoint "https://s3.us-east-1.amazonaws.com"
```

For MinIO, add `--initial-s3-force-path-style`.

**Docker Compose with MinIO:**

```sh
docker compose -f compose.s3.yml up -d
# MinIO console: http://localhost:9001 (minioadmin/minioadmin)
```

## Parameters

All parameters available as binary flags or environment variables. See `./any-sync-bundle --help`.

> **Note:** `--initial-*` options are used only on first run to create `bundle-config.yml`. Subsequent starts read from the persisted config.

### Global Flags

| Flag              | Description                                                                                                                                                  |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `--debug`         | Enable debug mode with detailed logging <br> ‣ Default: `false` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_DEBUG`                                         |
| `--log-level`     | Log level (debug, info, warn, error, fatal) <br> ‣ Default: `info` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_LOG_LEVEL`                                  |
| `--pprof`         | Enable pprof HTTP server for profiling <br> ‣ Default: `false` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_PPROF`                                          |
| `--pprof-addr`    | Address for pprof HTTP server (only used when --pprof is enabled) <br> ‣ Default: `localhost:6060` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_PPROF_ADDR` |
| `--help`, `-h`    | Show help                                                                                                                                                    |
| `--version`, `-v` | Print the version                                                                                                                                            |

### Commands

| Command            | Description                                                      |
| ------------------ | ---------------------------------------------------------------- |
| `start-bundle`     | Start with external MongoDB/Redis                                |
| `start-all-in-one` | Start with embedded MongoDB/Redis (used in all-in-one container) |

### Start Command Flags

| Flag                            | Description                                                                                                                                                                      |
| ------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--bundle-config`, `-c`         | Path to the bundle configuration YAML file <br> ‣ Default: `./data/bundle-config.yml` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_CONFIG`                                      |
| `--client-config`, `--cc`       | Path where write to the Anytype client configuration YAML file if needed <br> ‣ Default: `./data/client-config.yml` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_CLIENT_CONFIG` |
| `--initial-storage`             | Initial path to the bundle data directory (must be writable) <br> ‣ Default: `./data/storage/` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_INIT_STORAGE`                       |
| `--initial-external-addrs`      | Initial external addresses for the bundle <br> ‣ Default: `192.168.8.214,example.local` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS`                       |
| `--initial-mongo-uri`           | Initial MongoDB URI for the bundle <br> ‣ Default: `mongodb://127.0.0.1:27017/` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_INIT_MONGO_URI`                                    |
| `--initial-redis-uri`           | Initial Redis URI for the bundle <br> ‣ Default: `redis://127.0.0.1:6379/` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_INIT_REDIS_URI`                                         |
| `--initial-s3-bucket`           | S3 bucket name <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_INIT_S3_BUCKET`                                                                                                     |
| `--initial-s3-endpoint`         | S3 endpoint URL <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_INIT_S3_ENDPOINT`                                                                                                  |
| `--initial-s3-force-path-style` | Use path-style S3 URLs (required for MinIO) <br> ‣ Default: `false` <br> ‣ Environment Variable: `ANY_SYNC_BUNDLE_INIT_S3_FORCE_PATH_STYLE`                                      |

## Operations

### Backup & Recovery

**Backup:**

```sh
# Stop service first
tar -czf backup-$(date +%Y%m%d-%H%M%S).tar.gz ./data/
```

**Restore:**

```sh
rm -rf ./data && tar -xzf backup-YYYYMMDD-HHMMSS.tar.gz
```

### Upgrading

1. **Backup** your `./data/` directory
2. Stop the current service
3. Pull new image: `docker pull ghcr.io/grishy/any-sync-bundle:NEW_VERSION`
4. Start the service
5. Verify: check logs and `--version`

If upgrade fails, restore from backup and use the previous version.

### Troubleshooting

**Client can't connect:**

- Verify `ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS` matches your server's reachable IP/hostname
- Check firewall: TCP 33010 and UDP 33020 must be open

**Wrong external address after first run:**

- Edit `./data/bundle-config.yml` → `externalAddr:` list
- Restart server (new `client-config.yml` will be generated)

**MongoDB crashes with "illegal instruction" (AIO):**

- Your CPU doesn't support AVX instructions required by MongoDB 5.0+
- Common on older Intel Celeron, Atom, and similar low-power CPUs
- Use `:minimal` image with external MongoDB 4.4 or earlier

## Development

See [CONTRIBUTING.md](./CONTRIBUTING.md) for build instructions, development setup, and release process.

## Acknowledgments

This project wouldn't exist without:

- **[Anytype](https://anytype.io/)** – For creating an amazing local-first, privacy-focused tool
- The **any-sync** team – For open-sourcing the sync infrastructure
- The **self-hosting community** – For testing, feedback, and support

## License

© 2025 [Sergei G.](https://github.com/grishy)
Licensed under [MIT](./LICENSE).

<p align="center">
  <sub>Built with ❤️ for data ownership, local-first, and open-source</sub>
</p>

# Any-Sync-Bundle

<p align="center">
  <img src="./docs/logo.png" width="550">
</p>

<p align="center">
  <table align="center">
    <tr>
      <td><strong>Status</strong></td>
      <td><b>Maintained</b></td>
    </tr>
    <tr>
      <td><strong>Stable Version</strong></td>
      <td><a href="https://github.com/grishy/any-sync-bundle/tags"><img src="https://img.shields.io/github/v/tag/grishy/any-sync-bundle" alt="GitHub tag"></a></td>
    </tr>
    <tr>
      <td><strong>CI/CD</strong></td>
      <td><a href="https://github.com/grishy/any-sync-bundle/actions"><img src="https://github.com/grishy/any-sync-bundle/actions/workflows/release.yml/badge.svg" alt="Build Status"></a></td>
    </tr>
  </table>
</p>

---

<div style="border: 1px solid #ffa500; background-color: #fff7e6; padding: 16px; border-radius: 6px; margin: 16px 0;">
  <p>It is better to use <b>Release</b>. The main branch contains code that is under active development. Available variants:</p>
  <ul>
    <li><strong>✅ Bundle (all-in-one container)</strong>: Bundled with MongoDB and Redis built in.</li>
    <li><strong>✅ Bundle (solo bundle)</strong>: A variant without MongoDB and Redis inside. You can use your own instances.</li>
  </ul>
</div>

---

## TL;DR – How to start a self-hosted Anytype server

This is a zero-config version of the official Anytype server. It uses the same upstream modules Anytype ships, but compacts them into a single binary - think of it as "K3s for Any Sync".

Replace the external address (e.g., `192.168.100.9`) with an address or hostname clients should connect to.

```sh
docker run \
    -e ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS="192.168.100.9" \
    -p 33010:33010 \
    -p 33020:33020/udp \
    -v $(pwd)/data:/data \
  ghcr.io/grishy/any-sync-bundle:0.6.0+2025-09-08
```

After the first run, point Anytype desktop/mobile apps at the generated client config in `./data/client-config.yml`.

## Key features

- **Easy to start**: A single command to launch the server
- **All-in-one option**: All services in a single container or in separate binaries
- **Lightweight**: No MinIO, and no duplicate logical services
- **Only 2 opne ports**: TCP 33010 and UDP 33020 (configurable)

## Architecture

![Comparison with original deployment](./docs/arch.png)

## Version format

The project version combines the bundle version and the original Anytype version.
Example: `v0.6.0+2025-09-08`

- `v0.6.0` – The bundle’s semver version
- `2025-09-08` – The Anytype any-sync compatibility version from [anytype.io](https://puppetdoc.anytype.io/api/v1/prod-any-sync-compatible-versions/)

## How to start

### Container

Pick one of the published tags, for example `v0.6.0+2025-09-08` (see [Packages](https://github.com/grishy/any-sync-bundle/pkgs/container/any-sync-bundle)).

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
     ghcr.io/grishy/any-sync-bundle:0.6.0+2025-09-08
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
     ghcr.io/grishy/any-sync-bundle:0.6.0+2025-09-08-minimal
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

### Without container (binary)

1. Download the binary from the [Release page](https://github.com/grishy/any-sync-bundle/releases)
2. Start as below (replace IP and URIs as needed):

   ```sh
   ./any-sync-bundle start-bundle \
     --initial-external-addrs "192.168.100.9" \
     --initial-mongo-uri "mongodb://127.0.0.1:27017/" \
     --initial-redis-uri "redis://127.0.0.1:6379/" \
     --storage ./data/storage
   ```

## Release

Reminder for releasing a new version.

```sh
# 1. Check locally
goreleaser release --snapshot --clean
```

```sh
# 1. Set variables (fish-shell)
set VERSION v0.6.0
set ANYTYPE_UNIX_TIMESTAMP 1757347920

# 2. Format date
set ANYTYPE_FORMATTED (date -r $ANYTYPE_UNIX_TIMESTAMP +'%Y-%m-%d')
set FINAL_VERSION $VERSION+$ANYTYPE_FORMATTED

# 3. Create tag and push
git tag -a $FINAL_VERSION -m "Release $FINAL_VERSION"
git push origin tag $FINAL_VERSION
```

> Because I stand on the shoulders of giants, I can see further than they can.

> "Perfection is achieved, not when there is nothing more to add, but when there is nothing left to take away" – Antoine de Saint-Exupéry

## License

© 2025 [Sergei G.](https://github.com/grishy)
Licensed under [MIT](./LICENSE).

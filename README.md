# Any-Sync-Bundle

<p align="center">
  <img src="./docs/logo.png" width="550">
</p>

<p align="center">
  <table align="center">
    <tr>
      <td><strong>Status</strong></td>
      <td><b>⚠️ Under Validation</b></td>
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

TODO: Table with benchmaks and cpu/memory/etc.

for ortiginal, allinone, light

## TL;DR – How to start a self-hosted Anytype server

This is a zero config version of official Anytype server. Base on original modules, that are used in official Anytype server, but merged into one binary.

Replace the external address (e.g., `192.168.100.9`) with a local IP address or domain.  
Multiple addresses can be added, separated by commas.  
Then use the client config YAML in `./data/client-config.yml`.

**Container (solo bundle, external MongoDB/Redis)**

Pick one of the published tags, for example `0.5.0-2024-12-18` (see [Packages](https://github.com/grishy/any-sync-bundle/pkgs/container/any-sync-bundle)).

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
  ghcr.io/grishy/any-sync-bundle:0.5.0-2024-12-18-minimal
```

**Container (all-in-one with embedded MongoDB/Redis)**

```sh
docker run -d \
    -p 33010:33010 \
    -p 33020:33020/udp \
    -v $(pwd)/data:/data \
    --restart unless-stopped \
    --name any-sync-bundle-aio \
  ghcr.io/grishy/any-sync-bundle:0.5.0-2024-12-18
```

Latest tags are also available (`ghcr.io/grishy/any-sync-bundle:latest`, `:minimal`), but using an explicit release tag keeps upgrades deliberate.

**Without container (binary)**

```sh
ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS="192.168.100.9" \
ANY_SYNC_BUNDLE_INIT_MONGO_URI="mongodb://127.0.0.1:27017/" \
ANY_SYNC_BUNDLE_INIT_REDIS_URI="redis://127.0.0.1:6379/" \
./any-sync-bundle start-bundle
```

## Version

### Bundle version

The project version combines the bundle version and the original Anytype version.  
Example: `v0.6.0+2025-09-08`

- `v0.6.0` – The bundle’s semver version
- `2025-09-08` – The Anytype any-sync compatibility version from [anytype.io](https://puppetdoc.anytype.io/api/v1/prod-any-sync-compatible-versions/)

### Bundle start version

1. Binary file for each release on the [Release page](https://github.com/grishy/any-sync-bundle/releases)
2. All-in-one container on [ghcr.io/grishy/any-sync-bundle](https://github.com/grishy/any-sync-bundle/pkgs/container/any-sync-bundle) with Mongo and Redis included
3. Minimal container (`-minimal`) with only any-sync-bundle, without Mongo or Redis

## Key features

- **Easy to start**: A single command to launch the server
- **All-in-one option**: All services in a single container or in separate binaries
- **Lightweight**: No MinIO included, and plans exist to reduce size further

## Why created?

1. Existing solutions required many containers and complicated config
2. MinIO was too large for some servers
3. Documentation and generated configs were incomplete

## Issues on Anytype side in work to improve bundle

1. https://github.com/anyproto/any-sync/issues/373
2. https://github.com/anyproto/any-sync-dockercompose/issues/126
3. https://github.com/anyproto/any-sync/pull/374
4. https://github.com/anyproto/anytype-ts/pull/1186

## Release

Reminder for releasing a new version.

```sh
# 1. Check locally
goreleaser release --snapshot --clean
```

```sh
# 1. Set variables (fish-shell)
set VERSION v0.5.0
set ANYTYPE_UNIX_TIMESTAMP 1734517522

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

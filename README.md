# Any-Sync-Bundle

<p align="center">
  <img src="./docs/logo.png" width="550">
</p>

<p align="center">
  <table align="center">
    <tr>
      <td><strong>Status</strong></td>
      <td>Maintained</td>
    </tr>
    <tr>
      <td><strong>Version</strong></td>
      <td><a href="https://github.com/grishy/any-sync-bundle/tags"><img src="https://img.shields.io/github/v/tag/grishy/any-sync-bundle" alt="GitHub tag"></a></td>
    </tr>
    <tr>
      <td><strong>CI/CD</strong></td>
      <td><a href="https://github.com/grishy/any-sync-bundle/actions"><img src="https://github.com/grishy/any-sync-bundle/actions/workflows/release.yml/badge.svg" alt="Build Status"></a></td>
    </tr>
  </table>
</p>

## TL;DR – How to start a self-hosted Anytype server

Replace the external address (`192.168.100.9`) with your machine’s IP address or domain.  
You can specify multiple addresses, separated by commas.  
After that you can use client config YAML from the `./data` folder.

```bash
docker run -d \
    -e ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS="192.168.100.9" \
    -p 33010-33013:33010-33013 \
    -p 33020-33023:33020-33023/udp \
    -v $(pwd)/data:/data \
    --restart unless-stopped \
    --name any-sync-bundle \
  ghcr.io/grishy/any-sync-bundle:latest
```

## Key features
- **Easy to start**: Just one command to start the server
- **Lightweight**: No MinIO inside and in plan to make it even smaller
- **All-in-one option**: All services in one container, or you can download binaries separately 



## TODO

- https://github.com/quic-go/quic-go/wiki/UDP-Buffer-Sizes#non-bsd
- Improve logging and add a prefix for each service (e.g., `any-sync-coordinator:`)
  - Consider replacing Supervisor with a simpler script,maybe Go
- Merge all Docker outputs into one with multi-stage builds and target stages
- Check other Docker builds, e.g., docker-mastodon
- Add an option to control the logger level, with a default of “warning”
- Create CI to check versions weekly
- Write a blog post in English, Russian, and Spanish
- Create a video in English and Russian
- Publish on Telegram, Reddit, GitHub issues, and the Anytype forum
- Add Docker Compose example
- Add tests for the bundle

## Why created?

- Hard to start, a lot of containers
- Existing documentation was inaccurate, and created configs were not fully correct
- MinIO made it overly large

## Ideas

- Allow get each node config
- Avoid Redis and use Go internal structure approach, implemented step by step for each service
- Use one port for all services

## Issues on Anytype side

- https://github.com/anyproto/any-sync/issues/373
- https://github.com/anyproto/any-sync-dockercompose/issues/126
- https://github.com/anyproto/any-sync/pull/374
- Found this: https://github.com/anyproto/any-sync-coordinator/issues/80#issuecomment-2220554099
- Add API support for end users to get notes
- Anytype app only supports .yml files, not .yaml: https://github.com/anyproto/anytype-ts/pull/1186

## Release

Reminder for myself on how to release a new version.  
Format: `v0.2.0+2024-12-18` (v<srm-version>+<date-of-anytype-release-from-gomod>)

```bash
# 1. Check locally
goreleaser release --snapshot --clean

# 2. Set variables
set VERSION v0.3.8
set ANYTYPE_UNIX_TIMESTAMP 1734517522

# 3. Set version
set ANYTYPE_FORMATTED `date -r $ANYTYPE_UNIX_TIMESTAMP +'%Y-%m-%d'`
set FINAL_VERSION $VERSION+$ANYTYPE_FORMATTED

# Create tag and push
git tag -a $FINAL_VERSION -m "Release $FINAL_VERSION"
git push origin tag $FINAL_VERSION
```

> Because I stand on the shoulders of giants, I can see further than they can.

## License

© 2025 [Sergei G.](https://github.com/grishy)  
This project is [MIT](./LICENSE) licensed.

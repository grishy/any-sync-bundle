<p align="center">
  <img src="./docs/logo.png" width="550">
   <br />
   <strong>Status: </strong>Maintained
</p>

<p align="center">
  <img src="https://img.shields.io/github/v/tag/grishy/any-sync-bundle" alt="GitHub tag (with filter)">
  <img src="https://github.com/grishy/any-sync-bundle/actions/workflows/release.yml/badge.svg" alt="Build Status">
</p>

## TL;DR - How to start self-hosted Anytype server

<!-- TODO -->

Version from [here](https://puppetdoc.anytype.io/api/v1/prod-any-sync-compatible-versions/).

## Notes

Need to create replica set for MongoDB. Manually or with some script.  
Check that address should be same as when we will start to use it?

```bash
docker build -t any --progress=plain -f docker/Dockerfile.all-in-one .

docker run --rm -it \
  -e ANY_SYNC_BUNDLE_INIT_EXTERNAL_ADDRS="192.168.100.9" \
  -p 33010-33013:33010-33013 \
  -p 33020-33023:33020-33023/udp \
  -p 27017:27017 \
  -p 6379:6379 \
  -v $(pwd)/data:/data \
  --name any-sync-bundle \
  any:latest
```

## TODO

- RAM up to 1GB with cache, usually ~300MB
- https://github.com/quic-go/quic-go/wiki/UDP-Buffer-Sizes#non-bsd
- Create first tech version
- Improve loggings and add prefix for each service, like `any-sync-coordinator:`
  - Maybe replace supervisor with some simple script
- Add release with binaries and containers for all platforms
- use port range to public for simplicity
- merge all docker outputs into one with multi-stage and target stage
- check other docker build, like docker-mastodon
- use go-avahi-cname release process
- Add way to controll logger level and default warning
- Build in version into binary
- Create CI to check versions once a week
- Create images with arch and logo?
  - Use box with anytype logo inside and glow around
- Write a blog post on eng,rus,esp
- Create a video in eng and rus
- Publish on tegegram, reddit, issue on github into the ticket and docker-compose, forum on anytype and into the question as response
- Add tests for the bundle

## Why created?

- Hard to start, a lot of containers
- Docs inacurate and created configs not fully correct
- Heavy because of MinIO
- Save also the client for each release

## Ideas

- Allow each to generate client config
- Use SQLite instead of MinIO
  - used keyval
- Don't use Redis, use in-memory storage, step by step for each service
- Use one port for all services

## Issues on Anytype side

- https://github.com/anyproto/any-sync/issues/373
- https://github.com/anyproto/any-sync-dockercompose/issues/126
- https://github.com/anyproto/any-sync/pull/374
- Found https://github.com/anyproto/any-sync-coordinator/issues/80#issuecomment-2220554099
- Add API support for end user to get notes
- Anytype app sypport only .yml files, not .yaml https://github.com/anyproto/anytype-ts/pull/1186

> Because I stand on the shoulders of giants, I can see further than they can.

## Release

Reminder for me 🙂  
Format: `v0.2.0+2024-12-18` (v<srm-version>+<date-of-anytype-release-from-gomod>)

```bash
# 1. Check localy
goreleaser release --snapshot --clean

# 2. Set variables
set VERSION v0.3.7
set ANYTYPE_UNIX_TIMESTAMP 1734517522

# 3. Set version
set ANYTYPE_FORMATTED `date -r $ANYTYPE_UNIX_TIMESTAMP +'%Y-%m-%d'`
set FINAL_VERSION $VERSION+$ANYTYPE_FORMATTED

# Create tag and push
git tag -a $FINAL_VERSION -m "Release $FINAL_VERSION"
git push origin tag $FINAL_VERSION
```

## License

© 2025 [Sergei G.](https://github.com/grishy)  
This project is [MIT](./LICENSE) licensed.

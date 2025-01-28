# Any-sync Bundle

<p align="center">
  <img src="./docs/todo" width="350">
   <br />
   <strong>Status: </strong>Maintained
</p>

<p align="center">
  <img src="https://img.shields.io/github/v/tag/grishy/any-sync-bundle" alt="GitHub tag (with filter)">
  <img src="https://goreportcard.com/badge/github.com/grishy/any-sync-bundle" alt="Go Report Card">
  <img src="https://github.com/grishy/any-sync-bundle/actions/workflows/release.yml/badge.svg" alt="Build Status">
</p>

## TL;DR - How to start self-hosted Anytype server

<!-- TODO -->

Version from [here](https://puppetdoc.anytype.io/api/v1/prod-any-sync-compatible-versions/).

## Notes

Need to create replica set for MongoDB. Manually or with some script.  
Check that address should be same as when we will start to use it?

```bash
docker build -t any --progress=plain -f docker/Dockerfile .

docker run -it \
    -p 33010:33010 \
    -p 33011:33011/udp \
    -p 33020:33020 \
    -p 33021:33021/udp \
    -p 33030:33030 \
    -p 33031:33031/udp \
    -p 33040:33040 \
    -p 33041:33041/udp \
    -p 27017:27017 \
    -p 6379:6379 \
    -v $(pwd)/data:/data \
    --name any-sync-bundle \
    any:latest
```

## TODO

- Create first tech version
- Add release with binaries and containers for all platforms
- use port range to public for simplicity
- check other docker build, like docker-mastodon
- use go-avahi-cname release process
- Build in version into binary
- Create CI to check versions
- Conjure up a version format
  - Recheck https://semver.org/
  - contain version of any-bundle, to track breaking changes
  - contain date of release, used as base
  - Ideas
    - `1.0.0-2025-09-01`
    - `1.2021.09`
    - `2021.09.01`
    - `1.0.0+20250901`
    - `v0.1.0+20241218-1734517522` - my version plus human readable date and original version
    - `v0.1.0+snapshot.2024-12-18` - human readable date
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
- TODO: Anytype app sypport only .yml files, not .yaml
- Found https://github.com/anyproto/any-sync-coordinator/issues/80#issuecomment-2220554099
- Add API support for end user to get notes

> Because I stand on the shoulders of giants, I can see further than they can.

## Release

Reminder for me, just create a tag and push it.  
Format: `v0.2.0+snapshot.2024-12-18` (v<srm-version>+snapshot.<date-of-anytype-release-from-gomod>)

```bash
git tag -a v0.2.0+snapshot.2024-12-18 -m "Release v0.2.0+snapshot.2024-12-18"
git push origin tag v0.2.0+snapshot.2024-12-18
```

## License

© 2025 [Sergei G.](https://github.com/grishy)  
This project is [MIT](./LICENSE) licensed.

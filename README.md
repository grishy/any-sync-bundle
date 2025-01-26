# any-sync-bundle

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
- Build in version into binary
- Conjure up a version format
  - Recheck https://semver.org/
  - contain version of any-bundle, to track breaking changes
  - contain date of release, used as base
  - Ideas
    - `1.0.0-2025-09-01`
    - `1.2021.09`
    - `2021.09.01`
    - `1.0.0+20250901`
- Create images with arch and logo?
- Write a blog post on both languages
- Create a video (on both languages?)
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

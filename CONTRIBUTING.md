## Contributing

Thanks for your interest in improving any-sync-bundle! This document explains how to build, test, and propose changes.

### Build and test

- Build: `go build -o any-sync-bundle .`
- Lint: `golangci-lint run --fix`
- Test: `go test -race -shuffle=on -vet=all -failfast ./...`

### Local dependencies

- For a quick local stack, see `compose.dev.yml` (MongoDB replica set + Redis Stack). Start with:

```bash
docker compose -f compose.dev.yml up -d
```

### Running

- All-in-one container (bundled MongoDB/Redis): see `compose.aio.yml` or the README TL;DR.
- Minimal container (external MongoDB/Redis): see `compose.external.yml`.
- Binary (no container): use `start-bundle` and supply your own MongoDB/Redis.

### Pull requests

- Keep PRs small and focused.
- Include a brief description of what changed and why.
- Update README/docs when behavior or flags/env variables change.

By submitting a PR, you agree that your contribution is licensed under the repository’s MIT license.
Good luck and have fun! 😉

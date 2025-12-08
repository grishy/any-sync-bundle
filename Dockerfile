# syntax=docker/dockerfile:1

#
# Stage: Initial bin build
#
FROM --platform=$BUILDPLATFORM golang:1.25.2-alpine AS stage-bin
WORKDIR /app

# Use mount cache for dependencies
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

ARG VERSION
ARG COMMIT
ARG COMMIT_DATE

ARG TARGETARCH
ARG TARGETOS

# Build platform-specific
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
    -trimpath \
    -ldflags="-w -s \
    -X github.com/grishy/any-sync-bundle/cmd.version=${VERSION} \
    -X github.com/grishy/any-sync-bundle/cmd.commit=${COMMIT} \
    -X github.com/grishy/any-sync-bundle/cmd.date=${COMMIT_DATE}" \
    -o /bin/any-sync-bundle

#
# Stage: stage-release-minimal
#
FROM gcr.io/distroless/static-debian12 AS stage-release-minimal

# Bundle network ports
EXPOSE 33010
EXPOSE 33020/udp

VOLUME /data

COPY --from=stage-bin /bin/any-sync-bundle /usr/local/bin/any-sync-bundle

ENTRYPOINT ["/usr/local/bin/any-sync-bundle"]
CMD ["start-bundle"]

#
# Stage: stage-release-all-in-one (with AVX support)
#
FROM docker.io/redis/redis-stack-server:7.4.0-v7 AS stage-release-all-in-one

# MongoDB version configuration
ARG MONGODB_VERSION_AVX=8.0

# Bundle network ports
EXPOSE 33010
EXPOSE 33020/udp

VOLUME /data

# Install prerequisites and MongoDB (AVX version)
RUN DEBIAN_FRONTEND=noninteractive \
    && apt-get update && apt-get install -y --no-install-recommends \
    gnupg \
    curl \
    ca-certificates \
    # Setup MongoDB AVX version repository (requires AVX)
    && curl -fsSL https://pgp.mongodb.com/server-${MONGODB_VERSION_AVX}.asc | gpg -o /usr/share/keyrings/mongodb-server-${MONGODB_VERSION_AVX}.gpg --dearmor \
    && echo "deb [ signed-by=/usr/share/keyrings/mongodb-server-${MONGODB_VERSION_AVX}.gpg ] http://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/${MONGODB_VERSION_AVX} multiverse" | tee /etc/apt/sources.list.d/mongodb-org-${MONGODB_VERSION_AVX}.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends mongodb-org-server \
    # Remove unnecessary packages
    && apt-get remove -y gnupg curl python3-pip \
    && apt-get autoremove -y \
    && apt-get clean \
    && rm -rf \
    /var/lib/apt/lists/* \
    /tmp/* \
    /var/tmp/* \
    /usr/share/keyrings/mongodb-server-*.gpg \
    /etc/apt/sources.list.d/mongodb-org-*.list

COPY --from=stage-bin /bin/any-sync-bundle /usr/local/bin/any-sync-bundle

ENTRYPOINT ["/usr/local/bin/any-sync-bundle"]
CMD ["start-all-in-one"]

#
# Stage: stage-release-all-in-one-noavx (without AVX support)
#
# Use Ubuntu 20.04 (focal) as base since MongoDB 4.4 only supports up to Ubuntu 20.04
FROM ubuntu:20.04 AS stage-release-all-in-one-noavx

# Bundle network ports
EXPOSE 33010
EXPOSE 33020/udp

VOLUME /data

# Install prerequisites, Redis Stack Server, and MongoDB 4.4
RUN DEBIAN_FRONTEND=noninteractive \
    && apt-get update && apt-get install -y --no-install-recommends \
    gnupg \
    curl \
    ca-certificates \
    # Install Redis Stack Server
    && curl -fsSL https://packages.redis.io/gpg | gpg --batch --yes --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb focal main" | tee /etc/apt/sources.list.d/redis.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends redis-stack-server \
    # Setup MongoDB 4.4 repository (no AVX required - officially supported on Ubuntu 20.04)
    && curl -fsSL https://pgp.mongodb.com/server-4.4.asc | gpg --batch --yes -o /usr/share/keyrings/mongodb-server-4.4.gpg --dearmor \
    && echo "deb [ signed-by=/usr/share/keyrings/mongodb-server-4.4.gpg ] http://repo.mongodb.org/apt/ubuntu focal/mongodb-org/4.4 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-4.4.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends mongodb-org-server \
    # Remove unnecessary packages
    && apt-get remove -y gnupg curl python3-pip \
    && apt-get autoremove -y \
    && apt-get clean \
    && rm -rf \
    /var/lib/apt/lists/* \
    /tmp/* \
    /var/tmp/* \
    /usr/share/keyrings/mongodb-server-*.gpg \
    /usr/share/keyrings/redis-archive-keyring.gpg \
    /etc/apt/sources.list.d/mongodb-org-*.list \
    /etc/apt/sources.list.d/redis.list

COPY --from=stage-bin /bin/any-sync-bundle /usr/local/bin/any-sync-bundle

ENTRYPOINT ["/usr/local/bin/any-sync-bundle"]
CMD ["start-all-in-one"]

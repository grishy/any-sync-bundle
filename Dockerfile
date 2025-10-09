# syntax=docker/dockerfile:1


# 
# Stage: Initial bin build
# 
FROM --platform=$BUILDPLATFORM golang:1.25.1-alpine AS stage-bin
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
# Stage: stage-final-minimal
# 
FROM gcr.io/distroless/static-debian12 AS stage-release-minimal

COPY --from=stage-bin /bin/any-sync-bundle /usr/local/bin/any-sync-bundle

# Bundle network ports (TCP 33010, UDP 33020)
EXPOSE 33010
EXPOSE 33020/udp

VOLUME /data

ENTRYPOINT ["/usr/local/bin/any-sync-bundle"]
CMD ["start-bundle"]

# 
# Stage: stage-final
#
FROM docker.io/redis/redis-stack-server:7.4.0-v2 AS stage-release

# Install prerequisites and MongoDB
RUN DEBIAN_FRONTEND=noninteractive \
 && apt-get update && apt-get install -y --no-install-recommends \
        gnupg \
        curl \
        ca-certificates \
 # Install MongoDB
 && curl -fsSL https://pgp.mongodb.com/server-8.0.asc | gpg -o /usr/share/keyrings/mongodb-server-8.0.gpg --dearmor \
 && echo "deb [ signed-by=/usr/share/keyrings/mongodb-server-8.0.gpg ] http://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/8.0 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-8.0.list \
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
    /usr/share/keyrings/mongodb-server-8.0.gpg \
    /etc/apt/sources.list.d/mongodb-org-8.0.list

COPY --from=stage-bin /bin/any-sync-bundle /usr/local/bin/any-sync-bundle

# Bundle network ports (TCP 33010, UDP 33020)
EXPOSE 33010
EXPOSE 33020/udp

VOLUME /data

ENTRYPOINT ["/usr/local/bin/any-sync-bundle"]
CMD ["start-all-in-one"]

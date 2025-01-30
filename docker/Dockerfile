# syntax=docker/dockerfile:1

# 
# Stage: Initial bin build
# 
FROM golang:1.23.5 AS stage-bin
WORKDIR /app

ARG VERSION
ARG COMMIT
ARG COMMIT_DATE

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./
RUN CGO_ENABLED=0 \
    go build \
       -trimpath \
       -ldflags="-w -s \
              -X github.com/grishy/any-sync-bundle/cmd.version=${VERSION} \
              -X github.com/grishy/any-sync-bundle/cmd.commit=${COMMIT} \
              -X github.com/grishy/any-sync-bundle/cmd.date=${COMMIT_DATE}" \
       -o any-sync-bundle

# 
# Stage: stage-final
#
FROM docker.io/redis/redis-stack-server:7.4.0-v2 AS stage-release

# Install supervisor and tool to intall mongoDB
RUN DEBIAN_FRONTEND=noninteractive \
 && apt-get update && apt-get install -y --no-install-recommends \
        gnupg \
        curl \
        ca-certificates \
        supervisor \
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

COPY ./docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY --from=stage-bin /app/any-sync-bundle /usr/local/bin/any-sync-bundle

# All default ports of any-sync-bundle
# NOTE: Need to specify UDP, because the default is TCP
EXPOSE 33010-33013 \
       33020-33023/udp

VOLUME /data

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]

# 
# Stage: stage-final-minimal
# 
FROM gcr.io/distroless/static-debian12 AS stage-release-minimal

COPY --from=stage-bin /app/any-sync-bundle /usr/local/bin/any-sync-bundle

# All default ports of any-sync-bundle
# NOTE: Need to specify UDP, because the default is TCP
EXPOSE 33010-33013 \
       33020-33023/udp

VOLUME /data

ENTRYPOINT ["/usr/local/bin/any-sync-bundle"]
CMD ["start"]

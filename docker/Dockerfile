# syntax=docker/dockerfile:1

FROM golang:1.23.5 AS build
WORKDIR /app

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

FROM gcr.io/distroless/static-debian12

COPY --from=build /app/any-sync-bundle /usr/local/bin/any-sync-bundle

# All default ports of any-sync-bundle
# NOTE: Need to specify UDP, because the default is TCP
EXPOSE 33010-33013 \
       33020-33023/udp

VOLUME /data

ENTRYPOINT ["/usr/local/bin/any-sync-bundle"]
CMD ["start"]

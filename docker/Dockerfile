# syntax=docker/dockerfile:1

FROM golang:1.23.5 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./ ./
RUN go build -ldflags="-s -w" -o any-sync-bundle

FROM gcr.io/distroless/base-debian12
LABEL maintainer="Sergei G. <mail@grishy.dev>"
LABEL description="Anytype Self-Hosting: All-in-One Prepared for You"

COPY --from=build /app/any-sync-bundle /usr/local/bin/any-sync-bundle

# All default ports of any-sync-bundle
# NOTE: Need to specify UDP, because the default is TCP
EXPOSE 33010 33011/udp \
       33020 33021/udp \
       33030 33031/udp \
       33040 33041/udp

VOLUME /data

CMD ["/usr/local/bin/any-sync-bundle"]

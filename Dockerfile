FROM golang:1.23 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./ ./
RUN go build -o app-sync-bundle

FROM scratch AS release 
COPY --from=build /app/app-sync-bundle /app-sync-bundle

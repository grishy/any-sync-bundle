package storepb

//go:generate protoc --go_out=. --go_opt=default_api_level=API_OPAQUE --go_opt=paths=source_relative store.proto

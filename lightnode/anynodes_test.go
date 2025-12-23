package lightnode

import (
	"reflect"
	"testing"

	filenodeConfig "github.com/anyproto/any-sync-filenode/config"
	"github.com/anyproto/any-sync-filenode/store/s3store"
	"github.com/stretchr/testify/assert"
)

func TestSelectFileStore_S3(t *testing.T) {
	cfg := &filenodeConfig.Config{
		S3Store: s3store.Config{
			Bucket:   "my-bucket",
			Endpoint: "https://s3.amazonaws.com",
		},
	}

	store := selectFileStore(cfg, "/tmp/filestore")

	// Check type name since s3store.New() returns an interface
	typeName := reflect.TypeOf(store).String()
	assert.Contains(t, typeName, "s3store", "should return S3 store when bucket is configured")
}

func TestSelectFileStore_BadgerDB(t *testing.T) {
	cfg := &filenodeConfig.Config{
		S3Store: s3store.Config{
			// Empty bucket means no S3
			Bucket: "",
		},
	}

	store := selectFileStore(cfg, "/tmp/filestore")

	// Check type name
	typeName := reflect.TypeOf(store).String()
	assert.Contains(t, typeName, "LightFileNodeStore", "should return BadgerDB store when bucket is empty")
}

func TestSelectFileStore_EmptyS3Config(t *testing.T) {
	cfg := &filenodeConfig.Config{
		// Default zero value for S3Store
	}

	store := selectFileStore(cfg, "/tmp/filestore")

	// Check type name
	typeName := reflect.TypeOf(store).String()
	assert.Contains(t, typeName, "LightFileNodeStore", "should return BadgerDB store for empty S3 config")
}

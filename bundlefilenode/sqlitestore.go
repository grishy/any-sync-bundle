package bundlefilenode

import (
	"context"

	"github.com/anyproto/any-sync-filenode/store/s3store"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/commonfile/fileblockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
)

var _ s3store.S3Store = (*SQLite)(nil)

const CName = fileblockstore.CName

type SQLite struct{}

func NewSqlStorage() *SQLite {
	return &SQLite{}
}

func (s *SQLite) Get(ctx context.Context, k cid.Cid) (blocks.Block, error) {
	// TODO implement me
	panic("implement me")
}

func (s *SQLite) GetMany(ctx context.Context, ks []cid.Cid) <-chan blocks.Block {
	// TODO implement me
	panic("implement me")
}

func (s *SQLite) Add(ctx context.Context, b []blocks.Block) error {
	// TODO implement me
	panic("implement me")
}

func (s *SQLite) Delete(ctx context.Context, c cid.Cid) error {
	// TODO implement me
	panic("implement me")
}

func (s *SQLite) DeleteMany(ctx context.Context, toDelete []cid.Cid) error {
	// TODO implement me
	panic("implement me")
}

func (s *SQLite) IndexGet(ctx context.Context, key string) (value []byte, err error) {
	// TODO implement me
	panic("implement me")
}

func (s *SQLite) IndexPut(ctx context.Context, key string, value []byte) (err error) {
	// TODO implement me
	panic("implement me")
}

func (s *SQLite) Init(a *app.App) (err error) {
	// TODO implement me
	return nil
}

func (s *SQLite) Name() (name string) {
	return CName
}

func (s *SQLite) Run(ctx context.Context) (err error) {
	// TODO implement me
	panic("implement me")
}

func (s *SQLite) Close(ctx context.Context) (err error) {
	// TODO implement me
	panic("implement me")
}

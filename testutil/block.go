package testutil

import (
	"crypto/rand"
	"io"

	"github.com/anyproto/any-sync/util/cidutil"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
)

// TODO: Replace with original function from Anytype after fixing the issue on Windowns
// https://github.com/anyproto/any-sync-filenode/issues/142

func NewRandSpaceId() string {
	b := NewRandBlock(256)
	return b.Cid().String() + ".123456"
}

func NewRandBlock(size int) blocks.Block {
	p := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, p)
	if err != nil {
		panic("can't fill testdata from random: " + err.Error())
	}

	c, err := cidutil.NewCidFromBytes(p)
	if err != nil {
		panic("can't create CID from bytes: " + err.Error())
	}

	b, err := blocks.NewBlockWithCid(p, cid.MustParse(c))
	if err != nil {
		panic("can't create block with CID: " + err.Error())
	}
	return b
}

func BlocksToKeys(bs []blocks.Block) (cids []cid.Cid) {
	cids = make([]cid.Cid, len(bs))
	for i, b := range bs {
		cids[i] = b.Cid()
	}

	return
}

func NewRandBlocks(l int) []blocks.Block {
	bs := make([]blocks.Block, l)
	for i := range bs {
		bs[i] = NewRandBlock(10 * l)
	}

	return bs
}

func NewRandCid() cid.Cid {
	return NewRandBlock(1024).Cid()
}

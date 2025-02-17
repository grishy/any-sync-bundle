package lightfilenoderpc

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/anyproto/any-sync/commonfile/fileproto/fileprotoerr"
	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
)

const (
	// Store reference counter and block's data
	// [8-byte refCount][raw block bytes]
	kPrefixBlock = "fn:b:"
)

type dbBlock struct {
	refCount uint64
	data     []byte
}

func newDbBlock(data []byte) *dbBlock {
	return &dbBlock{
		refCount: 1,
		data:     data,
	}
}

func encodeDbBlock(b *dbBlock) []byte {
	buf := make([]byte, 8+len(b.data))
	binary.LittleEndian.PutUint64(buf, b.refCount)
	copy(buf[8:], b.data)
	return buf
}

func decodeDbBlock(buf []byte) *dbBlock {
	_ = buf[7] // bounds check hint to compiler; see golang.org/issue/14808

	return &dbBlock{
		refCount: binary.LittleEndian.Uint64(buf[:8]),
		data:     buf[8:],
	}
}

func decodeDbBlockData(buf []byte) []byte {
	_ = buf[7] // bounds check hint to compiler; see golang.org/issue/14808
	return buf[8:]
}

func uint64ToBytes(i uint64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], i)
	return buf[:]
}

func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

func (r *lightfilenoderpc) checkBlock(ctx context.Context, k cid.Cid) (bool, error) {
	return false, nil
}

func (r *lightfilenoderpc) getBlock(ctx context.Context, k cid.Cid, wait bool) (*dbBlock, error) {
	key := []byte(kPrefixBlock + k.String())

	if wait {
		// NOTE: Use cond with broadcast?
		// TODO: If "wait" is true, you might do some blocking logic, but let's skip for simplicity
		// NOTE: 2025-02-15: Wait is not used on client side...
		// TODO: Implement simple for loop with sleep for waiting
		panic("wait not supported")
	}

	var block dbBlock

	// NOTE: Idea, use Meta to store reference counter
	// Store here number of links and if more 255, store as part of value.
	e := badger.NewEntry(key, nil).WithMeta(1<<63 - 1)
	_ = e

	err := r.badgerDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return fileprotoerr.ErrCIDNotFound
			}

			return fmt.Errorf("failed to get block: %w", err)
		}

		val, errCopy := item.ValueCopy(nil)
		if errCopy != nil {
			return fmt.Errorf("failed to copy block value: %w", errCopy)
		}

		decodeDbBlockData(val)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed transaction: %w", err)
	}

	return &block, nil
}

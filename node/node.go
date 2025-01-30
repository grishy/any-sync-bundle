package node

import (
	"os"

	"github.com/anyproto/any-sync/app/logger"
	"go.uber.org/zap"
)

var log = logger.NewNamed("node")

func MustMkdirAll(p string) {
	// TODO: Remove when merged https://github.com/anyproto/any-sync/pull/374

	if err := os.MkdirAll(p, 0o775); err != nil {
		log.Panic("can't create directory network store", zap.Error(err))
	}
}

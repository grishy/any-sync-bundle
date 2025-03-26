//go:generate go tool moq -fmt gofumpt -rm -out store_moq_test.go . configService DBService

package lightdb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"go.uber.org/zap"
)

const (
	CName = "light.db"

	// GC configuration
	defaultGCInterval    = 29 * time.Hour // Prime number to avoid alignment
	defaultMaxGCDuration = time.Minute
	defaultGCThreshold   = 0.5 // Reclaim files with >= 50% garbage
)

var log = logger.NewNamed(CName)

type DBService interface {
	app.ComponentRunnable

	TxView(f func(txn *badger.Txn) error) error
	TxUpdate(f func(txn *badger.Txn) error) error
}

type lightdb struct {
	srvCfg configService
	cfg    storeConfig
	db     *badger.DB
}

type storeConfig struct {
	gcInterval    time.Duration
	maxGCDuration time.Duration
	gcThreshold   float64
}

type configService interface {
	app.Component
	GetDBDir() string
}

func New() *lightdb {
	return &lightdb{
		cfg: storeConfig{
			gcInterval:    defaultGCInterval,
			maxGCDuration: defaultMaxGCDuration,
			gcThreshold:   defaultGCThreshold,
		},
	}
}

//
// App Component
//

func (r *lightdb) Init(a *app.App) error {
	log.Info("call Init")

	r.srvCfg = app.MustComponent[configService](a)

	return nil
}

func (r *lightdb) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (r *lightdb) Run(ctx context.Context) error {
	log.Info("call Run")

	storePath := r.srvCfg.GetDBDir()

	opts := badger.DefaultOptions(storePath).
		WithLogger(badgerLogger{}).
		WithCompression(options.None).
		WithZSTDCompressionLevel(0)

	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to open badger db: %w", err)
	}

	r.db = db

	go startRunGC(ctx, r.cfg, r.db)
	return nil
}

func (r *lightdb) Close(ctx context.Context) error {
	log.Info("call Close")
	return r.db.Close()
}

//
// Component methods
//

func (r *lightdb) TxView(f func(txn *badger.Txn) error) error {
	return r.db.View(f)
}

func (r *lightdb) TxUpdate(f func(txn *badger.Txn) error) error {
	return r.db.Update(f)
}

//
// Garbage collection
//

type dbGC interface {
	RunValueLogGC(float64) error
}

func startRunGC(ctx context.Context, cfg storeConfig, db dbGC) {
	log.Info("starting badger garbage collection routine")
	ticker := time.NewTicker(cfg.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping badger garbage collection routine")
			return
		case <-ticker.C:
			start := time.Now()
			gcCount := 0

			// Run GC until either we hit time limit or no more files need GC
			for time.Since(start) < cfg.maxGCDuration {
				gcCount++
				if err := db.RunValueLogGC(cfg.gcThreshold); err != nil {
					if errors.Is(err, badger.ErrNoRewrite) {
						break // No more files need GC
					}
					log.Warn("badger gc failed", zap.Error(err))
					break
				}
			}

			log.Info("badger garbage collection completed",
				zap.Duration("duration", time.Since(start)),
				zap.Int("filesGCed", gcCount))
		}
	}
}

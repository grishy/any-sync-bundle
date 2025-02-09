package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	consensus "github.com/anyproto/any-sync-consensusnode"
	"github.com/anyproto/any-sync-node/config"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"

	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

const CName = "light.consensus.db"

//go:embed schema.sql
var sqlSchema embed.FS

var log = logger.NewNamed(CName)

func New() *service {
	return &service{}
}

type cfgSrv interface {
	GetDBPath() string
}

type service struct {
	cfg    cfgSrv
	dbPath string
	db     *sql.DB
	mu     sync.RWMutex // Protect concurrent access (even though it's single-instance)
}

//
// App Component
//

func (s *service) Init(a *app.App) (err error) {
	log.Info("call Init")
	s.cfg = a.MustComponent(config.CName).(cfgSrv)

	return nil
}

func (s *service) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (s *service) Run(ctx context.Context) error {
	log.Info("call Run")

	s.mu.Lock()
	defer s.mu.Unlock()

	s.dbPath = s.cfg.GetDBPath()

	log.Info("sqlite db path", zap.String("dbPath", s.dbPath))

	if err := os.MkdirAll(filepath.Dir(s.dbPath), 0o700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Configure connection pool
	s.configureConnPool()

	// Configure SQLite settings
	if err := s.configureSQLite(); err != nil {
		_ = s.db.Close()
		return fmt.Errorf("failed to configure SQLite: %w", err)
	}

	if err := s.createTables(ctx); err != nil {
		_ = s.db.Close()
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

func (s *service) configureConnPool() {
	s.db.SetMaxOpenConns(1)    // SQLite requires single writer
	s.db.SetMaxIdleConns(10)   // Allow multiple readers
	s.db.SetConnMaxLifetime(0) // Keep connections open indefinitely
}

func (s *service) configureSQLite() error {
	settings := []struct {
		pragma string
		value  string
	}{
		{"foreign_keys", "1"},    // Enable foreign key support
		{"journal_mode", "WAL"},  // Enable Write-Ahead Logging
		{"busy_timeout", "5000"}, // Wait up to 5s on busy DB
	}

	for _, setting := range settings {
		if _, err := s.db.Exec(fmt.Sprintf("PRAGMA %s = %s;", setting.pragma, setting.value)); err != nil {
			return fmt.Errorf("failed to set %s: %w", setting.pragma, err)
		}
	}
	return nil
}

func (s *service) Close(ctx context.Context) error {
	log.Info("call Close")

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	s.db = nil
	return nil
}

//
// Component
//

func (s *service) AddLog(ctx context.Context, l consensus.Log) error {
	log.Info("call AddLog", zap.String("logID", l.Id))

	return nil
}

func (s *service) DeleteLog(ctx context.Context, logId string) error {
	log.Info("call DeleteLog", zap.String("logId", logId))

	return nil
}

func (s *service) AddRecord(ctx context.Context, logId string, record consensus.Record) error {
	log.Info("call AddRecord", zap.String("logId", logId), zap.Any("record", record))

	return nil
}

func (s *service) FetchLog(ctx context.Context, logId string) (consensus.Log, error) {
	log.Info("call FetchLog", zap.String("logId", logId))

	return consensus.Log{}, nil
}

//
// Helper
//

func (s *service) createTables(ctx context.Context) error {
	schema, err := sqlSchema.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema.sql: %w", err)
	}

	_, err = s.db.ExecContext(ctx, string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

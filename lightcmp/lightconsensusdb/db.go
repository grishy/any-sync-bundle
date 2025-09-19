package lightconsensusdb

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	consensus "github.com/anyproto/any-sync-consensusnode"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/consensus/consensusproto/consensuserr"

	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

// TODO: Use a BadgerDB because filenode already use it.

const CName = "light.consensus.db"

//go:embed schema.sql
var sqlSchema embed.FS

var log = logger.NewNamed(CName)

func New() *lightConsensusDB {
	return &lightConsensusDB{}
}

type cfgSrv interface {
	GetConsensusDBPath() string
}

type lightConsensusDB struct {
	cfg    cfgSrv
	dbPath string
	db     *sql.DB
	mu     sync.RWMutex // Protect concurrent access (even though it's single-instance)
}

//
// App Component.
//

func (d *lightConsensusDB) Init(a *app.App) (err error) {
	log.Info("call Init")
	d.cfg = app.MustComponent[cfgSrv](a)

	return nil
}

func (d *lightConsensusDB) Name() (name string) {
	return CName
}

//
// App Component Runnable.
//

func (d *lightConsensusDB) Run(ctx context.Context) error {
	log.Info("call Run")

	d.mu.Lock()
	defer d.mu.Unlock()

	d.dbPath = d.cfg.GetConsensusDBPath()

	log.Info("sqlite db path", zap.String("dbPath", d.dbPath))

	// Ensure directory exists.
	if err := os.MkdirAll(filepath.Dir(d.dbPath), 0o700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database connection.
	db, err := sql.Open("sqlite", d.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	d.db = db

	// Configure connection pool.
	d.configureConnPool()

	// Configure SQLite settings.
	if err := d.configureSQLite(); err != nil {
		_ = d.db.Close()
		return fmt.Errorf("failed to configure SQLite: %w", err)
	}

	if err := d.createTables(ctx); err != nil {
		_ = d.db.Close()
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

func (d *lightConsensusDB) configureConnPool() {
	d.db.SetMaxOpenConns(1)    // SQLite requires single writer
	d.db.SetMaxIdleConns(10)   // Allow multiple readers
	d.db.SetConnMaxLifetime(0) // Keep connections open indefinitely
}

func (d *lightConsensusDB) configureSQLite() error {
	settings := []struct {
		pragma string
		value  string
	}{
		{"foreign_keys", "1"},    // Enable foreign key support
		{"journal_mode", "WAL"},  // Enable Write-Ahead Logging
		{"busy_timeout", "5000"}, // Wait up to 5s on busy DB
	}

	for _, setting := range settings {
		if _, err := d.db.Exec(fmt.Sprintf("PRAGMA %s = %s;", setting.pragma, setting.value)); err != nil {
			return fmt.Errorf("failed to set %s: %w", setting.pragma, err)
		}
	}
	return nil
}

func (d *lightConsensusDB) Close(ctx context.Context) error {
	log.Info("call Close")

	d.mu.Lock()
	defer d.mu.Unlock()

	if err := d.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	d.db = nil
	return nil
}

//
// Component.
//

func (d *lightConsensusDB) AddLog(ctx context.Context, l consensus.Log) error {
	log.Info("adding log", zap.String("logID", l.Id))

	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer func(tx *sql.Tx) {
		errRb := tx.Rollback()
		if errRb != nil {
			log.Warn("rollback transaction failed", zap.Error(errRb))
		}
	}(tx)

	// Check if log already exists.
	var exists bool
	err = tx.QueryRowContext(ctx, "SELECT 1 FROM logs WHERE id = ?", l.Id).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check log existence failed: %w", err)
	}
	if exists {
		return consensuserr.ErrLogExists
	}

	// Insert log.
	_, err = tx.ExecContext(ctx,
		"INSERT INTO logs (id, created_at) VALUES (?, datetime('now'))",
		l.Id,
	)
	if err != nil {
		return fmt.Errorf("insert log failed: %w", err)
	}

	// Insert initial records if any.
	if len(l.Records) > 0 {
		stmt, err := tx.PrepareContext(ctx,
			"INSERT INTO records (id, log_id, prev_id, payload, created_at) VALUES (?, ?, ?, ?, datetime('now'))")
		if err != nil {
			return fmt.Errorf("prepare record insert statement failed: %w", err)
		}
		defer stmt.Close()

		for _, record := range l.Records {
			_, err = stmt.ExecContext(ctx, record.Id, l.Id, record.PrevId, record.Payload)
			if err != nil {
				return fmt.Errorf("insert record failed: %w", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction failed: %w", err)
	}

	return nil
}

func (d *lightConsensusDB) DeleteLog(ctx context.Context, logId string) error {
	log.Info("deleting log", zap.String("logID", logId))

	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer func(tx *sql.Tx) {
		errRb := tx.Rollback()
		if errRb != nil {
			log.Warn("rollback transaction failed", zap.Error(errRb))
		}
	}(tx)

	// Check if log exists before deletion.
	var exists bool
	err = tx.QueryRowContext(ctx, "SELECT 1 FROM logs WHERE id = ?", logId).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check log existence failed: %w", err)
	}
	if !exists {
		return consensuserr.ErrLogNotFound
	}

	// Delete the log (records will be automatically deleted due to ON DELETE CASCADE).
	_, err = tx.ExecContext(ctx, "DELETE FROM logs WHERE id = ?", logId)
	if err != nil {
		return fmt.Errorf("delete log failed: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction failed: %w", err)
	}

	return nil
}

func (d *lightConsensusDB) AddRecord(ctx context.Context, logId string, record consensus.Record) error {
	log.Info("adding record",
		zap.String("logId", logId),
		zap.String("recordId", record.Id),
		zap.String("prevId", record.PrevId),
	)

	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer func(tx *sql.Tx) {
		errRb := tx.Rollback()
		if errRb != nil {
			log.Warn("rollback transaction failed", zap.Error(errRb))
		}
	}(tx)

	// Check if log exists.
	var exists bool
	err = tx.QueryRowContext(ctx, "SELECT 1 FROM logs WHERE id = ?", logId).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check log existence failed: %w", err)
	}
	if !exists {
		return consensuserr.ErrConflict
	}

	// Get the latest record for this log.
	var lastRecordId *string
	err = tx.QueryRowContext(ctx, `
        SELECT id 
        FROM records 
        WHERE log_id = ? 
        ORDER BY created_at DESC 
        LIMIT 1`, logId).Scan(&lastRecordId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("query last record failed: %w", err)
	}

	// Check if the record chain is valid.
	if (lastRecordId == nil && record.PrevId != "") ||
		(lastRecordId != nil && *lastRecordId != record.PrevId) {
		return consensuserr.ErrConflict
	}

	// Check if record already exists.
	var recordExists bool
	err = tx.QueryRowContext(ctx, "SELECT 1 FROM records WHERE id = ?", record.Id).Scan(&recordExists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check record existence failed: %w", err)
	}
	if recordExists {
		return consensuserr.ErrConflict
	}

	// Insert the new record.
	_, err = tx.ExecContext(ctx,
		"INSERT INTO records (id, log_id, prev_id, payload, created_at) VALUES (?, ?, ?, ?, datetime('now'))",
		record.Id, logId, record.PrevId, record.Payload)
	if err != nil {
		return fmt.Errorf("insert record failed: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction failed: %w", err)
	}

	return nil
}

func (d *lightConsensusDB) FetchLog(ctx context.Context, logId string) (consensus.Log, error) {
	log.Info("fetching log", zap.String("logId", logId))

	d.mu.RLock()
	defer d.mu.RUnlock()

	tx, err := d.db.BeginTx(ctx, &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		return consensus.Log{}, fmt.Errorf("begin transaction failed: %w", err)
	}
	defer func(tx *sql.Tx) {
		errRb := tx.Rollback()
		if errRb != nil {
			log.Warn("rollback transaction failed", zap.Error(errRb))
		}
	}(tx)

	// First check if log exists and get its details.
	var l consensus.Log
	err = tx.QueryRowContext(ctx, `
        SELECT id 
        FROM logs 
        WHERE id = ?`, logId).Scan(&l.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return consensus.Log{}, consensuserr.ErrLogNotFound
		}
		return consensus.Log{}, fmt.Errorf("query log failed: %w", err)
	}

	// Fetch all records for this log.
	rows, err := tx.QueryContext(ctx, `
        SELECT id, prev_id, payload, created_at 
        FROM records 
        WHERE log_id = ? 
        ORDER BY created_at ASC`, logId)
	if err != nil {
		return consensus.Log{}, fmt.Errorf("query records failed: %w", err)
	}
	defer rows.Close()

	// Scan all records.
	l.Records = make([]consensus.Record, 0)
	for rows.Next() {
		var record consensus.Record
		if err := rows.Scan(&record.Id, &record.PrevId, &record.Payload, &record.Created); err != nil {
			return consensus.Log{}, fmt.Errorf("scan record failed: %w", err)
		}
		l.Records = append(l.Records, record)
	}

	if err = rows.Err(); err != nil {
		return consensus.Log{}, fmt.Errorf("iterate records failed: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return consensus.Log{}, fmt.Errorf("commit transaction failed: %w", err)
	}

	return l, nil
}

//
// Helper.
//

func (d *lightConsensusDB) createTables(ctx context.Context) error {
	// TODO: Add migrations later go-migrate or similar.
	schema, err := sqlSchema.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema.sql: %w", err)
	}

	_, err = d.db.ExecContext(ctx, string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

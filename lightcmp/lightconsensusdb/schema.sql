CREATE TABLE IF NOT EXISTS logs (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS records (
    id TEXT PRIMARY KEY,
    log_id TEXT NOT NULL,
    prev_id TEXT,
    payload BLOB NOT NULL,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (log_id) REFERENCES logs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_records_log_id ON records(log_id);
CREATE TABLE IF NOT EXISTS logs (
    id TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS records (
    id TEXT PRIMARY KEY,
    log_id TEXT NOT NULL,
    prev_id TEXT,
    payload BLOB NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (log_id) REFERENCES logs(id)
);

-- Add an index for efficient lookups by log_id
CREATE INDEX IF NOT EXISTS idx_records_log_id ON records (log_id);

-- Add an index to prev id
CREATE INDEX IF NOT EXISTS idx_records_prev_id ON records (prev_id);

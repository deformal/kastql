CREATE TABLE IF NOT EXISTS query_log (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp       TEXT NOT NULL DEFAULT (datetime('now')),
    operation_type  TEXT,
    operation_name  TEXT,
    duration_ms     INTEGER NOT NULL,
    success         INTEGER NOT NULL,
    error_message   TEXT,
    services_called TEXT NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_query_log_timestamp ON query_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_query_log_success ON query_log(success);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

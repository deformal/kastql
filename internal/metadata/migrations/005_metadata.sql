-- CORS allowed origins
CREATE TABLE IF NOT EXISTS cors_origins (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    origin     TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- IP allow/deny rules  (mode = 'allow' | 'deny')
CREATE TABLE IF NOT EXISTS ip_rules (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    cidr       TEXT NOT NULL UNIQUE,
    mode       TEXT NOT NULL DEFAULT 'deny',
    note       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Persisted (pre-approved) queries  (id = client-supplied SHA-256 hash)
CREATE TABLE IF NOT EXISTS persisted_queries (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    query      TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Audit log: admin mutations
CREATE TABLE IF NOT EXISTS audit_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    admin      TEXT NOT NULL,
    action     TEXT NOT NULL,
    detail     TEXT NOT NULL DEFAULT '',
    ip         TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Blocked request log (sampled, capped at 10 000 rows by trigger)
CREATE TABLE IF NOT EXISTS blocked_requests (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    reason     TEXT NOT NULL,
    ip         TEXT NOT NULL DEFAULT '',
    path       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Delete oldest rows when blocked_requests exceeds 10 000
CREATE TRIGGER IF NOT EXISTS trg_blocked_cap
AFTER INSERT ON blocked_requests
BEGIN
    DELETE FROM blocked_requests
    WHERE id IN (
        SELECT id FROM blocked_requests ORDER BY id ASC LIMIT -1 OFFSET 10000
    );
END;

-- Security feature-flag defaults (INSERT OR IGNORE so existing values are untouched)
INSERT OR IGNORE INTO settings (key, value) VALUES ('cors_enabled',          '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cors_allow_all',        '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('ip_filter_enabled',     '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('ip_filter_default',     'allow');
INSERT OR IGNORE INTO settings (key, value) VALUES ('rate_limit_enabled',    '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('rate_limit_global_rpm', '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('rate_limit_ip_rpm',     '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('rate_limit_mutation_rpm','0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('query_depth_limit',     '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('query_complexity_limit','0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('query_alias_limit',     '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('query_directive_limit', '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('query_timeout_ms',      '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('max_request_body_kb',   '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('max_response_body_kb',  '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('persisted_only',        '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('audit_log_enabled',     '1');
INSERT OR IGNORE INTO settings (key, value) VALUES ('ws_max_connections',    '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('batch_queries_enabled', '1');

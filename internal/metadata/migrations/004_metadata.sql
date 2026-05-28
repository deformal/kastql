CREATE TABLE IF NOT EXISTS settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Default values (INSERT OR IGNORE so existing values are never overwritten)
INSERT OR IGNORE INTO settings (key, value) VALUES ('introspection_enabled', '1');

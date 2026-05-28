-- Per-service retry and timeout configuration
ALTER TABLE services ADD COLUMN timeout_ms  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE services ADD COLUMN retry_count INTEGER NOT NULL DEFAULT 0;

-- Health monitor + cache settings
INSERT OR IGNORE INTO settings (key, value) VALUES ('health_check_interval_s',  '30');
INSERT OR IGNORE INTO settings (key, value) VALUES ('health_check_timeout_ms',  '5000');
INSERT OR IGNORE INTO settings (key, value) VALUES ('circuit_fail_threshold',   '3');
INSERT OR IGNORE INTO settings (key, value) VALUES ('circuit_recovery_s',       '30');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cache_enabled',            '0');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cache_default_ttl_s',      '60');
INSERT OR IGNORE INTO settings (key, value) VALUES ('cache_max_entries',        '1000');

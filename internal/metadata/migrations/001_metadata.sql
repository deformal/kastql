CREATE TABLE IF NOT EXISTS services (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    url         TEXT NOT NULL,
    type        TEXT NOT NULL CHECK(type IN ('federation', 'stitching')),
    headers     TEXT NOT NULL DEFAULT '{}',
    enabled     INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS relationships (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL UNIQUE,
    source_service  TEXT NOT NULL,
    source_type     TEXT NOT NULL,
    source_field    TEXT NOT NULL,
    target_service  TEXT NOT NULL,
    target_type     TEXT NOT NULL,
    join_config     TEXT NOT NULL DEFAULT '{}',
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS permissions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    role        TEXT NOT NULL,
    service     TEXT NOT NULL,
    type_name   TEXT NOT NULL,
    field_name  TEXT NOT NULL DEFAULT '',
    allow       INTEGER NOT NULL DEFAULT 1,
    condition   TEXT NOT NULL DEFAULT '{}',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(role, service, type_name, field_name)
);

CREATE TABLE IF NOT EXISTS rest_endpoints (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL UNIQUE,
    method          TEXT NOT NULL CHECK(method IN ('GET','POST','PUT','DELETE','PATCH')),
    path            TEXT NOT NULL UNIQUE,
    graphql_query   TEXT NOT NULL,
    variables       TEXT NOT NULL DEFAULT '{}',
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS schema_cache (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    service_name    TEXT NOT NULL UNIQUE,
    sdl             TEXT NOT NULL,
    fetched_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

package metadata

import (
	"database/sql"
	"fmt"
	"time"
)

// --- Services ---

func (s *Store) UpsertService(svc *Service) error {
	_, err := s.db.Exec(`
		INSERT INTO services (name, url, type, headers, enabled, timeout_ms, retry_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(name) DO UPDATE SET
			url         = excluded.url,
			type        = excluded.type,
			headers     = excluded.headers,
			enabled     = excluded.enabled,
			timeout_ms  = excluded.timeout_ms,
			retry_count = excluded.retry_count,
			updated_at  = excluded.updated_at
	`, svc.Name, svc.URL, svc.Type, svc.Headers, boolToInt(svc.Enabled), svc.TimeoutMs, svc.RetryCount)
	if err != nil {
		return fmt.Errorf("upsert service %s: %w", svc.Name, err)
	}
	return nil
}

func (s *Store) DeleteService(name string) error {
	_, err := s.db.Exec(`DELETE FROM services WHERE name = ?`, name)
	return err
}

func (s *Store) GetService(name string) (*Service, error) {
	row := s.db.QueryRow(`
		SELECT id, name, url, type, headers, enabled, timeout_ms, retry_count, created_at, updated_at
		FROM services WHERE name = ?
	`, name)
	return scanService(row)
}

func (s *Store) ListServices() ([]*Service, error) {
	rows, err := s.db.Query(`
		SELECT id, name, url, type, headers, enabled, timeout_ms, retry_count, created_at, updated_at
		FROM services ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var svcs []*Service
	for rows.Next() {
		svc, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		svcs = append(svcs, svc)
	}
	return svcs, rows.Err()
}

func scanService(s scanner) (*Service, error) {
	var svc Service
	var createdAt, updatedAt string
	var enabled int
	err := s.Scan(
		&svc.ID, &svc.Name, &svc.URL, &svc.Type,
		&svc.Headers, &enabled, &svc.TimeoutMs, &svc.RetryCount,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	svc.Enabled = enabled == 1
	svc.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	svc.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return &svc, nil
}

// --- Schema cache ---

func (s *Store) UpsertSchemaCache(serviceName, sdl string) error {
	_, err := s.db.Exec(`
		INSERT INTO schema_cache (service_name, sdl, fetched_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(service_name) DO UPDATE SET
			sdl        = excluded.sdl,
			fetched_at = excluded.fetched_at
	`, serviceName, sdl)
	return err
}

func (s *Store) GetSchemaCache(serviceName string) (*SchemaCache, error) {
	var sc SchemaCache
	var fetchedAt string
	err := s.db.QueryRow(`
		SELECT id, service_name, sdl, fetched_at FROM schema_cache WHERE service_name = ?
	`, serviceName).Scan(&sc.ID, &sc.ServiceName, &sc.SDL, &fetchedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sc.FetchedAt, _ = time.Parse("2006-01-02 15:04:05", fetchedAt)
	return &sc, nil
}

func (s *Store) DeleteSchemaCache(serviceName string) error {
	_, err := s.db.Exec(`DELETE FROM schema_cache WHERE service_name = ?`, serviceName)
	return err
}

// --- helpers ---

type scanner interface {
	Scan(dest ...any) error
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

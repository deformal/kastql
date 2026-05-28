package metadata

import (
	"database/sql"
	"fmt"
	"time"
)

func (s *Store) UpsertRelationship(rel *Relationship) error {
	_, err := s.db.Exec(`
		INSERT INTO relationships (name, source_service, source_type, source_field, target_service, target_type, join_config)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			source_service = excluded.source_service,
			source_type    = excluded.source_type,
			source_field   = excluded.source_field,
			target_service = excluded.target_service,
			target_type    = excluded.target_type,
			join_config    = excluded.join_config
	`, rel.Name, rel.SourceService, rel.SourceType, rel.SourceField,
		rel.TargetService, rel.TargetType, rel.JoinConfig)
	if err != nil {
		return fmt.Errorf("upsert relationship %s: %w", rel.Name, err)
	}
	return nil
}

func (s *Store) DeleteRelationship(name string) error {
	_, err := s.db.Exec(`DELETE FROM relationships WHERE name = ?`, name)
	return err
}

func (s *Store) GetRelationship(name string) (*Relationship, error) {
	row := s.db.QueryRow(`
		SELECT id, name, source_service, source_type, source_field, target_service, target_type, join_config, created_at
		FROM relationships WHERE name = ?
	`, name)
	return scanRelationship(row)
}

func (s *Store) ListRelationships() ([]*Relationship, error) {
	rows, err := s.db.Query(`
		SELECT id, name, source_service, source_type, source_field, target_service, target_type, join_config, created_at
		FROM relationships ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []*Relationship
	for rows.Next() {
		rel, err := scanRelationship(rows)
		if err != nil {
			return nil, err
		}
		rels = append(rels, rel)
	}
	return rels, rows.Err()
}

func scanRelationship(s scanner) (*Relationship, error) {
	var rel Relationship
	var createdAt string
	err := s.Scan(&rel.ID, &rel.Name, &rel.SourceService, &rel.SourceType,
		&rel.SourceField, &rel.TargetService, &rel.TargetType, &rel.JoinConfig, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rel.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &rel, nil
}

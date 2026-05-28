package metadata

import (
	"database/sql"
	"fmt"
	"time"
)

func (s *Store) UpsertPermission(p *Permission) error {
	_, err := s.db.Exec(`
		INSERT INTO permissions (role, service, type_name, field_name, allow, condition)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(role, service, type_name, field_name) DO UPDATE SET
			allow     = excluded.allow,
			condition = excluded.condition
	`, p.Role, p.Service, p.TypeName, p.FieldName, boolToInt(p.Allow), p.Condition)
	if err != nil {
		return fmt.Errorf("upsert permission %s/%s.%s: %w", p.Role, p.TypeName, p.FieldName, err)
	}
	return nil
}

func (s *Store) DeletePermission(role, service, typeName, fieldName string) error {
	_, err := s.db.Exec(
		`DELETE FROM permissions WHERE role = ? AND service = ? AND type_name = ? AND field_name = ?`,
		role, service, typeName, fieldName,
	)
	return err
}

func (s *Store) ListPermissions() ([]*Permission, error) {
	rows, err := s.db.Query(`
		SELECT id, role, service, type_name, field_name, allow, condition, created_at
		FROM permissions ORDER BY role, type_name, field_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []*Permission
	for rows.Next() {
		p, err := scanPermission(rows)
		if err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func scanPermission(s scanner) (*Permission, error) {
	var p Permission
	var allow int
	var createdAt string
	err := s.Scan(&p.ID, &p.Role, &p.Service, &p.TypeName, &p.FieldName, &allow, &p.Condition, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Allow = allow != 0
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &p, nil
}

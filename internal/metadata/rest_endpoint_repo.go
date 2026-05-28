package metadata

import (
	"database/sql"
	"fmt"
	"time"
)

func (s *Store) UpsertRESTEndpoint(ep *RESTEndpoint) error {
	_, err := s.db.Exec(`
		INSERT INTO rest_endpoints (name, method, path, graphql_query, variables)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			method        = excluded.method,
			path          = excluded.path,
			graphql_query = excluded.graphql_query,
			variables     = excluded.variables
	`, ep.Name, ep.Method, ep.Path, ep.GraphQLQuery, ep.Variables)
	if err != nil {
		return fmt.Errorf("upsert rest endpoint %s: %w", ep.Name, err)
	}
	return nil
}

func (s *Store) DeleteRESTEndpoint(name string) error {
	_, err := s.db.Exec(`DELETE FROM rest_endpoints WHERE name = ?`, name)
	return err
}

func (s *Store) GetRESTEndpoint(name string) (*RESTEndpoint, error) {
	row := s.db.QueryRow(`
		SELECT id, name, method, path, graphql_query, variables, created_at
		FROM rest_endpoints WHERE name = ?
	`, name)
	return scanRESTEndpoint(row)
}

func (s *Store) GetRESTEndpointByPath(method, path string) (*RESTEndpoint, error) {
	row := s.db.QueryRow(`
		SELECT id, name, method, path, graphql_query, variables, created_at
		FROM rest_endpoints WHERE method = ? AND path = ?
	`, method, path)
	return scanRESTEndpoint(row)
}

func (s *Store) ListRESTEndpoints() ([]*RESTEndpoint, error) {
	rows, err := s.db.Query(`
		SELECT id, name, method, path, graphql_query, variables, created_at
		FROM rest_endpoints ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var eps []*RESTEndpoint
	for rows.Next() {
		ep, err := scanRESTEndpoint(rows)
		if err != nil {
			return nil, err
		}
		eps = append(eps, ep)
	}
	return eps, rows.Err()
}

func scanRESTEndpoint(s scanner) (*RESTEndpoint, error) {
	var ep RESTEndpoint
	var createdAt string
	err := s.Scan(&ep.ID, &ep.Name, &ep.Method, &ep.Path, &ep.GraphQLQuery, &ep.Variables, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ep.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &ep, nil
}

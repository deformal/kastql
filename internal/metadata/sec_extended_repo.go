package metadata

import (
	"errors"
	"fmt"
)

var (
	ErrCORSOriginNotFound    = errors.New("cors origin not found")
	ErrIPRuleNotFound        = errors.New("ip rule not found")
	ErrPersistedQueryExists  = errors.New("persisted query id already registered")
	ErrPersistedQueryNotFound = errors.New("persisted query not found")
)

// ── CORS Origins ──────────────────────────────────────────────────────────────

type CORSOrigin struct {
	ID        int64  `json:"id"`
	Origin    string `json:"origin"`
	CreatedAt string `json:"created_at"`
}

func (s *Store) AddCORSOrigin(origin string) (*CORSOrigin, error) {
	row := &CORSOrigin{}
	err := s.db.QueryRow(
		`INSERT INTO cors_origins (origin) VALUES (?) RETURNING id, origin, created_at`,
		origin,
	).Scan(&row.ID, &row.Origin, &row.CreatedAt)
	if err != nil {
		if isUniqueConstraint(err) {
			return nil, ErrNameTaken
		}
		return nil, fmt.Errorf("add cors origin: %w", err)
	}
	return row, nil
}

func (s *Store) ListCORSOrigins() ([]*CORSOrigin, error) {
	rows, err := s.db.Query(`SELECT id, origin, created_at FROM cors_origins ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list cors origins: %w", err)
	}
	defer rows.Close()
	var out []*CORSOrigin
	for rows.Next() {
		r := &CORSOrigin{}
		if err := rows.Scan(&r.ID, &r.Origin, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DeleteCORSOrigin(id int64) error {
	res, err := s.db.Exec(`DELETE FROM cors_origins WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete cors origin: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrCORSOriginNotFound
	}
	return nil
}

// ── IP Rules ──────────────────────────────────────────────────────────────────

type IPRule struct {
	ID        int64  `json:"id"`
	CIDR      string `json:"cidr"`
	Mode      string `json:"mode"` // "allow" | "deny"
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"`
}

func (s *Store) AddIPRule(cidr, mode, note string) (*IPRule, error) {
	if mode != "allow" && mode != "deny" {
		return nil, fmt.Errorf("ip rule mode must be 'allow' or 'deny'")
	}
	row := &IPRule{}
	err := s.db.QueryRow(
		`INSERT INTO ip_rules (cidr, mode, note) VALUES (?, ?, ?)
		 RETURNING id, cidr, mode, note, created_at`,
		cidr, mode, note,
	).Scan(&row.ID, &row.CIDR, &row.Mode, &row.Note, &row.CreatedAt)
	if err != nil {
		if isUniqueConstraint(err) {
			return nil, ErrNameTaken
		}
		return nil, fmt.Errorf("add ip rule: %w", err)
	}
	return row, nil
}

func (s *Store) ListIPRules() ([]*IPRule, error) {
	rows, err := s.db.Query(`SELECT id, cidr, mode, note, created_at FROM ip_rules ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list ip rules: %w", err)
	}
	defer rows.Close()
	var out []*IPRule
	for rows.Next() {
		r := &IPRule{}
		if err := rows.Scan(&r.ID, &r.CIDR, &r.Mode, &r.Note, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DeleteIPRule(id int64) error {
	res, err := s.db.Exec(`DELETE FROM ip_rules WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete ip rule: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrIPRuleNotFound
	}
	return nil
}

// ── Persisted Queries ─────────────────────────────────────────────────────────

type PersistedQuery struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Query     string `json:"query"`
	CreatedAt string `json:"created_at"`
}

func (s *Store) AddPersistedQuery(id, name, query string) (*PersistedQuery, error) {
	row := &PersistedQuery{}
	err := s.db.QueryRow(
		`INSERT INTO persisted_queries (id, name, query) VALUES (?, ?, ?)
		 RETURNING id, name, query, created_at`,
		id, name, query,
	).Scan(&row.ID, &row.Name, &row.Query, &row.CreatedAt)
	if err != nil {
		if isUniqueConstraint(err) {
			return nil, ErrPersistedQueryExists
		}
		return nil, fmt.Errorf("add persisted query: %w", err)
	}
	return row, nil
}

func (s *Store) GetPersistedQuery(id string) (*PersistedQuery, error) {
	row := &PersistedQuery{}
	err := s.db.QueryRow(
		`SELECT id, name, query, created_at FROM persisted_queries WHERE id = ?`, id,
	).Scan(&row.ID, &row.Name, &row.Query, &row.CreatedAt)
	if err != nil {
		return nil, ErrPersistedQueryNotFound
	}
	return row, nil
}

func (s *Store) ListPersistedQueries() ([]*PersistedQuery, error) {
	rows, err := s.db.Query(
		`SELECT id, name, query, created_at FROM persisted_queries ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list persisted queries: %w", err)
	}
	defer rows.Close()
	var out []*PersistedQuery
	for rows.Next() {
		r := &PersistedQuery{}
		if err := rows.Scan(&r.ID, &r.Name, &r.Query, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DeletePersistedQuery(id string) error {
	res, err := s.db.Exec(`DELETE FROM persisted_queries WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete persisted query: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrPersistedQueryNotFound
	}
	return nil
}

// ── Audit Log ─────────────────────────────────────────────────────────────────

type AuditEntry struct {
	ID        int64  `json:"id"`
	Admin     string `json:"admin"`
	Action    string `json:"action"`
	Detail    string `json:"detail"`
	IP        string `json:"ip"`
	CreatedAt string `json:"created_at"`
}

func (s *Store) AppendAuditLog(admin, action, detail, ip string) error {
	_, err := s.db.Exec(
		`INSERT INTO audit_log (admin, action, detail, ip) VALUES (?, ?, ?, ?)`,
		admin, action, detail, ip,
	)
	return err
}

func (s *Store) ListAuditLog(limit int) ([]*AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(
		`SELECT id, admin, action, detail, ip, created_at
		 FROM audit_log ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit log: %w", err)
	}
	defer rows.Close()
	var out []*AuditEntry
	for rows.Next() {
		e := &AuditEntry{}
		if err := rows.Scan(&e.ID, &e.Admin, &e.Action, &e.Detail, &e.IP, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ── Blocked Requests ──────────────────────────────────────────────────────────

type BlockedRequest struct {
	ID        int64  `json:"id"`
	Reason    string `json:"reason"`
	IP        string `json:"ip"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
}

func (s *Store) AppendBlockedRequest(reason, ip, path string) error {
	_, err := s.db.Exec(
		`INSERT INTO blocked_requests (reason, ip, path) VALUES (?, ?, ?)`,
		reason, ip, path,
	)
	return err
}

func (s *Store) ListBlockedRequests(limit int) ([]*BlockedRequest, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(
		`SELECT id, reason, ip, path, created_at
		 FROM blocked_requests ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list blocked requests: %w", err)
	}
	defer rows.Close()
	var out []*BlockedRequest
	for rows.Next() {
		r := &BlockedRequest{}
		if err := rows.Scan(&r.ID, &r.Reason, &r.IP, &r.Path, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

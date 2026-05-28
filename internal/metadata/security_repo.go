package metadata

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

var ErrSecretNotFound   = errors.New("jwt secret not found")
var ErrRouterKeyNotFound = errors.New("router key not found")
var ErrNameTaken         = errors.New("name already taken")

// ── JWT Secrets ───────────────────────────────────────────────────────────────

type JWTSecret struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Algorithm string `json:"algorithm"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
}

func (s *Store) AddJWTSecret(name, secret, algorithm string) (*JWTSecret, error) {
	if algorithm == "" {
		algorithm = "HS256"
	}
	row := &JWTSecret{}
	var active int
	err := s.db.QueryRow(
		`INSERT INTO jwt_secrets (name, secret, algorithm) VALUES (?, ?, ?)
		 RETURNING id, name, algorithm, active, created_at`,
		name, secret, algorithm,
	).Scan(&row.ID, &row.Name, &row.Algorithm, &active, &row.CreatedAt)
	if err != nil {
		if isUniqueConstraint(err) {
			return nil, ErrNameTaken
		}
		return nil, fmt.Errorf("add jwt secret: %w", err)
	}
	row.Active = active == 1
	return row, nil
}

func (s *Store) ListJWTSecrets() ([]*JWTSecret, error) {
	rows, err := s.db.Query(
		`SELECT id, name, algorithm, active, created_at FROM jwt_secrets ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list jwt secrets: %w", err)
	}
	defer rows.Close()

	var out []*JWTSecret
	for rows.Next() {
		r := &JWTSecret{}
		var active int
		if err := rows.Scan(&r.ID, &r.Name, &r.Algorithm, &active, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Active = active == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// ActiveJWTSecretValues returns the raw secret strings for all active records.
// Called by the JWT middleware on every request to validate incoming tokens.
func (s *Store) ActiveJWTSecretValues() ([]string, error) {
	rows, err := s.db.Query(`SELECT secret FROM jwt_secrets WHERE active = 1`)
	if err != nil {
		return nil, fmt.Errorf("load active jwt secrets: %w", err)
	}
	defer rows.Close()

	var secrets []string
	for rows.Next() {
		var sec string
		if err := rows.Scan(&sec); err != nil {
			return nil, err
		}
		secrets = append(secrets, sec)
	}
	return secrets, rows.Err()
}

func (s *Store) DeactivateJWTSecret(id int64) error {
	res, err := s.db.Exec(`UPDATE jwt_secrets SET active = 0 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deactivate jwt secret: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrSecretNotFound
	}
	return nil
}

// ── Router Keys ───────────────────────────────────────────────────────────────

type RouterKey struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
}

// GenerateRouterKey returns a cryptographically random 64-char hex string (256 bits).
// The caller must treat this as a secret — the raw value is never stored.
func GenerateRouterKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate router key: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func hashRouterKey(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}

func (s *Store) AddRouterKey(name, rawKey string) (*RouterKey, error) {
	row := &RouterKey{}
	var active int
	err := s.db.QueryRow(
		`INSERT INTO router_keys (name, key_hash) VALUES (?, ?)
		 RETURNING id, name, active, created_at`,
		name, hashRouterKey(rawKey),
	).Scan(&row.ID, &row.Name, &active, &row.CreatedAt)
	if err != nil {
		if isUniqueConstraint(err) {
			return nil, ErrNameTaken
		}
		return nil, fmt.Errorf("add router key: %w", err)
	}
	row.Active = active == 1
	return row, nil
}

func (s *Store) ListRouterKeys() ([]*RouterKey, error) {
	rows, err := s.db.Query(
		`SELECT id, name, active, created_at FROM router_keys ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list router keys: %w", err)
	}
	defer rows.Close()

	var out []*RouterKey
	for rows.Next() {
		r := &RouterKey{}
		var active int
		if err := rows.Scan(&r.ID, &r.Name, &active, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Active = active == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// ValidateRouterKey returns true if rawKey matches any active router key.
// SHA-256 comparison is fast and safe for 256-bit random keys.
func (s *Store) ValidateRouterKey(rawKey string) (bool, error) {
	hash := hashRouterKey(rawKey)
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM router_keys WHERE key_hash = ? AND active = 1`, hash,
	).Scan(&count)
	return count > 0, err
}

func (s *Store) HasActiveRouterKeys() (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM router_keys WHERE active = 1`).Scan(&count)
	return count > 0, err
}

func (s *Store) DeactivateRouterKey(id int64) error {
	res, err := s.db.Exec(`UPDATE router_keys SET active = 0 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deactivate router key: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrRouterKeyNotFound
	}
	return nil
}

package auth

import (
	"context"
	"database/sql"

	"github.com/deformal/kastql/internal/metadata"
)

// Checker implements planner.PermissionChecker against the metadata SQLite DB.
type Checker struct {
	store *metadata.Store
}

// NewChecker creates a Checker backed by the given metadata store.
func NewChecker(store *metadata.Store) *Checker {
	return &Checker{store: store}
}

// CanAccess returns true if the given role may access typeName.fieldName.
//
// Rule precedence (most specific wins):
//  1. role + type + field  → exact rule
//  2. role + type          → applies to all fields of the type
//  3. No rule found        → public role is allowed; all others are denied
func (c *Checker) CanAccess(ctx context.Context, role, typeName, fieldName string) (bool, error) {
	if role == "" {
		role = "public"
	}

	// Skip permission checks for introspection fields — always allowed.
	if fieldName == "__schema" || fieldName == "__type" || fieldName == "__typename" {
		return true, nil
	}

	var allow int
	err := c.store.DB().QueryRowContext(ctx, `
		SELECT allow FROM permissions
		WHERE role = ?
		  AND (type_name = ? OR type_name = '')
		  AND (field_name = ? OR field_name = '')
		ORDER BY
		  CASE WHEN type_name  = ? THEN 0 ELSE 1 END,
		  CASE WHEN field_name = ? THEN 0 ELSE 1 END
		LIMIT 1
	`, role, typeName, fieldName, typeName, fieldName).Scan(&allow)

	if err == sql.ErrNoRows {
		// No rule found: public role has default access; others are denied.
		return role == "public", nil
	}
	if err != nil {
		return false, err
	}
	return allow == 1, nil
}

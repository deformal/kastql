package auth

import (
	"context"
	"testing"

	"github.com/deformal/kastql/internal/metadata"
)

func TestCheckerNoRules(t *testing.T) {
	store, err := metadata.Open(t.TempDir()+"/meta.db", "metadata")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	c := NewChecker(store)

	// public role → allow by default (no rules)
	ok, err := c.CanAccess(context.Background(), "public", "User", "name")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected public role to be allowed with no rules")
	}

	// non-public role → deny by default (no rules)
	ok, err = c.CanAccess(context.Background(), "admin", "User", "name")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected non-public role to be denied with no rules")
	}
}

func TestCheckerExplicitAllow(t *testing.T) {
	store, err := metadata.Open(t.TempDir()+"/meta.db", "metadata")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Insert an explicit allow for admin on User.name
	_, err = store.DB().Exec(`
		INSERT INTO permissions (role, service, type_name, field_name, allow)
		VALUES ('admin', '', 'User', 'name', 1)
	`)
	if err != nil {
		t.Fatal(err)
	}

	c := NewChecker(store)

	ok, err := c.CanAccess(context.Background(), "admin", "User", "name")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected admin to be allowed for User.name after explicit rule")
	}
}

func TestCheckerExplicitDenyPublic(t *testing.T) {
	store, err := metadata.Open(t.TempDir()+"/meta.db", "metadata")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Deny public access to a sensitive field
	_, err = store.DB().Exec(`
		INSERT INTO permissions (role, service, type_name, field_name, allow)
		VALUES ('public', '', 'User', 'password', 0)
	`)
	if err != nil {
		t.Fatal(err)
	}

	c := NewChecker(store)

	ok, err := c.CanAccess(context.Background(), "public", "User", "password")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected public to be denied for User.password after explicit deny rule")
	}
}

func TestCheckerIntrospectionAlwaysAllowed(t *testing.T) {
	store, err := metadata.Open(t.TempDir()+"/meta.db", "metadata")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	c := NewChecker(store)

	for _, field := range []string{"__schema", "__type", "__typename"} {
		ok, err := c.CanAccess(context.Background(), "restricted", "Query", field)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Errorf("introspection field %q should always be allowed", field)
		}
	}
}

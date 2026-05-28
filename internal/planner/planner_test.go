package planner

import (
	"context"
	"testing"

	"github.com/deformal/kastql/internal/metadata"
	"github.com/deformal/kastql/internal/registry"
	"go.uber.org/zap"
)

var usersSDL = `
type Query {
  users: [User!]!
  user(id: ID!): User
}
type User {
  id: ID!
  name: String!
  email: String!
}
`

var ordersSDL = `
type Query {
  orders: [Order!]!
  order(id: ID!): Order
}
type Order {
  id: ID!
  total: Float!
}
`

var federationUsersSDL = `
type Query {
  user(id: ID!): User
}
type User @key(fields: "id") {
  id: ID!
  name: String!
  email: String!
}
`

var federationOrdersSDL = `
type Query {
  orders: [Order!]!
}
type Order @key(fields: "id") {
  id: ID!
  total: Float!
  user: User!
}
type User @key(fields: "id") {
  id: ID! @external
}
`

func makeEntry(name, url, serviceType, sdl string) *registry.ServiceEntry {
	return &registry.ServiceEntry{
		Service: metadata.Service{
			Name:    name,
			URL:     url,
			Type:    metadata.ServiceType(serviceType),
			Enabled: true,
		},
		SDL: sdl,
	}
}

func TestMergeStitching(t *testing.T) {
	entries := []*registry.ServiceEntry{
		makeEntry("users-svc", "http://users/graphql", "stitching", usersSDL),
		makeEntry("orders-svc", "http://orders/graphql", "stitching", ordersSDL),
	}

	merged, err := Merge(entries)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if merged.QueryOwnership["users"] != "users-svc" {
		t.Errorf("expected users owned by users-svc, got %q", merged.QueryOwnership["users"])
	}
	if merged.QueryOwnership["orders"] != "orders-svc" {
		t.Errorf("expected orders owned by orders-svc, got %q", merged.QueryOwnership["orders"])
	}
	if merged.Schema == nil {
		t.Fatal("merged schema is nil")
	}
}

func TestMergeFederation(t *testing.T) {
	entries := []*registry.ServiceEntry{
		makeEntry("users-svc", "http://users/graphql", "federation", federationUsersSDL),
		makeEntry("orders-svc", "http://orders/graphql", "federation", federationOrdersSDL),
	}

	merged, err := Merge(entries)
	if err != nil {
		t.Fatalf("Merge federation: %v", err)
	}

	if merged.TypeOwnership["User"] != "users-svc" {
		t.Errorf("expected User owned by users-svc, got %q", merged.TypeOwnership["User"])
	}
	keys := merged.EntityKeys["User"]["users-svc"]
	if len(keys) == 0 || keys[0] != "id" {
		t.Errorf("expected User @key=[id] for users-svc, got %v", keys)
	}
}

func TestPlanSingleService(t *testing.T) {
	entries := []*registry.ServiceEntry{
		makeEntry("users-svc", "http://users/graphql", "stitching", usersSDL),
	}

	store, _ := metadata.Open(t.TempDir()+"/meta.db", "metadata")
	defer store.Close()

	p := New(store, zap.NewNop())
	if err := p.Update(entries); err != nil {
		t.Fatalf("Update: %v", err)
	}

	plan, err := p.Plan(context.Background(), `{ users { id name } }`, nil, "public")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}
	s := plan.Steps[0]
	if s.ServiceName != "users-svc" {
		t.Errorf("expected users-svc, got %q", s.ServiceName)
	}
	if s.Meta.Kind != StepKindRoot {
		t.Errorf("expected root step, got %q", s.Meta.Kind)
	}
}

func TestPlanMultiService(t *testing.T) {
	entries := []*registry.ServiceEntry{
		makeEntry("users-svc", "http://users/graphql", "stitching", usersSDL),
		makeEntry("orders-svc", "http://orders/graphql", "stitching", ordersSDL),
	}

	store, _ := metadata.Open(t.TempDir()+"/meta.db", "metadata")
	defer store.Close()

	p := New(store, zap.NewNop())
	if err := p.Update(entries); err != nil {
		t.Fatalf("Update: %v", err)
	}

	plan, err := p.Plan(context.Background(), `{ users { id name } orders { id total } }`, nil, "public")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(plan.Steps))
	}

	services := map[string]bool{}
	for _, s := range plan.Steps {
		services[s.ServiceName] = true
	}
	if !services["users-svc"] || !services["orders-svc"] {
		t.Errorf("expected both services in plan, got %v", services)
	}
}

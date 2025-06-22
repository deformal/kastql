package introspection

import (
	"testing"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine()
	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}
	if engine.client == nil {
		t.Fatal("Engine client is nil")
	}
}

func TestIntrospectionQuery(t *testing.T) {
	// Test that the introspection query is valid
	if IntrospectionQuery == "" {
		t.Fatal("IntrospectionQuery is empty")
	}

	// Test that it contains expected GraphQL keywords
	expectedKeywords := []string{"__schema", "queryType", "mutationType", "subscriptionType", "types"}
	for _, keyword := range expectedKeywords {
		if !contains(IntrospectionQuery, keyword) {
			t.Errorf("IntrospectionQuery missing expected keyword: %s", keyword)
		}
	}
}

func TestIntrospectionRequest(t *testing.T) {
	request := IntrospectionRequest{
		Query: IntrospectionQuery,
	}

	if request.Query != IntrospectionQuery {
		t.Errorf("Expected query to be %s, got %s", IntrospectionQuery, request.Query)
	}
}

func TestSchemaStructure(t *testing.T) {
	// Test that Schema struct can be created
	schema := &Schema{
		QueryType: &TypeRef{
			Kind: "OBJECT",
			Name: "Query",
		},
		Types: []Type{
			{
				Kind: "OBJECT",
				Name: "User",
				Fields: []Field{
					{
						Name: "id",
						Type: TypeRef{
							Kind: "NON_NULL",
							OfType: &TypeRef{
								Kind: "SCALAR",
								Name: "ID",
							},
						},
					},
				},
			},
		},
	}

	if schema.QueryType == nil {
		t.Fatal("Schema QueryType is nil")
	}
	if schema.QueryType.Name != "Query" {
		t.Errorf("Expected QueryType name to be 'Query', got %s", schema.QueryType.Name)
	}
	if len(schema.Types) != 1 {
		t.Errorf("Expected 1 type, got %d", len(schema.Types))
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			contains(s[1:len(s)-1], substr)))
}

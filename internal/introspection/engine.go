package introspection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// IntrospectionQuery is the standard GraphQL introspection query
const IntrospectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
    directives {
      name
      description
      locations
      args {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type { ...TypeRef }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
      }
    }
  }
}
`

// IntrospectionRequest represents the request structure for introspection
type IntrospectionRequest struct {
	Query string `json:"query"`
}

// IntrospectionResponse represents the response from introspection
type IntrospectionResponse struct {
	Data struct {
		Schema Schema `json:"__schema"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []ErrorLocation        `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// ErrorLocation represents the location of an error
type ErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Schema represents the GraphQL schema structure
type Schema struct {
	QueryType        *TypeRef    `json:"queryType"`
	MutationType     *TypeRef    `json:"mutationType"`
	SubscriptionType *TypeRef    `json:"subscriptionType"`
	Types            []Type      `json:"types"`
	Directives       []Directive `json:"directives"`
}

// TypeRef represents a reference to a GraphQL type
type TypeRef struct {
	Kind   string   `json:"kind"`
	Name   string   `json:"name"`
	OfType *TypeRef `json:"ofType"`
}

// Type represents a GraphQL type
type Type struct {
	Kind          string       `json:"kind"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	Fields        []Field      `json:"fields,omitempty"`
	InputFields   []InputValue `json:"inputFields,omitempty"`
	Interfaces    []TypeRef    `json:"interfaces,omitempty"`
	EnumValues    []EnumValue  `json:"enumValues,omitempty"`
	PossibleTypes []TypeRef    `json:"possibleTypes,omitempty"`
}

// Field represents a GraphQL field
type Field struct {
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	Args              []InputValue `json:"args"`
	Type              TypeRef      `json:"type"`
	IsDeprecated      bool         `json:"isDeprecated"`
	DeprecationReason string       `json:"deprecationReason"`
}

// InputValue represents a GraphQL input value
type InputValue struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Type         TypeRef `json:"type"`
	DefaultValue string  `json:"defaultValue"`
}

// EnumValue represents a GraphQL enum value
type EnumValue struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	IsDeprecated      bool   `json:"isDeprecated"`
	DeprecationReason string `json:"deprecationReason"`
}

// Directive represents a GraphQL directive
type Directive struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Locations   []string     `json:"locations"`
	Args        []InputValue `json:"args"`
}

// Engine handles GraphQL introspection
type Engine struct {
	client *http.Client
}

// NewEngine creates a new introspection engine
func NewEngine() *Engine {
	return &Engine{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Introspect performs introspection on a GraphQL server
func (e *Engine) Introspect(endpoint string) (*Schema, error) {
	request := IntrospectionRequest{
		Query: IntrospectionQuery,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal introspection request: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform introspection request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("introspection failed with status %d: %s", resp.StatusCode, string(body))
	}

	var introspectionResp IntrospectionResponse
	if err := json.Unmarshal(body, &introspectionResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal introspection response: %w", err)
	}

	if len(introspectionResp.Errors) > 0 {
		return nil, fmt.Errorf("introspection errors: %v", introspectionResp.Errors)
	}

	return &introspectionResp.Data.Schema, nil
}

// ValidateEndpoint checks if a GraphQL endpoint is accessible
func (e *Engine) ValidateEndpoint(endpoint string) error {
	// Try a simple introspection query to validate the endpoint
	request := IntrospectionRequest{
		Query: `query { __schema { queryType { name } } }`,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal validation request: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate endpoint: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body to check for GraphQL errors
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read validation response: %w", err)
	}

	// Check if we got a valid GraphQL response (even if it has errors)
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("endpoint returned invalid JSON: %w", err)
	}

	// If we got a response with data or errors, it's a valid GraphQL endpoint
	if _, hasData := response["data"]; hasData {
		return nil
	}
	if _, hasErrors := response["errors"]; hasErrors {
		return nil
	}

	// If we got a 200 response but no GraphQL structure, it might still be valid
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf("endpoint validation failed with status %d: %s", resp.StatusCode, string(body))
}

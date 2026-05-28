package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const federationSDLQuery = `{ _service { sdl } }`

const introspectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types { ...FullType }
    directives {
      name description locations
      args { ...InputValue }
    }
  }
}
fragment FullType on __Type {
  kind name description
  fields(includeDeprecated: true) {
    name description
    args { ...InputValue }
    type { ...TypeRef }
    isDeprecated deprecationReason
  }
  inputFields { ...InputValue }
  interfaces { ...TypeRef }
  enumValues(includeDeprecated: true) {
    name description isDeprecated deprecationReason
  }
  possibleTypes { ...TypeRef }
}
fragment InputValue on __InputValue {
  name description
  type { ...TypeRef }
  defaultValue
}
fragment TypeRef on __Type {
  kind name
  ofType { kind name ofType { kind name ofType { kind name ofType {
    kind name ofType { kind name ofType { kind name ofType { kind name } } }
  } } } }
}`

type gqlRequest struct {
	Query string `json:"query"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func fetchSDL(ctx context.Context, url string, headers map[string]string, serviceType string) (string, error) {
	if serviceType == "federation" {
		return fetchFederationSDL(ctx, url, headers)
	}
	return fetchIntrospectionSDL(ctx, url, headers)
}

func fetchFederationSDL(ctx context.Context, url string, headers map[string]string) (string, error) {
	var result struct {
		Service struct {
			SDL string `json:"sdl"`
		} `json:"_service"`
	}
	if err := doGQLRequest(ctx, url, headers, federationSDLQuery, &result); err != nil {
		return "", fmt.Errorf("federation sdl query: %w", err)
	}
	if result.Service.SDL == "" {
		return "", fmt.Errorf("empty sdl from federation service at %s", url)
	}
	return result.Service.SDL, nil
}

func fetchIntrospectionSDL(ctx context.Context, url string, headers map[string]string) (string, error) {
	var result struct {
		Schema introspectionSchema `json:"__schema"`
	}
	if err := doGQLRequest(ctx, url, headers, introspectionQuery, &result); err != nil {
		return "", fmt.Errorf("introspection query: %w", err)
	}
	return schemaToSDL(&result.Schema), nil
}

func doGQLRequest(ctx context.Context, url string, headers map[string]string, query string, out any) error {
	body, _ := json.Marshal(gqlRequest{Query: query})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var gqlResp gqlResponse
	if err := json.Unmarshal(raw, &gqlResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}
	return json.Unmarshal(gqlResp.Data, out)
}

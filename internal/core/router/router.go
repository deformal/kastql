package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/deformal/kastql/internal/storage"
)

type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

type GraphQLResponse struct {
	Data       interface{}            `json:"data,omitempty"`
	Errors     []GraphQLError         `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []ErrorLocation        `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

type ErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Router struct {
	registry *storage.Registry
	client   *http.Client
}

func NewRouter(registry *storage.Registry) *Router {
	return &Router{
		registry: registry,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *Router) RouteRequest(request *GraphQLRequest) (*GraphQLResponse, error) {
	_, fieldName, err := r.parseQuery(request.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	server, err := r.findServerForField(fieldName)
	if err != nil {
		return nil, fmt.Errorf("no server found for field %s: %w", fieldName, err)
	}

	return r.forwardRequest(server, request)
}

func (r *Router) parseQuery(query string) (string, string, error) {
	query = strings.TrimSpace(query)

	lines := strings.Split(query, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			cleanLines = append(cleanLines, line)
		}
	}
	query = strings.Join(cleanLines, " ")

	var operationType string
	if strings.HasPrefix(strings.ToLower(query), "query") {
		operationType = "query"
	} else if strings.HasPrefix(strings.ToLower(query), "mutation") {
		operationType = "mutation"
	} else if strings.HasPrefix(strings.ToLower(query), "subscription") {
		operationType = "subscription"
	} else {
		operationType = "query"
	}

	parts := strings.Fields(query)
	for i, part := range parts {
		if part == operationType || part == "{" {
			continue
		}
		if strings.Contains(part, "(") {
			continue
		}
		if i+1 < len(parts) && parts[i+1] != "{" && !strings.HasPrefix(parts[i+1], "(") {
			fieldName := strings.TrimSpace(parts[i+1])
			fieldName = strings.TrimRight(fieldName, "{(")
			return operationType, fieldName, nil
		}
	}

	return operationType, "", fmt.Errorf("could not extract field name from query")
}

func (r *Router) findServerForField(fieldName string) (*storage.ServerInfo, error) {
	server, err := r.registry.FindServerByField(fieldName)
	if err == nil {
		return server, nil
	}

	activeServers := r.registry.GetActiveServers()
	for _, server := range activeServers {
		if server.Schema == nil {
			continue
		}

		for _, t := range server.Schema.Types {
			if t.Name == fieldName {
				return server, nil
			}
			for _, field := range t.Fields {
				if field.Name == fieldName {
					return server, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no server found for field %s", fieldName)
}

func (r *Router) forwardRequest(server *storage.ServerInfo, request *GraphQLRequest) (*GraphQLResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", server.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to forward request to %s: %w", server.Endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var response GraphQLResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func (r *Router) BatchRouteRequest(requests []*GraphQLRequest) ([]*GraphQLResponse, error) {
	var responses []*GraphQLResponse

	for _, request := range requests {
		response, err := r.RouteRequest(request)
		if err != nil {
			return nil, fmt.Errorf("failed to route request: %w", err)
		}
		responses = append(responses, response)
	}

	return responses, nil
}

func (r *Router) GetAvailableFields() map[string][]string {
	fields := make(map[string][]string)
	activeServers := r.registry.GetActiveServers()

	for _, server := range activeServers {
		if server.Schema == nil {
			continue
		}

		var serverFields []string

		if server.Schema.QueryType != nil {
			serverFields = append(serverFields, server.Schema.QueryType.Name)
		}

		if server.Schema.MutationType != nil {
			serverFields = append(serverFields, server.Schema.MutationType.Name)
		}

		if server.Schema.SubscriptionType != nil {
			serverFields = append(serverFields, server.Schema.SubscriptionType.Name)
		}

		for _, t := range server.Schema.Types {
			serverFields = append(serverFields, t.Name)
		}

		fields[server.Name] = serverFields
	}

	return fields
}

func (r *Router) ValidateQuery(query string) (bool, []string, error) {
	_, fieldName, err := r.parseQuery(query)
	if err != nil {
		return false, nil, fmt.Errorf("failed to parse query: %w", err)
	}

	availableFields := r.GetAvailableFields()
	var availableServers []string

	for serverName, fields := range availableFields {
		for _, field := range fields {
			if field == fieldName {
				availableServers = append(availableServers, serverName)
				break
			}
		}
	}

	if len(availableServers) == 0 {
		return false, nil, nil
	}

	return true, availableServers, nil
}

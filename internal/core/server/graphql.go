package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/deformal/kastql/internal/core/router"
	"github.com/deformal/kastql/internal/storage"
)

type GraphQLHandler struct {
	router *router.Router
}

func NewGraphQLHandler(registry *storage.Registry) *GraphQLHandler {
	return &GraphQLHandler{
		router: router.NewRouter(registry),
	}
}

func (h *GraphQLHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var graphqlRequest router.GraphQLRequest
	if err := json.Unmarshal(body, &graphqlRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if graphqlRequest.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	response, err := h.router.RouteRequest(&graphqlRequest)
	if err != nil {
		errorResponse := router.GraphQLResponse{
			Errors: []router.GraphQLError{
				{
					Message: err.Error(),
				},
			},
		}
		h.writeJSONResponse(w, errorResponse)
		return
	}

	h.writeJSONResponse(w, response)
}

func (h *GraphQLHandler) writeJSONResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
}

type HealthHandler struct {
	registry *storage.Registry
}

func NewHealthHandler(registry *storage.Registry) *HealthHandler {
	return &HealthHandler{
		registry: registry,
	}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status": "healthy",
		"servers": map[string]interface{}{
			"total":  h.registry.GetServerCount(),
			"active": h.registry.GetActiveServerCount(),
		},
	}

	jsonData, err := json.Marshal(health)
	if err != nil {
		http.Error(w, "Failed to marshal health response", http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
}

type SchemaHandler struct {
	registry *storage.Registry
}

func NewSchemaHandler(registry *storage.Registry) *SchemaHandler {
	return &SchemaHandler{
		registry: registry,
	}
}

func (h *SchemaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	servers := h.registry.GetActiveServers()

	schemaInfo := map[string]interface{}{
		"servers":       servers,
		"total_servers": len(servers),
	}

	jsonData, err := json.Marshal(schemaInfo)
	if err != nil {
		http.Error(w, "Failed to marshal schema response", http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
}

func SetupGraphQLServer(registry *storage.Registry) *http.ServeMux {
	mux := http.NewServeMux()

	graphqlHandler := NewGraphQLHandler(registry)
	mux.HandleFunc("/graphql", graphqlHandler.ServeHTTP)

	healthHandler := NewHealthHandler(registry)
	mux.HandleFunc("/health", healthHandler.ServeHTTP)

	schemaHandler := NewSchemaHandler(registry)
	mux.HandleFunc("/schema", schemaHandler.ServeHTTP)

	return mux
}

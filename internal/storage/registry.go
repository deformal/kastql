package storage

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/deformal/kastql/internal/introspection"
)

// ServerInfo contains information about a GraphQL server
type ServerInfo struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Endpoint    string                `json:"endpoint"`
	Description string                `json:"description"`
	Schema      *introspection.Schema `json:"schema"`
	AddedAt     time.Time             `json:"added_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	IsActive    bool                  `json:"is_active"`
}

// Registry manages GraphQL server registrations and schemas
type Registry struct {
	servers map[string]*ServerInfo
	mutex   sync.RWMutex
}

// NewRegistry creates a new schema registry
func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]*ServerInfo),
	}
}

// RegisterServer adds a new GraphQL server to the registry
func (r *Registry) RegisterServer(id, name, endpoint, description string, schema *introspection.Schema) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	server := &ServerInfo{
		ID:          id,
		Name:        name,
		Endpoint:    endpoint,
		Description: description,
		Schema:      schema,
		AddedAt:     now,
		UpdatedAt:   now,
		IsActive:    true,
	}

	r.servers[id] = server
	return nil
}

// UpdateServer updates an existing server's information
func (r *Registry) UpdateServer(id string, updates map[string]interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	server, exists := r.servers[id]
	if !exists {
		return fmt.Errorf("server with ID %s not found", id)
	}

	// Update fields based on the updates map
	if name, ok := updates["name"].(string); ok {
		server.Name = name
	}
	if endpoint, ok := updates["endpoint"].(string); ok {
		server.Endpoint = endpoint
	}
	if description, ok := updates["description"].(string); ok {
		server.Description = description
	}
	if schema, ok := updates["schema"].(*introspection.Schema); ok {
		server.Schema = schema
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		server.IsActive = isActive
	}

	server.UpdatedAt = time.Now()
	return nil
}

// GetServer retrieves a server by ID
func (r *Registry) GetServer(id string) (*ServerInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	server, exists := r.servers[id]
	if !exists {
		return nil, fmt.Errorf("server with ID %s not found", id)
	}

	return server, nil
}

// GetAllServers returns all registered servers
func (r *Registry) GetAllServers() []*ServerInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	servers := make([]*ServerInfo, 0, len(r.servers))
	for _, server := range r.servers {
		servers = append(servers, server)
	}

	return servers
}

// GetActiveServers returns only active servers
func (r *Registry) GetActiveServers() []*ServerInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var activeServers []*ServerInfo
	for _, server := range r.servers {
		if server.IsActive {
			activeServers = append(activeServers, server)
		}
	}

	return activeServers
}

// RemoveServer removes a server from the registry
func (r *Registry) RemoveServer(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.servers[id]; !exists {
		return fmt.Errorf("server with ID %s not found", id)
	}

	delete(r.servers, id)
	return nil
}

// DeactivateServer marks a server as inactive
func (r *Registry) DeactivateServer(id string) error {
	return r.UpdateServer(id, map[string]interface{}{
		"is_active": false,
	})
}

// ActivateServer marks a server as active
func (r *Registry) ActivateServer(id string) error {
	return r.UpdateServer(id, map[string]interface{}{
		"is_active": true,
	})
}

// FindServerByField searches for a server by field name
func (r *Registry) FindServerByField(fieldName string) (*ServerInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, server := range r.servers {
		if server.Schema == nil {
			continue
		}

		// Check query type fields
		if server.Schema.QueryType != nil {
			if server.Schema.QueryType.Name == fieldName {
				return server, nil
			}
		}

		// Check mutation type fields
		if server.Schema.MutationType != nil {
			if server.Schema.MutationType.Name == fieldName {
				return server, nil
			}
		}

		// Check subscription type fields
		if server.Schema.SubscriptionType != nil {
			if server.Schema.SubscriptionType.Name == fieldName {
				return server, nil
			}
		}

		// Check all types for the field
		for _, t := range server.Schema.Types {
			if t.Name == fieldName {
				return server, nil
			}
		}
	}

	return nil, fmt.Errorf("no server found with field %s", fieldName)
}

// ExportRegistry exports the registry to JSON
func (r *Registry) ExportRegistry() ([]byte, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return json.MarshalIndent(r.servers, "", "  ")
}

// ImportRegistry imports servers from JSON
func (r *Registry) ImportRegistry(data []byte) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var servers map[string]*ServerInfo
	if err := json.Unmarshal(data, &servers); err != nil {
		return fmt.Errorf("failed to unmarshal registry data: %w", err)
	}

	r.servers = servers
	return nil
}

// GetServerCount returns the total number of registered servers
func (r *Registry) GetServerCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.servers)
}

// GetActiveServerCount returns the number of active servers
func (r *Registry) GetActiveServerCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	count := 0
	for _, server := range r.servers {
		if server.IsActive {
			count++
		}
	}

	return count
}

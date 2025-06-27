package storage

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/deformal/kastql/internal/introspection"
)

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

type Registry struct {
	servers map[string]*ServerInfo
	mutex   sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]*ServerInfo),
	}
}

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

func (r *Registry) UpdateServer(id string, updates map[string]interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	server, exists := r.servers[id]
	if !exists {
		return fmt.Errorf("server with ID %s not found", id)
	}

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

func (r *Registry) GetServer(id string) (*ServerInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	server, exists := r.servers[id]
	if !exists {
		return nil, fmt.Errorf("server with ID %s not found", id)
	}

	return server, nil
}

func (r *Registry) GetAllServers() []*ServerInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	servers := make([]*ServerInfo, 0, len(r.servers))
	for _, server := range r.servers {
		servers = append(servers, server)
	}

	return servers
}

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

func (r *Registry) RemoveServer(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.servers[id]; !exists {
		return fmt.Errorf("server with ID %s not found", id)
	}

	delete(r.servers, id)
	return nil
}

func (r *Registry) DeactivateServer(id string) error {
	return r.UpdateServer(id, map[string]interface{}{
		"is_active": false,
	})
}

func (r *Registry) ActivateServer(id string) error {
	return r.UpdateServer(id, map[string]interface{}{
		"is_active": true,
	})
}

func (r *Registry) FindServerByField(fieldName string) (*ServerInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, server := range r.servers {
		if server.Schema == nil {
			continue
		}

		if server.Schema.QueryType != nil {
			if server.Schema.QueryType.Name == fieldName {
				return server, nil
			}
		}

		if server.Schema.MutationType != nil {
			if server.Schema.MutationType.Name == fieldName {
				return server, nil
			}
		}

		if server.Schema.SubscriptionType != nil {
			if server.Schema.SubscriptionType.Name == fieldName {
				return server, nil
			}
		}

		for _, t := range server.Schema.Types {
			if t.Name == fieldName {
				return server, nil
			}
		}
	}

	return nil, fmt.Errorf("no server found with field %s", fieldName)
}

func (r *Registry) ExportRegistry() ([]byte, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return json.MarshalIndent(r.servers, "", "  ")
}

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

func (r *Registry) GetServerCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.servers)
}

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

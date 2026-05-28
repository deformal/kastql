package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/deformal/kastql/internal/config"
	"github.com/deformal/kastql/internal/health"
	"github.com/deformal/kastql/internal/metadata"
)

// ServiceEntry is an in-memory enriched view of a registered service.
type ServiceEntry struct {
	metadata.Service
	SDL       string
	UpdatedAt time.Time
}

// Registry manages the set of active upstream GraphQL services.
type Registry struct {
	mu       sync.RWMutex
	services map[string]*ServiceEntry
	store    *metadata.Store
	log      *zap.Logger
}

func New(store *metadata.Store, log *zap.Logger) *Registry {
	return &Registry{
		services: make(map[string]*ServiceEntry),
		store:    store,
		log:      log,
	}
}

// Bootstrap loads services from config and persists them, then loads all
// enabled services from the DB and introspects their schemas.
func (r *Registry) Bootstrap(ctx context.Context, cfgServices []config.Service) error {
	for _, cs := range cfgServices {
		svc := &metadata.Service{
			Name:    cs.Name,
			URL:     cs.URL,
			Type:    metadata.ServiceType(cs.Type),
			Headers: headersToJSON(cs.Headers),
			Enabled: cs.Enabled,
		}
		if err := r.store.UpsertService(svc); err != nil {
			return fmt.Errorf("bootstrap upsert %s: %w", cs.Name, err)
		}
	}
	return r.loadAll(ctx)
}

// loadAll reads all enabled services from DB and introspects them.
func (r *Registry) loadAll(ctx context.Context) error {
	svcs, err := r.store.ListServices()
	if err != nil {
		return err
	}
	for _, svc := range svcs {
		if !svc.Enabled {
			continue
		}
		if err := r.introspectAndCache(ctx, svc); err != nil {
			// Non-fatal: log and continue so one bad upstream doesn't kill startup.
			r.log.Warn("failed to introspect service",
				zap.String("service", svc.Name),
				zap.Error(err),
			)
			continue
		}
	}
	return nil
}

// Add registers a new service, introspects it, and persists everything.
func (r *Registry) Add(ctx context.Context, svc *metadata.Service) error {
	if err := r.store.UpsertService(svc); err != nil {
		return err
	}
	return r.introspectAndCache(ctx, svc)
}

// Remove deregisters a service.
func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	delete(r.services, name)
	r.mu.Unlock()

	if err := r.store.DeleteSchemaCache(name); err != nil {
		return err
	}
	return r.store.DeleteService(name)
}

// Reload re-introspects a single service.
func (r *Registry) Reload(ctx context.Context, name string) error {
	svc, err := r.store.GetService(name)
	if err != nil {
		return err
	}
	if svc == nil {
		return fmt.Errorf("service %q not found", name)
	}
	return r.introspectAndCache(ctx, svc)
}

// ReloadAll re-introspects every enabled service.
func (r *Registry) ReloadAll(ctx context.Context) error {
	return r.loadAll(ctx)
}

// Get returns a service entry by name (nil if not found).
func (r *Registry) Get(name string) *ServiceEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.services[name]
}

// List returns all loaded service entries.
func (r *Registry) List() []*ServiceEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ServiceEntry, 0, len(r.services))
	for _, e := range r.services {
		out = append(out, e)
	}
	return out
}

// HealthTargets returns the minimal info needed by the health monitor to probe each service.
// Implements health.ServiceProvider.
func (r *Registry) HealthTargets() []health.ServiceTarget {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]health.ServiceTarget, 0, len(r.services))
	for _, e := range r.services {
		if !e.Enabled {
			continue
		}
		headers, _ := jsonToHeaders(e.Headers)
		out = append(out, health.ServiceTarget{
			Name:    e.Name,
			URL:     e.URL,
			Headers: headers,
		})
	}
	return out
}

// introspectAndCache fetches the SDL, stores it in the DB, and updates the
// in-memory registry entry.
func (r *Registry) introspectAndCache(ctx context.Context, svc *metadata.Service) error {
	headers, err := jsonToHeaders(svc.Headers)
	if err != nil {
		return fmt.Errorf("parse headers for %s: %w", svc.Name, err)
	}

	r.log.Info("introspecting service", zap.String("service", svc.Name), zap.String("url", svc.URL))

	sdl, err := fetchSDL(ctx, svc.URL, headers, string(svc.Type))
	if err != nil {
		return fmt.Errorf("introspect %s: %w", svc.Name, err)
	}

	if err := r.store.UpsertSchemaCache(svc.Name, sdl); err != nil {
		return fmt.Errorf("cache sdl for %s: %w", svc.Name, err)
	}

	r.mu.Lock()
	r.services[svc.Name] = &ServiceEntry{
		Service:   *svc,
		SDL:       sdl,
		UpdatedAt: time.Now(),
	}
	r.mu.Unlock()

	r.log.Info("service ready", zap.String("service", svc.Name), zap.Int("sdl_bytes", len(sdl)))
	return nil
}

func headersToJSON(h map[string]string) string {
	if len(h) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(h)
	return string(b)
}

func jsonToHeaders(raw string) (map[string]string, error) {
	if raw == "" || raw == "{}" {
		return nil, nil
	}
	var h map[string]string
	if err := json.Unmarshal([]byte(raw), &h); err != nil {
		return nil, err
	}
	return h, nil
}

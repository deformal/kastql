// Package health provides upstream health monitoring and circuit breaking for kastql.
//
// Circuit states:
//   CLOSED    — service is healthy; all requests pass through.
//   OPEN      — service is failing; requests are rejected immediately (fail-fast).
//   HALF_OPEN — cooldown elapsed; health loop is probing; requests still rejected
//               until probe succeeds and circuit returns to CLOSED.
package health

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CircuitState is the circuit-breaker state for one upstream service.
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half_open"
)

// ServiceStatus is a point-in-time snapshot of one upstream's health.
type ServiceStatus struct {
	Name           string       `json:"name"`
	URL            string       `json:"url"`
	Circuit        CircuitState `json:"circuit"`
	Healthy        bool         `json:"healthy"`
	ConsecFailures int          `json:"consec_failures"`
	LatencyMs      int64        `json:"latency_ms"`
	CheckedAt      time.Time    `json:"checked_at"`
	HealthySince   time.Time    `json:"healthy_since,omitempty"`
	LastError      string       `json:"last_error,omitempty"`
}

// ServiceTarget is the minimal info the monitor needs to probe an upstream.
type ServiceTarget struct {
	Name    string
	URL     string
	Headers map[string]string
}

// ServiceProvider is implemented by the registry.
type ServiceProvider interface {
	HealthTargets() []ServiceTarget
}

// Reloader is called when a service transitions from OPEN → CLOSED so that
// the registry can re-introspect and update the planner's merged schema.
type Reloader interface {
	Reload(ctx context.Context, name string) error
}

// Config tunes the health monitor behaviour.
// All values default to sane production values if zero.
type Config struct {
	CheckInterval    time.Duration
	CheckTimeout     time.Duration
	FailThreshold    int
	RecoveryCooldown time.Duration
}

func (c *Config) withDefaults() Config {
	out := *c
	if out.CheckInterval <= 0 {
		out.CheckInterval = 30 * time.Second
	}
	if out.CheckTimeout <= 0 {
		out.CheckTimeout = 5 * time.Second
	}
	if out.FailThreshold <= 0 {
		out.FailThreshold = 3
	}
	if out.RecoveryCooldown <= 0 {
		out.RecoveryCooldown = 30 * time.Second
	}
	return out
}

// Monitor runs a background health-check loop and maintains per-service circuit state.
type Monitor struct {
	mu       sync.RWMutex
	services map[string]*serviceState

	cfg      Config
	provider ServiceProvider
	reloader Reloader
	log      *zap.Logger
	client   *http.Client
}

type serviceState struct {
	ServiceStatus
	openedAt time.Time // when circuit last transitioned to OPEN
}

// New creates a Monitor. Call Start to begin the background loop.
func New(cfg Config, provider ServiceProvider, reloader Reloader, log *zap.Logger) *Monitor {
	c := cfg.withDefaults()
	return &Monitor{
		services: make(map[string]*serviceState),
		cfg:      c,
		provider: provider,
		reloader: reloader,
		log:      log,
		client:   &http.Client{Timeout: c.CheckTimeout},
	}
}

// Start launches the background health-check loop. It runs until ctx is cancelled.
func (m *Monitor) Start(ctx context.Context) {
	// Seed known services immediately so the circuit map is populated before
	// the first request arrives — avoids false "unknown service" opens.
	for _, t := range m.provider.HealthTargets() {
		m.mu.Lock()
		if _, exists := m.services[t.Name]; !exists {
			m.services[t.Name] = &serviceState{
				ServiceStatus: ServiceStatus{
					Name:    t.Name,
					URL:     t.URL,
					Circuit: CircuitClosed,
					Healthy: true, // optimistic until first check
				},
			}
		}
		m.mu.Unlock()
	}

	go func() {
		m.checkAll(ctx)
		ticker := time.NewTicker(m.cfg.CheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.checkAll(ctx)
			}
		}
	}()
}

// ── CircuitBreaker interface (used by executor) ───────────────────────────────

// Allow returns false when the circuit for serviceName is OPEN or HALF_OPEN.
// Returns true (optimistic) for unknown services so new services aren't blocked
// before their first health check completes.
func (m *Monitor) Allow(serviceName string) bool {
	m.mu.RLock()
	s, ok := m.services[serviceName]
	m.mu.RUnlock()
	if !ok {
		return true
	}
	return s.Circuit == CircuitClosed
}

// RecordSuccess reports a successful upstream call from the executor.
// Resets consecutive failure counter and updates latency.
func (m *Monitor) RecordSuccess(serviceName string, latencyMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.getOrCreate(serviceName)
	s.ConsecFailures = 0
	s.LatencyMs = latencyMs
	s.Healthy = true
	now := time.Now()
	if s.HealthySince.IsZero() {
		s.HealthySince = now
	}
	s.CheckedAt = now
}

// RecordFailure reports a failed upstream call from the executor.
// Opens the circuit when consecutive failures exceed the threshold.
func (m *Monitor) RecordFailure(serviceName string, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.getOrCreate(serviceName)
	s.ConsecFailures++
	s.Healthy = false
	s.LastError = errMsg
	s.CheckedAt = time.Now()

	if s.Circuit == CircuitClosed && s.ConsecFailures >= m.cfg.FailThreshold {
		m.openCircuit(s)
	}
}

// ── Status queries ────────────────────────────────────────────────────────────

// Status returns the current health snapshot for one service.
func (m *Monitor) Status(name string) *ServiceStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.services[name]; ok {
		cp := s.ServiceStatus
		return &cp
	}
	return nil
}

// AllStatuses returns snapshots for all known services.
func (m *Monitor) AllStatuses() []*ServiceStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*ServiceStatus, 0, len(m.services))
	for _, s := range m.services {
		cp := s.ServiceStatus
		out = append(out, &cp)
	}
	return out
}

// ── Health check loop ─────────────────────────────────────────────────────────

func (m *Monitor) checkAll(ctx context.Context) {
	targets := m.provider.HealthTargets()
	for _, t := range targets {
		go m.checkOne(ctx, t)
	}
}

func (m *Monitor) checkOne(ctx context.Context, t ServiceTarget) {
	m.mu.Lock()
	s := m.getOrCreate(t.Name)
	s.URL = t.URL

	// Decide whether to probe.
	switch s.Circuit {
	case CircuitOpen:
		if time.Since(s.openedAt) < m.cfg.RecoveryCooldown {
			m.mu.Unlock()
			return // still in cooldown
		}
		// Cooldown elapsed → move to HALF_OPEN and probe.
		s.Circuit = CircuitHalfOpen
	case CircuitClosed, CircuitHalfOpen:
		// always probe
	}
	wasHalfOpen := s.Circuit == CircuitHalfOpen
	m.mu.Unlock()

	start := time.Now()
	err := m.probe(ctx, t)
	latency := time.Since(start).Milliseconds()

	m.mu.Lock()
	defer m.mu.Unlock()

	s.CheckedAt = time.Now()
	s.LatencyMs = latency

	if err != nil {
		s.Healthy = false
		s.LastError = err.Error()
		s.ConsecFailures++
		if s.Circuit == CircuitClosed && s.ConsecFailures >= m.cfg.FailThreshold {
			m.openCircuit(s)
		} else if wasHalfOpen {
			// Probe failed → back to OPEN
			m.openCircuit(s)
		}
		m.log.Warn("health check failed",
			zap.String("service", t.Name),
			zap.Error(err),
			zap.String("circuit", string(s.Circuit)),
		)
		return
	}

	// Probe succeeded.
	s.LastError = ""
	s.ConsecFailures = 0
	s.Healthy = true
	now := time.Now()
	if s.HealthySince.IsZero() || wasHalfOpen {
		s.HealthySince = now
	}

	wasOpen := wasHalfOpen || s.Circuit == CircuitOpen
	s.Circuit = CircuitClosed

	if wasOpen {
		m.log.Info("service recovered — circuit closed",
			zap.String("service", t.Name),
			zap.Int64("latency_ms", latency),
		)
		// Trigger schema re-introspection in the background.
		if m.reloader != nil {
			go func(name string) {
				ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := m.reloader.Reload(ctx2, name); err != nil {
					m.log.Warn("schema reload after recovery failed",
						zap.String("service", name), zap.Error(err))
				}
			}(t.Name)
		}
	}
}

// probe sends a lightweight { __typename } query to verify the GQL endpoint.
func (m *Monitor) probe(ctx context.Context, t ServiceTarget) error {
	body, _ := json.Marshal(map[string]string{"query": "{ __typename }"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("probe failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("probe returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (m *Monitor) openCircuit(s *serviceState) {
	s.Circuit = CircuitOpen
	s.openedAt = time.Now()
	s.HealthySince = time.Time{}
	m.log.Warn("circuit opened",
		zap.String("service", s.Name),
		zap.Int("consec_failures", s.ConsecFailures),
	)
}

// getOrCreate returns the serviceState for name, creating it if absent.
// Caller must hold m.mu.
func (m *Monitor) getOrCreate(name string) *serviceState {
	if s, ok := m.services[name]; ok {
		return s
	}
	s := &serviceState{
		ServiceStatus: ServiceStatus{
			Name:    name,
			Circuit: CircuitClosed,
			Healthy: true,
		},
	}
	m.services[name] = s
	return s
}

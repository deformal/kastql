// Package security provides runtime-configurable middleware for kastql:
// CORS, IP filtering, rate limiting, query guards, size limits, and logging.
package security

import (
	"strconv"
	"sync"
	"time"

	"github.com/deformal/kastql/internal/metadata"
)

// Config is a snapshot of all security settings, loaded from the DB.
type Config struct {
	// CORS
	CORSEnabled  bool
	CORSAllowAll bool
	CORSOrigins  []string // ignored when CORSAllowAll is true

	// IP filtering
	IPFilterEnabled bool
	IPFilterDefault string // "allow" | "deny"
	IPRules         []*metadata.IPRule

	// Rate limiting (0 = disabled)
	RateLimitEnabled   bool
	GlobalRPM          int
	PerIPRPM           int
	MutationRPM        int

	// Query guards (0 = disabled)
	DepthLimit      int
	ComplexityLimit int
	AliasLimit      int
	DirectiveLimit  int

	// Timeouts / sizes (0 = disabled)
	QueryTimeoutMs     int
	MaxRequestBodyKB   int
	MaxResponseBodyKB  int

	// Persisted queries
	PersistedOnly bool

	// WebSocket
	WSMaxConnections int

	// Other
	BatchQueriesEnabled bool
	AuditLogEnabled     bool

	// version counter — incremented every reload so middlewares can detect staleness
	version uint64
}

// Store is the database interface needed by Manager.
type Store interface {
	AllSettings() (map[string]string, error)
	ListCORSOrigins() ([]*metadata.CORSOrigin, error)
	ListIPRules() ([]*metadata.IPRule, error)
	AppendBlockedRequest(reason, ip, path string) error
	AppendAuditLog(admin, action, detail, ip string) error
}

// Manager loads security config from the DB with a short TTL cache and exposes
// it to all middleware in a lock-free snapshot pattern.
type Manager struct {
	store Store

	mu        sync.RWMutex
	cfg       *Config
	loadedAt  time.Time
	cacheTTL  time.Duration

	// rate limiter state (recreated when RPM settings change)
	rl *rateLimitState

	// async blocked-request writer
	blockCh chan blockEntry

	// WebSocket connection counter
	wsMu    sync.Mutex
	wsConns int
}

type blockEntry struct {
	reason string
	ip     string
	path   string
}

func New(store Store) *Manager {
	m := &Manager{
		store:    store,
		cacheTTL: 10 * time.Second,
		blockCh:  make(chan blockEntry, 256),
	}
	go m.blockWriter()
	return m
}

// Config returns the current (possibly cached) security config.
func (m *Manager) Config() *Config {
	m.mu.RLock()
	cfg := m.cfg
	age := time.Since(m.loadedAt)
	m.mu.RUnlock()

	if cfg != nil && age < m.cacheTTL {
		return cfg
	}
	return m.reload()
}

func (m *Manager) reload() *Config {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-checked locking — another goroutine may have loaded while we waited.
	if m.cfg != nil && time.Since(m.loadedAt) < m.cacheTTL {
		return m.cfg
	}

	cfg := m.loadFromDB()
	if cfg == nil {
		if m.cfg != nil {
			return m.cfg // keep stale config rather than breaking
		}
		cfg = &Config{BatchQueriesEnabled: true} // safe defaults
	}

	if m.cfg != nil {
		cfg.version = m.cfg.version + 1
	}

	// Recreate rate limit buckets if RPM settings changed.
	if m.rl == nil ||
		m.cfg == nil ||
		m.cfg.GlobalRPM != cfg.GlobalRPM ||
		m.cfg.PerIPRPM != cfg.PerIPRPM ||
		m.cfg.MutationRPM != cfg.MutationRPM {
		m.rl = newRateLimitState(cfg.GlobalRPM, cfg.PerIPRPM, cfg.MutationRPM)
	}

	m.cfg = cfg
	m.loadedAt = time.Now()
	return cfg
}

// InvalidateCache forces the next Config() call to reload from the DB.
// Call this after any admin mutation that changes security settings.
func (m *Manager) InvalidateCache() {
	m.mu.Lock()
	m.loadedAt = time.Time{}
	m.mu.Unlock()
}

// RateLimiter returns the current rate-limit state (matches the loaded config).
func (m *Manager) RateLimiter() *rateLimitState {
	m.Config() // ensure loaded
	m.mu.RLock()
	rl := m.rl
	m.mu.RUnlock()
	return rl
}

// LogBlocked queues a blocked-request record for async DB write.
func (m *Manager) LogBlocked(reason, ip, path string) {
	select {
	case m.blockCh <- blockEntry{reason, ip, path}:
	default: // drop on full channel — blocked log is best-effort
	}
}

func (m *Manager) blockWriter() {
	for e := range m.blockCh {
		_ = m.store.AppendBlockedRequest(e.reason, e.ip, e.path)
	}
}

// WebSocket connection tracking ───────────────────────────────────────────────

func (m *Manager) WSAcquire() bool {
	cfg := m.Config()
	m.wsMu.Lock()
	defer m.wsMu.Unlock()
	if cfg.WSMaxConnections > 0 && m.wsConns >= cfg.WSMaxConnections {
		return false
	}
	m.wsConns++
	return true
}

func (m *Manager) WSRelease() {
	m.wsMu.Lock()
	if m.wsConns > 0 {
		m.wsConns--
	}
	m.wsMu.Unlock()
}

// ── DB loader ─────────────────────────────────────────────────────────────────

func (m *Manager) loadFromDB() *Config {
	settings, err := m.store.AllSettings()
	if err != nil {
		return nil
	}

	origins, _ := m.store.ListCORSOrigins()
	ipRules, _ := m.store.ListIPRules()

	originStrs := make([]string, 0, len(origins))
	for _, o := range origins {
		originStrs = append(originStrs, o.Origin)
	}

	cfg := &Config{
		CORSEnabled:         boolSetting(settings, "cors_enabled"),
		CORSAllowAll:        boolSetting(settings, "cors_allow_all"),
		CORSOrigins:         originStrs,
		IPFilterEnabled:     boolSetting(settings, "ip_filter_enabled"),
		IPFilterDefault:     strSetting(settings, "ip_filter_default", "allow"),
		IPRules:             ipRules,
		RateLimitEnabled:    boolSetting(settings, "rate_limit_enabled"),
		GlobalRPM:           intSetting(settings, "rate_limit_global_rpm"),
		PerIPRPM:            intSetting(settings, "rate_limit_ip_rpm"),
		MutationRPM:         intSetting(settings, "rate_limit_mutation_rpm"),
		DepthLimit:          intSetting(settings, "query_depth_limit"),
		ComplexityLimit:     intSetting(settings, "query_complexity_limit"),
		AliasLimit:          intSetting(settings, "query_alias_limit"),
		DirectiveLimit:      intSetting(settings, "query_directive_limit"),
		QueryTimeoutMs:      intSetting(settings, "query_timeout_ms"),
		MaxRequestBodyKB:    intSetting(settings, "max_request_body_kb"),
		MaxResponseBodyKB:   intSetting(settings, "max_response_body_kb"),
		PersistedOnly:       boolSetting(settings, "persisted_only"),
		WSMaxConnections:    intSetting(settings, "ws_max_connections"),
		BatchQueriesEnabled: boolSettingDefault(settings, "batch_queries_enabled", true),
		AuditLogEnabled:     boolSettingDefault(settings, "audit_log_enabled", true),
	}
	return cfg
}

// ── Setting helpers ───────────────────────────────────────────────────────────

func boolSetting(m map[string]string, key string) bool {
	v := m[key]
	return v == "1" || v == "true"
}

func boolSettingDefault(m map[string]string, key string, def bool) bool {
	v, ok := m[key]
	if !ok {
		return def
	}
	return v == "1" || v == "true"
}

func intSetting(m map[string]string, key string) int {
	n, _ := strconv.Atoi(m[key])
	return n
}

func strSetting(m map[string]string, key, def string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return def
}

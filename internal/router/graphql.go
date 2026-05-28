package router

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/deformal/kastql/internal/auth"
	"github.com/deformal/kastql/internal/cache"
	"github.com/deformal/kastql/internal/executor"
	"github.com/deformal/kastql/internal/metrics"
	"github.com/deformal/kastql/internal/planner"
	"github.com/deformal/kastql/internal/security"
)

type graphqlHandler struct {
	planner              *planner.Planner
	executor             *executor.Executor
	metrics              *metrics.Store
	log                  *zap.Logger
	introspectionEnabled func() bool
	secMgr               *security.Manager // nil = security disabled
	cache                *cache.Cache      // nil = caching disabled
}

type graphqlRequest struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables"`
	OperationName string         `json:"operationName"`
}

func (h *graphqlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// ── Security: persisted query / batch guard ───────────────────────────────
	var cfg *security.Config
	if h.secMgr != nil {
		cfg = h.secMgr.Config()
	}

	var req graphqlRequest

	if cfg != nil && h.secMgr != nil {
		// Decode raw body to detect batch arrays and handle persisted queries.
		var raw json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			writeGQLError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Batch query detection — reject if batching disabled.
		if len(raw) > 0 && raw[0] == '[' {
			if !cfg.BatchQueriesEnabled {
				ip := security.ClientIP(r)
				h.secMgr.LogBlocked("batch_not_allowed", ip, r.URL.Path)
				writeGQLError(w, "batch queries are disabled", http.StatusBadRequest)
				return
			}
			// Batch is enabled: decode all and process first (simple forward for now).
			var batch []graphqlRequest
			if err := json.Unmarshal(raw, &batch); err != nil {
				writeGQLError(w, "invalid batch body: "+err.Error(), http.StatusBadRequest)
				return
			}
			if len(batch) == 0 {
				writeGQLError(w, "empty batch", http.StatusBadRequest)
				return
			}
			req = batch[0]
		} else {
			if err := json.Unmarshal(raw, &req); err != nil {
				writeGQLError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		// Persisted-query enforcement.
		if cfg.PersistedOnly && req.Query != "" {
			ip := security.ClientIP(r)
			h.secMgr.LogBlocked("non_persisted_query", ip, r.URL.Path)
			writeGQLError(w, "only persisted queries are accepted", http.StatusForbidden)
			return
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeGQLError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	if req.Query == "" {
		writeGQLError(w, "query is required", http.StatusBadRequest)
		return
	}

	// ── Security: query analysis (depth / complexity / aliases / directives) ──
	if cfg != nil && h.secMgr != nil {
		analysis := h.planner.Analyze(req.Query)
		if analysis != nil {
			ip := security.ClientIP(r)

			if cfg.DepthLimit > 0 && analysis.Depth > cfg.DepthLimit {
				h.secMgr.LogBlocked("query_too_deep", ip, r.URL.Path)
				writeGQLError(w, "query exceeds maximum depth", http.StatusBadRequest)
				return
			}
			if cfg.ComplexityLimit > 0 && analysis.Complexity > cfg.ComplexityLimit {
				h.secMgr.LogBlocked("query_too_complex", ip, r.URL.Path)
				writeGQLError(w, "query exceeds maximum complexity", http.StatusBadRequest)
				return
			}
			if cfg.AliasLimit > 0 && analysis.Aliases > cfg.AliasLimit {
				h.secMgr.LogBlocked("too_many_aliases", ip, r.URL.Path)
				writeGQLError(w, "query exceeds maximum alias count", http.StatusBadRequest)
				return
			}
			if cfg.DirectiveLimit > 0 && analysis.Directives > cfg.DirectiveLimit {
				h.secMgr.LogBlocked("too_many_directives", ip, r.URL.Path)
				writeGQLError(w, "query exceeds maximum directive count", http.StatusBadRequest)
				return
			}

			// Mutation-specific rate limit (checked here after body parse).
			if analysis.IsMutation && cfg.RateLimitEnabled {
				rl := h.secMgr.RateLimiter()
				if !rl.AllowMutation() {
					h.secMgr.LogBlocked("mutation_rate_limit", ip, r.URL.Path)
					writeGQLError(w, "mutation rate limit exceeded", http.StatusTooManyRequests)
					return
				}
			}
		}
	}

	// ── Introspection check ───────────────────────────────────────────────────
	if isIntrospectionQuery(req.Query) {
		if h.introspectionEnabled != nil && !h.introspectionEnabled() {
			writeGQLError(w, "GraphQL introspection is disabled.", http.StatusOK)
			return
		}
		h.serveIntrospection(w, r)
		return
	}

	// ── Query timeout ─────────────────────────────────────────────────────────
	ctx := r.Context()
	if cfg != nil && cfg.QueryTimeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(cfg.QueryTimeoutMs)*time.Millisecond)
		defer cancel()
	}

	start := time.Now()
	role := auth.GetRole(ctx)

	// ── Response cache (queries only, skip mutations/introspection) ───────────
	var cacheKey string
	if h.cache != nil && !strings.Contains(strings.ToLower(req.Query), "mutation") {
		cacheKey = cache.QueryKey(req.Query, req.OperationName, role, req.Variables)
		if cached, ok := h.cache.Get(cacheKey); ok {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			w.Write(cached)
			return
		}
	}

	headers := forwardHeaders(r)
	h.log.Debug("incoming graphql request",
		zap.String("operation", req.OperationName),
		zap.Strings("forwarded_headers", headerKeys(headers)),
		zap.String("role", role),
	)

	plan, err := h.planner.Plan(ctx, req.Query, req.Variables, role)
	if err != nil {
		h.recordMetric(plan, time.Since(start), false, err.Error())
		writeGQLError(w, err.Error(), http.StatusOK)
		return
	}

	// ── Response size cap ─────────────────────────────────────────────────────
	var rw http.ResponseWriter = w
	if cfg != nil && cfg.MaxResponseBodyKB > 0 {
		rw = security.NewCappedWriter(w, cfg.MaxResponseBodyKB)
	}

	result, err := h.executor.Execute(ctx, plan, headers)
	elapsed := time.Since(start)

	if err != nil {
		h.recordMetric(plan, elapsed, false, err.Error())
		writeGQLError(rw, err.Error(), http.StatusOK)
		return
	}

	success := len(result.Errors) == 0
	errMsg := ""
	if !success && len(result.Errors) > 0 {
		errMsg = result.Errors[0].Message
	}
	h.recordMetric(plan, elapsed, success, errMsg)

	rw.Header().Set("Content-Type", "application/json")
	if cacheKey != "" {
		rw.Header().Set("X-Cache", "MISS")
	}
	rw.WriteHeader(http.StatusOK)

	if cacheKey != "" && success {
		// Encode into a buffer so we can cache and write in one pass.
		buf, _ := json.Marshal(result)
		h.cache.Set(cacheKey, buf)
		rw.Write(buf)
	} else {
		json.NewEncoder(rw).Encode(result)
	}
}

func (h *graphqlHandler) recordMetric(plan *planner.QueryPlan, d time.Duration, success bool, errMsg string) {
	if h.metrics == nil {
		return
	}
	entry := &metrics.QueryEntry{
		DurationMs:   d.Milliseconds(),
		Success:      success,
		ErrorMessage: errMsg,
	}
	if plan != nil {
		entry.OperationType = plan.OperationType
		entry.OperationName = plan.OperationName
		seen := map[string]bool{}
		for _, s := range plan.Steps {
			if !seen[s.ServiceName] {
				seen[s.ServiceName] = true
				entry.ServicesCalled = append(entry.ServicesCalled, s.ServiceName)
			}
		}
	}
	if err := h.metrics.RecordQuery(entry); err != nil {
		h.log.Warn("failed to record metrics", zap.Error(err))
	}
}

// hopByHopHeaders are connection-scoped headers that must never be forwarded.
var hopByHopHeaders = map[string]bool{
	"connection":          true,
	"keep-alive":          true,
	"proxy-authenticate":  true,
	"proxy-authorization": true,
	"te":                  true,
	"trailers":            true,
	"transfer-encoding":   true,
	"upgrade":             true,
	// added by the router itself when it re-encodes the body
	"content-length": true,
	"content-type":   true,
	// kastql-internal headers that must not leak to upstreams
	"cookie":        true,
	"x-router-key": true,
}

// forwardHeaders passes every non-hop-by-hop header from the client request
// to the upstream service.
func forwardHeaders(r *http.Request) map[string]string {
	forward := map[string]string{}
	for name, values := range r.Header {
		lower := strings.ToLower(name)
		if hopByHopHeaders[lower] {
			continue
		}
		if len(values) > 0 {
			forward[name] = values[0]
		}
	}
	return forward
}

func headerKeys(h map[string]string) []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}

func writeGQLError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]any{{"message": msg}},
	})
}

// isIntrospectionQuery returns true when the query contains meta-fields.
func isIntrospectionQuery(query string) bool {
	q := strings.TrimSpace(query)
	return strings.Contains(q, "__schema") || strings.Contains(q, "__type")
}

// serveIntrospection builds the introspection response from kastql's merged schema.
func (h *graphqlHandler) serveIntrospection(w http.ResponseWriter, _ *http.Request) {
	schema := h.planner.Schema()
	if schema == nil {
		writeGQLError(w, "no schema loaded — register at least one service", http.StatusOK)
		return
	}

	data := planner.BuildIntrospectionResponse(schema)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"data": data})
}

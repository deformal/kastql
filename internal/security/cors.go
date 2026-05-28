package security

import (
	"net/http"
)

// CORSMiddleware adds CORS headers and handles preflight requests.
// When cors_allow_all is true, any origin is accepted.
// Otherwise, only origins in the cors_origins table are accepted.
func CORSMiddleware(mgr *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := mgr.Config()
			if !cfg.CORSEnabled {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if !isAllowedOrigin(origin, cfg) {
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers",
					"Content-Type, Authorization, X-Router-Key, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IsAllowedOrigin is exported so ws.go can use the same logic for the WebSocket upgrader.
func IsAllowedOrigin(origin string, mgr *Manager) bool {
	return isAllowedOrigin(origin, mgr.Config())
}

func isAllowedOrigin(origin string, cfg *Config) bool {
	if cfg.CORSAllowAll {
		return true
	}
	for _, o := range cfg.CORSOrigins {
		if o == origin {
			return true
		}
	}
	return false
}

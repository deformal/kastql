package security

import (
	"net"
	"net/http"
	"strings"
)

// IPFilterMiddleware allows or denies requests based on IP rules stored in the DB.
// Decision logic:
//  1. If a rule matches → use that rule's mode.
//  2. If no rule matches → use cfg.IPFilterDefault ("allow" | "deny").
func IPFilterMiddleware(mgr *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := mgr.Config()
			if !cfg.IPFilterEnabled {
				next.ServeHTTP(w, r)
				return
			}

			ip := clientIP(r)
			netIP := net.ParseIP(ip)

			decision := cfg.IPFilterDefault // "allow" | "deny"
			for _, rule := range cfg.IPRules {
				_, cidr, err := net.ParseCIDR(rule.CIDR)
				if err != nil {
					continue
				}
				if netIP != nil && cidr.Contains(netIP) {
					decision = rule.Mode
					break
				}
			}

			if decision == "deny" {
				mgr.LogBlocked("ip_filter", ip, r.URL.Path)
				http.Error(w, `{"errors":[{"message":"access denied"}]}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP extracts the real client IP, respecting X-Forwarded-For and X-Real-IP.
func ClientIP(r *http.Request) string {
	return clientIP(r)
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For: client, proxy1, proxy2 — take leftmost
		if idx := strings.IndexByte(xff, ','); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

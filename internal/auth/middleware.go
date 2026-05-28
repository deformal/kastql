package auth

import (
	"context"
	"net/http"
	"slices"

	"go.uber.org/zap"

	"github.com/deformal/kastql/internal/config"
)

// Middleware validates JWT tokens and populates the role in the request context.
type Middleware struct {
	cfg           config.AuthConfig
	secretsLoader func() []string // if set, DB-backed secrets take precedence over cfg.JWTSecret
	log           *zap.Logger
}

// New creates an auth Middleware from the given config.
func New(cfg config.AuthConfig, log *zap.Logger) *Middleware {
	return &Middleware{cfg: cfg, log: log}
}

// SetSecretsLoader wires a function that returns active JWT secrets from the DB.
// Called on every request; results are not cached here — cache at the call site
// if needed. When the loader returns a non-empty slice, cfg.JWTSecret is ignored.
func (m *Middleware) SetSecretsLoader(fn func() []string) {
	m.secretsLoader = fn
}

// Handler returns a chi-compatible middleware function.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ctx := m.resolveRole(r)
		ctx = SetRole(ctx, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// resolveRole determines the effective role and returns the (possibly enriched)
// context alongside it. Precedence:
//  1. Valid JWT → role from configured claim
//  2. X-Kastql-Role header → trusted only in dev mode (no secrets configured)
//     or when the role is in the JWT's allowed_roles claim
//  3. Default role from config
func (m *Middleware) resolveRole(r *http.Request) (role string, ctx context.Context) {
	ctx = r.Context()
	authHeader := r.Header.Get(m.cfg.JWTHeader)
	headerRole := r.Header.Get("X-Kastql-Role")

	secrets := m.activeSecrets()

	for _, secret := range secrets {
		jwtRole, claims, err := ParseBearer(authHeader, secret, m.cfg.RoleClaim)
		if err != nil {
			m.log.Debug("jwt validation failed", zap.Error(err))
			continue
		}
		if claims == nil {
			// No JWT present — not an error, just unauthenticated.
			break
		}

		// JWT is valid — store claims and resolve role.
		ctx = SetClaims(ctx, claims)

		if headerRole != "" && slices.Contains(AllowedRoles(claims), headerRole) {
			return headerRole, ctx
		}
		if jwtRole != "" {
			return jwtRole, ctx
		}
		return m.cfg.DefaultRole, ctx
	}

	// No valid JWT or no secrets configured.
	if headerRole != "" && len(secrets) == 0 {
		// Dev mode: trust X-Kastql-Role header directly.
		return headerRole, ctx
	}

	return m.cfg.DefaultRole, ctx
}

func (m *Middleware) activeSecrets() []string {
	if m.secretsLoader != nil {
		if secrets := m.secretsLoader(); len(secrets) > 0 {
			return secrets
		}
	}
	if m.cfg.JWTSecret != "" {
		return []string{m.cfg.JWTSecret}
	}
	return nil
}

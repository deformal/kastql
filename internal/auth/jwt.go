package auth

import (
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ParseBearer validates the "Authorization: Bearer <token>" header value and
// returns the role extracted from the configured claim.
// Returns ("", nil) when the header is empty (unauthenticated is OK — caller
// falls back to the default role).
func ParseBearer(authHeader, secret, roleClaim string) (role string, claims map[string]any, err error) {
	if authHeader == "" {
		return "", nil, nil
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", nil, errors.New("authorization header must be 'Bearer <token>'")
	}

	token, parseErr := jwt.Parse(parts[1], func(t *jwt.Token) (any, error) {
		switch t.Method.(type) {
		case *jwt.SigningMethodHMAC:
			return []byte(secret), nil
		default:
			return nil, fmt.Errorf("unsupported signing method: %v", t.Header["alg"])
		}
	})
	if parseErr != nil {
		return "", nil, fmt.Errorf("invalid JWT: %w", parseErr)
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", nil, errors.New("invalid token claims")
	}

	// Convert to plain map for context storage.
	raw := make(map[string]any, len(mapClaims))
	maps.Copy(raw, map[string]any(mapClaims))

	if roleClaim != "" {
		if r, ok := mapClaims[roleClaim].(string); ok {
			return r, raw, nil
		}
	}

	return "", raw, nil
}

// AllowedRoles extracts the list of roles the token is permitted to use.
// Looks for a "allowed_roles" or "x-kastql-allowed-roles" claim (string slice).
func AllowedRoles(claims map[string]any) []string {
	for _, key := range []string{"allowed_roles", "x-kastql-allowed-roles"} {
		switch v := claims[key].(type) {
		case []any:
			roles := make([]string, 0, len(v))
			for _, r := range v {
				if s, ok := r.(string); ok {
					roles = append(roles, s)
				}
			}
			if len(roles) > 0 {
				return roles
			}
		case []string:
			return v
		}
	}
	return nil
}

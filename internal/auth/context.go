package auth

import "context"

type contextKey string

const (
	roleKey   contextKey = "kastql_role"
	claimsKey contextKey = "kastql_claims"
)

// SetRole stores the resolved role in the context.
func SetRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

// GetRole returns the role stored by SetRole, or "" if none.
func GetRole(ctx context.Context) string {
	v, _ := ctx.Value(roleKey).(string)
	return v
}

// SetClaims stores the raw JWT claims map in the context.
func SetClaims(ctx context.Context, claims map[string]any) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// GetClaims returns the JWT claims stored by SetClaims.
func GetClaims(ctx context.Context) map[string]any {
	v, _ := ctx.Value(claimsKey).(map[string]any)
	return v
}

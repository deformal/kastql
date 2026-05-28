package auth

import "net/http"

// RouterKeyStore is satisfied by *metadata.Store.
type RouterKeyStore interface {
	ValidateRouterKey(key string) (bool, error)
	HasActiveRouterKeys() (bool, error)
}

// RouterKeyMiddleware enforces the X-Router-Key header when the DB has active keys.
// If no keys are configured, all requests pass through (open mode).
func RouterKeyMiddleware(store RouterKeyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hasKeys, err := store.HasActiveRouterKeys()
			if err != nil || !hasKeys {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("X-Router-Key")
			if key == "" {
				writeAPIUnauthorized(w, "X-Router-Key header required")
				return
			}

			valid, err := store.ValidateRouterKey(key)
			if err != nil || !valid {
				writeAPIUnauthorized(w, "invalid router key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeAPIUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"errors":[{"message":"` + msg + `"}]}`))
}

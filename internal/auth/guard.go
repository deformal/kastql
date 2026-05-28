package auth

import (
	"net/http"
)

// AdminGuard redirects to /admin/login if the admin session cookie is missing or invalid.
func AdminGuard(session *SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := session.Validate(r, AdminCookieName); err != nil {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserGuard serves a 401 page if the user session cookie is missing or invalid.
func UserGuard(session *SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := session.Validate(r, UserCookieName); err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

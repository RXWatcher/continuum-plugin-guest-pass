package server

import (
	"net/http"

	"github.com/ContinuumApp/continuum-plugin-guest-pass/internal/auth"
)

// requireStore short-circuits with 503 when the plugin hasn't been
// configured yet (no DB pool). Mounted on every API surface so handlers
// can assume d.Store is non-nil.
func requireStore(d Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if d.Store == nil {
				writeErr(w, http.StatusServiceUnavailable, "not_configured", "guest pass plugin is not configured")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requireAdmin gates admin routes on the host-provided identity headers.
func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := auth.FromRequest(r)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "unauthenticated", "missing identity")
			return
		}
		if !id.IsAdmin {
			writeErr(w, http.StatusForbidden, "forbidden", "admin required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

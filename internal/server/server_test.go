package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// New(Deps{}) should mount the routes but every API call should hit the
// "not configured" gate. The point of the test is to guarantee the
// handler boots before the store is wired — early SDK lifecycle.
func TestAPIReturnsNotConfiguredBeforeStoreWired(t *testing.T) {
	h := New(Deps{})
	for _, path := range []string{
		"/api/public/passes/test-token",
		"/api/admin/passes",
		"/api/admin/config",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Continuum-User-Id", "admin-1")
		req.Header.Set("X-Continuum-User-Role", "admin")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s status = %d, want %d; body=%s", path, rec.Code, http.StatusServiceUnavailable, rec.Body.String())
		}
	}
}

func TestPluginBaseHref(t *testing.T) {
	tests := map[string]string{
		"":                      "./",
		"/api/v1/plugins/48":    "/api/v1/plugins/48/",
		"/api/v1/plugins/slug/": "/api/v1/plugins/slug/",
	}
	for mountPath, want := range tests {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		if mountPath != "" {
			req.Header.Set("X-Continuum-Plugin-Mount-Path", mountPath)
		}
		if got := pluginBaseHref(req); got != want {
			t.Fatalf("pluginBaseHref(%q) = %q, want %q", mountPath, got, want)
		}
	}
}

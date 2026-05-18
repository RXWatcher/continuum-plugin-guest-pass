package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAPIReportsNotConfiguredInsteadOfPanicking(t *testing.T) {
	h := New(Deps{})
	for _, path := range []string{"/api/public/passes/test-token", "/api/admin/passes"} {
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
		"/p/token":                         "/",
		"/api/v1/plugins/guest-pass/admin": "/api/v1/plugins/guest-pass/",
		"/api/v1/plugins/42/p/token":       "/api/v1/plugins/42/",
		"/api/v1/plugins/slug":             "/api/v1/plugins/slug/",
	}
	for path, want := range tests {
		if got := pluginBaseHref(path); got != want {
			t.Fatalf("pluginBaseHref(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestParseExpiryValidation(t *testing.T) {
	if _, err := parseExpiry("", 24*365+1); err == nil {
		t.Fatal("expected excessive relative expiry to fail")
	}
	if _, err := parseExpiry(time.Now().Add(-time.Hour).Format(time.RFC3339), 0); err == nil {
		t.Fatal("expected past absolute expiry to fail")
	}
	got, err := parseExpiry("", 0)
	if err != nil {
		t.Fatalf("default expiry failed: %v", err)
	}
	if time.Until(got) < 23*time.Hour || time.Until(got) > 25*time.Hour {
		t.Fatalf("default expiry = %s, want about 24h from now", got)
	}
}

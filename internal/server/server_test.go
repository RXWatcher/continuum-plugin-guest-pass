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

func TestValidTargetTypeOnlyAllowsPlayableMediaFile(t *testing.T) {
	if !validTargetType("media_file") {
		t.Fatal("media_file should be accepted")
	}
	for _, targetType := range []string{"movie", "episode", "series", "collection", "watch_room", "ebook", "audiobook", ""} {
		if validTargetType(targetType) {
			t.Fatalf("target type %q should be rejected until playback grants support it", targetType)
		}
	}
}

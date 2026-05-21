package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RXWatcher/continuum-plugin-guest-pass/internal/store"
)

func TestClientIPReadsHostHeaderOnly(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "127.0.0.1:99"
	r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")

	// Without the host-stamped header, clientIP returns "" — never the
	// XFF value and never the gRPC peer address.
	if got := clientIP(r); got != "" {
		t.Errorf("clientIP without host header = %q, want empty", got)
	}

	r.Header.Set(HeaderClientIP, "203.0.113.7")
	if got := clientIP(r); got != "203.0.113.7" {
		t.Errorf("clientIP with host header = %q, want 203.0.113.7", got)
	}
}

func TestRequestCountryFromKnownHeaders(t *testing.T) {
	for _, h := range []string{"CF-IPCountry", "X-Geo-Country", "X-Country-Code"} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set(h, "nl")
		if got := requestCountry(r); got != "NL" {
			t.Errorf("requestCountry via %s = %q, want NL", h, got)
		}
	}
}

func TestAllowsCountryUnionsAllowlistAndGeofence(t *testing.T) {
	p := &store.Pass{CountryAllowlist: []string{"US"}, Geofence: []string{"NL"}}
	if !allowsCountry(p, "US") {
		t.Error("US should be allowed via allowlist")
	}
	if !allowsCountry(p, "NL") {
		t.Error("NL should be allowed via geofence")
	}
	if allowsCountry(p, "DE") {
		t.Error("DE should be rejected")
	}
	if !allowsCountry(&store.Pass{}, "DE") {
		t.Error("empty allowlist+geofence should allow all")
	}
	if allowsCountry(p, "") {
		t.Error("configured allowlist with empty country should reject")
	}
}

func TestFallbackDeviceIDDiffersByUserAgent(t *testing.T) {
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r1.Header.Set(HeaderClientIP, "10.0.0.1")
	r1.Header.Set("User-Agent", "test/1")
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.Header.Set(HeaderClientIP, "10.0.0.1")
	r2.Header.Set("User-Agent", "test/1")
	if fallbackDeviceID(r1) != fallbackDeviceID(r2) {
		t.Error("fallback device ID should be stable across requests for same IP+UA")
	}
	r3 := httptest.NewRequest(http.MethodGet, "/", nil)
	r3.Header.Set(HeaderClientIP, "10.0.0.1")
	r3.Header.Set("User-Agent", "test/2")
	if fallbackDeviceID(r1) == fallbackDeviceID(r3) {
		t.Error("fallback device ID should differ when UA differs")
	}
}

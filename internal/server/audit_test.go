package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RXWatcher/silo-plugin-guest-pass/internal/auth"
)

func TestAdminAttrsStampsActorAndReason(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/admin/passes", nil)
	req.Header.Set(auth.HeaderUserID, "user-42")
	req.Header.Set(auth.HeaderRole, "admin")

	attrs := adminAttrs(req, "admin minted guest pass")
	if attrs["actor"] != "user-42" {
		t.Fatalf("actor = %v, want user-42", attrs["actor"])
	}
	if attrs["reason"] != "admin minted guest pass" {
		t.Fatalf("reason = %v, want minted reason", attrs["reason"])
	}
}

func TestAdminAttrsUnknownActorWhenHeaderMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/admin/passes", nil)
	attrs := adminAttrs(req, "admin revoked guest pass")
	if attrs["actor"] != "unknown" {
		t.Fatalf("actor = %v, want unknown when no X-Silo-User-Id", attrs["actor"])
	}
}

func TestRedemptionAttrsStampsDeviceAndReason(t *testing.T) {
	attrs := redemptionAttrs(accessRequest{DeviceID: "dev-1"}, "open allowed")
	if attrs["device_id"] != "dev-1" {
		t.Fatalf("device_id = %v, want dev-1", attrs["device_id"])
	}
	if attrs["reason"] != "open allowed" {
		t.Fatalf("reason = %v, want open allowed", attrs["reason"])
	}
}

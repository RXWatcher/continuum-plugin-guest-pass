package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/RXWatcher/silo-plugin-guest-pass/internal/store"
)

// HeaderClientIP is the host-injected header containing the resolved
// client IP. The host's clientip.Resolver normalises this against the
// configured trusted-proxy set before forwarding to the plugin, so a
// plugin should only read this header and never re-parse X-Forwarded-For
// itself (the host strips XFF from the forwarded request anyway).
//
// At the time of this writing the host does not yet stamp this header.
// When that lands the existing IP-based features (LockToFirstIP,
// IPAllowlist, audit IPs, watermark {{ip}} substitution) become live
// automatically. Until then clientIP returns "" and those features
// are effectively inert.
const HeaderClientIP = "X-Silo-Client-IP"

// accessRequest is the JSON body sent to public open/play endpoints.
type accessRequest struct {
	PIN      string `json:"pin"`
	DeviceID string `json:"device_id"`
}

func readAccessRequest(r *http.Request) accessRequest {
	var req accessRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	req.PIN = strings.TrimSpace(req.PIN)
	req.DeviceID = normalizeDeviceID(req.DeviceID)
	if req.DeviceID == "" {
		req.DeviceID = fallbackDeviceID(r)
	}
	return req
}

func normalizeDeviceID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 128 {
		value = value[:128]
	}
	return value
}

// fallbackDeviceID synthesises a stable device id from request features
// when the client didn't supply one. Used for IP+UA based de-dup so we
// can still count devices for clients that don't persist a UUID.
func fallbackDeviceID(r *http.Request) string {
	sum := sha256.Sum256([]byte(clientIP(r) + "\n" + r.UserAgent()))
	return "derived-" + hex.EncodeToString(sum[:16])
}

// clientIP returns the requester's IP as resolved by the host. Reads
// only HeaderClientIP — never X-Forwarded-For (which the host doesn't
// forward) and never r.RemoteAddr (which is the gRPC peer, i.e. the
// host process itself, not the user). Returns "" when the host has not
// stamped a client IP so callers can fail safely instead of relying on
// a misleading value.
func clientIP(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get(HeaderClientIP))
}

// requestCountry extracts an ISO 3166-1 alpha-2 country code from common
// proxy headers. Returns empty when unknown.
func requestCountry(r *http.Request) string {
	for _, header := range []string{"CF-IPCountry", "X-Geo-Country", "X-Country-Code"} {
		if value := strings.ToUpper(strings.TrimSpace(r.Header.Get(header))); value != "" {
			return value
		}
	}
	return ""
}

// allowsCountry merges country_allowlist and geofence (both treated as
// allowlists by union). Empty merged list means no geo restriction.
func allowsCountry(p *store.Pass, country string) bool {
	allowed := append([]string{}, p.CountryAllowlist...)
	allowed = append(allowed, p.Geofence...)
	if len(allowed) == 0 {
		return true
	}
	country = strings.ToUpper(strings.TrimSpace(country))
	if country == "" {
		return false
	}
	for _, item := range allowed {
		if strings.ToUpper(strings.TrimSpace(item)) == country {
			return true
		}
	}
	return false
}

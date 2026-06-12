package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/RXWatcher/silo-plugin-guest-pass/internal/auth"
	"github.com/RXWatcher/silo-plugin-guest-pass/internal/store"
)

// createPassRequest mirrors the admin create-pass JSON body.
type createPassRequest struct {
	Title                    string `json:"title"`
	TargetType               string `json:"target_type"`
	TargetID                 string `json:"target_id"`
	Note                     string `json:"note"`
	ExpiresAt                string `json:"expires_at"`
	ExpiresInHours           int    `json:"expires_in_hours"`
	ValidHoursAfterFirstOpen int    `json:"valid_hours_after_first_open"`
	MaxOpens                 int    `json:"max_opens"`
	MaxPlays                 int    `json:"max_plays"`
	MaxWatchMinutes          int    `json:"max_watch_minutes"`
	MaxConcurrentStreams     int    `json:"max_concurrent_streams"`
	MaxDevices               int    `json:"max_devices"`
	MaxResolution            string `json:"max_resolution"`
	AllowDownloads           bool   `json:"allow_downloads"`
	AllowDirectPlay          bool   `json:"allow_direct_play"`
	LockToFirstIP            bool   `json:"lock_to_first_ip"`
	RequirePIN               bool   `json:"require_pin"`
	PIN                      string `json:"pin"`
	DisableSeeking           bool   `json:"disable_seeking"`
	WatermarkMode            string `json:"watermark_mode"`
	WatermarkProfile         string `json:"watermark_profile"`
	WatermarkLogoURL         string `json:"watermark_logo_url"`
	IPAllowlist              string `json:"ip_allowlist"`
	CountryAllowlist         string `json:"country_allowlist"`
	SessionGraceMinutes      int    `json:"session_grace_minutes"`
	PerItemPlayCount         bool   `json:"per_item_play_count"`
	Geofence                 string `json:"geofence"`
}

func hListPasses(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		passes, err := d.Store.ListPasses(r.Context(), 100)
		if err != nil {
			writeInternal(w, r, d, "list_failed", err)
			return
		}
		now := time.Now()
		out := make([]passResponse, 0, len(passes))
		base := publicBaseURL(r, d)
		for _, p := range passes {
			out = append(out, decoratePass(p, "", base, now))
		}
		writeJSON(w, http.StatusOK, map[string]any{"passes": out})
	}
}

func hCreatePass(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := auth.FromRequest(r)
		var req createPassRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad_json", "invalid JSON body")
			return
		}
		req.Title = strings.TrimSpace(req.Title)
		req.TargetType = strings.TrimSpace(req.TargetType)
		req.TargetID = strings.TrimSpace(req.TargetID)
		if req.Title == "" || req.TargetType == "" || req.TargetID == "" {
			writeErr(w, http.StatusBadRequest, "missing_fields", "title, target_type, and target_id are required")
			return
		}
		if !validTargetType(req.TargetType) {
			writeErr(w, http.StatusBadRequest, "bad_target_type", "target_type must be media_file")
			return
		}
		if req.RequirePIN && strings.TrimSpace(req.PIN) == "" {
			writeErr(w, http.StatusBadRequest, "missing_pin", "pin is required when require_pin is enabled")
			return
		}
		expiresAt, err := parseExpiry(req.ExpiresAt, req.ExpiresInHours)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "bad_expiry", err.Error())
			return
		}
		p, token, err := d.Store.CreatePass(r.Context(), store.CreatePassInput{
			Title:                    req.Title,
			TargetType:               req.TargetType,
			TargetID:                 req.TargetID,
			Note:                     req.Note,
			CreatedBy:                id.UserID,
			ExpiresAt:                expiresAt,
			ValidHoursAfterFirstOpen: nonNegative(req.ValidHoursAfterFirstOpen),
			MaxOpens:                 capUnlimited(req.MaxOpens, maxOpensCap),
			MaxPlays:                 capUnlimited(req.MaxPlays, maxPlaysCap),
			MaxWatchMinutes:          capUnlimited(req.MaxWatchMinutes, maxWatchMinutesCap),
			MaxConcurrentStreams:     capPositive(req.MaxConcurrentStreams, 1, maxConcurrentStreamsCap),
			MaxDevices:               capPositive(req.MaxDevices, 1, maxDevicesCap),
			MaxResolution:            defaultString(req.MaxResolution, "1080p"),
			AllowDownloads:           req.AllowDownloads,
			AllowDirectPlay:          req.AllowDirectPlay,
			LockToFirstIP:            req.LockToFirstIP,
			RequirePIN:               req.RequirePIN,
			PIN:                      strings.TrimSpace(req.PIN),
			DisableSeeking:           req.DisableSeeking,
			WatermarkMode:            normalizeWatermarkMode(req.WatermarkMode, req.WatermarkProfile),
			WatermarkProfile:         strings.TrimSpace(req.WatermarkProfile),
			WatermarkLogoURL:         strings.TrimSpace(req.WatermarkLogoURL),
			IPAllowlist:              splitList(req.IPAllowlist),
			CountryAllowlist:         splitList(req.CountryAllowlist),
			SessionGraceMinutes:      nonNegative(req.SessionGraceMinutes),
			PerItemPlayCount:         req.PerItemPlayCount,
			Geofence:                 splitList(req.Geofence),
		})
		if err != nil {
			writeInternal(w, r, d, "create_failed", err)
			return
		}
		_ = d.Store.RecordEvent(r.Context(), p.ID, "created", clientIP(r), r.UserAgent(), adminAttrs(r, "admin minted guest pass"))
		baseURL := publicBaseURL(r, d)
		writeJSON(w, http.StatusCreated, map[string]any{
			"pass":      decoratePass(*p, token, baseURL, time.Now()),
			"token":     token,
			"share_url": shareURL(baseURL, token),
		})
	}
}

func hRevokePass(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			writeErr(w, http.StatusBadRequest, "bad_id", "invalid pass id")
			return
		}
		if err := d.Store.RevokePass(r.Context(), id); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeErr(w, http.StatusNotFound, "not_found", "guest pass not found")
				return
			}
			writeInternal(w, r, d, "revoke_failed", err)
			return
		}
		_ = d.Store.RecordEvent(r.Context(), id, "revoked", clientIP(r), r.UserAgent(), adminAttrs(r, "admin revoked guest pass"))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

// adminAttrs builds the audit attrs for an admin-initiated action, stamping
// the acting user (X-Silo-User-Id) and a human-readable reason so the event
// log records who minted or revoked a pass and why. When the host did not
// stamp a user id the actor is recorded as "unknown" rather than dropped.
func adminAttrs(r *http.Request, reason string) map[string]any {
	actor := "unknown"
	if id, ok := auth.FromRequest(r); ok && strings.TrimSpace(id.UserID) != "" {
		actor = id.UserID
	}
	return map[string]any{
		"actor":  actor,
		"reason": reason,
	}
}

func hListEvents(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			writeErr(w, http.StatusBadRequest, "bad_id", "invalid pass id")
			return
		}
		events, err := d.Store.ListEvents(r.Context(), id)
		if err != nil {
			writeInternal(w, r, d, "events_failed", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": events})
	}
}

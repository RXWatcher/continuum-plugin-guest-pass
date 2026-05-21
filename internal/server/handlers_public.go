package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	sdkruntime "github.com/ContinuumApp/continuum-plugin-sdk/pkg/pluginsdk/runtime"
	"github.com/ContinuumApp/continuum-plugin-sdk/pkg/pluginsdk/runtimehost"

	"github.com/RXWatcher/continuum-plugin-guest-pass/internal/store"
)

// hPublicPassPreview returns the pass shape without recording an "opened"
// event. The SPA uses this to decide whether to prompt for a PIN first.
func hPublicPassPreview(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := chi.URLParam(r, "token")
		p, err := d.Store.GetPassByToken(r.Context(), token)
		if err != nil {
			publicNotFound(w)
			return
		}
		status := p.Status(time.Now())
		if status != "active" {
			_ = d.Store.RecordEvent(r.Context(), p.ID, "rejected_"+status, clientIP(r), r.UserAgent(), nil)
			writeJSON(w, http.StatusForbidden, map[string]any{"status": status})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"pass": decoratePass(*p, "", publicBaseURL(r, d), time.Now())})
	}
}

// hPublicPassOpen records a guest viewing the landing page. The verify
// flow runs first; on success it registers the device and bumps the
// open counter atomically (ErrOpenLimit from the store).
func hPublicPassOpen(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := chi.URLParam(r, "token")
		p, err := d.Store.GetPassByToken(r.Context(), token)
		if err != nil {
			publicNotFound(w)
			return
		}
		status := p.Status(time.Now())
		if status != "active" {
			_ = d.Store.RecordEvent(r.Context(), p.ID, "rejected_"+status, clientIP(r), r.UserAgent(), nil)
			writeJSON(w, http.StatusForbidden, map[string]any{"status": status})
			return
		}

		req := readAccessRequest(r)
		if ok := verifyPassAccess(d, w, r, p, req, "open"); !ok {
			return
		}

		if err := d.Store.RegisterDevice(r.Context(), p.ID, req.DeviceID, clientIP(r), r.UserAgent(), p.MaxDevices); err != nil {
			if errors.Is(err, store.ErrDeviceLimit) {
				_ = d.Store.RecordEvent(r.Context(), p.ID, "rejected_device_limit_reached", clientIP(r), r.UserAgent(), nil)
				writeJSON(w, http.StatusForbidden, map[string]any{"status": "device_limit_reached"})
				return
			}
			writeInternal(w, r, d, "device_register_failed", err)
			return
		}

		updated, err := d.Store.RecordOpen(r.Context(), p.ID, clientIP(r))
		if err != nil {
			if errors.Is(err, store.ErrOpenLimit) {
				_ = d.Store.RecordEvent(r.Context(), p.ID, "rejected_open_limit_reached", clientIP(r), r.UserAgent(), nil)
				writeJSON(w, http.StatusForbidden, map[string]any{"status": "open_limit_reached"})
				return
			}
			writeInternal(w, r, d, "open_failed", err)
			return
		}
		_ = d.Store.RecordEvent(r.Context(), p.ID, "opened", clientIP(r), r.UserAgent(), nil)
		writeJSON(w, http.StatusOK, map[string]any{"pass": decoratePass(*updated, "", publicBaseURL(r, d), time.Now())})
	}
}

// hPlayAttempt validates access, atomically increments the play counter,
// reserves a concurrent-stream slot, mints a host stream grant, and
// audits the result. If the host mint fails after a slot is reserved,
// the slot is released so it doesn't count against the cap.
func hPlayAttempt(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := chi.URLParam(r, "token")
		p, err := d.Store.GetPassByToken(r.Context(), token)
		if err != nil {
			publicNotFound(w)
			return
		}
		status := p.Status(time.Now())
		if status != "active" {
			_ = d.Store.RecordEvent(r.Context(), p.ID, "play_rejected_"+status, clientIP(r), r.UserAgent(), nil)
			writeJSON(w, http.StatusForbidden, map[string]any{"status": status})
			return
		}
		req := readAccessRequest(r)
		if ok := verifyPassAccess(d, w, r, p, req, "play"); !ok {
			return
		}

		if err := d.Store.RegisterDevice(r.Context(), p.ID, req.DeviceID, clientIP(r), r.UserAgent(), p.MaxDevices); err != nil {
			if errors.Is(err, store.ErrDeviceLimit) {
				_ = d.Store.RecordEvent(r.Context(), p.ID, "play_rejected_device_limit_reached", clientIP(r), r.UserAgent(), nil)
				writeJSON(w, http.StatusForbidden, map[string]any{"status": "device_limit_reached"})
				return
			}
			writeInternal(w, r, d, "device_register_failed", err)
			return
		}

		updated, err := d.Store.RecordPlay(r.Context(), p.ID)
		if err != nil {
			if errors.Is(err, store.ErrPlayLimit) {
				_ = d.Store.RecordEvent(r.Context(), p.ID, "play_rejected_play_limit_reached", clientIP(r), r.UserAgent(), nil)
				writeJSON(w, http.StatusForbidden, map[string]any{"status": "play_limit_reached"})
				return
			}
			writeInternal(w, r, d, "play_failed", err)
			return
		}

		slotID, err := d.Store.ReserveGrantSlot(r.Context(), updated.ID, req.DeviceID, updated.EffectiveExpiresAt(), updated.MaxConcurrentStreams)
		if err != nil {
			if errors.Is(err, store.ErrConcurrencyLimit) {
				_ = d.Store.RecordEvent(r.Context(), updated.ID, "play_rejected_concurrent_stream_limit_reached", clientIP(r), r.UserAgent(), nil)
				writeJSON(w, http.StatusTooManyRequests, map[string]any{"status": "concurrent_stream_limit_reached"})
				return
			}
			writeInternal(w, r, d, "grant_reserve_failed", err)
			return
		}

		watermarkText := renderWatermarkText(updated.WatermarkProfile, updated, req, clientIP(r))
		grant, grantErr := mintPlaybackGrant(r.Context(), updated, watermarkText)
		if grantErr != nil {
			// Release the reserved slot so it doesn't count against concurrency.
			if err := d.Store.ReleaseGrantSlot(r.Context(), slotID); err != nil && d.Logger != nil {
				d.Logger.Warn("guest-pass: failed to release grant slot after mint failure", "pass_id", updated.ID, "slot_id", slotID, "err", err)
			}
			_ = d.Store.RecordEvent(r.Context(), updated.ID, "grant_failed", clientIP(r), r.UserAgent(), map[string]any{"error": grantErr.Error()})
			writeErr(w, http.StatusBadRequest, "grant_failed", grantErr.Error())
			return
		}

		_ = d.Store.RecordEvent(r.Context(), updated.ID, "grant_minted", clientIP(r), r.UserAgent(), map[string]any{
			"play_method": grant.PlayMethod,
			"expires_at":  grant.ExpiresAt.Format(time.RFC3339),
			"device_id":   req.DeviceID,
			"watermark":   watermarkText,
			"logo":        updated.WatermarkLogoURL,
		})
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      updated.Status(time.Now()),
			"pass":        decoratePass(*updated, "", publicBaseURL(r, d), time.Now()),
			"stream_url":  grant.StreamURL,
			"play_method": grant.PlayMethod,
			"expires_at":  grant.ExpiresAt,
			"watermark":   watermarkText,
			"logo_url":    updated.WatermarkLogoURL,
		})
	}
}

// verifyPassAccess runs the per-request policy gates: IP allowlist, geo
// allowlist, lock-to-first-IP, and PIN. On a legacy PIN match the stored
// hash is upgraded in the background. Returns false after writing an
// error response.
func verifyPassAccess(d Deps, w http.ResponseWriter, r *http.Request, p *store.Pass, req accessRequest, action string) bool {
	ip := clientIP(r)
	if !p.AllowsIP(ip) {
		_ = d.Store.RecordEvent(r.Context(), p.ID, action+"_rejected_ip_not_allowed", ip, r.UserAgent(), nil)
		writeJSON(w, http.StatusForbidden, map[string]any{"status": "ip_not_allowed"})
		return false
	}
	if !allowsCountry(p, requestCountry(r)) {
		_ = d.Store.RecordEvent(r.Context(), p.ID, action+"_rejected_country_not_allowed", ip, r.UserAgent(), map[string]any{"country": requestCountry(r)})
		writeJSON(w, http.StatusForbidden, map[string]any{"status": "country_not_allowed"})
		return false
	}
	if p.LockToFirstIP && p.FirstIP != "" && p.FirstIP != ip {
		_ = d.Store.RecordEvent(r.Context(), p.ID, action+"_rejected_ip_locked", ip, r.UserAgent(), map[string]any{"first_ip": p.FirstIP})
		writeJSON(w, http.StatusForbidden, map[string]any{"status": "ip_locked"})
		return false
	}
	matched, needsRehash := p.PINMatch(req.PIN)
	if !matched {
		_ = d.Store.RecordEvent(r.Context(), p.ID, action+"_rejected_bad_pin", ip, r.UserAgent(), nil)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"status": "pin_required", "message": "PIN required"})
		return false
	}
	if needsRehash {
		// Background upgrade so a slow bcrypt hash doesn't block the user.
		go func(passID int, pin string) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := d.Store.RehashPIN(ctx, passID, pin); err != nil && d.Logger != nil {
				d.Logger.Warn("guest-pass: pin rehash failed", "pass_id", passID, "err", err)
			}
		}(p.ID, req.PIN)
	}
	return true
}

// mintPlaybackGrant calls into the host SDK to mint a scoped stream
// token. The pass must have a media_file target with a numeric file id.
func mintPlaybackGrant(ctx context.Context, p *store.Pass, watermarkText string) (*runtimehost.ScopedStreamGrant, error) {
	if p.TargetType != "media_file" {
		return nil, fmt.Errorf("playback grants require target_type media_file")
	}
	fileID, err := strconv.Atoi(p.TargetID)
	if err != nil || fileID <= 0 {
		return nil, fmt.Errorf("target_id must be a media file id")
	}
	host := sdkruntime.Host()
	if host == nil {
		return nil, fmt.Errorf("host RuntimeHost is not available yet")
	}
	return host.MintScopedStream(ctx, runtimehost.ScopedStreamRequest{
		MediaFileID:         fileID,
		PlayMethod:          "auto",
		ExpiresAt:           p.EffectiveExpiresAt(),
		MaxWatchMinutes:     p.MaxWatchMinutes,
		MaxResolutionHeight: resolutionHeight(p.MaxResolution),
		AllowDirectPlay:     p.AllowDirectPlay,
		AllowDownloads:      p.AllowDownloads,
		DisableSeeking:      p.DisableSeeking,
		AuditSubject:        fmt.Sprintf("guest-pass:%d", p.ID),
		WatermarkMode:       p.WatermarkMode,
		WatermarkText:       watermarkText,
		WatermarkLogoURL:    p.WatermarkLogoURL,
	})
}

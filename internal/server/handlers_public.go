package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	sdkruntime "github.com/Silo-Server/silo-plugin-sdk/pkg/pluginsdk/runtime"
	"github.com/Silo-Server/silo-plugin-sdk/pkg/pluginsdk/runtimehost"

	"github.com/RXWatcher/silo-plugin-guest-pass/internal/store"
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
		// Withhold the pass payload (target_id, policy, watermark, limits)
		// behind the PIN gate. On a bare token the preview only reveals that
		// a PIN is required and a human-readable title so the SPA can render
		// the prompt. The full pass shape is returned by open/play once
		// verifyPassAccess succeeds.
		if p.RequirePIN {
			writeJSON(w, http.StatusOK, map[string]any{
				"require_pin": true,
				"status":      status,
				"title":       p.Title,
			})
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

		grantExpiry := cappedGrantExpiry(updated, time.Now())
		slotID, err := d.Store.ReserveGrantSlot(r.Context(), updated.ID, req.DeviceID, grantExpiry, updated.MaxConcurrentStreams)
		if err != nil {
			// The play was already counted; reverse it so a slot/grant
			// failure doesn't burn a play against MaxPlays.
			releasePlayCount(d, r, updated.ID)
			if errors.Is(err, store.ErrConcurrencyLimit) {
				_ = d.Store.RecordEvent(r.Context(), updated.ID, "play_rejected_concurrent_stream_limit_reached", clientIP(r), r.UserAgent(), nil)
				writeJSON(w, http.StatusTooManyRequests, map[string]any{"status": "concurrent_stream_limit_reached"})
				return
			}
			writeInternal(w, r, d, "grant_reserve_failed", err)
			return
		}

		watermarkText := renderWatermarkText(updated.WatermarkProfile, updated, req, clientIP(r))
		grant, grantErr := mintPlaybackGrant(r.Context(), updated, watermarkText, grantExpiry)
		if grantErr != nil {
			// Release the reserved slot so it doesn't count against
			// concurrency, and reverse the counted play.
			if err := d.Store.ReleaseGrantSlot(r.Context(), slotID); err != nil && d.Logger != nil {
				d.Logger.Warn("guest-pass: failed to release grant slot after mint failure", "pass_id", updated.ID, "slot_id", slotID, "err", err)
			}
			releasePlayCount(d, r, updated.ID)
			// Log the host detail server-side, but return an opaque message
			// to the guest so internal failure detail doesn't leak.
			if d.Logger != nil {
				d.Logger.Error("guest-pass: grant mint failed", "pass_id", updated.ID, "err", grantErr)
			}
			_ = d.Store.RecordEvent(r.Context(), updated.ID, "grant_failed", clientIP(r), r.UserAgent(), map[string]any{"error": grantErr.Error()})
			writeErr(w, http.StatusBadGateway, "grant_failed", "could not start playback")
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

	// Fail closed: an IP-dependent control (allowlist or lock-to-first-IP)
	// that is configured but cannot be evaluated because the host did not
	// stamp a client IP must DENY, not silently allow. Otherwise an empty
	// IP would bypass the restriction the operator explicitly enabled.
	if ip == "" && (len(p.IPAllowlist) > 0 || p.LockToFirstIP) {
		_ = d.Store.RecordEvent(r.Context(), p.ID, action+"_rejected_ip_unavailable", ip, r.UserAgent(), nil)
		writeJSON(w, http.StatusForbidden, map[string]any{"status": "ip_unavailable"})
		return false
	}

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

	// Brute-force backoff: temporarily lock the pass after repeated bad-PIN
	// attempts. Checked before evaluating the supplied PIN so an attacker
	// cannot keep guessing during the lockout window.
	if p.RequirePIN {
		if locked, retryAfter := pinLockout(d, r, p, ip); locked {
			_ = d.Store.RecordEvent(r.Context(), p.ID, action+"_rejected_pin_locked_out", ip, r.UserAgent(), nil)
			if retryAfter > 0 {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			}
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"status":      "pin_locked_out",
				"message":     "too many incorrect PIN attempts, try again later",
				"retry_after": retryAfter,
			})
			return false
		}
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

// grantTTLCap bounds how long a minted host grant and its reserved
// concurrency slot stay valid. The pass's EffectiveExpiresAt() can be far in
// the future, which would let an in-flight stream outlive a revocation
// (revoking a pass doesn't kill a live grant). Capping the grant TTL forces
// the client back through /play — and therefore back through the Status and
// policy checks — every few minutes, bounding the revocation/exfiltration
// window without disrupting normal playback.
const grantTTLCap = 5 * time.Minute

// cappedGrantExpiry returns the earlier of the pass's effective expiry and
// now+grantTTLCap, so a grant never outlives the policy by more than the cap.
func cappedGrantExpiry(p *store.Pass, now time.Time) time.Time {
	effective := p.EffectiveExpiresAt()
	capped := now.Add(grantTTLCap)
	if capped.Before(effective) {
		return capped
	}
	return effective
}

// releasePlayCount reverses a counted play on a failure path, logging but
// not surfacing any error (the user already gets the real failure response).
func releasePlayCount(d Deps, r *http.Request, passID int) {
	if err := d.Store.DecrementPlay(r.Context(), passID); err != nil && d.Logger != nil {
		d.Logger.Warn("guest-pass: failed to reverse play count after grant failure", "pass_id", passID, "err", err)
	}
}

// pinLockout enforces an exponential-backoff lockout against PIN
// brute-forcing. It counts recent "*_rejected_bad_pin" audit rows for the
// pass (the per-IP count when a client IP is available, otherwise the
// per-pass count) and locks out once the count crosses a threshold. The
// lockout window grows exponentially with the number of failures so a
// determined attacker is throttled hard while a legitimate guest who
// fat-fingers the PIN a couple of times is not affected.
//
// Returns (locked, retryAfterSeconds). Audit-read failures fail open for
// availability (the request still has to pass the actual PIN check), but
// are logged for visibility.
func pinLockout(d Deps, r *http.Request, p *store.Pass, ip string) (bool, int) {
	const (
		// Allow a few honest mistakes before any throttling kicks in.
		freeAttempts = 5
		// Window over which failures are counted and the base lockout unit.
		window      = 15 * time.Minute
		baseLockout = 30 * time.Second
		maxLockout  = 15 * time.Minute
	)

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	fails, err := d.Store.CountRecentBadPINs(ctx, p.ID, ip, window)
	if err != nil {
		if d.Logger != nil {
			d.Logger.Warn("guest-pass: bad-pin count failed, allowing attempt", "pass_id", p.ID, "err", err)
		}
		return false, 0
	}
	if fails < freeAttempts {
		return false, 0
	}

	// Exponential backoff: each failure past the free allowance doubles the
	// lockout, capped at maxLockout.
	over := fails - freeAttempts
	lockout := baseLockout << over // may overflow for huge values; guarded below
	if over >= 32 || lockout <= 0 || lockout > maxLockout {
		lockout = maxLockout
	}
	retryAfter := int(lockout.Seconds())
	if retryAfter < 1 {
		retryAfter = 1
	}
	return true, retryAfter
}

// mintPlaybackGrant calls into the host SDK to mint a scoped stream
// token. The pass must have a media_file target with a numeric file id.
func mintPlaybackGrant(ctx context.Context, p *store.Pass, watermarkText string, expiresAt time.Time) (*runtimehost.ScopedStreamGrant, error) {
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
		ExpiresAt:           expiresAt,
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

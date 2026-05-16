package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-hclog"

	sdkruntime "github.com/ContinuumApp/continuum-plugin-sdk/pkg/pluginsdk/runtime"
	"github.com/ContinuumApp/continuum-plugin-sdk/pkg/pluginsdk/runtimehost"

	"github.com/ContinuumApp/continuum-plugin-guest-pass/internal/auth"
	"github.com/ContinuumApp/continuum-plugin-guest-pass/internal/store"
)

type Deps struct {
	Store         *store.Store
	Logger        hclog.Logger
	PublicBaseURL string
	WebFS         fs.FS
}

type passResponse struct {
	store.Pass
	Status             string    `json:"status"`
	EffectiveExpiresAt time.Time `json:"effective_expires_at"`
	ShareURL           string    `json:"share_url,omitempty"`
}

func New(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(requireAdmin)
		r.Get("/passes", hListPasses(d))
		r.Post("/passes", hCreatePass(d))
		r.Post("/passes/{id}/revoke", hRevokePass(d))
		r.Get("/passes/{id}/events", hListEvents(d))
	})

	r.Route("/api/public", func(r chi.Router) {
		r.Get("/passes/{token}", hPublicPass(d, false))
		r.Post("/passes/{token}/open", hPublicPass(d, true))
		r.Post("/passes/{token}/play", hPlayAttempt(d))
	})

	if d.WebFS != nil {
		r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.FS(mustSub(d.WebFS, "assets")))))
		r.Get("/p/{token}", hSPA(d))
		r.Get("/admin/*", hSPA(d))
		r.Get("/admin", hSPA(d))
	}
	return r
}

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

func hListPasses(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		passes, err := d.Store.ListPasses(r.Context(), 100)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "list_failed", err.Error())
			return
		}
		now := time.Now()
		out := make([]passResponse, 0, len(passes))
		for _, p := range passes {
			out = append(out, decoratePass(p, "", d.PublicBaseURL, now))
		}
		writeJSON(w, http.StatusOK, map[string]any{"passes": out})
	}
}

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
			writeErr(w, http.StatusBadRequest, "bad_target_type", "target_type must be movie, episode, series, collection, watch_room, ebook, or audiobook")
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
			MaxOpens:                 nonNegative(req.MaxOpens),
			MaxPlays:                 nonNegative(req.MaxPlays),
			MaxWatchMinutes:          nonNegative(req.MaxWatchMinutes),
			MaxConcurrentStreams:     defaultPositive(req.MaxConcurrentStreams, 1),
			MaxDevices:               defaultPositive(req.MaxDevices, 1),
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
			writeErr(w, http.StatusInternalServerError, "create_failed", err.Error())
			return
		}
		_ = d.Store.RecordEvent(r.Context(), p.ID, "created", clientIP(r), r.UserAgent(), nil)
		writeJSON(w, http.StatusCreated, map[string]any{
			"pass":      decoratePass(*p, token, d.PublicBaseURL, time.Now()),
			"token":     token,
			"share_url": shareURL(d.PublicBaseURL, token),
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
			writeErr(w, http.StatusInternalServerError, "revoke_failed", err.Error())
			return
		}
		_ = d.Store.RecordEvent(r.Context(), id, "revoked", clientIP(r), r.UserAgent(), nil)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
			writeErr(w, http.StatusInternalServerError, "events_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": events})
	}
}

func hPublicPass(d Deps, markOpen bool) http.HandlerFunc {
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
		if markOpen {
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
				writeErr(w, http.StatusInternalServerError, "device_register_failed", err.Error())
				return
			}
			if p.MaxOpens > 0 && p.OpenCount >= p.MaxOpens {
				_ = d.Store.RecordEvent(r.Context(), p.ID, "rejected_open_limit_reached", clientIP(r), r.UserAgent(), nil)
				writeJSON(w, http.StatusForbidden, map[string]any{"status": "open_limit_reached"})
				return
			}
			p, err = d.Store.RecordOpen(r.Context(), p.ID, clientIP(r), r.UserAgent())
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "open_failed", err.Error())
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"pass": decoratePass(*p, "", d.PublicBaseURL, time.Now())})
	}
}

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
			writeErr(w, http.StatusInternalServerError, "device_register_failed", err.Error())
			return
		}
		activeGrants, err := d.Store.ActiveGrantCount(r.Context(), p.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "grant_check_failed", err.Error())
			return
		}
		if p.MaxConcurrentStreams > 0 && activeGrants >= p.MaxConcurrentStreams {
			_ = d.Store.RecordEvent(r.Context(), p.ID, "play_rejected_concurrent_stream_limit_reached", clientIP(r), r.UserAgent(), map[string]any{"active_grants": activeGrants})
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"status": "concurrent_stream_limit_reached"})
			return
		}
		p, err = d.Store.RecordPlay(r.Context(), p.ID, clientIP(r), r.UserAgent())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "play_failed", err.Error())
			return
		}
		watermarkText := renderWatermarkText(p.WatermarkProfile, p, req, clientIP(r))
		grant, grantErr := mintPlaybackGrant(r, p, watermarkText)
		if grantErr != nil {
			_ = d.Store.RecordEvent(r.Context(), p.ID, "grant_failed", clientIP(r), r.UserAgent(), map[string]any{"error": grantErr.Error()})
			writeErr(w, http.StatusBadRequest, "grant_failed", grantErr.Error())
			return
		}
		if err := d.Store.RecordGrant(r.Context(), p.ID, req.DeviceID, grant.ExpiresAt); err != nil {
			_ = d.Store.RecordEvent(r.Context(), p.ID, "grant_record_failed", clientIP(r), r.UserAgent(), map[string]any{"error": err.Error()})
			writeErr(w, http.StatusInternalServerError, "grant_record_failed", err.Error())
			return
		}
		_ = d.Store.RecordEvent(r.Context(), p.ID, "grant_minted", clientIP(r), r.UserAgent(), map[string]any{
			"play_method": grant.PlayMethod,
			"expires_at":  grant.ExpiresAt.Format(time.RFC3339),
			"device_id":   req.DeviceID,
			"watermark":   watermarkText,
			"logo":        p.WatermarkLogoURL,
		})
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      p.Status(time.Now()),
			"pass":        decoratePass(*p, "", d.PublicBaseURL, time.Now()),
			"stream_url":  grant.StreamURL,
			"play_method": grant.PlayMethod,
			"expires_at":  grant.ExpiresAt,
			"watermark":   watermarkText,
			"logo_url":    p.WatermarkLogoURL,
		})
	}
}

func mintPlaybackGrant(r *http.Request, p *store.Pass, watermarkText string) (*runtimehost.ScopedStreamGrant, error) {
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
	return host.MintScopedStream(r.Context(), runtimehost.ScopedStreamRequest{
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
	if !p.PINMatches(req.PIN) {
		_ = d.Store.RecordEvent(r.Context(), p.ID, action+"_rejected_bad_pin", ip, r.UserAgent(), nil)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"status": "pin_required", "message": "PIN required"})
		return false
	}
	return true
}

func hSPA(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := loadIndex(d.WebFS)
		if err != nil {
			http.Error(w, "spa not available", http.StatusServiceUnavailable)
			return
		}
		baseHref := pluginBaseHref(r.URL.Path)
		body = []byte(strings.Replace(string(body), "<head>", `<head><base href="`+baseHref+`">`, 1))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(body)
	}
}

func loadIndex(webFS fs.FS) ([]byte, error) {
	f, err := webFS.Open("index.html")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func pluginBaseHref(path string) string {
	const marker = "/api/v1/plugins/"
	i := strings.Index(path, marker)
	if i < 0 {
		return "/"
	}
	rest := path[i+len(marker):]
	j := strings.IndexByte(rest, '/')
	if j < 0 {
		return path + "/"
	}
	return path[:i+len(marker)+j] + "/"
}

func mustSub(webFS fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(webFS, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

func decoratePass(p store.Pass, token, publicBaseURL string, now time.Time) passResponse {
	return passResponse{
		Pass:               p,
		Status:             p.Status(now),
		EffectiveExpiresAt: p.EffectiveExpiresAt(),
		ShareURL:           shareURL(publicBaseURL, token),
	}
}

func shareURL(publicBaseURL, token string) string {
	if token == "" {
		return ""
	}
	path := "/p/" + token
	if publicBaseURL == "" {
		return path
	}
	return strings.TrimRight(publicBaseURL, "/") + path
}

func parseExpiry(raw string, hours int) (time.Time, error) {
	if raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}, fmt.Errorf("expires_at must be RFC3339")
		}
		if !t.After(time.Now()) {
			return time.Time{}, fmt.Errorf("expires_at must be in the future")
		}
		return t, nil
	}
	if hours <= 0 {
		hours = 24
	}
	if hours > 24*365 {
		return time.Time{}, fmt.Errorf("expires_in_hours cannot exceed one year")
	}
	return time.Now().Add(time.Duration(hours) * time.Hour), nil
}

func validTargetType(v string) bool {
	switch v {
	case "media_file", "movie", "episode", "series", "collection", "watch_room", "ebook", "audiobook":
		return true
	default:
		return false
	}
}

func resolutionHeight(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "480p":
		return 480
	case "720p":
		return 720
	case "1080p":
		return 1080
	case "4k", "2160p":
		return 2160
	default:
		return 0
	}
}

func normalizeDeviceID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 128 {
		value = value[:128]
	}
	return value
}

func fallbackDeviceID(r *http.Request) string {
	sum := sha256.Sum256([]byte(clientIP(r) + "\n" + r.UserAgent()))
	return "derived-" + hex.EncodeToString(sum[:16])
}

func requestCountry(r *http.Request) string {
	for _, header := range []string{"CF-IPCountry", "X-Geo-Country", "X-Country-Code"} {
		if value := strings.ToUpper(strings.TrimSpace(r.Header.Get(header))); value != "" {
			return value
		}
	}
	return ""
}

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

func clientIP(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); v != "" {
		if i := strings.IndexByte(v, ','); i >= 0 {
			return strings.TrimSpace(v[:i])
		}
		return v
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func limitSuffix(limit int) string {
	if limit <= 0 {
		return " / unlimited"
	}
	return fmt.Sprintf(" / %d", limit)
}

func nonNegative(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func defaultPositive(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func normalizeWatermarkMode(mode, profile string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		if strings.TrimSpace(profile) == "" {
			return "none"
		}
		return "all"
	}
	valid := map[string]bool{
		"none": true, "visible": true, "burned_in": true, "forensic": true, "all": true,
	}
	parts := strings.FieldsFunc(mode, func(r rune) bool {
		return r == ',' || r == ' ' || r == '+'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ReplaceAll(part, "-", "_")
		if part == "burnedin" {
			part = "burned_in"
		}
		if valid[part] {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, ",")
}

func renderWatermarkText(template string, p *store.Pass, req accessRequest, ip string) string {
	template = strings.TrimSpace(template)
	if template == "" {
		if strings.TrimSpace(p.WatermarkLogoURL) != "" {
			return ""
		}
		template = "Guest pass {{pass_id}} · {{ip}} · {{time}}"
	}
	values := map[string]string{
		"{{pass_id}}":   strconv.Itoa(p.ID),
		"{{title}}":     p.Title,
		"{{ip}}":        ip,
		"{{device_id}}": req.DeviceID,
		"{{subject}}":   fmt.Sprintf("guest-pass:%d", p.ID),
		"{{time}}":      time.Now().UTC().Format(time.RFC3339),
	}
	for key, value := range values {
		template = strings.ReplaceAll(template, key, value)
	}
	if len(template) > 160 {
		template = template[:160]
	}
	return template
}

func splitList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func publicNotFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]any{"status": "not_found"})
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": msg}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

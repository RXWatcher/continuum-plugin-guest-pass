package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/RXWatcher/silo-plugin-guest-pass/internal/store"
)

// passResponse decorates a Pass with derived fields the UI uses.
type passResponse struct {
	store.Pass
	Status             string    `json:"status"`
	EffectiveExpiresAt time.Time `json:"effective_expires_at"`
	ShareURL           string    `json:"share_url,omitempty"`
}

// catalogItemResponse is the catalog search result projection.
type catalogItemResponse struct {
	ContentID        string   `json:"content_id"`
	MediaFileID      int      `json:"media_file_id"`
	Type             string   `json:"type"`
	Title            string   `json:"title"`
	Year             int      `json:"year,omitempty"`
	Overview         string   `json:"overview,omitempty"`
	PosterURL        string   `json:"poster_url,omitempty"`
	Genres           []string `json:"genres,omitempty"`
	RuntimeMinutes   int      `json:"runtime_minutes,omitempty"`
	ContentRating    string   `json:"content_rating,omitempty"`
	Playable         bool     `json:"playable"`
	ExternalID       string   `json:"external_id,omitempty"`
	ExternalProvider string   `json:"external_provider,omitempty"`
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

// publicBaseURL reads the plugin's configured external origin, or empty
// to fall back to relative URLs.
func publicBaseURL(r *http.Request, d Deps) string {
	if d.Store == nil {
		return ""
	}
	cfg, err := d.Store.GetAppConfig(r.Context())
	if err != nil {
		return ""
	}
	return cfg.PublicBaseURL
}

func publicNotFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]any{"status": "not_found"})
}

// writeErr returns an error envelope with code+message. Use for callers
// who legitimately need to know what they did wrong (validation, missing
// resource, auth). Never pass through internal error messages.
func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": msg}})
}

// writeInternal logs the underlying error server-side (with request method
// and path for triage) and returns an opaque 500 to the client. Use this
// instead of leaking err.Error() to the wire.
func writeInternal(w http.ResponseWriter, r *http.Request, d Deps, code string, err error) {
	if d.Logger != nil {
		d.Logger.Error("guest-pass: internal error",
			"method", r.Method, "path", r.URL.Path, "code", code, "err", err)
	}
	writeErr(w, http.StatusInternalServerError, code, "internal error")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

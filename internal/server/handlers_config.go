package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func hGetConfig(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := d.Store.GetAppConfig(r.Context())
		if err != nil {
			writeInternal(w, r, d, "config_failed", err)
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	}
}

func hUpdateConfig(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cur, err := d.Store.GetAppConfig(r.Context())
		if err != nil {
			writeInternal(w, r, d, "config_failed", err)
			return
		}
		var req struct {
			PublicBaseURL     *string `json:"public_base_url"`
			AuditRetentionDay *int    `json:"audit_retention_days"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad_json", "invalid JSON body")
			return
		}
		if req.PublicBaseURL != nil {
			cur.PublicBaseURL = strings.TrimSpace(*req.PublicBaseURL)
			if cur.PublicBaseURL != "" && !validAbsoluteURL(cur.PublicBaseURL) {
				writeErr(w, http.StatusBadRequest, "bad_public_base_url", "public_base_url must be an absolute URL")
				return
			}
		}
		if req.AuditRetentionDay != nil {
			cur.AuditRetentionDay = *req.AuditRetentionDay
		}
		if err := d.Store.UpdateAppConfig(r.Context(), cur); err != nil {
			writeInternal(w, r, d, "config_failed", err)
			return
		}
		cfg, err := d.Store.GetAppConfig(r.Context())
		if err != nil {
			writeInternal(w, r, d, "config_failed", err)
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	}
}

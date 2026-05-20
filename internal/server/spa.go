package server

import (
	"html"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

// hSPA serves the embedded SPA, injecting a runtime <base href> derived
// from the plugin's mount path header, and a data-theme attribute when
// the host signals a theme. The handler is mounted on both the guest
// pass URL (/p/{token}) and the admin URL (/admin*) so the same bundle
// renders both faces.
func hSPA(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := loadIndex(d.WebFS)
		if err != nil {
			http.Error(w, "spa not available", http.StatusServiceUnavailable)
			return
		}
		baseHref := pluginBaseHref(r)
		body = []byte(strings.Replace(string(body), "<head>", `<head><base href="`+baseHref+`">`, 1))
		body = injectTheme(body, r)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(body)
	}
}

func injectTheme(body []byte, r *http.Request) []byte {
	theme := r.Header.Get("X-Continuum-Theme")
	if theme == "" {
		theme = r.URL.Query().Get("theme")
	}
	if theme == "" {
		return body
	}
	safe := html.EscapeString(theme)
	htmlBody := string(body)
	if strings.Contains(htmlBody, "<html ") {
		return []byte(strings.Replace(htmlBody, "<html ", `<html data-theme="`+safe+`" `, 1))
	}
	return []byte(strings.Replace(htmlBody, "<html>", `<html data-theme="`+safe+`">`, 1))
}

func loadIndex(webFS fs.FS) ([]byte, error) {
	f, err := webFS.Open("index.html")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// pluginBaseHref derives the SPA's <base href> from the mount-path header
// the host injects. Without it the SPA falls back to relative URLs.
func pluginBaseHref(r *http.Request) string {
	mountPath := strings.TrimRight(r.Header.Get("X-Continuum-Plugin-Mount-Path"), "/")
	if mountPath != "" {
		return mountPath + "/"
	}
	return "./"
}

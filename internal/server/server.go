// Package server is the chi-mounted HTTP handler for the guest-pass plugin.
// It composes the admin API (/api/admin/*), the public guest API
// (/api/public/*), and the embedded SPA fallback. The httproutes package
// wraps the resulting http.Handler for the SDK's HttpRoutes RPC.
//
// Files in this package:
//   - server.go         router wiring and the Deps struct
//   - middleware.go     auth + configured-store gates
//   - handlers_admin.go pass CRUD + event listing
//   - handlers_public.go guest-facing open/play flow
//   - handlers_catalog.go admin catalog search
//   - handlers_config.go runtime config read/write
//   - spa.go            embedded SPA serving + theme injection
//   - request_ctx.go    request inspection helpers (clientIP, country, device)
//   - watermark.go      watermark mode normalisation and template render
//   - response.go       wire-shape decoration plus error/JSON writers
//   - validation.go     pure input-validation helpers
package server

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-hclog"

	"github.com/ContinuumApp/continuum-plugin-guest-pass/internal/store"
)

// Deps wires the server's collaborators. WebFS may be nil during early
// dev — when it is, the SPA routes are not mounted.
type Deps struct {
	Store  *store.Store
	Logger hclog.Logger
	WebFS  fs.FS
}

// New constructs the fully-composed handler.
func New(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(requireStore(d))
		r.Use(requireAdmin)
		r.Get("/passes", hListPasses(d))
		r.Post("/passes", hCreatePass(d))
		r.Post("/passes/{id}/revoke", hRevokePass(d))
		r.Get("/passes/{id}/events", hListEvents(d))
		r.Get("/catalog/search", hSearchCatalog(d))
		r.Get("/config", hGetConfig(d))
		r.Patch("/config", hUpdateConfig(d))
	})

	r.Route("/api/public", func(r chi.Router) {
		r.Use(requireStore(d))
		r.Get("/passes/{token}", hPublicPassPreview(d))
		r.Post("/passes/{token}/open", hPublicPassOpen(d))
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

func mustSub(webFS fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(webFS, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

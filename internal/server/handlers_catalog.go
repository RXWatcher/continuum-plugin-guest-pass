package server

import (
	"net/http"
	"strings"

	sdkruntime "github.com/Silo-Server/silo-plugin-sdk/pkg/pluginsdk/runtime"
	"github.com/Silo-Server/silo-plugin-sdk/pkg/pluginsdk/runtimehost"
)

// hSearchCatalog asks the host for matching media items via the SDK
// (rich metadata, no cross-schema reach), then resolves each item to a
// preferred playable file via store.PlayableFileIDs (the irreducible
// cross-schema piece — see catalog.go for the rationale).
//
// Items without a playable file are still returned with media_file_id=0
// and playable=false so the UI can grey them out instead of silently
// dropping search results.
func hSearchCatalog(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(query) < 2 {
			writeJSON(w, http.StatusOK, map[string]any{"items": []catalogItemResponse{}})
			return
		}
		limit := parseBoundedInt(r.URL.Query().Get("limit"), 20, 1, 50)
		host := sdkruntime.Host()
		if host == nil {
			writeErr(w, http.StatusServiceUnavailable, "host_unavailable", "host runtime is not ready")
			return
		}
		searchResp, err := host.ListLibraryMedia(r.Context(), runtimehost.ListLibraryMediaRequest{
			MediaTypes: catalogMediaTypes(r.URL.Query().Get("type")),
			Query:      query,
			PageSize:   limit,
		})
		if err != nil {
			writeInternal(w, r, d, "catalog_search_failed", err)
			return
		}
		mediaIDs := make([]string, 0, len(searchResp.Items))
		for _, it := range searchResp.Items {
			if it.MediaID != "" {
				mediaIDs = append(mediaIDs, it.MediaID)
			}
		}
		fileIDs, err := d.Store.PlayableFileIDs(r.Context(), mediaIDs)
		if err != nil {
			writeInternal(w, r, d, "catalog_resolve_failed", err)
			return
		}

		items := make([]catalogItemResponse, 0, len(searchResp.Items))
		for _, it := range searchResp.Items {
			fileID := fileIDs[it.MediaID]
			items = append(items, catalogItemResponse{
				ContentID:        it.MediaID,
				MediaFileID:      fileID,
				Type:             it.MediaType,
				Title:            it.Title,
				Year:             it.Year,
				Overview:         it.Overview,
				PosterURL:        it.PosterURL,
				Genres:           it.Genres,
				RuntimeMinutes:   it.RuntimeMinutes,
				ContentRating:    it.ContentRating,
				Playable:         fileID > 0,
				ExternalID:       it.ExternalID,
				ExternalProvider: it.ExternalProvider,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items": items,
			"total": len(items),
		})
	}
}

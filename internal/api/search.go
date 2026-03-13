package api

import (
	"log/slog"
	"net/http"
	"strings"

	"md/internal/storage"
)

// searchHandler provides full-text search across all stored markdown files.
type searchHandler struct {
	store *storage.Storage
}

func newSearchHandler(store *storage.Storage) *searchHandler {
	return &searchHandler{store: store}
}

// GET /api/search?q=term&path=folder
//
// Performs a case-insensitive search in both file names and content.
// Returns up to 50 matching results with contextual snippets.
func (h *searchHandler) search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	pathFilter := strings.TrimSpace(r.URL.Query().Get("path"))

	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	if len(query) > 500 {
		writeError(w, http.StatusBadRequest, "query too long (max 500 characters)")
		return
	}

	results, err := h.store.Search(query)
	if err != nil {
		slog.Error("search failed", "query", query, "error", err)
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}

	// Apply optional path filter
	if pathFilter != "" {
		filtered := make([]storage.SearchResult, 0, len(results))
		for _, sr := range results {
			if strings.HasPrefix(sr.Path, pathFilter) || sr.Path == pathFilter {
				filtered = append(filtered, sr)
			}
		}
		results = filtered
	}

	if results == nil {
		results = []storage.SearchResult{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"query":   query,
		"path":    pathFilter,
		"results": results,
		"count":   len(results),
	})
}

package api

import (
	"net/http"

	"devhub/internal/search"
)

// SearchHandlers manages REST endpoints for file search.
type SearchHandlers struct {
	Base   *Handlers
	Search *search.SearchService
}

// FileSearch handles GET /api/projects/{id}/files/search?q=...&glob=...
func (sh *SearchHandlers) FileSearch(w http.ResponseWriter, r *http.Request) {
	projectPath, err := sh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		jsonError(w, "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	glob := r.URL.Query().Get("glob")

	results, err := sh.Search.Search(projectPath, query, glob)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, results)
}

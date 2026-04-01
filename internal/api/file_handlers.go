package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// readmeNames lists files to look for in order of priority.
var readmeNames = []string{
	"README.md",
	"readme.md",
	"Readme.md",
	"README",
	"README.txt",
}

// Directories to skip when scanning for markdown files.
var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	".git":         true,
	".claude":      true,
	".superpowers": true,
	"dist":         true,
	"build":        true,
	"__pycache__":  true,
	".venv":        true,
	"target":       true,
}

// GetReadme handles GET /api/projects/{id}/readme
func (h *Handlers) GetReadme(w http.ResponseWriter, r *http.Request) {
	projectPath, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, name := range readmeNames {
		path := filepath.Join(projectPath, name)
		data, err := os.ReadFile(path)
		if err == nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write(data)
			return
		}
	}

	jsonError(w, "README not found", http.StatusNotFound)
}

// ListMarkdownFiles handles GET /api/projects/{id}/markdown
// Returns a JSON array of relative paths to all .md files in the project.
func (h *Handlers) ListMarkdownFiles(w http.ResponseWriter, r *http.Request) {
	projectPath, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var files []string
	filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			rel, _ := filepath.Rel(projectPath, path)
			files = append(files, rel)
		}
		return nil
	})

	if files == nil {
		files = []string{}
	}
	jsonResponse(w, files)
}

// GetMarkdownFile handles GET /api/projects/{id}/markdown/{path:.*}
// Returns the content of a specific .md file.
func (h *Handlers) GetMarkdownFile(w http.ResponseWriter, r *http.Request) {
	projectPath, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	mdPath := mux.Vars(r)["path"]
	if mdPath == "" {
		jsonError(w, "path required", http.StatusBadRequest)
		return
	}

	// Security: prevent path traversal
	clean := filepath.Clean(mdPath)
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		jsonError(w, "invalid path", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(projectPath, clean)

	// Ensure the resolved path is still within the project directory
	if !strings.HasPrefix(fullPath, projectPath) {
		jsonError(w, "invalid path", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		jsonError(w, "file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

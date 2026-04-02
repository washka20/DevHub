package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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

// ToggleMarkdownCheckbox handles PUT /api/projects/{id}/markdown/{path:.*}
// Toggles a task list checkbox at the given line number.
func (h *Handlers) ToggleMarkdownCheckbox(w http.ResponseWriter, r *http.Request) {
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

	clean := filepath.Clean(mdPath)
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		jsonError(w, "invalid path", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(projectPath, clean)
	if !strings.HasPrefix(fullPath, projectPath) {
		jsonError(w, "invalid path", http.StatusBadRequest)
		return
	}

	var body struct {
		Line int `json:"line"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Line < 1 {
		jsonError(w, "line number required (1-based)", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		jsonError(w, "file not found", http.StatusNotFound)
		return
	}

	lines := strings.Split(string(data), "\n")
	idx := body.Line - 1
	if idx >= len(lines) {
		jsonError(w, fmt.Sprintf("line %d out of range (file has %d lines)", body.Line, len(lines)), http.StatusBadRequest)
		return
	}

	line := lines[idx]
	if strings.Contains(line, "- [ ]") {
		lines[idx] = strings.Replace(line, "- [ ]", "- [x]", 1)
	} else if strings.Contains(line, "- [x]") {
		lines[idx] = strings.Replace(line, "- [x]", "- [ ]", 1)
	} else if strings.Contains(line, "- [X]") {
		lines[idx] = strings.Replace(line, "- [X]", "- [ ]", 1)
	} else {
		jsonError(w, fmt.Sprintf("line %d is not a task list item", body.Line), http.StatusBadRequest)
		return
	}

	result := strings.Join(lines, "\n")
	if err := os.WriteFile(fullPath, []byte(result), 0644); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

// --- File tree / editor API ---

// Directories to skip when building the file tree.
var treeSkipDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"vendor":        true,
	"dist":          true,
	"build":         true,
	".superpowers":  true,
	".claude":       true,
	".idea":         true,
	".vscode":       true,
	".worktrees":    true,
}

// FileNode represents a node in the project file tree.
type FileNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	IsDir    bool       `json:"is_dir"`
	Children []FileNode `json:"children,omitempty"`
}

// safePath resolves a relative path inside projectDir and guards against path traversal.
func (h *Handlers) safePath(projectDir, relPath string) (string, error) {
	full := filepath.Join(projectDir, filepath.Clean(relPath))
	full, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	absProject, _ := filepath.Abs(projectDir)
	if !strings.HasPrefix(full, absProject+string(filepath.Separator)) && full != absProject {
		return "", fmt.Errorf("path traversal detected")
	}
	return full, nil
}

// buildTree recursively builds a FileNode tree, skipping hidden dirs and stopping at maxDepth.
func buildTree(root, dir string, depth int) []FileNode {
	if depth > 10 {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	// Sort: dirs first, then alphabetical within each group.
	sort.Slice(entries, func(i, j int) bool {
		iDir := entries[i].IsDir()
		jDir := entries[j].IsDir()
		if iDir != jDir {
			return iDir
		}
		return entries[i].Name() < entries[j].Name()
	})

	var nodes []FileNode
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			// Skip known heavy/meta dirs and any hidden dir (starting with '.')
			if treeSkipDirs[name] || strings.HasPrefix(name, ".") {
				continue
			}
		}

		fullPath := filepath.Join(dir, name)
		rel, _ := filepath.Rel(root, fullPath)

		node := FileNode{
			Name:  name,
			Path:  rel,
			IsDir: e.IsDir(),
		}

		if e.IsDir() {
			node.Children = buildTree(root, fullPath, depth+1)
			if node.Children == nil {
				node.Children = []FileNode{}
			}
		}

		nodes = append(nodes, node)
	}

	return nodes
}

// FileTree handles GET /api/projects/{id}/files/tree
func (h *Handlers) FileTree(w http.ResponseWriter, r *http.Request) {
	projectPath, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	nodes := buildTree(projectPath, projectPath, 0)
	if nodes == nil {
		nodes = []FileNode{}
	}
	jsonResponse(w, nodes)
}

// FileContent handles GET /api/projects/{id}/files/content/{path:.*}
func (h *Handlers) FileContent(w http.ResponseWriter, r *http.Request) {
	projectPath, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	relPath := mux.Vars(r)["path"]
	if relPath == "" {
		jsonError(w, "path required", http.StatusBadRequest)
		return
	}

	fullPath, err := h.safePath(projectPath, relPath)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Raw file serving (for images etc.)
	if r.URL.Query().Get("raw") == "true" {
		ext := strings.ToLower(filepath.Ext(relPath))
		contentTypes := map[string]string{
			".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
			".gif": "image/gif", ".svg": "image/svg+xml", ".webp": "image/webp",
			".ico": "image/x-icon", ".bmp": "image/bmp",
		}
		if ct, ok := contentTypes[ext]; ok {
			w.Header().Set("Content-Type", ct)
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		http.ServeFile(w, r, fullPath)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		jsonError(w, "file not found", http.StatusNotFound)
		return
	}
	if info.IsDir() {
		jsonError(w, "path is a directory", http.StatusBadRequest)
		return
	}
	const maxSize = 1 << 20 // 1 MB
	if info.Size() > maxSize {
		jsonError(w, "file too large (max 1MB)", http.StatusRequestEntityTooLarge)
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

// FileWrite handles PUT /api/projects/{id}/files/content/{path:.*}
func (h *Handlers) FileWrite(w http.ResponseWriter, r *http.Request) {
	projectPath, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	relPath := mux.Vars(r)["path"]
	if relPath == "" {
		jsonError(w, "path required", http.StatusBadRequest)
		return
	}

	fullPath, err := h.safePath(projectPath, relPath)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	content, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

// FileCreate handles POST /api/projects/{id}/files/create
func (h *Handlers) FileCreate(w http.ResponseWriter, r *http.Request) {
	projectPath, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var body struct {
		Path  string `json:"path"`
		IsDir bool   `json:"is_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		jsonError(w, "path required", http.StatusBadRequest)
		return
	}

	fullPath, err := h.safePath(projectPath, body.Path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if body.IsDir {
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		parentDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(fullPath, []byte(""), 0644); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

// FileDelete handles DELETE /api/projects/{id}/files/delete/{path:.*}
func (h *Handlers) FileDelete(w http.ResponseWriter, r *http.Request) {
	projectPath, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	relPath := mux.Vars(r)["path"]
	if relPath == "" {
		jsonError(w, "path required", http.StatusBadRequest)
		return
	}

	fullPath, err := h.safePath(projectPath, relPath)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := os.RemoveAll(fullPath); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

// FileRename handles PATCH /api/projects/{id}/files/rename/{path:.*}
func (h *Handlers) FileRename(w http.ResponseWriter, r *http.Request) {
	projectPath, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	relPath := mux.Vars(r)["path"]
	if relPath == "" {
		jsonError(w, "path required", http.StatusBadRequest)
		return
	}

	var body struct {
		NewPath string `json:"new_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.NewPath == "" {
		jsonError(w, "new_path required", http.StatusBadRequest)
		return
	}

	oldFull, err := h.safePath(projectPath, relPath)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	newFull, err := h.safePath(projectPath, body.NewPath)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	newParentDir := filepath.Dir(newFull)
	if err := os.MkdirAll(newParentDir, 0755); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.Rename(oldFull, newFull); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

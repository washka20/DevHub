package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gorilla/mux"
)

// NotesHandlers manages REST endpoints for project notes.
type NotesHandlers struct {
	Base *Handlers
}

var multipleHyphens = regexp.MustCompile(`-{2,}`)

// noteInfo represents a note in the list response.
type noteInfo struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updated_at"`
}

// notesDir returns the notes directory for a project, creating it if needed.
func notesDir(projectName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".devhub", "notes", projectName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// slugify converts a title to a filesystem-safe slug.
func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		if r == ' ' || r == '-' || r == '_' {
			return '-'
		}
		return -1
	}, s)
	s = multipleHyphens.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "untitled"
	}
	return s
}

// extractTitle reads the first # heading from markdown content.
func extractTitle(content, fallback string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return fallback
}

// validateSlug checks the slug for path traversal.
func validateSlug(slug string) bool {
	clean := filepath.Clean(slug)
	return slug != "" &&
		!strings.Contains(clean, "/") &&
		!strings.Contains(clean, "\\") &&
		!strings.HasPrefix(clean, "..")
}

// validateProjectName checks the project name for path traversal.
func validateProjectName(name string) bool {
	clean := filepath.Clean(name)
	return name != "" &&
		!strings.Contains(clean, "/") &&
		!strings.Contains(clean, "\\") &&
		!strings.HasPrefix(clean, "..")
}

// ListNotes handles GET /api/projects/{id}/notes
func (nh *NotesHandlers) ListNotes(w http.ResponseWriter, r *http.Request) {
	projectName := mux.Vars(r)["id"]
	if !validateProjectName(projectName) {
		jsonError(w, "invalid project name", http.StatusBadRequest)
		return
	}
	dir, err := notesDir(projectName)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		jsonResponse(w, []noteInfo{})
		return
	}

	var notes []noteInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), ".md")
		info, err := entry.Info()
		if err != nil {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		title := slug
		if err == nil {
			title = extractTitle(string(content), slug)
		}
		notes = append(notes, noteInfo{
			Slug:      slug,
			Title:     title,
			UpdatedAt: info.ModTime().Format(time.RFC3339),
		})
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].UpdatedAt > notes[j].UpdatedAt
	})

	if notes == nil {
		notes = []noteInfo{}
	}
	jsonResponse(w, notes)
}

// GetNote handles GET /api/projects/{id}/notes/{slug}
func (nh *NotesHandlers) GetNote(w http.ResponseWriter, r *http.Request) {
	projectName := mux.Vars(r)["id"]
	if !validateProjectName(projectName) {
		jsonError(w, "invalid project name", http.StatusBadRequest)
		return
	}
	slug := mux.Vars(r)["slug"]
	if !validateSlug(slug) {
		jsonError(w, "invalid slug", http.StatusBadRequest)
		return
	}

	dir, err := notesDir(projectName)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := os.ReadFile(filepath.Join(dir, slug+".md"))
	if err != nil {
		jsonError(w, "note not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

// CreateNote handles POST /api/projects/{id}/notes
func (nh *NotesHandlers) CreateNote(w http.ResponseWriter, r *http.Request) {
	projectName := mux.Vars(r)["id"]
	if !validateProjectName(projectName) {
		jsonError(w, "invalid project name", http.StatusBadRequest)
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
		jsonError(w, "title required", http.StatusBadRequest)
		return
	}

	dir, err := notesDir(projectName)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slug := slugify(body.Title)
	filePath := filepath.Join(dir, slug+".md")

	if _, err := os.Stat(filePath); err == nil {
		found := false
		for i := 2; i < 100; i++ {
			candidate := filepath.Join(dir, slug+"-"+strconv.Itoa(i)+".md")
			if _, err := os.Stat(candidate); os.IsNotExist(err) {
				slug = slug + "-" + strconv.Itoa(i)
				filePath = candidate
				found = true
				break
			}
		}
		if !found {
			jsonError(w, "too many notes with this title", http.StatusConflict)
			return
		}
	}

	content := "# " + body.Title + "\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, map[string]string{"slug": slug})
}

// UpdateNote handles PUT /api/projects/{id}/notes/{slug}
func (nh *NotesHandlers) UpdateNote(w http.ResponseWriter, r *http.Request) {
	projectName := mux.Vars(r)["id"]
	if !validateProjectName(projectName) {
		jsonError(w, "invalid project name", http.StatusBadRequest)
		return
	}
	slug := mux.Vars(r)["slug"]
	if !validateSlug(slug) {
		jsonError(w, "invalid slug", http.StatusBadRequest)
		return
	}

	dir, err := notesDir(projectName)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filePath := filepath.Join(dir, slug+".md")

	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			jsonError(w, "note not found", http.StatusNotFound)
		} else {
			jsonError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	defer f.Close()
	if _, err := f.WriteString(body.Content); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

// DeleteNote handles DELETE /api/projects/{id}/notes/{slug}
func (nh *NotesHandlers) DeleteNote(w http.ResponseWriter, r *http.Request) {
	projectName := mux.Vars(r)["id"]
	if !validateProjectName(projectName) {
		jsonError(w, "invalid project name", http.StatusBadRequest)
		return
	}
	slug := mux.Vars(r)["slug"]
	if !validateSlug(slug) {
		jsonError(w, "invalid slug", http.StatusBadRequest)
		return
	}

	dir, err := notesDir(projectName)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filePath := filepath.Join(dir, slug+".md")
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			jsonError(w, "note not found", http.StatusNotFound)
		} else {
			jsonError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"devhub/internal/makefile"
	"devhub/internal/scanner"

	"github.com/gorilla/mux"
)

// Handlers holds shared state for all domain handler structs.
type Handlers struct {
	ProjectsDir string
	Hub         *Hub
	mu          sync.RWMutex
	projects    []scanner.Project
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(projectsDir string, hub *Hub) *Handlers {
	return &Handlers{
		ProjectsDir: projectsDir,
		Hub:         hub,
	}
}

// RefreshProjects rescans the projects directory and caches results.
func (h *Handlers) RefreshProjects() {
	projects, err := scanner.Scan(h.ProjectsDir)
	if err == nil {
		h.mu.Lock()
		h.projects = projects
		h.mu.Unlock()
	}
}

// --- helpers ---

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// projectPath resolves the project {id} to its actual directory path from cached scan.
func (h *Handlers) projectPath(r *http.Request) (string, error) {
	id := mux.Vars(r)["id"]
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.projects {
		if p.Name == id {
			return p.Path, nil
		}
	}
	return "", fmt.Errorf("project %q not found", id)
}

// composeFilePath returns the docker-compose file path for a project, or error.
func composeFilePath(projectPath string) (string, error) {
	p := scanner.FindComposeFile(projectPath)
	if p == "" {
		return "", fmt.Errorf("docker-compose file not found")
	}
	return p, nil
}

// --- Project endpoints ---

// ListProjects handles GET /api/projects
func (h *Handlers) ListProjects(w http.ResponseWriter, r *http.Request) {
	h.RefreshProjects()
	h.mu.RLock()
	projects := h.projects
	h.mu.RUnlock()
	jsonResponse(w, projects)
}

// GetProject handles GET /api/projects/{id}
func (h *Handlers) GetProject(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.projects {
		if p.Name == id {
			jsonResponse(w, p)
			return
		}
	}
	jsonError(w, "project not found", http.StatusNotFound)
}

// --- Exec endpoints ---

// ExecHandlers manages REST endpoints for Makefile command execution.
type ExecHandlers struct {
	Base *Handlers
}

// ListCommands handles GET /api/projects/{id}/commands
func (eh *ExecHandlers) ListCommands(w http.ResponseWriter, r *http.Request) {
	path, err := eh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	makefilePath := filepath.Join(path, "Makefile")
	commands, err := makefile.Parse(makefilePath)
	if err != nil {
		jsonError(w, "makefile not found or unreadable", http.StatusNotFound)
		return
	}
	jsonResponse(w, commands)
}

// ExecCommand handles POST /api/projects/{id}/exec
func (eh *ExecHandlers) ExecCommand(w http.ResponseWriter, r *http.Request) {
	path, err := eh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var body struct {
		Cmd string `json:"cmd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Cmd == "" {
		jsonError(w, "invalid request: cmd required", http.StatusBadRequest)
		return
	}

	makefilePath := filepath.Join(path, "Makefile")
	commands, err := makefile.Parse(makefilePath)
	if err != nil {
		jsonError(w, "makefile not found", http.StatusNotFound)
		return
	}

	allowed := false
	for _, c := range commands {
		if c.Name == body.Cmd {
			allowed = true
			break
		}
	}
	if !allowed {
		jsonError(w, "command not found in Makefile targets", http.StatusForbidden)
		return
	}

	projectName := mux.Vars(r)["id"]
	cmdName := body.Cmd

	outputChan, errChan := execMake(r.Context(), path, cmdName)

	go func() {
		for line := range outputChan {
			eh.Base.Hub.Broadcast(Event{
				Type:    "exec_output",
				Project: projectName,
				Cmd:     cmdName,
				Data:    line,
			})
		}

		exitCode := 0
		execErr := <-errChan
		if execErr != nil {
			exitCode = 1
			log.Printf("exec %s/%s failed: %v", projectName, cmdName, execErr)
		}
		eh.Base.Hub.Broadcast(Event{
			Type:    "exec_done",
			Project: projectName,
			Cmd:     cmdName,
			Data:    map[string]interface{}{"exit_code": exitCode},
		})
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "started", "cmd": cmdName})
}

// execMake is a wrapper to allow testing; uses executor package.
func execMake(ctx context.Context, dir, target string) (chan string, chan error) {
	return execMakeImpl(ctx, dir, target)
}

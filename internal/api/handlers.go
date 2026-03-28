package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"devhub/internal/docker"
	"devhub/internal/git"
	"devhub/internal/makefile"
	"devhub/internal/scanner"

	"github.com/gorilla/mux"
)

// Handlers holds dependencies for API request handlers.
type Handlers struct {
	ProjectsDir string
	Hub         *Hub
	projects    []scanner.Project // cached project list
	Git         *git.GitService
	Docker      *docker.DockerService
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(projectsDir string, hub *Hub, gitSvc *git.GitService, dockerSvc *docker.DockerService) *Handlers {
	return &Handlers{
		ProjectsDir: projectsDir,
		Hub:         hub,
		Git:         gitSvc,
		Docker:      dockerSvc,
	}
}

// RefreshProjects rescans the projects directory and caches results.
func (h *Handlers) RefreshProjects() {
	projects, err := scanner.Scan(h.ProjectsDir)
	if err == nil {
		h.projects = projects
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
	// Look up in cached projects (handles nested dirs like poop/status-online)
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
	jsonResponse(w, h.projects)
}

// GetProject handles GET /api/projects/{id}
func (h *Handlers) GetProject(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	for _, p := range h.projects {
		if p.Name == id {
			jsonResponse(w, p)
			return
		}
	}
	jsonError(w, "project not found", http.StatusNotFound)
}

// ListCommands handles GET /api/projects/{id}/commands
func (h *Handlers) ListCommands(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
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
// SECURITY: Only allows execution of targets defined in the project's Makefile.
// Returns 202 Accepted immediately and streams output via WebSocket.
func (h *Handlers) ExecCommand(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
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

	// Validate that the command is an actual Makefile target
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

	// Execute make target in a goroutine and stream output via WebSocket
	ctx := context.Background()
	outputChan, errChan := execMake(ctx, path, cmdName)

	go func() {
		for line := range outputChan {
			h.Hub.Broadcast(Event{
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
		h.Hub.Broadcast(Event{
			Type:    "exec_done",
			Project: projectName,
			Cmd:     cmdName,
			Data:    map[string]interface{}{"exit_code": exitCode},
		})
	}()

	// Return 202 Accepted immediately -- output streams via WebSocket
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "started", "cmd": cmdName})
}

// execMake is a wrapper to allow testing; uses executor package.
func execMake(ctx context.Context, dir, target string) (chan string, chan error) {
	return execMakeImpl(ctx, dir, target)
}

// --- Git endpoints ---

// GitStatus handles GET /api/projects/{id}/git/status
func (h *Handlers) GitStatus(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := h.Git.Status(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, status)
}

// GitBranches handles GET /api/projects/{id}/git/branches
func (h *Handlers) GitBranches(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	branches, err := h.Git.BranchesDetailed(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, branches)
}

// GitLog handles GET /api/projects/{id}/git/log
func (h *Handlers) GitLog(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	commits, err := h.Git.LogWithGraph(path, limit, offset)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, commits)
}

// GitGraph handles GET /api/projects/{id}/git/graph
func (h *Handlers) GitGraph(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	topology, err := h.Git.LogTopology(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := git.BuildFullGraph(topology)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, result)
}

// GitLogMetadata handles GET /api/projects/{id}/git/log/metadata
func (h *Handlers) GitLogMetadata(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	metas, err := h.Git.LogMetadata(path, limit, offset)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, metas)
}

// GitDiff handles GET /api/projects/{id}/git/diff
func (h *Handlers) GitDiff(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	file := r.URL.Query().Get("file")
	var diff string
	if file != "" {
		diff, err = h.Git.DiffFile(path, file)
	} else {
		diff, err = h.Git.Diff(path)
	}
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"diff": diff})
}

// GitCommit handles POST /api/projects/{id}/git/commit
func (h *Handlers) GitCommit(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var body struct {
		Message string   `json:"message"`
		Files   []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" || len(body.Files) == 0 {
		jsonError(w, "message and files required", http.StatusBadRequest)
		return
	}

	if err := h.Git.CommitChanges(path, body.Message, body.Files); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Hub.Broadcast(Event{
		Type:    "git_changed",
		Project: mux.Vars(r)["id"],
		Data:    "commit",
	})

	jsonResponse(w, map[string]string{"status": "ok"})
}

// GitStage handles POST /api/projects/{id}/git/stage
func (h *Handlers) GitStage(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	var body struct {
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Files) == 0 {
		jsonError(w, "files required", http.StatusBadRequest)
		return
	}
	if err := h.Git.StageFiles(path, body.Files); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "ok"})
}

// GitUnstage handles POST /api/projects/{id}/git/unstage
func (h *Handlers) GitUnstage(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	var body struct {
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Files) == 0 {
		jsonError(w, "files required", http.StatusBadRequest)
		return
	}
	if err := h.Git.UnstageFiles(path, body.Files); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "ok"})
}

// GitCheckout handles POST /api/projects/{id}/git/checkout
func (h *Handlers) GitCheckout(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var body struct {
		Branch string `json:"branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Branch == "" {
		jsonError(w, "branch required", http.StatusBadRequest)
		return
	}

	if err := h.Git.Checkout(path, body.Branch); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Hub.Broadcast(Event{
		Type:    "git_changed",
		Project: mux.Vars(r)["id"],
		Data:    "checkout",
	})

	jsonResponse(w, map[string]string{"status": "ok"})
}

// GitPull handles POST /api/projects/{id}/git/pull
func (h *Handlers) GitPull(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := h.Git.Pull(path)
	if err != nil {
		jsonError(w, fmt.Sprintf("%s: %s", err.Error(), out), http.StatusInternalServerError)
		return
	}

	h.Hub.Broadcast(Event{
		Type:    "git_changed",
		Project: mux.Vars(r)["id"],
		Data:    "pull",
	})

	jsonResponse(w, map[string]string{"status": "ok", "output": out})
}

// GitPush handles POST /api/projects/{id}/git/push
func (h *Handlers) GitPush(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := h.Git.Push(path)
	if err != nil {
		jsonError(w, fmt.Sprintf("%s: %s", err.Error(), out), http.StatusInternalServerError)
		return
	}

	h.Hub.Broadcast(Event{
		Type:    "git_changed",
		Project: mux.Vars(r)["id"],
		Data:    "push",
	})

	jsonResponse(w, map[string]string{"status": "ok", "output": out})
}

// GitGenerateCommit handles POST /api/projects/{id}/git/generate-commit
// Accepts optional {"files": [...]} to stage before generating.
func (h *Handlers) GitGenerateCommit(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Optionally stage files first
	var body struct {
		Files []string `json:"files"`
	}
	json.NewDecoder(r.Body).Decode(&body) // ignore error -- body is optional

	if len(body.Files) > 0 {
		if err := h.Git.StageFiles(path, body.Files); err != nil {
			jsonError(w, "failed to stage files: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	message, err := h.Git.GenerateCommitMessage(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"message": message})
}

// GitCommitDetail handles GET /api/projects/{id}/git/commits/{hash}
func (h *Handlers) GitCommitDetail(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash := mux.Vars(r)["hash"]
	if hash == "" {
		jsonError(w, "commit hash required", http.StatusBadRequest)
		return
	}

	detail, err := h.Git.CommitDetail(path, hash)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, detail)
}

// GitCommitDiff handles GET /api/projects/{id}/git/commits/{hash}/diff
func (h *Handlers) GitCommitDiff(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash := mux.Vars(r)["hash"]
	if hash == "" {
		jsonError(w, "commit hash required", http.StatusBadRequest)
		return
	}

	file := r.URL.Query().Get("file")

	diff, err := h.Git.CommitDiff(path, hash, file)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"diff": diff})
}

// --- Docker endpoints ---

// DockerContainers handles GET /api/projects/{id}/docker/containers
func (h *Handlers) DockerContainers(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	composePath, err := composeFilePath(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	containers, err := h.Docker.Containers(composePath)
	if err != nil {
		log.Printf("docker containers error: %v", err)
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, containers)
}

// DockerAction handles POST /api/projects/{id}/docker/{name}/{action}
func (h *Handlers) DockerAction(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	containerName := vars["name"]
	action := vars["action"]

	composePath, err := composeFilePath(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := h.Docker.Action(composePath, containerName, action); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Hub.Broadcast(Event{
		Type:    "container_status",
		Project: vars["id"],
		Data:    map[string]string{"name": containerName, "action": action},
	})

	jsonResponse(w, map[string]string{"status": "ok"})
}

// DockerLogs handles GET /api/projects/{id}/docker/{name}/logs
// It streams container logs via Server-Sent Events (SSE).
func (h *Handlers) DockerLogs(w http.ResponseWriter, r *http.Request) {
	path, err := h.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	containerName := vars["name"]

	// Validate container name -- only allow alphanumeric, dash, underscore, dot
	for _, c := range containerName {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
			jsonError(w, "invalid container name", http.StatusBadRequest)
			return
		}
	}

	composePath, err := composeFilePath(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	// Check that the response writer supports flushing (required for SSE)
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering if proxied

	// Create a context that cancels when the client disconnects
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	lines, errCh := h.Docker.StreamLogs(ctx, composePath, containerName, 100)

	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-lines:
			if !ok {
				// Channel closed -- check if there was an error
				if streamErr := <-errCh; streamErr != nil {
					errMsg := streamErr.Error()
					// Context cancellation is expected (client disconnect)
					if !strings.Contains(errMsg, "context canceled") &&
						!strings.Contains(errMsg, "signal: killed") {
						log.Printf("docker logs stream error for %s: %v", containerName, streamErr)
					}
				}
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		}
	}
}

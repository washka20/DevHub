package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"devhub/internal/docker"
	"devhub/internal/terminal"

	"github.com/gorilla/mux"
)

// DockerHandlers manages REST endpoints for Docker operations.
type DockerHandlers struct {
	Base        *Handlers
	Docker      *docker.DockerService
	TermManager *terminal.Manager
}

// DockerStats handles GET /api/projects/{id}/docker/stats
func (dh *DockerHandlers) DockerStats(w http.ResponseWriter, r *http.Request) {
	path, err := dh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	composePath, err := composeFilePath(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	stats, err := dh.Docker.Stats(composePath)
	if err != nil {
		log.Printf("docker stats error: %v", err)
		jsonError(w, "stats failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, stats)
}

// DockerContainers handles GET /api/projects/{id}/docker/containers
func (dh *DockerHandlers) DockerContainers(w http.ResponseWriter, r *http.Request) {
	path, err := dh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	composePath, err := composeFilePath(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	containers, err := dh.Docker.Containers(composePath)
	if err != nil {
		log.Printf("docker containers error: %v", err)
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, containers)
}

// DockerInspect handles GET /api/projects/{id}/docker/{name}/inspect
func (dh *DockerHandlers) DockerInspect(w http.ResponseWriter, r *http.Request) {
	path, err := dh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	composePath, err2 := composeFilePath(path)
	if err2 != nil {
		jsonError(w, err2.Error(), http.StatusNotFound)
		return
	}
	name := mux.Vars(r)["name"]
	inspect, err := dh.Docker.Inspect(composePath, name)
	if err != nil {
		jsonError(w, "inspect failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, inspect)
}

// DockerAction handles POST /api/projects/{id}/docker/{name}/{action}
func (dh *DockerHandlers) DockerAction(w http.ResponseWriter, r *http.Request) {
	path, err := dh.Base.projectPath(r)
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

	if err := dh.Docker.Action(composePath, containerName, action); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dh.Base.Hub.Broadcast(Event{
		Type:    "container_status",
		Project: vars["id"],
		Data:    map[string]string{"name": containerName, "action": action},
	})

	jsonResponse(w, map[string]string{"status": "ok"})
}

// DockerLogs handles GET /api/projects/{id}/docker/{name}/logs
func (dh *DockerHandlers) DockerLogs(w http.ResponseWriter, r *http.Request) {
	path, err := dh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	containerName := vars["name"]

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

	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	lines, errCh := dh.Docker.StreamLogs(ctx, composePath, containerName, 100)

	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-lines:
			if !ok {
				if streamErr := <-errCh; streamErr != nil {
					errMsg := streamErr.Error()
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

// DockerComposeUp handles POST /api/projects/{id}/docker/compose/up
func (dh *DockerHandlers) DockerComposeUp(w http.ResponseWriter, r *http.Request) {
	path, err := dh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	composePath, err := composeFilePath(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	out, err := dh.Docker.ComposeUp(composePath)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dh.Base.Hub.Broadcast(Event{
		Type:    "docker:update",
		Project: mux.Vars(r)["id"],
		Data:    "compose-up",
	})

	jsonResponse(w, map[string]string{"status": "ok", "output": out})
}

// DockerComposeUpBuild handles POST /api/projects/{id}/docker/compose/up-build
func (dh *DockerHandlers) DockerComposeUpBuild(w http.ResponseWriter, r *http.Request) {
	path, err := dh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	composePath, err := composeFilePath(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	out, err := dh.Docker.ComposeUpBuild(composePath)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dh.Base.Hub.Broadcast(Event{
		Type:    "docker:update",
		Project: mux.Vars(r)["id"],
		Data:    "compose-up-build",
	})

	jsonResponse(w, map[string]string{"status": "ok", "output": out})
}

// DockerComposeDown handles POST /api/projects/{id}/docker/compose/down
func (dh *DockerHandlers) DockerComposeDown(w http.ResponseWriter, r *http.Request) {
	path, err := dh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	composePath, err := composeFilePath(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	out, err := dh.Docker.ComposeDown(composePath)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dh.Base.Hub.Broadcast(Event{
		Type:    "docker:update",
		Project: mux.Vars(r)["id"],
		Data:    "compose-down",
	})

	jsonResponse(w, map[string]string{"status": "ok", "output": out})
}

// DockerExec handles POST /api/projects/{id}/docker/{name}/exec
func (dh *DockerHandlers) DockerExec(w http.ResponseWriter, r *http.Request) {
	path, err := dh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	containerName := mux.Vars(r)["name"]

	composePath, err := composeFilePath(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	var body struct {
		Cols uint16 `json:"cols"`
		Rows uint16 `json:"rows"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Cols == 0 {
		body.Cols = 80
	}
	if body.Rows == 0 {
		body.Rows = 24
	}

	composeDir := filepath.Dir(composePath)
	composeFile := filepath.Base(composePath)

	id := generateID()
	sess, err := dh.TermManager.CreateWithCommand(
		id, composeDir, body.Cols, body.Rows,
		"docker", "compose", "-f", composeFile, "exec", containerName, "sh", "-c",
		"if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi",
	)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"session_id": sess.ID,
		"container":  containerName,
	})
}

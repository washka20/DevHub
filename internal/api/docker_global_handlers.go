package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"devhub/internal/docker"

	"github.com/gorilla/mux"
)

// DockerAllGroup is a single compose-project (or the "Standalone" bucket)
// rendered in the global Docker view. Exposed to the frontend as JSON.
type DockerAllGroup struct {
	Project    string                   `json:"project"`
	Path       string                   `json:"path"`
	Containers []docker.GlobalContainer `json:"containers"`
}

// DockerAllResponse is the payload for GET /api/docker/all.
type DockerAllResponse struct {
	Groups []DockerAllGroup `json:"groups"`
}

// DockerGlobalService is the minimum surface of docker.DockerService that the
// global handlers need. Defined here to keep the test-double shape small.
type DockerGlobalService interface {
	ListAll() ([]docker.GlobalContainer, error)
	ContainerAction(id string, action string) error
	StreamContainerLogs(ctx context.Context, id string, tail int) (<-chan string, <-chan error)
}

// DockerGlobalHandlers manages docker endpoints that are not scoped to a
// DevHub project (global "All containers" view).
type DockerGlobalHandlers struct {
	Base   *Handlers
	Docker DockerGlobalService
}

// DockerAll handles GET /api/docker/all — returns every container on the host
// grouped by compose project. Containers without a compose label go into an
// empty-named group (rendered as "Standalone" in the UI).
func (h *DockerGlobalHandlers) DockerAll(w http.ResponseWriter, r *http.Request) {
	all, err := h.Docker.ListAll()
	if err != nil {
		log.Printf("docker ps -a: %v", err)
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	groupIdx := map[string]int{}
	var groups []DockerAllGroup
	for _, c := range all {
		key := c.ComposeProj
		idx, ok := groupIdx[key]
		if !ok {
			idx = len(groups)
			groupIdx[key] = idx
			groups = append(groups, DockerAllGroup{
				Project: c.ComposeProj,
				Path:    c.ComposeDir,
			})
		}
		groups[idx].Containers = append(groups[idx].Containers, c)
	}

	jsonResponse(w, DockerAllResponse{Groups: groups})
}

// ContainerAction handles POST /api/docker/containers/{id}/{action}
// where action is one of start/stop/restart/kill/remove. Operates by
// container ID, so it works for any docker container on the host.
func (h *DockerGlobalHandlers) ContainerAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	action := vars["action"]

	if err := h.Docker.ContainerAction(id, action); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Base.Hub.Broadcast(Event{
		Type: "docker:update",
		Data: "global-action",
	})

	jsonResponse(w, map[string]string{"status": "ok"})
}

// ContainerLogs handles GET /api/docker/containers/{id}/logs — SSE stream of
// `docker logs -f` for the given container ID.
func (h *DockerGlobalHandlers) ContainerLogs(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	// Allow hex container IDs (12 or 64 chars). Reject anything suspicious.
	if len(id) < 4 || len(id) > 64 {
		jsonError(w, "invalid container id", http.StatusBadRequest)
		return
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			jsonError(w, "invalid container id", http.StatusBadRequest)
			return
		}
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

	lines, errCh := h.Docker.StreamContainerLogs(ctx, id, 100)

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
						log.Printf("global docker logs error: %v", streamErr)
					}
				}
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		}
	}
}

// Ensure docker.DockerService satisfies DockerGlobalService at compile time.
var _ DockerGlobalService = (*docker.DockerService)(nil)

// tie json package in — prevent unused import when handlers trim bodies.
var _ = json.Encoder{}

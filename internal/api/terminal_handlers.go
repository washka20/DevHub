package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"devhub/internal/config"
	"devhub/internal/terminal"

	"github.com/gorilla/mux"
)

// TerminalHandlers manages REST endpoints for terminal sessions.
type TerminalHandlers struct {
	Manager *terminal.Manager
	Cfg     *config.Config
}

type createSessionRequest struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
	CWD  string `json:"cwd"`
}

// CreateSession handles POST /api/terminal/sessions.
func (th *TerminalHandlers) CreateSession(w http.ResponseWriter, r *http.Request) {
	var body createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.Cols == 0 {
		body.Cols = 80
	}
	if body.Rows == 0 {
		body.Rows = 24
	}

	cwd := body.CWD
	if cwd == "" {
		cwd, _ = os.UserHomeDir()
	}

	shell := th.Cfg.Terminal.Shell
	if shell == "" {
		shell = os.Getenv("SHELL")
	}
	if shell == "" {
		shell = "/bin/bash"
	}

	id := generateID()
	sess, err := th.Manager.Create(id, shell, cwd, body.Cols, body.Rows)
	if err != nil {
		if errors.Is(err, terminal.ErrMaxSessions) {
			jsonError(w, err.Error(), http.StatusTooManyRequests)
		} else {
			jsonError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"session_id": sess.ID,
		"shell":      shell,
	})
}

// ListSessions handles GET /api/terminal/sessions.
func (th *TerminalHandlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	list := th.Manager.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// DestroyAllSessions handles DELETE /api/terminal/sessions (no {id}).
func (th *TerminalHandlers) DestroyAllSessions(w http.ResponseWriter, r *http.Request) {
	th.Manager.DestroyAll()
	w.WriteHeader(http.StatusNoContent)
}

// DestroySession handles DELETE /api/terminal/sessions/{id}.
func (th *TerminalHandlers) DestroySession(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if _, ok := th.Manager.Get(id); !ok {
		jsonError(w, "session not found", http.StatusNotFound)
		return
	}
	th.Manager.Destroy(id)
	w.WriteHeader(http.StatusNoContent)
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

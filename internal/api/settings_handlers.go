package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"devhub/internal/config"
)

// SettingsHandlers manages REST endpoints for application settings.
type SettingsHandlers struct {
	Cfg *config.Config
}

type settingsResponse struct {
	Port           int                      `json:"port"`
	ProjectsDir    string                   `json:"projects_dir"`
	DefaultProject string                   `json:"default_project"`
	Terminal       terminalSettingsResponse  `json:"terminal"`
}

type terminalSettingsResponse struct {
	MaxSessions int    `json:"max_sessions"`
	Shell       string `json:"shell"`
}

func settingsFromConfig(cfg *config.Config) settingsResponse {
	return settingsResponse{
		Port:           cfg.Port,
		ProjectsDir:    cfg.ProjectsDir,
		DefaultProject: cfg.DefaultProject,
		Terminal: terminalSettingsResponse{
			MaxSessions: cfg.Terminal.MaxSessions,
			Shell:       cfg.Terminal.Shell,
		},
	}
}

// GetSettings handles GET /api/settings.
func (sh *SettingsHandlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, settingsFromConfig(sh.Cfg))
}

type updateTerminalRequest struct {
	Shell       *string `json:"shell"`
	MaxSessions *int    `json:"max_sessions"`
}

type updateSettingsRequest struct {
	Port           *int                   `json:"port"`
	ProjectsDir    *string                `json:"projects_dir"`
	DefaultProject *string                `json:"default_project"`
	Terminal       *updateTerminalRequest `json:"terminal"`
}

// UpdateSettings handles PUT /api/settings.
// Applies only the fields present in the request body (partial update).
func (sh *SettingsHandlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req updateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Port != nil {
		sh.Cfg.Port = *req.Port
	}
	if req.ProjectsDir != nil {
		sh.Cfg.ProjectsDir = config.ExpandHome(*req.ProjectsDir)
	}
	if req.DefaultProject != nil {
		sh.Cfg.DefaultProject = *req.DefaultProject
	}
	if req.Terminal != nil {
		if req.Terminal.Shell != nil {
			sh.Cfg.Terminal.Shell = *req.Terminal.Shell
		}
		if req.Terminal.MaxSessions != nil {
			sh.Cfg.Terminal.MaxSessions = *req.Terminal.MaxSessions
		}
	}

	if err := sh.Cfg.Save(); err != nil {
		jsonError(w, "failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, settingsFromConfig(sh.Cfg))
}

// ListShells handles GET /api/settings/shells.
// Reads /etc/shells and returns a JSON array of available shell paths.
func (sh *SettingsHandlers) ListShells(w http.ResponseWriter, r *http.Request) {
	shells, err := readShells("/etc/shells")
	if err != nil {
		jsonError(w, "failed to read /etc/shells: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, shells)
}

// readShells parses a shells file (e.g. /etc/shells) and returns valid entries.
func readShells(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var shells []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		shells = append(shells, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return shells, nil
}

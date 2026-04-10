package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"devhub/internal/docker"
	"devhub/internal/git"
	"devhub/internal/testutil"

	"github.com/gorilla/mux"
)

// setupTestHandlers creates domain handlers with a test project directory and mock services.
func setupTestHandlers(t *testing.T, runner *testutil.MockRunner) (*Handlers, *GitHandlers, *DockerHandlers, *ExecHandlers, *mux.Router) {
	t.Helper()

	dir := t.TempDir()

	// Create a test project with git and Makefile
	projDir := filepath.Join(dir, "testproj")
	os.MkdirAll(filepath.Join(projDir, ".git"), 0755)
	os.WriteFile(filepath.Join(projDir, "Makefile"), []byte("## Build\nbuild:\n\tgo build\n\n## Test\ntest:\n\tgo test\n"), 0644)
	os.WriteFile(filepath.Join(projDir, "docker-compose.yml"), []byte("version: '3'\nservices:\n  web:\n    image: nginx\n"), 0644)

	gitSvc := git.NewGitService(runner)
	dockerSvc := docker.NewDockerService(runner)
	hub := NewHub()
	h := NewHandlers(dir, hub)
	h.RefreshProjects()

	gitH := &GitHandlers{Base: h, Git: gitSvc}
	dockerH := &DockerHandlers{Base: h, Docker: dockerSvc}
	execH := &ExecHandlers{Base: h}

	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()

	api.HandleFunc("/projects", h.ListProjects).Methods("GET")
	api.HandleFunc("/projects/{id}", h.GetProject).Methods("GET")
	api.HandleFunc("/projects/{id}/commands", execH.ListCommands).Methods("GET")
	api.HandleFunc("/projects/{id}/git/status", gitH.GitStatus).Methods("GET")
	api.HandleFunc("/projects/{id}/git/commit", gitH.GitCommit).Methods("POST")
	api.HandleFunc("/projects/{id}/git/stage", gitH.GitStage).Methods("POST")
	api.HandleFunc("/projects/{id}/docker/containers", dockerH.DockerContainers).Methods("GET")

	return h, gitH, dockerH, execH, r
}

func TestListProjects(t *testing.T) {
	runner := &testutil.MockRunner{Calls: []testutil.MockCall{}}
	_, _, _, _, router := setupTestHandlers(t, runner)

	req := httptest.NewRequest("GET", "/api/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var projects []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&projects); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(projects) == 0 {
		t.Fatal("expected at least 1 project")
	}

	found := false
	for _, p := range projects {
		if p["name"] == "testproj" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected testproj in project list")
	}
}

func TestGetProject_Found(t *testing.T) {
	runner := &testutil.MockRunner{Calls: []testutil.MockCall{}}
	_, _, _, _, router := setupTestHandlers(t, runner)

	req := httptest.NewRequest("GET", "/api/projects/testproj", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var project map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&project); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if project["name"] != "testproj" {
		t.Errorf("expected name testproj, got %v", project["name"])
	}
}

func TestGetProject_NotFound(t *testing.T) {
	runner := &testutil.MockRunner{Calls: []testutil.MockCall{}}
	_, _, _, _, router := setupTestHandlers(t, runner)

	req := httptest.NewRequest("GET", "/api/projects/unknown", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestListCommands(t *testing.T) {
	runner := &testutil.MockRunner{Calls: []testutil.MockCall{}}
	_, _, _, _, router := setupTestHandlers(t, runner)

	req := httptest.NewRequest("GET", "/api/projects/testproj/commands", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var commands []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&commands); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}
}

func TestGitStatus(t *testing.T) {
	runner := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: "main\n"},       // git rev-parse --abbrev-ref HEAD
		{Output: "0\t0\n"},       // git rev-list --left-right --count
		{Output: " M file.go\n"}, // git status --porcelain
	}}
	_, _, _, _, router := setupTestHandlers(t, runner)

	req := httptest.NewRequest("GET", "/api/projects/testproj/git/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var status map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if status["branch"] != "main" {
		t.Errorf("expected branch main, got %v", status["branch"])
	}
}

func TestGitCommit_EmptyMessage(t *testing.T) {
	runner := &testutil.MockRunner{Calls: []testutil.MockCall{}}
	_, _, _, _, router := setupTestHandlers(t, runner)

	body := `{"files":["file.go"]}`
	req := httptest.NewRequest("POST", "/api/projects/testproj/git/commit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGitStage(t *testing.T) {
	runner := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: ""}, // git add -- file.go
	}}
	_, _, _, _, router := setupTestHandlers(t, runner)

	body := `{"files":["file.go"]}`
	req := httptest.NewRequest("POST", "/api/projects/testproj/git/stage", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDockerContainers(t *testing.T) {
	jsonOutput := `[{"Name":"testproj-web-1","Image":"nginx","Status":"Up","State":"running","Service":"web","Ports":"80/tcp"}]`

	runner := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: jsonOutput},
	}}
	_, _, _, _, router := setupTestHandlers(t, runner)

	req := httptest.NewRequest("GET", "/api/projects/testproj/docker/containers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var containers []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&containers); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}
}

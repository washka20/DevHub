# Web Console Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a full system terminal to DevHub's browser UI with tabbed sessions and split panels.

**Architecture:** Go backend allocates PTY sessions via `creack/pty` and streams raw bytes over a dedicated WebSocket endpoint (`/api/terminal/ws/{id}`). Vue 3 frontend renders each session with `xterm.js`, organises tabs and split panes via a Pinia store and `splitpanes`, and keeps everything alive across route navigation with `<keep-alive>`.

**Tech Stack:** Go 1.25 + creack/pty, gorilla/websocket, Vue 3 + TypeScript, xterm.js, splitpanes, Pinia, Vite

**Spec:** `docs/superpowers/specs/2026-03-28-web-console-design.md`

---

### Task 1: Add `creack/pty` dependency and terminal config

**Files:**
- Modify: `go.mod`
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Add creack/pty to go.mod**

Run:
```bash
cd /home/washka/project/devhub && go get github.com/creack/pty@v1.1.24
```

Expected: `go.mod` and `go.sum` updated with `github.com/creack/pty v1.1.24`.

- [ ] **Step 2: Add TerminalConfig to config.go**

In `internal/config/config.go`, add the struct and embed it in `Config`:

```go
// TerminalConfig holds terminal-related settings.
type TerminalConfig struct {
	MaxSessions int `yaml:"max_sessions"`
}

// Config holds application configuration.
type Config struct {
	Port           int            `yaml:"port"`
	ProjectsDir    string         `yaml:"projects_dir"`
	DefaultProject string         `yaml:"default_project"`
	Terminal       TerminalConfig `yaml:"terminal"`
}
```

Update `DefaultConfig()`:

```go
func DefaultConfig() *Config {
	return &Config{
		Port:           9000,
		ProjectsDir:    "~/project",
		DefaultProject: "cfa",
		Terminal: TerminalConfig{
			MaxSessions: 10,
		},
	}
}
```

- [ ] **Step 3: Add test for terminal config defaults**

In `internal/config/config_test.go`, add:

```go
func TestDefaultConfig_TerminalMaxSessions(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Terminal.MaxSessions != 10 {
		t.Errorf("expected MaxSessions=10, got %d", cfg.Terminal.MaxSessions)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `cd /home/washka/project/devhub && go test ./internal/config/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/config/config.go internal/config/config_test.go
git commit -m "feat(terminal): add creack/pty dep and terminal config"
```

---

### Task 2: PTY Session Manager

**Files:**
- Create: `internal/terminal/terminal.go`
- Create: `internal/terminal/terminal_test.go`

- [ ] **Step 1: Write terminal_test.go with basic session tests**

Create `internal/terminal/terminal_test.go`:

```go
package terminal

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager(5)
	if m.Count() != 0 {
		t.Errorf("expected 0 sessions, got %d", m.Count())
	}
}

func TestCreateAndGet(t *testing.T) {
	m := NewManager(5)
	sess, err := m.Create("test-1", "/bin/sh", t.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if sess.ID != "test-1" {
		t.Errorf("expected ID test-1, got %s", sess.ID)
	}

	got, ok := m.Get("test-1")
	if !ok {
		t.Fatal("Get returned false for existing session")
	}
	if got.ID != "test-1" {
		t.Errorf("expected ID test-1, got %s", got.ID)
	}

	_, ok = m.Get("nonexistent")
	if ok {
		t.Error("Get returned true for nonexistent session")
	}

	m.Destroy("test-1")
}

func TestMaxSessions(t *testing.T) {
	m := NewManager(2)
	_, err := m.Create("s1", "/bin/sh", t.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create s1 failed: %v", err)
	}
	_, err = m.Create("s2", "/bin/sh", t.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create s2 failed: %v", err)
	}
	_, err = m.Create("s3", "/bin/sh", t.TempDir(), 80, 24)
	if err == nil {
		t.Error("expected error when exceeding max sessions, got nil")
	}

	m.DestroyAll()
}

func TestDestroyAll(t *testing.T) {
	m := NewManager(5)
	m.Create("s1", "/bin/sh", t.TempDir(), 80, 24)
	m.Create("s2", "/bin/sh", t.TempDir(), 80, 24)

	m.DestroyAll()

	if m.Count() != 0 {
		t.Errorf("expected 0 sessions after DestroyAll, got %d", m.Count())
	}
}

func TestPtyReadWrite(t *testing.T) {
	m := NewManager(5)
	sess, err := m.Create("rw", "/bin/sh", t.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer m.Destroy("rw")

	// Write a command to the PTY
	_, err = sess.Pty.Write([]byte("echo hello_pty_test\n"))
	if err != nil {
		t.Fatalf("Write to PTY failed: %v", err)
	}

	// Read output -- PTY is raw so we may get echoed input + output
	buf := make([]byte, 4096)
	deadline := time.Now().Add(3 * time.Second)
	var totalRead int
	for time.Now().Before(deadline) {
		sess.Pty.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := sess.Pty.Read(buf[totalRead:])
		totalRead += n
		if err != nil {
			break
		}
		if totalRead > 0 {
			output := string(buf[:totalRead])
			// The output should contain our string somewhere (echoed input or echo output)
			if containsSubstring(output, "hello_pty_test") {
				return // success
			}
		}
	}
	t.Errorf("expected output containing hello_pty_test, got %q", string(buf[:totalRead]))
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestResize(t *testing.T) {
	m := NewManager(5)
	sess, err := m.Create("resize", "/bin/sh", t.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer m.Destroy("resize")

	err = sess.Resize(120, 40)
	if err != nil {
		t.Errorf("Resize failed: %v", err)
	}
}

func TestListSessions(t *testing.T) {
	m := NewManager(5)
	m.Create("a", "/bin/sh", t.TempDir(), 80, 24)
	m.Create("b", "/bin/sh", t.TempDir(), 80, 24)

	list := m.List()
	if len(list) != 2 {
		t.Errorf("expected 2 sessions in list, got %d", len(list))
	}

	m.DestroyAll()
}
```

- [ ] **Step 2: Run tests -- expect compile failure**

Run: `cd /home/washka/project/devhub && go test ./internal/terminal/ -v`
Expected: compilation error -- package `terminal` does not exist yet.

- [ ] **Step 3: Implement terminal.go**

Create `internal/terminal/terminal.go`:

```go
package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// Session represents a single PTY session.
type Session struct {
	ID        string
	Cmd       *exec.Cmd
	Pty       *os.File // master side of the PTY
	CreatedAt time.Time
	CWD       string
	mu        sync.Mutex
	closed    bool
}

// Resize changes the PTY window size.
func (s *Session) Resize(cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("session %s is closed", s.ID)
	}
	return pty.Setsize(s.Pty, &pty.Winsize{Cols: cols, Rows: rows})
}

// Close terminates the session's process and closes the PTY.
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true

	// Send SIGHUP to the process group
	if s.Cmd.Process != nil {
		syscall.Kill(-s.Cmd.Process.Pid, syscall.SIGHUP)

		done := make(chan struct{})
		go func() {
			s.Cmd.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			syscall.Kill(-s.Cmd.Process.Pid, syscall.SIGKILL)
			<-done
		}
	}

	s.Pty.Close()
}

// SessionInfo is a JSON-safe summary of a session.
type SessionInfo struct {
	ID        string `json:"id"`
	CWD       string `json:"cwd"`
	CreatedAt string `json:"created_at"`
}

// Manager manages PTY sessions.
type Manager struct {
	sessions    map[string]*Session
	mu          sync.RWMutex
	maxSessions int
}

// NewManager creates a Manager with the given session limit.
func NewManager(maxSessions int) *Manager {
	return &Manager{
		sessions:    make(map[string]*Session),
		maxSessions: maxSessions,
	}
}

// Create starts a new PTY session.
func (m *Manager) Create(id, shell, cwd string, cols, rows uint16) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) >= m.maxSessions {
		return nil, fmt.Errorf("max sessions limit reached (%d)", m.maxSessions)
	}

	if _, exists := m.sessions[id]; exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	cmd := exec.Command(shell)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)
	// Create a new process group so SIGHUP reaches children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, fmt.Errorf("pty start: %w", err)
	}

	sess := &Session{
		ID:        id,
		Cmd:       cmd,
		Pty:       ptmx,
		CreatedAt: time.Now(),
		CWD:       cwd,
	}
	m.sessions[id] = sess
	return sess, nil
}

// Get returns a session by ID.
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// Destroy closes and removes a session.
func (m *Manager) Destroy(id string) {
	m.mu.Lock()
	sess, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()

	if ok {
		sess.Close()
	}
}

// DestroyAll closes and removes all sessions.
func (m *Manager) DestroyAll() {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()

	for _, s := range sessions {
		s.Close()
	}
}

// Count returns the number of active sessions.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// List returns info about all active sessions.
func (m *Manager) List() []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]SessionInfo, 0, len(m.sessions))
	for _, s := range m.sessions {
		list = append(list, SessionInfo{
			ID:        s.ID,
			CWD:       s.CWD,
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
		})
	}
	return list
}
```

- [ ] **Step 4: Run tests -- expect all pass**

Run: `cd /home/washka/project/devhub && go test ./internal/terminal/ -v -count=1`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/terminal/
git commit -m "feat(terminal): PTY session manager with create/destroy/resize"
```

---

### Task 3: Terminal REST handlers

**Files:**
- Create: `internal/api/terminal_handlers.go`
- Create: `internal/api/terminal_handlers_test.go`

- [ ] **Step 1: Write terminal_handlers_test.go**

Create `internal/api/terminal_handlers_test.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devhub/internal/terminal"

	"github.com/gorilla/mux"
)

func setupTerminalRouter(m *terminal.Manager) *mux.Router {
	th := &TerminalHandlers{Manager: m}
	r := mux.NewRouter()
	r.HandleFunc("/api/terminal/sessions", th.CreateSession).Methods("POST")
	r.HandleFunc("/api/terminal/sessions", th.ListSessions).Methods("GET")
	r.HandleFunc("/api/terminal/sessions/{id}", th.DestroySession).Methods("DELETE")
	return r
}

func TestCreateSession(t *testing.T) {
	m := terminal.NewManager(5)
	defer m.DestroyAll()
	router := setupTerminalRouter(m)

	body := `{"cols":80,"rows":24}`
	req := httptest.NewRequest("POST", "/api/terminal/sessions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["session_id"] == "" {
		t.Error("expected non-empty session_id")
	}
	if resp["shell"] == "" {
		t.Error("expected non-empty shell")
	}

	if m.Count() != 1 {
		t.Errorf("expected 1 session, got %d", m.Count())
	}
}

func TestListSessions(t *testing.T) {
	m := terminal.NewManager(5)
	defer m.DestroyAll()
	router := setupTerminalRouter(m)

	// Create two sessions
	m.Create("s1", "/bin/sh", t.TempDir(), 80, 24)
	m.Create("s2", "/bin/sh", t.TempDir(), 80, 24)

	req := httptest.NewRequest("GET", "/api/terminal/sessions", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var list []terminal.SessionInfo
	json.Unmarshal(rr.Body.Bytes(), &list)
	if len(list) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(list))
	}
}

func TestDestroySession(t *testing.T) {
	m := terminal.NewManager(5)
	defer m.DestroyAll()
	router := setupTerminalRouter(m)

	m.Create("del-me", "/bin/sh", t.TempDir(), 80, 24)

	req := httptest.NewRequest("DELETE", "/api/terminal/sessions/del-me", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}

	if m.Count() != 0 {
		t.Errorf("expected 0 sessions, got %d", m.Count())
	}
}

func TestDestroySession_NotFound(t *testing.T) {
	m := terminal.NewManager(5)
	router := setupTerminalRouter(m)

	req := httptest.NewRequest("DELETE", "/api/terminal/sessions/nope", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestCreateSession_MaxExceeded(t *testing.T) {
	m := terminal.NewManager(1)
	defer m.DestroyAll()
	router := setupTerminalRouter(m)

	m.Create("s1", "/bin/sh", t.TempDir(), 80, 24)

	body := `{"cols":80,"rows":24}`
	req := httptest.NewRequest("POST", "/api/terminal/sessions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", rr.Code, rr.Body.String())
	}
}
```

- [ ] **Step 2: Run tests -- expect compile failure**

Run: `cd /home/washka/project/devhub && go test ./internal/api/ -run TestCreate -v`
Expected: compilation error -- `TerminalHandlers` not defined.

- [ ] **Step 3: Implement terminal_handlers.go**

Create `internal/api/terminal_handlers.go`:

```go
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"

	"devhub/internal/terminal"

	"github.com/gorilla/mux"
)

// TerminalHandlers manages REST endpoints for terminal sessions.
type TerminalHandlers struct {
	Manager *terminal.Manager
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

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	id := generateID()
	sess, err := th.Manager.Create(id, shell, cwd, body.Cols, body.Rows)
	if err != nil {
		if th.Manager.Count() >= th.Manager.MaxSessions() {
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
```

Also add `MaxSessions()` getter to `internal/terminal/terminal.go`:

```go
// MaxSessions returns the session limit.
func (m *Manager) MaxSessions() int {
	return m.maxSessions
}
```

- [ ] **Step 4: Run tests**

Run: `cd /home/washka/project/devhub && go test ./internal/api/ -run "Terminal|CreateSession|ListSession|DestroySession" -v -count=1`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/api/terminal_handlers.go internal/api/terminal_handlers_test.go internal/terminal/terminal.go
git commit -m "feat(terminal): REST handlers for session create/list/destroy"
```

---

### Task 4: Terminal WebSocket handler

**Files:**
- Create: `internal/api/terminal_ws.go`

- [ ] **Step 1: Create terminal_ws.go**

Create `internal/api/terminal_ws.go`:

```go
package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"devhub/internal/terminal"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type terminalControlMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// HandleTerminalWS upgrades to WebSocket and bridges to a PTY session.
func HandleTerminalWS(manager *terminal.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		sess, ok := manager.Get(id)
		if !ok {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("terminal ws upgrade error: %v", err)
			return
		}

		var closeOnce sync.Once
		cleanup := func() {
			closeOnce.Do(func() {
				conn.Close()
				manager.Destroy(id)
			})
		}
		defer cleanup()

		// PTY -> WebSocket (binary frames)
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := sess.Pty.Read(buf)
				if n > 0 {
					if wErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); wErr != nil {
						cleanup()
						return
					}
				}
				if err != nil {
					// PTY closed (shell exited)
					exitCode := 0
					if err != io.EOF {
						exitCode = 1
					}
					exitMsg, _ := json.Marshal(map[string]interface{}{
						"type": "exit",
						"code": exitCode,
					})
					conn.WriteMessage(websocket.TextMessage, exitMsg)
					cleanup()
					return
				}
			}
		}()

		// WebSocket -> PTY
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				cleanup()
				return
			}

			switch msgType {
			case websocket.BinaryMessage:
				// Raw keystrokes -> PTY stdin
				if _, err := sess.Pty.Write(data); err != nil {
					cleanup()
					return
				}
			case websocket.TextMessage:
				// JSON control message
				var msg terminalControlMsg
				if err := json.Unmarshal(data, &msg); err != nil {
					continue
				}
				if msg.Type == "resize" && msg.Cols > 0 && msg.Rows > 0 {
					sess.Resize(msg.Cols, msg.Rows)
				}
			}
		}
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /home/washka/project/devhub && go build ./...`
Expected: builds successfully.

- [ ] **Step 3: Run all existing tests to check nothing is broken**

Run: `cd /home/washka/project/devhub && go test ./... -count=1`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/api/terminal_ws.go
git commit -m "feat(terminal): WebSocket handler bridging browser to PTY"
```

---

### Task 5: Wire terminal into server and main

**Files:**
- Modify: `internal/server/server.go`
- Modify: `cmd/main.go`

- [ ] **Step 1: Add terminal routes to server.go**

In `internal/server/server.go`, add the import and terminal setup inside `New()`:

Add to imports:
```go
"devhub/internal/terminal"
```

After `h.RefreshProjects()` (around line 36), add:

```go
	// Terminal
	termManager := terminal.NewManager(cfg.Terminal.MaxSessions)
	th := &api.TerminalHandlers{Manager: termManager}
	apiRouter.HandleFunc("/terminal/sessions", th.CreateSession).Methods("POST")
	apiRouter.HandleFunc("/terminal/sessions", th.ListSessions).Methods("GET")
	apiRouter.HandleFunc("/terminal/sessions/{id}", th.DestroySession).Methods("DELETE")
	apiRouter.HandleFunc("/terminal/ws/{id}", api.HandleTerminalWS(termManager))
```

Add `termManager` to the `Server` struct and return:

```go
type Server struct {
	cfg         *config.Config
	router      *mux.Router
	hub         *api.Hub
	termManager *terminal.Manager
}
```

In the return block:
```go
	s := &Server{
		cfg:         cfg,
		router:      router,
		hub:         hub,
		termManager: termManager,
	}
```

Add a `Shutdown` method:

```go
// Shutdown cleans up all terminal sessions.
func (s *Server) Shutdown() {
	if s.termManager != nil {
		s.termManager.DestroyAll()
	}
}
```

- [ ] **Step 2: Add signal handling to main.go**

Replace `cmd/main.go` with:

```go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"devhub/internal/config"
	"devhub/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	srv := server.New(cfg)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		srv.Shutdown()
		os.Exit(0)
	}()

	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

- [ ] **Step 3: Verify compilation and tests**

Run: `cd /home/washka/project/devhub && go build ./... && go test ./... -count=1`
Expected: builds and all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/server/server.go cmd/main.go
git commit -m "feat(terminal): wire PTY manager and routes into server"
```

---

### Task 6: Install frontend dependencies

**Files:**
- Modify: `frontend/package.json`

- [ ] **Step 1: Install npm packages**

Run:
```bash
cd /home/washka/project/devhub/frontend && npm install @xterm/xterm @xterm/addon-fit @xterm/addon-web-links @xterm/addon-webgl splitpanes
```

- [ ] **Step 2: Install splitpanes types (if available) or skip**

Run:
```bash
cd /home/washka/project/devhub/frontend && npm install -D @types/splitpanes 2>/dev/null || true
```

Note: splitpanes ships its own types, so this may not be needed.

- [ ] **Step 3: Add Vite proxy for terminal WebSocket**

In `frontend/vite.config.ts`, add the terminal WS proxy **before** the existing `/api/ws` rule:

```typescript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api/terminal/ws': {
        target: 'ws://localhost:9000',
        ws: true,
      },
      '/api/ws': {
        target: 'ws://localhost:9000',
        ws: true,
      },
      '/api': {
        target: 'http://localhost:9000',
        changeOrigin: true,
      },
    },
  },
})
```

- [ ] **Step 4: Verify frontend builds**

Run: `cd /home/washka/project/devhub/frontend && npx vue-tsc --noEmit`
Expected: no errors (we haven't added components yet, just deps).

- [ ] **Step 5: Commit**

```bash
cd /home/washka/project/devhub
git add frontend/package.json frontend/package-lock.json frontend/vite.config.ts
git commit -m "feat(terminal): add xterm.js, splitpanes and configure Vite proxy"
```

---

### Task 7: Pinia terminal store

**Files:**
- Create: `frontend/src/stores/terminal.ts`
- Modify: `frontend/src/types/index.ts`

- [ ] **Step 1: Add terminal types**

Append to `frontend/src/types/index.ts`:

```typescript
export interface TerminalSession {
  id: string
  label: string
  cwd: string
}

export interface TerminalPane {
  id: string
  sessionId: string
}

export interface TerminalTab {
  id: string
  label: string
  panes: TerminalPane[]
  splitDirection: 'horizontal' | 'vertical' | null
}
```

- [ ] **Step 2: Create terminal store**

Create `frontend/src/stores/terminal.ts`:

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { TerminalSession, TerminalTab, TerminalPane } from '../types'

let counter = 0
function nextId(): string {
  return `pane-${++counter}`
}

export const useTerminalStore = defineStore('terminal', () => {
  const sessions = ref<Map<string, TerminalSession>>(new Map())
  const tabs = ref<TerminalTab[]>([])
  const activeTabId = ref<string | null>(null)

  const activeTab = computed(() =>
    tabs.value.find((t) => t.id === activeTabId.value) ?? null,
  )

  async function createSession(cwd: string, cols = 80, rows = 24): Promise<TerminalSession> {
    const res = await fetch('/api/terminal/sessions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cols, rows, cwd }),
    })
    if (!res.ok) {
      throw new Error(`Failed to create session: ${res.statusText}`)
    }
    const data = await res.json()
    const session: TerminalSession = {
      id: data.session_id,
      label: data.shell.split('/').pop() || 'shell',
      cwd,
    }
    sessions.value.set(session.id, session)
    return session
  }

  async function destroySession(id: string) {
    await fetch(`/api/terminal/sessions/${id}`, { method: 'DELETE' })
    sessions.value.delete(id)
  }

  async function addTab(cwd: string): Promise<TerminalTab> {
    const session = await createSession(cwd)
    const pane: TerminalPane = { id: nextId(), sessionId: session.id }
    const tab: TerminalTab = {
      id: `tab-${session.id}`,
      label: session.label,
      panes: [pane],
      splitDirection: null,
    }
    tabs.value.push(tab)
    activeTabId.value = tab.id
    return tab
  }

  async function closeTab(tabId: string) {
    const tab = tabs.value.find((t) => t.id === tabId)
    if (!tab) return
    for (const pane of tab.panes) {
      await destroySession(pane.sessionId)
    }
    tabs.value = tabs.value.filter((t) => t.id !== tabId)
    if (activeTabId.value === tabId) {
      activeTabId.value = tabs.value.length > 0 ? tabs.value[tabs.value.length - 1].id : null
    }
  }

  function setActiveTab(tabId: string) {
    activeTabId.value = tabId
  }

  async function splitPane(tabId: string, direction: 'horizontal' | 'vertical', cwd: string) {
    const tab = tabs.value.find((t) => t.id === tabId)
    if (!tab) return
    if (tab.panes.length >= 2) return // max 2 panes per tab

    const session = await createSession(cwd)
    const pane: TerminalPane = { id: nextId(), sessionId: session.id }
    tab.panes.push(pane)
    tab.splitDirection = direction
  }

  async function closePane(tabId: string, paneId: string) {
    const tab = tabs.value.find((t) => t.id === tabId)
    if (!tab) return

    const pane = tab.panes.find((p) => p.id === paneId)
    if (!pane) return

    await destroySession(pane.sessionId)
    tab.panes = tab.panes.filter((p) => p.id !== paneId)

    if (tab.panes.length <= 1) {
      tab.splitDirection = null
    }

    if (tab.panes.length === 0) {
      await closeTab(tabId)
    }
  }

  return {
    sessions,
    tabs,
    activeTabId,
    activeTab,
    addTab,
    closeTab,
    setActiveTab,
    splitPane,
    closePane,
  }
})
```

- [ ] **Step 3: Verify types compile**

Run: `cd /home/washka/project/devhub/frontend && npx vue-tsc --noEmit`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
cd /home/washka/project/devhub
git add frontend/src/types/index.ts frontend/src/stores/terminal.ts
git commit -m "feat(terminal): Pinia store for terminal sessions, tabs, and split panes"
```

---

### Task 8: WebTerminal component (xterm.js)

**Files:**
- Create: `frontend/src/components/WebTerminal.vue`

- [ ] **Step 1: Create WebTerminal.vue**

Create `frontend/src/components/WebTerminal.vue`:

```vue
<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  sessionId: string
}>()

const emit = defineEmits<{
  exit: [code: number]
}>()

const terminalEl = ref<HTMLDivElement>()
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null

function connectWs(sessionId: string) {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const url = `${proto}//${host}/api/terminal/ws/${sessionId}`

  ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    // Send initial size
    if (term && fitAddon) {
      fitAddon.fit()
      sendResize(term.cols, term.rows)
    }
  }

  ws.onmessage = (event: MessageEvent) => {
    if (!term) return
    if (event.data instanceof ArrayBuffer) {
      const text = new TextDecoder().decode(event.data)
      term.write(text)
    } else if (typeof event.data === 'string') {
      try {
        const msg = JSON.parse(event.data)
        if (msg.type === 'exit') {
          emit('exit', msg.code)
        }
      } catch {
        // ignore
      }
    }
  }

  ws.onclose = () => {
    ws = null
  }
}

function sendResize(cols: number, rows: number) {
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'resize', cols, rows }))
  }
}

function initTerminal() {
  if (!terminalEl.value) return

  term = new Terminal({
    cursorBlink: true,
    fontFamily: "'JetBrains Mono', 'SF Mono', 'Fira Code', 'Cascadia Code', monospace",
    fontSize: 14,
    lineHeight: 1.2,
    scrollback: 10000,
    theme: {
      background: '#0d1117',
      foreground: '#c9d1d9',
      cursor: '#58a6ff',
      selectionBackground: 'rgba(88, 166, 255, 0.3)',
      black: '#484f58',
      red: '#ff7b72',
      green: '#3fb950',
      yellow: '#d29922',
      blue: '#58a6ff',
      magenta: '#bc8cff',
      cyan: '#39d353',
      white: '#b1bac4',
      brightBlack: '#6e7681',
      brightRed: '#ffa198',
      brightGreen: '#56d364',
      brightYellow: '#e3b341',
      brightBlue: '#79c0ff',
      brightMagenta: '#d2a8ff',
      brightCyan: '#56d364',
      brightWhite: '#f0f6fc',
    },
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.loadAddon(new WebLinksAddon())

  // Try WebGL addon, fall back silently
  import('@xterm/addon-webgl')
    .then(({ WebglAddon }) => {
      if (term) {
        try {
          term.loadAddon(new WebglAddon())
        } catch {
          // WebGL not available, canvas renderer is fine
        }
      }
    })
    .catch(() => {})

  term.open(terminalEl.value)
  fitAddon.fit()

  // Forward keystrokes to PTY
  term.onData((data: string) => {
    if (ws?.readyState === WebSocket.OPEN) {
      const encoder = new TextEncoder()
      ws.send(encoder.encode(data))
    }
  })

  // Forward resize
  term.onResize(({ cols, rows }) => {
    sendResize(cols, rows)
  })

  // Watch container size
  resizeObserver = new ResizeObserver(() => {
    fitAddon?.fit()
  })
  resizeObserver.observe(terminalEl.value)

  connectWs(props.sessionId)
}

onMounted(() => {
  initTerminal()
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  ws?.close()
  term?.dispose()
  term = null
  ws = null
  fitAddon = null
  resizeObserver = null
})

// If sessionId changes (e.g. reconnect), reconnect WS
watch(
  () => props.sessionId,
  (newId, oldId) => {
    if (newId !== oldId) {
      ws?.close()
      connectWs(newId)
    }
  },
)
</script>

<template>
  <div ref="terminalEl" class="web-terminal"></div>
</template>

<style scoped>
.web-terminal {
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.web-terminal :deep(.xterm) {
  height: 100%;
  padding: 4px;
}

.web-terminal :deep(.xterm-viewport) {
  overflow-y: auto !important;
}
</style>
```

- [ ] **Step 2: Verify types compile**

Run: `cd /home/washka/project/devhub/frontend && npx vue-tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd /home/washka/project/devhub
git add frontend/src/components/WebTerminal.vue
git commit -m "feat(terminal): WebTerminal component with xterm.js and PTY WebSocket"
```

---

### Task 9: TerminalTabBar component

**Files:**
- Create: `frontend/src/components/TerminalTabBar.vue`

- [ ] **Step 1: Create TerminalTabBar.vue**

Create `frontend/src/components/TerminalTabBar.vue`:

```vue
<script setup lang="ts">
import { useTerminalStore } from '../stores/terminal'
import { useProjectsStore } from '../stores/projects'

const terminalStore = useTerminalStore()
const projectsStore = useProjectsStore()

const emit = defineEmits<{
  split: [direction: 'horizontal' | 'vertical']
}>()

function handleNewTab() {
  const cwd = projectsStore.currentProject?.path || ''
  terminalStore.addTab(cwd)
}
</script>

<template>
  <div class="tab-bar">
    <div class="tabs">
      <div
        v-for="tab in terminalStore.tabs"
        :key="tab.id"
        class="tab"
        :class="{ active: terminalStore.activeTabId === tab.id }"
        @click="terminalStore.setActiveTab(tab.id)"
      >
        <span class="tab-dot" :class="{ active: terminalStore.activeTabId === tab.id }"></span>
        <span class="tab-label">{{ tab.label }}</span>
        <span class="tab-close" @click.stop="terminalStore.closeTab(tab.id)">&#10005;</span>
      </div>
      <button class="tab-add" @click="handleNewTab" title="New terminal">+</button>
    </div>

    <div class="toolbar">
      <button
        class="toolbar-btn"
        :class="{ active: terminalStore.activeTab?.splitDirection === 'horizontal' }"
        @click="emit('split', 'horizontal')"
        :disabled="!terminalStore.activeTab || (terminalStore.activeTab?.panes.length ?? 0) >= 2"
        title="Split horizontal"
      >
        &#9776; Split H
      </button>
      <button
        class="toolbar-btn"
        :class="{ active: terminalStore.activeTab?.splitDirection === 'vertical' }"
        @click="emit('split', 'vertical')"
        :disabled="!terminalStore.activeTab || (terminalStore.activeTab?.panes.length ?? 0) >= 2"
        title="Split vertical"
      >
        &#9783; Split V
      </button>
    </div>
  </div>
</template>

<style scoped>
.tab-bar {
  display: flex;
  align-items: center;
  background: #161b22;
  border-bottom: 1px solid var(--border);
  padding: 0 8px;
  height: 36px;
  flex-shrink: 0;
}

.tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
  min-width: 0;
  overflow-x: auto;
}

.tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 6px 6px 0 0;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  white-space: nowrap;
  user-select: none;
}

.tab.active {
  background: #0d1117;
  border: 1px solid var(--border);
  border-bottom: 1px solid #0d1117;
  margin-bottom: -1px;
  color: var(--text-primary);
}

.tab-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-secondary);
}

.tab-dot.active {
  background: var(--accent-green);
}

.tab-close {
  font-size: 10px;
  color: var(--text-secondary);
  opacity: 0.5;
  cursor: pointer;
  padding: 0 2px;
}

.tab-close:hover {
  opacity: 1;
  color: var(--accent-red, #f85149);
}

.tab-add {
  padding: 4px 8px;
  font-size: 14px;
  color: var(--text-secondary);
  cursor: pointer;
  border: none;
  background: none;
  border-radius: 4px;
}

.tab-add:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.toolbar {
  display: flex;
  gap: 4px;
  align-items: center;
}

.toolbar-btn {
  padding: 3px 8px;
  font-size: 11px;
  color: var(--text-secondary);
  border: 1px solid var(--border);
  border-radius: 4px;
  cursor: pointer;
  background: none;
  white-space: nowrap;
}

.toolbar-btn:hover:not(:disabled) {
  color: var(--text-primary);
  border-color: var(--text-secondary);
}

.toolbar-btn.active {
  color: var(--accent-blue);
  border-color: var(--accent-blue);
  background: rgba(88, 166, 255, 0.1);
}

.toolbar-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
</style>
```

- [ ] **Step 2: Verify types compile**

Run: `cd /home/washka/project/devhub/frontend && npx vue-tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd /home/washka/project/devhub
git add frontend/src/components/TerminalTabBar.vue
git commit -m "feat(terminal): tab bar component with new/close/split controls"
```

---

### Task 10: ConsoleView with split panels

**Files:**
- Create: `frontend/src/views/ConsoleView.vue`

- [ ] **Step 1: Create ConsoleView.vue**

Create `frontend/src/views/ConsoleView.vue`:

```vue
<script setup lang="ts">
import { onMounted } from 'vue'
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import TerminalTabBar from '../components/TerminalTabBar.vue'
import WebTerminal from '../components/WebTerminal.vue'
import { useTerminalStore } from '../stores/terminal'
import { useProjectsStore } from '../stores/projects'

const terminalStore = useTerminalStore()
const projectsStore = useProjectsStore()

onMounted(async () => {
  // Create initial tab if none exist
  if (terminalStore.tabs.length === 0) {
    const cwd = projectsStore.currentProject?.path || ''
    await terminalStore.addTab(cwd)
  }
})

function handleSplit(direction: 'horizontal' | 'vertical') {
  if (!terminalStore.activeTab) return
  const cwd = projectsStore.currentProject?.path || ''
  terminalStore.splitPane(terminalStore.activeTab.id, direction, cwd)
}

function handlePaneClose(paneId: string) {
  if (!terminalStore.activeTab) return
  terminalStore.closePane(terminalStore.activeTab.id, paneId)
}
</script>

<template>
  <div class="console-view">
    <TerminalTabBar @split="handleSplit" />

    <div class="terminal-area" v-if="terminalStore.activeTab">
      <!-- Single pane (no split) -->
      <template v-if="terminalStore.activeTab.panes.length === 1">
        <WebTerminal :session-id="terminalStore.activeTab.panes[0].sessionId" />
      </template>

      <!-- Split panes -->
      <template v-else-if="terminalStore.activeTab.panes.length > 1">
        <Splitpanes
          :horizontal="terminalStore.activeTab.splitDirection === 'vertical'"
          class="default-theme"
        >
          <Pane
            v-for="pane in terminalStore.activeTab.panes"
            :key="pane.id"
          >
            <div class="pane-container">
              <div class="pane-header">
                <span class="pane-title">
                  {{ terminalStore.sessions.get(pane.sessionId)?.label || 'shell' }}
                </span>
                <span class="pane-close" @click="handlePaneClose(pane.id)">&#10005;</span>
              </div>
              <div class="pane-body">
                <WebTerminal :session-id="pane.sessionId" />
              </div>
            </div>
          </Pane>
        </Splitpanes>
      </template>
    </div>

    <div v-else class="empty-state">
      <p>No terminal sessions. Click + to open one.</p>
    </div>
  </div>
</template>

<style scoped>
.console-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  margin: -16px -32px;
  background: #0d1117;
}

.terminal-area {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.pane-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.pane-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 8px;
  background: #161b22;
  border-bottom: 1px solid #21262d;
  font-size: 11px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.pane-title {
  font-family: var(--font-mono);
}

.pane-close {
  cursor: pointer;
  color: var(--text-secondary);
  opacity: 0.5;
  padding: 0 4px;
}

.pane-close:hover {
  opacity: 1;
  color: var(--accent-red, #f85149);
}

.pane-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  font-size: 14px;
}

/* Override splitpanes theme for DevHub dark mode */
:deep(.splitpanes.default-theme .splitpanes__splitter) {
  background: #30363d;
  min-width: 4px;
  min-height: 4px;
}

:deep(.splitpanes.default-theme .splitpanes__splitter:hover) {
  background: var(--accent-blue);
}

:deep(.splitpanes.default-theme .splitpanes__splitter::before),
:deep(.splitpanes.default-theme .splitpanes__splitter::after) {
  display: none;
}
</style>
```

- [ ] **Step 2: Verify types compile**

Run: `cd /home/washka/project/devhub/frontend && npx vue-tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd /home/washka/project/devhub
git add frontend/src/views/ConsoleView.vue
git commit -m "feat(terminal): ConsoleView with splitpanes and tab-based terminal layout"
```

---

### Task 11: Route, sidebar, and keep-alive

**Files:**
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/AppSidebar.vue`
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: Add /console route**

In `frontend/src/router/index.ts`, add after the docker route:

```typescript
    {
      path: '/console',
      name: 'console',
      component: () => import('../views/ConsoleView.vue'),
    },
```

The full file should be:

```typescript
import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: () => import('../views/DashboardView.vue'),
    },
    {
      path: '/git',
      name: 'git',
      component: () => import('../views/GitView.vue'),
    },
    {
      path: '/commands',
      name: 'commands',
      component: () => import('../views/CommandsView.vue'),
    },
    {
      path: '/docker',
      name: 'docker',
      component: () => import('../views/DockerView.vue'),
    },
    {
      path: '/console',
      name: 'console',
      component: () => import('../views/ConsoleView.vue'),
    },
  ],
})

export default router
```

- [ ] **Step 2: Add Console to sidebar**

In `frontend/src/components/AppSidebar.vue`, after the Docker `<router-link>` (line 51), add:

```html
      <router-link to="/console" class="nav-item" active-class="active">
        <span class="nav-icon">&#9002;</span>
        Console
      </router-link>
```

- [ ] **Step 3: Add keep-alive in App.vue**

In `frontend/src/App.vue`, replace the `<router-view>` block:

```vue
      <router-view v-slot="{ Component, route }">
        <keep-alive include="ConsoleView">
          <component :is="Component" :key="route.name" class="route-view" />
        </keep-alive>
      </router-view>
```

Note: For keep-alive `include` to work, `ConsoleView.vue` needs a component name. Add `defineOptions` to the top of `ConsoleView.vue` `<script setup>`:

```typescript
defineOptions({ name: 'ConsoleView' })
```

This line should be the first statement inside `<script setup>` in `ConsoleView.vue`, before any other code.

- [ ] **Step 4: Verify frontend compiles**

Run: `cd /home/washka/project/devhub/frontend && npx vue-tsc --noEmit`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
cd /home/washka/project/devhub
git add frontend/src/router/index.ts frontend/src/components/AppSidebar.vue frontend/src/App.vue frontend/src/views/ConsoleView.vue
git commit -m "feat(terminal): add console route, sidebar nav, and keep-alive persistence"
```

---

### Task 12: End-to-end verification

**Files:** None -- manual testing.

- [ ] **Step 1: Build and start the project**

Run: `cd /home/washka/project/devhub && make dev`

Wait for both Go server (:9000) and Vite dev server (:5173) to start.

- [ ] **Step 2: Open browser and navigate to Console tab**

Open `http://localhost:5173/console` in the browser.
Expected: A terminal opens with your shell prompt in the current project directory.

- [ ] **Step 3: Test basic commands**

Type `ls` and press Enter.
Expected: File listing appears with colors.

Type `echo $TERM` and press Enter.
Expected: `xterm-256color`.

- [ ] **Step 4: Test interactive TUI**

Type `htop` or `vim test.txt` and press Enter.
Expected: Full-screen TUI renders correctly. Press `q` to exit.

- [ ] **Step 5: Test Claude Code**

Type `claude` and press Enter.
Expected: Claude Code TUI opens with full colors, markdown rendering, and interactive prompts.

- [ ] **Step 6: Test tab management**

Click "+" button to create a new terminal tab.
Expected: New tab appears and activates with a fresh shell session.

Click on the first tab to switch back.
Expected: First terminal is still showing its previous content.

Click "x" on a tab to close it.
Expected: Tab is removed, session destroyed.

- [ ] **Step 7: Test split panels**

Click "Split H" button.
Expected: Terminal splits horizontally into two panes with a draggable divider.

Drag the divider.
Expected: Panes resize, terminal content re-fits.

Click the "x" on the pane header to close a split pane.
Expected: Pane removed, returns to single-pane layout.

- [ ] **Step 8: Test navigation persistence**

Navigate to /git tab, then back to /console.
Expected: All terminal sessions, tabs, and their content are preserved.

- [ ] **Step 9: Test window resize**

Resize the browser window.
Expected: Terminal re-fits to the new size automatically.

- [ ] **Step 10: Run all backend tests**

Run: `cd /home/washka/project/devhub && go test ./... -count=1`
Expected: all PASS.

- [ ] **Step 11: Run frontend type check**

Run: `cd /home/washka/project/devhub/frontend && npx vue-tsc --noEmit`
Expected: no errors.

- [ ] **Step 12: Final commit (if any fixes were needed)**

```bash
git add -A
git commit -m "fix(terminal): adjustments from end-to-end testing"
```

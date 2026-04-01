package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devhub/internal/config"
	"devhub/internal/terminal"

	"github.com/gorilla/mux"
)

func setupTerminalRouter(m *terminal.Manager) *mux.Router {
	th := &TerminalHandlers{Manager: m, Cfg: config.DefaultConfig()}
	r := mux.NewRouter()
	r.HandleFunc("/api/terminal/sessions", th.CreateSession).Methods("POST")
	r.HandleFunc("/api/terminal/sessions", th.ListSessions).Methods("GET")
	r.HandleFunc("/api/terminal/sessions/{id}", th.GetSession).Methods("GET")
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

func TestGetSession(t *testing.T) {
	m := terminal.NewManager(5)
	defer m.DestroyAll()
	router := setupTerminalRouter(m)

	t.Run("id too long returns 400", func(t *testing.T) {
		longID := strings.Repeat("a", 65)
		req := httptest.NewRequest("GET", "/api/terminal/sessions/"+longID, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("unknown id returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/terminal/sessions/nope", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("existing session returns 200 with JSON body", func(t *testing.T) {
		m.Create("get-me", "/bin/sh", t.TempDir(), 80, 24)

		req := httptest.NewRequest("GET", "/api/terminal/sessions/get-me", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp terminal.SessionInfo
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.ID != "get-me" {
			t.Errorf("expected id 'get-me', got %q", resp.ID)
		}
		if resp.CreatedAt == "" {
			t.Error("expected non-empty created_at")
		}
	})
}

package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"devhub/internal/api"
	"devhub/internal/config"
	"devhub/internal/docker"
	"devhub/internal/git"
	"devhub/internal/runner"
	"devhub/internal/terminal"
	"devhub/internal/watcher"

	"github.com/gorilla/mux"
)

// Server is the main HTTP server for DevHub.
type Server struct {
	cfg         *config.Config
	router      *mux.Router
	hub         *api.Hub
	termManager *terminal.Manager
	fileWatcher *watcher.Watcher
}

// New creates a new Server with all routes configured.
func New(cfg *config.Config) *Server {
	hub := api.NewHub()

	r := runner.ExecRunner{}
	gitSvc := git.NewGitService(r)
	dockerSvc := docker.NewDockerService(r)

	// Terminal
	termManager := terminal.NewManager(cfg.Terminal.MaxSessions)

	h := api.NewHandlers(cfg.ProjectsDir, hub, gitSvc, dockerSvc)
	h.TermManager = termManager
	h.RefreshProjects()

	// File watcher: broadcast debounced fs changes to WebSocket clients.
	var fw *watcher.Watcher
	fw, err := watcher.New(func(ev watcher.Event) {
		hub.Broadcast(api.Event{
			Type:    ev.Type,
			Project: ev.Project,
			Data:    ev.Paths,
		})
	})
	if err != nil {
		log.Printf("file watcher init failed: %v", err)
	} else if cfg.ProjectsDir != "" {
		if watchErr := fw.Watch(cfg.ProjectsDir); watchErr != nil {
			log.Printf("file watcher watch failed: %v", watchErr)
		} else {
			fw.Start()
			log.Printf("file watcher started on %s", cfg.ProjectsDir)
		}
	}

	router := mux.NewRouter()

	// API routes
	apiRouter := router.PathPrefix("/api").Subrouter()

	// Projects
	apiRouter.HandleFunc("/projects", h.ListProjects).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}", h.GetProject).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/commands", h.ListCommands).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/exec", h.ExecCommand).Methods("POST")

	// Git
	apiRouter.HandleFunc("/projects/{id}/git/status", h.GitStatus).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/branches", h.GitBranches).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/log", h.GitLog).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/graph", h.GitGraph).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/log/metadata", h.GitLogMetadata).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/branches/{name:.+}/commits", h.GitBranchCommits).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/diff", h.GitDiff).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/commit", h.GitCommit).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/checkout", h.GitCheckout).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/pull", h.GitPull).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/push", h.GitPush).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/generate-commit", h.GitGenerateCommit).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/stage", h.GitStage).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/unstage", h.GitUnstage).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/commits/{hash}", h.GitCommitDetail).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/commits/{hash}/diff", h.GitCommitDiff).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/stash", h.GitStashList).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/stash", h.GitStashPush).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/stash/{index}/apply", h.GitStashApply).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/stash/{index}/pop", h.GitStashPop).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/stash/{index}", h.GitStashDrop).Methods("DELETE")
	apiRouter.HandleFunc("/projects/{id}/git/stash/{index}/diff", h.GitStashDiff).Methods("GET")

	// Files
	apiRouter.HandleFunc("/projects/{id}/readme", h.GetReadme).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/markdown", h.ListMarkdownFiles).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/markdown/{path:.*}", h.GetMarkdownFile).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/markdown/{path:.*}", h.ToggleMarkdownCheckbox).Methods("PUT")

	// File editor API
	apiRouter.HandleFunc("/projects/{id}/files/tree", h.FileTree).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/files/content/{path:.*}", h.FileContent).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/files/content/{path:.*}", h.FileWrite).Methods("PUT")
	apiRouter.HandleFunc("/projects/{id}/files/create", h.FileCreate).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/files/delete/{path:.*}", h.FileDelete).Methods("DELETE")
	apiRouter.HandleFunc("/projects/{id}/files/rename/{path:.*}", h.FileRename).Methods("PATCH")
	apiRouter.HandleFunc("/projects/{id}/open-in-fm", h.OpenInFileManager).Methods("POST")

	// Notes
	apiRouter.HandleFunc("/projects/{id}/notes", h.ListNotes).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/notes/{slug}", h.GetNote).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/notes", h.CreateNote).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/notes/{slug}", h.UpdateNote).Methods("PUT")
	apiRouter.HandleFunc("/projects/{id}/notes/{slug}", h.DeleteNote).Methods("DELETE")

	// Docker
	apiRouter.HandleFunc("/projects/{id}/docker/containers", h.DockerContainers).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/docker/compose/up", h.DockerComposeUp).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/docker/compose/up-build", h.DockerComposeUpBuild).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/docker/compose/down", h.DockerComposeDown).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/docker/{name}/logs", h.DockerLogs).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/docker/{name}/exec", h.DockerExec).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/docker/{name}/{action}", h.DockerAction).Methods("POST")

	// Terminal
	th := &api.TerminalHandlers{Manager: termManager, Cfg: cfg}
	apiRouter.HandleFunc("/terminal/sessions", th.CreateSession).Methods("POST")
	apiRouter.HandleFunc("/terminal/sessions", th.ListSessions).Methods("GET")
	apiRouter.HandleFunc("/terminal/sessions", th.DestroyAllSessions).Methods("DELETE")
	apiRouter.HandleFunc("/terminal/sessions/{id}", th.GetSession).Methods("GET")
	apiRouter.HandleFunc("/terminal/sessions/{id}", th.DestroySession).Methods("DELETE")
	apiRouter.HandleFunc("/terminal/sessions/{id}/cwd", th.GetSessionCWD).Methods("GET")
	apiRouter.HandleFunc("/terminal/ws/{id}", api.HandleTerminalWS(termManager))

	// Settings
	settingsH := &api.SettingsHandlers{Cfg: cfg}
	apiRouter.HandleFunc("/settings", settingsH.GetSettings).Methods("GET")
	apiRouter.HandleFunc("/settings", settingsH.UpdateSettings).Methods("PUT")
	apiRouter.HandleFunc("/settings/shells", settingsH.ListShells).Methods("GET")

	// WebSocket (on apiRouter so it matches /api/ws path used by frontend)
	apiRouter.HandleFunc("/ws", hub.HandleWS)

	// SPA fallback: serve frontend/dist or fall back to index.html
	spa := spaHandler{staticDir: "frontend/dist"}
	router.PathPrefix("/").Handler(spa)

	s := &Server{
		cfg:         cfg,
		router:      router,
		hub:         hub,
		termManager: termManager,
		fileWatcher: fw,
	}

	return s
}

// Shutdown cleans up all terminal sessions and the file watcher.
func (s *Server) Shutdown() {
	if s.termManager != nil {
		s.termManager.DestroyAll()
	}
	if s.fileWatcher != nil {
		s.fileWatcher.Close()
	}
}

// Start launches the HTTP server on localhost:port.
func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
	log.Printf("DevHub server starting on http://%s", addr)

	srv := &http.Server{
		Addr:        addr,
		Handler:     corsMiddleware(loggerMiddleware(s.router)),
		ReadTimeout: 15 * time.Second,
		// WriteTimeout is 0 to support SSE (docker logs) and WebSocket
		// connections that stay open indefinitely. Request-scoped contexts
		// handle cleanup when clients disconnect.
		IdleTimeout: 120 * time.Second,
	}

	return srv.ListenAndServe()
}

// --- Middleware ---

// loggerMiddleware logs each HTTP request to stdout.
func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// corsMiddleware adds CORS headers for dev mode (Vite on :5173).
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// --- SPA handler ---

// spaHandler serves static files from staticDir and falls back to index.html
// for any path that doesn't match a file (SPA routing).
type spaHandler struct {
	staticDir string
}

func (s spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fs := http.Dir(s.staticDir)

	// Try to serve the file directly
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	f, err := fs.Open(path)
	if err != nil {
		// File not found -- serve index.html for SPA routing
		http.ServeFile(w, r, s.staticDir+"/index.html")
		return
	}
	f.Close()

	http.FileServer(fs).ServeHTTP(w, r)
}

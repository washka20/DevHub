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

	"github.com/gorilla/mux"
)

// Server is the main HTTP server for DevHub.
type Server struct {
	cfg    *config.Config
	router *mux.Router
	hub    *api.Hub
}

// New creates a new Server with all routes configured.
func New(cfg *config.Config) *Server {
	hub := api.NewHub()

	r := runner.ExecRunner{}
	gitSvc := git.NewGitService(r)
	dockerSvc := docker.NewDockerService(r)

	h := api.NewHandlers(cfg.ProjectsDir, hub, gitSvc, dockerSvc)
	h.RefreshProjects()

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

	// Docker
	apiRouter.HandleFunc("/projects/{id}/docker/containers", h.DockerContainers).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/docker/{name}/logs", h.DockerLogs).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/docker/{name}/{action}", h.DockerAction).Methods("POST")

	// WebSocket (on apiRouter so it matches /api/ws path used by frontend)
	apiRouter.HandleFunc("/ws", hub.HandleWS)

	// SPA fallback: serve frontend/dist or fall back to index.html
	spa := spaHandler{staticDir: "frontend/dist"}
	router.PathPrefix("/").Handler(spa)

	s := &Server{
		cfg:    cfg,
		router: router,
		hub:    hub,
	}

	return s
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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

package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"devhub/internal/api"
	"devhub/internal/config"
	"devhub/internal/docker"
	"devhub/internal/git"
	"devhub/internal/gitlab"
	"devhub/internal/runner"
	"devhub/internal/search"
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
	httpSrv     *http.Server
}

// New creates a new Server with all routes configured.
func New(cfg *config.Config) *Server {
	hub := api.NewHub()

	r := runner.ExecRunner{}
	gitSvc := git.NewGitService(r)
	dockerSvc := docker.NewDockerService(r)

	// Terminal
	termManager := terminal.NewManager(cfg.Terminal.MaxSessions)

	h := api.NewHandlers(cfg.ProjectsDir, hub)
	h.RefreshProjects()

	// Domain handler structs
	gitH := &api.GitHandlers{Base: h, Git: gitSvc}
	dockerH := &api.DockerHandlers{Base: h, Docker: dockerSvc, TermManager: termManager}
	fileH := &api.FileHandlers{Base: h}
	mdH := &api.MarkdownHandlers{Base: h}
	notesH := &api.NotesHandlers{Base: h}
	execH := &api.ExecHandlers{Base: h}

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
	apiRouter.HandleFunc("/projects/{id}/commands", execH.ListCommands).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/exec", execH.ExecCommand).Methods("POST")

	// Git
	apiRouter.HandleFunc("/projects/{id}/git/status", gitH.GitStatus).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/branches", gitH.GitBranches).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/log", gitH.GitLog).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/graph", gitH.GitGraph).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/log/metadata", gitH.GitLogMetadata).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/branches/{name:.+}/commits", gitH.GitBranchCommits).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/diff", gitH.GitDiff).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/commit", gitH.GitCommit).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/checkout", gitH.GitCheckout).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/pull", gitH.GitPull).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/push", gitH.GitPush).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/generate-commit", gitH.GitGenerateCommit).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/stage", gitH.GitStage).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/unstage", gitH.GitUnstage).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/commits/{hash}", gitH.GitCommitDetail).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/commits/{hash}/diff", gitH.GitCommitDiff).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/stash", gitH.GitStashList).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/stash", gitH.GitStashPush).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/stash/{index}/apply", gitH.GitStashApply).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/stash/{index}/pop", gitH.GitStashPop).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/git/stash/{index}", gitH.GitStashDrop).Methods("DELETE")
	apiRouter.HandleFunc("/projects/{id}/git/stash/{index}/diff", gitH.GitStashDiff).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/blame", gitH.GitBlame).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/git/cherry-pick", gitH.GitCherryPick).Methods("POST")

	// Files
	apiRouter.HandleFunc("/projects/{id}/readme", mdH.GetReadme).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/markdown", mdH.ListMarkdownFiles).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/markdown/{path:.*}", mdH.GetMarkdownFile).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/markdown/{path:.*}", mdH.ToggleMarkdownCheckbox).Methods("PUT")

	// File editor API
	apiRouter.HandleFunc("/projects/{id}/files/tree", fileH.FileTree).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/files/content/{path:.*}", fileH.FileContent).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/files/content/{path:.*}", fileH.FileWrite).Methods("PUT")
	apiRouter.HandleFunc("/projects/{id}/files/create", fileH.FileCreate).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/files/delete/{path:.*}", fileH.FileDelete).Methods("DELETE")
	apiRouter.HandleFunc("/projects/{id}/files/rename/{path:.*}", fileH.FileRename).Methods("PATCH")
	apiRouter.HandleFunc("/projects/{id}/open-in-fm", fileH.OpenInFileManager).Methods("POST")

	// Search
	searchSvc := search.NewSearchService(r)
	searchH := &api.SearchHandlers{Base: h, Search: searchSvc}
	apiRouter.HandleFunc("/projects/{id}/files/search", searchH.FileSearch).Methods("GET")

	// Notes
	apiRouter.HandleFunc("/projects/{id}/notes", notesH.ListNotes).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/notes/{slug}", notesH.GetNote).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/notes", notesH.CreateNote).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/notes/{slug}", notesH.UpdateNote).Methods("PUT")
	apiRouter.HandleFunc("/projects/{id}/notes/{slug}", notesH.DeleteNote).Methods("DELETE")

	// Docker
	apiRouter.HandleFunc("/projects/{id}/docker/stats", dockerH.DockerStats).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/docker/containers", dockerH.DockerContainers).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/docker/compose/up", dockerH.DockerComposeUp).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/docker/compose/up-build", dockerH.DockerComposeUpBuild).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/docker/compose/down", dockerH.DockerComposeDown).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/docker/{name}/inspect", dockerH.DockerInspect).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/docker/{name}/logs", dockerH.DockerLogs).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/docker/{name}/exec", dockerH.DockerExec).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/docker/{name}/{action}", dockerH.DockerAction).Methods("POST")

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

	// GitLab (only if enabled in config)
	if cfg.Services.GitLab.Enabled && cfg.Services.GitLab.Token != "" {
		glClient := gitlab.NewClient(cfg.Services.GitLab.URL, cfg.Services.GitLab.Token)
		glh := &api.GitLabHandlers{Client: glClient, Handlers: h}

		// Enabled check
		apiRouter.HandleFunc("/gitlab/enabled", glh.GitLabEnabled).Methods("GET")

		// File proxy (images, attachments)
		apiRouter.HandleFunc("/gitlab/proxy", glh.GitLabProxy).Methods("GET")

		// Cross-project (no {id} in path)
		apiRouter.HandleFunc("/gitlab/my/issues", glh.GitLabMyIssues).Methods("GET")
		apiRouter.HandleFunc("/gitlab/my/merge-requests", glh.GitLabMyMRs).Methods("GET")
		apiRouter.HandleFunc("/gitlab/my/review-merge-requests", glh.GitLabMyReviewMRs).Methods("GET")
		apiRouter.HandleFunc("/gitlab/my/todos", glh.GitLabMyTodos).Methods("GET")
		apiRouter.HandleFunc("/gitlab/my/todos/{todoId:[0-9]+}/done", glh.GitLabMarkTodoDone).Methods("POST")
		apiRouter.HandleFunc("/gitlab/my/todos/mark-all-done", glh.GitLabMarkAllTodosDone).Methods("POST")
		apiRouter.HandleFunc("/gitlab/user", glh.GitLabCurrentUser).Methods("GET")
		apiRouter.HandleFunc("/gitlab/labels", glh.GitLabLabels).Methods("GET")
		apiRouter.HandleFunc("/gitlab/milestones", glh.GitLabMilestones).Methods("GET")

		// Per-project: existing
		apiRouter.HandleFunc("/projects/{id}/gitlab/project", glh.GitLabProject).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/issues", glh.GitLabIssues).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests", glh.GitLabMergeRequests).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/pipelines", glh.GitLabPipelines).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/environments", glh.GitLabEnvironments).Methods("GET")

		// Per-project: detail + notes
		apiRouter.HandleFunc("/projects/{id}/gitlab/issues/{iid:[0-9]+}", glh.GitLabIssueDetail).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/issues/{iid:[0-9]+}/notes", glh.GitLabIssueNotes).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/issues/{iid:[0-9]+}/notes", glh.GitLabAddIssueNote).Methods("POST")
		apiRouter.HandleFunc("/projects/{id}/gitlab/issues/{iid:[0-9]+}", glh.GitLabUpdateIssue).Methods("PUT")
		apiRouter.HandleFunc("/projects/{id}/gitlab/issues", glh.GitLabCreateIssue).Methods("POST")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests/{iid:[0-9]+}/notes", glh.GitLabMRNotes).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests/{iid:[0-9]+}/notes", glh.GitLabAddMRNote).Methods("POST")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests", glh.GitLabCreateMR).Methods("POST")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests/{iid:[0-9]+}/approvals", glh.GitLabMRApprovals).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests/{iid:[0-9]+}/approve", glh.GitLabApproveMR).Methods("POST")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests/{iid:[0-9]+}/unapprove", glh.GitLabUnapproveMR).Methods("POST")
		apiRouter.HandleFunc("/projects/{id}/gitlab/pipelines/{pipelineId:[0-9]+}/jobs", glh.GitLabPipelineJobs).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/jobs/{jobId:[0-9]+}/trace", glh.GitLabJobTrace).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/jobs/{jobId:[0-9]+}/retry", glh.GitLabRetryJob).Methods("POST")
		apiRouter.HandleFunc("/projects/{id}/gitlab/jobs/{jobId:[0-9]+}/cancel", glh.GitLabCancelJob).Methods("POST")
		apiRouter.HandleFunc("/projects/{id}/gitlab/members", glh.GitLabProjectMembers).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests/{iid:[0-9]+}/discussions", glh.GitLabMRDiscussions).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests/{iid:[0-9]+}/discussions/{discussionId}", glh.GitLabResolveMRDiscussion).Methods("PUT")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests/{iid:[0-9]+}/discussions/{discussionId}/notes", glh.GitLabReplyToDiscussion).Methods("POST")

		// Per-project: time tracking
		apiRouter.HandleFunc("/projects/{id}/gitlab/issues/{iid:[0-9]+}/time-stats", glh.GitLabIssueTimeStats).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/issues/{iid:[0-9]+}/add-spent-time", glh.GitLabIssueAddSpentTime).Methods("POST")
		apiRouter.HandleFunc("/projects/{id}/gitlab/issues/{iid:[0-9]+}/time-estimate", glh.GitLabIssueSetTimeEstimate).Methods("POST")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests/{iid:[0-9]+}/time-stats", glh.GitLabMRTimeStats).Methods("GET")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests/{iid:[0-9]+}/add-spent-time", glh.GitLabMRAddSpentTime).Methods("POST")
		apiRouter.HandleFunc("/projects/{id}/gitlab/merge-requests/{iid:[0-9]+}/time-estimate", glh.GitLabMRSetTimeEstimate).Methods("POST")

		// Direct by GitLab project ID (no DevHub project binding)
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/issues/{iid:[0-9]+}", glh.DirectIssueDetail).Methods("GET")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/issues/{iid:[0-9]+}/notes", glh.DirectIssueNotes).Methods("GET")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/issues/{iid:[0-9]+}/notes", glh.DirectAddIssueNote).Methods("POST")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/issues/{iid:[0-9]+}", glh.DirectUpdateIssue).Methods("PUT")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/issues", glh.DirectCreateIssue).Methods("POST")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/merge-requests/{iid:[0-9]+}/notes", glh.DirectMRNotes).Methods("GET")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/merge-requests/{iid:[0-9]+}/notes", glh.DirectAddMRNote).Methods("POST")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/merge-requests", glh.DirectCreateMR).Methods("POST")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/merge-requests/{iid:[0-9]+}/approvals", glh.DirectMRApprovals).Methods("GET")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/merge-requests/{iid:[0-9]+}/approve", glh.DirectApproveMR).Methods("POST")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/merge-requests/{iid:[0-9]+}/unapprove", glh.DirectUnapproveMR).Methods("POST")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/pipelines/{pipelineId:[0-9]+}/jobs", glh.DirectPipelineJobs).Methods("GET")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/jobs/{jobId:[0-9]+}/trace", glh.DirectJobTrace).Methods("GET")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/jobs/{jobId:[0-9]+}/retry", glh.DirectRetryJob).Methods("POST")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/jobs/{jobId:[0-9]+}/cancel", glh.DirectCancelJob).Methods("POST")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/members", glh.DirectProjectMembers).Methods("GET")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/merge-requests/{iid:[0-9]+}/discussions", glh.DirectMRDiscussions).Methods("GET")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/merge-requests/{iid:[0-9]+}/discussions/{discussionId}", glh.DirectResolveMRDiscussion).Methods("PUT")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/merge-requests/{iid:[0-9]+}/discussions/{discussionId}/notes", glh.DirectReplyToDiscussion).Methods("POST")

		// Direct: time tracking
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/issues/{iid:[0-9]+}/time-stats", glh.DirectIssueTimeStats).Methods("GET")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/issues/{iid:[0-9]+}/add-spent-time", glh.DirectIssueAddSpentTime).Methods("POST")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/issues/{iid:[0-9]+}/time-estimate", glh.DirectIssueSetTimeEstimate).Methods("POST")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/merge-requests/{iid:[0-9]+}/time-stats", glh.DirectMRTimeStats).Methods("GET")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/merge-requests/{iid:[0-9]+}/add-spent-time", glh.DirectMRAddSpentTime).Methods("POST")
		apiRouter.HandleFunc("/gitlab/projects/{pid:[0-9]+}/merge-requests/{iid:[0-9]+}/time-estimate", glh.DirectMRSetTimeEstimate).Methods("POST")

		log.Printf("GitLab integration enabled for %s", cfg.Services.GitLab.URL)
	}

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

// Shutdown gracefully stops the HTTP server, closes the WebSocket hub,
// cleans up terminal sessions, and stops the file watcher.
func (s *Server) Shutdown(ctx context.Context) {
	if s.httpSrv != nil {
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			log.Printf("http server shutdown error: %v", err)
		}
	}
	if s.hub != nil {
		s.hub.Close()
	}
	if s.termManager != nil {
		s.termManager.DestroyAll()
	}
	if s.fileWatcher != nil {
		s.fileWatcher.Close()
	}
}

// Start launches the HTTP server on localhost:port.
// It blocks until the server is shut down; returns http.ErrServerClosed
// on graceful shutdown.
func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
	log.Printf("DevHub server starting on http://%s", addr)

	s.httpSrv = &http.Server{
		Addr:        addr,
		Handler:     corsMiddleware(loggerMiddleware(s.router)),
		ReadTimeout: 15 * time.Second,
		// WriteTimeout is 0 to support SSE (docker logs) and WebSocket
		// connections that stay open indefinitely. Request-scoped contexts
		// handle cleanup when clients disconnect.
		IdleTimeout: 120 * time.Second,
	}

	return s.httpSrv.ListenAndServe()
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

package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"devhub/internal/gitlab"

	"github.com/gorilla/mux"
)

// GitLabHandlers manages REST endpoints for GitLab integration.
type GitLabHandlers struct {
	Client   *gitlab.Client
	Handlers *Handlers
}

// GitLabEnabled handles GET /api/gitlab/enabled
func (gh *GitLabHandlers) GitLabEnabled(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, map[string]bool{"enabled": true})
}

// getRemoteURL runs `git remote get-url origin` in the project directory.
func getRemoteURL(projectPath string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = projectPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// resolveGitLabProject resolves the DevHub project {id} to a GitLab project ID.
func (gh *GitLabHandlers) resolveGitLabProject(r *http.Request) (*gitlab.Project, error) {
	projectPath, err := gh.Handlers.projectPath(r)
	if err != nil {
		return nil, err
	}

	remoteURL, err := getRemoteURL(projectPath)
	if err != nil {
		return nil, err
	}

	return gh.Client.ProjectByRemote(remoteURL)
}

func (gh *GitLabHandlers) glProjectID(r *http.Request) (int, error) {
	return strconv.Atoi(mux.Vars(r)["pid"])
}

// resolveProjectID extracts GitLab project ID from either DevHub project or direct {pid}.
func (gh *GitLabHandlers) resolveProjectID(r *http.Request, direct bool) (int, error) {
	if direct {
		return gh.glProjectID(r)
	}
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		return 0, err
	}
	return project.ID, nil
}

func (gh *GitLabHandlers) projectIDError(w http.ResponseWriter, err error, direct bool) {
	if direct {
		jsonError(w, "invalid project ID", http.StatusBadRequest)
	} else {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
	}
}

// --- Shared private methods ---

func (gh *GitLabHandlers) issueDetail(w http.ResponseWriter, pid, iid int) {
	issue, err := gh.Client.IssueDetail(pid, iid)
	if err != nil {
		jsonError(w, "failed to fetch issue: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, issue)
}

func (gh *GitLabHandlers) issueNotes(w http.ResponseWriter, pid, iid int) {
	notes, err := gh.Client.IssueNotes(pid, iid)
	if err != nil {
		jsonError(w, "failed to fetch issue notes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, notes)
}

func (gh *GitLabHandlers) addIssueNote(w http.ResponseWriter, r *http.Request, pid, iid int) {
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Body == "" {
		jsonError(w, "body is required", http.StatusBadRequest)
		return
	}
	note, err := gh.Client.AddIssueNote(pid, iid, body.Body)
	if err != nil {
		jsonError(w, "failed to add issue note: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

func (gh *GitLabHandlers) updateIssue(w http.ResponseWriter, r *http.Request, pid, iid int) {
	var req gitlab.UpdateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	issue, err := gh.Client.UpdateIssue(pid, iid, req)
	if err != nil {
		jsonError(w, "failed to update issue: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, issue)
}

func (gh *GitLabHandlers) createIssue(w http.ResponseWriter, r *http.Request, pid int) {
	var req gitlab.CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		jsonError(w, "title is required", http.StatusBadRequest)
		return
	}
	issue, err := gh.Client.CreateIssue(pid, req)
	if err != nil {
		jsonError(w, "failed to create issue: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(issue)
}

func (gh *GitLabHandlers) mrNotes(w http.ResponseWriter, pid, iid int) {
	notes, err := gh.Client.MRNotes(pid, iid)
	if err != nil {
		jsonError(w, "failed to fetch MR notes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, notes)
}

func (gh *GitLabHandlers) addMRNote(w http.ResponseWriter, r *http.Request, pid, iid int) {
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Body == "" {
		jsonError(w, "body is required", http.StatusBadRequest)
		return
	}
	note, err := gh.Client.AddMRNote(pid, iid, body.Body)
	if err != nil {
		jsonError(w, "failed to add MR note: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

func (gh *GitLabHandlers) createMR(w http.ResponseWriter, r *http.Request, pid int) {
	var req gitlab.CreateMRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Title == "" || req.SourceBranch == "" || req.TargetBranch == "" {
		jsonError(w, "title, source_branch, and target_branch are required", http.StatusBadRequest)
		return
	}
	mr, err := gh.Client.CreateMR(pid, req)
	if err != nil {
		jsonError(w, "failed to create merge request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(mr)
}

func (gh *GitLabHandlers) projectMembers(w http.ResponseWriter, pid int) {
	members, err := gh.Client.ProjectMembers(pid)
	if err != nil {
		jsonError(w, "failed to fetch project members: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, members)
}

func (gh *GitLabHandlers) mrApprovals(w http.ResponseWriter, pid, iid int) {
	approvals, err := gh.Client.MRApprovals(pid, iid)
	if err != nil {
		jsonError(w, "failed to fetch approvals: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, approvals)
}

func (gh *GitLabHandlers) approveMR(w http.ResponseWriter, pid, iid int) {
	if err := gh.Client.ApproveMR(pid, iid); err != nil {
		jsonError(w, "failed to approve: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]bool{"ok": true})
}

func (gh *GitLabHandlers) unapproveMR(w http.ResponseWriter, pid, iid int) {
	if err := gh.Client.UnapproveMR(pid, iid); err != nil {
		jsonError(w, "failed to unapprove: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]bool{"ok": true})
}

// --- Handler that resolves project ID and IID, then delegates ---

func (gh *GitLabHandlers) handleWithProjectAndIID(w http.ResponseWriter, r *http.Request, direct bool, fn func(http.ResponseWriter, *http.Request, int, int)) {
	pid, err := gh.resolveProjectID(r, direct)
	if err != nil {
		gh.projectIDError(w, err, direct)
		return
	}
	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid IID", http.StatusBadRequest)
		return
	}
	fn(w, r, pid, iid)
}

func (gh *GitLabHandlers) handleWithProject(w http.ResponseWriter, r *http.Request, direct bool, fn func(http.ResponseWriter, *http.Request, int)) {
	pid, err := gh.resolveProjectID(r, direct)
	if err != nil {
		gh.projectIDError(w, err, direct)
		return
	}
	fn(w, r, pid)
}

// --- Per-project handlers ---

// GitLabProject handles GET /api/projects/{id}/gitlab/project
func (gh *GitLabHandlers) GitLabProject(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}
	jsonResponse(w, project)
}

// GitLabIssues handles GET /api/projects/{id}/gitlab/issues?state=opened
func (gh *GitLabHandlers) GitLabIssues(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		state = "opened"
	}

	issues, err := gh.Client.Issues(project.ID, state)
	if err != nil {
		jsonError(w, "failed to fetch issues: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, issues)
}

// GitLabMergeRequests handles GET /api/projects/{id}/gitlab/merge-requests?state=opened
func (gh *GitLabHandlers) GitLabMergeRequests(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		state = "opened"
	}

	mrs, err := gh.Client.MergeRequests(project.ID, state)
	if err != nil {
		jsonError(w, "failed to fetch merge requests: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, mrs)
}

// GitLabPipelines handles GET /api/projects/{id}/gitlab/pipelines
func (gh *GitLabHandlers) GitLabPipelines(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}

	pipelines, err := gh.Client.Pipelines(project.ID)
	if err != nil {
		jsonError(w, "failed to fetch pipelines: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, pipelines)
}

// --- Cross-project handlers ---

// GitLabMyIssues handles GET /api/gitlab/my/issues?state=opened
func (gh *GitLabHandlers) GitLabMyIssues(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		state = "opened"
	}

	issues, err := gh.Client.MyIssues(state)
	if err != nil {
		jsonError(w, "failed to fetch my issues: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, issues)
}

// GitLabMyMRs handles GET /api/gitlab/my/merge-requests?state=opened
func (gh *GitLabHandlers) GitLabMyMRs(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		state = "opened"
	}

	mrs, err := gh.Client.MyMergeRequests(state)
	if err != nil {
		jsonError(w, "failed to fetch my merge requests: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, mrs)
}

// GitLabMyReviewMRs handles GET /api/gitlab/my/review-merge-requests?state=opened
func (gh *GitLabHandlers) GitLabMyReviewMRs(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		state = "opened"
	}

	mrs, err := gh.Client.MyMergeRequestsToReview(state)
	if err != nil {
		jsonError(w, "failed to fetch review MRs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, mrs)
}

// GitLabCurrentUser handles GET /api/gitlab/user
func (gh *GitLabHandlers) GitLabCurrentUser(w http.ResponseWriter, _ *http.Request) {
	user, err := gh.Client.CurrentUser()
	if err != nil {
		jsonError(w, "failed to fetch current user: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, user)
}

// GitLabLabels handles GET /api/gitlab/labels
func (gh *GitLabHandlers) GitLabLabels(w http.ResponseWriter, _ *http.Request) {
	labels, err := gh.Client.AllLabels()
	if err != nil {
		jsonError(w, "failed to fetch labels: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, labels)
}

// GitLabMilestones handles GET /api/gitlab/milestones
func (gh *GitLabHandlers) GitLabMilestones(w http.ResponseWriter, _ *http.Request) {
	milestones, err := gh.Client.AllMilestones()
	if err != nil {
		jsonError(w, "failed to fetch milestones: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, milestones)
}

// --- Per-project detail handlers (delegate to shared methods) ---

func (gh *GitLabHandlers) GitLabIssueDetail(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, false, func(w http.ResponseWriter, _ *http.Request, pid, iid int) {
		gh.issueDetail(w, pid, iid)
	})
}

func (gh *GitLabHandlers) GitLabIssueNotes(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, false, func(w http.ResponseWriter, _ *http.Request, pid, iid int) {
		gh.issueNotes(w, pid, iid)
	})
}

func (gh *GitLabHandlers) GitLabAddIssueNote(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, false, func(w http.ResponseWriter, r *http.Request, pid, iid int) {
		gh.addIssueNote(w, r, pid, iid)
	})
}

func (gh *GitLabHandlers) GitLabUpdateIssue(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, false, func(w http.ResponseWriter, r *http.Request, pid, iid int) {
		gh.updateIssue(w, r, pid, iid)
	})
}

func (gh *GitLabHandlers) GitLabCreateIssue(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProject(w, r, false, func(w http.ResponseWriter, r *http.Request, pid int) {
		gh.createIssue(w, r, pid)
	})
}

func (gh *GitLabHandlers) GitLabMRNotes(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, false, func(w http.ResponseWriter, _ *http.Request, pid, iid int) {
		gh.mrNotes(w, pid, iid)
	})
}

func (gh *GitLabHandlers) GitLabAddMRNote(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, false, func(w http.ResponseWriter, r *http.Request, pid, iid int) {
		gh.addMRNote(w, r, pid, iid)
	})
}

func (gh *GitLabHandlers) GitLabCreateMR(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProject(w, r, false, func(w http.ResponseWriter, r *http.Request, pid int) {
		gh.createMR(w, r, pid)
	})
}

func (gh *GitLabHandlers) GitLabProjectMembers(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProject(w, r, false, func(w http.ResponseWriter, _ *http.Request, pid int) {
		gh.projectMembers(w, pid)
	})
}

// --- Direct handlers by GitLab project ID ---

func (gh *GitLabHandlers) DirectIssueDetail(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, true, func(w http.ResponseWriter, _ *http.Request, pid, iid int) {
		gh.issueDetail(w, pid, iid)
	})
}

func (gh *GitLabHandlers) DirectIssueNotes(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, true, func(w http.ResponseWriter, _ *http.Request, pid, iid int) {
		gh.issueNotes(w, pid, iid)
	})
}

func (gh *GitLabHandlers) DirectAddIssueNote(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, true, func(w http.ResponseWriter, r *http.Request, pid, iid int) {
		gh.addIssueNote(w, r, pid, iid)
	})
}

func (gh *GitLabHandlers) DirectUpdateIssue(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, true, func(w http.ResponseWriter, r *http.Request, pid, iid int) {
		gh.updateIssue(w, r, pid, iid)
	})
}

func (gh *GitLabHandlers) DirectCreateIssue(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProject(w, r, true, func(w http.ResponseWriter, r *http.Request, pid int) {
		gh.createIssue(w, r, pid)
	})
}

func (gh *GitLabHandlers) DirectMRNotes(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, true, func(w http.ResponseWriter, _ *http.Request, pid, iid int) {
		gh.mrNotes(w, pid, iid)
	})
}

func (gh *GitLabHandlers) DirectAddMRNote(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, true, func(w http.ResponseWriter, r *http.Request, pid, iid int) {
		gh.addMRNote(w, r, pid, iid)
	})
}

func (gh *GitLabHandlers) DirectCreateMR(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProject(w, r, true, func(w http.ResponseWriter, r *http.Request, pid int) {
		gh.createMR(w, r, pid)
	})
}

func (gh *GitLabHandlers) DirectProjectMembers(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProject(w, r, true, func(w http.ResponseWriter, _ *http.Request, pid int) {
		gh.projectMembers(w, pid)
	})
}

// --- MR Approval handlers (per-project) ---

func (gh *GitLabHandlers) GitLabMRApprovals(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, false, func(w http.ResponseWriter, _ *http.Request, pid, iid int) {
		gh.mrApprovals(w, pid, iid)
	})
}

func (gh *GitLabHandlers) GitLabApproveMR(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, false, func(w http.ResponseWriter, _ *http.Request, pid, iid int) {
		gh.approveMR(w, pid, iid)
	})
}

func (gh *GitLabHandlers) GitLabUnapproveMR(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, false, func(w http.ResponseWriter, _ *http.Request, pid, iid int) {
		gh.unapproveMR(w, pid, iid)
	})
}

// --- MR Approval handlers (direct by GitLab project ID) ---

func (gh *GitLabHandlers) DirectMRApprovals(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, true, func(w http.ResponseWriter, _ *http.Request, pid, iid int) {
		gh.mrApprovals(w, pid, iid)
	})
}

func (gh *GitLabHandlers) DirectApproveMR(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, true, func(w http.ResponseWriter, _ *http.Request, pid, iid int) {
		gh.approveMR(w, pid, iid)
	})
}

func (gh *GitLabHandlers) DirectUnapproveMR(w http.ResponseWriter, r *http.Request) {
	gh.handleWithProjectAndIID(w, r, true, func(w http.ResponseWriter, _ *http.Request, pid, iid int) {
		gh.unapproveMR(w, pid, iid)
	})
}

// GitLabMyTodos handles GET /api/gitlab/my/todos
func (gh *GitLabHandlers) GitLabMyTodos(w http.ResponseWriter, r *http.Request) {
	todos, err := gh.Client.MyTodos()
	if err != nil {
		jsonError(w, "failed to fetch todos: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, todos)
}

// GitLabMarkTodoDone handles POST /api/gitlab/my/todos/{todoId}/done
func (gh *GitLabHandlers) GitLabMarkTodoDone(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["todoId"])
	if err != nil {
		jsonError(w, "invalid todo ID", http.StatusBadRequest)
		return
	}
	if err := gh.Client.MarkTodoDone(id); err != nil {
		jsonError(w, "failed to mark todo: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]bool{"ok": true})
}

// GitLabMarkAllTodosDone handles POST /api/gitlab/my/todos/mark-all-done
func (gh *GitLabHandlers) GitLabMarkAllTodosDone(w http.ResponseWriter, r *http.Request) {
	if err := gh.Client.MarkAllTodosDone(); err != nil {
		jsonError(w, "failed to mark all todos: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]bool{"ok": true})
}

// GitLabProxy handles GET /api/gitlab/proxy?url=<encoded-url>
func (gh *GitLabHandlers) GitLabProxy(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		jsonError(w, "url parameter is required", http.StatusBadRequest)
		return
	}

	body, contentType, contentLength, err := gh.Client.FetchRaw(targetURL)
	if err != nil {
		jsonError(w, "proxy fetch failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer body.Close()

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if contentLength > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	}
	w.Header().Set("Cache-Control", "public, max-age=86400, immutable")

	const maxProxySize = 50 << 20
	io.Copy(w, io.LimitReader(body, maxProxySize))
}

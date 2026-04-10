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

// --- Existing per-project handlers ---

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

// --- Cross-project handlers (no project ID needed) ---

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

// --- Per-project detail handlers ---

// GitLabIssueDetail handles GET /api/projects/{id}/gitlab/issues/{iid}
func (gh *GitLabHandlers) GitLabIssueDetail(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}

	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid issue IID", http.StatusBadRequest)
		return
	}

	issue, err := gh.Client.IssueDetail(project.ID, iid)
	if err != nil {
		jsonError(w, "failed to fetch issue: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, issue)
}

// GitLabIssueNotes handles GET /api/projects/{id}/gitlab/issues/{iid}/notes
func (gh *GitLabHandlers) GitLabIssueNotes(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}

	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid issue IID", http.StatusBadRequest)
		return
	}

	notes, err := gh.Client.IssueNotes(project.ID, iid)
	if err != nil {
		jsonError(w, "failed to fetch issue notes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, notes)
}

// GitLabMRNotes handles GET /api/projects/{id}/gitlab/merge-requests/{iid}/notes
func (gh *GitLabHandlers) GitLabMRNotes(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}

	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid merge request IID", http.StatusBadRequest)
		return
	}

	notes, err := gh.Client.MRNotes(project.ID, iid)
	if err != nil {
		jsonError(w, "failed to fetch MR notes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, notes)
}

// GitLabProjectMembers handles GET /api/projects/{id}/gitlab/members
func (gh *GitLabHandlers) GitLabProjectMembers(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}

	members, err := gh.Client.ProjectMembers(project.ID)
	if err != nil {
		jsonError(w, "failed to fetch project members: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, members)
}

// --- Write handlers ---

// GitLabCreateIssue handles POST /api/projects/{id}/gitlab/issues
func (gh *GitLabHandlers) GitLabCreateIssue(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}

	var req gitlab.CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		jsonError(w, "title is required", http.StatusBadRequest)
		return
	}

	issue, err := gh.Client.CreateIssue(project.ID, req)
	if err != nil {
		jsonError(w, "failed to create issue: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(issue)
}

// GitLabCreateMR handles POST /api/projects/{id}/gitlab/merge-requests
func (gh *GitLabHandlers) GitLabCreateMR(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}

	var req gitlab.CreateMRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Title == "" || req.SourceBranch == "" || req.TargetBranch == "" {
		jsonError(w, "title, source_branch, and target_branch are required", http.StatusBadRequest)
		return
	}

	mr, err := gh.Client.CreateMR(project.ID, req)
	if err != nil {
		jsonError(w, "failed to create merge request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(mr)
}

// GitLabAddIssueNote handles POST /api/projects/{id}/gitlab/issues/{iid}/notes
func (gh *GitLabHandlers) GitLabAddIssueNote(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}

	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid issue IID", http.StatusBadRequest)
		return
	}

	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Body == "" {
		jsonError(w, "body is required", http.StatusBadRequest)
		return
	}

	note, err := gh.Client.AddIssueNote(project.ID, iid, body.Body)
	if err != nil {
		jsonError(w, "failed to add issue note: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

// GitLabAddMRNote handles POST /api/projects/{id}/gitlab/merge-requests/{iid}/notes
func (gh *GitLabHandlers) GitLabAddMRNote(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}

	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid merge request IID", http.StatusBadRequest)
		return
	}

	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Body == "" {
		jsonError(w, "body is required", http.StatusBadRequest)
		return
	}

	note, err := gh.Client.AddMRNote(project.ID, iid, body.Body)
	if err != nil {
		jsonError(w, "failed to add MR note: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

// GitLabUpdateIssue handles PUT /api/projects/{id}/gitlab/issues/{iid}
func (gh *GitLabHandlers) GitLabUpdateIssue(w http.ResponseWriter, r *http.Request) {
	project, err := gh.resolveGitLabProject(r)
	if err != nil {
		jsonError(w, "gitlab project not found: "+err.Error(), http.StatusNotFound)
		return
	}

	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid issue IID", http.StatusBadRequest)
		return
	}

	var req gitlab.UpdateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	issue, err := gh.Client.UpdateIssue(project.ID, iid, req)
	if err != nil {
		jsonError(w, "failed to update issue: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, issue)
}

// --- Direct handlers by GitLab project ID (no DevHub project binding) ---

func (gh *GitLabHandlers) glProjectID(r *http.Request) (int, error) {
	return strconv.Atoi(mux.Vars(r)["pid"])
}

// DirectIssueDetail handles GET /api/gitlab/projects/{pid}/issues/{iid}
func (gh *GitLabHandlers) DirectIssueDetail(w http.ResponseWriter, r *http.Request) {
	pid, err := gh.glProjectID(r)
	if err != nil {
		jsonError(w, "invalid project ID", http.StatusBadRequest)
		return
	}
	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid issue IID", http.StatusBadRequest)
		return
	}
	issue, err := gh.Client.IssueDetail(pid, iid)
	if err != nil {
		jsonError(w, "failed to fetch issue: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, issue)
}

// DirectIssueNotes handles GET /api/gitlab/projects/{pid}/issues/{iid}/notes
func (gh *GitLabHandlers) DirectIssueNotes(w http.ResponseWriter, r *http.Request) {
	pid, err := gh.glProjectID(r)
	if err != nil {
		jsonError(w, "invalid project ID", http.StatusBadRequest)
		return
	}
	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid issue IID", http.StatusBadRequest)
		return
	}
	notes, err := gh.Client.IssueNotes(pid, iid)
	if err != nil {
		jsonError(w, "failed to fetch issue notes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, notes)
}

// DirectAddIssueNote handles POST /api/gitlab/projects/{pid}/issues/{iid}/notes
func (gh *GitLabHandlers) DirectAddIssueNote(w http.ResponseWriter, r *http.Request) {
	pid, err := gh.glProjectID(r)
	if err != nil {
		jsonError(w, "invalid project ID", http.StatusBadRequest)
		return
	}
	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid issue IID", http.StatusBadRequest)
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Body == "" {
		jsonError(w, "body is required", http.StatusBadRequest)
		return
	}
	note, err := gh.Client.AddIssueNote(pid, iid, body.Body)
	if err != nil {
		jsonError(w, "failed to add note: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

// DirectUpdateIssue handles PUT /api/gitlab/projects/{pid}/issues/{iid}
func (gh *GitLabHandlers) DirectUpdateIssue(w http.ResponseWriter, r *http.Request) {
	pid, err := gh.glProjectID(r)
	if err != nil {
		jsonError(w, "invalid project ID", http.StatusBadRequest)
		return
	}
	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid issue IID", http.StatusBadRequest)
		return
	}
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

// DirectMRNotes handles GET /api/gitlab/projects/{pid}/merge-requests/{iid}/notes
func (gh *GitLabHandlers) DirectMRNotes(w http.ResponseWriter, r *http.Request) {
	pid, err := gh.glProjectID(r)
	if err != nil {
		jsonError(w, "invalid project ID", http.StatusBadRequest)
		return
	}
	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid MR IID", http.StatusBadRequest)
		return
	}
	notes, err := gh.Client.MRNotes(pid, iid)
	if err != nil {
		jsonError(w, "failed to fetch MR notes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, notes)
}

// DirectAddMRNote handles POST /api/gitlab/projects/{pid}/merge-requests/{iid}/notes
func (gh *GitLabHandlers) DirectAddMRNote(w http.ResponseWriter, r *http.Request) {
	pid, err := gh.glProjectID(r)
	if err != nil {
		jsonError(w, "invalid project ID", http.StatusBadRequest)
		return
	}
	iid, err := strconv.Atoi(mux.Vars(r)["iid"])
	if err != nil {
		jsonError(w, "invalid MR IID", http.StatusBadRequest)
		return
	}
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

// DirectCreateIssue handles POST /api/gitlab/projects/{pid}/issues
func (gh *GitLabHandlers) DirectCreateIssue(w http.ResponseWriter, r *http.Request) {
	pid, err := gh.glProjectID(r)
	if err != nil {
		jsonError(w, "invalid project ID", http.StatusBadRequest)
		return
	}
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

// DirectCreateMR handles POST /api/gitlab/projects/{pid}/merge-requests
func (gh *GitLabHandlers) DirectCreateMR(w http.ResponseWriter, r *http.Request) {
	pid, err := gh.glProjectID(r)
	if err != nil {
		jsonError(w, "invalid project ID", http.StatusBadRequest)
		return
	}
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
		jsonError(w, "failed to create MR: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(mr)
}

// GitLabProxy handles GET /api/gitlab/proxy?url=<encoded-url>
// Proxies files (images, attachments) from GitLab with authentication.
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
	// Uploaded files in GitLab are immutable (hash-addressed), safe to cache.
	w.Header().Set("Cache-Control", "public, max-age=86400, immutable")

	const maxProxySize = 50 << 20 // 50 MB
	io.Copy(w, io.LimitReader(body, maxProxySize))
}

// DirectProjectMembers handles GET /api/gitlab/projects/{pid}/members
func (gh *GitLabHandlers) DirectProjectMembers(w http.ResponseWriter, r *http.Request) {
	pid, err := gh.glProjectID(r)
	if err != nil {
		jsonError(w, "invalid project ID", http.StatusBadRequest)
		return
	}
	members, err := gh.Client.ProjectMembers(pid)
	if err != nil {
		jsonError(w, "failed to fetch members: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, members)
}

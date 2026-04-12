package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"devhub/internal/git"

	"github.com/gorilla/mux"
)

// GitHandlers manages REST endpoints for Git operations.
type GitHandlers struct {
	Base *Handlers
	Git  *git.GitService
}

// GitStatus handles GET /api/projects/{id}/git/status
func (gh *GitHandlers) GitStatus(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := gh.Git.Status(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, status)
}

// GitBranches handles GET /api/projects/{id}/git/branches
func (gh *GitHandlers) GitBranches(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	branches, err := gh.Git.BranchesDetailed(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, branches)
}

// GitLog handles GET /api/projects/{id}/git/log
func (gh *GitHandlers) GitLog(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	commits, err := gh.Git.Log(path, limit, offset)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, commits)
}

// GitGraph handles GET /api/projects/{id}/git/graph
func (gh *GitHandlers) GitGraph(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	topology, err := gh.Git.LogTopology(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, topology)
}

// GitLogMetadata handles GET /api/projects/{id}/git/log/metadata
func (gh *GitHandlers) GitLogMetadata(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	branch := r.URL.Query().Get("branch")
	metas, err := gh.Git.LogMetadata(path, limit, offset, branch)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, metas)
}

// GitBranchCommits handles GET /api/projects/{id}/git/branches/{name:.+}/commits
func (gh *GitHandlers) GitBranchCommits(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	branchName := mux.Vars(r)["name"]
	if branchName == "" {
		jsonError(w, "branch name required", http.StatusBadRequest)
		return
	}

	limit := 5
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	metas, err := gh.Git.LogMetadata(path, limit, 0, branchName)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, metas)
}

// GitDiff handles GET /api/projects/{id}/git/diff
func (gh *GitHandlers) GitDiff(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	file := r.URL.Query().Get("file")
	var diff string
	if file != "" {
		diff, err = gh.Git.DiffFile(path, file)
	} else {
		diff, err = gh.Git.Diff(path)
	}
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	const maxDiffBytes = 512 * 1024
	if len(diff) > maxDiffBytes {
		diff = diff[:maxDiffBytes] + "\n\n... (diff truncated, file too large to display)"
	}

	jsonResponse(w, map[string]string{"diff": diff})
}

// GitCommit handles POST /api/projects/{id}/git/commit
func (gh *GitHandlers) GitCommit(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var body struct {
		Message string   `json:"message"`
		Files   []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" || len(body.Files) == 0 {
		jsonError(w, "message and files required", http.StatusBadRequest)
		return
	}

	if err := gh.Git.CommitChanges(path, body.Message, body.Files); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	gh.Base.Hub.Broadcast(Event{
		Type:    "git_changed",
		Project: mux.Vars(r)["id"],
		Data:    "commit",
	})

	jsonResponse(w, map[string]string{"status": "ok"})
}

// GitStage handles POST /api/projects/{id}/git/stage
func (gh *GitHandlers) GitStage(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	var body struct {
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Files) == 0 {
		jsonError(w, "files required", http.StatusBadRequest)
		return
	}
	if err := gh.Git.StageFiles(path, body.Files); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "ok"})
}

// GitUnstage handles POST /api/projects/{id}/git/unstage
func (gh *GitHandlers) GitUnstage(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	var body struct {
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Files) == 0 {
		jsonError(w, "files required", http.StatusBadRequest)
		return
	}
	if err := gh.Git.UnstageFiles(path, body.Files); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "ok"})
}

// GitCheckout handles POST /api/projects/{id}/git/checkout
func (gh *GitHandlers) GitCheckout(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var body struct {
		Branch string `json:"branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Branch == "" {
		jsonError(w, "branch required", http.StatusBadRequest)
		return
	}

	if err := gh.Git.Checkout(path, body.Branch); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	gh.Base.Hub.Broadcast(Event{
		Type:    "git_changed",
		Project: mux.Vars(r)["id"],
		Data:    "checkout",
	})

	jsonResponse(w, map[string]string{"status": "ok"})
}

// GitPull handles POST /api/projects/{id}/git/pull
func (gh *GitHandlers) GitPull(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := gh.Git.Pull(path)
	if err != nil {
		jsonError(w, fmt.Sprintf("%s: %s", err.Error(), out), http.StatusInternalServerError)
		return
	}

	gh.Base.Hub.Broadcast(Event{
		Type:    "git_changed",
		Project: mux.Vars(r)["id"],
		Data:    "pull",
	})

	jsonResponse(w, map[string]string{"status": "ok", "output": out})
}

// GitPush handles POST /api/projects/{id}/git/push
func (gh *GitHandlers) GitPush(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := gh.Git.Push(path)
	if err != nil {
		jsonError(w, fmt.Sprintf("%s: %s", err.Error(), out), http.StatusInternalServerError)
		return
	}

	gh.Base.Hub.Broadcast(Event{
		Type:    "git_changed",
		Project: mux.Vars(r)["id"],
		Data:    "push",
	})

	jsonResponse(w, map[string]string{"status": "ok", "output": out})
}

// GitGenerateCommit handles POST /api/projects/{id}/git/generate-commit
func (gh *GitHandlers) GitGenerateCommit(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var body struct {
		Files []string `json:"files"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if len(body.Files) > 0 {
		if err := gh.Git.StageFiles(path, body.Files); err != nil {
			jsonError(w, "failed to stage files: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	message, err := gh.Git.GenerateCommitMessage(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"message": message})
}

// GitCommitDetail handles GET /api/projects/{id}/git/commits/{hash}
func (gh *GitHandlers) GitCommitDetail(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash := mux.Vars(r)["hash"]
	if hash == "" {
		jsonError(w, "commit hash required", http.StatusBadRequest)
		return
	}

	detail, err := gh.Git.CommitDetail(path, hash)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, detail)
}

// GitCommitDiff handles GET /api/projects/{id}/git/commits/{hash}/diff
func (gh *GitHandlers) GitCommitDiff(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash := mux.Vars(r)["hash"]
	if hash == "" {
		jsonError(w, "commit hash required", http.StatusBadRequest)
		return
	}

	file := r.URL.Query().Get("file")

	diff, err := gh.Git.CommitDiff(path, hash, file)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	const maxDiffBytes = 512 * 1024
	if len(diff) > maxDiffBytes {
		diff = diff[:maxDiffBytes] + "\n\n... (diff truncated, file too large to display)"
	}

	jsonResponse(w, map[string]string{"diff": diff})
}

// GitBlame handles GET /api/projects/{id}/git/blame?file=path/to/file
func (gh *GitHandlers) GitBlame(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		jsonError(w, "file parameter required", http.StatusBadRequest)
		return
	}

	entries, err := gh.Git.Blame(path, filePath)
	if err != nil {
		jsonError(w, "blame failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, entries)
}

// GitCherryPick handles POST /api/projects/{id}/git/cherry-pick
func (gh *GitHandlers) GitCherryPick(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	var body struct {
		Hash string `json:"hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Hash == "" {
		jsonError(w, "hash is required", http.StatusBadRequest)
		return
	}

	if err := gh.Git.CherryPick(path, body.Hash); err != nil {
		jsonError(w, "cherry-pick failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	gh.Base.Hub.Broadcast(Event{
		Type:    "git_changed",
		Project: mux.Vars(r)["id"],
		Data:    "cherry-pick",
	})

	jsonResponse(w, map[string]bool{"ok": true})
}

// --- Git Stash endpoints ---

// GitStashList handles GET /api/projects/{id}/git/stash
func (gh *GitHandlers) GitStashList(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	entries, err := gh.Git.StashList(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, entries)
}

// GitStashPush handles POST /api/projects/{id}/git/stash
func (gh *GitHandlers) GitStashPush(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var body struct {
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if err := gh.Git.StashPush(path, body.Message); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "ok"})
}

// GitStashApply handles POST /api/projects/{id}/git/stash/{index}/apply
func (gh *GitHandlers) GitStashApply(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(mux.Vars(r)["index"])
	if err != nil {
		jsonError(w, "invalid stash index", http.StatusBadRequest)
		return
	}

	if err := gh.Git.StashApply(path, index); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "ok"})
}

// GitStashPop handles POST /api/projects/{id}/git/stash/{index}/pop
func (gh *GitHandlers) GitStashPop(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(mux.Vars(r)["index"])
	if err != nil {
		jsonError(w, "invalid stash index", http.StatusBadRequest)
		return
	}

	if err := gh.Git.StashPop(path, index); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "ok"})
}

// GitStashDrop handles DELETE /api/projects/{id}/git/stash/{index}
func (gh *GitHandlers) GitStashDrop(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(mux.Vars(r)["index"])
	if err != nil {
		jsonError(w, "invalid stash index", http.StatusBadRequest)
		return
	}

	if err := gh.Git.StashDrop(path, index); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "ok"})
}

// GitStashDiff handles GET /api/projects/{id}/git/stash/{index}/diff
func (gh *GitHandlers) GitStashDiff(w http.ResponseWriter, r *http.Request) {
	path, err := gh.Base.projectPath(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(mux.Vars(r)["index"])
	if err != nil {
		jsonError(w, "invalid stash index", http.StatusBadRequest)
		return
	}

	diff, err := gh.Git.StashDiff(path, index)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"diff": diff})
}

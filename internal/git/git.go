package git

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"devhub/internal/runner"
)

// GitStatus represents the current state of a git repository.
type GitStatus struct {
	Branch    string   `json:"branch"`
	Modified  []string `json:"modified"`
	Staged    []string `json:"staged"`
	Untracked []string `json:"untracked"`
	Ahead     int      `json:"ahead"`
	Behind    int      `json:"behind"`
}

// Commit represents a single git commit entry.
type Commit struct {
	Hash      string     `json:"hash"`
	ShortHash string     `json:"short_hash"`
	Message   string     `json:"message"`
	Author    string     `json:"author"`
	Date      string     `json:"date"`
	Refs      []string   `json:"refs"`
	Parents   []string   `json:"parents"`
	Graph     string     `json:"graph,omitempty"`
	GraphOnly bool       `json:"graph_only,omitempty"`
	GraphData *GraphData `json:"graph_data,omitempty"`
}

// CommitDetailInfo holds detailed information about a single commit.
type CommitDetailInfo struct {
	Hash    string       `json:"hash"`
	Message string       `json:"message"`
	Author  string       `json:"author"`
	Email   string       `json:"email"`
	Date    string       `json:"date"`
	Body    string       `json:"body"`
	Files   []FileChange `json:"files"`
	Stats   string       `json:"stats"`
}

// FileChange represents a single file changed in a commit.
type FileChange struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

// BranchInfo holds detailed information about a git branch.
type BranchInfo struct {
	Name      string `json:"name"`
	ShortHash string `json:"short_hash"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	Date      string `json:"date"`
	IsCurrent bool   `json:"is_current"`
	Ahead     int    `json:"ahead"`
	Behind    int    `json:"behind"`
	IsMerged  bool   `json:"is_merged"`
}

// GitService provides git operations using a CommandRunner.
type GitService struct {
	runner runner.CommandRunner
}

// NewGitService creates a new GitService with the given runner.
func NewGitService(r runner.CommandRunner) *GitService {
	return &GitService{runner: r}
}

// Status returns the current git status of the repository at dir.
func (g *GitService) Status(dir string) (*GitStatus, error) {
	st := &GitStatus{}

	// Current branch
	branch, err := g.runner.Run(dir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	st.Branch = strings.TrimSpace(branch)

	// Ahead/behind
	revList, _ := g.runner.Run(dir, "git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	parts := strings.Fields(strings.TrimSpace(revList))
	if len(parts) == 2 {
		st.Ahead, _ = strconv.Atoi(parts[0])
		st.Behind, _ = strconv.Atoi(parts[1])
	}

	// Porcelain status for modified/staged
	porcelain, err := g.runner.Run(dir, "git", "status", "--porcelain")
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(porcelain, "\n") {
		if len(line) < 4 {
			continue
		}
		indexStatus := line[0]
		workTreeStatus := line[1]
		file := strings.TrimSpace(line[3:])

		if indexStatus != ' ' && indexStatus != '?' {
			st.Staged = append(st.Staged, file)
		}
		if workTreeStatus != ' ' && workTreeStatus != '?' {
			st.Modified = append(st.Modified, file)
		}
		// Untracked files
		if indexStatus == '?' {
			st.Untracked = append(st.Untracked, file)
		}
	}

	return st, nil
}

// Branches returns a list of local branch names.
func (g *GitService) Branches(dir string) ([]string, error) {
	out, err := g.runner.Run(dir, "git", "branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	var branches []string
	for _, b := range strings.Split(strings.TrimSpace(out), "\n") {
		b = strings.TrimSpace(b)
		if b != "" {
			branches = append(branches, b)
		}
	}
	return branches, nil
}

// Log returns commits across all branches with parent hash information.
// Uses --graph for ASCII art positioning and %P for parent hashes.
// offset=0 means start from HEAD; limit controls how many commits to return.
func (g *GitService) Log(dir string, limit int, offset int) ([]Commit, error) {
	args := []string{"log", "--all", "--graph", "--topo-order",
		"--format=%H|%h|%s|%an|%ar|%D|%P", "-n", strconv.Itoa(limit)}
	if offset > 0 {
		args = append(args, "--skip", strconv.Itoa(offset))
	}
	out, err := g.runner.Run(dir, "git", args...)
	if err != nil {
		return nil, err
	}

	hashRe := regexp.MustCompile(`[0-9a-f]{40}`)

	var commits []Commit
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}

		loc := hashRe.FindStringIndex(line)
		if loc == nil {
			graphStr := strings.TrimRight(line, " ")
			if graphStr != "" {
				commits = append(commits, Commit{
					Graph:     graphStr,
					GraphOnly: true,
				})
			}
			continue
		}

		graphPrefix := line[:loc[0]]
		data := line[loc[0]:]

		parts := strings.SplitN(data, "|", 7)
		if len(parts) < 7 {
			continue
		}

		var refs []string
		if strings.TrimSpace(parts[5]) != "" {
			for _, ref := range strings.Split(parts[5], ", ") {
				ref = strings.TrimSpace(ref)
				if ref != "" {
					refs = append(refs, ref)
				}
			}
		}

		var parents []string
		if strings.TrimSpace(parts[6]) != "" {
			for _, p := range strings.Fields(parts[6]) {
				parents = append(parents, p)
			}
		}

		commits = append(commits, Commit{
			Hash:      parts[0],
			ShortHash: parts[1],
			Message:   parts[2],
			Author:    parts[3],
			Date:      parts[4],
			Refs:      refs,
			Parents:   parents,
			Graph:     strings.TrimRight(graphPrefix, " "),
		})
	}
	return commits, nil
}


// TopologyNode содержит минимальные данные коммита для построения графа.
type TopologyNode struct {
	Hash    string   `json:"id"`
	Parents []string `json:"parents"`
}

// LogTopology возвращает полную топологию коммитов (только hash и parents).
func (g *GitService) LogTopology(dir string) ([]TopologyNode, error) {
	out, err := g.runner.Run(dir, "git", "log", "--all", "--topo-order", "--format=%H|%P")
	if err != nil {
		return nil, err
	}

	var nodes []TopologyNode
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 1 || parts[0] == "" {
			continue
		}

		var parents []string
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			for _, p := range strings.Fields(parts[1]) {
				parents = append(parents, p)
			}
		}

		nodes = append(nodes, TopologyNode{
			Hash:    parts[0],
			Parents: parents,
		})
	}
	return nodes, nil
}

// CommitMeta содержит метаданные коммита без графа.
type CommitMeta struct {
	Hash      string   `json:"hash"`
	ShortHash string   `json:"short_hash"`
	Message   string   `json:"message"`
	Author    string   `json:"author"`
	Date      string   `json:"date"`
	Refs      []string `json:"refs"`
}

// LogMetadata возвращает метаданные коммитов порциями (без графа).
// Если branch не пустой, фильтрует по этой ветке вместо --all.
func (g *GitService) LogMetadata(dir string, limit int, offset int, branch string) ([]CommitMeta, error) {
	ref := "--all"
	if branch != "" {
		ref = branch
	}
	args := []string{"log", ref, "--topo-order",
		"--format=%H|%h|%s|%an|%ar|%D", "-n", strconv.Itoa(limit)}
	if offset > 0 {
		args = append(args, "--skip", strconv.Itoa(offset))
	}
	out, err := g.runner.Run(dir, "git", args...)
	if err != nil {
		return nil, err
	}

	var metas []CommitMeta
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 6 {
			continue
		}

		var refs []string
		if strings.TrimSpace(parts[5]) != "" {
			for _, ref := range strings.Split(parts[5], ", ") {
				ref = strings.TrimSpace(ref)
				if ref != "" {
					refs = append(refs, ref)
				}
			}
		}

		metas = append(metas, CommitMeta{
			Hash:      parts[0],
			ShortHash: parts[1],
			Message:   parts[2],
			Author:    parts[3],
			Date:      parts[4],
			Refs:      refs,
		})
	}
	return metas, nil
}

// Diff returns the full diff of unstaged changes.
func (g *GitService) Diff(dir string) (string, error) {
	return g.runner.Run(dir, "git", "diff")
}

// DiffFile returns the diff for a single file.
// For untracked files, uses --no-index against /dev/null to show full content.
func (g *GitService) DiffFile(dir string, file string) (string, error) {
	out, err := g.runner.Run(dir, "git", "diff", "--", file)
	if err != nil {
		return "", err
	}
	// If diff is empty, the file might be untracked — try --no-index
	if strings.TrimSpace(out) == "" {
		noIndex, niErr := g.runner.Run(dir, "git", "diff", "--no-index", "--", "/dev/null", file)
		// --no-index exits with code 1 when there are differences, which is normal
		if niErr != nil && strings.TrimSpace(noIndex) == "" {
			return "", nil
		}
		if noIndex != "" {
			return noIndex, nil
		}
	}
	return out, nil
}

// StageFiles runs git add on the given files.
func (g *GitService) StageFiles(dir string, files []string) error {
	args := append([]string{"add", "--"}, files...)
	_, err := g.runner.Run(dir, "git", args...)
	return err
}

// UnstageFiles runs git reset HEAD on the given files.
func (g *GitService) UnstageFiles(dir string, files []string) error {
	args := append([]string{"reset", "HEAD", "--"}, files...)
	_, err := g.runner.Run(dir, "git", args...)
	return err
}

// CommitChanges stages the given files and creates a commit.
func (g *GitService) CommitChanges(dir string, message string, files []string) error {
	args := append([]string{"add", "--"}, files...)
	if _, err := g.runner.Run(dir, "git", args...); err != nil {
		return err
	}
	_, err := g.runner.Run(dir, "git", "commit", "-m", message)
	return err
}

// Checkout switches to the given branch.
func (g *GitService) Checkout(dir string, branch string) error {
	_, err := g.runner.Run(dir, "git", "checkout", branch)
	return err
}

// Pull runs git pull and returns the output.
func (g *GitService) Pull(dir string) (string, error) {
	return g.runner.Run(dir, "git", "pull")
}

// Push runs git push and returns the output.
func (g *GitService) Push(dir string) (string, error) {
	return g.runner.Run(dir, "git", "push")
}

// GenerateCommitMessage generates a commit message for staged changes using Claude CLI.
func (g *GitService) GenerateCommitMessage(dir string) (string, error) {
	// Check that claude CLI exists
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude CLI not found")
	}

	// Get staged diff
	diff, err := g.runner.Run(dir, "git", "diff", "--staged")
	if err != nil {
		return "", fmt.Errorf("git diff --staged failed: %w", err)
	}
	diff = strings.TrimSpace(diff)
	if diff == "" {
		return "", fmt.Errorf("no staged changes")
	}

	// Build prompt
	prompt := fmt.Sprintf(
		"Напиши коммит для этих изменений в формате: <тип>: описание\n\n"+
			"Типы: feature, fix, refactor, style, docs, test, chore, build, ci, perf, merge\n\n"+
			"Только одну строку, на русском, без кавычек.\n\nDiff:\n%s", diff)

	// Run claude CLI with 30s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, claudePath, "--print", "-p", prompt)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("claude CLI timed out after 30s")
		}
		return "", fmt.Errorf("claude CLI error: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// CommitDetail returns detailed information about a specific commit.
func (g *GitService) CommitDetail(dir string, hash string) (*CommitDetailInfo, error) {
	// Validate hash: only hex characters allowed
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return nil, fmt.Errorf("invalid commit hash")
		}
	}

	// Get commit details
	showOut, err := g.runner.Run(dir, "git", "show", "--stat",
		"--format=%H|%s|%an|%ae|%ai|%b", hash)
	if err != nil {
		return nil, fmt.Errorf("git show failed: %w", err)
	}

	// The first line contains the formatted commit info.
	// Subsequent lines are the stat output until the diff starts.
	lines := strings.SplitN(strings.TrimSpace(showOut), "\n", 2)
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty git show output")
	}

	parts := strings.SplitN(lines[0], "|", 6)
	if len(parts) < 5 {
		return nil, fmt.Errorf("unexpected git show format")
	}

	detail := &CommitDetailInfo{
		Hash:    parts[0],
		Message: parts[1],
		Author:  parts[2],
		Email:   parts[3],
		Date:    parts[4],
	}
	if len(parts) >= 6 {
		detail.Body = strings.TrimSpace(parts[5])
	}

	// Stats: everything after the first line
	if len(lines) > 1 {
		detail.Stats = strings.TrimSpace(lines[1])
	}

	// Get file changes
	treeOut, err := g.runner.Run(dir, "git", "diff-tree", "--no-commit-id", "-r",
		"--name-status", hash)
	if err != nil {
		return nil, fmt.Errorf("git diff-tree failed: %w", err)
	}

	for _, line := range strings.Split(strings.TrimSpace(treeOut), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fileParts := strings.SplitN(line, "\t", 2)
		if len(fileParts) < 2 {
			continue
		}
		detail.Files = append(detail.Files, FileChange{
			Status: fileParts[0],
			Path:   fileParts[1],
		})
	}

	return detail, nil
}

// BranchesDetailed returns detailed information about all branches.
func (g *GitService) BranchesDetailed(dir string) ([]BranchInfo, error) {
	// Get current branch name
	currentBranch, err := g.runner.Run(dir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	currentBranch = strings.TrimSpace(currentBranch)

	// Get all branches with details
	out, err := g.runner.Run(dir, "git", "branch", "-a",
		"--format=%(refname:short)|%(objectname:short)|%(subject)|%(authorname)|%(committerdate:relative)|%(upstream:track)")
	if err != nil {
		return nil, err
	}

	// Get merged branches list
	mergedOut, _ := g.runner.Run(dir, "git", "branch", "--merged")
	mergedSet := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(mergedOut), "\n") {
		b := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if b != "" {
			mergedSet[b] = true
		}
	}

	var branches []BranchInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 5 {
			continue
		}

		name := parts[0]

		// Skip HEAD symbolic ref entries like "origin/HEAD"
		if strings.HasSuffix(name, "/HEAD") {
			continue
		}

		bi := BranchInfo{
			Name:      name,
			ShortHash: parts[1],
			Message:   parts[2],
			Author:    parts[3],
			Date:      parts[4],
			IsCurrent: name == currentBranch,
			IsMerged:  mergedSet[name],
		}

		// Get ahead/behind relative to current branch (skip for current branch itself)
		if name != currentBranch {
			revOut, revErr := g.runner.Run(dir, "git", "rev-list", "--left-right", "--count",
				currentBranch+"..."+name)
			if revErr == nil {
				countParts := strings.Fields(strings.TrimSpace(revOut))
				if len(countParts) == 2 {
					bi.Behind, _ = strconv.Atoi(countParts[0])
					bi.Ahead, _ = strconv.Atoi(countParts[1])
				}
			}
		}

		branches = append(branches, bi)
	}

	return branches, nil
}

// StashEntry represents a single git stash entry.
type StashEntry struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
	Date    string `json:"date"`
}

// StashList returns all stash entries for the repository at dir.
func (g *GitService) StashList(dir string) ([]StashEntry, error) {
	out, err := g.runner.Run(dir, "git", "stash", "list", "--format=%gd|%gs|%ci")
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return []StashEntry{}, nil
	}
	var entries []StashEntry
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		idx := 0
		fmt.Sscanf(parts[0], "stash@{%d}", &idx)
		entries = append(entries, StashEntry{
			Index:   idx,
			Message: parts[1],
			Date:    parts[2],
		})
	}
	return entries, nil
}

// StashPush creates a new stash entry with an optional message.
func (g *GitService) StashPush(dir, message string) error {
	args := []string{"stash", "push"}
	if message != "" {
		args = append(args, "-m", message)
	}
	_, err := g.runner.Run(dir, "git", args...)
	return err
}

// StashApply applies the stash entry at the given index without removing it.
func (g *GitService) StashApply(dir string, index int) error {
	_, err := g.runner.Run(dir, "git", "stash", "apply", fmt.Sprintf("stash@{%d}", index))
	return err
}

// StashPop applies the stash entry at the given index and removes it.
func (g *GitService) StashPop(dir string, index int) error {
	_, err := g.runner.Run(dir, "git", "stash", "pop", fmt.Sprintf("stash@{%d}", index))
	return err
}

// StashDrop removes the stash entry at the given index.
func (g *GitService) StashDrop(dir string, index int) error {
	_, err := g.runner.Run(dir, "git", "stash", "drop", fmt.Sprintf("stash@{%d}", index))
	return err
}

// StashDiff returns the diff for the stash entry at the given index.
func (g *GitService) StashDiff(dir string, index int) (string, error) {
	out, err := g.runner.Run(dir, "git", "stash", "show", "-p", fmt.Sprintf("stash@{%d}", index))
	if err != nil {
		return "", err
	}
	return out, nil
}

// CommitDiff returns the diff of a specific commit, optionally filtered to a single file.
func (g *GitService) CommitDiff(dir string, hash string, file string) (string, error) {
	// Validate hash
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", fmt.Errorf("invalid commit hash")
		}
	}

	var out string
	var err error
	if file != "" {
		out, err = g.runner.Run(dir, "git", "diff", hash+"~1", hash, "--", file)
	} else {
		out, err = g.runner.Run(dir, "git", "diff", hash+"~1", hash)
	}
	// Fallback for initial commit (no parent)
	if err != nil {
		if file != "" {
			out, err = g.runner.Run(dir, "git", "show", "--format=", hash, "--", file)
		} else {
			out, err = g.runner.Run(dir, "git", "show", "--format=", hash)
		}
	}
	if err != nil {
		return "", err
	}
	return out, nil
}

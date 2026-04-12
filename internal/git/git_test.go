package git

import (
	"testing"

	"devhub/internal/testutil"
)

func TestStatus_Clean(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: "main\n"},
		{Output: "0\t0\n"},
		{Output: ""},
	}}

	svc := NewGitService(mock)
	st, err := svc.Status("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Branch != "main" {
		t.Errorf("expected branch main, got %s", st.Branch)
	}
	if len(st.Modified) != 0 {
		t.Errorf("expected no modified files, got %v", st.Modified)
	}
	if len(st.Staged) != 0 {
		t.Errorf("expected no staged files, got %v", st.Staged)
	}
	if st.Ahead != 0 || st.Behind != 0 {
		t.Errorf("expected ahead=0 behind=0, got ahead=%d behind=%d", st.Ahead, st.Behind)
	}
}

func TestStatus_Modified(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: "develop\n"},
		{Output: "2\t1\n"},
		{Output: " M file.go\n M main.go\n"},
	}}

	svc := NewGitService(mock)
	st, err := svc.Status("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(st.Modified) != 2 {
		t.Fatalf("expected 2 modified files, got %d: %v", len(st.Modified), st.Modified)
	}
	if st.Modified[0] != "file.go" || st.Modified[1] != "main.go" {
		t.Errorf("unexpected modified: %v", st.Modified)
	}
	if st.Ahead != 2 || st.Behind != 1 {
		t.Errorf("expected ahead=2 behind=1, got ahead=%d behind=%d", st.Ahead, st.Behind)
	}
}

func TestStatus_Staged(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: "main\n"},
		{Output: ""},
		{Output: "M  file.go\nA  new.go\n"},
	}}

	svc := NewGitService(mock)
	st, err := svc.Status("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(st.Staged) != 2 {
		t.Fatalf("expected 2 staged files, got %d: %v", len(st.Staged), st.Staged)
	}
	if st.Staged[0] != "file.go" || st.Staged[1] != "new.go" {
		t.Errorf("unexpected staged: %v", st.Staged)
	}
}

func TestBranches(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: "main\ndevelop\nfeature/auth\n"},
	}}

	svc := NewGitService(mock)
	branches, err := svc.Branches("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(branches) != 3 {
		t.Fatalf("expected 3 branches, got %d", len(branches))
	}
	if branches[0] != "main" || branches[1] != "develop" || branches[2] != "feature/auth" {
		t.Errorf("unexpected branches: %v", branches)
	}
}

func TestBranchesDetailed(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: "main\n"},
		{Output: "main|abc1234|initial commit|John|2 hours ago|\ndevelop|def5678|add feature|Jane|1 hour ago|\n"},
		{Output: "* main\n  develop\n"},
		{Output: "0\t3\n"},
	}}

	svc := NewGitService(mock)
	branches, err := svc.BranchesDetailed("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(branches))
	}

	if !branches[0].IsCurrent {
		t.Error("expected main to be current")
	}
	if branches[1].IsCurrent {
		t.Error("expected develop to not be current")
	}
	if branches[1].Ahead != 3 {
		t.Errorf("expected develop ahead=3, got %d", branches[1].Ahead)
	}
	if !branches[0].IsMerged || !branches[1].IsMerged {
		t.Error("expected both branches to be merged")
	}
}

func TestLog(t *testing.T) {
	logOutput := `* abc1234567890abc1234567890abc1234567890ab|abc1234|initial commit|John|2 hours ago|HEAD -> main|
* def5678901234def5678901234def5678901234de|def5678|add feature|Jane|1 hour ago||abc1234567890abc1234567890abc1234567890ab`

	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: logOutput},
	}}

	svc := NewGitService(mock)
	commits, err := svc.Log("/test", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	if commits[0].ShortHash != "abc1234" {
		t.Errorf("expected short hash abc1234, got %s", commits[0].ShortHash)
	}
	if commits[0].Graph != "*" {
		t.Errorf("expected graph *, got %q", commits[0].Graph)
	}
	if len(commits[0].Refs) != 1 {
		t.Errorf("expected 1 ref, got %d: %v", len(commits[0].Refs), commits[0].Refs)
	}
}

func TestLog_WithParents(t *testing.T) {
	logOutput := `* abc1234567890abc1234567890abc1234567890ab|abc1234|initial commit|John|2 hours ago||
* def5678901234def5678901234def5678901234de|def5678|add feature|Jane|1 hour ago|HEAD -> main|abc1234567890abc1234567890abc1234567890ab
*   cde9012345678cde9012345678cde9012345678cd|cde9012|merge branch|Bob|30 min ago||def5678901234def5678901234def5678901234de abc1234567890abc1234567890abc1234567890ab`

	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: logOutput},
	}}

	svc := NewGitService(mock)
	commits, err := svc.Log("/test", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(commits))
	}
	if len(commits[0].Parents) != 0 {
		t.Errorf("root commit should have 0 parents, got %d", len(commits[0].Parents))
	}
	if len(commits[1].Parents) != 1 {
		t.Errorf("expected 1 parent, got %d", len(commits[1].Parents))
	}
	if len(commits[2].Parents) != 2 {
		t.Errorf("merge commit should have 2 parents, got %d", len(commits[2].Parents))
	}
}

func TestDiff(t *testing.T) {
	diffOutput := "diff --git a/file.go b/file.go\n--- a/file.go\n+++ b/file.go\n@@ -1 +1 @@\n-old\n+new\n"

	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: diffOutput},
	}}

	svc := NewGitService(mock)
	diff, err := svc.Diff("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != diffOutput {
		t.Errorf("unexpected diff output")
	}
}

func TestDiffFile(t *testing.T) {
	diffOutput := "diff --git a/main.go b/main.go\n"

	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: diffOutput},
	}}

	svc := NewGitService(mock)
	diff, err := svc.DiffFile("/test", "main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != diffOutput {
		t.Errorf("unexpected diff output")
	}
}

func TestCommitChanges(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: ""},
		{Output: "[main abc] msg\n"},
	}}

	svc := NewGitService(mock)
	err := svc.CommitChanges("/test", "msg", []string{"file.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStageFiles(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: ""},
	}}

	svc := NewGitService(mock)
	err := svc.StageFiles("/test", []string{"a.go", "b.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnstageFiles(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: ""},
	}}

	svc := NewGitService(mock)
	err := svc.UnstageFiles("/test", []string{"a.go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckout(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: "Switched to branch 'develop'\n"},
	}}

	svc := NewGitService(mock)
	err := svc.Checkout("/test", "develop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommitDetail(t *testing.T) {
	showOutput := "abc123def456abc123def456abc123def456abc123|fix bug|John|john@test.com|2024-01-15 10:00:00 +0300|\n file.go | 2 +-\n 1 file changed"
	treeOutput := "M\tfile.go\n"

	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: showOutput},
		{Output: treeOutput},
	}}

	svc := NewGitService(mock)
	detail, err := svc.CommitDetail("/test", "abc123def456abc123def456abc123def456abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Message != "fix bug" {
		t.Errorf("expected message 'fix bug', got %s", detail.Message)
	}
	if detail.Author != "John" {
		t.Errorf("expected author John, got %s", detail.Author)
	}
	if detail.Email != "john@test.com" {
		t.Errorf("expected email john@test.com, got %s", detail.Email)
	}
	if len(detail.Files) != 1 {
		t.Fatalf("expected 1 file change, got %d", len(detail.Files))
	}
	if detail.Files[0].Status != "M" || detail.Files[0].Path != "file.go" {
		t.Errorf("unexpected file change: %+v", detail.Files[0])
	}
}

func TestCommitDetail_InvalidHash(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{}}

	svc := NewGitService(mock)
	_, err := svc.CommitDetail("/test", "not-a-valid-hash!")
	if err == nil {
		t.Error("expected error for invalid hash, got nil")
	}
}

func TestCommitDiff(t *testing.T) {
	diffOutput := "commit abc123\nAuthor: John\n\ndiff content\n"

	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: diffOutput},
	}}

	svc := NewGitService(mock)
	diff, err := svc.CommitDiff("/test", "abc123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != diffOutput {
		t.Errorf("unexpected diff output")
	}
}

func TestBlame(t *testing.T) {
	// Simulate git blame --porcelain output with two commits
	hash1 := "abc1234567890abc1234567890abc1234567890a" // 40 chars
	hash2 := "def5678901234def5678901234def5678901234d" // 40 chars
	blameOutput := hash1 + " 1 1 2\n" +
		"author John Doe\n" +
		"author-mail <john@example.com>\n" +
		"author-time 1700000000\n" +
		"author-tz +0000\n" +
		"committer John Doe\n" +
		"committer-mail <john@example.com>\n" +
		"committer-time 1700000000\n" +
		"committer-tz +0000\n" +
		"summary initial commit\n" +
		"filename main.go\n" +
		"\tpackage main\n" +
		hash1 + " 2 2\n" +
		"\timport \"fmt\"\n" +
		hash2 + " 3 3 1\n" +
		"author Jane Smith\n" +
		"author-mail <jane@example.com>\n" +
		"author-time 1700100000\n" +
		"author-tz +0000\n" +
		"committer Jane Smith\n" +
		"committer-mail <jane@example.com>\n" +
		"committer-time 1700100000\n" +
		"committer-tz +0000\n" +
		"summary added function\n" +
		"filename main.go\n" +
		"\tfunc main() {}\n"

	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: blameOutput},
	}}

	svc := NewGitService(mock)
	entries, err := svc.Blame("/test", "main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 blame entries, got %d", len(entries))
	}

	// First entry: lines 1-2, John Doe
	if entries[0].LineStart != 1 || entries[0].LineEnd != 2 {
		t.Errorf("entry[0] lines: expected 1-2, got %d-%d", entries[0].LineStart, entries[0].LineEnd)
	}
	if entries[0].Author != "John Doe" {
		t.Errorf("entry[0] author: expected 'John Doe', got %q", entries[0].Author)
	}
	if entries[0].ShortHash != hash1[:7] {
		t.Errorf("entry[0] short_hash: expected %q, got %q", hash1[:7], entries[0].ShortHash)
	}
	if entries[0].Message != "initial commit" {
		t.Errorf("entry[0] message: expected 'initial commit', got %q", entries[0].Message)
	}

	// Second entry: line 3, Jane Smith
	if entries[1].LineStart != 3 || entries[1].LineEnd != 3 {
		t.Errorf("entry[1] lines: expected 3-3, got %d-%d", entries[1].LineStart, entries[1].LineEnd)
	}
	if entries[1].Author != "Jane Smith" {
		t.Errorf("entry[1] author: expected 'Jane Smith', got %q", entries[1].Author)
	}
	if entries[1].Message != "added function" {
		t.Errorf("entry[1] message: expected 'added function', got %q", entries[1].Message)
	}
}

func TestBlame_EmptyOutput(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: ""},
	}}

	svc := NewGitService(mock)
	entries, err := svc.Blame("/test", "empty.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty output, got %d", len(entries))
	}
}

func TestCommitDiff_WithFile(t *testing.T) {
	diffOutput := "diff for specific file\n"

	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: diffOutput},
	}}

	svc := NewGitService(mock)
	diff, err := svc.CommitDiff("/test", "abc123", "main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != diffOutput {
		t.Errorf("unexpected diff output")
	}
}

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

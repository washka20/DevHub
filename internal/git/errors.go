package git

import "fmt"

// InvalidHashError indicates that a commit hash contains invalid characters.
type InvalidHashError struct {
	Hash string
}

func (e *InvalidHashError) Error() string {
	return fmt.Sprintf("invalid commit hash: %s", e.Hash)
}

// NoStagedChangesError indicates there are no staged changes to commit.
type NoStagedChangesError struct{}

func (e *NoStagedChangesError) Error() string {
	return "no staged changes to commit"
}

// BranchNotFoundError indicates that the specified branch does not exist.
type BranchNotFoundError struct {
	Branch string
}

func (e *BranchNotFoundError) Error() string {
	return fmt.Sprintf("branch not found: %s", e.Branch)
}

// CherryPickConflictError indicates a conflict during cherry-pick.
type CherryPickConflictError struct {
	Hash   string
	Output string
}

func (e *CherryPickConflictError) Error() string {
	return fmt.Sprintf("cherry-pick conflict on %s: %s", e.Hash, e.Output)
}

// BlameError wraps a failure during git blame.
type BlameError struct {
	File string
	Err  error
}

func (e *BlameError) Error() string {
	return fmt.Sprintf("blame failed for %s: %v", e.File, e.Err)
}

func (e *BlameError) Unwrap() error { return e.Err }

package git

import "fmt"

// InvalidHashError indicates that a commit hash has invalid format.
type InvalidHashError struct {
	Hash string
}

func (e *InvalidHashError) Error() string {
	return fmt.Sprintf("invalid commit hash: %s", e.Hash)
}

// NoStagedChangesError indicates that there are no staged changes to commit.
type NoStagedChangesError struct{}

func (e *NoStagedChangesError) Error() string {
	return "no staged changes"
}

// BranchNotFoundError indicates that a branch was not found.
type BranchNotFoundError struct {
	Branch string
}

func (e *BranchNotFoundError) Error() string {
	return fmt.Sprintf("branch not found: %s", e.Branch)
}

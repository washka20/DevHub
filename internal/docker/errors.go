package docker

import "fmt"

// ComposeFileNotFoundError indicates that a docker-compose file was not found.
type ComposeFileNotFoundError struct {
	Dir string
}

func (e *ComposeFileNotFoundError) Error() string {
	return fmt.Sprintf("docker-compose file not found in %s", e.Dir)
}

// ContainerNotFoundError indicates that the specified container does not exist.
type ContainerNotFoundError struct {
	Name string
}

func (e *ContainerNotFoundError) Error() string {
	return fmt.Sprintf("container not found: %s", e.Name)
}

// ActionError wraps a failure during a docker action on a container.
type ActionError struct {
	Container string
	Action    string
	Err       error
}

func (e *ActionError) Error() string {
	return fmt.Sprintf("docker %s %s: %v", e.Action, e.Container, e.Err)
}

func (e *ActionError) Unwrap() error { return e.Err }

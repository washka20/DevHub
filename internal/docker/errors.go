package docker

import "fmt"

// ComposeFileNotFoundError indicates that a docker-compose file was not found.
type ComposeFileNotFoundError struct {
	Path string
}

func (e *ComposeFileNotFoundError) Error() string {
	return fmt.Sprintf("docker-compose file not found: %s", e.Path)
}

// ContainerNotFoundError indicates that a container was not found.
type ContainerNotFoundError struct {
	Name string
}

func (e *ContainerNotFoundError) Error() string {
	return fmt.Sprintf("container not found: %s", e.Name)
}

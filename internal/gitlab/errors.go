package gitlab

import "fmt"

// ProjectNotFoundError indicates that a GitLab project was not found.
type ProjectNotFoundError struct {
	Path string
}

func (e *ProjectNotFoundError) Error() string {
	return fmt.Sprintf("gitlab project not found: %s", e.Path)
}

// APIError represents an unexpected HTTP response from the GitLab API.
type APIError struct {
	Endpoint   string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gitlab API %s returned %d: %s", e.Endpoint, e.StatusCode, e.Body)
}

// AuthenticationError indicates that GitLab authentication failed.
type AuthenticationError struct{}

func (e *AuthenticationError) Error() string {
	return "gitlab authentication failed"
}

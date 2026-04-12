package gitlab

import "fmt"

// ProjectNotFoundError indicates that a GitLab project was not found.
type ProjectNotFoundError struct {
	Path string
}

func (e *ProjectNotFoundError) Error() string {
	return fmt.Sprintf("gitlab project not found: %s", e.Path)
}

// APIError represents a GitLab API error with HTTP status code.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gitlab API error %d: %s", e.StatusCode, e.Message)
}

// AuthenticationError indicates that GitLab authentication failed.
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("gitlab authentication failed: %s", e.Message)
	}
	return "gitlab authentication failed"
}

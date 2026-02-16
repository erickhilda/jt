package jira

import (
	"errors"
	"fmt"
)

// ErrUnauthorized indicates invalid or missing credentials.
var ErrUnauthorized = errors.New("unauthorized: check your email and API token")

// APIError represents a non-success HTTP response from Jira.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("jira API error (HTTP %d): %s", e.StatusCode, e.Message)
}

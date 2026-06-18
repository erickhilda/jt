package confluence

import (
	"errors"
	"fmt"
)

// ErrUnauthorized indicates invalid or missing credentials (HTTP 401).
var ErrUnauthorized = errors.New("unauthorized: check your email and Atlassian API token")

// ErrForbidden indicates the token is valid but lacks Confluence access (HTTP 403).
var ErrForbidden = errors.New("forbidden: token lacks Confluence access (use an unscoped Atlassian API token, or one with a Confluence read scope)")

// ErrNotFound indicates the requested resource does not exist (HTTP 404).
var ErrNotFound = errors.New("not found")

// APIError represents a non-success HTTP response from Confluence.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("confluence API error (HTTP %d): %s", e.StatusCode, e.Message)
}

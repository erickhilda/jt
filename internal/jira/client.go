package jira

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is an authenticated Jira Cloud HTTP client.
type Client struct {
	baseURL    string
	authHeader string
	http       *http.Client
}

// NewClient creates a Jira client with Basic auth (email:apiToken).
func NewClient(baseURL, email, token string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	creds := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
	return &Client{
		baseURL:    baseURL,
		authHeader: "Basic " + creds,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Myself calls GET /rest/api/3/myself to verify credentials and retrieve
// the authenticated user's profile.
func (c *Client) Myself() (*User, error) {
	resp, err := c.do(http.MethodGet, "/rest/api/3/myself", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decoding user response: %w", err)
	}
	return &user, nil
}

// GetIssue fetches a Jira issue by key using a two-pass decode to handle
// custom fields (sprint, epic) identified via the expand=names response.
func (c *Client) GetIssue(key string) (*Issue, error) {
	resp, err := c.do(http.MethodGet, "/rest/api/3/issue/"+key+"?expand=names", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// First pass: decode into raw to preserve fields JSON and get names map.
	var raw IssueRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decoding issue (raw): %w", err)
	}

	// Second pass: decode the known fields.
	var fields IssueFields
	if err := json.Unmarshal(raw.Fields, &fields); err != nil {
		return nil, fmt.Errorf("decoding issue fields: %w", err)
	}

	issue := &Issue{
		Key:    raw.Key,
		Fields: fields,
	}

	// Extract custom fields using the names map.
	extractCustomFields(issue, raw.Fields, raw.Names)

	return issue, nil
}

// extractCustomFields scans the names map to find Sprint and Epic custom
// field IDs, then parses their values from the raw fields JSON.
func extractCustomFields(issue *Issue, rawFields json.RawMessage, names map[string]string) {
	if len(names) == 0 || len(rawFields) == 0 {
		return
	}

	// Find custom field IDs by their display name.
	var sprintFieldID, epicFieldID string
	for id, name := range names {
		switch strings.ToLower(name) {
		case "sprint":
			sprintFieldID = id
		case "epic link":
			epicFieldID = id
		}
	}

	// Parse the entire fields object into a generic map for custom field access.
	var allFields map[string]json.RawMessage
	if err := json.Unmarshal(rawFields, &allFields); err != nil {
		return
	}

	if sprintFieldID != "" {
		if raw, ok := allFields[sprintFieldID]; ok {
			issue.Sprint = parseSprint(raw)
		}
	}
	if epicFieldID != "" {
		if raw, ok := allFields[epicFieldID]; ok {
			issue.Epic = parseEpic(raw)
		}
	}
}

// parseSprint extracts sprint info from a custom field value.
// The field can be a single object or an array; we take the last (active) sprint.
func parseSprint(raw json.RawMessage) *Sprint {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	// Try as array first (common format).
	var sprints []Sprint
	if err := json.Unmarshal(raw, &sprints); err == nil && len(sprints) > 0 {
		return &sprints[len(sprints)-1]
	}

	// Try as single object.
	var sprint Sprint
	if err := json.Unmarshal(raw, &sprint); err == nil && sprint.Name != "" {
		return &sprint
	}
	return nil
}

// parseEpic extracts epic info from a custom field value.
// It can be a string (epic key) or an object with key+summary.
func parseEpic(raw json.RawMessage) *Epic {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	// Try as object first.
	var epic Epic
	if err := json.Unmarshal(raw, &epic); err == nil && epic.Key != "" {
		return &epic
	}

	// Try as string (epic key only).
	var key string
	if err := json.Unmarshal(raw, &key); err == nil && key != "" {
		return &Epic{Key: key}
	}
	return nil
}

func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	return resp, nil
}

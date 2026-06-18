package confluence

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// apiPrefix is the Confluence Cloud v2 REST API path, relative to the Atlassian
// instance base URL (e.g. https://acme.atlassian.net).
const apiPrefix = "/wiki/api/v2"

// Client is an authenticated Confluence Cloud v2 REST API client.
type Client struct {
	baseURL    string
	authHeader string
	http       *http.Client
}

// NewClient creates a Confluence client using Basic auth (email:apiToken), the
// same Atlassian API-token scheme as the Jira client. instance is the Atlassian
// site base URL (e.g. https://acme.atlassian.net); Confluence lives under /wiki.
func NewClient(instance, email, token string) *Client {
	baseURL := strings.TrimRight(instance, "/")
	creds := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
	return &Client{
		baseURL:    baseURL,
		authHeader: "Basic " + creds,
		http:       &http.Client{Timeout: 30 * time.Second},
	}
}

// GetPage fetches a page by id with its body in Atlassian Document Format.
func (c *Client) GetPage(id string) (*Page, error) {
	path := apiPrefix + "/pages/" + url.PathEscape(id) + "?body-format=atlas_doc_format"
	body, err := c.getJSON(path)
	if err != nil {
		return nil, err
	}
	var p Page
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("decoding page: %w", err)
	}
	return &p, nil
}

// getJSON performs a GET expecting JSON, returning the body after status checks.
// pathOrURL may be a path (prefixed with baseURL) or a full URL (pagination next).
func (c *Client) getJSON(pathOrURL string) ([]byte, error) {
	resp, err := c.do(http.MethodGet, pathOrURL)
	if err != nil {
		return nil, err
	}
	body, status, err := readAndClose(resp)
	if err != nil {
		return nil, err
	}
	if err := classify(status, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (c *Client) do(method, pathOrURL string) (*http.Response, error) {
	target := pathOrURL
	if !strings.HasPrefix(target, "http") {
		target = c.baseURL + pathOrURL
	}
	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	return resp, nil
}

func readAndClose(resp *http.Response) ([]byte, int, error) {
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
	}
	return data, resp.StatusCode, nil
}

func classify(status int, body []byte) error {
	switch status {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	}
	if status < 200 || status >= 300 {
		return &APIError{StatusCode: status, Message: string(body)}
	}
	return nil
}

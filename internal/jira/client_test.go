package jira

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMyselfSuccess(t *testing.T) {
	want := User{
		AccountID:   "123abc",
		DisplayName: "Test User",
		Email:       "test@example.com",
		Active:      true,
		TimeZone:    "America/New_York",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/myself" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test@example.com", "token123")
	got, err := client.Myself()
	if err != nil {
		t.Fatalf("Myself: %v", err)
	}

	if got.AccountID != want.AccountID {
		t.Errorf("AccountID = %q, want %q", got.AccountID, want.AccountID)
	}
	if got.DisplayName != want.DisplayName {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, want.DisplayName)
	}
	if got.Email != want.Email {
		t.Errorf("Email = %q, want %q", got.Email, want.Email)
	}
	if got.Active != want.Active {
		t.Errorf("Active = %v, want %v", got.Active, want.Active)
	}
}

func TestMyselfUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "bad@example.com", "wrongtoken")
	_, err := client.Myself()
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestMyselfServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test@example.com", "token123")
	_, err := client.Myself()
	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got: %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

func TestAuthHeader(t *testing.T) {
	email := "user@example.com"
	token := "api-token"
	wantCreds := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		want := "Basic " + wantCreds
		if got != want {
			t.Errorf("Authorization = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(User{})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, email, token)
	client.Myself()
}

func TestMyselfForbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test@example.com", "token123")
	_, err := client.Myself()
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for 403, got: %v", err)
	}
}

// issueJSON is a realistic Jira issue response fixture for testing.
const issueJSON = `{
  "key": "PROJ-123",
  "names": {
    "summary": "Summary",
    "status": "Status",
    "customfield_10020": "Sprint",
    "customfield_10014": "Epic Link"
  },
  "fields": {
    "summary": "Implement OAuth2 flow",
    "description": {
      "type": "doc",
      "version": 1,
      "content": [
        {
          "type": "paragraph",
          "content": [{"type": "text", "text": "Build the OAuth2 integration."}]
        }
      ]
    },
    "status": {"name": "In Progress"},
    "issuetype": {"name": "Story"},
    "priority": {"name": "High"},
    "assignee": {"accountId": "abc", "displayName": "Alice", "emailAddress": "alice@co.com"},
    "reporter": {"accountId": "def", "displayName": "Bob", "emailAddress": "bob@co.com"},
    "labels": ["backend", "security"],
    "created": "2026-02-01T10:00:00.000+0000",
    "updated": "2026-02-10T14:30:00.000+0000",
    "comment": {
      "total": 1,
      "comments": [
        {
          "author": {"accountId": "abc", "displayName": "Alice", "emailAddress": "alice@co.com"},
          "body": {
            "type": "doc",
            "version": 1,
            "content": [{"type": "paragraph", "content": [{"type": "text", "text": "LGTM"}]}]
          },
          "created": "2026-02-05T09:00:00.000+0000"
        }
      ]
    },
    "subtasks": [
      {"key": "PROJ-124", "fields": {"summary": "Research libraries", "status": {"name": "Done"}}},
      {"key": "PROJ-125", "fields": {"summary": "Implement storage", "status": {"name": "In Progress"}}}
    ],
    "issuelinks": [
      {
        "type": {"name": "Blocks", "outward": "blocks", "inward": "is blocked by"},
        "outwardIssue": {"key": "PROJ-130", "fields": {"summary": "Protected endpoints", "status": {"name": "To Do"}}}
      }
    ],
    "parent": {"key": "PROJ-80", "fields": {"summary": "Authentication Epic", "status": {"name": "In Progress"}}},
    "customfield_10020": [{"name": "Sprint 14"}],
    "customfield_10014": "PROJ-80"
  }
}`

func TestGetIssueSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("expand") != "names" {
			t.Errorf("expected expand=names query param, got: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(issueJSON))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test@example.com", "token123")
	issue, err := client.GetIssue("PROJ-123")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}

	if issue.Key != "PROJ-123" {
		t.Errorf("Key = %q, want PROJ-123", issue.Key)
	}
	if issue.Fields.Summary != "Implement OAuth2 flow" {
		t.Errorf("Summary = %q", issue.Fields.Summary)
	}
	if issue.Fields.Status == nil || issue.Fields.Status.Name != "In Progress" {
		t.Errorf("Status = %v", issue.Fields.Status)
	}
	if issue.Fields.IssueType == nil || issue.Fields.IssueType.Name != "Story" {
		t.Errorf("IssueType = %v", issue.Fields.IssueType)
	}
	if issue.Fields.Priority == nil || issue.Fields.Priority.Name != "High" {
		t.Errorf("Priority = %v", issue.Fields.Priority)
	}
	if issue.Fields.Assignee == nil || issue.Fields.Assignee.DisplayName != "Alice" {
		t.Errorf("Assignee = %v", issue.Fields.Assignee)
	}
	if issue.Fields.Reporter == nil || issue.Fields.Reporter.DisplayName != "Bob" {
		t.Errorf("Reporter = %v", issue.Fields.Reporter)
	}
	if len(issue.Fields.Labels) != 2 {
		t.Errorf("Labels = %v", issue.Fields.Labels)
	}
	if issue.Fields.Description == nil {
		t.Error("Description is nil")
	}
	if issue.Fields.Comment == nil || issue.Fields.Comment.Total != 1 {
		t.Errorf("Comment = %v", issue.Fields.Comment)
	}
	if len(issue.Fields.Subtasks) != 2 {
		t.Errorf("Subtasks count = %d, want 2", len(issue.Fields.Subtasks))
	}
	if len(issue.Fields.IssueLinks) != 1 {
		t.Errorf("IssueLinks count = %d, want 1", len(issue.Fields.IssueLinks))
	}
	if issue.Fields.Parent == nil || issue.Fields.Parent.Key != "PROJ-80" {
		t.Errorf("Parent = %v", issue.Fields.Parent)
	}
}

func TestGetIssueCustomFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(issueJSON))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test@example.com", "token123")
	issue, err := client.GetIssue("PROJ-123")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}

	if issue.Sprint == nil {
		t.Fatal("Sprint is nil, expected Sprint 14")
	}
	if issue.Sprint.Name != "Sprint 14" {
		t.Errorf("Sprint.Name = %q, want 'Sprint 14'", issue.Sprint.Name)
	}

	if issue.Epic == nil {
		t.Fatal("Epic is nil, expected PROJ-80")
	}
	if issue.Epic.Key != "PROJ-80" {
		t.Errorf("Epic.Key = %q, want 'PROJ-80'", issue.Epic.Key)
	}
}

func TestGetIssueNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errorMessages":["Issue does not exist"]}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test@example.com", "token123")
	_, err := client.GetIssue("NOPE-999")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetIssueUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "bad@example.com", "wrong")
	_, err := client.GetIssue("PROJ-1")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestGetIssueForbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test@example.com", "token123")
	_, err := client.GetIssue("PROJ-1")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for 403, got: %v", err)
	}
}

func TestGetIssueServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test@example.com", "token123")
	_, err := client.GetIssue("PROJ-1")
	if err == nil {
		t.Fatal("expected error for 500")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got: %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

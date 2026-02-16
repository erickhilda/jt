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

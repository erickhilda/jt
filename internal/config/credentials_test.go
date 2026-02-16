package config

import (
	"os"
	"path/filepath"
	"testing"
)

// These tests exercise the file-based fallback path only.
// We don't test the keyring path in CI because it requires a running D-Bus session.

func TestSetTokenFile(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(ResetConfigDir)

	token := "my-secret-token"
	if err := setTokenFile(token); err != nil {
		t.Fatalf("setTokenFile: %v", err)
	}

	path := filepath.Join(dir, credFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat credentials: %v", err)
	}
	// Check file permissions (owner read/write only).
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("credentials file perms = %o, want 0600", perm)
	}
}

func TestGetTokenFile(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(ResetConfigDir)

	token := "roundtrip-token"
	if err := setTokenFile(token); err != nil {
		t.Fatalf("setTokenFile: %v", err)
	}

	got, err := getTokenFile()
	if err != nil {
		t.Fatalf("getTokenFile: %v", err)
	}
	if got != token {
		t.Errorf("getTokenFile = %q, want %q", got, token)
	}
}

func TestDeleteTokenFile(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(ResetConfigDir)

	if err := setTokenFile("to-delete"); err != nil {
		t.Fatalf("setTokenFile: %v", err)
	}

	if err := deleteTokenFile(); err != nil {
		t.Fatalf("deleteTokenFile: %v", err)
	}

	path := filepath.Join(dir, credFileName)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected credentials file to be deleted")
	}
}

func TestDeleteTokenFileNotExist(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(ResetConfigDir)

	// Should not error when file doesn't exist.
	if err := deleteTokenFile(); err != nil {
		t.Fatalf("deleteTokenFile on missing file: %v", err)
	}
}

func TestGetTokenViaConfig(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(ResetConfigDir)

	token := "config-token"
	if err := setTokenFile(token); err != nil {
		t.Fatalf("setTokenFile: %v", err)
	}

	cfg := &Config{
		Instance:     "https://test.atlassian.net",
		Email:        "a@b.com",
		TokenStorage: TokenStorageFile,
	}

	got, err := GetToken(cfg)
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if got != token {
		t.Errorf("GetToken = %q, want %q", got, token)
	}
}

func TestDeleteTokenViaConfig(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(ResetConfigDir)

	if err := setTokenFile("delete-me"); err != nil {
		t.Fatalf("setTokenFile: %v", err)
	}

	cfg := &Config{
		Instance:     "https://test.atlassian.net",
		Email:        "a@b.com",
		TokenStorage: TokenStorageFile,
	}

	if err := DeleteToken(cfg); err != nil {
		t.Fatalf("DeleteToken: %v", err)
	}

	path := filepath.Join(dir, credFileName)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected credentials file to be deleted")
	}
}

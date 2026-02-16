package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(ResetConfigDir)

	original := &Config{
		Instance:       "https://myorg.atlassian.net",
		Email:          "user@example.com",
		DefaultProject: "PROJ",
		TicketsDir:     "~/tickets",
		TokenStorage:   TokenStorageFile,
	}

	if err := Save(original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Instance != original.Instance {
		t.Errorf("Instance: got %q, want %q", loaded.Instance, original.Instance)
	}
	if loaded.Email != original.Email {
		t.Errorf("Email: got %q, want %q", loaded.Email, original.Email)
	}
	if loaded.DefaultProject != original.DefaultProject {
		t.Errorf("DefaultProject: got %q, want %q", loaded.DefaultProject, original.DefaultProject)
	}
	if loaded.TicketsDir != original.TicketsDir {
		t.Errorf("TicketsDir: got %q, want %q", loaded.TicketsDir, original.TicketsDir)
	}
	if loaded.TokenStorage != original.TokenStorage {
		t.Errorf("TokenStorage: got %q, want %q", loaded.TokenStorage, original.TokenStorage)
	}
}

func TestLoadNotFound(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(ResetConfigDir)

	_, err := Load()
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(ResetConfigDir)

	exists, err := Exists()
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Error("expected Exists()=false before Save")
	}

	cfg := &Config{
		Instance:     "https://test.atlassian.net",
		Email:        "a@b.com",
		TokenStorage: TokenStorageFile,
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	exists, err = Exists()
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Error("expected Exists()=true after Save")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Instance:     "https://myorg.atlassian.net",
				Email:        "user@example.com",
				TokenStorage: TokenStorageKeyring,
			},
			wantErr: false,
		},
		{
			name: "missing instance",
			cfg: Config{
				Email:        "user@example.com",
				TokenStorage: TokenStorageFile,
			},
			wantErr: true,
		},
		{
			name: "instance without https",
			cfg: Config{
				Instance:     "http://myorg.atlassian.net",
				Email:        "user@example.com",
				TokenStorage: TokenStorageFile,
			},
			wantErr: true,
		},
		{
			name: "missing email",
			cfg: Config{
				Instance:     "https://myorg.atlassian.net",
				TokenStorage: TokenStorageFile,
			},
			wantErr: true,
		},
		{
			name: "invalid token_storage",
			cfg: Config{
				Instance:     "https://myorg.atlassian.net",
				Email:        "user@example.com",
				TokenStorage: "magic",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/tickets", filepath.Join(home, "tickets")},
		{"~/.jt", filepath.Join(home, ".jt")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		got, err := ExpandPath(tt.input)
		if err != nil {
			t.Errorf("ExpandPath(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	SetConfigDir(dir)
	t.Cleanup(ResetConfigDir)

	cfg := &Config{
		Instance:     "https://test.atlassian.net",
		Email:        "a@b.com",
		TokenStorage: TokenStorageFile,
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat config dir: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected config dir to be a directory")
	}
}

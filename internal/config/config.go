package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TokenStorage indicates where the API token is stored.
type TokenStorage string

const (
	TokenStorageKeyring TokenStorage = "keyring"
	TokenStorageFile    TokenStorage = "file"
)

const (
	// configDirName is the current config/state directory under $HOME.
	configDirName = ".atlit"
	// legacyConfigDirName is the pre-rename directory, still read (and migrated
	// from) for backward compatibility. See `atlit migrate`.
	legacyConfigDirName = ".jt"
)

// ErrNotFound is returned when no config file exists.
var ErrNotFound = errors.New("config file not found; run 'atlit init' to set up")

// configDirOverride allows tests to redirect config storage.
var configDirOverride string

// Config holds the atlit configuration.
type Config struct {
	Instance       string       `yaml:"instance"`
	Email          string       `yaml:"email"`
	DefaultProject string       `yaml:"default_project,omitempty"`
	TicketsDir     string       `yaml:"tickets_dir"`
	TokenStorage   TokenStorage `yaml:"token_storage"`
	// FetchComments controls whether pull/diff/sync request and render comments.
	// Pointer so "field absent" means "default true" (backward compatible).
	FetchComments *bool `yaml:"fetch_comments,omitempty"`
	// FetchPullRequests controls whether `atlit pull` also fetches the development
	// panel's linked pull requests (via the dev-status API) and renders a
	// "## Pull Requests" section. Pointer so absent means "default true".
	FetchPullRequests *bool `yaml:"fetch_pull_requests,omitempty"`
	// BitbucketWorkspace is the default workspace for `atlit pr <repo>/<id>` refs.
	BitbucketWorkspace string `yaml:"bitbucket_workspace,omitempty"`
	// PRsDir is where `atlit pr` saves pull-request markdown (default <config-dir>/prs).
	PRsDir string `yaml:"prs_dir,omitempty"`
	// PagesDir is where `atlit page` saves Confluence page markdown (default <config-dir>/pages).
	PagesDir string `yaml:"pages_dir,omitempty"`
}

// PagesDirOrDefault returns the configured Confluence page storage directory,
// defaulting to <config-dir>/pages when unset.
func (c *Config) PagesDirOrDefault() string {
	if c == nil || c.PagesDir == "" {
		return defaultStateSubdir("pages")
	}
	return c.PagesDir
}

// PRsDirOrDefault returns the configured PR storage directory, defaulting to
// <config-dir>/prs when unset.
func (c *Config) PRsDirOrDefault() string {
	if c == nil || c.PRsDir == "" {
		return defaultStateSubdir("prs")
	}
	return c.PRsDir
}

// defaultStateSubdir returns <config-dir>/<name>, resolving the same
// new/legacy fallback as ConfigDir so PR/page storage tracks the config dir.
// Best-effort: if the home directory cannot be determined it falls back to a
// new-style tilde path.
func defaultStateSubdir(name string) string {
	dir, err := ConfigDir()
	if err != nil {
		return filepath.Join("~", configDirName, name)
	}
	return filepath.Join(dir, name)
}

// ShouldFetchComments returns whether comments should be fetched and rendered.
// Default (nil) is true to preserve prior behavior.
func (c *Config) ShouldFetchComments() bool {
	if c == nil || c.FetchComments == nil {
		return true
	}
	return *c.FetchComments
}

// ShouldFetchPullRequests returns whether `atlit pull` should fetch and render the
// development panel's linked pull requests. Default (nil) is true.
func (c *Config) ShouldFetchPullRequests() bool {
	if c == nil || c.FetchPullRequests == nil {
		return true
	}
	return *c.FetchPullRequests
}

// SetConfigDir overrides the config directory (for testing).
func SetConfigDir(dir string) {
	configDirOverride = dir
}

// ResetConfigDir clears the override.
func ResetConfigDir() {
	configDirOverride = ""
}

// ConfigDir returns the directory where atlit stores its config. It prefers
// the current ~/.atlit directory, falling back to the legacy ~/.jt only when
// ~/.atlit does not yet exist but ~/.jt does (pre-migration). Fresh installs
// therefore use ~/.atlit. Run `atlit migrate` to convert a legacy directory.
func ConfigDir() (string, error) {
	if configDirOverride != "" {
		return configDirOverride, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	current := filepath.Join(home, configDirName)
	legacy := filepath.Join(home, legacyConfigDirName)
	if !dirExists(current) && dirExists(legacy) {
		return legacy, nil
	}
	return current, nil
}

// dirExists reports whether path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ConfigDirCandidates returns the absolute current (~/.atlit) and legacy (~/.jt)
// config directory paths. Used by `atlit migrate` to relocate legacy state.
func ConfigDirCandidates() (current, legacy string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, configDirName), filepath.Join(home, legacyConfigDirName), nil
}

// HasLegacyState reports whether a legacy ~/.jt directory exists while the
// current ~/.atlit does not — i.e. an `atlit migrate` is pending.
func HasLegacyState() bool {
	current, legacy, err := ConfigDirCandidates()
	if err != nil {
		return false
	}
	return dirExists(legacy) && !dirExists(current)
}

// ConfigPath returns the full path to config.yaml.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Exists returns true if the config file is present on disk.
func Exists() (bool, error) {
	path, err := ConfigPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Load reads and parses the config file.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to disk, creating the directory if needed.
func Save(cfg *Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	path := filepath.Join(dir, "config.yaml")
	return os.WriteFile(path, data, 0600)
}

// Validate checks that required fields are present and well-formed.
func (c *Config) Validate() error {
	var errs []string
	if c.Instance == "" {
		errs = append(errs, "instance is required")
	} else if !strings.HasPrefix(c.Instance, "https://") {
		errs = append(errs, "instance must start with https://")
	}
	if c.Email == "" {
		errs = append(errs, "email is required")
	}
	if c.TokenStorage != TokenStorageKeyring && c.TokenStorage != TokenStorageFile {
		errs = append(errs, "token_storage must be 'keyring' or 'file'")
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid config: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ExpandPath expands a leading ~ to the user's home directory.
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, path[1:]), nil
}

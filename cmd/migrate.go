package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/erickhilda/atlit/internal/config"
	"github.com/erickhilda/atlit/internal/store"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate legacy 'jt' state (config dir, keyring, file markers) to atlit",
	Long: `One-time upgrade of on-disk state left by the previous 'jt' name:

  - rewrites the '<!-- jt:meta ... -->' header in pulled files to 'atlit:meta'
  - moves ~/.jt to ~/.atlit (config + credentials)
  - copies keyring tokens from the jt-cli service to atlit-cli

Everything keeps working without this (atlit reads the old names too); migrate
just makes the on-disk state match the new name. Safe to re-run — already
migrated state is skipped. Use --dry-run to preview.`,
	Args: cobra.NoArgs,
	RunE: runMigrate,
}

func init() {
	migrateCmd.Flags().Bool("dry-run", false, "Show what would change without modifying anything")
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, _ []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	cfg, err := config.Load()
	if errors.Is(err, config.ErrNotFound) {
		fmt.Println("No atlit config found; nothing to migrate.")
		return nil
	}
	if err != nil {
		return err
	}

	tag := ""
	if dryRun {
		tag = "[dry-run] "
	}

	// 1. Rewrite jt:meta -> atlit:meta markers in pulled files. Done before the
	//    directory move so files under ~/.jt are upgraded in place, then carried
	//    along by the move. Resolving the dirs now (pre-move) keeps dry-run and
	//    real runs pointed at the same physical locations.
	markerDirs := uniqueDirs(cfg.TicketsDir, cfg.PRsDirOrDefault(), cfg.PagesDirOrDefault())
	count, err := migrateMarkers(markerDirs, dryRun)
	if err != nil {
		return err
	}
	fmt.Printf("%sfiles with markers upgraded: %d\n", tag, count)

	// 2. Move the config/state directory ~/.jt -> ~/.atlit.
	current, legacy, err := config.ConfigDirCandidates()
	if err != nil {
		return err
	}
	switch {
	case dirExists(legacy) && !dirExists(current):
		fmt.Printf("%smove %s -> %s\n", tag, legacy, current)
		if !dryRun {
			if err := os.Rename(legacy, current); err != nil {
				return fmt.Errorf("moving config directory: %w", err)
			}
		}
	case dirExists(current):
		fmt.Printf("config directory already at %s\n", current)
	default:
		fmt.Println("no legacy config directory to move")
	}

	// 3. Migrate keyring tokens (keyring storage only).
	if cfg.TokenStorage == config.TokenStorageKeyring {
		migrated, err := config.MigrateKeyringTokens(cfg.Email, dryRun)
		if err != nil {
			return err
		}
		if len(migrated) > 0 {
			fmt.Printf("%skeyring tokens migrated: %s\n", tag, strings.Join(migrated, ", "))
		} else {
			fmt.Println("no legacy keyring tokens to migrate")
		}
	}

	// 4. Rewrite any literal legacy paths inside config.yaml (e.g. a tickets_dir
	//    of ~/.jt/tickets), then persist to the (now current) config dir.
	if rewriteConfigPaths(cfg) {
		fmt.Printf("%supdate legacy ~/.jt paths in config.yaml\n", tag)
		if !dryRun {
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
		}
	}

	if dryRun {
		fmt.Println("\nDry run complete; no changes were made.")
	} else {
		fmt.Println("\nMigration complete.")
	}
	return nil
}

// dirExists reports whether path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// uniqueDirs expands and de-duplicates a set of directory paths, dropping
// empties and unexpandable entries.
func uniqueDirs(paths ...string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, p := range paths {
		expanded, err := config.ExpandPath(p)
		if err != nil || expanded == "" || seen[expanded] {
			continue
		}
		seen[expanded] = true
		out = append(out, expanded)
	}
	return out
}

// migrateMarkers walks each dir for *.md files whose first line is a legacy
// jt:meta marker and rewrites it to atlit:meta. Non-existent dirs are skipped.
// Returns the number of files upgraded (or that would be, under dryRun).
func migrateMarkers(dirs []string, dryRun bool) (int, error) {
	count := 0
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return count, fmt.Errorf("reading %s: %w", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				return count, fmt.Errorf("reading %s: %w", path, err)
			}
			upgraded, changed := store.UpgradeMarker(string(data))
			if !changed {
				continue
			}
			count++
			if dryRun {
				continue
			}
			if err := os.WriteFile(path, []byte(upgraded), 0644); err != nil {
				return count, fmt.Errorf("writing %s: %w", path, err)
			}
		}
	}
	return count, nil
}

// rewriteConfigPaths replaces a leading legacy "~/.jt" (or absolute ".jt")
// segment with ".atlit" in the directory fields, reporting whether anything
// changed. Fields pointing elsewhere are left untouched.
func rewriteConfigPaths(cfg *config.Config) bool {
	changed := false
	for _, f := range []*string{&cfg.TicketsDir, &cfg.PRsDir, &cfg.PagesDir} {
		if *f == "" {
			continue
		}
		if n := strings.Replace(*f, "/.jt", "/.atlit", 1); n != *f {
			*f = n
			changed = true
		}
	}
	return changed
}

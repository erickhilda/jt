package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erickhilda/atlit/internal/config"
)

func TestMigrateMarkers(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "PROJ-1.md")
	current := filepath.Join(dir, "PROJ-2.md")
	plain := filepath.Join(dir, "notes.md")

	legacyContent := "<!-- jt:meta ticket=PROJ-1 fetched=2026-02-17T10:30:00Z -->\n# PROJ-1\n"
	currentContent := "<!-- atlit:meta ticket=PROJ-2 fetched=2026-02-17T10:30:00Z -->\n# PROJ-2\n"
	plainContent := "# notes\n\nno marker here\n"

	for path, content := range map[string]string{
		legacy: legacyContent, current: currentContent, plain: plainContent,
	} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}

	// Dry run: reports 1 but mutates nothing.
	count, err := migrateMarkers([]string{dir}, true)
	if err != nil {
		t.Fatalf("migrateMarkers dry-run: %v", err)
	}
	if count != 1 {
		t.Errorf("dry-run count = %d, want 1", count)
	}
	if got, _ := os.ReadFile(legacy); string(got) != legacyContent {
		t.Errorf("dry-run mutated the legacy file: %q", got)
	}

	// Real run: upgrades exactly the legacy file.
	count, err = migrateMarkers([]string{dir}, false)
	if err != nil {
		t.Fatalf("migrateMarkers: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if got, _ := os.ReadFile(legacy); string(got) != currentContentFor("PROJ-1") {
		t.Errorf("legacy file = %q, want upgraded marker", got)
	}
	if got, _ := os.ReadFile(current); string(got) != currentContent {
		t.Errorf("already-current file mutated: %q", got)
	}
	if got, _ := os.ReadFile(plain); string(got) != plainContent {
		t.Errorf("plain file mutated: %q", got)
	}

	// Idempotent: a second run changes nothing.
	count, err = migrateMarkers([]string{dir}, false)
	if err != nil {
		t.Fatalf("migrateMarkers re-run: %v", err)
	}
	if count != 0 {
		t.Errorf("re-run count = %d, want 0", count)
	}
}

func currentContentFor(key string) string {
	return "<!-- atlit:meta ticket=" + key + " fetched=2026-02-17T10:30:00Z -->\n# " + key + "\n"
}

func TestMigrateMarkersSkipsMissingDir(t *testing.T) {
	count, err := migrateMarkers([]string{filepath.Join(t.TempDir(), "does-not-exist")}, false)
	if err != nil {
		t.Fatalf("migrateMarkers: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestRewriteConfigPaths(t *testing.T) {
	t.Run("rewrites legacy .jt segment", func(t *testing.T) {
		cfg := &config.Config{
			TicketsDir: "~/.jt/tickets",
			PagesDir:   "~/custom/pages",
		}
		if !rewriteConfigPaths(cfg) {
			t.Fatal("expected changed=true")
		}
		if cfg.TicketsDir != "~/.atlit/tickets" {
			t.Errorf("TicketsDir = %q, want ~/.atlit/tickets", cfg.TicketsDir)
		}
		if cfg.PagesDir != "~/custom/pages" {
			t.Errorf("PagesDir should be untouched, got %q", cfg.PagesDir)
		}
	})

	t.Run("no legacy paths means no change", func(t *testing.T) {
		cfg := &config.Config{TicketsDir: "~/Dev/docs/tasks"}
		if rewriteConfigPaths(cfg) {
			t.Error("expected changed=false")
		}
	})
}

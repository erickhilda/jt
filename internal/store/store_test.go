package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	content := "# TEST-1: Sample ticket\n\nSome content.\n"

	if err := Save(dir, "TEST-1", content); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(dir, "TEST-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != content {
		t.Errorf("Load returned %q, want %q", got, content)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "sub", "tickets")

	if err := Save(dir, "TEST-2", "content"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "TEST-2.md")); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()

	exists, err := Exists(dir, "NOPE-1")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Error("expected Exists=false for missing file")
	}

	if err := Save(dir, "NOPE-1", "test"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	exists, err = Exists(dir, "NOPE-1")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Error("expected Exists=true after Save")
	}
}

func TestTicketPath(t *testing.T) {
	path, err := TicketPath("/tmp/tickets", "PROJ-99")
	if err != nil {
		t.Fatalf("TicketPath: %v", err)
	}
	want := "/tmp/tickets/PROJ-99.md"
	if path != want {
		t.Errorf("TicketPath = %q, want %q", path, want)
	}
}

func TestExtractNotes(t *testing.T) {
	content := `# PROJ-1: Title

## Description

Some description.

## My Notes

These are my local notes.
They span multiple lines.
`
	got := ExtractNotes(content)
	if !strings.HasPrefix(got, "## My Notes") {
		t.Errorf("expected notes to start with '## My Notes', got %q", got)
	}
	if !strings.Contains(got, "These are my local notes.") {
		t.Errorf("expected notes content, got %q", got)
	}
	if !strings.Contains(got, "They span multiple lines.") {
		t.Errorf("expected multi-line notes, got %q", got)
	}
}

func TestExtractNotesEmpty(t *testing.T) {
	content := "# PROJ-1: Title\n\n## Description\n\nNo notes here.\n"
	got := ExtractNotes(content)
	if got != "" {
		t.Errorf("expected empty notes, got %q", got)
	}
}

func TestReplaceSectionExisting(t *testing.T) {
	content := "# PROJ-1: Title\n\n## Description\n\nOld desc.\n\n## Comments (1)\n\n### Alice -- 2026-01-01\n\nOld comment.\n\n## My Notes\n\nMy notes here.\n"
	newComments := "## Comments (2)\n\n### Alice -- 2026-01-01\n\nOld comment.\n\n### Bob -- 2026-01-02\n\nNew comment.\n"

	got := ReplaceSection(content, "## Comments", newComments)

	if !strings.Contains(got, "## Comments (2)") {
		t.Errorf("expected updated comments header, got:\n%s", got)
	}
	if !strings.Contains(got, "New comment.") {
		t.Errorf("expected new comment, got:\n%s", got)
	}
	if !strings.Contains(got, "## My Notes") {
		t.Errorf("expected My Notes preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "My notes here.") {
		t.Errorf("expected notes content preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "## Description") {
		t.Errorf("expected description preserved, got:\n%s", got)
	}
}

func TestReplaceSectionInsertBeforeNotes(t *testing.T) {
	content := "# PROJ-1: Title\n\n## Description\n\nDesc.\n\n## My Notes\n\nNotes.\n"
	newComments := "## Comments (1)\n\n### Alice -- 2026-01-01\n\nHello.\n"

	got := ReplaceSection(content, "## Comments", newComments)

	if !strings.Contains(got, "## Comments (1)") {
		t.Errorf("expected comments inserted, got:\n%s", got)
	}
	if !strings.Contains(got, "## My Notes") {
		t.Errorf("expected My Notes preserved, got:\n%s", got)
	}
	// Comments should come before My Notes.
	commentsIdx := strings.Index(got, "## Comments")
	notesIdx := strings.Index(got, "## My Notes")
	if commentsIdx >= notesIdx {
		t.Errorf("expected comments before My Notes, commentsIdx=%d notesIdx=%d", commentsIdx, notesIdx)
	}
}

func TestReplaceSectionAppend(t *testing.T) {
	content := "# PROJ-1: Title\n\n## Description\n\nDesc.\n"
	newComments := "## Comments (1)\n\n### Alice -- 2026-01-01\n\nHello.\n"

	got := ReplaceSection(content, "## Comments", newComments)

	if !strings.Contains(got, "## Comments (1)") {
		t.Errorf("expected comments appended, got:\n%s", got)
	}
	if !strings.Contains(got, "## Description") {
		t.Errorf("expected description preserved, got:\n%s", got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir, "MISSING-1")
	if err == nil {
		t.Error("expected error loading missing file")
	}
}

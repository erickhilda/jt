package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestExtractSection(t *testing.T) {
	content := "# PROJ-1: Title\n\n## Description\n\nDesc.\n\n## Comments (1)\n\n### Alice -- 2026-01-01\n\nHello.\n\n## My Notes\n\nMine.\n"

	got := ExtractSection(content, "## Comments")
	if !strings.HasPrefix(got, "## Comments (1)") {
		t.Errorf("expected Comments block to start with heading, got: %q", got)
	}
	if !strings.Contains(got, "Hello.") {
		t.Errorf("expected comment body in block, got: %q", got)
	}
	if strings.Contains(got, "## My Notes") {
		t.Errorf("block should stop before next h2, got: %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("block should end with a single newline, got: %q", got)
	}
}

func TestExtractSectionAtEOF(t *testing.T) {
	content := "# PROJ-1\n\n## Comments (1)\n\nTail comment.\n"
	got := ExtractSection(content, "## Comments")
	if !strings.Contains(got, "Tail comment.") {
		t.Errorf("expected tail comment, got: %q", got)
	}
}

func TestExtractSectionMissing(t *testing.T) {
	content := "# PROJ-1\n\n## Description\n\nDesc.\n"
	if got := ExtractSection(content, "## Comments"); got != "" {
		t.Errorf("expected empty string when section absent, got: %q", got)
	}
}

func TestRemoveSectionMiddle(t *testing.T) {
	content := "# PROJ-1\n\n## Description\n\nDesc.\n\n## Comments (1)\n\nHello.\n\n## My Notes\n\nMine.\n"
	got := RemoveSection(content, "## Comments")
	if strings.Contains(got, "## Comments") {
		t.Errorf("expected Comments removed, got: %q", got)
	}
	if !strings.Contains(got, "## Description") || !strings.Contains(got, "## My Notes") {
		t.Errorf("expected surrounding sections preserved, got: %q", got)
	}
	if !strings.Contains(got, "Mine.") {
		t.Errorf("expected My Notes body preserved, got: %q", got)
	}
}

func TestRemoveSectionAtEOF(t *testing.T) {
	content := "# PROJ-1\n\n## Description\n\nDesc.\n\n## Comments (1)\n\nHello.\n"
	got := RemoveSection(content, "## Comments")
	if strings.Contains(got, "## Comments") {
		t.Errorf("expected Comments removed, got: %q", got)
	}
	if !strings.Contains(got, "## Description") {
		t.Errorf("expected Description preserved, got: %q", got)
	}
}

func TestRemoveSectionMissing(t *testing.T) {
	content := "# PROJ-1\n\n## Description\n\nDesc.\n"
	got := RemoveSection(content, "## Comments")
	if got != content {
		t.Errorf("expected content unchanged, got: %q", got)
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

func TestParseMetaValid(t *testing.T) {
	content := "<!-- jt:meta ticket=PROJ-123 fetched=2026-02-17T10:30:00Z -->\n# PROJ-123: Title\n"
	meta := ParseMeta(content)
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
	if meta.Ticket != "PROJ-123" {
		t.Errorf("Ticket = %q, want PROJ-123", meta.Ticket)
	}
	want := time.Date(2026, 2, 17, 10, 30, 0, 0, time.UTC)
	if !meta.Fetched.Equal(want) {
		t.Errorf("Fetched = %v, want %v", meta.Fetched, want)
	}
}

func TestParseMetaMissing(t *testing.T) {
	content := "# PROJ-123: Title\n\nNo meta comment here.\n"
	meta := ParseMeta(content)
	if meta != nil {
		t.Errorf("expected nil meta, got %+v", meta)
	}
}

func TestParseMetaMalformedTimestamp(t *testing.T) {
	content := "<!-- jt:meta ticket=PROJ-1 fetched=not-a-date -->\n"
	meta := ParseMeta(content)
	if meta != nil {
		t.Errorf("expected nil meta for malformed timestamp, got %+v", meta)
	}
}

func TestParseMetaIncompleteFields(t *testing.T) {
	content := "<!-- jt:meta ticket=PROJ-1 -->\n"
	meta := ParseMeta(content)
	if meta != nil {
		t.Errorf("expected nil meta when fetched is missing, got %+v", meta)
	}
}

func TestListTicketsPopulated(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"PROJ-200.md": "<!-- jt:meta ticket=PROJ-200 fetched=2026-02-17T08:00:00Z -->\n# PROJ-200\n",
		"PROJ-100.md": "<!-- jt:meta ticket=PROJ-100 fetched=2026-02-16T10:00:00Z -->\n# PROJ-100\n",
		"README.md":   "# Not a ticket\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	tickets, err := ListTickets(dir)
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if len(tickets) != 2 {
		t.Fatalf("got %d tickets, want 2", len(tickets))
	}
	// Should be sorted by key.
	if tickets[0].Key != "PROJ-100" {
		t.Errorf("tickets[0].Key = %q, want PROJ-100", tickets[0].Key)
	}
	if tickets[1].Key != "PROJ-200" {
		t.Errorf("tickets[1].Key = %q, want PROJ-200", tickets[1].Key)
	}
}

func TestListTicketsEmptyDir(t *testing.T) {
	dir := t.TempDir()
	tickets, err := ListTickets(dir)
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if len(tickets) != 0 {
		t.Errorf("expected 0 tickets, got %d", len(tickets))
	}
}

func TestListTicketsNonexistentDir(t *testing.T) {
	tickets, err := ListTickets("/tmp/nonexistent-jt-test-dir-12345")
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if tickets != nil {
		t.Errorf("expected nil tickets for nonexistent dir, got %v", tickets)
	}
}

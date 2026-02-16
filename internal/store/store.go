package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Save writes ticket content to <ticketsDir>/<key>.md, creating the
// directory if it doesn't exist.
func Save(ticketsDir, key, content string) error {
	dir, err := expandTilde(ticketsDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating tickets directory: %w", err)
	}
	path := filepath.Join(dir, key+".md")
	return os.WriteFile(path, []byte(content), 0644)
}

// Load reads the content of a saved ticket file.
func Load(ticketsDir, key string) (string, error) {
	path, err := TicketPath(ticketsDir, key)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Exists checks whether a ticket file exists on disk.
func Exists(ticketsDir, key string) (bool, error) {
	path, err := TicketPath(ticketsDir, key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// TicketPath returns the full filesystem path for a ticket file.
func TicketPath(ticketsDir, key string) (string, error) {
	dir, err := expandTilde(ticketsDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, key+".md"), nil
}

// ExtractNotes extracts the "## My Notes" section from existing file content.
// This section is preserved across re-pulls so users don't lose local annotations.
func ExtractNotes(content string) string {
	const marker = "## My Notes"
	idx := strings.Index(content, marker)
	if idx < 0 {
		return ""
	}
	return strings.TrimRight(content[idx:], "\n") + "\n"
}

// ReplaceSection replaces a ## section in content with new content.
// If the section doesn't exist, it inserts newSection before "## My Notes"
// (if present) or appends it at the end.
func ReplaceSection(content, sectionPrefix, newSection string) string {
	start := strings.Index(content, sectionPrefix)
	if start < 0 {
		// Section doesn't exist yet; insert before My Notes or append.
		notesIdx := strings.Index(content, "## My Notes")
		if notesIdx >= 0 {
			return strings.TrimRight(content[:notesIdx], "\n") + "\n\n" + newSection + "\n" + content[notesIdx:]
		}
		return strings.TrimRight(content, "\n") + "\n\n" + newSection
	}

	// Find end of this section (next ## heading at same level or EOF).
	rest := content[start+len(sectionPrefix):]
	end := findNextH2(rest)
	if end < 0 {
		// Check for My Notes after the section.
		notesIdx := strings.Index(rest, "## My Notes")
		if notesIdx >= 0 {
			return strings.TrimRight(content[:start], "\n") + "\n\n" + strings.TrimRight(newSection, "\n") + "\n\n" + rest[notesIdx:]
		}
		return strings.TrimRight(content[:start], "\n") + "\n\n" + newSection
	}
	after := rest[end:]
	return strings.TrimRight(content[:start], "\n") + "\n\n" + strings.TrimRight(newSection, "\n") + "\n\n" + after
}

// findNextH2 finds the index of the next "## " heading in text.
// Returns -1 if not found.
func findNextH2(text string) int {
	// Look for "\n## " which indicates a new h2 section.
	idx := strings.Index(text, "\n## ")
	if idx >= 0 {
		return idx + 1 // point at the "##"
	}
	return -1
}

func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expanding home directory: %w", err)
	}
	return filepath.Join(home, path[1:]), nil
}

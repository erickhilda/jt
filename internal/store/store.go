package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
	return ExtractSection(content, "## My Notes")
}

// ExtractSection returns the full "## <heading>" block from content, from the
// heading line through the line before the next "## " heading (or EOF).
// Returns "" if the heading is not present. The result always ends in a single
// trailing newline so callers can concatenate it directly.
func ExtractSection(content, heading string) string {
	idx := strings.Index(content, heading)
	if idx < 0 {
		return ""
	}
	rest := content[idx+len(heading):]
	end := findNextH2(rest)
	var block string
	if end < 0 {
		block = content[idx:]
	} else {
		block = content[idx : idx+len(heading)+end]
	}
	return strings.TrimRight(block, "\n") + "\n"
}

// RemoveSection strips a "## <heading>" block from content and returns the
// remainder. The block runs from the heading line through the line before the
// next "## " heading (or EOF). If the heading is absent, content is returned
// unchanged.
func RemoveSection(content, heading string) string {
	idx := strings.Index(content, heading)
	if idx < 0 {
		return content
	}
	rest := content[idx+len(heading):]
	end := findNextH2(rest)
	before := strings.TrimRight(content[:idx], "\n")
	if end < 0 {
		if before == "" {
			return ""
		}
		return before + "\n"
	}
	after := rest[end:]
	if before == "" {
		return after
	}
	return before + "\n\n" + after
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

// TicketMeta holds metadata parsed from the jt:meta comment in a ticket file.
type TicketMeta struct {
	Ticket  string
	Fetched time.Time
}

// TicketInfo describes a local ticket file.
type TicketInfo struct {
	Key     string
	Fetched time.Time
	Path    string
}

// ParseMeta extracts the jt:meta comment from the first line of content.
// Returns nil if the line is missing or malformed.
func ParseMeta(content string) *TicketMeta {
	line := content
	if idx := strings.IndexByte(content, '\n'); idx >= 0 {
		line = content[:idx]
	}
	const prefix = "<!-- jt:meta "
	const suffix = " -->"
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
		return nil
	}
	body := line[len(prefix) : len(line)-len(suffix)]

	var meta TicketMeta
	for _, part := range strings.Fields(body) {
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch key {
		case "ticket":
			meta.Ticket = val
		case "fetched":
			t, err := time.Parse(time.RFC3339, val)
			if err != nil {
				return nil
			}
			meta.Fetched = t
		}
	}
	if meta.Ticket == "" || meta.Fetched.IsZero() {
		return nil
	}
	return &meta
}

// ListTickets reads all .md files from ticketsDir, parses metadata from each,
// and returns them sorted by key. Files without valid metadata are skipped.
func ListTickets(ticketsDir string) ([]TicketInfo, error) {
	dir, err := expandTilde(ticketsDir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading tickets directory: %w", err)
	}

	var tickets []TicketInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		meta := ParseMeta(string(data))
		if meta == nil {
			continue
		}
		tickets = append(tickets, TicketInfo{
			Key:     meta.Ticket,
			Fetched: meta.Fetched,
			Path:    path,
		})
	}
	sort.Slice(tickets, func(i, j int) bool {
		return tickets[i].Key < tickets[j].Key
	})
	return tickets, nil
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

package renderer

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/erickhilda/jt/internal/confluence"
	"github.com/erickhilda/jt/internal/jira"
)

// RenderPage produces a self-contained markdown document for a Confluence page,
// intended for offline reading and as LLM context.
//
// spaceKey and webURL are resolved by the caller (the v2 API returns a numeric
// space id and relative links); both are optional and rendered as "-" when empty.
func RenderPage(p *confluence.Page, spaceKey, webURL string) string {
	var b strings.Builder

	now := time.Now().UTC().Format(time.RFC3339)
	fmt.Fprintf(&b, "<!-- jt:meta page=%s fetched=%s -->\n", p.ID, now)
	fmt.Fprintf(&b, "# %s\n\n", p.Title)

	// Metadata table.
	b.WriteString("| Field | Value |\n")
	b.WriteString("|-------|-------|\n")
	writeRow(&b, "Space", spaceKey)
	writeRow(&b, "Page ID", p.ID)
	writeRow(&b, "Status", p.Status)
	if p.Version != nil {
		writeRow(&b, "Version", strconv.Itoa(p.Version.Number))
		writeRow(&b, "Updated", formatDate(p.Version.CreatedAt))
	}
	writeRow(&b, "Created", formatDate(p.CreatedAt))
	writeRow(&b, "URL", webURL)
	b.WriteString("\n")

	// Content (ADF body converted to markdown).
	b.WriteString("## Content\n\n")
	if body := renderPageBody(p.Body.AtlasDocFormat.Value); body != "" {
		b.WriteString(body)
		b.WriteString("\n\n")
	} else {
		b.WriteString("*No content.*\n\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// renderPageBody converts an ADF document (encoded as a JSON string by the
// Confluence v2 API) to markdown via the shared jira.RenderADF converter. On a
// parse failure it embeds the raw value in a fenced block so content is never
// silently lost.
func renderPageBody(adfJSON string) string {
	if strings.TrimSpace(adfJSON) == "" {
		return ""
	}
	var doc jira.ADFDoc
	if err := json.Unmarshal([]byte(adfJSON), &doc); err != nil {
		return "```json\n" + strings.TrimRight(adfJSON, "\n") + "\n```"
	}
	return jira.RenderADF(&doc)
}

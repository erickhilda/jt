package renderer

import (
	"fmt"
	"strings"
	"time"

	"github.com/erickhilda/jt/internal/jira"
)

// RenderIssue produces a self-contained markdown document for a Jira issue.
func RenderIssue(issue *jira.Issue) string {
	var b strings.Builder

	now := time.Now().UTC().Format(time.RFC3339)

	// Meta comment.
	fmt.Fprintf(&b, "<!-- jt:meta ticket=%s fetched=%s -->\n", issue.Key, now)

	// Title.
	fmt.Fprintf(&b, "# %s: %s\n\n", issue.Key, issue.Fields.Summary)

	// Metadata table.
	b.WriteString("| Field | Value |\n")
	b.WriteString("|-------|-------|\n")
	writeRow(&b, "Status", safeName(issue.Fields.Status))
	writeRow(&b, "Type", safeTypeName(issue.Fields.IssueType))
	writeRow(&b, "Priority", safePriorityName(issue.Fields.Priority))
	writeRow(&b, "Assignee", safeUserDisplay(issue.Fields.Assignee))
	writeRow(&b, "Reporter", safeUserDisplay(issue.Fields.Reporter))
	if issue.Sprint != nil && issue.Sprint.Name != "" {
		writeRow(&b, "Sprint", issue.Sprint.Name)
	}
	if issue.Epic != nil {
		epicVal := issue.Epic.Key
		if issue.Epic.Summary != "" {
			epicVal = issue.Epic.Key + ": " + issue.Epic.Summary
		}
		writeRow(&b, "Epic", epicVal)
	}
	if issue.Fields.Parent != nil {
		parentVal := issue.Fields.Parent.Key
		if issue.Fields.Parent.Fields.Summary != "" {
			parentVal = issue.Fields.Parent.Key + ": " + issue.Fields.Parent.Fields.Summary
		}
		writeRow(&b, "Parent", parentVal)
	}
	if len(issue.Fields.Labels) > 0 {
		writeRow(&b, "Labels", strings.Join(issue.Fields.Labels, ", "))
	}
	writeRow(&b, "Created", formatDate(issue.Fields.Created))
	writeRow(&b, "Updated", formatDate(issue.Fields.Updated))
	b.WriteString("\n")

	// Description.
	b.WriteString("## Description\n\n")
	if issue.Fields.Description != nil {
		desc := jira.RenderADF(issue.Fields.Description)
		if desc != "" {
			b.WriteString(desc)
			b.WriteString("\n\n")
		}
	} else {
		b.WriteString("*No description provided.*\n\n")
	}

	// Subtasks.
	if len(issue.Fields.Subtasks) > 0 {
		b.WriteString("## Subtasks\n\n")
		for _, st := range issue.Fields.Subtasks {
			checkbox := " "
			statusName := ""
			if st.Fields.Status != nil {
				statusName = st.Fields.Status.Name
				lower := strings.ToLower(statusName)
				if lower == "done" || lower == "closed" || lower == "resolved" {
					checkbox = "x"
				}
			}
			suffix := ""
			if statusName != "" {
				suffix = " (" + statusName + ")"
			}
			fmt.Fprintf(&b, "- [%s] %s: %s%s\n", checkbox, st.Key, st.Fields.Summary, suffix)
		}
		b.WriteString("\n")
	}

	// Linked issues.
	if len(issue.Fields.IssueLinks) > 0 {
		b.WriteString("## Linked Issues\n\n")
		for _, link := range issue.Fields.IssueLinks {
			if link.OutwardIssue != nil && link.Type != nil {
				fmt.Fprintf(&b, "- %s %s: %s\n",
					link.Type.Outward,
					link.OutwardIssue.Key,
					link.OutwardIssue.Fields.Summary)
			}
			if link.InwardIssue != nil && link.Type != nil {
				fmt.Fprintf(&b, "- %s %s: %s\n",
					link.Type.Inward,
					link.InwardIssue.Key,
					link.InwardIssue.Fields.Summary)
			}
		}
		b.WriteString("\n")
	}

	// Comments.
	if issue.Fields.Comment != nil && issue.Fields.Comment.Total > 0 {
		fmt.Fprintf(&b, "## Comments (%d)\n\n", issue.Fields.Comment.Total)
		for _, comment := range issue.Fields.Comment.Comments {
			author := "Unknown"
			if comment.Author != nil {
				author = comment.Author.DisplayName
			}
			date := formatDate(comment.Created)
			fmt.Fprintf(&b, "### %s -- %s\n\n", author, date)
			if comment.Body != nil {
				body := jira.RenderADF(comment.Body)
				if body != "" {
					b.WriteString(body)
					b.WriteString("\n\n")
				}
			}
		}
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// RenderComments produces just the comments section markdown for an issue.
func RenderComments(issue *jira.Issue) string {
	var b strings.Builder
	if issue.Fields.Comment != nil && issue.Fields.Comment.Total > 0 {
		fmt.Fprintf(&b, "## Comments (%d)\n\n", issue.Fields.Comment.Total)
		for _, comment := range issue.Fields.Comment.Comments {
			author := "Unknown"
			if comment.Author != nil {
				author = comment.Author.DisplayName
			}
			date := formatDate(comment.Created)
			fmt.Fprintf(&b, "### %s -- %s\n\n", author, date)
			if comment.Body != nil {
				body := jira.RenderADF(comment.Body)
				if body != "" {
					b.WriteString(body)
					b.WriteString("\n\n")
				}
			}
		}
	} else {
		b.WriteString("## Comments (0)\n\n*No comments.*\n\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func writeRow(b *strings.Builder, field, value string) {
	if value == "" {
		value = "-"
	}
	fmt.Fprintf(b, "| %s | %s |\n", field, value)
}

func safeName(s *jira.Status) string {
	if s == nil {
		return ""
	}
	return s.Name
}

func safeTypeName(t *jira.IssueType) string {
	if t == nil {
		return ""
	}
	return t.Name
}

func safePriorityName(p *jira.Priority) string {
	if p == nil {
		return ""
	}
	return p.Name
}

func safeUserDisplay(u *jira.User) string {
	if u == nil {
		return ""
	}
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Email
}

// formatDate converts an ISO 8601 timestamp to YYYY-MM-DD.
func formatDate(iso string) string {
	if iso == "" {
		return "-"
	}
	// Try full ISO 8601 with timezone.
	for _, layout := range []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-0700",
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, iso); err == nil {
			return t.Format("2006-01-02")
		}
	}
	// If parsing fails, return first 10 chars if long enough.
	if len(iso) >= 10 {
		return iso[:10]
	}
	return iso
}

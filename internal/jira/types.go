package jira

import "encoding/json"

// User represents a Jira Cloud user from the /rest/api/3/myself endpoint.
type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"emailAddress"`
	Active      bool   `json:"active"`
	TimeZone    string `json:"timeZone"`
}

// Issue is the fully-parsed Jira issue returned by GetIssue.
type Issue struct {
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
	Sprint *Sprint     `json:"-"`
	Epic   *Epic       `json:"-"`
}

// IssueRaw is an intermediate type for two-pass JSON decoding.
// The Fields value is kept as raw JSON so we can extract custom fields
// after identifying them via the "names" map from ?expand=names.
type IssueRaw struct {
	Key    string          `json:"key"`
	Fields json.RawMessage `json:"fields"`
	Names  map[string]string `json:"names"`
}

// IssueFields holds the standard fields of a Jira issue.
type IssueFields struct {
	Summary     string       `json:"summary"`
	Description *ADFDoc      `json:"description"`
	Status      *Status      `json:"status"`
	IssueType   *IssueType   `json:"issuetype"`
	Priority    *Priority    `json:"priority"`
	Assignee    *User        `json:"assignee"`
	Reporter    *User        `json:"reporter"`
	Labels      []string     `json:"labels"`
	Created     string       `json:"created"`
	Updated     string       `json:"updated"`
	Comment     *CommentPage `json:"comment"`
	Subtasks    []Subtask    `json:"subtasks"`
	IssueLinks  []IssueLink  `json:"issuelinks"`
	Parent      *ParentIssue `json:"parent"`
}

// Status represents the issue status.
type Status struct {
	Name string `json:"name"`
}

// IssueType represents the issue type (Story, Bug, Task, etc.).
type IssueType struct {
	Name string `json:"name"`
}

// Priority represents the issue priority.
type Priority struct {
	Name string `json:"name"`
}

// CommentPage holds a page of comments from the Jira issue response.
type CommentPage struct {
	Total    int       `json:"total"`
	Comments []Comment `json:"comments"`
}

// Comment is a single issue comment.
type Comment struct {
	Author  *User   `json:"author"`
	Body    *ADFDoc `json:"body"`
	Created string  `json:"created"`
}

// Subtask represents a subtask of the issue.
type Subtask struct {
	Key    string        `json:"key"`
	Fields SubtaskFields `json:"fields"`
}

// SubtaskFields holds the fields we care about for subtasks.
type SubtaskFields struct {
	Summary string  `json:"summary"`
	Status  *Status `json:"status"`
}

// IssueLink represents a link between two issues.
type IssueLink struct {
	Type         *IssueLinkType `json:"type"`
	InwardIssue  *LinkedIssue   `json:"inwardIssue"`
	OutwardIssue *LinkedIssue   `json:"outwardIssue"`
}

// IssueLinkType describes the relationship (e.g. "blocks"/"is blocked by").
type IssueLinkType struct {
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
}

// LinkedIssue is a minimal issue representation used in links.
type LinkedIssue struct {
	Key    string            `json:"key"`
	Fields LinkedIssueFields `json:"fields"`
}

// LinkedIssueFields holds the fields for a linked issue.
type LinkedIssueFields struct {
	Summary string  `json:"summary"`
	Status  *Status `json:"status"`
}

// ParentIssue represents the parent of a sub-task or child issue.
type ParentIssue struct {
	Key    string            `json:"key"`
	Fields ParentIssueFields `json:"fields"`
}

// ParentIssueFields holds the fields for a parent issue.
type ParentIssueFields struct {
	Summary string  `json:"summary"`
	Status  *Status `json:"status"`
}

// Sprint represents a Jira sprint (extracted from custom fields).
type Sprint struct {
	Name string `json:"name"`
}

// Epic represents a Jira epic (extracted from custom fields).
type Epic struct {
	Key     string `json:"key"`
	Summary string `json:"summary"`
}

// ADFDoc is the top-level Atlassian Document Format document.
type ADFDoc struct {
	Type    string    `json:"type"`
	Version int       `json:"version"`
	Content []ADFNode `json:"content"`
}

// ADFNode is a node in the ADF tree (paragraph, heading, text, etc.).
type ADFNode struct {
	Type    string            `json:"type"`
	Text    string            `json:"text,omitempty"`
	Content []ADFNode         `json:"content,omitempty"`
	Marks   []ADFMark         `json:"marks,omitempty"`
	Attrs   map[string]any    `json:"attrs,omitempty"`
}

// ADFMark represents inline formatting (bold, italic, link, etc.).
type ADFMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

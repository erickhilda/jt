package renderer

import (
	"strings"
	"testing"

	"github.com/erickhilda/atlit/internal/confluence"
)

func TestRenderPage(t *testing.T) {
	p := &confluence.Page{
		ID:        "12345",
		Status:    "current",
		Title:     "Design Doc",
		CreatedAt: "2026-01-02T10:00:00.000Z",
		Version:   &confluence.Version{Number: 7, CreatedAt: "2026-06-15T09:00:00.000Z"},
	}
	p.Body.AtlasDocFormat.Value = `{"type":"doc","version":1,"content":[
		{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Overview"}]},
		{"type":"paragraph","content":[{"type":"text","text":"Hello world"}]}
	]}`

	out := RenderPage(p, "ENG", "https://acme.atlassian.net/wiki/spaces/ENG/pages/12345/Design+Doc", nil)

	for _, want := range []string{
		"<!-- atlit:meta page=12345 ",
		"# Design Doc",
		"| Space | ENG |",
		"| Page ID | 12345 |",
		"| Version | 7 |",
		"| Updated | 2026-06-15 |",
		"## Content",
		"## Overview",
		"Hello world",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderPageAttachments(t *testing.T) {
	p := &confluence.Page{ID: "12345", Title: "Doc", Status: "current"}
	p.Links.Base = "https://acme.atlassian.net/wiki"
	atts := []confluence.Attachment{
		{Title: "arch.png", MediaType: "image/png", DownloadLink: "/download/attachments/12345/arch.png?version=1"},
		{Title: "ext.png", MediaType: "image/png", DownloadLink: "https://cdn.example.com/ext.png"},
	}

	out := RenderPage(p, "ENG", "", atts)
	for _, want := range []string{
		"## Attachments",
		"- arch.png (image/png) - https://acme.atlassian.net/wiki/download/attachments/12345/arch.png?version=1",
		"- ext.png (image/png) - https://cdn.example.com/ext.png",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestAbsURL(t *testing.T) {
	cases := []struct{ base, link, want string }{
		{"https://acme.atlassian.net/wiki", "/download/x.png", "https://acme.atlassian.net/wiki/download/x.png"},
		{"https://acme.atlassian.net/wiki", "/wiki/download/x.png", "https://acme.atlassian.net/wiki/download/x.png"},
		{"https://acme.atlassian.net/wiki", "https://cdn/x.png", "https://cdn/x.png"},
		{"", "/download/x.png", "/download/x.png"},
		{"https://acme.atlassian.net/wiki", "", ""},
	}
	for _, c := range cases {
		if got := absURL(c.base, c.link); got != c.want {
			t.Errorf("absURL(%q,%q) = %q, want %q", c.base, c.link, got, c.want)
		}
	}
}

func TestRenderPageEmptyBody(t *testing.T) {
	p := &confluence.Page{ID: "1", Title: "Empty", Status: "current"}
	out := RenderPage(p, "", "", nil)
	if !strings.Contains(out, "*No content.*") {
		t.Errorf("expected empty-body placeholder, got:\n%s", out)
	}
	// Empty space/URL render as "-".
	if !strings.Contains(out, "| Space | - |") {
		t.Errorf("expected empty Space row, got:\n%s", out)
	}
}

func TestRenderPageInvalidBodyFallsBack(t *testing.T) {
	p := &confluence.Page{ID: "1", Title: "Bad", Status: "current"}
	p.Body.AtlasDocFormat.Value = "not valid json {"
	out := RenderPage(p, "ENG", "", nil)
	if !strings.Contains(out, "not valid json {") {
		t.Errorf("expected raw body fallback, got:\n%s", out)
	}
}

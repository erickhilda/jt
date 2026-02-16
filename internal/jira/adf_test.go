package jira

import "testing"

func TestRenderADFNilDoc(t *testing.T) {
	if got := RenderADF(nil); got != "" {
		t.Errorf("expected empty string for nil doc, got %q", got)
	}
}

func TestRenderADFEmptyDoc(t *testing.T) {
	doc := &ADFDoc{Type: "doc", Version: 1}
	if got := RenderADF(doc); got != "" {
		t.Errorf("expected empty string for empty doc, got %q", got)
	}
}

func TestRenderADFParagraph(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "paragraph",
				Content: []ADFNode{
					{Type: "text", Text: "Hello world"},
				},
			},
		},
	}
	want := "Hello world"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFHeadings(t *testing.T) {
	tests := []struct {
		name  string
		level int
		text  string
		want  string
	}{
		{"h1", 1, "Title", "# Title"},
		{"h2", 2, "Subtitle", "## Subtitle"},
		{"h3", 3, "Section", "### Section"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &ADFDoc{
				Type:    "doc",
				Version: 1,
				Content: []ADFNode{
					{
						Type:  "heading",
						Attrs: map[string]any{"level": float64(tt.level)},
						Content: []ADFNode{
							{Type: "text", Text: tt.text},
						},
					},
				},
			}
			got := RenderADF(doc)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderADFBoldItalicCode(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "paragraph",
				Content: []ADFNode{
					{
						Type:  "text",
						Text:  "bold",
						Marks: []ADFMark{{Type: "strong"}},
					},
					{Type: "text", Text: " and "},
					{
						Type:  "text",
						Text:  "italic",
						Marks: []ADFMark{{Type: "em"}},
					},
					{Type: "text", Text: " and "},
					{
						Type:  "text",
						Text:  "code",
						Marks: []ADFMark{{Type: "code"}},
					},
				},
			},
		},
	}
	want := "**bold** and *italic* and `code`"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFStrikethrough(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "paragraph",
				Content: []ADFNode{
					{
						Type:  "text",
						Text:  "removed",
						Marks: []ADFMark{{Type: "strike"}},
					},
				},
			},
		},
	}
	want := "~~removed~~"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFLink(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "paragraph",
				Content: []ADFNode{
					{
						Type: "text",
						Text: "click here",
						Marks: []ADFMark{
							{
								Type:  "link",
								Attrs: map[string]any{"href": "https://example.com"},
							},
						},
					},
				},
			},
		},
	}
	want := "[click here](https://example.com)"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFBulletList(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "bulletList",
				Content: []ADFNode{
					{
						Type: "listItem",
						Content: []ADFNode{
							{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Item A"}}},
						},
					},
					{
						Type: "listItem",
						Content: []ADFNode{
							{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Item B"}}},
						},
					},
				},
			},
		},
	}
	want := "- Item A\n- Item B"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFOrderedList(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "orderedList",
				Content: []ADFNode{
					{
						Type: "listItem",
						Content: []ADFNode{
							{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "First"}}},
						},
					},
					{
						Type: "listItem",
						Content: []ADFNode{
							{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Second"}}},
						},
					},
				},
			},
		},
	}
	want := "1. First\n2. Second"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFNestedList(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "bulletList",
				Content: []ADFNode{
					{
						Type: "listItem",
						Content: []ADFNode{
							{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Parent"}}},
							{
								Type: "bulletList",
								Content: []ADFNode{
									{
										Type: "listItem",
										Content: []ADFNode{
											{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Child"}}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	want := "- Parent\n  - Child"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFCodeBlock(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type:  "codeBlock",
				Attrs: map[string]any{"language": "go"},
				Content: []ADFNode{
					{Type: "text", Text: "fmt.Println(\"hello\")"},
				},
			},
		},
	}
	want := "```go\nfmt.Println(\"hello\")\n```"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFTable(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "table",
				Content: []ADFNode{
					{
						Type: "tableRow",
						Content: []ADFNode{
							{Type: "tableHeader", Content: []ADFNode{{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Name"}}}}},
							{Type: "tableHeader", Content: []ADFNode{{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Age"}}}}},
						},
					},
					{
						Type: "tableRow",
						Content: []ADFNode{
							{Type: "tableCell", Content: []ADFNode{{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Alice"}}}}},
							{Type: "tableCell", Content: []ADFNode{{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "30"}}}}},
						},
					},
				},
			},
		},
	}
	want := "| Name | Age |\n| --- | --- |\n| Alice | 30 |"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFPanel(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type:  "panel",
				Attrs: map[string]any{"panelType": "warning"},
				Content: []ADFNode{
					{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Be careful!"}}},
				},
			},
		},
	}
	want := "> **Warning:** Be careful!"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFMention(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "paragraph",
				Content: []ADFNode{
					{Type: "text", Text: "Hey "},
					{Type: "mention", Attrs: map[string]any{"text": "@Alice"}},
				},
			},
		},
	}
	want := "Hey @Alice"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFEmoji(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "paragraph",
				Content: []ADFNode{
					{Type: "text", Text: "Great "},
					{Type: "emoji", Attrs: map[string]any{"shortName": ":thumbsup:"}},
				},
			},
		},
	}
	want := "Great :thumbsup:"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFHardBreak(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "paragraph",
				Content: []ADFNode{
					{Type: "text", Text: "Line 1"},
					{Type: "hardBreak"},
					{Type: "text", Text: "Line 2"},
				},
			},
		},
	}
	want := "Line 1  \nLine 2"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFInlineCard(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "paragraph",
				Content: []ADFNode{
					{Type: "inlineCard", Attrs: map[string]any{"url": "https://example.com/page"}},
				},
			},
		},
	}
	want := "[https://example.com/page](https://example.com/page)"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFRule(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Above"}}},
			{Type: "rule"},
			{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Below"}}},
		},
	}
	want := "Above\n\n---\n\nBelow"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFBlockquote(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "blockquote",
				Content: []ADFNode{
					{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "Quoted text"}}},
				},
			},
		},
	}
	want := "> Quoted text"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderADFUnknownNode(t *testing.T) {
	doc := &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "unknownBlock",
				Content: []ADFNode{
					{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: "fallback content"}}},
				},
			},
		},
	}
	want := "fallback content"
	got := RenderADF(doc)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

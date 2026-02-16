package jira

import (
	"fmt"
	"strings"
)

// RenderADF converts an Atlassian Document Format document to markdown.
func RenderADF(doc *ADFDoc) string {
	if doc == nil || len(doc.Content) == 0 {
		return ""
	}
	c := &converter{}
	c.renderNodes(doc.Content, 0)
	return strings.TrimRight(c.buf.String(), "\n")
}

type converter struct {
	buf strings.Builder
}

func (c *converter) renderNodes(nodes []ADFNode, depth int) {
	for i, node := range nodes {
		c.renderNode(node, depth, i)
	}
}

func (c *converter) renderNode(node ADFNode, depth int, index int) {
	switch node.Type {
	case "paragraph":
		c.renderInlineChildren(node.Content, depth)
		c.buf.WriteString("\n\n")
	case "heading":
		level := attrInt(node.Attrs, "level", 1)
		c.buf.WriteString(strings.Repeat("#", level))
		c.buf.WriteString(" ")
		c.renderInlineChildren(node.Content, depth)
		c.buf.WriteString("\n\n")
	case "bulletList":
		c.renderList(node.Content, depth, false)
	case "orderedList":
		c.renderList(node.Content, depth, true)
	case "listItem":
		c.renderListItem(node, depth)
	case "codeBlock":
		lang := attrStr(node.Attrs, "language")
		c.buf.WriteString("```")
		c.buf.WriteString(lang)
		c.buf.WriteString("\n")
		c.renderPlainChildren(node.Content)
		c.buf.WriteString("\n```\n\n")
	case "blockquote":
		content := c.captureNodes(node.Content, depth)
		for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
			c.buf.WriteString("> ")
			c.buf.WriteString(line)
			c.buf.WriteString("\n")
		}
		c.buf.WriteString("\n")
	case "table":
		c.renderTable(node.Content)
	case "panel":
		panelType := attrStr(node.Attrs, "panelType")
		label := panelLabel(panelType)
		content := c.captureNodes(node.Content, depth)
		content = strings.TrimRight(content, "\n")
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			c.buf.WriteString("> ")
			if i == 0 && label != "" {
				c.buf.WriteString(label)
				c.buf.WriteString(" ")
			}
			c.buf.WriteString(line)
			c.buf.WriteString("\n")
		}
		c.buf.WriteString("\n")
	case "rule":
		c.buf.WriteString("---\n\n")
	case "mediaSingle", "mediaGroup":
		// Media nodes contain media children; skip gracefully.
		c.renderNodes(node.Content, depth)
	default:
		// Unknown block-level node: try to render children.
		if len(node.Content) > 0 {
			c.renderNodes(node.Content, depth)
		}
	}
}

func (c *converter) renderInlineChildren(nodes []ADFNode, depth int) {
	for _, node := range nodes {
		c.renderInline(node, depth)
	}
}

func (c *converter) renderInline(node ADFNode, depth int) {
	switch node.Type {
	case "text":
		text := node.Text
		text = applyMarks(text, node.Marks)
		c.buf.WriteString(text)
	case "mention":
		name := attrStr(node.Attrs, "text")
		if name == "" {
			name = node.Text
		}
		if !strings.HasPrefix(name, "@") {
			name = "@" + name
		}
		c.buf.WriteString(name)
	case "emoji":
		shortName := attrStr(node.Attrs, "shortName")
		if shortName != "" {
			c.buf.WriteString(shortName)
		}
	case "hardBreak":
		c.buf.WriteString("  \n")
	case "inlineCard":
		url := attrStr(node.Attrs, "url")
		if url != "" {
			c.buf.WriteString("[")
			c.buf.WriteString(url)
			c.buf.WriteString("](")
			c.buf.WriteString(url)
			c.buf.WriteString(")")
		}
	default:
		// Unknown inline: render children if any.
		if len(node.Content) > 0 {
			c.renderInlineChildren(node.Content, depth)
		}
		if node.Text != "" {
			c.buf.WriteString(node.Text)
		}
	}
}

func applyMarks(text string, marks []ADFMark) string {
	for _, mark := range marks {
		switch mark.Type {
		case "strong":
			text = "**" + text + "**"
		case "em":
			text = "*" + text + "*"
		case "code":
			text = "`" + text + "`"
		case "strike":
			text = "~~" + text + "~~"
		case "link":
			href := ""
			if mark.Attrs != nil {
				if h, ok := mark.Attrs["href"].(string); ok {
					href = h
				}
			}
			text = "[" + text + "](" + href + ")"
		}
	}
	return text
}

func (c *converter) renderList(items []ADFNode, depth int, ordered bool) {
	for i, item := range items {
		prefix := "- "
		if ordered {
			prefix = fmt.Sprintf("%d. ", i+1)
		}
		indent := strings.Repeat("  ", depth)
		c.buf.WriteString(indent)
		c.buf.WriteString(prefix)
		c.renderListItemContent(item, depth)
	}
	if depth == 0 {
		c.buf.WriteString("\n")
	}
}

func (c *converter) renderListItem(node ADFNode, depth int) {
	c.renderListItemContent(node, depth)
}

func (c *converter) renderListItemContent(item ADFNode, depth int) {
	for i, child := range item.Content {
		switch child.Type {
		case "paragraph":
			c.renderInlineChildren(child.Content, depth)
			if i < len(item.Content)-1 {
				c.buf.WriteString("\n")
			} else {
				c.buf.WriteString("\n")
			}
		case "bulletList":
			if i == 0 {
				c.buf.WriteString("\n")
			}
			c.renderList(child.Content, depth+1, false)
		case "orderedList":
			if i == 0 {
				c.buf.WriteString("\n")
			}
			c.renderList(child.Content, depth+1, true)
		default:
			captured := c.captureNode(child, depth)
			c.buf.WriteString(strings.TrimRight(captured, "\n"))
			c.buf.WriteString("\n")
		}
	}
}

func (c *converter) renderPlainChildren(nodes []ADFNode) {
	for _, node := range nodes {
		if node.Text != "" {
			c.buf.WriteString(node.Text)
		}
		if len(node.Content) > 0 {
			c.renderPlainChildren(node.Content)
		}
	}
}

func (c *converter) renderTable(rows []ADFNode) {
	if len(rows) == 0 {
		return
	}

	var table [][]string
	for _, row := range rows {
		if row.Type != "tableRow" {
			continue
		}
		var cells []string
		for _, cell := range row.Content {
			content := c.captureNodes(cell.Content, 0)
			content = strings.TrimRight(content, "\n")
			content = strings.ReplaceAll(content, "\n", " ")
			cells = append(cells, content)
		}
		table = append(table, cells)
	}

	if len(table) == 0 {
		return
	}

	// Determine column count from the widest row.
	maxCols := 0
	for _, row := range table {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Write header row.
	c.writeTableRow(table[0], maxCols)

	// Write separator.
	c.buf.WriteString("|")
	for i := 0; i < maxCols; i++ {
		c.buf.WriteString(" --- |")
	}
	c.buf.WriteString("\n")

	// Write data rows.
	for _, row := range table[1:] {
		c.writeTableRow(row, maxCols)
	}
	c.buf.WriteString("\n")
}

func (c *converter) writeTableRow(cells []string, maxCols int) {
	c.buf.WriteString("|")
	for i := 0; i < maxCols; i++ {
		c.buf.WriteString(" ")
		if i < len(cells) {
			c.buf.WriteString(cells[i])
		}
		c.buf.WriteString(" |")
	}
	c.buf.WriteString("\n")
}

// captureNodes renders nodes into a temporary buffer and returns the result.
func (c *converter) captureNodes(nodes []ADFNode, depth int) string {
	saved := c.buf
	c.buf = strings.Builder{}
	c.renderNodes(nodes, depth)
	result := c.buf.String()
	c.buf = saved
	return result
}

func (c *converter) captureNode(node ADFNode, depth int) string {
	saved := c.buf
	c.buf = strings.Builder{}
	c.renderNode(node, depth, 0)
	result := c.buf.String()
	c.buf = saved
	return result
}

func panelLabel(panelType string) string {
	switch strings.ToLower(panelType) {
	case "info":
		return "**Info:**"
	case "note":
		return "**Note:**"
	case "warning":
		return "**Warning:**"
	case "error":
		return "**Error:**"
	case "success":
		return "**Success:**"
	default:
		return ""
	}
}

func attrStr(attrs map[string]any, key string) string {
	if attrs == nil {
		return ""
	}
	if v, ok := attrs[key].(string); ok {
		return v
	}
	return ""
}

func attrInt(attrs map[string]any, key string, defaultVal int) int {
	if attrs == nil {
		return defaultVal
	}
	switch v := attrs[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return defaultVal
	}
}

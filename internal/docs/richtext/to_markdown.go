package richtext

import (
	"fmt"
	"strings"
)

// JSONToMarkdown renders a canonical Tiptap document as markdown.
func JSONToMarkdown(body string) (string, error) {
	doc, err := UnmarshalJSON(body)
	if err != nil {
		return "", err
	}
	var blocks []string
	for _, n := range doc.Content {
		rendered, err := renderBlock(n)
		if err != nil {
			return "", err
		}
		blocks = append(blocks, rendered)
	}
	return strings.Join(blocks, "\n\n"), nil
}

func renderBlock(n Node) (string, error) {
	switch n.Type {
	case "paragraph":
		return renderInline(n.Content), nil
	case "heading":
		return renderHeading(n), nil
	case "blockquote":
		return renderBlockquote(n)
	case "bulletList", "orderedList":
		return renderList(n)
	case "codeBlock":
		return renderCodeBlock(n), nil
	case "horizontalRule":
		return "---", nil
	case "image":
		return renderImage(n), nil
	case "text":
		return renderInline([]Node{n}), nil
	default:
		return renderUnknownBlock(n), nil
	}
}

func renderHeading(n Node) string {
	level := 1
	if l, ok := n.Attrs["level"].(float64); ok {
		level = int(l)
	}
	if level < 1 || level > 6 {
		level = 1
	}
	return strings.Repeat("#", level) + " " + renderInline(n.Content)
}

func renderBlockquote(n Node) (string, error) {
	var lines []string
	for _, c := range n.Content {
		rendered, err := renderBlock(c)
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(rendered, "\n") {
			lines = append(lines, "> "+line)
		}
	}
	return strings.Join(lines, "\n"), nil
}

func renderList(n Node) (string, error) {
	start := 1
	if n.Type == "orderedList" {
		if s, ok := n.Attrs["start"].(float64); ok {
			start = int(s)
		}
	}
	var items []string
	for i, item := range n.Content {
		marker := "- "
		if n.Type == "orderedList" {
			marker = fmt.Sprintf("%d. ", start+i)
		}
		rendered, err := renderListItem(item, marker)
		if err != nil {
			return "", err
		}
		items = append(items, rendered)
	}
	return strings.Join(items, "\n"), nil
}

func renderCodeBlock(n Node) string {
	lang, _ := n.Attrs["language"].(string)
	var code strings.Builder
	for _, c := range n.Content {
		if c.Type == "text" {
			code.WriteString(c.Text)
		}
	}
	text := strings.TrimSuffix(code.String(), "\n")
	if lang != "" {
		return "```" + lang + "\n" + text + "\n```"
	}
	return "```\n" + text + "\n```"
}

func renderImage(n Node) string {
	src, _ := n.Attrs["src"].(string)
	alt, _ := n.Attrs["alt"].(string)
	title, _ := n.Attrs["title"].(string)
	if title != "" {
		return "![" + escapeMarkdownText(alt) + "](" + src + " \"" + title + "\")"
	}
	return "![" + escapeMarkdownText(alt) + "](" + src + ")"
}

// renderUnknownBlock renders an unrecognized node's text content so nothing is lost.
func renderUnknownBlock(n Node) string {
	var b strings.Builder
	for _, c := range n.Content {
		b.WriteString(renderInline([]Node{c}))
	}
	return b.String()
}

func renderListItem(item Node, marker string) (string, error) {
	if len(item.Content) == 0 {
		return marker, nil
	}
	first, rest := item.Content[0], item.Content[1:]
	firstMD := renderInline([]Node{first})
	if first.Type == "paragraph" {
		firstMD = renderInline(first.Content)
	}
	lines := []string{marker + firstMD}
	for _, nested := range rest {
		rendered, err := renderBlock(nested)
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(rendered, "\n") {
			lines = append(lines, "  "+line)
		}
	}
	return strings.Join(lines, "\n"), nil
}

// renderInline renders inline nodes, keeping shared mark delimiters open across runs so nested emphasis round-trips.
func renderInline(nodes []Node) string {
	var b strings.Builder
	var open []Mark
	for _, n := range nodes {
		switch n.Type {
		case "hardBreak":
			closeMarks(&b, open, nil)
			open = nil
			b.WriteString("  \n")
		case "mention":
			kind, _ := n.Attrs["type"].(string)
			id, _ := n.Attrs["id"].(string)
			label, _ := n.Attrs["label"].(string)
			href := MentionHref(kind, id)
			if href == "" {
				continue
			}
			closeMarks(&b, open, nil)
			open = nil
			b.WriteString("[" + escapeMarkdownText(label) + "](" + href + ")")
		case "text":
			open = renderTextRun(&b, n.Text, n.Marks, open)
		case "horizontalRule":
			closeMarks(&b, open, nil)
			open = nil
			b.WriteString("\n\n---\n\n")
		default:
			rendered, err := renderBlock(n)
			if err == nil {
				closeMarks(&b, open, nil)
				open = nil
				b.WriteString(rendered)
			}
		}
	}
	closeMarks(&b, open, nil)
	return b.String()
}

// renderTextRun writes one text run, closing ended marks and opening new ones, keeping the shared prefix open.
func renderTextRun(b *strings.Builder, text string, marks []Mark, open []Mark) []Mark {
	text = escapeMarkdownText(text)
	shared := sharedMarks(open, marks)

	closeMarks(b, open, shared)
	openNewMarks(b, marks, shared)
	b.WriteString(text)
	return cloneMarks(marks)
}

func sharedMarks(a, b []Mark) []Mark {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var shared []Mark
	for i := 0; i < n; i++ {
		if a[i].Type != b[i].Type || fmt.Sprint(a[i].Attrs) != fmt.Sprint(b[i].Attrs) {
			break
		}
		shared = append(shared, a[i])
	}
	return shared
}

func closeMarks(b *strings.Builder, open, keep []Mark) {
	keepCount := len(keep)
	for i := len(open) - 1; i >= keepCount; i-- {
		b.WriteString(markDelimiter(open[i], false))
	}
}

func openNewMarks(b *strings.Builder, marks, already []Mark) {
	for i := len(already); i < len(marks); i++ {
		b.WriteString(markDelimiter(marks[i], true))
	}
}

func markDelimiter(m Mark, open bool) string {
	switch m.Type {
	case "bold":
		return "**"
	case "italic":
		return "*"
	case "strike":
		return "~~"
	case "code":
		return "`"
	case "link":
		if open {
			return "["
		}
		href, _ := m.Attrs["href"].(string)
		return "](" + href + ")"
	}
	return ""
}

func escapeMarkdownText(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\', '*', '_', '`', '[', ']', '#', '~':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

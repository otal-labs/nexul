package richtext

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownToDoc parses CommonMark/GFM into a Tiptap document; unmappable content flattens to text, never dropped.
func MarkdownToDoc(md string) (*Doc, error) {
	source := []byte(md)
	gm := goldmark.New(goldmark.WithExtensions(extension.GFM))
	doc := gm.Parser().Parse(text.NewReader(source))

	content, err := blocks(doc, source)
	if err != nil {
		return nil, err
	}
	if len(content) == 0 {
		content = []Node{{Type: "paragraph"}}
	}
	return &Doc{Type: "doc", Content: content}, nil
}

// blocks converts the block-level children of a node into Tiptap block nodes.
func blocks(parent ast.Node, source []byte) ([]Node, error) {
	var out []Node
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		n, err := block(child, source)
		if err != nil {
			return nil, err
		}
		if n.Type != "" {
			out = append(out, n)
		}
	}
	return out, nil
}

func block(n ast.Node, source []byte) (Node, error) {
	switch v := n.(type) {
	case *ast.Paragraph, *ast.TextBlock:
		return blockParagraph(n, source)
	case *ast.Heading:
		return blockHeading(v, source)
	case *ast.Blockquote:
		return blockBlockquote(v, source)
	case *ast.FencedCodeBlock:
		return blockFencedCode(v, source), nil
	case *ast.CodeBlock:
		return blockCodeBlock(v, source), nil
	case *ast.List:
		return blockList(v, source)
	case *ast.ThematicBreak:
		return Node{Type: "horizontalRule"}, nil
	case *ast.AutoLink:
		return blockAutoLink(v, source), nil
	default:
		return blockFallback(n, source), nil
	}
}

// blockParagraph maps a paragraph/text block. The editor's image node is block-level: an image-only paragraph
// maps to it; mixed-in images flatten to alt text.
func blockParagraph(n ast.Node, source []byte) (Node, error) {
	if img, ok := soleImage(n); ok {
		return imageNode(img, source), nil
	}
	inline, err := inlines(n, source, nil)
	if err != nil {
		return Node{}, err
	}
	return Node{Type: "paragraph", Content: inline}, nil
}

func blockHeading(v *ast.Heading, source []byte) (Node, error) {
	inline, err := inlines(v, source, nil)
	if err != nil {
		return Node{}, err
	}
	return Node{Type: "heading", Attrs: map[string]any{"level": v.Level}, Content: inline}, nil
}

func blockBlockquote(v *ast.Blockquote, source []byte) (Node, error) {
	content, err := blocks(v, source)
	if err != nil {
		return Node{}, err
	}
	return Node{Type: "blockquote", Content: content}, nil
}

func blockFencedCode(v *ast.FencedCodeBlock, source []byte) Node {
	var lang string
	if l := v.Language(source); l != nil {
		lang = string(l)
	}
	return Node{
		Type:    "codeBlock",
		Attrs:   map[string]any{"language": lang},
		Content: []Node{{Type: "text", Text: nodeText(v.Lines(), source)}},
	}
}

func blockCodeBlock(v *ast.CodeBlock, source []byte) Node {
	return Node{
		Type:    "codeBlock",
		Content: []Node{{Type: "text", Text: nodeText(v.Lines(), source)}},
	}
}

func blockList(v *ast.List, source []byte) (Node, error) {
	var items []Node
	for item := v.FirstChild(); item != nil; item = item.NextSibling() {
		if _, ok := item.(*ast.ListItem); !ok {
			continue
		}
		content, err := blocks(item, source)
		if err != nil {
			return Node{}, err
		}
		if len(content) == 0 {
			content = []Node{{Type: "paragraph"}}
		}
		items = append(items, Node{Type: "listItem", Content: content})
	}
	typ := "bulletList"
	var attrs map[string]any
	if v.IsOrdered() {
		typ = "orderedList"
		if v.Start != 1 {
			attrs = map[string]any{"start": v.Start}
		}
	}
	return Node{Type: typ, Attrs: attrs, Content: items}, nil
}

func blockAutoLink(v *ast.AutoLink, source []byte) Node {
	url := string(v.URL(source))
	return Node{Type: "text", Text: url, Marks: []Mark{{Type: "link", Attrs: map[string]any{"href": url}}}}
}

// blockFallback flattens unhandled nodes to their text so content is never silently dropped (images, task lists, tables).
func blockFallback(n ast.Node, source []byte) Node {
	var b bytes.Buffer
	_ = ast.Walk(n, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := node.(*ast.Text); ok {
			b.Write(t.Value(source))
		}
		if s, ok := node.(*ast.String); ok {
			b.Write(s.Value)
		}
		return ast.WalkContinue, nil
	})
	text := strings.TrimSpace(b.String())
	if text == "" {
		return Node{}
	}
	return Node{Type: "paragraph", Content: []Node{{Type: "text", Text: text}}}
}

func soleImage(n ast.Node) (*ast.Image, bool) {
	first := n.FirstChild()
	if first == nil || first != n.LastChild() {
		return nil, false
	}
	img, ok := first.(*ast.Image)
	return img, ok
}

func imageNode(img *ast.Image, source []byte) Node {
	attrs := map[string]any{"src": string(img.Destination), "alt": inlineText(img, source)}
	if len(img.Title) > 0 {
		attrs["title"] = string(img.Title)
	}
	return Node{Type: "image", Attrs: attrs}
}

// inlines converts the inline children of a node into Tiptap text nodes, carrying the active mark stack down the tree.
func inlines(parent ast.Node, source []byte, marks []Mark) ([]Node, error) {
	var out []Node
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		nodes, err := inlineNodes(child, source, marks)
		if err != nil {
			return nil, err
		}
		out = append(out, nodes...)
	}
	return mergeAdjacentText(out), nil
}

// inlineNodes converts one inline child into zero or more Tiptap nodes, carrying the active mark stack down
// nested spans (emphasis, strikethrough, links).
func inlineNodes(child ast.Node, source []byte, marks []Mark) ([]Node, error) {
	switch v := child.(type) {
	case *ast.Text:
		return inlineTextNode(v, source, marks), nil
	case *ast.String:
		return inlineStringNode(v, marks), nil
	case *ast.CodeSpan:
		return []Node{inlineCodeSpan(child, source, marks)}, nil
	case *ast.Emphasis:
		return inlineEmphasis(v, child, source, marks)
	case *gast.Strikethrough:
		return inlines(child, source, append(cloneMarks(marks), Mark{Type: "strike"}))
	case *ast.Link:
		return inlineLink(v, child, source, marks)
	case *ast.AutoLink:
		return []Node{inlineAutoLink(v, source, marks)}, nil
	case *ast.RawHTML:
		return inlineRawHTML(v, source, marks), nil
	default:
		return inlineFallback(child, source, marks), nil
	}
}

func inlineTextNode(v *ast.Text, source []byte, marks []Mark) []Node {
	text := string(v.Value(source))
	if v.HardLineBreak() {
		var out []Node
		if text != "" {
			out = append(out, Node{Type: "text", Text: text, Marks: cloneMarks(marks)})
		}
		return append(out, Node{Type: "hardBreak"})
	}
	if text == "" {
		return nil
	}
	return []Node{{Type: "text", Text: text, Marks: cloneMarks(marks)}}
}

func inlineStringNode(v *ast.String, marks []Mark) []Node {
	text := string(v.Value)
	if text == "" {
		return nil
	}
	return []Node{{Type: "text", Text: text, Marks: cloneMarks(marks)}}
}

func inlineCodeSpan(child ast.Node, source []byte, marks []Mark) Node {
	var b bytes.Buffer
	for c := child.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			b.Write(t.Value(source))
		}
		if s, ok := c.(*ast.String); ok {
			b.Write(s.Value)
		}
	}
	return Node{Type: "text", Text: b.String(), Marks: append(cloneMarks(marks), Mark{Type: "code"})}
}

func inlineEmphasis(v *ast.Emphasis, child ast.Node, source []byte, marks []Mark) ([]Node, error) {
	mark := "italic"
	if v.Level == 2 {
		mark = "bold"
	}
	return inlines(child, source, append(cloneMarks(marks), Mark{Type: mark}))
}

func inlineLink(v *ast.Link, child ast.Node, source []byte, marks []Mark) ([]Node, error) {
	href := string(v.Destination)
	if kind, id, ok := ParseMentionHref(href); ok {
		label := inlineText(child, source)
		return []Node{{Type: "mention", Attrs: map[string]any{"type": kind, "id": id, "label": label}}}, nil
	}
	return inlines(child, source, append(cloneMarks(marks), Mark{Type: "link", Attrs: map[string]any{"href": href}}))
}

func inlineAutoLink(v *ast.AutoLink, source []byte, marks []Mark) Node {
	url := string(v.URL(source))
	return Node{Type: "text", Text: url, Marks: append(cloneMarks(marks), Mark{Type: "link", Attrs: map[string]any{"href": url}})}
}

// inlineRawHTML matches the Tiptap markdown pipeline, which turns inline HTML into text.
func inlineRawHTML(v *ast.RawHTML, source []byte, marks []Mark) []Node {
	text := strings.TrimSpace(string(v.Segments.Value(source)))
	if text == "" {
		return nil
	}
	return []Node{{Type: "text", Text: text, Marks: cloneMarks(marks)}}
}

// inlineFallback flattens unknown inline nodes to text (images).
func inlineFallback(child ast.Node, source []byte, marks []Mark) []Node {
	var b bytes.Buffer
	_ = ast.Walk(child, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := node.(*ast.Text); ok {
			b.Write(t.Value(source))
		}
		if s, ok := node.(*ast.String); ok {
			b.Write(s.Value)
		}
		return ast.WalkContinue, nil
	})
	text := strings.TrimSpace(b.String())
	if text == "" {
		return nil
	}
	return []Node{{Type: "text", Text: text, Marks: cloneMarks(marks)}}
}

// mergeAdjacentText keeps the document small and round-trippable.
func mergeAdjacentText(nodes []Node) []Node {
	out := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		if n.Type == "text" && len(out) > 0 {
			last := &out[len(out)-1]
			if last.Type == "text" && marksEqual(last.Marks, n.Marks) {
				last.Text += n.Text
				continue
			}
		}
		out = append(out, n)
	}
	return out
}

func marksEqual(a, b []Mark) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Type != b[i].Type || fmt.Sprint(a[i].Attrs) != fmt.Sprint(b[i].Attrs) {
			return false
		}
	}
	return true
}

func cloneMarks(marks []Mark) []Mark {
	if len(marks) == 0 {
		return nil
	}
	out := make([]Mark, len(marks))
	copy(out, marks)
	return out
}

func nodeText(segments *text.Segments, source []byte) string {
	return strings.TrimSpace(string(segments.Value(source)))
}

// inlineText extracts the plain text of an inline node's children (e.g. a link label), flattening any marks inside.
func inlineText(parent ast.Node, source []byte) string {
	var b bytes.Buffer
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			b.Write(t.Value(source))
		}
		if s, ok := c.(*ast.String); ok {
			b.Write(s.Value)
		}
	}
	return b.String()
}

package richtext

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownToDoc_BasicStructure(t *testing.T) {
	doc, err := MarkdownToDoc("# Hello\n\nThis is **bold** and *italic*.")
	require.NoError(t, err)

	require.Len(t, doc.Content, 2)
	heading := doc.Content[0]
	assert.Equal(t, "heading", heading.Type)
	assert.Equal(t, 1, heading.Attrs["level"])
	assert.Equal(t, "Hello", heading.Content[0].Text)

	para := doc.Content[1]
	assert.Equal(t, "paragraph", para.Type)
	texts := collectText(para.Content)
	require.Len(t, texts, 5)
	assert.Equal(t, "This is ", texts[0].Text)
	assert.Equal(t, "bold", texts[1].Text)
	assert.Equal(t, "bold", texts[1].Marks[0].Type)
	assert.Equal(t, "italic", texts[3].Marks[0].Type)
}

// markdownToJSON is a test-only convenience wrapping MarkdownToDoc and MarshalJSON, because production code has no caller for the combined step.
func markdownToJSON(md string) (string, error) {
	doc, err := MarkdownToDoc(md)
	if err != nil {
		return "", err
	}
	return MarshalJSON(doc)
}

func TestMarkdownToJSON_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		md   string
	}{
		{"empty", ""},
		{"plain paragraph", "just some text"},
		{"heading + para", "# Title\n\nBody text."},
		{"bold italic strike", "**bold** *italic* ~~strike~~"},
		{"inline code", "use `go build` here"},
		{"fenced code", "```go\npackage main\n```"},
		{"bullet list", "- one\n- two\n- three"},
		{"ordered list", "1. first\n2. second"},
		{"blockquote", "> quoted line"},
		{"hr", "---"},
		{"link", "see [docs](https://example.com) please"},
		{"hard break", "line one  \nline two"},
		{"nested list", "- a\n  - b\n- c"},
		{"autolink", "visit https://example.com now"},
		{"emphasis nesting", "***both*** and **only**"},
		{"strikethrough", "~~gone~~ here"},
		{"inline html", "text <u>underlined</u> more"},
		{"raw html", "<div>raw block</div>"},
		{"image flattened", "![alt](img.png)"},
		{"fenced with lang", "```javascript\nvar x = 1;\n```"},
		{"ordered with start", "3. three\n4. four"},
		{"nested blockquote", "> outer\n>\n> > inner"},
		{"list with code", "- item\n\n  ```go\n  x\n  ```"},
		{"emphasis mixed", "a **b *c* d** e"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr, err := markdownToJSON(tt.md)
			require.NoError(t, err)
			require.True(t, IsStructured(jsonStr), "output must be structured")

			back, err := JSONToMarkdown(jsonStr)
			require.NoError(t, err)

			// Re-parse the markdown we produced: it must stay stable.
			again, err := markdownToJSON(back)
			require.NoError(t, err)
			assert.Equal(t, jsonStr, again, "markdown round-trip must be idempotent")
		})
	}
}

func TestIsStructured(t *testing.T) {
	assert.True(t, IsStructured(`{"type":"doc","content":[{"type":"paragraph"}]}`))
	assert.True(t, IsStructured(`[{"type":"paragraph"}]`))
	assert.False(t, IsStructured(""))
	assert.False(t, IsStructured("# markdown heading"))
	assert.False(t, IsStructured("plain text"))
	assert.False(t, IsStructured(`{"type":"not-a-doc"}`))
}

func TestNormalize(t *testing.T) {
	t.Run("structured passes through", func(t *testing.T) {
		body := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hi"}]}]}`
		got, err := Normalize(body)
		require.NoError(t, err)
		assert.Equal(t, body, got)
	})
	t.Run("markdown converts", func(t *testing.T) {
		got, err := Normalize("# Hi")
		require.NoError(t, err)
		require.True(t, IsStructured(got))
		md, err := JSONToMarkdown(got)
		require.NoError(t, err)
		assert.Equal(t, "# Hi", md)
	})
	t.Run("empty becomes empty doc", func(t *testing.T) {
		got, err := Normalize("")
		require.NoError(t, err)
		require.True(t, IsStructured(got))
	})
}

func TestToMarkdown(t *testing.T) {
	t.Run("legacy markdown passes through", func(t *testing.T) {
		got, err := ToMarkdown("plain markdown body")
		require.NoError(t, err)
		assert.Equal(t, "plain markdown body", got)
	})
	t.Run("structured converts", func(t *testing.T) {
		got, err := ToMarkdown(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hi"}]}]}`)
		require.NoError(t, err)
		assert.Equal(t, "hi", got)
	})
}

func TestSearchText(t *testing.T) {
	body := `{"type":"doc","content":[{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Storage"}]},{"type":"paragraph","content":[{"type":"text","text":"SQLite is the spine"}]}]}`
	got := SearchText(body)
	assert.Contains(t, got, "Storage")
	assert.Contains(t, got, "SQLite")

	legacy := SearchText("legacy markdown ## content")
	assert.Equal(t, "legacy markdown ## content", legacy)
}

func TestUnmarshalErrors(t *testing.T) {
	_, err := UnmarshalJSON("")
	require.Error(t, err)
	_, err = UnmarshalJSON("{not json")
	require.Error(t, err)
}

func TestEscapeMarkdownText(t *testing.T) {
	got := escapeMarkdownText(`a * b _ c ` + "`" + ` d [ e ] f # g ~ h \ i`)
	assert.Equal(t, `a \* b \_ c \`+"`"+` d \[ e \] f \# g \~ h \\ i`, got)
}

func TestSearchTextErrorFallback(t *testing.T) {
	// A body that is neither JSON nor markdown falls back to itself.
	assert.Equal(t, "raw", SearchText("raw"))
}

func TestMarshalJSON(t *testing.T) {
	doc := &Doc{Type: "doc", Content: []Node{{Type: "paragraph"}}}
	got, err := MarshalJSON(doc)
	require.NoError(t, err)
	assert.Equal(t, `{"type":"doc","content":[{"type":"paragraph"}]}`, got)
}

func TestUnmarshalJSON_ArrayForm(t *testing.T) {
	doc, err := UnmarshalJSON(`[{"type":"paragraph"}]`)
	require.NoError(t, err)
	assert.Equal(t, "doc", doc.Type)
	require.Len(t, doc.Content, 1)
}

func TestNormalize_InvalidBody(t *testing.T) {
	_, err := Normalize("")
	require.NoError(t, err) // empty → empty doc
}

func collectText(nodes []Node) []Node {
	var out []Node
	for _, n := range nodes {
		if n.Type == "text" {
			out = append(out, n)
		}
	}
	return out
}

var _ = strings.TrimSpace

func TestMarkdownToJSON_SpecificNodes(t *testing.T) {
	tests := []struct {
		md       string
		wantType string
	}{
		{"3. three\n4. four", "orderedList"},
		{"> quote", "blockquote"},
		{"```go\nx\n```", "codeBlock"},
		{"- item with ![img](x.png)", "bulletList"},
	}
	for _, tt := range tests {
		t.Run(tt.md, func(t *testing.T) {
			doc, err := MarkdownToDoc(tt.md)
			require.NoError(t, err)
			require.NotEmpty(t, doc.Content)
			assert.Equal(t, tt.wantType, doc.Content[0].Type)
		})
	}
}

func TestMarkdownToDoc_TightListItemIsParagraph(t *testing.T) {
	doc, err := MarkdownToDoc("- one\n- two")
	require.NoError(t, err)
	list := doc.Content[0]
	require.Equal(t, "bulletList", list.Type)
	require.Len(t, list.Content, 2)
	item := list.Content[0]
	require.Equal(t, "listItem", item.Type)
	require.Equal(t, "paragraph", item.Content[0].Type)
}

func TestMarkdownToDoc_HardBreakRoundTrip(t *testing.T) {
	doc, err := MarkdownToDoc("line one  \nline two")
	require.NoError(t, err)
	para := doc.Content[0]
	require.Equal(t, "paragraph", para.Type)
	var hasBreak bool
	for _, c := range para.Content {
		if c.Type == "hardBreak" {
			hasBreak = true
		}
	}
	assert.True(t, hasBreak, "hard break node present")

	body, err := MarshalJSON(doc)
	require.NoError(t, err)
	md, err := JSONToMarkdown(body)
	require.NoError(t, err)
	assert.Contains(t, md, "line one")
	assert.Contains(t, md, "line two")
}

func TestJSONToMarkdown_RichDoc(t *testing.T) {
	doc := &Doc{Type: "doc", Content: []Node{
		{Type: "heading", Attrs: map[string]any{"level": 2}, Content: []Node{{Type: "text", Text: "Section"}}},
		{Type: "paragraph", Content: []Node{{Type: "text", Text: "bold", Marks: []Mark{{Type: "bold"}}}}},
		{Type: "codeBlock", Attrs: map[string]any{"language": "go"}, Content: []Node{{Type: "text", Text: "package x"}}},
		{Type: "blockquote", Content: []Node{{Type: "paragraph", Content: []Node{{Type: "text", Text: "quoted"}}}}},
		{Type: "bulletList", Content: []Node{{Type: "listItem", Content: []Node{{Type: "paragraph", Content: []Node{{Type: "text", Text: "a"}}}}}}},
		{Type: "horizontalRule"},
		{Type: "paragraph", Content: []Node{{Type: "text", Text: "link", Marks: []Mark{{Type: "link", Attrs: map[string]any{"href": "https://x.com"}}}}}},
	}}
	body, err := MarshalJSON(doc)
	require.NoError(t, err)
	md, err := JSONToMarkdown(body)
	require.NoError(t, err)
	assert.Contains(t, md, "## Section")
	assert.Contains(t, md, "**bold**")
	assert.Contains(t, md, "```go")
	assert.Contains(t, md, "> quoted")
	assert.Contains(t, md, "- a")
	assert.Contains(t, md, "---")
	assert.Contains(t, md, "[link](https://x.com)")
}

func TestMarkdownToDoc_IndentedCodeBlock(t *testing.T) {
	doc, err := MarkdownToDoc("    indented code")
	require.NoError(t, err)
	require.Equal(t, "codeBlock", doc.Content[0].Type)
}

func TestMarkdownToDoc_AutoLinkInParagraph(t *testing.T) {
	doc, err := MarkdownToDoc("visit https://example.com now")
	require.NoError(t, err)
	para := doc.Content[0]
	var found bool
	for _, c := range para.Content {
		if c.Type == "text" && len(c.Marks) == 1 && c.Marks[0].Type == "link" {
			found = true
		}
	}
	assert.True(t, found, "autolink mark found in paragraph")
}

func TestMarkdownToDoc_StrikethroughInline(t *testing.T) {
	doc, err := MarkdownToDoc("a ~~gone~~ b")
	require.NoError(t, err)
	para := doc.Content[0]
	var found bool
	for _, c := range para.Content {
		if c.Type == "text" && len(c.Marks) == 1 && c.Marks[0].Type == "strike" {
			found = true
		}
	}
	assert.True(t, found, "strike mark found")
}

func TestMarkdownToDoc_ImageInsideEmphasis(t *testing.T) {
	doc, err := MarkdownToDoc("*text with ![alt](img.png) inside*")
	require.NoError(t, err)
	require.NotEmpty(t, doc.Content)
	assert.Equal(t, "paragraph", doc.Content[0].Type)
}

func TestJSONToMarkdown_UnknownNodeFlattens(t *testing.T) {
	doc := &Doc{Type: "doc", Content: []Node{
		{Type: "custom", Content: []Node{{Type: "text", Text: "kept text"}}},
	}}
	body, err := MarshalJSON(doc)
	require.NoError(t, err)
	md, err := JSONToMarkdown(body)
	require.NoError(t, err)
	assert.Equal(t, "kept text", md)
}

func TestMarkdownToDoc_SoleImageParagraphIsImageNode(t *testing.T) {
	doc, err := MarkdownToDoc("before\n\n![A shot](/api/attachments/abc \"Title\")\n\nafter")
	require.NoError(t, err)
	require.Len(t, doc.Content, 3)
	assert.Equal(t, "image", doc.Content[1].Type)
	assert.Equal(t, map[string]any{"src": "/api/attachments/abc", "alt": "A shot", "title": "Title"}, doc.Content[1].Attrs)
}

func TestImageMarkdown_RoundTrip(t *testing.T) {
	md := "![shot](/api/attachments/abc)"
	doc, err := MarkdownToDoc(md)
	require.NoError(t, err)
	body, err := MarshalJSON(doc)
	require.NoError(t, err)
	got, err := JSONToMarkdown(body)
	require.NoError(t, err)
	assert.Equal(t, md, got)

	titled := &Doc{Type: "doc", Content: []Node{{Type: "image", Attrs: map[string]any{"src": "/x.png", "alt": "a*b", "title": "T"}}}}
	body, err = MarshalJSON(titled)
	require.NoError(t, err)
	got, err = JSONToMarkdown(body)
	require.NoError(t, err)
	assert.Equal(t, `![a\*b](/x.png "T")`, got)
}

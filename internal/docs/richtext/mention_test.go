package richtext

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMentionHref(t *testing.T) {
	tests := []struct {
		href     string
		kind, id string
		ok       bool
	}{
		{"/tickets/t-1", "ticket", "t-1", true},
		{"/docs/d-1", "doc", "d-1", true},
		{"/tickets/", "", "", false},
		{"/docs/", "", "", false},
		{"/ticket/t-1", "", "", false},
		{"https://example.com/tickets/t-1", "", "", false},
		{"#anchor", "", "", false},
	}
	for _, tt := range tests {
		kind, id, ok := ParseMentionHref(tt.href)
		assert.Equal(t, tt.kind, kind, "kind for %s", tt.href)
		assert.Equal(t, tt.id, id, "id for %s", tt.href)
		assert.Equal(t, tt.ok, ok, "ok for %s", tt.href)
	}
}

func TestMentionHref_RoundTrip(t *testing.T) {
	for _, kind := range []string{MentionTypeTicket, MentionTypeDoc} {
		href := MentionHref(kind, "abc-123")
		gotKind, gotID, ok := ParseMentionHref(href)
		require.True(t, ok, "round-trip %s", href)
		assert.Equal(t, kind, gotKind)
		assert.Equal(t, "abc-123", gotID)
	}
}

func TestMarkdownToDoc_MentionLinks(t *testing.T) {
	tests := []struct {
		name string
		md   string
		kind string
		id   string
	}{
		{"ticket", "see [Fix the bug](/tickets/t-1) now", "ticket", "t-1"},
		{"doc", "per [architecture](/docs/d-9)", "doc", "d-9"},
		{"label with spaces", "[Onboarding guide](/docs/d-2)", "doc", "d-2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := MarkdownToDoc(tt.md)
			require.NoError(t, err)
			require.Len(t, doc.Content, 1)
			para := doc.Content[0]
			var mention *Node
			for i := range para.Content {
				if para.Content[i].Type == "mention" {
					mention = &para.Content[i]
				}
			}
			require.NotNil(t, mention, "mention node present")
			assert.Equal(t, tt.kind, mention.Attrs["type"])
			assert.Equal(t, tt.id, mention.Attrs["id"])
			assert.Contains(t, tt.md, mention.Attrs["label"].(string))
		})
	}
}

func TestMarkdownToDoc_NonMentionLinksStayLinks(t *testing.T) {
	doc, err := MarkdownToDoc("see [site](https://example.com) please")
	require.NoError(t, err)
	para := doc.Content[0]
	for _, n := range para.Content {
		assert.NotEqual(t, "mention", n.Type)
	}
	require.Len(t, para.Content, 3)
	assert.Equal(t, "link", para.Content[1].Marks[0].Type)
	assert.Equal(t, "https://example.com", para.Content[1].Marks[0].Attrs["href"])
}

func TestJSONToMarkdown_MentionNodes(t *testing.T) {
	body := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"see "},{"type":"mention","attrs":{"type":"ticket","id":"t-1","label":"Fix the bug"}},{"type":"text","text":" now"}]}]}`
	md, err := JSONToMarkdown(body)
	require.NoError(t, err)
	assert.Equal(t, "see [Fix the bug](/tickets/t-1) now", md)
}

func TestMentionMarkdown_RoundTrip(t *testing.T) {
	for _, md := range []string{
		"see [Fix the bug](/tickets/t-1) now",
		"[Design doc](/docs/d-9) covers it",
	} {
		doc, err := MarkdownToDoc(md)
		require.NoError(t, err)
		out, err := MarshalJSON(doc)
		require.NoError(t, err)
		got, err := JSONToMarkdown(out)
		require.NoError(t, err)
		assert.Equal(t, md, got)
	}
}

func TestMentionWithMarks_Serializes(t *testing.T) {
	body := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"mention","attrs":{"type":"doc","id":"d-1","label":"Architecture"}}]}]}`
	md, err := JSONToMarkdown(body)
	require.NoError(t, err)
	assert.Equal(t, "[Architecture](/docs/d-1)", md)
}

package imports

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs/richtext"
)

const (
	pageA = "11111111-1111-4111-8111-111111111111"
	pageB = "22222222-2222-4222-8222-222222222222"
	dbA   = "33333333-3333-4333-8333-333333333333"
)

func testWriter() *notionWriter {
	return &notionWriter{
		link: func(id string) (notionLink, bool) {
			if canonicalNotionID(id) == pageA {
				return notionLink{Href: "/docs/doc-a", Title: "Page A"}, true
			}
			return notionLink{}, false
		},
		rows: func(id string) []notionLink {
			if id == dbA {
				return []notionLink{{Href: "/docs/row-1", Title: "Row 1"}, {Href: "/docs/row-2", Title: "Row 2"}}
			}
			return nil
		},
	}
}

func blocksOf(t *testing.T, raw string) []notionBlock {
	t.Helper()
	var bs []notionBlock
	require.NoError(t, json.Unmarshal([]byte(raw), &bs))
	return bs
}

const rt = `"rich_text":[{"type":"text","plain_text":"Hello","annotations":{}}]`

func TestNotionWriter_ConvertsEverySupportedBlock(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		// node is the type the markdown becomes in a doc body.
		node string
	}{
		{"paragraph", `[{"type":"paragraph","paragraph":{` + rt + `}}]`, "Hello", "paragraph"},
		{"heading 1", `[{"type":"heading_1","heading_1":{` + rt + `}}]`, "# Hello", "heading"},
		{"heading 2", `[{"type":"heading_2","heading_2":{` + rt + `}}]`, "## Hello", "heading"},
		{"heading 3", `[{"type":"heading_3","heading_3":{` + rt + `}}]`, "### Hello", "heading"},
		{"bulleted list", `[{"type":"bulleted_list_item","bulleted_list_item":{` + rt + `}},{"type":"bulleted_list_item","bulleted_list_item":{` + rt + `}}]`, "- Hello\n- Hello", "bulletList"},
		{"numbered list", `[{"type":"numbered_list_item","numbered_list_item":{` + rt + `}}]`, "1. Hello", "orderedList"},
		{"to-do", `[{"type":"to_do","to_do":{` + rt + `,"checked":true}},{"type":"to_do","to_do":{` + rt + `,"checked":false}}]`, "- [x] Hello\n- [ ] Hello", "taskList"},
		{"toggle", `[{"type":"toggle","toggle":{` + rt + `}}]`, "Hello", "paragraph"},
		{"quote", `[{"type":"quote","quote":{` + rt + `}}]`, "> Hello", "blockquote"},
		{"callout", `[{"type":"callout","callout":{` + rt + `,"icon":{"type":"emoji","emoji":"💡"}}}]`, "> 💡 Hello", "blockquote"},
		{"code", `[{"type":"code","code":{"rich_text":[{"type":"text","plain_text":"x := 1","annotations":{}}],"language":"go"}}]`, "```go\nx := 1\n```", "codeBlock"},
		{"divider", `[{"type":"divider","divider":{}}]`, "---", "horizontalRule"},
		{"external image", `[{"type":"image","image":{"type":"external","external":{"url":"https://cdn.example.com/a.png"},"caption":[]}}]`, "![](https://cdn.example.com/a.png)", "image"},
		{"file", `[{"type":"file","file":{"type":"external","external":{"url":"https://cdn.example.com/spec.pdf"},"name":"Spec"}}]`, "[Spec](https://cdn.example.com/spec.pdf)", "paragraph"},
		{"pdf", `[{"type":"pdf","pdf":{"type":"external","external":{"url":"https://cdn.example.com/spec.pdf"}}}]`, "[spec.pdf](https://cdn.example.com/spec.pdf)", "paragraph"},
		{"bookmark", `[{"type":"bookmark","bookmark":{"url":"https://example.com","caption":[]}}]`, "[https://example.com](https://example.com)", "paragraph"},
		{"table", `[{"type":"table","has_children":true,"table":{"has_column_header":true}}]`, "| Name | Role |\n| --- | --- |\n| alice | Owner |", "table"},
		{"child page", `[{"id":"` + pageA + `","type":"child_page","child_page":{"title":"Page A"}}]`, "[Page A](/docs/doc-a)", "paragraph"},
		{"child page outside the import", `[{"id":"` + pageB + `","type":"child_page","child_page":{"title":"Elsewhere"}}]`, "Elsewhere", "paragraph"},
		{"link to page", `[{"type":"link_to_page","link_to_page":{"type":"page_id","page_id":"` + pageA + `"}}]`, "[Page A](/docs/doc-a)", "paragraph"},
		{"database", `[{"id":"` + dbA + `","type":"child_database","child_database":{"title":"Policies"}}]`, "**Policies**\n\n- [Row 1](/docs/row-1)\n- [Row 2](/docs/row-2)", "paragraph"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := blocksOf(t, tt.raw)
			if tt.name == "table" {
				bs[0].Children = blocksOf(t, `[{"type":"table_row","table_row":{"cells":[[{"type":"text","plain_text":"Name","annotations":{}}],[{"type":"text","plain_text":"Role","annotations":{}}]]}},
					{"type":"table_row","table_row":{"cells":[[{"type":"text","plain_text":"alice","annotations":{}}],[{"type":"text","plain_text":"Owner","annotations":{}}]]}}]`)
			}
			w := testWriter()
			got := w.blocks(bs)
			assert.Equal(t, tt.want, got)
			assert.Zero(t, w.unsupported)
			doc, err := richtext.MarkdownToDoc(got)
			require.NoError(t, err)
			require.NotEmpty(t, doc.Content)
			assert.Equal(t, tt.node, doc.Content[0].Type)
		})
	}
}

func TestNotionWriter_NestsListChildrenAndFlattensOthers(t *testing.T) {
	item := blocksOf(t, `[{"type":"bulleted_list_item","has_children":true,"bulleted_list_item":{`+rt+`}}]`)
	item[0].Children = blocksOf(t, `[{"type":"bulleted_list_item","bulleted_list_item":{"rich_text":[{"type":"text","plain_text":"Inner","annotations":{}}]}}]`)
	assert.Equal(t, "- Hello\n  - Inner", testWriter().blocks(item))

	toggle := blocksOf(t, `[{"type":"toggle","has_children":true,"toggle":{`+rt+`}}]`)
	toggle[0].Children = blocksOf(t, `[{"type":"paragraph","paragraph":{"rich_text":[{"type":"text","plain_text":"Inside","annotations":{}}]}}]`)
	assert.Equal(t, "Hello\n\nInside", testWriter().blocks(toggle))
}

func TestNotionWriter_KeepsLineBreaksInsideTheirBlock(t *testing.T) {
	two := `"rich_text":[{"type":"text","plain_text":"one\ntwo","annotations":{}}]`
	tests := []struct {
		name, raw, want, node string
	}{
		{"a list item", `[{"type":"bulleted_list_item","bulleted_list_item":{` + two + `}}]`, "- one\\\n  two", "bulletList"},
		{"a heading", `[{"type":"heading_2","heading_2":{` + two + `}}]`, "## one two", "heading"},
		{"a paragraph that reads like a list", `[{"type":"paragraph","paragraph":{"rich_text":[{"type":"text","plain_text":"1. Not a list","annotations":{}}]}}]`, "1\\. Not a list", "paragraph"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testWriter().blocks(blocksOf(t, tt.raw))
			assert.Equal(t, tt.want, got)
			doc, err := richtext.MarkdownToDoc(got)
			require.NoError(t, err)
			require.Len(t, doc.Content, 1)
			assert.Equal(t, tt.node, doc.Content[0].Type)
		})
	}
}

func TestNotionWriter_TurnsUnsupportedBlocksIntoTextOrLinksAndCountsThem(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"embed", `[{"type":"embed","embed":{"url":"https://maps.example.com/x","caption":[]}}]`, "[https://maps.example.com/x](https://maps.example.com/x)"},
		{"equation", `[{"type":"equation","equation":{"expression":"e = mc^2"}}]`, "```\ne = mc^2\n```"},
		{"database view", `[{"id":"` + pageB + `","type":"child_database","child_database":{"title":"Linked view"}}]`, ""},
		{"table of contents", `[{"type":"table_of_contents","table_of_contents":{}}]`, ""},
		{"a type Notion adds later", `[{"type":"meeting_notes","meeting_notes":{}}]`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := testWriter()
			assert.Equal(t, tt.want, w.blocks(blocksOf(t, tt.raw)))
			assert.Equal(t, 1, w.unsupported)
		})
	}

	columns := blocksOf(t, `[{"type":"column_list","has_children":true,"column_list":{}}]`)
	columns[0].Children = blocksOf(t, `[{"type":"column","has_children":true,"column":{}},{"type":"column","has_children":true,"column":{}}]`)
	columns[0].Children[0].Children = blocksOf(t, `[{"type":"paragraph","paragraph":{"rich_text":[{"type":"text","plain_text":"Left","annotations":{}}]}}]`)
	columns[0].Children[1].Children = blocksOf(t, `[{"type":"paragraph","paragraph":{"rich_text":[{"type":"text","plain_text":"Right","annotations":{}}]}}]`)
	w := testWriter()
	assert.Equal(t, "Left\n\nRight", w.blocks(columns))
	assert.Equal(t, 1, w.unsupported, "columns count once")

	synced := blocksOf(t, `[{"type":"synced_block","has_children":true,"synced_block":{"synced_from":null}}]`)
	synced[0].Children = blocksOf(t, `[{"type":"paragraph","paragraph":{"rich_text":[{"type":"text","plain_text":"Shared","annotations":{}}]}}]`)
	w = testWriter()
	assert.Equal(t, "Shared", w.blocks(synced))
	assert.Equal(t, 1, w.unsupported)
}

func TestNotionWriter_RendersRichText(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"marks", `[{"type":"text","plain_text":"bold ","annotations":{"bold":true}},{"type":"text","plain_text":"it","annotations":{"italic":true}},{"type":"text","plain_text":" gone","annotations":{"strikethrough":true}},{"type":"text","plain_text":" x*y","annotations":{"code":true}}]`, "**bold** *it* ~~gone~~ `x*y`"},
		{"markdown in plain text stays text", `[{"type":"text","plain_text":"a *b* [c](d) | e","annotations":{}}]`, `a \*b\* \[c\](d) \| e`},
		{"an outside link", `[{"type":"text","plain_text":"site","href":"https://example.com","annotations":{}}]`, "[site](https://example.com)"},
		{"a relative link to an imported page", `[{"type":"text","plain_text":"see","href":"/11111111111141118111111111111111","annotations":{}}]`, "[see](/docs/doc-a)"},
		{"a notion.so link to an imported page", `[{"type":"text","plain_text":"see","href":"https://www.notion.so/acme/Page-A-11111111111141118111111111111111?pvs=4","annotations":{}}]`, "[see](/docs/doc-a)"},
		{"a mention of a page outside the import", `[{"type":"mention","plain_text":"Other","mention":{"type":"page","page":{"id":"` + pageB + `"}},"annotations":{}}]`, "[Other](https://www.notion.so/22222222222242228222222222222222)"},
		{"a person mention stays their name", `[{"type":"mention","plain_text":"@alice","mention":{"type":"user"},"annotations":{}}]`, "@alice"},
		{"a line break", `[{"type":"text","plain_text":"one\ntwo","annotations":{}}]`, "one\\\ntwo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rts []notionRichText
			require.NoError(t, json.Unmarshal([]byte(tt.raw), &rts))
			assert.Equal(t, tt.want, testWriter().text(rts))
		})
	}
}

func TestNotionWriter_CollectsNotionHostedFilesAndKeepsExternalOnesLinks(t *testing.T) {
	w := testWriter()
	got := w.blocks(blocksOf(t, `[
		{"type":"image","image":{"type":"file","file":{"url":"https://prod-files-secure.s3.us-west-2.amazonaws.com/w/a.png?X-Amz-Signature=1"},"caption":[{"type":"text","plain_text":"Logo","annotations":{}}]}},
		{"type":"image","image":{"type":"external","external":{"url":"https://cdn.example.com/b.png"},"caption":[]}}]`))
	assert.Equal(t, "![Logo](https://prod-files-secure.s3.us-west-2.amazonaws.com/w/a.png?X-Amz-Signature=1)\n\n![](https://cdn.example.com/b.png)", got)
	assert.Equal(t, []string{"https://prod-files-secure.s3.us-west-2.amazonaws.com/w/a.png?X-Amz-Signature=1"}, w.files)
}

func TestNotionWriter_RowTableShowsEveryPropertyButTheTitle(t *testing.T) {
	var props map[string]notionProperty
	require.NoError(t, json.Unmarshal([]byte(`{
		"Name": {"type":"title","title":[{"type":"text","plain_text":"Policy","annotations":{}}]},
		"Owner": {"type":"people","people":[{"name":"alice"},{"name":"bob"}]},
		"Score": {"type":"number","number":4.5},
		"Done": {"type":"checkbox","checkbox":true},
		"Tags": {"type":"multi_select","multi_select":[{"name":"hr"},{"name":"legal"}]},
		"Due": {"type":"date","date":{"start":"2026-05-01","end":null}},
		"Link": {"type":"url","url":"https://example.com"},
		"Related": {"type":"relation","relation":[{"id":"`+pageA+`"},{"id":"`+pageB+`"}]},
		"Ref": {"type":"unique_id","unique_id":{"prefix":"POL","number":7}},
		"Note": {"type":"rich_text","rich_text":[{"type":"text","plain_text":"a | b","annotations":{}}]}
	}`), &props))
	assert.Equal(t, "| Property | Value |\n| --- | --- |\n"+
		"| Done | Yes |\n| Due | 2026-05-01 |\n| Link | [https://example.com](https://example.com) |\n| Note | a \\| b |\n"+
		"| Owner | alice, bob |\n| Ref | POL-7 |\n| Related | [Page A](/docs/doc-a) |\n| Score | 4.5 |\n| Tags | hr, legal |",
		testWriter().rowTable(props))
	assert.Equal(t, "Policy", notionTitle(props))
}

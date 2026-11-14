package richtext

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A two-column table with a 180px first column and a 64px second row, as the editor saves it.
const sizedTable = `{"type":"doc","content":[{"type":"table","content":[` +
	`{"type":"tableRow","content":[{"type":"tableHeader","attrs":{"colwidth":[180]},"content":[{"type":"paragraph","content":[{"type":"text","text":"Item"}]}]},{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"Qty"}]}]}]},` +
	`{"type":"tableRow","attrs":{"rowheight":64},"content":[{"type":"tableCell","attrs":{"colwidth":[180]},"content":[{"type":"paragraph","content":[{"type":"text","text":"Bolts"}]}]},{"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"4"}]}]}]}]}]}`

func TestKeepTableSizes(t *testing.T) {
	tests := []struct {
		name      string
		prev      string
		markdown  string
		wantWidth any
		wantRow   any
	}{
		{"an edited cell keeps its column width and row height", sizedTable, "| Item | Qty |\n| --- | --- |\n| Nuts | 4 |", []any{180.0}, 64.0},
		{"an appended row keeps the sizes above it", sizedTable, "| Item | Qty |\n| --- | --- |\n| Bolts | 4 |\n| Nuts | 9 |", []any{180.0}, 64.0},
		{"a table with a column added drops the widths it no longer matches", sizedTable, "| Item | Qty | Bin |\n| --- | --- | --- |\n| Bolts | 4 | A |", nil, 64.0},
		{"a legacy markdown body has no sizes to keep", "| Item | Qty |\n| --- | --- |\n| Bolts | 4 |", "| Item | Qty |\n| --- | --- |\n| Nuts | 4 |", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, err := MarkdownToDoc(tt.markdown)
			require.NoError(t, err)
			body, err := MarshalJSON(next)
			require.NoError(t, err)

			doc, err := UnmarshalJSON(KeepTableSizes(tt.prev, body))
			require.NoError(t, err)
			rows := doc.Content[0].Content
			assert.Equal(t, tt.wantWidth, rows[1].Content[0].Attrs["colwidth"])
			assert.Equal(t, tt.wantRow, rows[1].Attrs["rowheight"])
		})
	}
}

package richtext

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompatProbe pins the Go converter to the exact JSON shape the Tiptap v3 @tiptap/markdown extension produces (verified against editor.markdown.parse).
func TestCompatProbe(t *testing.T) {
	tests := []struct{ md, want string }{
		{"# Hello\n\nThis is **bold** and *italic* with `code`.",
			`{"type":"doc","content":[{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Hello"}]},{"type":"paragraph","content":[{"type":"text","text":"This is "},{"type":"text","marks":[{"type":"bold"}],"text":"bold"},{"type":"text","text":" and "},{"type":"text","marks":[{"type":"italic"}],"text":"italic"},{"type":"text","text":" with "},{"type":"text","marks":[{"type":"code"}],"text":"code"},{"type":"text","text":"."}]}]}`},
		{"## Heading\n\n- one\n- two\n\n> quote",
			`{"type":"doc","content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Heading"}]},{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"one"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"two"}]}]}]},{"type":"blockquote","content":[{"type":"paragraph","content":[{"type":"text","text":"quote"}]}]}]}`},
	}
	for _, tt := range tests {
		got, err := markdownToJSON(tt.md)
		require.NoError(t, err)
		var gotV, wantV any
		require.NoError(t, json.Unmarshal([]byte(got), &gotV))
		require.NoError(t, json.Unmarshal([]byte(tt.want), &wantV))
		assert.Equal(t, wantV, gotV)
	}
}

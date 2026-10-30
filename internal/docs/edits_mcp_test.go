package docs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestDocUpdate_Edits_SaveTheEditedBodyAsOneVersion(t *testing.T) {
	s := newTestService(newFakeRepo())
	d := mustDoc(t, s, "project-1", "Spec", "The plan has three steps.\n\nShip it on Friday.")

	out, err := callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","edits":[{"find":"three steps","replace":"four steps"},{"find":"Friday","replace":"Monday"}]}`)

	require.NoError(t, err)
	got := out.(docResult)
	assert.Equal(t, "The plan has four steps.\n\nShip it on Monday.", got.Body)
	assert.Equal(t, 2, got.Version)
	h, err := s.History(testCtx(), d.ID)
	require.NoError(t, err)
	assert.Len(t, h.Versions, 2, "one version for the whole call")
	assert.Equal(t, viaMCP, h.Versions[0].Via)
}

func TestDocUpdate_Edits_WhenRefused_SaveNothing(t *testing.T) {
	tests := []struct{ name, args, wantMsg string }{
		{"a find that matches nothing, after an edit that fits", `"edits":[{"find":"Ship","replace":"Send"},{"find":"Tuesday","replace":"Monday"}]`, "edit 2: find matches nothing"},
		{"a find that matches twice", `"edits":[{"find":"the","replace":"a"}]`, "edit 1: find matches 2 places"},
		{"an empty find", `"edits":[{"find":"","replace":"x"}]`, "edit 1: find is empty"},
		{"the body beside them", `"body":"new","edits":[{"find":"Ship","replace":"Send"}]`, "not both"},
		{"a revert beside them", `"revert_to_version":1,"edits":[{"find":"Ship","replace":"Send"}]`, "send it alone"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService(newFakeRepo())
			d := mustDoc(t, s, "project-1", "Spec", "Ship the plan on the Friday.")

			_, err := callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`",`+tt.args+`}`)

			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.ErrorContains(t, err, tt.wantMsg)
			got, err := s.Get(testCtx(), d.ID)
			require.NoError(t, err)
			assert.Equal(t, 1, got.Version)
			md, err := richtext.ToMarkdown(got.Body)
			require.NoError(t, err)
			assert.Equal(t, "Ship the plan on the Friday.", md)
		})
	}
}

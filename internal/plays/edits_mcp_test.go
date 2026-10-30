package plays

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestPlayUpdate_Edits_SaveTheEditedInstructionsAsOneVersion(t *testing.T) {
	_, tools, p := playToolsFixture(t)

	out, err := callTool(t, tools, ctxAs("owner"), "play_update", `{"workspace_id":"$WS","id":"`+p.ID+`","enabled":false,"edits":[{"find":"Fix the","replace":"Repair the"},{"find":"Repair the ticket","replace":"Repair the bug"}]}`)

	require.NoError(t, err)
	assert.Equal(t, "Repair the bug.", out.(playResult).Instructions)
	out, err = callTool(t, tools, ctxAs("owner"), "play_list", `{"workspace_id":"$WS","id":"`+p.ID+`"}`)
	require.NoError(t, err)
	got := out.(playResult)
	assert.Equal(t, []any{"Repair the bug.", false, 2}, []any{got.Instructions, got.Enabled, len(got.Versions)}, "one version, beside the other field")
	assert.Equal(t, "mcp", got.Versions[0].Via)
}

func TestPlayUpdate_Edits_WhenRefused_SaveNothing(t *testing.T) {
	_, tools, p := playToolsFixture(t)
	tests := []struct{ name, args, wantMsg string }{
		{"a find that matches nothing, after an edit that fits", `"edits":[{"find":"Fix","replace":"Mend"},{"find":"bug","replace":"x"}]`, "edit 2: find matches nothing"},
		{"a find that matches twice", `"edits":[{"find":"i","replace":"o"}]`, "edit 1: find matches 2 places"},
		{"an empty find", `"edits":[{"find":"","replace":"x"}]`, "edit 1: find is empty"},
		{"the instructions beside them", `"instructions":"new","edits":[{"find":"Fix","replace":"Mend"}]`, "not both"},
		{"a revert beside them", `"revert_to_version":1,"edits":[{"find":"Fix","replace":"Mend"}]`, "send it alone"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(t, tools, ctxAs("owner"), "play_update", `{"workspace_id":"$WS","id":"`+p.ID+`","label":"Renamed",`+tt.args+`}`)

			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.ErrorContains(t, err, tt.wantMsg)
			out, err := callTool(t, tools, ctxAs("owner"), "play_list", `{"workspace_id":"$WS","id":"`+p.ID+`"}`)
			require.NoError(t, err)
			got := out.(playResult)
			assert.Equal(t, []any{"Fix the ticket.", "Fix with AI", 1}, []any{got.Instructions, got.Label, len(got.Versions)}, "a refused call changes no field")
		})
	}
}

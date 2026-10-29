package plays

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// editInstructions saves new instructions through play_update as the owner, so the fixture's play reaches version 2.
func editInstructions(t *testing.T, p *Play, tools []mcptool.Tool, text string) {
	t.Helper()
	_, err := callTool(t, tools, ctxAs("owner"), "play_update", `{"workspace_id":"$WS","id":"`+p.ID+`","instructions":"`+text+`"}`)
	require.NoError(t, err)
}

func TestPlayList_WithAnID_ReadsAnOlderVersionBesideTheRecentHistory(t *testing.T) {
	_, tools, p := playToolsFixture(t)
	editInstructions(t, p, tools, "Second.")

	out, err := callTool(t, tools, ctxAs("reader"), "play_list", `{"workspace_id":"$WS","id":"`+p.ID+`","version":1}`)
	require.NoError(t, err)
	got := out.(playResult)
	assert.Equal(t, []any{"Fix the ticket.", 1, 2}, []any{got.Instructions, got.Version, got.CurrentVersion})
	require.Len(t, got.Versions, 2)
	assert.Equal(t, []any{2, "mcp"}, []any{got.Versions[0].Version, got.Versions[0].Via}, "an MCP save is recorded as one")
	assert.Equal(t, []any{1, ""}, []any{got.Versions[1].Version, got.Versions[1].Via})

	out, err = callTool(t, tools, ctxAs("reader"), "play_list", `{"workspace_id":"$WS","id":"`+p.ID+`"}`)
	require.NoError(t, err)
	assert.Equal(t, []any{"Second.", 0}, []any{out.(playResult).Instructions, out.(playResult).Version}, "without version it is the current text")
}

func TestPlayList_ErrorPaths(t *testing.T) {
	_, tools, p := playToolsFixture(t)
	tests := []struct {
		name string
		args string
		want error
	}{
		{"version without an id is invalid", `{"workspace_id":"$WS","version":1}`, apperrs.ErrInvalid},
		{"an id beside a type filter is invalid", `{"workspace_id":"$WS","id":"` + p.ID + `","type":"ticket"}`, apperrs.ErrInvalid},
		{"a version never written is not found", `{"workspace_id":"$WS","id":"` + p.ID + `","version":7}`, apperrs.ErrNotFound},
		{"an id from another workspace is not found", `{"workspace_id":"$WS-other","id":"` + p.ID + `"}`, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(t, tools, ctxAs("reader"), "play_list", tt.args)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestPlayUpdate_RevertToVersion_RestoresItAsANewVersion(t *testing.T) {
	_, tools, p := playToolsFixture(t)
	editInstructions(t, p, tools, "Second.")

	out, err := callTool(t, tools, ctxAs("owner"), "play_update", `{"workspace_id":"$WS","id":"`+p.ID+`","revert_to_version":1}`)
	require.NoError(t, err)
	assert.Equal(t, "Fix the ticket.", out.(playResult).Instructions)

	out, err = callTool(t, tools, ctxAs("owner"), "play_list", `{"workspace_id":"$WS","id":"`+p.ID+`"}`)
	require.NoError(t, err)
	got := out.(playResult)
	require.Len(t, got.Versions, 3)
	assert.Equal(t, []any{3, "mcp"}, []any{got.Versions[0].Version, got.Versions[0].Via})
}

func TestPlayUpdate_RevertToVersion_ErrorPaths(t *testing.T) {
	_, tools, p := playToolsFixture(t)
	editInstructions(t, p, tools, "Second.")
	tests := []struct {
		name  string
		actor string
		args  string
		want  error
	}{
		{"sent with another field is invalid", "owner", `{"workspace_id":"$WS","id":"` + p.ID + `","revert_to_version":1,"label":"Renamed"}`, apperrs.ErrInvalid},
		{"sent with an auto play change is invalid", "owner", `{"workspace_id":"$WS","id":"` + p.ID + `","revert_to_version":1,"remove_auto_plays":["x"]}`, apperrs.ErrInvalid},
		{"a reader is refused", "reader", `{"workspace_id":"$WS","id":"` + p.ID + `","revert_to_version":1}`, apperrs.ErrForbidden},
		{"a version never written is not found", "owner", `{"workspace_id":"$WS","id":"` + p.ID + `","revert_to_version":9}`, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(t, tools, ctxAs(tt.actor), "play_update", tt.args)
			require.ErrorIs(t, err, tt.want)
		})
	}

	out, err := callTool(t, tools, ctxAs("owner"), "play_list", `{"workspace_id":"$WS","id":"`+p.ID+`"}`)
	require.NoError(t, err)
	got := out.(playResult)
	assert.Equal(t, []any{"Second.", "Fix with AI", 2}, []any{got.Instructions, got.Label, len(got.Versions)}, "every refused revert left the play as it was")
}

package memories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestMemoryUpdate_Edits_SaveTheEditedBodyAsOneVersion(t *testing.T) {
	s := newTestService(newFakeRepo())
	m := mustMemory(t, s, "project-1", "Deploy quirks", "when deploying", "Run the migration before the deploy.\n\nRestart the worker after.", false)

	got := mustCall(t, s, "memory_update", `{"id":"`+m.ID+`","edits":[{"find":"before the deploy","replace":"after the deploy"},{"find":"Restart","replace":"Reload"}]}`).(memoryResult)

	assert.Equal(t, "Run the migration after the deploy.\n\nReload the worker after.", got.Body)
	assert.Equal(t, 2, got.Version)
	v, err := s.GetVersion(testCtx(), m.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, viaMCP, v.AuthorVia)
	_, err = s.GetVersion(testCtx(), m.ID, 3)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "one version for the whole call")
}

func TestMemoryUpdate_Edits_WhenRefused_SaveNothing(t *testing.T) {
	tests := []struct{ name, args, wantMsg string }{
		{"a find that matches nothing, after an edit that fits", `"edits":[{"find":"Run","replace":"Start"},{"find":"Tuesday","replace":"Monday"}]`, "edit 2: find matches nothing"},
		{"a find that matches twice", `"edits":[{"find":"the","replace":"a"}]`, "edit 1: find matches 2 places"},
		{"an empty find", `"edits":[{"find":"","replace":"x"}]`, "edit 1: find is empty"},
		{"the body beside them", `"body":"new","edits":[{"find":"Run","replace":"Start"}]`, "not both"},
		{"a revert beside them", `"revert_to_version":1,"edits":[{"find":"Run","replace":"Start"}]`, "send it alone"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService(newFakeRepo())
			m := mustMemory(t, s, "project-1", "Deploy quirks", "when deploying", "Run the plan on the day.", false)

			_, err := callTool(testCtx(), t, s, "memory_update", `{"id":"`+m.ID+`",`+tt.args+`}`)

			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.ErrorContains(t, err, tt.wantMsg)
			got := mustCall(t, s, "memory_get", `{"id":"`+m.ID+`"}`).(memoryResult)
			assert.Equal(t, 1, got.Version)
			assert.Equal(t, "Run the plan on the day.", got.Body)
		})
	}
}

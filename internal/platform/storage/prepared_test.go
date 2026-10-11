package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreparedReads_PrepareEachTextOnceAndKeepNoFailure(t *testing.T) {
	p := newPreparedReads(newTestStore(t).db)

	_, err := p.QueryContext(t.Context(), "SELECT nothing FROM nowhere")
	require.Error(t, err)
	assert.Empty(t, p.stmts, "a statement that fails to prepare is not kept")

	for range 2 {
		rows, err := p.QueryContext(t.Context(), "SELECT id FROM workspaces WHERE id = ?", "workspace-default")
		require.NoError(t, err)
		require.True(t, rows.Next())
		require.NoError(t, rows.Close())
	}
	assert.Len(t, p.stmts, 1, "the same text reuses its statement")
}

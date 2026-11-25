package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/docs/richtext"
)

// TestMigration0170_StoredDocsGetTheSnippetTheirListShowed upgrades docs of every snippet shape and checks each
// stored snippet is the one the list built from the body before it was stored.
func TestMigration0170_StoredDocsGetTheSnippetTheirListShowed(t *testing.T) {
	db := migrateBefore(t, "0170")
	want := map[string]string{}
	for shape, body := range snippetBodies {
		_, err := db.Exec(`INSERT INTO docs (id, title, body, version, created_at, updated_at) VALUES (?, 'Doc', ?, 1, 1, 1)`, shape, body)
		require.NoError(t, err)
		want[shape] = richtext.Snippet(body, docs.SnippetRunes)
	}

	require.NoError(t, Migrate(db), "0170 and every later migration apply on top, as an upgrade would")

	rows, err := db.Query(`SELECT id, snippet FROM docs`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	got := map[string]string{}
	for rows.Next() {
		var id, snippet string
		require.NoError(t, rows.Scan(&id, &snippet))
		got[id] = snippet
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, want, got)
}

package storage

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0144_KeptMemoryVersionsAreCreditedPrunedToTenAndSized upgrades over a memory with 12 saves and one with 2:
// the long one keeps its newest 10 with the oldest unsized, and every stored author becomes the sole editor.
func TestMigration0144_KeptMemoryVersionsAreCreditedPrunedToTenAndSized(t *testing.T) {
	db := migrateBefore(t, "0144")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-seed', 'Seed', 'seed', 1, 1);
INSERT INTO projects (id, workspace_id, name, position, created_at, updated_at) VALUES ('p-1', 'ws-seed', 'Backend', 0, 1, 1);
INSERT INTO memories (id, workspace_id, project_id, title, body, version, created_at, updated_at) VALUES
    ('m-long', 'ws-seed', 'p-1', 'Long', 'one two', 12, 1, 1), ('m-short', 'ws-seed', 'p-1', 'Short', 'solo', 2, 1, 1);
WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 12)
INSERT INTO memory_versions (id, memory_id, version, title, body, author_id, created_at)
    SELECT 'v-long-' || i, 'm-long', i, 'Long', 'w' || i || ' x', CASE WHEN i = 4 THEN '' ELSE 'alice' END, i FROM n;
INSERT INTO memory_versions (id, memory_id, version, title, body, author_id, created_at) VALUES
    ('v-short-1', 'm-short', 1, 'Short', 'solo', 'bob', 1),
    ('v-short-2', 'm-short', 2, 'Short', 'solo two', 'bob', 2);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0144 and every later migration apply on top, as an upgrade would")

	rows := func(memoryID string) []string {
		r, err := db.Query(`SELECT version, editor_ids, words_added, words_removed FROM memory_versions WHERE memory_id = ? ORDER BY version`, memoryID)
		require.NoError(t, err)
		defer func() { require.NoError(t, r.Close()) }()
		var out []string
		for r.Next() {
			var version int
			var editors string
			var added, removed sql.NullInt64
			require.NoError(t, r.Scan(&version, &editors, &added, &removed))
			out = append(out, fmt.Sprint(version, " ", editors, " ", sizeLabel(nullableInt(added), nullableInt(removed))))
		}
		require.NoError(t, r.Err())
		return out
	}
	long := rows("m-long")
	require.Len(t, long, 10)
	assert.Equal(t, `3 ["alice"] none`, long[0], "the new oldest version has no predecessor left to be sized against")
	assert.Equal(t, `4 [] +1 -1`, long[1], "a version with no stored author is credited to nobody")
	assert.Equal(t, `12 ["alice"] +1 -1`, long[9])
	assert.Equal(t, []string{`1 ["bob"] none`, `2 ["bob"] +1 -0`}, rows("m-short"))
}

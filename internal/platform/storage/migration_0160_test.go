package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0160_StoredRunsGetTheReadsTheirStepsShow upgrades a run whose steps read two memories of its project
// under two harnesses' names, one of them twice, beside steps that read nothing (another tool, a failed call, an open
// call, a memory of another project, a memory deleted since), and a run stored as plain lines.
func TestMigration0160_StoredRunsGetTheReadsTheirStepsShow(t *testing.T) {
	db := migrateBefore(t, "0160")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-1', 'One', 'one', 1, 1);
INSERT INTO projects (id, workspace_id, name, position, created_at, updated_at) VALUES ('p-1', 'ws-1', 'One', 0, 1, 1), ('p-2', 'ws-1', 'Two', 1, 1, 1);
INSERT INTO memories (id, workspace_id, project_id, kind, title, body, version, created_at, updated_at) VALUES
    ('m-build', 'ws-1', 'p-1', '', 'Building a ticket', 'Build.', 1, 1, 1),
    ('m-review', 'ws-1', 'p-1', '', 'Reviewing a change', 'Review.', 1, 1, 1),
    ('m-far', 'ws-1', 'p-2', '', 'Elsewhere', 'Far.', 1, 1, 1);
INSERT INTO play_trails (id, workspace_id, play_id, play_label, target_type, target_id, project_id, starter_id, via, state, started_at, activity) VALUES
    ('tr-1', 'ws-1', 'play-1', 'Fix with AI', 'ticket', 't-1', 'p-1', 'u-1', 'web', 'done', 1, ?1),
    ('tr-lines', 'ws-1', 'play-1', 'Fix with AI', 'ticket', 't-2', 'p-1', 'u-1', 'web', 'done', 1, '["Read memory_get", "done"]');`,
		`[
  {"kind":"tool_result","tool":"mcp__nexul__memory_list","summary":"{}","detail":"{\"input\":{\"project_id\":\"p-1\"}}","at":"2026-11-10T09:00:00Z"},
  {"kind":"tool_result","tool":"mcp__nexul__memory_get","summary":"{\"id\":\"m-review\"}","detail":"{\"input\":{\"id\":\"m-review\"},\"result\":{\"content\":\"Rev","at":"2026-11-10T09:01:00Z"},
  {"kind":"tool_result","tool":"nexul · memory_get","summary":"{\"id\":\"m-build\"}","detail":"{\"input\":{\"id\":\"m-build\"}}","at":"2026-11-10T09:02:00Z"},
  {"kind":"tool_result","tool":"mcp__nexul__memory_get","summary":"{\"id\":\"m-review\"}","detail":"{\"input\":{\"id\":\"m-review\"}}","at":"2026-11-10T09:03:00Z"},
  {"kind":"tool_result","tool":"mcp__nexul__memory_get","summary":"{\"id\":\"m-far\"}","detail":"{\"input\":{\"id\":\"m-far\"}}","at":"2026-11-10T09:04:00Z"},
  {"kind":"tool_result","tool":"mcp__nexul__memory_get","summary":"{\"id\":\"m-gone\"}","detail":"{\"input\":{\"id\":\"m-gone\"}}","at":"2026-11-10T09:05:00Z"},
  {"kind":"tool_result","tool":"memory_get","summary":"{\"id\":\"m-build\"} · failed","detail":"{\"input\":{\"id\":\"m-build\"}}","at":"2026-11-10T09:06:00Z"},
  {"kind":"tool_call","tool":"memory_get","summary":"{}","detail":"{\"input\":{\"id\":\"m-far\"}}","at":"2026-11-10T09:07:00Z"},
  {"kind":"text","summary":"Done.","at":"2026-11-10T09:08:00Z"}
]`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0160 and every later migration apply on top, as an upgrade would")

	rows, err := db.Query(`SELECT trail_id || ' ' || memory_id || ' ' || title || ' ' || read_at FROM trail_memory_reads ORDER BY read_at, rowid`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	var got []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		got = append(got, s)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"tr-1 m-review Reviewing a change 1794301260", "tr-1 m-build Building a ticket 1794301320"}, got,
		"each memory once, at its first read, as the live path writes it")
}

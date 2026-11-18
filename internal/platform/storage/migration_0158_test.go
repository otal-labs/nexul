package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/memories"
)

// TestMigration0158_ProjectsWithoutMemoriesOfTheirOwnGetTheDefaults upgrades four projects: one holding only the
// untouched placeholder (with a version row, an attachment and an auto play naming it), one with only an interview
// memory and a decisions log, one whose placeholder was edited, and one with a memory of its own beside the placeholder.
func TestMigration0158_ProjectsWithoutMemoriesOfTheirOwnGetTheDefaults(t *testing.T) {
	db := migrateBefore(t, "0158")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-1', 'One', 'one', 1, 1);
INSERT INTO projects (id, workspace_id, name, position, created_at, updated_at) VALUES
    ('p-placeholder', 'ws-1', 'Placeholder', 0, 1, 1), ('p-special', 'ws-1', 'Special', 1, 1, 1),
    ('p-edited', 'ws-1', 'Edited', 2, 1, 1), ('p-own', 'ws-1', 'Own', 3, 1, 1);
INSERT INTO memories (id, workspace_id, project_id, kind, title, body, always_included, version, created_at, updated_at) VALUES
    ('m-placeholder', 'ws-1', 'p-placeholder', '', ?1, ?2, 1, 1, 1, 1),
    ('m-interview', 'ws-1', 'p-special', 'interview', 'Interview', 'Rules.', 0, 1, 1, 1),
    ('m-decisions', 'ws-1', 'p-special', 'decisions', 'Decisions log', '- One.', 0, 1, 1, 1),
    ('m-edited', 'ws-1', 'p-edited', '', ?1, 'Our rules.', 1, 2, 1, 1),
    ('m-own-placeholder', 'ws-1', 'p-own', '', ?1, ?2, 1, 1, 1, 1),
    ('m-own', 'ws-1', 'p-own', '', 'Deploys', 'Ship on Fridays.', 0, 1, 1, 1);
INSERT INTO memory_versions (id, memory_id, version, title, body, created_at) VALUES ('v-placeholder', 'm-placeholder', 1, ?1, ?2, 1);
INSERT INTO attachments (id, memory_id, name, content_type, size, data, created_at) VALUES ('a-placeholder', 'm-placeholder', 'x.png', 'image/png', 1, 'x', 1);
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at) VALUES
    ('play-1', 'ws-1', 'Mine', 'ticket', '', 'Mine.', 1, 'review', '[]', '', 'u-1', 1, 1);
INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, run_on, memory_ids, created_at, updated_at)
    VALUES ('ap-1', 'play-1', 'ws-1', 1, 'ticket.created', 'developer', '["m-placeholder","m-own"]', 1, 1);`,
		placeholderMemoryTitle, placeholderMemoryBody)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0158 and every later migration apply on top, as an upgrade would")

	titles := func(projectID string) []string {
		rows, err := db.Query(`SELECT m.title || ' v' || m.version || ' ' || m.always_included || m.footer || ' ' ||
			(SELECT count(*) FROM memory_versions v WHERE v.memory_id = m.id) FROM memories m WHERE m.project_id = ? ORDER BY m.created_at, m.id`, projectID)
		require.NoError(t, err)
		defer func() { require.NoError(t, rows.Close()) }()
		var out []string
		for rows.Next() {
			var s string
			require.NoError(t, rows.Scan(&s))
			out = append(out, s)
		}
		require.NoError(t, rows.Err())
		return out
	}
	defaults := []string{
		"Writing a doc v1 00 1", "Writing tickets v1 00 1", "Building a ticket v1 00 1",
		"Reviewing a change v1 00 1", "Finishing a ticket v1 11 1",
	}
	assert.Equal(t, defaults, titles("p-placeholder"), "the placeholder is replaced by the five defaults, each with its first version")
	assert.Equal(t, append([]string{"Decisions log v1 00 0", "Interview v1 00 0"}, defaults...), titles("p-special"),
		"the interview memory and the decisions log are not memories of the project's own")
	assert.Equal(t, []string{placeholderMemoryTitle + " v2 10 0"}, titles("p-edited"), "an edited placeholder is the team's own")
	assert.Equal(t, []string{"Deploys v1 00 0", placeholderMemoryTitle + " v1 10 0"}, titles("p-own"), "a project with its own memories keeps exactly what it has")

	var orphans int
	require.NoError(t, db.QueryRow(`SELECT (SELECT count(*) FROM memory_versions WHERE memory_id = 'm-placeholder') + (SELECT count(*) FROM attachments WHERE memory_id = 'm-placeholder')`).Scan(&orphans))
	assert.Zero(t, orphans, "the placeholder's versions and attachments go with it")
	var memoryIDs string
	require.NoError(t, db.QueryRow(`SELECT memory_ids FROM auto_plays WHERE id = 'ap-1'`).Scan(&memoryIDs))
	assert.JSONEq(t, `["m-own"]`, memoryIDs, "an auto play no longer names the deleted placeholder")

	var body string
	require.NoError(t, db.QueryRow(`SELECT body FROM memories WHERE project_id = 'p-placeholder' AND title = 'Writing tickets'`).Scan(&body))
	seed, err := memories.SeedOf(memories.Defaults()[1].Text)
	require.NoError(t, err)
	assert.Equal(t, seed.Body, body, "the body is stored as rich text, the way every memory body is")
}

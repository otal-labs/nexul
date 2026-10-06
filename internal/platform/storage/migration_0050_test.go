package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
)

func TestMigration0050_CreatorsAndVersionAuthorsWatchTheirDocs(t *testing.T) {
	db := migrateBefore(t, "0050")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-onik', 'onik', 0, 0), ('u-nor', 'nor', 0, 0);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0);
INSERT INTO docs (id, title, body, project_id, folder_id, version, archived, locked, created_by, created_at, updated_at) VALUES
    ('d-plan', 'Plan', '', 'p-web', '', 3, 0, 0, 'u-onik', 100, 300),
    ('d-seeded', 'Seeded', '', 'p-web', '', 1, 0, 0, '', 200, 200),
    ('d-system', 'System', '', 'p-web', '', 1, 0, 0, 'system', 200, 200);
INSERT INTO doc_versions (doc_id, version, title, body, name, author_id, created_at) VALUES
    ('d-plan', 1, 'Plan', '', '', '', 100),
    ('d-plan', 2, 'Plan', '', 'Draft', 'u-nor', 200),
    ('d-plan', 3, 'Plan', '', 'Final', 'u-nor', 300),
    ('d-plan', 4, 'Plan', '', 'Mine', 'u-onik', 400),
    ('d-seeded', 1, 'Seeded', '', 'Kickoff', 'u-nor', 250),
    ('d-system', 1, 'System', '', 'Import', 'nexul-system', 250);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0050 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	watchers := func(docID string) map[string]docs.WatcherSource {
		ws, err := s.Docs.ListWatchers(ctx, docID)
		require.NoError(t, err)
		out := map[string]docs.WatcherSource{}
		for _, w := range ws {
			out[w.UserID] = w.Source
		}
		return out
	}
	assert.Equal(t, map[string]docs.WatcherSource{"u-onik": docs.WatcherAuto, "u-nor": docs.WatcherAuto}, watchers("d-plan"),
		"the creator and each distinct version author, once")
	assert.Equal(t, map[string]docs.WatcherSource{"u-nor": docs.WatcherAuto}, watchers("d-seeded"), "a doc with no creator keeps its authors")
	assert.Empty(t, watchers("d-system"), "a creator or author who is not a user is skipped")

	require.NoError(t, s.Docs.SetWatching(ctx, "d-plan", "u-nor", false, time.Now()))
	assert.NotContains(t, watchers("d-plan"), "u-nor", "a backfilled watcher can stop watching like any other")
}

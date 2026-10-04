package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0065_RoomsOlderThanAServerWriteAreCleared upgrades over rooms from before a server write reset them:
// one written after its last edit replays stale headings, one edited after the write and one only named stay.
func TestMigration0065_RoomsOlderThanAServerWriteAreCleared(t *testing.T) {
	db := migrateBefore(t, "0065")
	_, err := db.Exec(`
INSERT INTO docs (id, title, body, version, created_at, updated_at) VALUES
    ('d-stale', 'Stale', '{}', 2, 1, 200), ('d-edited', 'Edited', '{}', 2, 1, 300), ('d-named', 'Named', '{}', 2, 1, 200);
INSERT INTO collab_updates (doc_id, kind, actor_id, payload, created_at) VALUES
    ('d-stale', 'snapshot', 'u-1', 'old', 100), ('d-stale', 'update', 'u-1', 'old-heading', 150),
    ('d-edited', 'snapshot', 'u-1', 'after', 300),
    ('d-named', 'snapshot', 'u-1', 'kept', 100);
INSERT INTO doc_versions (doc_id, version, title, body, created_at) VALUES ('d-stale', 2, 'Stale', '{}', 200), ('d-edited', 2, 'Edited', '{}', 200);
INSERT INTO doc_versions (doc_id, version, title, body, created_at, name) VALUES ('d-named', 2, 'Named', '{}', 200, 'Milestone');
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0065 applies on top, as an upgrade would")

	count := func(docID string) int {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM collab_updates WHERE doc_id = ?`, docID).Scan(&n))
		return n
	}
	assert.Zero(t, count("d-stale"), "a room older than the doc's last server write no longer replays over its body")
	assert.Equal(t, 1, count("d-edited"), "a room edited after the write is the current state")
	assert.Equal(t, 1, count("d-named"), "a named version leaves the body alone, so the room stays")
}

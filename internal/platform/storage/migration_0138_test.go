package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0138_PlayTextStopsNamingDeployGet rewrites deploy_get in stored play instructions and leaves other text alone.
func TestMigration0138_PlayTextStopsNamingDeployGet(t *testing.T) {
	db := migrateBefore(t, "0138")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws', 'Acme', 'acme', 1, 1);
INSERT INTO plays (id, workspace_id, label, type, instructions, created_by, created_at, updated_at) VALUES
    ('p-code', 'ws', 'Watch', 'stack', 'Poll ` + "`deploy_get`" + ` until it settles.', 'u', 1, 1),
    ('p-bare', 'ws', 'Retry', 'stack', 'Read the log with deploy_get.', 'u', 1, 1),
    ('p-other', 'ws', 'Fix', 'ticket', 'Read the ticket with ticket_get.', 'u', 1, 1);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0138 and every later migration apply on top, as an upgrade would")

	text := func(id string) (s string) {
		require.NoError(t, db.QueryRow(`SELECT instructions FROM plays WHERE id = ?`, id).Scan(&s))
		return s
	}
	assert.Equal(t, "Poll `deploy_list` with an `id` until it settles.", text("p-code"))
	assert.Equal(t, "Read the log with deploy_list.", text("p-bare"))
	assert.Equal(t, "Read the ticket with ticket_get.", text("p-other"))
}

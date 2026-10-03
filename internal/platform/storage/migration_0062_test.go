package storage

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
)

// TestMigration0062_NewInterviewInstructionsReachOnlyUneditedPlays upgrades a database with a workspace still on the
// seeded Interview play and one that edited it, plus an instance template saved with the seeded text or its own.
func TestMigration0062_NewInterviewInstructionsReachOnlyUneditedPlays(t *testing.T) {
	var want plays.Builtin
	for _, b := range plays.Builtins() {
		if b.Key == "interview" {
			want = b
		}
	}
	for name, instanceEdited := range map[string]bool{"instance template on the seeded text": false, "instance template edited": true} {
		t.Run(name, func(t *testing.T) {
			db := migrateBefore(t, "0062")
			var seeded string
			require.NoError(t, db.QueryRow(`SELECT instructions FROM plays WHERE workspace_id = 'workspace-default' AND builtin_key = 'interview'`).Scan(&seeded))
			instanceBody := seeded
			if instanceEdited {
				instanceBody = "Ask our way."
			}
			_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-own', 'Own', 'own', 1, 1);
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, created_by, created_at, updated_at, builtin_key) VALUES
    ('p-own', 'ws-own', 'Interview', 'interview', 'Ours.', 'Interview them our way.', 1, NULL, '[]', '', 1, 1, 'interview');
INSERT INTO instance_templates (kind, key, body, updated_at) VALUES ('play_instructions', 'interview', ?, 1);`, instanceBody)
			require.NoError(t, err)

			require.NoError(t, Migrate(db), "0062 applies on top, as an upgrade would")

			var got, gotDesc string
			require.NoError(t, db.QueryRow(`SELECT instructions, description FROM plays WHERE workspace_id = 'workspace-default' AND builtin_key = 'interview'`).Scan(&got, &gotDesc))
			assert.Equal(t, want.Instructions, got, "the unedited play gets the code default, so the migration's text matches it")
			assert.Equal(t, want.Description, gotDesc)
			require.NoError(t, db.QueryRow(`SELECT instructions, description FROM plays WHERE id = 'p-own'`).Scan(&got, &gotDesc))
			assert.Equal(t, []string{"Interview them our way.", "Ours."}, []string{got, gotDesc}, "an edited play keeps its own")

			var body string
			err = db.QueryRow(`SELECT body FROM instance_templates WHERE kind = 'play_instructions' AND key = 'interview'`).Scan(&body)
			if !instanceEdited {
				assert.ErrorIs(t, err, sql.ErrNoRows, "an instance template on the seeded text follows the code default again")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "Ask our way.", body)
		})
	}
}

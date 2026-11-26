package storage

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
)

// TestMigration0159_SeedsTheFlowsAutoPlaysOffAndCarriesTheNewTexts upgrades the default workspace, unedited and with
// no auto plays, beside one whose Fix with AI and To tickets were edited and whose Fix with AI already starts when a
// ticket enters progress.
func TestMigration0159_SeedsTheFlowsAutoPlaysOffAndCarriesTheNewTexts(t *testing.T) {
	db := migrateBefore(t, "0159")
	var oldFix string
	require.NoError(t, db.QueryRow(`SELECT instructions FROM plays WHERE workspace_id = 'workspace-default' AND builtin_key = 'fix-with-ai'`).Scan(&oldFix))
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-own', 'Own', 'own', 1, 1);
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, created_by, created_at, updated_at, builtin_key) VALUES
    ('p-fix', 'ws-own', 'Fix with AI', 'ticket', '', 'Fix it our way.', 1, 'progress', '[]', '', 1, 1, 'fix-with-ai'),
    ('p-cut', 'ws-own', 'To tickets via AI', 'doc', '', 'Cut it our way.', 1, NULL, '[]', '', 1, 1, 'to-tickets-via-ai'),
    ('p-test', 'ws-own', 'Test with AI', 'ticket', '', 'Test it our way.', 1, 'testing', '[]', '', 1, 1, 'test-with-ai');
INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, moment_stage, run_on, created_at, updated_at)
    VALUES ('ap-own', 'p-fix', 'ws-own', 1, 'ticket.entered_stage', 'progress', 'causer', 1, 1);
INSERT INTO instance_templates (kind, key, body, updated_at) VALUES ('play_instructions', 'fix-with-ai', ?1, 1);`, oldFix)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0159 applies on top, as an upgrade would")

	code := map[string]plays.Builtin{}
	for _, b := range plays.Builtins() {
		code[b.Key] = b
	}
	const gone = "gone"
	for _, tt := range []struct{ name, query, want string }{
		{"an unedited Fix with AI gets the new text", `SELECT instructions FROM plays WHERE workspace_id = 'workspace-default' AND builtin_key = 'fix-with-ai'`, code["fix-with-ai"].Instructions},
		{"an unedited To tickets gets the new text", `SELECT instructions FROM plays WHERE workspace_id = 'workspace-default' AND builtin_key = 'to-tickets-via-ai'`, code["to-tickets-via-ai"].Instructions},
		{"its history ends at that text, as the seeded default", `SELECT body || '|' || builtin_default FROM play_instruction_versions
            WHERE play_id = (SELECT id FROM plays WHERE workspace_id = 'workspace-default' AND builtin_key = 'to-tickets-via-ai') ORDER BY version DESC LIMIT 1`, code["to-tickets-via-ai"].Instructions + "|1"},
		{"an edited Fix with AI keeps its own", `SELECT instructions FROM plays WHERE id = 'p-fix'`, "Fix it our way."},
		{"an edited To tickets keeps its own", `SELECT instructions FROM plays WHERE id = 'p-cut'`, "Cut it our way."},
		{"an instance template on the 0157 text follows the code default", `SELECT body FROM instance_templates WHERE kind = 'play_instructions' AND key = 'fix-with-ai'`, gone},
		{"an auto play the workspace built is left as it was", `SELECT enabled || run_on FROM auto_plays WHERE id = 'ap-own'`, "1causer"},
	} {
		var got string
		err := db.QueryRow(tt.query).Scan(&got)
		if tt.want == gone {
			assert.ErrorIs(t, err, sql.ErrNoRows, tt.name)
			continue
		}
		require.NoError(t, err, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}

	seeded := func(key string, skip int) []string {
		var out []string
		for _, in := range code[key].AutoPlays[skip:] {
			conditions, err := json.Marshal(in.Conditions)
			require.NoError(t, err)
			priority, err := json.Marshal(in.Priority)
			require.NoError(t, err)
			stage := ""
			if in.MomentStage != nil {
				stage = string(*in.MomentStage)
			}
			out = append(out, string(in.Moment)+"|"+stage+"|"+string(conditions)+"|"+string(priority)+"|"+string(in.RunOn)+"|"+map[bool]string{true: "1", false: "0"}[in.OneAtATime])
		}
		return out
	}
	stored := func(ws, key string) []string {
		rows, err := db.Query(`
SELECT a.moment || '|' || coalesce(a.moment_stage, '') || '|' || a.conditions || '|' || a.priority || '|' || a.run_on || '|' || a.one_at_a_time
FROM auto_plays a JOIN plays p ON p.id = a.play_id
WHERE p.workspace_id = ? AND p.builtin_key = ? AND a.id != 'ap-own'
  AND a.enabled = 0 AND a.schedule IS NULL AND a.once_within_minutes = 0
  AND a.run_as = '' AND a.memory_ids = '[]' AND a.resource = '' AND a.created_by = ''
ORDER BY a.moment, coalesce(a.moment_stage, '')`, ws, key)
		require.NoError(t, err)
		var out []string
		for rows.Next() {
			var row string
			require.NoError(t, rows.Scan(&row))
			out = append(out, row)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		return out
	}
	for _, tt := range []struct {
		name, ws, key string
		want          []string
	}{
		{"Fix with AI gets both, off, as a new workspace's", "workspace-default", "fix-with-ai", seeded("fix-with-ai", 0)},
		{"Test with AI gets its one, off", "workspace-default", "test-with-ai", seeded("test-with-ai", 0)},
		{"a Fix with AI that starts on progress already gets only the unblocked one", "ws-own", "fix-with-ai", seeded("fix-with-ai", 1)},
		{"an edited Test with AI still gets its auto play", "ws-own", "test-with-ai", seeded("test-with-ai", 0)},
	} {
		assert.Equal(t, tt.want, stored(tt.ws, tt.key), tt.name)
	}
}

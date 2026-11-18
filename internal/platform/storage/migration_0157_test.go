package storage

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/workspace"
)

// TestMigration0157_NewStartingTemplatesReachOnlyUneditedCopies upgrades a database holding a built-in play, ticket
// types and instance templates still on the previous defaults beside edited ones of each.
func TestMigration0157_NewStartingTemplatesReachOnlyUneditedCopies(t *testing.T) {
	db := migrateBefore(t, "0157")
	var oldFix string
	require.NoError(t, db.QueryRow(`SELECT instructions FROM plays WHERE workspace_id = 'workspace-default' AND builtin_key = 'fix-with-ai'`).Scan(&oldFix))
	const oldTask = "## What needs doing\n\n\n## Acceptance criteria\n\n"
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-own', 'Own', 'own', 1, 1);
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, created_by, created_at, updated_at, builtin_key) VALUES
    ('p-own', 'ws-own', 'Fix with AI', 'ticket', 'Ours.', 'Fix it our way.', 1, 'progress', '[]', '', 1, 1, 'fix-with-ai');
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('pr-1', 'Web', 'WEB', 0, 'ws-own', 1, 1);
INSERT INTO ticket_types (id, project_id, name, position, color, created_at, updated_at, body_template) VALUES
    ('tt-task', 'pr-1', 'task', 0, '', 1, 1, ?1),
    ('tt-bug', 'pr-1', 'bug', 1, '', 1, 1, '## What broke');
INSERT INTO instance_templates (kind, key, body, updated_at) VALUES
    ('play_instructions', 'fix-with-ai', ?2, 1),
    ('play_instructions', 'clarify', 'Clarify our way.', 1),
    ('ticket_body', 'task', ?1, 1),
    ('ticket_body', 'bug', '## Our bug', 1);`, oldTask, oldFix)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0157 applies on top, as an upgrade would")

	var fix string
	for _, b := range plays.Builtins() {
		if b.Key == "fix-with-ai" {
			fix = b.Instructions
		}
	}
	const gone = "gone"
	for _, tt := range []struct{ name, query, want string }{
		{"an unedited play gets the code default", `SELECT instructions FROM plays WHERE workspace_id = 'workspace-default' AND builtin_key = 'fix-with-ai'`, fix},
		{"its history ends at that text, as the seeded default", `SELECT body || '|' || builtin_default FROM play_instruction_versions
            WHERE play_id = (SELECT id FROM plays WHERE workspace_id = 'workspace-default' AND builtin_key = 'fix-with-ai') ORDER BY version DESC LIMIT 1`, fix + "|1"},
		{"an edited play keeps its own", `SELECT instructions FROM plays WHERE id = 'p-own'`, "Fix it our way."},
		{"an instance template on the previous play text follows the code default", `SELECT body FROM instance_templates WHERE kind = 'play_instructions' AND key = 'fix-with-ai'`, gone},
		{"an edited instance play template stays", `SELECT body FROM instance_templates WHERE kind = 'play_instructions' AND key = 'clarify'`, "Clarify our way."},
		{"an unedited ticket type gets the code default", `SELECT body_template FROM ticket_types WHERE id = 'tt-task'`, workspace.DefaultTicketTypes[0].BodyTemplate},
		{"an edited ticket type keeps its own", `SELECT body_template FROM ticket_types WHERE id = 'tt-bug'`, "## What broke"},
		{"an instance body template on the previous default follows the code default", `SELECT body FROM instance_templates WHERE kind = 'ticket_body' AND key = 'task'`, gone},
		{"an edited instance body template stays", `SELECT body FROM instance_templates WHERE kind = 'ticket_body' AND key = 'bug'`, "## Our bug"},
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
}

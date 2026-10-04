package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/templates"
)

func TestMigration0055_ExistingWorkspacesFollowTheInstanceAndKeepTheirPlays(t *testing.T) {
	db := migrateBefore(t, "0055")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-old', 'Old', 'old', 1, 1);
INSERT INTO workspaces (id, name, slug, mention_chip_template, created_at, updated_at) VALUES ('ws-own', 'Own', 'own', '{ticket.Status}', 1, 1);
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, created_by, created_at, updated_at) VALUES
    ('p-fix', 'ws-own', 'Fix with AI', 'ticket', '', 'mine', 1, 'progress', '[]', '', 1, 1),
    ('p-fix-again', 'ws-own', 'Fix with AI', 'ticket', '', 'a second one', 1, 'progress', '[]', '', 2, 2),
    ('p-renamed', 'ws-own', 'Ship it', 'doc', '', 'renamed seed', 1, NULL, '[]', '', 1, 1);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0055 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	for id, edited := range map[string]bool{"workspace-default": false, "ws-old": false, "ws-own": true} {
		ws, err := s.Workspaces.Get(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, edited, ws.MentionChipTemplateEdited, "%s: only a workspace that chose its own chip keeps it", id)
	}
	own, err := s.Workspaces.Get(ctx, "ws-own")
	require.NoError(t, err)
	assert.Equal(t, "{ticket.Status}", own.MentionChipTemplate)

	keys := map[string]string{}
	seeded, err := s.Plays.List(ctx, "workspace-default")
	require.NoError(t, err)
	for _, p := range seeded {
		keys[p.Label] = p.BuiltinKey
	}
	assert.Equal(t, map[string]string{"Fix with AI": "fix-with-ai", "To tickets via AI": "to-tickets-via-ai", "Interview": "interview", "Test with AI": "test-with-ai", "Draft interview": "interview-draft"}, keys)
	for id, key := range map[string]string{"p-fix": "fix-with-ai", "p-fix-again": "", "p-renamed": ""} {
		p, err := s.Plays.Get(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, key, p.BuiltinKey, "%s: the first copy of a label is the built-in one; a renamed one is not matched", id)
		assert.NotEmpty(t, p.Instructions, "instructions are untouched")
	}

	stored, err := s.InstanceTemplates.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, stored, "no instance template is edited by the upgrade")
	require.NoError(t, s.InstanceTemplates.Save(ctx, &templates.Record{Kind: "interview", Body: "## Ours", UpdatedBy: "u-1", UpdatedAt: time.Unix(5, 0)},
		eventbus.OutboxEvent{ID: "evt-1", Topic: templates.TopicUpdated, Payload: templates.UpdatedEvent{Kind: "interview"}}))
	got, err := s.InstanceTemplates.Get(ctx, "interview", "")
	require.NoError(t, err)
	assert.Equal(t, "## Ours", got.Body)
	require.NoError(t, s.InstanceTemplates.Delete(ctx, "interview", ""))
	_, err = s.InstanceTemplates.Get(ctx, "interview", "")
	require.Error(t, err)
}

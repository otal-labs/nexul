package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/pairing"
)

// TestMigration0054_ASharedLinkGoesToItsComputersOwnerOnly upgrades a database holding one project link set by the
// computer's owner, then checks the owner keeps it whole, a teammate gets none, and the teammate can now set their own.
func TestMigration0054_ASharedLinkGoesToItsComputersOwnerOnly(t *testing.T) {
	db := migrateBefore(t, "0054")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-owner', 'owner', 1, 1), ('u-mate', 'mate', 1, 1);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0);
INSERT INTO pairing_computers (id, user_id, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at) VALUES
    ('c-owner', 'u-owner', 'owner-desktop', 'https://owner.example.com', 'sealed', 2000000000, '0.0.34', 1, 1),
    ('c-mate', 'u-mate', 'mate-laptop', 'https://mate.example.com', 'sealed', 2000000000, '0.0.34', 1, 1);
INSERT INTO pairing_project_links (project_id, computer_id, harness_project_id, provider, model, model_options, updated_at) VALUES
    ('p-web', 'c-owner', 't3-web', 'claude', 'sonnet-5', '[{"id":"effort","value":"high"}]', 1000);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0054 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	owner, err := s.Pairing.GetProjectLink(ctx, "u-owner", "p-web")
	require.NoError(t, err)
	assert.Equal(t, pairing.ProjectLink{
		UserID: "u-owner", ProjectID: "p-web", ComputerID: "c-owner", HarnessProjectID: "t3-web", Provider: "claude", Model: "sonnet-5",
		ModelOptions: []harness.OptionSetting{{ID: "effort", Value: "high"}}, UpdatedAt: time.Unix(1000, 0).UTC(),
	}, owner, "the person who set the link keeps every column of it")

	mate, err := s.Pairing.GetProjectLink(ctx, "u-mate", "p-web")
	require.NoError(t, err)
	assert.Empty(t, mate.ComputerID, "a teammate is given no copy and falls back to their own defaults")

	require.NoError(t, s.Pairing.SaveProjectLink(ctx, pairing.ProjectLink{UserID: "u-mate", ProjectID: "p-web", ComputerID: "c-mate", HarnessProjectID: "t3-mine"}))
	owner, err = s.Pairing.GetProjectLink(ctx, "u-owner", "p-web")
	require.NoError(t, err)
	assert.Equal(t, "c-owner", owner.ComputerID, "the teammate's own link sits beside the owner's, never over it")
}

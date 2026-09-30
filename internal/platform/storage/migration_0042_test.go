package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/pairing"
)

func TestMigrations0042And0043_RowsFromBeforeThemReadBackWithNothingPicked(t *testing.T) {
	db := migrateBefore(t, "0042")
	_, err := db.Exec(`INSERT INTO users (id, login, created_at, updated_at) VALUES ('u1', 'onik97', 1, 1);
INSERT INTO pairing_computers (id, user_id, kind, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at, setup_skipped_providers)
VALUES ('c-old', 'u1', 't3code', 'home', 'https://home.example.com', 'sealed', 2000000000, '0.0.34', 1, 1, '["codex"]');
INSERT INTO pairing_user_defaults (user_id, default_computer_id, fallback_project_id, provider, model) VALUES ('u1', 'c-old', 'p', 'claudeAgent', 'claude-opus-5-5')`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0042, 0043, and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)

	defaults, err := s.Pairing.GetDefaults(t.Context(), "u1")
	require.NoError(t, err)
	assert.Equal(t, "claude-opus-5-5", defaults.Model)
	assert.Nil(t, defaults.ModelOptions, "a stored model keeps running on the harness defaults")

	computer, err := s.Pairing.GetComputer(t.Context(), "u1", "c-old")
	require.NoError(t, err)
	assert.Equal(t, pairing.SetupChoices{
		Skipped: []string{"codex"}, Models: map[string]string{}, ModelOptions: map[string][]harness.OptionSetting{},
	}, computer.SetupChoices, "the skipped providers survive; no model or folder is picked yet")
}

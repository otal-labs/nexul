package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/pairing"
)

func TestMigration0056_TurnsFromBeforeItReadAsSetupTurns(t *testing.T) {
	db := migrateBefore(t, "0056")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-1', 'one', 1, 1);
INSERT INTO pairing_computers (id, user_id, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at) VALUES
    ('c-1', 'u-1', 'desk', 'https://desk.example.com', 'sealed', 2000000000, '0.0.34', 1, 1);
INSERT INTO pairing_setup_turns (id, run_id, computer_id, provider, provider_name, state, status, started_at, updated_at) VALUES
    ('t-1', 'r-1', 'c-1', 'codex', 'Codex', 'failed', 'npx: command not found', 1, 1);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0056 and every later migration apply on top, as an upgrade would")

	got, err := New(db, testEncKey).Pairing.ListLatestSetupTurns(t.Context(), "c-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, pairing.SetupTurnSetup, got[0].Kind, "a failed turn from before still retries as setup")
}

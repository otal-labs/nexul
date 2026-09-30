package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration0041_ComputersPairedBeforeItSkipNoProvider(t *testing.T) {
	db := migrateBefore(t, "0041")
	_, err := db.Exec(`INSERT INTO users (id, login, created_at, updated_at) VALUES ('u1', 'onik97', 1, 1);
INSERT INTO pairing_computers (id, user_id, kind, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at, setup_confirmed_at)
VALUES ('c-old', 'u1', 't3code', 'home', 'https://home.example.com', 'sealed', 2000000000, '0.0.34', 1, 1, 1700000000)`)
	require.NoError(t, err)

	script, err := migrationFS.ReadFile("migrations/0041_computer_setup_skipped_providers.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0041_computer_setup_skipped_providers", string(script)))
	require.NoError(t, Migrate(db), "every later migration applies on top, as an upgrade would")
	s := New(db, testEncKey)

	got, err := s.Pairing.GetComputer(t.Context(), "u1", "c-old")
	require.NoError(t, err)
	assert.Equal(t, []string{}, got.SetupSkipped, "an existing computer keeps setting up every provider")
	assert.NotNil(t, got.SetupConfirmedAt, "its confirmation is untouched")
}

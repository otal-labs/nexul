package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// TestMigration0185_DropsOnlyUnfinishedSetupsOlderThanADay keeps a day-old setup that has a runner, a named one and a
// fresh one, and drops the abandoned install.
func TestMigration0185_DropsOnlyUnfinishedSetupsOlderThanADay(t *testing.T) {
	db := migrateBefore(t, "0185")
	fresh := time.Now().Unix()
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-alice', 'alice', 1, 1);
INSERT INTO pairing_computers (id, user_id, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at) VALUES
    ('c-abandoned', 'u-alice', '', '', '', -62135596800, '', 1, 1),
    ('c-runner', 'u-alice', '', '', '', -62135596800, '', 1, 1),
    ('c-named', 'u-alice', 'desk', '', '', 0, '', 1, 1),
    ('c-fresh', 'u-alice', '', '', '', -62135596800, '', ?1, ?1);
INSERT INTO runners (id, name, last_seen, created_at, computer_id) VALUES ('r-1', 'r', 1, 1, 'c-runner');`, fresh)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	rows, err := db.Query(`SELECT id FROM pairing_computers ORDER BY id`)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })
	var left []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		left = append(left, id)
	}
	assert.Equal(t, []string{"c-fresh", "c-named", "c-runner"}, left)
}

func TestPairingRepo_AutoPlaysOffAndUnfinishedRoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	blank := newTestComputer("c2", "u1", "")
	blank.BearerToken, blank.TokenExpiresAt = "", time.Time{}
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))
	require.NoError(t, s.Pairing.SaveComputer(ctx, blank))

	require.NoError(t, s.Pairing.SetComputerAutoPlaysOff(ctx, "u1", "c1", true, time.Now()))
	require.ErrorIs(t, s.Pairing.SetComputerAutoPlaysOff(ctx, "u2", "c1", false, time.Now()), apperrs.ErrNotFound)

	home, err := s.Pairing.GetComputer(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.True(t, home.AutoPlaysOff)
	assert.False(t, home.PlaysOff)
	assert.False(t, home.Unfinished)
	setup, err := s.Pairing.GetComputer(ctx, "u1", "c2")
	require.NoError(t, err)
	assert.True(t, setup.Unfinished)
}

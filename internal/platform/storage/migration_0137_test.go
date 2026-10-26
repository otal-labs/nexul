package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0137_TicketPeopleBecomeUserIDs turns each developer and tester login into its account's id, the exact
// spelling first when two accounts share it in different cases, and keeps a login nobody signed in with as typed.
func TestMigration0137_TicketPeopleBecomeUserIDs(t *testing.T) {
	db := migrateBefore(t, "0137")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-onik', 'Onik97', 1, 1), ('u-lena', 'lena', 1, 1), ('u-lena-2', 'Lena', 1, 1);
INSERT INTO tickets (id, title, body, status, developer, tester, created_at, updated_at) VALUES
    ('t-people', 'Fix login', '', 'open', 'onik97', 'Lena', 1, 1),
    ('t-stranger', 'Fix logout', '', 'open', 'Sam', '', 1, 1);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0137 and every later migration apply on top, as an upgrade would")
	require.NoError(t, Migrate(db), "a second run changes nothing")

	people := func(id string) (developer, tester string) {
		require.NoError(t, db.QueryRow(`SELECT developer, tester FROM tickets WHERE id = ?`, id).Scan(&developer, &tester))
		return developer, tester
	}
	developer, tester := people("t-people")
	assert.Equal(t, "u-onik", developer)
	assert.Equal(t, "u-lena-2", tester)
	developer, tester = people("t-stranger")
	assert.Equal(t, "Sam", developer)
	assert.Empty(t, tester)
}

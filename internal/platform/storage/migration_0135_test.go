package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0135_TicketPeopleTakeTheirAccountsSpelling rewrites a developer or tester typed in another case to the
// account's login and leaves a login nobody signed in with as it was.
func TestMigration0135_TicketPeopleTakeTheirAccountsSpelling(t *testing.T) {
	db := migrateBefore(t, "0135")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-onik', 'Onik97', 1, 1), ('u-lena', 'lena', 1, 1);
INSERT INTO tickets (id, title, body, status, developer, tester, created_at, updated_at) VALUES
    ('t-mixed', 'Fix login', '', 'open', 'onik97', 'LENA', 1, 1),
    ('t-stranger', 'Fix logout', '', 'open', 'Sam', '', 1, 1);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0135 and every later migration apply on top, as an upgrade would")

	people := func(id string) (developer, tester string) {
		require.NoError(t, db.QueryRow(`SELECT developer, tester FROM tickets WHERE id = ?`, id).Scan(&developer, &tester))
		return developer, tester
	}
	developer, tester := people("t-mixed")
	assert.Equal(t, "Onik97", developer)
	assert.Equal(t, "lena", tester)
	developer, tester = people("t-stranger")
	assert.Equal(t, "Sam", developer)
	assert.Empty(t, tester)
}

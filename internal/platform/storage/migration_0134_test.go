package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0134_ExistingTicketsAndPeopleStartWithNoDueDateOrTimezone upgrades a database holding a person and a
// ticket: the ticket has no due date, the person no timezone, and the calendar starts empty.
func TestMigration0134_ExistingTicketsAndPeopleStartWithNoDueDateOrTimezone(t *testing.T) {
	db := migrateBefore(t, "0134")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-1', 'alice', 1, 1);
INSERT INTO tickets (id, title, body, status, developer, created_at, updated_at) VALUES
    ('t-1', 'Fix login', '', 'open', 'alice', 1, 1);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0134 and every later migration apply on top, as an upgrade would")

	var dueOn, timezone string
	require.NoError(t, db.QueryRow(`SELECT due_on FROM tickets WHERE id = 't-1'`).Scan(&dueOn))
	require.NoError(t, db.QueryRow(`SELECT timezone FROM users WHERE id = 'u-1'`).Scan(&timezone))
	assert.Empty(t, dueOn)
	assert.Empty(t, timezone)

	var meetings int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM meetings`).Scan(&meetings))
	assert.Zero(t, meetings)
}

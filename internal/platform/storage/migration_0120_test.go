package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/storage/testutil"
)

// TestMigration0120_StacksStartWithCrashTicketsOff upgrades a database that holds stacks, a ticket and a stack's
// service: every stack reads with the switch off and keeps its row, the link table appears empty, a link follows its
// stack and ticket when either is deleted, and a stack can switch the setting on afterwards.
func TestMigration0120_StacksStartWithCrashTicketsOff(t *testing.T) {
	db := migrateBefore(t, "0120")
	_, err := db.Exec(`
INSERT INTO stacks (id, name, slug, machine, strategy, created_at, updated_at) VALUES
    ('st-acme', 'acme-web', 'acme-web', 'box', 'compose', 1, 1),
    ('st-globex', 'globex-api', 'globex-api', 'box', 'compose', 1, 1);
INSERT INTO services (id, stack_id, name) VALUES ('svc-web', 'st-acme', 'web');`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	rows, err := db.Query(`SELECT id, crash_tickets FROM stacks ORDER BY id`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	got := map[string]int{}
	for rows.Next() {
		var id string
		var on int
		require.NoError(t, rows.Scan(&id, &on))
		got[id] = on
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, map[string]int{"st-acme": 0, "st-globex": 0}, got)

	var links, services int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM stack_crash_tickets`).Scan(&links))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&services))
	assert.Zero(t, links)
	assert.Equal(t, 1, services, "existing rows are untouched")

	require.NoError(t, testutil.SeedGeneralProject(db))
	s := New(db, testEncKey)
	require.NoError(t, s.Tickets.Create(t.Context(), newTestTicket("t-1", "")))
	_, err = db.Exec(`INSERT INTO stack_crash_tickets (stack_id, subject, ticket_id) VALUES ('st-acme', 'web', 't-1'), ('st-globex', 'api', 't-1')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO stack_crash_tickets (stack_id, subject, ticket_id) VALUES ('st-acme', 'web', 't-1')`)
	assert.Error(t, err, "a subject has one link")

	_, err = db.Exec(`DELETE FROM stacks WHERE id = 'st-acme'`)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM stack_crash_tickets`).Scan(&links))
	assert.Equal(t, 1, links, "a deleted stack takes only its own links")
	_, err = db.Exec(`DELETE FROM tickets WHERE id = 't-1'`)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM stack_crash_tickets`).Scan(&links))
	assert.Zero(t, links, "a deleted ticket takes its links")

	_, err = db.Exec(`UPDATE stacks SET crash_tickets = 1 WHERE id = 'st-globex'`)
	require.NoError(t, err)
}

package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0136_StacksStartWithNoCrashNotices upgrades a database that holds a stack: it keeps its row and reads
// with no notice conversation or bot, and can name both afterwards.
func TestMigration0136_StacksStartWithNoCrashNotices(t *testing.T) {
	db := migrateBefore(t, "0136")
	_, err := db.Exec(`INSERT INTO stacks (id, name, slug, machine, strategy, crash_tickets, created_at, updated_at)
VALUES ('st-acme', 'acme-web', 'acme-web', 'box', 'compose', 1, 1, 1)`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	var conversation, bot string
	var crashTickets int
	require.NoError(t, db.QueryRow(`SELECT crash_notice_conversation_id, crash_notice_bot_id, crash_tickets FROM stacks WHERE id = 'st-acme'`).
		Scan(&conversation, &bot, &crashTickets))
	assert.Empty(t, conversation)
	assert.Empty(t, bot)
	assert.Equal(t, 1, crashTickets, "the crash ticket switch is untouched")

	_, err = db.Exec(`UPDATE stacks SET crash_notice_conversation_id = 'c-alerts', crash_notice_bot_id = 'bot-1' WHERE id = 'st-acme'`)
	require.NoError(t, err)
}

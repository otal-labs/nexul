package storage

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/tickets"
)

// TestMigration0141_ExistingTicketsStartTheirHistory gives each ticket one created change by its stored reporter, at
// its creation time, and its current description as version 1; a second run writes nothing more.
func TestMigration0141_ExistingTicketsStartTheirHistory(t *testing.T) {
	db := migrateBefore(t, "0141")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-onik', 'Onik97', 1, 1);
INSERT INTO tickets (id, title, body, status, reporter_kind, reporter_login, reporter_automation_id, reporter_automation_name, created_at, updated_at) VALUES
    ('t-person', 'Fix login', 'login body', 'open', 'user', 'onik97', '', '', 100, 300),
    ('t-agent', 'Fix logout', '', 'open', 'user:mcp', 'Onik97', '', '', 200, 200),
    ('t-bot', 'Triage', 'bot body', 'open', 'automation', '', 'auto-1', 'Nightly sync', 150, 150),
    ('t-stranger', 'Fix signup', 'stranger body', 'open', 'user', 'Sam', '', '', 250, 250);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0141 and every later migration apply on top, as an upgrade would")
	require.NoError(t, Migrate(db), "a second run changes nothing")

	type created struct {
		fromTo    string
		actor     tickets.Actor
		createdAt int64
	}
	createdOf := func(id string) (c created) {
		var raw string
		require.NoError(t, db.QueryRow(`SELECT from_value || to_value, actor, created_at FROM ticket_changes WHERE ticket_id = ? AND field = 'created'`, id).Scan(&c.fromTo, &raw, &c.createdAt))
		require.NoError(t, json.Unmarshal([]byte(raw), &c.actor))
		return c
	}
	assert.Equal(t, created{"", tickets.Actor{Kind: "user", UserID: "u-onik"}, 100}, createdOf("t-person"), "a login finds its account whatever its case")
	assert.Equal(t, created{"", tickets.Actor{Kind: "user:mcp", UserID: "u-onik"}, 200}, createdOf("t-agent"))
	assert.Equal(t, created{"", tickets.Actor{Kind: "automation", AutomationID: "auto-1", AutomationName: "Nightly sync"}, 150}, createdOf("t-bot"))
	assert.Equal(t, created{"", tickets.Actor{Kind: "user"}, 250}, createdOf("t-stranger"), "a login nobody signed in with names no user")

	var changes int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM ticket_changes`).Scan(&changes))
	assert.Equal(t, 4, changes, "one change per ticket")

	var body string
	var version int
	var editors, via string
	var at int64
	require.NoError(t, db.QueryRow(`SELECT version, body, editor_ids, via, created_at FROM ticket_description_versions WHERE ticket_id = 't-person'`).Scan(&version, &body, &editors, &via, &at))
	assert.Equal(t, 1, version)
	assert.Equal(t, "login body", body)
	assert.Equal(t, "[]", editors)
	assert.Empty(t, via)
	assert.Equal(t, int64(300), at, "version 1 is dated when the ticket last changed, which is no earlier than its body")
	var versions int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM ticket_description_versions`).Scan(&versions))
	assert.Equal(t, 4, versions, "one version per ticket, an empty description included")
}

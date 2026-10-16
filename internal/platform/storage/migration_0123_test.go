package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0123_StacksStartWithTheErrorRateRuleOffAtItsDefaults upgrades a database that holds stacks: each reads
// with the rule off and the starting numbers and keeps its row, the episodes table appears empty beside them, a stack
// holds at most one open episode while closed ones stack up, a deleted stack takes its episodes, and a rule can be
// switched on afterwards.
func TestMigration0123_StacksStartWithTheErrorRateRuleOffAtItsDefaults(t *testing.T) {
	db := migrateBefore(t, "0123")
	_, err := db.Exec(`
INSERT INTO stacks (id, name, slug, machine, strategy, created_at, updated_at) VALUES
    ('st-acme', 'acme-web', 'acme-web', 'box', 'compose', 1, 1),
    ('st-globex', 'globex-api', 'globex-api', 'box', 'compose', 1, 1);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	rows, err := db.Query(`SELECT id, error_rate_enabled, error_rate_percent, error_rate_minutes, error_rate_min_requests FROM stacks ORDER BY id`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	type rule struct {
		Enabled, Minutes, MinRequests int
		Percent                       float64
	}
	got := map[string]rule{}
	for rows.Next() {
		var id string
		var r rule
		require.NoError(t, rows.Scan(&id, &r.Enabled, &r.Percent, &r.Minutes, &r.MinRequests))
		got[id] = r
	}
	require.NoError(t, rows.Err())
	start := rule{Enabled: 0, Percent: 5, Minutes: 5, MinRequests: 20}
	assert.Equal(t, map[string]rule{"st-acme": start, "st-globex": start}, got)

	var episodes int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM error_rate_episodes`).Scan(&episodes))
	assert.Zero(t, episodes)

	insert := `INSERT INTO error_rate_episodes (id, stack_id, started_at, peak_percent, requests, errors, limit_percent, window_minutes, min_requests, ended_at)
        VALUES (?, 'st-acme', 10, 7.5, 400, 30, 5, 5, 20, ?)`
	_, err = db.Exec(insert, "closed-1", 20)
	require.NoError(t, err)
	_, err = db.Exec(insert, "closed-2", 30)
	require.NoError(t, err)
	_, err = db.Exec(insert, "open-1", nil)
	require.NoError(t, err)
	_, err = db.Exec(insert, "open-2", nil)
	assert.Error(t, err, "a stack never has two open episodes")

	_, err = db.Exec(`DELETE FROM stacks WHERE id = 'st-acme'`)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM error_rate_episodes`).Scan(&episodes))
	assert.Zero(t, episodes, "a deleted stack takes its episodes with it")

	_, err = db.Exec(`UPDATE stacks SET error_rate_enabled = 1 WHERE id = 'st-globex'`)
	require.NoError(t, err)
}

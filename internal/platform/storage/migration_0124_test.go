package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0124_RequestMinutesKeepTheirCountsAndGainASource upgrades minutes of a stack behind a tunnel, one
// behind a proxy and one with no exposure: each keeps its counts, a missing minute stays NULL, the tunnel stack's
// go to the tunnel source and the others to the proxy, and a second source can now share a minute.
func TestMigration0124_RequestMinutesKeepTheirCountsAndGainASource(t *testing.T) {
	db := migrateBefore(t, "0124")
	_, err := db.Exec(`
INSERT INTO stacks (id, name, slug, machine, strategy, created_at, updated_at) VALUES
    ('st-tunnel', 'acme-web', 'acme-web', 'box', 'compose', 1, 1),
    ('st-proxy', 'globex-api', 'globex-api', 'box', 'compose', 1, 1),
    ('st-none', 'northwind-job', 'northwind-job', 'box', 'compose', 1, 1);
INSERT INTO services (id, stack_id, name) VALUES
    ('c-tunnel', 'st-tunnel', 'web'), ('c-proxy', 'st-proxy', 'api');
INSERT INTO dns_gateways (id, kind, docker_network, created_at, updated_at) VALUES
    ('gw-tunnel', 'tunnel', 'net-t', 1, 1), ('gw-proxy', 'proxy', 'net-p', 1, 1);
INSERT INTO dns_exposures (id, gateway_id, hostname, service, port, created_at, updated_at, service_id) VALUES
    ('e1', 'gw-tunnel', 'app.example.com', 'web', 80, 1, 1, 'c-tunnel'),
    ('e2', 'gw-proxy', 'api.example.com', 'api', 80, 1, 1, 'c-proxy');
INSERT INTO stack_request_minutes (stack_id, minute, requests, errors_5xx) VALUES
    ('st-tunnel', 60, 56, 4), ('st-tunnel', 120, NULL, NULL),
    ('st-proxy', 60, 10, 0), ('st-none', 60, 3, 1);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	rows := func() []string {
		r, err := db.Query(`SELECT stack_id || '|' || source || '|' || minute || '|' || COALESCE(requests, 'null') || '|' || COALESCE(errors_5xx, 'null')
FROM stack_request_minutes ORDER BY 1`)
		require.NoError(t, err)
		defer func() { require.NoError(t, r.Close()) }()
		var out []string
		for r.Next() {
			var line string
			require.NoError(t, r.Scan(&line))
			out = append(out, line)
		}
		require.NoError(t, r.Err())
		return out
	}
	assert.Equal(t, []string{
		"st-none|proxy|60|3|1",
		"st-proxy|proxy|60|10|0",
		"st-tunnel|tunnel|120|null|null",
		"st-tunnel|tunnel|60|56|4",
	}, rows())

	_, err = db.Exec(`INSERT INTO stack_request_minutes (stack_id, source, minute, requests, errors_5xx) VALUES ('st-tunnel', 'proxy', 60, 9, 0)`)
	require.NoError(t, err, "the proxy and the tunnel each keep a row for the same minute")
	_, err = db.Exec(`INSERT INTO stack_request_minutes (stack_id, source, minute) VALUES ('st-tunnel', 'proxy', 60)`)
	assert.Error(t, err, "a source has one row a minute")

	var index int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM pragma_index_list('stack_request_minutes') WHERE name = 'idx_stack_request_minutes_minute'`).Scan(&index))
	assert.Equal(t, 1, index, "the prune index is rebuilt")

	_, err = db.Exec(`DELETE FROM stacks WHERE id = 'st-tunnel'`)
	require.NoError(t, err)
	var left int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM stack_request_minutes WHERE stack_id = 'st-tunnel'`).Scan(&left))
	assert.Zero(t, left, "a deleted stack takes its minutes with it")
}

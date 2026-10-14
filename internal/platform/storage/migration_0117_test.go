package storage

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	stackEventWithValues = `{"stack":{"id":"st-acme","name":"acme-web","machine":"box","strategy":"compose",` +
		`"env":{"DATABASE_URL":"example-secret-db","API_KEY":"example-secret-key","EMPTY":""},` +
		`"branch_deploy_rules":[{"pattern":"feature/*","docker_network":"acme-net","overrides":{"API_KEY":"example-secret-preview"}},` +
		`{"pattern":"main","docker_network":"acme-net"}],"managed":true},"services":[{"id":"c-1","name":"web"}]}`
	stackEventBlanked = `{"stack":{"id":"st-acme","name":"acme-web","machine":"box","strategy":"compose",` +
		`"env":{"DATABASE_URL":"","API_KEY":"","EMPTY":""},` +
		`"branch_deploy_rules":[{"pattern":"feature/*","docker_network":"acme-net","overrides":{"API_KEY":""}},` +
		`{"pattern":"main","docker_network":"acme-net"}],"managed":true},"services":[{"id":"c-1","name":"web"}]}`
	stackEventOverridesOnly = `{"stack":{"id":"st-globex","env":{"PORT":""},"branch_deploy_rules":[{"pattern":"*","overrides":{"TOKEN":"example-secret-token"}}]}}`
	stackEventOverridesGone = `{"stack":{"id":"st-globex","env":{"PORT":""},"branch_deploy_rules":[{"pattern":"*","overrides":{"TOKEN":""}}]}}`
	stackEventAlreadyKeys   = `{"stack":{"id":"st-northwind","env":{"PORT":""},"branch_deploy_rules":[{"pattern":"*","overrides":{"TOKEN":""}}],"env_masked":true}}`
	stackEventNoEnv         = `{"stack":{"id":"st-bare","name":"bare"}}`
	otherTopicWithEnvShape  = `{"id":"d-1","env":{"PORT":"8080"},"stack":{"env":{"PORT":"8080"}}}`
)

type storedEvent struct{ state, payload string }

// eventCopies reads every stored copy of an event payload, keyed by table and id; state is the column a migration
// must leave alone (published, attempts) or move on purpose (a delivery's status).
func eventCopies(t *testing.T, db *sql.DB) map[string]storedEvent {
	t.Helper()
	rows, err := db.Query(`
SELECT 'outbox/' || id, CAST(published AS TEXT), CAST(payload AS TEXT) FROM outbox
UNION ALL SELECT 'dead_letters/' || id, CAST(attempts AS TEXT), CAST(payload AS TEXT) FROM dead_letters
UNION ALL SELECT 'integration_deliveries/' || id, status, CAST(payload AS TEXT) FROM integration_deliveries`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	out := map[string]storedEvent{}
	for rows.Next() {
		var key string
		var e storedEvent
		require.NoError(t, rows.Scan(&key, &e.state, &e.payload))
		out[key] = e
	}
	require.NoError(t, rows.Err())
	return out
}

func deliveryBody(id, topic, data string) string {
	return `{"delivery_id":"dl-` + id + `","event_id":"` + id + `","topic":"` + topic + `","data":` + data + `}`
}

// TestMigration0117_StackEventsLoseTheirEnvValues upgrades a database holding stack events from before values were kept
// out of them, in the outbox, the dead letters and the webhook deliveries: every value goes while every key and every
// other field stays, events with no value and events of other topics are byte-identical, a delivery that was still due
// ends instead of going out unsigned, the stack's own env is untouched, and a second run changes nothing.
func TestMigration0117_StackEventsLoseTheirEnvValues(t *testing.T) {
	db := migrateBefore(t, "0117")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('alice', 'alice', 1, 1);
INSERT INTO integration_installs (id, name, trust_tier, webhook_url, webhook_secret, scopes, created_by, created_at)
    VALUES ('install-acme', 'Acme hooks', 'community', 'https://example.com/hook', 's', '[]', 'alice', 1);
INSERT INTO stacks (id, name, slug, machine, strategy, env, created_at, updated_at)
    VALUES ('st-acme', 'acme-web', 'acme-web', 'box', 'compose', '{"DATABASE_URL":"example-secret-db"}', 1, 1);`)
	require.NoError(t, err)

	events := []struct{ id, topic, payload string }{
		{"created-with-values", "service.created", stackEventWithValues},
		{"updated-with-values", "service.updated", stackEventWithValues},
		{"updated-overrides-only", "service.updated", stackEventOverridesOnly},
		{"updated-already-keys", "service.updated", stackEventAlreadyKeys},
		{"updated-no-env", "service.updated", stackEventNoEnv},
		{"other-topic", "deploy.requested", otherTopicWithEnvShape},
	}
	for _, e := range events {
		_, err := db.Exec(`INSERT INTO outbox (id, topic, payload, created_at, published) VALUES (?, ?, ?, 1, 1)`, e.id, e.topic, []byte(e.payload))
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO dead_letters (id, topic, payload, error, attempts) VALUES (?, ?, ?, 'boom', 3)`, e.id, e.topic, []byte(e.payload))
		require.NoError(t, err)
	}
	deliveries := []struct{ id, topic, status, data string }{
		{"delivered", "service.created", "delivered", stackEventWithValues},
		{"pending", "service.updated", "pending", stackEventWithValues},
		{"failed-overrides", "service.updated", "failed", stackEventOverridesOnly},
		{"pending-already-keys", "service.updated", "pending", stackEventAlreadyKeys},
		{"pending-other-topic", "deploy.requested", "pending", otherTopicWithEnvShape},
	}
	for _, d := range deliveries {
		_, err := db.Exec(`INSERT INTO integration_deliveries (id, install_id, topic, event_id, payload, signature, url, status, attempts, next_attempt_at, created_at)
			VALUES (?, 'install-acme', ?, ?, ?, 'sha256=abc', 'https://example.com/hook', ?, 1, 5, 1)`,
			d.id, d.topic, d.id, []byte(deliveryBody(d.id, d.topic, d.data)), d.status)
		require.NoError(t, err)
	}
	before := eventCopies(t, db)

	require.NoError(t, Migrate(db))
	after := eventCopies(t, db)

	blanked := map[string]string{
		"created-with-values":    stackEventBlanked,
		"updated-with-values":    stackEventBlanked,
		"updated-overrides-only": stackEventOverridesGone,
	}
	for _, table := range []string{"outbox", "dead_letters"} {
		for _, e := range events {
			key := table + "/" + e.id
			want, scrubbed := blanked[e.id]
			if !scrubbed {
				assert.Equal(t, before[key], after[key], key+" is not touched")
				continue
			}
			assert.Equal(t, before[key].state, after[key].state, key)
			assert.JSONEq(t, want, after[key].payload, key)
			assert.NotContains(t, after[key].payload, "example-secret", key)
		}
	}
	for _, d := range deliveries {
		key := "integration_deliveries/" + d.id
		want, scrubbed := map[string]struct{ status, data string }{
			"delivered":        {"delivered", stackEventBlanked},
			"pending":          {"dead", stackEventBlanked},
			"failed-overrides": {"dead", stackEventOverridesGone},
		}[d.id]
		if !scrubbed {
			assert.Equal(t, before[key], after[key], key+" is not touched")
			continue
		}
		assert.Equal(t, want.status, after[key].state, key)
		assert.JSONEq(t, deliveryBody(d.id, d.topic, want.data), after[key].payload, key)
		assert.NotContains(t, after[key].payload, "example-secret", key)
	}

	var env string
	require.NoError(t, db.QueryRow(`SELECT env FROM stacks WHERE id = 'st-acme'`).Scan(&env))
	assert.JSONEq(t, `{"DATABASE_URL":"example-secret-db"}`, env, "the stack's own env is the live config")

	script, err := migrationFS.ReadFile("migrations/0117_scrub_stack_event_env.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(script))
	require.NoError(t, err)
	assert.Equal(t, after, eventCopies(t, db), "a second run changes nothing")
}

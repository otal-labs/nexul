package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
)

var retentionCutoff = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)

func countRows(t *testing.T, s *Store, query string) int {
	t.Helper()
	var n int
	require.NoError(t, s.db.QueryRowContext(context.Background(), query).Scan(&n))
	return n
}

func TestEventRetention_DeleteProcessedBefore_DropsOnlyRowsBeforeTheCutoff(t *testing.T) {
	s := newTestStore(t)
	ctx := t.Context()
	for id, at := range map[string]time.Time{"old-1": retentionCutoff.Add(-time.Hour), "old-2": retentionCutoff.Add(-2 * time.Hour), "new": retentionCutoff} {
		_, err := s.db.ExecContext(ctx, `INSERT INTO processed_events (event_id, processed_at) VALUES (?, ?)`, id, at.Unix())
		require.NoError(t, err)
	}

	n, err := s.EventRetention.DeleteProcessedBefore(ctx, retentionCutoff, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "one call deletes at most limit rows")
	_, err = s.EventRetention.DeleteProcessedBefore(ctx, retentionCutoff, 10)
	require.NoError(t, err)

	assert.Equal(t, 1, countRows(t, s, `SELECT COUNT(*) FROM processed_events WHERE event_id = 'new'`))
	assert.Equal(t, 1, countRows(t, s, `SELECT COUNT(*) FROM processed_events`))
}

func TestEventRetention_DeleteDeployLogBefore_DropsOnlyLinesBeforeTheCutoffInMilliseconds(t *testing.T) {
	s := newTestStore(t)
	ctx := t.Context()
	require.NoError(t, s.Deploys.Create(ctx, newTestDeploy("dep-1")))
	lines := []deploy.LogLine{
		{TS: retentionCutoff.Add(-time.Hour).UnixMilli(), Text: "old"},
		{TS: retentionCutoff.UnixMilli(), Text: "new"},
	}
	require.NoError(t, s.Deploys.AppendLogLines(ctx, "dep-1", lines))

	_, err := s.EventRetention.DeleteDeployLogBefore(ctx, retentionCutoff, 10)
	require.NoError(t, err)

	kept, err := s.Deploys.ListLogLines(ctx, "dep-1")
	require.NoError(t, err)
	require.Len(t, kept, 1)
	assert.Equal(t, "new", kept[0].Text)
}

func TestEventRetention_DeleteOutboxBefore_KeepsUnpublishedAndWhatAnEnabledAutomationHasNotRead(t *testing.T) {
	s := newTestStore(t)
	ctx := t.Context()
	old := retentionCutoff.Add(-time.Hour).Unix()
	older := retentionCutoff.Add(-2 * time.Hour).Unix()
	for _, r := range []struct {
		id        string
		at        int64
		published int
	}{
		{"published-old", older, 1},
		{"published-new", retentionCutoff.Unix(), 1},
		{"unpublished-old", older, 0},
		{"unread-by-live", old, 1},
	} {
		_, err := s.db.ExecContext(ctx, `INSERT INTO outbox (id, topic, payload, created_at, published) VALUES (?, 't', x'7b7d', ?, ?)`, r.id, r.at, r.published)
		require.NoError(t, err)
	}
	cursor := func(automationID string, enabled int, at int64) {
		_, err := s.db.ExecContext(ctx, `INSERT INTO automations (id, name, kind, enabled, token_hash, created_at, updated_at) VALUES (?, ?, 'k', ?, 'h', 1, 1)`, automationID, automationID, enabled)
		require.NoError(t, err)
		_, err = s.db.ExecContext(ctx, `INSERT INTO automation_cursors (automation_id, last_created_at, last_event_id, updated_at) VALUES (?, ?, '', 1)`, automationID, at)
		require.NoError(t, err)
	}
	cursor("live", 1, older+1)
	cursor("paused", 0, 0)
	_, err := s.db.ExecContext(ctx, `INSERT INTO automation_cursors (automation_id, last_created_at, last_event_id, updated_at) VALUES ('deleted', 0, '', 1)`)
	require.NoError(t, err)

	_, err = s.EventRetention.DeleteOutboxBefore(ctx, retentionCutoff, 10)
	require.NoError(t, err)

	var kept []string
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM outbox ORDER BY id`)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		kept = append(kept, id)
	}
	assert.Equal(t, []string{"published-new", "unpublished-old", "unread-by-live"}, kept,
		"a paused or deleted automation's cursor does not pin the outbox; unpublished and unread rows stay")

	_, err = s.db.ExecContext(ctx, `UPDATE automations SET enabled = 0 WHERE id = 'live'`)
	require.NoError(t, err)
	_, err = s.EventRetention.DeleteOutboxBefore(ctx, retentionCutoff, 10)
	require.NoError(t, err)
	assert.Equal(t, 0, countRows(t, s, `SELECT COUNT(*) FROM outbox WHERE id = 'unread-by-live'`), "once no enabled automation is behind, the old row goes")
	assert.Equal(t, 1, countRows(t, s, `SELECT COUNT(*) FROM outbox WHERE id = 'unpublished-old'`))
}

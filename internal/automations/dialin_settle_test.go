package automations_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

const settleStoreKey = "0123456789abcdef0123456789abcdef"

// seedOtherTopicRows fills the outbox with n rows on a topic nobody subscribes to, ten to a second, all long past.
func (s *dialinSetup) seedOtherTopicRows(t *testing.T, n int) {
	t.Helper()
	_, err := s.db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < ?)
		INSERT INTO outbox (id, topic, payload, created_at, published)
		SELECT printf('n-%07d', i), 'chat.message.posted', x'7b7d', 1700000000 + i / 10, 1 FROM n`, n)
	require.NoError(t, err)
}

// rowsPastCursor counts the log rows the next read walks before it finds a match: everything after the cursor.
func (s *dialinSetup) rowsPastCursor(t *testing.T, automationID string) int {
	t.Helper()
	cursor, _, err := storage.New(s.db, []byte(settleStoreKey)).AutomationCursors.Get(context.Background(), automationID)
	require.NoError(t, err)
	var n int
	require.NoError(t, s.db.QueryRow(`SELECT count(*) FROM outbox WHERE created_at > ? OR (created_at = ? AND id > ?)`,
		cursor.CreatedAt.Unix(), cursor.CreatedAt.Unix(), cursor.ID).Scan(&n))
	return n
}

// enabledFromTheStartOfTheLog makes an enabled automation whose cursor sits at the beginning of the log, as it does
// after downtime, so a backlog is there to be read.
func (s *dialinSetup) enabledFromTheStartOfTheLog(t *testing.T) (id, token string) {
	t.Helper()
	created, token, err := s.svc.Create(t.Context(), "owner", "ws-1", "x", []string{"tickets:read"})
	require.NoError(t, err)
	_, err = s.svc.SetEnabled(t.Context(), "owner", created.ID, true)
	require.NoError(t, err)
	_, err = s.db.Exec(`UPDATE automation_cursors SET last_created_at = 0, last_event_id = '' WHERE automation_id = ?`, created.ID)
	require.NoError(t, err)
	return created.ID, token
}

// TestDialinHandler_RareTopicCursorFollowsTheLog: an automation whose topic almost never occurs leaves the cursor
// behind its last match, so every wake re-reads the whole log after it; the number of rows left to read stays the
// same however long the log is.
func TestDialinHandler_RareTopicCursorFollowsTheLog(t *testing.T) {
	for _, size := range []int{1_000, 20_000} {
		t.Run(fmt.Sprintf("%d rows", size), func(t *testing.T) {
			setup := newDialinSetup(t)
			setup.seedOtherTopicRows(t, size)
			setup.seedEvent(t, "n-newest", "chat.message.posted", `{}`, time.Now())
			id, token := setup.enabledFromTheStartOfTheLog(t)

			dialAnnounce(t, setup.wsURL(token), automations.Frame{Subscriptions: []string{"ticket.created"}})

			require.Eventually(t, func() bool { return setup.rowsPastCursor(t, id) < size/2 },
				2*time.Second, 10*time.Millisecond, "rows left to read after a wake: %d", setup.rowsPastCursor(t, id))
			assert.Equal(t, 1, setup.rowsPastCursor(t, id), "only the newest second is left to read")
		})
	}
}

// TestDialinHandler_MatchAfterManyOtherRows_DeliveredOnceAcrossAReconnect: the cursor skipping past rows that do not
// match never skips one that does, including a row that commits later with a lower id in the same second as the
// newest row, and a delivered event is not delivered again when the automation reconnects.
func TestDialinHandler_MatchAfterManyOtherRows_DeliveredOnceAcrossAReconnect(t *testing.T) {
	setup := newDialinSetup(t)
	setup.seedOtherTopicRows(t, 5_000)
	now := time.Now()
	setup.seedEvent(t, "z-other", "chat.message.posted", `{}`, now)
	id, token := setup.enabledFromTheStartOfTheLog(t)
	frame := automations.Frame{Subscriptions: []string{"ticket.created"}}

	conn := dialAnnounce(t, setup.wsURL(token), frame)
	require.Eventually(t, func() bool { return setup.rowsPastCursor(t, id) == 1 }, 2*time.Second, 10*time.Millisecond)
	setup.seedEvent(t, "a-match", "ticket.created", `{"id":"t1"}`, now)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	var ev automations.Frame
	require.NoError(t, wsjson.Read(ctx, conn, &ev))
	require.Equal(t, "a-match", ev.EventID, "a match committed late with a lower id in the newest second is still delivered")
	require.NoError(t, wsjson.Write(ctx, conn, automations.Frame{Type: automations.FrameRunStarted, RunID: ev.RunID}))
	require.NoError(t, wsjson.Write(ctx, conn, automations.Frame{Type: automations.FrameRunFinished, RunID: ev.RunID, Outcome: automations.OutcomeSuccess}))
	store := storage.New(setup.db, []byte(settleStoreKey))
	require.Eventually(t, func() bool {
		cursor, _, err := store.AutomationCursors.Get(ctx, id)
		return err == nil && cursor.ID == "a-match"
	}, 2*time.Second, 10*time.Millisecond)
	require.NoError(t, conn.CloseNow())

	again := dialAnnounce(t, setup.wsURL(token), frame)
	quiet, stop := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer stop()
	assert.Error(t, wsjson.Read(quiet, again, &ev), "a delivered event is not delivered a second time")
	assert.ErrorIs(t, quiet.Err(), context.DeadlineExceeded, "the connection stayed open and quiet")
}

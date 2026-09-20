package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
)

// insertOutboxTestRowAt seeds an outbox row with an explicit created_at, so
// ordering and cursor tests can control which rows land on which side of a
// cursor without racing the wall clock.
func insertOutboxTestRowAt(t *testing.T, s *Store, id, topic string, payload []byte, createdAt time.Time) {
	t.Helper()
	_, err := s.db.Exec(`INSERT INTO outbox (id, topic, payload, created_at) VALUES (?, ?, ?, ?)`,
		id, topic, payload, createdAt.Unix())
	require.NoError(t, err)
}

func TestAutomationEventLogRepo_Latest_EmptyLog_ReturnsNotOK(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, ok, err := s.AutomationEventLog.Latest(context.Background())
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestAutomationEventLogRepo_Latest_ReturnsNewestRow(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	base := time.Now().UTC()
	insertOutboxTestRowAt(t, s, "e1", "ticket.created", []byte(`{}`), base)
	insertOutboxTestRowAt(t, s, "e2", "ticket.created", []byte(`{}`), base.Add(time.Minute))

	got, ok, err := s.AutomationEventLog.Latest(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "e2", got.ID)
}

func TestAutomationEventLogRepo_After_EmptyTopics_ReturnsNil(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	got, err := s.AutomationEventLog.After(context.Background(), nil, automations.Cursor{}, 10)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestAutomationEventLogRepo_After_ZeroCursor_ReturnsFromBeginning(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	base := time.Now().UTC()
	insertOutboxTestRowAt(t, s, "e1", "ticket.created", []byte(`{"n":1}`), base)
	insertOutboxTestRowAt(t, s, "e2", "ticket.created", []byte(`{"n":2}`), base.Add(time.Minute))

	got, err := s.AutomationEventLog.After(context.Background(), []string{"ticket.created"}, automations.Cursor{}, 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "e1", got[0].ID)
	assert.Equal(t, "e2", got[1].ID)
}

func TestAutomationEventLogRepo_After_FiltersByTopic(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	base := time.Now().UTC()
	insertOutboxTestRowAt(t, s, "e1", "ticket.created", []byte(`{}`), base)
	insertOutboxTestRowAt(t, s, "e2", "doc.created", []byte(`{}`), base.Add(time.Minute))

	got, err := s.AutomationEventLog.After(context.Background(), []string{"ticket.created"}, automations.Cursor{}, 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "e1", got[0].ID)
}

func TestAutomationEventLogRepo_After_ExcludesUpToAndIncludingCursor(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	base := time.Now().UTC()
	insertOutboxTestRowAt(t, s, "e1", "ticket.created", []byte(`{}`), base)
	insertOutboxTestRowAt(t, s, "e2", "ticket.created", []byte(`{}`), base.Add(time.Minute))
	insertOutboxTestRowAt(t, s, "e3", "ticket.created", []byte(`{}`), base.Add(2*time.Minute))

	cursor := automations.Cursor{CreatedAt: base.Add(time.Minute), ID: "e2"}
	got, err := s.AutomationEventLog.After(context.Background(), []string{"ticket.created"}, cursor, 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "e3", got[0].ID)
}

func TestAutomationEventLogRepo_After_SameSecond_TiebreaksByID(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	same := time.Now().UTC()
	// Same created_at second, ids in reverse insertion order — the tiebreak
	// must be by id, not insertion order (borrowed-practices ordering rule).
	insertOutboxTestRowAt(t, s, "b", "ticket.created", []byte(`{}`), same)
	insertOutboxTestRowAt(t, s, "a", "ticket.created", []byte(`{}`), same)

	cursor := automations.Cursor{CreatedAt: same, ID: "a"}
	got, err := s.AutomationEventLog.After(context.Background(), []string{"ticket.created"}, cursor, 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "b", got[0].ID)
}

func TestAutomationEventLogRepo_After_RespectsLimit(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	base := time.Now().UTC()
	for i, id := range []string{"e1", "e2", "e3"} {
		insertOutboxTestRowAt(t, s, id, "ticket.created", []byte(`{}`), base.Add(time.Duration(i)*time.Minute))
	}

	got, err := s.AutomationEventLog.After(context.Background(), []string{"ticket.created"}, automations.Cursor{}, 2)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestAutomationEventLogRepo_After_IncludesUnpublishedRows(t *testing.T) {
	// Automations read independently of the relay's fan-out — a row that
	// hasn't been relayed yet still reaches a subscribed automation.
	t.Parallel()
	s := newTestStore(t)
	insertOutboxTestRowAt(t, s, "e1", "ticket.created", []byte(`{}`), time.Now().UTC())

	pending, err := s.Outbox.Unpublished(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	got, err := s.AutomationEventLog.After(context.Background(), []string{"ticket.created"}, automations.Cursor{}, 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "e1", got[0].ID)
}

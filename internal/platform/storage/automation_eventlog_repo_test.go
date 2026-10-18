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

func TestAutomationEventLogRepo_After(t *testing.T) {
	t.Parallel()
	base := time.Now().UTC()
	minute := func(n int) time.Time { return base.Add(time.Duration(n) * time.Minute) }
	type row struct {
		id, topic string
		at        time.Time
	}
	tests := []struct {
		name   string
		rows   []row
		topics []string
		cursor automations.Cursor
		limit  int
		want   []string
	}{
		{"the zero cursor reads from the beginning", []row{{"e1", "ticket.created", minute(0)}, {"e2", "ticket.created", minute(1)}}, []string{"ticket.created"}, automations.Cursor{}, 10, []string{"e1", "e2"}},
		{"only the subscribed topics", []row{{"e1", "ticket.created", minute(0)}, {"e2", "doc.created", minute(1)}}, []string{"ticket.created"}, automations.Cursor{}, 10, []string{"e1"}},
		{"nothing up to and including the cursor", []row{{"e1", "ticket.created", minute(0)}, {"e2", "ticket.created", minute(1)}, {"e3", "ticket.created", minute(2)}}, []string{"ticket.created"}, automations.Cursor{CreatedAt: minute(1), ID: "e2"}, 10, []string{"e3"}},
		// ids in reverse insertion order: the tiebreak inside one second is the id, not the insertion order
		{"rows of the cursor's second tiebreak by id", []row{{"b", "ticket.created", minute(0)}, {"a", "ticket.created", minute(0)}}, []string{"ticket.created"}, automations.Cursor{CreatedAt: minute(0), ID: "a"}, 10, []string{"b"}},
		{"the limit caps the page", []row{{"e1", "ticket.created", minute(0)}, {"e2", "ticket.created", minute(1)}, {"e3", "ticket.created", minute(2)}}, []string{"ticket.created"}, automations.Cursor{}, 2, []string{"e1", "e2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestStore(t)
			for _, r := range tt.rows {
				insertOutboxTestRowAt(t, s, r.id, r.topic, []byte(`{}`), r.at)
			}
			got, err := s.AutomationEventLog.After(t.Context(), tt.topics, tt.cursor, tt.limit)
			require.NoError(t, err)
			ids := make([]string, 0, len(got))
			for _, e := range got {
				ids = append(ids, e.ID)
			}
			assert.Equal(t, tt.want, ids)
		})
	}
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

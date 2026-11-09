package storage

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// insertOutboxTestRow seeds a row directly; production goes through insertOutboxRow in a domain repo's transaction.
func insertOutboxTestRow(t *testing.T, s *Store, id, topic string, payload []byte) {
	t.Helper()
	_, err := s.db.Exec(`INSERT INTO outbox (id, topic, payload) VALUES (?, ?, ?)`, id, topic, payload)
	require.NoError(t, err)
}

func TestOutboxRepo_Pending_ReturnsUnpublishedInOrder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	insertOutboxTestRow(t, s, "evt-1", "doc.created", []byte(`{"a":1}`))
	insertOutboxTestRow(t, s, "evt-2", "ticket.created", []byte(`{"b":2}`))

	got, err := s.Outbox.Unpublished(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "evt-1", got[0].ID)
	assert.Equal(t, "doc.created", got[0].Topic)
	assert.Equal(t, []byte(`{"a":1}`), got[0].Payload)
	assert.Equal(t, "evt-2", got[1].ID)

	first, err := s.Outbox.Unpublished(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, first, 1, "the limit caps the page")
	assert.Equal(t, "evt-1", first[0].ID)
}

func TestOutboxRepo_MarkPublished_RemovesFromPending(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	insertOutboxTestRow(t, s, "evt-1", "doc.created", []byte(`{}`))
	require.NoError(t, s.Outbox.MarkPublished(context.Background(), "evt-1"))

	got, err := s.Outbox.Unpublished(context.Background(), 10)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestInsertOutboxRow_WritesAnEmptyListAsAnArray(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	payload := struct {
		UserIDs []string `json:"user_ids"`
	}{}
	require.NoError(t, s.w.WithTx(context.Background(), s.db, func(tx *sql.Tx) error {
		return insertOutboxRow(context.Background(), tx, "evt-1", "notification.created", payload)
	}))

	got, err := s.Outbox.Unpublished(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.JSONEq(t, `{"user_ids":[]}`, string(got[0].Payload))
}

func TestOutboxRepo_Record_WritesOncePerReceipt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		receipts []string
		want     int
	}{
		{"a repeated receipt writes one row", []string{"hook:d1", "hook:d1"}, 1},
		{"distinct receipts write a row each", []string{"hook:d1", "hook:d2"}, 2},
		{"no receipt always writes", []string{"", ""}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestStore(t)
			for i, receipt := range tt.receipts {
				ev := eventbus.OutboxEvent{ID: "evt-" + strconv.Itoa(i), Topic: "git.pr_opened", Payload: map[string]int{"n": i}}
				require.NoError(t, s.Outbox.Record(t.Context(), receipt, ev))
			}
			got, err := s.Outbox.Unpublished(t.Context(), 10)
			require.NoError(t, err)
			assert.Len(t, got, tt.want)
		})
	}
}

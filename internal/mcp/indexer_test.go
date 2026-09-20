package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestIndexer_HandleDocEvent(t *testing.T) {
	idx := NewIndexer(nil)

	t.Run("valid payload is acked", func(t *testing.T) {
		ev := eventbus.Event{Topic: "doc.created", Payload: json.RawMessage(`{"doc":{"id":"d1"}}`)}
		require.NoError(t, idx.HandleDocEvent(context.Background(), ev))
	})
	t.Run("malformed payload is fatal", func(t *testing.T) {
		ev := eventbus.Event{Topic: "doc.created", Payload: json.RawMessage(`{not json`)}
		err := idx.HandleDocEvent(context.Background(), ev)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("missing doc id is fatal", func(t *testing.T) {
		ev := eventbus.Event{Topic: "doc.updated", Payload: json.RawMessage(`{"doc":{}}`)}
		err := idx.HandleDocEvent(context.Background(), ev)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
}

func TestIndexer_HandleTicketCreated(t *testing.T) {
	idx := NewIndexer(nil)

	t.Run("valid payload is acked", func(t *testing.T) {
		ev := eventbus.Event{Topic: "ticket.created", Payload: json.RawMessage(`{"ticket":{"id":"t1"}}`)}
		require.NoError(t, idx.HandleTicketCreated(context.Background(), ev))
	})
	t.Run("malformed payload is fatal", func(t *testing.T) {
		ev := eventbus.Event{Topic: "ticket.created", Payload: json.RawMessage(`nope`)}
		err := idx.HandleTicketCreated(context.Background(), ev)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("missing ticket id is fatal", func(t *testing.T) {
		ev := eventbus.Event{Topic: "ticket.created", Payload: json.RawMessage(`{"ticket":{}}`)}
		err := idx.HandleTicketCreated(context.Background(), ev)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
}

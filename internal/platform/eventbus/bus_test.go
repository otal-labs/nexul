package eventbus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChain_AppliesOutermostFirst(t *testing.T) {
	var order []string
	mw := func(name string) Middleware {
		return func(h Handler) Handler {
			return func(ctx context.Context, ev Event) error {
				order = append(order, "before-"+name)
				err := h(ctx, ev)
				order = append(order, "after-"+name)
				return err
			}
		}
	}
	h := Chain(func(ctx context.Context, ev Event) error {
		order = append(order, "handler")
		return nil
	}, mw("first"), mw("second"))

	require.NoError(t, h(context.Background(), Event{ID: "e"}))
	assert.Equal(t, []string{
		"before-first", "before-second", "handler", "after-second", "after-first",
	}, order)
}

func TestChain_NoMiddlewareCallsHandlerDirectly(t *testing.T) {
	called := false
	h := Chain(func(ctx context.Context, ev Event) error {
		called = true
		return nil
	})
	require.NoError(t, h(context.Background(), Event{ID: "e"}))
	assert.True(t, called)
}

func TestEvent_JSONRoundTrip(t *testing.T) {
	ev := Event{
		ID:        "ev-1",
		TraceID:   "tr-1",
		Topic:     "order.created",
		Timestamp: time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC),
		Payload:   json.RawMessage(`{"n":1}`),
	}
	data, err := json.Marshal(ev)
	require.NoError(t, err)

	var got Event
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, ev.ID, got.ID)
	assert.Equal(t, ev.TraceID, got.TraceID)
	assert.Equal(t, ev.Topic, got.Topic)
	assert.True(t, ev.Timestamp.Equal(got.Timestamp))
	assert.JSONEq(t, `{"n":1}`, string(got.Payload))
}

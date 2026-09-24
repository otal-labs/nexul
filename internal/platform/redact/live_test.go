package redact

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingPublisher struct {
	payloads []any
}

func (r *recordingPublisher) Publish(_ context.Context, _ string, payload any) error {
	r.payloads = append(r.payloads, payload)
	return nil
}

func TestLive_Publish(t *testing.T) {
	t.Parallel()
	token := "dep_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNO-_"
	inner := &recordingPublisher{}
	live := Live{Publisher: inner}
	type frame struct {
		Text string `json:"text"`
	}

	require.NoError(t, live.Publish(t.Context(), "t", frame{Text: "plain"}))
	require.NoError(t, live.Publish(t.Context(), "t", frame{Text: "bearer " + token}))
	require.Error(t, live.Publish(t.Context(), "t", make(chan int)))

	require.Len(t, inner.payloads, 2, "a frame that cannot be marshalled never leaves")
	assert.Equal(t, frame{Text: "plain"}, inner.payloads[0], "a token-free frame keeps its type")
	raw, ok := inner.payloads[1].(json.RawMessage)
	require.True(t, ok)
	assert.JSONEq(t, `{"text":"bearer `+Placeholder+`"}`, string(raw))
}

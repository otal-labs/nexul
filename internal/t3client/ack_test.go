package t3client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The real server parks each stream on a latch that only an Ack opens
// (RpcServer case "Ack" → latch.open) — found live when a finished T3 turn
// never delivered a single event. This fake mimics the latch: every chunk
// after the first is withheld until the previous one is acked, so a client
// that stops acking hangs this test instead of passing silently.
func TestSubscription_AcksEveryChunkSoTheLatchedStreamKeepsFlowing(t *testing.T) {
	f := newFakeT3(t)
	c := f.connect(t, testCtx(t))

	sub, err := c.SubscribeThread(testCtx(t), "thread-1")
	require.NoError(t, err)
	reqID := waitFor(t, f.subscribed, "subscription")

	// Chunk 1: the initial thread snapshot (skipped by the client, but it
	// must still be acked or nothing else ever arrives).
	f.write(chunk(reqID, map[string]any{"kind": "snapshot", "snapshot": map[string]any{"turns": []any{}}}))
	waitFor(t, f.acks, "ack for the snapshot chunk")

	// Chunk 2: a streaming assistant snapshot, latched behind ack #1.
	f.write(chunk(reqID, messageSent("thread-1", "m-1", "assistant", "hello", true)))
	waitFor(t, f.acks, "ack for the streaming chunk")

	// Chunk 3: final text + settled session, latched behind ack #2.
	f.write(chunk(reqID,
		messageSent("thread-1", "m-1", "assistant", "hello there", false),
		sessionSet("thread-1", "idle", nil, nil),
	))

	snaps, _, _, terminal := collect(t, sub)
	require.Len(t, snaps, 2)
	assert.Equal(t, "hello there", snaps[1].Text)
	require.NotNil(t, terminal)
	assert.Equal(t, TurnDone, terminal.State)
}

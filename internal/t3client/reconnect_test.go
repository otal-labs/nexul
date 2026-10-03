package t3client

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

// sequenced stamps a stream item's event with T3's global event sequence, the cursor a resume replays after.
func sequenced(n int, item map[string]any) map[string]any {
	item["event"].(map[string]any)["sequence"] = n
	return item
}

func namedTool(id, summary, createdAt string) map[string]any {
	return map[string]any{"id": id, "tone": "tool", "kind": "tool.completed", "summary": summary, "payload": map[string]any{}, "createdAt": createdAt}
}

func toolEvent(activity map[string]any) map[string]any {
	return eventItem("thread.activity-appended", map[string]any{"threadId": "th-1", "activity": activity})
}

func snapshotItem(sequence int, messages, activities []map[string]any, status string) map[string]any {
	return map[string]any{"kind": "snapshot", "snapshot": map[string]any{
		"snapshotSequence": sequence,
		"thread": map[string]any{
			"id": "th-1", "deletedAt": nil, "messages": messages, "activities": activities,
			"session": map[string]any{"threadId": "th-1", "status": status, "activeTurnId": nil, "lastError": nil},
		},
	}}
}

var oldReply = map[string]any{"id": "m-old", "role": "assistant", "text": "an earlier turn's reply", "streaming": false, "updatedAt": "2026-10-02T10:00:00Z"}

// startDroppableTurn runs a real Harness turn against the fake up to a half-streamed reply, then drops the socket.
func startDroppableTurn(t *testing.T, f *t3rpctest.Server) <-chan harness.Update {
	t.Helper()
	h := NewHarness(Options{HTTPClient: f.Client(), RPCTimeout: 5 * time.Second})
	result, err := h.StartTurn(testCtx(t), harness.Target{Session: f.Session(), SessionID: "th-1"}, "title", testPrompts())
	require.NoError(t, err)
	subID := t3rpctest.WaitFor(t, f.Subscribed, "first subscribeThread")
	assert.NotContains(t, t3rpctest.WaitFor(t, f.SubscribeIn, "first subscribe input"), "afterSequence", "a fresh watch starts from the snapshot")
	t3rpctest.WaitFor(t, f.Dispatched, "thread.turn.start dispatch")

	f.Write(t3rpctest.Chunk(subID, snapshotItem(10, []map[string]any{oldReply}, nil, "ready")))
	f.Write(t3rpctest.Chunk(subID,
		sequenced(11, sessionSet("th-1", "running", "turn-1", nil)),
		sequenced(12, toolEvent(namedTool("act-a", "Read main.go", "2026-10-02T11:00:01Z"))),
		sequenced(13, messageSent("th-1", "m1", "assistant", "Hel", true)),
	))
	assert.Equal(t, "Read main.go", t3rpctest.WaitFor(t, result.Updates, "first step").Activity.Summary)
	assert.Equal(t, "Hel", t3rpctest.WaitFor(t, result.Updates, "first reply chunk").Snapshot.Text)

	f.Drop()
	note := t3rpctest.WaitFor(t, result.Updates, "reconnect note").Activity
	require.NotNil(t, note)
	assert.Equal(t, harness.Activity{Kind: harness.ActivityNote, Summary: "Reconnecting to T3 Code…", At: note.At}, *note)
	return result.Updates
}

func TestHarness_ConnectionDropsMidTurn_ResumesFromTheLastSequenceWithoutGapsOrDuplicates(t *testing.T) {
	f := t3rpctest.New(t)
	updates := startDroppableTurn(t, f)

	subID := t3rpctest.WaitFor(t, f.Subscribed, "resubscribe after the drop")
	assert.Equal(t, map[string]any{"threadId": "th-1", "afterSequence": float64(13)}, t3rpctest.WaitFor(t, f.SubscribeIn, "resume input"))
	// T3 replays what happened while away; the live tail may overlap the replay, as its own clients expect.
	f.Write(t3rpctest.Chunk(subID,
		sequenced(13, messageSent("th-1", "m1", "assistant", "Hel", true)),
		sequenced(14, toolEvent(namedTool("act-b", "Bash go test", "2026-10-02T11:00:05Z"))),
		sequenced(15, messageSent("th-1", "m1", "assistant", "lo", true)),
	))
	f.Write(t3rpctest.Chunk(subID,
		sequenced(15, messageSent("th-1", "m1", "assistant", "lo", true)),
		sequenced(16, messageSent("th-1", "m1", "assistant", "", false)),
		sequenced(17, sessionSet("th-1", "ready", nil, nil)),
	))

	rest := drain(t, updates)
	require.Len(t, rest, 4, "every event after the drop arrives exactly once: %+v", rest)
	assert.Equal(t, "Bash go test", rest[0].Activity.Summary)
	assert.Equal(t, harness.Snapshot{MessageID: "m1", Text: "Hello", Streaming: true}, *rest[1].Snapshot)
	assert.Equal(t, harness.Snapshot{MessageID: "m1", Text: "Hello", Streaming: false}, *rest[2].Snapshot)
	assert.Equal(t, harness.TurnResult{State: harness.TurnDone}, *rest[3].Terminal)
}

func TestHarness_TurnFinishedWhileDisconnected_TakesTheEndFromTheResumeSnapshot(t *testing.T) {
	f := t3rpctest.New(t)
	updates := startDroppableTurn(t, f)

	subID := t3rpctest.WaitFor(t, f.Subscribed, "resubscribe after the drop")
	t3rpctest.WaitFor(t, f.SubscribeIn, "resume input")
	// Too far behind to replay, T3 answers with the whole thread instead.
	f.Write(t3rpctest.Chunk(subID, snapshotItem(2000,
		[]map[string]any{
			oldReply,
			{"id": "m-user", "role": "user", "text": "do it", "streaming": false, "updatedAt": "2026-10-02T11:00:00Z"},
			{"id": "m1", "role": "assistant", "text": "Hello world", "streaming": false, "updatedAt": "2026-10-02T11:00:09Z"},
		},
		[]map[string]any{
			namedTool("act-a", "Read main.go", "2026-10-02T11:00:01Z"),
			namedTool("act-b", "Bash go test", "2026-10-02T11:00:05Z"),
		},
		"ready",
	)))

	rest := drain(t, updates)
	require.Len(t, rest, 3, "only what was missed, then the end: %+v", rest)
	assert.Equal(t, "Bash go test", rest[0].Activity.Summary)
	assert.Equal(t, harness.Snapshot{MessageID: "m1", Text: "Hello world", Streaming: false}, *rest[1].Snapshot)
	assert.Equal(t, harness.TurnResult{State: harness.TurnDone}, *rest[2].Terminal)
}

func TestHarness_ReconnectKeepsFailing_EndsTheTurnAfterTheWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dropped := &fakeT3Client{subscription: &fakeSubscription{ch: closedUpdates(), dropped: &turnWatch{}}}
		dials := 0
		h := NewHarness(Options{})
		h.connect = func(context.Context, harness.Session, Options) (rpcConn, error) {
			dials++
			if dials == 1 {
				return dropped, nil
			}
			return nil, errors.New("dial T3 websocket: connection refused")
		}
		start := time.Now()

		result, err := h.StartTurn(t.Context(), harness.Target{SessionID: "th-1"}, "title", testPrompts())
		require.NoError(t, err)
		updates := drain(t, result.Updates)

		require.Len(t, updates, 2)
		assert.Equal(t, harness.ActivityNote, updates[0].Activity.Kind)
		assert.Equal(t, harness.TurnResult{State: harness.TurnError,
			LastError: "Lost the connection to T3 Code and couldn't reconnect for 5m: dial T3 websocket: connection refused"}, *updates[1].Terminal)
		assert.Equal(t, reconnectWindow, time.Since(start), "gives up when the window closes, not before")
		assert.Greater(t, dials, 3, "keeps redialing through the window")
		assert.Less(t, dials, 30, "backs off instead of spinning")
	})
}

func closedUpdates() chan Update {
	ch := make(chan Update)
	close(ch)
	return ch
}

func TestHarness_ReconnectFindsT3Updated_EndsTheTurnAtOnce(t *testing.T) {
	t.Parallel()
	for _, protocol := range []int{2, 3} {
		t.Run(fmt.Sprintf("426 naming protocol %d", protocol), func(t *testing.T) {
			t.Parallel()
			f := t3rpctest.New(t)
			f.Protocol = protocol
			dropped := &fakeT3Client{subscription: &fakeSubscription{ch: closedUpdates(), dropped: &turnWatch{}}}
			dials := 0
			h := NewHarness(Options{HTTPClient: f.Client(), RPCTimeout: 5 * time.Second})
			h.connect = func(ctx context.Context, s harness.Session, opts Options) (rpcConn, error) {
				dials++
				if dials == 1 {
					return dropped, nil
				}
				return connect(ctx, s, opts)
			}

			result, err := h.StartTurn(testCtx(t), harness.Target{Session: f.Session(), SessionID: "th-1"}, "title", testPrompts())
			require.NoError(t, err)
			updates := drain(t, result.Updates)

			require.Len(t, updates, 2, "the reconnect note, then the end: %+v", updates)
			assert.Equal(t, reconnectingNote, updates[0].Activity.Summary)
			assert.Equal(t, harness.TurnResult{State: harness.TurnError, LastError: "T3 Code was updated during this turn; ask again"}, *updates[1].Terminal)
			assert.Equal(t, 2, dials, "no redial: the protocol-1 watch cannot resume on another protocol")
		})
	}
}

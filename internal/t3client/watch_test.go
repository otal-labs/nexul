package t3client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

// watchAsync runs Watch beside the test, which has to answer the subscribe before Watch can return.
func watchAsync(t *testing.T, f *t3rpctest.Server) (<-chan harness.StartResult, <-chan error) {
	t.Helper()
	h := NewHarness(Options{HTTPClient: f.Client(), RPCTimeout: 5 * time.Second})
	started, failed := make(chan harness.StartResult, 1), make(chan error, 1)
	go func() {
		result, err := h.Watch(testCtx(t), harness.Target{Session: f.Session(), SessionID: "th-1"})
		if err != nil {
			failed <- err
			return
		}
		started <- result
	}()
	return started, failed
}

// withSession sets the snapshot's session status, active turn, and last error.
func withSession(item map[string]any, status string, activeTurnID, lastError any) map[string]any {
	session := item["snapshot"].(map[string]any)["thread"].(map[string]any)["session"].(map[string]any)
	session["status"], session["activeTurnId"], session["lastError"] = status, activeTurnID, lastError
	return item
}

func reply(id, turnID, text string, streaming bool) map[string]any {
	return map[string]any{"id": id, "role": "assistant", "text": text, "streaming": streaming, "turnId": turnID, "updatedAt": "2026-10-04T08:00:05Z"}
}

func TestHarness_Watch_EmptySessionID_IsInvalid(t *testing.T) {
	fake := &fakeT3Client{}
	_, err := harnessWithFake(fake).Watch(t.Context(), harness.Target{})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Equal(t, 0, fake.subscribeCalls)
}

func TestHarness_Watch_DeletedThread_FailsWithoutSendingAnything(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	_, failed := watchAsync(t, f)
	deleted := snapshotItem(5, []map[string]any{oldReply}, nil, "idle")
	deleted["snapshot"].(map[string]any)["thread"].(map[string]any)["deletedAt"] = "2026-10-04T07:00:00Z"
	f.Write(t3rpctest.Chunk(t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"), deleted))

	err := t3rpctest.WaitFor(t, failed, "Watch error")
	assert.ErrorContains(t, err, "the thread was deleted in T3 Code")
	assert.Empty(t, f.Dispatched)
}

func TestHarness_Watch_NeverStartsATurn(t *testing.T) {
	fake := &fakeT3Client{subscription: newFakeSubscription(Update{Terminal: &TurnResult{State: TurnDone}})}
	result, err := harnessWithFake(fake).Watch(t.Context(), harness.Target{SessionID: "th-1"})
	require.NoError(t, err)
	drain(t, result.Updates)

	assert.Equal(t, harness.StartResult{SessionID: "th-1", Updates: result.Updates, PromptSent: false}, result)
	assert.Equal(t, 0, fake.startTurnCalls)
	assert.Equal(t, 0, fake.createThreadCalls)
	assert.True(t, fake.closed)
}

func TestHarness_Watch_IdleThread_EndsAtOnce(t *testing.T) {
	t.Parallel()
	userMsg := map[string]any{"id": "m-user", "role": "user", "text": "do it", "streaming": false, "updatedAt": "2026-10-04T08:00:00Z"}
	tests := []struct {
		name     string
		snapshot map[string]any
		want     []harness.Update
	}{
		{"settled with a reply ends done on that reply",
			snapshotItem(10, []map[string]any{oldReply, userMsg, reply("m1", "turn-1", "All done", false)}, []map[string]any{namedTool("act-a", "Read main.go", "2026-10-04T08:00:01Z")}, "ready"),
			[]harness.Update{{Snapshot: &harness.Snapshot{MessageID: "m1", Text: "All done"}}, {Terminal: &harness.TurnResult{State: harness.TurnDone}}}},
		{"no session and no reply ends done",
			snapshotItem(10, nil, nil, "idle"),
			[]harness.Update{{Terminal: &harness.TurnResult{State: harness.TurnDone}}}},
		{"a session that errored ends with its error",
			withSession(snapshotItem(10, []map[string]any{oldReply}, nil, ""), "error", nil, "provider exploded"),
			[]harness.Update{{Terminal: &harness.TurnResult{State: harness.TurnError, LastError: "provider exploded"}}}},
		{"an interrupted session ends interrupted",
			withSession(snapshotItem(10, []map[string]any{oldReply}, nil, ""), "interrupted", nil, nil),
			[]harness.Update{{Terminal: &harness.TurnResult{State: harness.TurnInterrupted}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := t3rpctest.New(t)
			started, _ := watchAsync(t, f)
			f.Write(t3rpctest.Chunk(t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"), tt.snapshot))

			assert.Equal(t, tt.want, drain(t, t3rpctest.WaitFor(t, started, "Watch").Updates))
			assert.Empty(t, f.Dispatched)
		})
	}
}

func TestHarness_Watch_TurnMidRun_FollowsItToTheEnd(t *testing.T) {
	t.Parallel()
	before := []map[string]any{namedTool("act-a", "Read main.go", "2026-10-04T08:00:01Z")}
	tests := []struct {
		name     string
		snapshot map[string]any
		events   []any
		want     []harness.Update
	}{
		{"a streaming reply carries its text over",
			withSession(snapshotItem(10, []map[string]any{oldReply, reply("m1", "turn-1", "Hel", true)}, before, ""), "running", "turn-1", nil),
			[]any{
				sequenced(11, toolEvent(namedTool("act-b", "Bash go test", "2026-10-04T08:00:06Z"))),
				sequenced(12, messageSent("th-1", "m1", "assistant", "lo", true)),
				sequenced(13, messageSent("th-1", "m1", "assistant", "", false)),
				sequenced(14, sessionSet("th-1", "ready", nil, nil)),
			},
			[]harness.Update{
				{Snapshot: &harness.Snapshot{MessageID: "m1", Text: "Hel", Streaming: true}},
				{Activity: &harness.Activity{Kind: harness.ActivityToolResult, Summary: "Bash go test", At: time.Date(2026, 10, 4, 8, 0, 6, 0, time.UTC)}},
				{Snapshot: &harness.Snapshot{MessageID: "m1", Text: "Hello", Streaming: true}},
				{Snapshot: &harness.Snapshot{MessageID: "m1", Text: "Hello", Streaming: false}},
				{Terminal: &harness.TurnResult{State: harness.TurnDone}},
			}},
		{"a reply streaming past a settled session still ends it",
			snapshotItem(10, []map[string]any{oldReply, reply("m1", "turn-1", "Hel", true)}, before, "ready"),
			[]any{
				sequenced(11, messageSent("th-1", "m1", "assistant", "lo", true)),
				sequenced(12, messageSent("th-1", "m1", "assistant", "", false)),
			},
			[]harness.Update{
				{Snapshot: &harness.Snapshot{MessageID: "m1", Text: "Hel", Streaming: true}},
				{Snapshot: &harness.Snapshot{MessageID: "m1", Text: "Hello", Streaming: true}},
				{Snapshot: &harness.Snapshot{MessageID: "m1", Text: "Hello", Streaming: false}},
				{Terminal: &harness.TurnResult{State: harness.TurnDone}},
			}},
		{"a reply closed before the session settled ends on the settle",
			withSession(snapshotItem(10, []map[string]any{oldReply, reply("m1", "turn-1", "Hello", false)}, before, ""), "running", "turn-1", nil),
			[]any{sequenced(11, sessionSet("th-1", "ready", nil, nil))},
			[]harness.Update{
				{Snapshot: &harness.Snapshot{MessageID: "m1", Text: "Hello", Streaming: false}},
				{Terminal: &harness.TurnResult{State: harness.TurnDone}},
			}},
		{"a turn that errors ends with its error",
			withSession(snapshotItem(10, []map[string]any{oldReply}, before, ""), "running", "turn-1", nil),
			[]any{sequenced(11, sessionSet("th-1", "error", nil, "provider exploded"))},
			[]harness.Update{{Terminal: &harness.TurnResult{State: harness.TurnError, LastError: "provider exploded"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := t3rpctest.New(t)
			started, _ := watchAsync(t, f)
			subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
			f.Write(t3rpctest.Chunk(subID, tt.snapshot))
			result := t3rpctest.WaitFor(t, started, "Watch")
			// Events at or before the snapshot are already in it, so a replayed one is dropped.
			f.Write(t3rpctest.Chunk(subID, sequenced(10, toolEvent(namedTool("act-a", "Read main.go", "2026-10-04T08:00:01Z")))))
			f.Write(t3rpctest.Chunk(subID, tt.events...))

			assert.Equal(t, tt.want, drain(t, result.Updates))
			assert.False(t, result.PromptSent)
			assert.Empty(t, f.Dispatched)
		})
	}
}

func TestHarness_Watch_ConnectionDropsMidFollow_ResumesAfterTheSnapshot(t *testing.T) {
	f := t3rpctest.New(t)
	started, _ := watchAsync(t, f)
	f.Write(t3rpctest.Chunk(t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"),
		withSession(snapshotItem(10, []map[string]any{reply("m1", "turn-1", "Hel", true)}, nil, ""), "running", "turn-1", nil)))
	updates := t3rpctest.WaitFor(t, started, "Watch").Updates
	assert.Equal(t, "Hel", t3rpctest.WaitFor(t, updates, "reply so far").Snapshot.Text)

	f.Drop()
	assert.Equal(t, reconnectingNote, t3rpctest.WaitFor(t, updates, "reconnect note").Activity.Summary)
	subID := t3rpctest.WaitFor(t, f.Subscribed, "resubscribe after the drop")
	t3rpctest.WaitFor(t, f.SubscribeIn, "first subscribe input")
	assert.Equal(t, map[string]any{"threadId": "th-1", "afterSequence": float64(10)}, t3rpctest.WaitFor(t, f.SubscribeIn, "resume input"))
	f.Write(t3rpctest.Chunk(subID,
		sequenced(11, messageSent("th-1", "m1", "assistant", "lo", true)),
		sequenced(12, messageSent("th-1", "m1", "assistant", "", false)),
		sequenced(13, sessionSet("th-1", "ready", nil, nil)),
	))

	assert.Equal(t, []harness.Update{
		{Snapshot: &harness.Snapshot{MessageID: "m1", Text: "Hello", Streaming: true}},
		{Snapshot: &harness.Snapshot{MessageID: "m1", Text: "Hello", Streaming: false}},
		{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	}, drain(t, updates))
	assert.Empty(t, f.Dispatched)
}

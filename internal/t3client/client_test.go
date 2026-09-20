package t3client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func testCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestStartTurn_EncodesAttachmentsAsDataURLs(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	ctx := testCtx(t)
	c := f.connect(t, ctx)

	png := []byte{0x89, 'P', 'N', 'G', 0, 1, 2}
	require.NoError(t, c.StartTurn(ctx, "th-1", "look at this", "", []harness.Attachment{{Name: "shot.png", MIME: "image/png", Bytes: png}}))
	cmd := waitFor(t, f.dispatched, "thread.turn.start dispatch")
	msg, ok := cmd["message"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{map[string]any{
		"type":      "image",
		"name":      "shot.png",
		"mimeType":  "image/png",
		"sizeBytes": float64(len(png)),
		"dataUrl":   "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
	}}, msg["attachments"])
}

func TestConnect_HandshakeMintsTicketAndCallsGetConfig(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	f.connect(t, testCtx(t))
	assert.Equal(t, int32(1), f.configCalls.Load())
}

func TestConnect_AcceptsAlternateTicketFieldName(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	f.ticketField = "wsTicket"
	f.connect(t, testCtx(t))
	assert.Equal(t, int32(1), f.configCalls.Load())
}

func TestConnect_RejectedBearerIsUnauthorized(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	computer := f.session()
	computer.BearerToken = "wrong"
	_, err := Connect(testCtx(t), computer, Options{HTTPClient: f.srv.Client()})
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestCreateThread_DispatchesThreadCreate(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	ctx := testCtx(t)
	c := f.connect(t, ctx)

	threadID, err := c.CreateThread(ctx, "proj-1", "Chat: #general", "claude-code", "opus-4", "")
	require.NoError(t, err)
	require.NotEmpty(t, threadID)

	cmd := waitFor(t, f.dispatched, "thread.create dispatch")
	assert.Equal(t, "thread.create", cmd["type"])
	assert.Equal(t, threadID, cmd["threadId"])
	assert.Equal(t, "proj-1", cmd["projectId"])
	assert.Equal(t, "Chat: #general", cmd["title"])
	assert.Equal(t, map[string]any{"instanceId": "claude-code", "model": "opus-4"}, cmd["modelSelection"])
	assert.Equal(t, RuntimeModeFullAccess, cmd["runtimeMode"], "empty runtimeMode defaults to full-access")
	assert.Equal(t, "default", cmd["interactionMode"])
	branch, ok := cmd["branch"]
	assert.True(t, ok, "branch must be present as an explicit null")
	assert.Nil(t, branch)
	assert.NotEmpty(t, cmd["commandId"])
	assert.NotEmpty(t, cmd["createdAt"])
}

func TestFullTurn_CumulativeSnapshotsThenDone(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	ctx := testCtx(t)
	c := f.connect(t, ctx)

	sub, err := c.SubscribeThread(ctx, "th-1")
	require.NoError(t, err)
	subID := waitFor(t, f.subscribed, "subscribeThread request")

	require.NoError(t, c.StartTurn(ctx, "th-1", "hello agent", "", nil))
	cmd := waitFor(t, f.dispatched, "thread.turn.start dispatch")
	assert.Equal(t, "thread.turn.start", cmd["type"])
	assert.Equal(t, "th-1", cmd["threadId"])
	msg, ok := cmd["message"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "user", msg["role"])
	assert.Equal(t, "hello agent", msg["text"])
	assert.NotEmpty(t, msg["messageId"])
	assert.Equal(t, []any{}, msg["attachments"])

	// Initial state + snapshot item must not terminate the watch prematurely.
	f.write(chunk(subID, map[string]any{"kind": "snapshot", "snapshot": map[string]any{"thread": map[string]any{"id": "th-1"}}}))
	f.write(chunk(subID, sessionSet("th-1", "ready", nil, nil)))
	f.write(chunk(subID, sessionSet("th-1", "running", "turn-1", nil)))
	// The user message echo is not an assistant snapshot.
	f.write(chunk(subID, messageSent("th-1", "m-user", "user", "hello agent", false)))
	// T3 streams deltas (thread.message.assistant.delta -> message-sent{streaming:true}); the close carries "".
	f.write(chunk(subID,
		messageSent("th-1", "m1", "assistant", "Hel", true),
		messageSent("th-1", "m1", "assistant", "lo wor", true),
	))
	f.write(chunk(subID, toolActivity("th-1", "tool.started", "Read main.go started")))
	f.write(chunk(subID, messageSent("th-1", "m1", "assistant", "ld", true)))
	f.write(chunk(subID, messageSent("th-1", "m1", "assistant", "", false)))
	f.write(chunk(subID, sessionSet("th-1", "idle", nil, nil)))

	snaps, activities, approvals, terminal := collect(t, sub)
	require.Equal(t, []MessageSnapshot{
		{MessageID: "m1", Text: "Hel", Streaming: true},
		{MessageID: "m1", Text: "Hello wor", Streaming: true},
		{MessageID: "m1", Text: "Hello world", Streaming: true},
		{MessageID: "m1", Text: "Hello world", Streaming: false},
	}, snaps)
	require.Len(t, activities, 1)
	assert.Equal(t, harness.ActivityToolCall, activities[0].Kind)
	assert.Equal(t, "Read main.go started", activities[0].Summary, "an empty payload keeps T3's own label")
	assert.Empty(t, approvals)
	require.NotNil(t, terminal)
	assert.Equal(t, TurnDone, terminal.State)
	assert.Empty(t, terminal.LastError)
}

// Captured live 2026-09-03 (opencode, fresh thread): T3 settles the session
// one second into the turn and streams the reply outside it. Settling alone
// must not end the watch, or the reply never reaches chat.
func TestFullTurn_SessionSettlesBeforeReply_WaitsForReplyClose(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	ctx := testCtx(t)
	c := f.connect(t, ctx)

	sub, err := c.SubscribeThread(ctx, "th-1")
	require.NoError(t, err)
	subID := waitFor(t, f.subscribed, "subscribeThread request")
	require.NoError(t, c.StartTurn(ctx, "th-1", "hello agent", "", nil))
	waitFor(t, f.dispatched, "thread.turn.start dispatch")

	f.write(chunk(subID, sessionSet("th-1", "starting", nil, nil)))
	f.write(chunk(subID, sessionSet("th-1", "running", "turn-1", nil)))
	f.write(chunk(subID, sessionSet("th-1", "ready", nil, nil)))
	f.write(chunk(subID, messageSent("th-1", "m1", "assistant", "Hello world", true)))
	f.write(chunk(subID, messageSent("th-1", "m1", "assistant", "", false)))

	snaps, _, _, terminal := collect(t, sub)
	require.Equal(t, []MessageSnapshot{
		{MessageID: "m1", Text: "Hello world", Streaming: true},
		{MessageID: "m1", Text: "Hello world", Streaming: false},
	}, snaps)
	require.NotNil(t, terminal)
	assert.Equal(t, TurnDone, terminal.State)
}

func TestRespondUserInput_DispatchesTheAnswersInT3sShape(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	ctx := testCtx(t)
	c := f.connect(t, ctx)

	require.NoError(t, c.RespondUserInput(ctx, "th-1", "req-1", harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{
		"one":   {Selected: []string{"Yes"}},
		"many":  {Selected: []string{"A", "B"}},
		"typed": {Selected: []string{"ignored"}, Text: "something else"},
	}}))
	cmd := waitFor(t, f.dispatched, "thread.user-input.respond dispatch")
	assert.Equal(t, "thread.user-input.respond", cmd["type"])
	assert.Equal(t, "th-1", cmd["threadId"])
	assert.Equal(t, "req-1", cmd["requestId"])
	assert.NotEmpty(t, cmd["commandId"])
	assert.NotEmpty(t, cmd["createdAt"])
	assert.Equal(t, map[string]any{"one": "Yes", "many": []any{"A", "B"}, "typed": "something else"}, cmd["answers"],
		"free text wins, one pick is a string, several are an array")
}

func TestInterrupt_DispatchesAndTerminalIsInterrupted(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	ctx := testCtx(t)
	c := f.connect(t, ctx)

	sub, err := c.SubscribeThread(ctx, "th-1")
	require.NoError(t, err)
	subID := waitFor(t, f.subscribed, "subscribeThread request")

	require.NoError(t, c.Interrupt(ctx, "th-1"))
	cmd := waitFor(t, f.dispatched, "thread.turn.interrupt dispatch")
	assert.Equal(t, "thread.turn.interrupt", cmd["type"])
	assert.Equal(t, "th-1", cmd["threadId"])
	_, hasTurnID := cmd["turnId"]
	assert.False(t, hasTurnID, "turnId omitted = interrupt whatever's active (ticket 03)")

	f.write(chunk(subID, sessionSet("th-1", "interrupted", nil, nil)))
	_, _, _, terminal := collect(t, sub)
	require.NotNil(t, terminal)
	assert.Equal(t, TurnInterrupted, terminal.State)
}

func approvalActivity(threadID, activityID string, payload any) map[string]any {
	return eventItem("thread.activity-appended", map[string]any{
		"threadId": threadID,
		"activity": map[string]any{
			"id": activityID, "tone": "approval", "kind": "command",
			"summary": "Run `rm -rf /tmp/x`?", "payload": payload, "turnId": "turn-1",
		},
	})
}

func TestApproval_SurfacedAndAutoDeclined(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	ctx := testCtx(t)
	c := f.connect(t, ctx)

	sub, err := c.SubscribeThread(ctx, "th-1")
	require.NoError(t, err)
	subID := waitFor(t, f.subscribed, "subscribeThread request")

	// The upstream approval payload shape is unknown: one plausible shape with
	// a requestId, one hostile shape without — neither may crash the client.
	f.write(chunk(subID, approvalActivity("th-1", "act-1", map[string]any{"requestId": "apr-1", "command": "rm -rf /tmp/x"})))
	f.write(chunk(subID, approvalActivity("th-1", "act-2", "some opaque string payload")))

	first := waitFor(t, sub.Updates(), "first approval update")
	require.NotNil(t, first.Approval)
	assert.Equal(t, "apr-1", first.Approval.RequestID)
	assert.Equal(t, "command", first.Approval.Kind)

	second := waitFor(t, sub.Updates(), "second approval update")
	require.NotNil(t, second.Approval)
	assert.Equal(t, "act-2", second.Approval.RequestID, "no requestId in payload falls back to the activity id")

	require.NoError(t, c.RespondApproval(ctx, "th-1", first.Approval.RequestID, DecisionDecline))
	cmd := waitFor(t, f.dispatched, "thread.approval.respond dispatch")
	assert.Equal(t, "thread.approval.respond", cmd["type"])
	assert.Equal(t, "th-1", cmd["threadId"])
	assert.Equal(t, "apr-1", cmd["requestId"])
	assert.Equal(t, "decline", cmd["decision"])

	f.write(chunk(subID, sessionSet("th-1", "error", nil, "declined and gave up")))
	_, _, _, terminal := collect(t, sub)
	require.NotNil(t, terminal)
	assert.Equal(t, TurnError, terminal.State)
}

func TestErrorSession_TerminalCarriesLastError(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	ctx := testCtx(t)
	c := f.connect(t, ctx)

	sub, err := c.SubscribeThread(ctx, "th-1")
	require.NoError(t, err)
	subID := waitFor(t, f.subscribed, "subscribeThread request")

	f.write(chunk(subID, sessionSet("th-1", "error", nil, "provider exploded")))
	snaps, _, _, terminal := collect(t, sub)
	assert.Empty(t, snaps)
	require.NotNil(t, terminal)
	assert.Equal(t, TurnError, terminal.State)
	assert.Equal(t, "provider exploded", terminal.LastError)
}

func TestMalformedAndUnknownFrames_AreSkippedNotFatal(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	ctx := testCtx(t)
	c := f.connect(t, ctx)

	sub, err := c.SubscribeThread(ctx, "th-1")
	require.NoError(t, err)
	subID := waitFor(t, f.subscribed, "subscribeThread request")

	// Garbage and drift, all of which the protocol WILL eventually produce.
	f.writeRaw(`{this is not json`)
	f.writeRaw(`"just a string frame"`)
	f.write(map[string]any{"_tag": "SomeFutureFrame", "requestId": subID})
	f.write(chunk(subID, map[string]any{"kind": "mystery", "data": 42}))
	f.write(chunk(subID, eventItem("thread.pinned", map[string]any{"threadId": "th-1"})))
	f.write(chunk(subID, map[string]any{"kind": "event", "event": map[string]any{"payload": map[string]any{}}}))
	// A batched frame (JSON array) must also decode.
	batch, err := json.Marshal([]any{chunk(subID, messageSent("th-1", "m1", "assistant", "still alive", false))})
	require.NoError(t, err)
	f.writeRaw(string(batch))
	f.write(chunk(subID, sessionSet("th-1", "idle", nil, nil)))

	snaps, _, _, terminal := collect(t, sub)
	require.Equal(t, []MessageSnapshot{{MessageID: "m1", Text: "still alive", Streaming: false}}, snaps)
	require.NotNil(t, terminal)
	assert.Equal(t, TurnDone, terminal.State)

	// The connection survived all of it: a follow-up RPC still round-trips.
	require.NoError(t, c.Interrupt(ctx, "th-1"))
	waitFor(t, f.dispatched, "post-garbage dispatch")
}

func TestVersion_ProbesWellKnownEndpoint(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	version, err := NewHarness(Options{HTTPClient: f.srv.Client()}).Version(testCtx(t), f.srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "0.0.34", version)
}

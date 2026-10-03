package t3clientv2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
)

const (
	nightly      = ".0.0.46-nightly.20261003.2632.ndjson"
	fromSource   = "run-waits-then-completes.source-ce90eec1ff.ndjson"
	codexMessage = "message:provider:codex:native-item:native-1"
	claudeAuth   = "Claude could not authenticate. For subscription login, run `claude auth login` on this environment's machine, then start a new thread. For API-key authentication, check this instance's configured credentials."
)

// recorded is every item a fixture's subscribeThread streams delivered, in the order they arrived.
func recorded(t *testing.T, fixture string) []json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", fixture))
	require.NoError(t, err)
	streams := map[string]bool{}
	var items []json.RawMessage
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
		var rec struct {
			Dir   string `json:"dir"`
			Frame struct {
				Tag       string            `json:"_tag"`
				ID        string            `json:"id"`
				RPC       string            `json:"tag"`
				RequestID string            `json:"requestId"`
				Values    []json.RawMessage `json:"values"`
			} `json:"frame"`
		}
		require.NoError(t, json.Unmarshal(line, &rec))
		if rec.Dir == "send" && rec.Frame.RPC == subscribeThread {
			streams[rec.Frame.ID] = true
		}
		if rec.Dir == "recv" && rec.Frame.Tag == "Chunk" && streams[rec.Frame.RequestID] {
			items = append(items, rec.Frame.Values...)
		}
	}
	return items
}

// fold runs items through a watch for messageID and returns what it emitted, the terminal result as the last update.
func fold(t *testing.T, w *watch, items []json.RawMessage) []harness.Update {
	t.Helper()
	var out []harness.Update
	for _, raw := range items {
		var item streamItem
		require.NoError(t, json.Unmarshal(raw, &item))
		updates, end := w.apply(item)
		out = append(out, updates...)
		if end != nil {
			out = append(out, harness.Update{Terminal: end})
		}
	}
	return out
}

func snapshotOf(messageID, text string, streaming bool) harness.Update {
	return harness.Update{Snapshot: &harness.Snapshot{MessageID: messageID, Text: text, Streaming: streaming}}
}

func ended(state harness.TurnState, lastError string) harness.Update {
	return harness.Update{Terminal: &harness.TurnResult{State: state, LastError: lastError}}
}

// rewriteRunStatus maps the status every run update carries through edit; edit returns "" to leave the update out.
func rewriteRunStatus(edit func(status string) string) func([]json.RawMessage) []json.RawMessage {
	return func(items []json.RawMessage) []json.RawMessage {
		var out []json.RawMessage
		for _, raw := range items {
			var item map[string]any
			if err := json.Unmarshal(raw, &item); err != nil {
				panic(err)
			}
			event, _ := item["event"].(map[string]any)
			if event == nil || event["type"] != "run.updated" {
				out = append(out, raw)
				continue
			}
			payload := event["payload"].(map[string]any)
			status := edit(payload["status"].(string))
			if status == "" {
				continue
			}
			payload["status"] = status
			b, err := json.Marshal(item)
			if err != nil {
				panic(err)
			}
			out = append(out, b)
		}
		return out
	}
}

// replayOverlap sends items[from:to] again after items[to-1], the way a resume's replay overlaps the live tail.
func replayOverlap(from, to int) func([]json.RawMessage) []json.RawMessage {
	return func(items []json.RawMessage) []json.RawMessage {
		out := append([]json.RawMessage{}, items[:to]...)
		out = append(out, items[from:to]...)
		return append(out, items[to:]...)
	}
}

func TestWatch_RecordedTurns(t *testing.T) {
	t.Parallel()
	hello := []harness.Update{
		snapshotOf(codexMessage, "Hel", true),
		snapshotOf(codexMessage, "Hello", true),
		snapshotOf(codexMessage, "Hello there.", false),
	}
	tests := []struct {
		name      string
		fixture   string
		messageID string
		// after starts the watch the way a resume does, from a cursor with no snapshot.
		after int64
		edit  func([]json.RawMessage) []json.RawMessage
		want  []harness.Update
	}{
		{name: "done at waiting, once: the checkpoint, completion and delivery updates after it emit nothing",
			fixture: fromSource, messageID: "msg-1",
			want: append(hello, ended(harness.TurnDone, ""))},
		{name: "done at completed when waiting was missed", fixture: fromSource, messageID: "msg-1",
			edit: rewriteRunStatus(func(s string) string {
				if s == runWaiting {
					return ""
				}
				return s
			}),
			want: append(hello, ended(harness.TurnDone, ""))},
		{name: "a rolled back run is interrupted", fixture: fromSource, messageID: "msg-1",
			edit: rewriteRunStatus(func(s string) string {
				if s == runWaiting {
					return "rolled_back"
				}
				return s
			}),
			want: append(hello, ended(harness.TurnInterrupted, ""))},
		{name: "a replay overlapping what was seen does not rewind the reply", fixture: fromSource, messageID: "msg-1",
			edit: replayOverlap(5, 7),
			want: append(hello, ended(harness.TurnDone, ""))},
		{name: "a failed run's reply is its root error, not the assistant text it streamed",
			fixture: "dispatch-provider-unauthenticated" + nightly, messageID: "msg-1",
			want: []harness.Update{
				snapshotOf("message:provider:claudeAgent:native-item:native-2", "Not logged in · Please run /login", false),
				ended(harness.TurnError, claudeAuth),
			}},
		{name: "a failed start names the root error", fixture: "dispatch-provider-missing" + nightly, messageID: "msg-1",
			want: []harness.Update{ended(harness.TurnError, "Provider turn failed.")}},
		{name: "a queued run cancelled in T3 is interrupted", fixture: "queued-run-cancel" + nightly, messageID: "msg-2",
			want: []harness.Update{ended(harness.TurnInterrupted, "")}},
		{name: "a resume replays events without a snapshot", fixture: "subscribe-resume" + nightly, messageID: "msg-1", after: 51,
			want: []harness.Update{
				snapshotOf("message:provider:claudeAgent:native-item:native-2", "Not logged in · Please run /login", false),
				ended(harness.TurnError, claudeAuth),
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			items := recorded(t, tt.fixture)
			if tt.edit != nil {
				items = tt.edit(items)
			}
			w := newWatch(tt.messageID)
			w.cursor = tt.after
			assert.Equal(t, tt.want, fold(t, w, items))
		})
	}
}

func TestWatch_AnotherRunCancelledMidTurn_IsNotTheTurnsEnd(t *testing.T) {
	t.Parallel()
	items := recorded(t, "queued-run-cancel"+nightly)
	// msg-2's queued run is cancelled at sequence 64, while msg-1's own run is still preparing.
	split := slices.IndexFunc(items, func(raw json.RawMessage) bool {
		var item streamItem
		require.NoError(t, json.Unmarshal(raw, &item))
		return item.Sequence > 64
	})
	w := newWatch("msg-1")
	ends := func(u harness.Update) bool { return u.Terminal != nil }
	assert.False(t, slices.ContainsFunc(fold(t, w, items[:split]), ends), "msg-2's cancel ends nothing")
	after := fold(t, w, items[split:])
	require.NotEmpty(t, after)
	assert.Equal(t, ended(harness.TurnInterrupted, ""), after[len(after)-1], "msg-1's own interrupt does")
}

func TestWatch_RecordedToolSteps_ShowEachChangeOfTheTurnsOwnItems(t *testing.T) {
	t.Parallel()
	label := func(u harness.Update) string {
		switch {
		case u.Activity != nil:
			return string(u.Activity.Kind) + " " + strings.TrimPrefix(u.Activity.CallID, claudeItem)
		case u.Question != nil:
			return "asks " + u.Question.RequestID
		case u.Approval != nil:
			return "approval " + u.Approval.Summary
		case u.Snapshot != nil:
			return "reply " + u.Snapshot.Text
		}
		return "end " + string(u.Terminal.State)
	}
	var got []string
	for _, u := range fold(t, newWatch("msg-1"), recorded(t, toolSteps)) {
		got = append(got, label(u))
	}
	assert.Equal(t, []string{
		"tool_call native-1", "tool_result native-1", "tool_result native-2", "tool_result native-3", "tool_result native-4",
		"tool_result native-5", "tool_result native-6", "question native-7", "asks " + question1, "question native-7",
		"approval " + approvalPrompt, "tool_call native-8", "tool_result native-8", "reply Done.", "end done",
	}, got, "the answered question and the declined approval are not raised again, and the runs typed in T3 show nothing")
}

func TestWatch_ItemSentAgain_EmitsOnce(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		item turnItem
	}{
		{"a step", recordedItem(t, toolSteps, claudeItem+"native-1", "running")},
		{"a question", recordedItem(t, toolSteps, questionItem, "waiting")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			it := tt.item
			it.RunID = runOne
			resubscribed := snapshotItem(9, map[string]any{"thread": map[string]any{"id": "th-1", "runtimeMode": "full-access"},
				"runs": []any{runOf("msg-1", "running")}, "turnItems": []any{it}, "providerSessions": []any{}})
			got := fold(t, newWatch("msg-1"), []json.RawMessage{
				event(2, "run.created", runOf("msg-1", "running")),
				event(3, "turn-item.updated", it), event(4, "turn-item.updated", it), resubscribed,
			})
			assert.Len(t, got, 1, "re-sent by T3 and again by a resubscribe's snapshot")
		})
	}
}

// event is a fixture-shaped event item; the payload keys follow the captures.
func event(seq int, typ string, payload any) json.RawMessage {
	b, err := json.Marshal(map[string]any{"kind": "event", "sequence": seq, "event": map[string]any{
		"id": "ev-x", "threadId": "th-1", "occurredAt": "2026-10-03T16:00:00.000Z", "type": typ, "payload": payload}})
	if err != nil {
		panic(err)
	}
	return b
}

func snapshotItem(seq int, projection map[string]any) json.RawMessage {
	b, err := json.Marshal(map[string]any{"kind": "snapshot", "snapshotSequence": seq, "projection": projection,
		"historyCursor": nil, "hasMoreHistory": false, "latestLocalTurnOrdinal": nil, "payloadBudgetExceeded": false})
	if err != nil {
		panic(err)
	}
	return b
}

const (
	runOne  = "run:thread:th-1:ordinal:1"
	rootOne = "node:run:run%3Athread%3Ath-1%3Aordinal%3A1:root"
)

func runOf(messageID, status string) map[string]any {
	return map[string]any{"id": runOne, "threadId": "th-1", "ordinal": 1, "userMessageId": messageID, "rootNodeId": rootOne,
		"status": status, "queuePosition": nil, "startedAt": nil, "completedAt": nil, "checkpointId": nil}
}

// assistantItem is the turn's reply as T3 sends it: the whole text so far, every time.
func assistantItem(text string, streaming bool) map[string]any {
	status := "completed"
	if streaming {
		status = "running"
	}
	return map[string]any{"id": "turn-item:provider:codex:native-item:native-1", "threadId": "th-1", "runId": runOne,
		"nodeId": "node:provider:codex:native-item:native-1", "ordinal": 2, "status": status, "type": "assistant_message",
		"messageId": codexMessage, "text": text, "streaming": streaming}
}

func errorItem(id, nodeID, status string, failure map[string]any) map[string]any {
	return map[string]any{"id": id, "threadId": "th-1", "runId": runOne, "nodeId": nodeID, "ordinal": 3, "status": status,
		"title": "Provider error", "updatedAt": "2026-10-03T16:00:00.500Z", "type": "error", "failure": failure}
}

func providerFailure(class, message string, resetAt any) map[string]any {
	f := map[string]any{"class": class, "message": message, "code": nil, "retryable": nil}
	if resetAt != nil {
		f["resetAt"] = resetAt
	}
	return f
}

func TestWatch_FailedRun_NamesTheMostSpecificReason(t *testing.T) {
	t.Parallel()
	session := event(3, "provider-session.updated", map[string]any{"id": "ps-1", "status": "error", "lastError": "Codex exited with code 1"})
	tests := []struct {
		name  string
		items []json.RawMessage
		want  string
	}{
		{"the root error wins over the session's", []json.RawMessage{
			session,
			event(4, "turn-item.updated", errorItem("e-1", rootOne, "failed", providerFailure("provider_error", "Model overloaded", nil))),
		}, "Model overloaded"},
		{"a usage limit names when it resets", []json.RawMessage{
			event(4, "turn-item.updated", errorItem("e-1", rootOne, "failed", providerFailure("usage_limit", "You've hit your usage limit", "2026-10-03T18:00:00.000Z"))),
		}, "You've hit your usage limit (the limit resets at 2026-10-03 18:00 UTC)"},
		{"a usage limit without a reset time takes the thread's recovery", []json.RawMessage{
			event(4, "turn-item.updated", errorItem("e-1", rootOne, "failed", providerFailure("usage_limit", "You've hit your usage limit", nil))),
			event(5, "thread.metadata-updated", map[string]any{"id": "th-1", "runtimeMode": "full-access", "deletedAt": nil,
				"limitRecovery": map[string]any{"runId": runOne, "resetAt": "2026-10-04T09:30:00.000Z", "autoResume": false}}),
		}, "You've hit your usage limit (the limit resets at 2026-10-04 09:30 UTC)"},
		{"a running error item is a retry, not the failure", []json.RawMessage{
			session,
			event(4, "turn-item.updated", errorItem("e-1", rootOne, "running", providerFailure("transport_error", "Reconnecting... 1/5", nil))),
		}, "Codex exited with code 1"},
		{"an error on another node is not the run's", []json.RawMessage{
			event(4, "turn-item.updated", errorItem("e-2", "node:provider:codex:native-item:native-9", "failed", providerFailure("provider_error", "Tool crashed", nil))),
		}, noReason},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			items := append([]json.RawMessage{event(2, "run.created", runOf("msg-1", "running"))}, tt.items...)
			items = append(items, event(9, "run.updated", runOf("msg-1", "failed")))
			assert.Equal(t, []harness.Update{ended(harness.TurnError, tt.want)}, fold(t, newWatch("msg-1"), items))
		})
	}
}

func TestWatch_ResubscribeSnapshotOfAWaitingRun_EndsOnce(t *testing.T) {
	t.Parallel()
	snapshot := snapshotItem(40, map[string]any{
		"thread": map[string]any{"id": "th-1", "runtimeMode": "full-access", "deletedAt": nil},
		"runs":   []any{runOf("msg-1", runWaiting)}, "turnItems": []any{assistantItem("Hello there.", false)}, "providerSessions": []any{},
	})
	w := newWatch("msg-1")
	streamed := fold(t, w, []json.RawMessage{
		event(2, "run.created", runOf("msg-1", "running")),
		event(3, "turn-item.updated", assistantItem("Hello", true)),
	})
	require.Equal(t, []harness.Update{snapshotOf(codexMessage, "Hello", true)}, streamed)

	assert.Equal(t, []harness.Update{snapshotOf(codexMessage, "Hello there.", false), ended(harness.TurnDone, "")},
		fold(t, w, []json.RawMessage{snapshot}), "the reply missed while away, then the end")
	assert.Empty(t, fold(t, w, []json.RawMessage{snapshot, event(41, "run.updated", runOf("msg-1", runCompleted))}),
		"a second snapshot and the completion after it change nothing")
}

const wakeMessage = "message:delegated-completion:msg-1:1"

// merged is a fixture-shaped payload with T3's optional keys added, the way they appear only when set.
func merged(payload map[string]any, keys map[string]any) map[string]any {
	maps.Copy(payload, keys)
	return payload
}

// userMessage is a message.updated payload: a user message in run n, carrying whichever link keys T3 set.
func userMessage(id string, n int, keys map[string]any) map[string]any {
	return merged(map[string]any{"id": id, "threadId": "th-1", "runId": fmt.Sprintf("run:thread:th-1:ordinal:%d", n),
		"nodeId": nil, "role": "user", "text": "Delegated task reached a terminal state.", "attachments": []any{}, "streaming": false,
		"createdBy": "agent", "creationSource": "server", "createdAt": "2026-10-03T16:00:00.000Z", "updatedAt": "2026-10-03T16:00:00.000Z"}, keys)
}

// handedOff is a subagent.updated payload for work run 1 handed off; childThreadId is th-2.
func handedOff(id, origin, status string, keys map[string]any) map[string]any {
	return merged(map[string]any{"id": id, "threadId": "th-1", "runId": runOne, "parentNodeId": "node:tool-1", "origin": origin,
		"driver": "claudeAgent", "providerInstanceId": "claudeAgent", "providerThreadId": nil, "childThreadId": "th-2",
		"prompt": "Audit the handlers.", "title": "Audit", "model": nil, "status": status, "result": nil,
		"startedAt": "2026-10-03T16:00:01.000Z", "completedAt": nil, "updatedAt": "2026-10-03T16:00:01.000Z"}, keys)
}

func delivery(state string) map[string]any {
	return map[string]any{"completionDelivery": map[string]any{"state": state, "observedByRunId": nil}}
}

// replyIn is run n's finished assistant message.
func replyIn(n int, text string) map[string]any {
	return map[string]any{"id": fmt.Sprintf("turn-item:provider:claudeAgent:native-item:reply-%d", n), "threadId": "th-1",
		"runId": fmt.Sprintf("run:thread:th-1:ordinal:%d", n), "nodeId": "node:reply", "ordinal": n * 1000, "status": "completed",
		"type": "assistant_message", "messageId": fmt.Sprintf("message:reply-%d", n), "text": text, "streaming": false}
}

// story numbers events from sequence 2, the way one subscription delivers them.
func story(events ...[2]any) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(events))
	for i, e := range events {
		out = append(out, event(i+2, e[0].(string), e[1]))
	}
	return out
}

func labels(updates []harness.Update) []string {
	var got []string
	for _, u := range updates {
		switch {
		case u.Snapshot != nil:
			got = append(got, "reply "+u.Snapshot.Text)
		case u.Activity != nil:
			got = append(got, string(u.Activity.Kind)+" "+u.Activity.Summary)
		case u.Terminal != nil:
			got = append(got, "end "+string(u.Terminal.State))
		}
	}
	return got
}

func TestWatch_HandedOffWork(t *testing.T) {
	t.Parallel()
	// Nexul's run hands an audit to T3's delegate_task and replies before the audit is done.
	handsOff := [][2]any{
		{"run.created", runOf("msg-1", "running")},
		{"subagent.updated", handedOff("task-1", "app_owned", "running", map[string]any{"completionWake": "always"})},
		{"turn-item.updated", replyIn(1, "Handed the audit off.")},
		{"run.updated", runOf("msg-1", runWaiting)},
	}
	linkedToRunOne := map[string]any{"delegatedCompletion": map[string]any{"parentRunId": runOne, "generation": 1, "taskIds": []any{"task-1"}}}
	wakeReplies := [][2]any{
		{"run.updated", runAt(2, wakeMessage, "running")},
		{"turn-item.updated", replyIn(2, "The audit found three issues.")},
		{"subagent.updated", handedOff("task-1", "app_owned", "completed", delivery("delivered"))},
		{"run.updated", runAt(2, wakeMessage, runWaiting)},
	}
	tests := []struct {
		name   string
		events [][2]any
		want   []string
	}{
		{name: "an async delegate's wake joins and the turn ends on the wake's reply",
			events: slices.Concat(handsOff, [][2]any{
				{"subagent.updated", handedOff("task-1", "app_owned", "completed", delivery("pending"))},
				{"run.updated", merged(runOf("msg-1", runCompleted), map[string]any{"delegatedCompletion": map[string]any{
					"disposition": "open", "nextGeneration": 2, "delivery": map[string]any{"generation": 1, "messageId": wakeMessage, "taskIds": []any{"task-1"}}}})},
				{"run.created", runAt(2, wakeMessage, "starting")},
				{"run.updated", runOf("msg-1", runCompleted)},
				{"subagent.updated", handedOff("task-1", "app_owned", "completed", delivery("claimed"))},
			}, wakeReplies),
			want: []string{"reply Handed the audit off.", "reply The audit found three issues.", "end done"}},
		{name: "a queued wake whose run.created precedes its message is adopted once the message links it",
			events: slices.Concat(handsOff, [][2]any{
				{"subagent.updated", handedOff("task-1", "app_owned", "completed", delivery("pending"))},
				{"run.created", runAt(2, wakeMessage, runQueued)},
				{"message.updated", userMessage(wakeMessage, 2, linkedToRunOne)},
			}, wakeReplies),
			want: []string{"reply Handed the audit off.", "reply The audit found three issues.", "end done"}},
		{name: "a subagent update that leaves out its result's delivery keeps that result pending",
			events: slices.Concat(handsOff, [][2]any{
				{"subagent.updated", handedOff("task-1", "app_owned", "completed", delivery("pending"))},
				{"subagent.updated", handedOff("task-1", "app_owned", "completed", map[string]any{"result": "Three issues."})},
				{"run.created", runAt(2, wakeMessage, runQueued)},
				{"message.updated", userMessage(wakeMessage, 2, linkedToRunOne)},
			}, wakeReplies),
			want: []string{"reply Handed the audit off.", "reply The audit found three issues.", "end done"}},
		{name: "a provider's own background subagent joins through its notification wake",
			events: [][2]any{
				{"run.created", runOf("msg-1", "running")},
				{"subagent.updated", handedOff("native-1", "provider_native", "running", nil)},
				{"turn-item.updated", replyIn(1, "Started a background agent.")},
				{"run.updated", runOf("msg-1", runWaiting)},
				{"run.created", runAt(2, "message:provider-continuation:native-9", runQueued)},
				{"message.updated", userMessage("message:provider-continuation:native-9", 2, map[string]any{"creationSource": "provider",
					"notification": map[string]any{"source": map[string]any{"kind": "background_task", "work": "subagent", "childThreadId": "th-2"},
						"outcome": "completed", "summary": "Background agent finished"}})},
				{"run.updated", runAt(2, "message:provider-continuation:native-9", "running")},
				// T3 replays the buffered notification inside the wake run, so the subagent ends only then.
				{"subagent.updated", handedOff("native-1", "provider_native", "completed", nil)},
				{"turn-item.updated", replyIn(2, "The background agent found two callers.")},
				{"run.updated", runAt(2, "message:provider-continuation:native-9", runWaiting)},
			},
			want: []string{"reply Started a background agent.", "reply The background agent found two callers.", "end done"}},
		{name: "a PR-watch wake, a scheduled run, another thread's send, a command's notice and a run typed in T3 are not the turn's",
			events: slices.Concat(handsOff, [][2]any{
				{"run.created", runAt(2, "message:pr-watch:3f2a", "running")},
				{"message.updated", userMessage("message:pr-watch:3f2a", 2, map[string]any{
					"notification": map[string]any{"source": map[string]any{"kind": "monitor"}, "outcome": "updated", "summary": "Checks passed"}})},
				{"turn-item.updated", replyIn(2, "The PR is green.")},
				{"run.updated", runAt(2, "message:pr-watch:3f2a", runWaiting)},
				{"run.created", runAt(3, "scheduled-task-message:st-1:1:cron", "running")},
				{"message.updated", userMessage("scheduled-task-message:st-1:1:cron", 3, map[string]any{"scheduledTaskId": "st-1", "creationSource": "mcp"})},
				{"turn-item.updated", replyIn(3, "Nightly report.")},
				{"run.updated", runAt(3, "scheduled-task-message:st-1:1:cron", runWaiting)},
				{"run.created", runAt(4, "msg-other", "running")},
				{"message.updated", userMessage("msg-other", 4, map[string]any{"senderThreadId": "th-9"})},
				{"turn-item.updated", replyIn(4, "Sent from another thread.")},
				{"run.updated", runAt(4, "msg-other", runWaiting)},
				{"run.created", runAt(5, "message:provider-continuation:native-3", "running")},
				{"message.updated", userMessage("message:provider-continuation:native-3", 5, map[string]any{"creationSource": "provider",
					"notification": map[string]any{"source": map[string]any{"kind": "background_command"}, "outcome": "completed", "summary": "npm test finished"}})},
				{"turn-item.updated", replyIn(5, "Tests passed.")},
				{"run.updated", runAt(5, "message:provider-continuation:native-3", runWaiting)},
				{"run.created", runAt(6, "msg-typed", "running")},
				{"message.updated", userMessage("msg-typed", 6, map[string]any{"createdBy": "user", "creationSource": "web"})},
				{"turn-item.updated", replyIn(6, "Typed in T3.")},
				{"run.updated", runAt(6, "msg-typed", runWaiting)},
				{"subagent.updated", handedOff("task-1", "app_owned", "completed", delivery("disposed"))},
			}),
			want: []string{"reply Handed the audit off.", "end done"}},
		{name: "a result T3 steered into a later run ends the wait with a note",
			events: slices.Concat(handsOff, [][2]any{
				{"subagent.updated", handedOff("task-1", "app_owned", "completed", delivery("pending"))},
				{"run.created", runAt(2, "msg-typed", "running")},
				{"message.updated", userMessage(wakeMessage, 2, linkedToRunOne)},
				{"turn-item.updated", replyIn(2, "Answering the typed message, with the audit.")},
			}),
			want: []string{"reply Handed the audit off.", "note " + steeredNote, "end done"}},
		{name: "a wake T3 continues after a restart joins through the continuation",
			events: slices.Concat(handsOff, [][2]any{
				{"subagent.updated", handedOff("task-1", "app_owned", "completed", delivery("claimed"))},
				{"run.created", runAt(2, wakeMessage, "running")},
				{"message.updated", userMessage(wakeMessage, 2, linkedToRunOne)},
				{"run.updated", runAt(2, wakeMessage, "interrupted")},
				{"run.created", merged(runAt(3, "message:restart-continuation:run-2", runQueued), map[string]any{"restartContinuationOfRunId": runTwo})},
				{"turn-item.updated", replyIn(3, "Picking the audit up again: three issues.")},
				{"subagent.updated", handedOff("task-1", "app_owned", "completed", delivery("delivered"))},
				{"run.updated", runAt(3, "message:restart-continuation:run-2", runWaiting)},
			}),
			want: []string{"reply Handed the audit off.", "reply Picking the audit up again: three issues.", "end done"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, labels(fold(t, newWatch("msg-1"), story(tt.events...))))
		})
	}
}

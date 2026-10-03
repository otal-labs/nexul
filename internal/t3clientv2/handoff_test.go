package t3clientv2

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
)

// childRow is the row of the fixtures' Claude subagent, which handed "Find callers" to th-2, as edit changes it.
func childRow(edit func(*subagent)) subagent {
	row := subagent{ID: "native-8", RunID: runOne, Origin: "provider_native", Status: "running", ChildThreadID: "th-2",
		Driver: "claudeAgent", Title: "Find callers", Prompt: "Find every caller of mapItem.", StartedAt: "2026-10-03T17:00:43.000Z"}
	if edit != nil {
		edit(&row)
	}
	return row
}

func streamItems(t *testing.T, raws []json.RawMessage) []streamItem {
	t.Helper()
	out := make([]streamItem, 0, len(raws))
	for _, raw := range raws {
		var item streamItem
		require.NoError(t, json.Unmarshal(raw, &item))
		out = append(out, item)
	}
	return out
}

// pillOf feeds a pill its row and its thread's items, and returns its last emitted state and the notes it raised.
func pillOf(t *testing.T, row subagent, thread []json.RawMessage) (harness.Handoff, []harness.Update) {
	t.Helper()
	var ps pills
	ps.rows([]subagent{row})
	var notes []harness.Update
	for _, item := range streamItems(t, thread) {
		notes = append(notes, ps.apply(row.ID, item)...)
	}
	changed := ps.changed()
	require.Len(t, changed, 1)
	return *changed[0].Handoff, notes
}

func TestPills_HandedOffAgentsThread(t *testing.T) {
	t.Parallel()
	at := func(second int) time.Time { return time.Date(2026, 10, 3, 17, 0, second, 0, time.UTC) }
	shell := harness.Activity{Kind: harness.ActivityToolResult, CallID: claudeItem + "native-81", Tool: "Shell",
		Summary: "rg -n mapItem internal", Detail: `{"input":"rg -n mapItem internal"}`, At: at(45)}
	childsReply := "Two callers: watch.step and the child map."
	asking := recordedItem(t, toolSteps, questionItem, "waiting")
	tests := []struct {
		name   string
		row    subagent
		thread func(t *testing.T) []json.RawMessage
		want   harness.Handoff
		notes  []string
	}{
		{name: "a provider's own subagent: its thread's items, which carry no run, are its steps; its thread fills in the model and reply",
			row:    childRow(nil),
			thread: func(t *testing.T) []json.RawMessage { return recorded(t, childThread) },
			want: harness.Handoff{ID: "native-8", Driver: "claudeAgent", Model: "claude-opus-5-5", Title: "Find callers",
				Prompt: "Find every caller of mapItem.", State: harness.HandoffRunning, Reply: childsReply, Steps: []harness.Activity{shell}}},
		{name: "the row's own model and result win over its thread's, and a finished subagent is done",
			row: childRow(func(r *subagent) {
				r.Status, r.Model, r.Result = "completed", "claude-sonnet-5", "Found two callers."
			}),
			thread: func(t *testing.T) []json.RawMessage { return recorded(t, childThread) },
			want: harness.Handoff{ID: "native-8", Driver: "claudeAgent", Model: "claude-sonnet-5", Title: "Find callers",
				Prompt: "Find every caller of mapItem.", State: harness.HandoffDone, Reply: "Found two callers.", Steps: []harness.Activity{shell}}},
		{name: "work with no thread of its own is its prompt and result, titled by the prompt's first line",
			row: childRow(func(r *subagent) {
				r.Origin, r.ChildThreadID, r.Title, r.Status = "app_owned", "", "", "completed"
				r.Prompt, r.Result = "  Audit the handlers.\nList what each one checks.", "Three issues."
			}),
			thread: func(*testing.T) []json.RawMessage { return nil },
			want: harness.Handoff{ID: "native-8", Driver: "claudeAgent", Title: "Audit the handlers.",
				Prompt: "  Audit the handlers.\nList what each one checks.", State: harness.HandoffDone, Reply: "Three issues."}},
		{name: "its own hand-offs are Agent steps inside it, one level only",
			row: childRow(nil),
			thread: func(t *testing.T) []json.RawMessage {
				return []json.RawMessage{event(2, "turn-item.updated", recordedItem(t, toolSteps, claudeItem+"native-8", "running"))}
			},
			want: harness.Handoff{ID: "native-8", Driver: "claudeAgent", Title: "Find callers", Prompt: "Find every caller of mapItem.",
				State: harness.HandoffRunning, Steps: []harness.Activity{{Kind: harness.ActivityToolCall, CallID: claudeItem + "native-8",
					Tool: "Agent", Summary: "Find callers", At: at(43)}}}},
		{name: "a question it asks is a step, noted on the turn once while it waits for its answer in T3 Code",
			row: childRow(nil),
			thread: func(t *testing.T) []json.RawMessage {
				return []json.RawMessage{
					event(2, "turn-item.updated", asking),
					event(3, "turn-item.updated", asking),
					event(4, "turn-item.updated", recordedItem(t, toolSteps, questionItem, "completed")),
				}
			},
			want: harness.Handoff{ID: "native-8", Driver: "claudeAgent", Title: "Find callers", Prompt: "Find every caller of mapItem.",
				State: harness.HandoffRunning, Steps: []harness.Activity{{Kind: harness.ActivityQuestion, CallID: questionItem,
					Summary: "Which database should the cache use?", At: itemTime(recordedItem(t, toolSteps, questionItem, "completed").UpdatedAt)}}},
			notes: []string{"note " + askingNote}},
		{name: "an approval it waits on is noted the same way",
			row: childRow(nil),
			thread: func(t *testing.T) []json.RawMessage {
				return []json.RawMessage{event(2, "turn-item.updated", recordedItem(t, toolSteps, approvalItem, "waiting"))}
			},
			want: harness.Handoff{ID: "native-8", Driver: "claudeAgent", Title: "Find callers", Prompt: "Find every caller of mapItem.",
				State: harness.HandoffRunning, Steps: []harness.Activity{{Kind: harness.ActivityQuestion, CallID: approvalItem,
					Summary: approvalPrompt, At: itemTime(recordedItem(t, toolSteps, approvalItem, "waiting").UpdatedAt)}}},
			notes: []string{"note " + askingNote}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, notes := pillOf(t, tt.row, tt.thread(t))
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.notes, labels(notes))
		})
	}
}

func TestPills_StateFollowsTheSubagentUntilTheTurnLeavesIt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		status, state string
		// left is the state once the turn ended without waiting for it.
		left string
	}{
		{"pending", harness.HandoffRunning, harness.HandoffLeftRunning},
		{"running", harness.HandoffRunning, harness.HandoffLeftRunning},
		{"waiting", harness.HandoffRunning, harness.HandoffLeftRunning},
		{"completed", harness.HandoffDone, harness.HandoffDone},
		{"idle", harness.HandoffDone, harness.HandoffDone},
		{"failed", harness.HandoffFailed, harness.HandoffFailed},
		{"cancelled", harness.HandoffInterrupted, harness.HandoffInterrupted},
		{"interrupted", harness.HandoffInterrupted, harness.HandoffInterrupted},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			t.Parallel()
			var ps pills
			ps.rows([]subagent{childRow(func(r *subagent) { r.Status = tt.status })})
			assert.Equal(t, tt.state, ps.changed()[0].Handoff.State)
			ended := ps.end(harness.HandoffLeftRunning)
			if tt.left == tt.state {
				assert.Empty(t, ended, "the end changes nothing it settled")
				return
			}
			require.Len(t, ended, 1)
			assert.Equal(t, tt.left, ended[0].Handoff.State)
		})
	}
}

func TestPills_Caps(t *testing.T) {
	t.Parallel()
	var rows []subagent
	for i := range maxChildren + 1 {
		rows = append(rows, childRow(func(r *subagent) {
			r.ID, r.StartedAt = fmt.Sprintf("task-%02d", i), fmt.Sprintf("2026-10-03T17:00:%02d.000Z", i)
		}))
	}
	rows[0].Prompt = strings.Repeat("é", maxPrompt)
	var ps pills
	ps.rows(rows)
	require.Len(t, ps.order, maxChildren, "a reply carries at most 20 hand-offs, the first T3 started")
	assert.Nil(t, ps.find("task-20"))

	long := strings.Repeat("x", 3*maxChildDetail)
	thread := []json.RawMessage{}
	for i := range maxChildSteps + 50 {
		thread = append(thread, event(i+2, "turn-item.updated", map[string]any{"id": fmt.Sprintf("c-%03d", i), "threadId": "th-2",
			"runId": nil, "ordinal": i, "status": "completed", "type": "command_execution", "input": long,
			"updatedAt": "2026-10-03T17:01:00.000Z"}))
	}
	for _, item := range streamItems(t, thread) {
		ps.apply("task-00", item)
	}
	h := ps.changed()[0].Handoff
	require.Len(t, h.Steps, maxChildSteps)
	assert.Equal(t, "c-050", h.Steps[0].CallID, "the newest 200 steps stay")
	assert.Len(t, h.Steps[0].Detail, maxChildDetail)
	assert.True(t, strings.HasPrefix(`{"input":"`+long, h.Steps[0].Detail))
	assert.Len(t, h.Prompt, maxPrompt, "cut on a rune boundary")
	assert.Equal(t, strings.Repeat("é", maxPrompt/2), h.Prompt)
}

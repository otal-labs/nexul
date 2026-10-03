package t3clientv2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

func runAt(ordinal int, messageID, status string) map[string]any {
	r := runOf(messageID, status)
	r["id"] = fmt.Sprintf("run:thread:th-1:ordinal:%d", ordinal)
	r["ordinal"] = ordinal
	return r
}

// projectionWith is the recorded getThreadProjection answer for th-1, with its runs and runtime requests replaced.
func projectionWith(t *testing.T, runs []any, requests ...any) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "get-thread-projection"+nightly))
	require.NoError(t, err)
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
		var rec struct {
			Frame struct {
				Exit struct {
					Tag   string         `json:"_tag"`
					Value map[string]any `json:"value"`
				} `json:"exit"`
			} `json:"frame"`
		}
		require.NoError(t, json.Unmarshal(line, &rec))
		if rec.Frame.Exit.Tag != "Success" {
			continue
		}
		p := rec.Frame.Exit.Value
		p["runs"] = runs
		p["runtimeRequests"] = append([]any{}, requests...)
		return p
	}
	t.Fatal("the fixture holds no projection")
	return nil
}

// rejected is T3 refusing a command with message, as a dispatch Exit's cause.
func rejected(commandType, message string) []any {
	return []any{map[string]any{"_tag": "Fail", "error": map[string]any{
		"_tag": "OrchestrationV2DispatchCommandError", "commandId": "cmd-9", "commandType": commandType,
		"message": message, "detail": message}}}
}

func TestInterrupt_StopsTheRunTheTurnIsFor(t *testing.T) {
	t.Parallel()
	const notInterruptible = "Run run:thread:th-1:ordinal:1 is not interruptible."
	interrupt := func(ordinal int) map[string]any {
		return map[string]any{"type": "run.interrupt", "threadId": "th-1", "runId": fmt.Sprintf("run:thread:th-1:ordinal:%d", ordinal),
			"reason": "Stopped from Nexul"}
	}
	tests := []struct {
		name string
		// watching is the message id of the turn watching th-1, "" for none.
		watching string
		runs     []any
		causes   map[string]any
		want     map[string]any // the command sent, commandId aside; nil for none
		wantErr  string
		errIs    error
	}{
		{name: "the turn's own live run is interrupted without holding the queue behind it", watching: "msg-1",
			runs: []any{runAt(1, "msg-1", "running"), runAt(2, "msg-x", runQueued)},
			want: interrupt(1)},
		{name: "the turn's own queued run is cancelled", watching: "msg-2",
			runs: []any{runAt(1, "msg-x", "running"), runAt(2, "msg-2", runQueued)},
			want: map[string]any{"type": "queued-run.cancel", "threadId": "th-1", "runId": "run:thread:th-1:ordinal:2"}},
		{name: "with no turn watching, the newest unfinished run is stopped",
			runs: []any{runAt(1, "msg-a", runCompleted), runAt(2, "msg-b", "running"), runAt(3, "msg-c", runQueued)},
			want: map[string]any{"type": "queued-run.cancel", "threadId": "th-1", "runId": "run:thread:th-1:ordinal:3"}},
		{name: "a message id T3 has no run for falls back to the newest unfinished run", watching: "msg-gone",
			runs: []any{runAt(1, "msg-a", runCompleted), runAt(2, "msg-b", "starting")},
			want: interrupt(2)},
		{name: "a waiting latest run is interrupted, which stops its background work", watching: "msg-1",
			runs: []any{runAt(1, "msg-1", runWaiting)},
			want: interrupt(1)},
		{name: "a waiting latest run T3 calls not interruptible has nothing left to stop", watching: "msg-1",
			runs:   []any{runAt(1, "msg-1", runWaiting)},
			causes: map[string]any{"run.interrupt": rejected("run.interrupt", notInterruptible)}},
		{name: "a live run T3 will not interrupt fails with T3's message", watching: "msg-1",
			runs:    []any{runAt(1, "msg-1", "running")},
			causes:  map[string]any{"run.interrupt": rejected("run.interrupt", notInterruptible)},
			wantErr: "invalid: stop the T3 run: " + notInterruptible, errIs: apperrs.ErrInvalid},
		{name: "a waiting run with a run after it is past stopping", watching: "msg-1",
			runs:    []any{runAt(1, "msg-1", runWaiting), runAt(2, "msg-x", runQueued)},
			wantErr: "conflict: nothing is running on this T3 thread", errIs: apperrs.ErrConflict},
		{name: "the turn's own run finished, and another run on the thread is left alone", watching: "msg-1",
			runs:    []any{runAt(1, "msg-1", runCompleted), runAt(2, "msg-x", "running")},
			wantErr: "conflict: nothing is running on this T3 thread", errIs: apperrs.ErrConflict},
		{name: "an idle thread has nothing to stop",
			runs:    []any{runAt(1, "msg-1", runCompleted)},
			wantErr: "conflict: nothing is running on this T3 thread", errIs: apperrs.ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			f.Projections = map[string]any{"th-1": projectionWith(t, tt.runs)}
			f.CommandCauses = tt.causes
			if tt.watching != "" {
				h.turns.Store("th-1", newLive(t.Context(), tt.watching))
			}

			err := h.Interrupt(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1"})
			if tt.wantErr != "" {
				require.ErrorIs(t, err, tt.errIs)
				assert.EqualError(t, err, tt.wantErr)
			}
			if tt.wantErr == "" {
				require.NoError(t, err)
			}
			if tt.want == nil {
				assert.Empty(t, f.Dispatched)
				return
			}
			cmd := t3rpctest.WaitFor(t, f.Dispatched, "the stop command")
			assert.NotEmpty(t, cmd["commandId"])
			delete(cmd, "commandId")
			assert.Equal(t, tt.want, cmd)
		})
	}
}

func TestInterruptAndAnswer_ThreadT3CannotLoad_FailWithoutACommand(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	f.ProjectionCause = missingThreadCause(t)
	target := harness.Target{Session: laptop(f), SessionID: "th-1"}

	err := h.Interrupt(t.Context(), target)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.EqualError(t, err, "invalid: read the T3 thread: Failed to load orchestration V2 thread th-1")
	assert.EqualError(t, h.Answer(t.Context(), target, "rq-1", harness.QuestionAnswer{}), err.Error())
	assert.Empty(t, f.Dispatched)
}

// The setup turn stops at its first question; a live question keeps the run running, so Stop interrupts it.
func TestInterrupt_AfterStartTurn_StopsThatTurnsRunNotTheNewest(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	done := begin(t, h, laptop(f), "th-1")
	f.Write(t3rpctest.Chunk(t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"), snapshotWith(t, nil)))
	messageID, _ := t3rpctest.WaitFor(t, f.Dispatched, "message.dispatch")["messageId"].(string)
	require.NoError(t, (<-done).err)
	f.Projections = map[string]any{"th-1": projectionWith(t, []any{runAt(1, messageID, "running"), runAt(2, "msg-x", runQueued)},
		map[string]any{"id": "rq-1", "status": "pending", "responseCapability": map[string]any{"type": "live", "providerSessionId": "ps-1"}})}

	require.NoError(t, h.Interrupt(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1"}))
	cmd := t3rpctest.WaitFor(t, f.Dispatched, "run.interrupt")
	assert.Equal(t, "run.interrupt", cmd["type"])
	assert.Equal(t, "run:thread:th-1:ordinal:1", cmd["runId"], "the turn's own run, though a newer one is queued")
}

func TestStartTurn_ContextEndsWhileQueued_CancelsOwnRun(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan started, 1)
	go func() {
		r, err := h.StartTurn(ctx, harness.Target{Session: laptop(f), ProjectID: "pr-1", Provider: "claudeAgent", SessionID: "th-1"}, "Fix login", testPrompts)
		done <- started{r, err}
	}()
	subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
	f.Write(t3rpctest.Chunk(subID, snapshotWith(t, nil, runAt(1, "msg-0", "running"))))
	messageID, _ := t3rpctest.WaitFor(t, f.Dispatched, "message.dispatch")["messageId"].(string)
	s := <-done
	require.NoError(t, s.err)
	f.Write(t3rpctest.Chunk(subID, event(5, "run.created", runAt(2, messageID, runQueued))))
	note := t3rpctest.WaitFor(t, s.result.Updates, "the queued note")
	require.NotNil(t, note.Activity)
	require.Equal(t, queuedNote, note.Activity.Summary)

	cancel()
	cmd := t3rpctest.WaitFor(t, f.Dispatched, "queued-run.cancel")
	assert.Equal(t, map[string]any{"type": "queued-run.cancel", "commandId": cmd["commandId"], "threadId": "th-1",
		"runId": "run:thread:th-1:ordinal:2"}, cmd, "nobody will read the reply of a run T3 starts later")
}

func TestInterrupt_TurnWaitingOnHandedOffWork_StopsItBeforeItsOwnRunAndEndsInterrupted(t *testing.T) {
	t.Parallel()
	const childRun = "run:thread:th-2:ordinal:1"
	interrupt := func(threadID, runID string) map[string]any {
		return map[string]any{"type": "run.interrupt", "threadId": threadID, "runId": runID, "reason": "Stopped from Nexul"}
	}
	dispose := map[string]any{"type": "delegated_task.completion-delivery.dispose", "parentThreadId": "th-1", "taskId": "task-1"}
	running := projectionWith(t, []any{merged(runAt(1, "msg-child", "running"), map[string]any{"id": childRun, "threadId": "th-2"})})
	tests := []struct {
		name string
		// events follow the turn's own run, given its message id; ownRuns is that run and those after it in the projection.
		events  func(messageID string) []json.RawMessage
		ownRuns func(messageID string) []any
		child   any // th-2's projection; nil fails its read
		want    []map[string]any
		notes   []string
	}{
		{name: "an async child's result is dropped before its run is interrupted, then the turn's own waiting run",
			events: func(id string) []json.RawMessage {
				return story([2]any{"run.created", runOf(id, "running")},
					[2]any{"subagent.updated", handedOff("task-1", "app_owned", "running", map[string]any{"completionWake": "always"})},
					[2]any{"run.updated", runOf(id, runWaiting)})
			},
			ownRuns: func(id string) []any { return []any{runOf(id, runWaiting)} },
			child:   running,
			want:    []map[string]any{dispose, interrupt("th-2", childRun), interrupt("th-1", runOne)}},
		{name: "a child that cannot be stopped is noted and the turn's own run is still interrupted; a dropped result is not dropped again",
			events: func(id string) []json.RawMessage {
				return story([2]any{"run.created", runOf(id, "running")},
					[2]any{"subagent.updated", handedOff("task-1", "app_owned", "running", delivery("disposed"))},
					[2]any{"run.updated", runOf(id, runWaiting)})
			},
			ownRuns: func(id string) []any { return []any{runOf(id, runWaiting)} },
			want:    []map[string]any{interrupt("th-1", runOne)},
			notes:   []string{"Could not stop handed-off work in T3 Code: invalid: read the T3 thread: Failed to load orchestration V2 thread th-2"}},
		{name: "a queued wake is cancelled, though the turn's own run is over",
			events: func(id string) []json.RawMessage {
				return story([2]any{"run.created", runOf(id, "running")},
					[2]any{"subagent.updated", handedOff("task-1", "app_owned", "completed", delivery("claimed"))},
					[2]any{"run.updated", merged(runOf(id, runCompleted), map[string]any{"delegatedCompletion": map[string]any{
						"delivery": map[string]any{"generation": 1, "messageId": wakeMessage, "taskIds": []any{"task-1"}}}})},
					[2]any{"run.created", runAt(2, wakeMessage, runQueued)})
			},
			ownRuns: func(id string) []any { return []any{runOf(id, runCompleted), runAt(2, wakeMessage, runQueued)} },
			want: []map[string]any{dispose,
				{"type": "queued-run.cancel", "threadId": "th-1", "runId": runTwo}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			done := begin(t, h, laptop(f), "th-1")
			subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
			f.Write(t3rpctest.Chunk(subID, snapshotWith(t, nil)))
			messageID, _ := t3rpctest.WaitFor(t, f.Dispatched, "message.dispatch")["messageId"].(string)
			s := <-done
			require.NoError(t, s.err)
			values := []any{}
			for _, e := range tt.events(messageID) {
				values = append(values, e)
			}
			f.Write(t3rpctest.Chunk(subID, values...))
			waiting := t3rpctest.WaitFor(t, s.result.Updates, "the waiting step")
			require.NotNil(t, waiting.Activity)
			require.Equal(t, handoffNote, waiting.Activity.Summary)
			f.Projections = map[string]any{"th-1": projectionWith(t, tt.ownRuns(messageID))}
			if tt.child != nil {
				f.Projections["th-2"] = tt.child
			}

			require.NoError(t, h.Interrupt(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1"}))
			var sent []map[string]any
			for range tt.want {
				cmd := t3rpctest.WaitFor(t, f.Dispatched, "a stop command")
				delete(cmd, "commandId")
				sent = append(sent, cmd)
			}
			assert.Equal(t, tt.want, sent, "in T3's order, so no child's end can wake the thread")
			want := []string{}
			for _, n := range tt.notes {
				want = append(want, "note "+n)
			}
			want = append(want, "end "+string(harness.TurnInterrupted))
			assert.Equal(t, want, labels(drainUpdates(t, s.result.Updates)),
				"the turn ends at Stop, whatever T3 reports next, so no later wake is followed")
			assert.Empty(t, f.Dispatched)
		})
	}
}

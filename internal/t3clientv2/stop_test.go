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
			f.Projection = projectionWith(t, tt.runs)
			f.CommandCauses = tt.causes
			if tt.watching != "" {
				h.messages.Store("th-1", tt.watching)
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
	f.Projection = projectionWith(t, []any{runAt(1, messageID, "running"), runAt(2, "msg-x", runQueued)},
		map[string]any{"id": "rq-1", "status": "pending", "responseCapability": map[string]any{"type": "live", "providerSessionId": "ps-1"}})

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

package t3clientv2

import (
	"context"
	"log/slog"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

func TestSteer_HandsTheQueuedRunToTheRunningOne(t *testing.T) {
	t.Parallel()
	const refusal = "Target run run:thread:th-1:ordinal:1 is waiting and cannot be steered."
	tests := []struct {
		name     string
		watching bool
		runs     []any
		causes   map[string]any
		wantErr  error
		// want is the command T3 took, commandId aside; nil for none.
		want        map[string]any
		wantSteered bool
	}{
		{name: "the watched turn's queued run joins the running run", watching: true,
			runs: []any{runAt(1, "msg-0", "running"), runAt(2, "msg-1", runQueued)},
			want: map[string]any{"type": "queued-message.promote-to-steer", "threadId": "th-1",
				"queuedRunId": "run:thread:th-1:ordinal:2", "targetRunId": "run:thread:th-1:ordinal:1"},
			wantSteered: true},
		{name: "a steer T3 refuses leaves the turn queued and unsteered", watching: true,
			runs:    []any{runAt(1, "msg-0", "running"), runAt(2, "msg-1", runQueued)},
			causes:  map[string]any{"queued-message.promote-to-steer": rejected("queued-message.promote-to-steer", refusal)},
			wantErr: apperrs.ErrInvalid},
		{name: "no turn watches the message", runs: []any{runAt(1, "msg-0", "running"), runAt(2, "msg-1", runQueued)}, wantErr: errNotQueued},
		{name: "the message's run already started", watching: true,
			runs: []any{runAt(1, "msg-0", runCompleted), runAt(2, "msg-1", "running")}, wantErr: errNotQueued},
		{name: "no run is under way to take it", watching: true,
			runs: []any{runAt(1, "msg-0", runWaiting), runAt(2, "msg-1", runQueued)}, wantErr: errNothingToSteer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			f.Projections = map[string]any{"th-1": projectionWith(t, tt.runs)}
			f.CommandCauses = tt.causes
			l := newRunningTurn(context.Background(), "msg-1")
			if tt.watching {
				h.turns.Store(turnKey{"th-1", "msg-1"}, l)
			}

			err := h.Steer(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1", TurnID: "msg-1"})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
			if tt.wantErr == nil {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantSteered, l.steered.Load())
			if tt.want == nil {
				assert.Empty(t, f.Dispatched)
				return
			}
			assert.Equal(t, tt.want, withoutID(t3rpctest.WaitFor(t, f.Dispatched, "the steer command")))
		})
	}
}

func TestPump_QueuedRunCancelled_EndsDoneOnlyWhenSteered(t *testing.T) {
	tests := []struct {
		name    string
		steered bool
		want    harness.TurnState
	}{
		{name: "steered into the running run, which replies to it", steered: true, want: harness.TurnDone},
		{name: "cancelled any other way", want: harness.TurnInterrupted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				src := newFakeSource(items(event(2, "run.created", runOf("msg-1", runQueued))))
				out := make(chan harness.Update, 16)
				p := &pump{w: newWatch("msg-1"), log: slog.Default(), live: newRunningTurn(t.Context(), "msg-1")}
				go p.run(t.Context(), src, out)
				synctest.Wait()
				p.live.steered.Store(tt.steered)

				src.ch <- items(event(3, "run.updated", runOf("msg-1", "cancelled")))
				synctest.Wait()
				got := collect(out)
				require.NotEmpty(t, got)
				end := got[len(got)-1].Terminal
				require.NotNil(t, end)
				assert.Equal(t, tt.want, end.State)
			})
		})
	}
}

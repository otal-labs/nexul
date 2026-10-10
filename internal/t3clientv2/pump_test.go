package t3clientv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/t3rpc"
)

type chunkOrEnd struct {
	values []json.RawMessage
	err    error
}

// fakeSource is a subscription fed through a channel, so the pump's timers run on synctest's clock.
type fakeSource struct {
	ch     chan chunkOrEnd
	closed atomic.Bool
}

func newFakeSource(chunks ...chunkOrEnd) *fakeSource {
	s := &fakeSource{ch: make(chan chunkOrEnd, 8)}
	for _, c := range chunks {
		s.ch <- c
	}
	return s
}

func (s *fakeSource) Next(ctx context.Context) ([]json.RawMessage, error) {
	select {
	case c := <-s.ch:
		return c.values, c.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *fakeSource) Close() { s.closed.Store(true) }

func items(values ...json.RawMessage) chunkOrEnd { return chunkOrEnd{values: values} }

// quietDrop is a subscription that opens, sends nothing, and drops after d, as a tunnel cutting an idle socket does.
func quietDrop(d time.Duration, err error) func() (source, error) {
	return func() (source, error) {
		src := newFakeSource()
		time.AfterFunc(d, func() { src.ch <- chunkOrEnd{err: err} })
		return src, nil
	}
}

// silentThread is a handed-off agent's thread that sends nothing.
func silentThread(context.Context, string, int64) (source, error) { return newFakeSource(), nil }

// collect takes every update the pump has sent so far without waiting for more, its end's Marker dropped.
func collect(out <-chan harness.Update) []harness.Update {
	got := collectMarked(out)
	for i, u := range got {
		got[i] = unmarked(u)
	}
	return got
}

// collectMarked is collect with the end's Marker kept.
func collectMarked(out <-chan harness.Update) []harness.Update {
	var got []harness.Update
	for {
		select {
		case u, ok := <-out:
			if !ok {
				return got
			}
			got = append(got, u)
		default:
			return got
		}
	}
}

func TestPump_QueuedRun_NoteRepeatsEveryFiveMinutesUnderOneCallID(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		src := newFakeSource(items(event(2, "run.created", runOf("msg-1", runQueued))))
		out := make(chan harness.Update, 16)
		p := &pump{w: newWatch("msg-1"), log: slog.Default(), live: newRunningTurn(t.Context(), "msg-1")}
		start := time.Now()
		go p.run(t.Context(), src, out)

		time.Sleep(11 * time.Minute)
		synctest.Wait()
		notes := collect(out)
		require.Len(t, notes, 3, "shown at once, then at five and ten minutes")
		for i, u := range notes {
			require.NotNil(t, u.Activity)
			assert.Equal(t, harness.Activity{Kind: harness.ActivityNote, CallID: "queue:" + runOne, Summary: queuedNote,
				At: start.Add(time.Duration(i) * noteEvery).UTC()}, *u.Activity)
		}

		heldRun := runOf("msg-1", runQueued)
		heldRun["queueHeld"] = true
		src.ch <- items(event(3, "run.updated", heldRun))
		synctest.Wait()
		held := collect(out)
		require.Len(t, held, 1, "a held queue changes the note at once")
		assert.Equal(t, heldNote, held[0].Activity.Summary)
		assert.Equal(t, "queue:"+runOne, held[0].Activity.CallID)

		src.ch <- items(event(4, "run.updated", runOf("msg-1", "running")))
		time.Sleep(11 * time.Minute)
		synctest.Wait()
		assert.Empty(t, collect(out), "a started run is no longer queued")

		src.ch <- items(event(5, "run.updated", runOf("msg-1", runWaiting)))
		synctest.Wait()
		assert.Equal(t, []harness.Update{ended(harness.TurnDone, "")}, collect(out))
		assert.True(t, src.closed.Load())
	})
}

func TestPump_StreamEndsMidTurn(t *testing.T) {
	die := &t3rpc.ExitError{Method: subscribeThread, Causes: []t3rpc.ExitCause{{Tag: "Die"}}}
	lost := fmt.Errorf("orchestration.subscribeThread: %w", t3rpc.ErrConnectionLost)
	refused := errors.New("dial T3 websocket: connection refused")
	waitingSnapshot := items(snapshotItem(40, map[string]any{
		"thread": map[string]any{"id": "th-1", "runtimeMode": "full-access", "deletedAt": nil},
		"runs":   []any{runOf("msg-1", runWaiting)}, "turnItems": []any{}, "providerSessions": []any{},
	}))
	tests := []struct {
		name string
		end  error
		// opens is what each resubscribe gets, in order.
		opens   []func() (source, error)
		want    harness.Update
		elapsed time.Duration
		// marker is the run the turn saw end; none when it never heard T3 say so.
		marker string
	}{
		{"a stream T3 gave up on resubscribes after the cursor", die,
			[]func() (source, error){func() (source, error) { return newFakeSource(waitingSnapshot), nil }},
			ended(harness.TurnDone, ""), time.Second, runOne},
		{"a stream that ended resubscribes", io.EOF,
			[]func() (source, error){func() (source, error) { return newFakeSource(waitingSnapshot), nil }},
			ended(harness.TurnDone, ""), time.Second, runOne},
		{"a dropped connection resubscribes once it redials", lost,
			[]func() (source, error){
				func() (source, error) { return nil, refused },
				func() (source, error) { return newFakeSource(waitingSnapshot), nil },
			},
			ended(harness.TurnDone, ""), 6 * time.Second, runOne},
		{"three failed resubscribes end the turn", die,
			[]func() (source, error){
				func() (source, error) { return nil, refused },
				func() (source, error) { return newFakeSource(chunkOrEnd{err: die}), nil },
				func() (source, error) { return nil, refused },
			},
			ended(harness.TurnError, "Lost the connection to T3 Code and couldn't resume the turn after 3 tries: dial T3 websocket: connection refused"),
			31 * time.Second, ""},
		{"reconnects that stay up and drop later keep the turn", lost,
			[]func() (source, error){
				quietDrop(2*time.Minute, lost), quietDrop(2*time.Minute, lost), quietDrop(2*time.Minute, lost),
				quietDrop(2*time.Minute, lost), func() (source, error) { return newFakeSource(waitingSnapshot), nil },
			},
			ended(harness.TurnDone, ""), 4*(time.Second+2*time.Minute) + time.Second, runOne},
		{"a T3 Code that changed protocol ends the turn at once", lost,
			[]func() (source, error){func() (source, error) {
				return nil, harness.ProtocolRefusal("T3 Code on laptop went back to its old orchestrator; Nexul only moves forward. Update T3 Code there.")
			}},
			ended(harness.TurnError, updatedNote), time.Second, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				first := newFakeSource(
					items(event(5, "run.created", runOf("msg-1", "running")), event(6, "run.future-event", map[string]any{})),
					chunkOrEnd{err: tt.end},
				)
				var afters []int64
				p := &pump{w: newWatch("msg-1"), log: slog.Default(), live: newRunningTurn(t.Context(), "msg-1"), open: func(_ context.Context, after int64) (source, error) {
					afters = append(afters, after)
					return tt.opens[len(afters)-1]()
				}}
				// What an earlier, refused Stop could not stop shows before whichever end the turn comes to.
				p.live.setNotes([]string{childUnreadable})
				out := make(chan harness.Update, 16)
				start := time.Now()

				p.run(t.Context(), first, out)

				got := collectMarked(out)
				require.Len(t, got, 2)
				assert.Equal(t, childUnreadable, got[0].Activity.Summary)
				assert.Equal(t, tt.want, unmarked(got[1]))
				assert.Equal(t, tt.marker, got[1].Terminal.Marker, "a turn that lost T3 keeps no marker, so the thread reads as news")
				assert.Equal(t, tt.elapsed, time.Since(start), "backs off before each try")
				require.Len(t, afters, len(tt.opens), "every try is made, and no more")
				for _, after := range afters {
					assert.Equal(t, int64(6), after, "resumes after the last event, an unknown one included")
				}
				assert.True(t, first.closed.Load())
			})
		})
	}
}

func TestPump_StopDuringAResubscribeBackoff_EndsInterruptedAtOnceWithItsNote(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lost := fmt.Errorf("orchestration.subscribeThread: %w", t3rpc.ErrConnectionLost)
		first := newFakeSource(items(event(5, "run.created", runOf("msg-1", "running"))), chunkOrEnd{err: lost})
		var opens atomic.Int32
		p := &pump{w: newWatch("msg-1"), log: slog.Default(), live: newRunningTurn(t.Context(), "msg-1"),
			open: func(context.Context, int64) (source, error) {
				opens.Add(1)
				return nil, errors.New("dial T3 websocket: connection refused")
			}}
		out := make(chan harness.Update, 16)
		start := time.Now()
		go p.run(t.Context(), first, out)
		synctest.Wait()

		p.live.setNotes([]string{childUnreadable})
		p.live.halt()
		synctest.Wait()

		assert.Equal(t, []string{"note " + childUnreadable, "end " + string(harness.TurnInterrupted)}, labels(collect(out)))
		assert.Zero(t, time.Since(start), "Stop does not wait out the backoff")
		assert.Zero(t, opens.Load(), "nor resubscribes a turn it stopped")
	})
}

func TestPump_HandedOffWorkStillRunning_StepRepeatsUntilTheCapEndsTheTurnDone(t *testing.T) {
	for _, tt := range []struct{ name, ends string }{
		{"its own run done", runWaiting},
		{"its own run cut by a T3 restart", "cancelled"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				src := newFakeSource(items(
					event(2, "run.created", runOf("msg-1", "running")),
					event(3, "subagent.updated", handedOff("task-1", "app_owned", "running", map[string]any{"completionWake": "settled_only"})),
				))
				out := make(chan harness.Update, 32)
				p := &pump{w: newWatch("msg-1"), log: slog.Default(), live: newRunningTurn(t.Context(), "msg-1"), openChild: silentThread}
				start := time.Now()
				go p.run(t.Context(), src, out)

				// A wait-mode delegation keeps the parent running and the stream silent.
				time.Sleep(40 * time.Minute)
				synctest.Wait()
				steps := collect(out)
				require.NotEmpty(t, steps)
				assert.Equal(t, harness.HandoffRunning, steps[0].Handoff.State, "the pill shows first")
				steps = steps[1:]
				require.Len(t, steps, 9, "at once, then every five minutes: well inside chat's and plays' 15-minute silence windows")
				for i, u := range steps {
					require.NotNil(t, u.Activity, "nothing ends the turn while the handed-off work runs")
					assert.Equal(t, harness.Activity{Kind: harness.ActivityNote, CallID: "handoff:" + runOne, Summary: handoffNote,
						At: start.Add(time.Duration(i) * noteEvery).UTC()}, *u.Activity)
				}

				src.ch <- items(event(4, "run.updated", runOf("msg-1", tt.ends)))
				time.Sleep(handoffCap - time.Second)
				synctest.Wait()
				waited := collect(out)
				require.NotEmpty(t, waited)
				for _, u := range waited {
					require.NotNil(t, u.Activity, "only the step until the cap, which counts from the run's own end")
					assert.Equal(t, "handoff:"+runOne, u.Activity.CallID)
					assert.Equal(t, handoffNote, u.Activity.Summary)
				}

				time.Sleep(time.Second)
				synctest.Wait()
				last := collect(out)
				require.Len(t, last, 2)
				assert.Equal(t, harness.HandoffLeftRunning, last[0].Handoff.State, "the pill still running at the cap")
				assert.Equal(t, harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone, LeftRunning: true}}, last[1])
				assert.True(t, src.closed.Load())
			})
		})
	}
}

func TestPump_HandedOffAgentsThread_StreamsIntoItsPillUntilTheTurnEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		running := childRow(nil)
		row := map[string]any{"id": running.ID, "threadId": "th-1", "runId": runOne, "origin": running.Origin, "driver": running.Driver,
			"status": "running", "title": running.Title, "childThreadId": "th-2", "prompt": running.Prompt, "result": nil}
		first := newFakeSource(items(event(2, "run.created", runOf("msg-1", "running")), event(3, "subagent.updated", row)))
		lost := fmt.Errorf("orchestration.subscribeThread: %w", t3rpc.ErrConnectionLost)
		children := []*fakeSource{newFakeSource(), newFakeSource()}
		type opened struct {
			thread string
			after  int64
		}
		var opens []opened
		second := newFakeSource()
		p := &pump{w: newWatch("msg-1"), log: slog.Default(), live: newRunningTurn(t.Context(), "msg-1"),
			open: func(context.Context, int64) (source, error) { return second, nil },
			openChild: func(_ context.Context, thread string, after int64) (source, error) {
				opens = append(opens, opened{thread, after})
				return children[len(opens)-1], nil
			}}
		out := make(chan harness.Update, 32)
		go p.run(t.Context(), first, out)
		synctest.Wait()
		require.Equal(t, []opened{{"th-2", 0}}, opens, "the agent's own thread is followed from a snapshot")
		collect(out)

		children[0].ch <- items(recorded(t, childThread)...)
		synctest.Wait()
		pills := handoffsIn(collect(out))
		require.Len(t, pills, 1, "the child's thread wakes the turn while its own stream is quiet")
		assert.Equal(t, "Two callers: watch.step and the child map.", pills[0].Reply)
		assert.Equal(t, "Shell", pills[0].Steps[0].Tool)

		first.ch <- chunkOrEnd{err: lost}
		time.Sleep(time.Second)
		second.ch <- items(event(4, "run.updated", runOf("msg-1", runWaiting)))
		synctest.Wait()
		assert.Equal(t, []opened{{"th-2", 0}, {"th-2", 560}}, opens, "once the connection is back, the child's thread resumes after its cursor")
		assert.True(t, children[0].closed.Load())

		row["status"], row["result"] = "completed", "Two callers."
		second.ch <- items(event(5, "subagent.updated", row))
		synctest.Wait()
		last := collect(out)
		require.Len(t, last, 2)
		assert.Equal(t, harness.HandoffDone, last[0].Handoff.State, "the pill's last state comes before the end")
		assert.Equal(t, "Two callers.", last[0].Handoff.Reply)
		assert.Equal(t, ended(harness.TurnDone, ""), last[1])
		assert.True(t, children[1].closed.Load(), "the child's thread is let go with the turn")
	})
}

func handoffsIn(updates []harness.Update) []harness.Handoff {
	var got []harness.Handoff
	for _, u := range updates {
		if u.Handoff != nil {
			got = append(got, *u.Handoff)
		}
	}
	return got
}

func TestPump_HandedOffAgentsThreadUnreadable_ItsPillComesFromItsRowAndTheTurnGoesOn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		row := func(id, thread, status string, result any) map[string]any {
			return map[string]any{"id": id, "threadId": "th-1", "runId": runOne, "origin": "app_owned", "driver": "codex", "status": status,
				"title": "Audit " + id, "childThreadId": thread, "prompt": "Audit.", "result": result}
		}
		src := newFakeSource(items(event(2, "run.created", runOf("msg-1", "running")),
			event(3, "subagent.updated", row("task-1", "th-2", "running", nil)), event(4, "subagent.updated", row("task-2", "th-3", "running", nil))))
		missing := newFakeSource(chunkOrEnd{err: &t3rpc.ExitError{Method: subscribeThread, Causes: []t3rpc.ExitCause{{Tag: "Fail"}}}})
		p := &pump{w: newWatch("msg-1"), log: slog.Default(), live: newRunningTurn(t.Context(), "msg-1"),
			openChild: func(_ context.Context, thread string, _ int64) (source, error) {
				if thread == "th-2" {
					return nil, errors.New("dial T3 websocket: connection refused")
				}
				return missing, nil
			}}
		out := make(chan harness.Update, 32)
		go p.run(t.Context(), src, out)
		synctest.Wait()
		assert.True(t, missing.closed.Load(), "a thread T3 cannot stream is let go")

		src.ch <- items(event(5, "subagent.updated", row("task-1", "th-2", "completed", "One issue.")),
			event(6, "subagent.updated", row("task-2", "th-3", "failed", nil)), event(7, "run.updated", runOf("msg-1", runWaiting)))
		synctest.Wait()
		got := collect(out)
		require.NotEmpty(t, got)
		assert.Equal(t, ended(harness.TurnDone, ""), got[len(got)-1], "neither thread fails the turn")
		pills := map[string]harness.Handoff{}
		for _, h := range handoffsIn(got) {
			pills[h.ID] = h
		}
		assert.Equal(t, harness.Handoff{ID: "task-1", Driver: "codex", Title: "Audit task-1", Prompt: "Audit.", State: harness.HandoffDone,
			Reply: "One issue."}, pills["task-1"])
		assert.Equal(t, harness.HandoffFailed, pills["task-2"].State)
	})
}

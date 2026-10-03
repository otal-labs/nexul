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

// collect takes every update the pump has sent so far without waiting for more.
func collect(out <-chan harness.Update) []harness.Update {
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
		p := &pump{w: newWatch("msg-1"), log: slog.Default()}
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
	}{
		{"a stream T3 gave up on resubscribes after the cursor", die,
			[]func() (source, error){func() (source, error) { return newFakeSource(waitingSnapshot), nil }},
			ended(harness.TurnDone, ""), time.Second},
		{"a stream that ended resubscribes", io.EOF,
			[]func() (source, error){func() (source, error) { return newFakeSource(waitingSnapshot), nil }},
			ended(harness.TurnDone, ""), time.Second},
		{"a dropped connection resubscribes once it redials", lost,
			[]func() (source, error){
				func() (source, error) { return nil, refused },
				func() (source, error) { return newFakeSource(waitingSnapshot), nil },
			},
			ended(harness.TurnDone, ""), 6 * time.Second},
		{"three failed resubscribes end the turn", die,
			[]func() (source, error){
				func() (source, error) { return nil, refused },
				func() (source, error) { return newFakeSource(chunkOrEnd{err: die}), nil },
				func() (source, error) { return nil, refused },
			},
			ended(harness.TurnError, "Lost the connection to T3 Code and couldn't resume the turn after 3 tries: dial T3 websocket: connection refused"),
			31 * time.Second},
		{"a T3 Code that changed protocol ends the turn at once", lost,
			[]func() (source, error){func() (source, error) {
				return nil, harness.ProtocolRefusal("T3 Code on laptop went back to its old orchestrator; Nexul only moves forward. Update T3 Code there.")
			}},
			ended(harness.TurnError, updatedNote), time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				first := newFakeSource(
					items(event(5, "run.created", runOf("msg-1", "running")), event(6, "run.future-event", map[string]any{})),
					chunkOrEnd{err: tt.end},
				)
				var afters []int64
				p := &pump{w: newWatch("msg-1"), log: slog.Default(), open: func(_ context.Context, after int64) (source, error) {
					afters = append(afters, after)
					return tt.opens[len(afters)-1]()
				}}
				out := make(chan harness.Update, 16)
				start := time.Now()

				p.run(t.Context(), first, out)

				assert.Equal(t, []harness.Update{tt.want}, collect(out))
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

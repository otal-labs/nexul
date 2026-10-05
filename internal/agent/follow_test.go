package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
)

// followRig is a conversation on thread th-1 whose harness records each Watch and ends it done at run-9.
type followRig struct {
	svc     *Service
	conv    *fakeConversations
	targets *fakeTargets
	watched chan harness.Target
}

func newFollowRig(t *testing.T, seen string) *followRig {
	t.Helper()
	conv := newFakeConversations(Conversation{ID: "conv-1", ThreadID: "th-1"})
	if seen != "" {
		conv.seen["conv-1"] = seen
	}
	r := &followRig{conv: conv, targets: &fakeTargets{target: testTarget()}, watched: make(chan harness.Target, 4)}
	client := &harnesstest.Client{WatchFn: func(_ context.Context, target harness.Target) (harness.StartResult, error) {
		r.watched <- target
		return harness.StartResult{SessionID: "th-1", Updates: updatesChan(
			harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-9", Text: "Built the worker.", Streaming: false}},
			harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone, Marker: "run-9"}},
		)}, nil
	}}
	r.svc = NewService(Config{Conversations: conv, Targets: r.targets, Harnesses: harnesstest.Registry(client), Live: &fakeLive{}})
	return r
}

func (r *followRig) nextWatch(t *testing.T) harness.Target {
	t.Helper()
	select {
	case target := <-r.watched:
		return target
	case <-time.After(5 * time.Second):
		t.Fatal("the thread was never followed")
		return harness.Target{}
	}
}

func TestOnSessionUpdate_NewsNoTurnWatches_IsCaughtUpFromWhatWasSeen(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		update harness.SessionUpdate
	}{
		{"a turn newer than the one seen", harness.SessionUpdate{SessionID: "th-1", Latest: "run-8"}},
		{"work under way on the turn seen", harness.SessionUpdate{SessionID: "th-1", Latest: "run-7", Working: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := newFollowRig(t, "run-7")

			r.svc.OnSessionUpdate("u-1", "c-2", tt.update)

			target := r.nextWatch(t)
			assert.Equal(t, "th-1", target.SessionID)
			assert.Equal(t, "run-7", target.Since, "the catch-up starts after the last turn seen to end")
			waitFor(t, 5*time.Second, func() bool { return r.conv.seenMarker("conv-1") == "run-9" })
			replies, _ := r.conv.snapshot()
			require.Len(t, replies, 1)
			assert.Equal(t, "Built the worker.", replies[0].body)
			assert.Equal(t, "u-1", replies[0].viaUserID, "it runs as the person whose computer reported it")
			assert.Equal(t, &TargetOverride{ComputerID: "c-2"}, r.targets.override, "on the computer that reported it")
		})
	}
}

func TestOnSessionUpdate_NothingToFollow_LeavesTheThreadAlone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		update harness.SessionUpdate
	}{
		{"the turn seen is still the newest and nothing runs", harness.SessionUpdate{SessionID: "th-1", Latest: "run-7"}},
		{"a deleted thread", harness.SessionUpdate{SessionID: "th-1", Latest: "run-8", Gone: true}},
		{"a thread no conversation uses", harness.SessionUpdate{SessionID: "th-other", Latest: "run-8", Working: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := newFollowRig(t, "run-7")

			r.svc.OnSessionUpdate("u-1", "c-2", tt.update)

			assert.True(t, r.svc.claimFollow("conv-1"), "no catch-up holds the conversation")
			assert.Empty(t, r.watched)
		})
	}
}

func TestOnSessionUpdate_ThreadFromBeforeMarkers_StartsFromNowAndFollowsOnlyLiveWork(t *testing.T) {
	t.Parallel()
	idle := newFollowRig(t, "")
	idle.svc.OnSessionUpdate("u-1", "c-2", harness.SessionUpdate{SessionID: "th-1", Latest: "run-4"})
	assert.Equal(t, "run-4", idle.conv.seenMarker("conv-1"), "its past is taken as seen, so it never replays")
	assert.Empty(t, idle.watched)

	live := newFollowRig(t, "")
	live.svc.OnSessionUpdate("u-1", "c-2", harness.SessionUpdate{SessionID: "th-1", Latest: "run-4", Working: true})
	assert.Equal(t, "", live.nextWatch(t).Since, "work under way is followed from its newest turn, as after a restart")
}

func TestOnSessionUpdate_TurnAlreadyRunsThere_IsLeftToIt(t *testing.T) {
	t.Parallel()
	r := newFollowRig(t, "run-7")
	r.svc.setActive("conv-1", &activeTurn{})

	r.svc.OnSessionUpdate("u-1", "c-2", harness.SessionUpdate{SessionID: "th-1", Latest: "run-8", Working: true})

	assert.Empty(t, r.watched)
}

// fakeFollower takes every thread and holds it until release closes.
type fakeFollower struct {
	calls   chan []string
	release chan struct{}
}

func (f *fakeFollower) FollowThread(_ context.Context, conversationID, threadID, userID, computerID, since string) (<-chan struct{}, bool) {
	f.calls <- []string{conversationID, threadID, userID, computerID, since}
	return f.release, true
}

func TestOnSessionUpdate_FollowerTakesIt_HoldsTheConversationUntilItsTurnEnds(t *testing.T) {
	t.Parallel()
	r := newFollowRig(t, "run-7")
	f := &fakeFollower{calls: make(chan []string, 4), release: make(chan struct{})}
	r.svc.SetFollower(f)

	r.svc.OnSessionUpdate("u-1", "c-2", harness.SessionUpdate{SessionID: "th-1", Latest: "run-8"})
	assert.Equal(t, []string{"conv-1", "th-1", "u-1", "c-2", "run-7"}, <-f.calls)
	r.svc.OnSessionUpdate("u-1", "c-2", harness.SessionUpdate{SessionID: "th-1", Latest: "run-8", Working: true})
	assert.Empty(t, f.calls, "a second update while the catch-up runs starts nothing")
	assert.Empty(t, r.watched, "the follower's turn, not a plain one")

	close(f.release)
	waitFor(t, 5*time.Second, func() bool {
		if !r.svc.claimFollow("conv-1") {
			return false
		}
		r.svc.releaseFollow("conv-1")
		return true
	})
}

func TestRunTurn_TextSteps_AreNamedByTheirMessageSoAReplayReplacesThem(t *testing.T) {
	t.Parallel()
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "th-1", Updates: updatesChan(
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "Reading", Streaming: true}},
		harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, CallID: "c-1", Tool: "Read"}},
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "Reading done", Streaming: false}},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)}}
	obs := &fakeObserver{}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "go", Observer: obs})

	var ids []string
	for _, a := range obs.activity {
		ids = append(ids, a.CallID)
	}
	assert.Equal(t, []string{"text:m-1:0", "c-1", "text:m-1:7"}, ids)
	assert.Empty(t, conv.seenMarker("conv-1"), "a turn whose harness kept no marker records none")
}

func TestRunTurn_KeptThreadGone_EndsSessionGoneWithoutTellingTheThread(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		thread string
	}{
		{"the harness no longer has it", "th-1"},
		{"the conversation never had one", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			conv := newFakeConversations(Conversation{ID: "conv-1", ThreadID: tt.thread})
			client := &fakeHarness{startErr: fmt.Errorf("watch t3 thread th-1: %w", harness.ErrSessionGone)}
			obs := &fakeObserver{}
			svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})

			svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "more", KeepThread: true, Observer: obs})

			require.NotNil(t, obs.result)
			assert.True(t, obs.result.SessionGone)
			_, systemPosts := conv.snapshot()
			assert.Empty(t, systemPosts, "the caller says what a gone thread means")
		})
	}
}

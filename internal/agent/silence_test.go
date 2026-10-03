package agent

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
)

const noSignalReply = "Agent turn failed: turn gave no completion signal"

// chatTurn fires an @Agent mention whose harness turn reads the returned channel.
func chatTurn(t *testing.T) (*Service, *fakeConversations, chan harness.Update) {
	t.Helper()
	ch := make(chan harness.Update, 16)
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "sess-1", Updates: ch}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	require.NoError(t, svc.HandleMessageCreated(t.Context(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	synctest.Wait()
	return svc, conv, ch
}

func TestHandleMessageCreated_UpdatesKeepArriving_TurnRunsPastTenMinutes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, conv, ch := chatTurn(t)
		for range 8 {
			time.Sleep(5 * time.Minute)
			ch <- harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, Tool: "Bash", Summary: "go test"}}
		}
		ch <- harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "All green.", Streaming: false}}
		ch <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
		close(ch)
		synctest.Wait()

		replies, systemPosts := conv.snapshot()
		assert.Empty(t, systemPosts, "forty minutes of steady updates is a working turn, not a lost one")
		require.Len(t, replies, 1)
		assert.Equal(t, "All green.", replies[0].body)
	})
}

func TestHandleMessageCreated_FifteenMinutesOfSilence_PostsTheNoSignalMessage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, conv, _ := chatTurn(t)

		time.Sleep(15*time.Minute - time.Second)
		synctest.Wait()
		_, systemPosts := conv.snapshot()
		require.Empty(t, systemPosts, "the window has not elapsed yet")

		time.Sleep(time.Second)
		synctest.Wait()
		_, systemPosts = conv.snapshot()
		require.Len(t, systemPosts, 1)
		assert.Contains(t, systemPosts[0].body, noSignalReply)
	})
}

func TestHandleMessageCreated_HarnessNeverAnswersDuringSetup_FailsAtFifteenMinutes(t *testing.T) {
	startHangs := func(ctx context.Context, _ harness.Target, _ string, _ harness.TurnPrompts) (harness.StartResult, error) {
		<-ctx.Done()
		return harness.StartResult{}, ctx.Err()
	}
	tests := []struct {
		name    string
		targets *fakeTargets
		client  *harnesstest.Client
		want    string
	}{
		{"target check", &fakeTargets{hang: true}, &harnesstest.Client{}, "Agent isn't configured to run yet"},
		{"version probe", &fakeTargets{target: testTarget()}, &harnesstest.Client{
			VersionFn:   func(ctx context.Context, _ string) (string, error) { <-ctx.Done(); return "", ctx.Err() },
			StartTurnFn: startHangs,
		}, "Agent turn failed to start: the harness did not answer within 15m0s"},
		{"turn start", &fakeTargets{target: testTarget()}, &harnesstest.Client{StartTurnFn: startHangs},
			"Agent turn failed to start: the harness did not answer within 15m0s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				conv := newFakeConversations(Conversation{ID: "conv-1"})
				svc := NewService(Config{Conversations: conv, Targets: tt.targets, Harnesses: harnesstest.Registry(tt.client), Live: &fakeLive{}})
				require.NoError(t, svc.HandleMessageCreated(t.Context(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))

				time.Sleep(15*time.Minute - time.Second)
				synctest.Wait()
				_, systemPosts := conv.snapshot()
				require.Empty(t, systemPosts, "setup gets the whole window")

				time.Sleep(time.Second)
				synctest.Wait()
				_, systemPosts = conv.snapshot()
				require.Len(t, systemPosts, 1, "a harness that never answers fails the turn instead of hanging it")
				assert.Contains(t, systemPosts[0].body, tt.want)
			})
		})
	}
}

func TestHandleMessageCreated_PendingQuestion_PausesTheWindowUntilAnswered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc, conv, ch := chatTurn(t)
		q := question()
		ch <- harness.Update{Question: &q}

		time.Sleep(45 * time.Minute)
		synctest.Wait()
		replies, systemPosts := conv.snapshot()
		require.Empty(t, systemPosts, "the user's silence is not the harness's")
		require.Len(t, replies, 1, "only the question card")

		answer := harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"q1": {Selected: []string{"Yes"}}}}
		require.NoError(t, svc.Answer(t.Context(), "conv-1", "req-1", answer))
		time.Sleep(15*time.Minute - time.Second)
		synctest.Wait()
		_, systemPosts = conv.snapshot()
		require.Empty(t, systemPosts, "the answer starts a whole new window")

		time.Sleep(time.Second)
		synctest.Wait()
		_, systemPosts = conv.snapshot()
		require.Len(t, systemPosts, 1, "once answered, a harness that goes quiet is caught again")
		assert.Contains(t, systemPosts[0].body, noSignalReply)
	})
}

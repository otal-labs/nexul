package plays

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/redact"
)

const mcpToken = "dep_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNO-_"

// savedText is everything a run persisted: trails, their events, thread posts, and the agent's own chat messages.
func savedText(t *testing.T, f *runnerFixture, convs *agentConvs) string {
	t.Helper()
	replies, notes := convs.snapshot()
	f.trails.mu.Lock()
	events := append(f.trails.events[:0:0], f.trails.events...)
	f.trails.mu.Unlock()
	raw, err := json.Marshal([]any{f.trails.all(), events, f.threads.snapshot(), replies, notes})
	require.NoError(t, err)
	return string(raw)
}

func TestRun_SavedTranscriptNeverContainsAToken(t *testing.T) {
	tests := []struct {
		name     string
		terminal harness.TurnResult
	}{
		{"done with a reply", harness.TurnResult{State: harness.TurnDone}},
		{"failed with the token in the error", harness.TurnResult{State: harness.TurnError, LastError: "codex rejected " + mcpToken}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunnerFixture()
			client := &harnesstest.Client{StartTurnFn: func(context.Context, harness.Target, string, harness.TurnPrompts) (harness.StartResult, error) {
				ch := make(chan harness.Update, 3)
				write := step("Bash", "codex mcp add nexul --bearer "+mcpToken)
				write.Detail = `[mcp_servers.nexul]` + "\n" + `bearer_token = "` + mcpToken + `"`
				ch <- harness.Update{Activity: &write}
				ch <- harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "Connected with " + mcpToken, Streaming: false}}
				terminal := tt.terminal
				ch <- harness.Update{Terminal: &terminal}
				close(ch)
				return harness.StartResult{SessionID: "sess-1", Updates: ch}, nil
			}}
			convs := newHarnessRunner(f, client)
			in := ticketRun()
			in.CustomInstructions = "Use " + mcpToken

			_, err := f.runner.Run(ctxAs(starter), in)
			require.NoError(t, err)
			final := <-f.trails.terminal

			saved := savedText(t, f, convs)
			assert.NotContains(t, saved, mcpToken)
			assert.Contains(t, saved, redact.Placeholder)
			assert.Contains(t, final.Activity[0].Detail, "bearer_token = \""+redact.Placeholder)

			live, err := json.Marshal(f.live.snapshot())
			require.NoError(t, err)
			assert.NotContains(t, string(live), mcpToken, "a live frame reaches everyone who can see the thread")
			assert.Contains(t, string(live), redact.Placeholder)
		})
	}
}

func TestRedactTrail_TokenFreeTrailPassesThroughUntouched(t *testing.T) {
	trail := &Trail{ID: "tr-1", LastError: "nothing secret"}
	evt := RunFinishedEvent{Outcome: TrailFailed, LastError: "nothing secret"}
	saved, evts, err := redactTrail(trail, nil)
	require.NoError(t, err)
	assert.Same(t, trail, saved)
	assert.Empty(t, evts)

	_, evts, err = redactTrail(trail, []eventbus.OutboxEvent{{ID: "e1", Payload: evt}})
	require.NoError(t, err)
	assert.Equal(t, evt, evts[0].Payload, "a payload without a token keeps its type")
}

func TestRedactTrail_UnmarshalablePayloadIsAnError(t *testing.T) {
	_, _, err := redactTrail(&Trail{ID: "tr-1"}, []eventbus.OutboxEvent{{ID: "e1", Payload: make(chan int)}})
	require.Error(t, err)
}

package t3clientv2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

func pendingRequest(capability string) map[string]any {
	return map[string]any{"id": "rq-1", "nodeId": "node-1", "kind": "user_input", "status": "pending",
		"responseCapability": map[string]any{"type": capability}}
}

var multiAnswer = harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{
	"Which DB?":     {Text: "sqlite"},
	"Pick one":      {Selected: []string{"auth"}},
	"Pick features": {Selected: []string{"auth", "search"}},
	"Other":         {Text: "my own", Selected: []string{"auth", "search"}},
}}

func TestAnswer_EncodesForHowT3ResumesTheRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		requests []any
		want     map[string]any
	}{
		{"a live question takes several choices as a list; free text wins", []any{pendingRequest("live")},
			map[string]any{"Which DB?": "sqlite", "Pick one": "auth", "Pick features": []any{"auth", "search"}, "Other": "my own"}},
		{"a question answered by a message of its own takes one string per question", []any{pendingRequest("message")},
			map[string]any{"Which DB?": "sqlite", "Pick one": "auth", "Pick features": "auth, search", "Other": "my own"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			f.Projections = map[string]any{"th-1": projectionWith(t, []any{runAt(1, "msg-1", "running")}, tt.requests...)}

			require.NoError(t, h.Answer(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1"}, "rq-1", multiAnswer))
			cmd := t3rpctest.WaitFor(t, f.Dispatched, "runtime-request.respond")
			assert.Equal(t, map[string]any{"type": "runtime-request.respond", "commandId": cmd["commandId"], "threadId": "th-1",
				"requestId": "rq-1", "answers": tt.want}, cmd)
		})
	}
}

func TestAnswer_Refused(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		message string
		errIs   error
		want    string
	}{
		{"a question answered already conflicts", "Runtime request rq-1 is resolved.", apperrs.ErrConflict,
			"conflict: question rq-1 was already answered"},
		{"any other refusal carries T3's message", "Runtime request rq-1 is expired.", apperrs.ErrInvalid,
			"invalid: answer the T3 question: Runtime request rq-1 is expired."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			f.Projections = map[string]any{"th-1": projectionWith(t, []any{runAt(1, "msg-1", "running")}, pendingRequest("live"))}
			f.CommandCauses = map[string]any{"runtime-request.respond": rejected("runtime-request.respond", tt.message)}

			err := h.Answer(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1"}, "rq-1", multiAnswer)
			require.ErrorIs(t, err, tt.errIs)
			assert.EqualError(t, err, tt.want)
		})
	}
}

const runTwo = "run:thread:th-1:ordinal:2"

// askedSnapshot is th-1's first snapshot holding runs and, unless request is nil, the request rq-1 asked on run 1's node-1.
func askedSnapshot(t *testing.T, request map[string]any, runs ...any) json.RawMessage {
	t.Helper()
	var item map[string]any
	require.NoError(t, json.Unmarshal(snapshotWith(t, nil, runs...), &item))
	p := item["projection"].(map[string]any)
	p["runtimeRequests"] = []any{}
	if request != nil {
		p["runtimeRequests"] = []any{request}
	}
	p["nodes"] = []any{map[string]any{"id": "node-1", "runId": runOne}}
	b, err := json.Marshal(item)
	require.NoError(t, err)
	return b
}

func answering(answer harness.QuestionAnswer) harness.TurnPrompts {
	prompts := testPrompts
	prompts.Answer = &harness.PendingAnswer{RequestID: "rq-1", Answer: answer}
	return prompts
}

// sentCommands is every command T3 took on the turn's connection, once the pump's first read acks the snapshot.
func sentCommands(t *testing.T, f *t3rpctest.Server, subID string) []map[string]any {
	t.Helper()
	require.Equal(t, subID, t3rpctest.WaitFor(t, f.Acks, "the snapshot's ack, sent after StartTurn's last command"))
	var commands []map[string]any
	for len(f.Dispatched) > 0 {
		commands = append(commands, <-f.Dispatched)
	}
	return commands
}

func TestStartTurn_AnswerToPendingLiveRequest_SendsOnlyRespond(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	done := beginWith(t, h, laptop(f), "th-1", answering(harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"Which DB?": {Text: "sqlite"}}}))
	subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
	f.Write(t3rpctest.Chunk(subID, askedSnapshot(t, pendingRequest("live"), runAt(1, "msg-0", "running"))))

	s := <-done
	require.NoError(t, s.err)
	commands := sentCommands(t, f, subID)
	require.Len(t, commands, 1, "a message would queue behind the run the question holds open, forever")
	assert.Equal(t, map[string]any{"type": "runtime-request.respond", "commandId": commands[0]["commandId"], "threadId": "th-1",
		"requestId": "rq-1", "answers": map[string]any{"Which DB?": "sqlite"}}, commands[0])
	assert.False(t, s.result.PromptSent, "the messages posted since the question are still owed to the next prompt")

	f.Write(t3rpctest.Chunk(subID,
		event(3, "turn-item.updated", assistantItem("Using sqlite.", false)),
		event(4, "run.updated", runAt(1, "msg-0", runWaiting)),
	))
	assert.Equal(t, []harness.Update{snapshotOf(codexMessage, "Using sqlite.", false), ended(harness.TurnDone, "")},
		drainUpdates(t, s.result.Updates), "the turn is the run the answer resumed")
}

func TestStartTurn_AnswerToAThreadInAnotherProject_StillGoesToThatThread(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	target := harness.Target{Session: laptop(f), ProjectID: "pr-2", Provider: "claudeAgent", SessionID: "th-1"}
	done := beginAs(t, h, target, answering(harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"Which DB?": {Text: "sqlite"}}}))
	subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
	f.Write(t3rpctest.Chunk(subID, askedSnapshot(t, pendingRequest("live"), runAt(1, "msg-0", "running"))))

	first := t3rpctest.WaitFor(t, f.Dispatched, "the first command")
	require.Equal(t, "runtime-request.respond", first["type"], "no new thread for an answer")
	s := t3rpctest.WaitFor(t, done, "StartTurn")
	require.NoError(t, s.err)
	assert.Equal(t, "th-1", s.result.SessionID, "the question's own thread, wherever it lives")
}

func TestStartTurn_AnswerByMessage_FollowsTheRunT3TakesItIn(t *testing.T) {
	t.Parallel()
	steered := map[string]any{"id": "turn-item:message:async-answer:rq-1", "threadId": "th-1", "runId": runTwo,
		"nodeId": "node:run:2:root", "ordinal": 5, "status": "completed", "type": "user_message",
		"messageId": "async-answer:rq-1", "text": "Pick features\nauth, search", "streaming": false}
	tests := []struct {
		name     string
		runs     []any
		taken    json.RawMessage
		finished map[string]any
	}{
		{"a run of its own when nothing else is running", []any{runAt(1, "msg-0", runCompleted)},
			event(3, "run.created", runAt(2, "async-answer:rq-1", "running")), runAt(2, "async-answer:rq-1", runWaiting)},
		{"the live run it was steered into", []any{runAt(1, "msg-0", runCompleted), runAt(2, "msg-x", "running")},
			event(3, "turn-item.updated", steered), runAt(2, "msg-x", runWaiting)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			done := beginWith(t, h, laptop(f), "th-1", answering(harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{
				"Pick features": {Selected: []string{"auth", "search"}}}}))
			subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
			f.Write(t3rpctest.Chunk(subID, askedSnapshot(t, pendingRequest("message"), tt.runs...)))

			s := <-done
			require.NoError(t, s.err)
			commands := sentCommands(t, f, subID)
			require.Len(t, commands, 1, "T3 runs the answer itself")
			assert.Equal(t, map[string]any{"Pick features": "auth, search"}, commands[0]["answers"], "one string per question")
			assert.False(t, s.result.PromptSent)

			reply := assistantItem("Both added.", false)
			reply["runId"] = runTwo
			f.Write(t3rpctest.Chunk(subID, tt.taken, event(6, "turn-item.updated", reply), event(7, "run.updated", tt.finished)))
			assert.Equal(t, []harness.Update{snapshotOf(codexMessage, "Both added.", false), ended(harness.TurnDone, "")},
				drainUpdates(t, s.result.Updates))
		})
	}
}

func TestStartTurn_PendingAnswer_GoesAsAMessageOnlyWhenT3CannotTakeIt(t *testing.T) {
	t.Parallel()
	with := func(status, capability string) map[string]any {
		r := pendingRequest(capability)
		r["status"] = status
		return r
	}
	tests := []struct {
		name    string
		request map[string]any
		// refusal is T3's answer to the respond, "" to take it.
		refusal string
		// sent is the type of every command T3 took, in order; one message.dispatch at most, so never two runs.
		sent     []string
		answered bool
		wantErr  string
	}{
		{name: "a question answered in T3 ends the turn with a note", request: with("resolved", "live"), answered: true},
		{name: "an expired question goes as a message", request: with("expired", "live"), sent: []string{"message.dispatch"}},
		{name: "a cancelled question goes as a message", request: with("cancelled", "message"), sent: []string{"message.dispatch"}},
		{name: "a question its session cannot resume goes as a message", request: with("pending", "not_resumable"), sent: []string{"message.dispatch"}},
		{name: "a question T3 does not list goes as a message", sent: []string{"message.dispatch"}},
		{name: "a live question whose run the snapshot does not hold goes as a message",
			request: merged(with("pending", "live"), map[string]any{"nodeId": "node-9"}), sent: []string{"message.dispatch"}},
		{name: "answered in T3 after the snapshot ends the turn with a note", request: with("pending", "live"),
			refusal: "Runtime request rq-1 is resolved.", answered: true},
		{name: "expired after the snapshot goes as a message", request: with("pending", "live"),
			refusal: "Runtime request rq-1 is expired.", sent: []string{"message.dispatch"}},
		{name: "any other refusal fails the turn with T3's message", request: with("pending", "message"),
			refusal: "Answer each question before sending.", wantErr: "invalid: answer the T3 question: Answer each question before sending."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			if tt.refusal != "" {
				f.CommandCauses = map[string]any{"runtime-request.respond": rejected("runtime-request.respond", tt.refusal)}
			}
			done := beginWith(t, h, laptop(f), "th-1", answering(multiAnswer))
			subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
			f.Write(t3rpctest.Chunk(subID, askedSnapshot(t, tt.request, runAt(1, "msg-0", runCompleted))))

			s := <-done
			if tt.wantErr != "" {
				require.ErrorIs(t, s.err, apperrs.ErrInvalid)
				assert.EqualError(t, s.err, tt.wantErr)
				assert.Empty(t, f.Dispatched)
				return
			}
			require.NoError(t, s.err)
			assert.Equal(t, "th-1", s.result.SessionID)
			if tt.answered {
				updates := drainUpdates(t, s.result.Updates)
				require.Len(t, updates, 2)
				assert.Equal(t, answeredNote, updates[0].Activity.Summary)
				assert.Equal(t, ended(harness.TurnDone, ""), updates[1])
				assert.Empty(t, f.Dispatched, "T3 already runs on the answer it holds")
				assert.False(t, s.result.PromptSent)
				return
			}
			var sent []string
			for _, c := range sentCommands(t, f, subID) {
				sent = append(sent, c["type"].(string))
			}
			assert.Equal(t, tt.sent, sent)
			assert.True(t, s.result.PromptSent)
		})
	}
}

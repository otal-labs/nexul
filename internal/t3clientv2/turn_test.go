package t3clientv2

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

var testPrompts = harness.TurnPrompts{Full: "full prompt", Incremental: "what is new"}

type started struct {
	result harness.StartResult
	err    error
}

// begin starts a turn the way the pipeline does; StartTurn returns only once the message is out, so it runs aside.
func begin(t *testing.T, h *Harness, s harness.Session, sessionID string) <-chan started {
	t.Helper()
	ch := make(chan started, 1)
	go func() {
		r, err := h.StartTurn(t.Context(), harness.Target{Session: s, ProjectID: "pr-1", Provider: "claudeAgent", SessionID: sessionID}, "Fix login", testPrompts)
		ch <- started{r, err}
	}()
	return ch
}

// commandsUntil reads dispatched commands up to and including the first of type last.
func commandsUntil(t *testing.T, f *t3rpctest.Server, last string) []map[string]any {
	t.Helper()
	var commands []map[string]any
	for {
		cmd := t3rpctest.WaitFor(t, f.Dispatched, last)
		commands = append(commands, cmd)
		if cmd["type"] == last {
			return commands
		}
	}
}

// snapshotWith is the recorded first snapshot of a fresh thread, as th-1, with the thread's keys and runs changed.
func snapshotWith(t *testing.T, thread map[string]any, runs ...any) json.RawMessage {
	t.Helper()
	var item map[string]any
	require.NoError(t, json.Unmarshal(recorded(t, "thread-create"+nightly)[0], &item))
	projection := item["projection"].(map[string]any)
	for k, v := range thread {
		projection["thread"].(map[string]any)[k] = v
	}
	if runs != nil {
		projection["runs"] = runs
	}
	b, err := json.Marshal(item)
	require.NoError(t, err)
	return b
}

func drainUpdates(t *testing.T, updates <-chan harness.Update) []harness.Update {
	t.Helper()
	var got []harness.Update
	for {
		select {
		case u, ok := <-updates:
			if !ok {
				return got
			}
			got = append(got, u)
		case <-time.After(5 * time.Second):
			t.Fatalf("the turn never ended; got %+v", got)
		}
	}
}

func TestStartTurn_NewThread_CreatesItWithEveryRequiredKeyAndWatchesItsRun(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	done := begin(t, h, laptop(f), "")

	create := t3rpctest.WaitFor(t, f.Dispatched, "thread.create")
	threadID, _ := create["threadId"].(string)
	require.NotEmpty(t, threadID)
	require.NotEmpty(t, create["commandId"])
	assert.Equal(t, map[string]any{
		"type": "thread.create", "createdBy": "user", "creationSource": "web", "commandId": create["commandId"],
		"threadId": threadID, "projectId": "pr-1", "title": "Fix login",
		"modelSelection": map[string]any{"instanceId": "claudeAgent", "model": "claude-opus-5-5"},
		"runtimeMode":    "full-access", "interactionMode": "default", "branch": nil, "worktreePath": nil,
	}, create, "an empty model resolves to the provider's default; branch and worktreePath are present as null")

	subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
	assert.Equal(t, map[string]any{"threadId": threadID, "acceptBoundedSnapshot": true}, t3rpctest.WaitFor(t, f.SubscribeIn, "subscribe input"))
	f.Write(t3rpctest.Chunk(subID, recorded(t, "thread-create"+nightly)[0]))
	dispatch := t3rpctest.WaitFor(t, f.Dispatched, "message.dispatch")
	assert.Equal(t, "full prompt", dispatch["text"], "a new thread holds nothing yet")
	s := <-done
	require.NoError(t, s.err)
	assert.Equal(t, threadID, s.result.SessionID)

	messageID, _ := dispatch["messageId"].(string)
	f.Write(t3rpctest.Chunk(subID,
		event(3, "run.created", runOf(messageID, "running")),
		event(4, "turn-item.updated", assistantItem("Hello there.", false)),
		event(5, "run.updated", runOf(messageID, runWaiting)),
	))
	assert.Equal(t, []harness.Update{snapshotOf(codexMessage, "Hello there.", false), ended(harness.TurnDone, "")},
		drainUpdates(t, s.result.Updates))
}

func TestStartTurn_WaitsForSnapshotBeforeDispatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		thread   map[string]any
		runs     []any
		commands []string
		text     string
		noted    bool
	}{
		{"a native thread gets what is new", nil, nil,
			[]string{"message.dispatch"}, "what is new", false},
		{"an imported thread with no finished run gets the full prompt", map[string]any{"historyOrigin": "v1_import"}, nil,
			[]string{"message.dispatch"}, "full prompt", false},
		{"an imported thread that finished a run gets what is new", map[string]any{"historyOrigin": "v1_import"},
			[]any{runOf("msg-0", runCompleted)}, []string{"message.dispatch"}, "what is new", false},
		{"an idle thread out of full access is set to it first", map[string]any{"runtimeMode": "approval-required"},
			[]any{runOf("msg-0", runCompleted)}, []string{"thread.runtime-mode.set", "message.dispatch"}, "what is new", false},
		{"a busy thread out of full access is left as it is, with a note", map[string]any{"runtimeMode": "approval-required"},
			[]any{runOf("msg-0", "running")}, []string{"message.dispatch"}, "what is new", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			done := begin(t, h, laptop(f), "th-1")
			subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
			f.Write(t3rpctest.Chunk(subID, snapshotWith(t, tt.thread, tt.runs...)))

			commands := commandsUntil(t, f, "message.dispatch")
			var types []string
			for _, c := range commands {
				types = append(types, c["type"].(string))
			}
			assert.Equal(t, tt.commands, types)
			if len(commands) == 2 {
				assert.Equal(t, map[string]any{"type": "thread.runtime-mode.set", "commandId": commands[0]["commandId"],
					"threadId": "th-1", "runtimeMode": "full-access"}, commands[0])
			}
			assert.Equal(t, tt.text, commands[len(commands)-1]["text"])

			s := <-done
			require.NoError(t, s.err)
			assert.Equal(t, "th-1", s.result.SessionID)
			notes := collect(s.result.Updates)
			if !tt.noted {
				assert.Empty(t, notes)
				return
			}
			require.Len(t, notes, 1)
			assert.Equal(t, harness.ActivityNote, notes[0].Activity.Kind)
			assert.Equal(t, notFullAccess, notes[0].Activity.Summary)
		})
	}
}

func TestStartTurn_NeverSteers(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	done := begin(t, h, laptop(f), "th-1")
	subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
	f.Write(t3rpctest.Chunk(subID, snapshotWith(t, nil, runOf("msg-0", "running"))))

	dispatch := t3rpctest.WaitFor(t, f.Dispatched, "message.dispatch")
	require.NoError(t, (<-done).err)
	assert.Equal(t, map[string]any{
		"type": "message.dispatch", "createdBy": "user", "creationSource": "web", "commandId": dispatch["commandId"],
		"threadId": "th-1", "messageId": dispatch["messageId"], "text": "what is new", "attachments": []any{},
		"dispatchMode": map[string]any{"type": "queue_after_active"},
	}, dispatch, "queued behind the running run, never steered into it, so the message gets a run of its own")
}

// missingThreadCause is the recorded failure of a subscribe to a thread that never existed.
func missingThreadCause(t *testing.T) any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "subscribe-missing-thread"+nightly))
	require.NoError(t, err)
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
		var rec struct {
			Frame struct {
				Exit struct {
					Tag   string `json:"_tag"`
					Cause any    `json:"cause"`
				} `json:"exit"`
			} `json:"frame"`
		}
		require.NoError(t, json.Unmarshal(line, &rec))
		if rec.Frame.Exit.Tag == "Failure" {
			return rec.Frame.Exit.Cause
		}
	}
	t.Fatal("the fixture holds no failure")
	return nil
}

func failExit(requestID string, cause any) map[string]any {
	return map[string]any{"_tag": "Exit", "requestId": requestID, "exit": map[string]any{"_tag": "Failure", "cause": cause}}
}

func TestStartTurn_GoneThread_IsRecreatedOnceWithTheFullPrompt(t *testing.T) {
	t.Parallel()
	deleted := recorded(t, "subscribe-deleted-thread"+nightly)
	tests := []struct {
		name   string
		answer func(t *testing.T, f *t3rpctest.Server, subID string)
	}{
		{"the thread no longer exists", func(t *testing.T, f *t3rpctest.Server, subID string) {
			f.Write(failExit(subID, missingThreadCause(t)))
		}},
		{"the snapshot says it was deleted", func(_ *testing.T, f *t3rpctest.Server, subID string) {
			f.Write(t3rpctest.Chunk(subID, deleted[2]))
		}},
		{"a delete arrives with the snapshot", func(_ *testing.T, f *t3rpctest.Server, subID string) {
			f.Write(t3rpctest.Chunk(subID, deleted[0], deleted[1]))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			done := begin(t, h, laptop(f), "th-1")
			tt.answer(t, f, t3rpctest.WaitFor(t, f.Subscribed, "subscribe to the stored thread"))

			create := t3rpctest.WaitFor(t, f.Dispatched, "thread.create")
			require.Equal(t, "thread.create", create["type"], "a soft-deleted thread still takes a message, so it is never dispatched to")
			fresh, _ := create["threadId"].(string)
			assert.NotEqual(t, "th-1", fresh)
			subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribe to the new thread")
			f.Write(t3rpctest.Chunk(subID, recorded(t, "thread-create"+nightly)[0]))

			dispatch := t3rpctest.WaitFor(t, f.Dispatched, "message.dispatch")
			assert.Equal(t, fresh, dispatch["threadId"])
			assert.Equal(t, "full prompt", dispatch["text"], "the new thread has none of the old one's context")
			s := <-done
			require.NoError(t, s.err)
			assert.Equal(t, fresh, s.result.SessionID)
		})
	}
}

func TestStartTurn_RecreatedThreadGoneToo_FailsWithoutAThirdThread(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	done := begin(t, h, laptop(f), "th-1")
	f.Write(failExit(t3rpctest.WaitFor(t, f.Subscribed, "first subscribe"), missingThreadCause(t)))
	t3rpctest.WaitFor(t, f.Dispatched, "thread.create")
	f.Write(failExit(t3rpctest.WaitFor(t, f.Subscribed, "second subscribe"), missingThreadCause(t)))

	s := <-done
	require.ErrorIs(t, s.err, errThreadGone)
	assert.Empty(t, f.Dispatched, "one replacement thread, and no message")
}

func TestStartTurn_DispatchRefused_FailsWithT3sMessage(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	f.CommandCauses = map[string]any{"message.dispatch": []any{map[string]any{"_tag": "Fail", "error": map[string]any{
		"_tag": "OrchestrationV2DispatchCommandError", "commandId": "cmd-2", "commandType": "message.dispatch",
		"message": "Thread th-1 is archived.", "detail": "Thread th-1 is archived."}}}}
	done := begin(t, h, laptop(f), "th-1")
	f.Write(t3rpctest.Chunk(t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"), snapshotWith(t, nil)))

	s := <-done
	require.ErrorIs(t, s.err, apperrs.ErrInvalid)
	assert.EqualError(t, s.err, "invalid: send the message to T3 Code: Thread th-1 is archived.")
	assert.Empty(t, f.Dispatched, "nothing was taken, and the turn never started")
}

func TestStartTurn_ConnectionDropsMidTurn_RedialsAndResumesAfterTheCursor(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	done := begin(t, h, laptop(f), "th-1")
	subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
	t3rpctest.WaitFor(t, f.SubscribeIn, "first subscribe input")
	f.Write(t3rpctest.Chunk(subID, snapshotWith(t, nil)))
	messageID, _ := t3rpctest.WaitFor(t, f.Dispatched, "message.dispatch")["messageId"].(string)
	s := <-done
	require.NoError(t, s.err)
	f.Write(t3rpctest.Chunk(subID, event(7, "run.created", runOf(messageID, "running"))))

	f.Drop()
	subID = t3rpctest.WaitFor(t, f.Subscribed, "resubscribe on a new connection")
	assert.Equal(t, map[string]any{"threadId": "th-1", "afterSequence": float64(7), "acceptBoundedSnapshot": true},
		t3rpctest.WaitFor(t, f.SubscribeIn, "resume input"))
	f.Write(t3rpctest.Chunk(subID, event(8, "run.updated", runOf(messageID, runWaiting))))
	assert.Equal(t, []harness.Update{ended(harness.TurnDone, "")}, drainUpdates(t, s.result.Updates))
}

func TestStartTurn_NewThreadNotCreated_FailsWithTheReason(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		provider string
		causes   map[string]any
		want     string
	}{
		{"T3 refuses the thread", "claudeAgent", map[string]any{"thread.create": []any{map[string]any{"_tag": "Fail", "error": map[string]any{
			"_tag": "OrchestrationV2DispatchCommandError", "commandType": "thread.create", "message": "Project pr-1 was not found."}}}},
			"invalid: create t3 thread: Project pr-1 was not found."},
		{"the provider is not on the computer, so no default model", "codex", nil, "invalid: provider codex not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			f.CommandCauses = tt.causes
			_, err := h.StartTurn(t.Context(), harness.Target{Session: laptop(f), ProjectID: "pr-1", Provider: tt.provider}, "Fix login", testPrompts)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.EqualError(t, err, tt.want)
			assert.Empty(t, f.Subscribed, "nothing to watch")
		})
	}
}

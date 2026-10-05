package t3clientv2

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"strings"
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
	return beginWith(t, h, s, sessionID, testPrompts)
}

func beginWith(t *testing.T, h *Harness, s harness.Session, sessionID string, prompts harness.TurnPrompts) <-chan started {
	t.Helper()
	return beginAs(t, h, harness.Target{Session: s, ProjectID: "pr-1", Provider: "claudeAgent", SessionID: sessionID}, prompts)
}

func beginAs(t *testing.T, h *Harness, target harness.Target, prompts harness.TurnPrompts) <-chan started {
	t.Helper()
	ch := make(chan started, 1)
	go func() {
		r, err := h.StartTurn(t.Context(), target, "Fix login", prompts)
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
			got = append(got, unmarked(u))
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
	assert.True(t, strings.HasPrefix(messageID, nexulMessagePrefix), "Nexul marks its own messages, so one without the mark was typed in T3")
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
	assert.Empty(t, f.Persisted, "a turn with no images makes no upload")
}

// threadOn is the selection a snapshot's thread runs on: its provider instance, model and options.
func threadOn(instance, model string, options ...map[string]any) map[string]any {
	selection := map[string]any{"instanceId": instance, "model": model}
	if options != nil {
		selection["options"] = options
	}
	return map[string]any{"providerInstanceId": instance, "modelSelection": selection}
}

func opt(id string, value any) map[string]any { return map[string]any{"id": id, "value": value} }

func TestStartTurn_ReusedThread_SendsTheSelectionOnlyWhenTheTargetAsksForAnother(t *testing.T) {
	t.Parallel()
	const old, other = "claude-fable-5-1", "claude-opus-5-5"
	effort, fast := opt("effort", "high"), opt("fastMode", true)
	tests := []struct {
		name    string
		thread  map[string]any
		model   string
		options []harness.OptionSetting
		want    map[string]any
		text    string
	}{
		{"the thread's own model and options are not sent again", threadOn("claudeAgent", old, effort), old,
			[]harness.OptionSetting{{ID: "effort", Value: "high"}}, nil, "what is new"},
		{"no model and no options keep the thread's", threadOn("claudeAgent", old, effort), "", nil, nil, "what is new"},
		{"the same options in another order are not another selection", threadOn("claudeAgent", old, effort, fast), "",
			[]harness.OptionSetting{{ID: "fastMode", Value: true}, {ID: "effort", Value: "high"}}, nil, "what is new"},
		{"another model is sent", threadOn("claudeAgent", old), other, nil,
			map[string]any{"instanceId": "claudeAgent", "model": other}, "what is new"},
		{"another model starts on its own defaults, not the old model's options", threadOn("claudeAgent", old, effort), other, nil,
			map[string]any{"instanceId": "claudeAgent", "model": other}, "what is new"},
		{"other options on an unnamed model are sent with the thread's model", threadOn("claudeAgent", old, effort), "",
			[]harness.OptionSetting{{ID: "effort", Value: "low"}},
			map[string]any{"instanceId": "claudeAgent", "model": old, "options": []any{opt("effort", "low")}}, "what is new"},
		{"an option left out goes back to its default", threadOn("claudeAgent", old, effort, fast), "",
			[]harness.OptionSetting{{ID: "effort", Value: "high"}},
			map[string]any{"instanceId": "claudeAgent", "model": old, "options": []any{effort}}, "what is new"},
		{"another provider instance gets its default model, no options, and the full prompt", threadOn("codex", "gpt-6", effort), "", nil,
			map[string]any{"instanceId": "claudeAgent", "model": other}, "full prompt"},
		{"another provider instance gets the model and options as picked, and the full prompt", threadOn("codex", "gpt-6"), other,
			[]harness.OptionSetting{{ID: "effort", Value: "max"}},
			map[string]any{"instanceId": "claudeAgent", "model": other, "options": []any{opt("effort", "max")}}, "full prompt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			done := beginAs(t, h, harness.Target{Session: laptop(f), ProjectID: "pr-1", Provider: "claudeAgent", Model: tt.model,
				ModelOptions: tt.options, SessionID: "th-1"}, testPrompts)
			f.Write(t3rpctest.Chunk(t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"), snapshotWith(t, tt.thread)))

			dispatch := t3rpctest.WaitFor(t, f.Dispatched, "message.dispatch")
			require.NoError(t, (<-done).err)
			if tt.want == nil {
				assert.NotContains(t, dispatch, "modelSelection")
			} else {
				assert.Equal(t, tt.want, dispatch["modelSelection"])
			}
			assert.Equal(t, tt.text, dispatch["text"])
		})
	}
}

func TestStartTurn_ReusedThreadOnAnotherProviderThatIsNotThere_FailsBeforeUploadingOrSending(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	prompts := testPrompts
	prompts.Attachments = []harness.Attachment{{Name: "shot.png", MIME: "image/png", Bytes: []byte{1}}}
	done := beginAs(t, h, harness.Target{Session: laptop(f), ProjectID: "pr-1", Provider: "codex", SessionID: "th-1"}, prompts)
	f.Write(t3rpctest.Chunk(t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"), snapshotWith(t, nil)))

	s := <-done
	require.ErrorIs(t, s.err, apperrs.ErrInvalid)
	assert.EqualError(t, s.err, "invalid: provider codex not found")
	assert.Empty(t, f.Persisted, "the images stay with the caller")
	assert.Empty(t, f.Dispatched, "an empty model would reach T3, which rejects it, so nothing is sent")
}

func TestStartTurn_Images_AreUploadedFirstAndTheirReferencesSentVerbatim(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	prompts := testPrompts
	prompts.Attachments = []harness.Attachment{
		{Name: "shot.png", MIME: "image/png", Bytes: []byte{0x89, 'P', 'N', 'G'}},
		{Name: "photo.JPG", MIME: "IMAGE/JPEG", Bytes: []byte{0xff, 0xd8, 0xff}},
	}
	done := beginWith(t, h, laptop(f), "", prompts)
	threadID, _ := t3rpctest.WaitFor(t, f.Dispatched, "thread.create")["threadId"].(string)
	f.Write(t3rpctest.Chunk(t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"), snapshotWith(t, nil)))

	dispatch := t3rpctest.WaitFor(t, f.Dispatched, "message.dispatch")
	assert.Equal(t, map[string]any{"threadId": threadID, "messageId": dispatch["messageId"], "attachments": []any{
		map[string]any{"type": "image", "name": "shot.png", "mimeType": "image/png", "sizeBytes": float64(4), "dataUrl": "data:image/png;base64,iVBORw=="},
		map[string]any{"type": "image", "name": "photo.JPG", "mimeType": "image/jpeg", "sizeBytes": float64(3), "dataUrl": "data:image/jpeg;base64,/9j/"},
	}}, t3rpctest.WaitFor(t, f.Persisted, "the upload"), "T3 reads the mime off the data URL and wants it lowercased to match mimeType")
	assert.Equal(t, []any{
		map[string]any{"type": "image", "id": threadID + "-ref-0", "name": "shot.png", "mimeType": "image/png", "sizeBytes": float64(4)},
		map[string]any{"type": "image", "id": threadID + "-ref-1", "name": "photo.JPG", "mimeType": "image/jpeg", "sizeBytes": float64(3)},
	}, dispatch["attachments"], "the message carries what T3 answered, ids it alone can mint")
	assert.Equal(t, "full prompt", dispatch["text"])
	require.NoError(t, (<-done).err)
}

func TestStartTurn_Images_OnlyWhatTheSentPromptRefersToAndT3Takes(t *testing.T) {
	t.Parallel()
	svg := harness.Attachment{Name: "logo.svg", MIME: "image/svg+xml", Bytes: []byte("<svg/>")}
	bmp := harness.Attachment{Name: "scan.bmp", MIME: "image/bmp", Bytes: []byte("BM")}
	gif := harness.Attachment{Name: "loop.gif", MIME: "image/gif", Bytes: []byte("GIF")}
	webp := harness.Attachment{Name: "page.webp", MIME: "image/webp", Bytes: []byte("RIFF")}
	tests := []struct {
		name      string
		sessionID string
		thread    map[string]any
		images    []harness.Attachment
		uploaded  []string
		note      string
	}{
		{"an unsupported type is left out and noted", "", nil, []harness.Attachment{svg, gif, bmp, webp},
			[]string{"loop.gif", "page.webp"}, "Not sent to T3 Code: logo.svg, scan.bmp. It takes only gif, jpeg, png and webp images."},
		{"nothing supported means no upload", "", nil, []harness.Attachment{svg},
			nil, "Not sent to T3 Code: logo.svg. It takes only gif, jpeg, png and webp images."},
		{"a reused thread already holds them", "th-1", nil, []harness.Attachment{gif}, nil, ""},
		{"an imported thread gets them with its full prompt", "th-1", map[string]any{"historyOrigin": "v1_import"}, []harness.Attachment{gif},
			[]string{"loop.gif"}, ""},
		{"a thread on another provider instance gets them with its full prompt", "th-1", threadOn("codex", "gpt-6"), []harness.Attachment{gif},
			[]string{"loop.gif"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			prompts := testPrompts
			prompts.Attachments = tt.images
			done := beginWith(t, h, laptop(f), tt.sessionID, prompts)
			f.Write(t3rpctest.Chunk(t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"), snapshotWith(t, tt.thread)))
			commands := commandsUntil(t, f, "message.dispatch")
			dispatch := commands[len(commands)-1]

			var uploaded []string
			for len(f.Persisted) > 0 {
				for _, a := range (<-f.Persisted)["attachments"].([]any) {
					uploaded = append(uploaded, a.(map[string]any)["name"].(string))
				}
			}
			assert.Equal(t, tt.uploaded, uploaded)
			assert.Len(t, dispatch["attachments"], len(tt.uploaded))
			s := <-done
			require.NoError(t, s.err)
			notes := collect(s.result.Updates)
			if tt.note == "" {
				assert.Empty(t, notes)
				return
			}
			require.Len(t, notes, 1)
			assert.Equal(t, harness.ActivityNote, notes[0].Activity.Kind)
			assert.Equal(t, tt.note, notes[0].Activity.Summary)
		})
	}
}

func TestStartTurn_ImageUploadRefused_FailsWithT3sMessageAndSendsNothing(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	f.CommandCauses = map[string]any{"assets.persistChatAttachments": []any{map[string]any{"_tag": "Fail", "error": map[string]any{
		"_tag": "PersistChatAttachmentsError", "message": "Attachment shot.png has an invalid image payload."}}}}
	prompts := testPrompts
	prompts.Attachments = []harness.Attachment{{Name: "shot.png", MIME: "image/png", Bytes: []byte{1}}}
	done := beginWith(t, h, laptop(f), "th-1", prompts)
	f.Write(t3rpctest.Chunk(t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"), snapshotWith(t, map[string]any{"historyOrigin": "v1_import"})))

	s := <-done
	require.ErrorIs(t, s.err, apperrs.ErrInvalid)
	assert.EqualError(t, s.err, "invalid: upload the images to T3 Code: Attachment shot.png has an invalid image payload.")
	assert.Len(t, f.Persisted, 1)
	assert.Empty(t, f.Dispatched, "the agent never gets a prompt that points at images it cannot see")
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
	// A chunk is acked on the pump's next read, after the watch applied it: the snapshot's ack, then event 7's.
	for acked := 0; acked < 2; {
		if t3rpctest.WaitFor(t, f.Acks, "acks for the snapshot and event 7") == subID {
			acked++
		}
	}

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

func worktreeTarget(f *t3rpctest.Server) harness.Target {
	return harness.Target{Session: laptop(f), ProjectID: "proj-live", Provider: "claudeAgent", Worktree: true}
}

func TestStartTurn_Worktree_LaunchesOnTheFoldersBranchWithThePromptAndWatchesItsRun(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	f.Branches = map[string]string{"/home/me/app": "master"}
	done := beginAs(t, h, worktreeTarget(f), testPrompts)

	launch := t3rpctest.WaitFor(t, f.Launched, "launchThread")
	threadID, _ := launch["threadId"].(string)
	message, _ := launch["initialMessage"].(map[string]any)
	messageID, _ := message["messageId"].(string)
	require.NotEmpty(t, threadID)
	require.NotEmpty(t, messageID)
	assert.Equal(t, map[string]any{
		"commandId": launch["commandId"], "creationSource": "web", "threadId": threadID, "projectId": "proj-live", "title": "Fix login",
		"modelSelection": map[string]any{"instanceId": "claudeAgent", "model": "claude-opus-5-5"},
		"runtimeMode":    "full-access", "interactionMode": "default",
		"workspaceStrategy": map[string]any{"type": "worktree", "baseRef": "master"},
		"initialMessage":    map[string]any{"messageId": messageID, "text": "full prompt", "attachments": []any{}},
	}, launch, "T3 holds back only a launched message until the worktree is ready")

	subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
	f.Write(t3rpctest.Chunk(subID, snapshotWith(t, nil)))
	s := <-done
	require.NoError(t, s.err)
	assert.Equal(t, threadID, s.result.SessionID)
	assert.True(t, s.result.PromptSent)
	assert.Empty(t, f.Dispatched, "the prompt went with the launch, so nothing is created or dispatched")

	f.Write(t3rpctest.Chunk(subID,
		event(3, "run.created", runOf(messageID, "preparing")),
		event(4, "run.updated", runOf(messageID, "running")),
		event(5, "turn-item.updated", assistantItem("Done.", false)),
		event(6, "run.updated", runOf(messageID, runWaiting)),
	))
	assert.Equal(t, []harness.Update{snapshotOf(codexMessage, "Done.", false), ended(harness.TurnDone, "")}, drainUpdates(t, s.result.Updates))
}

func TestStartTurn_WorktreeButFolderOnNoBranch_StartsInTheFolderWithANote(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	done := beginAs(t, h, worktreeTarget(f), testPrompts)

	t3rpctest.WaitFor(t, f.Dispatched, "thread.create")
	subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
	f.Write(t3rpctest.Chunk(subID, snapshotWith(t, nil)))
	assert.Equal(t, "full prompt", t3rpctest.WaitFor(t, f.Dispatched, "message.dispatch")["text"])
	s := <-done
	require.NoError(t, s.err)
	assert.Empty(t, f.Launched)
	first := <-s.result.Updates
	require.NotNil(t, first.Activity)
	assert.Equal(t, noBranchNote, first.Activity.Summary)
}

func TestStartTurn_WorktreeLaunchRefused_FailsWithT3sMessage(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	f.Branches = map[string]string{"/home/me/app": "master"}
	f.CommandCauses = map[string]any{"orchestration.launchThread": []any{map[string]any{"_tag": "Fail", "error": map[string]any{
		"_tag": "ThreadLaunchError", "message": "Thread launch failed during provision-worktree."}}}}
	_, err := h.StartTurn(t.Context(), worktreeTarget(f), "Fix login", testPrompts)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.EqualError(t, err, "invalid: start the t3 thread in a new worktree: Thread launch failed during provision-worktree.")
	assert.Empty(t, f.Subscribed, "nothing to watch")
}

func TestStartTurn_ApprovalRequest_IsDeclinedOnce(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		causes map[string]any
	}{
		{"T3 takes the decline", nil},
		{"a refused decline still lets the turn finish", map[string]any{"runtime-request.respond": []any{map[string]any{"_tag": "Fail", "error": map[string]any{
			"_tag": "OrchestrationV2DispatchCommandError", "commandType": "runtime-request.respond", "message": "Runtime request is resolved."}}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			f.CommandCauses = tt.causes
			done := begin(t, h, laptop(f), "th-1")
			subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
			f.Write(t3rpctest.Chunk(subID, snapshotWith(t, nil)))
			messageID, _ := t3rpctest.WaitFor(t, f.Dispatched, "message.dispatch")["messageId"].(string)
			s := <-done
			require.NoError(t, s.err)

			asking := recordedItem(t, toolSteps, approvalItem, "waiting")
			declined := recordedItem(t, toolSteps, approvalItem, "cancelled")
			asking.RunID, declined.RunID = runOne, runOne
			f.Write(t3rpctest.Chunk(subID,
				event(3, "run.created", runOf(messageID, "running")),
				event(4, "turn-item.updated", asking), event(5, "turn-item.updated", asking), event(6, "turn-item.updated", declined),
				event(7, "run.updated", runOf(messageID, runWaiting)),
			))
			assert.Equal(t, []harness.Update{{Approval: &harness.Approval{Kind: "command", Summary: approvalPrompt}}, ended(harness.TurnDone, "")},
				drainUpdates(t, s.result.Updates))
			if tt.causes != nil {
				return
			}
			respond := t3rpctest.WaitFor(t, f.Dispatched, "runtime-request.respond")
			assert.Equal(t, map[string]any{"type": "runtime-request.respond", "commandId": respond["commandId"], "threadId": "th-1",
				"requestId": approval1, "decision": "decline"}, respond)
			assert.Empty(t, f.Dispatched, "declined once")
		})
	}
}

// watchThread calls Watch on th-1 and answers its subscribe with first; Watch returns once that snapshot is in.
func watchThread(t *testing.T, f *t3rpctest.Server, h *Harness, first json.RawMessage) (string, harness.StartResult) {
	t.Helper()
	done := make(chan started, 1)
	go func() {
		r, err := h.Watch(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1"})
		done <- started{r, err}
	}()
	subID := t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread")
	f.Write(t3rpctest.Chunk(subID, first))
	s := <-done
	require.NoError(t, s.err)
	assert.Equal(t, "th-1", s.result.SessionID)
	assert.False(t, s.result.PromptSent)
	return subID, s.result
}

// threadAt is th-1's snapshot holding runs and turn items, out of full access so a runtime-mode change would show.
func threadAt(runs []any, extra map[string]any, turnItems ...any) json.RawMessage {
	p := map[string]any{"thread": map[string]any{"id": "th-1", "runtimeMode": "approval-required", "deletedAt": nil},
		"runs": runs, "turnItems": append([]any{}, turnItems...), "providerSessions": []any{}}
	maps.Copy(p, extra)
	return snapshotItem(2, p)
}

// shellStep is a finished command of run 1.
func shellStep(id, command string) map[string]any {
	return map[string]any{"id": id, "threadId": "th-1", "runId": runOne, "nodeId": "node:" + id, "ordinal": 1, "status": "completed",
		"type": "command_execution", "input": command, "updatedAt": "2026-10-03T16:00:00.000Z"}
}

func TestWatch_NoThreadToFollow_FailsWithoutSendingAnything(t *testing.T) {
	t.Parallel()
	deleted := recorded(t, "subscribe-deleted-thread"+nightly)
	tests := []struct {
		name   string
		answer func(t *testing.T, f *t3rpctest.Server, subID string)
		errIs  error
	}{
		{"the thread no longer exists", func(t *testing.T, f *t3rpctest.Server, subID string) {
			f.Write(failExit(subID, missingThreadCause(t)))
		}, errThreadGone},
		{"the snapshot says it was deleted", func(_ *testing.T, f *t3rpctest.Server, subID string) {
			f.Write(t3rpctest.Chunk(subID, deleted[2]))
		}, errThreadGone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			done := make(chan error, 1)
			go func() {
				_, err := h.Watch(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1"})
				done <- err
			}()
			tt.answer(t, f, t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"))

			require.ErrorIs(t, <-done, tt.errIs)
			assert.Empty(t, f.Dispatched, "a gone thread is never recreated by a watch")
			assert.Empty(t, f.Launched)
		})
	}
}

func TestWatch_NoSession_IsInvalid(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	_, err := h.Watch(t.Context(), harness.Target{Session: laptop(f)})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Empty(t, f.Dialed, "nothing to watch, so nothing is dialed")
}

func TestWatch_RunStillRunning_FollowsItToItsEndAndSendsNothing(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	subID, r := watchThread(t, f, h, threadAt([]any{runOf("msg-1", "running")}, nil, shellStep("step-1", "ls"), assistantItem("Hel", true)))

	f.Write(t3rpctest.Chunk(subID,
		event(3, "turn-item.updated", shellStep("step-1", "ls")),
		event(4, "turn-item.updated", shellStep("step-2", "go test ./...")),
		event(5, "turn-item.updated", assistantItem("Hello there.", false)),
		event(6, "run.updated", runOf("msg-1", runWaiting)),
	))
	assert.Equal(t, []string{"reply Hel", "tool_result go test ./...", "reply Hello there.", "end done"}, labels(drainUpdates(t, r.Updates)),
		"the partial reply carries on and the step shown before the watch is not shown again")
	assert.Empty(t, f.Dispatched, "no message, and the runtime mode is left as it is")
	assert.Empty(t, f.Launched)
	assert.Empty(t, f.Persisted)
}

func TestWatch_RunAskingAQuestion_RaisesItAgain(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	asking := recordedItem(t, toolSteps, questionItem, "waiting")
	asking.RunID = runOne
	_, r := watchThread(t, f, h, threadAt([]any{runOf("msg-1", "running")}, nil, asking))

	u := t3rpctest.WaitFor(t, r.Updates, "the open question")
	require.NotNil(t, u.Question, "the run waits on it, so the turn parks again")
	assert.Equal(t, question1, u.Question.RequestID)
}

func TestWatch_RunAlreadyOver_EndsAtOnceWithItsFinalReply(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		runs      []any
		turnItems []any
		want      []harness.Update
	}{
		{"completed", []any{runOf("msg-1", runCompleted)}, []any{shellStep("step-1", "ls"), assistantItem("Hello there.", false)},
			[]harness.Update{snapshotOf(codexMessage, "Hello there.", false), ended(harness.TurnDone, "")}},
		{"waiting on nothing handed off", []any{runOf("msg-1", runWaiting)}, []any{assistantItem("Hello there.", false)},
			[]harness.Update{snapshotOf(codexMessage, "Hello there.", false), ended(harness.TurnDone, "")}},
		{"interrupted mid-reply, which is no longer streaming", []any{runOf("msg-1", "interrupted")}, []any{assistantItem("Hel", true)},
			[]harness.Update{snapshotOf(codexMessage, "Hel", false), ended(harness.TurnInterrupted, "")}},
		{"cancelled", []any{runOf("msg-1", "cancelled")}, nil, []harness.Update{ended(harness.TurnInterrupted, "")}},
		{"failed", []any{runOf("msg-1", "failed")}, nil, []harness.Update{ended(harness.TurnError, noReason)}},
		{"no run at all", []any{}, nil, []harness.Update{ended(harness.TurnDone, "")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			_, r := watchThread(t, f, h, threadAt(tt.runs, nil, tt.turnItems...))

			assert.Equal(t, tt.want, drainUpdates(t, r.Updates))
			watching := 0
			h.turns.Range(func(any, any) bool { watching++; return true })
			assert.Zero(t, watching, "nothing left for Stop to reach")
			assert.Empty(t, f.Dispatched)
		})
	}
}

func TestWatch_NewestRunIsAWakeOfAnEarlierRun_FollowsTheRunThatHandedOff(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	linked := map[string]any{"delegatedCompletion": map[string]any{"parentRunId": runOne, "generation": 1, "taskIds": []any{"task-1"}}}
	subID, r := watchThread(t, f, h, threadAt([]any{runOf("msg-1", runWaiting), runAt(2, wakeMessage, "running")}, map[string]any{
		"subagents": []any{handedOff("task-1", "app_owned", "completed", delivery("claimed"))},
		"messages":  []any{userMessage(wakeMessage, 2, linked)},
	}, replyIn(1, "Handed the audit off.")))

	f.Write(t3rpctest.Chunk(subID,
		event(3, "turn-item.updated", replyIn(2, "The audit found three issues.")),
		event(4, "subagent.updated", handedOff("task-1", "app_owned", "completed", delivery("delivered"))),
		event(5, "run.updated", runAt(2, wakeMessage, runWaiting)),
	))
	assert.Equal(t, []string{"reply Handed the audit off.", "note " + handoffNote, "reply The audit found three issues.", "end done"},
		labels(drainUpdates(t, r.Updates)))
}

func TestWatch_AfterAT3Restart_FollowsTheRunT3RanLast(t *testing.T) {
	t.Parallel()
	// msg-2 queued behind msg-1; the restart cut msg-1, held msg-2, and ran msg-1's continuation ahead of it.
	cut := merged(runAt(1, "msg-1", "cancelled"), map[string]any{"completedAt": "2026-10-03T16:00:00.000Z"})
	continued := merged(runAt(3, "message:restart-continuation:"+runOne, runCompleted), map[string]any{
		"restartContinuationOfRunId": runOne, "completedAt": "2026-10-03T16:05:00.000Z"})
	resumed := merged(runAt(2, "msg-2", runCompleted), map[string]any{"completedAt": "2026-10-03T16:09:00.000Z"})
	tests := []struct {
		name  string
		runs  []any
		items []any
		then  []any
	}{
		{"the held run resumed and still running", []any{cut, runAt(2, "msg-2", "running"), continued},
			[]any{replyIn(3, "Continued after the restart.")},
			[]any{event(3, "turn-item.updated", replyIn(2, "Answered the queued ask.")), event(4, "run.updated", runAt(2, "msg-2", runWaiting))}},
		{"the held run resumed and completed after the continuation", []any{cut, resumed, continued},
			[]any{replyIn(3, "Continued after the restart."), replyIn(2, "Answered the queued ask.")}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			subID, r := watchThread(t, f, h, threadAt(tt.runs, nil, tt.items...))
			if tt.then != nil {
				f.Write(t3rpctest.Chunk(subID, tt.then...))
			}
			assert.Equal(t, []string{"reply Answered the queued ask.", "end done"}, labels(drainUpdates(t, r.Updates)),
				"not the cut run's continuation, which T3 ran before it")
		})
	}
}

func TestWatch_Interrupted_StopsTheWatchedRunAndEndsInterrupted(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	_, r := watchThread(t, f, h, threadAt([]any{runAt(1, "msg-1", "running")}, nil))
	f.Projections = map[string]any{"th-1": projectionWith(t, []any{runAt(1, "msg-1", "running")})}

	require.NoError(t, h.Interrupt(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1", TurnID: r.TurnID}))
	cmd := t3rpctest.WaitFor(t, f.Dispatched, "run.interrupt")
	assert.Equal(t, "run.interrupt", cmd["type"])
	assert.Equal(t, runOne, cmd["runId"])
	assert.Equal(t, []harness.Update{ended(harness.TurnInterrupted, "")}, drainUpdates(t, r.Updates),
		"Stop ends the watched turn at once, without waiting for T3 to report the run's end")
}

func TestAdoptedSince_PicksTheFirstRunAfterTheMarker(t *testing.T) {
	t.Parallel()
	runs := func(lastStatus string) projection {
		return projection{Runs: []run{
			{ID: "r-3", Ordinal: 3, UserMessageID: "m-3", Status: lastStatus},
			{ID: "r-1", Ordinal: 1, UserMessageID: "m-1", Status: runCompleted},
			{ID: "r-2", Ordinal: 2, UserMessageID: "m-2", Status: runCompleted},
		}}
	}
	tests := []struct {
		name, since, lastStatus, want string
	}{
		{"the run right after the marker, whatever came later", "r-1", runCompleted, "m-2"},
		{"the marker's own run while it still runs", "r-3", "running", "m-3"},
		{"nothing once the marker's run is the newest and over", "r-3", runCompleted, ""},
		{"the oldest the snapshot holds once the marker fell out of it", "r-0", runCompleted, "m-1"},
		{"the newest without a marker, as after a restart", "", runCompleted, "m-3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tr := &turn{target: harness.Target{Since: tt.since}}
			assert.Equal(t, tt.want, tr.adoptedSince(runs(tt.lastStatus)))
		})
	}
}

// drainEnd reads a turn to its end and returns its terminal result with the Marker kept.
func drainEnd(t *testing.T, updates <-chan harness.Update) (labels []string, end *harness.TurnResult) {
	t.Helper()
	for {
		select {
		case u, ok := <-updates:
			if !ok {
				return labels, end
			}
			if u.Terminal != nil {
				end = u.Terminal
				continue
			}
			labels = append(labels, label(u))
		case <-time.After(5 * time.Second):
			t.Fatalf("the turn never ended; got %v", labels)
		}
	}
}

func label(u harness.Update) string {
	if u.Snapshot != nil {
		return "reply " + u.Snapshot.Text
	}
	return string(u.Activity.Kind) + " " + u.Activity.Summary
}

func TestWatch_SinceAMarker_CatchesUpOnWhatRanAfterItStepsIncluded(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	typedStep := shellStep("step-2", "go test ./...")
	typedStep["runId"], typedStep["ordinal"] = "run:thread:th-1:ordinal:2", 2000
	snapshot := threadAt([]any{runOf(nexulMessagePrefix+"play", runCompleted), runAt(2, "msg-typed", runCompleted)}, nil,
		shellStep("step-1", "ls"), replyIn(1, "Which policy?"),
		typedIn(2, "msg-typed", "Keep two threads"), typedStep, replyIn(2, "Built the worker."))
	done := make(chan started, 1)
	go func() {
		r, err := h.Watch(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1", Since: "run:thread:th-1:ordinal:0"})
		done <- started{r, err}
	}()
	f.Write(t3rpctest.Chunk(t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"), snapshot))
	s := <-done
	require.NoError(t, s.err)

	labels, end := drainEnd(t, s.result.Updates)
	assert.Equal(t, []string{"tool_result ls", "reply Which policy?", "user_message Keep two threads", "tool_result go test ./...", "reply Built the worker."}, labels,
		"a catch-up shows the steps no watcher saw, and the typed run joins the play's")
	require.NotNil(t, end)
	assert.Equal(t, harness.TurnDone, end.State)
	assert.Equal(t, "run:thread:th-1:ordinal:2", end.Marker, "the next catch-up starts after the typed run")
}

func TestWatch_SinceTheNewestFinishedRun_EndsDoneWithNothingToShow(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	done := make(chan started, 1)
	go func() {
		r, err := h.Watch(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1", Since: runOne})
		done <- started{r, err}
	}()
	f.Write(t3rpctest.Chunk(t3rpctest.WaitFor(t, f.Subscribed, "subscribeThread"),
		threadAt([]any{runOf("msg-1", runCompleted)}, nil, replyIn(1, "Already posted."))))
	s := <-done
	require.NoError(t, s.err)

	labels, end := drainEnd(t, s.result.Updates)
	assert.Empty(t, labels, "the reply that run gave was posted when it ended")
	assert.Equal(t, &harness.TurnResult{State: harness.TurnDone, Marker: runOne}, end)
}

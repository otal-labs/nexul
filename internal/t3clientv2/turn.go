package t3clientv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/t3rpc"
)

const (
	dispatchCommand = "orchestration.dispatchCommand"
	subscribeThread = "orchestration.subscribeThread"
	fullAccess      = "full-access"
	// missingThread is the failure an initial subscribe ends with when the thread does not exist.
	missingThread = "OrchestrationV2GetThreadProjectionError"
	notFullAccess = "This T3 thread is not in full access"
	// defaultTitle stands in for an empty title, which thread.create refuses.
	defaultTitle = "Nexul chat"
	// cancelWithin bounds cancelling the queued run of a turn that stopped watching, whose own context may be over.
	cancelWithin = 10 * time.Second
)

// errThreadGone is a reused thread that no longer exists or was deleted in T3 Code.
var errThreadGone = errors.New("the T3 thread is gone")

type modelSelection struct {
	InstanceID string                  `json:"instanceId"`
	Model      string                  `json:"model"`
	Options    []harness.OptionSetting `json:"options,omitempty"`
}

// threadCreate sends every key T3 requires; branch and worktreePath must be present as null.
type threadCreate struct {
	Type            string         `json:"type"`
	CreatedBy       string         `json:"createdBy"`
	CreationSource  string         `json:"creationSource"`
	CommandID       string         `json:"commandId"`
	ThreadID        string         `json:"threadId"`
	ProjectID       string         `json:"projectId"`
	Title           string         `json:"title"`
	ModelSelection  modelSelection `json:"modelSelection"`
	RuntimeMode     string         `json:"runtimeMode"`
	InteractionMode string         `json:"interactionMode"`
	Branch          *string        `json:"branch"`
	WorktreePath    *string        `json:"worktreePath"`
}

// messageDispatch always queues behind an active run: a steered message gets no run of its own to watch.
type messageDispatch struct {
	Type           string            `json:"type"`
	CreatedBy      string            `json:"createdBy"`
	CreationSource string            `json:"creationSource"`
	CommandID      string            `json:"commandId"`
	ThreadID       string            `json:"threadId"`
	MessageID      string            `json:"messageId"`
	Text           string            `json:"text"`
	Attachments    []json.RawMessage `json:"attachments"`
	DispatchMode   dispatchMode      `json:"dispatchMode"`
}

type dispatchMode struct {
	Type string `json:"type"`
}

type runtimeModeSet struct {
	Type        string `json:"type"`
	CommandID   string `json:"commandId"`
	ThreadID    string `json:"threadId"`
	RuntimeMode string `json:"runtimeMode"`
}

type runtimeRequestRespond struct {
	Type      string `json:"type"`
	CommandID string `json:"commandId"`
	ThreadID  string `json:"threadId"`
	RequestID string `json:"requestId"`
	Decision  string `json:"decision"`
}

type subscribeInput struct {
	ThreadID              string `json:"threadId"`
	AfterSequence         *int64 `json:"afterSequence,omitempty"`
	AcceptBoundedSnapshot bool   `json:"acceptBoundedSnapshot"`
}

// turn is one StartTurn: the connection it owns, the thread, and the message id whose run is the turn.
type turn struct {
	h         *Harness
	target    harness.Target
	conn      *t3rpc.Conn
	threadID  string
	messageID string
}

// StartTurn implements harness.Client: the message goes out after the thread's snapshot, and its run is the turn.
func (h *Harness) StartTurn(ctx context.Context, target harness.Target, title string, prompts harness.TurnPrompts) (harness.StartResult, error) {
	c, err := h.connect(ctx, target.Session)
	if err != nil {
		return harness.StartResult{}, err
	}
	t := &turn{h: h, target: target, conn: c, messageID: ids.New()}
	src, w, notes, err := t.start(ctx, title, prompts)
	if err != nil {
		t.close()
		return harness.StartResult{}, err
	}
	h.messages.Store(t.threadID, t.messageID)
	updates := make(chan harness.Update, 16)
	for _, n := range notes {
		updates <- n
	}
	p := &pump{w: w, open: t.open, decline: t.decline, log: h.log()}
	go func() {
		p.run(ctx, src, updates)
		t.end(ctx, w)
	}()
	return harness.StartResult{SessionID: t.threadID, Updates: updates}, nil
}

// start creates or reuses the thread, reads its snapshot, and dispatches the prompt that snapshot calls for.
func (t *turn) start(ctx context.Context, title string, prompts harness.TurnPrompts) (source, *watch, []harness.Update, error) {
	fresh := t.target.SessionID == ""
	t.threadID = t.target.SessionID
	if fresh {
		if err := t.create(ctx, title); err != nil {
			return nil, nil, nil, err
		}
	}
	src, w, err := t.subscribe(ctx)
	if errors.Is(err, errThreadGone) && !fresh {
		t.h.log().Info("t3clientv2: reused thread is gone, creating a new one", "thread", t.threadID)
		fresh = true
		if err := t.create(ctx, title); err != nil {
			return nil, nil, nil, err
		}
		src, w, err = t.subscribe(ctx)
	}
	if err != nil {
		return nil, nil, nil, fmt.Errorf("watch t3 thread %s: %w", t.threadID, err)
	}
	notes, err := t.dispatch(ctx, w, fresh, prompts)
	if err != nil {
		src.Close()
		return nil, nil, nil, err
	}
	return src, w, notes, nil
}

// create makes the turn's thread, resolving an empty model to the provider's default first.
func (t *turn) create(ctx context.Context, title string) error {
	model := t.target.Model
	if model == "" {
		providers, err := t.conn.Providers()
		if err != nil {
			return fmt.Errorf("list t3 providers: %w", err)
		}
		if model, err = t3rpc.DefaultModel(providers, t.target.Provider); err != nil {
			return err
		}
	}
	if strings.TrimSpace(title) == "" {
		title = defaultTitle
	}
	t.threadID = ids.New()
	_, err := t.conn.Call(ctx, dispatchCommand, threadCreate{
		Type: "thread.create", CreatedBy: "user", CreationSource: "web", CommandID: ids.New(), ThreadID: t.threadID,
		ProjectID: t.target.ProjectID, Title: title,
		ModelSelection: modelSelection{InstanceID: t.target.Provider, Model: model, Options: t.target.ModelOptions},
		RuntimeMode:    fullAccess, InteractionMode: "default",
	})
	if err != nil {
		return refused("create t3 thread", err)
	}
	return nil
}

// subscribe opens the watch and reads up to the thread's first snapshot; errThreadGone when there is no live thread.
func (t *turn) subscribe(ctx context.Context) (source, *watch, error) {
	src, err := t.conn.Stream(ctx, subscribeThread, subscribeInput{ThreadID: t.threadID, AcceptBoundedSnapshot: true})
	if err != nil {
		return nil, nil, err
	}
	w := newWatch(t.messageID)
	for !w.synced {
		values, err := src.Next(ctx)
		var exit *t3rpc.ExitError
		if errors.As(err, &exit) && exit.Failed(missingThread) {
			src.Close()
			return nil, nil, errThreadGone
		}
		if err != nil {
			src.Close()
			return nil, nil, err
		}
		for _, raw := range values {
			w.apply(decodeItem(t.h.log(), raw))
		}
	}
	// A deleted thread still takes a message, so only its snapshot or a delete event says it is gone.
	if w.thread.DeletedAt != nil {
		src.Close()
		return nil, nil, errThreadGone
	}
	return src, w, nil
}

// dispatch uploads the prompt's images, then sends the prompt the snapshot calls for once the thread is in full access or noted as not.
func (t *turn) dispatch(ctx context.Context, w *watch, fresh bool, prompts harness.TurnPrompts) ([]harness.Update, error) {
	text, images := prompts.Incremental, []harness.Attachment(nil)
	if fresh || w.imported() {
		text, images = prompts.Full, prompts.Attachments
	}
	// Uploaded before anything touches the thread, so a refused upload leaves it as it was.
	refs, notes, err := t.persistImages(ctx, images)
	if err != nil {
		return nil, err
	}
	more, err := t.fullAccess(ctx, w)
	if err != nil {
		return nil, err
	}
	_, err = t.conn.Call(ctx, dispatchCommand, messageDispatch{
		Type: "message.dispatch", CreatedBy: "user", CreationSource: "web", CommandID: ids.New(), ThreadID: t.threadID,
		MessageID: t.messageID, Text: text, Attachments: refs, DispatchMode: dispatchMode{Type: "queue_after_active"},
	})
	if err != nil {
		return nil, refused("send the message to T3 Code", err)
	}
	return append(notes, more...), nil
}

// fullAccess sets the thread to full access, unless that would detach a queued or live run; then it notes it.
func (t *turn) fullAccess(ctx context.Context, w *watch) ([]harness.Update, error) {
	if w.thread.RuntimeMode == fullAccess {
		return nil, nil
	}
	if w.busy() {
		return []harness.Update{note(notFullAccess)}, nil
	}
	_, err := t.conn.Call(ctx, dispatchCommand, runtimeModeSet{Type: "thread.runtime-mode.set", CommandID: ids.New(), ThreadID: t.threadID, RuntimeMode: fullAccess})
	if err != nil {
		return nil, refused("set the t3 thread to full access", err)
	}
	return nil, nil
}

func note(summary string) harness.Update {
	return harness.Update{Activity: &harness.Activity{Kind: harness.ActivityNote, Summary: summary, At: time.Now().UTC()}}
}

// open resubscribes after cursor, redialing first when the turn's connection died.
func (t *turn) open(ctx context.Context, after int64) (source, error) {
	c, err := t.live(ctx)
	if err != nil {
		return nil, err
	}
	return c.Stream(ctx, subscribeThread, subscribeInput{ThreadID: t.threadID, AfterSequence: &after, AcceptBoundedSnapshot: true})
}

// live is the turn's connection, redialed when it died.
func (t *turn) live(ctx context.Context) (*t3rpc.Conn, error) {
	if t.conn != nil && !closed(t.conn.Done()) {
		return t.conn, nil
	}
	t.close()
	c, err := t.h.connect(ctx, t.target.Session)
	if err != nil {
		return nil, err
	}
	t.conn = c
	return c, nil
}

// end lets the thread go once the pump stops; a run T3 still holds queued is cancelled, since nobody would read its reply.
func (t *turn) end(ctx context.Context, w *watch) {
	defer t.close()
	t.h.messages.CompareAndDelete(t.threadID, t.messageID)
	if !w.queued() {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cancelWithin)
	defer cancel()
	c, err := t.live(ctx)
	if err == nil {
		err = cancelRun(ctx, c, t.threadID, w.run.ID)
	}
	if err != nil {
		t.h.log().Warn("t3clientv2: could not cancel the turn's queued run", "thread", t.threadID, "run", w.run.ID, "error", err)
	}
}

// decline refuses an approval, since an unattended turn has nobody to grant it.
func (t *turn) decline(ctx context.Context, requestID string) error {
	_, err := t.conn.Call(ctx, dispatchCommand, runtimeRequestRespond{
		Type: "runtime-request.respond", CommandID: ids.New(), ThreadID: t.threadID, RequestID: requestID, Decision: "decline",
	})
	if err != nil {
		return refused("decline the T3 approval", err)
	}
	return nil
}

func (t *turn) close() {
	if t.conn == nil {
		return
	}
	// The turn is over with this connection either way; a close error on a dead socket says nothing new.
	_ = t.conn.Close()
	t.conn = nil
}

func closed(done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

// refused is a failed command's error carrying T3's own message when it gave one, so a person reads why.
func refused(what string, err error) error {
	if msg := t3Message(err); msg != "" {
		return fmt.Errorf("%w: %s: %s", apperrs.ErrInvalid, what, msg)
	}
	return fmt.Errorf("%s: %w", what, err)
}

// t3Message is the message T3 refused a call with, "" when the call failed another way or not at all.
func t3Message(err error) string {
	var exit *t3rpc.ExitError
	if !errors.As(err, &exit) {
		return ""
	}
	for _, c := range exit.Causes {
		if c.Error.Message != "" {
			return c.Error.Message
		}
	}
	return ""
}

// decodeItem reads one stream item; one that does not decode is skipped and leaves the cursor where it was.
func decodeItem(log *slog.Logger, raw json.RawMessage) streamItem {
	var item streamItem
	if err := json.Unmarshal(raw, &item); err != nil {
		log.Warn("t3clientv2: malformed stream item skipped", "error", err, "item", t3rpc.Snippet(raw))
		return streamItem{}
	}
	return item
}

func (h *Harness) log() *slog.Logger {
	if h.Options.Logger != nil {
		return h.Options.Logger
	}
	return slog.Default()
}

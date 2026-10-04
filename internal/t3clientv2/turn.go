package t3clientv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
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
	launchThread    = "orchestration.launchThread"
	refreshStatus   = "vcs.refreshStatus"
	fullAccess      = "full-access"
	// missingThread is the failure an initial subscribe ends with when the thread does not exist.
	missingThread = "OrchestrationV2GetThreadProjectionError"
	notFullAccess = "This T3 thread is not in full access"
	noBranchNote  = "Started in the T3 project's folder: it is not on a git branch, so there is nothing to base a worktree on"
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

// threadLaunch creates a thread in a fresh worktree of baseRef, holding its initial message until the worktree is ready.
type threadLaunch struct {
	CommandID         string            `json:"commandId"`
	CreationSource    string            `json:"creationSource"`
	ThreadID          string            `json:"threadId"`
	ProjectID         string            `json:"projectId"`
	Title             string            `json:"title"`
	ModelSelection    modelSelection    `json:"modelSelection"`
	RuntimeMode       string            `json:"runtimeMode"`
	InteractionMode   string            `json:"interactionMode"`
	WorkspaceStrategy workspaceStrategy `json:"workspaceStrategy"`
	InitialMessage    launchMessage     `json:"initialMessage"`
}

type workspaceStrategy struct {
	Type    string `json:"type"`
	BaseRef string `json:"baseRef"`
}

type launchMessage struct {
	MessageID   string            `json:"messageId"`
	Text        string            `json:"text"`
	Attachments []json.RawMessage `json:"attachments"`
}

type statusInput struct {
	Cwd string `json:"cwd"`
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
	ModelSelection *modelSelection   `json:"modelSelection,omitempty"`
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
	// snapshot is the thread's first snapshot, which says how a pending answer reaches T3.
	snapshot projection
	prompted bool
	// adopt makes the turn a Watch: its first snapshot names the run to follow instead of a message it sends.
	adopt bool
	// caught is what the first snapshot's batch emitted, and over the turn's end if that batch already ended it.
	caught []harness.Update
	over   *harness.TurnResult
}

// StartTurn implements harness.Client: the message goes out after the thread's snapshot, and its run is the turn.
func (h *Harness) StartTurn(ctx context.Context, target harness.Target, title string, prompts harness.TurnPrompts) (harness.StartResult, error) {
	c, err := h.connect(ctx, target.Session)
	if err != nil {
		return harness.StartResult{}, err
	}
	t := &turn{h: h, target: target, conn: c, messageID: ids.New()}
	src, w, notes, err := t.start(ctx, title, prompts)
	if errors.Is(err, errAnswered) {
		t.close()
		return answeredTurn(t.threadID), nil
	}
	if err != nil {
		t.close()
		return harness.StartResult{}, err
	}
	return harness.StartResult{SessionID: t.threadID, Updates: t.follow(ctx, src, w, notes), PromptSent: t.prompted}, nil
}

// Watch implements harness.Client: it follows the thread's newest run without sending T3 anything.
func (h *Harness) Watch(ctx context.Context, target harness.Target) (harness.StartResult, error) {
	if target.SessionID == "" {
		return harness.StartResult{}, fmt.Errorf("%w: no session to watch", apperrs.ErrInvalid)
	}
	c, err := h.connect(ctx, target.Session)
	if err != nil {
		return harness.StartResult{}, err
	}
	t := &turn{h: h, target: target, conn: c, threadID: target.SessionID, adopt: true}
	src, w, err := t.subscribe(ctx)
	if err != nil {
		t.close()
		return harness.StartResult{}, fmt.Errorf("watch t3 thread %s: %w", t.threadID, err)
	}
	// The snapshot's steps were shown before; its replies and open questions still belong to the turn.
	caught := slices.DeleteFunc(t.caught, func(u harness.Update) bool { return u.Activity != nil && u.Activity.Kind != harness.ActivityNote })
	if t.over == nil && w.run.ID != "" {
		return harness.StartResult{SessionID: t.threadID, Updates: t.follow(ctx, src, w, caught)}, nil
	}
	src.Close()
	t.close()
	return harness.StartResult{SessionID: t.threadID, Updates: finished(caught, t.over)}, nil
}

// adopted is the message id of the run a Watch follows: the newest run, or the earliest run whose hand-off or restart led to it.
func adopted(p projection) string {
	if len(p.Runs) == 0 {
		return ""
	}
	newest := p.Runs[len(p.Runs)-1]
	for _, r := range p.Runs[:len(p.Runs)-1] {
		w := newWatch(r.UserMessageID)
		w.reset(p)
		if w.followed[newest.ID] {
			return r.UserMessageID
		}
	}
	return newest.UserMessageID
}

// finished is a watched turn whose run ended before the watch began, done when the thread had no run.
func finished(caught []harness.Update, end *harness.TurnResult) <-chan harness.Update {
	if end == nil {
		end = &harness.TurnResult{State: harness.TurnDone}
	}
	updates := make(chan harness.Update, len(caught)+1)
	for _, u := range caught {
		if u.Snapshot != nil {
			final := *u.Snapshot
			final.Streaming = false
			u.Snapshot = &final
		}
		updates <- u
	}
	updates <- harness.Update{Terminal: end}
	close(updates)
	return updates
}

// follow registers the turn for Stop and pumps its stream after first until the turn ends.
func (t *turn) follow(ctx context.Context, src source, w *watch, first []harness.Update) <-chan harness.Update {
	l := newRunningTurn(ctx, t.messageID)
	t.h.turns.Store(t.threadID, l)
	updates := make(chan harness.Update, len(first)+16)
	for _, u := range first {
		updates <- u
	}
	p := &pump{w: w, open: t.open, openChild: t.openThread, decline: t.decline, log: t.h.log(), live: l}
	go func() {
		p.declineApprovals(ctx)
		p.run(ctx, src, updates)
		t.end(ctx, w, l)
	}()
	return updates
}

// start creates or reuses the thread, reads its snapshot, and dispatches the prompt that snapshot calls for.
func (t *turn) start(ctx context.Context, title string, prompts harness.TurnPrompts) (source, *watch, []harness.Update, error) {
	fresh := t.target.SessionID == ""
	t.threadID = t.target.SessionID
	var notes []harness.Update
	var err error
	if fresh {
		if notes, err = t.create(ctx, title, prompts); err != nil {
			return nil, nil, nil, err
		}
	}
	src, w, err := t.subscribe(ctx)
	if errors.Is(err, errThreadGone) && !fresh {
		t.h.log().Info("t3clientv2: reused thread is gone, creating a new one", "thread", t.threadID)
		fresh = true
		if notes, err = t.create(ctx, title, prompts); err != nil {
			return nil, nil, nil, err
		}
		src, w, err = t.subscribe(ctx)
	}
	if err != nil {
		return nil, nil, nil, fmt.Errorf("watch t3 thread %s: %w", t.threadID, err)
	}
	if t.prompted {
		return src, w, notes, nil
	}
	if prompts.Answer != nil && !fresh {
		responded, err := t.deliver(ctx, w, *prompts.Answer)
		if err != nil {
			src.Close()
			return nil, nil, nil, err
		}
		if responded {
			return src, w, nil, nil
		}
	}
	more, err := t.dispatch(ctx, w, fresh, prompts)
	if err != nil {
		src.Close()
		return nil, nil, nil, err
	}
	return src, w, append(notes, more...), nil
}

// create makes the turn's thread on the target's model and options, launching it with the prompt for a worktree.
func (t *turn) create(ctx context.Context, title string, prompts harness.TurnPrompts) ([]harness.Update, error) {
	model, err := t.model()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(title) == "" {
		title = defaultTitle
	}
	t.threadID = ids.New()
	selection := modelSelection{InstanceID: t.target.Provider, Model: model, Options: t.target.ModelOptions}
	var notes []harness.Update
	if t.target.Worktree {
		base, err := t.worktreeBase(ctx)
		if err != nil {
			return nil, err
		}
		if base != "" {
			return t.launch(ctx, title, base, selection, prompts)
		}
		notes = append(notes, note(noBranchNote))
	}
	_, err = t.conn.Call(ctx, dispatchCommand, threadCreate{
		Type: "thread.create", CreatedBy: "user", CreationSource: "web", CommandID: ids.New(), ThreadID: t.threadID,
		ProjectID: t.target.ProjectID, Title: title, ModelSelection: selection, RuntimeMode: fullAccess, InteractionMode: "default",
	})
	if err != nil {
		return nil, refused("create t3 thread", err)
	}
	return notes, nil
}

// launch creates the thread in a new worktree of base with the full prompt; T3 holds back only a launched message until the worktree is ready.
func (t *turn) launch(ctx context.Context, title, base string, selection modelSelection, prompts harness.TurnPrompts) ([]harness.Update, error) {
	refs, notes, err := t.persistImages(ctx, prompts.Attachments)
	if err != nil {
		return nil, err
	}
	_, err = t.conn.Call(ctx, launchThread, threadLaunch{
		CommandID: ids.New(), CreationSource: "web", ThreadID: t.threadID, ProjectID: t.target.ProjectID, Title: title,
		ModelSelection: selection, RuntimeMode: fullAccess, InteractionMode: "default",
		WorkspaceStrategy: workspaceStrategy{Type: "worktree", BaseRef: base},
		InitialMessage:    launchMessage{MessageID: t.messageID, Text: prompts.Full, Attachments: refs},
	})
	if err != nil {
		return nil, refused("start the t3 thread in a new worktree", err)
	}
	t.prompted = true
	return notes, nil
}

// worktreeBase is the branch the T3 project's folder is on, which T3 bases its own new worktrees on; "" when there is none.
func (t *turn) worktreeBase(ctx context.Context) (string, error) {
	projects, err := t.conn.ListProjects(ctx)
	if err != nil {
		return "", fmt.Errorf("list t3 projects: %w", err)
	}
	i := slices.IndexFunc(projects, func(p harness.Project) bool { return p.ID == t.target.ProjectID })
	// An unknown project is left to thread.create, which refuses it in T3's own words.
	if i < 0 {
		return "", nil
	}
	out, err := t.conn.Call(ctx, refreshStatus, statusInput{Cwd: projects[i].Path})
	if err != nil {
		return "", refused("read the t3 project's branch", err)
	}
	var status struct {
		RefName *string `json:"refName"`
	}
	if err := json.Unmarshal(out, &status); err != nil {
		return "", fmt.Errorf("read the t3 project's branch: %w", err)
	}
	if status.RefName == nil {
		return "", nil
	}
	return *status.RefName, nil
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
			item := decodeItem(t.h.log(), raw)
			if item.Kind == "snapshot" {
				t.snapshot = item.Projection
			}
			if item.Kind == "snapshot" && t.adopt && !w.synced {
				t.messageID = adopted(item.Projection)
				w.messageID = t.messageID
			}
			updates, end := w.apply(item)
			t.caught = append(t.caught, updates...)
			if end != nil {
				t.over = end
			}
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
	selection, switched, err := t.selection(w, fresh)
	if err != nil {
		return nil, err
	}
	text, images := prompts.Incremental, []harness.Attachment(nil)
	// T3 gives a new provider instance only a summary of the thread, as it gives an imported thread an excerpt.
	if fresh || switched || w.imported() {
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
		MessageID: t.messageID, Text: text, Attachments: refs, ModelSelection: selection, DispatchMode: dispatchMode{Type: "queue_after_active"},
	})
	if err != nil {
		return nil, refused("send the message to T3 Code", err)
	}
	t.prompted = true
	return append(notes, more...), nil
}

// selection is what a reused thread must switch to for the target, nil if it already runs on it; switched is another provider instance.
func (t *turn) selection(w *watch, fresh bool) (selection *modelSelection, switched bool, err error) {
	if fresh {
		return nil, false, nil
	}
	current := w.thread.ModelSelection
	switched = t.target.Provider != current.InstanceID
	model := t.target.Model
	if model == "" && !switched {
		model = current.Model
	}
	if model == "" {
		if model, err = t.model(); err != nil {
			return nil, false, err
		}
	}
	if !switched && model == current.Model && (len(t.target.ModelOptions) == 0 || sameOptions(t.target.ModelOptions, current.Options)) {
		return nil, false, nil
	}
	return &modelSelection{InstanceID: t.target.Provider, Model: model, Options: t.target.ModelOptions}, switched, nil
}

// model is the target's model, or its provider's default when it names none, since T3 rejects an empty one.
func (t *turn) model() (string, error) {
	if t.target.Model != "" {
		return t.target.Model, nil
	}
	providers, err := t.conn.Providers()
	if err != nil {
		return "", fmt.Errorf("list t3 providers: %w", err)
	}
	return t3rpc.DefaultModel(providers, t.target.Provider)
}

// sameOptions is whether two lists set the same options to the same values, in any order.
func sameOptions(a, b []harness.OptionSetting) bool {
	return len(a) == len(b) && !slices.ContainsFunc(a, func(x harness.OptionSetting) bool {
		return !slices.ContainsFunc(b, func(y harness.OptionSetting) bool { return x.ID == y.ID && reflect.DeepEqual(x.Value, y.Value) })
	})
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
	return t.openThread(ctx, t.threadID, after)
}

// openThread subscribes to threadID after a sequence, from a snapshot when after is 0, on the turn's connection.
func (t *turn) openThread(ctx context.Context, threadID string, after int64) (source, error) {
	c, err := t.live(ctx)
	if err != nil {
		return nil, err
	}
	in := subscribeInput{ThreadID: threadID, AcceptBoundedSnapshot: true}
	if after > 0 {
		in.AfterSequence = &after
	}
	return c.Stream(ctx, subscribeThread, in)
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
func (t *turn) end(ctx context.Context, w *watch, l *runningTurn) {
	defer t.close()
	l.halt() // frees the context only Stop would have cancelled
	t.h.turns.CompareAndDelete(t.threadID, l)
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
	c, err := t.live(ctx)
	if err != nil {
		return err
	}
	_, err = c.Call(ctx, dispatchCommand, requestRespond{
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

package t3client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// RuntimeMode values (thread.create); the agent pipeline defaults to full-access.
const (
	RuntimeModeApprovalRequired = "approval-required"
	RuntimeModeAutoAcceptEdits  = "auto-accept-edits"
	RuntimeModeAuto             = "auto"
	RuntimeModeFullAccess       = "full-access"
)

// DecisionDecline is the unattended-safe approval decision (auto-decline).
const DecisionDecline = "decline"

// TurnState is the terminal outcome of a watched turn.
type TurnState string

const (
	TurnDone        TurnState = "done"
	TurnInterrupted TurnState = "interrupted"
	TurnError       TurnState = "error"
)

// MessageSnapshot is one cumulative message state, re-emitted per messageID until Streaming goes false.
// T3 sends deltas on the wire; the client sums them so every snapshot is self-contained for late joiners.
type MessageSnapshot struct {
	MessageID string
	Text      string
	Streaming bool
}

// ApprovalRequest surfaces a provider approval prompt; RequestID is best-effort, the upstream shape is unknown.
type ApprovalRequest struct {
	RequestID  string
	Kind       string
	Summary    string
	RawPayload json.RawMessage
}

// TurnResult is the terminal state of a subscription, derived from thread.session-set + streaming:false.
type TurnResult struct {
	State     TurnState
	LastError string
}

// Update is one subscription item, exactly one field set; a Terminal update is always last.
type Update struct {
	Snapshot *MessageSnapshot
	Activity *harness.Activity
	Approval *ApprovalRequest
	Question *harness.Question
	Terminal *TurnResult
}

// Subscription watches one thread's turn to a terminal state, then closes; one turn per subscription.
type Subscription struct {
	updates chan Update
	cancel  context.CancelFunc
	// dropped is set before updates closes when the connection died mid-turn; ResumeThread continues from it.
	dropped *turnWatch
}

// Updates yields snapshots/approvals and finally one Terminal, then closes; a dropped connection closes it without one.
func (s *Subscription) Updates() <-chan Update { return s.updates }

// Close stops watching; the server-side stream is interrupted best-effort.
func (s *Subscription) Close() { s.cancel() }

// Dropped is the watch to resume once Updates has closed without a Terminal because the connection died; nil otherwise.
func (s *Subscription) Dropped() *turnWatch { return s.dropped }

type modelSelection struct {
	InstanceID string                  `json:"instanceId"`
	Model      string                  `json:"model"`
	Options    []harness.OptionSetting `json:"options,omitempty"`
}

type threadCreateCommand struct {
	Type            string         `json:"type"`
	CommandID       string         `json:"commandId"`
	ThreadID        string         `json:"threadId"`
	ProjectID       string         `json:"projectId"`
	Title           string         `json:"title"`
	ModelSelection  modelSelection `json:"modelSelection"`
	RuntimeMode     string         `json:"runtimeMode"`
	InteractionMode string         `json:"interactionMode"`
	Branch          *string        `json:"branch"`
	WorktreePath    *string        `json:"worktreePath"`
	CreatedAt       string         `json:"createdAt"`
}

// turnAttachment is T3's thread.turn.start attachment shape; T3 never fetches a URL, so the bytes ride inline.
type turnAttachment struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	MIMEType  string `json:"mimeType"`
	SizeBytes int    `json:"sizeBytes"`
	DataURL   string `json:"dataUrl"`
}

type turnMessage struct {
	MessageID   string           `json:"messageId"`
	Role        string           `json:"role"`
	Text        string           `json:"text"`
	Attachments []turnAttachment `json:"attachments"`
}

type threadTurnStartCommand struct {
	Type            string      `json:"type"`
	CommandID       string      `json:"commandId"`
	ThreadID        string      `json:"threadId"`
	Message         turnMessage `json:"message"`
	RuntimeMode     string      `json:"runtimeMode"`
	InteractionMode string      `json:"interactionMode"`
	CreatedAt       string      `json:"createdAt"`
}

type threadTurnInterruptCommand struct {
	Type      string `json:"type"`
	CommandID string `json:"commandId"`
	ThreadID  string `json:"threadId"`
	CreatedAt string `json:"createdAt"`
}

// threadSettleCommand carries no createdAt: protocol 1's thread.settle schema has none, unlike the turn commands.
type threadSettleCommand struct {
	Type      string `json:"type"`
	CommandID string `json:"commandId"`
	ThreadID  string `json:"threadId"`
}

type threadApprovalRespondCommand struct {
	Type      string `json:"type"`
	CommandID string `json:"commandId"`
	ThreadID  string `json:"threadId"`
	RequestID string `json:"requestId"`
	Decision  string `json:"decision"`
	CreatedAt string `json:"createdAt"`
}

// threadUserInputRespondCommand answers a pending user-input request; answers is keyed by question id and holds
// the free text, one option value, or the option values of a multi-select, the shapes T3's own composer sends.
type threadUserInputRespondCommand struct {
	Type      string         `json:"type"`
	CommandID string         `json:"commandId"`
	ThreadID  string         `json:"threadId"`
	RequestID string         `json:"requestId"`
	Answers   map[string]any `json:"answers"`
	CreatedAt string         `json:"createdAt"`
}

type subscribeThreadInput struct {
	ThreadID string `json:"threadId"`
	// AfterSequence makes T3 replay this thread's events after that global sequence instead of sending a snapshot.
	AfterSequence int64 `json:"afterSequence,omitempty"`
}

// dispatch routes one client command through orchestration.dispatchCommand; the ack carries nothing callers need.
func (c *Client) dispatch(ctx context.Context, command any) error {
	_, err := c.call(ctx, "orchestration.dispatchCommand", command)
	return err
}

// CreateThread creates a T3 thread in the given T3 project and returns its client-generated thread id.
func (c *Client) CreateThread(ctx context.Context, t3ProjectID, title, providerInstanceID, model string, options []harness.OptionSetting, runtimeMode string) (string, error) {
	threadID := ids.New()
	err := c.dispatch(ctx, threadCreateCommand{
		Type:            "thread.create",
		CommandID:       ids.New(),
		ThreadID:        threadID,
		ProjectID:       t3ProjectID,
		Title:           title,
		ModelSelection:  modelSelection{InstanceID: providerInstanceID, Model: model, Options: options},
		RuntimeMode:     orRuntimeDefault(runtimeMode),
		InteractionMode: "default",
		CreatedAt:       isoNow(),
	})
	if err != nil {
		return "", err
	}
	return threadID, nil
}

// StartTurn sends one user message; watch the reply via a SubscribeThread opened before this call.
func (c *Client) StartTurn(ctx context.Context, threadID, text, runtimeMode string, attachments []harness.Attachment) error {
	return c.dispatch(ctx, threadTurnStartCommand{
		Type:      "thread.turn.start",
		CommandID: ids.New(),
		ThreadID:  threadID,
		Message: turnMessage{
			MessageID:   ids.New(),
			Role:        "user",
			Text:        text,
			Attachments: encodeAttachments(attachments),
		},
		RuntimeMode:     orRuntimeDefault(runtimeMode),
		InteractionMode: "default",
		CreatedAt:       isoNow(),
	})
}

// Interrupt aborts whatever turn is active on the thread (turnId omitted = "whatever's active", ticket 03).
func (c *Client) Interrupt(ctx context.Context, threadID string) error {
	return c.dispatch(ctx, threadTurnInterruptCommand{
		Type:      "thread.turn.interrupt",
		CommandID: ids.New(),
		ThreadID:  threadID,
		CreatedAt: isoNow(),
	})
}

// Settle moves the thread to T3's settled list; T3 refuses while a turn runs or a question waits.
func (c *Client) Settle(ctx context.Context, threadID string) error {
	return c.dispatch(ctx, threadSettleCommand{Type: "thread.settle", CommandID: ids.New(), ThreadID: threadID})
}

// RespondApproval answers a provider approval request (auto-decline with DecisionDecline).
func (c *Client) RespondApproval(ctx context.Context, threadID, requestID, decision string) error {
	return c.dispatch(ctx, threadApprovalRespondCommand{
		Type:      "thread.approval.respond",
		CommandID: ids.New(),
		ThreadID:  threadID,
		RequestID: requestID,
		Decision:  decision,
		CreatedAt: isoNow(),
	})
}

// alreadyAnswered is T3's invariant message for a question resolved elsewhere, in T3 itself or by an earlier answer.
const alreadyAnswered = "This question has already been answered"

// RespondUserInput answers the question requestID on the thread with thread.user-input.respond; ErrConflict when
// T3 already holds an answer for it.
func (c *Client) RespondUserInput(ctx context.Context, threadID, requestID string, answer harness.QuestionAnswer) error {
	err := c.dispatch(ctx, threadUserInputRespondCommand{
		Type:      "thread.user-input.respond",
		CommandID: ids.New(),
		ThreadID:  threadID,
		RequestID: requestID,
		Answers:   encodeAnswers(answer),
		CreatedAt: isoNow(),
	})
	if err != nil && strings.Contains(err.Error(), alreadyAnswered) {
		return fmt.Errorf("%w: question %s was already answered", apperrs.ErrConflict, requestID)
	}
	return err
}

// encodeAnswers mirrors T3's composer: free text wins, one selection is a string, several are an array.
func encodeAnswers(answer harness.QuestionAnswer) map[string]any {
	out := make(map[string]any, len(answer.Answers))
	for id, v := range answer.Answers {
		switch {
		case v.Text != "":
			out[id] = v.Text
		case len(v.Selected) == 1:
			out[id] = v.Selected[0]
		default:
			out[id] = v.Selected
		}
	}
	return out
}

// SubscribeThread watches until a terminal state, ctx ends, or the connection drops; open before StartTurn.
func (c *Client) SubscribeThread(ctx context.Context, threadID string) (*Subscription, error) {
	return c.subscribe(ctx, threadID, &turnWatch{})
}

// ResumeThread carries a dropped watch on over this connection, reporting only what the watch has not seen yet.
func (c *Client) ResumeThread(ctx context.Context, threadID string, w *turnWatch) (*Subscription, error) {
	return c.subscribe(ctx, threadID, w)
}

func (c *Client) subscribe(ctx context.Context, threadID string, w *turnWatch) (*Subscription, error) {
	id, ch := c.register()
	input := subscribeThreadInput{ThreadID: threadID, AfterSequence: w.lastSeq}
	env := requestEnvelope{Tag: "Request", ID: id, RPCTag: "orchestration.subscribeThread", Payload: input, Headers: [][]string{}}
	if err := c.send(ctx, env); err != nil {
		c.unregister(id)
		return nil, apperrs.Retryable(fmt.Errorf("subscribe thread: %w", err))
	}
	subCtx, cancel := context.WithCancel(ctx)
	sub := &Subscription{updates: make(chan Update, 16), cancel: cancel}
	go c.runSubscription(subCtx, id, ch, sub, w)
	return sub, nil
}

func (c *Client) runSubscription(ctx context.Context, id string, ch chan serverEnvelope, sub *Subscription, w *turnWatch) {
	defer func() {
		c.unregister(id)
		_ = c.send(c.ctx, interruptEnvelope{Tag: "Interrupt", RequestID: id})
		close(sub.updates)
	}()
	emit := func(u Update) bool {
		select {
		case sub.updates <- u:
			return true
		case <-ctx.Done():
			return false
		}
	}
	for {
		select {
		case env := <-ch:
			switch env.Tag {
			case "Chunk":
				for _, raw := range env.Values {
					updates, terminal := c.streamItemUpdates(raw, w)
					for _, u := range updates {
						if !emit(u) {
							return
						}
					}
					if terminal != nil {
						emit(Update{Terminal: terminal})
						return
					}
				}
				c.ack(id)
			case "Exit":
				emit(Update{Terminal: &TurnResult{State: TurnError, LastError: exitReason(env.Exit)}})
				return
			}
		case <-ctx.Done():
			return
		case <-c.done:
			c.log.Warn("t3client: connection lost mid-turn", "request", id, "last_sequence", w.lastSeq, "error", c.err)
			sub.dropped = w
			return
		}
	}
}

// exitReason is why a thread stream ended early; a failure carries T3's cause, such as the thread no longer existing.
func exitReason(raw json.RawMessage) string {
	if _, err := decodeExit("subscribe thread", raw, nil); err != nil {
		return err.Error()
	}
	return "subscription stream ended before the turn completed"
}

// turnWatch: "done" needs both the session settling and the reply closing; settling alone arrived 33s early once.
type turnWatch struct {
	turnSeen    bool
	settled     bool
	replyClosed bool
	text        map[string]string // messageID -> text summed from streaming deltas
	// lastSeq is the highest global event sequence applied, the cursor a resume replays after and dedupes against.
	lastSeq int64
	// synced marks the first snapshot taken; seen holds message and activity ids reported or older than the turn.
	synced bool
	seen   map[string]bool
}

func (w *turnWatch) see(id string) {
	if w.seen == nil {
		w.seen = map[string]bool{}
	}
	w.seen[id] = true
}

// accumulate mirrors T3's own reducer: a streaming event appends its delta, a closing one replaces only when non-empty.
func (w *turnWatch) accumulate(p messageSentPayload) string {
	if w.text == nil {
		w.text = map[string]string{}
	}
	if p.Streaming {
		w.text[p.MessageID] += p.Text
		return w.text[p.MessageID]
	}
	if p.Text != "" {
		w.text[p.MessageID] = p.Text
	}
	return w.text[p.MessageID]
}

// ponytail: the first closed reply ends the watch; track per-message open/close if multi-message turns show up.
func (w *turnWatch) done() *TurnResult {
	if w.settled && w.replyClosed {
		return &TurnResult{State: TurnDone}
	}
	return nil
}

// wire shapes below decode leniently: any miss is a log-and-skip, since the protocol is reverse-engineered.

type streamItem struct {
	Kind     string          `json:"kind"`
	Event    json.RawMessage `json:"event"`
	Snapshot json.RawMessage `json:"snapshot"`
}

type wireEvent struct {
	Sequence int64           `json:"sequence"`
	Type     string          `json:"type"`
	Tag      string          `json:"_tag"`
	Payload  json.RawMessage `json:"payload"`
}

type messageSentPayload struct {
	MessageID string `json:"messageId"`
	Role      string `json:"role"`
	Text      string `json:"text"`
	Streaming bool   `json:"streaming"`
}

type wireSession struct {
	Status       string  `json:"status"`
	ActiveTurnID *string `json:"activeTurnId"`
	LastError    *string `json:"lastError"`
}

type sessionSetPayload struct {
	Session wireSession `json:"session"`
}

// threadSnapshot is the slice of T3's thread detail snapshot a watch reads: what the thread holds and where it stands.
type threadSnapshot struct {
	SnapshotSequence int64 `json:"snapshotSequence"`
	Thread           struct {
		DeletedAt *string `json:"deletedAt"`
		Messages  []struct {
			ID        string `json:"id"`
			Role      string `json:"role"`
			Text      string `json:"text"`
			Streaming bool   `json:"streaming"`
			UpdatedAt string `json:"updatedAt"`
		} `json:"messages"`
		Activities []wireActivity `json:"activities"`
		Session    *wireSession   `json:"session"`
	} `json:"thread"`
}

type wireActivity struct {
	ID        string          `json:"id"`
	CreatedAt string          `json:"createdAt"`
	Tone      string          `json:"tone"`
	Kind      string          `json:"kind"`
	Summary   string          `json:"summary"`
	Payload   json.RawMessage `json:"payload"`
}

type activityAppendedPayload struct {
	Activity wireActivity `json:"activity"`
}

// toolPayload is the tool-tone activity payload: one tool call's id, status, and its input and (once done) result.
// ItemType tells a built-in command, file change, or image view from a plain tool call; Detail is T3's own
// "Tool: {args...}" label, the only place an in-progress file change names its path.
type toolPayload struct {
	ItemType   string `json:"itemType"`
	ToolCallID string `json:"toolCallId"`
	Status     string `json:"status"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Data       struct {
		ToolName  string `json:"toolName"`
		Command   string `json:"command"`
		ImagePath string `json:"imagePath"`
		// Item is Codex's raw item, the only place its command and an MCP call's arguments and result ride.
		Item struct {
			Command   string          `json:"command"`
			Arguments json.RawMessage `json:"arguments"`
			Result    json.RawMessage `json:"result"`
		} `json:"item"`
		Input  json.RawMessage `json:"input"`
		Result json.RawMessage `json:"result"`
	} `json:"data"`
}

// Built-in item types T3 labels beyond a bare tool call.
const (
	itemCommand    = "command_execution"
	itemFileChange = "file_change"
	itemImageView  = "image_view"
)

// builtinInput is the slice of a built-in tool's arguments the row label names.
type builtinInput struct {
	Command  string `json:"command"`
	FilePath string `json:"file_path"`
}

// userInputPayload is the info-tone user-input.requested payload: the request id the answer names and its questions.
type userInputPayload struct {
	RequestID string `json:"requestId"`
	Questions []struct {
		ID          string `json:"id"`
		Question    string `json:"question"`
		Header      string `json:"header"`
		MultiSelect bool   `json:"multiSelect"`
		Options     []struct {
			Label       string `json:"label"`
			Description string `json:"description"`
			Value       string `json:"value"`
		} `json:"options"`
	} `json:"questions"`
}

// userInputRequested is the activity kind T3 emits when the provider stops on a question; the tool call stays open.
const userInputRequested = "user-input.requested"

type activityDetail struct {
	Input  json.RawMessage `json:"input,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
}

// questionTool is the tool a provider raises a question through; it is the one tool call that is a question, not a step.
const questionTool = "AskUserQuestion"

const summaryRunes = 160

// shellTool names a command whose provider names no tool (Codex, OpenCode), so it reads as a command like Claude's Bash.
const shellTool = "Shell"

// toolActivity maps one tool-tone activity onto the harness seam; a payload that fails to decode keeps T3's own label.
func (c *Client) toolActivity(a wireActivity) *harness.Activity {
	var p toolPayload
	if err := json.Unmarshal(a.Payload, &p); err != nil {
		c.log.Debug("t3client: tool activity payload not decoded", "error", err, "payload", snippet(a.Payload))
	}
	kind := harness.ActivityToolCall
	if a.Kind == "tool.completed" {
		kind = harness.ActivityToolResult
	}
	if p.Data.ToolName == questionTool {
		kind = harness.ActivityQuestion
	}
	tool := p.Data.ToolName
	if tool == "" && p.ItemType == itemCommand {
		tool = shellTool
	}
	if tool == "" {
		tool = p.Title
	}
	summary := harness.Preview(c.builtinSummary(p), summaryRunes)
	if summary == "" {
		summary = harness.Preview(string(p.Data.Input), summaryRunes)
	}
	if summary == "" || summary == "{}" {
		summary = a.Summary
	}
	if a.Kind == "tool.completed" && p.Status == "failed" {
		summary += " · failed"
	}
	at, err := time.Parse(time.RFC3339Nano, a.CreatedAt)
	if err != nil {
		at = time.Now().UTC()
	}
	return &harness.Activity{Kind: kind, CallID: p.ToolCallID, Tool: tool, Summary: summary, Detail: c.detailJSON(p), At: at}
}

// builtinSummary is the command, path, or image a built-in tool acts on, so the row reads that instead of JSON;
// empty for every other tool. An in-progress file change carries its path only inside T3's detail label.
func (c *Client) builtinSummary(p toolPayload) string {
	var in builtinInput
	if len(p.Data.Input) > 0 {
		if err := json.Unmarshal(p.Data.Input, &in); err != nil {
			c.log.Debug("t3client: built-in tool input not decoded", "error", err, "input", snippet(p.Data.Input))
		}
	}
	switch p.ItemType {
	case itemCommand:
		return firstNonEmpty(p.Data.Command, in.Command, p.Data.Item.Command)
	case itemFileChange:
		return firstNonEmpty(in.FilePath, quotedField(p.Detail, "file_path"))
	case itemImageView:
		return p.Data.ImagePath
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// firstNonEmptyJSON skips an absent or null value, so a Codex item's fields stand in for the input and result it lacks.
func firstNonEmptyJSON(values ...json.RawMessage) json.RawMessage {
	for _, v := range values {
		if len(v) > 0 && string(v) != "null" {
			return v
		}
	}
	return nil
}

// quotedField reads the string value of key out of a JSON-looking label that may be cut short, so it never parses.
func quotedField(label, key string) string {
	marker := `"` + key + `":"`
	start := strings.Index(label, marker)
	if start < 0 {
		return ""
	}
	rest := label[start+len(marker):]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func (c *Client) detailJSON(p toolPayload) string {
	input := firstNonEmptyJSON(p.Data.Input, p.Data.Item.Arguments)
	result := firstNonEmptyJSON(p.Data.Result, p.Data.Item.Result)
	if len(input) == 0 && len(result) == 0 {
		return ""
	}
	b, err := json.Marshal(activityDetail{Input: input, Result: result})
	if err != nil {
		c.log.Debug("t3client: tool activity detail not encoded", "error", err)
		return ""
	}
	return harness.CapDetail(string(b))
}

func (c *Client) streamItemUpdates(raw json.RawMessage, w *turnWatch) ([]Update, *TurnResult) {
	var item streamItem
	if err := json.Unmarshal(raw, &item); err != nil {
		c.log.Warn("t3client: malformed stream item skipped", "error", err, "item", snippet(raw))
		return nil, nil
	}
	switch item.Kind {
	case "synchronized":
		return nil, nil
	case "snapshot":
		return c.snapshotUpdates(item.Snapshot, w)
	case "event":
		update, terminal := c.eventUpdate(item.Event, w)
		if update == nil {
			return nil, terminal
		}
		return []Update{*update}, terminal
	default:
		c.log.Debug("t3client: unknown stream item kind skipped", "kind", item.Kind, "item", snippet(raw))
		return nil, nil
	}
}

// snapshotUpdates remembers the first snapshot as history; a later one answers a resume T3 could not replay.
func (c *Client) snapshotUpdates(raw json.RawMessage, w *turnWatch) ([]Update, *TurnResult) {
	var s threadSnapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		c.log.Warn("t3client: malformed thread snapshot skipped", "error", err, "snapshot", snippet(raw))
		return nil, nil
	}
	if !w.synced {
		w.synced = true
		for _, m := range s.Thread.Messages {
			w.see(m.ID)
		}
		for _, a := range s.Thread.Activities {
			w.see(a.ID)
		}
		return nil, nil
	}
	if s.Thread.DeletedAt != nil {
		return nil, &TurnResult{State: TurnError, LastError: "the thread was deleted in T3 Code"}
	}
	missed := c.missedUpdates(s, w)
	w.lastSeq = max(w.lastSeq, s.SnapshotSequence)
	if s.Thread.Session == nil {
		return missed, nil
	}
	return missed, c.sessionTerminal(sessionSetPayload{Session: *s.Thread.Session}, w)
}

// missedUpdates is every step and reply in s the watch has not reported, in the order they happened.
func (c *Client) missedUpdates(s threadSnapshot, w *turnWatch) []Update {
	type timed struct {
		at string
		u  Update
	}
	var missed []timed
	for _, a := range s.Thread.Activities {
		if w.seen[a.ID] {
			continue
		}
		w.see(a.ID)
		w.turnSeen = true
		if u := c.activityUpdate(a); u != nil {
			missed = append(missed, timed{a.CreatedAt, *u})
		}
	}
	for _, m := range s.Thread.Messages {
		if m.Role != "assistant" || w.seen[m.ID] {
			continue
		}
		w.turnSeen = true
		text := w.accumulate(messageSentPayload{MessageID: m.ID, Text: m.Text})
		if !m.Streaming {
			w.see(m.ID)
			w.replyClosed = true
		}
		missed = append(missed, timed{m.UpdatedAt, Update{Snapshot: &MessageSnapshot{MessageID: m.ID, Text: text, Streaming: m.Streaming}}})
	}
	// ISO timestamps in one format sort as strings; a reply sorts by its last change so it lands after its steps.
	sort.SliceStable(missed, func(i, j int) bool { return missed[i].at < missed[j].at })
	out := make([]Update, 0, len(missed))
	for _, m := range missed {
		out = append(out, m.u)
	}
	return out
}

func (c *Client) eventUpdate(raw json.RawMessage, w *turnWatch) (*Update, *TurnResult) {
	var ev wireEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		c.log.Warn("t3client: malformed event skipped", "error", err, "event", snippet(raw))
		return nil, nil
	}
	if ev.Sequence > 0 {
		if ev.Sequence <= w.lastSeq {
			return nil, nil
		}
		w.lastSeq = ev.Sequence
	}
	eventType := ev.Type
	if eventType == "" {
		eventType = ev.Tag
	}
	switch eventType {
	case "thread.message-sent":
		var p messageSentPayload
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			c.log.Warn("t3client: malformed message-sent payload skipped", "error", err, "payload", snippet(ev.Payload))
			return nil, nil
		}
		if p.Role != "assistant" {
			return nil, nil
		}
		w.turnSeen = true
		if !p.Streaming {
			w.see(p.MessageID)
			w.replyClosed = true
		}
		return &Update{Snapshot: &MessageSnapshot{MessageID: p.MessageID, Text: w.accumulate(p), Streaming: p.Streaming}}, w.done()
	case "thread.session-set":
		var p sessionSetPayload
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			c.log.Warn("t3client: malformed session-set payload skipped", "error", err, "payload", snippet(ev.Payload))
			return nil, nil
		}
		return nil, c.sessionTerminal(p, w)
	case "thread.activity-appended":
		var p activityAppendedPayload
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			c.log.Warn("t3client: malformed activity-appended payload skipped", "error", err, "payload", snippet(ev.Payload))
			return nil, nil
		}
		w.see(p.Activity.ID)
		return c.activityUpdate(p.Activity), nil
	default:
		c.log.Debug("t3client: unknown event type skipped", "type", eventType)
		return nil, nil
	}
}

// activityUpdate maps one thread activity by tone: tool steps, the user-input question, and approvals.
func (c *Client) activityUpdate(a wireActivity) *Update {
	switch {
	case a.Tone == "error":
		c.log.Warn("t3client: thread error activity", "kind", a.Kind, "summary", a.Summary)
		return nil
	case a.Tone == "tool":
		return &Update{Activity: c.toolActivity(a)}
	case a.Tone == "info" && a.Kind == userInputRequested:
		return c.questionUpdate(a)
	case a.Tone != "approval":
		return nil
	}
	// The approval payload shape is unknown upstream; log it loudly to learn it from real captures.
	c.log.Warn("t3client: approval request received (payload shape unverified upstream)",
		"activity_id", a.ID, "kind", a.Kind, "summary", a.Summary,
		"raw_payload", snippet(a.Payload))
	return &Update{Approval: &ApprovalRequest{
		RequestID:  approvalRequestID(a.ID, a.Payload),
		Kind:       a.Kind,
		Summary:    a.Summary,
		RawPayload: a.Payload,
	}}
}

// questionUpdate maps user-input.requested onto the harness seam; a payload without a request id cannot be answered, so it is skipped.
func (c *Client) questionUpdate(a wireActivity) *Update {
	var p userInputPayload
	if err := json.Unmarshal(a.Payload, &p); err != nil || p.RequestID == "" {
		c.log.Warn("t3client: user-input payload not decoded", "error", err, "payload", snippet(a.Payload))
		return nil
	}
	q := &harness.Question{RequestID: p.RequestID, Questions: make([]harness.QuestionItem, 0, len(p.Questions))}
	for _, item := range p.Questions {
		options := make([]harness.QuestionOption, 0, len(item.Options))
		for _, o := range item.Options {
			options = append(options, harness.QuestionOption{Label: o.Label, Description: o.Description, Value: o.Value})
		}
		id := item.ID
		if id == "" {
			id = item.Question
		}
		q.Questions = append(q.Questions, harness.QuestionItem{ID: id, Text: item.Question, Header: item.Header, Options: options, MultiSelect: item.MultiSelect})
	}
	return &Update{Question: q}
}

// sessionTerminal maps a session status to a terminal state; "done" also needs turnWatch's closed reply.
func (c *Client) sessionTerminal(p sessionSetPayload, w *turnWatch) *TurnResult {
	s := p.Session
	switch s.Status {
	case "error":
		lastError := "t3 session error"
		if s.LastError != nil && *s.LastError != "" {
			lastError = *s.LastError
		}
		return &TurnResult{State: TurnError, LastError: lastError}
	case "interrupted":
		return &TurnResult{State: TurnInterrupted}
	case "running", "starting":
		w.turnSeen = true
		return nil
	case "idle", "ready", "stopped":
		if !w.turnSeen || s.ActiveTurnID != nil {
			return nil
		}
		// A stopped provider session sends nothing further; don't hold out for a reply that cannot arrive.
		if s.Status == "stopped" {
			return &TurnResult{State: TurnDone}
		}
		w.settled = true
		return w.done()
	default:
		c.log.Debug("t3client: unknown session status skipped", "status", s.Status)
		return nil
	}
}

// approvalRequestID digs a plausible id out of the untyped payload, falling back to the activity id.
func approvalRequestID(activityID string, payload json.RawMessage) string {
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err == nil {
		for _, key := range []string{"requestId", "approvalRequestId"} {
			if s, ok := m[key].(string); ok && s != "" {
				return s
			}
		}
		if req, ok := m["request"].(map[string]any); ok {
			for _, key := range []string{"requestId", "id"} {
				if s, ok := req[key].(string); ok && s != "" {
					return s
				}
			}
		}
	}
	return activityID
}

// encodeAttachments returns a non-nil slice so an empty turn marshals `attachments: []`, the shape T3 expects.
func encodeAttachments(attachments []harness.Attachment) []turnAttachment {
	out := make([]turnAttachment, 0, len(attachments))
	for _, a := range attachments {
		out = append(out, turnAttachment{
			Type:      "image",
			Name:      a.Name,
			MIMEType:  a.MIME,
			SizeBytes: len(a.Bytes),
			DataURL:   "data:" + a.MIME + ";base64," + base64.StdEncoding.EncodeToString(a.Bytes),
		})
	}
	return out
}

func orRuntimeDefault(runtimeMode string) string {
	if runtimeMode == "" {
		return RuntimeModeFullAccess
	}
	return runtimeMode
}

func isoNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

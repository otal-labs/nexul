package t3client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
}

// Updates yields snapshots/approvals and finally one Terminal, then closes.
func (s *Subscription) Updates() <-chan Update { return s.updates }

// Close stops watching; the server-side stream is interrupted best-effort.
func (s *Subscription) Close() { s.cancel() }

type modelSelection struct {
	InstanceID string `json:"instanceId"`
	Model      string `json:"model"`
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
}

// dispatch routes one client command through orchestration.dispatchCommand; the ack carries nothing callers need.
func (c *Client) dispatch(ctx context.Context, command any) error {
	_, err := c.call(ctx, "orchestration.dispatchCommand", command)
	return err
}

// CreateThread creates a T3 thread in the given T3 project and returns its client-generated thread id.
func (c *Client) CreateThread(ctx context.Context, t3ProjectID, title, providerInstanceID, model, runtimeMode string) (string, error) {
	threadID := ids.New()
	err := c.dispatch(ctx, threadCreateCommand{
		Type:            "thread.create",
		CommandID:       ids.New(),
		ThreadID:        threadID,
		ProjectID:       t3ProjectID,
		Title:           title,
		ModelSelection:  modelSelection{InstanceID: providerInstanceID, Model: model},
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

// RespondUserInput answers the question requestID on the thread with thread.user-input.respond.
func (c *Client) RespondUserInput(ctx context.Context, threadID, requestID string, answer harness.QuestionAnswer) error {
	return c.dispatch(ctx, threadUserInputRespondCommand{
		Type:      "thread.user-input.respond",
		CommandID: ids.New(),
		ThreadID:  threadID,
		RequestID: requestID,
		Answers:   encodeAnswers(answer),
		CreatedAt: isoNow(),
	})
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
	id, ch := c.register()
	env := requestEnvelope{Tag: "Request", ID: id, RPCTag: "orchestration.subscribeThread", Payload: subscribeThreadInput{ThreadID: threadID}, Headers: [][]string{}}
	if err := c.send(ctx, env); err != nil {
		c.unregister(id)
		return nil, apperrs.Retryable(fmt.Errorf("subscribe thread: %w", err))
	}
	subCtx, cancel := context.WithCancel(ctx)
	sub := &Subscription{updates: make(chan Update, 16), cancel: cancel}
	go c.runSubscription(subCtx, id, ch, sub)
	return sub, nil
}

func (c *Client) runSubscription(ctx context.Context, id string, ch chan serverEnvelope, sub *Subscription) {
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
	var w turnWatch
	for {
		select {
		case env := <-ch:
			switch env.Tag {
			case "Chunk":
				for _, raw := range env.Values {
					update, terminal := c.streamItemUpdate(raw, &w)
					if update != nil && !emit(*update) {
						return
					}
					if terminal != nil {
						emit(Update{Terminal: terminal})
						return
					}
				}
				c.ack(id)
			case "Exit":
				emit(Update{Terminal: &TurnResult{State: TurnError, LastError: "subscription stream ended before the turn completed"}})
				return
			}
		case <-ctx.Done():
			return
		case <-c.done:
			emit(Update{Terminal: &TurnResult{State: TurnError, LastError: fmt.Sprintf("T3 connection lost: %v", c.err)}})
			return
		}
	}
}

// turnWatch: "done" needs both the session settling and the reply closing; settling alone arrived 33s early once.
type turnWatch struct {
	turnSeen    bool
	settled     bool
	replyClosed bool
	text        map[string]string // messageID -> text summed from streaming deltas
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
	Type    string          `json:"type"`
	Tag     string          `json:"_tag"`
	Payload json.RawMessage `json:"payload"`
}

type messageSentPayload struct {
	MessageID string `json:"messageId"`
	Role      string `json:"role"`
	Text      string `json:"text"`
	Streaming bool   `json:"streaming"`
}

type sessionSetPayload struct {
	Session struct {
		Status       string  `json:"status"`
		ActiveTurnID *string `json:"activeTurnId"`
		LastError    *string `json:"lastError"`
	} `json:"session"`
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
		ToolName  string          `json:"toolName"`
		Command   string          `json:"command"`
		ImagePath string          `json:"imagePath"`
		Input     json.RawMessage `json:"input"`
		Result    json.RawMessage `json:"result"`
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
		return firstNonEmpty(p.Data.Command, in.Command)
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
	if len(p.Data.Input) == 0 && len(p.Data.Result) == 0 {
		return ""
	}
	b, err := json.Marshal(activityDetail{Input: p.Data.Input, Result: p.Data.Result})
	if err != nil {
		c.log.Debug("t3client: tool activity detail not encoded", "error", err)
		return ""
	}
	return harness.CapDetail(string(b))
}

func (c *Client) streamItemUpdate(raw json.RawMessage, w *turnWatch) (*Update, *TurnResult) {
	var item streamItem
	if err := json.Unmarshal(raw, &item); err != nil {
		c.log.Warn("t3client: malformed stream item skipped", "error", err, "item", snippet(raw))
		return nil, nil
	}
	switch item.Kind {
	case "synchronized":
		return nil, nil
	case "snapshot":
		// The full-thread snapshot only carries history from before we subscribed; turn watching starts from live events.
		c.log.Debug("t3client: thread snapshot skipped", "bytes", len(item.Snapshot))
		return nil, nil
	case "event":
		return c.eventUpdate(item.Event, w)
	default:
		c.log.Debug("t3client: unknown stream item kind skipped", "kind", item.Kind, "item", snippet(raw))
		return nil, nil
	}
}

func (c *Client) eventUpdate(raw json.RawMessage, w *turnWatch) (*Update, *TurnResult) {
	var ev wireEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		c.log.Warn("t3client: malformed event skipped", "error", err, "event", snippet(raw))
		return nil, nil
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
		return c.activityUpdate(ev.Payload), nil
	default:
		c.log.Debug("t3client: unknown event type skipped", "type", eventType)
		return nil, nil
	}
}

// activityUpdate maps one thread.activity-appended by tone: tool steps, the user-input question, and approvals.
func (c *Client) activityUpdate(payload json.RawMessage) *Update {
	var p activityAppendedPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		c.log.Warn("t3client: malformed activity-appended payload skipped", "error", err, "payload", snippet(payload))
		return nil
	}
	switch {
	case p.Activity.Tone == "error":
		c.log.Warn("t3client: thread error activity", "kind", p.Activity.Kind, "summary", p.Activity.Summary)
		return nil
	case p.Activity.Tone == "tool":
		return &Update{Activity: c.toolActivity(p.Activity)}
	case p.Activity.Tone == "info" && p.Activity.Kind == userInputRequested:
		return c.questionUpdate(p.Activity)
	case p.Activity.Tone != "approval":
		return nil
	}
	// The approval payload shape is unknown upstream; log it loudly to learn it from real captures.
	c.log.Warn("t3client: approval request received (payload shape unverified upstream)",
		"activity_id", p.Activity.ID, "kind", p.Activity.Kind, "summary", p.Activity.Summary,
		"raw_payload", snippet(p.Activity.Payload))
	return &Update{Approval: &ApprovalRequest{
		RequestID:  approvalRequestID(p.Activity.ID, p.Activity.Payload),
		Kind:       p.Activity.Kind,
		Summary:    p.Activity.Summary,
		RawPayload: p.Activity.Payload,
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

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/redact"
)

// TopicAgentStream is the live-hub topic for ephemeral turn snapshots; never persisted.
const TopicAgentStream = "chat.agent.stream"

// StreamFrame is TopicAgentStream's payload.
type StreamFrame struct {
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
	Text           string `json:"text"`
	Streaming      bool   `json:"streaming"`
	// Activity is the latest step's summary since the last text, so the bubble reads "Read main.go" instead of a bare timer.
	Activity     string               `json:"activity,omitempty"`
	ActivityKind harness.ActivityKind `json:"activity_kind,omitempty"`
	ActivityTool string               `json:"activity_tool,omitempty"`
}

// Conversation is the pipeline's slice of a chat conversation (ADR 0017 seam onto chat).
type Conversation struct {
	ID             string
	WorkspaceID    string
	IsTicketThread bool
	TicketID       string
	IsDocThread    bool
	DocID          string
	// ProjectID is set for a thread that belongs to a project directly (an interview thread) rather than through a ticket or doc.
	ProjectID string
	// ProjectName is ProjectID's project name, "" when unread; Name is a channel's name, "" for any other kind.
	ProjectName string
	Name        string
	ThreadID    string
	SyncedAt    time.Time
}

// ConversationMessage is one prior message, before its author's display name is resolved.
type ConversationMessage struct {
	AuthorID   string
	AuthorKind string
	Body       string
	CreatedAt  time.Time
}

// Conversations is the pipeline's consumer-side seam onto chat (ADR 0017).
type Conversations interface {
	GetConversation(ctx context.Context, id string) (Conversation, error)
	MessagesSince(ctx context.Context, conversationID string, since time.Time) ([]ConversationMessage, error)
	SetThread(ctx context.Context, conversationID, threadID string) error
	MarkSynced(ctx context.Context, conversationID string, at time.Time) error
	// PostAgentReply returns the posted message's id so a caller keeping its own record can point at the reply.
	PostAgentReply(ctx context.Context, conversationID, viaUserID, body string) (string, error)
	PostSystemNote(ctx context.Context, conversationID, viaUserID, body string) error
	// PostUserMessage posts as userID themselves: the answer to an Agent question is the user's own message.
	PostUserMessage(ctx context.Context, conversationID, userID, body string) error
}

// Observer receives one turn's lifecycle; callers that keep their own record of a turn (a play's trail) implement it.
type Observer interface {
	OnStarted(sessionID string)
	OnActivity(a harness.Activity)
	// OnSnapshot carries no payload: it is the heartbeat a silence timeout resets on.
	OnSnapshot()
	// OnQuestion means the turn is waiting on the user; it is not terminal, the harness keeps the turn open.
	OnQuestion(q harness.Question)
	OnFinished(result harness.TurnResult, replyMessageID string)
}

type nopObserver struct{}

func (nopObserver) OnStarted(string)                      {}
func (nopObserver) OnActivity(harness.Activity)           {}
func (nopObserver) OnSnapshot()                           {}
func (nopObserver) OnQuestion(harness.Question)           {}
func (nopObserver) OnFinished(harness.TurnResult, string) {}

// TargetResolver is the pipeline's seam onto pairing.
type TargetResolver interface {
	ResolveTarget(ctx context.Context, userID, projectID string) (*pairing.ResolvedTarget, error)
	// ResolveTargetOverride pins the computer, provider, and model a turn resolves against, checked against
	// the caller's own; plain strings, not a TargetOverride, so pairing's implementation needs no agent import.
	ResolveTargetOverride(ctx context.Context, userID, projectID, computerID, provider, model string) (*pairing.ResolvedTarget, error)
}

// TargetOverride pins a turn's computer, provider, and model, chosen for one run instead of derived from
// the caller's project link or pairing defaults (a play's own run dialog, ticket 31).
type TargetOverride struct {
	ComputerID string
	Provider   string
	Model      string
}

// Ticket is the slice of a ticket the pipeline needs for a ticket thread's prompt context and project resolution.
type Ticket struct {
	ProjectID string
	Title     string
	Body      string
}

// TicketReader is the agent pipeline's seam onto tickets (ADR 0017).
type TicketReader interface {
	Get(ctx context.Context, id string) (Ticket, error)
}

// Doc is the slice of a doc the pipeline needs for a doc thread's prompt context and project resolution.
type Doc struct {
	ProjectID    string
	Title        string
	BodyMarkdown string
}

// DocReader is the agent pipeline's seam onto docs (ADR 0017).
type DocReader interface {
	Get(ctx context.Context, id string) (Doc, error)
}

// UserReader resolves a user id to a display label for context lines (ADR 0017 seam onto auth/users).
type UserReader interface {
	Login(ctx context.Context, userID string) (string, error)
}

// MemoriesReader is the pipeline's seam onto memories (ADR 0017); a memory belongs to the workspace or to one
// project (ADR 0059), so every turn resolves its workspace even when projectID is empty (a plain chat).
type MemoriesReader interface {
	ListMemories(ctx context.Context, workspaceID, projectID string) (MemoriesIndex, error)
}

// LivePublisher is the ephemeral live-hub seam; streaming bypasses the outbox.
type LivePublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

// Config wires the agent pipeline.
type Config struct {
	Conversations Conversations
	Targets       TargetResolver
	Harnesses     harness.Registry
	// Tickets is optional; a nil Tickets means ticket threads skip ticket context rather than failing the turn.
	Tickets TicketReader
	// Docs is optional; a nil Docs means doc threads run without doc context rather than failing the turn.
	Docs  DocReader
	Users UserReader
	// Memories is optional; a nil Memories means turns run without a memories index rather than failing the turn.
	Memories MemoriesReader
	// Attachments is optional; nil means images embedded in a ticket or doc body are left as markdown, unresolved.
	Attachments AttachmentReader
	Live        LivePublisher
	Logger      *slog.Logger
	Now         func() time.Time
}

// Service is the agent pipeline: reacts to @Agent mentions and runs a turn.
type Service struct {
	conversations Conversations
	targets       TargetResolver
	harnesses     harness.Registry
	tickets       TicketReader
	docs          DocReader
	users         UserReader
	memories      MemoriesReader
	attachments   AttachmentReader
	live          LivePublisher
	log           *slog.Logger
	now           func() time.Time

	mu     sync.Mutex
	active map[string]activeTurn // conversationID -> in-flight turn, for Interrupt
}

// activeTurn is an in-flight turn: the client it runs on and the target to interrupt.
type activeTurn struct {
	client harness.Client
	target harness.Target
}

// NewService wires the agent pipeline.
func NewService(cfg Config) *Service {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{
		conversations: redactedConversations{cfg.Conversations},
		targets:       cfg.Targets,
		harnesses:     cfg.Harnesses,
		tickets:       cfg.Tickets,
		docs:          cfg.Docs,
		users:         cfg.Users,
		memories:      cfg.Memories,
		attachments:   cfg.Attachments,
		live:          redact.Live{Publisher: cfg.Live},
		log:           cfg.Logger,
		now:           cfg.Now,
		active:        map[string]activeTurn{},
	}
}

// mentionPayload/messageCreatedPayload decode chat.message.created's wire shape (ADR 0017) without importing chat.
type mentionPayload struct {
	Kind string `json:"kind"`
}

type messageCreatedPayload struct {
	Message struct {
		ConversationID string           `json:"conversation_id"`
		AuthorID       string           `json:"author_id"`
		AuthorKind     string           `json:"author_kind"`
		Body           string           `json:"body"`
		Mentions       []mentionPayload `json:"mentions"`
		DeletedAt      *time.Time       `json:"deleted_at,omitempty"`
	} `json:"message"`
}

// mentionAgentKind mirrors chat.MentionAgent's wire value.
const mentionAgentKind = "agent"

// maxTurnDuration bounds a turn; hitting it means the completion signal was lost, not a slow turn.
const maxTurnDuration = 10 * time.Minute

// HandleMessageCreated is the chat.message.created bus subscription.
func (s *Service) HandleMessageCreated(_ context.Context, ev eventbus.Event) error {
	var p messageCreatedPayload
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	// Only human messages trigger a turn: the pipeline's own replies say "@Agent" too and would re-trigger forever.
	if p.Message.AuthorKind != "user" || p.Message.DeletedAt != nil || !mentionsAgent(p.Message.Mentions) {
		return nil
	}
	// ponytail: fire-and-forget, a turn runs minutes and must not block the bus handler.
	go func() {
		// A hung subscription must surface as a system note, not fail silently forever.
		ctx, cancel := context.WithTimeout(context.Background(), maxTurnDuration)
		defer cancel()
		s.RunTurn(ctx, TurnRequest{ConversationID: p.Message.ConversationID, ViaUserID: p.Message.AuthorID, RequestBody: p.Message.Body})
	}()
	return nil
}

func mentionsAgent(mentions []mentionPayload) bool {
	for _, m := range mentions {
		if m.Kind == mentionAgentKind {
			return true
		}
	}
	return false
}

// TurnRequest is one Agent turn to run on a conversation, as ViaUserID.
type TurnRequest struct {
	ConversationID string
	ViaUserID      string
	RequestBody    string
	// ExtraRequestBlocks are appended inside the request block, in order, after the request body.
	ExtraRequestBlocks []string
	// Attachments travel to the harness with the prompt.
	Attachments []harness.Attachment
	// Target is optional; nil means resolve the caller's own project link or pairing defaults as usual.
	Target *TargetOverride
	// Observer is optional; nil means nobody keeps a record beyond the conversation itself.
	Observer Observer
}

// RunTurn runs one Agent turn and blocks until it ends; every failure surfaces as a system note or a log line.
func (s *Service) RunTurn(ctx context.Context, req TurnRequest) {
	obs := req.Observer
	if obs == nil {
		obs = nopObserver{}
	}
	failed := func(reason string) {
		obs.OnFinished(harness.TurnResult{State: harness.TurnError, LastError: reason}, "")
	}
	conversationID, viaUserID := req.ConversationID, req.ViaUserID
	conv, err := s.conversations.GetConversation(ctx, conversationID)
	if err != nil {
		s.log.Error("agent: get conversation failed", "conversation", conversationID, "error", err)
		failed(fmt.Sprintf("get conversation: %v", err))
		return
	}

	ticket, projectID := s.loadTicketContext(ctx, conv)
	doc, docProjectID := s.loadDocContext(ctx, conv, viaUserID)
	if projectID == "" {
		projectID = docProjectID
	}
	if projectID == "" {
		projectID = conv.ProjectID
	}

	target, err := s.resolveTarget(ctx, viaUserID, projectID, req.Target)
	if err != nil {
		s.replyNotConfigured(ctx, conversationID, viaUserID, err)
		failed(err.Error())
		return
	}
	client, ok := s.harnesses[target.Computer.Kind]
	if !ok {
		reason := fmt.Sprintf("no client for harness kind %q", target.Computer.Kind)
		s.postSystemNote(ctx, conversationID, viaUserID, "Agent turn failed to start: "+reason)
		failed(reason)
		return
	}
	s.warnVersionIfChanged(ctx, conversationID, viaUserID, client, target.Computer)

	prompts, err := s.buildTurnPrompts(ctx, conv, ticket, doc, projectID, req)
	if err != nil {
		s.log.Error("agent: list context messages failed", "conversation", conversationID, "error", err)
		failed(fmt.Sprintf("list context messages: %v", err))
		return
	}

	title := threadTitle(conv, ticket, doc)

	turn := activeTurn{client: client, target: harness.Target{
		Session:   target.Computer.Session(),
		ProjectID: target.HarnessProjectID,
		Provider:  target.Provider,
		Model:     target.Model,
		SessionID: conv.ThreadID,
	}}
	s.setActive(conversationID, turn)
	defer s.clearActive(conversationID)

	result, err := client.StartTurn(ctx, turn.target, title, prompts)
	if err != nil {
		s.postSystemNote(ctx, conversationID, viaUserID, fmt.Sprintf("Agent turn failed to start: %v", err))
		failed(err.Error())
		return
	}
	s.announceTurnStarted(ctx, conversationID, conv.ThreadID, &turn, result)
	obs.OnStarted(turn.target.SessionID)

	finalText, term := s.drainTurn(ctx, conversationID, viaUserID, result.Updates, obs)

	if err := s.conversations.MarkSynced(ctx, conversationID, s.now().UTC()); err != nil {
		s.log.Error("agent: mark synced failed", "conversation", conversationID, "error", err)
	}

	s.finishTurn(ctx, conversationID, viaUserID, finalText, term, obs)
}

// threadTitle names a freshly created harness session after what the conversation is about, never its raw id.
func threadTitle(conv Conversation, ticket *TicketContext, doc *DocContext) string {
	if ticket != nil && ticket.Title != "" {
		return ticket.Title
	}
	if doc != nil && doc.Title != "" {
		return doc.Title
	}
	if conv.ProjectName != "" {
		return "Interview: " + conv.ProjectName
	}
	if conv.Name != "" {
		return "#" + conv.Name
	}
	return "Nexul chat"
}

// resolveTarget honors an override, if given, over the caller's own project link or pairing defaults.
func (s *Service) resolveTarget(ctx context.Context, userID, projectID string, override *TargetOverride) (*pairing.ResolvedTarget, error) {
	if override == nil {
		return s.targets.ResolveTarget(ctx, userID, projectID)
	}
	return s.targets.ResolveTargetOverride(ctx, userID, projectID, override.ComputerID, override.Provider, override.Model)
}

// loadTicketContext fetches the thread's linked ticket, if any, for prompt context.
func (s *Service) loadTicketContext(ctx context.Context, conv Conversation) (*TicketContext, string) {
	if !conv.IsTicketThread || conv.TicketID == "" || s.tickets == nil {
		return nil, ""
	}
	t, err := s.tickets.Get(ctx, conv.TicketID)
	if err != nil {
		s.log.Warn("agent: get ticket for thread context failed", "ticket", conv.TicketID, "error", err)
		return nil, ""
	}
	return &TicketContext{Title: t.Title, Body: t.Body}, t.ProjectID
}

// loadDocContext fetches the thread's linked doc, if any, for prompt context; the read is checked
// against viaUserID's own docs:read, matching "acts only within that user's own Nexul permissions".
func (s *Service) loadDocContext(ctx context.Context, conv Conversation, viaUserID string) (*DocContext, string) {
	if !conv.IsDocThread || conv.DocID == "" || s.docs == nil {
		return nil, ""
	}
	d, err := s.docs.Get(identity.WithActor(ctx, identity.Actor{ID: viaUserID}), conv.DocID)
	if err != nil {
		s.log.Warn("agent: get doc for thread context failed", "doc", conv.DocID, "error", err)
		return nil, ""
	}
	return &DocContext{Title: d.Title, Body: d.BodyMarkdown}, d.ProjectID
}

// buildTurnPrompts assembles the full and incremental prompts from unsynced history plus the new request.
func (s *Service) buildTurnPrompts(ctx context.Context, conv Conversation, ticket *TicketContext, doc *DocContext, projectID string, req TurnRequest) (harness.TurnPrompts, error) {
	history, err := s.conversations.MessagesSince(ctx, conv.ID, conv.SyncedAt)
	if err != nil {
		return harness.TurnPrompts{}, err
	}
	ctxMsgs := make([]ContextMessage, len(history))
	for i, m := range history {
		ctxMsgs[i] = ContextMessage{Author: s.authorLabel(ctx, m.AuthorID, m.AuthorKind), Body: m.Body, At: m.CreatedAt}
	}
	budget := NewAttachmentBudget()
	actorCtx := identity.WithActor(ctx, identity.Actor{ID: req.ViaUserID})
	index, alwaysIncludedBlock, memoryAttachments := s.splitMemories(actorCtx, conv.WorkspaceID, projectID, budget)
	targetAttachments := s.extractTargetAttachments(actorCtx, ticket, doc, budget)
	in := PromptInput{
		Ticket:             ticket,
		Doc:                doc,
		ContextMessages:    ctxMsgs,
		Memories:           index,
		RequestAuthor:      s.authorLabel(ctx, req.ViaUserID, "user"),
		RequestBody:        req.RequestBody,
		RequestAt:          s.now().UTC(),
		ExtraRequestBlocks: prependBlock(alwaysIncludedBlock, req.ExtraRequestBlocks),
	}
	attachments := append([]harness.Attachment{}, req.Attachments...)
	attachments = append(attachments, memoryAttachments...)
	attachments = append(attachments, targetAttachments...)
	return harness.TurnPrompts{Full: ComposePrompt(in), Incremental: ComposeIncrementalPrompt(in), Attachments: attachments}, nil
}

// extractTargetAttachments rewrites the ticket or doc body's embedded attachment references in place (they
// are mutually exclusive, one thread carries one target) and returns the images among them for the harness.
func (s *Service) extractTargetAttachments(ctx context.Context, ticket *TicketContext, doc *DocContext, budget *AttachmentBudget) []harness.Attachment {
	if s.attachments == nil {
		return nil
	}
	if ticket != nil {
		rewritten, atts := ExtractAttachments(ctx, ticket.Body, s.attachments, budget)
		ticket.Body = rewritten
		return atts
	}
	if doc != nil {
		rewritten, atts := ExtractAttachments(ctx, doc.Body, s.attachments, budget)
		doc.Body = rewritten
		return atts
	}
	return nil
}

// splitMemories loads a turn's memories index (workspace-scoped plus the project's own) and separates its
// always-included memories, inlined in full as an extra request block with their images extracted for the
// harness, from the index of the rest.
func (s *Service) splitMemories(ctx context.Context, workspaceID, projectID string, budget *AttachmentBudget) (MemoriesIndex, string, []harness.Attachment) {
	index, always := splitAlwaysIncluded(s.loadMemories(ctx, workspaceID, projectID))
	if len(always) == 0 {
		return index, "", nil
	}
	var attachments []harness.Attachment
	if s.attachments != nil {
		for i := range always {
			body, atts := ExtractAttachments(ctx, always[i].Body, s.attachments, budget)
			always[i].Body = body
			attachments = append(attachments, atts...)
		}
	}
	return index, "Always-included memories, follow them:\n" + InlineMemoriesTrimmed(always, DefaultInlineLimits(), s.log), attachments
}

// prependBlock puts b first among extra request blocks, if non-empty; a play run's own blocks follow it.
func prependBlock(b string, blocks []string) []string {
	if b == "" {
		return blocks
	}
	return append([]string{b}, blocks...)
}

// announceTurnStarted shows a "working" indicator immediately and persists a new session id, if any.
func (s *Service) announceTurnStarted(ctx context.Context, conversationID, priorThreadID string, turn *activeTurn, result harness.StartResult) {
	if err := s.live.Publish(ctx, TopicAgentStream, StreamFrame{ConversationID: conversationID, Streaming: true}); err != nil {
		s.log.Warn("agent: publish working frame failed", "conversation", conversationID, "error", err)
	}
	if result.SessionID == "" || result.SessionID == priorThreadID {
		return
	}
	if err := s.conversations.SetThread(ctx, conversationID, result.SessionID); err != nil {
		s.log.Error("agent: persist thread id failed", "conversation", conversationID, "error", err)
	}
	turn.target.SessionID = result.SessionID
	s.setActive(conversationID, *turn)
}

// drainTurn reads a turn's updates to completion, forwarding snapshots and approvals live.
func (s *Service) drainTurn(ctx context.Context, conversationID, viaUserID string, updates <-chan harness.Update, obs Observer) (finalText string, term *harness.TurnResult) {
	frame := StreamFrame{ConversationID: conversationID, Streaming: true}
	publish := func() {
		if err := s.live.Publish(ctx, TopicAgentStream, frame); err != nil {
			s.log.Warn("agent: publish stream frame failed", "conversation", conversationID, "error", err)
		}
	}
	var seg textSegments
	for {
		var u harness.Update
		var ok bool
		select {
		case u, ok = <-updates:
			if !ok {
				return finalText, term
			}
		case <-ctx.Done():
			return finalText, &harness.TurnResult{State: harness.TurnError, LastError: fmt.Sprintf(
				"turn gave no completion signal within %s — the harness may have finished without our client seeing it; check the server log's harness client warnings", maxTurnDuration)}
		}
		switch {
		case u.Snapshot != nil:
			// A streaming:false finalize may carry empty text (a close marker); only non-empty text may set the reply.
			if u.Snapshot.Text != "" {
				finalText = u.Snapshot.Text
			}
			frame.MessageID, frame.Text, frame.Streaming = u.Snapshot.MessageID, u.Snapshot.Text, u.Snapshot.Streaming
			frame.Activity, frame.ActivityKind, frame.ActivityTool = "", "", ""
			publish()
			obs.OnSnapshot()
			seg.observe(*u.Snapshot)
			if !u.Snapshot.Streaming {
				seg.flush(obs)
			}
		case u.Activity != nil:
			frame.Activity, frame.ActivityKind, frame.ActivityTool = u.Activity.Summary, u.Activity.Kind, u.Activity.Tool
			publish()
			if u.Activity.Kind == harness.ActivityToolCall {
				seg.flush(obs)
			}
			obs.OnActivity(*u.Activity)
		case u.Question != nil:
			// The question lands as the Agent's own message so the thread shows the card; the stream bubble yields to it.
			if _, err := s.conversations.PostAgentReply(ctx, conversationID, viaUserID, QuestionMessageBody(*u.Question)); err != nil {
				s.log.Error("agent: post question failed", "conversation", conversationID, "error", err)
			}
			obs.OnQuestion(*u.Question)
		case u.Approval != nil:
			s.postSystemNote(ctx, conversationID, viaUserID, fmt.Sprintf(
				"Agent's environment asked for approval (%s: %s) — auto-declined; adjust its runtime mode if you want it to proceed unattended.",
				u.Approval.Kind, u.Approval.Summary))
		case u.Terminal != nil:
			term = u.Terminal
		}
	}
}

// textSegments cuts one growing reply into the pieces written between tool calls, so a trail reads what the
// Agent said before each action instead of the whole reply once at the end.
type textSegments struct {
	messageID string
	text      string
	emitted   int
}

func (s *textSegments) observe(snap harness.Snapshot) {
	if snap.MessageID != s.messageID {
		s.messageID, s.emitted = snap.MessageID, 0
	}
	if snap.Text != "" {
		s.text = snap.Text
	}
	if s.emitted > len(s.text) {
		s.emitted = 0
	}
}

// flush emits the text not yet seen as a step and marks it seen; nothing new means no step.
func (s *textSegments) flush(obs Observer) {
	piece := strings.TrimSpace(s.text[s.emitted:])
	s.emitted = len(s.text)
	if piece == "" {
		return
	}
	obs.OnActivity(textActivity(piece))
}

// textActivity is one piece of the reply as a transcript step.
func textActivity(text string) harness.Activity {
	return harness.Activity{Kind: harness.ActivityText, Summary: harness.Preview(text, 160), Detail: harness.CapDetail(text), At: time.Now().UTC()}
}

func (s *Service) finishTurn(ctx context.Context, conversationID, viaUserID, finalText string, term *harness.TurnResult, obs Observer) {
	if term == nil {
		term = &harness.TurnResult{State: harness.TurnError, LastError: "turn ended without a terminal result"}
	}
	// Every exit but done-with-text must clear the ephemeral bubble itself, or it spins forever.
	if term.State != harness.TurnDone || strings.TrimSpace(finalText) == "" {
		if err := s.live.Publish(ctx, TopicAgentStream, StreamFrame{ConversationID: conversationID, Streaming: false}); err != nil {
			s.log.Warn("agent: publish clearing frame failed", "conversation", conversationID, "error", err)
		}
	}
	replyID := ""
	switch term.State {
	case harness.TurnDone, harness.TurnInterrupted:
		if strings.TrimSpace(finalText) != "" {
			id, err := s.conversations.PostAgentReply(ctx, conversationID, viaUserID, finalText)
			if err != nil {
				s.log.Error("agent: persist reply failed", "conversation", conversationID, "error", err)
			}
			replyID = id
		}
		if term.State == harness.TurnInterrupted {
			s.postSystemNote(ctx, conversationID, viaUserID, "Agent turn interrupted.")
		}
	case harness.TurnError:
		if !cancelledWithCause(ctx) {
			s.postSystemNote(ctx, conversationID, viaUserID, fmt.Sprintf("Agent turn failed: %s", term.LastError))
		}
	}
	obs.OnFinished(*term, replyID)
}

// cancelledWithCause is true when the caller cancelled the turn with its own reason; that caller owns the thread note.
func cancelledWithCause(ctx context.Context) bool {
	cause := context.Cause(ctx)
	return cause != nil && !errors.Is(cause, context.Canceled) && !errors.Is(cause, context.DeadlineExceeded)
}

// replyNotConfigured turns a ResolveTarget failure into a specific, never-silent system reply.
func (s *Service) replyNotConfigured(ctx context.Context, conversationID, viaUserID string, err error) {
	msg := "Agent isn't configured to run yet — check Settings → Pairing."
	var nc *pairing.NotConfiguredError
	if !errors.As(err, &nc) {
		s.log.Error("agent: resolve target failed", "conversation", conversationID, "error", err)
		s.postSystemNote(ctx, conversationID, viaUserID, msg)
		return
	}
	switch nc.Reason {
	case pairing.ReasonUnpaired:
		msg = "@Agent needs a paired computer — connect one in Settings → Pairing."
	case pairing.ReasonExpiredToken:
		msg = "@Agent's paired computer's session has expired — re-pair it in Settings → Pairing."
	case pairing.ReasonNoDefault:
		msg = "@Agent needs a project on the paired computer to run against — link one in this project's settings, or set a fallback in Settings → Pairing."
	case pairing.ReasonNoDefaultComputer:
		msg = "@Agent found several paired computers — pick a default one in Settings → Pairing."
	case pairing.ReasonSetupRequired, pairing.ReasonOffline:
		msg = nc.Error()
	}
	s.postSystemNote(ctx, conversationID, viaUserID, msg)
}

// warnVersionIfChanged warns, never blocks, when the live harness version differs from pairing time.
func (s *Service) warnVersionIfChanged(ctx context.Context, conversationID, viaUserID string, client harness.Client, computer pairing.Computer) {
	live, err := client.Version(ctx, computer.ServerURL)
	if err != nil {
		s.log.Warn("agent: version probe failed", "server", computer.ServerURL, "error", err)
		return
	}
	if live == "" {
		return // a harness that reports no version cannot drift
	}
	if baseRelease(live) == baseRelease(computer.HarnessVersion) {
		return
	}
	s.postSystemNote(ctx, conversationID, viaUserID, fmt.Sprintf(
		"This computer's agent harness changed release since it was paired (%s → %s) — if the agent misbehaves, re-pair or check Settings → Pairing.",
		computer.HarnessVersion, live))
}

// loadMemories fetches the memories index, best-effort: a failure logs and returns empty, never blocks the turn.
func (s *Service) loadMemories(ctx context.Context, workspaceID, projectID string) MemoriesIndex {
	if s.memories == nil {
		return MemoriesIndex{}
	}
	idx, err := s.memories.ListMemories(ctx, workspaceID, projectID)
	if err != nil {
		s.log.Warn("agent: load memories index failed", "workspace", workspaceID, "project", projectID, "error", err)
		return MemoriesIndex{}
	}
	return idx
}

// baseRelease strips a nightly suffix ("v0.0.34" in "v0.0.34-nightly.5").
func baseRelease(v string) string {
	if i := strings.IndexByte(v, '-'); i >= 0 {
		return v[:i]
	}
	return v
}

func (s *Service) authorLabel(ctx context.Context, userID, kind string) string {
	switch kind {
	case "agent":
		return "Agent"
	case "system":
		return "System"
	}
	if s.users == nil {
		return userID
	}
	login, err := s.users.Login(ctx, userID)
	if err != nil || login == "" {
		return userID
	}
	return login
}

func (s *Service) postSystemNote(ctx context.Context, conversationID, viaUserID, body string) {
	if err := s.conversations.PostSystemNote(ctx, conversationID, viaUserID, body); err != nil {
		s.log.Error("agent: post system note failed", "conversation", conversationID, "error", err)
	}
}

func (s *Service) setActive(conversationID string, t activeTurn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active[conversationID] = t
}

func (s *Service) clearActive(conversationID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.active, conversationID)
}

// questionFence marks an Agent message whose body is a Question as JSON; the web renders it as the card.
const questionFence = "```nexul-question\n"

// QuestionMessageBody renders q as the Agent message the thread shows as a question card.
func QuestionMessageBody(q harness.Question) string {
	b, err := json.Marshal(q)
	if err != nil {
		return "The Agent asked a question that could not be rendered."
	}
	return questionFence + string(b) + "\n```"
}

// Answer resolves a pending question on the conversation's live turn; ErrNotFound when no turn is running here.
func (s *Service) Answer(ctx context.Context, conversationID, requestID string, answer harness.QuestionAnswer) error {
	s.mu.Lock()
	t, ok := s.active[conversationID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: no active agent turn on conversation %s", apperrs.ErrNotFound, conversationID)
	}
	return t.client.Answer(ctx, t.target, requestID, answer)
}

// AnswerFromChat posts the answer as userID's message and resolves the live turn; a turn the harness already closed
// (or one lost to a restart) is resumed as a fresh turn on the same session with the answer as its request.
func (s *Service) AnswerFromChat(ctx context.Context, conversationID, userID, requestID string, answer harness.QuestionAnswer) error {
	if strings.TrimSpace(requestID) == "" {
		return fmt.Errorf("%w: request id is required", apperrs.ErrInvalid)
	}
	if len(answer.Answers) == 0 {
		return fmt.Errorf("%w: an answer is required", apperrs.ErrInvalid)
	}
	body := answer.Summary(nil)
	if err := s.conversations.PostUserMessage(ctx, conversationID, userID, body); err != nil {
		return fmt.Errorf("post answer: %w", err)
	}
	err := s.Answer(ctx, conversationID, requestID, answer)
	if !errors.Is(err, apperrs.ErrNotFound) {
		return err
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), maxTurnDuration)
		defer cancel()
		s.RunTurn(ctx, TurnRequest{ConversationID: conversationID, ViaUserID: userID, RequestBody: body})
	}()
	return nil
}

// Interrupt stops the in-flight turn on a conversation, if any (stop control).
func (s *Service) Interrupt(ctx context.Context, conversationID string) error {
	s.mu.Lock()
	t, ok := s.active[conversationID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: no active agent turn on conversation %s", apperrs.ErrNotFound, conversationID)
	}
	return t.client.Interrupt(ctx, t.target)
}

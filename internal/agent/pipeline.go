package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/chat"
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
	// Handoff is set only on the frame for a hand-off that changed; the web merges it by id.
	Handoff *chat.Handoff `json:"handoff,omitempty"`
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
	// Note is the markdown file a note carries, nil for any other message (ADR 0108).
	Note *NoteFile
}

// NoteFile is a note's markdown file, read as text: it never travels as an attachment.
type NoteFile struct {
	Name     string
	Markdown string
}

// Conversations is the pipeline's consumer-side seam onto chat (ADR 0017).
type Conversations interface {
	GetConversation(ctx context.Context, id string) (Conversation, error)
	MessagesSince(ctx context.Context, conversationID string, since time.Time) ([]ConversationMessage, error)
	SetThread(ctx context.Context, conversationID, threadID string) error
	MarkSynced(ctx context.Context, conversationID string, at time.Time) error
	// PostAgentReply stores the reply and the work it handed off, nil for none, and returns the posted message's id.
	PostAgentReply(ctx context.Context, conversationID, viaUserID, body string, handoffs []harness.Handoff) (string, error)
	PostSystemMessage(ctx context.Context, conversationID, viaUserID, body string) error
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
	ResolveTargetOverride(ctx context.Context, userID, projectID, computerID, provider, model string, options []harness.OptionSetting) (*pairing.ResolvedTarget, error)
}

// TargetOverride pins a turn's computer, provider, and model, chosen for one run instead of derived from
// the caller's project link or pairing defaults (a play's own run dialog, ticket 31).
type TargetOverride struct {
	ComputerID   string
	Provider     string
	Model        string
	ModelOptions []harness.OptionSetting
}

// Ticket is the slice of a ticket the pipeline needs to name it, attach its images, and resolve its project.
type Ticket struct {
	ProjectID string
	// Key is the ticket's key, or its id when its project has no prefix.
	Key   string
	Title string
	// Body is markdown, read only for the images it embeds.
	Body string
	// Done is true when the ticket's column is in the done stage; its thread is then settled once no turn runs.
	Done bool
}

// TicketReader is the agent pipeline's seam onto tickets (ADR 0017).
type TicketReader interface {
	Get(ctx context.Context, id string) (Ticket, error)
}

// Doc is the slice of a doc the pipeline needs to name it, attach its images, and resolve its project.
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

// MemoriesReader is the pipeline's seam onto memories (ADR 0017); a memory belongs to one project (ADR 0099).
type MemoriesReader interface {
	ListMemories(ctx context.Context, projectID string) ([]MemoryItem, error)
}

// Templates reads the instance's agent_prompt templates; templates.Service satisfies it.
type Templates interface {
	Effective(ctx context.Context, kind, key string) (string, error)
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
	// Memories is optional; a nil Memories means a mention names no memories rather than failing the turn.
	Memories MemoriesReader
	// Attachments is optional; nil means images embedded in a ticket or doc body are not attached.
	Attachments AttachmentReader
	// Templates is optional; nil, or a failed read, means the code's DefaultIntro and DefaultFooter.
	Templates Templates
	Live      LivePublisher
	Logger    *slog.Logger
	Now       func() time.Time
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
	templates     Templates
	live          LivePublisher
	log           *slog.Logger
	now           func() time.Time

	mu     sync.Mutex
	active map[string][]*activeTurn // conversationID -> its in-flight turns, oldest first, for Interrupt and Answer
	ended  map[string]endedTurn     // ticketID -> its thread's last turn, until that thread is settled
}

// activeTurn is an in-flight turn: the client it runs on and the target to interrupt.
type activeTurn struct {
	client harness.Client
	target harness.Target
	// answered wakes the turn's silence window once an answer has reached the harness.
	answered chan struct{}
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
		templates:     cfg.Templates,
		live:          redact.Live{Publisher: cfg.Live},
		log:           cfg.Logger,
		now:           cfg.Now,
		active:        map[string][]*activeTurn{},
		ended:         map[string]endedTurn{},
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

// leftRunningLine closes a reply whose turn stopped waiting for work it handed off (ADR 0116).
const leftRunningLine = "Part of this work is still running in T3 Code."

// chatSilence ends a chat turn whose harness has sent nothing for this long; a turn that keeps reporting has no ceiling.
const chatSilence = 15 * time.Minute

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
	go s.runChatTurn(TurnRequest{ConversationID: p.Message.ConversationID, ViaUserID: p.Message.AuthorID, RequestBody: p.Message.Body})
	return nil
}

// runChatTurn runs a chat-started turn under chat's silence window; cancelling on return releases a turn the window ended.
func (s *Service) runChatTurn(req TurnRequest) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req.Silence = chatSilence
	s.RunTurn(ctx, req)
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
	// RequestBody is the message the turn answers; empty for a play run's start, whose request is the play itself.
	RequestBody string
	// Play is set for a play run: it names the run's memories and replaces the conversation history (ADR 0111).
	Play *PlayContext
	// Target is optional; nil means resolve the caller's own project link or pairing defaults as usual.
	Target *TargetOverride
	// Observer is optional; nil means nobody keeps a record beyond the conversation itself.
	Observer Observer
	// Silence ends the turn after this long without a harness update, paused while a question waits; zero means the caller times it.
	Silence time.Duration
	// Answer is set when the turn delivers an answer to a question its earlier turn ended under.
	Answer *harness.PendingAnswer
}

// RunTurn runs one Agent turn and blocks until it ends; every failure surfaces as a system message or a log line.
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

	thread, projectID := s.loadThreadTarget(ctx, conv, viaUserID)
	if projectID == "" {
		projectID = conv.ProjectID
	}

	// Failure posts stay on ctx: the window cancels setupCtx when it cuts setup off, and a post on it would fail.
	setupCtx, disarm := setupWindow(ctx, req.Silence)
	defer disarm()
	target, err := s.resolveTarget(setupCtx, viaUserID, projectID, req.Target)
	if err != nil {
		s.replyNotConfigured(ctx, conversationID, viaUserID, err)
		failed(err.Error())
		return
	}
	client, ok := s.harnesses[target.Computer.Kind]
	if !ok {
		reason := fmt.Sprintf("no client for harness kind %q", target.Computer.Kind)
		s.postSystemMessage(ctx, conversationID, viaUserID, "Agent turn failed to start: "+reason)
		failed(reason)
		return
	}
	s.warnVersionIfChanged(setupCtx, conversationID, viaUserID, client, target.Computer)

	prompts, sentThrough, err := s.buildTurnPrompts(ctx, conv, thread, projectID, req)
	if err != nil {
		s.log.Error("agent: list context messages failed", "conversation", conversationID, "error", err)
		failed(fmt.Sprintf("list context messages: %v", err))
		return
	}

	title := threadTitle(conv, thread)

	turn := &activeTurn{client: client, answered: make(chan struct{}, 1), target: harness.Target{
		Session:      target.Computer.Session(),
		ProjectID:    target.HarnessProjectID,
		Provider:     target.Provider,
		Model:        target.Model,
		ModelOptions: target.ModelOptions,
		SessionID:    conv.ThreadID,
		Worktree:     target.Worktree,
	}}
	s.setActive(conversationID, turn)
	defer s.endTurn(ctx, conv, turn)

	result, err := client.StartTurn(setupCtx, turn.target, title, prompts)
	disarm()
	if err != nil {
		if setupCtx.Err() != nil && ctx.Err() == nil {
			err = context.Cause(setupCtx)
		}
		s.postSystemMessage(ctx, conversationID, viaUserID, fmt.Sprintf("Agent turn failed to start: %v", err))
		failed(err.Error())
		return
	}
	s.announceTurnStarted(ctx, conversationID, conv.ThreadID, turn, result)
	if result.PromptSent {
		s.markSent(ctx, conversationID, sentThrough)
	}
	obs.OnStarted(turn.target.SessionID)

	finalText, handoffs, term := s.drainTurn(ctx, conversationID, viaUserID, result.Updates, obs, newSilenceWindow(req.Silence, turn.answered))

	s.finishTurn(ctx, conversationID, viaUserID, finalText, handoffs, term, obs)
}

// setupWindow cuts off the harness calls made before the stream exists, which have no HTTP timeout, after d; zero is no bound.
func setupWindow(ctx context.Context, d time.Duration) (context.Context, func() bool) {
	if d <= 0 {
		return ctx, func() bool { return false }
	}
	bounded, cancel := context.WithCancelCause(ctx)
	t := time.AfterFunc(d, func() { cancel(fmt.Errorf("the harness did not answer within %s", d)) })
	return bounded, t.Stop
}

// threadTitle names a freshly created harness session after what the conversation is about, never its raw id.
func threadTitle(conv Conversation, target *threadTarget) string {
	if target != nil && target.title != "" {
		return target.title
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
	return s.targets.ResolveTargetOverride(ctx, userID, projectID, override.ComputerID, override.Provider, override.Model, override.ModelOptions)
}

// threadTarget is the ticket or doc a thread belongs to: the prompt line naming it, its title, and the markdown
// body its images are attached from.
type threadTarget struct {
	line  string
	title string
	body  string
}

// loadThreadTarget reads the thread's ticket or doc, if any; a doc is read as viaUserID, under their own docs:read.
func (s *Service) loadThreadTarget(ctx context.Context, conv Conversation, viaUserID string) (*threadTarget, string) {
	if conv.IsTicketThread && conv.TicketID != "" && s.tickets != nil {
		t, err := s.tickets.Get(ctx, conv.TicketID)
		if err != nil {
			s.log.Warn("agent: get ticket for thread context failed", "ticket", conv.TicketID, "error", err)
			return nil, ""
		}
		return &threadTarget{line: fmt.Sprintf(ticketLine, t.Key, t.Title), title: t.Title, body: t.Body}, t.ProjectID
	}
	if !conv.IsDocThread || conv.DocID == "" || s.docs == nil {
		return nil, ""
	}
	d, err := s.docs.Get(identity.WithActor(ctx, identity.Actor{ID: viaUserID}), conv.DocID)
	if err != nil {
		s.log.Warn("agent: get doc for thread context failed", "doc", conv.DocID, "error", err)
		return nil, ""
	}
	return &threadTarget{line: fmt.Sprintf(docLine, d.Title, conv.DocID), title: d.Title, body: d.BodyMarkdown}, d.ProjectID
}

// buildTurnPrompts assembles both prompts and returns the newest message time the conversation has, sent or not.
func (s *Service) buildTurnPrompts(ctx context.Context, conv Conversation, target *threadTarget, projectID string, req TurnRequest) (harness.TurnPrompts, time.Time, error) {
	history, err := s.conversations.MessagesSince(ctx, conv.ID, conv.SyncedAt)
	if err != nil {
		return harness.TurnPrompts{}, time.Time{}, err
	}
	all, others, sentThrough := s.contextMessages(ctx, history, req)
	actorCtx := identity.WithActor(ctx, identity.Actor{ID: req.ViaUserID})
	in := PromptInput{
		Play:          req.Play,
		RequestAuthor: s.authorLabel(ctx, req.ViaUserID, "user"),
		RequestBody:   req.RequestBody,
		RequestAt:     s.now().UTC(),
	}
	in.Memories = s.turnMemories(actorCtx, projectID, req.Play)
	if req.Play != nil {
		all, others = nil, nil
	}
	attachments := s.targetAttachments(actorCtx, target)
	in.Target = targetSection(target, len(attachments))
	incremental := in
	incremental.ContextMessages = others
	in.ContextMessages = all
	in.Intro = s.template(ctx, TemplateIntro, DefaultIntro)
	in.Footer = s.template(ctx, TemplateFooter, DefaultFooter)
	return harness.TurnPrompts{Full: ComposePrompt(in), Incremental: ComposeIncrementalPrompt(incremental), Attachments: attachments, Answer: req.Answer}, sentThrough, nil
}

// contextMessages resolves history; others drops the Agent's own replies, which a live session already holds, but
// keeps notes, which were left over MCP rather than written in this session. The request's own message is left out,
// since the request block already carries it, and so are system messages, which are the server's notices to people.
func (s *Service) contextMessages(ctx context.Context, history []ConversationMessage, req TurnRequest) (all, others []ContextMessage, newest time.Time) {
	skip := requestMessageIndex(history, req)
	for i, m := range history {
		if m.CreatedAt.After(newest) {
			newest = m.CreatedAt
		}
		if i == skip || m.AuthorKind == "system" {
			continue
		}
		cm := ContextMessage{Author: s.authorLabel(ctx, m.AuthorID, m.AuthorKind), Body: noteBody(m), At: m.CreatedAt}
		all = append(all, cm)
		if m.AuthorKind != "agent" || m.Note != nil {
			others = append(others, cm)
		}
	}
	return all, others, newest
}

// requestMessageIndex finds the newest message that is the request itself, or -1 when the request was never posted.
func requestMessageIndex(history []ConversationMessage, req TurnRequest) int {
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m.AuthorKind == "user" && m.AuthorID == req.ViaUserID && m.Body == req.RequestBody {
			return i
		}
	}
	return -1
}

// noteBody follows a note's one-liner with its file, so the turn reads the note rather than a pointer to it.
func noteBody(m ConversationMessage) string {
	if m.Note == nil {
		return m.Body
	}
	return fmt.Sprintf("%s\n\nNote file %s:\n%s", m.Body, m.Note.Name, m.Note.Markdown)
}

// markSent advances the cursor once the harness has the prompt, so a mention mid-turn sends only what came after.
func (s *Service) markSent(ctx context.Context, conversationID string, through time.Time) {
	if through.IsZero() {
		return
	}
	if err := s.conversations.MarkSynced(ctx, conversationID, through); err != nil {
		s.log.Error("agent: mark synced failed", "conversation", conversationID, "error", err)
	}
}

// targetAttachments collects the images the ticket or doc body embeds; the body itself never enters the prompt.
func (s *Service) targetAttachments(ctx context.Context, target *threadTarget) []harness.Attachment {
	if target == nil || s.attachments == nil {
		return nil
	}
	_, atts := ExtractAttachments(ctx, target.body, s.attachments, NewAttachmentBudget())
	return atts
}

// targetSection is the line naming the thread's ticket or doc, with the images line when any image was attached.
func targetSection(target *threadTarget, images int) string {
	if target == nil {
		return ""
	}
	if images == 0 {
		return target.line
	}
	return target.line + "\n" + imagesLine
}

// turnMemories is what a play names, or for a mention the project's always-included memories.
func (s *Service) turnMemories(ctx context.Context, projectID string, play *PlayContext) []MemoryRef {
	if play != nil {
		return play.Memories
	}
	return alwaysIncludedRefs(s.loadMemories(ctx, projectID))
}

// alwaysIncludedRefs names a project's always-included memories for a mention, the interview memory first (ADR 0065).
func alwaysIncludedRefs(items []MemoryItem) []MemoryRef {
	var refs []MemoryRef
	for _, m := range items {
		ref := MemoryRef{ID: m.ID, Name: m.Name}
		if m.Interview {
			refs = append([]MemoryRef{ref}, refs...)
			continue
		}
		if m.AlwaysIncluded {
			refs = append(refs, ref)
		}
	}
	return refs
}

// template reads one of the instance's agent_prompt templates, falling back to the code default on any failure.
func (s *Service) template(ctx context.Context, key, fallback string) string {
	if s.templates == nil {
		return fallback
	}
	body, err := s.templates.Effective(ctx, TemplateKind, key)
	if err != nil {
		s.log.Warn("agent: read prompt template failed", "key", key, "error", err)
		return fallback
	}
	return body
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
	s.mu.Lock()
	turn.target.SessionID = result.SessionID
	s.mu.Unlock()
}

// drainTurn reads a turn's updates to completion, forwarding snapshots, hand-offs and approvals live.
func (s *Service) drainTurn(ctx context.Context, conversationID, viaUserID string, updates <-chan harness.Update, obs Observer, window *silenceWindow) (finalText string, handoffs []harness.Handoff, term *harness.TurnResult) {
	frame := StreamFrame{ConversationID: conversationID, Streaming: true}
	publish := func() {
		if err := s.live.Publish(ctx, TopicAgentStream, frame); err != nil {
			s.log.Warn("agent: publish stream frame failed", "conversation", conversationID, "error", err)
		}
	}
	var seg textSegments
	for {
		u, end, ok := window.receive(ctx, updates)
		if end != nil {
			return finalText, handoffs, end
		}
		if !ok {
			return finalText, handoffs, term
		}
		switch {
		case u.Snapshot != nil:
			// A streaming:false finalize may carry empty text (a close marker); only non-empty text may set the reply.
			if u.Snapshot.Text != "" {
				finalText = u.Snapshot.Text
			}
			// One message finishing is not the turn finishing; the frame streams until finishTurn clears it.
			frame.MessageID, frame.Text = u.Snapshot.MessageID, u.Snapshot.Text
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
			if _, err := s.conversations.PostAgentReply(ctx, conversationID, viaUserID, QuestionMessageBody(*u.Question), nil); err != nil {
				s.log.Error("agent: post question failed", "conversation", conversationID, "error", err)
			}
			obs.OnQuestion(*u.Question)
		case u.Handoff != nil:
			handoffs = keepHandoff(handoffs, *u.Handoff)
			h := chat.NewHandoff(*u.Handoff)
			frame.Handoff = &h
			publish()
			frame.Handoff = nil
			obs.OnSnapshot()
		case u.Approval != nil:
			s.postSystemMessage(ctx, conversationID, viaUserID, fmt.Sprintf(
				"Agent's environment asked for approval (%s: %s) — auto-declined; adjust its runtime mode if you want it to proceed unattended.",
				u.Approval.Kind, u.Approval.Summary))
		case u.Terminal != nil:
			term = u.Terminal
		}
	}
}

// keepHandoff replaces h's earlier snapshot, else appends it, so the reply keeps the order the hand-offs started in.
func keepHandoff(hs []harness.Handoff, h harness.Handoff) []harness.Handoff {
	if i := slices.IndexFunc(hs, func(x harness.Handoff) bool { return x.ID == h.ID }); i >= 0 {
		hs[i] = h
		return hs
	}
	return append(hs, h)
}

// silenceWindow ends a turn after d without a harness update; a question pauses it until answered, and a zero d never fires.
type silenceWindow struct {
	d        time.Duration
	timer    *time.Timer
	paused   bool
	answered <-chan struct{}
}

func newSilenceWindow(d time.Duration, answered <-chan struct{}) *silenceWindow {
	w := &silenceWindow{d: d, answered: answered}
	if d > 0 {
		w.timer = time.NewTimer(d)
	}
	return w
}

// receive returns the next update and whether the stream is still open, or end when the window or ctx closed the turn first.
func (w *silenceWindow) receive(ctx context.Context, updates <-chan harness.Update) (u harness.Update, end *harness.TurnResult, ok bool) {
	var expired <-chan time.Time
	if w.timer != nil {
		expired = w.timer.C
	}
	for {
		select {
		case u, ok = <-updates:
			w.heard(u.Question != nil)
			return u, nil, ok
		case <-w.answered:
			w.paused = false
			w.heard(false)
		case <-expired:
			return u, &harness.TurnResult{State: harness.TurnError, LastError: fmt.Sprintf(
				"turn gave no completion signal after %s of silence — the harness may have finished without our client seeing it; check the server log's harness client warnings", w.d)}, false
		case <-ctx.Done():
			return u, &harness.TurnResult{State: harness.TurnError, LastError: "turn cancelled: " + context.Cause(ctx).Error()}, false
		}
	}
}

// heard restarts the window; a question stops it, since the user's silence is not the harness's.
func (w *silenceWindow) heard(question bool) {
	if w.timer == nil {
		return
	}
	if question {
		w.paused = true
		w.timer.Stop()
		return
	}
	if w.paused {
		return
	}
	w.timer.Reset(w.d)
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

func (s *Service) finishTurn(ctx context.Context, conversationID, viaUserID, finalText string, handoffs []harness.Handoff, term *harness.TurnResult, obs Observer) {
	if term == nil {
		term = &harness.TurnResult{State: harness.TurnError, LastError: "turn ended without a terminal result"}
	}
	if term.LeftRunning {
		finalText = strings.TrimSpace(finalText + "\n\n" + leftRunningLine)
	}
	// Every exit but a persisted done-with-text must clear the ephemeral bubble itself, or it spins forever.
	if term.State != harness.TurnDone || strings.TrimSpace(finalText) == "" {
		s.clearStream(ctx, conversationID)
	}
	replyID := ""
	switch term.State {
	case harness.TurnDone, harness.TurnInterrupted:
		if strings.TrimSpace(finalText) != "" {
			id, err := s.conversations.PostAgentReply(ctx, conversationID, viaUserID, finalText, handoffs)
			if err != nil {
				s.log.Error("agent: persist reply failed", "conversation", conversationID, "error", err)
				s.clearStream(ctx, conversationID)
			}
			replyID = id
		}
		if term.State == harness.TurnInterrupted {
			s.postSystemMessage(ctx, conversationID, viaUserID, "Agent turn interrupted.")
		}
	case harness.TurnError:
		if !cancelledWithCause(ctx) {
			s.postSystemMessage(ctx, conversationID, viaUserID, fmt.Sprintf("Agent turn failed: %s", term.LastError))
		}
	}
	obs.OnFinished(*term, replyID)
}

// clearStream publishes the empty non-streaming frame that removes the ephemeral bubble.
func (s *Service) clearStream(ctx context.Context, conversationID string) {
	if err := s.live.Publish(ctx, TopicAgentStream, StreamFrame{ConversationID: conversationID, Streaming: false}); err != nil {
		s.log.Warn("agent: publish clearing frame failed", "conversation", conversationID, "error", err)
	}
}

// cancelledWithCause is true when the caller cancelled the turn with its own reason; that caller owns the thread note.
func cancelledWithCause(ctx context.Context) bool {
	cause := context.Cause(ctx)
	return cause != nil && !errors.Is(cause, context.Canceled) && !errors.Is(cause, context.DeadlineExceeded)
}

// replyNotConfigured turns a ResolveTarget failure into a specific, never-silent system reply.
func (s *Service) replyNotConfigured(ctx context.Context, conversationID, viaUserID string, err error) {
	msg := "Agent isn't configured to run yet — check Settings → Pairing."
	if errors.Is(err, harness.ErrProtocol) {
		s.postSystemMessage(ctx, conversationID, viaUserID, err.Error())
		return
	}
	var nc *pairing.NotConfiguredError
	if !errors.As(err, &nc) {
		s.log.Error("agent: resolve target failed", "conversation", conversationID, "error", err)
		s.postSystemMessage(ctx, conversationID, viaUserID, msg)
		return
	}
	switch nc.Reason {
	case pairing.ReasonUnpaired:
		msg = "@Agent needs a paired computer — connect one in Settings → Pairing."
	case pairing.ReasonExpiredToken:
		msg = "@Agent's paired computer's session has expired — re-pair it in Settings → Pairing."
	case pairing.ReasonNoDefault:
		msg = "@Agent needs a project on the paired computer to run against — link this project in Settings → T3 pairing → Projects, or set a fallback under Defaults."
	case pairing.ReasonNoDefaultComputer:
		msg = "@Agent found several paired computers — pick a default one in Settings → Pairing."
	case pairing.ReasonSetupRequired, pairing.ReasonOffline:
		msg = nc.Error()
	}
	s.postSystemMessage(ctx, conversationID, viaUserID, msg)
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
	s.postSystemMessage(ctx, conversationID, viaUserID, fmt.Sprintf(
		"This computer's agent harness changed release since it was paired (%s → %s) — if the agent misbehaves, re-pair or check Settings → Pairing.",
		computer.HarnessVersion, live))
}

// loadMemories lists the project's memories, best-effort: a failure logs and returns none, never blocks the turn.
// A turn with no project (a plain chat) names no memories (ADR 0099).
func (s *Service) loadMemories(ctx context.Context, projectID string) []MemoryItem {
	if s.memories == nil || projectID == "" {
		return nil
	}
	items, err := s.memories.ListMemories(ctx, projectID)
	if err != nil {
		s.log.Warn("agent: load memories failed", "project", projectID, "error", err)
		return nil
	}
	return items
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

func (s *Service) postSystemMessage(ctx context.Context, conversationID, viaUserID, body string) {
	if err := s.conversations.PostSystemMessage(ctx, conversationID, viaUserID, body); err != nil {
		s.log.Error("agent: post system message failed", "conversation", conversationID, "error", err)
	}
}

func (s *Service) setActive(conversationID string, t *activeTurn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active[conversationID] = append(s.active[conversationID], t)
}

// clearActive drops only t, so a turn ending first never orphans another turn's Stop or Answer.
func (s *Service) clearActive(conversationID string, t *activeTurn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	turns := slices.DeleteFunc(s.active[conversationID], func(a *activeTurn) bool { return a == t })
	if len(turns) == 0 {
		delete(s.active, conversationID)
		return
	}
	s.active[conversationID] = turns
}

// liveTurns copies the conversation's in-flight turns, newest first.
func (s *Service) liveTurns(conversationID string) []activeTurn {
	s.mu.Lock()
	defer s.mu.Unlock()
	turns := make([]activeTurn, 0, len(s.active[conversationID]))
	for _, t := range slices.Backward(s.active[conversationID]) {
		turns = append(turns, *t)
	}
	return turns
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

// Answer resolves a pending question through the conversation's live turns, newest first; ErrNotFound when none runs here.
func (s *Service) Answer(ctx context.Context, conversationID, requestID string, answer harness.QuestionAnswer) error {
	turns := s.liveTurns(conversationID)
	if len(turns) == 0 {
		return fmt.Errorf("%w: no active agent turn on conversation %s", apperrs.ErrNotFound, conversationID)
	}
	var first error
	for _, t := range turns {
		err := t.client.Answer(ctx, t.target, requestID, answer)
		if errors.Is(err, apperrs.ErrConflict) {
			return err
		}
		if err == nil {
			wakeAll(turns)
			return nil
		}
		if first == nil {
			first = err
		}
	}
	return first
}

// wakeAll restarts the silence window of every turn, since whichever asked is the one paused on the question.
func wakeAll(turns []activeTurn) {
	for _, t := range turns {
		// A wake already pending covers this answer too.
		select {
		case t.answered <- struct{}{}:
		default:
		}
	}
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
	if errors.Is(err, apperrs.ErrConflict) {
		return nil
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return err
	}
	go s.runChatTurn(TurnRequest{ConversationID: conversationID, ViaUserID: userID, RequestBody: body,
		Answer: &harness.PendingAnswer{RequestID: requestID, Answer: answer}})
	return nil
}

// Interrupt stops every in-flight turn on a conversation (stop control); the caller must be able to read it.
func (s *Service) Interrupt(ctx context.Context, conversationID string) error {
	if _, err := s.conversations.GetConversation(ctx, conversationID); err != nil {
		return fmt.Errorf("interrupt agent turn: %w", err)
	}
	turns := s.liveTurns(conversationID)
	if len(turns) == 0 {
		return fmt.Errorf("%w: no active agent turn on conversation %s", apperrs.ErrNotFound, conversationID)
	}
	var first error
	stopped := false
	for _, t := range turns {
		err := t.client.Interrupt(ctx, t.target)
		stopped = stopped || err == nil
		if first == nil {
			first = err
		}
	}
	if stopped {
		return nil
	}
	return first
}

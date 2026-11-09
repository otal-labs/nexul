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
	// ActivityCallID names the tool call, so its result replaces the call in the bubble's steps instead of following it.
	ActivityCallID string `json:"activity_call_id,omitempty"`
	// ActivityText is a reasoning step's whole text, which the bubble shows; Activity stays its one-line preview.
	ActivityText string `json:"activity_text,omitempty"`
	// Handoff is set only on the frame for a hand-off that changed; the web merges it by id.
	Handoff *chat.Handoff `json:"handoff,omitempty"`
	// QueuedMessageID names the request message of a turn that started or stopped waiting in the harness's queue, as
	// Queued says; such a frame carries nothing else and leaves the bubble alone.
	QueuedMessageID string `json:"queued_message_id,omitempty"`
	Queued          bool   `json:"queued,omitempty"`
	// TicketID names a ticket thread's ticket, so its board card shows the turn as work under way.
	TicketID string `json:"ticket_id,omitempty"`
	// Background, when set, is whether the ticket thread has harness work running past its turn; the frame carries nothing else.
	Background *bool `json:"background,omitempty"`
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
	// SeenMarker is the newest harness turn a turn here saw end on ThreadID, a harness.TurnResult.Marker; "" before any.
	SeenMarker string
}

// ConversationMessage is one prior message, before its author's display name is resolved.
type ConversationMessage struct {
	AuthorID   string
	AuthorKind string
	Body       string
	CreatedAt  time.Time
	// Note is the markdown file a note carries, nil for any other message (ADR 0108).
	Note *NoteFile
	// Via names the harness a person's message was relayed from, empty for one written in Nexul; on a bot's message it
	// names the bot behind an overridden name instead.
	Via string
	// AuthorName is the name a bot's message showed, the only name a bot has.
	AuthorName string
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
	// PostHarnessMessage relays a message userID wrote in the harness itself, at the time written; once per key.
	PostHarnessMessage(ctx context.Context, conversationID, userID, body, via, key string, at time.Time) error
	// MarkSeen records the newest harness turn seen to end on the conversation's thread.
	MarkSeen(ctx context.Context, conversationID, marker string) error
	// ConversationByThread is the conversation whose harness thread is threadID; ErrNotFound for none.
	ConversationByThread(ctx context.Context, threadID string) (Conversation, error)
}

// Observer receives one turn's lifecycle; callers that keep their own record of a turn (a play's trail) implement it.
type Observer interface {
	OnStarted(sessionID string)
	OnActivity(a harness.Activity)
	// OnSnapshot carries no payload: it is the heartbeat a silence timeout resets on.
	OnSnapshot()
	// OnQuestion means the turn is waiting on the user; it is not terminal, the harness keeps the turn open.
	OnQuestion(q harness.Question)
	// OnAnswered is a question answered in the harness's own app, never through Nexul's Answer.
	OnAnswered(a harness.AnsweredQuestion)
	OnFinished(result harness.TurnResult, replyMessageID string)
}

type nopObserver struct{}

func (nopObserver) OnStarted(string)                      {}
func (nopObserver) OnActivity(harness.Activity)           {}
func (nopObserver) OnSnapshot()                           {}
func (nopObserver) OnQuestion(harness.Question)           {}
func (nopObserver) OnAnswered(harness.AnsweredQuestion)   {}
func (nopObserver) OnFinished(harness.TurnResult, string) {}

// TargetResolver is the pipeline's seam onto pairing.
type TargetResolver interface {
	ResolveConfirmedTarget(ctx context.Context, userID string, target pairing.ResolvedTarget) (*pairing.ResolvedTarget, error)
	ResolveTarget(ctx context.Context, userID, projectID string) (*pairing.ResolvedTarget, error)
	// ResolveTargetOverride pins the computer, provider, and model a turn resolves against, checked against
	// the caller's own; plain strings, not a TargetOverride, so pairing's implementation needs no agent import.
	ResolveTargetOverride(ctx context.Context, userID, projectID, computerID, provider, model string, options []harness.OptionSetting) (*pairing.ResolvedTarget, error)
}

// TargetOverride pins a turn's computer, provider, and model, chosen for one run instead of derived from
// the caller's project link or pairing defaults (a play's own run dialog, ticket 31).
type TargetOverride struct {
	ComputerID       string
	HarnessProjectID string
	Worktree         bool
	Provider         string
	Model            string
	ModelOptions     []harness.OptionSetting
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
	// Clarified is true once the doc's newest clarification round found no gaps left or was closed.
	Clarified(ctx context.Context, id string) (bool, error)
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
	ended  map[string]endedTurn     // ticket or doc id -> its thread's last turn, until that thread is settled
	// following holds the conversations a catch-up was started on, until its turn ends (ADR 0127).
	following map[string]bool
	// background holds the ticket threads, by harness thread, the harness reports work running on past its turn.
	background map[string]backgroundWork
	follower   ThreadFollower
	continuer  RunContinuer
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
		following:     map[string]bool{},
		background:    map[string]backgroundWork{},
	}
}

// mentionPayload/messageCreatedPayload decode chat.message.created's wire shape (ADR 0017) without importing chat.
type mentionPayload struct {
	Kind string `json:"kind"`
}

type messageCreatedPayload struct {
	Message struct {
		ID             string           `json:"id"`
		ConversationID string           `json:"conversation_id"`
		AuthorID       string           `json:"author_id"`
		AuthorKind     string           `json:"author_kind"`
		Body           string           `json:"body"`
		Mentions       []mentionPayload `json:"mentions"`
		DeletedAt      *time.Time       `json:"deleted_at,omitempty"`
	} `json:"message"`
	Queue bool `json:"queue"`
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
	go s.answerMention(TurnRequest{ConversationID: p.Message.ConversationID, ViaUserID: p.Message.AuthorID, RequestBody: p.Message.Body,
		RequestMessageID: p.Message.ID, Queue: p.Queue})
	return nil
}

// answerMention continues the conversation's ended run when the continuer takes the mention, else runs a chat turn.
func (s *Service) answerMention(req TurnRequest) {
	s.mu.Lock()
	continuer := s.continuer
	s.mu.Unlock()
	if continuer != nil && continuer.ContinueThread(context.Background(), req.ConversationID, req.ViaUserID, req.RequestBody) {
		return
	}
	s.runChatTurn(req)
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
	// RequestMessageID is that message's id, which the thread marks while the harness queues the turn; "" for none.
	RequestMessageID string
	// Queue keeps such a turn waiting behind the turn under way; without it the harness's queue steers it into that turn.
	Queue bool
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
	// Watch follows the turn already running on the conversation's thread, sending nothing; a restart lost its watcher.
	Watch bool
	// Since makes a Watch catch up from the first harness turn after this marker instead of following the newest.
	Since string
	// KeepThread continues the conversation's own harness thread or nothing: a gone one ends the turn SessionGone.
	KeepThread bool
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
	target, err := s.resolveWithin(ctx, setupCtx, viaUserID, projectID, req)
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
	var prompts harness.TurnPrompts
	var sentThrough time.Time
	if !req.Watch {
		s.warnVersionIfChanged(setupCtx, conversationID, viaUserID, client, target.Computer)
		prompts, sentThrough, err = s.buildTurnPrompts(ctx, client, conv, thread, projectID, req)
		if err != nil {
			s.log.Error("agent: list context messages failed", "conversation", conversationID, "error", err)
			failed(fmt.Sprintf("list context messages: %v", err))
			return
		}
	}
	if conv.ThreadID == "" && (req.Watch || req.KeepThread) {
		obs.OnFinished(noThread(req), "")
		return
	}

	turn := &activeTurn{client: client, answered: make(chan struct{}, 1), requestMessageID: req.RequestMessageID, queue: req.Queue, ticketID: conv.TicketID, target: harness.Target{
		Session:      target.Computer.Session(),
		ProjectID:    target.HarnessProjectID,
		Provider:     target.Provider,
		Model:        target.Model,
		ModelOptions: target.ModelOptions,
		SessionID:    conv.ThreadID,
		Worktree:     target.Worktree,
		Since:        req.Since,
		KeepSession:  req.KeepThread,
	}}
	s.setActive(conversationID, turn)
	defer s.endTurn(ctx, conv, turn)

	result, err := openTurn(setupCtx, client, turn.target, threadTitle(conv, thread), prompts, req.Watch)
	disarm()
	if err != nil {
		if setupCtx.Err() != nil && ctx.Err() == nil {
			err = context.Cause(setupCtx)
		}
		s.openFailed(ctx, req, err, obs)
		return
	}
	s.announceTurnStarted(ctx, conversationID, conv.ThreadID, turn, result)
	if result.PromptSent {
		s.markSent(ctx, conversationID, sentThrough)
	}
	obs.OnStarted(turn.target.SessionID)

	finalText, handoffs, term := s.drainTurn(ctx, turn, conversationID, viaUserID, result.Updates, obs, newSilenceWindow(req.Silence, turn.answered))

	s.setQueued(ctx, conversationID, turn, false)
	s.finishTurn(ctx, conversationID, viaUserID, finalText, handoffs, term, obs, turn)
	s.markSeen(ctx, conversationID, term)
}

// markSeen records how far the turn saw its thread end, so only later work there reads as news; a failure is only logged.
func (s *Service) markSeen(ctx context.Context, conversationID string, term *harness.TurnResult) {
	if term == nil || term.Marker == "" {
		return
	}
	if err := s.conversations.MarkSeen(ctx, conversationID, term.Marker); err != nil {
		s.log.Warn("agent: record the turn's end failed", "conversation", conversationID, "error", err)
	}
}

// noThread ends a turn that needs the conversation's harness thread on one that never had one.
func noThread(req TurnRequest) harness.TurnResult {
	if req.KeepThread {
		return harness.TurnResult{State: harness.TurnError, LastError: harness.ErrSessionGone.Error(), SessionGone: true}
	}
	return harness.TurnResult{State: harness.TurnError, LastError: "the conversation has no harness thread to follow"}
}

// openFailed ends a turn the harness would not open; a caller that asked to keep the thread decides what a gone one
// means, so the thread hears nothing then.
func (s *Service) openFailed(ctx context.Context, req TurnRequest, err error, obs Observer) {
	if req.KeepThread && errors.Is(err, harness.ErrSessionGone) {
		obs.OnFinished(harness.TurnResult{State: harness.TurnError, LastError: err.Error(), SessionGone: true}, "")
		return
	}
	s.postSystemMessage(ctx, req.ConversationID, req.ViaUserID, fmt.Sprintf("Agent turn failed to %s: %v", startVerb(req), err))
	obs.OnFinished(harness.TurnResult{State: harness.TurnError, LastError: err.Error()}, "")
}

// openTurn starts the turn, or with watch follows the one already running on t's session.
func openTurn(ctx context.Context, client harness.Client, t harness.Target, title string, prompts harness.TurnPrompts, watch bool) (harness.StartResult, error) {
	if watch {
		return client.Watch(ctx, t)
	}
	return client.StartTurn(ctx, t, title, prompts)
}

func startVerb(req TurnRequest) string {
	if req.Watch {
		return "reconnect"
	}
	return "start"
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

// resolveWithin resolves the turn's target inside the setup window, naming the window's cut-off rather than a bare cancel.
func (s *Service) resolveWithin(ctx, setupCtx context.Context, userID, projectID string, req TurnRequest) (*pairing.ResolvedTarget, error) {
	target, err := s.resolveTarget(setupCtx, userID, projectID, req)
	if err != nil && setupCtx.Err() != nil && ctx.Err() == nil {
		return nil, context.Cause(setupCtx)
	}
	return target, err
}

// resolveTarget keeps a validated launch choice while rechecking its computer, otherwise resolving current settings.
func (s *Service) resolveTarget(ctx context.Context, userID, projectID string, req TurnRequest) (*pairing.ResolvedTarget, error) {
	override := req.Target
	if override == nil {
		return s.targets.ResolveTarget(ctx, userID, projectID)
	}
	if override.HarnessProjectID != "" {
		return s.targets.ResolveConfirmedTarget(ctx, userID, pairing.ResolvedTarget{
			Computer: pairing.Computer{ID: override.ComputerID}, HarnessProjectID: override.HarnessProjectID,
			Provider: override.Provider, Model: override.Model, ModelOptions: override.ModelOptions, Worktree: override.Worktree,
		})
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
func (s *Service) buildTurnPrompts(ctx context.Context, client harness.Client, conv Conversation, target *threadTarget, projectID string, req TurnRequest) (harness.TurnPrompts, time.Time, error) {
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
	targetImgs, contextImgs, requestImgs := s.turnImages(actorCtx, client, target, others, req.RequestBody)
	in.Target = targetSection(target, targetImgs)
	in.RequestBody = nameAttached(req.RequestBody, requestImgs)
	all, others = nameAttachedIn(all, contextImgs), nameAttachedIn(others, contextImgs)
	incremental := in
	incremental.ContextMessages = others
	in.ContextMessages = all
	in.Intro = s.template(ctx, TemplateIntro, DefaultIntro)
	in.Footer = s.template(ctx, TemplateFooter, DefaultFooter)
	return harness.TurnPrompts{Full: ComposePrompt(in), Incremental: ComposeIncrementalPrompt(incremental), Attachments: attached(slices.Concat(targetImgs, contextImgs, requestImgs)),
		IncrementalAttachments: attached(slices.Concat(contextImgs, requestImgs)), Answer: req.Answer}, sentThrough, nil
}

// contextMessages resolves history; others drops the Agent's own replies and the messages relayed from the harness,
// which a live session already holds, but keeps notes, which were left over MCP rather than written in this session. The request's own message is left out,
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
		author := s.authorLabel(ctx, m.AuthorID, m.AuthorKind)
		if m.AuthorKind == "bot" {
			author = m.AuthorName + " (bot)"
		}
		cm := ContextMessage{Author: author, Body: noteBody(m), At: m.CreatedAt}
		all = append(all, cm)
		relayed := m.Via != "" && m.AuthorKind != "bot"
		if !relayed && (m.AuthorKind != "agent" || m.Note != nil) {
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

// turnImages reads the images the request, the people's messages sent with it, and the ticket or doc body embed,
// attaching those client sends its agent; they are read in that order, so the request's keep their place within the turn's total.
func (s *Service) turnImages(ctx context.Context, client harness.Client, target *threadTarget, msgs []ContextMessage, request string) (targetImgs, contextImgs, requestImgs []bodyImage) {
	if s.attachments == nil {
		return nil, nil, nil
	}
	takes := func(string) bool { return true }
	if t, ok := client.(harness.ImageTaker); ok {
		takes = t.TakesImage
	}
	budget := MaxTurnAttachmentBytes
	requestImgs = bodyImages(ctx, request, s.attachments, takes, &budget)
	bodies := make([]string, len(msgs))
	for i, m := range msgs {
		bodies[i] = m.Body
	}
	contextImgs = bodyImages(ctx, strings.Join(bodies, "\n"), s.attachments, takes, &budget)
	if target != nil {
		targetImgs = bodyImages(ctx, target.body, s.attachments, takes, &budget)
	}
	return targetImgs, contextImgs, requestImgs
}

func attached(images []bodyImage) []harness.Attachment {
	var out []harness.Attachment
	for _, img := range images {
		if img.attachment != nil {
			out = append(out, *img.attachment)
		}
	}
	return out
}

// targetSection names the thread's ticket or doc, listing by link each image not attached so it never claims one.
func targetSection(target *threadTarget, images []bodyImage) string {
	if target == nil {
		return ""
	}
	lines := []string{target.line}
	var links []string
	for _, img := range images {
		if img.attachment == nil {
			links = append(links, "- "+img.name+": "+img.link)
		}
	}
	if len(links) > 0 {
		lines = append(lines, linksLine)
		lines = append(lines, links...)
	}
	sent := len(images) - len(links)
	if sent > 0 && len(links) == 0 {
		lines = append(lines, imagesLine)
	}
	if sent > 0 && len(links) > 0 {
		lines = append(lines, otherImagesLine)
	}
	return strings.Join(lines, "\n")
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

// announceTurnStarted shows a "working" indicator at once, names the turn for Stop, and persists a new session id, if any.
func (s *Service) announceTurnStarted(ctx context.Context, conversationID, priorThreadID string, turn *activeTurn, result harness.StartResult) {
	if err := s.live.Publish(ctx, TopicAgentStream, StreamFrame{ConversationID: conversationID, TicketID: turn.ticketID, Streaming: true}); err != nil {
		s.log.Warn("agent: publish working frame failed", "conversation", conversationID, "error", err)
	}
	s.mu.Lock()
	turn.target.TurnID = result.TurnID
	s.mu.Unlock()
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

func (s *Service) finishTurn(ctx context.Context, conversationID, viaUserID, finalText string, handoffs []harness.Handoff, term *harness.TurnResult, obs Observer, turn *activeTurn) {
	if term == nil {
		term = &harness.TurnResult{State: harness.TurnError, LastError: "turn ended without a terminal result"}
	}
	if term.LeftRunning {
		finalText = strings.TrimSpace(finalText + "\n\n" + leftRunningLine)
	}
	// Every exit but a persisted done-with-text must clear the ephemeral bubble itself, or it spins forever, unless the
	// bubble is another turn's still under way, such as the one a steered turn's message joined.
	if (term.State != harness.TurnDone || strings.TrimSpace(finalText) == "") && !s.underWayBeside(conversationID, turn) {
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
	msg := "Agent isn't configured to run yet — check Settings → Computers."
	if errors.Is(err, harness.ErrProtocol) {
		s.postSystemMessage(ctx, conversationID, viaUserID, err.Error())
		return
	}
	var nc *pairing.NotConfiguredError
	if !errors.As(err, &nc) {
		s.log.Error("agent: resolve target failed", "conversation", conversationID, "error", err)
		s.postSystemMessage(ctx, conversationID, viaUserID, "@Agent couldn't start: "+err.Error())
		return
	}
	switch nc.Reason {
	case pairing.ReasonUnpaired:
		msg = "@Agent needs a paired computer — add one in Settings → Computers."
	case pairing.ReasonExpiredToken:
		msg = "@Agent's paired computer's session has expired — re-pair it in Settings → Computers."
	case pairing.ReasonNoComputerReady:
		msg = "@Agent couldn't start: " + strings.TrimPrefix(nc.Error(), "none of your computers can take it: ")
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
		"This computer's agent harness changed release since it was paired (%s → %s) — if the agent misbehaves, re-pair or check Settings → Computers.",
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

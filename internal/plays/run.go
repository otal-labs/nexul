package plays

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/redact"
)

// Resource types the access overwrite table keys per-resource permissions on; they mirror access's own.
const (
	resourceTypePlay    = "play"
	resourceTypeDoc     = "doc"
	resourceTypeProject = "project"
)

// HarnessSilenceTimeout ends a run whose harness has sent nothing for this long; a run that keeps reporting has no ceiling.
const HarnessSilenceTimeout = 15 * time.Minute

// silenceInterruptTimeout bounds the stop a silent run sends its harness, which may be as unresponsive as the run.
const silenceInterruptTimeout = 30 * time.Second

// TicketTarget is the slice of a ticket the runner needs: its project, its key, and the stage of its current column.
type TicketTarget struct {
	ProjectID string
	Key       string
	Stage     Stage
}

// DocTarget is the slice of a doc the runner needs.
type DocTarget struct {
	ProjectID string
	Title     string
}

// StatusTarget is the slice of a board column the move-to needs.
type StatusTarget struct {
	Name  string
	Stage Stage
}

// TargetReader is the runner's seam onto tickets, docs, and board columns (ADR 0017); a deleted column is ErrNotFound.
type TargetReader interface {
	GetTicket(ctx context.Context, id string) (TicketTarget, error)
	GetDoc(ctx context.Context, id string) (DocTarget, error)
	GetStatus(ctx context.Context, id string) (StatusTarget, error)
}

// docLockedNote is the trail's and the thread's line for a doc play locking its doc (ADR 0107); a Clarify run's lock
// comes off when it ends (ADR 0121).
const (
	docLockedNote     = "Locked the doc because the run started; it stays locked after the run ends."
	clarifyLockedNote = "Locked the doc while this round runs; it unlocks when the run ends."
)

// DocLocker locks a doc play's doc as the run starts, whatever the starter's own bits; true when it was unlocked.
type DocLocker interface {
	LockForPlay(ctx context.Context, docID string) (bool, error)
}

// ClarifyRounds opens a Clarify run's round on its doc and ends it at any end of the run, unlocking the doc if the
// run took its lock (ADR 0121); ending a trail that opened no round does nothing.
type ClarifyRounds interface {
	OpenRound(ctx context.Context, docID, starterID, trailID string, tookLock bool) (int, error)
	EndRound(ctx context.Context, docID, trailID string) error
}

// ProjectTarget is the slice of a project an interview run needs; TestsLocation is the wizard's answer, "" when unanswered.
type ProjectTarget struct {
	Name          string
	TestsLocation string
}

// ProjectLookup resolves a project's workspace for the play check and reads the project an interview run targets.
type ProjectLookup interface {
	WorkspaceForProject(ctx context.Context, projectID string) (string, error)
	GetProject(ctx context.Context, projectID string) (ProjectTarget, error)
}

// HarnessChoice is the computer, provider, and model a run uses: either the caller's pick from the run
// dialog, or what the resolution picked from their project link or pairing defaults when they picked none.
type HarnessChoice struct {
	ComputerID string
	Worktree   bool
	// HarnessProjectID is the T3 project a person picked beside ComputerID, saved as their project link as the run starts.
	HarnessProjectID string
	Provider         string
	Model            string
	ModelOptions     []harness.OptionSetting
}

// HarnessResolver is the runner's seam onto pairing: the readiness check before a run is started, given the
// caller's pick (a zero HarnessChoice means none), and what it actually resolved to for the trail to record.
type HarnessResolver interface {
	// ResolveTarget resolves a run nobody is there to ask, falling back to the starter's pairing defaults.
	ResolveTarget(ctx context.Context, userID, projectID string, choice HarnessChoice) (HarnessChoice, error)
	// ResolvePersonTarget requires an explicit location or the caller's project link (ADR 0145).
	ResolvePersonTarget(ctx context.Context, userID, projectID string, choice HarnessChoice) (HarnessChoice, error)
}

// RefusalNeedsLocation is the refusal of a person's run in a project they have not linked: a question, not a failure.
const RefusalNeedsLocation = "needs_location"

// HarnessRefusal is a resolver refusal naming its computer and provider, kept on the failed trail so the web can offer the fix.
type HarnessRefusal struct {
	Reason     string
	ComputerID string
	Provider   string
	Err        error
}

func (e *HarnessRefusal) Error() string { return e.Err.Error() }

func (e *HarnessRefusal) Unwrap() error { return e.Err }

// Memory is the slice of a memory the runner names for the agent to read; never its body (ADR 0105).
type Memory struct {
	ID             string
	Title          string
	AlwaysIncluded bool
	Interview      bool
	Footer         bool
}

// MemoryReader is the runner's seam onto memories (ADR 0017).
type MemoryReader interface {
	ListForProject(ctx context.Context, projectID string) ([]Memory, error)
}

// skippedAnswer is the text the Interview page sends for a skipped follow-up, since a harness may refuse an empty answer.
const skippedAnswer = "Skipped"

// FollowUp is one question an interview run asked and the answer it got; no picks and no text is a skip.
type FollowUp struct {
	Question    string
	Why         string
	Options     []harness.QuestionOption
	MultiSelect bool
	Selected    []string
	Text        string
}

// InterviewSource is one source of a project's interview as the caller reads it; Unreadable when they cannot see its ref.
type InterviewSource struct {
	ID         string
	Kind       string
	Ref        string
	Label      string
	Stance     string
	Unreadable bool
}

// Interview source kinds and stances the runner acts on; they mirror the memories domain's.
const (
	SourceKindProject = "project"
	StanceFollow      = "follow"
	StanceQuestion    = "question"
)

// InterviewAnswers is the runner's seam onto a project's stored interview (ADR 0017): rounds, sources, and drafts.
type InterviewAnswers interface {
	RecordRound(ctx context.Context, projectID, answeredBy string, followUps []FollowUp) error
	ListSources(ctx context.Context, projectID string) ([]InterviewSource, error)
	// ClearSuggestions deletes the project's drafts on questions that already have an answer.
	ClearSuggestions(ctx context.Context, projectID string) error
}

// Checkouts is the runner's seam onto pairing for where another project's checkout is on a computer.
type Checkouts interface {
	// LinkedProject is userID's project link for projectID: its computer and T3 project, empty when unset.
	LinkedProject(ctx context.Context, userID, projectID string) (computerID, harnessProjectID string, err error)
	ListProjects(ctx context.Context, userID, computerID string) ([]harness.Project, error)
}

// Threads is the runner's seam onto chat: the target's thread, the starter's request in it, and the run's notes.
type Threads interface {
	GetOrCreateTicketThread(ctx context.Context, workspaceID, ticketID, userID string) (string, error)
	GetOrCreateDocThread(ctx context.Context, workspaceID, docID, userID string) (string, error)
	GetOrCreateInterviewThread(ctx context.Context, workspaceID, projectID, userID string) (string, error)
	PostMessage(ctx context.Context, conversationID, authorID, body string) (string, error)
	PostSystemMessage(ctx context.Context, conversationID, viaUserID, body string) error
}

// TurnRunner is the runner's seam onto the agent pipeline: start a turn, stop or answer the one on a conversation.
type TurnRunner interface {
	RunTurn(ctx context.Context, req agent.TurnRequest)
	Interrupt(ctx context.Context, conversationID string) error
	// Answer resolves the live turn's pending question; ErrNotFound when no turn is running on the conversation.
	Answer(ctx context.Context, conversationID, requestID string, answer harness.QuestionAnswer) error
}

// errTurnGone says the turn an answer was meant for no longer runs here; the answer resumes the run as a fresh turn.
var errTurnGone = errors.New("turn gone")

// LivePublisher is the ephemeral live-hub seam; trail frames bypass the outbox.
type LivePublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

// UserReader resolves a stopper's login for the trail's note, and the login a moment names to the user who caused it.
type UserReader interface {
	Login(ctx context.Context, userID string) (string, error)
	UserID(ctx context.Context, login string) (string, error)
}

// RunnerConfig wires the run pipeline.
type RunnerConfig struct {
	Plays    Repo
	Trails   TrailRepo
	Perm     PermissionGate
	Targets  TargetReader
	Docs     DocLocker
	Projects ProjectLookup
	Harness  HarnessResolver
	Memories MemoryReader
	Threads  Threads
	Turns    TurnRunner
	Live     LivePublisher
	Users    UserReader
	// Links is optional; nil means a ticket play runs without its found-in and blocked-by context.
	Links LinkReader
	// Answers is optional; nil means an interview run's follow-ups live only on the trail and it names no sources.
	Answers InterviewAnswers
	// Rounds is optional; nil means a Clarify run opens no round.
	Rounds ClarifyRounds
	// Checkouts is optional; nil names every project source with no checkout on the run's computer.
	Checkouts Checkouts
	// Queue and Facts are optional; without both no auto play matches and the queue never runs.
	Queue  QueueRepo
	Facts  FactReader
	Logger *slog.Logger
	Now    func() time.Time
	// SilenceTimeout defaults to HarnessSilenceTimeout; tests shorten it.
	SilenceTimeout time.Duration
}

// Runner starts play runs and keeps their trails (ADR 0055).
type Runner struct {
	plays     Repo
	trails    TrailRepo
	perm      PermissionGate
	targets   TargetReader
	docs      DocLocker
	rounds    ClarifyRounds
	projects  ProjectLookup
	harness   HarnessResolver
	memories  MemoryReader
	threads   Threads
	turns     TurnRunner
	live      LivePublisher
	users     UserReader
	links     LinkReader
	answers   InterviewAnswers
	checkouts Checkouts
	queue     QueueRepo
	facts     FactReader
	log       *slog.Logger
	now       func() time.Time
	silence   time.Duration
	// kick wakes the queue's dispatcher; one buffered kick is enough, since a pass reads everything due.
	kick chan struct{}

	mu   sync.Mutex
	runs map[string]*trailObserver // trail id -> the live run, so it can be stopped
}

// NewRunner wires the run pipeline over its seams.
func NewRunner(cfg RunnerConfig) *Runner {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.SilenceTimeout <= 0 {
		cfg.SilenceTimeout = HarnessSilenceTimeout
	}
	if cfg.Live != nil {
		cfg.Live = redact.Live{Publisher: cfg.Live}
	}
	return &Runner{
		plays: cfg.Plays, trails: redactedTrails{cfg.Trails}, perm: cfg.Perm, targets: cfg.Targets, docs: cfg.Docs, rounds: cfg.Rounds, projects: cfg.Projects,
		harness: cfg.Harness, memories: cfg.Memories, threads: redactedThreads{cfg.Threads}, turns: cfg.Turns,
		live: cfg.Live, users: cfg.Users, links: cfg.Links, answers: cfg.Answers, checkouts: cfg.Checkouts, queue: cfg.Queue, facts: cfg.Facts,
		log: cfg.Logger, now: cfg.Now, silence: cfg.SilenceTimeout, kick: make(chan struct{}, 1),
		runs: map[string]*trailObserver{},
	}
}

// RunInput is one press of a play. ComputerID, Provider, and Model are optional: left blank, the run
// resolves the caller's own project link or pairing defaults exactly as a chat mention does.
type RunInput struct {
	PlayID             string
	TargetType         TargetType
	TargetID           string
	MemoryIDs          []string
	CustomInstructions string
	ComputerID         string
	HarnessProjectID   string
	Provider           string
	Model              string
	ModelOptions       []harness.OptionSetting
	Via                Via
}

// Choices is what the run dialog pre-selects from the starter's latest trail of a play in a project.
type Choices struct {
	MemoryIDs    []string                `json:"memory_ids"`
	ComputerID   string                  `json:"computer_id"`
	Provider     string                  `json:"provider"`
	Model        string                  `json:"model"`
	ModelOptions []harness.OptionSetting `json:"model_options"`
}

// target is what the runner read about a ticket, doc, or interview at press time.
type target struct {
	projectID     string
	stage         Stage
	title         string
	testsLocation string
}

func (r *Runner) finish(ctx context.Context, trail *Trail, targetTitle string, result harness.TurnResult, replyMessageID, note string) {
	now := r.now().UTC()
	trail.State = trailStateOf(result.State)
	trail.EndedAt = &now
	trail.LastError = result.LastError
	trail.ReplyMessageID = replyMessageID
	if note != "" {
		r.note(ctx, trail, note)
	}
	r.save(ctx, trail, r.finishedEvent(trail, targetTitle))
	r.endRound(ctx, trail)
}

func (r *Runner) createFailed(ctx context.Context, trail *Trail, targetTitle, reason string) {
	now := r.now().UTC()
	trail.State, trail.EndedAt, trail.LastError = TrailFailed, &now, reason
	var evts []eventbus.OutboxEvent
	// play.run_finished names its starter; a check nobody could run on is kept on the ticket alone.
	if trail.StarterID != "" {
		evts = append(evts, r.finishedEvent(trail, targetTitle))
	}
	if err := r.trails.CreateTrail(ctx, trail, evts...); err != nil {
		r.log.Error("plays: record failed trail", "trail", trail.ID, "error", err)
	}
	r.publishLive(ctx, trail)
}

func (r *Runner) save(ctx context.Context, trail *Trail, evts ...eventbus.OutboxEvent) {
	if err := r.trails.UpdateTrail(ctx, trail, evts...); err != nil {
		r.log.Error("plays: update trail failed", "trail", trail.ID, "state", trail.State, "error", err)
	}
	r.publishLive(ctx, trail)
}

func (r *Runner) publishLive(ctx context.Context, trail *Trail) {
	if r.live == nil {
		return
	}
	if err := r.live.Publish(ctx, TopicPlayRun, runFrame(trail)); err != nil {
		r.log.Warn("plays: publish run frame failed", "trail", trail.ID, "error", err)
	}
}

// note records the runner's own line on the trail and in the thread; the trail's copy is the caller's to save.
func (r *Runner) note(ctx context.Context, trail *Trail, body string) {
	trail.AppendActivity(ActivityEntry{Kind: ActivityNote, Summary: body, At: r.now().UTC()})
	if trail.ConversationID == "" {
		return
	}
	if err := r.threads.PostSystemMessage(ctx, trail.ConversationID, trail.StarterID, body); err != nil {
		r.log.Error("plays: post system message failed", "trail", trail.ID, "error", err)
	}
}

func (r *Runner) startedEvent(trail *Trail, targetTitle string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicRunStarted, Payload: RunStartedEvent{
		RunRef: runRef(trail, targetTitle), HarnessSessionID: trail.HarnessSessionID,
	}}
}

func (r *Runner) waitingEvent(trail *Trail, targetTitle string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicRunWaiting, Payload: RunWaitingEvent{RunRef: runRef(trail, targetTitle)}}
}

func (r *Runner) finishedEvent(trail *Trail, targetTitle string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicRunFinished, Payload: RunFinishedEvent{
		RunRef: runRef(trail, targetTitle), Outcome: trail.State, LastError: trail.LastError, ReplyMessageID: trail.ReplyMessageID,
	}}
}

func trailStateOf(s harness.TurnState) TrailState {
	switch s {
	case harness.TurnDone:
		return TrailDone
	case harness.TurnInterrupted:
		return TrailInterrupted
	}
	return TrailFailed
}

func formatDuration(d time.Duration) string {
	if d >= time.Minute && d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return d.String()
}

func (r *Runner) login(ctx context.Context, userID string) string {
	if r.users == nil {
		return userID
	}
	login, err := r.users.Login(ctx, userID)
	if err != nil || login == "" {
		return userID
	}
	return login
}

func startedMessage(label, custom string) string {
	if custom == "" {
		return "Started " + label
	}
	return "Started " + label + "\n\n" + custom
}

// interviewBlock names the project an interview run is for and the answers the project already records.
func interviewBlock(tgt target) string {
	tests := "not answered yet; ask it"
	if tgt.testsLocation == "same" {
		tests = "in the deployed repository"
	}
	if tgt.testsLocation == "separate" {
		tests = "in a separate tests repository"
	}
	return fmt.Sprintf("Interview for project %q (project id %s).\nWhere its tests live, as answered in the project wizard: %s.", tgt.title, tgt.projectID, tests)
}

func stageName(s *Stage) string {
	if s == nil {
		return "(none)"
	}
	return string(*s)
}

func (r *Runner) setRun(trailID string, o *trailObserver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs[trailID] = o
}

// clearRun releases o's entry only; a resumed turn on the same trail must not be dropped by the old turn's exit.
func (r *Runner) clearRun(trailID string, o *trailObserver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runs[trailID] == o {
		delete(r.runs, trailID)
	}
}

func (r *Runner) run(trailID string) *trailObserver {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runs[trailID]
}

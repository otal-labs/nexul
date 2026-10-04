package plays

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
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
	ComputerID   string
	Provider     string
	Model        string
	ModelOptions []harness.OptionSetting
}

// HarnessResolver is the runner's seam onto pairing: the readiness check before a run is started, given the
// caller's pick (a zero HarnessChoice means none), and what it actually resolved to for the trail to record.
type HarnessResolver interface {
	ResolveTarget(ctx context.Context, userID, projectID string, choice HarnessChoice) (HarnessChoice, error)
}

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

// UserReader resolves a stopper's login for the trail's note, and a developer's login to the user the decisions check runs as.
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
	Logger    *slog.Logger
	Now       func() time.Time
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
	log       *slog.Logger
	now       func() time.Time
	silence   time.Duration

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
		live: cfg.Live, users: cfg.Users, links: cfg.Links, answers: cfg.Answers, checkouts: cfg.Checkouts, log: cfg.Logger, now: cfg.Now, silence: cfg.SilenceTimeout,
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

// Run checks the play, the target, the caller, and the harness, then starts the turn in the background and
// returns the trail in state starting; the trail's later states are the observer's to record.
func (r *Runner) Run(ctx context.Context, in RunInput) (*Trail, error) {
	starter := actorID(ctx)
	if starter == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	play, tgt, err := r.checkPlayAndTarget(ctx, starter, in)
	if err != nil {
		return nil, err
	}
	trail := &Trail{
		ID: ids.New(), WorkspaceID: play.WorkspaceID, PlayID: play.ID, PlayLabel: play.Label,
		TargetType: in.TargetType, TargetID: strings.TrimSpace(in.TargetID), ProjectID: tgt.projectID,
		StarterID: starter, Via: in.Via, SelectedMemoryIDs: normalizeIDs(in.MemoryIDs),
		CustomInstructions: strings.TrimSpace(in.CustomInstructions),
		State:              TrailStarting, StartedAt: r.now().UTC(), Activity: []ActivityEntry{},
	}
	options, err := harness.CleanOptions(in.ModelOptions)
	if err != nil {
		return nil, err
	}
	return r.launch(ctx, play, trail, tgt, HarnessChoice{ComputerID: in.ComputerID, Provider: in.Provider, Model: in.Model, ModelOptions: options}, false)
}

// launch resolves the harness and starts the turn; a harness refusal is always kept as a failed trail, recordRefusals keeps the rest too.
func (r *Runner) launch(ctx context.Context, play *Play, trail *Trail, tgt target, pick HarnessChoice, recordRefusals bool) (*Trail, error) {
	targetTitle := tgt.title
	refuse := func(err error) (*Trail, error) {
		if recordRefusals {
			r.createFailed(ctx, trail, targetTitle, err.Error())
		}
		return nil, err
	}
	choice, err := r.harness.ResolveTarget(ctx, trail.StarterID, trail.ProjectID, pick)
	if err != nil {
		var refusal *HarnessRefusal
		if errors.As(err, &refusal) {
			trail.FailureReason, trail.ComputerID, trail.Provider = refusal.Reason, refusal.ComputerID, refusal.Provider
		}
		r.createFailed(ctx, trail, targetTitle, err.Error())
		return nil, err
	}
	trail.ComputerID, trail.Provider, trail.Model, trail.ModelOptions = choice.ComputerID, choice.Provider, choice.Model, choice.ModelOptions
	if err := r.refuseIfActive(ctx, trail.TargetType, trail.TargetID); err != nil {
		return refuse(err)
	}
	memories, err := r.memoriesToRead(ctx, trail.ProjectID, trail.SelectedMemoryIDs)
	if err != nil {
		return refuse(err)
	}
	trail.SelectedMemoryIDs = memories.ids()
	links, err := r.linkBlocks(ctx, trail.TargetType, trail.TargetID)
	if err != nil {
		return refuse(err)
	}
	links = append(links, r.interviewBlocks(ctx, play, trail, tgt)...)
	drafting := trail.TargetType == TargetInterview && play.BuiltinKey == DraftInterviewKey
	conversationID, err := r.openThread(ctx, play.WorkspaceID, trail.TargetType, trail.TargetID, trail.StarterID)
	if err != nil {
		return refuse(err)
	}
	trail.ConversationID = conversationID
	if err := r.trails.CreateTrail(ctx, trail); err != nil {
		return nil, fmt.Errorf("create trail for play %s: %w", play.ID, err)
	}
	body := startedMessage(play.Label, trail.CustomInstructions)
	if _, err := r.threads.PostMessage(ctx, conversationID, trail.StarterID, body); err != nil {
		reason := "post started message: " + err.Error()
		r.finish(ctx, trail, targetTitle, harness.TurnResult{State: harness.TurnError, LastError: reason}, "", "Run failed: "+reason)
		return nil, fmt.Errorf("post started message: %w", err)
	}
	if err := r.prepareDoc(ctx, play, trail); err != nil {
		r.finish(ctx, trail, targetTitle, harness.TurnResult{State: harness.TurnError, LastError: err.Error()}, "", "Run failed: "+err.Error())
		return nil, err
	}
	if drafting {
		r.clearSuggestions(ctx, trail)
	}
	// Copied before the turn starts: from here on the observer's goroutine owns trail.
	snapshot := *trail
	r.startTurn(ctx, trail, targetTitle, agent.TurnRequest{
		ConversationID: conversationID, ViaUserID: trail.StarterID,
		Play:   &agent.PlayContext{Label: play.Label, Instructions: play.Instructions, Blocks: links, Memories: memories.read, Custom: trail.CustomInstructions, Conclude: memories.conclude},
		Target: &agent.TargetOverride{ComputerID: choice.ComputerID, Provider: choice.Provider, Model: choice.Model, ModelOptions: choice.ModelOptions},
	}, false)
	return &snapshot, nil
}

// clearSuggestions drops the project's suggested changes as a drafting run starts, so only those it drafts again come back;
// a failure is logged, never the run's.
func (r *Runner) clearSuggestions(ctx context.Context, trail *Trail) {
	if r.answers == nil {
		return
	}
	if err := r.answers.ClearSuggestions(ctx, trail.ProjectID); err != nil {
		r.log.Warn("plays: clear suggested changes failed", "trail", trail.ID, "project", trail.ProjectID, "error", err)
	}
}

// interviewBlocks names an interview run's project, then a drafting run's context or the project sources under question.
func (r *Runner) interviewBlocks(ctx context.Context, play *Play, trail *Trail, tgt target) []string {
	if trail.TargetType != TargetInterview {
		return nil
	}
	blocks := []string{interviewBlock(tgt)}
	if play.BuiltinKey == DraftInterviewKey {
		return append(blocks, r.draftingBlock(ctx, trail))
	}
	if sources := r.projectSourcesBlock(ctx, trail, StanceQuestion); sources != "" {
		blocks = append(blocks, sources)
	}
	return blocks
}

// draftingBlock tells a drafting run its trail id, for the drafts it saves, and where its follow project sources are.
func (r *Runner) draftingBlock(ctx context.Context, trail *Trail) string {
	block := fmt.Sprintf("This run's trail id is %s; pass it as `trail_id` on every draft.", trail.ID)
	if sources := r.projectSourcesBlock(ctx, trail, StanceFollow); sources != "" {
		block += "\n" + sources
	}
	return block
}

// projectSourcesBlock names the interview's project sources of stance with each one's checkout on the run's computer,
// read through the starter's project link for that project; "" when there are none. Unreadable sources are left out.
func (r *Runner) projectSourcesBlock(ctx context.Context, trail *Trail, stance string) string {
	if r.answers == nil {
		return ""
	}
	ctx = identity.WithActor(ctx, identity.Actor{ID: trail.StarterID})
	srcs, err := r.answers.ListSources(ctx, trail.ProjectID)
	if err != nil {
		r.log.Warn("plays: list interview sources failed", "trail", trail.ID, "project", trail.ProjectID, "error", err)
		return ""
	}
	var lines []string
	paths := r.checkoutPaths(ctx, trail)
	for _, src := range srcs {
		if src.Kind != SourceKindProject || src.Stance != stance || src.Unreadable {
			continue
		}
		where := "no checkout on this computer; use only its memories and interview answers"
		if path := paths(src.Ref); path != "" {
			where = "its checkout on this computer is " + path
		}
		lines = append(lines, fmt.Sprintf("- %q (project id %s, source id %s): %s.", src.Label, src.Ref, src.ID, where))
	}
	if len(lines) == 0 {
		return ""
	}
	return fmt.Sprintf("Project sources with stance %s:\n%s", stance, strings.Join(lines, "\n"))
}

// checkoutPaths returns a lookup of a project's checkout path on the run's computer: the starter's project link for it
// must name that computer and a T3 project the computer still lists. The computer's list is read once, on first need.
func (r *Runner) checkoutPaths(ctx context.Context, trail *Trail) func(projectID string) string {
	var listed map[string]string
	return func(projectID string) string {
		if r.checkouts == nil || trail.ComputerID == "" {
			return ""
		}
		computerID, harnessProjectID, err := r.checkouts.LinkedProject(ctx, trail.StarterID, projectID)
		if err != nil {
			r.log.Warn("plays: read project link failed", "trail", trail.ID, "project", projectID, "error", err)
			return ""
		}
		if computerID != trail.ComputerID || harnessProjectID == "" {
			return ""
		}
		if listed == nil {
			listed = map[string]string{}
			projects, err := r.checkouts.ListProjects(ctx, trail.StarterID, trail.ComputerID)
			if err != nil {
				r.log.Warn("plays: list the computer's projects failed", "trail", trail.ID, "computer", trail.ComputerID, "error", err)
				projects = nil
			}
			for _, p := range projects {
				listed[p.ID] = p.Path
			}
		}
		return listed[harnessProjectID]
	}
}

// prepareDoc locks a doc play's doc and, for a Clarify run, opens its round, which the agent posts to (ADR 0121).
func (r *Runner) prepareDoc(ctx context.Context, play *Play, trail *Trail) error {
	if trail.TargetType != TargetDoc {
		return nil
	}
	clarify := play.BuiltinKey == ClarifyKey && r.rounds != nil
	tookLock := r.lockDoc(ctx, trail, clarify)
	if !clarify {
		return nil
	}
	// ponytail: a failed open leaves the lock this run took on the doc; a docs:lock holder unlocks it until plays gets an unlock seam.
	if _, err := r.rounds.OpenRound(ctx, trail.TargetID, trail.StarterID, trail.ID, tookLock); err != nil {
		return fmt.Errorf("open the clarification round: %w", err)
	}
	return nil
}

// lockDoc locks a doc play's doc as the run starts (ADR 0107), true when this run took the lock; a failed lock is
// logged, never the run's failure.
func (r *Runner) lockDoc(ctx context.Context, trail *Trail, clarify bool) bool {
	if r.docs == nil {
		return false
	}
	locked, err := r.docs.LockForPlay(ctx, trail.TargetID)
	if err != nil {
		r.log.Warn("plays: lock doc failed", "trail", trail.ID, "doc", trail.TargetID, "error", err)
		return false
	}
	if !locked {
		return false
	}
	note := docLockedNote
	if clarify {
		note = clarifyLockedNote
	}
	r.note(ctx, trail, note)
	r.save(ctx, trail)
	return true
}

// endRound ends the Clarify round a doc run opened, if any; a failure is logged, since the trail has ended anyway.
func (r *Runner) endRound(ctx context.Context, trail *Trail) {
	if r.rounds == nil || trail.TargetType != TargetDoc {
		return
	}
	if err := r.rounds.EndRound(ctx, trail.TargetID, trail.ID); err != nil {
		r.log.Error("plays: end clarification round failed", "trail", trail.ID, "doc", trail.TargetID, "error", err)
	}
}

// Answer resolves a waiting run's question: the answer is posted as the caller's message and handed to the live
// turn; a turn the harness already closed under the question resumes as a fresh turn on the same session, under
// the same trail. The starter or a plays:write holder may answer.
func (r *Runner) Answer(ctx context.Context, trailID string, answer harness.QuestionAnswer) (*Trail, error) {
	trailID = strings.TrimSpace(trailID)
	if trailID == "" {
		return nil, fmt.Errorf("%w: trail id is required", apperrs.ErrInvalid)
	}
	if len(answer.Answers) == 0 {
		return nil, fmt.Errorf("%w: an answer is required", apperrs.ErrInvalid)
	}
	trail, err := r.trails.GetTrail(ctx, trailID)
	if err != nil {
		return nil, fmt.Errorf("get trail %s: %w", trailID, err)
	}
	actor := actorID(ctx)
	if actor != trail.StarterID && !r.perm.HasPermission(ctx, actor, trail.WorkspaceID, permissions.PlaysWrite, "", "") {
		return nil, fmt.Errorf("%w: only the starter or a %s holder can answer", apperrs.ErrForbidden, permissions.PlaysWrite)
	}
	if trail.State != TrailWaiting || trail.Question == nil {
		return nil, fmt.Errorf("%w: the run is not waiting for an answer", apperrs.ErrConflict)
	}
	body := answer.Summary(&trail.Question.Question)
	if _, err := r.threads.PostMessage(ctx, trail.ConversationID, actor, body); err != nil {
		return nil, fmt.Errorf("post answer: %w", err)
	}
	if o := r.run(trailID); o != nil {
		err := o.answer(ctx, answer)
		if err == nil {
			r.recordFollowUps(ctx, trail, actor, answer)
			return r.trails.GetTrail(ctx, trailID)
		}
		if !errors.Is(err, errTurnGone) {
			return nil, err
		}
	}
	trail.Question.Answer = &answer
	trail.State = TrailRunning
	r.save(ctx, trail)
	r.recordFollowUps(ctx, trail, actor, answer)
	tgt, err := r.readTarget(ctx, trail.TargetType, trail.TargetID)
	if err != nil {
		r.log.Warn("plays: answer could not read the target", "trail", trailID, "error", err)
	}
	snapshot := *trail
	r.startTurn(ctx, trail, tgt.title, agent.TurnRequest{
		ConversationID: trail.ConversationID, ViaUserID: trail.StarterID, RequestBody: body,
		Answer: &harness.PendingAnswer{RequestID: trail.Question.RequestID, Answer: answer},
		Play:   r.answerContext(ctx, trail),
		Target: &agent.TargetOverride{ComputerID: trail.ComputerID, Provider: trail.Provider, Model: trail.Model, ModelOptions: trail.ModelOptions},
	}, true)
	return &snapshot, nil
}

// recordFollowUps keeps an interview run's answered question as the project's next round; a failed record is only logged.
func (r *Runner) recordFollowUps(ctx context.Context, trail *Trail, answeredBy string, answer harness.QuestionAnswer) {
	if r.answers == nil || trail.TargetType != TargetInterview {
		return
	}
	items := trail.Question.Questions
	followUps := make([]FollowUp, 0, len(items))
	for _, item := range items {
		v := answer.Answers[item.ID]
		if len(v.Selected) == 0 && strings.TrimSpace(v.Text) == skippedAnswer {
			v.Text = ""
		}
		question, why := splitWhy(item.Text)
		followUps = append(followUps, FollowUp{
			Question: question, Why: why, Options: item.Options, MultiSelect: item.MultiSelect,
			Selected: optionLabels(item.Options, v.Selected), Text: v.Text,
		})
	}
	if err := r.answers.RecordRound(ctx, trail.TargetID, answeredBy, followUps); err != nil {
		r.log.Error("plays: record interview follow-ups failed", "trail", trail.ID, "project", trail.TargetID, "error", err)
	}
}

// splitWhy splits a follow-up's text at the first "?" followed by whitespace: the question, then the why sentence after it.
func splitWhy(text string) (question, why string) {
	for i := 0; i+1 < len(text); i++ {
		if text[i] == '?' && unicode.IsSpace(rune(text[i+1])) {
			return text[:i+1], strings.TrimSpace(text[i+1:])
		}
	}
	return text, ""
}

// optionLabels names each pick by its option's label, which is what the stored options keep; a value no option has stays as given.
func optionLabels(options []harness.QuestionOption, selected []string) []string {
	out := make([]string, 0, len(selected))
	for _, v := range selected {
		i := slices.IndexFunc(options, func(o harness.QuestionOption) bool { return o.Value != "" && o.Value == v })
		if i >= 0 {
			v = options[i].Label
		}
		out = append(out, v)
	}
	return out
}

// checkPlayAndTarget is the run's first gate: the play, its type against the target, the workspace, enabled,
// exclusion, the caller's plays:run on this play, and a ticket play's stage.
func (r *Runner) checkPlayAndTarget(ctx context.Context, starter string, in RunInput) (*Play, target, error) {
	if !in.TargetType.valid() {
		return nil, target{}, fmt.Errorf("%w: target type must be ticket, doc, or interview", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.TargetID) == "" {
		return nil, target{}, fmt.Errorf("%w: target id is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.PlayID) == "" {
		return nil, target{}, fmt.Errorf("%w: play id is required", apperrs.ErrInvalid)
	}
	play, err := r.plays.Get(ctx, strings.TrimSpace(in.PlayID))
	if err != nil {
		return nil, target{}, fmt.Errorf("get play %s: %w", in.PlayID, err)
	}
	if string(play.Type) != string(in.TargetType) {
		return nil, target{}, fmt.Errorf("%w: %s plays run on %s targets", apperrs.ErrInvalid, play.Type, play.Type)
	}
	tgt, err := r.readTarget(ctx, in.TargetType, strings.TrimSpace(in.TargetID))
	if err != nil {
		return nil, target{}, err
	}
	if err := r.checkPlay(ctx, starter, play, tgt.projectID, tgt.stage); err != nil {
		return nil, target{}, err
	}
	return play, tgt, nil
}

func (r *Runner) checkPlay(ctx context.Context, starter string, play *Play, projectID string, stage Stage) error {
	workspaceID, err := r.projects.WorkspaceForProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("resolve workspace for project %s: %w", projectID, err)
	}
	if play.WorkspaceID != workspaceID {
		return fmt.Errorf("get play %s: %w", play.ID, apperrs.ErrNotFound)
	}
	if !play.Enabled {
		return fmt.Errorf("%w: play %q is disabled", apperrs.ErrInvalid, play.Label)
	}
	if !r.perm.HasPermission(ctx, starter, workspaceID, permissions.Member, resourceTypeProject, projectID) {
		return fmt.Errorf("get project %s: %w", projectID, apperrs.ErrNotFound)
	}
	if slices.Contains(play.ExcludedProjectIDs, projectID) {
		return fmt.Errorf("%w: play %q is excluded from this project", apperrs.ErrInvalid, play.Label)
	}
	if !r.perm.HasPermission(ctx, starter, workspaceID, permissions.PlaysRun, resourceTypePlay, play.ID) {
		return fmt.Errorf("%w: %s required on play %q", apperrs.ErrForbidden, permissions.PlaysRun, play.Label)
	}
	if play.Type == TypeTicket && (play.ShowWhenStage == nil || *play.ShowWhenStage != stage) {
		return fmt.Errorf("%w: play %q runs on tickets in the %s stage; this ticket is in %s", apperrs.ErrInvalid, play.Label, stageName(play.ShowWhenStage), stage)
	}
	return nil
}

func (r *Runner) readTarget(ctx context.Context, targetType TargetType, targetID string) (target, error) {
	if targetType == TargetInterview {
		p, err := r.projects.GetProject(ctx, targetID)
		if err != nil {
			return target{}, fmt.Errorf("get project %s: %w", targetID, err)
		}
		return target{projectID: targetID, title: p.Name, testsLocation: p.TestsLocation}, nil
	}
	if targetType == TargetDoc {
		d, err := r.targets.GetDoc(ctx, targetID)
		if err != nil {
			return target{}, fmt.Errorf("get doc %s: %w", targetID, err)
		}
		return target{projectID: d.ProjectID, title: d.Title}, nil
	}
	t, err := r.targets.GetTicket(ctx, targetID)
	if err != nil {
		return target{}, fmt.Errorf("get ticket %s: %w", targetID, err)
	}
	return target{projectID: t.ProjectID, stage: t.Stage, title: t.Key}, nil
}

func (r *Runner) refuseIfActive(ctx context.Context, targetType TargetType, targetID string) error {
	existing, err := r.trails.ListTrailsByTarget(ctx, targetType, targetID)
	if err != nil {
		return fmt.Errorf("list trails for %s %s: %w", targetType, targetID, err)
	}
	for _, t := range existing {
		if t.State.Active() {
			return fmt.Errorf("%w: a run is in progress", apperrs.ErrConflict)
		}
	}
	return nil
}

// memoriesToRead names the run's memories for the agent to read itself, refusing an id outside the project: the
// interview memory, then the other always-included ones, then the rest of the selection; footer memories come apart.
func (r *Runner) memoriesToRead(ctx context.Context, projectID string, selected []string) (runMemories, error) {
	all, err := r.memories.ListForProject(ctx, projectID)
	if err != nil {
		return runMemories{}, fmt.Errorf("list memories for project %s: %w", projectID, err)
	}
	ms, missing := orderMemories(all, selected)
	if len(missing) > 0 {
		return runMemories{}, fmt.Errorf("%w: memory %s is not in this project", apperrs.ErrInvalid, missing[0])
	}
	return ms, nil
}

// answerContext is a resumed run's play part for a session the harness may have lost: the play and the trail's
// recorded memories, read as its starter. A failed read leaves that part out, so the answer still goes through.
func (r *Runner) answerContext(ctx context.Context, trail *Trail) *agent.PlayContext {
	play, err := r.playOf(ctx, trail)
	if err != nil {
		r.log.Warn("plays: answer could not read the run's play", "trail", trail.ID, "error", err)
		play = &Play{}
	}
	pc := &agent.PlayContext{Label: play.Label, Instructions: play.Instructions}
	all, err := r.memories.ListForProject(identity.WithActor(ctx, identity.Actor{ID: trail.StarterID}), trail.ProjectID)
	if err != nil {
		r.log.Warn("plays: answer could not list the run's memories", "trail", trail.ID, "error", err)
		return pc
	}
	ms, _ := orderMemories(all, trail.SelectedMemoryIDs)
	pc.Memories, pc.Conclude = ms.read, ms.conclude
	return pc
}

// playOf reads a trail's play; the decisions check has no stored row.
func (r *Runner) playOf(ctx context.Context, trail *Trail) (*Play, error) {
	if trail.PlayID == DecisionsCheckPlayID {
		return decisionsCheckPlay(trail.WorkspaceID, true), nil
	}
	return r.plays.Get(ctx, trail.PlayID)
}

// runMemories are a run's memories as the agent reads them: first, and the footer ones it concludes with (ADR 0112).
type runMemories struct {
	read     []agent.MemoryRef
	conclude []agent.MemoryRef
}

// ids lists the run's memories in the order the agent reads them, footers last.
func (ms runMemories) ids() []string {
	ids := make([]string, 0, len(ms.read)+len(ms.conclude))
	for _, m := range slices.Concat(ms.read, ms.conclude) {
		ids = append(ids, m.ID)
	}
	return ids
}

// orderMemories orders a selection the way the agent reads it, and returns the selected ids the project lacks.
func orderMemories(all []Memory, selected []string) (runMemories, []string) {
	byID := make(map[string]Memory, len(all))
	var ordered []Memory
	for _, m := range all {
		byID[m.ID] = m
		if m.Interview {
			ordered = append(ordered, m)
		}
	}
	for _, m := range all {
		if m.AlwaysIncluded && !m.Interview {
			ordered = append(ordered, m)
		}
	}
	var missing []string
	for _, id := range selected {
		m, ok := byID[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		if !m.AlwaysIncluded && !m.Interview {
			ordered = append(ordered, m)
		}
	}
	var ms runMemories
	for _, m := range ordered {
		ref := agent.MemoryRef{ID: m.ID, Name: m.Title}
		if m.Footer {
			ms.conclude = append(ms.conclude, ref)
			continue
		}
		ms.read = append(ms.read, ref)
	}
	return ms, missing
}

func (r *Runner) openThread(ctx context.Context, workspaceID string, targetType TargetType, targetID, starter string) (string, error) {
	if targetType == TargetInterview {
		id, err := r.threads.GetOrCreateInterviewThread(ctx, workspaceID, targetID, starter)
		if err != nil {
			return "", fmt.Errorf("open interview thread for project %s: %w", targetID, err)
		}
		return id, nil
	}
	if targetType == TargetDoc {
		id, err := r.threads.GetOrCreateDocThread(ctx, workspaceID, targetID, starter)
		if err != nil {
			return "", fmt.Errorf("open doc thread for %s: %w", targetID, err)
		}
		return id, nil
	}
	id, err := r.threads.GetOrCreateTicketThread(ctx, workspaceID, targetID, starter)
	if err != nil {
		return "", fmt.Errorf("open ticket thread for %s: %w", targetID, err)
	}
	return id, nil
}

// startTurn runs the turn detached from the request's cancellation: it outlives the HTTP call, and chat's
// silence window is chat's, not a play's. A resumed turn continues an existing trail, so it announces no start.
func (r *Runner) startTurn(ctx context.Context, trail *Trail, targetTitle string, req agent.TurnRequest, resumed bool) {
	runCtx, cancel := context.WithCancelCause(context.WithoutCancel(ctx))
	o := &trailObserver{r: r, trail: trail, targetTitle: targetTitle, ctx: context.WithoutCancel(runCtx), cancel: cancel, resumed: resumed}
	o.timer = time.AfterFunc(r.silence, o.onSilence)
	r.setRun(trail.ID, o)
	req.Observer = o
	go func() {
		defer r.clearRun(trail.ID, o)
		defer cancel(nil)
		r.turns.RunTurn(runCtx, req)
	}()
}

// trailObserver records the turn's lifecycle on the trail; pipeline, silence timer, and Stop race for it, first terminal wins.
type trailObserver struct {
	r           *Runner
	trail       *Trail
	targetTitle string
	resumed     bool
	ctx         context.Context // detached, so the terminal state is saved even after the turn is cancelled
	cancel      context.CancelCauseFunc

	mu         sync.Mutex
	timer      *time.Timer
	done       bool
	stopReason string
}

func (o *trailObserver) OnStarted(sessionID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return
	}
	o.timer.Reset(o.r.silence)
	o.trail.State = TrailRunning
	o.trail.HarnessSessionID = sessionID
	if o.resumed {
		o.r.save(o.ctx, o.trail)
		return
	}
	o.r.save(o.ctx, o.trail, o.r.startedEvent(o.trail, o.targetTitle))
}

// ponytail: one trail write per activity step; batch them if a busy turn ever makes the writer visible.
func (o *trailObserver) OnActivity(a harness.Activity) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return
	}
	o.resetSilence()
	o.trail.AppendActivity(ActivityEntry(a))
	o.r.save(o.ctx, o.trail)
}

func (o *trailObserver) OnSnapshot() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return
	}
	o.resetSilence()
}

// resetSilence restarts the silence window unless the run is waiting on the user, whose silence is not the harness's.
func (o *trailObserver) resetSilence() {
	if o.trail.State == TrailWaiting {
		return
	}
	o.timer.Reset(o.r.silence)
}

// OnQuestion parks the run: the trail waits, the silence timer stops, and the starter is told the run needs them.
func (o *trailObserver) OnQuestion(q harness.Question) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return
	}
	o.timer.Stop()
	o.trail.State = TrailWaiting
	o.trail.Question = &TrailQuestion{Question: q, AskedAt: o.r.now().UTC()}
	o.r.save(o.ctx, o.trail, o.r.waitingEvent(o.trail, o.targetTitle))
}

// answer hands the answer to the live turn and sets the run going again; errTurnGone when the turn is no longer here.
func (o *trailObserver) answer(ctx context.Context, answer harness.QuestionAnswer) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done || o.trail.Question == nil {
		return errTurnGone
	}
	err := o.r.turns.Answer(ctx, o.trail.ConversationID, o.trail.Question.RequestID, answer)
	if errors.Is(err, apperrs.ErrNotFound) {
		return errTurnGone
	}
	// Answered already, in the harness or by a racing submit: the turn has its answer, so the run carries on.
	if err != nil && !errors.Is(err, apperrs.ErrConflict) {
		return fmt.Errorf("answer harness question: %w", err)
	}
	o.trail.Question.Answer = &answer
	o.trail.State = TrailRunning
	o.timer.Reset(o.r.silence)
	o.r.save(o.ctx, o.trail)
	return nil
}

func (o *trailObserver) OnFinished(result harness.TurnResult, replyMessageID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return
	}
	o.done = true
	o.timer.Stop()
	if o.trail.State == TrailWaiting && o.stopReason == "" && result.State != harness.TurnInterrupted {
		// The harness closed the turn under an unanswered question; the trail keeps waiting and the answer resumes it.
		return
	}
	if o.stopReason != "" {
		// Stop was requested on this trail; a race with the harness's own terminal state never reads as done.
		result.State = harness.TurnInterrupted
		if result.LastError == "" {
			result.LastError = o.stopReason
		}
	}
	o.r.finish(o.ctx, o.trail, o.targetTitle, result, replyMessageID, "")
}

// onSilence fails the run when the window elapses with nothing from the harness; done makes the pipeline's later terminal a no-op.
func (o *trailObserver) onSilence() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return
	}
	o.done = true
	reason := "no harness update for " + formatDuration(o.r.silence)
	o.r.finish(o.ctx, o.trail, o.targetTitle, harness.TurnResult{State: harness.TurnError, LastError: reason}, "", "Run failed: "+reason)
	o.cancel(errors.New(reason))
}

// stop asks the harness to interrupt; when it cannot, the run is closed here and the turn's context cancelled.
func (o *trailObserver) stop(ctx context.Context, reason string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return fmt.Errorf("%w: the run has already ended", apperrs.ErrConflict)
	}
	o.stopReason = reason
	o.r.note(ctx, o.trail, "Run "+reason+".")
	if err := o.r.turns.Interrupt(ctx, o.trail.ConversationID); err != nil {
		o.r.log.Warn("plays: harness interrupt failed, closing the trail", "trail", o.trail.ID, "error", err)
		o.done = true
		o.timer.Stop()
		o.r.finish(o.ctx, o.trail, o.targetTitle, harness.TurnResult{State: harness.TurnInterrupted, LastError: reason}, "", "")
		o.cancel(errors.New(reason))
	}
	return nil
}

// Stop interrupts an active run; the starter or a plays:write holder may, and the trail keeps its record.
func (r *Runner) Stop(ctx context.Context, trailID string) (*Trail, error) {
	trailID = strings.TrimSpace(trailID)
	if trailID == "" {
		return nil, fmt.Errorf("%w: trail id is required", apperrs.ErrInvalid)
	}
	trail, err := r.trails.GetTrail(ctx, trailID)
	if err != nil {
		return nil, fmt.Errorf("get trail %s: %w", trailID, err)
	}
	actor := actorID(ctx)
	if actor != trail.StarterID && !r.perm.HasPermission(ctx, actor, trail.WorkspaceID, permissions.PlaysWrite, "", "") {
		return nil, fmt.Errorf("%w: only the starter or a %s holder can stop a run", apperrs.ErrForbidden, permissions.PlaysWrite)
	}
	if !trail.State.Active() {
		return nil, fmt.Errorf("%w: the run has already ended", apperrs.ErrConflict)
	}
	reason := "stopped by " + r.login(ctx, actor)
	if o := r.run(trailID); o != nil {
		if err := o.stop(ctx, reason); err != nil {
			return nil, err
		}
		return r.trails.GetTrail(ctx, trailID)
	}
	// No live turn on this server (a restart mid-run): closing the record here frees the target again.
	tgt, err := r.readTarget(ctx, trail.TargetType, trail.TargetID)
	if err != nil {
		r.log.Warn("plays: stop could not read the target", "trail", trailID, "error", err)
	}
	r.finish(ctx, trail, tgt.title, harness.TurnResult{State: harness.TurnInterrupted, LastError: reason}, "", "Run "+reason+".")
	return trail, nil
}

// ResumeRunsAfterRestart follows again the runs whose turns only the previous process watched; call it at boot, before any run starts.
func (r *Runner) ResumeRunsAfterRestart(ctx context.Context) error {
	trails, err := r.trails.ListRunningTrails(ctx)
	if err != nil {
		return fmt.Errorf("list running trails: %w", err)
	}
	for _, trail := range trails {
		tgt, err := r.readTarget(ctx, trail.TargetType, trail.TargetID)
		if err != nil {
			r.log.Warn("plays: restart could not read the run's target", "trail", trail.ID, "error", err)
		}
		// A run the harness never accepted has no thread to follow.
		if trail.HarnessSessionID == "" || trail.ConversationID == "" {
			const reason = "Nexul restarted before the run started"
			r.finish(ctx, trail, tgt.title, harness.TurnResult{State: harness.TurnInterrupted, LastError: reason}, "", reason+".")
			continue
		}
		r.note(ctx, trail, "Nexul restarted; following the run in T3 Code again.")
		r.save(ctx, trail)
		r.startTurn(ctx, trail, tgt.title, agent.TurnRequest{
			ConversationID: trail.ConversationID, ViaUserID: trail.StarterID, Watch: true,
			Target: &agent.TargetOverride{ComputerID: trail.ComputerID, Provider: trail.Provider, Model: trail.Model, ModelOptions: trail.ModelOptions},
		}, true)
	}
	return nil
}

// finish is the run's terminal step; note is the runner's own reason for the thread, empty when the pipeline already posted one.
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

// GetTrail returns one trail; plays:read in its workspace, and docs:thread on the doc for a doc trail.
func (r *Runner) GetTrail(ctx context.Context, id string) (*Trail, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: trail id is required", apperrs.ErrInvalid)
	}
	t, err := r.trails.GetTrail(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get trail %s: %w", id, err)
	}
	if !r.opensProject(ctx, t) {
		return nil, fmt.Errorf("get trail %s: %w", id, apperrs.ErrNotFound)
	}
	if err := r.requireTrailAccess(ctx, t.WorkspaceID, t.TargetType, t.TargetID); err != nil {
		return nil, err
	}
	return t, nil
}

// opensProject keeps a trail on a project its reader can no longer open out of every read; the row stays, so access
// given back brings it back.
func (r *Runner) opensProject(ctx context.Context, t *Trail) bool {
	return t.ProjectID == "" || r.perm.HasPermission(ctx, actorID(ctx), t.WorkspaceID, permissions.Member, resourceTypeProject, t.ProjectID)
}

// ListTrails returns a target's trails newest first, under the same gate as GetTrail.
func (r *Runner) ListTrails(ctx context.Context, targetType TargetType, targetID string) ([]*Trail, error) {
	targetID = strings.TrimSpace(targetID)
	if !targetType.valid() || targetID == "" {
		return nil, fmt.Errorf("%w: target type (ticket, doc, or interview) and target id are required", apperrs.ErrInvalid)
	}
	list, err := r.trails.ListTrailsByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("list trails for %s %s: %w", targetType, targetID, err)
	}
	list = slices.DeleteFunc(list, func(t *Trail) bool { return !r.opensProject(ctx, t) })
	if len(list) == 0 {
		return []*Trail{}, nil
	}
	if err := r.requireTrailAccess(ctx, list[0].WorkspaceID, targetType, targetID); err != nil {
		return nil, err
	}
	return list, nil
}

// ActiveTrails maps each target with an active trail to that trail; targets the caller cannot read are left out, not refused.
func (r *Runner) ActiveTrails(ctx context.Context, targetType TargetType, targetIDs []string) (map[string]*Trail, error) {
	if !targetType.valid() {
		return nil, fmt.Errorf("%w: target type (ticket, doc, or interview) is required", apperrs.ErrInvalid)
	}
	ids := make([]string, 0, len(targetIDs))
	for _, id := range targetIDs {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	out := map[string]*Trail{}
	if len(ids) == 0 {
		return out, nil
	}
	list, err := r.trails.ListActiveTrailsByTargets(ctx, targetType, ids)
	if err != nil {
		return nil, fmt.Errorf("list active trails: %w", err)
	}
	actor := actorID(ctx)
	readable := map[string]bool{}
	for _, t := range list {
		ok, seen := readable[t.WorkspaceID]
		if !seen {
			ok = r.perm.HasPermission(ctx, actor, t.WorkspaceID, permissions.PlaysRead, "", "")
			readable[t.WorkspaceID] = ok
		}
		if ok && r.opensProject(ctx, t) {
			out[t.TargetID] = t
		}
	}
	return out, nil
}

// LatestChoices returns what starterID last picked for playID in projectID; empty choices when never run.
func (r *Runner) LatestChoices(ctx context.Context, starterID, playID, projectID string) (*Choices, error) {
	starterID, playID, projectID = strings.TrimSpace(starterID), strings.TrimSpace(playID), strings.TrimSpace(projectID)
	if starterID == "" || playID == "" || projectID == "" {
		return nil, fmt.Errorf("%w: starter, play id, and project id are required", apperrs.ErrInvalid)
	}
	play, err := r.plays.Get(ctx, playID)
	if err != nil {
		return nil, fmt.Errorf("get play %s: %w", playID, err)
	}
	if !r.perm.HasPermission(ctx, actorID(ctx), play.WorkspaceID, permissions.PlaysRead, "", "") {
		return nil, fmt.Errorf("%w: %s required", apperrs.ErrForbidden, permissions.PlaysRead)
	}
	t, err := r.trails.LatestTrailForChoices(ctx, starterID, playID, projectID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return &Choices{MemoryIDs: []string{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest trail for play %s: %w", playID, err)
	}
	return &Choices{MemoryIDs: t.SelectedMemoryIDs, ComputerID: t.ComputerID, Provider: t.Provider, Model: t.Model, ModelOptions: t.ModelOptions}, nil
}

func (r *Runner) requireTrailAccess(ctx context.Context, workspaceID string, targetType TargetType, targetID string) error {
	actor := actorID(ctx)
	if !r.perm.HasPermission(ctx, actor, workspaceID, permissions.PlaysRead, "", "") {
		return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, permissions.PlaysRead)
	}
	if targetType == TargetDoc && !r.perm.HasPermission(ctx, actor, workspaceID, permissions.DocsThread, resourceTypeDoc, targetID) {
		return fmt.Errorf("%w: %s required on doc %s", apperrs.ErrForbidden, permissions.DocsThread, targetID)
	}
	return nil
}

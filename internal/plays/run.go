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

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/redact"
)

// Resource types the access overwrite table keys per-resource permissions on; they mirror access's own.
const (
	resourceTypePlay = "play"
	resourceTypeDoc  = "doc"
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

// PlayActor is the provenance a play's ticket move carries: the play's label, the trail, and how the run was started.
type PlayActor struct {
	PlayLabel string
	TrailID   string
	Via       Via
}

// StatusMover is the runner's seam onto tickets' status setter (ADR 0017).
type StatusMover interface {
	MoveTicket(ctx context.Context, ticketID, statusID string, actor PlayActor) error
}

// ProjectLookup resolves a project's workspace, so a play is checked against the target's own workspace.
type ProjectLookup interface {
	WorkspaceForProject(ctx context.Context, projectID string) (string, error)
}

// HarnessChoice is the computer, provider, and model a run uses: either the caller's pick from the run
// dialog, or what the resolution picked from their project link or pairing defaults when they picked none.
type HarnessChoice struct {
	ComputerID string
	Provider   string
	Model      string
}

// HarnessResolver is the runner's seam onto pairing: the readiness check before a run is started, given the
// caller's pick (a zero HarnessChoice means none), and what it actually resolved to for the trail to record.
type HarnessResolver interface {
	ResolveTarget(ctx context.Context, userID, projectID string, choice HarnessChoice) (HarnessChoice, error)
}

// Memory is the slice of a memory the runner inlines, its body already exported to markdown.
type Memory struct {
	ID             string
	Title          string
	Markdown       string
	AlwaysIncluded bool
}

// MemoryReader is the runner's seam onto memories (ADR 0017).
type MemoryReader interface {
	ListForProject(ctx context.Context, projectID string) ([]Memory, error)
}

// Threads is the runner's seam onto chat: the target's thread, the starter's request in it, and the run's notes.
type Threads interface {
	GetOrCreateTicketThread(ctx context.Context, workspaceID, ticketID, userID string) (string, error)
	GetOrCreateDocThread(ctx context.Context, workspaceID, docID, userID string) (string, error)
	PostMessage(ctx context.Context, conversationID, authorID, body string) (string, error)
	PostSystemNote(ctx context.Context, conversationID, viaUserID, body string) error
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

// UserReader resolves the starter's login for the custom-instructions block.
type UserReader interface {
	Login(ctx context.Context, userID string) (string, error)
}

// RunnerConfig wires the run pipeline.
type RunnerConfig struct {
	Plays    Repo
	Trails   TrailRepo
	Perm     PermissionGate
	Targets  TargetReader
	Projects ProjectLookup
	Harness  HarnessResolver
	Memories MemoryReader
	Threads  Threads
	Turns    TurnRunner
	Tickets  StatusMover
	Live     LivePublisher
	Users    UserReader
	// Links is optional; nil means a ticket play runs without its found-in and blocked-by context.
	Links LinkReader
	// Attachments is optional; nil means images embedded in an inlined memory are left as markdown, unresolved.
	Attachments agent.AttachmentReader
	Logger      *slog.Logger
	Now         func() time.Time
	// SilenceTimeout defaults to HarnessSilenceTimeout; tests shorten it.
	SilenceTimeout time.Duration
}

// Runner starts play runs and keeps their trails (ADR 0055).
type Runner struct {
	plays       Repo
	trails      TrailRepo
	perm        PermissionGate
	targets     TargetReader
	projects    ProjectLookup
	harness     HarnessResolver
	memories    MemoryReader
	threads     Threads
	turns       TurnRunner
	tickets     StatusMover
	live        LivePublisher
	users       UserReader
	links       LinkReader
	attachments agent.AttachmentReader
	log         *slog.Logger
	now         func() time.Time
	silence     time.Duration

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
		plays: cfg.Plays, trails: redactedTrails{cfg.Trails}, perm: cfg.Perm, targets: cfg.Targets, projects: cfg.Projects,
		harness: cfg.Harness, memories: cfg.Memories, threads: redactedThreads{cfg.Threads}, turns: cfg.Turns, tickets: cfg.Tickets,
		live: cfg.Live, users: cfg.Users, links: cfg.Links, attachments: cfg.Attachments, log: cfg.Logger, now: cfg.Now, silence: cfg.SilenceTimeout,
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
	MoveToStatusID     string
	ComputerID         string
	Provider           string
	Model              string
	Via                Via
}

// Choices is what the run dialog pre-selects from the starter's latest trail of a play in a project.
type Choices struct {
	MemoryIDs      []string `json:"memory_ids"`
	MoveToStatusID string   `json:"move_to_status_id"`
	ComputerID     string   `json:"computer_id"`
	Provider       string   `json:"provider"`
	Model          string   `json:"model"`
}

// target is what the runner read about a ticket or doc at press time.
type target struct {
	projectID string
	stage     Stage
	title     string
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
		CustomInstructions: strings.TrimSpace(in.CustomInstructions), MoveToStatusID: strings.TrimSpace(in.MoveToStatusID),
		State: TrailStarting, StartedAt: r.now().UTC(), Activity: []ActivityEntry{},
	}
	choice, err := r.harness.ResolveTarget(ctx, starter, tgt.projectID, HarnessChoice{ComputerID: in.ComputerID, Provider: in.Provider, Model: in.Model})
	if err != nil {
		r.createFailed(ctx, trail, tgt.title, err.Error())
		return nil, err
	}
	trail.ComputerID, trail.Provider, trail.Model = choice.ComputerID, choice.Provider, choice.Model
	if err := r.refuseIfActive(ctx, trail.TargetType, trail.TargetID); err != nil {
		return nil, err
	}
	memoriesBlock, memoryIDs, memoryAttachments, err := r.inlineMemories(ctx, tgt.projectID, trail.SelectedMemoryIDs)
	if err != nil {
		return nil, err
	}
	trail.SelectedMemoryIDs = memoryIDs
	links, err := r.linkBlocks(ctx, trail.TargetType, trail.TargetID)
	if err != nil {
		return nil, err
	}
	conversationID, err := r.openThread(ctx, play.WorkspaceID, trail.TargetType, trail.TargetID, starter)
	if err != nil {
		return nil, err
	}
	trail.ConversationID = conversationID
	if err := r.trails.CreateTrail(ctx, trail); err != nil {
		return nil, fmt.Errorf("create trail for play %s: %w", play.ID, err)
	}
	body := startedMessage(play.Label, trail.CustomInstructions)
	if _, err := r.threads.PostMessage(ctx, conversationID, starter, body); err != nil {
		reason := "post started message: " + err.Error()
		r.finish(ctx, trail, tgt.title, harness.TurnResult{State: harness.TurnError, LastError: reason}, "", "Run failed: "+reason)
		return nil, fmt.Errorf("post started message: %w", err)
	}
	// Copied before the turn starts: from here on the observer's goroutine owns trail.
	snapshot := *trail
	r.startTurn(ctx, trail, tgt.title, agent.TurnRequest{
		ConversationID: conversationID, ViaUserID: starter, RequestBody: body, Attachments: memoryAttachments,
		ExtraRequestBlocks: requestBlocks(play, links, memoriesBlock, r.login(ctx, starter), trail.CustomInstructions),
		Target:             &agent.TargetOverride{ComputerID: choice.ComputerID, Provider: choice.Provider, Model: choice.Model},
	}, false)
	return &snapshot, nil
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
			return r.trails.GetTrail(ctx, trailID)
		}
		if !errors.Is(err, errTurnGone) {
			return nil, err
		}
	}
	trail.Question.Answer = &answer
	trail.State = TrailRunning
	r.save(ctx, trail)
	tgt, err := r.readTarget(ctx, trail.TargetType, trail.TargetID)
	if err != nil {
		r.log.Warn("plays: answer could not read the target", "trail", trailID, "error", err)
	}
	snapshot := *trail
	r.startTurn(ctx, trail, tgt.title, agent.TurnRequest{
		ConversationID: trail.ConversationID, ViaUserID: trail.StarterID, RequestBody: body,
		Target: &agent.TargetOverride{ComputerID: trail.ComputerID, Provider: trail.Provider, Model: trail.Model},
	}, true)
	return &snapshot, nil
}

// checkPlayAndTarget is the run's first gate: the play, its type against the target, the workspace, enabled,
// exclusion, the caller's plays:run on this play, and a ticket play's stage.
func (r *Runner) checkPlayAndTarget(ctx context.Context, starter string, in RunInput) (*Play, target, error) {
	if !in.TargetType.valid() {
		return nil, target{}, fmt.Errorf("%w: target type must be ticket or doc", apperrs.ErrInvalid)
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
		return nil, target{}, fmt.Errorf("%w: a %s play needs a %s target", apperrs.ErrInvalid, play.Type, play.Type)
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

// inlineMemories renders the caller's selection, refusing an id outside the project and a selection over the
// ceilings; always-included memories count toward the ceiling first and lead the returned ids, but the turn
// pipeline inlines their bodies itself, so the play's block carries only the picked ones. Each picked memory's
// markdown goes through the attachment helper on one shared budget, so an embedded image travels to the harness
// alongside text that names it instead of its URL.
func (r *Runner) inlineMemories(ctx context.Context, projectID string, selected []string) (string, []string, []harness.Attachment, error) {
	all, err := r.memories.ListForProject(ctx, projectID)
	if err != nil {
		return "", nil, nil, fmt.Errorf("list memories for project %s: %w", projectID, err)
	}
	inProject := make(map[string]bool, len(all))
	for _, m := range all {
		inProject[m.ID] = true
	}
	wanted := make(map[string]bool, len(selected))
	for _, id := range selected {
		if !inProject[id] {
			return "", nil, nil, fmt.Errorf("%w: memory %s is not in this project", apperrs.ErrInvalid, id)
		}
		wanted[id] = true
	}
	budget := agent.NewAttachmentBudget()
	var counted, picked []agent.InlinedMemory
	var attachments []harness.Attachment
	inlined := []string{}
	for _, m := range all {
		if m.AlwaysIncluded {
			counted = append(counted, agent.InlinedMemory{Title: m.Title, Body: m.Markdown})
			inlined = append(inlined, m.ID)
		}
	}
	for _, m := range all {
		if wanted[m.ID] && !m.AlwaysIncluded {
			body, atts := agent.ExtractAttachments(ctx, m.Markdown, r.attachments, budget)
			item := agent.InlinedMemory{Title: m.Title, Body: body}
			counted = append(counted, item)
			picked = append(picked, item)
			attachments = append(attachments, atts...)
			inlined = append(inlined, m.ID)
		}
	}
	if _, _, err := agent.InlineMemories(counted, agent.DefaultInlineLimits()); err != nil {
		return "", nil, nil, err
	}
	if len(picked) == 0 {
		return "", inlined, nil, nil
	}
	block, _, err := agent.InlineMemories(picked, agent.DefaultInlineLimits())
	if err != nil {
		return "", nil, nil, err
	}
	return block, inlined, attachments, nil
}

func (r *Runner) openThread(ctx context.Context, workspaceID string, targetType TargetType, targetID, starter string) (string, error) {
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
// ten-minute cap is chat's, not a play's. A resumed turn continues an existing trail, so it announces no start.
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
	if err != nil {
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
	if trail.State != TrailDone {
		return
	}
	if skipped := r.moveOnDone(ctx, trail); skipped != "" {
		r.note(ctx, trail, skipped)
		r.save(ctx, trail)
	}
}

// moveOnDone applies the chosen column, never backwards (ADR 0055); it returns the note for a skipped or failed
// move, empty when the ticket moved or there was nothing to move.
func (r *Runner) moveOnDone(ctx context.Context, trail *Trail) string {
	if trail.TargetType != TargetTicket || trail.MoveToStatusID == "" || r.tickets == nil {
		return ""
	}
	column, err := r.targets.GetStatus(ctx, trail.MoveToStatusID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "Chosen column no longer exists; ticket left where it is"
	}
	if err != nil {
		r.log.Error("plays: read move-to column failed", "trail", trail.ID, "status", trail.MoveToStatusID, "error", err)
		return ""
	}
	ticket, err := r.targets.GetTicket(ctx, trail.TargetID)
	if err != nil {
		r.log.Error("plays: read ticket for move-to failed", "trail", trail.ID, "ticket", trail.TargetID, "error", err)
		return ""
	}
	if slices.Index(stages, column.Stage) < slices.Index(stages, ticket.Stage) {
		return fmt.Sprintf("Ticket is already in %s; not moving it back to %s", ticket.Stage, column.Name)
	}
	actor := PlayActor{PlayLabel: trail.PlayLabel, TrailID: trail.ID, Via: trail.Via}
	if err := r.tickets.MoveTicket(ctx, trail.TargetID, trail.MoveToStatusID, actor); err != nil {
		r.log.Error("plays: move-to failed", "trail", trail.ID, "status", trail.MoveToStatusID, "error", err)
		return fmt.Sprintf("Could not move the ticket to %s: %v", column.Name, err)
	}
	return ""
}

func (r *Runner) createFailed(ctx context.Context, trail *Trail, targetTitle, reason string) {
	now := r.now().UTC()
	trail.State, trail.EndedAt, trail.LastError = TrailFailed, &now, reason
	if err := r.trails.CreateTrail(ctx, trail, r.finishedEvent(trail, targetTitle)); err != nil {
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
	if err := r.threads.PostSystemNote(ctx, trail.ConversationID, trail.StarterID, body); err != nil {
		r.log.Error("plays: post system note failed", "trail", trail.ID, "error", err)
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

// requestBlocks is the play material that rides inside the request block: play, memories, custom, in that order.
func requestBlocks(play *Play, links []string, memoriesBlock, login, custom string) []string {
	blocks := append([]string{"Play: " + play.Label + "\n" + play.Instructions}, links...)
	if memoriesBlock != "" {
		blocks = append(blocks, "Memories the user selected for this run, follow them:\n"+memoriesBlock)
	}
	if custom != "" {
		blocks = append(blocks, fmt.Sprintf("Instructions from %s for this run; where these conflict with the play's instructions, these win:\n%s", login, custom))
	}
	return blocks
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
	if err := r.requireTrailAccess(ctx, t.WorkspaceID, t.TargetType, t.TargetID); err != nil {
		return nil, err
	}
	return t, nil
}

// ListTrails returns a target's trails newest first, under the same gate as GetTrail.
func (r *Runner) ListTrails(ctx context.Context, targetType TargetType, targetID string) ([]*Trail, error) {
	targetID = strings.TrimSpace(targetID)
	if !targetType.valid() || targetID == "" {
		return nil, fmt.Errorf("%w: target type (ticket or doc) and target id are required", apperrs.ErrInvalid)
	}
	list, err := r.trails.ListTrailsByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("list trails for %s %s: %w", targetType, targetID, err)
	}
	if len(list) == 0 {
		return []*Trail{}, nil
	}
	if err := r.requireTrailAccess(ctx, list[0].WorkspaceID, targetType, targetID); err != nil {
		return nil, err
	}
	return list, nil
}

// ActiveTrails maps each target with an active trail to its id; targets the caller cannot read are left out, not refused.
func (r *Runner) ActiveTrails(ctx context.Context, targetType TargetType, targetIDs []string) (map[string]string, error) {
	if !targetType.valid() {
		return nil, fmt.Errorf("%w: target type (ticket or doc) is required", apperrs.ErrInvalid)
	}
	ids := make([]string, 0, len(targetIDs))
	for _, id := range targetIDs {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	out := map[string]string{}
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
		if ok {
			out[t.TargetID] = t.ID
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
	return &Choices{MemoryIDs: t.SelectedMemoryIDs, MoveToStatusID: t.MoveToStatusID, ComputerID: t.ComputerID, Provider: t.Provider, Model: t.Model}, nil
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

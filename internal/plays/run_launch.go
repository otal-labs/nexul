package plays

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

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

// interviewBlocks names an interview run's project, then its drafting context or question sources; an audit adds date and target.
func (r *Runner) interviewBlocks(ctx context.Context, play *Play, trail *Trail, tgt target) []string {
	if trail.TargetType != TargetInterview {
		return nil
	}
	blocks := []string{interviewBlock(tgt)}
	if play.BuiltinKey == DraftInterviewKey {
		return append(blocks, r.draftingBlock(ctx, trail))
	}
	auditing := play.BuiltinKey == AuditKey
	if auditing {
		blocks = append(blocks, fmt.Sprintf("Today is %s. This run's trail id is %s.", r.now().UTC().Format(time.DateOnly), trail.ID))
	}
	sources := r.projectSourcesBlock(ctx, trail, StanceQuestion)
	if sources != "" {
		return append(blocks, sources)
	}
	if auditing {
		blocks = append(blocks, "This project has no project source under question: audit its own code, in the checkout you are running in.")
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

// Continue sends message to an ended run's own harness thread, as the next turn of the same trail, instead of pressing
// the play again (ADR 0128). The starter or a plays:write holder may; it waits for the harness to take the turn, and when
// the thread is gone it starts the play again as a new run carrying message, and returns that trail instead.
func (r *Runner) Continue(ctx context.Context, trailID, message string, via Via) (*Trail, error) {
	trailID, message = strings.TrimSpace(trailID), strings.TrimSpace(message)
	if trailID == "" {
		return nil, fmt.Errorf("%w: trail id is required", apperrs.ErrInvalid)
	}
	if message == "" {
		return nil, fmt.Errorf("%w: a message is required", apperrs.ErrInvalid)
	}
	trail, err := r.trails.GetTrail(ctx, trailID)
	if err != nil {
		return nil, fmt.Errorf("get trail %s: %w", trailID, err)
	}
	actor := actorID(ctx)
	if actor != trail.StarterID && !r.perm.HasPermission(ctx, actor, trail.WorkspaceID, permissions.PlaysWrite, "", "") {
		return nil, fmt.Errorf("%w: only the starter or a %s holder can continue a run", apperrs.ErrForbidden, permissions.PlaysWrite)
	}
	if trail.State.Active() {
		return nil, fmt.Errorf("%w: the run is still going; answer or stop it instead", apperrs.ErrConflict)
	}
	if err := r.refuseIfActive(ctx, trail.TargetType, trail.TargetID); err != nil {
		return nil, err
	}
	tgt, err := r.readTarget(ctx, trail.TargetType, trail.TargetID)
	if err != nil {
		r.log.Warn("plays: continue could not read the target", "trail", trailID, "error", err)
	}
	if _, err := r.threads.PostMessage(ctx, trail.ConversationID, actor, message); err != nil {
		return nil, fmt.Errorf("post message: %w", err)
	}
	before := *trail
	trail.AppendActivity(ActivityEntry{Kind: harness.ActivityUserMessage, Summary: harness.Preview(message, 160), Detail: harness.CapDetail(message), At: r.now().UTC()})
	trail.State, trail.EndedAt, trail.LastError = TrailRunning, nil, ""
	r.save(ctx, trail)
	snapshot := *trail
	o := &trailObserver{trail: trail, targetTitle: tgt.title, resumed: true, opened: make(chan bool, 1), before: &before}
	r.start(ctx, o, agent.TurnRequest{
		ConversationID: trail.ConversationID, ViaUserID: trail.StarterID, RequestBody: message, KeepThread: true,
		Target: &agent.TargetOverride{ComputerID: trail.ComputerID, Provider: trail.Provider, Model: trail.Model, ModelOptions: trail.ModelOptions},
	})
	wait := time.NewTimer(continueWait)
	defer wait.Stop()
	select {
	case gone := <-o.opened:
		if gone {
			return r.runAgain(ctx, &before, message, via)
		}
	case <-wait.C:
	case <-ctx.Done():
	}
	return &snapshot, nil
}

// continueWait bounds how long Continue waits to learn whether the harness took the turn; past it the trail just runs.
const continueWait = 30 * time.Second

// runAgain starts a gone thread's play again on the same target with the same choices, its instructions ending with message.
func (r *Runner) runAgain(ctx context.Context, old *Trail, message string, via Via) (*Trail, error) {
	custom := message
	if old.CustomInstructions != "" {
		custom = old.CustomInstructions + "\n\n" + message
	}
	return r.Run(ctx, RunInput{
		PlayID: old.PlayID, TargetType: old.TargetType, TargetID: old.TargetID, MemoryIDs: old.SelectedMemoryIDs, CustomInstructions: custom,
		ComputerID: old.ComputerID, Provider: old.Provider, Model: old.Model, ModelOptions: old.ModelOptions, Via: via,
	})
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
// The channel closes once the turn has ended.
func (r *Runner) startTurn(ctx context.Context, trail *Trail, targetTitle string, req agent.TurnRequest, resumed bool) <-chan struct{} {
	return r.start(ctx, &trailObserver{trail: trail, targetTitle: targetTitle, resumed: resumed}, req)
}

// start runs o's turn; o names the trail and how its start is recorded, start wires the rest.
func (r *Runner) start(ctx context.Context, o *trailObserver, req agent.TurnRequest) <-chan struct{} {
	trail := o.trail
	runCtx, cancel := context.WithCancelCause(context.WithoutCancel(ctx))
	o.r, o.ctx, o.cancel = r, context.WithoutCancel(runCtx), cancel
	o.timer = time.AfterFunc(r.silence, o.onSilence)
	r.setRun(trail.ID, o)
	req.Observer = o
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer r.clearRun(trail.ID, o)
		defer cancel(nil)
		r.turns.RunTurn(runCtx, req)
	}()
	return done
}

// followedAgainNote is the trail's and the thread's line for a run reopened by news on its thread (ADR 0127).
const followedAgainNote = "New activity on this run's thread in T3 Code; following it again."

// FollowThread implements agent.ThreadFollower: news on the thread of a conversation whose newest run ran there,
// on userID's computerID, and has ended or is parked on a question reopens that run's trail and catches it up from since (ADR 0127).
func (r *Runner) FollowThread(ctx context.Context, conversationID, threadID, userID, computerID, since string) (<-chan struct{}, bool) {
	trail, err := r.trails.LatestTrailInConversation(ctx, conversationID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, false
	}
	if err != nil {
		r.log.Warn("plays: read the run to follow again failed", "conversation", conversationID, "error", err)
		return nil, false
	}
	// A trail waiting on a question whose turn closed is parked, not running; its answer may have come in T3.
	parked := trail.State == TrailWaiting && r.run(trail.ID) == nil
	if (trail.State.Active() && !parked) || trail.HarnessSessionID != threadID || trail.StarterID != userID || trail.ComputerID != computerID {
		return nil, false
	}
	tgt, err := r.readTarget(ctx, trail.TargetType, trail.TargetID)
	if err != nil {
		r.log.Warn("plays: follow again could not read the target", "trail", trail.ID, "error", err)
	}
	trail.State, trail.EndedAt, trail.LastError = TrailRunning, nil, ""
	r.note(ctx, trail, followedAgainNote)
	r.save(ctx, trail)
	return r.startTurn(ctx, trail, tgt.title, agent.TurnRequest{
		ConversationID: conversationID, ViaUserID: trail.StarterID, Watch: true, Since: since,
		Target: &agent.TargetOverride{ComputerID: trail.ComputerID, Provider: trail.Provider, Model: trail.Model, ModelOptions: trail.ModelOptions},
	}, true), true
}

// trailObserver records the turn's lifecycle on the trail; pipeline, silence timer, and Stop race for it, first terminal wins.

package plays

import (
	"cmp"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// QueueRig is a Runner over real stores with every other seam faked, for the queue's tests over real SQLite in plays_test.
type QueueRig struct {
	Runner   *Runner
	harness  *switchableHarness
	perm     *fakePerm
	facts    *fakeFacts
	targets  *fakeTargets
	shares   *fakeComputers
	turns    *idleTurns
	memories *fakeMemories
}

// NewQueueRig wires a runner whose plays, trails, and queue are the given stores; projectID is in workspaceID.
func NewQueueRig(playsRepo Repo, trails TrailRepo, queue QueueRepo, workspaceID, projectID string, now func() time.Time) *QueueRig {
	q := &QueueRig{
		harness:  &switchableHarness{},
		perm:     &fakePerm{grants: map[string][]permissions.Action{}, denied: map[string]map[string]bool{}, projects: []string{projectID}},
		facts:    &fakeFacts{byID: map[string]Facts{}},
		targets:  &fakeTargets{tickets: map[string]TicketTarget{}, docs: map[string]DocTarget{}, statuses: map[string]StatusTarget{}, stacks: map[string]StackTarget{}},
		turns:    &idleTurns{started: make(chan struct{})},
		shares:   &fakeComputers{owners: map[string]string{}, held: map[string]permissions.Set{}},
		memories: &fakeMemories{byProject: map[string][]Memory{}},
	}
	q.Runner = NewRunner(RunnerConfig{
		Plays: playsRepo, Trails: trails, Queue: queue, Facts: q.facts, Perm: q.perm, Targets: q.targets,
		Projects: &fakeProjects{workspaces: map[string]string{projectID: workspaceID}},
		Harness:  q.harness, Checkouts: q.harness, Memories: q.memories, Threads: &fakeThreads{}, Turns: q.turns, Users: fakeUsers{}, Computers: q.shares, Now: now,
	})
	return q
}

// SetMemories gives a project the memories with these ids, none always included.
func (q *QueueRig) SetMemories(projectID string, ids ...string) {
	for _, id := range ids {
		q.memories.byProject[projectID] = append(q.memories.byProject[projectID], Memory{ID: id, Title: id})
	}
}

// SetTicket gives a ticket its facts now, and the stage and key a run reads.
func (q *QueueRig) SetTicket(id string, f Facts) {
	q.facts.set(id, f)
	q.targets.mu.Lock()
	defer q.targets.mu.Unlock()
	q.targets.tickets[id] = TicketTarget{ProjectID: f.ProjectID, Key: id, Stage: f.Stage}
}

// SetStack gives a stack its facts now, and the project and name a run reads.
func (q *QueueRig) SetStack(id, projectID, name string) {
	q.facts.set(id, Facts{ProjectID: projectID, StackID: id})
	q.targets.mu.Lock()
	defer q.targets.mu.Unlock()
	q.targets.stacks[id] = StackTarget{ProjectID: projectID, Name: name}
}

// OwnComputer makes ownerID the owner of computerID.
func (q *QueueRig) OwnComputer(computerID, ownerID string) { q.shares.owners[computerID] = ownerID }

// FailComputerCheck makes every check of what a person holds on a computer fail with err; nil answers again.
func (q *QueueRig) FailComputerCheck(err error) { q.shares.err = err }

// ShareComputer gives userID actions on computerID through its owner's rule; none takes the share away.
func (q *QueueRig) ShareComputer(computerID, userID string, actions ...permissions.Action) {
	q.shares.held[computerID+"/"+userID] = permissions.SetOf(actions...)
}

// LastPlayBlocks waits for the first started run, which a detached goroutine hands its request to, then returns the
// context blocks the last one was handed, one per line; nil when ctx ends first.
func (q *QueueRig) LastPlayBlocks(ctx context.Context) []string {
	select {
	case <-q.turns.started:
	case <-ctx.Done():
		return nil
	}
	q.turns.mu.Lock()
	defer q.turns.mu.Unlock()
	return q.turns.last.Play.Blocks
}

// LastChoice is the computer and T3 project the last start was asked to run on.
func (q *QueueRig) LastChoice() HarnessChoice { return q.harness.lastChoice() }

// SetStatus names a board column's stage, for moments read off a move.
func (q *QueueRig) SetStatus(id string, stage Stage) {
	q.targets.statuses[id] = StatusTarget{Name: id, Stage: stage}
}

// Grant lets userID run every play and read every project.
func (q *QueueRig) Grant(userID string) {
	q.perm.grants[userID] = []permissions.Action{permissions.PlaysRead, permissions.PlaysRun}
}

// GrantWrite adds autoplays:write to userID's grants.
func (q *QueueRig) GrantWrite(userID string) {
	q.perm.grants[userID] = append(q.perm.grants[userID], permissions.AutoplaysWrite)
}

// GrantRead adds autoplays:read to userID's grants.
func (q *QueueRig) GrantRead(userID string) {
	q.perm.grants[userID] = append(q.perm.grants[userID], permissions.AutoplaysRead)
}

// HideProject makes projectID one userID may not open.
func (q *QueueRig) HideProject(userID, projectID string) { q.perm.hideProject(userID, projectID) }

// Exclude denies userID plays:run on playID, as a play's own settings exclude a person.
func (q *QueueRig) Exclude(userID, playID string) { q.perm.deny(userID, playID) }

// Hide is a play rule denying userID See on playID.
func (q *QueueRig) Hide(userID, playID string) { q.perm.hide(userID, playID) }

// Offline makes every computer refuse as offline, or answer again.
func (q *QueueRig) Offline(offline bool) {
	reason := ""
	if offline {
		reason = ReasonOffline
	}
	q.harness.refuse(reason)
}

// CountRunningWith makes each person's computer take runsAtOnce queued runs, counting what goes on it with running.
func (q *QueueRig) CountRunningWith(runsAtOnce int, running func(ctx context.Context, computerID string) (int, error)) {
	q.harness.mu.Lock()
	defer q.harness.mu.Unlock()
	q.harness.runsAtOnce, q.harness.running = runsAtOnce, running
}

// Refuse makes every computer refuse with reason, such as setup_required; "" answers again.
func (q *QueueRig) Refuse(reason string) { q.harness.refuse(reason) }

// Resolves counts the harness readiness checks, one per start tried.
func (q *QueueRig) Resolves() int { return q.harness.count() }

// Dispatch makes one pass of the queue and returns how long the dispatcher would sleep.
func (q *QueueRig) Dispatch(ctx context.Context) (time.Duration, error) {
	return q.Runner.dispatch(ctx)
}

// OnNextFacts runs fn once, at the next read of a target's facts: during a pass, after the queue was read and before
// the dispatcher claims an item.
func (q *QueueRig) OnNextFacts(fn func()) {
	q.facts.mu.Lock()
	defer q.facts.mu.Unlock()
	q.facts.next = fn
}

// OnNextResolve runs fn once, when a start next asks where to run: after the dispatcher claimed its item.
func (q *QueueRig) OnNextResolve(fn func()) {
	q.harness.mu.Lock()
	defer q.harness.mu.Unlock()
	q.harness.next = fn
}

// Kicked drains the dispatcher's kick, reporting whether one was waiting.
func (q *QueueRig) Kicked() bool {
	select {
	case <-q.Runner.kick:
		return true
	default:
		return false
	}
}

type fakeFacts struct {
	mu   sync.Mutex
	byID map[string]Facts
	next func()
}

func (f *fakeFacts) set(id string, facts Facts) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[id] = facts
}

func (f *fakeFacts) Facts(_ context.Context, _ TargetType, id string) (Facts, error) {
	f.mu.Lock()
	next := f.next
	f.next = nil
	f.mu.Unlock()
	if next != nil {
		next()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	facts, ok := f.byID[id]
	if !ok {
		return Facts{}, apperrs.ErrNotFound
	}
	return facts, nil
}

// switchableHarness answers every start, or refuses each with a reason. Each person has one computer, c-<person>, that
// takes as many queued runs at once as runsAtOnce, counted by running, the way pairing counts a computer's room.
type switchableHarness struct {
	mu         sync.Mutex
	reason     string
	resolves   int
	runsAtOnce int
	running    func(ctx context.Context, computerID string) (int, error)
	next       func()
	last       HarnessChoice
}

func (h *switchableHarness) lastChoice() HarnessChoice {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.last
}

// takeNext returns the hook OnNextResolve set, once.
func (h *switchableHarness) takeNext() func() {
	h.mu.Lock()
	defer h.mu.Unlock()
	next := h.next
	h.next = nil
	if next == nil {
		return func() {}
	}
	return next
}

func (h *switchableHarness) refuse(reason string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reason = reason
}

func (h *switchableHarness) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.resolves
}

func (h *switchableHarness) ResolveTarget(_ context.Context, userID, _ string, choice HarnessChoice) (HarnessChoice, error) {
	h.takeNext()()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.resolves++
	h.last = choice
	computerID := cmp.Or(choice.ComputerID, "c-"+userID)
	if h.reason != "" {
		pass := PassedOver{ComputerID: computerID, Name: computerID, Why: h.reason, Text: computerID + " can't: " + h.reason}
		return HarnessChoice{}, &HarnessRefusal{Reason: h.reason, ComputerID: computerID, PassedOver: []PassedOver{pass}, Err: apperrs.ErrInvalid}
	}
	return HarnessChoice{ComputerID: computerID, Provider: "codex"}, nil
}

// ResolveQueuedTarget refuses while the person's computer is full, naming it, as pairing's no_computer_ready does.
func (h *switchableHarness) ResolveQueuedTarget(ctx context.Context, userID, projectID string, choice HarnessChoice) (HarnessChoice, error) {
	computerID := cmp.Or(choice.ComputerID, "c-"+userID)
	h.mu.Lock()
	limit, running := max(h.runsAtOnce, 1), h.running
	h.mu.Unlock()
	if running == nil {
		return h.ResolveTarget(ctx, userID, projectID, choice)
	}
	n, err := running(ctx, computerID)
	if err != nil {
		return HarnessChoice{}, err
	}
	if n < limit {
		return h.ResolveTarget(ctx, userID, projectID, choice)
	}
	text := fmt.Sprintf("%s is full (%d of %d running)", computerID, n, limit)
	pass := PassedOver{ComputerID: computerID, Name: computerID, Why: "full", Text: text}
	return HarnessChoice{}, &HarnessRefusal{Reason: "no_computer_ready", PassedOver: []PassedOver{pass}, Err: apperrs.ErrInvalid}
}

func (h *switchableHarness) ResolvePersonTarget(ctx context.Context, userID, projectID string, choice HarnessChoice) (HarnessChoice, error) {
	return h.ResolveTarget(ctx, userID, projectID, choice)
}

// ResolveRecordedTarget runs a recorded computer and T3 project as given.
func (h *switchableHarness) ResolveRecordedTarget(_ context.Context, _, _ string, choice HarnessChoice) (HarnessChoice, error) {
	h.takeNext()()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.resolves++
	if h.reason != "" {
		return HarnessChoice{}, &HarnessRefusal{Reason: h.reason, ComputerID: choice.ComputerID, Err: apperrs.ErrInvalid}
	}
	return choice, nil
}

func (h *switchableHarness) CheckoutOn(context.Context, string, string, string) (string, error) {
	return "", nil
}

func (h *switchableHarness) ListProjects(context.Context, string, string) ([]harness.Project, error) {
	return nil, nil
}

// idleTurns accepts a turn and never reports on it, so its trail stays starting and holds its slot until a test ends it.
// It keeps the last request it was given.
type idleTurns struct {
	mu      sync.Mutex
	last    agent.TurnRequest
	started chan struct{}
	once    sync.Once
}

func (t *idleTurns) RunTurn(_ context.Context, req agent.TurnRequest) {
	t.mu.Lock()
	t.last = req
	t.mu.Unlock()
	t.once.Do(func() { close(t.started) })
}

func (*idleTurns) Interrupt(context.Context, string) error { return nil }

func (*idleTurns) Answer(context.Context, string, string, harness.QuestionAnswer) error { return nil }

// fakeComputers answers ComputerGate the way access does: the owner holds every computer action, a person a computer
// is shared with what the rule gives them, anyone else finds no such computer.
type fakeComputers struct {
	owners map[string]string
	held   map[string]permissions.Set
	err    error
}

func (f *fakeComputers) ComputerPermissions(_ context.Context, userID, computerID string) (string, permissions.Set, error) {
	if f.err != nil {
		return "", nil, f.err
	}
	owner, ok := f.owners[computerID]
	if ok && owner == userID {
		return owner, permissions.SetOf(permissions.ComputerActions...), nil
	}
	held, shared := f.held[computerID+"/"+userID]
	if !ok || !shared || !held.Has(permissions.ComputersSee) {
		return "", nil, apperrs.ErrNotFound
	}
	return owner, held, nil
}

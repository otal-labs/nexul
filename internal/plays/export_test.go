package plays

import (
	"context"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// QueueRig is a Runner over real stores with every other seam faked, for the queue's tests over real SQLite in plays_test.
type QueueRig struct {
	Runner  *Runner
	harness *switchableHarness
	perm    *fakePerm
	facts   *fakeFacts
	targets *fakeTargets
}

// NewQueueRig wires a runner whose plays, trails, and queue are the given stores; projectID is in workspaceID.
func NewQueueRig(playsRepo Repo, trails TrailRepo, queue QueueRepo, workspaceID, projectID string, now func() time.Time) *QueueRig {
	q := &QueueRig{
		harness: &switchableHarness{links: map[string][2]string{}},
		perm:    &fakePerm{grants: map[string][]permissions.Action{}, denied: map[string]map[string]bool{}, projects: []string{projectID}},
		facts:   &fakeFacts{byID: map[string]Facts{}},
		targets: &fakeTargets{tickets: map[string]TicketTarget{}, docs: map[string]DocTarget{}, statuses: map[string]StatusTarget{}},
	}
	q.Runner = NewRunner(RunnerConfig{
		Plays: playsRepo, Trails: trails, Queue: queue, Facts: q.facts, Perm: q.perm, Targets: q.targets,
		Projects: &fakeProjects{workspaces: map[string]string{projectID: workspaceID}},
		Harness:  q.harness, Checkouts: q.harness, Memories: &fakeMemories{}, Threads: &fakeThreads{}, Turns: idleTurns{}, Users: fakeUsers{}, Now: now,
	})
	return q
}

// SetTicket gives a ticket its facts now, and the stage and key a run reads.
func (q *QueueRig) SetTicket(id string, f Facts) {
	q.facts.set(id, f)
	q.targets.mu.Lock()
	defer q.targets.mu.Unlock()
	q.targets.tickets[id] = TicketTarget{ProjectID: f.ProjectID, Key: id, Stage: f.Stage}
}

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
		reason = RefusalOffline
	}
	q.harness.refuse(reason)
}

// Refuse makes every computer refuse with reason, such as setup_required; "" answers again.
func (q *QueueRig) Refuse(reason string) { q.harness.refuse(reason) }

// FailLinkSave makes saving a project link fail with err; nil saves again.
func (q *QueueRig) FailLinkSave(err error) {
	q.harness.mu.Lock()
	defer q.harness.mu.Unlock()
	q.harness.saveErr = err
}

// Link is userID's saved link for projectID: its computer and T3 project, empty when unset.
func (q *QueueRig) Link(userID, projectID string) [2]string { return q.harness.Link(userID, projectID) }

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

// switchableHarness answers every start, or refuses each with a reason. It also keeps project links, which answer
// for their person and project without a default computer, as pairing reads a link before the defaults.
type switchableHarness struct {
	mu       sync.Mutex
	reason   string
	resolves int
	links    map[string][2]string // user id + "/" + project id -> computer id, T3 project id
	saveErr  error
	next     func()
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

func (h *switchableHarness) ResolveTarget(_ context.Context, userID, projectID string, _ HarnessChoice) (HarnessChoice, error) {
	h.takeNext()()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.resolves++
	if link, ok := h.links[userID+"/"+projectID]; ok && h.reason == RefusalNoDefaultComputer {
		return HarnessChoice{ComputerID: link[0], HarnessProjectID: link[1], Provider: "codex"}, nil
	}
	if h.reason != "" {
		return HarnessChoice{}, &HarnessRefusal{Reason: h.reason, ComputerID: "c-1", Err: apperrs.ErrInvalid}
	}
	return HarnessChoice{ComputerID: "c-1", Provider: "codex"}, nil
}

func (h *switchableHarness) ResolvePersonTarget(ctx context.Context, userID, projectID string, choice HarnessChoice) (HarnessChoice, error) {
	return h.ResolveTarget(ctx, userID, projectID, choice)
}

// ResolveRecordedTarget runs a picked computer and T3 project as given: a pinned location needs no default computer.
func (h *switchableHarness) ResolveRecordedTarget(_ context.Context, _, _ string, choice HarnessChoice) (HarnessChoice, error) {
	h.takeNext()()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.resolves++
	if h.reason != "" && h.reason != RefusalNoDefaultComputer {
		return HarnessChoice{}, &HarnessRefusal{Reason: h.reason, ComputerID: choice.ComputerID, Err: apperrs.ErrInvalid}
	}
	return choice, nil
}

func (h *switchableHarness) LinkedProject(_ context.Context, userID, projectID string) (string, string, error) {
	link := h.Link(userID, projectID)
	return link[0], link[1], nil
}

func (h *switchableHarness) ListProjects(context.Context, string, string) ([]harness.Project, error) {
	return nil, nil
}

func (h *switchableHarness) SaveLink(_ context.Context, userID, projectID, computerID, harnessProjectID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.saveErr != nil {
		return h.saveErr
	}
	h.links[userID+"/"+projectID] = [2]string{computerID, harnessProjectID}
	return nil
}

func (h *switchableHarness) Link(userID, projectID string) [2]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.links[userID+"/"+projectID]
}

// idleTurns accepts a turn and never reports on it, so its trail stays starting and holds its slot until a test ends it.
type idleTurns struct{}

func (idleTurns) RunTurn(context.Context, agent.TurnRequest) {}

func (idleTurns) Interrupt(context.Context, string) error { return nil }

func (idleTurns) Answer(context.Context, string, string, harness.QuestionAnswer) error { return nil }

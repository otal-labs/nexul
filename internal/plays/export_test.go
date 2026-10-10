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
		harness: &switchableHarness{},
		perm:    &fakePerm{grants: map[string][]permissions.Action{}, denied: map[string]map[string]bool{}, projects: []string{projectID}},
		facts:   &fakeFacts{byID: map[string]Facts{}},
		targets: &fakeTargets{tickets: map[string]TicketTarget{}, docs: map[string]DocTarget{}, statuses: map[string]StatusTarget{}},
	}
	q.Runner = NewRunner(RunnerConfig{
		Plays: playsRepo, Trails: trails, Queue: queue, Facts: q.facts, Perm: q.perm, Targets: q.targets,
		Projects: &fakeProjects{workspaces: map[string]string{projectID: workspaceID}},
		Harness:  q.harness, Memories: &fakeMemories{}, Threads: &fakeThreads{}, Turns: idleTurns{}, Users: fakeUsers{}, Now: now,
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

// Resolves counts the harness readiness checks, one per start tried.
func (q *QueueRig) Resolves() int { return q.harness.count() }

// Dispatch makes one pass of the queue and returns how long the dispatcher would sleep.
func (q *QueueRig) Dispatch(ctx context.Context) (time.Duration, error) {
	return q.Runner.dispatch(ctx)
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
}

func (f *fakeFacts) set(id string, facts Facts) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[id] = facts
}

func (f *fakeFacts) Facts(_ context.Context, _ TargetType, id string) (Facts, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	facts, ok := f.byID[id]
	if !ok {
		return Facts{}, apperrs.ErrNotFound
	}
	return facts, nil
}

// switchableHarness answers every start, or refuses each with a reason.
type switchableHarness struct {
	mu       sync.Mutex
	reason   string
	resolves int
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

func (h *switchableHarness) ResolveTarget(context.Context, string, string, HarnessChoice) (HarnessChoice, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.resolves++
	if h.reason != "" {
		return HarnessChoice{}, &HarnessRefusal{Reason: h.reason, ComputerID: "c-1", Err: apperrs.ErrInvalid}
	}
	return HarnessChoice{ComputerID: "c-1", Provider: "codex"}, nil
}

// idleTurns accepts a turn and never reports on it, so its trail stays starting and holds its slot until a test ends it.
type idleTurns struct{}

func (idleTurns) RunTurn(context.Context, agent.TurnRequest) {}

func (idleTurns) Interrupt(context.Context, string) error { return nil }

func (idleTurns) Answer(context.Context, string, string, harness.QuestionAnswer) error { return nil }

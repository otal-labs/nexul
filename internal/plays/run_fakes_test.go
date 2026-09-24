package plays

import (
	"context"
	"encoding/json"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// fakeTrailRepo is the in-memory TrailRepo; terminal receives every trail saved in a terminal state.
type fakeTrailRepo struct {
	mu                            sync.Mutex
	byID                          map[string]*Trail
	states                        []TrailState
	events                        []eventbus.OutboxEvent
	terminal                      chan Trail
	createErr, listErr, latestErr error
}

func newFakeTrailRepo() *fakeTrailRepo {
	return &fakeTrailRepo{byID: map[string]*Trail{}, terminal: make(chan Trail, 4)}
}

func (f *fakeTrailRepo) CreateTrail(_ context.Context, t *Trail, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	cp := *t
	f.byID[t.ID] = &cp
	f.states = append(f.states, t.State)
	f.events = append(f.events, evts...)
	f.notifyTerminal(cp)
	return nil
}

func (f *fakeTrailRepo) GetTrail(_ context.Context, id string) (*Trail, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.byID[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (f *fakeTrailRepo) UpdateTrail(_ context.Context, t *Trail, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.byID[t.ID]; !ok {
		return apperrs.ErrNotFound
	}
	cp := *t
	cp.Activity = append([]ActivityEntry{}, t.Activity...)
	f.byID[t.ID] = &cp
	f.states = append(f.states, t.State)
	f.events = append(f.events, evts...)
	f.notifyTerminal(cp)
	return nil
}

func (f *fakeTrailRepo) notifyTerminal(t Trail) {
	if !t.State.Active() {
		f.terminal <- t
	}
}

func (f *fakeTrailRepo) ListTrailsByTarget(_ context.Context, targetType TargetType, targetID string) ([]*Trail, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Trail
	for _, t := range f.byID {
		if t.TargetType == targetType && t.TargetID == targetID {
			cp := *t
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out, nil
}

func (f *fakeTrailRepo) ListActiveTrailsByTargets(_ context.Context, targetType TargetType, targetIDs []string) ([]*Trail, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Trail
	for _, t := range f.byID {
		if t.TargetType == targetType && t.State.Active() && slices.Contains(targetIDs, t.TargetID) {
			cp := *t
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeTrailRepo) LatestTrailForChoices(_ context.Context, starterID, playID, projectID string) (*Trail, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.latestErr != nil {
		return nil, f.latestErr
	}
	var latest *Trail
	for _, t := range f.byID {
		if t.StarterID != starterID || t.PlayID != playID || t.ProjectID != projectID {
			continue
		}
		if latest == nil || t.StartedAt.After(latest.StartedAt) {
			latest = t
		}
	}
	if latest == nil {
		return nil, apperrs.ErrNotFound
	}
	cp := *latest
	return &cp, nil
}

func (f *fakeTrailRepo) all() []*Trail {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Trail
	for _, t := range f.byID {
		cp := *t
		out = append(out, &cp)
	}
	return out
}

func (f *fakeTrailRepo) recordedStates() []TrailState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]TrailState{}, f.states...)
}

func (f *fakeTrailRepo) eventsFor(topic string) []eventbus.OutboxEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []eventbus.OutboxEvent
	for _, e := range f.events {
		if e.Topic == topic {
			out = append(out, e)
		}
	}
	return out
}

type fakeTargets struct {
	mu       sync.Mutex
	tickets  map[string]TicketTarget
	docs     map[string]DocTarget
	statuses map[string]StatusTarget
}

func (f *fakeTargets) GetTicket(_ context.Context, id string) (TicketTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tickets[id]
	if !ok {
		return TicketTarget{}, apperrs.ErrNotFound
	}
	return t, nil
}

func (f *fakeTargets) GetDoc(_ context.Context, id string) (DocTarget, error) {
	d, ok := f.docs[id]
	if !ok {
		return DocTarget{}, apperrs.ErrNotFound
	}
	return d, nil
}

func (f *fakeTargets) GetStatus(_ context.Context, id string) (StatusTarget, error) {
	s, ok := f.statuses[id]
	if !ok {
		return StatusTarget{}, apperrs.ErrNotFound
	}
	return s, nil
}

func (f *fakeTargets) setTicketStage(id string, stage Stage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t := f.tickets[id]
	t.Stage = stage
	f.tickets[id] = t
}

type fakeMove struct {
	ticketID, statusID string
	actor              PlayActor
}

// fakeMover records the runner's move-to calls.
type fakeMover struct {
	mu    sync.Mutex
	moves []fakeMove
	err   error
}

func (f *fakeMover) MoveTicket(_ context.Context, ticketID, statusID string, actor PlayActor) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.moves = append(f.moves, fakeMove{ticketID, statusID, actor})
	return nil
}

func (f *fakeMover) snapshot() []fakeMove {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]fakeMove{}, f.moves...)
}

// fakeLive records every frame published on the live topic.
type fakeLive struct {
	mu     sync.Mutex
	frames []RunFrame
}

func (f *fakeLive) Publish(_ context.Context, topic string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if topic != TopicPlayRun {
		return nil
	}
	if raw, ok := payload.(json.RawMessage); ok {
		var frame RunFrame
		if err := json.Unmarshal(raw, &frame); err != nil {
			return err
		}
		payload = frame
	}
	f.frames = append(f.frames, payload.(RunFrame))
	return nil
}

func (f *fakeLive) snapshot() []RunFrame {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]RunFrame{}, f.frames...)
}

type fakeProjects struct {
	workspaces map[string]string
}

func (f *fakeProjects) WorkspaceForProject(_ context.Context, projectID string) (string, error) {
	ws, ok := f.workspaces[projectID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return ws, nil
}

type fakeHarnessResolver struct {
	err error
	// resolved is what ResolveTarget returns on success; a zero value means "echo the caller's own choice".
	resolved HarnessChoice
	// lastChoice records the caller's requested choice, for asserting an override reached the seam.
	lastChoice HarnessChoice
}

func (f *fakeHarnessResolver) ResolveTarget(_ context.Context, _, _ string, choice HarnessChoice) (HarnessChoice, error) {
	f.lastChoice = choice
	if f.err != nil {
		return HarnessChoice{}, f.err
	}
	if f.resolved != (HarnessChoice{}) {
		return f.resolved, nil
	}
	return choice, nil
}

type fakeMemories struct {
	byProject map[string][]Memory
	err       error
}

func (f *fakeMemories) ListForProject(_ context.Context, projectID string) ([]Memory, error) {
	return f.byProject[projectID], f.err
}

// fakeAttachmentReader is the runner's AttachmentReader seam, keyed by attachment id.
type fakeAttachmentReader struct {
	items map[string]agent.StoredAttachment
	err   map[string]error
}

func (f *fakeAttachmentReader) Get(_ context.Context, id string) (agent.StoredAttachment, error) {
	if err, ok := f.err[id]; ok {
		return agent.StoredAttachment{}, err
	}
	return f.items[id], nil
}

type fakePost struct {
	conversationID, authorID, body string
}

type fakeThreads struct {
	mu        sync.Mutex
	docErr    error
	ticketErr error
	postErr   error
	posts     []fakePost
	notes     []fakePost
}

func (f *fakeThreads) GetOrCreateTicketThread(_ context.Context, _, ticketID, _ string) (string, error) {
	if f.ticketErr != nil {
		return "", f.ticketErr
	}
	return "conv-ticket-" + ticketID, nil
}

func (f *fakeThreads) GetOrCreateDocThread(_ context.Context, _, docID, _ string) (string, error) {
	if f.docErr != nil {
		return "", f.docErr
	}
	return "conv-doc-" + docID, nil
}

func (f *fakeThreads) PostMessage(_ context.Context, conversationID, authorID, body string) (string, error) {
	if f.postErr != nil {
		return "", f.postErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.posts = append(f.posts, fakePost{conversationID, authorID, body})
	return "msg-1", nil
}

func (f *fakeThreads) PostSystemNote(_ context.Context, conversationID, viaUserID, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notes = append(f.notes, fakePost{conversationID, viaUserID, body})
	return nil
}

func (f *fakeThreads) snapshot() []fakePost {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]fakePost{}, f.posts...)
}

func (f *fakeThreads) noteBodies() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []string{}
	for _, n := range f.notes {
		out = append(out, n.body)
	}
	return out
}

// fakeTurns captures the turn request and drives the observer the way the pipeline would.
type fakeTurns struct {
	mu           sync.Mutex
	reqs         []agent.TurnRequest
	interrupted  []string
	interruptErr error
	answered     []harness.QuestionAnswer
	answerErr    error
	done         chan struct{}
	// hold keeps RunTurn blocked until release, so the run stays live for Answer and Stop as a real turn would.
	hold    bool
	release chan struct{}
	once    sync.Once
}

func newFakeTurns() *fakeTurns {
	return &fakeTurns{done: make(chan struct{}, 4), release: make(chan struct{})}
}

func (f *fakeTurns) RunTurn(ctx context.Context, req agent.TurnRequest) {
	f.mu.Lock()
	f.reqs = append(f.reqs, req)
	f.mu.Unlock()
	f.done <- struct{}{}
	if !f.hold {
		return
	}
	select {
	case <-ctx.Done():
	case <-f.release:
	}
}

func (f *fakeTurns) releaseAll() {
	f.once.Do(func() { close(f.release) })
}

func (f *fakeTurns) Interrupt(_ context.Context, conversationID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.interruptErr != nil {
		return f.interruptErr
	}
	f.interrupted = append(f.interrupted, conversationID)
	return nil
}

func (f *fakeTurns) Answer(_ context.Context, _, _ string, answer harness.QuestionAnswer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.answerErr != nil {
		return f.answerErr
	}
	f.answered = append(f.answered, answer)
	return nil
}

func (f *fakeTurns) last() agent.TurnRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reqs[len(f.reqs)-1]
}

type fakeUsers struct{}

func (fakeUsers) Login(_ context.Context, userID string) (string, error) {
	return "login-" + userID, nil
}

// --- the agent pipeline's own seams, for the happy path against harnesstest.Client -------------------------

type agentConvs struct {
	mu      sync.Mutex
	replies []string
	notes   []string
}

func (a *agentConvs) GetConversation(_ context.Context, id string) (agent.Conversation, error) {
	return agent.Conversation{ID: id}, nil
}

func (a *agentConvs) MessagesSince(context.Context, string, time.Time) ([]agent.ConversationMessage, error) {
	return nil, nil
}

func (a *agentConvs) SetThread(context.Context, string, string) error { return nil }

func (a *agentConvs) MarkSynced(context.Context, string, time.Time) error { return nil }

func (a *agentConvs) PostAgentReply(_ context.Context, _, _, body string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.replies = append(a.replies, body)
	return "reply-1", nil
}

func (a *agentConvs) PostSystemNote(_ context.Context, _, _, body string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.notes = append(a.notes, body)
	return nil
}

func (a *agentConvs) PostUserMessage(context.Context, string, string, string) error { return nil }

func (a *agentConvs) snapshot() ([]string, []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string{}, a.replies...), append([]string{}, a.notes...)
}

type agentTargets struct{}

func (agentTargets) ResolveTarget(context.Context, string, string) (*pairing.ResolvedTarget, error) {
	return &pairing.ResolvedTarget{Computer: pairing.Computer{ID: "c-1", Kind: "t3code"}, HarnessProjectID: "hp-1"}, nil
}

func (agentTargets) ResolveTargetOverride(_ context.Context, _, _, computerID, provider, model string) (*pairing.ResolvedTarget, error) {
	return &pairing.ResolvedTarget{Computer: pairing.Computer{ID: computerID, Kind: "t3code"}, HarnessProjectID: "hp-1", Provider: provider, Model: model}, nil
}

type agentLive struct{}

func (agentLive) Publish(context.Context, string, any) error { return nil }

// step is a tool step as the harness seam hands it over; the pipeline forwards it to the trail unchanged.
func step(tool, summary string) harness.Activity {
	return harness.Activity{Kind: harness.ActivityToolCall, Tool: tool, Summary: summary, At: time.Unix(1_800_000_000, 0).UTC()}
}

func summaries(entries []ActivityEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Summary)
	}
	return out
}

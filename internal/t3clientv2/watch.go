package t3clientv2

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
)

// streamItem is one item of an orchestration.subscribeThread stream; its cursor is Sequence, not the event's.
type streamItem struct {
	Kind             string     `json:"kind"`
	Sequence         int64      `json:"sequence"`
	Event            wireEvent  `json:"event"`
	SnapshotSequence int64      `json:"snapshotSequence"`
	Projection       projection `json:"projection"`
}

type wireEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// projection is the slice of a thread projection the turn reads; T3 leaves optional keys out rather than nulling them.
type projection struct {
	Thread           appThread         `json:"thread"`
	Runs             []run             `json:"runs"`
	TurnItems        []turnItem        `json:"turnItems"`
	ProviderSessions []providerSession `json:"providerSessions"`
	RuntimeRequests  []runtimeRequest  `json:"runtimeRequests"`
	Nodes            []node            `json:"nodes"`
	Subagents        []subagent        `json:"subagents"`
	Messages         []message         `json:"messages"`
}

type appThread struct {
	RuntimeMode    string         `json:"runtimeMode"`
	ModelSelection modelSelection `json:"modelSelection"`
	// HistoryOrigin is absent on a native thread and "v1_import" on one T3 copied from protocol 1.
	HistoryOrigin string  `json:"historyOrigin"`
	DeletedAt     *string `json:"deletedAt"`
	LimitRecovery *struct {
		RunID   string `json:"runId"`
		ResetAt string `json:"resetAt"`
	} `json:"limitRecovery"`
}

type run struct {
	ID                         string  `json:"id"`
	Ordinal                    int     `json:"ordinal"`
	UserMessageID              string  `json:"userMessageId"`
	RootNodeID                 string  `json:"rootNodeId"`
	Status                     string  `json:"status"`
	QueueHeld                  bool    `json:"queueHeld"`
	CompletedAt                *string `json:"completedAt"`
	RestartContinuationOfRunID string  `json:"restartContinuationOfRunId"`
	// DelegatedCompletion names the message that will carry the run's delegated results back, once T3 dispatches it.
	DelegatedCompletion *struct {
		Delivery *struct {
			MessageID string `json:"messageId"`
		} `json:"delivery"`
	} `json:"delegatedCompletion"`
}

// subagent is work a run handed to another agent: T3's own delegated task (app_owned) or a provider's subagent.
type subagent struct {
	ID                 string `json:"id"`
	RunID              string `json:"runId"`
	Origin             string `json:"origin"`
	Status             string `json:"status"`
	ChildThreadID      string `json:"childThreadId"`
	Driver             string `json:"driver"`
	Model              string `json:"model"`
	Title              string `json:"title"`
	Prompt             string `json:"prompt"`
	Result             string `json:"result"`
	StartedAt          string `json:"startedAt"`
	CompletionDelivery *struct {
		State string `json:"state"`
	} `json:"completionDelivery"`
}

// message is a user message's link back to handed-off work: a delegated task's result, or a notice about a subagent.
type message struct {
	ID                  string `json:"id"`
	RunID               string `json:"runId"`
	Role                string `json:"role"`
	DelegatedCompletion *struct {
		ParentRunID string `json:"parentRunId"`
	} `json:"delegatedCompletion"`
	Notification *struct {
		Source struct {
			Kind          string `json:"kind"`
			Work          string `json:"work"`
			ChildThreadID string `json:"childThreadId"`
		} `json:"source"`
	} `json:"notification"`
}

// subagentThread is the child thread a subagent's notice is about, "" for any other message.
func (m message) subagentThread() string {
	if m.Notification == nil || m.Notification.Source.Kind != "background_task" || m.Notification.Source.Work != "subagent" {
		return ""
	}
	return m.Notification.Source.ChildThreadID
}

type failure struct {
	Class   string  `json:"class"`
	Message string  `json:"message"`
	ResetAt *string `json:"resetAt"`
}

// runtimeRequest is a question or approval T3 holds open; capability "message" answers it with a run of its own.
type runtimeRequest struct {
	ID                 string `json:"id"`
	NodeID             string `json:"nodeId"`
	Status             string `json:"status"`
	ResponseCapability struct {
		Type string `json:"type"`
	} `json:"responseCapability"`
}

// node is one step of a run; a pending request's node is always in a snapshot, which names the request's run.
type node struct {
	ID    string `json:"id"`
	RunID string `json:"runId"`
}

type providerSession struct {
	LastError *string `json:"lastError"`
}

const (
	runQueued    = "queued"
	runWaiting   = "waiting"
	runCompleted = "completed"
)

// workingRuns are the statuses of a run whose reply is still to come.
var workingRuns = []string{runQueued, "preparing", "starting", "running"}

// liveRuns are the statuses that make a thread busy: T3 queues a new message behind them.
var liveRuns = append(slices.Clone(workingRuns), runWaiting)

// undelivered are the states of a delegated task's result T3 has yet to hand back to its parent.
var undelivered = []string{"pending", "claimed"}

const (
	queuedNote  = "Queued in T3 Code behind another run on this thread"
	heldNote    = "Held in T3 Code's queue; resume it in T3 Code or stop it here"
	handoffNote = "Waiting for work handed off in T3 Code"
	steeredNote = "The handed-off result went to a later reply in T3 Code"
	noReason    = "T3 Code reported the run failed without saying why."
)

// watch folds one thread's stream into the updates of the turn whose run carries messageID; it does no I/O.
type watch struct {
	messageID string
	cursor    int64
	synced    bool
	thread    appThread
	// runs is every run by id, for the checks made before dispatching and the run an answer resumes.
	runs map[string]run
	// run is the turn's own run, its ID empty until T3 reports it.
	run run
	// ownReplied is whether the turn's own run reached waiting or completed, which a later restart cancel hides.
	ownReplied bool
	// followed is the turn's own run and every run its handed-off work woke; their items are the turn's (ADR 0116).
	followed map[string]bool
	// deliveries maps each wake message id a run's delegatedCompletion named to that run; a run.updated without one keeps it.
	deliveries map[string]string
	// messages are the user messages that may link a wake run to handed-off work, kept beyond any snapshot's window.
	messages  map[string]message
	subagents map[string]subagent
	// failures holds the newest failed error of each node of the turn's run; the root node's is the turn's failure.
	failures     map[string]failure
	sessionError string
	sent         map[string]harness.Snapshot
	// steps is the step last emitted per item, so an item a snapshot re-sends unchanged emits nothing.
	steps map[string]harness.Activity
	// asked holds the request ids of the questions and approvals already raised.
	asked map[string]bool
	// declines are the approvals raised since the pump last answered them.
	declines []string
	ended    bool
}

func newWatch(messageID string) *watch {
	return &watch{messageID: messageID, runs: map[string]run{}, followed: map[string]bool{}, deliveries: map[string]string{},
		messages: map[string]message{}, subagents: map[string]subagent{}, failures: map[string]failure{}, sent: map[string]harness.Snapshot{},
		steps: map[string]harness.Activity{}, asked: map[string]bool{}}
}

// apply folds one stream item into the watch and returns what the harness should emit for it.
func (w *watch) apply(item streamItem) ([]harness.Update, *harness.TurnResult) {
	if item.Kind == "snapshot" {
		w.cursor = item.SnapshotSequence
		return w.reset(item.Projection)
	}
	if item.Kind != "event" || item.Sequence <= w.cursor {
		return nil, nil
	}
	w.cursor = item.Sequence
	return w.event(item.Event)
}

// reset takes a snapshot as the thread's state; the replies already emitted stay emitted.
func (w *watch) reset(p projection) ([]harness.Update, *harness.TurnResult) {
	w.synced = true
	w.thread = p.Thread
	w.runs = map[string]run{}
	for _, r := range p.Runs {
		w.track(r)
	}
	// A bounded snapshot holds every working subagent, but a finished one only while its run is in the snapshot's window.
	w.subagents = map[string]subagent{}
	for _, s := range p.Subagents {
		w.subagents[s.ID] = s
	}
	for _, m := range p.Messages {
		w.message(m)
	}
	w.link()
	for _, s := range p.ProviderSessions {
		w.sessionLastError(s)
	}
	items := slices.Clone(p.TurnItems)
	slices.SortStableFunc(items, func(a, b turnItem) int { return a.Ordinal - b.Ordinal })
	var out []harness.Update
	for _, it := range items {
		out = append(out, w.item(it)...)
	}
	notes, end := w.end()
	return append(out, notes...), end
}

func (w *watch) event(e wireEvent) ([]harness.Update, *harness.TurnResult) {
	switch {
	case e.Type == "run.created" || e.Type == "run.updated":
		var r run
		if !decode(e, &r) {
			return nil, nil
		}
		w.track(r)
		w.link()
		return w.end()
	case e.Type == "message.updated":
		var m message
		if !decode(e, &m) {
			return nil, nil
		}
		w.message(m)
		w.link()
		return w.end()
	case e.Type == "subagent.updated":
		var s subagent
		if !decode(e, &s) {
			return nil, nil
		}
		w.subagent(s)
		w.link()
		return w.end()
	case e.Type == "turn-item.updated":
		var it turnItem
		if !decode(e, &it) {
			return nil, nil
		}
		return w.item(it), nil
	case e.Type == "provider-session.attached" || e.Type == "provider-session.updated":
		var s providerSession
		if decode(e, &s) {
			w.sessionLastError(s)
		}
	case strings.HasPrefix(e.Type, "thread."):
		var t appThread
		if decode(e, &t) {
			w.thread = t
		}
	}
	return nil, nil
}

// decode reads an event's payload; a payload that does not decode is skipped like an unknown event.
func decode(e wireEvent, into any) bool {
	return json.Unmarshal(e.Payload, into) == nil
}

func (w *watch) track(r run) {
	if d := r.DelegatedCompletion; d != nil && d.Delivery != nil {
		w.deliveries[d.Delivery.MessageID] = r.ID
	}
	w.runs[r.ID] = r
	if r.UserMessageID == w.messageID || (w.run.ID != "" && r.ID == w.run.ID) {
		w.own(r)
	}
}

func (w *watch) own(r run) {
	w.run = r
	w.ownReplied = w.ownReplied || replied(r)
	if r.ID != "" {
		w.followed[r.ID] = true
	}
}

// follow makes the turn the run messageID starts or is steered into, for an answer that resumes a run of T3's own.
func (w *watch) follow(messageID string) {
	w.messageID = messageID
	for _, r := range w.runs {
		if r.UserMessageID == messageID {
			w.own(r)
		}
	}
}

// subagent keeps a row's known completionDelivery when an update leaves it out, as T3's own projection does.
func (w *watch) subagent(s subagent) {
	if s.CompletionDelivery == nil {
		s.CompletionDelivery = w.subagents[s.ID].CompletionDelivery
	}
	w.subagents[s.ID] = s
}

// message keeps a user message that could link a run to handed-off work; no other message changes what the turn follows.
func (w *watch) message(m message) {
	if m.Role == "user" && (m.DelegatedCompletion != nil || m.subagentThread() != "") {
		w.messages[m.ID] = m
	}
}

// link grows the followed set by every run the followed runs' handed-off work caused, until nothing more joins.
func (w *watch) link() {
	for grew := true; grew; {
		grew = false
		for _, r := range w.runs {
			if !w.followed[r.ID] && w.caused(r) {
				w.followed[r.ID], grew = true, true
			}
		}
	}
}

// caused is a run that carries a followed run's handed-off result back, or continues a followed run after a restart.
func (w *watch) caused(r run) bool {
	if w.followed[r.RestartContinuationOfRunID] || w.followed[w.deliveries[r.UserMessageID]] {
		return true
	}
	m, ok := w.messages[r.UserMessageID]
	return ok && w.linked(m)
}

// linked is a message carrying back work a followed run handed off: its delegated result, or a notice about its subagent.
func (w *watch) linked(m message) bool {
	if m.DelegatedCompletion != nil && w.followed[m.DelegatedCompletion.ParentRunID] {
		return true
	}
	child := m.subagentThread()
	for _, s := range w.subagents {
		if child != "" && s.ChildThreadID == child && w.followed[s.RunID] {
			return true
		}
	}
	return false
}

// steered is whether T3 steered a handed-off result into a run outside the turn, which will reply with it instead.
func (w *watch) steered() bool {
	for _, m := range w.messages {
		r, ok := w.runs[m.RunID]
		if ok && r.UserMessageID != m.ID && !w.followed[r.ID] && w.linked(m) {
			return true
		}
	}
	return false
}

// pending is whether work the turn handed off is still running or its result is still on its way back.
func (w *watch) pending() bool {
	if w.woken(func(r run) bool { return slices.Contains(workingRuns, r.Status) }) {
		return true
	}
	for _, s := range w.subagents {
		if !w.followed[s.RunID] {
			continue
		}
		if slices.Contains(openItems, s.Status) {
			return true
		}
		if s.Origin == "app_owned" && s.CompletionDelivery != nil && slices.Contains(undelivered, s.CompletionDelivery.State) {
			return true
		}
	}
	return false
}

// woken is whether a run the turn's hand-offs woke, not its own run, matches is.
func (w *watch) woken(is func(run) bool) bool {
	for id := range w.followed {
		if id != w.run.ID && is(w.runs[id]) {
			return true
		}
	}
	return false
}

// replied is a run that has its reply: waiting or completed.
func replied(r run) bool { return r.Status == runWaiting || r.Status == runCompleted }

func (w *watch) sessionLastError(s providerSession) {
	if s.LastError != nil && *s.LastError != "" {
		w.sessionError = *s.LastError
	}
}

// item maps one turn item of a followed run; T3 sends an assistant message's whole text each time, never a delta.
func (w *watch) item(it turnItem) []harness.Update {
	// A message T3 steered into a running run gets no run of its own; its item names the run that took it.
	if w.run.ID == "" && it.Type == "user_message" && it.MessageID == w.messageID {
		w.own(w.runs[it.RunID])
	}
	if w.ended || w.run.ID == "" || !w.followed[it.RunID] {
		return nil
	}
	if it.Type == "error" && it.Status == "failed" && it.Failure != nil {
		w.failures[it.NodeID] = *it.Failure
		return nil
	}
	if it.Type != "assistant_message" {
		return w.step(it)
	}
	snap := harness.Snapshot{MessageID: it.MessageID, Text: it.Text, Streaming: it.Streaming}
	if w.sent[it.MessageID] == snap {
		return nil
	}
	w.sent[it.MessageID] = snap
	return []harness.Update{{Snapshot: &snap}}
}

// step emits what the mapper makes of an item: a step each time it changes, a question or approval once per request.
func (w *watch) step(it turnItem) []harness.Update {
	u, ok := mapItem(it)
	if !ok {
		return nil
	}
	if u.Activity != nil {
		if w.steps[it.ID] == *u.Activity {
			return nil
		}
		w.steps[it.ID] = *u.Activity
		return []harness.Update{u}
	}
	if w.asked[it.RequestID] {
		return nil
	}
	w.asked[it.RequestID] = true
	if u.Approval != nil {
		w.declines = append(w.declines, it.RequestID)
	}
	return []harness.Update{u}
}

// end is the turn's terminal result, once: at its run's end, but for a done or cut run only after the work it handed off.
func (w *watch) end() ([]harness.Update, *harness.TurnResult) {
	if w.ended || w.run.ID == "" {
		return nil, nil
	}
	result := terminal(w.run.Status)
	if w.cut() {
		result = &harness.TurnResult{State: harness.TurnDone}
	}
	if result == nil {
		return nil, nil
	}
	if result.State == harness.TurnError {
		result.LastError = w.failure()
	}
	if result.State == harness.TurnDone && w.steered() {
		w.ended = true
		return []harness.Update{note(steeredNote)}, result
	}
	if result.State == harness.TurnDone && w.pending() {
		return nil, nil
	}
	if w.cut() && !w.ownReplied && !w.woken(replied) {
		result.State = harness.TurnInterrupted
	}
	w.ended = true
	return nil, result
}

// cut is the turn's run cancelled by a T3 restart, which leaves its handed-off work running; a deleted thread's cancel ends it all.
func (w *watch) cut() bool {
	return w.run.Status == "cancelled" && w.thread.DeletedAt == nil
}

// waiting is whether the turn's own run is done or cut and only the work it handed off keeps the turn open.
func (w *watch) waiting() bool {
	return !w.ended && (replied(w.run) || w.cut())
}

// leave ends a turn that stopped waiting for the work it handed off, which T3 still runs.
func (w *watch) leave() *harness.TurnResult {
	w.ended = true
	return &harness.TurnResult{State: harness.TurnDone, LeftRunning: true}
}

// stop ends a turn without waiting on T3, since a stopped waiting run can stay waiting and dropped work reports no end.
func (w *watch) stop() *harness.TurnResult {
	w.ended = true
	return &harness.TurnResult{State: harness.TurnInterrupted}
}

// handoffs is what Stop must reach besides the turn's own run.
func (w *watch) handoffs() handoffs {
	var h handoffs
	for id := range w.followed {
		if id != w.run.ID {
			h.runs = append(h.runs, id)
		}
	}
	for _, s := range w.subagents {
		if w.followed[s.RunID] {
			h.subagents = append(h.subagents, s)
		}
	}
	return h
}

// terminal maps a run status to the turn's end; waiting is done, since T3 only captures a checkpoint after it.
func terminal(status string) *harness.TurnResult {
	switch status {
	case runWaiting, runCompleted:
		return &harness.TurnResult{State: harness.TurnDone}
	case "interrupted", "cancelled", "rolled_back":
		return &harness.TurnResult{State: harness.TurnInterrupted}
	case "failed":
		return &harness.TurnResult{State: harness.TurnError}
	}
	return nil
}

// failure is why the turn's run failed: its root error, then the provider session's, then a generic line.
func (w *watch) failure() string {
	f, ok := w.failures[w.run.RootNodeID]
	if !ok {
		if w.sessionError != "" {
			return w.sessionError
		}
		return noReason
	}
	if f.Class != "usage_limit" {
		return f.Message
	}
	resetAt := ""
	if f.ResetAt != nil {
		resetAt = *f.ResetAt
	}
	if resetAt == "" && w.thread.LimitRecovery != nil && w.thread.LimitRecovery.RunID == w.run.ID {
		resetAt = w.thread.LimitRecovery.ResetAt
	}
	if resetAt == "" {
		return f.Message
	}
	return fmt.Sprintf("%s (the limit resets at %s)", f.Message, readableTime(resetAt))
}

func readableTime(iso string) string {
	t, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		return iso
	}
	return t.UTC().Format("2006-01-02 15:04 UTC")
}

// queued is whether T3 still holds the turn's run in the thread's queue.
func (w *watch) queued() bool {
	return !w.ended && w.run.Status == runQueued
}

// standingNote is the note a waiting turn repeats: its run held in T3's queue, or handed-off work still pending.
func (w *watch) standingNote() *harness.Activity {
	if w.queued() {
		summary := queuedNote
		if w.run.QueueHeld {
			summary = heldNote
		}
		return &harness.Activity{Kind: harness.ActivityNote, CallID: "queue:" + w.run.ID, Summary: summary}
	}
	if w.ended || w.run.ID == "" || !w.pending() {
		return nil
	}
	summary := handoffNote
	if w.woken(func(r run) bool { return r.Status == runQueued && r.QueueHeld }) {
		summary = heldNote
	}
	return &harness.Activity{Kind: harness.ActivityNote, CallID: "handoff:" + w.run.ID, Summary: summary}
}

// busy is whether a run on the thread is queued or live, which a runtime-mode change would detach.
func (w *watch) busy() bool {
	for _, r := range w.runs {
		if slices.Contains(liveRuns, r.Status) {
			return true
		}
	}
	return false
}

// imported is a thread T3 copied from protocol 1 with no completed run, which holds only an excerpt of its history.
func (w *watch) imported() bool {
	if w.thread.HistoryOrigin != "v1_import" {
		return false
	}
	for _, r := range w.runs {
		if r.Status == runCompleted {
			return false
		}
	}
	return true
}

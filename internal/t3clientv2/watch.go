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
}

type appThread struct {
	RuntimeMode string `json:"runtimeMode"`
	// HistoryOrigin is absent on a native thread and "v1_import" on one T3 copied from protocol 1.
	HistoryOrigin string  `json:"historyOrigin"`
	DeletedAt     *string `json:"deletedAt"`
	LimitRecovery *struct {
		RunID   string `json:"runId"`
		ResetAt string `json:"resetAt"`
	} `json:"limitRecovery"`
}

type run struct {
	ID            string `json:"id"`
	UserMessageID string `json:"userMessageId"`
	RootNodeID    string `json:"rootNodeId"`
	Status        string `json:"status"`
	QueueHeld     bool   `json:"queueHeld"`
}

type turnItem struct {
	ID        string   `json:"id"`
	RunID     string   `json:"runId"`
	NodeID    string   `json:"nodeId"`
	Ordinal   int      `json:"ordinal"`
	Status    string   `json:"status"`
	Type      string   `json:"type"`
	MessageID string   `json:"messageId"`
	Text      string   `json:"text"`
	Streaming bool     `json:"streaming"`
	Failure   *failure `json:"failure"`
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

// liveRuns are the statuses that make a thread busy: T3 queues a new message behind them.
var liveRuns = []string{runQueued, "preparing", "starting", "running", runWaiting}

const (
	queuedNote = "Queued in T3 Code behind another run on this thread"
	heldNote   = "Held in T3 Code's queue; resume it in T3 Code or stop it here"
	noReason   = "T3 Code reported the run failed without saying why."
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
	// failures holds the newest failed error of each node of the turn's run; the root node's is the turn's failure.
	failures     map[string]failure
	sessionError string
	sent         map[string]harness.Snapshot
	ended        bool
}

func newWatch(messageID string) *watch {
	return &watch{messageID: messageID, runs: map[string]run{}, failures: map[string]failure{}, sent: map[string]harness.Snapshot{}}
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
	for _, s := range p.ProviderSessions {
		w.sessionLastError(s)
	}
	items := slices.Clone(p.TurnItems)
	slices.SortStableFunc(items, func(a, b turnItem) int { return a.Ordinal - b.Ordinal })
	var out []harness.Update
	for _, it := range items {
		out = append(out, w.item(it)...)
	}
	return out, w.end()
}

func (w *watch) event(e wireEvent) ([]harness.Update, *harness.TurnResult) {
	switch {
	case e.Type == "run.created" || e.Type == "run.updated":
		var r run
		if !decode(e, &r) {
			return nil, nil
		}
		w.track(r)
		return nil, w.end()
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
	w.runs[r.ID] = r
	if r.UserMessageID == w.messageID || (w.run.ID != "" && r.ID == w.run.ID) {
		w.run = r
	}
}

// follow makes the turn the run messageID starts or is steered into, for an answer that resumes a run of T3's own.
func (w *watch) follow(messageID string) {
	w.messageID = messageID
	for _, r := range w.runs {
		if r.UserMessageID == messageID {
			w.run = r
		}
	}
}

func (w *watch) sessionLastError(s providerSession) {
	if s.LastError != nil && *s.LastError != "" {
		w.sessionError = *s.LastError
	}
}

// item maps one turn item of the turn's own run; T3 sends an assistant message's whole text each time, never a delta.
func (w *watch) item(it turnItem) []harness.Update {
	// A message T3 steered into a running run gets no run of its own; its item names the run that took it.
	if w.run.ID == "" && it.Type == "user_message" && it.MessageID == w.messageID {
		w.run = w.runs[it.RunID]
	}
	if w.run.ID == "" || it.RunID != w.run.ID {
		return nil
	}
	if it.Type == "error" && it.Status == "failed" && it.Failure != nil {
		w.failures[it.NodeID] = *it.Failure
		return nil
	}
	if it.Type != "assistant_message" {
		return nil
	}
	snap := harness.Snapshot{MessageID: it.MessageID, Text: it.Text, Streaming: it.Streaming}
	if w.sent[it.MessageID] == snap {
		return nil
	}
	w.sent[it.MessageID] = snap
	return []harness.Update{{Snapshot: &snap}}
}

// end is the turn's terminal result the first time its run reaches one; a run that stopped reports its end twice.
func (w *watch) end() *harness.TurnResult {
	if w.ended || w.run.ID == "" {
		return nil
	}
	result := terminal(w.run.Status)
	if result == nil {
		return nil
	}
	if result.State == harness.TurnError {
		result.LastError = w.failure()
	}
	w.ended = true
	return result
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

// queueNote is the note for the turn's run while T3 holds it in the thread's queue, nil otherwise.
func (w *watch) queueNote() *harness.Activity {
	if !w.queued() {
		return nil
	}
	summary := queuedNote
	if w.run.QueueHeld {
		summary = heldNote
	}
	return &harness.Activity{Kind: harness.ActivityNote, CallID: "queue:" + w.run.ID, Summary: summary}
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

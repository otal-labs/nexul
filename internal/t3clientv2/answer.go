package t3clientv2

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// answeredNote ends a turn whose pending answer T3 already holds, from its own UI.
const answeredNote = "Already answered in T3 Code"

// errAnswered is a pending answer T3 already holds; the turn ends before it sends anything.
var errAnswered = errors.New("already answered in T3 Code")

type requestRespond struct {
	Type      string         `json:"type"`
	CommandID string         `json:"commandId"`
	ThreadID  string         `json:"threadId"`
	RequestID string         `json:"requestId"`
	Answers   map[string]any `json:"answers"`
}

// Answer implements harness.Client with runtime-request.respond, encoded for how T3 resumes that request.
func (h *Harness) Answer(ctx context.Context, target harness.Target, requestID string, answer harness.QuestionAnswer) (err error) {
	if target.SessionID == "" {
		return fmt.Errorf("%w: no active session to answer", apperrs.ErrInvalid)
	}
	c, err := h.connect(ctx, target.Session)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, c.Close())
	}()
	p, err := readProjection(ctx, c, target.SessionID)
	if err != nil {
		return err
	}
	req, _ := p.request(requestID)
	asMessage := req.ResponseCapability.Type == "message"
	_, err = c.Call(ctx, dispatchCommand, requestRespond{Type: "runtime-request.respond", CommandID: ids.New(),
		ThreadID: target.SessionID, RequestID: requestID, Answers: encodeAnswers(answer, asMessage)})
	if t3Message(err) == "Runtime request "+requestID+" is resolved." {
		return fmt.Errorf("%w: question %s was already answered", apperrs.ErrConflict, requestID)
	}
	if err != nil {
		return refused("answer the T3 question", err)
	}
	return nil
}

// encodeAnswers keys each answer by question id: free text wins, one choice is a string and several a list, except that
// a request answered by a message of its own needs one non-empty string per question.
func encodeAnswers(answer harness.QuestionAnswer, asMessage bool) map[string]any {
	out := make(map[string]any, len(answer.Answers))
	for id, v := range answer.Answers {
		if asMessage || v.Text != "" || len(v.Selected) <= 1 {
			out[id] = v.String()
			continue
		}
		out[id] = v.Selected
	}
	return out
}

// deliver responds to the question an ended turn left open; false means send it as a message, errAnswered means T3 holds one.
func (t *turn) deliver(ctx context.Context, w *watch, a harness.PendingAnswer) (bool, error) {
	req, ok := t.snapshot.request(a.RequestID)
	if ok && req.Status == "resolved" {
		return false, errAnswered
	}
	capability := req.ResponseCapability.Type
	if !ok || req.Status != "pending" || (capability != "live" && capability != "message") {
		return false, nil
	}
	_, err := t.conn.Call(ctx, dispatchCommand, requestRespond{Type: "runtime-request.respond", CommandID: ids.New(),
		ThreadID: t.threadID, RequestID: a.RequestID, Answers: encodeAnswers(a.Answer, capability == "message")})
	// The request can change after the snapshot; only then does T3's refusal decide.
	prefix := "Runtime request " + a.RequestID
	switch t3Message(err) {
	case prefix + " is resolved.":
		return false, errAnswered
	case prefix + " is expired.", prefix + " is cancelled.", prefix + " was not found.":
		return false, nil
	}
	if err != nil {
		return false, refused("answer the T3 question", err)
	}
	// T3 runs a message-mode answer as a message of its own, queued or steered into the live run.
	messageID := "async-answer:" + a.RequestID
	if capability == "live" {
		messageID = w.runs[t.snapshot.runOf(req.NodeID)].UserMessageID
	}
	t.messageID = messageID
	w.follow(messageID)
	return true, nil
}

// answeredTurn ends the turn before it starts: the run T3 began from its own answer carries on there.
func answeredTurn(threadID string) harness.StartResult {
	updates := make(chan harness.Update, 2)
	updates <- note(answeredNote)
	updates <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
	close(updates)
	return harness.StartResult{SessionID: threadID, Updates: updates}
}

// request is the runtime request id, if the projection holds it.
func (p projection) request(id string) (runtimeRequest, bool) {
	i := slices.IndexFunc(p.RuntimeRequests, func(r runtimeRequest) bool { return r.ID == id })
	if i < 0 {
		return runtimeRequest{}, false
	}
	return p.RuntimeRequests[i], true
}

// runOf is the run nodeID belongs to, "" when the projection does not hold the node.
func (p projection) runOf(nodeID string) string {
	i := slices.IndexFunc(p.Nodes, func(n node) bool { return n.ID == nodeID })
	if i < 0 {
		return ""
	}
	return p.Nodes[i].RunID
}

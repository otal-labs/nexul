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
	asMessage := slices.ContainsFunc(p.RuntimeRequests, func(r runtimeRequest) bool {
		return r.ID == requestID && r.ResponseCapability.Type == "message"
	})
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

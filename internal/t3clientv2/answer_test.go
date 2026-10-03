package t3clientv2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

func pendingRequest(capability string) map[string]any {
	return map[string]any{"id": "rq-1", "nodeId": "node-1", "kind": "user_input", "status": "pending",
		"responseCapability": map[string]any{"type": capability}}
}

var multiAnswer = harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{
	"Which DB?":     {Text: "sqlite"},
	"Pick one":      {Selected: []string{"auth"}},
	"Pick features": {Selected: []string{"auth", "search"}},
	"Other":         {Text: "my own", Selected: []string{"auth", "search"}},
}}

func TestAnswer_EncodesForHowT3ResumesTheRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		requests []any
		want     map[string]any
	}{
		{"a live question takes several choices as a list; free text wins", []any{pendingRequest("live")},
			map[string]any{"Which DB?": "sqlite", "Pick one": "auth", "Pick features": []any{"auth", "search"}, "Other": "my own"}},
		{"a question answered by a message of its own takes one string per question", []any{pendingRequest("message")},
			map[string]any{"Which DB?": "sqlite", "Pick one": "auth", "Pick features": "auth, search", "Other": "my own"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			f.Projection = projectionWith(t, []any{runAt(1, "msg-1", "running")}, tt.requests...)

			require.NoError(t, h.Answer(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1"}, "rq-1", multiAnswer))
			cmd := t3rpctest.WaitFor(t, f.Dispatched, "runtime-request.respond")
			assert.Equal(t, map[string]any{"type": "runtime-request.respond", "commandId": cmd["commandId"], "threadId": "th-1",
				"requestId": "rq-1", "answers": tt.want}, cmd)
		})
	}
}

func TestAnswer_Refused(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		message string
		errIs   error
		want    string
	}{
		{"a question answered already conflicts", "Runtime request rq-1 is resolved.", apperrs.ErrConflict,
			"conflict: question rq-1 was already answered"},
		{"any other refusal carries T3's message", "Runtime request rq-1 is expired.", apperrs.ErrInvalid,
			"invalid: answer the T3 question: Runtime request rq-1 is expired."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			f.Projection = projectionWith(t, []any{runAt(1, "msg-1", "running")}, pendingRequest("live"))
			f.CommandCauses = map[string]any{"runtime-request.respond": rejected("runtime-request.respond", tt.message)}

			err := h.Answer(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1"}, "rq-1", multiAnswer)
			require.ErrorIs(t, err, tt.errIs)
			assert.EqualError(t, err, tt.want)
		})
	}
}

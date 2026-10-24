package agent

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler_Steer(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "a body naming no message is refused", body: `{}`, want: http.StatusBadRequest},
		{name: "a message no queued turn answers is not found", body: `{"message_id":"msg-1"}`, want: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(Config{Conversations: newFakeConversations(Conversation{ID: "conv-1"}), Targets: &fakeTargets{},
				Harnesses: registryOf(&fakeHarness{}), Live: &fakeLive{}})
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/agent/conversations/conv-1/steer", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			NewHandler(svc).Routes().ServeHTTP(rec, req)

			assert.Equal(t, tt.want, rec.Code, rec.Body.String())
		})
	}
}

package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"

	"github.com/otal-labs/nexul/internal/platform/identity"
)

// fakeProjectTickets lists one project's tickets; any other project is refused the way an unreadable one is.
type fakeProjectTickets struct {
	projectID string
	ids       []string
}

func (f fakeProjectTickets) TicketIDs(_ context.Context, projectID string) ([]string, error) {
	if projectID != f.projectID {
		return nil, apperrs.ErrForbidden
	}
	return f.ids, nil
}

func TestHandler_HasTicketThreads_ByProject(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.GetOrCreateTicketThread(t.Context(), "w-1", "t-1", "u-1")
	require.NoError(t, err)
	h := NewHandler(s, fakeProjectTickets{projectID: "p-1", ids: []string{"t-1", "t-2"}}).Routes()
	get := func(query string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(identity.WithUserID(t.Context(), "u-1"), http.MethodGet, "/api/chat/tickets/thread-status?"+query, nil))
		return rec
	}

	rec := get("project_id=p-1")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"t-1":true}`, rec.Body.String())

	rec = get("ticket_ids=t-1")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"t-1":true}`, rec.Body.String(), "ticket_ids keeps working")

	assert.Equal(t, http.StatusForbidden, get("project_id=p-other").Code)
}

func TestHandler_ScheduledMessages_ScheduleChangeSendNowAndCancel(t *testing.T) {
	s := newTestService(newFakeRepo())
	c, err := s.CreateChannel(t.Context(), "w-1", "u-1", "eng")
	require.NoError(t, err)
	h := NewHandler(s, fakeProjectTickets{}).Routes()
	call := func(user, method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(identity.WithUserID(t.Context(), user), method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rec, req)
		return rec
	}

	rec := call("u-1", http.MethodPost, "/api/chat/conversations/"+c.ID+"/scheduled-messages",
		`{"body":"Heads up","send_at":"2026-08-27T09:00:00Z","repeats":"weekly","timezone":"Europe/London"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var saved ScheduledMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &saved))

	rec = call("u-1", http.MethodPatch, "/api/chat/scheduled-messages/"+saved.ID, `{"body":"Heads up: moved"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"send_at":"2026-08-27T09:00:00Z"`, "a patch without send_at keeps it")

	assert.Equal(t, http.StatusNotFound, call("u-2", http.MethodDelete, "/api/chat/scheduled-messages/"+saved.ID, "").Code)
	rec = call("u-1", http.MethodPost, "/api/chat/scheduled-messages/"+saved.ID+"/send", "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"body":"Heads up: moved"`)

	assert.Equal(t, http.StatusNoContent, call("u-1", http.MethodDelete, "/api/chat/scheduled-messages/"+saved.ID, "").Code)
	rec = call("u-1", http.MethodGet, "/api/chat/conversations/"+c.ID+"/scheduled-messages", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `[]`, rec.Body.String())
}

package chat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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
		h.ServeHTTP(rec, httptest.NewRequestWithContext(WithUserID(t.Context(), "u-1"), http.MethodGet, "/api/chat/tickets/thread-status?"+query, nil))
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

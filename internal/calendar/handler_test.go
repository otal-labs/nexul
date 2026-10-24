package calendar

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandler_Routes drives each calendar route as the web app sends it, with the status each answer carries.
func TestHandler_Routes(t *testing.T) {
	h := newHarness()
	routes := NewHandler(h.svc).Routes()
	serve := func(user, method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(as(t, user), method, path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}

	body := `{"title":"Standup","starts_at":"2026-11-02T10:00:00Z","ends_at":"2026-11-02T10:30:00Z","timezone":"Europe/London",` +
		`"repeats":"weekly","repeat_until":"2026-12-31T00:00:00Z","invitee_ids":["bob"]}`
	rec := serve("alice", http.MethodPost, "/api/workspaces/ws-1/meetings", body)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created Meeting
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, "Europe/London", created.Timezone)
	assert.Equal(t, at("2026-12-31T00:00:00Z"), *created.RepeatUntil)
	assert.NotContains(t, rec.Body.String(), "next_remind", "the reminder plan stays on the server")

	assert.Equal(t, http.StatusBadRequest, serve("alice", http.MethodPost, "/api/workspaces/ws-1/meetings", `{`).Code)
	assert.Equal(t, http.StatusBadRequest, serve("alice", http.MethodPost, "/api/workspaces/ws-1/meetings", `{"repeat_until":"soon"}`).Code)
	assert.Equal(t, http.StatusForbidden, serve("bob", http.MethodPost, "/api/workspaces/ws-1/meetings", body).Code)

	rec = serve("bob", http.MethodGet, "/api/workspaces/ws-1/meetings?from=2026-11-01T00:00:00Z&to=2026-11-15T00:00:00Z", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var cal calendarResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cal))
	require.Len(t, cal.Meetings, 1)
	assert.Len(t, cal.Occurrences, 2)
	assert.Equal(t, http.StatusBadRequest, serve("bob", http.MethodGet, "/api/workspaces/ws-1/meetings?from=yesterday&to=2026-11-15T00:00:00Z", "").Code)
	assert.Equal(t, http.StatusBadRequest, serve("bob", http.MethodGet, "/api/workspaces/ws-1/meetings?from=2026-11-01T00:00:00Z", "").Code)

	assert.Equal(t, http.StatusOK, serve("bob", http.MethodGet, "/api/meetings/"+created.ID, "").Code)
	assert.Equal(t, http.StatusNotFound, serve("dave", http.MethodGet, "/api/meetings/"+created.ID, "").Code)

	rec = serve("alice", http.MethodPatch, "/api/meetings/"+created.ID, `{"slot":"2026-11-09T10:00:00Z","scope":"this","title":"Short one"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"series_id":"`+created.ID+`"`)
	assert.Equal(t, http.StatusBadRequest, serve("alice", http.MethodPatch, "/api/meetings/"+created.ID, `[`).Code)
	assert.Equal(t, http.StatusBadRequest, serve("alice", http.MethodPatch, "/api/meetings/"+created.ID, `{"repeat_until":"later"}`).Code)
	rec = serve("alice", http.MethodPatch, "/api/meetings/"+created.ID, `{"repeat_until":""}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "repeat_until", "an empty repeat_until has it repeat for good")

	rec = serve("bob", http.MethodPut, "/api/meetings/"+created.ID+"/reply", `{"reply":"accepted"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"reply":"accepted"`)
	assert.Equal(t, http.StatusBadRequest, serve("bob", http.MethodPut, "/api/meetings/"+created.ID+"/reply", `{`).Code)
	assert.Equal(t, http.StatusForbidden, serve("alice", http.MethodPut, "/api/meetings/"+created.ID+"/reply", `{"reply":"accepted"}`).Code)
}

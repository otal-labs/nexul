package mentions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

type stubAccess struct {
	canOpen map[string]bool
}

func (s *stubAccess) Can(_ context.Context, _, docID string, action permissions.Action) (bool, error) {
	if action != permissions.DocsRead {
		return false, nil
	}
	return s.canOpen[docID], nil
}

func newHandler(t *testing.T) *Handler {
	t.Helper()
	svc := New(Config{
		Tickets: &fakeTicketSource{
			tickets: map[string]Ticket{
				"t-1": {ID: "t-1", Title: "Fix the bug", Status: "open"},
			},
			search: []SearchHit{{ID: "t-1", Title: "Fix the bug"}},
		},
		Docs: &fakeDocSource{
			docs: map[string]Doc{
				"d-1": {ID: "d-1", Title: "Architecture"},
			},
			search: []SearchHit{{ID: "d-1", Title: "Architecture"}},
		},
		Statuses:    &fakeStatusSource{statuses: map[string]Status{"open": {ID: "open", Name: "Open"}}},
		Access:      &stubAccess{canOpen: map[string]bool{"d-1": true}},
		Projects:    &fakeProjectSource{projects: map[string]Project{}},
		TicketTypes: &fakeTicketTypeSource{types: map[string]TicketType{}},
	})
	return NewHandler(svc)
}

func doRequest(t *testing.T, h *Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	}
	req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{ID: "u-1"}))
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	return rec
}

func TestResolveEndpoint(t *testing.T) {
	h := newHandler(t)

	rec := doRequest(t, h, "POST", "/api/mentions/resolve",
		`{"refs":[{"type":"ticket","id":"t-1"},{"type":"doc","id":"d-1"},{"type":"ticket","id":"nope"}]}`)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Chips []Chip `json:"chips"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Chips, 2)
	assert.Equal(t, "Fix the bug", resp.Chips[0].Title)
	assert.Equal(t, "Open", resp.Chips[0].StatusLabel)
	assert.Equal(t, "Architecture", resp.Chips[1].Title)
	assert.True(t, resp.Chips[1].CanOpen)
}

func TestResolveEndpoint_BadJSON(t *testing.T) {
	h := newHandler(t)
	rec := doRequest(t, h, "POST", "/api/mentions/resolve", `{"refs":`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSearchEndpoint(t *testing.T) {
	h := newHandler(t)
	rec := doRequest(t, h, "GET", "/api/mentions/search?q=fix&limit=10", "")
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Results []SearchResult `json:"results"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Results)
	assert.Equal(t, "ticket", resp.Results[0].Type)
}

func TestSearchEndpoint_EmptyQuery(t *testing.T) {
	h := newHandler(t)
	rec := doRequest(t, h, "GET", "/api/mentions/search?q=", "")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSearchEndpoint_ErrorMapping(t *testing.T) {
	svc := New(Config{Tickets: &fakeTicketSource{searchErr: errors.New("boom")}, Docs: &fakeDocSource{}, Statuses: &fakeStatusSource{}})
	h := NewHandler(svc)
	rec := doRequest(t, h, "GET", "/api/mentions/search?q=x", "")
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandler_Unauthorized(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequest("POST", "/api/mentions/resolve", bytes.NewBufferString(`{"refs":[{"type":"ticket","id":"t-1"}]}`))
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req) // no actor in ctx
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

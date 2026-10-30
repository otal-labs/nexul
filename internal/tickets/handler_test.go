package tickets

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serve(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func newTicketsHandler() http.Handler {
	return NewHandler(newTestService(newFakeRepo())).Routes()
}

func decodeTicket(t *testing.T, rec *httptest.ResponseRecorder) *Ticket {
	t.Helper()
	var tk Ticket
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tk))
	return &tk
}

func TestTicketsHandler_Create(t *testing.T) {
	h := newTicketsHandler()

	t.Run("creates a ticket", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/tickets", `{"title":"Fix","body":"b","doc_id":"doc-1","developer":"onik97","tester":"lena","project_id":"p-1"}`)
		require.Equal(t, http.StatusCreated, rec.Code)
		tk := decodeTicket(t, rec)
		assert.Equal(t, "Fix", tk.Title)
		assert.Equal(t, "doc-1", tk.DocID)
		assert.Equal(t, "p-1", tk.ProjectID)
		assert.Equal(t, StatusOpen, tk.Status)
	})
}

func TestTicketsHandler_List(t *testing.T) {
	h := newTicketsHandler()
	serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","doc_id":"doc-1","project_id":"p-1"}`)
	serve(t, h, http.MethodPost, "/api/tickets", `{"title":"B","doc_id":"doc-2","project_id":"p-2"}`)

	t.Run("lists all tickets", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/tickets", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []*Ticket
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		assert.Len(t, list, 2)
	})
	t.Run("filters by doc", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/tickets?doc_id=doc-1", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []*Ticket
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list, 1)
		assert.Equal(t, "A", list[0].Title)
	})
	t.Run("filters by project", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/tickets?project_id=p-2", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []*Ticket
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list, 1)
		assert.Equal(t, "B", list[0].Title)
	})
}

func TestTicketsHandler_List_WithoutBody_LeavesBodiesOut(t *testing.T) {
	h := newTicketsHandler()
	serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","body":"a long spec","project_id":"p-1"}`)
	bodies := func(path string) []string {
		rec := serve(t, h, http.MethodGet, path, "")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []*Ticket
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		out := make([]string, len(list))
		for i, tk := range list {
			out[i] = tk.Body
		}
		return out
	}

	assert.Equal(t, []string{""}, bodies("/api/tickets?without=body"))
	assert.Equal(t, []string{""}, bodies("/api/tickets?project_id=p-1&without=body"))
	assert.Equal(t, []string{"a long spec"}, bodies("/api/tickets?project_id=p-1"))
	assert.Equal(t, []string{"a long spec"}, bodies("/api/tickets"), "leaving a body out never reaches the ticket the service holds")
}

func TestTicketsHandler_Get(t *testing.T) {
	h := newTicketsHandler()
	created := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","project_id":"p-1"}`))

	t.Run("gets a ticket by id", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/tickets/"+created.ID, "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "A", decodeTicket(t, rec).Title)
	})
}

func TestTicketsHandler_Update(t *testing.T) {
	h := newTicketsHandler()
	created := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","body":"old body","project_id":"p-1"}`))

	t.Run("edits title and body", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID, `{"title":"B","body":"new body"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		tk := decodeTicket(t, rec)
		assert.Equal(t, "B", tk.Title)
		assert.Equal(t, "new body", tk.Body)
	})
}

func TestTicketsHandler_UpdateStatus(t *testing.T) {
	h := newTicketsHandler()
	created := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","project_id":"p-1"}`))

	t.Run("transitions status", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/status", `{"status":"in_progress"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, StatusInProgress, decodeTicket(t, rec).Status)
	})
}

func TestTicketsHandler_UpdatePosition(t *testing.T) {
	h := newTicketsHandler()
	created := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","project_id":"p-1"}`))

	t.Run("sets position", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/position", `{"position":3}`)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, 3, decodeTicket(t, rec).Position)
	})
}

func TestTicketsHandler_Delete(t *testing.T) {
	h := newTicketsHandler()
	created := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","project_id":"p-1"}`))

	t.Run("deletes a ticket", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, "/api/tickets/"+created.ID, "")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

func TestTicketsHandler_Search(t *testing.T) {
	h := newTicketsHandler()
	serve(t, h, http.MethodPost, "/api/tickets", `{"title":"Storage spine","body":"sqlite migrations","project_id":"p-1"}`)

	t.Run("finds matching tickets", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/tickets/search?q=sqlite", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var results []SearchResult
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &results))
		require.Len(t, results, 1)
	})
}

func TestTicketsHandler_Links(t *testing.T) {
	h := newTicketsHandler()
	created := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","project_id":"p-1"}`))
	base := "/api/tickets/" + created.ID

	t.Run("links a branch", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, base+"/branches", `{"owner":"acme","repo":"app","branch":"ticket/1"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		var links ticketLinks
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &links))
		require.Len(t, links.Branches, 1)
		assert.Equal(t, "ticket/1", links.Branches[0].Branch)
	})
	t.Run("links a pr", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, base+"/prs", `{"owner":"acme","repo":"app","number":7,"title":"F"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		var links ticketLinks
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &links))
		require.Len(t, links.PRs, 1)
		assert.Equal(t, 7, links.PRs[0].Number)
		assert.Equal(t, PRStateOpen, links.PRs[0].State)
	})
	t.Run("lists links", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, base+"/links", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var links ticketLinks
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &links))
		assert.Len(t, links.PRs, 1)
		assert.Len(t, links.Branches, 1)
	})
	t.Run("unlinks a branch whose name has a slash and keeps the pr", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, base+"/branches?owner=acme&repo=app&branch=ticket%2F1", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var links ticketLinks
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &links))
		assert.Empty(t, links.Branches)
		assert.Len(t, links.PRs, 1)
	})
	t.Run("unlinks a pr", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, base+"/prs?owner=acme&repo=app&number=7", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var links ticketLinks
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &links))
		assert.Empty(t, links.PRs)
	})
}

func TestTicketsHandler_DevStatus(t *testing.T) {
	h := newTicketsHandler()
	a := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","project_id":"p-1"}`))
	serve(t, h, http.MethodPost, "/api/tickets/"+a.ID+"/prs", `{"owner":"acme","repo":"app","number":7,"title":"F"}`)

	t.Run("returns counts for the requested ids", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/tickets/dev-status", `{"ticket_ids":["`+a.ID+`","nope"]}`)
		require.Equal(t, http.StatusOK, rec.Code)
		var statuses map[string]DevStatusCounts
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statuses))
		assert.Equal(t, DevStatusCounts{Open: 1}, statuses[a.ID])
		assert.Equal(t, DevStatusCounts{}, statuses["nope"])
	})
	t.Run("malformed body is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/tickets/dev-status", `{`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("repo error is 500", func(t *testing.T) {
		repo := newFakeRepo()
		repo.prLinksErr = errors.New("db down")
		errHandler := NewHandler(newTestService(repo)).Routes()
		rec := serve(t, errHandler, http.MethodPost, "/api/tickets/dev-status", `{"ticket_ids":["t-1"]}`)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestTicketsHandler_TypeAndLabels(t *testing.T) {
	h := newTicketsHandler()
	created := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","project_id":"p-1","type_id":"ticket-type-bug","category_id":"c-1"}`))
	base := "/api/tickets/" + created.ID

	t.Run("create carries type and category", func(t *testing.T) {
		assert.Equal(t, "ticket-type-bug", created.TypeID)
		assert.Equal(t, "c-1", created.CategoryID)
	})
	t.Run("sets a type", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, base+"/type", `{"type_id":"ticket-type-task"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ticket-type-task", decodeTicket(t, rec).TypeID)
	})
	t.Run("adds a label", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, base+"/labels", `{"label":"bug"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, decodeTicket(t, rec).Labels, "bug")
	})
	t.Run("lists labels", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, base+"/labels", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var labels []string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &labels))
		assert.Contains(t, labels, "bug")
	})
	t.Run("removes a label", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, base+"/labels/bug", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, decodeTicket(t, rec).Labels, "bug")
	})
	t.Run("lists all labels across tickets", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/tickets/labels", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var labels []string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &labels))
		assert.NotContains(t, labels, "bug", "label was removed")
	})
}

func TestTicketsHandler_LabelColors(t *testing.T) {
	h := newTicketsHandler()

	t.Run("sets a label's color", func(t *testing.T) {
		rec := serve(t, h, http.MethodPut, "/api/tickets/labels/bug/color", `{"project_id":"p-1","color":"cyan"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		var lc LabelColor
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &lc))
		assert.Equal(t, "bug", lc.Label)
		assert.Equal(t, "cyan", string(lc.Color))
	})
	t.Run("batch-fetches colors for the requested labels", func(t *testing.T) {
		serve(t, h, http.MethodPut, "/api/tickets/labels/urgent/color", `{"project_id":"p-1","color":"orange"}`)
		rec := serve(t, h, http.MethodPost, "/api/tickets/labels/colors", `{"project_id":"p-1","labels":["bug","urgent","nope"]}`)
		require.Equal(t, http.StatusOK, rec.Code)
		var got map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, "cyan", got["bug"])
		assert.Equal(t, "orange", got["urgent"])
		_, ok := got["nope"]
		assert.False(t, ok)
	})
	t.Run("deletes a label no ticket carries", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, "/api/tickets/labels/colors/urgent?project_id=p-1", "")
		require.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, http.StatusConflict, serve(t, h, http.MethodDelete, "/api/tickets/labels/colors/urgent?project_id=p-1", "").Code)
	})
}

func TestTicketsHandler_Refusals(t *testing.T) {
	failLabelColors := func(r *fakeRepo) { r.labelColorErr = errors.New("db down") }
	tests := []struct {
		name   string
		seed   func(repo *fakeRepo)
		method string
		path   string
		body   string
		want   int
	}{
		{"create without a project", nil, http.MethodPost, "/api/tickets", `{"title":"Fix"}`, http.StatusBadRequest},
		{"create without a title", nil, http.MethodPost, "/api/tickets", `{"title":" ","project_id":"p-1"}`, http.StatusBadRequest},
		{"create from a malformed body", nil, http.MethodPost, "/api/tickets", `{"title":`, http.StatusBadRequest},
		{"get a missing ticket", nil, http.MethodGet, "/api/tickets/nope", "", http.StatusNotFound},
		{"edit without a title", nil, http.MethodPatch, "/api/tickets/{id}", `{"title":" "}`, http.StatusBadRequest},
		{"edit from a malformed body", nil, http.MethodPatch, "/api/tickets/{id}", `{"title":`, http.StatusBadRequest},
		{"edit a missing ticket", nil, http.MethodPatch, "/api/tickets/nope", `{"title":"B"}`, http.StatusNotFound},
		{"move status without a status", nil, http.MethodPatch, "/api/tickets/{id}/status", `{}`, http.StatusBadRequest},
		{"move status to an unconfigured one", nil, http.MethodPatch, "/api/tickets/{id}/status", `{"status":"nope"}`, http.StatusBadRequest},
		{"move status of a missing ticket", nil, http.MethodPatch, "/api/tickets/nope/status", `{"status":"done"}`, http.StatusNotFound},
		{"set a negative position", nil, http.MethodPatch, "/api/tickets/{id}/position", `{"position":-1}`, http.StatusBadRequest},
		{"set a position from a malformed body", nil, http.MethodPatch, "/api/tickets/{id}/position", `{`, http.StatusBadRequest},
		{"set the position of a missing ticket", nil, http.MethodPatch, "/api/tickets/nope/position", `{"position":1}`, http.StatusNotFound},
		{"delete a missing ticket", nil, http.MethodDelete, "/api/tickets/nope", "", http.StatusNotFound},
		{"search without a query", nil, http.MethodGet, "/api/tickets/search", "", http.StatusBadRequest},
		{"link an incomplete branch", nil, http.MethodPost, "/api/tickets/{id}/branches", `{"owner":"","repo":"","branch":""}`, http.StatusBadRequest},
		{"link a pr from a malformed body", nil, http.MethodPost, "/api/tickets/{id}/prs", `{`, http.StatusBadRequest},
		{"unlink a pr without a number", nil, http.MethodDelete, "/api/tickets/{id}/prs?owner=acme&repo=app", "", http.StatusBadRequest},
		{"list the links of a missing ticket", nil, http.MethodGet, "/api/tickets/nope/links", "", http.StatusNotFound},
		{"set a type on a missing ticket", nil, http.MethodPatch, "/api/tickets/nope/type", `{"type_id":"ticket-type-task"}`, http.StatusNotFound},
		{"set a type without one", nil, http.MethodPatch, "/api/tickets/{id}/type", `{}`, http.StatusBadRequest},
		{"set a type from a malformed body", nil, http.MethodPatch, "/api/tickets/{id}/type", `{`, http.StatusBadRequest},
		{"add a label to a missing ticket", nil, http.MethodPost, "/api/tickets/nope/labels", `{"label":"bug"}`, http.StatusNotFound},
		{"add a blank label", nil, http.MethodPost, "/api/tickets/{id}/labels", `{"label":"  "}`, http.StatusBadRequest},
		{"add a label from a malformed body", nil, http.MethodPost, "/api/tickets/{id}/labels", `{`, http.StatusBadRequest},
		{"remove a label from a missing ticket", nil, http.MethodDelete, "/api/tickets/nope/labels/bug", "", http.StatusNotFound},
		{"list the labels of a missing ticket", nil, http.MethodGet, "/api/tickets/nope/labels", "", http.StatusNotFound},
		{"set a label color that is not one", nil, http.MethodPut, "/api/tickets/labels/bug/color", `{"project_id":"p-1","color":"magenta"}`, http.StatusBadRequest},
		{"set a label color from a malformed body", nil, http.MethodPut, "/api/tickets/labels/bug/color", `{`, http.StatusBadRequest},
		{"fetch label colors from a malformed body", nil, http.MethodPost, "/api/tickets/labels/colors", `{`, http.StatusBadRequest},
		{"set a label color when the repo fails", failLabelColors, http.MethodPut, "/api/tickets/labels/bug/color", `{"project_id":"p-1","color":"cyan"}`, http.StatusInternalServerError},
		{"fetch label colors when the repo fails", failLabelColors, http.MethodPost, "/api/tickets/labels/colors", `{"project_id":"p-1","labels":["bug"]}`, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			h := NewHandler(newTestService(repo)).Routes()
			created := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","project_id":"p-1"}`))
			if tt.seed != nil {
				tt.seed(repo)
			}
			rec := serve(t, h, tt.method, strings.ReplaceAll(tt.path, "{id}", created.ID), tt.body)
			assert.Equal(t, tt.want, rec.Code)
		})
	}
}

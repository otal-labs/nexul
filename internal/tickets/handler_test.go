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
		rec := serve(t, h, http.MethodPost, "/api/tickets", `{"title":"Fix","body":"b","doc_id":"doc-1","assignee":"onik97","project_id":"p-1"}`)
		require.Equal(t, http.StatusCreated, rec.Code)
		tk := decodeTicket(t, rec)
		assert.Equal(t, "Fix", tk.Title)
		assert.Equal(t, "doc-1", tk.DocID)
		assert.Equal(t, "p-1", tk.ProjectID)
		assert.Equal(t, StatusOpen, tk.Status)
	})
	t.Run("missing project is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/tickets", `{"title":"Fix"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("empty title is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/tickets", `{"title":" "}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("malformed body is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/tickets", `{"title":`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
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

func TestTicketsHandler_Get(t *testing.T) {
	h := newTicketsHandler()
	created := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","project_id":"p-1"}`))

	t.Run("gets a ticket by id", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/tickets/"+created.ID, "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "A", decodeTicket(t, rec).Title)
	})
	t.Run("missing ticket is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/tickets/nope", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
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
	t.Run("empty title is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID, `{"title":" "}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("malformed body is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID, `{"title":`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("missing ticket is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/nope", `{"title":"B"}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
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
	t.Run("missing status is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/status", `{}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("unconfigured status is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/status", `{"status":"nope"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("missing ticket is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/nope/status", `{"status":"done"}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
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
	t.Run("negative position is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/position", `{"position":-1}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("malformed body is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/position", `{`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("missing ticket is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/nope/position", `{"position":1}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestTicketsHandler_Delete(t *testing.T) {
	h := newTicketsHandler()
	created := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","project_id":"p-1"}`))

	t.Run("deletes a ticket", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, "/api/tickets/"+created.ID, "")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("missing ticket is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, "/api/tickets/again", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
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
	t.Run("missing query is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/tickets/search", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
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
	t.Run("incomplete branch link is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, base+"/branches", `{"owner":"","repo":"","branch":""}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
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
	t.Run("missing ticket is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/tickets/nope/links", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("malformed link body is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, base+"/prs", `{`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
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
	t.Run("missing type is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, base+"/type", `{}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("adds a label", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, base+"/labels", `{"label":"bug"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, decodeTicket(t, rec).Labels, "bug")
	})
	t.Run("empty label is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, base+"/labels", `{"label":"  "}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
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
	t.Run("invalid color is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPut, "/api/tickets/labels/bug/color", `{"project_id":"p-1","color":"magenta"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("malformed set-color body is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPut, "/api/tickets/labels/bug/color", `{`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
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
	t.Run("malformed batch-fetch body is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/tickets/labels/colors", `{`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("repo error on set is 500", func(t *testing.T) {
		repo := newFakeRepo()
		repo.labelColorErr = errors.New("db down")
		errHandler := NewHandler(newTestService(repo)).Routes()
		rec := serve(t, errHandler, http.MethodPut, "/api/tickets/labels/bug/color", `{"project_id":"p-1","color":"cyan"}`)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("repo error on batch fetch is 500", func(t *testing.T) {
		repo := newFakeRepo()
		repo.labelColorErr = errors.New("db down")
		errHandler := NewHandler(newTestService(repo)).Routes()
		rec := serve(t, errHandler, http.MethodPost, "/api/tickets/labels/colors", `{"project_id":"p-1","labels":["bug"]}`)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestTicketsHandler_TypeAndLabelErrors(t *testing.T) {
	h := newTicketsHandler()

	t.Run("set type on missing ticket is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/nope/type", `{"type_id":"ticket-type-task"}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("add label on missing ticket is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/tickets/nope/labels", `{"label":"bug"}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("remove label on missing ticket is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, "/api/tickets/nope/labels/bug", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("list labels on missing ticket is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/tickets/nope/labels", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("malformed set type body is 400", func(t *testing.T) {
		created := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","project_id":"p-1"}`))
		rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/type", `{`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("malformed label body is 400", func(t *testing.T) {
		created := decodeTicket(t, serve(t, h, http.MethodPost, "/api/tickets", `{"title":"A","project_id":"p-1"}`))
		rec := serve(t, h, http.MethodPost, "/api/tickets/"+created.ID+"/labels", `{`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

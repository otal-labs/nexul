package workspace

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/colors"
)

func decodeCategory(t *testing.T, rec *httptest.ResponseRecorder) Category {
	t.Helper()
	var c Category
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &c))
	return c
}

func TestHandler_Categories(t *testing.T) {
	t.Run("creates a category", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/categories", `{"project_id":"p-1","name":"Sprint 1"}`, "u-1")
		require.Equal(t, http.StatusCreated, rec.Code)
		c := decodeCategory(t, rec)
		assert.Equal(t, "Sprint 1", c.Name)
		assert.Equal(t, "p-1", c.ProjectID)
		assert.Equal(t, colors.Color(""), c.Color)
	})
	t.Run("creates a category with a color", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/categories", `{"project_id":"p-1","name":"Sprint 1","color":"cyan"}`, "u-1")
		require.Equal(t, http.StatusCreated, rec.Code)
		c := decodeCategory(t, rec)
		assert.Equal(t, colors.Cyan, c.Color)
	})
	t.Run("invalid color is 400", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/categories", `{"project_id":"p-1","name":"Sprint 1","color":"bogus"}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("create requires owner", func(t *testing.T) {
		h, _ := newTestHandler(t, false)
		rec := do(t, h.Routes(), http.MethodPost, "/api/categories", `{"project_id":"p-1","name":"Sprint 1"}`, "u-1")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("empty name is 400", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/categories", `{"project_id":"p-1","name":"  "}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("lists categories filtered by project", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		_ = repo
		h.svc.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Name: "Sprint 1"}
		h.svc.cats.(*fakeCategoryRepo).cats["c-2"] = &Category{ID: "c-2", ProjectID: "p-2", Name: "Sprint 1"}
		rec := do(t, h.Routes(), http.MethodGet, "/api/categories?project_id=p-1", "", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var cats []Category
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cats))
		require.Len(t, cats, 1)
		assert.Equal(t, "c-1", cats[0].ID)
	})
	t.Run("renames a category", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Name: "Sprint 1"}
		rec := do(t, h.Routes(), http.MethodPatch, "/api/categories/c-1", `{"name":"Sprint 2"}`, "u-1")
		require.Equal(t, http.StatusOK, rec.Code)
		c := decodeCategory(t, rec)
		assert.Equal(t, "Sprint 2", c.Name)
	})
	t.Run("renames a category with a color", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Name: "Sprint 1"}
		rec := do(t, h.Routes(), http.MethodPatch, "/api/categories/c-1", `{"name":"Sprint 2","color":"fuchsia"}`, "u-1")
		require.Equal(t, http.StatusOK, rec.Code)
		c := decodeCategory(t, rec)
		assert.Equal(t, colors.Fuchsia, c.Color)
	})
	t.Run("rename with invalid color is 400", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Name: "Sprint 1"}
		rec := do(t, h.Routes(), http.MethodPatch, "/api/categories/c-1", `{"name":"Sprint 2","color":"bogus"}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("deletes a category", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		rec := do(t, h.Routes(), http.MethodDelete, "/api/categories/c-1", "", "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("moves a ticket into a category", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/categories/c-1/tickets/t-1", "", "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("clears a ticket category", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodDelete, "/api/categories/tickets/t-1", "", "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("reorders categories", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		h.svc.cats.(*fakeCategoryRepo).cats["c-2"] = &Category{ID: "c-2", ProjectID: "p-1"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/categories/reorder", `{"project_id":"p-1","ids":["c-2","c-1"]}`, "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

func TestHandler_TicketTypes(t *testing.T) {
	t.Run("creates a ticket type", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/ticket-types", `{"project_id":"p-1","name":"bug"}`, "u-1")
		require.Equal(t, http.StatusCreated, rec.Code)
		var tt TicketType
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tt))
		assert.Equal(t, "bug", tt.Name)
		assert.Equal(t, colors.Color(""), tt.Color)
	})
	t.Run("creates a ticket type with a color", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/ticket-types", `{"project_id":"p-1","name":"bug","color":"cyan"}`, "u-1")
		require.Equal(t, http.StatusCreated, rec.Code)
		var tt TicketType
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tt))
		assert.Equal(t, colors.Cyan, tt.Color)
	})
	t.Run("invalid color is 400", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/ticket-types", `{"project_id":"p-1","name":"bug","color":"bogus"}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("lists ticket types", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.types.(*fakeTicketTypeRepo).types["tt-1"] = &TicketType{ID: "tt-1", ProjectID: "p-1", Name: "bug"}
		rec := do(t, h.Routes(), http.MethodGet, "/api/ticket-types?project_id=p-1", "", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var types []TicketType
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &types))
		require.Len(t, types, 1)
	})
	t.Run("deletes an unused type", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.types.(*fakeTicketTypeRepo).types["tt-1"] = &TicketType{ID: "tt-1"}
		rec := do(t, h.Routes(), http.MethodDelete, "/api/ticket-types/tt-1", "", "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("create requires owner", func(t *testing.T) {
		h, _ := newTestHandler(t, false)
		rec := do(t, h.Routes(), http.MethodPost, "/api/ticket-types", `{"project_id":"p-1","name":"bug"}`, "u-1")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandler_Statuses(t *testing.T) {
	t.Run("creates a status", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/statuses", `{"project_id":"p-1","name":"Blocked","kind":"progress"}`, "u-1")
		require.Equal(t, http.StatusCreated, rec.Code)
		var st Status
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &st))
		assert.Equal(t, "Blocked", st.Name)
		assert.Equal(t, StatusKindProgress, st.Kind)
	})
	t.Run("invalid kind is 400", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/statuses", `{"project_id":"p-1","name":"Blocked","kind":"bogus"}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("lists statuses", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.statuses.(*fakeStatusRepo).statuses["s-1"] = &Status{ID: "s-1", ProjectID: "p-1", Name: "Blocked", Kind: StatusKindProgress}
		rec := do(t, h.Routes(), http.MethodGet, "/api/statuses?project_id=p-1", "", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var statuses []Status
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statuses))
		require.Len(t, statuses, 1)
	})
	t.Run("renames a status", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.statuses.(*fakeStatusRepo).statuses["s-1"] = &Status{ID: "s-1", Name: "Blocked", Kind: StatusKindProgress}
		rec := do(t, h.Routes(), http.MethodPatch, "/api/statuses/s-1", `{"name":"In review","kind":"done"}`, "u-1")
		require.Equal(t, http.StatusOK, rec.Code)
		var st Status
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &st))
		assert.Equal(t, "In review", st.Name)
		assert.Equal(t, StatusKindDone, st.Kind)
	})
	t.Run("create requires owner", func(t *testing.T) {
		h, _ := newTestHandler(t, false)
		rec := do(t, h.Routes(), http.MethodPost, "/api/statuses", `{"project_id":"p-1","name":"Blocked","kind":"progress"}`, "u-1")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandler_CategoryDetailRoutes(t *testing.T) {
	t.Run("gets a category", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Name: "Sprint 1"}
		rec := do(t, h.Routes(), http.MethodGet, "/api/categories/c-1", "", "")
		require.Equal(t, http.StatusOK, rec.Code)
		c := decodeCategory(t, rec)
		assert.Equal(t, "Sprint 1", c.Name)
	})
	t.Run("missing category is 404", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodGet, "/api/categories/nope", "", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("rename missing category is 404", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPatch, "/api/categories/nope", `{"name":"Sprint 2"}`, "u-1")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("delete missing category is 404", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodDelete, "/api/categories/nope", "", "u-1")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("reorder with a category not in the project is 400", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/categories/reorder", `{"project_id":"p-1","ids":["c-2"]}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("move ticket to missing category is 404", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/categories/nope/tickets/t-1", "", "u-1")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("create category malformed body is 400", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/categories", `{`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("list categories repo error is 500", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		_ = repo
		h.svc.cats.(*fakeCategoryRepo).listErr = errors.New("db down")
		rec := do(t, h.Routes(), http.MethodGet, "/api/categories", "", "")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestHandler_TicketTypeDetailRoutes(t *testing.T) {
	t.Run("gets a ticket type", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.types.(*fakeTicketTypeRepo).types["tt-1"] = &TicketType{ID: "tt-1", Name: "bug"}
		rec := do(t, h.Routes(), http.MethodGet, "/api/ticket-types/tt-1", "", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var tt TicketType
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tt))
		assert.Equal(t, "bug", tt.Name)
	})
	t.Run("missing ticket type is 404", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodGet, "/api/ticket-types/nope", "", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("renames a ticket type", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.types.(*fakeTicketTypeRepo).types["tt-1"] = &TicketType{ID: "tt-1", Name: "bug"}
		rec := do(t, h.Routes(), http.MethodPatch, "/api/ticket-types/tt-1", `{"name":"defect"}`, "u-1")
		require.Equal(t, http.StatusOK, rec.Code)
		var tt TicketType
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tt))
		assert.Equal(t, "defect", tt.Name)
	})
	t.Run("renames a ticket type with a color", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.types.(*fakeTicketTypeRepo).types["tt-1"] = &TicketType{ID: "tt-1", Name: "bug"}
		rec := do(t, h.Routes(), http.MethodPatch, "/api/ticket-types/tt-1", `{"name":"bug","color":"fuchsia"}`, "u-1")
		require.Equal(t, http.StatusOK, rec.Code)
		var tt TicketType
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tt))
		assert.Equal(t, colors.Fuchsia, tt.Color)
	})
	t.Run("rename with invalid color is 400", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.types.(*fakeTicketTypeRepo).types["tt-1"] = &TicketType{ID: "tt-1", Name: "bug"}
		rec := do(t, h.Routes(), http.MethodPatch, "/api/ticket-types/tt-1", `{"name":"bug","color":"bogus"}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("rename missing ticket type is 404", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPatch, "/api/ticket-types/nope", `{"name":"defect"}`, "u-1")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("delete missing ticket type is 404", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodDelete, "/api/ticket-types/nope", "", "u-1")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("reorders ticket types", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.types.(*fakeTicketTypeRepo).types["tt-1"] = &TicketType{ID: "tt-1", ProjectID: "p-1"}
		h.svc.types.(*fakeTicketTypeRepo).types["tt-2"] = &TicketType{ID: "tt-2", ProjectID: "p-1"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/ticket-types/reorder", `{"project_id":"p-1","ids":["tt-2","tt-1"]}`, "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("reorder with duplicate ids is 400", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.types.(*fakeTicketTypeRepo).types["tt-1"] = &TicketType{ID: "tt-1", ProjectID: "p-1"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/ticket-types/reorder", `{"project_id":"p-1","ids":["tt-1","tt-1"]}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_StatusDetailRoutes(t *testing.T) {
	t.Run("gets a status", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.statuses.(*fakeStatusRepo).statuses["s-1"] = &Status{ID: "s-1", Name: "Blocked", Kind: StatusKindProgress}
		rec := do(t, h.Routes(), http.MethodGet, "/api/statuses/s-1", "", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var st Status
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &st))
		assert.Equal(t, "Blocked", st.Name)
	})
	t.Run("missing status is 404", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodGet, "/api/statuses/nope", "", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("rename missing status is 404", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPatch, "/api/statuses/nope", `{"name":"Blocked","kind":"progress"}`, "u-1")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("delete missing status is 404", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodDelete, "/api/statuses/nope", "", "u-1")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("reorders statuses", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.statuses.(*fakeStatusRepo).statuses["s-1"] = &Status{ID: "s-1", ProjectID: "p-1"}
		h.svc.statuses.(*fakeStatusRepo).statuses["s-2"] = &Status{ID: "s-2", ProjectID: "p-1"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/statuses/reorder", `{"project_id":"p-1","ids":["s-2","s-1"]}`, "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("reorder with duplicate ids is 400", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.statuses.(*fakeStatusRepo).statuses["s-1"] = &Status{ID: "s-1", ProjectID: "p-1"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/statuses/reorder", `{"project_id":"p-1","ids":["s-1","s-1"]}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("delete in-use status is 409", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.statuses.(*fakeStatusRepo).statuses["s-1"] = &Status{ID: "s-1"}
		h.svc.statuses.(*fakeStatusRepo).count = 1
		rec := do(t, h.Routes(), http.MethodDelete, "/api/statuses/s-1", "", "u-1")
		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

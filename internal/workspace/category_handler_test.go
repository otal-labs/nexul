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
	t.Run("sets a body template", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.types.(*fakeTicketTypeRepo).types["tt-1"] = &TicketType{ID: "tt-1", Name: "bug"}
		rec := do(t, h.Routes(), http.MethodPut, "/api/ticket-types/tt-1/template", `{"body_template":"## Why\n\n"}`, "u-1")
		require.Equal(t, http.StatusOK, rec.Code)
		var tt TicketType
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tt))
		assert.Equal(t, "## Why\n\n", tt.BodyTemplate)
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
	t.Run("reorders ticket types", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.types.(*fakeTicketTypeRepo).types["tt-1"] = &TicketType{ID: "tt-1", ProjectID: "p-1"}
		h.svc.types.(*fakeTicketTypeRepo).types["tt-2"] = &TicketType{ID: "tt-2", ProjectID: "p-1"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/ticket-types/reorder", `{"project_id":"p-1","ids":["tt-2","tt-1"]}`, "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
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
	t.Run("reorders statuses", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		h.svc.statuses.(*fakeStatusRepo).statuses["s-1"] = &Status{ID: "s-1", ProjectID: "p-1"}
		h.svc.statuses.(*fakeStatusRepo).statuses["s-2"] = &Status{ID: "s-2", ProjectID: "p-1"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/statuses/reorder", `{"project_id":"p-1","ids":["s-2","s-1"]}`, "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

func TestHandler_CategoryTypeAndStatusRoutes_Refusals(t *testing.T) {
	seedCategory := func(h *Handler) {
		h.svc.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Name: "Sprint 1"}
	}
	seedType := func(h *Handler) {
		h.svc.types.(*fakeTicketTypeRepo).types["tt-1"] = &TicketType{ID: "tt-1", ProjectID: "p-1", Name: "bug"}
	}
	seedStatus := func(h *Handler) {
		h.svc.statuses.(*fakeStatusRepo).statuses["s-1"] = &Status{ID: "s-1", ProjectID: "p-1"}
	}
	tests := []struct {
		name   string
		owner  bool
		seed   func(h *Handler)
		method string
		path   string
		body   string
		want   int
	}{
		{"create a category with a bogus color", true, nil, http.MethodPost, "/api/categories", `{"project_id":"p-1","name":"Sprint 1","color":"bogus"}`, http.StatusBadRequest},
		{"create a category as a non-owner", false, nil, http.MethodPost, "/api/categories", `{"project_id":"p-1","name":"Sprint 1"}`, http.StatusForbidden},
		{"create a category with a blank name", true, nil, http.MethodPost, "/api/categories", `{"project_id":"p-1","name":"  "}`, http.StatusBadRequest},
		{"create a category from a malformed body", true, nil, http.MethodPost, "/api/categories", `{`, http.StatusBadRequest},
		{"rename a category with a bogus color", true, seedCategory, http.MethodPatch, "/api/categories/c-1", `{"name":"Sprint 2","color":"bogus"}`, http.StatusBadRequest},
		{"get a missing category", true, nil, http.MethodGet, "/api/categories/nope", "", http.StatusNotFound},
		{"rename a missing category", true, nil, http.MethodPatch, "/api/categories/nope", `{"name":"Sprint 2"}`, http.StatusNotFound},
		{"delete a missing category", true, nil, http.MethodDelete, "/api/categories/nope", "", http.StatusNotFound},
		{"reorder with a category outside the project", true, seedCategory, http.MethodPost, "/api/categories/reorder", `{"project_id":"p-1","ids":["c-2"]}`, http.StatusBadRequest},
		{"move a ticket into a missing category", true, nil, http.MethodPost, "/api/categories/nope/tickets/t-1", "", http.StatusNotFound},
		{"list categories when the repo fails", true, func(h *Handler) { h.svc.cats.(*fakeCategoryRepo).listErr = errors.New("db down") }, http.MethodGet, "/api/categories", "", http.StatusInternalServerError},

		{"create a ticket type with a bogus color", true, nil, http.MethodPost, "/api/ticket-types", `{"project_id":"p-1","name":"bug","color":"bogus"}`, http.StatusBadRequest},
		{"create a ticket type as a non-owner", false, nil, http.MethodPost, "/api/ticket-types", `{"project_id":"p-1","name":"bug"}`, http.StatusForbidden},
		{"set a template from a malformed body", true, nil, http.MethodPut, "/api/ticket-types/tt-1/template", `{`, http.StatusBadRequest},
		{"set a template on a missing type", true, nil, http.MethodPut, "/api/ticket-types/nope/template", `{"body_template":""}`, http.StatusNotFound},
		{"get a missing ticket type", true, nil, http.MethodGet, "/api/ticket-types/nope", "", http.StatusNotFound},
		{"rename a ticket type with a bogus color", true, seedType, http.MethodPatch, "/api/ticket-types/tt-1", `{"name":"bug","color":"bogus"}`, http.StatusBadRequest},
		{"rename a missing ticket type", true, nil, http.MethodPatch, "/api/ticket-types/nope", `{"name":"defect"}`, http.StatusNotFound},
		{"delete a missing ticket type", true, nil, http.MethodDelete, "/api/ticket-types/nope", "", http.StatusNotFound},
		{"reorder ticket types with duplicate ids", true, seedType, http.MethodPost, "/api/ticket-types/reorder", `{"project_id":"p-1","ids":["tt-1","tt-1"]}`, http.StatusBadRequest},

		{"create a status with a bogus kind", true, nil, http.MethodPost, "/api/statuses", `{"project_id":"p-1","name":"Blocked","kind":"bogus"}`, http.StatusBadRequest},
		{"create a status as a non-owner", false, nil, http.MethodPost, "/api/statuses", `{"project_id":"p-1","name":"Blocked","kind":"progress"}`, http.StatusForbidden},
		{"get a missing status", true, nil, http.MethodGet, "/api/statuses/nope", "", http.StatusNotFound},
		{"rename a missing status", true, nil, http.MethodPatch, "/api/statuses/nope", `{"name":"Blocked","kind":"progress"}`, http.StatusNotFound},
		{"delete a missing status", true, nil, http.MethodDelete, "/api/statuses/nope", "", http.StatusNotFound},
		{"reorder statuses with duplicate ids", true, seedStatus, http.MethodPost, "/api/statuses/reorder", `{"project_id":"p-1","ids":["s-1","s-1"]}`, http.StatusBadRequest},
		{"delete a status in use", true, func(h *Handler) {
			seedStatus(h)
			h.svc.statuses.(*fakeStatusRepo).count = 1
		}, http.MethodDelete, "/api/statuses/s-1", "", http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler(t, tt.owner)
			if tt.seed != nil {
				tt.seed(h)
			}
			user := "u-1"
			if tt.method == http.MethodGet {
				user = ""
			}
			rec := do(t, h.Routes(), tt.method, tt.path, tt.body, user)
			assert.Equal(t, tt.want, rec.Code)
		})
	}
}

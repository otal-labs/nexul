package plays

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
)

func newTestHandler() (*Handler, *fakeRepo) {
	repo := newFakeRepo()
	return NewHandler(newTestService(repo, allowAll("u-owner"))), repo
}

func do(t *testing.T, h http.Handler, method, path, body, userID string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if userID != "" {
		r = r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: userID}))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestHandler_Create(t *testing.T) {
	h, _ := newTestHandler()
	rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/plays",
		`{"label":"Fix with AI","type":"ticket","show_when_stage":"progress"}`, "u-owner")
	require.Equal(t, http.StatusCreated, rec.Code)
	var p Play
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
	assert.Equal(t, "Fix with AI", p.Label)
	assert.Equal(t, "ws-1", p.WorkspaceID)
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	h, _ := newTestHandler()
	rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/plays", `{`, "u-owner")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Create_TicketWithoutStageIsInvalid(t *testing.T) {
	h, _ := newTestHandler()
	rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/plays", `{"label":"Fix","type":"ticket"}`, "u-owner")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_List(t *testing.T) {
	h, _ := newTestHandler()
	do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/plays", `{"label":"Doc","type":"doc"}`, "u-owner")

	rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/plays", "", "u-owner")
	require.Equal(t, http.StatusOK, rec.Code)
	var list []Play
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	require.Len(t, list, 1)
}

func TestHandler_GetUpdateDelete(t *testing.T) {
	h, _ := newTestHandler()
	rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/plays", `{"label":"Doc","type":"doc"}`, "u-owner")
	var created Play
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	t.Run("get", func(t *testing.T) {
		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/plays/"+created.ID, "", "u-owner")
		assert.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("get missing is not found", func(t *testing.T) {
		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/plays/missing", "", "u-owner")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("update", func(t *testing.T) {
		rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/ws-1/plays/"+created.ID,
			`{"label":"Doc renamed","enabled":true}`, "u-owner")
		require.Equal(t, http.StatusOK, rec.Code)
		var updated Play
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updated))
		assert.Equal(t, "Doc renamed", updated.Label)
	})
	t.Run("delete", func(t *testing.T) {
		rec := do(t, h.Routes(), http.MethodDelete, "/api/workspaces/ws-1/plays/"+created.ID, "", "u-owner")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

func TestHandler_WithoutPlaysWrite_IsForbidden(t *testing.T) {
	h := NewHandler(newTestService(newFakeRepo(), newFakePerm(nil)))
	rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/plays", `{"label":"Doc","type":"doc"}`, "alice")
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_List_RepoError(t *testing.T) {
	repo := newFakeRepo()
	repo.listErr = assert.AnError
	h := NewHandler(newTestService(repo, allowAll("u-owner")))
	rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/plays", "", "u-owner")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandler_Update_InvalidBody(t *testing.T) {
	h, _ := newTestHandler()
	rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/plays", `{"label":"Doc","type":"doc"}`, "u-owner")
	var created Play
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = do(t, h.Routes(), http.MethodPatch, "/api/workspaces/ws-1/plays/"+created.ID, `{`, "u-owner")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Delete_Missing_IsNotFound(t *testing.T) {
	h, _ := newTestHandler()
	rec := do(t, h.Routes(), http.MethodDelete, "/api/workspaces/ws-1/plays/missing", "", "u-owner")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_ListApplicable(t *testing.T) {
	repo := newFakeRepo()
	perm := runnerPerm("alice")
	h := NewHandler(newTestService(repo, perm))
	rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/plays",
		`{"label":"Fix","type":"ticket","enabled":true,"show_when_stage":"progress"}`, "owner")
	var fix Play
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &fix))

	t.Run("lists the play for the permitted stage", func(t *testing.T) {
		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/plays/applicable?type=ticket&stage=progress&project_id=proj-1", "", "alice")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []Play
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list, 1)
		assert.Equal(t, fix.ID, list[0].ID)
	})
	t.Run("omits it once the play denies alice", func(t *testing.T) {
		perm.deny("alice", fix.ID)
		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/plays/applicable?type=ticket&stage=progress&project_id=proj-1", "", "alice")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []Play
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		assert.Empty(t, list)
	})
	t.Run("missing type is 400", func(t *testing.T) {
		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/plays/applicable", "", "alice")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

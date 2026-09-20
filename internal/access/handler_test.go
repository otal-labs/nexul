package access

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func serveAccess(t *testing.T, h http.Handler, actor identity.Actor, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req = req.WithContext(identity.WithActor(req.Context(), actor))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func newAccessHarness() (*Service, *fakeRepo, *fakeUsers) {
	users := newFakeUsers(
		&User{ID: "owner", Login: "owner", CanCreateWorkspace: true},
		&User{ID: "alice", Login: "alice"},
	)
	repo := newFakeRepo()
	return newService(repo, users), repo, users
}

func TestHandler_ListGrants(t *testing.T) {
	// Ticket 11: no instance-wide bypass, so "owner" here proves access through a real permissions:write overwrite, same as any other actor.
	svc, repo, _ := newAccessHarness()
	setOverwrite(repo, resourceTypeDoc, "doc-1", "owner", permissions.SetOf(permissions.PermissionsWrite))
	setOverwrite(repo, resourceTypeDoc, "doc-1", "alice", permissions.SetOf(permissions.DocsRead))
	h := NewHandler(svc).Routes()

	t.Run("owner lists grants", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "owner", CanCreateWorkspace: true}, http.MethodGet, "/api/permissions?doc_id=doc-1", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Grants []*Overwrite `json:"grants"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Grants, 2)
		ids := []string{body.Grants[0].UserID, body.Grants[1].UserID}
		assert.ElementsMatch(t, []string{"owner", "alice"}, ids)
	})
	t.Run("missing doc_id is 400", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "owner", CanCreateWorkspace: true}, http.MethodGet, "/api/permissions", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("non-manager is 403", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "alice"}, http.MethodGet, "/api/permissions?doc_id=doc-1", "")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("no actor is 401", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{}, http.MethodGet, "/api/permissions?doc_id=doc-1", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestHandler_ListGrants_Play(t *testing.T) {
	svc, repo, _ := newAccessHarness()
	setOverwrite(repo, resourceTypePlay, "play-1", "owner", permissions.SetOf(permissions.PlaysWrite))
	require.NoError(t, repo.Set(context.Background(), resourceTypePlay, "play-1", "alice", nil, permissions.SetOf(permissions.PlaysRun)))
	h := NewHandler(svc).Routes()

	t.Run("plays:write holder lists a play's exclusions", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "owner"}, http.MethodGet, "/api/permissions?resource_type=play&resource_id=play-1", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Grants []*Overwrite `json:"grants"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Grants, 2)
		var alice *Overwrite
		for _, g := range body.Grants {
			if g.UserID == "alice" {
				alice = g
			}
		}
		require.NotNil(t, alice, "alice's exclusion row is in the list")
		assert.Equal(t, []string{"plays:run"}, actionStrings(alice.Deny))
	})
	t.Run("missing resource_id is 400", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "owner"}, http.MethodGet, "/api/permissions?resource_type=play", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("non-manager is 403", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "alice"}, http.MethodGet, "/api/permissions?resource_type=play&resource_id=play-1", "")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func actionStrings(s permissions.Set) []string {
	out := make([]string, 0, len(s))
	for _, a := range s {
		out = append(out, string(a))
	}
	return out
}

func TestHandler_ListUsers(t *testing.T) {
	svc, _, _ := newAccessHarness()
	h := NewHandler(svc).Routes()

	t.Run("owner lists users", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "owner", CanCreateWorkspace: true}, http.MethodGet, "/api/permissions/users", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Users []*User `json:"users"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Len(t, body.Users, 2)
	})
	t.Run("non-owner denied", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "alice"}, http.MethodGet, "/api/permissions/users", "")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandler_Catalog_ReturnsTheWholeGridInOrder(t *testing.T) {
	svc, _, _ := newAccessHarness()
	rec := serveAccess(t, NewHandler(svc).Routes(), identity.Actor{ID: "alice"}, http.MethodGet, "/api/permissions/catalog", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Permissions []permissions.Info `json:"permissions"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, permissions.Catalog(), body.Permissions)
	assert.Equal(t, permissions.Info{Value: "docs:read", Label: "Read docs", Domain: "docs", Action: "read"}, body.Permissions[0])
}

func TestHandler_SetGrants(t *testing.T) {
	// Ticket 11: same reasoning as TestHandler_ListGrants — no instance-wide bypass, "owner" proves access through a real permissions:write overwrite on the doc it's granting on.
	svc, repo, _ := newAccessHarness()
	setOverwrite(repo, resourceTypeDoc, "doc-1", "owner", permissions.SetOf(permissions.PermissionsWrite))
	h := NewHandler(svc).Routes()

	t.Run("owner bulk applies", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "owner", CanCreateWorkspace: true}, http.MethodPut, "/api/permissions",
			`{"doc_ids":["doc-1"],"user_ids":["alice"],"actions":["docs:read","docs:write"],"grant":true}`)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		g, err := repo.Get(context.Background(), resourceTypeDoc, "doc-1", "alice")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(permissions.DocsRead, permissions.DocsWrite), g.Allow)
	})
	t.Run("unknown action is 400", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "owner", CanCreateWorkspace: true}, http.MethodPut, "/api/permissions",
			`{"doc_ids":["doc-1"],"user_ids":["alice"],"actions":["comment"],"grant":true}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("non-manager is 403", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "alice"}, http.MethodPut, "/api/permissions",
			`{"doc_ids":["doc-1"],"user_ids":["bob"],"actions":["docs:read"],"grant":true}`)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("malformed body is 400", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "owner", CanCreateWorkspace: true}, http.MethodPut, "/api/permissions", `{"doc_ids":`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_SetGrants_Play(t *testing.T) {
	svc, repo, _ := newAccessHarness()
	setOverwrite(repo, resourceTypePlay, "play-1", "owner", permissions.SetOf(permissions.PlaysWrite))
	h := NewHandler(svc).Routes()

	t.Run("plays:write holder denies a user's plays:run", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "owner"}, http.MethodPut, "/api/permissions",
			`{"resource_type":"play","resource_ids":["play-1"],"user_ids":["alice"],"actions":["plays:run"],"grant":false}`)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		g, err := repo.Get(context.Background(), resourceTypePlay, "play-1", "alice")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(permissions.PlaysRun), g.Deny)
	})
	t.Run("an action other than plays:run is 400", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "owner"}, http.MethodPut, "/api/permissions",
			`{"resource_type":"play","resource_ids":["play-1"],"user_ids":["alice"],"actions":["plays:write"],"grant":false}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("non-manager is 403", func(t *testing.T) {
		rec := serveAccess(t, h, identity.Actor{ID: "alice"}, http.MethodPut, "/api/permissions",
			`{"resource_type":"play","resource_ids":["play-1"],"user_ids":["bob"],"actions":["plays:run"],"grant":false}`)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

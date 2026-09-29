package tenancy

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPeopleFixture: lewis and carol share ws-1; bob is in ws-2 only.
func newPeopleFixture(t *testing.T) (*Handler, *Service, *fakeAccountGate) {
	t.Helper()
	repo := newFakeRepo()
	repo.workspaces["ws-1"] = &Workspace{ID: "ws-1", Name: "Acme"}
	repo.workspaces["ws-2"] = &Workspace{ID: "ws-2", Name: "Beta"}
	for _, m := range []*Member{{UserID: "lewis", WorkspaceID: "ws-1", RoleID: "r"}, {UserID: "carol", WorkspaceID: "ws-1", RoleID: "r"}, {UserID: "bob", WorkspaceID: "ws-2", RoleID: "r"}} {
		require.NoError(t, repo.AddMember(t.Context(), m))
	}
	accounts := newFakeAccountGate()
	accounts.accounts["lewis"] = &TeamAccount{ID: "lewis", Login: "LewisWelch94", Name: "Lewis Welch", DisplayName: "Lewis", AvatarURL: "https://avatars.example/lewis", Status: "active", AvatarOverride: "data:image/svg+xml;base64,PHN2Zz48L3N2Zz4="}
	accounts.accounts["carol"] = &TeamAccount{ID: "carol", Login: "carol", Status: "disabled"}
	accounts.accounts["bob"] = &TeamAccount{ID: "bob", Login: "bob", Status: "active"}
	perm := newFakePermissionGate()
	perm.allow = false
	svc := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, perm, newFakeRoleNameGate(), newFakeWorkspacePermissionGate(), newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{}, accounts)
	return NewHandler(svc), svc, accounts
}

func TestHandler_ListPeople(t *testing.T) {
	t.Run("a member gets id, login, display name and picture, and nothing else", func(t *testing.T) {
		h, _, _ := newPeopleFixture(t)
		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/people", "", "carol")
		require.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			People []map[string]any `json:"people"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.People, 2)
		for _, p := range body.People {
			keys := make([]string, 0, len(p))
			for k := range p {
				keys = append(keys, k)
			}
			assert.ElementsMatch(t, []string{"user_id", "login", "display_name", "avatar_url"}, keys)
		}
	})
	t.Run("a non-member is refused", func(t *testing.T) {
		h, _, _ := newPeopleFixture(t)
		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/people", "", "bob")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestListPeople_AvatarURLFollowsTheUploadedPicture(t *testing.T) {
	_, svc, accounts := newPeopleFixture(t)
	avatarOf := func() string {
		people, err := svc.ListPeople(t.Context(), "carol", "ws-1")
		require.NoError(t, err)
		for _, p := range people {
			if p.UserID == "lewis" {
				return p.AvatarURL
			}
		}
		require.FailNow(t, "lewis missing")
		return ""
	}

	first := avatarOf()
	accounts.accounts["lewis"].AvatarOverride = "data:image/png;base64,aGVsbG8="
	second := avatarOf()
	accounts.accounts["lewis"].AvatarOverride = ""
	removed := avatarOf()

	assert.Contains(t, first, "/api/people/lewis/avatar?v=")
	assert.NotEqual(t, first, second, "a new upload must be a new URL, or every cache keeps the old picture")
	assert.Equal(t, "https://avatars.example/lewis", removed, "removing the upload falls back to the sign-in account's picture")
}

func TestHandler_Avatar(t *testing.T) {
	t.Run("serves a co-member's picture, never as markup", func(t *testing.T) {
		h, _, _ := newPeopleFixture(t)
		rec := do(t, h.PeopleRoutes(), http.MethodGet, "/api/people/lewis/avatar", "", "carol")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"), "an SVG can carry script, so it is never served as an image")
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "<svg></svg>", rec.Body.String())
	})
	t.Run("refuses someone who shares no workspace", func(t *testing.T) {
		h, _, _ := newPeopleFixture(t)
		rec := do(t, h.PeopleRoutes(), http.MethodGet, "/api/people/lewis/avatar", "", "bob")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("a person with no upload has nothing to serve", func(t *testing.T) {
		h, _, _ := newPeopleFixture(t)
		rec := do(t, h.PeopleRoutes(), http.MethodGet, "/api/people/carol/avatar", "", "lewis")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

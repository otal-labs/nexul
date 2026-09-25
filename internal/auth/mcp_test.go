package auth

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// newAccountsHarness registers an instance admin ("admin") and a member ("member").
func newAccountsHarness(t *testing.T) (*Service, *fakeUserStore) {
	t.Helper()
	s, users, _, _ := newTestHarness(&fakeGitHub{})
	_, _, err := users.UpsertUser(t.Context(), &User{ID: "admin", Provider: ProviderGitHub, ProviderUserID: "1", Login: "onik97", CanCreateWorkspace: true})
	require.NoError(t, err)
	_, _, err = users.UpsertUser(t.Context(), &User{ID: "member", Provider: ProviderGitHub, ProviderUserID: "2", Login: "member", Name: "Mem Ber"})
	require.NoError(t, err)
	return s, users
}

func callAccountTool(t *testing.T, s *Service, actor, name, args string) (any, error) {
	t.Helper()
	ctx := identity.WithActor(t.Context(), identity.Actor{ID: actor})
	for _, tool := range MCPTools(s) {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not registered", name)
	return nil, nil
}

func TestAccountTools_Surface(t *testing.T) {
	t.Parallel()
	s, _ := newAccountsHarness(t)
	var names []string
	for _, tool := range MCPTools(s) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title)
	}
	assert.Equal(t, []string{"account_get", "account_list", "account_update", "account_delete"}, names)
}

func TestAccountTools_ErrorPaths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		actor   string
		tool    string
		args    string
		wantErr error
	}{
		{"get with a numeric id", "admin", "account_get", `{"id": 1}`, apperrs.ErrInvalid},
		{"update without a status", "admin", "account_update", `{"id": "member"}`, apperrs.ErrInvalid},
		{"update to removed", "admin", "account_update", `{"id": "member", "status": "removed"}`, apperrs.ErrInvalid},
		{"delete without an id", "admin", "account_delete", `{}`, apperrs.ErrInvalid},
		{"get without a caller", "", "account_get", `{}`, apperrs.ErrUnauthorized},
		{"get a caller that no longer exists", "ghost", "account_get", `{}`, apperrs.ErrNotFound},
		{"get an unknown account", "admin", "account_get", `{"id": "ghost"}`, apperrs.ErrNotFound},
		{"update an unknown account", "admin", "account_update", `{"id": "ghost", "status": "disabled"}`, apperrs.ErrNotFound},
		{"delete an unknown account", "admin", "account_delete", `{"id": "ghost"}`, apperrs.ErrNotFound},
		{"member gets another account", "member", "account_get", `{"id": "admin"}`, apperrs.ErrForbidden},
		{"member lists accounts", "member", "account_list", `{}`, apperrs.ErrForbidden},
		{"member disables an account", "member", "account_update", `{"id": "admin", "status": "disabled"}`, apperrs.ErrForbidden},
		{"member removes an account", "member", "account_delete", `{"id": "admin"}`, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, users := newAccountsHarness(t)
			_, err := callAccountTool(t, s, tt.actor, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.wantErr)
			member, err := users.GetUserByID(t.Context(), "member")
			require.NoError(t, err)
			assert.Equal(t, AccountActive, member.AccountStatus, "a refused call changes nothing")
		})
	}
}

func TestAccountGet_WithoutIDIsTheCaller(t *testing.T) {
	t.Parallel()
	s, _ := newAccountsHarness(t)

	got, err := callAccountTool(t, s, "member", "account_get", `{}`)
	require.NoError(t, err)
	assert.Equal(t, accountResult{ID: "member", Login: "member", Name: "Mem Ber", Provider: ProviderGitHub, Status: AccountActive}, got)

	got, err = callAccountTool(t, s, "member", "account_get", `{"id": "member"}`)
	require.NoError(t, err)
	assert.Equal(t, "member", got.(accountResult).ID, "a member may read their own account by id")

	got, err = callAccountTool(t, s, "admin", "account_get", `{"id": "member"}`)
	require.NoError(t, err)
	assert.Equal(t, "member", got.(accountResult).Login)

	_, err = s.UpdateProfileOverride(t.Context(), "member", "Mem", "")
	require.NoError(t, err)
	got, err = callAccountTool(t, s, "member", "account_get", `{}`)
	require.NoError(t, err)
	assert.Equal(t, "Mem", got.(accountResult).DisplayName, "the chosen display name sits beside the provider's name")
}

func TestAccountList_PagesEveryAccount(t *testing.T) {
	t.Parallel()
	s, _ := newAccountsHarness(t)
	got, err := callAccountTool(t, s, "admin", "account_list", `{"limit": 1}`)
	require.NoError(t, err)
	page := got.(mcptool.Page[accountResult])
	assert.Len(t, page.Items, 1)
	assert.Equal(t, 2, page.Total)
	assert.True(t, page.HasMore)
}

func TestAccountUpdate_ActivePicksReactivateOrRestore(t *testing.T) {
	t.Parallel()
	s, users := newAccountsHarness(t)
	status := func(args string) AccountStatus {
		t.Helper()
		got, err := callAccountTool(t, s, "admin", "account_update", args)
		require.NoError(t, err)
		return got.(accountResult).Status
	}

	assert.Equal(t, AccountDisabled, status(`{"id": "member", "status": "disabled"}`))
	assert.Equal(t, AccountActive, status(`{"id": "member", "status": "active"}`), "a disabled account is reactivated")
	assert.Equal(t, AccountActive, status(`{"id": "member", "status": "active"}`), "an active account stays active")

	got, err := callAccountTool(t, s, "admin", "account_delete", `{"id": "member"}`)
	require.NoError(t, err)
	assert.Equal(t, mcptool.Gone("member"), got)
	removed, err := users.GetUserByID(t.Context(), "member")
	require.NoError(t, err)
	assert.Equal(t, AccountRemoved, removed.AccountStatus)

	assert.Equal(t, AccountActive, status(`{"id": "member", "status": "active"}`), "a removed account is restored")
}

func TestHandler_UpdateAccountStatus_UsesTheSameRule(t *testing.T) {
	t.Parallel()
	s, users := newAccountsHarness(t)
	token, err := s.Sign("admin")
	require.NoError(t, err)
	routes := s.RequireAuth(NewHandler(s).ProtectedRoutes())
	patch := func(id, body string) int {
		return doRequest(routes, http.MethodPatch, "/api/auth/accounts/"+id, "Bearer "+token, body).Code
	}

	require.NoError(t, s.RemoveAccount(t.Context(), "admin", "member"))
	assert.Equal(t, http.StatusNoContent, patch("member", `{"status":"active"}`))
	restored, err := users.GetUserByID(t.Context(), "member")
	require.NoError(t, err)
	assert.Equal(t, AccountActive, restored.AccountStatus)
	assert.Equal(t, http.StatusBadRequest, patch("member", `{"status":"removed"}`))
	assert.Equal(t, http.StatusNotFound, patch("ghost", `{"status":"active"}`))
}

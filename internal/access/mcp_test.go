package access

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// newGrantsHarness gives "owner" the manage bit on doc-1 and play-1, and nothing to "alice".
func newGrantsHarness(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	svc, repo, _ := newAccessHarness()
	setOverwrite(repo, resourceTypeDoc, "doc-1", "owner", permissions.SetOf(permissions.PermissionsWrite))
	setOverwrite(repo, resourceTypePlay, "play-1", "owner", permissions.SetOf(permissions.PlaysWrite))
	return svc, repo
}

func callGrantTool(t *testing.T, svc *Service, actor, name, args string) (any, error) {
	t.Helper()
	ctx := identity.WithActor(t.Context(), identity.Actor{ID: actor})
	for _, tool := range MCPTools(svc) {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not registered", name)
	return nil, nil
}

func TestGrantTools_Surface(t *testing.T) {
	t.Parallel()
	svc, _ := newGrantsHarness(t)
	var names []string
	for _, tool := range MCPTools(svc) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title)
	}
	assert.Equal(t, []string{"permission_overwrite_list", "permission_overwrite_update"}, names)
}

func TestGrantTools_ErrorPaths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		actor   string
		tool    string
		args    string
		wantErr error
	}{
		{"list by the dropped doc_id alias", "owner", "permission_overwrite_list", `{"doc_id": "doc-1"}`, apperrs.ErrInvalid},
		{"list without a resource type", "owner", "permission_overwrite_list", `{"resource_id": "doc-1"}`, apperrs.ErrInvalid},
		{"list an unknown resource type", "owner", "permission_overwrite_list", `{"resource_type": "ticket", "resource_id": "t-1"}`, apperrs.ErrInvalid},
		{"update by the dropped doc_ids alias", "owner", "permission_overwrite_update", `{"resource_type": "doc", "doc_ids": ["doc-1"], "resource_ids": ["doc-1"], "user_ids": ["alice"], "actions": ["docs:read"], "grant": true}`, apperrs.ErrInvalid},
		{"update without grant", "owner", "permission_overwrite_update", `{"resource_type": "doc", "resource_ids": ["doc-1"], "user_ids": ["alice"], "actions": ["docs:read"]}`, apperrs.ErrInvalid},
		{"update with an unknown action", "owner", "permission_overwrite_update", `{"resource_type": "doc", "resource_ids": ["doc-1"], "user_ids": ["alice"], "actions": ["comment"], "grant": true}`, apperrs.ErrInvalid},
		{"update with no resources", "owner", "permission_overwrite_update", `{"resource_type": "doc", "resource_ids": [], "user_ids": ["alice"], "actions": ["docs:read"], "grant": true}`, apperrs.ErrInvalid},
		{"update a play with a docs action", "owner", "permission_overwrite_update", `{"resource_type": "play", "resource_ids": ["play-1"], "user_ids": ["alice"], "actions": ["docs:read"], "grant": false}`, apperrs.ErrInvalid},
		{"update an unknown resource type", "owner", "permission_overwrite_update", `{"resource_type": "ticket", "resource_ids": ["t-1"], "user_ids": ["alice"], "actions": ["docs:read"], "grant": true}`, apperrs.ErrInvalid},
		{"list a doc without the manage bit", "alice", "permission_overwrite_list", `{"resource_type": "doc", "resource_id": "doc-1"}`, apperrs.ErrForbidden},
		{"update an unknown doc", "owner", "permission_overwrite_update", `{"resource_type": "doc", "resource_ids": ["doc-1", "nope"], "user_ids": ["alice"], "actions": ["docs:read"], "grant": true}`, apperrs.ErrForbidden},
		{"update a play without plays:write", "alice", "permission_overwrite_update", `{"resource_type": "play", "resource_ids": ["play-1"], "user_ids": ["owner"], "actions": ["plays:run"], "grant": false}`, apperrs.ErrForbidden},
		{"update without a caller", "", "permission_overwrite_update", `{"resource_type": "doc", "resource_ids": ["doc-1"], "user_ids": ["alice"], "actions": ["docs:read"], "grant": true}`, apperrs.ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := newGrantsHarness(t)
			_, err := callGrantTool(t, svc, tt.actor, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.wantErr)
			_, err = repo.Get(t.Context(), resourceTypeDoc, "doc-1", "alice")
			require.ErrorIs(t, err, apperrs.ErrNotFound, "a refused call grants nothing")
		})
	}
}

func TestGrantUpdate_OnlyTheNamedActionsChange(t *testing.T) {
	t.Parallel()
	svc, repo := newGrantsHarness(t)
	update := func(actions string, grant bool) {
		t.Helper()
		args := fmt.Sprintf(`{"resource_type": "doc", "resource_ids": ["doc-1"], "user_ids": ["alice"], "actions": %s, "grant": %t}`, actions, grant)
		_, err := callGrantTool(t, svc, "owner", "permission_overwrite_update", args)
		require.NoError(t, err)
	}

	update(`["docs:read", "docs:write"]`, true)
	update(`["docs:delete"]`, true)
	g, err := repo.Get(t.Context(), resourceTypeDoc, "doc-1", "alice")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.DocsRead, permissions.DocsWrite, permissions.DocsDelete), g.Allow, "a later grant keeps the earlier ones")

	update(`["docs:write"]`, false)
	g, err = repo.Get(t.Context(), resourceTypeDoc, "doc-1", "alice")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.DocsRead, permissions.DocsDelete), g.Allow, "revoking one action keeps the rest")
}

func TestGrantList_ShowsEachUsersOverwrite(t *testing.T) {
	t.Parallel()
	svc, repo := newGrantsHarness(t)
	setOverwrite(repo, resourceTypeDoc, "doc-1", "alice", permissions.SetOf(permissions.DocsRead))

	got, err := callGrantTool(t, svc, "owner", "permission_overwrite_list", `{"resource_type": "doc", "resource_id": "doc-1"}`)
	require.NoError(t, err)
	page := got.(mcptool.Page[grantResult])
	assert.ElementsMatch(t, []grantResult{
		{UserID: "owner", Login: "owner", Allow: permissions.SetOf(permissions.PermissionsWrite)},
		{UserID: "alice", Login: "alice", Allow: permissions.SetOf(permissions.DocsRead)},
	}, page.Items)
}

func TestGrantList_AccountLookupFails_ReturnsTheError(t *testing.T) {
	t.Parallel()
	svc, repo, users := newAccessHarness()
	setOverwrite(repo, resourceTypeDoc, "doc-1", "owner", permissions.SetOf(permissions.PermissionsWrite))
	boom := errors.New("users unavailable")
	users.listErr = boom

	_, err := callGrantTool(t, svc, "owner", "permission_overwrite_list", `{"resource_type": "doc", "resource_id": "doc-1"}`)
	require.ErrorIs(t, err, boom)
}

func TestGrantTools_PlayExclusion(t *testing.T) {
	t.Parallel()
	svc, repo := newGrantsHarness(t)
	exclude := `{"resource_type": "play", "resource_ids": ["play-1"], "user_ids": ["alice"], "actions": ["plays:run"], "grant": false}`

	_, err := callGrantTool(t, svc, "owner", "permission_overwrite_update", exclude)
	require.NoError(t, err)
	g, err := repo.Get(t.Context(), resourceTypePlay, "play-1", "alice")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.PlaysRun), g.Deny)

	got, err := callGrantTool(t, svc, "owner", "permission_overwrite_list", `{"resource_type": "play", "resource_id": "play-1"}`)
	require.NoError(t, err)
	assert.Contains(t, got.(mcptool.Page[grantResult]).Items, grantResult{UserID: "alice", Login: "alice", Deny: permissions.SetOf(permissions.PlaysRun)})
}

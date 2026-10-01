package composite

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// fakeWorkspaces holds each user's workspaces and the role name they hold in each.
type fakeWorkspaces struct {
	byUser  map[string][]*tenancy.Workspace
	people  map[string][]tenancy.Person
	listErr error
	// renameErr is what Rename returns; renamed records the last call.
	renameErr error
	renamed   *renameCall
}

func (f *fakeWorkspaces) ListForUser(_ context.Context, userID string) ([]*tenancy.Workspace, error) {
	return f.byUser[userID], f.listErr
}

func (f *fakeWorkspaces) MemberRoleName(_ context.Context, _, _ string) (string, error) {
	return "Owner", nil
}

func (f *fakeWorkspaces) ListPeople(_ context.Context, _, workspaceID string) ([]tenancy.Person, error) {
	return f.people[workspaceID], nil
}

func (f *fakeWorkspaces) Rename(_ context.Context, userID, id string, name, slug *string) (*tenancy.Workspace, error) {
	f.renamed = &renameCall{userID: userID, id: id, name: name, slug: slug}
	return &tenancy.Workspace{ID: id, Name: "Acme Labs", Slug: "acme-labs"}, f.renameErr
}

type renameCall struct {
	userID, id string
	name, slug *string
}

type fakeRoles struct {
	byWorkspace map[string][]*roles.Role
	err         error
}

func (f fakeRoles) List(_ context.Context, workspaceID string) ([]*roles.Role, error) {
	return f.byWorkspace[workspaceID], f.err
}

func newWorkspaceListCall(rolesErr error) func(context.Context, json.RawMessage) (any, error) {
	ws := fakeWorkspaces{byUser: map[string][]*tenancy.Workspace{
		"u-1": {{ID: "ws-1", Name: "Acme"}, {ID: "ws-2", Name: "Beta"}},
		"u-2": {{ID: "ws-3", Name: "Other"}},
	}, people: map[string][]tenancy.Person{
		"ws-1": {{UserID: "u-1", Login: "LewisWelch94", DisplayName: "Lewis", AvatarURL: "https://avatars.example/1"}},
	}}
	rs := fakeRoles{err: rolesErr, byWorkspace: map[string][]*roles.Role{
		"ws-1": {
			{ID: "role-owner", WorkspaceID: "ws-1", Name: "Owner", IsOwnerRole: true},
			{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors", Permissions: permissions.SetOf(permissions.DocsRead, permissions.RolesClone)},
		},
	}}
	return WorkspaceTools(&ws, rs)[0].Call
}

func actorCtx(id string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: id})
}

func TestWorkspaceList_Errors(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		args     string
		rolesErr error
		want     error
	}{
		{"no caller is unauthorized", context.Background(), `{}`, nil, apperrs.ErrUnauthorized},
		{"an unknown argument is invalid", actorCtx("u-1"), `{"user_id":"u-2"}`, nil, apperrs.ErrInvalid},
		{"a workspace the caller is not in is not found", actorCtx("u-1"), `{"id":"ws-3"}`, nil, apperrs.ErrNotFound},
		{"a missing workspace is not found", actorCtx("u-1"), `{"id":"ghost"}`, nil, apperrs.ErrNotFound},
		{"a roles failure propagates", actorCtx("u-1"), `{"id":"ws-1"}`, errors.New("db down"), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newWorkspaceListCall(tt.rolesErr)(tt.ctx, json.RawMessage(tt.args))
			require.Error(t, err)
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
			}
		})
	}
}

func TestWorkspaceList_WithoutIDListsTheCallersWorkspacesOnly(t *testing.T) {
	out, err := newWorkspaceListCall(nil)(actorCtx("u-1"), json.RawMessage(`{}`))
	require.NoError(t, err)
	page, ok := out.(mcptool.Page[workspaceResult])
	require.True(t, ok)
	assert.Equal(t, []workspaceResult{{ID: "ws-1", Name: "Acme", Role: "Owner"}, {ID: "ws-2", Name: "Beta", Role: "Owner"}}, page.Items)
}

func TestWorkspaceList_WithIDReturnsPeopleRolesAndCatalog(t *testing.T) {
	out, err := newWorkspaceListCall(nil)(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1"}`))
	require.NoError(t, err)
	page, ok := out.(mcptool.Page[workspaceResult])
	require.True(t, ok)
	require.Len(t, page.Items, 1)
	got := page.Items[0]
	assert.Equal(t, "ws-1", got.ID)
	assert.Equal(t, []tenancy.Person{{UserID: "u-1", Login: "LewisWelch94", DisplayName: "Lewis", AvatarURL: "https://avatars.example/1"}}, got.People)
	assert.Equal(t, []roles.RoleResult{
		{ID: "role-owner", WorkspaceID: "ws-1", Name: "Owner", IsOwnerRole: true, Permissions: []string{}},
		{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors", Permissions: []string{"docs:read", "roles:clone"}},
	}, got.Roles)
	assert.Equal(t, permissions.Catalog(), got.PermissionCatalog)
	assert.Contains(t, got.PermissionCatalog, permissions.Info{Value: permissions.RolesClone, Label: "Clone roles to another workspace", Domain: "roles", Action: "clone", Area: permissions.AreaWorkspace})
}

func TestWorkspaceList_ListFailurePropagates(t *testing.T) {
	boom := errors.New("db down")
	call := WorkspaceTools(&fakeWorkspaces{listErr: boom}, fakeRoles{})[0].Call
	_, err := call(actorCtx("u-1"), json.RawMessage(`{}`))
	require.ErrorIs(t, err, boom)
}

func newWorkspaceUpdateCall(ws *fakeWorkspaces) func(context.Context, json.RawMessage) (any, error) {
	return WorkspaceTools(ws, fakeRoles{})[1].Call
}

func TestWorkspaceUpdate_PatchesAsTheCaller(t *testing.T) {
	ws := &fakeWorkspaces{}
	out, err := newWorkspaceUpdateCall(ws)(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","slug":"acme-labs"}`))
	require.NoError(t, err)
	assert.Equal(t, workspaceUpdateResult{ID: "ws-1", Name: "Acme Labs", Slug: "acme-labs"}, out)
	require.NotNil(t, ws.renamed)
	assert.Equal(t, "u-1", ws.renamed.userID)
	assert.Nil(t, ws.renamed.name, "an omitted name is passed on as omitted, so it keeps its value")
	require.NotNil(t, ws.renamed.slug)
	assert.Equal(t, "acme-labs", *ws.renamed.slug)
}

func TestWorkspaceUpdate_Errors(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		args      string
		renameErr error
		want      error
	}{
		{"no caller is unauthorized", context.Background(), `{"id":"ws-1","name":"Acme"}`, nil, apperrs.ErrUnauthorized},
		{"nothing to change is invalid", actorCtx("u-1"), `{"id":"ws-1"}`, nil, apperrs.ErrInvalid},
		{"a missing id is invalid", actorCtx("u-1"), `{"name":"Acme"}`, nil, apperrs.ErrInvalid},
		{"the use-case's refusal reaches the model", actorCtx("u-1"), `{"id":"ws-1","slug":"settings"}`, apperrs.ErrConflict, apperrs.ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newWorkspaceUpdateCall(&fakeWorkspaces{renameErr: tt.renameErr})(tt.ctx, json.RawMessage(tt.args))
			require.ErrorIs(t, err, tt.want)
		})
	}
}

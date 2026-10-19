package composite

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/plays"
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
	// archiving records Archive and Restore calls in order; the errors are what they return and archived what ListArchived lists.
	archiving                           []string
	archiveErr, restoreErr, archivedErr error
	archived                            []*tenancy.Workspace
}

func (f *fakeWorkspaces) ListForUser(_ context.Context, userID string) ([]*tenancy.Workspace, error) {
	return f.byUser[userID], f.listErr
}

func (f *fakeWorkspaces) MemberRoles(_ context.Context, workspaceID, _ string) ([]*tenancy.TeamRole, error) {
	return []*tenancy.TeamRole{{ID: "role-owner-" + workspaceID, Name: "Owner", IsOwner: true}}, nil
}

func (f *fakeWorkspaces) ListPeople(_ context.Context, _, workspaceID string) ([]tenancy.Person, error) {
	return f.people[workspaceID], nil
}

func (f *fakeWorkspaces) Rename(_ context.Context, userID, id string, name, slug *string) (*tenancy.Workspace, error) {
	f.renamed = &renameCall{userID: userID, id: id, name: name, slug: slug}
	return &tenancy.Workspace{ID: id, Name: "Acme Labs", Slug: "acme-labs"}, f.renameErr
}

func (f *fakeWorkspaces) Archive(_ context.Context, userID, id string) (*tenancy.Workspace, error) {
	f.archiving = append(f.archiving, "archive "+id+" by "+userID)
	return &tenancy.Workspace{ID: id, Name: "Acme Labs", Slug: "acme-labs"}, f.archiveErr
}

func (f *fakeWorkspaces) Restore(_ context.Context, userID, id string) (*tenancy.Workspace, error) {
	f.archiving = append(f.archiving, "restore "+id+" by "+userID)
	return &tenancy.Workspace{ID: id, Name: "Acme Labs", Slug: "acme-labs"}, f.restoreErr
}

func (f *fakeWorkspaces) ListArchived(context.Context, string) ([]*tenancy.Workspace, error) {
	return f.archived, f.archivedErr
}

type renameCall struct {
	userID, id string
	name, slug *string
}

// fakeLimits holds each workspace's auto play limits; err is what both calls return.
type fakeLimits struct {
	caps map[string]plays.AutoPlayLimits
	err  error
}

func (f *fakeLimits) AutoPlayLimits(_ context.Context, workspaceID string) (plays.AutoPlayLimits, error) {
	return f.caps[workspaceID], f.err
}

func (f *fakeLimits) SetAutoPlayLimits(_ context.Context, workspaceID string, patch plays.AutoPlayLimitsPatch) (plays.AutoPlayLimits, error) {
	if f.err != nil {
		return plays.AutoPlayLimits{}, f.err
	}
	limits := f.caps[workspaceID]
	if patch.DailyCapPerTicket != nil {
		limits.DailyCapPerTicket = *patch.DailyCapPerTicket
	}
	if patch.RunsAtOncePerPerson != nil {
		limits.RunsAtOncePerPerson = *patch.RunsAtOncePerPerson
	}
	f.caps[workspaceID] = limits
	return limits, nil
}

// fakeAccounts records which accounts were detached from which workspace; err refuses every change.
type fakeAccounts struct {
	changes []string
	err     error
}

func (f *fakeAccounts) UnassignInstallation(_ context.Context, account, workspaceID string) error {
	f.changes = append(f.changes, "-"+account+"@"+workspaceID)
	return f.err
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
			{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors", Permissions: permissions.SetOf(permissions.DocsRead, permissions.RolesClone), ShowLockedActions: true},
		},
	}}
	return WorkspaceTools(&ws, rs, &fakeLimits{err: apperrs.ErrForbidden}, &fakeAccounts{}, nil)[0].Call
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
	assert.Equal(t, []workspaceResult{
		{ID: "ws-1", Name: "Acme", YourRoles: []*tenancy.TeamRole{{ID: "role-owner-ws-1", Name: "Owner", IsOwner: true}}},
		{ID: "ws-2", Name: "Beta", YourRoles: []*tenancy.TeamRole{{ID: "role-owner-ws-2", Name: "Owner", IsOwner: true}}},
	}, page.Items)
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
		{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors", Permissions: []string{"docs:read", "roles:clone"}, ShowLockedActions: true},
	}, got.Roles)
	assert.Equal(t, permissions.Catalog(), got.PermissionCatalog)
	assert.Contains(t, got.PermissionCatalog, permissions.Info{Value: permissions.RolesClone, Label: "Clone roles to another workspace", Domain: "roles", Action: "clone", Area: permissions.AreaWorkspace})
}

func TestWorkspaceList_ListFailurePropagates(t *testing.T) {
	boom := errors.New("db down")
	call := WorkspaceTools(&fakeWorkspaces{listErr: boom}, fakeRoles{}, &fakeLimits{}, &fakeAccounts{}, nil)[0].Call
	_, err := call(actorCtx("u-1"), json.RawMessage(`{}`))
	require.ErrorIs(t, err, boom)
}

func newWorkspaceUpdateCall(ws *fakeWorkspaces) func(context.Context, json.RawMessage) (any, error) {
	return WorkspaceTools(ws, fakeRoles{}, &fakeLimits{caps: map[string]plays.AutoPlayLimits{}}, &fakeAccounts{}, nil)[1].Call
}

func TestWorkspaceList_WithIDShowsTheAutoPlayLimitsToItsReaders(t *testing.T) {
	ws := &fakeWorkspaces{byUser: map[string][]*tenancy.Workspace{"u-1": {{ID: "ws-1", Name: "Acme"}}}}
	limits := map[string]plays.AutoPlayLimits{"ws-1": {DailyCapPerTicket: 7, RunsAtOncePerPerson: 2}}
	call := WorkspaceTools(ws, fakeRoles{}, &fakeLimits{caps: limits}, &fakeAccounts{}, nil)[0].Call
	out, err := call(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1"}`))
	require.NoError(t, err)
	got := out.(mcptool.Page[workspaceResult]).Items[0]
	require.NotNil(t, got.AutoPlayDailyCap)
	require.NotNil(t, got.AutoPlayRunsAtOnce)
	assert.Equal(t, []int{7, 2}, []int{*got.AutoPlayDailyCap, *got.AutoPlayRunsAtOnce})

	call = WorkspaceTools(ws, fakeRoles{}, &fakeLimits{err: errors.New("db down")}, &fakeAccounts{}, nil)[0].Call
	_, err = call(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1"}`))
	require.Error(t, err, "a failure other than forbidden is not hidden")
}

func TestWorkspaceUpdate_DailyCap(t *testing.T) {
	member := map[string][]*tenancy.Workspace{"u-1": {{ID: "ws-1", Name: "Acme", Slug: "acme"}}}
	t.Run("alone it changes the cap and reports the workspace as it stands", func(t *testing.T) {
		ws := &fakeWorkspaces{byUser: member}
		out, err := newWorkspaceUpdateCall(ws)(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","auto_play_daily_cap":9}`))
		require.NoError(t, err)
		nine, none := 9, 0
		assert.Equal(t, workspaceUpdateResult{ID: "ws-1", Name: "Acme", Slug: "acme", AutoPlayDailyCap: &nine, AutoPlayRunsAtOnce: &none}, out)
		assert.Nil(t, ws.renamed, "no name or slug passed, so nothing is renamed")
	})
	t.Run("runs at once alone keeps the cap", func(t *testing.T) {
		limits := &fakeLimits{caps: map[string]plays.AutoPlayLimits{"ws-1": {DailyCapPerTicket: 5, RunsAtOncePerPerson: 1}}}
		call := WorkspaceTools(&fakeWorkspaces{byUser: member}, fakeRoles{}, limits, &fakeAccounts{}, nil)[1].Call
		out, err := call(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","auto_play_runs_at_once":3}`))
		require.NoError(t, err)
		five, three := 5, 3
		assert.Equal(t, workspaceUpdateResult{ID: "ws-1", Name: "Acme", Slug: "acme", AutoPlayDailyCap: &five, AutoPlayRunsAtOnce: &three}, out)
	})
	t.Run("in a workspace the caller is not in it is not found", func(t *testing.T) {
		_, err := newWorkspaceUpdateCall(&fakeWorkspaces{byUser: member})(actorCtx("u-2"), json.RawMessage(`{"id":"ws-1","auto_play_daily_cap":9}`))
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("a refused cap after a rename says the rename took effect", func(t *testing.T) {
		ws := &fakeWorkspaces{byUser: member}
		call := WorkspaceTools(ws, fakeRoles{}, &fakeLimits{err: apperrs.ErrForbidden}, &fakeAccounts{}, nil)[1].Call
		_, err := call(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","name":"Acme Labs","auto_play_daily_cap":9}`))
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		var partial *mcptool.PartialError
		require.ErrorAs(t, err, &partial)
		assert.Equal(t, []string{"name and slug"}, partial.Applied)
	})
	t.Run("a refused cap alone is the refusal", func(t *testing.T) {
		call := WorkspaceTools(&fakeWorkspaces{byUser: member}, fakeRoles{}, &fakeLimits{err: apperrs.ErrInvalid}, &fakeAccounts{}, nil)[1].Call
		_, err := call(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","auto_play_daily_cap":0}`))
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		var partial *mcptool.PartialError
		assert.False(t, errors.As(err, &partial))
	})
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

func TestWorkspaceUpdate_GitHubAccounts(t *testing.T) {
	member := map[string][]*tenancy.Workspace{"u-1": {{ID: "ws-1", Name: "Acme", Slug: "acme"}}}
	t.Run("detaches accounts from the workspace", func(t *testing.T) {
		accounts := &fakeAccounts{}
		call := WorkspaceTools(&fakeWorkspaces{byUser: member}, fakeRoles{}, &fakeLimits{}, accounts, nil)[1].Call
		out, err := call(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","remove_github_accounts":["initech"]}`))
		require.NoError(t, err)
		assert.Equal(t, []string{"-initech@ws-1"}, accounts.changes)
		assert.Equal(t, workspaceUpdateResult{ID: "ws-1", Name: "Acme", Slug: "acme", GitHubAccountsChanged: []string{"initech"}}, out)
	})
	t.Run("an account cannot be linked by hand", func(t *testing.T) {
		accounts := &fakeAccounts{}
		call := WorkspaceTools(&fakeWorkspaces{byUser: member}, fakeRoles{}, &fakeLimits{}, accounts, nil)[1].Call
		_, err := call(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","add_github_accounts":["bob"]}`))
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		assert.Empty(t, accounts.changes)
	})
	t.Run("a refused account after a rename says the rename took effect", func(t *testing.T) {
		call := WorkspaceTools(&fakeWorkspaces{byUser: member}, fakeRoles{}, &fakeLimits{}, &fakeAccounts{err: apperrs.ErrForbidden}, nil)[1].Call
		_, err := call(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","name":"Acme Labs","remove_github_accounts":["globex"]}`))
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		var partial *mcptool.PartialError
		require.ErrorAs(t, err, &partial)
		assert.Equal(t, []string{"name and slug"}, partial.Applied)
	})
}

// fakeSetup records whose finish it was given, or refuses it.
type fakeSetup struct {
	finishedBy string
	err        error
}

func (f *fakeSetup) SaveGrid(_ context.Context, actorID, _ string, in access.GridSave) error {
	if f.err != nil {
		return f.err
	}
	if in.Finish {
		f.finishedBy = actorID
	}
	return nil
}

func TestWorkspaceUpdate_FinishAccessSetup(t *testing.T) {
	member := map[string][]*tenancy.Workspace{"u-1": {{ID: "ws-1", Name: "Acme", Slug: "acme"}}}
	t.Run("ends the one-time setup", func(t *testing.T) {
		setup := &fakeSetup{}
		call := WorkspaceTools(&fakeWorkspaces{byUser: member}, fakeRoles{}, &fakeLimits{}, &fakeAccounts{}, setup)[1].Call
		out, err := call(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","finish_access_setup":true}`))
		require.NoError(t, err)
		assert.Equal(t, "u-1", setup.finishedBy)
		assert.Equal(t, workspaceUpdateResult{ID: "ws-1", Name: "Acme", Slug: "acme", AccessSetupFinished: true}, out)
	})
	t.Run("a refusal on its own reaches the model", func(t *testing.T) {
		call := WorkspaceTools(&fakeWorkspaces{byUser: member}, fakeRoles{}, &fakeLimits{}, &fakeAccounts{}, &fakeSetup{err: apperrs.ErrForbidden})[1].Call
		_, err := call(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","finish_access_setup":true}`))
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		var partial *mcptool.PartialError
		assert.False(t, errors.As(err, &partial))
	})
	t.Run("a refusal after a rename says the rename took effect", func(t *testing.T) {
		call := WorkspaceTools(&fakeWorkspaces{byUser: member}, fakeRoles{}, &fakeLimits{}, &fakeAccounts{}, &fakeSetup{err: apperrs.ErrForbidden})[1].Call
		_, err := call(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","name":"Acme Labs","finish_access_setup":true}`))
		var partial *mcptool.PartialError
		require.ErrorAs(t, err, &partial)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("someone else's workspace is not found", func(t *testing.T) {
		call := WorkspaceTools(&fakeWorkspaces{byUser: member}, fakeRoles{}, &fakeLimits{}, &fakeAccounts{}, &fakeSetup{})[1].Call
		_, err := call(actorCtx("u-1"), json.RawMessage(`{"id":"ws-9","finish_access_setup":true}`))
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestWorkspaceUpdate_Archived(t *testing.T) {
	tool := func(ws *fakeWorkspaces) func(context.Context, json.RawMessage) (any, error) {
		return WorkspaceTools(ws, fakeRoles{}, &fakeLimits{}, &fakeAccounts{}, nil)[1].Call
	}
	yes, no := true, false
	t.Run("archived true applies after the rename, which needs a workspace the caller still reaches", func(t *testing.T) {
		ws := &fakeWorkspaces{}
		out, err := tool(ws)(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","name":"Acme Labs","archived":true}`))
		require.NoError(t, err)
		require.NotNil(t, ws.renamed)
		assert.Equal(t, []string{"archive ws-1 by u-1"}, ws.archiving)
		assert.Equal(t, workspaceUpdateResult{ID: "ws-1", Name: "Acme Labs", Slug: "acme-labs", Archived: &yes}, out)
	})
	t.Run("archived false applies before the rename, since the instance Owner is not yet in it", func(t *testing.T) {
		ws := &fakeWorkspaces{}
		out, err := tool(ws)(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","name":"Acme Labs","archived":false}`))
		require.NoError(t, err)
		assert.Equal(t, []string{"restore ws-1 by u-1"}, ws.archiving)
		require.NotNil(t, ws.renamed)
		assert.Equal(t, workspaceUpdateResult{ID: "ws-1", Name: "Acme Labs", Slug: "acme-labs", Archived: &no}, out)
	})
	t.Run("a refused archive after a rename says the rename took effect", func(t *testing.T) {
		_, err := tool(&fakeWorkspaces{archiveErr: apperrs.ErrForbidden})(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","name":"Acme Labs","archived":true}`))
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		var partial *mcptool.PartialError
		require.ErrorAs(t, err, &partial)
		assert.Equal(t, []string{"every other field"}, partial.Applied)
	})
	t.Run("a refused rename after a restore says the restore took effect", func(t *testing.T) {
		_, err := tool(&fakeWorkspaces{renameErr: apperrs.ErrForbidden})(actorCtx("u-1"), json.RawMessage(`{"id":"ws-1","name":"Acme Labs","archived":false}`))
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		var partial *mcptool.PartialError
		require.ErrorAs(t, err, &partial)
		assert.Equal(t, []string{"archived"}, partial.Applied)
	})
	t.Run("a refusal on its own is just the refusal", func(t *testing.T) {
		for args, ws := range map[string]*fakeWorkspaces{
			`{"id":"ws-1","archived":true}`:  {archiveErr: apperrs.ErrForbidden},
			`{"id":"ws-1","archived":false}`: {restoreErr: apperrs.ErrForbidden},
		} {
			_, err := tool(ws)(actorCtx("u-1"), json.RawMessage(args))
			require.ErrorIs(t, err, apperrs.ErrForbidden, args)
			var partial *mcptool.PartialError
			assert.False(t, errors.As(err, &partial), args)
		}
	})
}

func TestWorkspaceList_ArchivedListsWhatTheInstanceOwnerRestores(t *testing.T) {
	at := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	ws := &fakeWorkspaces{archived: []*tenancy.Workspace{{ID: "ws-9", Name: "Globex", Slug: "globex", ArchivedAt: &at}}}
	call := WorkspaceTools(ws, fakeRoles{}, &fakeLimits{}, &fakeAccounts{}, nil)[0].Call

	out, err := call(actorCtx("u-1"), json.RawMessage(`{"archived":true}`))
	require.NoError(t, err)
	assert.Equal(t, []workspaceResult{{ID: "ws-9", Name: "Globex", Slug: "globex", ArchivedAt: &at}}, out.(mcptool.Page[workspaceResult]).Items)

	ws.archivedErr = apperrs.ErrForbidden
	_, err = call(actorCtx("u-1"), json.RawMessage(`{"archived":true}`))
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

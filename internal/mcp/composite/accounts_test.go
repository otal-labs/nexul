package composite

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// fakeTeam records every membership call in order; members maps workspace id to the user ids already in it.
type fakeTeam struct {
	team    *tenancy.Team
	members map[string][]string
	failOn  string
	failErr error
	calls   []string
}

func (f *fakeTeam) record(call string) error {
	f.calls = append(f.calls, call)
	if call != f.failOn {
		return nil
	}
	if f.failErr != nil {
		return f.failErr
	}
	return apperrs.ErrForbidden
}

func (f *fakeTeam) ListTeam(context.Context, string) (*tenancy.Team, error) {
	if f.team == nil {
		return nil, apperrs.ErrForbidden
	}
	return f.team, nil
}

func (f *fakeTeam) MemberRoleID(_ context.Context, workspaceID, userID string) (string, error) {
	for _, u := range f.members[workspaceID] {
		if u == userID {
			return "role", nil
		}
	}
	return "", apperrs.ErrNotFound
}

func (f *fakeTeam) AddMember(_ context.Context, _, workspaceID, _, roleID string) error {
	return f.record("add " + workspaceID + " " + roleID)
}

func (f *fakeTeam) ChangeMemberRole(_ context.Context, _, workspaceID, _, roleID string) error {
	return f.record("role " + workspaceID + " " + roleID)
}

func (f *fakeTeam) SetMemberOverrides(_ context.Context, _, workspaceID, _ string, allow, deny *permissions.Set) error {
	return f.record("overrides " + workspaceID)
}

func (f *fakeTeam) RemoveMember(_ context.Context, _, workspaceID, _ string) error {
	return f.record("remove " + workspaceID)
}

func (f *fakeTeam) SetEveryProject(_ context.Context, _, workspaceID, _, every string) error {
	return f.record("every " + workspaceID + " " + every)
}

func (f *fakeTeam) SetProjectAccess(_ context.Context, _, workspaceID, _, projectID string, allow permissions.Set) error {
	return f.record("access " + workspaceID + " " + projectID + " " + allow.String())
}

type fakeStatusSetter struct{ calls []auth.AccountStatus }

func (f *fakeStatusSetter) UpdateAccountStatus(_ context.Context, _, _ string, status auth.AccountStatus) error {
	f.calls = append(f.calls, status)
	return nil
}

func callAccountTool(t *testing.T, tools []mcptool.Tool, name, args string) (any, error) {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool.Call(actorCtx("admin"), json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not registered", name)
	return nil, nil
}

func TestAccountList_CarriesPresenceAndEachMembershipWithTheViewersManageFlag(t *testing.T) {
	seen := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	web := []*tenancy.ProjectAccess{{ProjectID: "p-web", ProjectName: "Web", Allow: permissions.SetOf(permissions.TicketsRead)}}
	team := &fakeTeam{team: &tenancy.Team{
		People: []*tenancy.TeamPerson{{
			TeamAccount: tenancy.TeamAccount{ID: "bob", Login: "bob", Status: "active", Online: true, LastSeenAt: &seen},
			Workspaces: []*tenancy.TeamMembership{
				{WorkspaceID: "ws-nexul", RoleName: "Editor", EveryProject: tenancy.EveryProjectRole, Projects: web},
				{WorkspaceID: "ws-acme", RoleName: "Viewer", EveryProject: tenancy.EveryProjectNone, Projects: web},
			},
		}},
		Workspaces: []*tenancy.TeamWorkspace{{ID: "ws-nexul", CanManageMembers: true}, {ID: "ws-acme"}},
	}}

	got, err := callAccountTool(t, AccountTools(&fakeStatusSetter{}, team), "account_list", `{}`)
	require.NoError(t, err)
	page := got.(mcptool.Page[accountResult])
	require.Len(t, page.Items, 1)
	assert.True(t, page.Items[0].Online)
	assert.Equal(t, &seen, page.Items[0].LastSeenAt)
	require.Len(t, page.Items[0].Workspaces, 2)
	assert.True(t, page.Items[0].Workspaces[0].CanManageMembers)
	assert.False(t, page.Items[0].Workspaces[1].CanManageMembers)
	assert.Nil(t, page.Items[0].Workspaces[0].Projects, "rows kept for a From role member grant nothing, so they stay out")
	assert.Equal(t, tenancy.EveryProjectNone, page.Items[0].Workspaces[1].EveryProject)
	assert.Equal(t, web, page.Items[0].Workspaces[1].Projects)
}

func TestAccountUpdate_RoutesEachWorkspaceToAddOrChangeAndStopsAtTheFirstFailure(t *testing.T) {
	tests := []struct {
		name      string
		args      string
		failOn    string
		wantCalls []string
		wantErr   error
	}{
		{"nothing to change is invalid", `{"id":"bob"}`, "", nil, apperrs.ErrInvalid},
		{"a workspace the account is not in is an add", `{"id":"bob","workspaces":[{"workspace_id":"ws-acme","role_id":"viewer"}]}`, "", []string{"add ws-acme viewer"}, nil},
		{"a workspace the account is in is a role change", `{"id":"bob","workspaces":[{"workspace_id":"ws-nexul","role_id":"editor"}]}`, "", []string{"role ws-nexul editor"}, nil},
		{"overrides alone leave the role alone", `{"id":"bob","workspaces":[{"workspace_id":"ws-nexul","deny":["docs:delete"]}]}`, "", []string{"overrides ws-nexul"}, nil},
		{"removal comes after the workspaces", `{"id":"bob","remove_workspace_ids":["ws-nexul"],"workspaces":[{"workspace_id":"ws-acme","role_id":"viewer"}]}`, "", []string{"add ws-acme viewer", "remove ws-nexul"}, nil},
		{"a refused workspace stops the rest", `{"id":"bob","workspaces":[{"workspace_id":"ws-acme","role_id":"viewer"}],"remove_workspace_ids":["ws-nexul"]}`, "add ws-acme viewer", []string{"add ws-acme viewer"}, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			team := &fakeTeam{members: map[string][]string{"ws-nexul": {"bob"}}, failOn: tt.failOn}
			_, err := callAccountTool(t, AccountTools(&fakeStatusSetter{}, team), "account_update", tt.args)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
			if tt.wantErr == nil {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantCalls, team.calls)
		})
	}
}

func TestAccountUpdate_ProjectAccessRunsAfterTheMembershipAndNamesTheFieldThatFailed(t *testing.T) {
	notHeld := fmt.Errorf("%w: tickets:delete is not yours to give", apperrs.ErrForbidden)
	tests := []struct {
		name      string
		args      string
		failOn    string
		failErr   error
		wantCalls []string
		wantErr   error
		wantMsg   string
	}{
		{"an invalid every_project changes nothing", `{"id":"bob","workspaces":[{"workspace_id":"ws-nexul","every_project":"some"}]}`, "every ws-nexul some", apperrs.ErrInvalid,
			[]string{"every ws-nexul some"}, apperrs.ErrInvalid, "workspaces[ws-nexul].every_project: invalid; nothing was changed"},
		{"an unknown project points at project_list", `{"id":"bob","workspaces":[{"workspace_id":"ws-nexul","project_access":[{"project_id":"p-gone","allow":["tickets:read"]}]}]}`, "access ws-nexul p-gone tickets:read", apperrs.ErrNotFound,
			[]string{"access ws-nexul p-gone tickets:read"}, apperrs.ErrNotFound, "project_list"},
		{"a level the caller does not hold stops the rest and says what applied", `{"id":"bob","workspaces":[{"workspace_id":"ws-nexul","every_project":"none","project_access":[{"project_id":"p-1","allow":["tickets:delete"]},{"project_id":"p-2","allow":["tickets:read"]}]}],"remove_workspace_ids":["ws-acme"]}`, "access ws-nexul p-1 tickets:delete", notHeld,
			[]string{"every ws-nexul none", "access ws-nexul p-1 tickets:delete"}, apperrs.ErrForbidden, "already applied: workspaces[ws-nexul].every_project"},
		{"role, overrides, every_project, then each project in order", `{"id":"bob","workspaces":[{"workspace_id":"ws-nexul","role_id":"editor","deny":["docs:delete"],"every_project":"none","project_access":[{"project_id":"p-1","allow":["tickets:read"]},{"project_id":"p-2","allow":[]}]}]}`, "", nil,
			[]string{"role ws-nexul editor", "overrides ws-nexul", "every ws-nexul none", "access ws-nexul p-1 tickets:read", "access ws-nexul p-2 "}, nil, ""},
		{"access fields alone leave the role alone", `{"id":"bob","workspaces":[{"workspace_id":"ws-nexul","project_access":[{"project_id":"p-1","allow":["tickets:read"]}]}]}`, "", nil,
			[]string{"access ws-nexul p-1 tickets:read"}, nil, ""},
		{"an omitted project_access keeps every project's levels", `{"id":"bob","workspaces":[{"workspace_id":"ws-nexul","every_project":"role"}]}`, "", nil,
			[]string{"every ws-nexul role"}, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			team := &fakeTeam{members: map[string][]string{"ws-nexul": {"bob"}}, failOn: tt.failOn, failErr: tt.failErr}
			_, err := callAccountTool(t, AccountTools(&fakeStatusSetter{}, team), "account_update", tt.args)
			assert.Equal(t, tt.wantCalls, team.calls)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

func TestAccountUpdate_StatusGoesToTheAccountAndTheResultOmitsAnUnreadableTeam(t *testing.T) {
	status := &fakeStatusSetter{}
	got, err := callAccountTool(t, AccountTools(status, &fakeTeam{}), "account_update", `{"id":"bob","status":"disabled"}`)
	require.NoError(t, err)
	assert.Equal(t, []auth.AccountStatus{auth.AccountDisabled}, status.calls)
	assert.Equal(t, accountUpdateResult{Applied: []string{"status"}}, got)
}

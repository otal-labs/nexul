package composite

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
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

func (f *fakeTeam) AddMember(_ context.Context, _, workspaceID, _ string, roleIDs ...string) error {
	return f.record("add " + workspaceID + " " + strings.Join(roleIDs, ","))
}

func (f *fakeTeam) UpdateMemberRoles(_ context.Context, _, workspaceID, _ string, add, remove []string) error {
	return f.record("roles " + workspaceID + " +" + strings.Join(add, ",") + " -" + strings.Join(remove, ","))
}

func (f *fakeTeam) RemoveMember(_ context.Context, _, workspaceID, _ string) error {
	return f.record("remove " + workspaceID)
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

func TestAccountList_CarriesPresenceProvidersAndEachMembershipWithTheViewersManageFlag(t *testing.T) {
	seen := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	team := &fakeTeam{team: &tenancy.Team{
		People: []*tenancy.TeamPerson{{
			TeamAccount: tenancy.TeamAccount{ID: "bob", Login: "bob", Status: "active", Online: true, LastSeenAt: &seen, Providers: []string{"github", "discord"}},
			Workspaces: []*tenancy.TeamMembership{
				{WorkspaceID: "ws-nexul", RoleName: "Editor, Frontend", Roles: []*tenancy.TeamRole{{ID: "r-editor", Name: "Editor"}, {ID: "r-front", Name: "Frontend"}}},
				{WorkspaceID: "ws-acme", RoleName: "Viewer"},
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
	assert.Equal(t, []string{"github", "discord"}, page.Items[0].Providers)
	require.Len(t, page.Items[0].Workspaces, 2)
	assert.True(t, page.Items[0].Workspaces[0].CanManageMembers)
	assert.False(t, page.Items[0].Workspaces[1].CanManageMembers)
	assert.Equal(t, []*tenancy.TeamRole{{ID: "r-editor", Name: "Editor"}, {ID: "r-front", Name: "Frontend"}}, page.Items[0].Workspaces[0].Roles)
	raw, err := json.Marshal(page.Items[0].Workspaces[0])
	require.NoError(t, err)
	for _, gone := range []string{"role_id", "role_name", "allow", "deny", "every_project", "projects"} {
		assert.NotContains(t, string(raw), `"`+gone+`"`, "the single role and the per-person fields are gone")
	}
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
		{"a workspace the account is not in is an add", `{"id":"bob","workspaces":[{"workspace_id":"ws-acme","add_roles":["viewer","front"]}]}`, "", []string{"add ws-acme viewer,front"}, nil},
		{"a workspace the account is in gives and takes away", `{"id":"bob","workspaces":[{"workspace_id":"ws-nexul","add_roles":["editor"],"remove_roles":["viewer"]}]}`, "", []string{"roles ws-nexul +editor -viewer"}, nil},
		{"taking a role from a non-member is invalid", `{"id":"bob","workspaces":[{"workspace_id":"ws-acme","remove_roles":["viewer"]}]}`, "", nil, apperrs.ErrInvalid},
		{"removal comes after the workspaces", `{"id":"bob","remove_workspace_ids":["ws-nexul"],"workspaces":[{"workspace_id":"ws-acme","add_roles":["viewer"]}]}`, "", []string{"add ws-acme viewer", "remove ws-nexul"}, nil},
		{"a refused workspace stops the rest", `{"id":"bob","workspaces":[{"workspace_id":"ws-acme","add_roles":["viewer"]}],"remove_workspace_ids":["ws-nexul"]}`, "add ws-acme viewer", []string{"add ws-acme viewer"}, apperrs.ErrForbidden},
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

// TestAccountUpdate_RemovedFieldsFailTheSchema: each field the switch removed is gone from the input, so sending one,
// even empty, is a schema error before anything changes.
func TestAccountUpdate_RemovedFieldsFailTheSchema(t *testing.T) {
	for _, entry := range []string{
		`{"workspace_id":"ws-nexul","role_id":"editor"}`,
		`{"workspace_id":"ws-nexul","deny":["docs:delete"]}`,
		`{"workspace_id":"ws-acme","allow":[]}`,
		`{"workspace_id":"ws-nexul","every_project":"none"}`,
		`{"workspace_id":"ws-nexul","project_access":[{"project_id":"p-1","allow":["tickets:read"]}]}`,
	} {
		team := &fakeTeam{members: map[string][]string{"ws-nexul": {"bob"}}}
		_, err := callAccountTool(t, AccountTools(&fakeStatusSetter{}, team), "account_update", `{"id":"bob","workspaces":[`+entry+`]}`)
		require.ErrorIs(t, err, apperrs.ErrInvalid, entry)
		assert.Contains(t, err.Error(), "input schema", entry)
		assert.Empty(t, team.calls, entry)
	}
}

func TestAccountUpdate_StatusGoesToTheAccountAndTheResultOmitsAnUnreadableTeam(t *testing.T) {
	status := &fakeStatusSetter{}
	got, err := callAccountTool(t, AccountTools(status, &fakeTeam{}), "account_update", `{"id":"bob","status":"disabled"}`)
	require.NoError(t, err)
	assert.Equal(t, []auth.AccountStatus{auth.AccountDisabled}, status.calls)
	assert.Equal(t, accountUpdateResult{Applied: []string{"status"}}, got)
}

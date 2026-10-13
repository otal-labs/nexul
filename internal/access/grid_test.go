package access

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// gridFixture is projectsFixture's workspace with a manager, whose role manages access in the projects it is listed in
// (p-open), a public channel, and the grid source the repo hands back for it.
func gridFixture(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	s, repo := projectsFixture(t)
	s.roles.(*fakeRoles).set("ws", "manager", RolesHeld(HeldRole{ID: "role-manager", Permissions: permissions.SetOf(permissions.ProjectsAccess)}))
	s.SetScopes(fakeScopes{projects: map[string]string{"p-open": "ws", "p-hidden": "ws"}, workspaces: map[string][]string{"manager": {"ws"}, "owner": {"ws"}}})
	require.NoError(t, repo.SetRole(t.Context(), "project", "p-open", "role-manager", permissions.SetOf(permissions.ProjectsOpen), nil))
	repo.grid = &GridSource{
		Grid: Grid{
			WorkspaceID: "ws",
			Roles:       []GridRole{{ID: "role-client", Name: "Client", Holders: 1}, {ID: "role-team", Name: "Team", Holders: 1}, {ID: "role-manager", Name: "Manager", Holders: 1}},
			Projects:    []GridProject{{ID: "p-open", Name: "Storefront", Prefix: "STO"}, {ID: "p-hidden", Name: "Backoffice", Prefix: "BO"}},
			Channels:    []GridChannel{{ID: "c-backend", Name: "backend", Kind: "channel"}},
		},
		MemberIDs: []string{"client", "team", "nobody", "owner", "manager"},
		Rules: []*Overwrite{
			{ResourceType: "project", ResourceID: "p-open", Target: TargetRole, RoleID: "role-team", Allow: permissions.SetOf(permissions.ProjectsOpen)},
			{ResourceType: "project", ResourceID: "p-open", Target: TargetRole, RoleID: "role-manager", Allow: permissions.SetOf(permissions.ProjectsOpen)},
			{ResourceType: "project", ResourceID: "p-open", Target: TargetPerson, UserID: "client", Allow: permissions.SetOf(permissions.ProjectsOpen)},
			{ResourceType: "project", ResourceID: "p-hidden", Target: TargetRole, RoleID: "role-client", Deny: permissions.SetOf(permissions.ProjectsOpen)},
			{ResourceType: "channel", ResourceID: "c-backend", Target: TargetRole, RoleID: "role-client", Deny: permissions.SetOf(permissions.ChannelsRead)},
		},
	}
	return s, repo
}

func TestGrid_ShowsWhatTheCallerManages(t *testing.T) {
	s, _ := gridFixture(t)

	owner, err := s.Grid(t.Context(), "owner", "ws")
	require.NoError(t, err)
	assert.Equal(t, []GridProject{
		{ID: "p-open", Name: "Storefront", Prefix: "STO", RoleIDs: []string{"role-team", "role-manager"}, UserIDs: []string{"client"}},
		{ID: "p-hidden", Name: "Backoffice", Prefix: "BO", RoleIDs: []string{}, UserIDs: []string{}},
	}, owner.Projects, "a role kept out by a Deny is not listed")
	assert.Equal(t, []GridChannel{{ID: "c-backend", Name: "backend", Kind: "channel", HiddenRoleIDs: []string{"role-client"}}}, owner.Channels)

	manager, err := s.Grid(t.Context(), "manager", "ws")
	require.NoError(t, err)
	require.Len(t, manager.Projects, 1, "only the project they manage")
	assert.Equal(t, "p-open", manager.Projects[0].ID)
	assert.Empty(t, manager.Channels, "public channels are the Owner's")

	_, err = s.Grid(t.Context(), "nobody", "ws")
	assert.ErrorIs(t, err, apperrs.ErrForbidden, "a member who manages no project")
	_, err = s.Grid(t.Context(), "stranger", "ws")
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.Grid(t.Context(), "", "ws")
	assert.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestSaveGrid_WritesOnlyWhatMoved(t *testing.T) {
	s, repo := gridFixture(t)

	require.NoError(t, s.SaveGrid(t.Context(), "owner", "ws", GridSave{
		Projects: []GridProjectSave{
			{ProjectID: "p-open", RoleIDs: []string{"role-team", "role-manager"}},
			{ProjectID: "p-hidden", RoleIDs: []string{"role-team"}, UserIDs: []string{"nobody"}},
		},
		Channels: []GridChannelSave{{ChannelID: "c-backend", HiddenRoleIDs: []string{"role-team"}}},
		Finish:   true,
	}))

	assert.Equal(t, []string{
		"channel c-backend role-client allow  deny ",
		"channel c-backend role-team allow  deny channels:read",
		"project p-hidden nobody allow projects:open deny ",
		"project p-hidden role-team allow projects:open deny ",
		"project p-open client allow  deny ",
	}, summarize(repo.saved), "the client's Deny on p-hidden stays, the rest moves")
	assert.Equal(t, "owner", repo.finishedBy)
	require.Len(t, repo.events, 1, "one frame for the whole save")
	assert.Equal(t, GrantEvent{ResourceType: "workspace", ResourceID: "ws", Target: TargetRole, ActorID: "owner"}, repo.events[0].Payload)

	repo.saved, repo.events = nil, nil
	require.NoError(t, s.SaveGrid(t.Context(), "owner", "ws", GridSave{Projects: []GridProjectSave{{ProjectID: "p-open", RoleIDs: []string{"role-team", "role-manager"}, UserIDs: []string{"client"}}}}))
	assert.Nil(t, repo.saved, "nothing moved, nothing written")
	assert.Empty(t, repo.events)

	repo.grid.Setup.Finished = true
	require.NoError(t, s.SaveGrid(t.Context(), "owner", "ws", GridSave{Finish: true}))
	assert.Empty(t, repo.events, "a finished setup isn't finished again")
}

func TestSaveGrid_FinishWithEveryProjectLeftEmpty(t *testing.T) {
	s, repo := gridFixture(t)
	repo.grid.Rules = nil
	require.NoError(t, s.SaveGrid(t.Context(), "owner", "ws", GridSave{Finish: true}))
	assert.Empty(t, repo.saved)
	assert.Equal(t, "owner", repo.finishedBy)
	assert.Len(t, repo.events, 1)
}

func TestSaveGrid_Refusals(t *testing.T) {
	tests := []struct {
		name  string
		actor string
		in    GridSave
		want  error
	}{
		{"a manager hides no channel", "manager", GridSave{Channels: []GridChannelSave{{ChannelID: "c-backend"}}}, apperrs.ErrForbidden},
		{"a manager doesn't finish the setup", "manager", GridSave{Finish: true}, apperrs.ErrForbidden},
		{"a project they don't manage is not found", "manager", GridSave{Projects: []GridProjectSave{{ProjectID: "p-hidden"}}}, apperrs.ErrNotFound},
		{"a role from elsewhere, or the Owner's", "owner", GridSave{Projects: []GridProjectSave{{ProjectID: "p-open", RoleIDs: []string{"role-owner"}}}}, apperrs.ErrNotFound},
		{"someone outside the workspace", "owner", GridSave{Projects: []GridProjectSave{{ProjectID: "p-open", UserIDs: []string{"stranger"}}}}, apperrs.ErrInvalid},
		{"an unknown channel", "owner", GridSave{Channels: []GridChannelSave{{ChannelID: "c-private"}}}, apperrs.ErrNotFound},
		{"a role from elsewhere on a channel", "owner", GridSave{Channels: []GridChannelSave{{ChannelID: "c-backend", HiddenRoleIDs: []string{"role-x"}}}}, apperrs.ErrNotFound},
		{"a member who manages nothing", "nobody", GridSave{Finish: true}, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo := gridFixture(t)
			require.ErrorIs(t, s.SaveGrid(t.Context(), tt.actor, "ws", tt.in), tt.want)
			assert.Nil(t, repo.saved, "nothing is written")
		})
	}
}

func TestSaveGrid_StorageFailures(t *testing.T) {
	s, repo := gridFixture(t)
	repo.setErr = errors.New("database is locked")
	require.ErrorIs(t, s.SaveGrid(t.Context(), "owner", "ws", GridSave{Finish: true}), repo.setErr)
	repo.setErr = nil
	repo.getErr = errors.New("database is locked")
	_, err := s.Grid(t.Context(), "owner", "ws")
	require.Error(t, err)
}

func summarize(rules []*Overwrite) []string {
	out := make([]string, 0, len(rules))
	for _, ow := range rules {
		out = append(out, fmt.Sprintf("%s %s %s allow %v deny %v", ow.ResourceType, ow.ResourceID, ruleTarget(ow), ow.Allow, ow.Deny))
	}
	return out
}

// fakeChannels resolves the channels it knows to their workspace.
type fakeChannels map[string]string

func (f fakeChannels) WorkspaceIDForChannel(_ context.Context, id string) (string, error) {
	ws, ok := f[id]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return ws, nil
}

func TestSetRules_OnAChannelAreTheOwners(t *testing.T) {
	s, repo := gridFixture(t)
	s.SetChannelWorkspaces(fakeChannels{"c-backend": "ws"})
	s.SetRoleWorkspaces(fakeRoleWorkspaces{"role-client": "ws"})
	read := []permissions.Action{permissions.ChannelsRead}

	require.NoError(t, s.SetRules(t.Context(), "owner", "channel", []string{"c-backend"}, Targets{RoleIDs: []string{"role-client"}}, read, StateDeny))
	rule, err := repo.GetRole(t.Context(), "channel", "c-backend", "role-client")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.ChannelsRead), rule.Deny)
	rules, err := s.ListRules(t.Context(), "owner", "channel", "c-backend")
	require.NoError(t, err)
	assert.Len(t, rules, 1)

	assert.True(t, s.ManagesRules(t.Context(), "owner", "channel", "c-backend"))
	assert.False(t, s.ManagesRules(t.Context(), "manager", "channel", "c-backend"))
	assert.False(t, s.ManagesRules(t.Context(), "", "channel", "c-backend"))
	assert.ErrorIs(t, s.SetRules(t.Context(), "manager", "channel", []string{"c-backend"}, Targets{UserIDs: []string{"client"}}, read, StateDeny), apperrs.ErrForbidden)
	assert.ErrorIs(t, s.SetRules(t.Context(), "owner", "channel", []string{"c-gone"}, Targets{UserIDs: []string{"client"}}, read, StateDeny), apperrs.ErrNotFound)
	assert.ErrorIs(t, s.SetRules(t.Context(), "owner", "channel", []string{"c-backend"}, Targets{UserIDs: []string{"client"}}, []permissions.Action{permissions.ChatWrite}, StateDeny), apperrs.ErrInvalid)
	_, err = s.ListRules(t.Context(), "owner", "ticket", "t-1")
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
}

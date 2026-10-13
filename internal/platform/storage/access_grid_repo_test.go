package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// TestAccessRepo_Grid_SavesAndReadsBack: one save sets a role rule and a person rule, empties another into nothing, and
// finishes the setup; the read gives back the roles but the Owner's with their holders, the public channels only, and
// the rules on the workspace's projects and channels alone.
func TestAccessRepo_Grid_SavesAndReadsBack(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now()
	mustCreateUser(t, s, "u-a", "alice")
	for _, r := range []*roles.Role{
		{ID: "role-owner", WorkspaceID: "workspace-default", Name: "Owner", IsOwnerRole: true},
		{ID: "role-dev", WorkspaceID: "workspace-default", Name: "Developer"},
	} {
		r.CreatedAt, r.UpdatedAt = now, now
		require.NoError(t, s.Roles.Create(ctx, r))
	}
	require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "u-a", WorkspaceID: "workspace-default", RoleID: "role-dev", CreatedAt: now}))
	_, err := s.db.Exec(`INSERT INTO conversations (id, workspace_id, kind, name, created_by, created_at, updated_at, private) VALUES
    ('c-pub', 'workspace-default', 'channel', 'backend', 'u-a', 1, 1, 0),
    ('c-priv', 'workspace-default', 'channel', 'leads', 'u-a', 1, 1, 1),
    ('c-dm', 'workspace-default', 'dm', '', 'u-a', 1, 1, 0)`)
	require.NoError(t, err)
	open := permissions.SetOf(permissions.ProjectsOpen)
	require.NoError(t, s.Access.Set(ctx, "project", "project-general", "u-a", open, nil))

	require.NoError(t, s.Access.SaveGrid(ctx, "workspace-default", []*access.Overwrite{
		{ResourceType: "project", ResourceID: "project-general", Target: access.TargetRole, RoleID: "role-dev", Allow: open},
		{ResourceType: "channel", ResourceID: "c-pub", Target: access.TargetRole, RoleID: "role-dev", Deny: permissions.SetOf(permissions.ChannelsRead)},
		{ResourceType: "project", ResourceID: "project-general", Target: access.TargetPerson, UserID: "u-a"},
	}, "u-a"))

	g, err := s.Access.Grid(ctx, "workspace-default")
	require.NoError(t, err)
	assert.True(t, g.Setup.Finished)
	assert.Equal(t, "alice", g.Setup.Login)
	require.NotNil(t, g.Setup.FinishedAt)
	assert.Equal(t, []access.GridRole{{ID: "role-dev", Name: "Developer", Holders: 1}}, g.Roles)
	require.Len(t, g.Channels, 1)
	assert.Equal(t, "c-pub", g.Channels[0].ID)
	assert.Equal(t, []string{"u-a"}, g.MemberIDs)
	assert.Contains(t, g.Projects, access.GridProject{ID: "project-general", Name: g.Projects[0].Name, Prefix: g.Projects[0].Prefix})
	got := map[string]string{}
	for _, ow := range g.Rules {
		got[ow.ResourceType+" "+ow.ResourceID+" "+ow.Target] = ow.Allow.String() + "|" + ow.Deny.String()
	}
	assert.Equal(t, map[string]string{"project project-general role": "projects:open|", "channel c-pub role": "|channels:read"}, got, "the emptied person rule is gone")

	_, err = s.Access.Grid(ctx, "ws-gone")
	require.Error(t, err)
}

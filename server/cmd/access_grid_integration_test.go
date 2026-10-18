package main

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/chat"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/workspace"
)

// TestIntegration_AccessGrid: the Owner's one save lists roles and people on projects, hides a public channel from a
// role and ends the setup, and every member's access follows in the same commit: the waiting member is in, the hidden
// channel is gone for its role, and the one frame the save sends reaches every member and nobody else.
func TestIntegration_AccessGrid(t *testing.T) {
	parallel(t)
	f := newPermFixture(t)
	s := f.svc
	ctx := context.Background()
	now := time.Now()
	require.NoError(t, f.store.Projects.Create(ctx, &workspace.Project{ID: "project-new", Name: "Storefront", Prefix: "STO", WorkspaceID: wsDefault, CreatedAt: now, UpdatedAt: now}))
	_, err := f.db.Exec(`UPDATE workspaces SET access_setup_finished = 0 WHERE id = ?`, wsDefault)
	require.NoError(t, err)

	me, err := s.tenancySvc.MyAccess(as(uPlain), wsDefault, uPlain)
	require.NoError(t, err)
	assert.False(t, me.AccessSetupFinished)
	assert.NotContains(t, projectIDsOf(me.Projects), "project-new")

	grid, err := s.accessSvc.Grid(as(uOwner), uOwner, wsDefault)
	require.NoError(t, err)
	assert.False(t, grid.Setup.Finished)
	assert.NotContains(t, roleIDsOf(grid.Roles), "role-owner", "the Owner sees every project anyway")
	assert.Contains(t, channelIDsOf(grid.Channels), f.channel.ID)

	require.NoError(t, s.accessSvc.SaveGrid(as(uOwner), uOwner, wsDefault, access.GridSave{
		Projects: []access.GridProjectSave{{ProjectID: "project-new", RoleIDs: []string{"role-plain"}, UserIDs: []string{uReader}}},
		Channels: []access.GridChannelSave{{ChannelID: f.channel.ID, HiddenRoleIDs: []string{"role-plain"}}},
		Finish:   true,
	}))

	me, err = s.tenancySvc.MyAccess(as(uPlain), wsDefault, uPlain)
	require.NoError(t, err)
	assert.True(t, me.AccessSetupFinished)
	assert.Contains(t, projectIDsOf(me.Projects), "project-new", "listed through their role")
	reader, err := s.tenancySvc.MyAccess(as(uReader), wsDefault, uReader)
	require.NoError(t, err)
	assert.Contains(t, projectIDsOf(reader.Projects), "project-new", "listed by name")
	cs, err := s.chatSvc.ListConversations(as(uPlain), wsDefault, uPlain)
	require.NoError(t, err)
	assert.False(t, slices.ContainsFunc(cs, func(c *chat.Conversation) bool { return c.ID == f.channel.ID }), "hidden from their role")

	grid, err = s.accessSvc.Grid(as(uOwner), uOwner, wsDefault)
	require.NoError(t, err)
	assert.True(t, grid.Setup.Finished)
	assert.Equal(t, uOwner, grid.Setup.Login)
	require.NotNil(t, grid.Setup.FinishedAt)
	for _, p := range grid.Projects {
		if p.ID == "project-new" {
			assert.Equal(t, []string{"role-plain"}, p.RoleIDs)
			assert.Equal(t, []string{uReader}, p.UserIDs)
		}
	}

	frame, err := json.Marshal(access.GrantEvent{ResourceType: "workspace", ResourceID: wsDefault, Target: access.TargetRole, ActorID: uOwner})
	require.NoError(t, err)
	audience := liveAudience{access: s.accessSvc, tickets: s.ticketsSvc, chat: s.chatSvc, deploy: s.deploySvc}
	for user, want := range map[string]bool{uPlain: true, uReader: true, uOwner: true, uOutsider: false} {
		assert.Equal(t, want, audience.allows(identity.WithActor(ctx, identity.Actor{ID: user}), access.TopicGrantChanged, json.RawMessage(frame)), user)
	}
	var sent int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM outbox WHERE topic = ? AND CAST(payload AS TEXT) LIKE '%"resource_type":"workspace"%'`, access.TopicGrantChanged).Scan(&sent))
	assert.Equal(t, 1, sent, "one frame for the whole save")

	// Taking a role off again, and a channel shown again, are rules back at Fallback: no row is left behind.
	require.NoError(t, s.accessSvc.SaveGrid(as(uOwner), uOwner, wsDefault, access.GridSave{
		Projects: []access.GridProjectSave{{ProjectID: "project-new"}},
		Channels: []access.GridChannelSave{{ChannelID: f.channel.ID}},
	}))
	var left int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM permission_overwrites WHERE resource_id IN (?, ?)`, "project-new", f.channel.ID).Scan(&left))
	assert.Zero(t, left)
}

// TestIntegration_AccessGrid_ChannelRuleFrames: a channel rule written one at a time reaches its role's holders and
// whoever manages it with channels:write, the Owner included, and nobody else.
func TestIntegration_AccessGrid_ChannelRuleFrames(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	s := f.svc
	require.NoError(t, s.accessSvc.SetRules(as(uOwner), uOwner, "channel", []string{f.channel.ID}, access.Targets{RoleIDs: []string{"role-plain"}}, readChannels, access.StateDeny))
	frame, err := json.Marshal(access.GrantEvent{ResourceType: "channel", ResourceID: f.channel.ID, Target: access.TargetRole, RoleID: "role-plain"})
	require.NoError(t, err)
	audience := liveAudience{access: s.accessSvc, tickets: s.ticketsSvc, chat: s.chatSvc, deploy: s.deploySvc}
	for user, want := range map[string]bool{uPlain: true, uOverwrite: true, uOwner: true, uWriter: true, uReader: false} {
		assert.Equal(t, want, audience.allows(identity.WithActor(context.Background(), identity.Actor{ID: user}), access.TopicGrantChanged, json.RawMessage(frame)), user)
	}
	require.ErrorIs(t, s.accessSvc.SetRules(as(uReader), uReader, "channel", []string{f.channel.ID}, access.Targets{RoleIDs: []string{"role-plain"}}, readChannels, access.StateFallback), apperrs.ErrForbidden)
	require.NoError(t, s.accessSvc.SetRules(as(uWriter), uWriter, "channel", []string{f.channel.ID}, access.Targets{RoleIDs: []string{"role-reader"}}, readChannels, access.StateDeny), "a writer of channels manages their rules")

	private, err := s.chatSvc.CreatePrivateChannel(as(uOwner), wsDefault, uOwner, "private", chat.KindChannel, nil)
	require.NoError(t, err)
	privateFrame, err := json.Marshal(access.GrantEvent{ResourceType: "channel", ResourceID: private.ID, Target: access.TargetRole, RoleID: "role-plain"})
	require.NoError(t, err)
	assert.False(t, audience.allows(as(uWriter), access.TopicGrantChanged, json.RawMessage(privateFrame)), "channel writers outside a private channel do not receive its rules")
	assert.True(t, audience.allows(as(uOwner), access.TopicGrantChanged, json.RawMessage(privateFrame)), "the Owner still reads every channel's rules")

	require.NoError(t, s.accessSvc.SetRules(as(uOwner), uOwner, "channel", []string{f.channel.ID}, access.Targets{RoleIDs: []string{"role-writer"}}, readChannels, access.StateDeny))
	assert.False(t, audience.allows(as(uWriter), access.TopicGrantChanged, json.RawMessage(frame)), "a public channel hidden from a writer does not send them other roles' rules")
	assert.True(t, audience.allows(as(uOwner), access.TopicGrantChanged, json.RawMessage(frame)))
}

// TestIntegration_MachineRuleFrames: a machine's rule written by the instance Owner reaches the Owner, the person it
// names and the holders of the role it names, and nobody else, and is stored and sent as one rule per target.
func TestIntegration_MachineRuleFrames(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	s := f.svc
	_, err := f.db.Exec(`INSERT INTO machines (id, name, stack_root, reported_hostname, first_seen, last_seen) VALUES ('mach-1', 'atlas', '/data/nexul', '', 1, 1)`)
	require.NoError(t, err)
	see := []permissions.Action{permissions.MachinesSee}
	require.ErrorIs(t, s.accessSvc.SetRules(as(uReader), uReader, "machine", []string{"mach-1"}, access.Targets{UserIDs: []string{uReader}}, see, access.StateAllow), apperrs.ErrNotFound,
		"a member who doesn't see the machine is told it isn't found")
	require.NoError(t, s.accessSvc.SetRules(as(uOwner), uOwner, "machine", []string{"mach-1"}, access.Targets{RoleIDs: []string{"role-reader"}, UserIDs: []string{uPlain}}, see, access.StateAllow))

	audience := liveAudience{access: s.accessSvc, tickets: s.ticketsSvc, chat: s.chatSvc, deploy: s.deploySvc}
	roleFrame, err := json.Marshal(access.GrantEvent{ResourceType: "machine", ResourceID: "mach-1", Target: access.TargetRole, RoleID: "role-reader"})
	require.NoError(t, err)
	personFrame, err := json.Marshal(access.GrantEvent{ResourceType: "machine", ResourceID: "mach-1", Target: access.TargetPerson, UserID: uPlain})
	require.NoError(t, err)
	for user, want := range map[string]bool{uOwner: true, uReader: true, uWriter: false, uPlain: false} {
		assert.Equal(t, want, audience.allows(as(user), access.TopicGrantChanged, json.RawMessage(roleFrame)), "role frame, "+user)
	}
	for user, want := range map[string]bool{uOwner: true, uPlain: true, uReader: false, uWriter: false} {
		assert.Equal(t, want, audience.allows(as(user), access.TopicGrantChanged, json.RawMessage(personFrame)), "person frame, "+user)
	}
	var sent int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM outbox WHERE topic = ? AND CAST(payload AS TEXT) LIKE '%"resource_type":"machine"%'`, access.TopicGrantChanged).Scan(&sent))
	assert.Equal(t, 2, sent)
	seen, err := s.accessSvc.MachinesSeen(as(uReader), uReader)
	require.NoError(t, err)
	assert.Equal(t, []string{"mach-1"}, seen)
}

var readChannels = []permissions.Action{permissions.ChannelsRead}

func projectIDsOf(ps []tenancy.MeProject) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.ProjectID)
	}
	return out
}

func roleIDsOf(roles []access.GridRole) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		out = append(out, r.ID)
	}
	return out
}

func channelIDsOf(cs []access.GridChannel) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.ID)
	}
	return out
}

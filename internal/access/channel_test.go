package access

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var (
	channelLead   = permissions.SetOf(permissions.ChannelsRead, permissions.ChannelsWrite, permissions.ChannelsPost)
	channelMember = permissions.SetOf(permissions.ChannelsRead, permissions.ChannelsPost)
	channelViewer = permissions.SetOf(permissions.ChannelsRead)
)

// channelFixture is workspace Acme: alice its Owner, lena a Lead who manages channels, bob a Member, vera a Viewer whose
// role doesn't post, wanda a Lead denied posting by name on #announce, and nina with no role. #announce takes posts
// from Leads only, and #secret is private to bob.
func channelFixture(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	roles := newFakeRoles()
	roles.set("ws", "alice", RolesHeld(HeldRole{ID: "role-owner", IsOwner: true}))
	roles.set("ws", "lena", RolesHeld(HeldRole{ID: "role-lead", Permissions: channelLead}))
	roles.set("ws", "wanda", RolesHeld(HeldRole{ID: "role-lead", Permissions: channelLead}))
	roles.set("ws", "bob", RolesHeld(HeldRole{ID: "role-member", Permissions: channelMember}))
	roles.set("ws", "vera", RolesHeld(HeldRole{ID: "role-viewer", Permissions: channelViewer}))
	roles.set("ws", "nina", RolesHeld())
	s := newService(repo, newFakeUsers())
	s.SetRoles(roles)
	s.SetNames(&fakeNames{})
	s.SetRoleWorkspaces(fakeRoleWorkspaces{"role-lead": "ws", "role-member": "ws", "role-viewer": "ws", "role-other": "ws-2"})
	s.SetChannelWorkspaces(fakeChannels{
		"c-announce": {WorkspaceID: "ws"},
		"c-general":  {WorkspaceID: "ws"},
		"c-secret":   {WorkspaceID: "ws", Private: true, MemberIDs: []string{"bob"}},
	})
	ctx := context.Background()
	post := permissions.SetOf(permissions.ChannelsPost)
	require.NoError(t, repo.SetRole(ctx, resourceTypeChannel, "c-announce", "role-lead", post, nil))
	require.NoError(t, repo.SetRole(ctx, resourceTypeChannel, "c-announce", "role-member", nil, post))
	require.NoError(t, repo.Set(ctx, resourceTypeChannel, "c-announce", "wanda", nil, post))
	return s, repo
}

func TestChannelPost_TheRolesThenTheChannelsRules(t *testing.T) {
	s, _ := channelFixture(t)

	tests := []struct {
		name, user, channel string
		// refused is the source sentence a refusal carries; empty when the post goes through.
		refused string
	}{
		{"the Owner posts whatever a rule says", "alice", "c-announce", ""},
		{"a role rule allows", "lena", "c-announce", ""},
		{"a role rule denies", "bob", "c-announce", "Denied on channel #announce for role Member"},
		{"the rule naming the person beats their role's", "wanda", "c-announce", "Denied on channel #announce for wanda"},
		{"no rule leaves it to the roles", "bob", "c-general", ""},
		{"a role that doesn't post", "vera", "c-general", "Not allowed: role Viewer doesn't grant it"},
		{"nobody posts with no role", "nina", "c-general", "Not allowed: nina holds no role"},
		{"outside the workspace", "outsider", "c-general", "Not found: outsider isn't a member of Acme"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.ChannelPost(t.Context(), tt.user, "ws", tt.channel)
			if tt.refused == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, apperrs.ErrForbidden)
			assert.Contains(t, err.Error(), "channels:post required ("+tt.refused+")")
		})
	}
}

func TestChannelRules_WrittenWithChannelsWriteUnderHoldWhatYouGive(t *testing.T) {
	post := []permissions.Action{permissions.ChannelsPost}
	read := []permissions.Action{permissions.ChannelsRead}

	tests := []struct {
		name    string
		actor   string
		channel string
		to      Targets
		actions []permissions.Action
		state   string
		want    error
	}{
		{"a holder of channels:write allows what they hold", "lena", "c-general", Targets{UserIDs: []string{"vera"}}, post, StateAllow, nil},
		{"they hide a channel from a role", "lena", "c-general", Targets{RoleIDs: []string{"role-viewer"}}, read, StateDeny, nil},
		{"adding a Deny is never limited", "wanda", "c-announce", Targets{UserIDs: []string{"bob"}}, post, StateDeny, nil},
		{"they can't allow what a rule denies them", "wanda", "c-announce", Targets{UserIDs: []string{"vera"}}, post, StateAllow, apperrs.ErrForbidden},
		{"nor lift a Deny on it", "wanda", "c-announce", Targets{RoleIDs: []string{"role-member"}}, post, StateFallback, apperrs.ErrForbidden},
		{"without channels:write they manage nothing", "bob", "c-general", Targets{UserIDs: []string{"vera"}}, read, StateDeny, apperrs.ErrForbidden},
		{"a channel they can't read is not found", "lena", "c-secret", Targets{UserIDs: []string{"bob"}}, post, StateDeny, apperrs.ErrNotFound},
		{"an unknown channel is not found", "lena", "c-gone", Targets{UserIDs: []string{"bob"}}, post, StateDeny, apperrs.ErrNotFound},
		{"a role of another workspace", "lena", "c-general", Targets{RoleIDs: []string{"role-other"}}, post, StateDeny, apperrs.ErrNotFound},
		{"only read and post are set on a channel", "lena", "c-general", Targets{UserIDs: []string{"bob"}}, []permissions.Action{permissions.ChannelsWrite}, StateDeny, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := channelFixture(t)
			err := s.SetRules(t.Context(), tt.actor, resourceTypeChannel, []string{tt.channel}, tt.to, tt.actions, tt.state)
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.want)
		})
	}

	s, _ := channelFixture(t)
	assert.True(t, s.ManagesRules(t.Context(), "lena", resourceTypeChannel, "c-announce"))
	assert.False(t, s.ManagesRules(t.Context(), "bob", resourceTypeChannel, "c-announce"))
	_, err := s.ListRules(t.Context(), "bob", resourceTypeChannel, "c-announce")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = s.ListRules(t.Context(), "lena", resourceTypeChannel, "c-secret")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a private channel they aren't in")
	rules, err := s.ListRules(t.Context(), "lena", resourceTypeChannel, "c-announce")
	require.NoError(t, err)
	assert.Len(t, rules, 3)
}

func TestExplain_CoversChannels(t *testing.T) {
	s, _ := channelFixture(t)
	answers := func(e *Explanation) map[permissions.Action]string {
		out := map[permissions.Action]string{}
		for _, a := range e.Effective {
			out[a.Action] = a.Source.Sentence
			if !a.Allowed {
				out[a.Action] = "refused: " + a.Source.Sentence
			}
		}
		return out
	}

	got, err := s.Explain(t.Context(), "lena", resourceTypeChannel, "c-announce", "bob")
	require.NoError(t, err)
	assert.Equal(t, map[permissions.Action]string{
		permissions.ChannelsRead: "Read by every member",
		permissions.ChannelsPost: "refused: Denied on channel #announce for role Member",
	}, answers(got))
	got, err = s.Explain(t.Context(), "lena", resourceTypeChannel, "c-announce", "wanda")
	require.NoError(t, err)
	assert.Equal(t, "refused: Denied on channel #announce for wanda", answers(got)[permissions.ChannelsPost])
	assert.Equal(t, "Allowed on channel #announce for role Lead", got.Effective[1].Fallback.Source.Sentence, "her role's rule, under her own Fallback")

	got, err = s.Explain(t.Context(), "bob", resourceTypeChannel, "c-secret", "")
	require.NoError(t, err)
	assert.Equal(t, "Member of private channel #secret", answers(got)[permissions.ChannelsRead])

	_, err = s.Explain(t.Context(), "lena", resourceTypeChannel, "c-secret", "bob")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "explaining takes reading the channel")
	_, err = s.Explain(t.Context(), "bob", resourceTypeChannel, "c-general", "vera")
	require.ErrorIs(t, err, apperrs.ErrForbidden, "someone else's answers take channels:write")

	role, err := s.ExplainRole(t.Context(), "lena", resourceTypeChannel, "c-announce", "role-member")
	require.NoError(t, err)
	assert.Equal(t, map[permissions.Action]string{
		permissions.ChannelsRead: "Read by every member",
		permissions.ChannelsPost: "refused: Denied on channel #announce for role Member",
	}, answers(role))
	assert.Equal(t, "Allowed by role Member", role.Effective[1].Fallback.Source.Sentence)
}

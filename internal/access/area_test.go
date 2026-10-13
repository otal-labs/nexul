package access

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// newAreaService has alice own ws-default, the instance Owner's workspace; bob hold Developer in ws-a and Frontend in
// ws-b; and olga own ws-b, which makes her an Owner but not the instance Owner.
func newAreaService(bobsDeveloper permissions.Set) (*Service, *fakeRepo) {
	repo := newFakeRepo()
	roles := newFakeRoles()
	roles.set("ws-default", "alice", RolesHeld(HeldRole{ID: "role-owner", IsOwner: true}))
	roles.set("ws-a", "bob", RolesHeld(HeldRole{ID: "role-dev", Permissions: bobsDeveloper}))
	roles.set("ws-b", "bob", RolesHeld(HeldRole{ID: "role-fe"}))
	roles.set("ws-b", "olga", RolesHeld(HeldRole{ID: "role-owner-b", IsOwner: true}))
	s := newService(repo, newFakeUsers())
	s.SetRoles(roles)
	s.SetScopes(fakeScopes{workspaces: map[string][]string{"alice": {"ws-default"}, "bob": {"ws-a", "ws-b"}, "olga": {"ws-b"}}})
	s.SetRoleWorkspaces(fakeRoleWorkspaces{"role-dev": "ws-a", "role-fe": "ws-b", "role-owner": "ws-default"})
	s.SetInstanceAreas("ws-default", []permissions.InstanceArea{permissions.ConnectorArea("github", "GitHub")})
	return s, repo
}

// TestDecide_InstanceAreaOrder walks every combination of bob's roles granting dns:write, a rule on DNS and gateways for
// each of his two roles (from two workspaces) and one naming him: the person beats the roles, Allow wins between roles,
// and any rule that sets the action replaces the source.
func TestDecide_InstanceAreaOrder(t *testing.T) {
	states := []string{StateFallback, StateAllow, StateDeny}
	for _, base := range []bool{false, true} {
		for _, dev := range states {
			for _, fe := range states {
				for _, person := range states {
					name := fmt.Sprintf("base %v, Developer %s, Frontend %s, bob %s", base, dev, fe, person)
					t.Run(name, func(t *testing.T) {
						var granted permissions.Set
						if base {
							granted = permissions.SetOf(permissions.DNSWrite)
						}
						s, repo := newAreaService(granted)
						ctx := context.Background()
						rule := func(target, id, state string) {
							allow, deny := moveTo(nil, nil, permissions.DNSWrite, state)
							if target == TargetRole {
								require.NoError(t, repo.SetRole(ctx, resourceTypeInstanceArea, "dns", id, allow, deny))
								return
							}
							require.NoError(t, repo.Set(ctx, resourceTypeInstanceArea, "dns", id, allow, deny))
						}
						rule(TargetRole, "role-dev", dev)
						rule(TargetRole, "role-fe", fe)
						rule(TargetPerson, "bob", person)

						want := Source{Kind: SourceRole, EntityType: resourceTypeWorkspace, EntityID: "ws-b", TargetType: TargetRole, TargetID: "role-fe", State: StateDeny}
						if base {
							want = Source{Kind: SourceRole, EntityType: resourceTypeWorkspace, EntityID: "ws-a", TargetType: TargetRole, TargetID: "role-dev", State: StateAllow}
						}
						byRule := func(target, id, state string) Source {
							return Source{Kind: SourceRule, EntityType: resourceTypeInstanceArea, EntityID: "dns", TargetType: target, TargetID: id, State: state}
						}
						switch {
						case person != StateFallback:
							want = byRule(TargetPerson, "bob", person)
						case dev == StateAllow:
							want = byRule(TargetRole, "role-dev", StateAllow)
						case fe == StateAllow:
							want = byRule(TargetRole, "role-fe", StateAllow)
						case dev == StateDeny:
							want = byRule(TargetRole, "role-dev", StateDeny)
						case fe == StateDeny:
							want = byRule(TargetRole, "role-fe", StateDeny)
						}
						held, src, err := s.decideArea(ctx, "bob", "dns", permissions.DNSWrite, true)
						require.NoError(t, err)
						assert.Equal(t, want, src)
						assert.Equal(t, want.State == StateAllow, held)
						other, err := s.HoldsArea(ctx, "bob", permissions.ConnectorAreaKey("github"), permissions.DNSWrite)
						require.NoError(t, err)
						assert.Equal(t, base, other, "a rule on one area says nothing on another")
					})
				}
			}
		}
	}
}

func TestInstanceAreas_TheOwnerOfAnyWorkspaceIsAllowedWhateverTheRules(t *testing.T) {
	s, repo := newAreaService(nil)
	ctx := context.Background()
	require.NoError(t, repo.Set(ctx, resourceTypeInstanceArea, "dns", "olga", nil, permissions.SetOf(permissions.DNSWrite)))
	require.NoError(t, repo.SetRole(ctx, resourceTypeInstanceArea, "dns", "role-owner-b", nil, permissions.SetOf(permissions.DNSWrite)))
	held, src, err := s.decideArea(ctx, "olga", "dns", permissions.DNSWrite, true)
	require.NoError(t, err)
	assert.True(t, held)
	assert.Equal(t, SourceOwner, src.Kind)
	assert.Equal(t, "Owner of ws-b", s.namer(ctx).named(src).Sentence)
}

func TestSetRules_OnAnInstanceArea(t *testing.T) {
	dnsWrite := []permissions.Action{permissions.DNSWrite}
	cases := []struct {
		name     string
		actor    string
		area     string
		to       Targets
		actions  []permissions.Action
		want     error
		wantRule bool
	}{
		{"an Owner of another workspace is forbidden", "olga", "dns", Targets{UserIDs: []string{"bob"}}, dnsWrite, apperrs.ErrForbidden, false},
		{"anyone else is forbidden", "bob", "dns", Targets{UserIDs: []string{"bob"}}, dnsWrite, apperrs.ErrForbidden, false},
		{"an unknown area is not found", "alice", "billing", Targets{UserIDs: []string{"bob"}}, dnsWrite, apperrs.ErrNotFound, false},
		{"an action of another area is refused", "alice", "dns", Targets{UserIDs: []string{"bob"}}, []permissions.Action{permissions.ConnectorsWrite}, apperrs.ErrInvalid, false},
		{"a project-area action is refused", "alice", "dns", Targets{UserIDs: []string{"bob"}}, []permissions.Action{permissions.DocsRead}, apperrs.ErrInvalid, false},
		{"an unknown role is not found", "alice", "dns", Targets{RoleIDs: []string{"role-gone"}}, dnsWrite, apperrs.ErrNotFound, false},
		{"the instance Owner names a person", "alice", "dns", Targets{UserIDs: []string{"bob"}}, dnsWrite, nil, true},
		{"the instance Owner names a role of any workspace", "alice", "connector:github", Targets{RoleIDs: []string{"role-fe"}}, []permissions.Action{permissions.ConnectorsRead}, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, repo := newAreaService(nil)
			err := s.SetRules(context.Background(), tc.actor, resourceTypeInstanceArea, []string{tc.area}, tc.to, tc.actions, StateAllow)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
				assert.Empty(t, repo.events)
				return
			}
			require.NoError(t, err)
			require.Len(t, repo.events, 1)
			event := repo.events[0].Payload.(GrantEvent)
			assert.Equal(t, resourceTypeInstanceArea, event.ResourceType)
			assert.Equal(t, tc.area, event.ResourceID)
			held, err := s.HoldsArea(context.Background(), "bob", tc.area, tc.actions[0])
			require.NoError(t, err)
			assert.True(t, held, "the rule hands bob the area")
		})
	}
}

func TestListRules_OnAnInstanceArea(t *testing.T) {
	s, _ := newAreaService(nil)
	ctx := context.Background()
	require.NoError(t, s.SetRules(ctx, "alice", resourceTypeInstanceArea, []string{"dns"}, Targets{UserIDs: []string{"bob"}}, []permissions.Action{permissions.DNSRead}, StateAllow))

	rules, err := s.ListRules(ctx, "alice", resourceTypeInstanceArea, "dns")
	require.NoError(t, err)
	require.Len(t, rules, 1)
	_, err = s.ListRules(ctx, "bob", resourceTypeInstanceArea, "dns")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = s.ListRules(ctx, "alice", resourceTypeInstanceArea, "billing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestExplain_OnAnInstanceArea(t *testing.T) {
	s, _ := newAreaService(nil)
	ctx := context.Background()
	require.NoError(t, s.SetRules(ctx, "alice", resourceTypeInstanceArea, []string{"dns"}, Targets{UserIDs: []string{"bob"}}, []permissions.Action{permissions.DNSWrite}, StateAllow))
	require.NoError(t, s.SetRules(ctx, "alice", resourceTypeInstanceArea, []string{"connector:github"}, Targets{RoleIDs: []string{"role-fe"}}, []permissions.Action{permissions.ConnectorsRead}, StateAllow))

	t.Run("anyone explains their own, each answer with its source and its Fallback", func(t *testing.T) {
		out, err := s.Explain(ctx, "bob", resourceTypeInstanceArea, "dns", "")
		require.NoError(t, err)
		require.Len(t, out.Effective, 3)
		write := out.Effective[1]
		assert.Equal(t, permissions.DNSWrite, write.Action)
		assert.True(t, write.Allowed)
		assert.Equal(t, "Allowed on DNS and gateways for bob", write.Source.Sentence)
		assert.False(t, write.Fallback.Allowed)
		assert.Equal(t, "Not allowed: role role-fe doesn't grant it", write.Fallback.Source.Sentence)

		out, err = s.Explain(ctx, "bob", resourceTypeInstanceArea, "connector:github", "")
		require.NoError(t, err)
		assert.Equal(t, "Allowed on connector GitHub for role role-fe", out.Effective[0].Source.Sentence)
	})
	t.Run("only the instance Owner explains someone else's", func(t *testing.T) {
		_, err := s.Explain(ctx, "olga", resourceTypeInstanceArea, "dns", "bob")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		out, err := s.Explain(ctx, "alice", resourceTypeInstanceArea, "dns", "bob")
		require.NoError(t, err)
		assert.True(t, out.Effective[1].Allowed)
	})
	t.Run("a role's answers, with its rule and at Fallback", func(t *testing.T) {
		out, err := s.ExplainRole(ctx, "alice", resourceTypeInstanceArea, "connector:github", "role-fe")
		require.NoError(t, err)
		assert.True(t, out.Effective[0].Allowed)
		assert.False(t, out.Effective[0].Fallback.Allowed)
		_, err = s.ExplainRole(ctx, "bob", resourceTypeInstanceArea, "connector:github", "role-fe")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		_, err = s.ExplainRole(ctx, "alice", resourceTypeInstanceArea, "connector:github", "role-gone")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("an unknown area is not found", func(t *testing.T) {
		_, err := s.Explain(ctx, "bob", resourceTypeInstanceArea, "billing", "")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestRequireArea_ARefusalNamesItsSource(t *testing.T) {
	s, _ := newAreaService(nil)
	ctx := identity.WithActor(context.Background(), identity.Actor{ID: "bob"})
	require.NoError(t, s.SetRules(context.Background(), "alice", resourceTypeInstanceArea, []string{"dns"}, Targets{UserIDs: []string{"bob"}}, []permissions.Action{permissions.DNSRead}, StateAllow))

	require.NoError(t, s.RequireAnywhere(ctx, permissions.DNSRead))
	err := s.RequireAnywhere(ctx, permissions.DNSWrite)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), "dns:write required")
	require.NoError(t, s.RequireArea(context.Background(), "dns", permissions.DNSWrite), "the server's own calls pass")

	held, err := s.PermissionsAnywhere(context.Background(), "bob")
	require.NoError(t, err)
	assert.Equal(t, []string{"dns:read"}, held)
}

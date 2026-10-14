package access

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// fakeMachines lists the fixture's machines.
type fakeMachines []string

func (f fakeMachines) MachineIDs(context.Context) ([]string, error) { return f, nil }

// machineRoles resolves the fixture's roles to their workspace and what they hold.
type machineRoles struct {
	workspace map[string]string
	held      map[string]HeldRole
}

func (m machineRoles) WorkspaceIDForRole(_ context.Context, roleID string) (string, error) {
	ws, ok := m.workspace[roleID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return ws, nil
}

func (m machineRoles) HeldRole(_ context.Context, roleID string) (HeldRole, error) {
	if _, ok := m.workspace[roleID]; !ok {
		return HeldRole{}, apperrs.ErrNotFound
	}
	return m.held[roleID], nil
}

type machineNames map[string]string

func (m machineNames) Name(_ context.Context, entityType, id string) (string, error) {
	if name, ok := m[entityType+"/"+id]; ok {
		return name, nil
	}
	return "", apperrs.ErrNotFound
}

// newMachineService is alice the instance Owner (Owner of ws-default); bob a Developer in ws-a, whose bits the caller
// picks, and Frontend in ws-b; olga the Owner of ws-b, an Owner but not the instance Owner; carl holding Deploy to and
// Manage through Ops in ws-a; dana holding nothing in ws-a. Machines m1 and m2 exist.
func newMachineService(bobsDeveloper permissions.Set) (*Service, *fakeRepo) {
	repo := newFakeRepo()
	roles := newFakeRoles()
	roles.set("ws-default", "alice", RolesHeld(HeldRole{ID: "role-owner", IsOwner: true}))
	roles.set("ws-a", "bob", RolesHeld(HeldRole{ID: "role-dev", Permissions: bobsDeveloper}))
	roles.set("ws-b", "bob", RolesHeld(HeldRole{ID: "role-fe"}))
	roles.set("ws-b", "olga", RolesHeld(HeldRole{ID: "role-owner-b", IsOwner: true}))
	roles.set("ws-a", "carl", RolesHeld(HeldRole{ID: "role-ops", Permissions: permissions.SetOf(permissions.MachinesDeploy, permissions.MachinesWrite)}))
	roles.set("ws-a", "dana", RolesHeld(HeldRole{ID: "role-guest"}))
	s := newService(repo, newFakeUsers())
	s.SetRoles(roles)
	s.SetScopes(fakeScopes{workspaces: map[string][]string{
		"alice": {"ws-default"}, "bob": {"ws-a", "ws-b"}, "olga": {"ws-b"}, "carl": {"ws-a"}, "dana": {"ws-a"},
	}})
	s.SetRoleWorkspaces(machineRoles{
		workspace: map[string]string{"role-owner": "ws-default", "role-dev": "ws-a", "role-fe": "ws-b", "role-owner-b": "ws-b", "role-ops": "ws-a", "role-guest": "ws-a"},
		held:      map[string]HeldRole{"role-owner": {ID: "role-owner", IsOwner: true}, "role-fe": {ID: "role-fe"}, "role-ops": {ID: "role-ops", Permissions: permissions.SetOf(permissions.MachinesDeploy, permissions.MachinesWrite)}},
	})
	s.SetInstanceAreas("ws-default", nil)
	s.SetMachines(fakeMachines{"m1", "m2"})
	s.SetNames(machineNames{"machine/m1": "Atlas", "workspace/ws-default": "Acme", "workspace/ws-b": "Bravo", "person/bob": "bob", "person/olga": "olga",
		"role/role-dev": "Developer", "role/role-fe": "Frontend", "role/role-owner": "Owner", "role/role-owner-b": "Owner"})
	return s, repo
}

func machineRule(t *testing.T, repo *fakeRepo, target, id string, allow, deny []permissions.Action) {
	t.Helper()
	var err error
	if target == TargetRole {
		err = repo.SetRole(t.Context(), resourceTypeMachine, "m1", id, permissions.SetOf(allow...), permissions.SetOf(deny...))
	}
	if target == TargetPerson {
		err = repo.Set(t.Context(), resourceTypeMachine, "m1", id, permissions.SetOf(allow...), permissions.SetOf(deny...))
	}
	require.NoError(t, err)
}

func acts(a ...permissions.Action) []permissions.Action { return a }

// TestDecide_MachineOrder: the instance Owner is never denied; an Owner of another workspace is not special; See comes
// only from the machine's rules; the other actions come from the union of the roles' bits across workspaces, then the
// rules for those roles with Allow winning, then the rule naming the person. Each answer names what decided it, and a
// machine someone doesn't see is not found for every action.
func TestDecide_MachineOrder(t *testing.T) {
	see, deploy, manage := permissions.MachinesSee, permissions.MachinesDeploy, permissions.MachinesWrite
	ruleOn := func(target, id, state string) Source {
		return Source{Kind: SourceRule, EntityType: resourceTypeMachine, EntityID: "m1", TargetType: target, TargetID: id, State: state}
	}
	notOn := func(user string) Source {
		return Source{Kind: SourceNotOnMachine, EntityType: resourceTypeMachine, EntityID: "m1", TargetType: TargetPerson, TargetID: user, State: StateDeny}
	}
	tests := []struct {
		name   string
		bobDev permissions.Set
		rules  func(t *testing.T, repo *fakeRepo)
		user   string
		action permissions.Action
		want   bool
		src    Source
		err    error
	}{
		{name: "the instance Owner holds every action with no rule", user: "alice", action: manage, want: true,
			src: Source{Kind: SourceOwner, EntityType: resourceTypeWorkspace, EntityID: "ws-default", TargetType: TargetRole, TargetID: "role-owner", State: StateAllow}},
		{name: "the instance Owner is never denied", user: "alice", action: deploy, want: true,
			rules: func(t *testing.T, repo *fakeRepo) {
				machineRule(t, repo, TargetPerson, "alice", nil, acts(see, deploy))
				machineRule(t, repo, TargetRole, "role-owner", nil, acts(see, deploy))
			},
			src: Source{Kind: SourceOwner, EntityType: resourceTypeWorkspace, EntityID: "ws-default", TargetType: TargetRole, TargetID: "role-owner", State: StateAllow}},
		{name: "an Owner of another workspace does not see a machine with no rule", user: "olga", action: see, src: notOn("olga"), err: apperrs.ErrNotFound},
		{name: "an Owner of another workspace holds nothing by right, even on a machine they see", user: "olga", action: deploy,
			rules: func(t *testing.T, repo *fakeRepo) { machineRule(t, repo, TargetPerson, "olga", acts(see), nil) },
			src:   Source{Kind: SourceRole, EntityType: resourceTypeWorkspace, EntityID: "ws-b", TargetType: TargetRole, TargetID: "role-owner-b", State: StateDeny}, err: apperrs.ErrForbidden},
		{name: "no rule: a role holding every bit still does not see it", bobDev: permissions.SetOf(deploy, manage), user: "bob", action: deploy,
			src: notOn("bob"), err: apperrs.ErrNotFound},
		{name: "a rule for one of the roles lets them see it", user: "bob", action: see, want: true,
			rules: func(t *testing.T, repo *fakeRepo) { machineRule(t, repo, TargetRole, "role-fe", acts(see), nil) },
			src:   ruleOn(TargetRole, "role-fe", StateAllow)},
		{name: "seen, the role's bit in another workspace holds the action", bobDev: permissions.SetOf(deploy), user: "bob", action: deploy, want: true,
			rules: func(t *testing.T, repo *fakeRepo) { machineRule(t, repo, TargetRole, "role-fe", acts(see), nil) },
			src:   Source{Kind: SourceRole, EntityType: resourceTypeWorkspace, EntityID: "ws-a", TargetType: TargetRole, TargetID: "role-dev", State: StateAllow}},
		{name: "seen, no bit and no rule on the action is a refusal", user: "bob", action: deploy, err: apperrs.ErrForbidden,
			rules: func(t *testing.T, repo *fakeRepo) { machineRule(t, repo, TargetRole, "role-fe", acts(see), nil) },
			src:   Source{Kind: SourceRole, EntityType: resourceTypeWorkspace, EntityID: "ws-b", TargetType: TargetRole, TargetID: "role-fe", State: StateDeny}},
		{name: "a rule for a role of one workspace allows what the other lacks", user: "bob", action: deploy, want: true,
			rules: func(t *testing.T, repo *fakeRepo) {
				machineRule(t, repo, TargetRole, "role-dev", acts(see), nil)
				machineRule(t, repo, TargetRole, "role-fe", acts(deploy), nil)
			},
			src: ruleOn(TargetRole, "role-fe", StateAllow)},
		{name: "Allow wins between the roles' rules", bobDev: permissions.SetOf(deploy), user: "bob", action: deploy, want: true,
			rules: func(t *testing.T, repo *fakeRepo) {
				machineRule(t, repo, TargetRole, "role-dev", acts(see), acts(deploy))
				machineRule(t, repo, TargetRole, "role-fe", acts(deploy), nil)
			},
			src: ruleOn(TargetRole, "role-fe", StateAllow)},
		{name: "a Deny on a role beats the bit the role gives", bobDev: permissions.SetOf(deploy), user: "bob", action: deploy, err: apperrs.ErrForbidden,
			rules: func(t *testing.T, repo *fakeRepo) {
				machineRule(t, repo, TargetRole, "role-dev", acts(see), acts(deploy))
			},
			src: ruleOn(TargetRole, "role-dev", StateDeny)},
		{name: "the rule naming the person beats the roles' rules", bobDev: permissions.SetOf(deploy), user: "bob", action: deploy, err: apperrs.ErrForbidden,
			rules: func(t *testing.T, repo *fakeRepo) {
				machineRule(t, repo, TargetRole, "role-fe", acts(see, deploy), nil)
				machineRule(t, repo, TargetPerson, "bob", nil, acts(deploy))
			},
			src: ruleOn(TargetPerson, "bob", StateDeny)},
		{name: "a person's Allow of See beats a role's Deny of it", user: "bob", action: see, want: true,
			rules: func(t *testing.T, repo *fakeRepo) {
				machineRule(t, repo, TargetRole, "role-fe", nil, acts(see))
				machineRule(t, repo, TargetPerson, "bob", acts(see), nil)
			},
			src: ruleOn(TargetPerson, "bob", StateAllow)},
		{name: "a person's Deny of See hides it whatever the roles say", bobDev: permissions.SetOf(deploy), user: "bob", action: deploy, err: apperrs.ErrNotFound,
			rules: func(t *testing.T, repo *fakeRepo) {
				machineRule(t, repo, TargetRole, "role-fe", acts(see, deploy), nil)
				machineRule(t, repo, TargetPerson, "bob", nil, acts(see))
			},
			src: ruleOn(TargetPerson, "bob", StateDeny)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo := newMachineService(tt.bobDev)
			if tt.rules != nil {
				tt.rules(t, repo)
			}
			ctx := context.Background()
			held, src, err := s.Decide(ctx, tt.user, resourceTypeMachine, "m1", tt.action)
			require.NoError(t, err)
			assert.Equal(t, tt.want, held)
			assert.Equal(t, tt.src, src)

			err = s.RequireMachine(identity.WithActor(ctx, identity.Actor{ID: tt.user}), "m1", tt.action)
			if tt.err == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.err)
		})
	}
}

func TestDecide_MachineANamedRefusalAndAnUnknownMachine(t *testing.T) {
	s, repo := newMachineService(permissions.SetOf(permissions.MachinesDeploy))
	machineRule(t, repo, TargetRole, "role-dev", acts(permissions.MachinesSee), acts(permissions.MachinesDeploy))
	ctx := identity.WithActor(context.Background(), identity.Actor{ID: "bob"})

	err := s.RequireMachine(ctx, "m1", permissions.MachinesDeploy)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), "Denied on machine Atlas for role Developer")
	err = s.RequireMachine(identity.WithActor(context.Background(), identity.Actor{ID: "olga"}), "m1", permissions.MachinesSee)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.ErrorIs(t, s.RequireMachine(identity.WithActor(context.Background(), identity.Actor{ID: "alice"}), "m-gone", permissions.MachinesSee), apperrs.ErrNotFound,
		"the instance Owner is told a machine that doesn't exist isn't found")
	require.ErrorIs(t, s.RequireMachine(ctx, "m1", permissions.MachinesCreate), apperrs.ErrInvalid, "Add machines is an instance area's, not a machine's")
	require.NoError(t, s.RequireMachine(context.Background(), "m1", permissions.MachinesDelete), "the server's own calls pass")
}

func TestMachineRules_NoRuleMeansNotFound(t *testing.T) {
	everything := permissions.SetOf(permissions.MachinesDeploy, permissions.MachinesWrite, permissions.MachinesDelete)
	s, repo := newMachineService(everything)
	ctx := context.Background()
	bob := identity.WithActor(ctx, identity.Actor{ID: "bob"})

	_, err := s.MachinePermissions(ctx, "bob", "m1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	for _, a := range permissions.MachineActions {
		require.ErrorIs(t, s.RequireMachine(bob, "m1", a), apperrs.ErrNotFound, a)
	}
	seen, err := s.MachinesSeen(ctx, "bob")
	require.NoError(t, err)
	assert.Empty(t, seen)
	_, err = s.MachinePermissions(ctx, "olga", "m1")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "an Owner of another workspace is no different")

	machineRule(t, repo, TargetRole, "role-fe", acts(permissions.MachinesSee), nil)
	held, err := s.MachinePermissions(ctx, "bob", "m1")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.MachinesSee, permissions.MachinesDeploy, permissions.MachinesWrite, permissions.MachinesDelete), held,
		"seen, the roles' bits hold the rest")
	seen, err = s.MachinesSeen(ctx, "bob")
	require.NoError(t, err)
	assert.Equal(t, []string{"m1"}, seen, "the machine with a rule, not m2")
	require.NoError(t, repo.Set(ctx, resourceTypeMachine, "m2", "bob", permissions.SetOf(permissions.MachinesSee), nil))
	seen, err = s.MachinesSeen(ctx, "bob")
	require.NoError(t, err)
	assert.Equal(t, []string{"m1", "m2"}, seen, "one by their role, one by name")

	all, err := s.MachinesSeen(ctx, "alice")
	require.NoError(t, err)
	assert.Equal(t, []string{"m1", "m2"}, all, "the instance Owner sees every machine")
	held, err = s.MachinePermissions(ctx, "alice", "m2")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.MachineActions...), held)
}

func TestMachineRules_OnlyTheInstanceOwnerAndManageHoldersWrite(t *testing.T) {
	see, deploy, del := permissions.MachinesSee, permissions.MachinesDeploy, permissions.MachinesDelete
	seeing := func(t *testing.T, repo *fakeRepo) {
		machineRule(t, repo, TargetPerson, "carl", acts(see), nil)
		machineRule(t, repo, TargetPerson, "olga", acts(see), nil)
	}
	tests := []struct {
		name    string
		setup   func(t *testing.T, repo *fakeRepo)
		actor   string
		machine string
		to      Targets
		actions []permissions.Action
		state   string
		want    error
	}{
		{name: "the instance Owner allows See to a person", actor: "alice", machine: "m1", to: Targets{UserIDs: []string{"dana"}}, actions: acts(see), state: StateAllow},
		{name: "the instance Owner names a role of any workspace", actor: "alice", machine: "m1", to: Targets{RoleIDs: []string{"role-fe"}}, actions: acts(see, deploy), state: StateAllow},
		{name: "the instance Owner is told an unknown machine isn't found", actor: "alice", machine: "m-gone", to: Targets{UserIDs: []string{"dana"}}, actions: acts(see), state: StateAllow, want: apperrs.ErrNotFound},
		{name: "a Manage holder allows what they hold", setup: seeing, actor: "carl", machine: "m1", to: Targets{UserIDs: []string{"dana"}}, actions: acts(see, deploy), state: StateAllow},
		{name: "a Manage holder can't allow what they don't hold", setup: seeing, actor: "carl", machine: "m1", to: Targets{UserIDs: []string{"dana"}}, actions: acts(del), state: StateAllow, want: apperrs.ErrForbidden},
		{name: "a Manage holder can't lift a Deny on what they don't hold", setup: func(t *testing.T, repo *fakeRepo) {
			seeing(t, repo)
			machineRule(t, repo, TargetPerson, "dana", nil, acts(del))
		}, actor: "carl", machine: "m1", to: Targets{UserIDs: []string{"dana"}}, actions: acts(del), state: StateFallback, want: apperrs.ErrForbidden},
		{name: "a Manage holder can lift a Deny on what they hold", setup: func(t *testing.T, repo *fakeRepo) {
			seeing(t, repo)
			machineRule(t, repo, TargetPerson, "dana", nil, acts(deploy))
		}, actor: "carl", machine: "m1", to: Targets{UserIDs: []string{"dana"}}, actions: acts(deploy), state: StateFallback},
		{name: "adding a Deny is never limited", setup: seeing, actor: "carl", machine: "m1", to: Targets{UserIDs: []string{"dana"}}, actions: acts(del), state: StateDeny},
		{name: "someone who sees the machine without Manage is refused", setup: seeing, actor: "olga", machine: "m1", to: Targets{UserIDs: []string{"dana"}}, actions: acts(see), state: StateAllow, want: apperrs.ErrForbidden},
		{name: "an Owner of another workspace who doesn't see it is told it isn't found", actor: "olga", machine: "m1", to: Targets{UserIDs: []string{"dana"}}, actions: acts(see), state: StateAllow, want: apperrs.ErrNotFound},
		{name: "a Manage holder of one machine is not found on another", setup: seeing, actor: "carl", machine: "m2", to: Targets{UserIDs: []string{"dana"}}, actions: acts(see), state: StateAllow, want: apperrs.ErrNotFound},
		{name: "an unknown role is not found", actor: "alice", machine: "m1", to: Targets{RoleIDs: []string{"role-gone"}}, actions: acts(see), state: StateAllow, want: apperrs.ErrNotFound},
		{name: "a machine's rules refuse another entity's actions", actor: "alice", machine: "m1", to: Targets{UserIDs: []string{"dana"}}, actions: acts(permissions.DocsRead), state: StateAllow, want: apperrs.ErrInvalid},
		{name: "a machine's rules refuse Add machines, which an instance area hands out", actor: "alice", machine: "m1", to: Targets{UserIDs: []string{"dana"}}, actions: acts(permissions.MachinesCreate), state: StateAllow, want: apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo := newMachineService(nil)
			if tt.setup != nil {
				tt.setup(t, repo)
			}
			before := len(repo.events)
			err := s.SetRules(context.Background(), tt.actor, resourceTypeMachine, []string{tt.machine}, tt.to, tt.actions, tt.state)
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
				assert.Len(t, repo.events, before, "a refused write changes nothing")
				return
			}
			require.NoError(t, err)
			require.Len(t, repo.events, before+len(tt.to.UserIDs)+len(tt.to.RoleIDs))
			event := repo.events[len(repo.events)-1].Payload.(GrantEvent)
			assert.Equal(t, resourceTypeMachine, event.ResourceType)
			assert.Equal(t, tt.machine, event.ResourceID)
		})
	}
}

func TestMachineRules_ListAndSeeSomeoneElsesAnswers(t *testing.T) {
	s, repo := newMachineService(nil)
	ctx := context.Background()
	machineRule(t, repo, TargetPerson, "carl", acts(permissions.MachinesSee), nil)
	machineRule(t, repo, TargetPerson, "olga", acts(permissions.MachinesSee), nil)

	rules, err := s.ListRules(ctx, "carl", resourceTypeMachine, "m1")
	require.NoError(t, err)
	assert.Len(t, rules, 2)
	_, err = s.ListRules(ctx, "olga", resourceTypeMachine, "m1")
	require.ErrorIs(t, err, apperrs.ErrForbidden, "seeing the machine isn't managing it")
	_, err = s.ListRules(ctx, "dana", resourceTypeMachine, "m1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	out, err := s.Explain(ctx, "carl", resourceTypeMachine, "m1", "olga")
	require.NoError(t, err)
	require.Len(t, out.Effective, 4)
	assert.Equal(t, permissions.MachinesSee, out.Effective[0].Action)
	assert.True(t, out.Effective[0].Allowed)
	assert.Equal(t, "Allowed on machine Atlas for olga", out.Effective[0].Source.Sentence)
	assert.False(t, out.Effective[0].Fallback.Allowed)
	assert.Equal(t, "Not found: no rule lets olga see machine Atlas", out.Effective[0].Fallback.Source.Sentence)
	_, err = s.Explain(ctx, "olga", resourceTypeMachine, "m1", "carl")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	own, err := s.Explain(ctx, "olga", resourceTypeMachine, "m1", "")
	require.NoError(t, err)
	assert.True(t, own.Effective[0].Allowed)

	require.NoError(t, s.SetRules(ctx, "alice", resourceTypeMachine, []string{"m1"}, Targets{RoleIDs: []string{"role-fe"}}, acts(permissions.MachinesSee), StateAllow))
	role, err := s.ExplainRole(ctx, "carl", resourceTypeMachine, "m1", "role-fe")
	require.NoError(t, err)
	assert.True(t, role.Effective[0].Allowed)
	assert.False(t, role.Effective[0].Fallback.Allowed)
	_, err = s.ExplainRole(ctx, "dana", resourceTypeMachine, "m1", "role-fe")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

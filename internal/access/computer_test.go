package access

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// fakeComputers reads rules from repo as the query does: sam owns laptop, and a rule counts while its person shares with sam.
type fakeComputers struct {
	repo    *fakeRepo
	shares  map[string]bool
	err     error
	reads   int
	ended   []string
	endErr  error
	sharErr error
}

func (f *fakeComputers) ComputerAccess(_ context.Context, computerID, userID string) (string, permissions.Set, error) {
	f.reads++
	if f.err != nil {
		return "", nil, f.err
	}
	if computerID != "laptop" {
		return "", nil, apperrs.ErrNotFound
	}
	if userID == "sam" {
		return "sam", nil, nil
	}
	ow, err := f.repo.get(resourceTypeComputer, computerID, userID)
	if err != nil || !f.shares[userID] {
		return "", nil, apperrs.ErrNotFound
	}
	return "sam", ow.Allow, nil
}

func (f *fakeComputers) SharesWithOwner(_ context.Context, _, personID string) (bool, error) {
	return f.shares[personID], f.sharErr
}

func (f *fakeComputers) EndComputerRules(_ context.Context, event func(computerID, ownerID, userID string) eventbus.OutboxEvent) error {
	if f.endErr != nil {
		return f.endErr
	}
	for _, userID := range f.ended {
		f.repo.events = append(f.repo.events, event("laptop", "sam", userID))
	}
	return nil
}

func newComputerService(t *testing.T) (*Service, *fakeRepo, *fakeComputers) {
	t.Helper()
	repo := newFakeRepo()
	computers := &fakeComputers{repo: repo, shares: map[string]bool{"bob": true, "owner": true}}
	s := newService(repo, newFakeUsers(&User{ID: "sam", Login: "sam"}, &User{ID: "bob", Login: "bob"}))
	s.SetComputers(computers)
	s.SetNames(&fakeNames{})
	roles := newFakeRoles()
	roles.set("ws", "owner", RoleInfo{RoleID: "role-owner", IsOwnerRole: true})
	s.SetRoles(roles)
	return s, repo, computers
}

func TestComputerPermissions_RefusesWhatNoRuleGives(t *testing.T) {
	s, repo, computers := newComputerService(t)

	_, _, err := s.ComputerPermissions(context.Background(), "", "laptop")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	_, _, err = s.ComputerPermissions(context.Background(), "bob", " ")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, _, err = s.ComputerPermissions(context.Background(), "owner", "laptop")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "the workspace Owner holds nothing on a computer no rule shares with them")

	setOverwrite(repo, resourceTypeComputer, "laptop", "bob", permissions.SetOf(permissions.DocsRead))
	_, _, err = s.ComputerPermissions(context.Background(), "bob", "laptop")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a rule that allows no computer permission shares nothing")

	computers.err = errors.New("disk on fire")
	_, _, err = s.ComputerPermissions(context.Background(), "bob", "laptop")
	require.ErrorContains(t, err, "disk on fire")

	_, _, err = NewService(repo, newFakeUsers()).ComputerPermissions(context.Background(), "bob", "laptop")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "unwired, every computer is not found")
}

func TestComputerPermissions_RunBringsSeeAndReadsFreshEveryTime(t *testing.T) {
	s, repo, computers := newComputerService(t)
	setOverwrite(repo, resourceTypeComputer, "laptop", "bob", permissions.SetOf(permissions.ComputersCommands))
	ctx := s.memoized(context.Background())

	owner, held, err := s.ComputerPermissions(ctx, "bob", "laptop")
	require.NoError(t, err)
	assert.Equal(t, "sam", owner)
	assert.Equal(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersCommands), held)
	_, held, err = s.ComputerPermissions(ctx, "sam", "laptop")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.ComputerActions...), held)

	delete(repo.overwrites, key(resourceTypeComputer, "laptop", "bob"))
	_, _, err = s.ComputerPermissions(ctx, "bob", "laptop")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "the next check under the same memo sees the revoke")
	assert.Equal(t, 3, computers.reads, "one read per check, none remembered")
}

func TestComputerPermissions_ReadLogsIsOffByDefaultAndBringsSee(t *testing.T) {
	s, repo, _ := newComputerService(t)
	setOverwrite(repo, resourceTypeComputer, "laptop", "bob", permissions.SetOf(permissions.ComputersSee))
	_, held, err := s.ComputerPermissions(context.Background(), "bob", "laptop")
	require.NoError(t, err)
	assert.False(t, held.Has(permissions.ComputersLogs), "a share is See alone until the owner adds Read logs")

	setOverwrite(repo, resourceTypeComputer, "laptop", "bob", permissions.SetOf(permissions.ComputersLogs))
	_, held, err = s.ComputerPermissions(context.Background(), "bob", "laptop")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersLogs), held)
	_, held, err = s.ComputerPermissions(context.Background(), "sam", "laptop")
	require.NoError(t, err)
	assert.True(t, held.Has(permissions.ComputersLogs), "the owner always holds it")
}

func TestSetRules_OnAComputer_ReadLogsKeepsSeeAndGoesWhenSeeDoes(t *testing.T) {
	s, repo, _ := newComputerService(t)
	require.NoError(t, s.SetRules(context.Background(), "sam", resourceTypeComputer, []string{"laptop"}, Targets{UserIDs: []string{"bob"}}, []permissions.Action{permissions.ComputersLogs}, StateAllow))
	ow, err := repo.get(resourceTypeComputer, "laptop", "bob")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersLogs), ow.Allow)

	require.NoError(t, s.SetRules(context.Background(), "sam", resourceTypeComputer, []string{"laptop"}, Targets{UserIDs: []string{"bob"}}, []permissions.Action{permissions.ComputersSee}, StateFallback))
	_, err = repo.get(resourceTypeComputer, "laptop", "bob")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "See going takes Read logs with it")
}

func TestSetRules_OnAComputer_RefusesRolesDenyStrangersAndAnyoneButItsOwner(t *testing.T) {
	s, repo, computers := newComputerService(t)
	see := []permissions.Action{permissions.ComputersSee}
	tests := []struct {
		name  string
		actor string
		to    Targets
		state string
		want  error
	}{
		{"a role", "sam", Targets{RoleIDs: []string{"role-plain"}}, StateAllow, apperrs.ErrInvalid},
		{"a deny", "sam", Targets{UserIDs: []string{"bob"}}, StateDeny, apperrs.ErrInvalid},
		{"the owner naming themselves", "sam", Targets{UserIDs: []string{"sam"}}, StateAllow, apperrs.ErrInvalid},
		{"a person sharing no workspace", "sam", Targets{UserIDs: []string{"stranger"}}, StateAllow, apperrs.ErrInvalid},
		{"the workspace Owner", "owner", Targets{UserIDs: []string{"bob"}}, StateAllow, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.SetRules(context.Background(), tt.actor, resourceTypeComputer, []string{"laptop"}, tt.to, see, tt.state)
			require.ErrorIs(t, err, tt.want)
		})
	}
	err := s.SetRules(context.Background(), "sam", resourceTypeComputer, []string{"laptop"}, Targets{UserIDs: []string{"bob"}}, []permissions.Action{permissions.DocsRead}, StateAllow)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a grid permission on a computer")
	err = s.SetRules(context.Background(), "sam", resourceTypeDoc, []string{"d"}, Targets{UserIDs: []string{"bob"}}, see, StateAllow)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a computer permission on a doc")

	computers.sharErr = errors.New("locked")
	err = s.SetRules(context.Background(), "sam", resourceTypeComputer, []string{"laptop"}, Targets{UserIDs: []string{"bob"}}, see, StateAllow)
	require.ErrorContains(t, err, "locked")

	setOverwrite(repo, resourceTypeComputer, "laptop", "bob", permissions.SetOf(permissions.ComputersSee))
	err = s.SetRules(context.Background(), "bob", resourceTypeComputer, []string{"laptop"}, Targets{UserIDs: []string{"owner"}}, see, StateAllow)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a grantee can't re-share")
	assert.Empty(t, repo.events, "nothing refused was stored")
}

func TestSetRules_OnAComputer_KeepsSeeWithRunAndEmitsTheComputersEvent(t *testing.T) {
	s, repo, computers := newComputerService(t)
	computers.shares["stranger"] = false

	require.NoError(t, s.SetRules(context.Background(), "sam", resourceTypeComputer, []string{"laptop"}, Targets{UserIDs: []string{"bob"}}, []permissions.Action{permissions.ComputersAgents}, StateAllow))
	ow, err := repo.get(resourceTypeComputer, "laptop", "bob")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersAgents), ow.Allow)
	require.Len(t, repo.events, 1)
	assert.Equal(t, TopicComputerGrantChanged, repo.events[0].Topic)
	assert.Equal(t, ComputerGrantEvent{ComputerID: "laptop", OwnerID: "sam", UserID: "bob", Permissions: []string{"computers:run_agents", "computers:see"}, ActorID: "sam", MembersOnly: true}, repo.events[0].Payload)

	require.NoError(t, s.SetRules(context.Background(), "sam", resourceTypeComputer, []string{"laptop"}, Targets{UserIDs: []string{"bob"}}, []permissions.Action{permissions.ComputersSee}, StateFallback))
	_, err = repo.get(resourceTypeComputer, "laptop", "bob")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "See going takes the Runs with it")

	require.NoError(t, s.SetRules(context.Background(), "sam", resourceTypeComputer, []string{"laptop"}, Targets{UserIDs: []string{"stranger"}}, []permissions.Action{permissions.ComputersSee}, StateFallback),
		"taking off a person who shares no workspace any more is allowed")

	rules, err := s.ListRules(context.Background(), "sam", resourceTypeComputer, "laptop")
	require.NoError(t, err)
	assert.Empty(t, rules)
	_, err = s.ListRules(context.Background(), "owner", resourceTypeComputer, "laptop")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.True(t, s.ManagesRules(context.Background(), "sam", resourceTypeComputer, "laptop"))
	assert.False(t, s.ManagesRules(context.Background(), "bob", resourceTypeComputer, "laptop"))
}

func TestExplain_OnAComputer(t *testing.T) {
	s, repo, _ := newComputerService(t)
	setOverwrite(repo, resourceTypeComputer, "laptop", "bob", permissions.SetOf(permissions.ComputersSee))

	got, err := s.Explain(context.Background(), "sam", resourceTypeComputer, "laptop", "")
	require.NoError(t, err)
	require.Len(t, got.Effective, 4)
	for _, a := range got.Effective {
		assert.True(t, a.Allowed)
		assert.Equal(t, "Owner of this computer", a.Source.Sentence)
	}

	got, err = s.Explain(context.Background(), "bob", resourceTypeComputer, "laptop", "")
	require.NoError(t, err)
	assert.Equal(t, "Shared with bob by sam", got.Effective[0].Source.Sentence)
	assert.Equal(t, "Not shared with bob", got.Effective[0].Fallback.Source.Sentence)
	assert.False(t, got.Effective[1].Allowed)
	assert.Equal(t, SourceNotShared, got.Effective[1].Source.Kind)

	got, err = s.Explain(context.Background(), "sam", resourceTypeComputer, "laptop", "lena")
	require.NoError(t, err)
	assert.Equal(t, "Not shared with lena", got.Effective[0].Source.Sentence, "the owner explains anyone")

	_, err = s.Explain(context.Background(), "bob", resourceTypeComputer, "laptop", "sam")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a grantee explains only themselves")
	_, err = s.Explain(context.Background(), "owner", resourceTypeComputer, "laptop", "")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.ExplainRole(context.Background(), "sam", resourceTypeComputer, "laptop", "role-plain")
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a computer has no role answers")
}

func TestHandleMembershipEnded_EndsTheRulesNoLongerLive(t *testing.T) {
	s, repo, computers := newComputerService(t)
	computers.ended = []string{"bob"}
	require.NoError(t, s.HandleMembershipEnded(context.Background(), eventbus.Event{}))
	require.Len(t, repo.events, 1)
	assert.Equal(t, ComputerGrantEvent{ComputerID: "laptop", OwnerID: "sam", UserID: "bob", Permissions: []string{}, MembersOnly: true}, repo.events[0].Payload)

	computers.endErr = errors.New("busy")
	require.ErrorContains(t, s.HandleMembershipEnded(context.Background(), eventbus.Event{}), "busy")
	require.NoError(t, NewService(repo, newFakeUsers()).HandleMembershipEnded(context.Background(), eventbus.Event{}), "unwired, nothing to end")
}

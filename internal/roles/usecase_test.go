package roles

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var fixedNow = time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)

type fakeRepo struct {
	mu        sync.Mutex
	byID      map[string]*Role
	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error
	events    []eventbus.OutboxEvent
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[string]*Role{}}
}

func (f *fakeRepo) Create(_ context.Context, r *Role) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if _, ok := f.byID[r.ID]; ok {
		return apperrs.ErrConflict
	}
	cp := *r
	f.byID[r.ID] = &cp
	return nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (*Role, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	r, ok := f.byID[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (f *fakeRepo) List(_ context.Context, workspaceID string) ([]*Role, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Role
	for _, r := range f.byID {
		if r.WorkspaceID == workspaceID {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, r *Role, events ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	if _, ok := f.byID[r.ID]; !ok {
		return apperrs.ErrNotFound
	}
	cp := *r
	f.byID[r.ID] = &cp
	f.events = append(f.events, events...)
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.byID[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

// fakeMemberGate maps (workspaceID, userID) -> roleID, standing in for the tenancy domain's workspace_members lookup.
type fakeMemberGate struct {
	roleIDs map[string]string // memberKey(workspaceID, userID) -> roleID
	err     error
}

func memberKey(workspaceID, userID string) string {
	return workspaceID + "|" + userID
}

func newFakeMemberGate() *fakeMemberGate {
	return &fakeMemberGate{roleIDs: map[string]string{}}
}

func (g *fakeMemberGate) MemberRoleID(_ context.Context, workspaceID, userID string) (string, error) {
	if g.err != nil {
		return "", g.err
	}
	roleID, ok := g.roleIDs[memberKey(workspaceID, userID)]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return roleID, nil
}

// fakeHeld reads a member's grid the way access does, minus overrides: the Owner holds everything, anyone else
// their role's set.
type fakeHeld struct {
	repo    *fakeRepo
	members *fakeMemberGate
}

func (h fakeHeld) WorkspacePermissions(ctx context.Context, userID, workspaceID string) []string {
	roleID, err := h.members.MemberRoleID(ctx, workspaceID, userID)
	if err != nil {
		return []string{}
	}
	r, err := h.repo.Get(ctx, roleID)
	if err != nil {
		return []string{}
	}
	held := r.Permissions
	if r.IsOwnerRole {
		held = permissions.SetOf(permissions.AllActions()...)
	}
	out := make([]string, len(held))
	for i, a := range held {
		out[i] = string(a)
	}
	return out
}

func newTestService(repo *fakeRepo, members *fakeMemberGate) *Service {
	s := NewService(repo, members)
	s.SetPermissionGate(fakeHeld{repo: repo, members: members})
	s.now = func() time.Time { return fixedNow }
	return s
}

// seedOwner creates an Owner role in repo and binds actorUserID to it via members, returning the role id.
func seedOwner(t *testing.T, repo *fakeRepo, members *fakeMemberGate, workspaceID, actorUserID string) string {
	t.Helper()
	owner := &Role{ID: "role-owner", WorkspaceID: workspaceID, Name: "Owner", IsOwnerRole: true, CreatedAt: fixedNow, UpdatedAt: fixedNow}
	require.NoError(t, repo.Create(context.Background(), owner))
	members.roleIDs[memberKey(workspaceID, actorUserID)] = owner.ID
	return owner.ID
}

// seedRoleWithMask creates a non-Owner role holding mask and binds actorUserID to it via members, returning the role id.
func seedRoleWithMask(t *testing.T, repo *fakeRepo, members *fakeMemberGate, workspaceID, actorUserID string, mask permissions.Set) string {
	t.Helper()
	r := &Role{ID: "role-actor", WorkspaceID: workspaceID, Name: "Actor", Permissions: mask, CreatedAt: fixedNow, UpdatedAt: fixedNow}
	require.NoError(t, repo.Create(context.Background(), r))
	members.roleIDs[memberKey(workspaceID, actorUserID)] = r.ID
	return r.ID
}

func TestCreateOwnerRole(t *testing.T) {
	t.Run("empty workspace id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeMemberGate())
		_, err := s.CreateOwnerRole(context.Background(), "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.createErr = errors.New("db down")
		s := newTestService(repo, newFakeMemberGate())
		_, err := s.CreateOwnerRole(context.Background(), "ws-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.createErr)
	})
	t.Run("creates a protected, empty-mask Owner role", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeMemberGate())
		r, err := s.CreateOwnerRole(context.Background(), "ws-1")
		require.NoError(t, err)
		assert.Equal(t, "ws-1", r.WorkspaceID)
		assert.Equal(t, "Owner", r.Name)
		assert.True(t, r.IsOwnerRole)
		assert.Equal(t, permissions.Set(nil), r.Permissions)
		assert.Equal(t, fixedNow, r.CreatedAt)
	})
}

func TestCreate(t *testing.T) {
	t.Run("empty workspace id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeMemberGate())
		_, err := s.Create(context.Background(), "  ", "u-1", "Editors", nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeMemberGate())
		_, err := s.Create(context.Background(), "ws-1", "u-1", "  ", nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("actor with no role in the workspace is rejected", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeMemberGate())
		_, err := s.Create(context.Background(), "ws-1", "u-stranger", "Editors", nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("actor without roles:write is forbidden", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedRoleWithMask(t, repo, members, "ws-1", "u-1", permissions.SetOf(permissions.ProjectsWrite))
		s := newTestService(repo, members)
		_, err := s.Create(context.Background(), "ws-1", "u-1", "Editors", nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("actor with roles:write can create a custom role", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedRoleWithMask(t, repo, members, "ws-1", "u-1", permissions.SetOf(permissions.RolesWrite, permissions.ProjectsWrite))
		s := newTestService(repo, members)
		r, err := s.Create(context.Background(), "ws-1", "u-1", "Editors", permissions.SetOf(permissions.ProjectsWrite))
		require.NoError(t, err)
		assert.Equal(t, "Editors", r.Name)
		assert.False(t, r.IsOwnerRole)
		assert.True(t, r.Permissions.Has(permissions.ProjectsWrite))
	})
	t.Run("a role can hold a domain-declared verb beside read, write, and delete", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		perms := permissions.SetOf(permissions.PlaysRun, permissions.MemoriesClone, permissions.DocsThread)
		seedRoleWithMask(t, repo, members, "ws-1", "u-1", perms.With(permissions.RolesWrite))
		s := newTestService(repo, members)
		r, err := s.Create(context.Background(), "ws-1", "u-1", "Runners", perms)
		require.NoError(t, err)
		assert.True(t, r.Permissions.Has(permissions.PlaysRun))
		assert.True(t, r.Permissions.Has(permissions.MemoriesClone))
		assert.True(t, r.Permissions.Has(permissions.DocsThread))
	})
	t.Run("a roles:write holder cannot give a role a permission they lack", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedRoleWithMask(t, repo, members, "ws-1", "u-1", permissions.SetOf(permissions.RolesWrite))
		s := newTestService(repo, members)
		_, err := s.Create(context.Background(), "ws-1", "u-1", "Admins", permissions.SetOf(permissions.InstanceWrite))
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("the Owner gives a role any permission", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		s := newTestService(repo, members)
		r, err := s.Create(context.Background(), "ws-1", "u-owner", "Admins", permissions.SetOf(permissions.InstanceWrite))
		require.NoError(t, err)
		assert.True(t, r.Permissions.Has(permissions.InstanceWrite))
	})
	t.Run("owner bypasses roles:write by is_owner_role, not by holding the bit", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		s := newTestService(repo, members)
		r, err := s.Create(context.Background(), "ws-1", "u-owner", "Editors", nil)
		require.NoError(t, err)
		assert.Equal(t, "Editors", r.Name)
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		repo.createErr = errors.New("db down")
		s := newTestService(repo, members)
		_, err := s.Create(context.Background(), "ws-1", "u-owner", "Editors", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.createErr)
	})
}

func TestList(t *testing.T) {
	t.Run("empty workspace id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeMemberGate())
		_, err := s.List(context.Background(), "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.listErr = errors.New("db down")
		s := newTestService(repo, newFakeMemberGate())
		_, err := s.List(context.Background(), "ws-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.listErr)
	})
	t.Run("lists roles scoped to the workspace", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		s := newTestService(repo, members)
		got, err := s.List(context.Background(), "ws-1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "Owner", got[0].Name)
	})
}

func TestGet(t *testing.T) {
	t.Run("wrong workspace is not found", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		s := newTestService(repo, members)
		_, err := s.Get(context.Background(), "ws-other", "role-owner")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("returns the role in its own workspace", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		s := newTestService(repo, members)
		r, err := s.Get(context.Background(), "ws-1", "role-owner")
		require.NoError(t, err)
		assert.Equal(t, "Owner", r.Name)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("empty name is invalid", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		s := newTestService(repo, members)
		_, err := s.Update(context.Background(), "ws-1", "role-owner", "u-owner", "  ", nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("owner role can't be renamed or edited", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		s := newTestService(repo, members)
		_, err := s.Update(context.Background(), "ws-1", "role-owner", "u-owner", "Not Owner", nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("actor without roles:write is forbidden", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		editors := &Role{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors", CreatedAt: fixedNow, UpdatedAt: fixedNow}
		require.NoError(t, repo.Create(context.Background(), editors))
		seedRoleWithMask(t, repo, members, "ws-1", "u-1", nil)
		s := newTestService(repo, members)
		_, err := s.Update(context.Background(), "ws-1", "role-editors", "u-1", "Reviewers", nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("actor with roles:write updates a custom role", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		editors := &Role{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors", CreatedAt: fixedNow, UpdatedAt: fixedNow}
		require.NoError(t, repo.Create(context.Background(), editors))
		s := newTestService(repo, members)
		r, err := s.Update(context.Background(), "ws-1", "role-editors", "u-owner", "Reviewers", permissions.SetOf(permissions.ProjectsWrite))
		require.NoError(t, err)
		assert.Equal(t, "Reviewers", r.Name)
		assert.True(t, r.Permissions.Has(permissions.ProjectsWrite))
		require.Len(t, repo.events, 1)
		assert.Equal(t, TopicUpdated, repo.events[0].Topic)
		assert.Equal(t, RoleEvent{RoleID: "role-editors", WorkspaceID: "ws-1", ActorID: "u-owner"}, repo.events[0].Payload)
	})
	t.Run("an edit may keep a permission the editor lacks but never add one", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		admins := &Role{ID: "role-admins", WorkspaceID: "ws-1", Name: "Admins", Permissions: permissions.SetOf(permissions.InstanceWrite), CreatedAt: fixedNow, UpdatedAt: fixedNow}
		require.NoError(t, repo.Create(context.Background(), admins))
		seedRoleWithMask(t, repo, members, "ws-1", "u-1", permissions.SetOf(permissions.RolesWrite, permissions.DocsRead))
		s := newTestService(repo, members)

		_, err := s.Update(context.Background(), "ws-1", "role-admins", "u-1", "Admins", permissions.SetOf(permissions.InstanceWrite, permissions.DocsRead))
		require.NoError(t, err)
		_, err = s.Update(context.Background(), "ws-1", "role-admins", "u-1", "Admins", permissions.SetOf(permissions.InstanceWrite, permissions.AccountsWrite))
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
}

func TestDelete(t *testing.T) {
	t.Run("owner role can't be deleted", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		s := newTestService(repo, members)
		err := s.Delete(context.Background(), "ws-1", "role-owner", "u-owner")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("actor without roles:write is forbidden", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		editors := &Role{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors", CreatedAt: fixedNow, UpdatedAt: fixedNow}
		require.NoError(t, repo.Create(context.Background(), editors))
		seedRoleWithMask(t, repo, members, "ws-1", "u-1", nil)
		s := newTestService(repo, members)
		err := s.Delete(context.Background(), "ws-1", "role-editors", "u-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("actor with roles:write deletes a custom role", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "u-owner")
		editors := &Role{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors", CreatedAt: fixedNow, UpdatedAt: fixedNow}
		require.NoError(t, repo.Create(context.Background(), editors))
		s := newTestService(repo, members)
		require.NoError(t, s.Delete(context.Background(), "ws-1", "role-editors", "u-owner"))

		_, err := repo.Get(context.Background(), "role-editors")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestSetMemberGate(t *testing.T) {
	t.Run("wires the gate used for later permission checks", func(t *testing.T) {
		repo := newFakeRepo()
		s := NewService(repo, nil)
		s.now = func() time.Time { return fixedNow }
		members := newFakeMemberGate()
		s.SetMemberGate(members)
		seedOwner(t, repo, members, "ws-1", "u-owner")

		r, err := s.Create(context.Background(), "ws-1", "u-owner", "Editors", nil)
		require.NoError(t, err)
		assert.Equal(t, "Editors", r.Name)
	})
}

// cloneMember is one workspace the clone actor belongs to, with the role it holds there.
type cloneMember struct {
	workspaceID string
	owner       bool
	mask        permissions.Set
}

// newCloneFixture seeds ws-src's Owner role and a custom "Editors" role, then binds actor to one role per membership.
func newCloneFixture(t *testing.T, actor string, memberships ...cloneMember) (*Service, *fakeRepo) {
	t.Helper()
	repo, members := newFakeRepo(), newFakeMemberGate()
	editors := &Role{ID: "role-editors", WorkspaceID: "ws-src", Name: "Editors", Permissions: permissions.SetOf(permissions.DocsRead, permissions.MemoriesClone, permissions.PlaysRun), CreatedAt: fixedNow, UpdatedAt: fixedNow}
	require.NoError(t, repo.Create(context.Background(), editors))
	owner := &Role{ID: "role-owner-src", WorkspaceID: "ws-src", Name: "Owner", IsOwnerRole: true, CreatedAt: fixedNow, UpdatedAt: fixedNow}
	require.NoError(t, repo.Create(context.Background(), owner))
	for _, m := range memberships {
		r := &Role{ID: "role-" + actor + "-" + m.workspaceID, WorkspaceID: m.workspaceID, Name: "Actor", IsOwnerRole: m.owner, Permissions: m.mask, CreatedAt: fixedNow, UpdatedAt: fixedNow}
		require.NoError(t, repo.Create(context.Background(), r))
		members.roleIDs[memberKey(m.workspaceID, actor)] = r.ID
	}
	return newTestService(repo, members), repo
}

func TestClone_Errors(t *testing.T) {
	cloner := cloneMember{workspaceID: "ws-src", mask: permissions.SetOf(permissions.RolesClone)}
	writer := cloneMember{workspaceID: "ws-dst", mask: permissions.SetOf(permissions.RolesWrite, permissions.DocsRead, permissions.MemoriesClone, permissions.PlaysRun)}
	tests := []struct {
		name        string
		memberships []cloneMember
		sourceWS    string
		roleID      string
		targetWS    string
		want        error
	}{
		{"missing target workspace id is invalid", []cloneMember{cloner, writer}, "ws-src", "role-editors", "  ", apperrs.ErrInvalid},
		{"missing role is not found", []cloneMember{cloner, writer}, "ws-src", "role-missing", "ws-dst", apperrs.ErrNotFound},
		{"role from another workspace is not found", []cloneMember{cloner, writer}, "ws-dst", "role-editors", "ws-other", apperrs.ErrNotFound},
		{"the Owner role is refused", []cloneMember{cloner, writer}, "ws-src", "role-owner-src", "ws-dst", apperrs.ErrInvalid},
		{"the source workspace as target is refused", []cloneMember{cloner, writer}, "ws-src", "role-editors", "ws-src", apperrs.ErrInvalid},
		{"no roles:clone in the source is forbidden", []cloneMember{{workspaceID: "ws-src", mask: permissions.SetOf(permissions.RolesWrite)}, writer}, "ws-src", "role-editors", "ws-dst", apperrs.ErrForbidden},
		{"no roles:write in the target is forbidden", []cloneMember{cloner, {workspaceID: "ws-dst", mask: permissions.SetOf(permissions.RolesClone)}}, "ws-src", "role-editors", "ws-dst", apperrs.ErrForbidden},
		{"not a member of the target is forbidden", []cloneMember{cloner}, "ws-src", "role-editors", "ws-dst", apperrs.ErrForbidden},
		{"a role carrying permissions the target writer lacks is forbidden", []cloneMember{cloner, {workspaceID: "ws-dst", mask: permissions.SetOf(permissions.RolesWrite)}}, "ws-src", "role-editors", "ws-dst", apperrs.ErrForbidden},
		{"not a member of the source is not found", []cloneMember{writer}, "ws-src", "role-editors", "ws-dst", apperrs.ErrNotFound},
		{"no source workspace and no role id is invalid", []cloneMember{cloner, writer}, "", " ", "ws-dst", apperrs.ErrInvalid},
		{"no source workspace and a missing role is not found", []cloneMember{cloner, writer}, "", "role-missing", "ws-dst", apperrs.ErrNotFound},
		{"no source workspace hides the role from a non-member", []cloneMember{writer}, "", "role-editors", "ws-dst", apperrs.ErrNotFound},
		{"no source workspace hides the Owner role from a non-member", []cloneMember{writer}, "", "role-owner-src", "ws-dst", apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newCloneFixture(t, "u-1", tt.memberships...)
			_, err := s.Clone(t.Context(), tt.sourceWS, tt.roleID, tt.targetWS, "u-1")
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestClone_RepoErrorsPropagate(t *testing.T) {
	owners := []cloneMember{{workspaceID: "ws-src", owner: true}, {workspaceID: "ws-dst", owner: true}}
	t.Run("list error", func(t *testing.T) {
		s, repo := newCloneFixture(t, "u-1", owners...)
		repo.listErr = errors.New("db down")
		_, err := s.Clone(t.Context(), "ws-src", "role-editors", "ws-dst", "u-1")
		require.ErrorIs(t, err, repo.listErr)
	})
	t.Run("create error", func(t *testing.T) {
		s, repo := newCloneFixture(t, "u-1", owners...)
		repo.createErr = errors.New("db down")
		_, err := s.Clone(t.Context(), "ws-src", "role-editors", "ws-dst", "u-1")
		require.ErrorIs(t, err, repo.createErr)
	})
}

func TestClone_CopiesNameAndPermissions(t *testing.T) {
	editorsAndRolesWrite := permissions.SetOf(permissions.RolesWrite, permissions.DocsRead, permissions.MemoriesClone, permissions.PlaysRun)
	tests := []struct {
		name        string
		memberships []cloneMember
	}{
		{"owner of both workspaces", []cloneMember{{workspaceID: "ws-src", owner: true}, {workspaceID: "ws-dst", owner: true}}},
		{"roles:clone in source and roles:write in target", []cloneMember{{workspaceID: "ws-src", mask: permissions.SetOf(permissions.RolesClone)}, {workspaceID: "ws-dst", mask: editorsAndRolesWrite}}},
		{"owner of the source, roles:write in the target", []cloneMember{{workspaceID: "ws-src", owner: true}, {workspaceID: "ws-dst", mask: editorsAndRolesWrite}}},
		{"roles:clone in the source, owner of the target", []cloneMember{{workspaceID: "ws-src", mask: permissions.SetOf(permissions.RolesClone)}, {workspaceID: "ws-dst", owner: true}}},
	}
	t.Run("no source workspace clones from the role's own", func(t *testing.T) {
		s, _ := newCloneFixture(t, "u-1", cloneMember{workspaceID: "ws-src", owner: true}, cloneMember{workspaceID: "ws-dst", owner: true})
		got, err := s.Clone(t.Context(), "", "role-editors", "ws-dst", "u-1")
		require.NoError(t, err)
		assert.Equal(t, "ws-dst", got.WorkspaceID)
		assert.Equal(t, "Editors", got.Name)
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo := newCloneFixture(t, "u-1", tt.memberships...)
			got, err := s.Clone(t.Context(), " ws-src ", "role-editors", " ws-dst ", "u-1")
			require.NoError(t, err)
			assert.Equal(t, "ws-dst", got.WorkspaceID)
			assert.Equal(t, "Editors", got.Name)
			assert.False(t, got.IsOwnerRole)
			assert.Equal(t, permissions.SetOf(permissions.DocsRead, permissions.MemoriesClone, permissions.PlaysRun), got.Permissions)
			assert.NotEqual(t, "role-editors", got.ID)
			assert.Equal(t, fixedNow, got.CreatedAt)

			stored, err := repo.Get(t.Context(), got.ID)
			require.NoError(t, err)
			assert.Equal(t, got, stored)
			source, err := repo.Get(t.Context(), "role-editors")
			require.NoError(t, err)
			assert.Equal(t, "ws-src", source.WorkspaceID)
		})
	}
}

func TestClone_NameCollision(t *testing.T) {
	tests := []struct {
		name  string
		taken []string
		want  string
	}{
		{"free name is kept", []string{"Reviewers"}, "Editors"},
		{"taken name gets (copy)", []string{"Editors"}, "Editors (copy)"},
		{"taken name matches case-insensitively", []string{"editors"}, "Editors (copy)"},
		{"taken (copy) gets (copy 2)", []string{"Editors", "Editors (copy)"}, "Editors (copy 2)"},
		{"counts past every taken copy", []string{"Editors", "Editors (copy)", "Editors (copy 2)", "Editors (copy 3)"}, "Editors (copy 4)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo := newCloneFixture(t, "u-1", cloneMember{workspaceID: "ws-src", owner: true}, cloneMember{workspaceID: "ws-dst", owner: true})
			for i, name := range tt.taken {
				require.NoError(t, repo.Create(t.Context(), &Role{ID: fmt.Sprintf("taken-%d", i), WorkspaceID: "ws-dst", Name: name}))
			}
			got, err := s.Clone(t.Context(), "ws-src", "role-editors", "ws-dst", "u-1")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Name)
		})
	}
}

func TestReadsForMembers(t *testing.T) {
	setup := func(t *testing.T) (*Service, *fakeMemberGate) {
		t.Helper()
		repo := newFakeRepo()
		members := newFakeMemberGate()
		seedOwner(t, repo, members, "ws-1", "member")
		return newTestService(repo, members), members
	}
	t.Run("a member lists and reads the workspace's roles", func(t *testing.T) {
		s, _ := setup(t)
		rs, err := s.ListForMember(t.Context(), "ws-1", "member")
		require.NoError(t, err)
		require.Len(t, rs, 1)
		got, err := s.GetForMember(t.Context(), "ws-1", "role-owner", "member")
		require.NoError(t, err)
		assert.Equal(t, "Owner", got.Name)
	})
	t.Run("someone from another workspace is told it does not exist", func(t *testing.T) {
		s, _ := setup(t)
		_, err := s.ListForMember(t.Context(), "ws-1", "outsider")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		_, err = s.GetForMember(t.Context(), "ws-1", "role-owner", "outsider")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("no caller is invalid", func(t *testing.T) {
		s, _ := setup(t)
		_, err := s.ListForMember(t.Context(), "ws-1", " ")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("a membership lookup failure surfaces as itself", func(t *testing.T) {
		s, members := setup(t)
		members.err = errors.New("db down")
		_, err := s.ListForMember(t.Context(), "ws-1", "member")
		require.Error(t, err)
		assert.NotErrorIs(t, err, apperrs.ErrNotFound)
	})
}

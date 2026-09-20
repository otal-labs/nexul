package roles

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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

func (f *fakeRepo) Update(_ context.Context, r *Role) error {
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
	roleIDs map[string]string // userID -> roleID (single workspace per test)
	err     error
}

func newFakeMemberGate() *fakeMemberGate {
	return &fakeMemberGate{roleIDs: map[string]string{}}
}

func (g *fakeMemberGate) MemberRoleID(_ context.Context, _, userID string) (string, error) {
	if g.err != nil {
		return "", g.err
	}
	roleID, ok := g.roleIDs[userID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return roleID, nil
}

func newTestService(repo *fakeRepo, members *fakeMemberGate) *Service {
	s := NewService(repo, members)
	s.now = func() time.Time { return fixedNow }
	return s
}

// seedOwner creates an Owner role in repo and binds actorUserID to it via members, returning the role id.
func seedOwner(t *testing.T, repo *fakeRepo, members *fakeMemberGate, workspaceID, actorUserID string) string {
	t.Helper()
	owner := &Role{ID: "role-owner", WorkspaceID: workspaceID, Name: "Owner", IsOwnerRole: true, CreatedAt: fixedNow, UpdatedAt: fixedNow}
	require.NoError(t, repo.Create(context.Background(), owner))
	members.roleIDs[actorUserID] = owner.ID
	return owner.ID
}

// seedRoleWithMask creates a non-Owner role holding mask and binds actorUserID to it via members, returning the role id.
func seedRoleWithMask(t *testing.T, repo *fakeRepo, members *fakeMemberGate, workspaceID, actorUserID string, mask permissions.Set) string {
	t.Helper()
	r := &Role{ID: "role-actor", WorkspaceID: workspaceID, Name: "Actor", Permissions: mask, CreatedAt: fixedNow, UpdatedAt: fixedNow}
	require.NoError(t, repo.Create(context.Background(), r))
	members.roleIDs[actorUserID] = r.ID
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
		seedRoleWithMask(t, repo, members, "ws-1", "u-1", permissions.SetOf(permissions.RolesWrite))
		s := newTestService(repo, members)
		r, err := s.Create(context.Background(), "ws-1", "u-1", "Editors", permissions.SetOf(permissions.ProjectsWrite))
		require.NoError(t, err)
		assert.Equal(t, "Editors", r.Name)
		assert.False(t, r.IsOwnerRole)
		assert.True(t, r.Permissions.Has(permissions.ProjectsWrite))
	})
	t.Run("a role can hold a domain-declared verb beside read, write, and delete", func(t *testing.T) {
		repo, members := newFakeRepo(), newFakeMemberGate()
		seedRoleWithMask(t, repo, members, "ws-1", "u-1", permissions.SetOf(permissions.RolesWrite))
		s := newTestService(repo, members)
		perms := permissions.SetOf(permissions.PlaysRun, permissions.MemoriesClone, permissions.DocsThread)
		r, err := s.Create(context.Background(), "ws-1", "u-1", "Runners", perms)
		require.NoError(t, err)
		assert.True(t, r.Permissions.Has(permissions.PlaysRun))
		assert.True(t, r.Permissions.Has(permissions.MemoriesClone))
		assert.True(t, r.Permissions.Has(permissions.DocsThread))
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

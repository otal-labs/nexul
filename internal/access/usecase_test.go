package access

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// fakeRepo is an in-memory access.Repo for use-case tests, generic over resource_type like the real permission_overwrites table.
type fakeRepo struct {
	mu         sync.Mutex
	overwrites map[string]*Overwrite // key: resourceType+"\x00"+resourceID+"\x00"+userID
	getErr     error
	deleteErr  error
	setErr     error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{overwrites: map[string]*Overwrite{}}
}

func key(resourceType, resourceID, userID string) string {
	return resourceType + "\x00" + resourceID + "\x00" + userID
}

func (f *fakeRepo) Get(_ context.Context, resourceType, resourceID, userID string) (*Overwrite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	ow, ok := f.overwrites[key(resourceType, resourceID, userID)]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return ow, nil
}

func (f *fakeRepo) ListByResource(_ context.Context, resourceType, resourceID string) ([]*Overwrite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Overwrite
	for _, ow := range f.overwrites {
		if ow.ResourceType == resourceType && ow.ResourceID == resourceID {
			out = append(out, ow)
		}
	}
	return out, nil
}

func (f *fakeRepo) Set(_ context.Context, resourceType, resourceID, userID string, allow, deny permissions.Set) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setErr != nil {
		return f.setErr
	}
	k := key(resourceType, resourceID, userID)
	if len(allow) == 0 && len(deny) == 0 {
		delete(f.overwrites, k)
		return nil
	}
	f.overwrites[k] = &Overwrite{ResourceType: resourceType, ResourceID: resourceID, UserID: userID, Allow: allow, Deny: deny}
	return nil
}

func (f *fakeRepo) DeleteByResource(_ context.Context, resourceType, resourceID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	for k, ow := range f.overwrites {
		if ow.ResourceType == resourceType && ow.ResourceID == resourceID {
			delete(f.overwrites, k)
		}
	}
	return nil
}

func (f *fakeRepo) HasAllowAny(_ context.Context, resourceType, userID string, action permissions.Action) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ow := range f.overwrites {
		if ow.ResourceType == resourceType && ow.UserID == userID && ow.Allow.Has(action) {
			return true, nil
		}
	}
	return false, nil
}

// setOverwrite is a small test helper mirroring the old SetGrant call shape.
func setOverwrite(f *fakeRepo, resourceType, resourceID, userID string, allow permissions.Set) {
	_ = f.Set(context.Background(), resourceType, resourceID, userID, allow, nil)
}

// fakeUsers is an in-memory access.Users.
type fakeUsers struct {
	mu      sync.Mutex
	byID    map[string]*User
	listErr error
}

func newFakeUsers(users ...*User) *fakeUsers {
	byID := map[string]*User{}
	for _, u := range users {
		byID[u.ID] = u
	}
	return &fakeUsers{byID: byID}
}

func (f *fakeUsers) GetUserByID(_ context.Context, id string) (*User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) ListUsers(_ context.Context) ([]*User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*User, 0, len(f.byID))
	for _, u := range f.byID {
		out = append(out, u)
	}
	return out, nil
}

// fakeRoles is an in-memory access.RoleResolver for HasPermission tests.
type fakeRoles struct {
	byKey map[string]RoleInfo // key: workspaceID+"\x00"+userID
}

func newFakeRoles() *fakeRoles {
	return &fakeRoles{byKey: map[string]RoleInfo{}}
}

func (f *fakeRoles) set(workspaceID, userID string, info RoleInfo) {
	f.byKey[workspaceID+"\x00"+userID] = info
}

func (f *fakeRoles) MemberRole(_ context.Context, workspaceID, userID string) (RoleInfo, error) {
	info, ok := f.byKey[workspaceID+"\x00"+userID]
	if !ok {
		return RoleInfo{}, apperrs.ErrNotFound
	}
	return info, nil
}

func newService(repo Repo, users Users) *Service {
	return NewService(repo, users)
}

func TestCan(t *testing.T) {
	users := newFakeUsers(
		&User{ID: "admin", CanCreateWorkspace: true},
		&User{ID: "alice"},
		&User{ID: "bob"},
	)
	tests := []struct {
		name   string
		userID string
		mask   permissions.Set
		action permissions.Action
		want   bool
	}{
		// Ticket 11: can_create_workspace must not bypass doc permission checks (see Can's doc comment); the equivalent bypass is the workspace-scoped Owner role, covered by TestHasPermission_Precedence and TestCan_ResolvesWorkspaceViaDocProject.
		{"instance admin bit alone does not bypass", "admin", nil, permissions.DocsRead, false},
		{"grant holds action", "alice", permissions.SetOf(permissions.DocsRead), permissions.DocsRead, true},
		{"grant lacks action", "alice", permissions.SetOf(permissions.DocsRead), permissions.DocsWrite, false},
		{"no grant denies", "bob", nil, permissions.DocsRead, false},
		{"empty user denies", "", nil, permissions.DocsRead, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			if tt.mask != nil {
				setOverwrite(repo, resourceTypeDoc, "doc-1", tt.userID, tt.mask)
			}
			s := newService(repo, users)
			got, err := s.Can(context.Background(), tt.userID, "doc-1", tt.action)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
	t.Run("unknown user is denied, not an error", func(t *testing.T) {
		s := newService(newFakeRepo(), newFakeUsers())
		got, err := s.Can(context.Background(), "ghost", "doc-1", permissions.DocsRead)
		require.NoError(t, err)
		assert.False(t, got)
	})
}

// fakeDocWorkspace is an in-memory access.DocWorkspaceResolver for tests.
type fakeDocWorkspace struct {
	byDoc map[string]string
	err   error
}

func (f *fakeDocWorkspace) WorkspaceIDForDoc(_ context.Context, docID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.byDoc[docID], nil
}

// TestCan_ResolvesWorkspaceViaDocProject covers ticket 10: once a DocWorkspaceResolver is wired, Can resolves the doc's workspace via its project so the role-mask layer applies to docs too, not just the resource-instance overwrite docs had before ticket 10.
func TestCan_ResolvesWorkspaceViaDocProject(t *testing.T) {
	const ws = "ws-1"

	t.Run("role mask alone grants the action once workspace resolves", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.DocsRead)})
		s := newService(repo, newFakeUsers(&User{ID: "alice"}))
		s.SetRoles(roles)
		s.SetDocWorkspaces(&fakeDocWorkspace{byDoc: map[string]string{"doc-1": ws}})

		got, err := s.Can(context.Background(), "alice", "doc-1", permissions.DocsRead)
		require.NoError(t, err)
		assert.True(t, got, "role mask on the doc's resolved workspace grants access")
	})

	t.Run("unresolved workspace falls back to resource-instance overwrite only", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.DocsRead)})
		s := newService(repo, newFakeUsers(&User{ID: "alice"}))
		s.SetRoles(roles)
		s.SetDocWorkspaces(&fakeDocWorkspace{err: apperrs.ErrNotFound})

		got, err := s.Can(context.Background(), "alice", "doc-1", permissions.DocsRead)
		require.NoError(t, err)
		assert.False(t, got, "no resource-instance overwrite and no resolvable workspace denies")
	})
}

// TestHasPermission_Precedence directly covers every most-specific-wins precedence case from ADR 0042 — the resilience-path logic the testing standard requires real, direct coverage for, not incidental line coverage.
func TestHasPermission_Precedence(t *testing.T) {
	const ws = "ws-1"

	t.Run("owner bypass beats everything, even an explicit resource deny", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "owner", RoleInfo{IsOwnerRole: true})
		// Even an explicit resource-instance deny can't override Owner.
		require.NoError(t, repo.Set(context.Background(), "doc", "doc-1", "owner", nil, permissions.SetOf(permissions.DocsRead)))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.True(t, s.HasPermission(context.Background(), "owner", ws, permissions.DocsRead, "doc", "doc-1"))
	})

	t.Run("owner bypass covers a domain-declared verb without holding it", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "owner", RoleInfo{IsOwnerRole: true})
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.True(t, s.HasPermission(context.Background(), "owner", ws, permissions.PlaysRun, "", ""))
		assert.True(t, s.HasPermission(context.Background(), "owner", ws, permissions.MemoriesClone, "", ""))
		assert.True(t, s.HasPermission(context.Background(), "owner", ws, permissions.DocsThread, "", ""))
	})

	t.Run("role mask alone grants the action", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.ProjectsWrite)})
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.True(t, s.HasPermission(context.Background(), "alice", ws, permissions.ProjectsWrite, "", ""))
	})

	t.Run("no role and no overwrite denies", func(t *testing.T) {
		s := newService(newFakeRepo(), newFakeUsers())
		s.SetRoles(newFakeRoles())
		assert.False(t, s.HasPermission(context.Background(), "ghost", ws, permissions.ProjectsWrite, "", ""))
	})

	t.Run("workspace-wide deny beats role allow", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.ProjectsWrite)})
		require.NoError(t, repo.Set(context.Background(), "workspace", ws, "alice", nil, permissions.SetOf(permissions.ProjectsWrite)))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.False(t, s.HasPermission(context.Background(), "alice", ws, permissions.ProjectsWrite, "", ""))
	})

	t.Run("workspace-wide allow grants even without a role mask bit", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{})
		require.NoError(t, repo.Set(context.Background(), "workspace", ws, "alice", permissions.SetOf(permissions.ProjectsWrite), nil))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.True(t, s.HasPermission(context.Background(), "alice", ws, permissions.ProjectsWrite, "", ""))
	})

	t.Run("resource-instance allow beats workspace-wide deny", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.ProjectsWrite)})
		require.NoError(t, repo.Set(context.Background(), "workspace", ws, "alice", nil, permissions.SetOf(permissions.ProjectsWrite)))
		require.NoError(t, repo.Set(context.Background(), "project", "proj-1", "alice", permissions.SetOf(permissions.ProjectsWrite), nil))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.True(t, s.HasPermission(context.Background(), "alice", ws, permissions.ProjectsWrite, "project", "proj-1"))
	})

	t.Run("resource-instance deny beats workspace-wide allow", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{})
		require.NoError(t, repo.Set(context.Background(), "workspace", ws, "alice", permissions.SetOf(permissions.ProjectsWrite), nil))
		require.NoError(t, repo.Set(context.Background(), "project", "proj-1", "alice", nil, permissions.SetOf(permissions.ProjectsWrite)))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.False(t, s.HasPermission(context.Background(), "alice", ws, permissions.ProjectsWrite, "project", "proj-1"))
	})

	t.Run("resource-instance deny beats role allow with no workspace overwrite at all", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.ProjectsWrite)})
		require.NoError(t, repo.Set(context.Background(), "project", "proj-1", "alice", nil, permissions.SetOf(permissions.ProjectsWrite)))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.False(t, s.HasPermission(context.Background(), "alice", ws, permissions.ProjectsWrite, "project", "proj-1"))
	})

	t.Run("no resourceType/resourceID skips the resource-instance layer entirely", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.ProjectsWrite)})
		require.NoError(t, repo.Set(context.Background(), "project", "proj-1", "alice", nil, permissions.SetOf(permissions.ProjectsWrite)))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.True(t, s.HasPermission(context.Background(), "alice", ws, permissions.ProjectsWrite, "", ""))
	})

	t.Run("empty workspaceID skips the role and workspace-wide layers, only the resource overwrite applies", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{IsOwnerRole: true}) // would bypass everything if workspaceID were honored
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.False(t, s.HasPermission(context.Background(), "alice", "", permissions.DocsRead, "doc", "doc-1"))

		setOverwrite(repo, "doc", "doc-1", "alice", permissions.SetOf(permissions.DocsRead))
		assert.True(t, s.HasPermission(context.Background(), "alice", "", permissions.DocsRead, "doc", "doc-1"))
	})

	t.Run("empty user id always denies", func(t *testing.T) {
		s := newService(newFakeRepo(), newFakeUsers())
		assert.False(t, s.HasPermission(context.Background(), "", ws, permissions.DocsRead, "doc", "doc-1"))
	})

	t.Run("nil RoleResolver behaves like no role found, not an error", func(t *testing.T) {
		s := newService(newFakeRepo(), newFakeUsers())
		assert.False(t, s.HasPermission(context.Background(), "alice", ws, permissions.ProjectsWrite, "", ""))
	})

	t.Run("a play deny beats a role holder's plays:run", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.PlaysRun)})
		require.NoError(t, repo.Set(context.Background(), resourceTypePlay, "play-1", "alice", nil, permissions.SetOf(permissions.PlaysRun)))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.False(t, s.HasPermission(context.Background(), "alice", ws, permissions.PlaysRun, resourceTypePlay, "play-1"))
	})

	t.Run("Owner bypass ignores a play deny (ticket 21)", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "owner", RoleInfo{IsOwnerRole: true})
		require.NoError(t, repo.Set(context.Background(), resourceTypePlay, "play-1", "owner", nil, permissions.SetOf(permissions.PlaysRun)))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.True(t, s.HasPermission(context.Background(), "owner", ws, permissions.PlaysRun, resourceTypePlay, "play-1"))
	})
}

// TestWorkspacePermissions covers the /me endpoint's permissions field: the whole grid, resolved through the same HasPermission precedence chain.
func TestWorkspacePermissions(t *testing.T) {
	const ws = "ws-1"

	t.Run("owner role grants the whole grid with an empty set", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "owner", RoleInfo{IsOwnerRole: true})
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		got := s.WorkspacePermissions(context.Background(), "owner", ws)
		want := make([]string, 0, 69)
		for _, a := range permissions.AllActions() {
			want = append(want, string(a))
		}
		assert.Equal(t, want, got)
	})

	t.Run("role set limits the list to its granted actions, in catalog order", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.MembersWrite, permissions.ProjectsWrite)})
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		got := s.WorkspacePermissions(context.Background(), "alice", ws)
		assert.Equal(t, []string{"projects:write", "members:write"}, got)
	})

	t.Run("role set granting only workspaces:write surfaces it alone", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.WorkspacesWrite)})
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		got := s.WorkspacePermissions(context.Background(), "alice", ws)
		assert.Equal(t, []string{"workspaces:write"}, got)
	})

	t.Run("no role and no overwrite is an empty list, not nil", func(t *testing.T) {
		s := newService(newFakeRepo(), newFakeUsers())
		s.SetRoles(newFakeRoles())
		got := s.WorkspacePermissions(context.Background(), "ghost", ws)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("a workspace-wide overwrite can grant beyond the role set", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{})
		require.NoError(t, repo.Set(context.Background(), "workspace", ws, "alice", permissions.SetOf(permissions.RolesWrite), nil))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		got := s.WorkspacePermissions(context.Background(), "alice", ws)
		assert.Equal(t, []string{"roles:write"}, got)
	})

	t.Run("a workspace-wide deny removes a role-granted action", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.ProjectsWrite, permissions.RolesWrite)})
		require.NoError(t, repo.Set(context.Background(), "workspace", ws, "alice", nil, permissions.SetOf(permissions.RolesWrite)))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		got := s.WorkspacePermissions(context.Background(), "alice", ws)
		assert.Equal(t, []string{"projects:write"}, got)
	})

	t.Run("empty user id is an empty list, not nil", func(t *testing.T) {
		s := newService(newFakeRepo(), newFakeUsers())
		got := s.WorkspacePermissions(context.Background(), "", ws)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}

func TestGrantCreator(t *testing.T) {
	t.Run("grants the full mask", func(t *testing.T) {
		repo := newFakeRepo()
		s := newService(repo, newFakeUsers())
		require.NoError(t, s.GrantCreator(context.Background(), "doc-1", "creator"))
		ow, err := repo.Get(context.Background(), resourceTypeDoc, "doc-1", "creator")
		require.NoError(t, err)
		assert.Equal(t, permissions.CreatorGrant, ow.Allow)
		assert.Equal(t, permissions.Set(nil), ow.Deny)
	})
	t.Run("missing ids are invalid", func(t *testing.T) {
		s := newService(newFakeRepo(), newFakeUsers())
		err := s.GrantCreator(context.Background(), "", "creator")
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
		err = s.GrantCreator(context.Background(), "doc-1", "")
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.setErr = errors.New("db down")
		s := newService(repo, newFakeUsers())
		err := s.GrantCreator(context.Background(), "doc-1", "creator")
		assert.ErrorIs(t, err, repo.setErr)
	})
}

func TestDeleteByDoc(t *testing.T) {
	t.Run("removes every overwrite on the doc", func(t *testing.T) {
		repo := newFakeRepo()
		s := newService(repo, newFakeUsers())
		require.NoError(t, s.GrantCreator(context.Background(), "doc-1", "creator"))
		require.NoError(t, s.DeleteByDoc(context.Background(), "doc-1"))
		_, err := repo.Get(context.Background(), resourceTypeDoc, "doc-1", "creator")
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.deleteErr = errors.New("db down")
		s := newService(repo, newFakeUsers())
		err := s.DeleteByDoc(context.Background(), "doc-1")
		assert.ErrorIs(t, err, repo.deleteErr)
	})
}

func TestSetGrants(t *testing.T) {
	// Ticket 11: no instance-wide owner bypass — every actor here proves access through a real permissions:write overwrite, the same way a workspace's Owner role would flow through HasPermission's role-mask layer.
	users := newFakeUsers(
		&User{ID: "manager"},
		&User{ID: "alice"},
	)
	t.Run("permissions:write holder grants read to alice across two docs", func(t *testing.T) {
		repo := newFakeRepo()
		setOverwrite(repo, resourceTypeDoc, "doc-1", "manager", permissions.SetOf(permissions.PermissionsWrite))
		setOverwrite(repo, resourceTypeDoc, "doc-2", "manager", permissions.SetOf(permissions.PermissionsWrite))
		s := newService(repo, users)
		err := s.SetGrants(context.Background(), "manager", []string{"doc-1", "doc-2"}, []string{"alice"}, []permissions.Action{permissions.DocsRead}, true)
		require.NoError(t, err)
		for _, docID := range []string{"doc-1", "doc-2"} {
			ow, err := repo.Get(context.Background(), resourceTypeDoc, docID, "alice")
			require.NoError(t, err)
			assert.Equal(t, permissions.SetOf(permissions.DocsRead), ow.Allow)
		}
	})
	t.Run("permissions:write holder can grant on their doc", func(t *testing.T) {
		repo := newFakeRepo()
		setOverwrite(repo, resourceTypeDoc, "doc-1", "alice", permissions.SetOf(permissions.PermissionsWrite))
		s := newService(repo, users)
		err := s.SetGrants(context.Background(), "alice", []string{"doc-1"}, []string{"bob"}, []permissions.Action{permissions.DocsWrite}, true)
		require.NoError(t, err)
		ow, err := repo.Get(context.Background(), resourceTypeDoc, "doc-1", "bob")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(permissions.DocsWrite), ow.Allow)
	})
	t.Run("non-manager cannot grant", func(t *testing.T) {
		repo := newFakeRepo()
		s := newService(repo, users)
		err := s.SetGrants(context.Background(), "alice", []string{"doc-1"}, []string{"bob"}, []permissions.Action{permissions.DocsRead}, true)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("revoke clears bits and deletes empty overwrite", func(t *testing.T) {
		repo := newFakeRepo()
		setOverwrite(repo, resourceTypeDoc, "doc-1", "manager", permissions.SetOf(permissions.PermissionsWrite))
		setOverwrite(repo, resourceTypeDoc, "doc-1", "alice", permissions.SetOf(permissions.DocsRead, permissions.DocsWrite))
		s := newService(repo, users)
		err := s.SetGrants(context.Background(), "manager", []string{"doc-1"}, []string{"alice"}, []permissions.Action{permissions.DocsRead, permissions.DocsWrite}, false)
		require.NoError(t, err)
		_, err = repo.Get(context.Background(), resourceTypeDoc, "doc-1", "alice")
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("revoke clears a stale deny bit for the same action", func(t *testing.T) {
		repo := newFakeRepo()
		setOverwrite(repo, resourceTypeDoc, "doc-1", "manager", permissions.SetOf(permissions.PermissionsWrite))
		require.NoError(t, repo.Set(context.Background(), resourceTypeDoc, "doc-1", "alice", nil, permissions.SetOf(permissions.DocsRead)))
		s := newService(repo, users)
		err := s.SetGrants(context.Background(), "manager", []string{"doc-1"}, []string{"alice"}, []permissions.Action{permissions.DocsRead}, true)
		require.NoError(t, err)
		ow, err := repo.Get(context.Background(), resourceTypeDoc, "doc-1", "alice")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(permissions.DocsRead), ow.Allow)
		assert.Equal(t, permissions.Set(nil), ow.Deny)
	})
	t.Run("no actor is unauthorized", func(t *testing.T) {
		s := newService(newFakeRepo(), users)
		err := s.SetGrants(context.Background(), "", []string{"doc-1"}, []string{"alice"}, []permissions.Action{permissions.DocsRead}, true)
		assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
	})
	t.Run("empty inputs are invalid", func(t *testing.T) {
		s := newService(newFakeRepo(), users)
		err := s.SetGrants(context.Background(), "manager", []string{}, []string{"alice"}, []permissions.Action{permissions.DocsRead}, true)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestListGrants(t *testing.T) {
	users := newFakeUsers(&User{ID: "manager"}, &User{ID: "alice"})
	t.Run("permissions:write holder lists overwrites, including their own grant", func(t *testing.T) {
		repo := newFakeRepo()
		setOverwrite(repo, resourceTypeDoc, "doc-1", "manager", permissions.SetOf(permissions.PermissionsWrite))
		setOverwrite(repo, resourceTypeDoc, "doc-1", "alice", permissions.SetOf(permissions.DocsRead))
		s := newService(repo, users)
		grants, err := s.ListGrants(context.Background(), "manager", "doc-1")
		require.NoError(t, err)
		require.Len(t, grants, 2)
		ids := []string{grants[0].UserID, grants[1].UserID}
		assert.ElementsMatch(t, []string{"manager", "alice"}, ids)
	})
	t.Run("permissions:write holder lists overwrites", func(t *testing.T) {
		repo := newFakeRepo()
		setOverwrite(repo, resourceTypeDoc, "doc-1", "alice", permissions.SetOf(permissions.PermissionsWrite))
		s := newService(repo, users)
		_, err := s.ListGrants(context.Background(), "alice", "doc-1")
		require.NoError(t, err)
	})
	t.Run("non-manager denied", func(t *testing.T) {
		repo := newFakeRepo()
		setOverwrite(repo, resourceTypeDoc, "doc-1", "alice", permissions.SetOf(permissions.DocsRead))
		s := newService(repo, users)
		_, err := s.ListGrants(context.Background(), "alice", "doc-1")
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

// fakePlayWorkspace is an in-memory access.PlayWorkspaceResolver for SetPlayGrants/ListPlayGrants tests.
type fakePlayWorkspace struct {
	byPlay map[string]string
}

func (f *fakePlayWorkspace) WorkspaceIDForPlay(_ context.Context, playID string) (string, error) {
	ws, ok := f.byPlay[playID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return ws, nil
}

func TestSetPlayGrants(t *testing.T) {
	const ws = "ws-1"
	users := newFakeUsers(&User{ID: "manager"}, &User{ID: "alice"})

	newPlaysService := func(repo *fakeRepo, roleGrants permissions.Set) *Service {
		roles := newFakeRoles()
		roles.set(ws, "manager", RoleInfo{Permissions: roleGrants})
		s := newService(repo, users)
		s.SetRoles(roles)
		s.SetPlayWorkspaces(&fakePlayWorkspace{byPlay: map[string]string{"play-1": ws}})
		return s
	}

	t.Run("plays:write holder denies a user's plays:run", func(t *testing.T) {
		repo := newFakeRepo()
		s := newPlaysService(repo, permissions.SetOf(permissions.PlaysWrite))
		err := s.SetPlayGrants(context.Background(), "manager", []string{"play-1"}, []string{"alice"}, []permissions.Action{permissions.PlaysRun}, false)
		require.NoError(t, err)
		ow, err := repo.Get(context.Background(), resourceTypePlay, "play-1", "alice")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(permissions.PlaysRun), ow.Deny)
		assert.Equal(t, permissions.Set(nil), ow.Allow)
	})

	t.Run("grant=true clears an existing deny instead of adding an allow", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.Set(context.Background(), resourceTypePlay, "play-1", "alice", nil, permissions.SetOf(permissions.PlaysRun)))
		s := newPlaysService(repo, permissions.SetOf(permissions.PlaysWrite))
		err := s.SetPlayGrants(context.Background(), "manager", []string{"play-1"}, []string{"alice"}, []permissions.Action{permissions.PlaysRun}, true)
		require.NoError(t, err)
		_, err = repo.Get(context.Background(), resourceTypePlay, "play-1", "alice")
		assert.True(t, errors.Is(err, apperrs.ErrNotFound), "clearing the only bit deletes the empty overwrite")
	})

	t.Run("rejects an action other than plays:run", func(t *testing.T) {
		repo := newFakeRepo()
		s := newPlaysService(repo, permissions.SetOf(permissions.PlaysWrite))
		err := s.SetPlayGrants(context.Background(), "manager", []string{"play-1"}, []string{"alice"}, []permissions.Action{permissions.PlaysWrite}, false)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})

	t.Run("without plays:write is forbidden", func(t *testing.T) {
		repo := newFakeRepo()
		s := newPlaysService(repo, nil)
		err := s.SetPlayGrants(context.Background(), "manager", []string{"play-1"}, []string{"alice"}, []permissions.Action{permissions.PlaysRun}, false)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

func TestListPlayGrants(t *testing.T) {
	const ws = "ws-1"
	users := newFakeUsers(&User{ID: "manager"}, &User{ID: "alice"})

	t.Run("plays:write holder lists a play's exclusions", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.Set(context.Background(), resourceTypePlay, "play-1", "alice", nil, permissions.SetOf(permissions.PlaysRun)))
		roles := newFakeRoles()
		roles.set(ws, "manager", RoleInfo{Permissions: permissions.SetOf(permissions.PlaysWrite)})
		s := newService(repo, users)
		s.SetRoles(roles)
		s.SetPlayWorkspaces(&fakePlayWorkspace{byPlay: map[string]string{"play-1": ws}})
		grants, err := s.ListPlayGrants(context.Background(), "manager", "play-1")
		require.NoError(t, err)
		require.Len(t, grants, 1)
		assert.Equal(t, permissions.SetOf(permissions.PlaysRun), grants[0].Deny)
	})

	t.Run("non plays:write holder denied", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{})
		s := newService(repo, users)
		s.SetRoles(roles)
		s.SetPlayWorkspaces(&fakePlayWorkspace{byPlay: map[string]string{"play-1": ws}})
		_, err := s.ListPlayGrants(context.Background(), "alice", "play-1")
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

func TestListUsers(t *testing.T) {
	users := newFakeUsers(
		&User{ID: "owner", CanCreateWorkspace: true},
		&User{ID: "manager", Login: "manager"},
		&User{ID: "plain", Login: "plain"},
	)
	t.Run("instance admin lists the directory", func(t *testing.T) {
		s := newService(newFakeRepo(), users)
		got, err := s.ListUsers(context.Background(), "owner")
		require.NoError(t, err)
		assert.Len(t, got, 3)
	})
	t.Run("permissions:write holder lists the directory", func(t *testing.T) {
		repo := newFakeRepo()
		setOverwrite(repo, resourceTypeDoc, "doc-1", "manager", permissions.SetOf(permissions.PermissionsWrite))
		s := newService(repo, users)
		got, err := s.ListUsers(context.Background(), "manager")
		require.NoError(t, err)
		assert.Len(t, got, 3)
	})
	t.Run("plain user denied", func(t *testing.T) {
		s := newService(newFakeRepo(), users)
		_, err := s.ListUsers(context.Background(), "plain")
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

func TestInstanceAdminID(t *testing.T) {
	t.Run("finds the instance admin", func(t *testing.T) {
		users := newFakeUsers(&User{ID: "owner", CanCreateWorkspace: true}, &User{ID: "alice"})
		s := newService(newFakeRepo(), users)
		id, err := s.InstanceAdminID(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "owner", id)
	})
	t.Run("no admin returns empty", func(t *testing.T) {
		s := newService(newFakeRepo(), newFakeUsers(&User{ID: "alice"}))
		id, err := s.InstanceAdminID(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "", id)
	})
}

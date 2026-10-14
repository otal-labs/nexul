package access

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// fakeRepo is an in-memory access.Repo for use-case tests, generic over resource_type like the real permission_overwrites table.
type fakeRepo struct {
	mu         sync.Mutex
	overwrites map[string]*Overwrite // key: resourceType+"\x00"+resourceID+"\x00"+userID
	getErr     error
	deleteErr  error
	setErr     error
	events     []eventbus.OutboxEvent
	reads      int
	// docProjects and docFolders are where each doc lives, for DocRuleTurns and DocRules, which a real repo joins in SQL.
	docProjects map[string]string
	docFolders  map[string]string
	// stackProjects is where each stack lives, for StackRuleTurns, which a real repo joins in SQL.
	stackProjects map[string]string
	// grid is what Grid hands back; saved and finishedBy are what SaveGrid was given.
	grid       *GridSource
	saved      []*Overwrite
	finishedBy string
}

// readCounts is how many overwrite and rule reads reached the fake, so a test can pin what a check costs.
func (f *fakeRepo) readCounts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads
}

func (f *fakeRepo) resetReads() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads = 0
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
	f.reads++
	return f.get(resourceType, resourceID, userID)
}

func (f *fakeRepo) get(resourceType, resourceID, userID string) (*Overwrite, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	ow, ok := f.overwrites[key(resourceType, resourceID, userID)]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return ow, nil
}

func (f *fakeRepo) RulesOfType(_ context.Context, resourceType, userID string) (map[string]*Rules, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.getErr != nil {
		return nil, f.getErr
	}
	out := map[string]*Rules{}
	rules := func(id string) *Rules {
		if out[id] == nil {
			out[id] = &Rules{}
		}
		return out[id]
	}
	for _, ow := range f.overwrites {
		if ow.ResourceType == resourceType && ow.UserID == userID {
			rules(ow.ResourceID).Person = ow
		}
		// Every role's rule comes back, as though the person held each: the check keeps only the roles they hold.
		if ow.ResourceType == resourceType && ow.Target == TargetRole {
			r := rules(ow.ResourceID)
			r.Roles = append(r.Roles, ow)
			slices.SortFunc(r.Roles, func(a, b *Overwrite) int { return strings.Compare(a.RoleID, b.RoleID) })
		}
	}
	return out, nil
}

func (f *fakeRepo) OwnRules(_ context.Context, userID string, projectIDs []string) (map[string]*Overwrite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.getErr != nil {
		return nil, f.getErr
	}
	out := map[string]*Overwrite{}
	for _, id := range projectIDs {
		if ow, ok := f.overwrites[key(resourceTypeProject, id, userID)]; ok {
			out[id] = ow
		}
	}
	return out, nil
}

// DocRules finds the person's own rules on each doc and its folder: the fake keeps no record of who holds which role.
func (f *fakeRepo) DocRules(_ context.Context, docIDs []string, userID string) (map[string]*DocRules, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	out := map[string]*DocRules{}
	for _, id := range docIDs {
		chain := &DocRules{}
		if ow, err := f.get(resourceTypeFolder, f.docFolders[id], userID); err == nil {
			chain.Folder = &Rules{Person: ow}
		}
		ow, err := f.get(resourceTypeDoc, id, userID)
		if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
			return nil, err
		}
		if err == nil {
			chain.Doc = &Rules{Person: ow}
		}
		if chain.Folder != nil || chain.Doc != nil {
			out[id] = chain
		}
	}
	return out, nil
}

// DocRuleTurns decides the person's own doc rules alone; the storage tests cover roles and folders.
func (f *fakeRepo) DocRuleTurns(_ context.Context, userID string, action permissions.Action, ruled, readable []string) (permissions.DocReach, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	allowed, denied := []string{}, []string{}
	for _, ow := range f.overwrites {
		project := f.docProjects[ow.ResourceID]
		if ow.ResourceType != resourceTypeDoc || ow.UserID != userID || !slices.Contains(ruled, project) {
			continue
		}
		state := stateOf(ow, action)
		if state == StateAllow && !slices.Contains(readable, project) {
			allowed = append(allowed, ow.ResourceID)
		}
		if state == StateDeny && slices.Contains(readable, project) {
			denied = append(denied, ow.ResourceID)
		}
	}
	slices.Sort(allowed)
	slices.Sort(denied)
	return permissions.DocReach{AllowedFolders: []string{}, DeniedFolders: []string{}, Allowed: allowed, Denied: denied}, nil
}

// StackRuleTurns decides the person's own stack rules alone; the storage tests cover roles.
func (f *fakeRepo) StackRuleTurns(_ context.Context, userID string, action permissions.Action, ruled, readable []string) ([]string, []string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	allowed, denied := []string{}, []string{}
	for _, ow := range f.overwrites {
		project := f.stackProjects[ow.ResourceID]
		if ow.ResourceType != resourceTypeStack || ow.UserID != userID || !slices.Contains(ruled, project) {
			continue
		}
		in := slices.Contains(readable, project)
		held := in
		if state := stateOf(ow, action); state != "" {
			held = state == StateAllow
		}
		held = held && stateOf(ow, permissions.StacksRead) != StateDeny
		if held && !in {
			allowed = append(allowed, ow.ResourceID)
		}
		if !held && in {
			denied = append(denied, ow.ResourceID)
		}
	}
	slices.Sort(allowed)
	slices.Sort(denied)
	return allowed, denied, nil
}

// roleKey keys a role's rule apart from any person's.
func roleKey(resourceType, resourceID, roleID string) string {
	return key(resourceType, resourceID, "role\x00"+roleID)
}

func (f *fakeRepo) GetRole(_ context.Context, resourceType, resourceID, roleID string) (*Overwrite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	ow, ok := f.overwrites[roleKey(resourceType, resourceID, roleID)]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return ow, nil
}

func (f *fakeRepo) SetRole(_ context.Context, resourceType, resourceID, roleID string, allow, deny permissions.Set, events ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setErr != nil {
		return f.setErr
	}
	f.events = append(f.events, events...)
	k := roleKey(resourceType, resourceID, roleID)
	if len(allow) == 0 && len(deny) == 0 {
		delete(f.overwrites, k)
		return nil
	}
	f.overwrites[k] = &Overwrite{ResourceType: resourceType, ResourceID: resourceID, Target: TargetRole, RoleID: roleID, Allow: allow, Deny: deny}
	return nil
}

// RoleRules finds the role rules on each resource for roleIDs, in role order; Rules finds people's only.
func (f *fakeRepo) RoleRules(_ context.Context, resourceType string, resourceIDs, roleIDs []string) (map[string][]*Overwrite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	out := map[string][]*Overwrite{}
	for _, id := range resourceIDs {
		for _, roleID := range slices.Sorted(slices.Values(roleIDs)) {
			if ow, ok := f.overwrites[roleKey(resourceType, id, roleID)]; ok {
				out[id] = append(out[id], ow)
			}
		}
	}
	return out, nil
}

func (f *fakeRepo) Rules(_ context.Context, resourceType string, resourceIDs []string, userID string) (map[string]*Rules, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	out := map[string]*Rules{}
	for _, id := range resourceIDs {
		ow, err := f.get(resourceType, id, userID)
		if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
			return nil, err
		}
		r := &Rules{Person: ow}
		// Every role's rule comes back, as though the person held each: the check keeps only the roles they hold.
		for _, role := range f.overwrites {
			if role.ResourceType == resourceType && role.ResourceID == id && role.Target == TargetRole {
				r.Roles = append(r.Roles, role)
			}
		}
		slices.SortFunc(r.Roles, func(a, b *Overwrite) int { return strings.Compare(a.RoleID, b.RoleID) })
		if r.Person != nil || len(r.Roles) > 0 {
			out[id] = r
		}
	}
	return out, nil
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

func (f *fakeRepo) Set(_ context.Context, resourceType, resourceID, userID string, allow, deny permissions.Set, events ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setErr != nil {
		return f.setErr
	}
	f.events = append(f.events, events...)
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

func (f *fakeRepo) Grid(context.Context, string) (*GridSource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.grid, nil
}

func (f *fakeRepo) SaveGrid(_ context.Context, _ string, rules []*Overwrite, finishedBy string, events ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setErr != nil {
		return f.setErr
	}
	f.saved, f.finishedBy = rules, finishedBy
	f.events = append(f.events, events...)
	return nil
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
	byDoc     map[string]string
	projectOf map[string]string
	folderOf  map[string]string
	folders   map[string]DocScope
	err       error
}

func (f *fakeDocWorkspace) FolderScope(_ context.Context, folderID string) (DocScope, error) {
	scope, ok := f.folders[folderID]
	if !ok {
		return DocScope{}, apperrs.ErrNotFound
	}
	return scope, nil
}

func (f *fakeDocWorkspace) DocScopes(_ context.Context, docIDs []string) (map[string]DocScope, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := map[string]DocScope{}
	for _, id := range docIDs {
		out[id] = DocScope{WorkspaceID: f.byDoc[id], ProjectID: f.projectOf[id], FolderID: f.folderOf[id]}
	}
	return out, nil
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
		s.SetDocWorkspaces(&fakeDocWorkspace{byDoc: map[string]string{"doc-1": ws}, projectOf: map[string]string{"doc-1": "p-1"}})
		setOverwrite(repo, "project", "p-1", "alice", permissions.SetOf(permissions.ProjectsOpen))

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
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.MembersWrite)})
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.True(t, s.HasPermission(context.Background(), "alice", ws, permissions.MembersWrite, "", ""))
	})

	t.Run("no role and no overwrite denies", func(t *testing.T) {
		s := newService(newFakeRepo(), newFakeUsers())
		s.SetRoles(newFakeRoles())
		assert.False(t, s.HasPermission(context.Background(), "ghost", ws, permissions.MembersWrite, "", ""))
	})

	t.Run("a rule left on the workspace is never read, and a resource's own allow goes beyond the role", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{})
		require.NoError(t, repo.Set(context.Background(), "workspace", ws, "alice", permissions.SetOf(permissions.MembersWrite), nil))
		require.NoError(t, repo.Set(context.Background(), "doc", "doc-1", "alice", permissions.SetOf(permissions.MembersWrite), nil))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.False(t, s.HasPermission(context.Background(), "alice", ws, permissions.MembersWrite, "", ""))
		assert.True(t, s.HasPermission(context.Background(), "alice", ws, permissions.MembersWrite, "doc", "doc-1"))
	})

	t.Run("resource-instance deny beats role allow", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.MembersWrite)})
		require.NoError(t, repo.Set(context.Background(), "doc", "doc-1", "alice", nil, permissions.SetOf(permissions.MembersWrite)))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.False(t, s.HasPermission(context.Background(), "alice", ws, permissions.MembersWrite, "doc", "doc-1"))
	})

	t.Run("no resourceType/resourceID skips the resource-instance layer entirely", func(t *testing.T) {
		repo := newFakeRepo()
		roles := newFakeRoles()
		roles.set(ws, "alice", RoleInfo{Permissions: permissions.SetOf(permissions.MembersWrite)})
		require.NoError(t, repo.Set(context.Background(), "project", "proj-1", "alice", nil, permissions.SetOf(permissions.MembersWrite)))
		s := newService(repo, newFakeUsers())
		s.SetRoles(roles)
		assert.True(t, s.HasPermission(context.Background(), "alice", ws, permissions.MembersWrite, "", ""))
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
		assert.False(t, s.HasPermission(context.Background(), "alice", ws, permissions.MembersWrite, "", ""))
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
		setOverwrite(repo, resourceTypeDoc, "doc-1", "manager", permissions.SetOf(permissions.PermissionsWrite, permissions.DocsRead))
		setOverwrite(repo, resourceTypeDoc, "doc-2", "manager", permissions.SetOf(permissions.PermissionsWrite, permissions.DocsRead))
		s := newService(repo, users)
		err := s.SetGrants(context.Background(), "manager", []string{"doc-1", "doc-2"}, []string{"alice"}, []permissions.Action{permissions.DocsRead}, StateAllow)
		require.NoError(t, err)
		for _, docID := range []string{"doc-1", "doc-2"} {
			ow, err := repo.Get(context.Background(), resourceTypeDoc, docID, "alice")
			require.NoError(t, err)
			assert.Equal(t, permissions.SetOf(permissions.DocsRead), ow.Allow)
		}
		require.Len(t, repo.events, 2)
		assert.Equal(t, TopicGrantChanged, repo.events[0].Topic)
		assert.Equal(t, GrantEvent{ResourceType: resourceTypeDoc, ResourceID: "doc-2", Target: TargetPerson, UserID: "alice", ActorID: "manager"}, repo.events[1].Payload)
	})
	t.Run("permissions:write holder can grant on their doc", func(t *testing.T) {
		repo := newFakeRepo()
		setOverwrite(repo, resourceTypeDoc, "doc-1", "alice", permissions.SetOf(permissions.PermissionsWrite, permissions.DocsWrite))
		s := newService(repo, users)
		err := s.SetGrants(context.Background(), "alice", []string{"doc-1"}, []string{"bob"}, []permissions.Action{permissions.DocsWrite}, StateAllow)
		require.NoError(t, err)
		ow, err := repo.Get(context.Background(), resourceTypeDoc, "doc-1", "bob")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(permissions.DocsWrite), ow.Allow)
	})
	t.Run("non-manager cannot grant", func(t *testing.T) {
		repo := newFakeRepo()
		s := newService(repo, users)
		err := s.SetGrants(context.Background(), "alice", []string{"doc-1"}, []string{"bob"}, []permissions.Action{permissions.DocsRead}, StateAllow)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("revoke clears bits and deletes empty overwrite", func(t *testing.T) {
		repo := newFakeRepo()
		setOverwrite(repo, resourceTypeDoc, "doc-1", "manager", permissions.SetOf(permissions.PermissionsWrite))
		setOverwrite(repo, resourceTypeDoc, "doc-1", "alice", permissions.SetOf(permissions.DocsRead, permissions.DocsWrite))
		s := newService(repo, users)
		err := s.SetGrants(context.Background(), "manager", []string{"doc-1"}, []string{"alice"}, []permissions.Action{permissions.DocsRead, permissions.DocsWrite}, stateRevoke)
		require.NoError(t, err)
		_, err = repo.Get(context.Background(), resourceTypeDoc, "doc-1", "alice")
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("an allow clears a deny bit for the same action", func(t *testing.T) {
		repo := newFakeRepo()
		setOverwrite(repo, resourceTypeDoc, "doc-1", "manager", permissions.SetOf(permissions.PermissionsWrite, permissions.DocsRead))
		require.NoError(t, repo.Set(context.Background(), resourceTypeDoc, "doc-1", "alice", nil, permissions.SetOf(permissions.DocsRead)))
		s := newService(repo, users)
		err := s.SetGrants(context.Background(), "manager", []string{"doc-1"}, []string{"alice"}, []permissions.Action{permissions.DocsRead}, StateAllow)
		require.NoError(t, err)
		ow, err := repo.Get(context.Background(), resourceTypeDoc, "doc-1", "alice")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(permissions.DocsRead), ow.Allow)
		assert.Equal(t, permissions.Set(nil), ow.Deny)
	})
	t.Run("no actor is unauthorized", func(t *testing.T) {
		s := newService(newFakeRepo(), users)
		err := s.SetGrants(context.Background(), "", []string{"doc-1"}, []string{"alice"}, []permissions.Action{permissions.DocsRead}, StateAllow)
		assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
	})
	t.Run("empty inputs are invalid", func(t *testing.T) {
		s := newService(newFakeRepo(), users)
		err := s.SetGrants(context.Background(), "manager", []string{}, []string{"alice"}, []permissions.Action{permissions.DocsRead}, StateAllow)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestSetGrants_States(t *testing.T) {
	users := newFakeUsers(&User{ID: "manager"}, &User{ID: "alice"})
	manager := permissions.SetOf(permissions.PermissionsWrite, permissions.DocsRead, permissions.DocsWrite)
	cases := []struct {
		name               string
		allow, deny        permissions.Set
		state              string
		wantAllow, wantDny permissions.Set
	}{
		{"deny moves an allow to deny", permissions.SetOf(permissions.DocsRead), nil, StateDeny, nil, permissions.SetOf(permissions.DocsRead)},
		{"fallback clears an allow", permissions.SetOf(permissions.DocsRead), nil, StateFallback, nil, nil},
		{"fallback lifts a deny", nil, permissions.SetOf(permissions.DocsRead), StateFallback, nil, nil},
		{"revoke keeps a deny", nil, permissions.SetOf(permissions.DocsRead), stateRevoke, nil, permissions.SetOf(permissions.DocsRead)},
		{"lift keeps an allow", permissions.SetOf(permissions.DocsRead), nil, stateLift, permissions.SetOf(permissions.DocsRead), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			setOverwrite(repo, resourceTypeDoc, "doc-1", "manager", manager)
			require.NoError(t, repo.Set(context.Background(), resourceTypeDoc, "doc-1", "alice", tc.allow, tc.deny))
			s := newService(repo, users)
			require.NoError(t, s.SetGrants(context.Background(), "manager", []string{"doc-1"}, []string{"alice"}, []permissions.Action{permissions.DocsRead}, tc.state))
			ow, err := repo.Get(context.Background(), resourceTypeDoc, "doc-1", "alice")
			if tc.wantAllow == nil && tc.wantDny == nil {
				assert.ErrorIs(t, err, apperrs.ErrNotFound, "a rule left at Fallback everywhere is deleted")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantAllow, ow.Allow)
			assert.Equal(t, tc.wantDny, ow.Deny)
		})
	}
	t.Run("an unknown state is invalid", func(t *testing.T) {
		repo := newFakeRepo()
		setOverwrite(repo, resourceTypeDoc, "doc-1", "manager", manager)
		err := newService(repo, users).SetGrants(context.Background(), "manager", []string{"doc-1"}, []string{"alice"}, []permissions.Action{permissions.DocsRead}, "maybe")
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
}

func TestGrantState_KeepsWhatGrantMeant(t *testing.T) {
	assert.Equal(t, StateAllow, GrantState(resourceTypeDoc, true))
	assert.Equal(t, stateRevoke, GrantState(resourceTypeDoc, false))
	assert.Equal(t, stateLift, GrantState(resourceTypePlay, true))
	assert.Equal(t, StateDeny, GrantState(resourceTypePlay, false))
}

func TestGrants_NobodyAllowsWhatTheyLack(t *testing.T) {
	const ws = "ws-1"
	users := newFakeUsers(&User{ID: "manager"}, &User{ID: "bob"})
	newDocService := func(repo *fakeRepo) *Service {
		setOverwrite(repo, resourceTypeDoc, "doc-1", "manager", permissions.SetOf(permissions.PermissionsWrite, permissions.DocsRead))
		setOverwrite(repo, resourceTypeDoc, "doc-2", "manager", permissions.SetOf(permissions.PermissionsWrite, permissions.DocsRead, permissions.DocsDelete))
		return newService(repo, users)
	}
	newPlayService := func(repo *fakeRepo, held permissions.Set) *Service {
		roles := newFakeRoles()
		roles.set(ws, "manager", RoleInfo{Permissions: held})
		s := newService(repo, users)
		s.SetRoles(roles)
		s.SetPlayWorkspaces(&fakePlayWorkspace{byPlay: map[string]string{"play-1": ws}})
		return s
	}
	cases := []struct {
		name     string
		build    func(*fakeRepo) *Service
		resource string
		ids      []string
		action   permissions.Action
		bobDeny  bool
		state    string
		wantErr  error
	}{
		{"a doc allow of what the giver lacks", newDocService, resourceTypeDoc, []string{"doc-1"}, permissions.DocsDelete, false, StateAllow, apperrs.ErrForbidden},
		{"a doc allow through grant of what the giver lacks", newDocService, resourceTypeDoc, []string{"doc-1"}, permissions.DocsDelete, false, GrantState(resourceTypeDoc, true), apperrs.ErrForbidden},
		{"a doc fallback lifting a deny the giver lacks", newDocService, resourceTypeDoc, []string{"doc-1"}, permissions.DocsDelete, true, StateFallback, apperrs.ErrForbidden},
		{"one doc of two where the giver lacks it", newDocService, resourceTypeDoc, []string{"doc-2", "doc-1"}, permissions.DocsDelete, false, StateAllow, apperrs.ErrForbidden},
		{"a doc deny of what the giver lacks", newDocService, resourceTypeDoc, []string{"doc-1"}, permissions.DocsDelete, false, StateDeny, nil},
		{"a doc allow of what the giver holds", newDocService, resourceTypeDoc, []string{"doc-2"}, permissions.DocsDelete, false, StateAllow, nil},
		{"a play allow of a run the giver lacks", func(r *fakeRepo) *Service {
			return newPlayService(r, permissions.SetOf(permissions.PlaysRead, permissions.PlaysWrite))
		}, resourceTypePlay, []string{"play-1"}, permissions.PlaysRun, false, StateAllow, apperrs.ErrForbidden},
		{"a play lift of a run the giver lacks", func(r *fakeRepo) *Service {
			return newPlayService(r, permissions.SetOf(permissions.PlaysRead, permissions.PlaysWrite))
		}, resourceTypePlay, []string{"play-1"}, permissions.PlaysRun, true, GrantState(resourceTypePlay, true), apperrs.ErrForbidden},
		{"a play deny of a run the giver lacks", func(r *fakeRepo) *Service {
			return newPlayService(r, permissions.SetOf(permissions.PlaysRead, permissions.PlaysWrite))
		}, resourceTypePlay, []string{"play-1"}, permissions.PlaysRun, false, StateDeny, nil},
		{"a play allow of a run the giver holds", func(r *fakeRepo) *Service {
			return newPlayService(r, permissions.SetOf(permissions.PlaysRead, permissions.PlaysWrite, permissions.PlaysRun))
		}, resourceTypePlay, []string{"play-1"}, permissions.PlaysRun, false, StateAllow, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			s := tc.build(repo)
			for _, id := range tc.ids {
				if tc.bobDeny {
					require.NoError(t, repo.Set(context.Background(), tc.resource, id, "bob", nil, permissions.SetOf(tc.action)))
				}
			}
			set := s.SetGrants
			if tc.resource == resourceTypePlay {
				set = s.SetPlayGrants
			}
			err := set(context.Background(), "manager", tc.ids, []string{"bob"}, []permissions.Action{tc.action}, tc.state)
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
			for _, id := range tc.ids {
				ow, getErr := repo.Get(context.Background(), tc.resource, id, "bob")
				if !tc.bobDeny {
					assert.ErrorIs(t, getErr, apperrs.ErrNotFound, "a refused write stores nothing on %s", id)
					continue
				}
				require.NoError(t, getErr)
				assert.Equal(t, permissions.SetOf(tc.action), ow.Deny, "a refused lift keeps the deny on %s", id)
			}
		})
	}
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
		s := newPlaysService(repo, permissions.SetOf(permissions.PlaysRead, permissions.PlaysWrite))
		err := s.SetPlayGrants(context.Background(), "manager", []string{"play-1"}, []string{"alice"}, []permissions.Action{permissions.PlaysRun}, StateDeny)
		require.NoError(t, err)
		ow, err := repo.Get(context.Background(), resourceTypePlay, "play-1", "alice")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(permissions.PlaysRun), ow.Deny)
		assert.Equal(t, permissions.Set(nil), ow.Allow)
	})

	t.Run("lifting clears an existing deny instead of adding an allow", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.Set(context.Background(), resourceTypePlay, "play-1", "alice", nil, permissions.SetOf(permissions.PlaysRun)))
		s := newPlaysService(repo, permissions.SetOf(permissions.PlaysRead, permissions.PlaysWrite, permissions.PlaysRun))
		err := s.SetPlayGrants(context.Background(), "manager", []string{"play-1"}, []string{"alice"}, []permissions.Action{permissions.PlaysRun}, stateLift)
		require.NoError(t, err)
		_, err = repo.Get(context.Background(), resourceTypePlay, "play-1", "alice")
		assert.True(t, errors.Is(err, apperrs.ErrNotFound), "clearing the only bit deletes the empty overwrite")
	})

	t.Run("sets See, Run, Edit and Delete, and nothing else", func(t *testing.T) {
		repo := newFakeRepo()
		s := newPlaysService(repo, permissions.SetOf(permissions.PlaysRead, permissions.PlaysWrite))
		see := []permissions.Action{permissions.PlaysRead, permissions.PlaysRun, permissions.PlaysWrite, permissions.PlaysDelete}
		require.NoError(t, s.SetPlayGrants(context.Background(), "manager", []string{"play-1"}, []string{"alice"}, see, StateDeny))
		ow, err := repo.Get(context.Background(), resourceTypePlay, "play-1", "alice")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(see...), ow.Deny)
		err = s.SetPlayGrants(context.Background(), "manager", []string{"play-1"}, []string{"alice"}, []permissions.Action{permissions.DocsRead}, StateDeny)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})

	t.Run("without plays:write is forbidden, and a play the manager can't see is not found", func(t *testing.T) {
		repo := newFakeRepo()
		err := newPlaysService(repo, permissions.SetOf(permissions.PlaysRead)).SetPlayGrants(context.Background(), "manager", []string{"play-1"}, []string{"alice"}, []permissions.Action{permissions.PlaysRun}, StateDeny)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
		err = newPlaysService(repo, permissions.SetOf(permissions.PlaysWrite)).SetPlayGrants(context.Background(), "manager", []string{"play-1"}, []string{"alice"}, []permissions.Action{permissions.PlaysRun}, StateDeny)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
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

// A rule naming an account the names read can't find carries no name, never its id, so a client can't show an id as a person.
func TestListPlayGrants_UnresolvedTarget_CarriesNoName(t *testing.T) {
	const ws = "ws-1"
	repo := newFakeRepo()
	for _, id := range []string{"alice", "ghost"} {
		require.NoError(t, repo.Set(context.Background(), resourceTypePlay, "play-1", id, nil, permissions.SetOf(permissions.PlaysRun)))
	}
	roles := newFakeRoles()
	roles.set(ws, "manager", RoleInfo{Permissions: permissions.SetOf(permissions.PlaysWrite)})
	s := newService(repo, newFakeUsers(&User{ID: "manager"}))
	s.SetRoles(roles)
	s.SetNames(&fakeNames{})
	s.SetPlayWorkspaces(&fakePlayWorkspace{byPlay: map[string]string{"play-1": ws}})

	grants, err := s.ListPlayGrants(context.Background(), "manager", "play-1")
	require.NoError(t, err)
	named := map[string]string{}
	for _, g := range grants {
		named[g.UserID] = g.TargetName
	}
	assert.Equal(t, map[string]string{"alice": "alice", "ghost": ""}, named)
}

func TestListUsers(t *testing.T) {
	users := newFakeUsers(
		&User{ID: "reader", Login: "reader"},
		&User{ID: "manager", Login: "manager"},
		&User{ID: "plain", Login: "plain"},
	)
	t.Run("an accounts:read holder in any workspace lists the directory", func(t *testing.T) {
		s := newService(newFakeRepo(), users)
		roles := newFakeRoles()
		roles.set("ws-b", "reader", RoleInfo{Permissions: permissions.SetOf(permissions.AccountsRead)})
		s.SetRoles(roles)
		s.SetScopes(fakeScopes{workspaces: map[string][]string{"reader": {"ws-a", "ws-b"}}})
		got, err := s.ListUsers(context.Background(), "reader")
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

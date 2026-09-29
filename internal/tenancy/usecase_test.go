package tenancy

import (
	"context"
	"errors"
	"strings"
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
	mu         sync.Mutex
	workspaces map[string]*Workspace
	members    map[string][]string           // workspaceID -> userIDs
	roleIDs    map[string]map[string]string  // workspaceID -> userID -> roleID
	overrides  map[string][2]permissions.Set // workspaceID/userID -> allow, deny
	events     []eventbus.OutboxEvent
	createErr  error
	getErr     error
	listErr    error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		workspaces: map[string]*Workspace{},
		members:    map[string][]string{},
		roleIDs:    map[string]map[string]string{},
		overrides:  map[string][2]permissions.Set{},
	}
}

func (f *fakeRepo) Create(_ context.Context, w *Workspace) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if _, ok := f.workspaces[w.ID]; ok {
		return apperrs.ErrConflict
	}
	f.workspaces[w.ID] = w
	return nil
}

func (f *fakeRepo) Update(_ context.Context, w *Workspace) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.workspaces[w.ID]; !ok {
		return apperrs.ErrNotFound
	}
	f.workspaces[w.ID] = w
	return nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (*Workspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	w, ok := f.workspaces[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return w, nil
}

func (f *fakeRepo) ListForUser(_ context.Context, userID string) ([]*Workspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Workspace
	for wsID, users := range f.members {
		for _, u := range users {
			if u == userID {
				out = append(out, f.workspaces[wsID])
			}
		}
	}
	return out, nil
}

func (f *fakeRepo) ListWithRoles(context.Context) ([]*TeamWorkspace, error) {
	return nil, nil
}

func (f *fakeRepo) AddMember(_ context.Context, m *Member, events ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, events...)
	f.members[m.WorkspaceID] = append(f.members[m.WorkspaceID], m.UserID)
	if f.roleIDs[m.WorkspaceID] == nil {
		f.roleIDs[m.WorkspaceID] = map[string]string{}
	}
	f.roleIDs[m.WorkspaceID][m.UserID] = m.RoleID
	return nil
}

func (f *fakeRepo) RoleIDFor(_ context.Context, workspaceID, userID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	roleID, ok := f.roleIDs[workspaceID][userID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return roleID, nil
}

func (f *fakeRepo) ListByWorkspace(_ context.Context, workspaceID string) ([]*Member, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Member
	for _, userID := range f.members[workspaceID] {
		out = append(out, &Member{UserID: userID, WorkspaceID: workspaceID, RoleID: f.roleIDs[workspaceID][userID]})
	}
	return out, nil
}

func (f *fakeRepo) RemoveMember(_ context.Context, workspaceID, userID string, events ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, events...)
	users := f.members[workspaceID]
	for i, u := range users {
		if u == userID {
			f.members[workspaceID] = append(users[:i], users[i+1:]...)
			delete(f.roleIDs[workspaceID], userID)
			return nil
		}
	}
	return apperrs.ErrNotFound
}

func (f *fakeRepo) SetRole(_ context.Context, workspaceID, userID, roleID string, events ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.roleIDs[workspaceID][userID]; !ok {
		return apperrs.ErrNotFound
	}
	f.roleIDs[workspaceID][userID] = roleID
	f.events = append(f.events, events...)
	return nil
}

func (f *fakeRepo) Overrides(_ context.Context, workspaceID, userID string) (permissions.Set, permissions.Set, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o := f.overrides[workspaceID+"/"+userID]
	return o[0], o[1], nil
}

func (f *fakeRepo) SetOverrides(_ context.Context, workspaceID, userID string, allow, deny permissions.Set, events ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.overrides[workspaceID+"/"+userID] = [2]permissions.Set{allow, deny}
	f.events = append(f.events, events...)
	return nil
}

func (f *fakeRepo) ListAllMemberships(context.Context) ([]*TeamMembership, error) {
	return nil, nil
}

// fakeInviteRepo is a minimal stand-in for the storage layer's WorkspaceInvitesRepo (separate from fakeRepo since InviteRepo and MemberRepo both declare a same-named, differently-typed ListByWorkspace that one Go type can't implement both of).
type fakeInviteRepo struct {
	mu      sync.Mutex
	invites map[string]map[string]*Invite // workspaceID -> login -> Invite
}

func newFakeInviteRepo() *fakeInviteRepo {
	return &fakeInviteRepo{invites: map[string]map[string]*Invite{}}
}

func (f *fakeInviteRepo) Upsert(_ context.Context, inv *Invite) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.invites[inv.WorkspaceID] == nil {
		f.invites[inv.WorkspaceID] = map[string]*Invite{}
	}
	f.invites[inv.WorkspaceID][inv.Login] = inv
	return nil
}

func (f *fakeInviteRepo) Delete(_ context.Context, workspaceID, login string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.invites[workspaceID], login)
	return nil
}

func (f *fakeInviteRepo) ListByWorkspace(_ context.Context, workspaceID string) ([]*Invite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Invite
	for _, inv := range f.invites[workspaceID] {
		out = append(out, inv)
	}
	return out, nil
}

func (f *fakeInviteRepo) ListByLogin(_ context.Context, login string) ([]*Invite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Invite
	for _, byLogin := range f.invites {
		if inv, ok := byLogin[login]; ok {
			out = append(out, inv)
		}
	}
	return out, nil
}

// fakeRoleGate is a minimal stand-in for the roles domain's RoleGate seam: hands back a deterministic "owner-role-<workspaceID>" id, mirroring what roles.Service.CreateOwnerRole would persist and return.
type fakeRoleGate struct {
	createErr error
}

func (g *fakeRoleGate) CreateOwnerRole(_ context.Context, workspaceID string) (string, error) {
	if g.createErr != nil {
		return "", g.createErr
	}
	return "owner-role-" + workspaceID, nil
}

// fakeChannelGate is a no-op stand-in for the chat domain's ChannelGate seam (live-chat ticket 07); tenancy tests care about workspace/role/membership bookkeeping, not chat.
type fakeChannelGate struct {
	createErr error
}

func (g *fakeChannelGate) CreateGeneralChannel(_ context.Context, _, _ string) error {
	return g.createErr
}

// fakePlaysGate is a no-op stand-in for the plays domain's PlaysGate seam (ticket 20); tenancy tests care
// about workspace/role/membership bookkeeping, not plays.
type fakePlaysGate struct {
	seedErr error
}

func (g *fakePlaysGate) SeedDefaultPlays(_ context.Context, _ string) error {
	return g.seedErr
}

// fakePermissionGate stands in for access's instance-level check: allow answers every action, defaulting to true so
// Create tests need not grant workspaces:create explicitly.
type fakePermissionGate struct {
	allow bool
	err   error
}

func newFakePermissionGate() *fakePermissionGate {
	return &fakePermissionGate{allow: true}
}

func (g *fakePermissionGate) HoldsAnywhere(_ context.Context, _ string, _ permissions.Action) (bool, error) {
	if g.err != nil {
		return false, g.err
	}
	return g.allow, nil
}

// fakeRoleNameGate is a minimal stand-in for the roles domain's RoleNameGate seam (ticket 14): hands back a deterministic "role-name-<roleID>" name unless a specific one is registered, mirroring roles.Service.Get.
type fakeRoleNameGate struct {
	names   map[string]string
	isOwner map[string]bool
	perms   map[string]permissions.Set
	err     error
}

func newFakeRoleNameGate() *fakeRoleNameGate {
	return &fakeRoleNameGate{names: map[string]string{}, isOwner: map[string]bool{}, perms: map[string]permissions.Set{}}
}

// RolePermissions is empty unless registered, so assigning an unregistered role grants nothing to check.
func (g *fakeRoleNameGate) RolePermissions(_ context.Context, _, roleID string) (permissions.Set, error) {
	return g.perms[roleID], nil
}

func (g *fakeRoleNameGate) RoleName(_ context.Context, _, roleID string) (string, error) {
	if g.err != nil {
		return "", g.err
	}
	if name, ok := g.names[roleID]; ok {
		return name, nil
	}
	return "role-name-" + roleID, nil
}

// IsOwnerRole defaults to true for any role id matching fakeRoleGate's deterministic "owner-role-<workspaceID>" naming unless overridden via g.isOwner, so tests that never touch isOwner keep treating the creator's role as Owner.
func (g *fakeRoleNameGate) IsOwnerRole(_ context.Context, _, roleID string) (bool, error) {
	if g.err != nil {
		return false, g.err
	}
	if v, ok := g.isOwner[roleID]; ok {
		return v, nil
	}
	return strings.HasPrefix(roleID, "owner-role-"), nil
}

// fakeWorkspacePermissionGate is a minimal stand-in for the access domain's HasPermission-backed gate (ticket 15): empty by default, mirroring a fresh member with no permissions, unless a list is registered for that user id.
type fakeWorkspacePermissionGate struct {
	perms map[string][]string
}

func newFakeWorkspacePermissionGate() *fakeWorkspacePermissionGate {
	return &fakeWorkspacePermissionGate{perms: map[string][]string{}}
}

func (g *fakeWorkspacePermissionGate) WorkspacePermissions(_ context.Context, userID, _ string) []string {
	// Mirrors access.Service.WorkspacePermissions' guarantee of a non-nil (if possibly empty) slice, so callers never see `null` on the wire for a member with no permissions.
	if perms, ok := g.perms[userID]; ok {
		return perms
	}
	return []string{}
}

// fakeAllowlistGate is a minimal stand-in for auth's sign-in allowlist (Membership invites): every login is allowed by default (mirroring newFakePermissionGate's opt-in-free default); tests exercising rejection register a false override.
type fakeAllowlistGate struct {
	allowed map[string]bool
}

func newFakeAllowlistGate() *fakeAllowlistGate {
	return &fakeAllowlistGate{allowed: map[string]bool{}}
}

func (g *fakeAllowlistGate) IsAllowlisted(_ context.Context, login string) (bool, error) {
	if v, ok := g.allowed[login]; ok {
		return v, nil
	}
	return true, nil
}

// fakeUserLookupGate is a minimal stand-in for auth's user lookups (Membership invites): no login resolves to a User by default (mirrors a never-signed-in login) unless registered via register.
type fakeUserLookupGate struct {
	byLogin map[string]string // login -> userID
	byID    map[string]string // userID -> login
}

func newFakeUserLookupGate() *fakeUserLookupGate {
	return &fakeUserLookupGate{byLogin: map[string]string{}, byID: map[string]string{}}
}

func (g *fakeUserLookupGate) register(login, userID string) {
	g.byLogin[login] = userID
	g.byID[userID] = login
}

func (g *fakeUserLookupGate) UserIDForLogin(_ context.Context, login string) (string, bool, error) {
	id, ok := g.byLogin[login]
	return id, ok, nil
}

func (g *fakeUserLookupGate) LoginForUserID(_ context.Context, userID string) (string, error) {
	login, ok := g.byID[userID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return login, nil
}

// fakeAccountGate is a minimal stand-in for auth's registered accounts: every id resolves to an active account unless registered otherwise.
type fakeAccountGate struct {
	accounts map[string]*TeamAccount
}

func newFakeAccountGate() *fakeAccountGate {
	return &fakeAccountGate{accounts: map[string]*TeamAccount{}}
}

func (g *fakeAccountGate) ListAccounts(context.Context) ([]*TeamAccount, error) {
	out := make([]*TeamAccount, 0, len(g.accounts))
	for _, a := range g.accounts {
		out = append(out, a)
	}
	return out, nil
}

func (g *fakeAccountGate) Account(_ context.Context, userID string) (*TeamAccount, error) {
	if a, ok := g.accounts[userID]; ok {
		return a, nil
	}
	return &TeamAccount{ID: userID, Login: userID, Status: "active"}, nil
}

func (g *fakeAccountGate) Presence(context.Context) (map[string]bool, map[string]time.Time, error) {
	return nil, nil, nil
}

func newTestService(repo *fakeRepo) *Service {
	s := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, newFakePermissionGate(), newFakeRoleNameGate(), newFakeWorkspacePermissionGate(), newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{}, newFakeAccountGate())
	s.now = func() time.Time { return fixedNow }
	return s
}

func TestCreate(t *testing.T) {
	t.Run("empty user id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Create(context.Background(), "  ", "Acme")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Create(context.Background(), "u-1", "   ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.createErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.Create(context.Background(), "u-1", "Acme")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.createErr)
	})
	t.Run("creates a workspace and adds the creator as a member", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		w, err := s.Create(context.Background(), "u-1", "Acme")
		require.NoError(t, err)
		assert.Equal(t, "Acme", w.Name)
		assert.Equal(t, fixedNow, w.CreatedAt)

		got, err := s.ListForUser(context.Background(), "u-1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, w.ID, got[0].ID)
	})
	t.Run("binds the creator to the workspace's owner role", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		w, err := s.Create(context.Background(), "u-1", "Acme")
		require.NoError(t, err)

		roleID, err := s.MemberRoleID(context.Background(), w.ID, "u-1")
		require.NoError(t, err)
		assert.Equal(t, "owner-role-"+w.ID, roleID)
	})
	t.Run("role gate error propagates and skips adding the member", func(t *testing.T) {
		repo := newFakeRepo()
		s := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{createErr: errors.New("roles unavailable")}, newFakePermissionGate(), newFakeRoleNameGate(), newFakeWorkspacePermissionGate(), newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{}, newFakeAccountGate())
		s.now = func() time.Time { return fixedNow }
		_, err := s.Create(context.Background(), "u-1", "Acme")
		require.Error(t, err)
		assert.ErrorContains(t, err, "roles unavailable")

		got, err := s.ListForUser(context.Background(), "u-1")
		require.NoError(t, err)
		assert.Empty(t, got)
	})
	t.Run("without workspaces:create is forbidden", func(t *testing.T) {
		repo := newFakeRepo()
		s := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, &fakePermissionGate{allow: false}, newFakeRoleNameGate(), newFakeWorkspacePermissionGate(), newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{}, newFakeAccountGate())
		s.now = func() time.Time { return fixedNow }
		_, err := s.Create(context.Background(), "u-1", "Acme")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("permission gate error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		s := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, &fakePermissionGate{err: errors.New("gate down")}, newFakeRoleNameGate(), newFakeWorkspacePermissionGate(), newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{}, newFakeAccountGate())
		s.now = func() time.Time { return fixedNow }
		_, err := s.Create(context.Background(), "u-1", "Acme")
		require.Error(t, err)
		assert.ErrorContains(t, err, "gate down")
	})
}

func TestRename(t *testing.T) {
	t.Run("renames and bumps updated_at", func(t *testing.T) {
		repo := newFakeRepo()
		repo.workspaces[DefaultWorkspaceID] = &Workspace{ID: DefaultWorkspaceID, Name: "Default"}
		wsPerms := newFakeWorkspacePermissionGate()
		wsPerms.perms["u-1"] = []string{"workspaces:write"}
		s := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, newFakePermissionGate(), newFakeRoleNameGate(), wsPerms, newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{}, newFakeAccountGate())
		s.now = func() time.Time { return fixedNow }

		ws, err := s.Rename(context.Background(), "u-1", DefaultWorkspaceID, "  Acme  ")
		require.NoError(t, err)
		assert.Equal(t, "Acme", ws.Name)
		assert.Equal(t, fixedNow.UTC(), ws.UpdatedAt)

		got, err := s.Get(context.Background(), DefaultWorkspaceID)
		require.NoError(t, err)
		assert.Equal(t, "Acme", got.Name)
	})
	t.Run("requires workspaces:write in that workspace", func(t *testing.T) {
		repo := newFakeRepo()
		repo.workspaces[DefaultWorkspaceID] = &Workspace{ID: DefaultWorkspaceID, Name: "Default"}
		s := newTestService(repo)
		_, err := s.Rename(context.Background(), "u-1", DefaultWorkspaceID, "Acme")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Rename(context.Background(), "u-1", DefaultWorkspaceID, "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestBindDefaultWorkspaceOwner(t *testing.T) {
	t.Run("empty user id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.BindDefaultWorkspaceOwner(context.Background(), "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing default workspace propagates not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.BindDefaultWorkspaceOwner(context.Background(), "u-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("binds the caller to the pre-seeded default workspace, not a new one", func(t *testing.T) {
		repo := newFakeRepo()
		repo.workspaces[DefaultWorkspaceID] = &Workspace{ID: DefaultWorkspaceID, Name: "Default"}
		s := newTestService(repo)

		require.NoError(t, s.BindDefaultWorkspaceOwner(context.Background(), "u-1"))

		got, err := s.ListForUser(context.Background(), "u-1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, DefaultWorkspaceID, got[0].ID)
		// Only the seeded workspace exists — no second workspace was created.
		assert.Len(t, repo.workspaces, 1)

		roleID, err := s.MemberRoleID(context.Background(), DefaultWorkspaceID, "u-1")
		require.NoError(t, err)
		assert.Equal(t, "owner-role-"+DefaultWorkspaceID, roleID)
	})
	t.Run("role gate error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.workspaces[DefaultWorkspaceID] = &Workspace{ID: DefaultWorkspaceID, Name: "Default"}
		s := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{createErr: errors.New("roles unavailable")}, newFakePermissionGate(), newFakeRoleNameGate(), newFakeWorkspacePermissionGate(), newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{}, newFakeAccountGate())
		s.now = func() time.Time { return fixedNow }
		err := s.BindDefaultWorkspaceOwner(context.Background(), "u-1")
		require.Error(t, err)
		assert.ErrorContains(t, err, "roles unavailable")
	})
	t.Run("second bind is an idempotent no-op, not a duplicate role + member conflict", func(t *testing.T) {
		repo := newFakeRepo()
		repo.workspaces[DefaultWorkspaceID] = &Workspace{ID: DefaultWorkspaceID, Name: "Default"}
		s := newTestService(repo)

		require.NoError(t, s.BindDefaultWorkspaceOwner(context.Background(), "u-1"))
		require.NoError(t, s.BindDefaultWorkspaceOwner(context.Background(), "u-1"),
			"a double-submitted wizard finish must not error")

		got, err := s.ListForUser(context.Background(), "u-1")
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})
}

func TestMemberRoleID(t *testing.T) {
	t.Run("returns the role id for a member", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		w, err := s.Create(context.Background(), "u-1", "Acme")
		require.NoError(t, err)

		roleID, err := s.MemberRoleID(context.Background(), w.ID, "u-1")
		require.NoError(t, err)
		assert.Equal(t, "owner-role-"+w.ID, roleID)
	})
	t.Run("non-member is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.MemberRoleID(context.Background(), "ws-1", "u-nobody")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestMemberRoleName(t *testing.T) {
	t.Run("empty workspace id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.MemberRoleName(context.Background(), "  ", "u-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty user id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.MemberRoleName(context.Background(), "ws-1", "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("non-member is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.MemberRoleName(context.Background(), "ws-1", "u-nobody")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("returns the owner role's name for the workspace creator", func(t *testing.T) {
		repo := newFakeRepo()
		nameGate := newFakeRoleNameGate()
		s := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, newFakePermissionGate(), nameGate, newFakeWorkspacePermissionGate(), newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{}, newFakeAccountGate())
		s.now = func() time.Time { return fixedNow }
		w, err := s.Create(context.Background(), "u-1", "Acme")
		require.NoError(t, err)
		nameGate.names["owner-role-"+w.ID] = "Owner"

		name, err := s.MemberRoleName(context.Background(), w.ID, "u-1")
		require.NoError(t, err)
		assert.Equal(t, "Owner", name)
	})
	t.Run("returns a custom role's real name", func(t *testing.T) {
		repo := newFakeRepo()
		nameGate := newFakeRoleNameGate()
		s := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, newFakePermissionGate(), nameGate, newFakeWorkspacePermissionGate(), newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{}, newFakeAccountGate())
		s.now = func() time.Time { return fixedNow }
		require.NoError(t, repo.AddMember(context.Background(), &Member{UserID: "u-2", WorkspaceID: "ws-1", RoleID: "role-editor"}))
		nameGate.names["role-editor"] = "Editor"

		name, err := s.MemberRoleName(context.Background(), "ws-1", "u-2")
		require.NoError(t, err)
		assert.Equal(t, "Editor", name)
	})
	t.Run("role name gate error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		nameGate := newFakeRoleNameGate()
		nameGate.err = errors.New("roles unavailable")
		s := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, newFakePermissionGate(), nameGate, newFakeWorkspacePermissionGate(), newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{}, newFakeAccountGate())
		s.now = func() time.Time { return fixedNow }
		w, err := s.Create(context.Background(), "u-1", "Acme")
		require.NoError(t, err)

		_, err = s.MemberRoleName(context.Background(), w.ID, "u-1")
		require.Error(t, err)
		assert.ErrorContains(t, err, "roles unavailable")
	})
}

func TestMemberPermissions(t *testing.T) {
	t.Run("delegates to the workspace-permission gate", func(t *testing.T) {
		repo := newFakeRepo()
		wsPerms := newFakeWorkspacePermissionGate()
		s := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, newFakePermissionGate(), newFakeRoleNameGate(), wsPerms, newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{}, newFakeAccountGate())
		s.now = func() time.Time { return fixedNow }
		w, err := s.Create(context.Background(), "u-1", "Acme")
		require.NoError(t, err)
		wsPerms.perms["u-1"] = []string{"projects:write"}

		got := s.MemberPermissions(context.Background(), w.ID, "u-1")
		assert.Equal(t, []string{"projects:write"}, got)
	})
	t.Run("no permissions registered is an empty list, not nil", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		got := s.MemberPermissions(context.Background(), "ws-1", "u-nobody")
		assert.Equal(t, []string{}, got)
	})
}

func TestGet(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Get(context.Background(), "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing workspace is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Get(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.getErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.Get(context.Background(), "ws-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.getErr)
	})
	t.Run("returns a stored workspace", func(t *testing.T) {
		repo := newFakeRepo()
		repo.workspaces["ws-1"] = &Workspace{ID: "ws-1", Name: "Acme"}
		s := newTestService(repo)
		w, err := s.Get(context.Background(), "ws-1")
		require.NoError(t, err)
		assert.Equal(t, "Acme", w.Name)
	})
}

func TestListForUser(t *testing.T) {
	t.Run("empty user id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.ListForUser(context.Background(), "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.listErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.ListForUser(context.Background(), "u-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.listErr)
	})
	t.Run("only returns workspaces the user is a member of", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		_, err := s.Create(context.Background(), "u-1", "Acme")
		require.NoError(t, err)
		_, err = s.Create(context.Background(), "u-2", "Other Co")
		require.NoError(t, err)

		got, err := s.ListForUser(context.Background(), "u-1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "Acme", got[0].Name)
	})
}

// inviteTestFixture bundles the fakes Membership invites' gated methods need direct control over, so each test can register exactly the permission, allowlist, and user-lookup state its scenario requires.
type inviteTestFixture struct {
	repo      *fakeRepo
	invites   *fakeInviteRepo
	wsPerms   *fakeWorkspacePermissionGate
	allowlist *fakeAllowlistGate
	users     *fakeUserLookupGate
	nameGate  *fakeRoleNameGate
	svc       *Service
}

func newInviteFixture() *inviteTestFixture {
	f := &inviteTestFixture{
		repo:      newFakeRepo(),
		invites:   newFakeInviteRepo(),
		wsPerms:   newFakeWorkspacePermissionGate(),
		allowlist: newFakeAllowlistGate(),
		users:     newFakeUserLookupGate(),
		nameGate:  newFakeRoleNameGate(),
	}
	f.svc = NewService(f.repo, f.repo, f.invites, &fakeRoleGate{}, newFakePermissionGate(), f.nameGate, f.wsPerms, f.allowlist, f.users, &fakeChannelGate{}, &fakePlaysGate{}, newFakeAccountGate())
	f.svc.now = func() time.Time { return fixedNow }
	return f
}

func (f *inviteTestFixture) grantManageMembers(actorID string) {
	f.wsPerms.perms[actorID] = []string{"members:write"}
}

func TestInviteMember(t *testing.T) {
	t.Run("actor without members:write is forbidden", func(t *testing.T) {
		f := newInviteFixture()
		err := f.svc.InviteMember(context.Background(), "actor", "ws-1", "bob", "role-editor")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("empty login is invalid", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		err := f.svc.InviteMember(context.Background(), "actor", "ws-1", "  ", "role-editor")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty role id is invalid", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		err := f.svc.InviteMember(context.Background(), "actor", "ws-1", "bob", "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("the Owner role cannot be invite-assigned", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		err := f.svc.InviteMember(context.Background(), "actor", "ws-1", "bob", "owner-role-ws-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("login not on the instance allowlist is invalid", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		f.allowlist.allowed["bob"] = false
		err := f.svc.InviteMember(context.Background(), "actor", "ws-1", "bob", "role-editor")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("existing user is added as a member immediately, no pending invite", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		f.users.register("bob", "u-bob")

		require.NoError(t, f.svc.InviteMember(context.Background(), "actor", "ws-1", "bob", "role-editor"))

		roleID, err := f.svc.MemberRoleID(context.Background(), "ws-1", "u-bob")
		require.NoError(t, err)
		assert.Equal(t, "role-editor", roleID)
		pending, err := f.invites.ListByWorkspace(context.Background(), "ws-1")
		require.NoError(t, err)
		assert.Empty(t, pending)
	})
	t.Run("login with no User yet is held as a pending invite", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")

		require.NoError(t, f.svc.InviteMember(context.Background(), "actor", "ws-1", "Bob", "role-editor"))

		pending, err := f.invites.ListByWorkspace(context.Background(), "ws-1")
		require.NoError(t, err)
		require.Len(t, pending, 1)
		assert.Equal(t, "bob", pending[0].Login, "login is normalized to lowercase")
		assert.Equal(t, "role-editor", pending[0].RoleID)
		assert.Equal(t, "actor", pending[0].InvitedBy)
	})
	t.Run("re-inviting the same pending login updates the role, not a duplicate", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")

		require.NoError(t, f.svc.InviteMember(context.Background(), "actor", "ws-1", "bob", "role-editor"))
		require.NoError(t, f.svc.InviteMember(context.Background(), "actor", "ws-1", "bob", "role-admin"))

		pending, err := f.invites.ListByWorkspace(context.Background(), "ws-1")
		require.NoError(t, err)
		require.Len(t, pending, 1)
		assert.Equal(t, "role-admin", pending[0].RoleID)
	})
}

func TestCancelInvite(t *testing.T) {
	t.Run("actor without members:write is forbidden", func(t *testing.T) {
		f := newInviteFixture()
		err := f.svc.CancelInvite(context.Background(), "actor", "ws-1", "bob")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("withdraws a pending invite", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.svc.InviteMember(context.Background(), "actor", "ws-1", "bob", "role-editor"))

		require.NoError(t, f.svc.CancelInvite(context.Background(), "actor", "ws-1", "bob"))

		pending, err := f.invites.ListByWorkspace(context.Background(), "ws-1")
		require.NoError(t, err)
		assert.Empty(t, pending)
	})
}

func TestListWorkspaceMembers(t *testing.T) {
	t.Run("actor without members:write is forbidden", func(t *testing.T) {
		f := newInviteFixture()
		_, err := f.svc.ListWorkspaceMembers(context.Background(), "actor", "ws-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("no pending invites yields an empty list, never nil (JSON null crashes the SPA)", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		got, err := f.svc.ListWorkspaceMembers(context.Background(), "actor", "ws-1")
		require.NoError(t, err)
		assert.NotNil(t, got.Invites)
		assert.Len(t, got.Invites, 0)
	})
	t.Run("returns members with resolved logins, plus pending invites", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		f.users.register("bob", "u-bob")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))
		require.NoError(t, f.svc.InviteMember(context.Background(), "actor", "ws-1", "carol", "role-editor"))

		got, err := f.svc.ListWorkspaceMembers(context.Background(), "actor", "ws-1")
		require.NoError(t, err)
		require.Len(t, got.Members, 1)
		assert.Equal(t, MemberView{UserID: "u-bob", Login: "bob", RoleID: "role-editor"}, got.Members[0])
		require.Len(t, got.Invites, 1)
		assert.Equal(t, "carol", got.Invites[0].Login)
	})
}

func TestRemoveMember(t *testing.T) {
	t.Run("actor without members:write is forbidden", func(t *testing.T) {
		f := newInviteFixture()
		err := f.svc.RemoveMember(context.Background(), "actor", "ws-1", "u-bob")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("the workspace Owner cannot be removed", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-owner", WorkspaceID: "ws-1", RoleID: "owner-role-ws-1"}))

		err := f.svc.RemoveMember(context.Background(), "actor", "ws-1", "u-owner")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("removes a non-owner member", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))

		require.NoError(t, f.svc.RemoveMember(context.Background(), "actor", "ws-1", "u-bob"))

		_, err := f.svc.MemberRoleID(context.Background(), "ws-1", "u-bob")
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestChangeMemberRole(t *testing.T) {
	t.Run("actor without members:write is forbidden", func(t *testing.T) {
		f := newInviteFixture()
		err := f.svc.ChangeMemberRole(context.Background(), "actor", "ws-1", "u-bob", "role-admin")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("the workspace Owner's role cannot be changed", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-owner", WorkspaceID: "ws-1", RoleID: "owner-role-ws-1"}))

		err := f.svc.ChangeMemberRole(context.Background(), "actor", "ws-1", "u-owner", "role-editor")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("assigning the Owner role via this path is invalid", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))

		err := f.svc.ChangeMemberRole(context.Background(), "actor", "ws-1", "u-bob", "owner-role-ws-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("changes a non-owner member's role", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))

		require.NoError(t, f.svc.ChangeMemberRole(context.Background(), "actor", "ws-1", "u-bob", "role-admin"))

		roleID, err := f.svc.MemberRoleID(context.Background(), "ws-1", "u-bob")
		require.NoError(t, err)
		assert.Equal(t, "role-admin", roleID)
	})
}

func TestResolvePendingInvites(t *testing.T) {
	t.Run("resolves every pending invite across workspaces into memberships and clears them", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.svc.InviteMember(context.Background(), "actor", "ws-1", "bob", "role-editor"))
		require.NoError(t, f.svc.InviteMember(context.Background(), "actor", "ws-2", "bob", "role-admin"))

		require.NoError(t, f.svc.ResolvePendingInvites(context.Background(), "bob", "u-bob"))

		roleID, err := f.svc.MemberRoleID(context.Background(), "ws-1", "u-bob")
		require.NoError(t, err)
		assert.Equal(t, "role-editor", roleID)
		roleID, err = f.svc.MemberRoleID(context.Background(), "ws-2", "u-bob")
		require.NoError(t, err)
		assert.Equal(t, "role-admin", roleID)

		pending, err := f.invites.ListByLogin(context.Background(), "bob")
		require.NoError(t, err)
		assert.Empty(t, pending)
	})
	t.Run("login with no pending invites is a no-op", func(t *testing.T) {
		f := newInviteFixture()
		require.NoError(t, f.svc.ResolvePendingInvites(context.Background(), "nobody", "u-nobody"))
	})
}

func TestSetMentionChipTemplate(t *testing.T) {
	newFixture := func() (*Service, *fakeRepo, *fakeWorkspacePermissionGate) {
		repo := newFakeRepo()
		wsPerms := newFakeWorkspacePermissionGate()
		s := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, newFakePermissionGate(), newFakeRoleNameGate(), wsPerms, newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{}, newFakeAccountGate())
		s.now = func() time.Time { return fixedNow }
		return s, repo, wsPerms
	}
	t.Run("actor without workspaces:write is forbidden and the template is untouched", func(t *testing.T) {
		s, _, wsPerms := newFixture()
		w, err := s.Create(context.Background(), "u-1", "Acme")
		require.NoError(t, err)
		wsPerms.perms["u-1"] = []string{"members:write"}

		_, err = s.SetMentionChipTemplate(context.Background(), "u-1", w.ID, "{ticket.Project}")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		got, err := s.Get(context.Background(), w.ID)
		require.NoError(t, err)
		assert.Equal(t, DefaultMentionChipTemplate, got.MentionChipTemplate)
	})
	t.Run("unknown workspace is not found", func(t *testing.T) {
		s, _, wsPerms := newFixture()
		wsPerms.perms["u-1"] = []string{"workspaces:write"}
		_, err := s.SetMentionChipTemplate(context.Background(), "u-1", "missing", "{ticket.Project}")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("new workspace starts with the default template", func(t *testing.T) {
		s, _, _ := newFixture()
		w, err := s.Create(context.Background(), "u-1", "Acme")
		require.NoError(t, err)
		assert.Equal(t, DefaultMentionChipTemplate, w.MentionChipTemplate)
	})
	t.Run("workspaces:write holder changes only that workspace's template", func(t *testing.T) {
		s, _, wsPerms := newFixture()
		a, err := s.Create(context.Background(), "u-1", "A")
		require.NoError(t, err)
		b, err := s.Create(context.Background(), "u-1", "B")
		require.NoError(t, err)
		wsPerms.perms["u-1"] = []string{"workspaces:write"}
		later := fixedNow.Add(time.Hour)
		s.now = func() time.Time { return later }

		got, err := s.SetMentionChipTemplate(context.Background(), "u-1", a.ID, "{ticket.Project} {ticket.Ticket}")
		require.NoError(t, err)
		assert.Equal(t, "{ticket.Project} {ticket.Ticket}", got.MentionChipTemplate)
		assert.Equal(t, "A", got.Name, "rename fields survive a template change")
		assert.Equal(t, later, got.UpdatedAt)

		stored, err := s.Get(context.Background(), a.ID)
		require.NoError(t, err)
		assert.Equal(t, "{ticket.Project} {ticket.Ticket}", stored.MentionChipTemplate)
		other, err := s.Get(context.Background(), b.ID)
		require.NoError(t, err)
		assert.Equal(t, DefaultMentionChipTemplate, other.MentionChipTemplate)
	})
}

func TestAddMember(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(f *inviteTestFixture, accounts *fakeAccountGate)
		userID  string
		roleID  string
		wantErr error
	}{
		{"empty user id is invalid", nil, " ", "role-editor", apperrs.ErrInvalid},
		{"empty role id is invalid", nil, "bob", " ", apperrs.ErrInvalid},
		{"removed account is refused", func(_ *inviteTestFixture, a *fakeAccountGate) {
			a.accounts["bob"] = &TeamAccount{ID: "bob", Login: "bob", Status: AccountStatusRemoved}
		}, "bob", "role-editor", apperrs.ErrInvalid},
		{"existing member is a conflict", func(f *inviteTestFixture, _ *fakeAccountGate) {
			require.NoError(t, f.repo.AddMember(t.Context(), &Member{UserID: "bob", WorkspaceID: "ws-1", RoleID: "role-viewer"}))
		}, "bob", "role-editor", apperrs.ErrConflict},
		{"a role carrying a permission the actor lacks is refused", func(f *inviteTestFixture, _ *fakeAccountGate) {
			f.nameGate.perms["role-instance"] = permissions.SetOf(permissions.InstanceWrite)
		}, "bob", "role-instance", apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newInviteFixture()
			accounts := newFakeAccountGate()
			f.svc.accounts = accounts
			f.grantManageMembers("actor")
			if tt.setup != nil {
				tt.setup(f, accounts)
			}
			err := f.svc.AddMember(t.Context(), "actor", "ws-1", tt.userID, tt.roleID)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
	t.Run("adds the account with the role and announces it", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.svc.AddMember(t.Context(), "actor", "ws-1", "bob", "role-editor"))
		roleID, err := f.repo.RoleIDFor(t.Context(), "ws-1", "bob")
		require.NoError(t, err)
		assert.Equal(t, "role-editor", roleID)
		require.Len(t, f.repo.events, 1)
		assert.Equal(t, TopicWorkspaceMemberAdded, f.repo.events[0].Topic)
		assert.Equal(t, MemberEvent{UserID: "bob", WorkspaceID: "ws-1", ActorID: "actor"}, f.repo.events[0].Payload)
	})
}

func TestSetMemberOverrides(t *testing.T) {
	set := func(actions ...permissions.Action) *permissions.Set {
		s := permissions.SetOf(actions...)
		return &s
	}
	newMemberFixture := func(t *testing.T) *inviteTestFixture {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.repo.AddMember(t.Context(), &Member{UserID: "bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))
		return f
	}
	tests := []struct {
		name        string
		allow, deny *permissions.Set
	}{
		{"neither set is invalid", nil, nil},
		{"an unknown permission is invalid", set("docs:fly"), nil},
		{"allowing and denying one permission is invalid", set(permissions.DocsWrite), set(permissions.DocsWrite)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newMemberFixture(t)
			err := f.svc.SetMemberOverrides(t.Context(), "actor", "ws-1", "bob", tt.allow, tt.deny)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
	t.Run("an allow the actor does not hold is refused", func(t *testing.T) {
		f := newMemberFixture(t)
		err := f.svc.SetMemberOverrides(t.Context(), "actor", "ws-1", "bob", set(permissions.InstanceWrite), nil)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("an allow the member already had stays without the actor holding it", func(t *testing.T) {
		f := newMemberFixture(t)
		require.NoError(t, f.repo.SetOverrides(t.Context(), "ws-1", "bob", permissions.SetOf(permissions.InstanceWrite), nil))
		require.NoError(t, f.svc.SetMemberOverrides(t.Context(), "actor", "ws-1", "bob", set(permissions.InstanceWrite), set(permissions.DocsDelete)))
	})
	t.Run("lifting a deny the actor does not hold is refused, since it hands back what the role gives", func(t *testing.T) {
		f := newMemberFixture(t)
		require.NoError(t, f.repo.SetOverrides(t.Context(), "ws-1", "bob", nil, permissions.SetOf(permissions.InstanceWrite)))
		err := f.svc.SetMemberOverrides(t.Context(), "actor", "ws-1", "bob", nil, set())
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("an omitted set keeps its current value", func(t *testing.T) {
		f := newMemberFixture(t)
		f.wsPerms.perms["actor"] = []string{"members:write", "docs:write", "projects:write"}
		require.NoError(t, f.svc.SetMemberOverrides(t.Context(), "actor", "ws-1", "bob", set(permissions.DocsWrite), set(permissions.DocsDelete)))
		require.NoError(t, f.svc.SetMemberOverrides(t.Context(), "actor", "ws-1", "bob", set(permissions.ProjectsWrite), nil))
		allow, deny, err := f.repo.Overrides(t.Context(), "ws-1", "bob")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(permissions.ProjectsWrite), allow)
		assert.Equal(t, permissions.SetOf(permissions.DocsDelete), deny)
		assert.Equal(t, TopicWorkspaceMemberUpdated, f.repo.events[len(f.repo.events)-1].Topic)
	})
}

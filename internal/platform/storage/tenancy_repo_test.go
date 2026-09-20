package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/tenancy"
)

func newTestWorkspace(id, name string) *tenancy.Workspace {
	now := time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)
	return &tenancy.Workspace{ID: id, Name: name, CreatedAt: now, UpdatedAt: now}
}

func TestWorkspacesRepo_Create_Get_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestWorkspace("ws-1", "Acme")
	require.NoError(t, s.Workspaces.Create(context.Background(), want))

	got, err := s.Workspaces.Get(context.Background(), "ws-1")
	require.NoError(t, err)
	assert.Equal(t, "Acme", got.Name)
}

func TestWorkspacesRepo_Create_Duplicate_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "A")))
	err := s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "B"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestWorkspacesRepo_Get_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Workspaces.Get(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestWorkspacesRepo_ListForUser_ScopesToMembership(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-2", "Other Co")))
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-1", "1", "u-1"))
	require.NoError(t, err)
	role := newTestRole("role-1", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role))

	require.NoError(t, s.WorkspaceMembers.AddMember(context.Background(), &tenancy.Member{
		UserID: "u-1", WorkspaceID: "ws-1", RoleID: role.ID, CreatedAt: time.Now(),
	}))

	got, err := s.Workspaces.ListForUser(context.Background(), "u-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "ws-1", got[0].ID)

	got, err = s.Workspaces.ListForUser(context.Background(), "u-nobody")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestWorkspaceMembersRepo_AddMember_DuplicateMembership_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-1", "1", "u-1"))
	require.NoError(t, err)
	role := newTestRole("role-1", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role))

	require.NoError(t, s.WorkspaceMembers.AddMember(context.Background(), &tenancy.Member{
		UserID: "u-1", WorkspaceID: "ws-1", RoleID: role.ID, CreatedAt: time.Now(),
	}))
	err = s.WorkspaceMembers.AddMember(context.Background(), &tenancy.Member{
		UserID: "u-1", WorkspaceID: "ws-1", RoleID: role.ID, CreatedAt: time.Now(),
	})
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestWorkspaceMembersRepo_AddMember_UnknownWorkspace_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-1", "1", "u-1"))
	require.NoError(t, err)

	err = s.WorkspaceMembers.AddMember(context.Background(), &tenancy.Member{
		UserID: "u-1", WorkspaceID: "missing", RoleID: "role-1", CreatedAt: time.Now(),
	})
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestWorkspaceMembersRepo_AddMember_UnknownRole_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-1", "1", "u-1"))
	require.NoError(t, err)

	err = s.WorkspaceMembers.AddMember(context.Background(), &tenancy.Member{
		UserID: "u-1", WorkspaceID: "ws-1", RoleID: "missing-role", CreatedAt: time.Now(),
	})
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestWorkspaceMembersRepo_RoleIDFor(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-1", "1", "u-1"))
	require.NoError(t, err)
	role := newTestRole("role-1", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role))
	require.NoError(t, s.WorkspaceMembers.AddMember(context.Background(), &tenancy.Member{
		UserID: "u-1", WorkspaceID: "ws-1", RoleID: role.ID, CreatedAt: time.Now(),
	}))

	got, err := s.WorkspaceMembers.RoleIDFor(context.Background(), "ws-1", "u-1")
	require.NoError(t, err)
	assert.Equal(t, role.ID, got)

	_, err = s.WorkspaceMembers.RoleIDFor(context.Background(), "ws-1", "u-nobody")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestWorkspaceMembersRepo_ListByWorkspace(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-1", "1", "u-1"))
	require.NoError(t, err)
	_, _, err = s.Users.UpsertUser(context.Background(), newTestUser("u-2", "2", "u-2"))
	require.NoError(t, err)
	role := newTestRole("role-1", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role))
	require.NoError(t, s.WorkspaceMembers.AddMember(context.Background(), &tenancy.Member{
		UserID: "u-1", WorkspaceID: "ws-1", RoleID: role.ID, CreatedAt: time.Now(),
	}))
	require.NoError(t, s.WorkspaceMembers.AddMember(context.Background(), &tenancy.Member{
		UserID: "u-2", WorkspaceID: "ws-1", RoleID: role.ID, CreatedAt: time.Now(),
	}))

	got, err := s.WorkspaceMembers.ListByWorkspace(context.Background(), "ws-1")
	require.NoError(t, err)
	require.Len(t, got, 2)

	got, err = s.WorkspaceMembers.ListByWorkspace(context.Background(), "ws-nobody")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestWorkspaceMembersRepo_RemoveMember(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-1", "1", "u-1"))
	require.NoError(t, err)
	role := newTestRole("role-1", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role))
	require.NoError(t, s.WorkspaceMembers.AddMember(context.Background(), &tenancy.Member{
		UserID: "u-1", WorkspaceID: "ws-1", RoleID: role.ID, CreatedAt: time.Now(),
	}))

	require.NoError(t, s.WorkspaceMembers.RemoveMember(context.Background(), "ws-1", "u-1"))

	_, err = s.WorkspaceMembers.RoleIDFor(context.Background(), "ws-1", "u-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	err = s.WorkspaceMembers.RemoveMember(context.Background(), "ws-1", "u-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestWorkspaceMembersRepo_SetRole(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-1", "1", "u-1"))
	require.NoError(t, err)
	role1 := newTestRole("role-1", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role1))
	role2 := newTestRole("role-2", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role2))
	require.NoError(t, s.WorkspaceMembers.AddMember(context.Background(), &tenancy.Member{
		UserID: "u-1", WorkspaceID: "ws-1", RoleID: role1.ID, CreatedAt: time.Now(),
	}))

	require.NoError(t, s.WorkspaceMembers.SetRole(context.Background(), "ws-1", "u-1", role2.ID))

	got, err := s.WorkspaceMembers.RoleIDFor(context.Background(), "ws-1", "u-1")
	require.NoError(t, err)
	assert.Equal(t, role2.ID, got)

	err = s.WorkspaceMembers.SetRole(context.Background(), "ws-1", "u-nobody", role2.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestWorkspaceInvitesRepo_Upsert_ListByWorkspace_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-owner", "1", "owner"))
	require.NoError(t, err)
	role := newTestRole("role-1", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role))

	require.NoError(t, s.WorkspaceInvites.Upsert(context.Background(), &tenancy.Invite{
		WorkspaceID: "ws-1", Login: "bob", RoleID: role.ID, InvitedBy: "u-owner", CreatedAt: time.Now(),
	}))

	got, err := s.WorkspaceInvites.ListByWorkspace(context.Background(), "ws-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "bob", got[0].Login)
	assert.Equal(t, role.ID, got[0].RoleID)
}

func TestWorkspaceInvitesRepo_Upsert_SameLoginTwice_UpdatesRoleNotDuplicate(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-owner", "1", "owner"))
	require.NoError(t, err)
	role1 := newTestRole("role-1", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role1))
	role2 := newTestRole("role-2", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role2))

	require.NoError(t, s.WorkspaceInvites.Upsert(context.Background(), &tenancy.Invite{
		WorkspaceID: "ws-1", Login: "bob", RoleID: role1.ID, InvitedBy: "u-owner", CreatedAt: time.Now(),
	}))
	require.NoError(t, s.WorkspaceInvites.Upsert(context.Background(), &tenancy.Invite{
		WorkspaceID: "ws-1", Login: "bob", RoleID: role2.ID, InvitedBy: "u-owner", CreatedAt: time.Now(),
	}))

	got, err := s.WorkspaceInvites.ListByWorkspace(context.Background(), "ws-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, role2.ID, got[0].RoleID)
}

func TestWorkspaceInvitesRepo_ListByLogin_AcrossWorkspaces(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-2", "Other Co")))
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-owner", "1", "owner"))
	require.NoError(t, err)
	role1 := newTestRole("role-1", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role1))
	role2 := newTestRole("role-2", "ws-2")
	require.NoError(t, s.Roles.Create(context.Background(), role2))

	require.NoError(t, s.WorkspaceInvites.Upsert(context.Background(), &tenancy.Invite{
		WorkspaceID: "ws-1", Login: "bob", RoleID: role1.ID, InvitedBy: "u-owner", CreatedAt: time.Now(),
	}))
	require.NoError(t, s.WorkspaceInvites.Upsert(context.Background(), &tenancy.Invite{
		WorkspaceID: "ws-2", Login: "bob", RoleID: role2.ID, InvitedBy: "u-owner", CreatedAt: time.Now(),
	}))

	got, err := s.WorkspaceInvites.ListByLogin(context.Background(), "bob")
	require.NoError(t, err)
	require.Len(t, got, 2)

	got, err = s.WorkspaceInvites.ListByLogin(context.Background(), "nobody")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestWorkspaceInvitesRepo_Delete_RemovesInvite(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-owner", "1", "owner"))
	require.NoError(t, err)
	role := newTestRole("role-1", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role))
	require.NoError(t, s.WorkspaceInvites.Upsert(context.Background(), &tenancy.Invite{
		WorkspaceID: "ws-1", Login: "bob", RoleID: role.ID, InvitedBy: "u-owner", CreatedAt: time.Now(),
	}))

	require.NoError(t, s.WorkspaceInvites.Delete(context.Background(), "ws-1", "bob"))

	got, err := s.WorkspaceInvites.ListByWorkspace(context.Background(), "ws-1")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestWorkspaceInvitesRepo_Upsert_UnknownWorkspaceOrRole_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u-owner", "1", "owner"))
	require.NoError(t, err)
	role := newTestRole("role-1", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role))

	err = s.WorkspaceInvites.Upsert(context.Background(), &tenancy.Invite{
		WorkspaceID: "missing", Login: "bob", RoleID: role.ID, InvitedBy: "u-owner", CreatedAt: time.Now(),
	})
	require.ErrorIs(t, err, apperrs.ErrConflict)

	err = s.WorkspaceInvites.Upsert(context.Background(), &tenancy.Invite{
		WorkspaceID: "ws-1", Login: "bob", RoleID: "missing-role", InvitedBy: "u-owner", CreatedAt: time.Now(),
	})
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

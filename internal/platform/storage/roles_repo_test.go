package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
)

func newTestRole(id, workspaceID string) *roles.Role {
	now := time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)
	return &roles.Role{ID: id, WorkspaceID: workspaceID, Name: "Editors", CreatedAt: now, UpdatedAt: now}
}

func TestRolesRepo_Create_Get_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	want := newTestRole("role-1", "ws-1")
	want.Permissions = permissions.SetOf(permissions.RolesWrite, permissions.ProjectsWrite)
	require.NoError(t, s.Roles.Create(context.Background(), want))

	got, err := s.Roles.Get(context.Background(), "role-1")
	require.NoError(t, err)
	assert.Equal(t, "Editors", got.Name)
	assert.Equal(t, "ws-1", got.WorkspaceID)
	assert.False(t, got.IsOwnerRole)
	assert.True(t, got.Permissions.Has(permissions.RolesWrite))
	assert.True(t, got.Permissions.Has(permissions.ProjectsWrite))
	assert.False(t, got.Permissions.Has(permissions.MembersWrite))
}

func TestRolesRepo_Get_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Roles.Get(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestRolesRepo_Create_UnknownWorkspace_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Roles.Create(context.Background(), newTestRole("role-1", "missing-workspace"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestRolesRepo_Create_SingletonOwnerRole_Constraint(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))

	first := newTestRole("role-owner-1", "ws-1")
	first.Name = "Owner"
	first.IsOwnerRole = true
	require.NoError(t, s.Roles.Create(context.Background(), first))

	second := newTestRole("role-owner-2", "ws-1")
	second.Name = "Owner"
	second.IsOwnerRole = true
	err := s.Roles.Create(context.Background(), second)
	require.ErrorIs(t, err, apperrs.ErrConflict, "at most one is_owner_role=true row per workspace")

	// A second workspace may still have its own Owner role.
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-2", "Other Co")))
	other := newTestRole("role-owner-3", "ws-2")
	other.Name = "Owner"
	other.IsOwnerRole = true
	require.NoError(t, s.Roles.Create(context.Background(), other))

	// Non-owner roles in the same workspace are unaffected by the singleton constraint.
	require.NoError(t, s.Roles.Create(context.Background(), newTestRole("role-1", "ws-1")))
	require.NoError(t, s.Roles.Create(context.Background(), newTestRole("role-2", "ws-1")))
}

func TestRolesRepo_List_ScopesToWorkspace(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-2", "Other Co")))
	require.NoError(t, s.Roles.Create(context.Background(), newTestRole("role-1", "ws-1")))
	require.NoError(t, s.Roles.Create(context.Background(), newTestRole("role-2", "ws-2")))

	got, err := s.Roles.List(context.Background(), "ws-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "role-1", got[0].ID)
}

func TestRolesRepo_Update(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	role := newTestRole("role-1", "ws-1")
	require.NoError(t, s.Roles.Create(context.Background(), role))

	role.Name = "Reviewers"
	role.Permissions = permissions.SetOf(permissions.RolesWrite)
	role.UpdatedAt = time.Now()
	require.NoError(t, s.Roles.Update(context.Background(), role))

	got, err := s.Roles.Get(context.Background(), "role-1")
	require.NoError(t, err)
	assert.Equal(t, "Reviewers", got.Name)
	assert.True(t, got.Permissions.Has(permissions.RolesWrite))
}

func TestRolesRepo_Update_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Roles.Update(context.Background(), newTestRole("missing", "ws-1"))
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestRolesRepo_Delete(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	require.NoError(t, s.Roles.Create(context.Background(), newTestRole("role-1", "ws-1")))

	require.NoError(t, s.Roles.Delete(context.Background(), "role-1"))
	_, err := s.Roles.Get(context.Background(), "role-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestRolesRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Roles.Delete(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

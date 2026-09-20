package storage_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

func newAccessStore(t *testing.T) *storage.Store {
	t.Helper()
	return storage.New(newDB(t), []byte("0123456789abcdef0123456789abcdef"))
}

// seedUserForOverwrite creates a user so an overwrite row satisfies its FK.
// permission_overwrites has no FK on resource_id (it spans every resource
// type), so no resource needs seeding.
func seedUserForOverwrite(t *testing.T, s *storage.Store, userID string) {
	t.Helper()
	_, _, err := s.Users.UpsertUser(context.Background(),
		&auth.User{ID: userID, Provider: auth.ProviderGitHub, ProviderUserID: userID, Login: userID})
	require.NoError(t, err)
}

func TestAccessRepo_OverwriteCRUD(t *testing.T) {
	ctx := context.Background()
	s := newAccessStore(t)
	seedUserForOverwrite(t, s, "user-1")
	require.NoError(t, s.Access.Set(ctx, "doc", "doc-1", "user-1", permissions.SetOf(permissions.DocsRead), nil))

	t.Run("get returns stored masks", func(t *testing.T) {
		ow, err := s.Access.Get(ctx, "doc", "doc-1", "user-1")
		require.NoError(t, err)
		assert.Equal(t, "doc", ow.ResourceType)
		assert.Equal(t, "doc-1", ow.ResourceID)
		assert.Equal(t, "user-1", ow.UserID)
		assert.Equal(t, permissions.SetOf(permissions.DocsRead), ow.Allow)
		assert.Equal(t, permissions.Set(nil), ow.Deny)
	})
	t.Run("missing overwrite is not found", func(t *testing.T) {
		_, err := s.Access.Get(ctx, "doc", "doc-1", "ghost")
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("set upserts both masks", func(t *testing.T) {
		require.NoError(t, s.Access.Set(ctx, "doc", "doc-1", "user-1",
			permissions.SetOf(permissions.DocsRead, permissions.DocsWrite), permissions.SetOf(permissions.DocsDelete)))
		ow, err := s.Access.Get(ctx, "doc", "doc-1", "user-1")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(permissions.DocsRead, permissions.DocsWrite), ow.Allow)
		assert.Equal(t, permissions.SetOf(permissions.DocsDelete), ow.Deny)
	})
	t.Run("both sets empty deletes the row", func(t *testing.T) {
		require.NoError(t, s.Access.Set(ctx, "doc", "doc-1", "user-1", nil, nil))
		_, err := s.Access.Get(ctx, "doc", "doc-1", "user-1")
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("list by resource", func(t *testing.T) {
		for _, uid := range []string{"u1", "u2"} {
			seedUserForOverwrite(t, s, uid)
		}
		require.NoError(t, s.Access.Set(ctx, "doc", "doc-1", "u1", permissions.SetOf(permissions.DocsRead), nil))
		require.NoError(t, s.Access.Set(ctx, "doc", "doc-1", "u2", permissions.SetOf(permissions.DocsWrite), nil))
		require.NoError(t, s.Access.Set(ctx, "doc", "doc-2", "u1", permissions.SetOf(permissions.DocsRead), nil))
		overwrites, err := s.Access.ListByResource(ctx, "doc", "doc-1")
		require.NoError(t, err)
		assert.Len(t, overwrites, 2)
	})
	t.Run("resource types are independent rows for the same id", func(t *testing.T) {
		require.NoError(t, s.Access.Set(ctx, "workspace", "doc-1", "u1", permissions.SetOf(permissions.ProjectsWrite), nil))
		docOw, err := s.Access.Get(ctx, "doc", "doc-1", "u1")
		require.NoError(t, err)
		wsOw, err := s.Access.Get(ctx, "workspace", "doc-1", "u1")
		require.NoError(t, err)
		assert.NotEqual(t, docOw.Allow, wsOw.Allow)
	})
	t.Run("delete by resource removes all", func(t *testing.T) {
		require.NoError(t, s.Access.DeleteByResource(ctx, "doc", "doc-1"))
		overwrites, err := s.Access.ListByResource(ctx, "doc", "doc-1")
		require.NoError(t, err)
		assert.Empty(t, overwrites)
	})
	t.Run("has allow any scopes by resource type", func(t *testing.T) {
		seedUserForOverwrite(t, s, "manager")
		require.NoError(t, s.Access.Set(ctx, "doc", "doc-3", "manager", permissions.SetOf(permissions.PermissionsWrite), nil))
		has, err := s.Access.HasAllowAny(ctx, "doc", "manager", permissions.PermissionsWrite)
		require.NoError(t, err)
		assert.True(t, has)
		has, err = s.Access.HasAllowAny(ctx, "doc", "manager", permissions.DocsRead)
		require.NoError(t, err)
		assert.False(t, has)
		has, err = s.Access.HasAllowAny(ctx, "doc", "manager", permissions.Action("permissions:writ"))
		require.NoError(t, err)
		assert.False(t, has, "the stored JSON is matched on the whole quoted value, never a prefix")
		has, err = s.Access.HasAllowAny(ctx, "workspace", "manager", permissions.PermissionsWrite)
		require.NoError(t, err)
		assert.False(t, has, "scoped to resource_type, a doc-level allow doesn't leak into workspace")
	})
}

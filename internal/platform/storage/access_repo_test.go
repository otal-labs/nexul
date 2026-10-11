package storage_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
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
		&auth.Identity{UserID: userID, Provider: auth.ProviderGitHub, ProviderUserID: userID, Login: userID})
	require.NoError(t, err)
}

func TestAccessRepo_Set_WritesItsEventsWithTheRowAndTheRemoval(t *testing.T) {
	ctx := context.Background()
	s := newAccessStore(t)
	seedUserForOverwrite(t, s, "user-1")
	event := func(id string) eventbus.OutboxEvent {
		return eventbus.OutboxEvent{ID: id, Topic: access.TopicGrantChanged, Payload: access.GrantEvent{ResourceType: "doc", ResourceID: "doc-1", UserID: "user-1"}}
	}

	require.NoError(t, s.Access.Set(ctx, "doc", "doc-1", "user-1", permissions.SetOf(permissions.DocsRead), nil, event("evt-grant")))
	require.NoError(t, s.Access.Set(ctx, "doc", "doc-1", "user-1", nil, nil, event("evt-revoke")))

	entries, err := s.Outbox.Unpublished(ctx, 10)
	require.NoError(t, err)
	var ids []string
	for _, e := range entries {
		ids = append(ids, e.ID)
	}
	assert.ElementsMatch(t, []string{"evt-grant", "evt-revoke"}, ids)
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

// roleRulesStore is a store where bob holds role-front and sam role-client, with doc rules for both roles, for bob, and
// for sam, written in SQL since no writer names a role yet.
func roleRulesStore(t *testing.T) (*storage.Store, *sql.DB) {
	t.Helper()
	ctx := context.Background()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	now := time.Now()
	for user, role := range map[string]string{"bob": "role-front", "sam": "role-client"} {
		seedUserForOverwrite(t, s, user)
		require.NoError(t, s.Roles.Create(ctx, &roles.Role{ID: role, WorkspaceID: "workspace-default", Name: role, CreatedAt: now, UpdatedAt: now}))
		require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: "workspace-default", RoleID: role, CreatedAt: now}))
	}
	require.NoError(t, s.Access.Set(ctx, "doc", "doc-1", "bob", nil, permissions.SetOf(permissions.DocsWrite)))
	require.NoError(t, s.Access.Set(ctx, "doc", "doc-1", "sam", permissions.SetOf(permissions.DocsRead), nil))
	_, err := db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, allow, deny, created_at, updated_at) VALUES
    ('doc', 'doc-1', 'role', 'role-front', '["docs:write","permissions:write"]', '[]', 1, 1),
    ('doc', 'doc-2', 'role', 'role-front', '[]', '["docs:read"]', 1, 1),
    ('doc', 'doc-1', 'role', 'role-client', '[]', '["docs:read"]', 1, 1)`)
	require.NoError(t, err)
	return s, db
}

func TestAccessRepo_RulesReachThePersonAndTheRolesTheyHold(t *testing.T) {
	ctx := context.Background()
	s, db := roleRulesStore(t)

	rules, err := s.Access.Rules(ctx, "doc", []string{"doc-1", "doc-2", "doc-3"}, "bob")
	require.NoError(t, err)
	require.Len(t, rules, 2, "a doc without rules is left out")
	assert.Equal(t, permissions.SetOf(permissions.DocsWrite), rules["doc-1"].Person.Deny)
	require.Len(t, rules["doc-1"].Roles, 1, "role-client's rule reaches only its holders")
	assert.Equal(t, access.Overwrite{ResourceType: "doc", ResourceID: "doc-1", Target: access.TargetRole, RoleID: "role-front",
		Allow: permissions.SetOf(permissions.DocsWrite, permissions.PermissionsWrite), CreatedAt: time.Unix(1, 0).UTC(), UpdatedAt: time.Unix(1, 0).UTC()}, *rules["doc-1"].Roles[0])
	assert.Nil(t, rules["doc-2"].Person)

	byType, err := s.Access.RulesOfType(ctx, "doc", "bob")
	require.NoError(t, err)
	assert.Equal(t, rules, byType)

	has, err := s.Access.HasAllowAny(ctx, "doc", "bob", permissions.PermissionsWrite)
	require.NoError(t, err)
	assert.True(t, has, "a rule for a role they hold counts")
	has, err = s.Access.HasAllowAny(ctx, "doc", "sam", permissions.PermissionsWrite)
	require.NoError(t, err)
	assert.False(t, has)

	all, err := s.Access.ListByResource(ctx, "doc", "doc-1")
	require.NoError(t, err)
	var targets []string
	for _, ow := range all {
		targets = append(targets, ow.Target+":"+ow.UserID+ow.RoleID)
	}
	assert.Equal(t, []string{"person:bob", "person:sam", "role:role-client", "role:role-front"}, targets)

	_, err = db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, allow, deny, created_at, updated_at)
VALUES ('doc', 'doc-1', 'role', 'role-front', '[]', '[]', 1, 1)`)
	require.Error(t, err, "one rule per role on an entity")
	_, err = db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, role_id, allow, deny, created_at, updated_at)
VALUES ('doc', 'doc-9', 'role', 'bob', 'role-front', '[]', '[]', 1, 1)`)
	require.Error(t, err, "a rule names a person or a role, never both")
}

func TestAccessRepo_DeletingARoleOrAnAccountDeletesItsRules(t *testing.T) {
	ctx := context.Background()
	s, db := roleRulesStore(t)
	count := func(where string) int {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM permission_overwrites WHERE `+where).Scan(&n))
		return n
	}

	require.NoError(t, s.WorkspaceMembers.RemoveMember(ctx, "workspace-default", "sam"))
	require.NoError(t, s.Roles.Delete(ctx, "role-client"))
	assert.Zero(t, count(`role_id = 'role-client'`))
	assert.Equal(t, 2, count(`role_id = 'role-front'`))

	require.NoError(t, s.Users.SetAccountStatus(ctx, "bob", auth.AccountRemoved))
	assert.Zero(t, count(`user_id = 'bob'`))
	assert.Equal(t, 1, count(`user_id = 'sam'`))
	assert.Equal(t, 2, count(`role_id = 'role-front'`), "a role's rules outlive one holder")

	_, err := db.Exec(`DELETE FROM users WHERE id = 'sam'`)
	require.NoError(t, err)
	assert.Zero(t, count(`user_id = 'sam'`))
}

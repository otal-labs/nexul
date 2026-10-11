package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// TestRoleSet_EveryEvaluatorAgrees: the grid, roles' check, the invitation SQL and the owner list answer from the union.
func TestRoleSet_EveryEvaluatorAgrees(t *testing.T) {
	db := mentionsTestDB(t)
	svc, store := wiredOver(t, db)
	seedPermFixture(t, db, svc, store)
	// In SQL, since every writer still gives one role.
	_, err := db.Exec(`INSERT INTO member_roles (user_id, workspace_id, role_id, created_at) VALUES
    ('u-plain', 'workspace-default', 'role-manager', 2),
    ('u-plain', 'workspace-default', 'role-reader', 3),
    ('u-writer', 'workspace-default', 'role-owner', 2)`)
	require.NoError(t, err)
	_, err = svc.authSvc.UpdateInstanceURL(as(uOwner), uOwner, "https://nexul.example.com")
	require.NoError(t, err)

	t.Run("the access grid", func(t *testing.T) {
		held := permissions.SetOfStrings(svc.accessSvc.WorkspacePermissions(as(uPlain), uPlain, wsDefault))
		for _, a := range []permissions.Action{permissions.RolesWrite, permissions.MembersWrite, permissions.DocsRead, permissions.ReposRead} {
			assert.True(t, held.Has(a), a)
		}
		assert.False(t, held.Has(permissions.TicketsWrite), "no role of theirs allows it")
		assert.Len(t, svc.accessSvc.WorkspacePermissions(as(uWriter), uWriter, wsDefault), len(permissions.AllActions()), "the Owner among their roles")
		owner, err := svc.accessSvc.IsOwner(as(uWriter), uWriter, wsDefault)
		require.NoError(t, err)
		assert.True(t, owner)
	})
	t.Run("the roles domain", func(t *testing.T) {
		_, err := svc.rolesSvc.Create(as(uPlain), wsDefault, uPlain, "Triage", grant("docs:read", "members:write"))
		require.NoError(t, err, "roles:write from one role, the grant from both")
		_, err = svc.rolesSvc.Create(as(uPlain), wsDefault, uPlain, "Shipper", grant("tickets:write"))
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		_, err = svc.rolesSvc.Create(as(uWriter), wsDefault, uWriter, "Upgrader", grant("instance:write"))
		require.NoError(t, err)
	})
	t.Run("the invitation SQL", func(t *testing.T) {
		created, err := svc.invitationSvc.Create(as(uPlain), uPlain, tenancy.CreateInvitationInput{ExpiresInDays: 1,
			Grants: []*tenancy.InvitationGrant{{WorkspaceID: wsDefault, RoleID: "role-reader"}}})
		require.NoError(t, err, "members:write from one role, the Reader's set from the other")
		listed, err := svc.invitationSvc.List(as(uPlain), uPlain)
		require.NoError(t, err)
		require.Len(t, listed, 1)
		assert.Equal(t, created.Invitation.ID, listed[0].ID)
		_, err = svc.invitationSvc.Create(as(uPlain), uPlain, tenancy.CreateInvitationInput{ExpiresInDays: 1,
			Grants: []*tenancy.InvitationGrant{{WorkspaceID: wsDefault, RoleID: "role-writer"}}})
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("the owner list", func(t *testing.T) {
		owners, err := store.Users.ListActiveOwnerIDs(as(uOwner))
		require.NoError(t, err)
		assert.Equal(t, []string{uOwner, uWriter}, owners)
	})
}

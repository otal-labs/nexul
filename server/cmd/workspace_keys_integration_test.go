package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// TestIntegration_TicketKeysAreUniquePerWorkspace walks two workspaces that both have a WEB project through the
// wired use-cases the HTTP gateway and MCP tools call (ADR 0089).
func TestIntegration_TicketKeysAreUniquePerWorkspace(t *testing.T) {
	f := newPermFixture(t)
	owner := as(uOwner)

	rix, err := f.svc.tenancySvc.Create(owner, uOwner, "Rixwave")
	require.NoError(t, err)
	otal, err := f.svc.tenancySvc.Create(owner, uOwner, "Otal")
	require.NoError(t, err)
	assert.Equal(t, []string{"rixwave", "otal"}, []string{rix.Slug, otal.Slug})

	rixWeb, err := f.svc.workspaceSvc.Create(owner, uOwner, rix.ID, "Web", "WEB", "")
	require.NoError(t, err)
	otalWeb, err := f.svc.workspaceSvc.Create(owner, uOwner, otal.ID, "Web", "WEB", "")
	require.NoError(t, err, "another workspace may use the same prefix")
	_, err = f.svc.workspaceSvc.Create(owner, uOwner, otal.ID, "Web again", "WEB", "")
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a prefix stays unique inside its workspace")

	rixTicket, err := f.svc.ticketsSvc.Create(owner, rixWeb.ID, "Rixwave login", "", "", "")
	require.NoError(t, err)
	otalTicket, err := f.svc.ticketsSvc.Create(owner, otalWeb.ID, "Otal login", "", "", "")
	require.NoError(t, err)

	_, err = f.svc.ticketsSvc.Resolve(owner, "", "WEB-1")
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.ErrorContains(t, err, "WEB-1 exists in otal and rixwave; pass workspace")
	got, err := f.svc.ticketsSvc.Resolve(owner, "rixwave", "WEB-1")
	require.NoError(t, err)
	assert.Equal(t, rixTicket.ID, got.ID)
	got, err = f.svc.ticketsSvc.Resolve(owner, otal.ID, "web-1")
	require.NoError(t, err)
	assert.Equal(t, otalTicket.ID, got.ID)

	now := time.Now()
	require.NoError(t, f.store.Roles.Create(owner, &roles.Role{ID: "role-otal-reader", WorkspaceID: otal.ID, Name: "Reader", Permissions: grant("tickets:read"), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(owner, &tenancy.Member{UserID: uReader, WorkspaceID: otal.ID, RoleID: "role-otal-reader", CreatedAt: now}))
	got, err = f.svc.ticketsSvc.Resolve(as(uReader), "", "WEB-1")
	require.NoError(t, err, "a key that only one of the caller's workspaces holds needs no workspace")
	assert.Equal(t, otalTicket.ID, got.ID)
	_, err = f.svc.ticketsSvc.Resolve(as(uReader), "rixwave", "WEB-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a workspace the caller is not in reads as nothing there")

	ticketIDs := func(workspaceID string) []string {
		results, err := f.svc.mentionsSvc.Search(owner, "WEB-1", workspaceID, 20)
		require.NoError(t, err)
		var ids []string
		for _, r := range results {
			ids = append(ids, r.ID)
		}
		return ids
	}
	assert.Equal(t, []string{otalTicket.ID}, ticketIDs(otal.ID), "the @ picker offers the key in the workspace it is used in")
	assert.ElementsMatch(t, []string{rixTicket.ID, otalTicket.ID}, ticketIDs(""), "and in every workspace without one")
}

package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/memories"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// TestIntegration_DocSourceFollowsTheReadersAccess checks a doc source through the wired lookup and access gate.
func TestIntegration_DocSourceFollowsTheReadersAccess(t *testing.T) {
	f := newPermFixture(t)
	ctx, mem := t.Context(), f.svc.memoriesSvc
	const uMemo = "u-memo"
	now := time.Now()
	_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: uMemo, Provider: auth.ProviderGitHub, ProviderUserID: uMemo, Login: uMemo})
	require.NoError(t, err)
	role := &roles.Role{ID: "role-memo", WorkspaceID: "workspace-default", Name: "Memo", Permissions: grant("memories:read", "memories:write"), CreatedAt: now, UpdatedAt: now}
	require.NoError(t, f.store.Roles.Create(ctx, role))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: uMemo, WorkspaceID: "workspace-default", RoleID: role.ID, CreatedAt: now}))

	doc := memories.InterviewSource{ProjectID: "project-general", Kind: memories.SourceDoc, Ref: f.doc, Stance: memories.StanceFollow}
	_, err = mem.AddSource(as(uMemo), doc)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "adding a doc takes docs:read on it")

	added, err := mem.AddSource(as(uOwner), doc)
	require.NoError(t, err)
	assert.Equal(t, "Spec", added.Label)

	srcs, err := mem.ListSources(as(uMemo), "project-general")
	require.NoError(t, err)
	require.Len(t, srcs, 1)
	assert.True(t, srcs[0].NotVisible)
	assert.Empty(t, srcs[0].Label)

	require.NoError(t, f.store.Docs.Delete(ctx, f.doc))
	srcs, err = mem.ListSources(as(uOwner), "project-general")
	require.NoError(t, err)
	assert.True(t, srcs[0].Gone)
}

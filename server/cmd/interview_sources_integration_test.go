package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/workspace"
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

// TestIntegration_DraftingRunThroughTheWiredSeams walks a drafting run's reads and writes on real SQLite: the runner's
// seams name the project source and its link, the start clears the stale suggestion, and the drafts the run writes land
// with its trail, an equal one dropped and one citing a source under question refused.
func TestIntegration_DraftingRunThroughTheWiredSeams(t *testing.T) {
	f := newPermFixture(t)
	ctx, mem, owner := t.Context(), f.svc.memoriesSvc, as(uOwner)
	require.NoError(t, f.store.Projects.Create(ctx, &workspace.Project{ID: "p-standards", WorkspaceID: wsDefault, Name: "Standards", Prefix: "STD"}))
	add := func(src memories.InterviewSource) string {
		src.ProjectID = pGeneral
		got, err := mem.AddSource(owner, src)
		require.NoError(t, err)
		return got.ID
	}
	followID := add(memories.InterviewSource{Kind: memories.SourcePath, Ref: "practices/testing.md", Stance: memories.StanceFollow})
	questionID := add(memories.InterviewSource{Kind: memories.SourcePath, Ref: "legacy/", Stance: memories.StanceQuestion})
	projectSrc := add(memories.InterviewSource{Kind: memories.SourceProject, Ref: "p-standards", Stance: memories.StanceFollow})
	_, err := mem.SaveAnswer(owner, memories.InterviewAnswer{ProjectID: pGeneral, Question: "Tests", Text: "Real SQLite"})
	require.NoError(t, err)
	_, err = mem.SaveDrafts(owner, pGeneral, []memories.InterviewDraft{
		{Question: "Tests", Text: "Mocks", SourceIDs: []string{followID}},
		{Question: "Stack", Text: "Go", SourceIDs: []string{followID}},
	})
	require.NoError(t, err)

	answers := playsInterviewAnswers{svc: mem}
	srcs, err := answers.ListSources(owner, pGeneral)
	require.NoError(t, err)
	assert.Contains(t, srcs, plays.InterviewSource{ID: projectSrc, Kind: plays.SourceKindProject, Ref: "p-standards", Label: "Standards", Stance: plays.StanceFollow})
	now := time.Now().UTC()
	require.NoError(t, f.store.Pairing.SaveComputer(ctx, pairing.Computer{ID: "c-1", UserID: uOwner, Kind: harness.KindT3Code, Name: "Laptop", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Pairing.SaveProjectLink(ctx, pairing.ProjectLink{UserID: uOwner, ProjectID: "p-standards", ComputerID: "c-1", HarnessProjectID: "t3-standards"}))
	computer, t3Project, err := playsCheckouts{svc: f.svc.pairingSvc}.LinkedProject(owner, uOwner, "p-standards")
	require.NoError(t, err)
	assert.Equal(t, []string{"c-1", "t3-standards"}, []string{computer, t3Project})

	require.NoError(t, answers.ClearSuggestions(owner, pGeneral))
	ds, err := mem.ListDrafts(owner, pGeneral)
	require.NoError(t, err)
	require.Len(t, ds, 1, "the suggested change on the answered question is gone")
	assert.Equal(t, "Stack", ds[0].Question)

	saved, err := mem.SaveDrafts(owner, pGeneral, []memories.InterviewDraft{
		{Question: "Stack", Text: "Go and React", SourceIDs: []string{followID, projectSrc}, Where: "practices/go.md", TrailID: "trail-1"},
		{Question: "Tests", Text: "Real SQLite", SourceIDs: []string{followID}, TrailID: "trail-1"},
	})
	require.NoError(t, err)
	require.Len(t, saved, 1, "the draft equal to the stored answer is dropped")
	assert.Equal(t, "trail-1", saved[0].TrailID)
	_, err = mem.SaveDrafts(owner, pGeneral, []memories.InterviewDraft{{Question: "Style", Text: "Ours", SourceIDs: []string{questionID}}})
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a source under question is never drafted from")
	ds, err = mem.ListDrafts(owner, pGeneral)
	require.NoError(t, err)
	require.Len(t, ds, 1)
	assert.Equal(t, "Go and React", ds[0].Text, "the run's draft replaces the earlier one")
}

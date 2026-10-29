package plays

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// historyFixture is a ticket play at version 1 with instructions "First.", saved by "owner"; "reader" may only read.
func historyFixture(t *testing.T) (*Service, *fakeRepo, *Play) {
	t.Helper()
	repo := newFakeRepo()
	perm := newFakePerm(map[string][]permissions.Action{
		"owner":  {permissions.PlaysRead, permissions.PlaysWrite},
		"reader": {permissions.PlaysRead},
	})
	s := newTestService(repo, perm)
	p, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{
		Label: "Fix", Type: TypeTicket, Description: "Fixes it.", Instructions: "First.", Enabled: true,
		ShowWhenStage: ticketStage(), ExcludedProjectIDs: []string{"proj-1", "proj-2"},
	})
	require.NoError(t, err)
	return s, repo, p
}

func unchanged(p *Play) UpdateInput {
	return UpdateInput{
		Label: p.Label, Description: p.Description, Instructions: p.Instructions, Enabled: p.Enabled,
		ShowWhenStage: p.ShowWhenStage, ExcludedProjectIDs: p.ExcludedProjectIDs,
	}
}

func fieldsOf(rev Revision) []string {
	var out []string
	for _, c := range rev.Changes {
		out = append(out, c.Field)
	}
	return out
}

func TestUpdate_ASave_WritesAVersionOnlyForInstructionsAndAChangePerOtherChangedField(t *testing.T) {
	review := StageReview
	tests := []struct {
		name        string
		edit        func(*UpdateInput)
		wantVersion bool
		wantFields  []string
	}{
		{"nothing changed", func(*UpdateInput) {}, false, nil},
		{"the same projects in another order", func(in *UpdateInput) { in.ExcludedProjectIDs = []string{"proj-2", "proj-1"} }, false, nil},
		{"instructions only", func(in *UpdateInput) { in.Instructions = "Second." }, true, nil},
		{"label and description", func(in *UpdateInput) { in.Label, in.Description = "Fix it", "Fixes more." }, false, []string{ChangedLabel, ChangedDescription}},
		{"enabled", func(in *UpdateInput) { in.Enabled = false }, false, []string{ChangedEnabled}},
		{"stage", func(in *UpdateInput) { in.ShowWhenStage = &review }, false, []string{ChangedShowWhenStage}},
		{"excluded projects", func(in *UpdateInput) { in.ExcludedProjectIDs = []string{"proj-1"} }, false, []string{ChangedExcluded}},
		{"everything at once", func(in *UpdateInput) {
			in.Label, in.Description, in.Instructions, in.Enabled, in.ShowWhenStage, in.ExcludedProjectIDs = "Fix it", "x", "Second.", false, &review, nil
		}, true, []string{ChangedLabel, ChangedDescription, ChangedEnabled, ChangedShowWhenStage, ChangedExcluded}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo, p := historyFixture(t)
			in := unchanged(p)
			tt.edit(&in)
			_, err := s.Update(ctxAs("owner"), workspaceID, p.ID, in)
			require.NoError(t, err)

			rev := repo.revisions[len(repo.revisions)-1]
			assert.Equal(t, tt.wantVersion, rev.Instructions != nil)
			assert.Equal(t, tt.wantFields, fieldsOf(rev))
		})
	}
}

func TestUpdate_AChangedField_RecordsItsOldAndNewValueAndWhoMadeIt(t *testing.T) {
	s, repo, p := historyFixture(t)
	in := unchanged(p)
	in.Enabled = false
	_, err := s.Update(ctxAs("owner"), workspaceID, p.ID, in)
	require.NoError(t, err)

	got := repo.revisions[len(repo.revisions)-1].Changes
	require.Len(t, got, 1)
	assert.Equal(t, []any{"enabled", "true", "false", Actor{Kind: ActorKindUser, UserID: "owner"}},
		[]any{got[0].Field, got[0].From, got[0].To, got[0].Actor})
	assert.Equal(t, fixedNow, got[0].CreatedAt)
}

func TestUpdate_ChangedInstructions_CreditTheEditorAndHowTheSaveArrived(t *testing.T) {
	automation := identity.WithActor(context.Background(), identity.Actor{ID: "owner", Automation: &identity.AutomationRef{ID: "auto-1", Name: "Nightly"}})
	tests := []struct {
		name       string
		ctx        context.Context
		wantEditor []string
		wantVia    string
		wantActor  Actor
	}{
		{"a person in the browser", ctxAs("owner"), []string{"owner"}, "", Actor{Kind: ActorKindUser, UserID: "owner"}},
		{"a person's agent over MCP", WithViaMCP(ctxAs("owner")), []string{"owner"}, "mcp", Actor{Kind: ActorKindUserMCP, UserID: "owner"}},
		{"an automation", automation, []string{}, "Nightly", Actor{Kind: ActorKindAutomation, AutomationID: "auto-1", AutomationName: "Nightly"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo, p := historyFixture(t)
			in := unchanged(p)
			in.Instructions, in.Label = "Second.", "Fix it"
			_, err := s.Update(tt.ctx, workspaceID, p.ID, in)
			require.NoError(t, err)

			rev := repo.revisions[len(repo.revisions)-1]
			require.NotNil(t, rev.Instructions)
			assert.Equal(t, tt.wantEditor, rev.Instructions.EditorIDs)
			assert.Equal(t, tt.wantVia, rev.Instructions.Via)
			assert.Equal(t, tt.wantActor, rev.Changes[0].Actor)
		})
	}
}

func TestCreate_WritesVersionOneOfTheInstructions(t *testing.T) {
	_, repo, _ := historyFixture(t)
	require.Len(t, repo.revisions, 1)
	v := repo.revisions[0].Instructions
	require.NotNil(t, v)
	assert.Equal(t, []any{"First.", []string{"owner"}, false}, []any{v.Body, v.EditorIDs, v.BuiltinDefault})
	assert.Empty(t, repo.revisions[0].Changes, "creating a play records no field change")
}

func TestSeedDefaults_EachBuiltInPlayStartsAsTheBuiltInDefault(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	require.NoError(t, s.SeedDefaults(ctxAs("owner"), workspaceID))

	require.NotEmpty(t, repo.revisions)
	for _, rev := range repo.revisions {
		require.NotNil(t, rev.Instructions)
		assert.True(t, rev.Instructions.BuiltinDefault)
	}
}

func TestRestore_AKeptVersion_LandsAsANewVersionOnTop(t *testing.T) {
	s, _, p := historyFixture(t)
	in := unchanged(p)
	in.Instructions = "Second."
	_, err := s.Update(ctxAs("owner"), workspaceID, p.ID, in)
	require.NoError(t, err)

	restored, err := s.Restore(WithViaMCP(ctxAs("owner")), p.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "First.", restored.Instructions)

	h, err := s.History(ctxAs("owner"), p.ID)
	require.NoError(t, err)
	require.Len(t, h.Versions, 3, "nothing is lost: the restore is a third version")
	assert.Equal(t, []any{3, true, "mcp"}, []any{h.Versions[0].Version, h.Versions[0].Current, h.Versions[0].Via})
	assert.False(t, h.Versions[1].Current)
	v, err := s.Version(ctxAs("owner"), p.ID, 3)
	require.NoError(t, err)
	assert.Equal(t, []any{"First.", true}, []any{v.Body, v.Current})
	old, err := s.Version(ctxAs("owner"), p.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, "Second.", old.Body, "the version the restore replaced stays readable")
}

func TestRestore_TheCurrentVersion_WritesNothing(t *testing.T) {
	s, repo, p := historyFixture(t)
	_, err := s.Restore(ctxAs("owner"), p.ID, 1)
	require.NoError(t, err)
	assert.Nil(t, repo.revisions[len(repo.revisions)-1].Instructions)
	assert.Len(t, repo.versions[p.ID], 1)
}

func TestHistoryAndRestore_ErrorPaths(t *testing.T) {
	tests := []struct {
		name string
		call func(s *Service, id string) error
		want error
	}{
		{"a reader cannot restore", func(s *Service, id string) error { _, err := s.Restore(ctxAs("reader"), id, 1); return err }, apperrs.ErrForbidden},
		{"someone who cannot see the play gets not found from the history", func(s *Service, id string) error { _, err := s.History(ctxAs("stranger"), id); return err }, apperrs.ErrNotFound},
		{"someone who cannot see the play gets not found from a version", func(s *Service, id string) error { _, err := s.Version(ctxAs("stranger"), id, 1); return err }, apperrs.ErrNotFound},
		{"a version never written is not found", func(s *Service, id string) error { _, err := s.Version(ctxAs("reader"), id, 9); return err }, apperrs.ErrNotFound},
		{"restoring a version never written is not found", func(s *Service, id string) error { _, err := s.Restore(ctxAs("owner"), id, 9); return err }, apperrs.ErrNotFound},
		{"version zero is invalid", func(s *Service, id string) error { _, err := s.Version(ctxAs("reader"), id, 0); return err }, apperrs.ErrInvalid},
		{"an unknown play is not found", func(s *Service, _ string) error { _, err := s.History(ctxAs("owner"), "missing"); return err }, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo, p := historyFixture(t)
			require.ErrorIs(t, tt.call(s, p.ID), tt.want)
			assert.Len(t, repo.revisions, 1, "a refused call writes no history")
		})
	}
}

func historyHandler(t *testing.T) (http.Handler, *Play) {
	t.Helper()
	s, _, p := historyFixture(t)
	return NewHistoryHandler(s).Routes(), p
}

func TestHistoryHandler_ReadsAnswerTheSharedShapeAndRestoreNeedsWrite(t *testing.T) {
	h, p := historyHandler(t)

	rec := do(t, h, http.MethodGet, "/api/plays/"+p.ID+"/history", "", "reader")
	require.Equal(t, http.StatusOK, rec.Code)
	var raw map[string][]map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))
	require.Len(t, raw["versions"], 1)
	assert.ElementsMatch(t, []string{"version", "created_at", "name", "editor_ids", "via", "current", "builtin_default", "words_added", "words_removed", "restored_from"}, keysOf(raw["versions"][0]))
	assert.Empty(t, raw["changes"])

	rec = do(t, h, http.MethodGet, "/api/plays/"+p.ID+"/versions/1", "", "reader")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"body":"First."`)

	rec = do(t, h, http.MethodPost, "/api/plays/"+p.ID+"/versions/1/restore", "", "reader")
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = do(t, h, http.MethodPost, "/api/plays/"+p.ID+"/versions/1/restore", "", "owner")
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = do(t, h, http.MethodGet, "/api/plays/"+p.ID+"/versions/abc", "", "reader")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

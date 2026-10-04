package memories

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func (f *fakeRepo) ListSources(_ context.Context, projectID string) ([]*InterviewSource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sourceErr != nil {
		return nil, f.sourceErr
	}
	out := []*InterviewSource{}
	for _, src := range f.sources {
		if src.ProjectID == projectID {
			cp := *src
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetSource(_ context.Context, id string) (*InterviewSource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, src := range f.sources {
		if src.ID == id {
			cp := *src
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) CountSources(_ context.Context, projectID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sourceErr != nil {
		return 0, f.sourceErr
	}
	n := 0
	for _, src := range f.sources {
		if src.ProjectID == projectID {
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) InsertSource(_ context.Context, src *InterviewSource, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.sources {
		if s.ProjectID == src.ProjectID && s.Kind == src.Kind && s.Ref == src.Ref && s.Kind != SourceText {
			return apperrs.ErrConflict
		}
	}
	cp := *src
	f.sources = append(f.sources, &cp)
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) UpdateSource(_ context.Context, src *InterviewSource, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.sources {
		if s.ID == src.ID {
			s.Stance, s.Label, s.UpdatedAt = src.Stance, src.Label, src.UpdatedAt
			f.events = append(f.events, evts...)
			return nil
		}
	}
	return apperrs.ErrNotFound
}

func (f *fakeRepo) DeleteSource(_ context.Context, id string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, s := range f.sources {
		if s.ID == id {
			f.sources = append(f.sources[:i], f.sources[i+1:]...)
			f.events = append(f.events, evts...)
			return nil
		}
	}
	return apperrs.ErrNotFound
}

func (f *fakeRepo) ListDrafts(_ context.Context, projectID string) ([]*InterviewDraft, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*InterviewDraft{}
	for _, d := range f.drafts {
		if d.ProjectID == projectID {
			cp := *d
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetDraft(_ context.Context, id string) (*InterviewDraft, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, d := range f.drafts {
		if d.ID == id {
			cp := *d
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) SaveDrafts(_ context.Context, drafts []*InterviewDraft, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, d := range drafts {
		cp := *d
		replaced := false
		for i, old := range f.drafts {
			if old.ProjectID == d.ProjectID && old.Question == d.Question {
				f.drafts[i], replaced = &cp, true
			}
		}
		if !replaced {
			f.drafts = append(f.drafts, &cp)
		}
	}
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) DeleteDraft(_ context.Context, id string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, d := range f.drafts {
		if d.ID == id {
			f.drafts = append(f.drafts[:i], f.drafts[i+1:]...)
			f.events = append(f.events, evts...)
			return nil
		}
	}
	return apperrs.ErrNotFound
}

// fakeDocs is a DocLookup stub over a map of doc id to its info.
type fakeDocs map[string]DocInfo

func (f fakeDocs) DocForSource(_ context.Context, docID string) (DocInfo, error) {
	d, ok := f[docID]
	if !ok {
		return DocInfo{}, apperrs.ErrNotFound
	}
	return d, nil
}

var docUpdated = time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)

func newSourcesService(repo *fakeRepo, access fakeAccess) *Service {
	s := newTestServiceWith(repo, access)
	s.SetDocLookup(fakeDocs{
		"doc-1":     {ProjectID: "project-3", WorkspaceID: "workspace-1", Title: "Standards", UpdatedAt: docUpdated},
		"doc-other": {ProjectID: "project-2", WorkspaceID: "workspace-2", Title: "Elsewhere", UpdatedAt: docUpdated},
	})
	return s
}

func follow(projectID, kind, ref string) InterviewSource {
	return InterviewSource{ProjectID: projectID, Kind: kind, Ref: ref, Stance: StanceFollow}
}

func TestAddSource_Refusals(t *testing.T) {
	repo := newFakeRepo()
	repo.memories["mem-3"] = &Memory{ID: "mem-3", WorkspaceID: "workspace-1", ProjectID: "project-3", Title: "Notes"}
	repo.memories["mem-2"] = &Memory{ID: "mem-2", WorkspaceID: "workspace-2", ProjectID: "project-2", Title: "Far"}
	text := InterviewSource{ProjectID: "project-1", Kind: SourceText, Label: "Wiki", Body: "Use tabs.", Stance: StanceFollow}
	with := func(f func(*InterviewSource)) InterviewSource {
		src := text
		f(&src)
		return src
	}
	tests := []struct {
		name    string
		access  fakeAccess
		src     InterviewSource
		wantErr error
	}{
		{"an absolute path", fakeAccess{can: true}, follow("project-1", SourcePath, "/etc/passwd"), apperrs.ErrInvalid},
		{"a windows drive path", fakeAccess{can: true}, follow("project-1", SourcePath, `C:\code\x`), apperrs.ErrInvalid},
		{"a path climbing out", fakeAccess{can: true}, follow("project-1", SourcePath, "docs/../../secrets"), apperrs.ErrInvalid},
		{"a bare ..", fakeAccess{can: true}, follow("project-1", SourcePath, ".."), apperrs.ErrInvalid},
		{"an empty path", fakeAccess{can: true}, follow("project-1", SourcePath, "  "), apperrs.ErrInvalid},
		{"a path with a label", fakeAccess{can: true}, InterviewSource{ProjectID: "project-1", Kind: SourcePath, Ref: "a.md", Label: "x", Stance: StanceFollow}, apperrs.ErrInvalid},
		{"an unknown kind", fakeAccess{can: true}, follow("project-1", "issue", "12"), apperrs.ErrInvalid},
		{"no stance", fakeAccess{can: true}, InterviewSource{ProjectID: "project-1", Kind: SourcePath, Ref: "a.md"}, apperrs.ErrInvalid},
		{"no project", fakeAccess{can: true}, follow("", SourcePath, "a.md"), apperrs.ErrInvalid},
		{"a doc without its id", fakeAccess{can: true}, follow("project-1", SourceDoc, ""), apperrs.ErrInvalid},
		{"text without a label", fakeAccess{can: true}, with(func(s *InterviewSource) { s.Label = " " }), apperrs.ErrInvalid},
		{"text without a body", fakeAccess{can: true}, with(func(s *InterviewSource) { s.Body = "" }), apperrs.ErrInvalid},
		{"text with a ref", fakeAccess{can: true}, with(func(s *InterviewSource) { s.Ref = "x" }), apperrs.ErrInvalid},
		{"text over 32,000 characters", fakeAccess{can: true}, with(func(s *InterviewSource) { s.Body = strings.Repeat("é", MaxSourceTextChars+1) }), apperrs.ErrInvalid},
		{"without memories:write", fakeAccess{can: true, deny: map[string]bool{"project-1:memories:write": true}}, follow("project-1", SourcePath, "a.md"), apperrs.ErrForbidden},
		{"a doc the adder cannot read", fakeAccess{can: true, deny: map[string]bool{"doc-1:docs:read": true}}, follow("project-1", SourceDoc, "doc-1"), apperrs.ErrForbidden},
		{"a doc that is gone", fakeAccess{can: true}, follow("project-1", SourceDoc, "doc-gone"), apperrs.ErrNotFound},
		{"a doc in another workspace", fakeAccess{can: true}, follow("project-1", SourceDoc, "doc-other"), apperrs.ErrInvalid},
		{"a memory the adder cannot read", fakeAccess{can: true, deny: map[string]bool{"project-3:memories:read": true}}, follow("project-1", SourceMemory, "mem-3"), apperrs.ErrForbidden},
		{"a memory that is gone", fakeAccess{can: true}, follow("project-1", SourceMemory, "mem-gone"), apperrs.ErrNotFound},
		{"a memory in another workspace", fakeAccess{can: true}, follow("project-1", SourceMemory, "mem-2"), apperrs.ErrInvalid},
		{"a project the adder cannot read", fakeAccess{can: true, deny: map[string]bool{"project-3:memories:read": true}}, follow("project-1", SourceProject, "project-3"), apperrs.ErrForbidden},
		{"a project hidden from the adder", fakeAccess{can: true, outside: map[string]bool{"project-3": true}}, follow("project-1", SourceProject, "project-3"), apperrs.ErrNotFound},
		{"a project in another workspace", fakeAccess{can: true}, follow("project-1", SourceProject, "project-2"), apperrs.ErrInvalid},
		{"the project itself", fakeAccess{can: true}, follow("project-1", SourceProject, "project-1"), apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newSourcesService(repo, tt.access).AddSource(testCtx(), tt.src)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, repo.sources)
			assert.Empty(t, repo.eventsFor(TopicSourceAdded))
		})
	}
}

func TestAddSource_WithoutAnActor_IsUnauthorized(t *testing.T) {
	_, err := newSourcesService(newFakeRepo(), fakeAccess{can: true}).AddSource(context.Background(), follow("project-1", SourcePath, "a.md"))
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestAddSource_CapsAProjectAt50(t *testing.T) {
	repo := newFakeRepo()
	s := newSourcesService(repo, fakeAccess{can: true})
	for i := range MaxSources {
		_, err := s.AddSource(testCtx(), follow("project-1", SourcePath, "docs/"+strings.Repeat("a", i+1)+".md"))
		require.NoError(t, err)
	}
	_, err := s.AddSource(testCtx(), InterviewSource{ProjectID: "project-1", Kind: SourceText, Label: "one more", Body: "x", Stance: StanceFollow})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Len(t, repo.sources, MaxSources)
}

func TestAddSource_TheSameRefTwice_IsAConflict(t *testing.T) {
	s := newSourcesService(newFakeRepo(), fakeAccess{can: true})
	_, err := s.AddSource(testCtx(), follow("project-1", SourcePath, "./docs/"))
	require.NoError(t, err)
	_, err = s.AddSource(testCtx(), follow("project-1", SourcePath, "docs"))
	require.ErrorIs(t, err, apperrs.ErrConflict, "a cleaned path is the same path")
	text := InterviewSource{ProjectID: "project-1", Kind: SourceText, Label: "Wiki", Body: "a", Stance: StanceFollow}
	_, err = s.AddSource(testCtx(), text)
	require.NoError(t, err)
	_, err = s.AddSource(testCtx(), text)
	require.NoError(t, err, "pasted text may repeat")
}

func TestAddSource_StoresEachKindAndResolvesItsLabel(t *testing.T) {
	repo := newFakeRepo()
	repo.memories["mem-3"] = &Memory{ID: "mem-3", WorkspaceID: "workspace-1", ProjectID: "project-3", Title: "Notes", UpdatedAt: docUpdated}
	s := newSourcesService(repo, fakeAccess{can: true})
	tests := []struct {
		src       InterviewSource
		wantRef   string
		wantLabel string
	}{
		{follow("project-1", SourcePath, `practices\go.md`), "practices/go.md", "practices/go.md"},
		{follow("project-1", SourceDoc, "doc-1"), "doc-1", "Standards"},
		{follow("project-1", SourceMemory, "mem-3"), "mem-3", "Notes"},
		{InterviewSource{ProjectID: "project-1", Kind: SourceProject, Ref: "project-3", Stance: StanceQuestion}, "project-3", "Project project-3"},
		{InterviewSource{ProjectID: "project-1", Kind: SourceText, Label: " Wiki ", Body: " Use tabs. ", Stance: StanceFollow}, "", "Wiki"},
	}
	for _, tt := range tests {
		t.Run(tt.src.Kind, func(t *testing.T) {
			got, err := s.AddSource(testCtx(), tt.src)
			require.NoError(t, err)
			assert.Equal(t, tt.wantRef, got.Ref)
			assert.Equal(t, tt.wantLabel, got.Label)
			assert.Equal(t, "workspace-1", got.WorkspaceID)
			assert.Equal(t, "user-1", got.AddedBy)
			assert.False(t, got.NotVisible || got.Gone)
		})
	}
	assert.Equal(t, "Use tabs.", repo.sources[4].Body)
	evts := repo.eventsFor(TopicSourceAdded)
	require.Len(t, evts, 5)
	assert.Equal(t, SourceEvent{WorkspaceID: "workspace-1", ProjectID: "project-1", SourceID: repo.sources[4].ID, Kind: SourceText, Stance: StanceFollow, AuthorID: "user-1", At: fixedNow}, evts[4].Payload)
}

func TestListSources_ResolvesWithTheReadersAccess(t *testing.T) {
	repo := newFakeRepo()
	repo.memories["mem-3"] = &Memory{ID: "mem-3", WorkspaceID: "workspace-1", ProjectID: "project-3", Title: "Notes", UpdatedAt: docUpdated}
	adder := newSourcesService(repo, fakeAccess{can: true})
	for _, ref := range []InterviewSource{follow("project-1", SourceDoc, "doc-1"), follow("project-1", SourceMemory, "mem-3"), follow("project-1", SourceProject, "project-3")} {
		_, err := adder.AddSource(testCtx(), ref)
		require.NoError(t, err)
	}

	t.Run("a reader who can read every ref sees names", func(t *testing.T) {
		got, err := adder.ListSources(testCtx(), "project-1")
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, []string{"Standards", "Notes", "Project project-3"}, []string{got[0].Label, got[1].Label, got[2].Label})
		assert.Equal(t, &docUpdated, got[0].RefUpdatedAt)
	})
	t.Run("a reader without access sees the kind alone", func(t *testing.T) {
		reader := newSourcesService(repo, fakeAccess{can: true, deny: map[string]bool{"doc-1:docs:read": true, "project-3:memories:read": true}})
		got, err := reader.ListSources(testCtx(), "project-1")
		require.NoError(t, err)
		for _, src := range got {
			assert.True(t, src.NotVisible, src.Kind)
			assert.Empty(t, src.Label, src.Kind)
			assert.Empty(t, src.Ref, src.Kind)
			assert.Nil(t, src.RefUpdatedAt, src.Kind)
		}
	})
	t.Run("a deleted ref reads as gone", func(t *testing.T) {
		gone := newTestServiceWith(repo, fakeAccess{can: true})
		gone.SetDocLookup(fakeDocs{})
		delete(repo.memories, "mem-3")
		gone.projects = fakeProjects{workspaces: map[string]string{"project-1": "workspace-1"}}
		got, err := gone.ListSources(testCtx(), "project-1")
		require.NoError(t, err)
		for _, src := range got {
			assert.True(t, src.Gone, src.Kind)
			assert.False(t, src.NotVisible, src.Kind)
		}
	})
	t.Run("without memories:read", func(t *testing.T) {
		_, err := newSourcesService(repo, fakeAccess{can: false}).ListSources(testCtx(), "project-1")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("a lookup failure is an error", func(t *testing.T) {
		broken := newSourcesService(repo, fakeAccess{can: true})
		broken.projects = fakeProjects{err: apperrs.ErrConflict}
		_, err := broken.ListSources(testCtx(), "project-1")
		require.Error(t, err)
	})
}

func TestUpdateSource(t *testing.T) {
	repo := newFakeRepo()
	s := newSourcesService(repo, fakeAccess{can: true})
	path, err := s.AddSource(testCtx(), follow("project-1", SourcePath, "legacy/"))
	require.NoError(t, err)
	text, err := s.AddSource(testCtx(), InterviewSource{ProjectID: "project-1", Kind: SourceText, Label: "Wiki", Body: "b", Stance: StanceFollow})
	require.NoError(t, err)
	question, label, bad := StanceQuestion, "Team wiki", "maybe"

	tests := []struct {
		name    string
		svc     *Service
		id      string
		stance  *string
		label   *string
		wantErr error
	}{
		{"nothing to change", s, path.ID, nil, nil, apperrs.ErrInvalid},
		{"an unknown stance", s, path.ID, &bad, nil, apperrs.ErrInvalid},
		{"a label on a path", s, path.ID, nil, &label, apperrs.ErrInvalid},
		{"no id", s, " ", &question, nil, apperrs.ErrInvalid},
		{"a missing source", s, "nope", &question, nil, apperrs.ErrNotFound},
		{"without memories:write", newSourcesService(repo, fakeAccess{can: true, deny: map[string]bool{"project-1:memories:write": true}}), path.ID, &question, nil, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.svc.UpdateSource(testCtx(), tt.id, tt.stance, tt.label)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
	assert.Empty(t, repo.eventsFor(TopicSourceChanged))

	got, err := s.UpdateSource(testCtx(), path.ID, &question, nil)
	require.NoError(t, err)
	assert.Equal(t, StanceQuestion, got.Stance)
	got, err = s.UpdateSource(testCtx(), text.ID, nil, &label)
	require.NoError(t, err)
	assert.Equal(t, "Team wiki", got.Label)
	assert.Equal(t, StanceFollow, got.Stance, "an omitted stance keeps its value")
	assert.Len(t, repo.eventsFor(TopicSourceChanged), 2)
}

func TestRemoveSource(t *testing.T) {
	repo := newFakeRepo()
	s := newSourcesService(repo, fakeAccess{can: true})
	src, err := s.AddSource(testCtx(), follow("project-1", SourcePath, "a.md"))
	require.NoError(t, err)

	require.ErrorIs(t, s.RemoveSource(testCtx(), ""), apperrs.ErrInvalid)
	require.ErrorIs(t, s.RemoveSource(testCtx(), "nope"), apperrs.ErrNotFound)
	deny := newSourcesService(repo, fakeAccess{can: true, deny: map[string]bool{"project-1:memories:write": true}})
	require.ErrorIs(t, deny.RemoveSource(testCtx(), src.ID), apperrs.ErrForbidden)

	require.NoError(t, s.RemoveSource(testCtx(), src.ID))
	assert.Empty(t, repo.sources)
	evts := repo.eventsFor(TopicSourceRemoved)
	require.Len(t, evts, 1)
	assert.Equal(t, src.ID, evts[0].Payload.(SourceEvent).SourceID)
}

func draftsFixture(t *testing.T) (*fakeRepo, *Service, string, string) {
	t.Helper()
	repo := newFakeRepo()
	s := newSourcesService(repo, fakeAccess{can: true})
	f, err := s.AddSource(testCtx(), follow("project-1", SourcePath, "practices/testing.md"))
	require.NoError(t, err)
	q, err := s.AddSource(testCtx(), InterviewSource{ProjectID: "project-1", Kind: SourcePath, Ref: "legacy/", Stance: StanceQuestion})
	require.NoError(t, err)
	_, err = s.SaveAnswer(testCtx(), InterviewAnswer{ProjectID: "project-1", Question: "Tests", Selected: []string{"Unit", "Integration"}, Text: "Real SQLite"})
	require.NoError(t, err)
	_, err = s.SaveAnswer(testCtx(), InterviewAnswer{ProjectID: "project-1", Question: "Style", Skipped: true})
	require.NoError(t, err)
	return repo, s, f.ID, q.ID
}

func TestSaveDrafts_Refusals(t *testing.T) {
	repo, s, followID, questionID := draftsFixture(t)
	ok := InterviewDraft{Question: "Stack", Text: "Go", SourceIDs: []string{followID}}
	with := func(f func(*InterviewDraft)) []InterviewDraft {
		d := ok
		f(&d)
		return []InterviewDraft{d}
	}
	tests := []struct {
		name    string
		svc     *Service
		project string
		drafts  []InterviewDraft
		wantErr error
	}{
		{"no drafts", s, "project-1", nil, apperrs.ErrInvalid},
		{"no project", s, "", []InterviewDraft{ok}, apperrs.ErrInvalid},
		{"no question", s, "project-1", with(func(d *InterviewDraft) { d.Question = "" }), apperrs.ErrInvalid},
		{"no picks and no text", s, "project-1", with(func(d *InterviewDraft) { d.Text = " " }), apperrs.ErrInvalid},
		{"no source", s, "project-1", with(func(d *InterviewDraft) { d.SourceIDs = nil }), apperrs.ErrInvalid},
		{"an unknown source", s, "project-1", with(func(d *InterviewDraft) { d.SourceIDs = []string{"nope"} }), apperrs.ErrInvalid},
		{"a question source", s, "project-1", with(func(d *InterviewDraft) { d.SourceIDs = []string{questionID} }), apperrs.ErrInvalid},
		{"a where line over 500", s, "project-1", with(func(d *InterviewDraft) { d.Where = strings.Repeat("w", MaxDraftWhereChars+1) }), apperrs.ErrInvalid},
		{"one question twice", s, "project-1", []InterviewDraft{ok, ok}, apperrs.ErrInvalid},
		{"one bad draft among good ones", s, "project-1", append([]InterviewDraft{ok}, with(func(d *InterviewDraft) { d.Question, d.SourceIDs = "Other", nil })...), apperrs.ErrInvalid},
		{"without memories:write", newSourcesService(repo, fakeAccess{can: true, deny: map[string]bool{"project-1:memories:write": true}}), "project-1", []InterviewDraft{ok}, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.svc.SaveDrafts(testCtx(), tt.project, tt.drafts)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, repo.drafts, "a refused batch saves none")
		})
	}
}

func TestSaveDrafts_DropsADraftEqualToTheStoredAnswer(t *testing.T) {
	repo, s, followID, _ := draftsFixture(t)
	saved, err := s.SaveDrafts(testCtx(), "project-1", []InterviewDraft{
		{Question: "Tests", Selected: []string{"Integration", "Unit"}, Text: " Real SQLite ", SourceIDs: []string{followID}},
		{Question: "Style", Selected: []string{"Early return"}, SourceIDs: []string{followID}, Where: "practices/go.md, section 9"},
		{Question: "Stack", Text: "Go and React", SourceIDs: []string{followID}, TrailID: "trail-1"},
	})
	require.NoError(t, err)
	require.Len(t, saved, 2, "the equal draft is dropped, picks compared in any order")
	assert.Equal(t, "Style", saved[0].Question, "a skipped question is drafted like an unanswered one")
	assert.Equal(t, "Stack", saved[1].Question)
	assert.Equal(t, "trail-1", saved[1].TrailID)
	assert.Equal(t, "user-1", saved[1].DraftedBy)
	assert.Len(t, repo.eventsFor(TopicDraftSaved), 2)

	saved, err = s.SaveDrafts(testCtx(), "project-1", []InterviewDraft{{Question: "Tests", Selected: []string{"Unit"}, Text: "Real SQLite", SourceIDs: []string{followID}}})
	require.NoError(t, err)
	require.Len(t, saved, 1, "a draft that disagrees with the answer is a suggested change")

	all, err := s.SaveDrafts(testCtx(), "project-1", []InterviewDraft{{Question: "Tests", Selected: []string{"Unit", "Integration"}, Text: "Real SQLite", SourceIDs: []string{followID}}})
	require.NoError(t, err)
	assert.Empty(t, all, "every draft equal to its answer stores nothing")

	_, err = s.SaveDrafts(testCtx(), "project-1", []InterviewDraft{{Question: "Stack", Text: "Go", SourceIDs: []string{followID}}})
	require.NoError(t, err)
	ds, err := s.ListDrafts(testCtx(), "project-1")
	require.NoError(t, err)
	require.Len(t, ds, 3, "a later draft replaces the question's earlier one")
}

func TestSaveDrafts_ListFailures(t *testing.T) {
	repo, s, followID, _ := draftsFixture(t)
	repo.answerErr = apperrs.ErrConflict
	_, err := s.SaveDrafts(testCtx(), "project-1", []InterviewDraft{{Question: "Stack", Text: "Go", SourceIDs: []string{followID}}})
	require.Error(t, err)
	repo.answerErr, repo.sourceErr = nil, apperrs.ErrConflict
	_, err = s.SaveDrafts(testCtx(), "project-1", []InterviewDraft{{Question: "Stack", Text: "Go", SourceIDs: []string{followID}}})
	require.Error(t, err)
}

func TestDismissDraft(t *testing.T) {
	repo, s, followID, _ := draftsFixture(t)
	saved, err := s.SaveDrafts(testCtx(), "project-1", []InterviewDraft{{Question: "Stack", Text: "Go", SourceIDs: []string{followID}}})
	require.NoError(t, err)
	id := saved[0].ID

	require.ErrorIs(t, s.DismissDraft(testCtx(), " "), apperrs.ErrInvalid)
	require.ErrorIs(t, s.DismissDraft(testCtx(), "nope"), apperrs.ErrNotFound)
	deny := newSourcesService(repo, fakeAccess{can: true, deny: map[string]bool{"project-1:memories:write": true}})
	require.ErrorIs(t, deny.DismissDraft(testCtx(), id), apperrs.ErrForbidden)

	require.NoError(t, s.DismissDraft(testCtx(), id))
	assert.Empty(t, repo.drafts)
	evts := repo.eventsFor(TopicDraftDismissed)
	require.Len(t, evts, 1)
	assert.Equal(t, DraftEvent{WorkspaceID: "workspace-1", ProjectID: "project-1", DraftID: id, Question: "Stack", AuthorID: "user-1", At: fixedNow}, evts[0].Payload)
}

func TestClearSuggestions_DeletesOnlyDraftsOnAnsweredQuestions(t *testing.T) {
	repo, s, followID, _ := draftsFixture(t)
	_, err := s.SaveDrafts(testCtx(), "project-1", []InterviewDraft{
		{Question: "Tests", Selected: []string{"Unit"}, SourceIDs: []string{followID}},
		{Question: "Style", Text: "Early return", SourceIDs: []string{followID}},
		{Question: "Stack", Text: "Go", SourceIDs: []string{followID}},
	})
	require.NoError(t, err)

	require.NoError(t, s.ClearSuggestions(testCtx(), "project-1"))

	ds, err := s.ListDrafts(testCtx(), "project-1")
	require.NoError(t, err)
	require.Len(t, ds, 2, "drafts on a skipped and an unanswered question stay for the run to replace")
	assert.Equal(t, []string{"Style", "Stack"}, []string{ds[0].Question, ds[1].Question})
	evts := repo.eventsFor(TopicDraftDismissed)
	require.Len(t, evts, 1, "the page hears the suggested change go")
	assert.Equal(t, "Tests", evts[0].Payload.(DraftEvent).Question)
}

func TestClearSuggestions_Refusals(t *testing.T) {
	repo, s, _, _ := draftsFixture(t)
	require.ErrorIs(t, s.ClearSuggestions(testCtx(), " "), apperrs.ErrInvalid)
	require.ErrorIs(t, s.ClearSuggestions(context.Background(), "project-1"), apperrs.ErrUnauthorized)
	deny := newSourcesService(repo, fakeAccess{can: true, deny: map[string]bool{"project-1:memories:write": true}})
	require.ErrorIs(t, deny.ClearSuggestions(testCtx(), "project-1"), apperrs.ErrForbidden)
	repo.answerErr = apperrs.ErrConflict
	require.Error(t, s.ClearSuggestions(testCtx(), "project-1"))
}

func TestListDrafts_RequiresMemoriesRead(t *testing.T) {
	_, err := newSourcesService(newFakeRepo(), fakeAccess{can: false}).ListDrafts(testCtx(), "project-1")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = newSourcesService(newFakeRepo(), fakeAccess{can: true}).ListDrafts(testCtx(), "")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestMCP_InterviewSourcesAndDrafts(t *testing.T) {
	repo := newFakeRepo()
	s := newSourcesService(repo, fakeAccess{can: true})
	interview, err := s.CreateInterview(testCtx(), "project-1", "")
	require.NoError(t, err)
	plain := mustMemory(t, s, "project-1", "Tone", "", "", false)
	id := `"id":"` + interview.ID + `"`

	got := mustCall(t, s, "memory_update", `{`+id+`,"add_sources":[
		{"kind":"path","ref":"practices/testing.md","stance":"follow"},
		{"kind":"text","label":"Wiki","body":"Squash merge.","stance":"follow"},
		{"kind":"project","ref":"project-3","stance":"question"}]}`).(memoryResult)
	require.Len(t, got.Sources, 3)
	assert.Equal(t, "Squash merge.", got.Sources[1].Body, "memory_get carries pasted text bodies")
	assert.Equal(t, 1, got.Version, "sources add no version")
	pathID, textID, projectID := got.Sources[0].ID, got.Sources[1].ID, got.Sources[2].ID

	got = mustCall(t, s, "memory_update", `{`+id+`,"update_sources":[{"id":"`+projectID+`","stance":"follow"}],"remove_sources":["`+textID+`"],
		"drafts":[{"question":"Stack","text":"Go","source_ids":["`+pathID+`"],"where":"go.mod"}]}`).(memoryResult)
	require.Len(t, got.Sources, 2)
	assert.Equal(t, StanceFollow, got.Sources[1].Stance)
	require.Len(t, got.Drafts, 1)
	assert.Equal(t, "go.mod", got.Drafts[0].Where)

	got = mustCall(t, s, "memory_get", `{`+id+`}`).(memoryResult)
	assert.Len(t, got.Sources, 2)
	assert.Len(t, got.Drafts, 1)

	got = mustCall(t, s, "memory_update", `{`+id+`,"dismiss_drafts":["`+got.Drafts[0].ID+`"]}`).(memoryResult)
	assert.Empty(t, got.Drafts)

	tests := []struct {
		name string
		args string
	}{
		{"sources on an ordinary memory", `{"id":"` + plain.ID + `","add_sources":[{"kind":"path","ref":"a.md","stance":"follow"}]}`},
		{"drafts on an ordinary memory", `{"id":"` + plain.ID + `","drafts":[{"question":"Stack","text":"Go","source_ids":["` + pathID + `"]}]}`},
		{"revert mixed with sources", `{` + id + `,"revert_to_version":1,"remove_sources":["` + pathID + `"]}`},
		{"a refused source", `{` + id + `,"add_sources":[{"kind":"path","ref":"/etc","stance":"follow"}]}`},
		{"a refused change", `{` + id + `,"update_sources":[{"id":"` + pathID + `","stance":"maybe"}]}`},
		{"a refused removal", `{` + id + `,"remove_sources":["nope"]}`},
		{"a refused draft", `{` + id + `,"drafts":[{"question":"Stack","text":"Go","source_ids":[]}]}`},
		{"a refused dismissal", `{` + id + `,"dismiss_drafts":["nope"]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(testCtx(), t, s, "memory_update", tt.args)
			require.Error(t, err)
		})
	}
}

func TestHandler_InterviewSourcesAndDrafts(t *testing.T) {
	repo := newFakeRepo()
	h := NewHandler(newSourcesService(repo, fakeAccess{can: true})).Routes()

	rec := serve(t, h, http.MethodPost, "/api/memories/interview-sources", `{"project_id":"project-1","kind":"doc","ref":"doc-1","stance":"follow"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var src InterviewSource
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &src))
	assert.Equal(t, "Standards", src.Label)

	assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPost, "/api/memories/interview-sources", `{"project_id":"project-1","kind":"path","ref":"../x","stance":"follow"}`).Code)
	assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPost, "/api/memories/interview-sources", `{`).Code)

	rec = serve(t, h, http.MethodPatch, "/api/memories/interview-sources/"+src.ID, `{"stance":"question"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPatch, "/api/memories/interview-sources/"+src.ID, `{}`).Code)
	assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPatch, "/api/memories/interview-sources/"+src.ID, `{`).Code)

	rec = serve(t, h, http.MethodGet, "/api/memories/interview-sources?project_id=project-1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var srcs []InterviewSource
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &srcs))
	require.Len(t, srcs, 1)
	assert.Equal(t, StanceQuestion, srcs[0].Stance)
	assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodGet, "/api/memories/interview-sources", "").Code)

	repo.drafts = append(repo.drafts, &InterviewDraft{ID: "draft-1", WorkspaceID: "workspace-1", ProjectID: "project-1", Question: "Stack", Text: "Go"})
	rec = serve(t, h, http.MethodGet, "/api/memories/interview-drafts?project_id=project-1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"draft-1"`)
	assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodGet, "/api/memories/interview-drafts", "").Code)
	assert.Equal(t, http.StatusNoContent, serve(t, h, http.MethodDelete, "/api/memories/interview-drafts/draft-1", "").Code)
	assert.Equal(t, http.StatusNotFound, serve(t, h, http.MethodDelete, "/api/memories/interview-drafts/draft-1", "").Code)

	assert.Equal(t, http.StatusNoContent, serve(t, h, http.MethodDelete, "/api/memories/interview-sources/"+src.ID, "").Code)
	assert.Equal(t, http.StatusNotFound, serve(t, h, http.MethodDelete, "/api/memories/interview-sources/"+src.ID, "").Code)
}

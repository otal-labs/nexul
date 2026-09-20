package docs

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var fixedNow = time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)

// fakeRepo is an in-memory docs.Repo for use-case tests.
type fakeRepo struct {
	mu        sync.Mutex
	docs      map[string]*Doc
	versions  map[string][]*DocVersion
	events    []eventbus.OutboxEvent
	createErr error
	getErr    error
	listErr   error
	updateErr error
	commitErr error
	deleteErr error
	searchErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{docs: map[string]*Doc{}, versions: map[string][]*DocVersion{}}
}

func (f *fakeRepo) Create(_ context.Context, d *Doc, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if _, ok := f.docs[d.ID]; ok {
		return apperrs.ErrConflict
	}
	f.docs[d.ID] = d
	f.appendVersion(d)
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id string) (*Doc, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	d, ok := f.docs[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return d, nil
}

func (f *fakeRepo) List(_ context.Context) ([]*Doc, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*Doc, 0, len(f.docs))
	for _, d := range f.docs {
		out = append(out, d)
	}
	return out, nil
}

func (f *fakeRepo) ListByProject(_ context.Context, projectID string) ([]*Doc, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*Doc, 0, len(f.docs))
	for _, d := range f.docs {
		if d.ProjectID == projectID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, d *Doc, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	if _, ok := f.docs[d.ID]; !ok {
		return apperrs.ErrNotFound
	}
	f.docs[d.ID] = d
	f.appendVersion(d)
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, id string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.docs[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.docs, id)
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) SetArchived(_ context.Context, id string, archived bool, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.docs[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	d.Archived = archived
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) Search(_ context.Context, query string, limit int) ([]SearchResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	var out []SearchResult
	for _, d := range f.docs {
		if strings.Contains(strings.ToLower(d.Title), strings.ToLower(query)) || strings.Contains(strings.ToLower(d.Body), strings.ToLower(query)) {
			out = append(out, SearchResult{ID: d.ID, Title: d.Title})
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeRepo) ListVersions(_ context.Context, docID string) ([]*DocVersion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]*DocVersion(nil), f.versions[docID]...)
	slices.SortFunc(out, func(a, b *DocVersion) int { return b.Version - a.Version })
	return out, nil
}

func (f *fakeRepo) GetVersion(_ context.Context, docID string, version int) (*DocVersion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, v := range f.versions[docID] {
		if v.Version == version {
			return v, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) appendVersion(d *Doc) {
	f.versions[d.ID] = append(f.versions[d.ID], &DocVersion{
		DocID: d.ID, Version: d.Version, Title: d.Title, Body: d.Body, CreatedAt: d.UpdatedAt,
	})
}

func (f *fakeRepo) CommitBody(_ context.Context, d *Doc, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.commitErr != nil {
		return f.commitErr
	}
	if _, ok := f.docs[d.ID]; !ok {
		return apperrs.ErrNotFound
	}
	f.docs[d.ID] = d
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) CreateNamedVersion(_ context.Context, v *DocVersion) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.docs[v.DocID]; !ok {
		return apperrs.ErrNotFound
	}
	f.versions[v.DocID] = append(f.versions[v.DocID], v)
	f.docs[v.DocID].Version = v.Version
	return nil
}

func (f *fakeRepo) eventsFor(topic string) []eventbus.OutboxEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []eventbus.OutboxEvent
	for _, e := range f.events {
		if e.Topic == topic {
			out = append(out, e)
		}
	}
	return out
}

func newTestService(repo *fakeRepo) *Service {
	s := NewService(repo, fakeAccess{can: true})
	s.now = func() time.Time { return fixedNow }
	return s
}

// newDenyService wires a service whose access checker denies every action.
func newDenyService(repo *fakeRepo) *Service {
	s := NewService(repo, fakeAccess{can: false})
	s.now = func() time.Time { return fixedNow }
	return s
}

// newScriptedService wires a checker whose Can result is driven by the test.
func newScriptedService(repo *fakeRepo, can bool, grantErr error) *Service {
	s := NewService(repo, fakeAccess{can: can, grant: grantErr})
	s.now = func() time.Time { return fixedNow }
	return s
}

// testCtx returns a context carrying a fixed actor for permission checks.
func testCtx() context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: "user-1", CanCreateWorkspace: false})
}

// fakeAccess is a docs.AccessChecker stub for unit tests.
type fakeAccess struct {
	can       bool
	canErr    error
	grant     error
	deleteErr error
}

func (f fakeAccess) Can(_ context.Context, _, _ string, _ permissions.Action) (bool, error) {
	return f.can, f.canErr
}

func (f fakeAccess) GrantCreator(_ context.Context, _, _ string) error {
	return f.grant
}

func (f fakeAccess) DeleteByDoc(_ context.Context, _ string) error {
	return f.deleteErr
}

func TestCreate(t *testing.T) {
	t.Run("empty title is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Create(testCtx(), "project-1", "  ", "body")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty project id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Create(testCtx(), "  ", "title", "body")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.createErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.Create(testCtx(), "project-1", "title", "body")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.createErr)
	})
	t.Run("creates version 1 and enqueues doc.created", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		d, err := s.Create(testCtx(), "project-1", "Storage Spine", "SQLite + FTS5")
		require.NoError(t, err)
		assert.NotEmpty(t, d.ID)
		assert.Equal(t, "project-1", d.ProjectID)
		assert.Equal(t, 1, d.Version)
		assert.Equal(t, fixedNow, d.CreatedAt)
		assert.Equal(t, fixedNow, d.UpdatedAt)
		assert.NotEmpty(t, repo.eventsFor(TopicCreated))
	})
}

func TestGet(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Get(testCtx(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing doc is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Get(testCtx(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.getErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.Get(testCtx(), "doc-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.getErr)
	})
	t.Run("returns stored doc", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(testCtx(), "project-1", "Storage Spine", "body")
		require.NoError(t, err)
		got, err := s.Get(testCtx(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Update(testCtx(), "", "title", "body")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty title is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Update(testCtx(), "doc-1", " ", "body")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing doc is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Update(testCtx(), "nope", "title", "body")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(testCtx(), "project-1", "title", "body")
		require.NoError(t, err)
		repo.updateErr = errors.New("disk full")
		_, err = s.Update(testCtx(), created.ID, "new title", "new body")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.updateErr)
	})
	t.Run("bumps version, records history, enqueues doc.updated", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(testCtx(), "project-1", "title", "body")
		require.NoError(t, err)

		updated, err := s.Update(testCtx(), created.ID, "Renamed", "new body")
		require.NoError(t, err)
		assert.Equal(t, 2, updated.Version)
		assert.Equal(t, "Renamed", updated.Title)
		assert.Equal(t, fixedNow, updated.UpdatedAt)

		vs, err := repo.ListVersions(context.Background(), created.ID)
		require.NoError(t, err)
		require.Len(t, vs, 2)
		assert.Equal(t, 2, vs[0].Version)
		assert.Equal(t, 1, vs[1].Version)
		assert.NotEmpty(t, repo.eventsFor(TopicUpdated))
	})
}

func TestDelete(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.Delete(testCtx(), "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing doc is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.Delete(testCtx(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(testCtx(), "project-1", "title", "body")
		require.NoError(t, err)
		repo.deleteErr = errors.New("db down")
		err = s.Delete(testCtx(), created.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.deleteErr)
	})
	t.Run("removes doc", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(testCtx(), "project-1", "title", "body")
		require.NoError(t, err)
		require.NoError(t, s.Delete(testCtx(), created.ID))
		_, err = s.Get(testCtx(), created.ID)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("publishes doc.deleted", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(testCtx(), "project-1", "title", "body")
		require.NoError(t, err)
		require.NoError(t, s.Delete(testCtx(), created.ID))
		evts := repo.eventsFor(TopicDeleted)
		require.Len(t, evts, 1)
		payload, ok := evts[0].Payload.(DeletedEvent)
		require.True(t, ok)
		assert.Equal(t, created.ID, payload.ID)
		assert.Equal(t, "title", payload.Title)
	})
	t.Run("access cleanup error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		s := NewService(repo, fakeAccess{can: true, deleteErr: errors.New("db down")})
		s.now = func() time.Time { return fixedNow }
		created, err := s.Create(testCtx(), "project-1", "title", "body")
		require.NoError(t, err)
		err = s.Delete(testCtx(), created.ID)
		require.Error(t, err)
	})
}

func TestSearch(t *testing.T) {
	t.Run("empty query is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Search(testCtx(), "  ", 10)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.searchErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.Search(testCtx(), "sqlite", 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.searchErr)
	})
	t.Run("defaults limit when not given", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		_, err := s.Create(testCtx(), "project-1", "migration runner", "body")
		require.NoError(t, err)
		results, err := s.Search(testCtx(), "runner", 0)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "migration runner", results[0].Title)
	})
}

func TestListVersionsAndGetVersion(t *testing.T) {
	t.Run("empty doc id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.ListVersions(testCtx(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("version below 1 is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.GetVersion(testCtx(), "doc-1", 0)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing version is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.GetVersion(testCtx(), "doc-1", 9)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("lists versions newest first and fetches one", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(testCtx(), "project-1", "v1", "one")
		require.NoError(t, err)
		_, err = s.Update(testCtx(), created.ID, "v2", "two")
		require.NoError(t, err)

		vs, err := s.ListVersions(testCtx(), created.ID)
		require.NoError(t, err)
		require.Len(t, vs, 2)
		assert.Equal(t, 2, vs[0].Version, "newest first")

		v, err := s.GetVersion(testCtx(), created.ID, 1)
		require.NoError(t, err)
		assert.Equal(t, "v1", v.Title)
	})
}

func TestCreate_RequiresActor(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Create(context.Background(), "project-1", "title", "body")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
}

func TestCreate_GrantErrorFails(t *testing.T) {
	repo := newFakeRepo()
	s := newScriptedService(repo, true, errors.New("grant failed"))
	_, err := s.Create(testCtx(), "project-1", "title", "body")
	require.Error(t, err)
}

func TestEnforcement_Deny(t *testing.T) {
	t.Run("get denied without read", func(t *testing.T) {
		repo := newFakeRepo()
		created, err := newTestService(repo).Create(testCtx(), "project-1", "t", "b")
		require.NoError(t, err)
		s := newDenyService(repo)
		_, err = s.Get(testCtx(), created.ID)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("update denied without edit", func(t *testing.T) {
		repo := newFakeRepo()
		created, err := newTestService(repo).Create(testCtx(), "project-1", "t", "b")
		require.NoError(t, err)
		s := newDenyService(repo)
		_, err = s.Update(testCtx(), created.ID, "new", "body")
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("delete denied without delete", func(t *testing.T) {
		repo := newFakeRepo()
		created, err := newTestService(repo).Create(testCtx(), "project-1", "t", "b")
		require.NoError(t, err)
		s := newDenyService(repo)
		err = s.Delete(testCtx(), created.ID)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("archive denied without archive bit", func(t *testing.T) {
		repo := newFakeRepo()
		created, err := newTestService(repo).Create(testCtx(), "project-1", "t", "b")
		require.NoError(t, err)
		s := newDenyService(repo)
		_, err = s.Archive(testCtx(), created.ID)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("versions denied without read", func(t *testing.T) {
		repo := newFakeRepo()
		created, err := newTestService(repo).Create(testCtx(), "project-1", "t", "b")
		require.NoError(t, err)
		s := newDenyService(repo)
		_, err = s.ListVersions(testCtx(), created.ID)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

func TestArchiveRestore(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "t", "b")
	require.NoError(t, err)

	archived, err := s.Archive(testCtx(), created.ID)
	require.NoError(t, err)
	assert.True(t, archived.Archived)
	assert.NotEmpty(t, repo.eventsFor(TopicUpdated))

	restored, err := s.Restore(testCtx(), created.ID)
	require.NoError(t, err)
	assert.False(t, restored.Archived)
}

func TestList_Disclosure(t *testing.T) {
	t.Run("openable docs carry can_open", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		_, err := s.Create(testCtx(), "project-1", "t", "b")
		require.NoError(t, err)
		items, err := s.List(testCtx())
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.True(t, items[0].CanOpen)
	})
	t.Run("unopenable docs disclose title but not body", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(testCtx(), "project-1", "secret title", "secret body")
		require.NoError(t, err)
		deny := newDenyService(repo)
		items, err := deny.List(testCtx())
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.False(t, items[0].CanOpen)
		assert.Equal(t, "secret title", items[0].Title)
		assert.Equal(t, created.ID, items[0].ID)
	})
}

func TestListByProject(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.ListByProject(testCtx(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.listErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.ListByProject(testCtx(), "project-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.listErr)
	})
	t.Run("returns only docs in the given project", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		inProject, err := s.Create(testCtx(), "project-1", "In project", "b")
		require.NoError(t, err)
		_, err = s.Create(testCtx(), "project-2", "Other project", "b")
		require.NoError(t, err)

		items, err := s.ListByProject(testCtx(), "project-1")
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, inProject.ID, items[0].ID)
		assert.Equal(t, "project-1", items[0].ProjectID)
	})
}

func TestSearch_FiltersUnreadable(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	openable, err := s.Create(testCtx(), "project-1", "Storage Spine", "SQLite")
	require.NoError(t, err)
	hidden, err := s.Create(testCtx(), "project-1", "Top Secret", "SQLite")
	require.NoError(t, err)
	_ = openable
	_ = hidden

	deny := newDenyService(repo)
	results, err := deny.Search(testCtx(), "sqlite", 10)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestCreate_NormalizesBodyToCanonicalRichText(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)

	t.Run("markdown body converts to structured JSON", func(t *testing.T) {
		d, err := s.Create(testCtx(), "project-1", "Spec", "# Title\n\nBody **bold**")
		require.NoError(t, err)
		require.True(t, richtext.IsStructured(d.Body), "stored body must be canonical rich text")
		md, err := richtext.JSONToMarkdown(d.Body)
		require.NoError(t, err)
		assert.Equal(t, "# Title\n\nBody **bold**", md)
	})
	t.Run("structured body passes through", func(t *testing.T) {
		body := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hi"}]}]}`
		d, err := s.Create(testCtx(), "project-1", "Spec", body)
		require.NoError(t, err)
		assert.Equal(t, body, d.Body)
	})
}

func TestUpdate_NormalizesBodyToCanonicalRichText(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "Spec", "plain body")
	require.NoError(t, err)

	updated, err := s.Update(testCtx(), created.ID, "Spec", "# Heading")
	require.NoError(t, err)
	require.True(t, richtext.IsStructured(updated.Body), "updated body must be canonical rich text")
}

func TestExportMarkdown(t *testing.T) {
	t.Run("exports structured doc as markdown", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		d, err := s.Create(testCtx(), "project-1", "Spec", "**bold** text")
		require.NoError(t, err)
		md, err := s.ExportMarkdown(testCtx(), d.ID)
		require.NoError(t, err)
		assert.Equal(t, "**bold** text", md)
	})
	t.Run("legacy markdown body passes through", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		// Simulate a pre-ws-23 row by writing markdown straight into the repo.
		legacy := &Doc{ID: "legacy-1", Title: "Old", Body: "plain markdown", Version: 1}
		require.NoError(t, repo.Create(testCtx(), legacy))
		md, err := s.ExportMarkdown(testCtx(), "legacy-1")
		require.NoError(t, err)
		assert.Equal(t, "plain markdown", md)
	})
	t.Run("missing doc is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.ExportMarkdown(testCtx(), "nope")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestImportMarkdown(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)

	d, err := s.ImportMarkdown(testCtx(), "project-1", "Imported", "# From markdown")
	require.NoError(t, err)
	require.True(t, richtext.IsStructured(d.Body))
	assert.Equal(t, 1, d.Version)
	assert.NotEmpty(t, repo.eventsFor(TopicCreated))
}

func TestCommitCollab(t *testing.T) {
	newDoc := func(t *testing.T, s *Service, repo *fakeRepo) *Doc {
		t.Helper()
		d, err := s.Create(testCtx(), "project-1", "Spec", "**bold**")
		require.NoError(t, err)
		return d
	}
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.CommitCollab(testCtx(), " ", "T", `{"type":"doc"}`)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("empty title keeps the current title — commits carry it only from the client that renamed", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(testCtx(), "project-1", "Original title", `{"type":"doc"}`)
		require.NoError(t, err)
		require.NoError(t, s.CommitCollab(testCtx(), created.ID, "  ", `{"type":"doc","content":[]}`))
		got, err := s.Get(testCtx(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Original title", got.Title)
	})
	t.Run("missing doc is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.CommitCollab(testCtx(), "nope", "T", `{"type":"doc"}`)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("repo failure propagates", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		d := newDoc(t, s, repo)
		repo.commitErr = errors.New("db down")
		err := s.CommitCollab(testCtx(), d.ID, "T", `{"type":"doc"}`)
		require.ErrorIs(t, err, repo.commitErr)
	})
	t.Run("writes converged body without a version row and fires doc.updated", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		d := newDoc(t, s, repo)
		before := len(repo.versions[d.ID])

		body := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"merged"}]}]}`
		require.NoError(t, s.CommitCollab(testCtx(), d.ID, "Spec v2", body))

		got, err := s.Get(testCtx(), d.ID)
		require.NoError(t, err)
		assert.Equal(t, "Spec v2", got.Title)
		assert.Equal(t, body, got.Body)
		assert.Equal(t, 2, got.Version, "revision counter advances")
		assert.Len(t, repo.versions[d.ID], before, "no heavyweight version row per commit")
		assert.NotEmpty(t, repo.eventsFor(TopicUpdated))
	})
}

func TestCreateNamedVersion(t *testing.T) {
	seed := func(t *testing.T, s *Service, repo *fakeRepo) *Doc {
		t.Helper()
		d, err := s.Create(testCtx(), "project-1", "Spec", "**bold**")
		require.NoError(t, err)
		return d
	}
	t.Run("empty doc id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.CreateNamedVersion(testCtx(), " ", "v1")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.CreateNamedVersion(testCtx(), "doc-1", "  ")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("missing doc is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.CreateNamedVersion(testCtx(), "nope", "v1")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("missing actor is unauthorized", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.CreateNamedVersion(context.Background(), "doc-1", "v1")
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})
	t.Run("no edit bit is forbidden", func(t *testing.T) {
		repo := newFakeRepo()
		s := newDenyService(repo)
		d := seed(t, s, repo)
		_, err := s.CreateNamedVersion(testCtx(), d.ID, "v1")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("captures current state with name author and timestamp", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		d := seed(t, s, repo)
		baseVersion := d.Version

		v, err := s.CreateNamedVersion(testCtx(), d.ID, "MVP cut")
		require.NoError(t, err)
		assert.Equal(t, d.ID, v.DocID)
		assert.Equal(t, baseVersion+1, v.Version, "named rows take a fresh version key (PK doc_id+version)")
		assert.Equal(t, "MVP cut", v.Name)
		assert.Equal(t, "user-1", v.AuthorID)
		assert.Equal(t, fixedNow, v.CreatedAt)

		vs, err := s.ListVersions(testCtx(), d.ID)
		require.NoError(t, err)
		require.Len(t, vs, 2)
		assert.Equal(t, "MVP cut", vs[0].Name)
		assert.Equal(t, "user-1", vs[0].AuthorID)

		got, err := s.Get(testCtx(), d.ID)
		require.NoError(t, err)
		assert.Equal(t, v.Version, got.Version, "doc counter advances to the named row's key")
	})
}

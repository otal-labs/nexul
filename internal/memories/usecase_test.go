package memories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var fixedNow = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

// fakeRepo is an in-memory memories.Repo for use-case tests.
type fakeRepo struct {
	mu          sync.Mutex
	memories    map[string]*Memory
	versions    map[string][]*MemoryVersion
	events      []eventbus.OutboxEvent
	createErr   error
	getErr      error
	listErr     error
	updateErr   error
	deleteErr   error
	templates   map[string]*InterviewTemplate
	templateErr error
	answers     []*InterviewAnswer
	answerErr   error
	sources     []*InterviewSource
	drafts      []*InterviewDraft
	sourceErr   error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{memories: map[string]*Memory{}, versions: map[string][]*MemoryVersion{}}
}

func (f *fakeRepo) Create(_ context.Context, m *Memory, authorVia string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if _, ok := f.memories[m.ID]; ok {
		return apperrs.ErrConflict
	}
	cp := *m
	f.memories[m.ID] = &cp
	f.versions[m.ID] = append(f.versions[m.ID], versionFrom(m, authorVia))
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id string) (*Memory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	m, ok := f.memories[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *m
	return &cp, nil
}

func (f *fakeRepo) ListByWorkspace(_ context.Context, workspaceID string) ([]*Memory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Memory
	for _, m := range f.memories {
		if m.WorkspaceID == workspaceID {
			cp := *m
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListByProject(_ context.Context, projectID string) ([]*Memory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Memory
	for _, m := range f.memories {
		if m.ProjectID == projectID {
			cp := *m
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, m *Memory, authorVia string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	if _, ok := f.memories[m.ID]; !ok {
		return apperrs.ErrNotFound
	}
	cp := *m
	f.memories[m.ID] = &cp
	f.versions[m.ID] = append(f.versions[m.ID], versionFrom(m, authorVia))
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) ListVersions(_ context.Context, memoryID string) ([]*MemoryVersion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	vs := f.versions[memoryID]
	out := make([]*MemoryVersion, len(vs))
	for i := range vs {
		cp := *vs[len(vs)-1-i]
		out[i] = &cp
	}
	return out, nil
}

func (f *fakeRepo) GetVersion(_ context.Context, memoryID string, version int) (*MemoryVersion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, v := range f.versions[memoryID] {
		if v.Version == version {
			cp := *v
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func versionFrom(m *Memory, authorVia string) *MemoryVersion {
	return &MemoryVersion{
		ID: fmt.Sprintf("version-%s-%d", m.ID, m.Version), MemoryID: m.ID, Version: m.Version,
		Title: m.Title, WhenToUse: m.WhenToUse, Body: m.Body, AlwaysIncluded: m.AlwaysIncluded,
		AuthorID: m.UpdatedBy, AuthorVia: authorVia, CreatedAt: m.UpdatedAt,
	}
}

func (f *fakeRepo) Delete(_ context.Context, id string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.memories[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.memories, id)
	f.events = append(f.events, evts...)
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

// fakeAccess is an AccessGate stub answering as access does: can answers every action check, deny refuses one
// "<scope>:<action>" key (a project or workspace id), membership alone passes, and a project outside
// testProjects or a scope in outside is not found.
type fakeAccess struct {
	can     bool
	deny    map[string]bool
	outside map[string]bool
}

var testProjects = map[string]string{"project-1": "workspace-1", "project-2": "workspace-2", "project-3": "workspace-1"}

func (f fakeAccess) Require(_ context.Context, workspaceID string, action permissions.Action) error {
	return f.check(workspaceID, action)
}

func (f fakeAccess) RequireProject(_ context.Context, projectID string, action permissions.Action) error {
	if _, ok := testProjects[projectID]; !ok {
		return fmt.Errorf("%w: project %s", apperrs.ErrNotFound, projectID)
	}
	return f.check(projectID, action)
}

func (f fakeAccess) Can(_ context.Context, _, docID string, action permissions.Action) (bool, error) {
	return f.check(docID, action) == nil, nil
}

func (f fakeAccess) check(scope string, action permissions.Action) error {
	if f.outside[scope] {
		return fmt.Errorf("%w: %s", apperrs.ErrNotFound, scope)
	}
	if action == permissions.Member {
		return nil
	}
	if !f.can || f.deny[scope+":"+string(action)] {
		return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, action)
	}
	return nil
}

// fakeProjects is a ProjectLookup stub mapping project id to workspace id.
type fakeProjects struct {
	workspaces map[string]string
	err        error
}

func (f fakeProjects) WorkspaceForProject(_ context.Context, projectID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	ws, ok := f.workspaces[projectID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return ws, nil
}

func (f fakeProjects) ProjectName(_ context.Context, projectID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if _, ok := f.workspaces[projectID]; !ok {
		return "", apperrs.ErrNotFound
	}
	return "Project " + projectID, nil
}

// fakeAttachments is an AttachmentsCopier stub: no attachments by default, records copy calls.
type fakeAttachments struct {
	ownerIDs map[string][]string
	copies   []copyCall
}

type copyCall struct {
	from, to string
	idMap    map[string]string
}

func (f *fakeAttachments) ListOwnerIDs(_ context.Context, memoryID string) ([]string, error) {
	return f.ownerIDs[memoryID], nil
}

func (f *fakeAttachments) CopyOwnerWithIDs(_ context.Context, from, to string, idMap map[string]string) error {
	f.copies = append(f.copies, copyCall{from: from, to: to, idMap: idMap})
	return nil
}

func newTestService(repo *fakeRepo) *Service {
	return newTestServiceWith(repo, fakeAccess{can: true})
}

func newDenyService(repo *fakeRepo) *Service {
	return newTestServiceWith(repo, fakeAccess{can: false})
}

func newTestServiceWith(repo *fakeRepo, access AccessGate) *Service {
	s := NewService(repo, access, fakeProjects{workspaces: testProjects}, &fakeAttachments{ownerIDs: map[string][]string{}})
	s.now = func() time.Time { return fixedNow }
	return s
}

func testCtx() context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: "user-1"})
}

func TestCreate_RequiresMemoriesWrite(t *testing.T) {
	deny := newDenyService(newFakeRepo())
	_, err := deny.Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestCreate_NoProject_IsInvalid(t *testing.T) {
	repo := newFakeRepo()
	_, err := newTestService(repo).Create(testCtx(), " ", "Team tone", "when", "body", false, "")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Empty(t, repo.memories, "a memory always names a project")
}

func TestCreate_EmptyTitle_IsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Create(testCtx(), "project-1", " ", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestCreate_NoActor_IsUnauthorized(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Create(context.Background(), "project-1", "Title", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
}

func TestCreate_UnknownProject_PropagatesLookupError(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Create(testCtx(), "missing-project", "Title", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestCreate_CapsWhenToUse_AndPublishesCreated(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	long := strings.Repeat("x", maxWhenToUseChars+50)

	m, err := s.Create(testCtx(), "project-1", "Deploy quirks", "  "+long+"  ", "body", true, "")
	require.NoError(t, err)
	assert.Equal(t, "workspace-1", m.WorkspaceID)
	assert.Equal(t, "project-1", m.ProjectID)
	assert.Len(t, m.WhenToUse, maxWhenToUseChars)
	assert.True(t, m.AlwaysIncluded)
	assert.Equal(t, "user-1", m.CreatedBy)
	require.Len(t, repo.eventsFor(TopicCreated), 1)
}

func TestGet_EmptyID_IsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Get(testCtx(), " ")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestGet_MissingMemory_PropagatesNotFound(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Get(testCtx(), "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestGet_RequiresMemoriesRead(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	deny := newDenyService(repo)
	_, err = deny.Get(testCtx(), m.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestList_EmptyWorkspaceID_IsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.List(testCtx(), " ")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestList_OutsideTheWorkspace_IsNotFound(t *testing.T) {
	s := newTestServiceWith(newFakeRepo(), fakeAccess{can: true, outside: map[string]bool{"workspace-1": true}})
	_, err := s.List(testCtx(), "workspace-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestList_LeavesOutProjectsTheCallerCannotRead(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.Create(testCtx(), "project-1", "Readable", "when", "body", false, "")
	require.NoError(t, err)
	hidden, err := s.Create(testCtx(), "project-2", "Hidden", "when", "body", false, "")
	require.NoError(t, err)
	hidden.WorkspaceID = "workspace-1"
	repo.memories[hidden.ID] = hidden

	got, err := newTestServiceWith(repo, fakeAccess{can: true, deny: map[string]bool{"project-2:memories:read": true}}).List(testCtx(), "workspace-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Readable", got[0].Title)
}

func TestList_ReturnsWorkspaceMemories(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	got, err := s.List(testCtx(), "workspace-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestListForProject_EmptyProjectID_IsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.ListForProject(testCtx(), " ")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestListForProject_RequiresMemoriesRead(t *testing.T) {
	deny := newDenyService(newFakeRepo())
	_, err := deny.ListForProject(testCtx(), "project-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestListMemoryItems_NoProject_ReturnsNone(t *testing.T) {
	repo := newFakeRepo()
	_, err := newTestService(repo).Create(testCtx(), "project-1", "Deploy quirks", "when to use", "body", false, "")
	require.NoError(t, err)

	items, err := newTestService(repo).ListMemoryItems(context.Background(), " ")
	require.NoError(t, err)
	assert.Empty(t, items, "a turn with no project carries no memories")
}

func TestListMemoryItems_NoActorInContext_ReturnsTheProjectsItems(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.Create(testCtx(), "project-1", "Deploy quirks", "when to use", "body", false, "")
	require.NoError(t, err)
	_, err = s.Create(testCtx(), "project-2", "Other project", "when to use", "body", false, "")
	require.NoError(t, err)

	// The turn pipeline reads the memories on the server's behalf; this seam carries no permission check by design.
	items, err := newDenyService(repo).ListMemoryItems(context.Background(), "project-1")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "Deploy quirks", items[0].Title)
}

func TestUpdate_EmptyID_IsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Update(testCtx(), " ", "Title", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestUpdate_EmptyTitle_IsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Update(testCtx(), "id", " ", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestUpdate_MissingMemory_PropagatesNotFound(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Update(testCtx(), "missing", "Title", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestUpdate_RequiresMemoriesWrite(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	deny := newDenyService(repo)
	_, err = deny.Update(testCtx(), m.ID, "New title", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestUpdate_ReplacesFields_AndPublishesUpdated(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	got, err := s.Update(testCtx(), m.ID, "New title", "new when", "new body", true, "")
	require.NoError(t, err)
	assert.Equal(t, "New title", got.Title)
	assert.Equal(t, "new when", got.WhenToUse)
	assert.True(t, got.AlwaysIncluded)
	assert.Equal(t, "user-1", got.UpdatedBy)
	require.Len(t, repo.eventsFor(TopicUpdated), 1)
}

func TestUpdateWithFooter_SetsIt_NilKeepsIt(t *testing.T) {
	s := newTestService(newFakeRepo())
	m, err := s.Create(testCtx(), "project-1", "Conclude", "", "body", false, "")
	require.NoError(t, err)
	footer := true

	got, err := s.UpdateWithFooter(testCtx(), m.ID, m.Title, "", "body", false, &footer, "")
	require.NoError(t, err)
	assert.True(t, got.Footer)

	got, err = s.Update(testCtx(), m.ID, "Renamed", "", "body", false, "")
	require.NoError(t, err)
	assert.True(t, got.Footer, "a save that does not name the footer keeps it")
}

func TestUpdateWithFooter_SpecialKinds_NeverFooters(t *testing.T) {
	s := newTestService(newFakeRepo())
	log, err := s.CreateWithKind(testCtx(), KindDecisionsLog, "project-1", "", "", "x", false, "")
	require.NoError(t, err)
	footer := true

	got, err := s.UpdateWithFooter(testCtx(), log.ID, log.Title, log.WhenToUse, "x", false, &footer, "")
	require.NoError(t, err)
	assert.False(t, got.Footer)
}

func TestDelete_EmptyID_IsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo())
	err := s.Delete(testCtx(), " ")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestDelete_MissingMemory_PropagatesNotFound(t *testing.T) {
	s := newTestService(newFakeRepo())
	err := s.Delete(testCtx(), "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestDelete_RequiresMemoriesDelete(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	deny := newDenyService(repo)
	err = deny.Delete(testCtx(), m.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestDelete_RemovesMemory_AndPublishesDeleted(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	require.NoError(t, s.Delete(testCtx(), m.ID))
	_, err = s.Get(testCtx(), m.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	require.Len(t, repo.eventsFor(TopicDeleted), 1)
}

func TestExportMarkdown_RendersBody(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Title", "when", "plain body", false, "")
	require.NoError(t, err)

	md, err := s.ExportMarkdown(testCtx(), m.ID)
	require.NoError(t, err)
	assert.Contains(t, md, "plain body")
}

func TestUpdate_TwiceThenRevertToFirst_AppendsThirdVersionMatchingIt(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Title", "when", "v1 body", false, "")
	require.NoError(t, err)
	require.Equal(t, 1, m.Version)

	_, err = s.Update(testCtx(), m.ID, "Title v2", "when", "v2 body", false, "")
	require.NoError(t, err)
	_, err = s.Update(testCtx(), m.ID, "Title v3", "when", "v3 body", false, "")
	require.NoError(t, err)

	versions, err := s.ListVersions(testCtx(), m.ID)
	require.NoError(t, err)
	require.Len(t, versions, 3)
	assert.Equal(t, 3, versions[0].Version) // newest first

	reverted, err := s.Revert(testCtx(), m.ID, 1, "")
	require.NoError(t, err)
	assert.Equal(t, 4, reverted.Version)
	assert.Contains(t, reverted.Body, "v1 body")
	assert.Equal(t, "Title", reverted.Title)

	versions, err = s.ListVersions(testCtx(), m.ID)
	require.NoError(t, err)
	require.Len(t, versions, 4)
}

func TestCreate_ViaMCP_TagsFirstVersion(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Title", "when", "body", false, "mcp")
	require.NoError(t, err)

	v, err := s.GetVersion(testCtx(), m.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "mcp", v.AuthorVia)
	assert.Equal(t, "user-1", v.AuthorID)
}

func TestUpdate_ViaMCP_TagsVersion(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	_, err = s.Update(testCtx(), m.ID, "Title", "when", "new body", false, "mcp")
	require.NoError(t, err)

	v, err := s.GetVersion(testCtx(), m.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, "mcp", v.AuthorVia)
}

func TestListVersions_RequiresMemoriesRead(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	deny := newDenyService(repo)
	_, err = deny.ListVersions(testCtx(), m.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestGetVersion_NegativeVersion_IsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.GetVersion(testCtx(), "id", 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestRevert_RequiresMemoriesWrite_NotJustRead(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	deny := newDenyService(repo)
	_, err = deny.Revert(testCtx(), m.ID, 1, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestCan_NilAccessGate_FailsClosed(t *testing.T) {
	s := NewService(newFakeRepo(), nil, fakeProjects{workspaces: testProjects}, nil)
	_, err := s.Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestClone_MissingMemoriesClone_NamesThatPermission(t *testing.T) {
	repo := newFakeRepo()
	m, err := newTestService(repo).Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	_, err = newDenyService(repo).Clone(testCtx(), m.ID, "project-2")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), "memories:clone")
}

func TestClone_NoDestinationProject_IsInvalid(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	_, err = s.Clone(testCtx(), m.ID, " ")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Len(t, repo.memories, 1, "there is no workspace to clone into")
}

func TestClone_DestinationInAWorkspaceTheCallerIsOutside_IsNotFound(t *testing.T) {
	repo := newFakeRepo()
	m, err := newTestService(repo).Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	s := newTestServiceWith(repo, fakeAccess{can: true, outside: map[string]bool{"project-2": true}})
	_, err = s.Clone(testCtx(), m.ID, "project-2")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestClone_MissingDestinationMemoriesWrite_NamesThatPermission(t *testing.T) {
	repo := newFakeRepo()
	m, err := newTestService(repo).Create(testCtx(), "project-1", "Title", "when", "body", false, "")
	require.NoError(t, err)

	// memories:clone on the source stays allowed; only memories:write on the destination project is denied.
	s := newTestServiceWith(repo, fakeAccess{can: true, deny: map[string]bool{"project-2:memories:write": true}})
	_, err = s.Clone(testCtx(), m.ID, "project-2")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), "memories:write")
}

func TestClone_CopiesFieldsAndStartsFreshVersionHistory(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "Original", "when to use", "body text", true, "")
	require.NoError(t, err)
	_, err = s.Update(testCtx(), m.ID, "Original", "when to use", "body text v2", true, "")
	require.NoError(t, err)

	clone, err := s.Clone(testCtx(), m.ID, "project-2")
	require.NoError(t, err)
	assert.NotEqual(t, m.ID, clone.ID)
	assert.Equal(t, "workspace-2", clone.WorkspaceID)
	assert.Equal(t, "project-2", clone.ProjectID)
	assert.Equal(t, "Original", clone.Title)
	assert.Equal(t, "when to use", clone.WhenToUse)
	assert.True(t, clone.AlwaysIncluded)
	assert.Equal(t, 1, clone.Version)

	versions, err := s.ListVersions(testCtx(), clone.ID)
	require.NoError(t, err)
	require.Len(t, versions, 1)

	// The source is untouched by cloning.
	source, err := s.Get(testCtx(), m.ID)
	require.NoError(t, err)
	assert.Contains(t, source.Body, "body text v2")
}

func TestClone_CopiesAttachmentsAndRewritesBodyReferences(t *testing.T) {
	repo := newFakeRepo()
	attachmentsCopier := &fakeAttachments{ownerIDs: map[string][]string{}}
	s := NewService(repo, fakeAccess{can: true}, fakeProjects{workspaces: testProjects}, attachmentsCopier)
	s.now = func() time.Time { return fixedNow }

	body := `{"type":"doc","content":[{"type":"image","attrs":{"src":"/api/attachments/att-1","alt":"shot"}}]}`
	m, err := s.Create(testCtx(), "project-1", "Title", "when", body, false, "")
	require.NoError(t, err)
	attachmentsCopier.ownerIDs[m.ID] = []string{"att-1"}

	clone, err := s.Clone(testCtx(), m.ID, "project-2")
	require.NoError(t, err)
	require.Len(t, attachmentsCopier.copies, 1)
	call := attachmentsCopier.copies[0]
	assert.Equal(t, m.ID, call.from)
	assert.Equal(t, clone.ID, call.to)
	newID, ok := call.idMap["att-1"]
	require.True(t, ok)
	assert.NotEqual(t, "att-1", newID)
	assert.Contains(t, clone.Body, "/api/attachments/"+newID)
	assert.NotContains(t, clone.Body, "/api/attachments/att-1")
}

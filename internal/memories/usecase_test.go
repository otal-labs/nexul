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

// ListByProject mirrors the real query: the project's own memories plus its workspace's workspace-scoped ones.
func (f *fakeRepo) ListByProject(_ context.Context, projectID, workspaceID string) ([]*Memory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Memory
	for _, m := range f.memories {
		if m.ProjectID == projectID || (m.ProjectID == "" && m.WorkspaceID == workspaceID) {
			cp := *m
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListWorkspaceScoped(_ context.Context, workspaceID string) ([]*Memory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Memory
	for _, m := range f.memories {
		if m.ProjectID == "" && m.WorkspaceID == workspaceID {
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

// fakeAccess is a PermissionGate stub for unit tests. deny overrides can for a specific
// "<workspaceID>:<action>" key, letting Clone tests deny exactly one of its three checks.
type fakeAccess struct {
	can  bool
	deny map[string]bool
}

func (f fakeAccess) HasPermission(_ context.Context, _, workspaceID string, action permissions.Action) bool {
	if allowed, ok := f.deny[workspaceID+":"+string(action)]; ok {
		return allowed
	}
	return f.can
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

// fakeMembership is a WorkspaceMembership stub mapping workspace id to member user ids.
type fakeMembership struct {
	members map[string][]string
	err     error
}

func (f fakeMembership) IsMember(_ context.Context, userID, workspaceID string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for _, m := range f.members[workspaceID] {
		if m == userID {
			return true, nil
		}
	}
	return false, nil
}

func newTestService(repo *fakeRepo) *Service {
	return newTestServiceWith(repo, fakeAccess{can: true})
}

func newDenyService(repo *fakeRepo) *Service {
	return newTestServiceWith(repo, fakeAccess{can: false})
}

func newTestServiceWith(repo *fakeRepo, access PermissionGate) *Service {
	s := NewService(repo, access, fakeProjects{workspaces: map[string]string{"project-1": "workspace-1", "project-2": "workspace-2"}},
		&fakeAttachments{ownerIDs: map[string][]string{}}, fakeMembership{members: map[string][]string{"workspace-1": {"user-1"}, "workspace-2": {"user-1"}}})
	s.now = func() time.Time { return fixedNow }
	return s
}

func testCtx() context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: "user-1"})
}

func TestCreate_EmptyProjectIDAndWorkspaceID_IsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Create(testCtx(), " ", " ", "Title", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestCreate_EmptyProjectID_CreatesWorkspaceScopedMemory(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "", "workspace-1", "Team tone", "when", "body", false, "")
	require.NoError(t, err)
	assert.Empty(t, m.ProjectID)
	assert.Equal(t, "workspace-1", m.WorkspaceID)
	require.Len(t, repo.eventsFor(TopicCreated), 1)
}

func TestCreate_EmptyTitle_IsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Create(testCtx(), "project-1", "", " ", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestCreate_NoActor_IsUnauthorized(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Create(context.Background(), "project-1", "", "Title", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
}

func TestCreate_UnknownProject_PropagatesLookupError(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Create(testCtx(), "missing-project", "", "Title", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestCreate_RequiresMemoriesWrite(t *testing.T) {
	deny := newDenyService(newFakeRepo())
	_, err := deny.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestCreate_CapsWhenToUse_AndPublishesCreated(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	long := strings.Repeat("x", maxWhenToUseChars+50)

	m, err := s.Create(testCtx(), "project-1", "", "Deploy quirks", "  "+long+"  ", "body", true, "")
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
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
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

func TestList_RequiresMemoriesRead(t *testing.T) {
	deny := newDenyService(newFakeRepo())
	_, err := deny.List(testCtx(), "workspace-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestList_ReturnsWorkspaceMemories(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
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

func TestListMemoryItems_EmptyWorkspaceAndProjectID_ReturnsEmptyNoError(t *testing.T) {
	deny := newDenyService(newFakeRepo())
	workspace, project, err := deny.ListMemoryItems(context.Background(), "", "")
	require.NoError(t, err)
	assert.Empty(t, workspace)
	assert.Empty(t, project)
}

func TestListMemoryItems_NoActorInContext_StillReturnsItems(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.Create(testCtx(), "project-1", "", "Deploy quirks", "when to use", "body", false, "")
	require.NoError(t, err)

	// The turn pipeline runs with no acting-user context; this seam carries no permission check by design.
	workspace, project, err := s.ListMemoryItems(context.Background(), "workspace-1", "project-1")
	require.NoError(t, err)
	assert.Empty(t, workspace)
	require.Len(t, project, 1)
	assert.Equal(t, "Deploy quirks", project[0].Title)
	assert.Equal(t, "when to use", project[0].WhenToUse)
}

func TestListMemoryItems_EmptyProjectID_ReturnsWorkspaceScopedOnly(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.Create(testCtx(), "project-1", "", "Deploy quirks", "when to use", "body", false, "")
	require.NoError(t, err)
	_, err = s.Create(testCtx(), "", "workspace-1", "Team tone", "when to use", "body", false, "")
	require.NoError(t, err)

	workspace, project, err := s.ListMemoryItems(context.Background(), "workspace-1", "")
	require.NoError(t, err)
	require.Len(t, workspace, 1)
	assert.Equal(t, "Team tone", workspace[0].Title)
	assert.Empty(t, project)
}

func TestListMemoryItems_WithProjectID_SplitsWorkspaceAndProjectItems(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.Create(testCtx(), "project-1", "", "Deploy quirks", "when to use", "body", false, "")
	require.NoError(t, err)
	_, err = s.Create(testCtx(), "", "workspace-1", "Team tone", "when to use", "body", false, "")
	require.NoError(t, err)

	workspace, project, err := s.ListMemoryItems(context.Background(), "workspace-1", "project-1")
	require.NoError(t, err)
	require.Len(t, workspace, 1)
	assert.Equal(t, "Team tone", workspace[0].Title)
	require.Len(t, project, 1)
	assert.Equal(t, "Deploy quirks", project[0].Title)
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
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)

	deny := newDenyService(repo)
	_, err = deny.Update(testCtx(), m.ID, "New title", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestUpdate_ReplacesFields_AndPublishesUpdated(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)

	got, err := s.Update(testCtx(), m.ID, "New title", "new when", "new body", true, "")
	require.NoError(t, err)
	assert.Equal(t, "New title", got.Title)
	assert.Equal(t, "new when", got.WhenToUse)
	assert.True(t, got.AlwaysIncluded)
	assert.Equal(t, "user-1", got.UpdatedBy)
	require.Len(t, repo.eventsFor(TopicUpdated), 1)
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
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)

	deny := newDenyService(repo)
	err = deny.Delete(testCtx(), m.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestDelete_RemovesMemory_AndPublishesDeleted(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
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
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "plain body", false, "")
	require.NoError(t, err)

	md, err := s.ExportMarkdown(testCtx(), m.ID)
	require.NoError(t, err)
	assert.Contains(t, md, "plain body")
}

func TestUpdate_TwiceThenRevertToFirst_AppendsThirdVersionMatchingIt(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "v1 body", false, "")
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
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "mcp")
	require.NoError(t, err)

	v, err := s.GetVersion(testCtx(), m.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "mcp", v.AuthorVia)
	assert.Equal(t, "user-1", v.AuthorID)
}

func TestUpdate_ViaMCP_TagsVersion(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
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
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
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
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)

	deny := newDenyService(repo)
	_, err = deny.Revert(testCtx(), m.ID, 1, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestCan_NilAccessGate_FailsClosed(t *testing.T) {
	s := NewService(newFakeRepo(), nil, fakeProjects{workspaces: map[string]string{"project-1": "workspace-1"}},
		nil, nil)
	_, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
}

func TestClone_EmptyDestinationProjectIDAndWorkspaceID_IsInvalid(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)

	_, err = s.Clone(testCtx(), m.ID, " ", " ")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestClone_EmptyDestinationProjectID_ClonesToWorkspace(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)

	clone, err := s.Clone(testCtx(), m.ID, "", "workspace-2")
	require.NoError(t, err)
	assert.Empty(t, clone.ProjectID)
	assert.Equal(t, "workspace-2", clone.WorkspaceID)
}

func TestClone_MissingMemoriesClone_NamesThatPermission(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)

	deny := newDenyService(repo)
	_, err = deny.Clone(testCtx(), m.ID, "project-2", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	assert.Contains(t, err.Error(), "memories:clone")
}

func TestClone_NotMemberOfDestinationWorkspace_NamesMembership(t *testing.T) {
	repo := newFakeRepo()
	s := NewService(repo, fakeAccess{can: true}, fakeProjects{workspaces: map[string]string{"project-1": "workspace-1", "project-2": "workspace-2"}},
		&fakeAttachments{ownerIDs: map[string][]string{}}, fakeMembership{members: map[string][]string{"workspace-1": {"user-1"}}})
	s.now = func() time.Time { return fixedNow }
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)

	_, err = s.Clone(testCtx(), m.ID, "project-2", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	assert.Contains(t, err.Error(), "member")
}

func TestClone_MissingDestinationMemoriesWrite_NamesThatPermission(t *testing.T) {
	repo := newFakeRepo()
	source := newTestService(repo)
	m, err := source.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)

	// memories:clone on workspace-1 stays allowed; only memories:write on the destination is denied.
	access := fakeAccess{can: true, deny: map[string]bool{"workspace-2:memories:write": false}}
	s := NewService(repo, access, fakeProjects{workspaces: map[string]string{"project-1": "workspace-1", "project-2": "workspace-2"}},
		&fakeAttachments{ownerIDs: map[string][]string{}}, fakeMembership{members: map[string][]string{"workspace-1": {"user-1"}, "workspace-2": {"user-1"}}})
	s.now = func() time.Time { return fixedNow }

	_, err = s.Clone(testCtx(), m.ID, "project-2", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	assert.Contains(t, err.Error(), "memories:write")
}

func TestClone_CopiesFieldsAndStartsFreshVersionHistory(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	m, err := s.Create(testCtx(), "project-1", "", "Original", "when to use", "body text", true, "")
	require.NoError(t, err)
	_, err = s.Update(testCtx(), m.ID, "Original", "when to use", "body text v2", true, "")
	require.NoError(t, err)

	clone, err := s.Clone(testCtx(), m.ID, "project-2", "")
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
	s := NewService(repo, fakeAccess{can: true}, fakeProjects{workspaces: map[string]string{"project-1": "workspace-1", "project-2": "workspace-2"}},
		attachmentsCopier, fakeMembership{members: map[string][]string{"workspace-1": {"user-1"}, "workspace-2": {"user-1"}}})
	s.now = func() time.Time { return fixedNow }

	body := `{"type":"doc","content":[{"type":"image","attrs":{"src":"/api/attachments/att-1","alt":"shot"}}]}`
	m, err := s.Create(testCtx(), "project-1", "", "Title", "when", body, false, "")
	require.NoError(t, err)
	attachmentsCopier.ownerIDs[m.ID] = []string{"att-1"}

	clone, err := s.Clone(testCtx(), m.ID, "project-2", "")
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

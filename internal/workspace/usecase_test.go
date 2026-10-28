package workspace

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var wsFixedNow = time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)

// fakeGate answers every permission check the same way: allowed, forbidden, or failing with err.
type fakeGate struct {
	mu    sync.Mutex
	allow bool
	err   error
}

func (f *fakeGate) Require(_ context.Context, _ string, action permissions.Action) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	if !f.allow && action != permissions.Member {
		return apperrs.ErrForbidden
	}
	return nil
}

func (f *fakeGate) RequireProject(ctx context.Context, _ string, action permissions.Action) error {
	return f.Require(ctx, "", action)
}

func (f *fakeGate) CallerProjectsIn(ctx context.Context, _ string, action permissions.Action) ([]string, bool, error) {
	err := f.Require(ctx, "", action)
	if permissions.Refused(err) {
		return []string{}, false, nil
	}
	return nil, err == nil, err
}

type fakeWorkspaceGate struct {
	mu     sync.Mutex
	exists bool
	err    error
}

func (f *fakeWorkspaceGate) WorkspaceExists(_ context.Context, _ string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.exists, f.err
}

type fakeRepo struct {
	mu        sync.Mutex
	projects  map[string]*Project
	repos     map[string][]RepoRef
	services  map[string][]string
	ticketPro map[string]string
	cats      map[string]*Category
	types     map[string]*TicketType
	statuses  map[string]*Status
	labels    map[string][]string
	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error
	countErr  error
	repoErr   error
	// repoLookupErr fails only GetRepoByFullName, so an add can conflict and its lookup fail.
	repoLookupErr error
	moveErr       error
	saved         []eventbus.OutboxEvent
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		projects:  map[string]*Project{},
		repos:     map[string][]RepoRef{},
		services:  map[string][]string{},
		ticketPro: map[string]string{},
		cats:      map[string]*Category{},
		types:     map[string]*TicketType{},
		statuses:  map[string]*Status{},
		labels:    map[string][]string{},
	}
}

func (f *fakeRepo) Create(_ context.Context, p *Project) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if _, ok := f.projects[p.ID]; ok {
		return apperrs.ErrConflict
	}
	f.projects[p.ID] = p
	return nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (*Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	p, ok := f.projects[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return p, nil
}

func (f *fakeRepo) List(_ context.Context, workspaceID string) ([]*Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*Project, 0, len(f.projects))
	for _, p := range f.projects {
		if p.WorkspaceID != workspaceID {
			continue
		}
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b *Project) int { return cmp.Or(cmp.Compare(a.Position, b.Position), cmp.Compare(a.ID, b.ID)) })
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, p *Project) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	cur, ok := f.projects[p.ID]
	if !ok {
		return apperrs.ErrNotFound
	}
	cur.Name = p.Name
	cur.Prefix = p.Prefix
	cur.TestsLocation = p.TestsLocation
	cur.UpdatedAt = p.UpdatedAt
	return nil
}

func (f *fakeRepo) SaveSetup(_ context.Context, id string, apply func(*Project) []eventbus.OutboxEvent) (*Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	cur, ok := f.projects[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	f.saved = append(f.saved, apply(cur)...)
	return cur, nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.projects[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.projects, id)
	return nil
}

func (f *fakeRepo) Reorder(_ context.Context, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, id := range ids {
		if _, ok := f.projects[id]; !ok {
			return apperrs.ErrNotFound
		}
		f.projects[id].Position = i
	}
	return nil
}

func (f *fakeRepo) CountTickets(_ context.Context, projectID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.countErr != nil {
		return 0, f.countErr
	}
	n := 0
	for _, p := range f.ticketPro {
		if p == projectID {
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) CountRepos(_ context.Context, projectID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.countErr != nil {
		return 0, f.countErr
	}
	return len(f.repos[projectID]), nil
}

func (f *fakeRepo) ProjectForStack(_ context.Context, stackID string) (string, error) {
	for projectID, stacks := range f.services {
		if slices.Contains(stacks, stackID) {
			return projectID, nil
		}
	}
	return "", apperrs.ErrNotFound
}

func (f *fakeRepo) CountServices(_ context.Context, projectID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.countErr != nil {
		return 0, f.countErr
	}
	return len(f.services[projectID]), nil
}

func (f *fakeRepo) AddRepo(_ context.Context, projectID string, r RepoRef) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.repoErr != nil {
		return f.repoErr
	}
	if _, ok := f.projects[projectID]; !ok {
		return apperrs.ErrNotFound
	}
	for _, repos := range f.repos {
		for _, existing := range repos {
			if existing.Owner == r.Owner && existing.Name == r.Name {
				return apperrs.ErrConflict
			}
		}
	}
	f.repos[projectID] = append(f.repos[projectID], r)
	return nil
}

func (f *fakeRepo) RemoveRepo(_ context.Context, owner, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.repoErr != nil {
		return f.repoErr
	}
	for id, repos := range f.repos {
		for i, r := range repos {
			if r.Owner == owner && r.Name == name {
				f.repos[id] = append(repos[:i], repos[i+1:]...)
				return nil
			}
		}
	}
	return apperrs.ErrNotFound
}

func (f *fakeRepo) GetRepoByFullName(_ context.Context, owner, name string) (RepoRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := cmp.Or(f.repoErr, f.repoLookupErr); err != nil {
		return RepoRef{}, err
	}
	for projectID, repos := range f.repos {
		for _, r := range repos {
			if r.Owner == owner && r.Name == name {
				r.ProjectID = projectID
				return r, nil
			}
		}
	}
	return RepoRef{}, apperrs.ErrNotFound
}

func (f *fakeRepo) ListRepos(_ context.Context, projectID string) ([]RepoRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]RepoRef(nil), f.repos[projectID]...), nil
}

func (f *fakeRepo) MoveTicket(_ context.Context, ticketID, projectID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.moveErr != nil {
		return f.moveErr
	}
	if _, ok := f.ticketPro[ticketID]; !ok {
		return apperrs.ErrNotFound
	}
	f.ticketPro[ticketID] = projectID
	return nil
}

func newTestService(repo *fakeRepo, owner Gate) *Service {
	s := NewService(repo, newFakeCategoryRepo(), newFakeTicketTypeRepo(), newFakeStatusRepo(), owner, &fakeWorkspaceGate{exists: true})
	s.now = func() time.Time { return wsFixedNow }
	return s
}

// fakeCategoryRepo is an in-memory CategoryRepo for use-case tests.
type fakeCategoryRepo struct {
	mu        sync.Mutex
	cats      map[string]*Category
	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error
	moveErr   error
	moved     []eventbus.OutboxEvent
}

func newFakeCategoryRepo() *fakeCategoryRepo {
	return &fakeCategoryRepo{cats: map[string]*Category{}}
}

func (f *fakeCategoryRepo) Create(_ context.Context, c *Category, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	f.cats[c.ID] = c
	return nil
}

func (f *fakeCategoryRepo) Get(_ context.Context, id string) (*Category, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	c, ok := f.cats[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return c, nil
}

func (f *fakeCategoryRepo) List(_ context.Context) ([]*Category, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*Category, 0, len(f.cats))
	for _, c := range f.cats {
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeCategoryRepo) ListByProject(_ context.Context, projectID string) ([]*Category, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Category
	for _, c := range f.cats {
		if c.ProjectID == projectID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeCategoryRepo) Update(_ context.Context, c *Category, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	cur, ok := f.cats[c.ID]
	if !ok {
		return apperrs.ErrNotFound
	}
	cur.Name = c.Name
	cur.UpdatedAt = c.UpdatedAt
	return nil
}

func (f *fakeCategoryRepo) Delete(_ context.Context, id string, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.cats[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.cats, id)
	return nil
}

func (f *fakeCategoryRepo) Reorder(_ context.Context, projectID string, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, id := range ids {
		c, ok := f.cats[id]
		if !ok {
			return apperrs.ErrNotFound
		}
		c.Position = i
	}
	return nil
}

func (f *fakeCategoryRepo) CountTickets(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (f *fakeCategoryRepo) SetTicketCategory(_ context.Context, ticketID, _ string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.moveErr != nil {
		return f.moveErr
	}
	if ticketID == "missing" {
		return apperrs.ErrNotFound
	}
	f.moved = append(f.moved, evts...)
	return nil
}

// fakeTicketProjects maps ticket ids to the project each sits in.
type fakeTicketProjects map[string]string

func (f fakeTicketProjects) ProjectOfTicket(_ context.Context, ticketID string) (string, error) {
	projectID, ok := f[ticketID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return projectID, nil
}

// fakeTicketTypeRepo is an in-memory TicketTypeRepo for use-case tests.
type fakeTicketTypeRepo struct {
	mu        sync.Mutex
	types     map[string]*TicketType
	count     int
	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error
}

func newFakeTicketTypeRepo() *fakeTicketTypeRepo {
	return &fakeTicketTypeRepo{types: map[string]*TicketType{}}
}

func (f *fakeTicketTypeRepo) Create(_ context.Context, t *TicketType, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	f.types[t.ID] = t
	return nil
}

func (f *fakeTicketTypeRepo) Get(_ context.Context, id string) (*TicketType, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	t, ok := f.types[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return t, nil
}

func (f *fakeTicketTypeRepo) ListByProject(_ context.Context, projectID string) ([]*TicketType, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*TicketType
	for _, t := range f.types {
		if t.ProjectID == projectID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeTicketTypeRepo) Update(_ context.Context, t *TicketType, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	cur, ok := f.types[t.ID]
	if !ok {
		return apperrs.ErrNotFound
	}
	cur.Name = t.Name
	cur.Color = t.Color
	cur.UpdatedAt = t.UpdatedAt
	return nil
}

func (f *fakeTicketTypeRepo) Delete(_ context.Context, id string, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.types[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.types, id)
	return nil
}

func (f *fakeTicketTypeRepo) Reorder(_ context.Context, _ string, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, id := range ids {
		t, ok := f.types[id]
		if !ok {
			return apperrs.ErrNotFound
		}
		t.Position = i
	}
	return nil
}

func (f *fakeTicketTypeRepo) CountTickets(_ context.Context, _ string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.count, nil
}

// fakeStatusRepo is an in-memory StatusRepo for use-case tests.
type fakeStatusRepo struct {
	mu        sync.Mutex
	statuses  map[string]*Status
	count     int
	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error
	updated   []eventbus.OutboxEvent
}

func newFakeStatusRepo() *fakeStatusRepo {
	return &fakeStatusRepo{statuses: map[string]*Status{}}
}

func (f *fakeStatusRepo) Create(_ context.Context, s *Status, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	f.statuses[s.ID] = s
	return nil
}

func (f *fakeStatusRepo) Get(_ context.Context, id string) (*Status, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	s, ok := f.statuses[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return s, nil
}

func (f *fakeStatusRepo) ListByProject(_ context.Context, projectID string) ([]*Status, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Status
	for _, s := range f.statuses {
		if s.ProjectID == projectID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeStatusRepo) Update(_ context.Context, s *Status, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = append(f.updated, evts...)
	cur, ok := f.statuses[s.ID]
	if !ok {
		return apperrs.ErrNotFound
	}
	cur.Name = s.Name
	cur.Kind = s.Kind
	cur.UpdatedAt = s.UpdatedAt
	return nil
}

func (f *fakeStatusRepo) Delete(_ context.Context, id string, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.statuses[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.statuses, id)
	return nil
}

func (f *fakeStatusRepo) Reorder(_ context.Context, _ string, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, id := range ids {
		s, ok := f.statuses[id]
		if !ok {
			return apperrs.ErrNotFound
		}
		s.Position = i
	}
	return nil
}

func (f *fakeStatusRepo) CountTickets(_ context.Context, _ string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.count, nil
}

func newOwnerRepo(t *testing.T, allow bool) (*Service, *fakeRepo, *fakeGate) {
	t.Helper()
	repo := newFakeRepo()
	owner := &fakeGate{allow: allow}
	return newTestService(repo, owner), repo, owner
}

func TestCreate(t *testing.T) {
	t.Run("unknown workspace is not found", func(t *testing.T) {
		repo := newFakeRepo()
		owner := &fakeGate{allow: true}
		s := NewService(repo, newFakeCategoryRepo(), newFakeTicketTypeRepo(), newFakeStatusRepo(), owner, &fakeWorkspaceGate{exists: false})
		_, err := s.Create(context.Background(), "ws-1", "Backend", "BE", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("workspace gate error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		owner := &fakeGate{allow: true}
		wsGate := &fakeWorkspaceGate{err: errors.New("db down")}
		s := NewService(repo, newFakeCategoryRepo(), newFakeTicketTypeRepo(), newFakeStatusRepo(), owner, wsGate)
		_, err := s.Create(context.Background(), "ws-1", "Backend", "BE", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, wsGate.err)
	})
	t.Run("creates with the next position", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", Prefix: "BE", Position: 0, WorkspaceID: "ws-1"}
		p, err := s.Create(context.Background(), "ws-1", "Frontend", "FE", "")
		require.NoError(t, err)
		assert.Equal(t, "Frontend", p.Name)
		assert.Equal(t, "ws-1", p.WorkspaceID)
		assert.Equal(t, 1, p.Position)
		assert.Equal(t, wsFixedNow, p.CreatedAt)
	})
	t.Run("scopes the next position and prefix uniqueness to the workspace", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", Prefix: "BE", Position: 0, WorkspaceID: "ws-other"}
		p, err := s.Create(context.Background(), "ws-1", "Frontend", "BE", "")
		require.NoError(t, err)
		assert.Equal(t, 0, p.Position)
	})
	prefixShapes := []struct {
		name   string
		prefix string
		want   string
	}{
		{"a letter then a digit", "P1", "P1"},
		{"two letters", "PH", "PH"},
		{"a letter, a digit, a letter", "PH1", "PH1"},
		{"five characters with a digit", "V2API", "V2API"},
		{"lowercase and padding are normalized", " nx ", "NX"},
		{"starting with a digit", "1P", ""},
		{"a single letter", "P", ""},
		{"a hyphen", "P-1", ""},
		{"six characters", "PHASE1X", ""},
		{"empty", "", ""},
	}
	for _, tt := range prefixShapes {
		t.Run("prefix shape: "+tt.name, func(t *testing.T) {
			s, _, _ := newOwnerRepo(t, true)
			p, err := s.Create(context.Background(), "ws-1", "Phase 1", tt.prefix, "")
			if tt.want == "" {
				require.ErrorIs(t, err, apperrs.ErrInvalid)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, p.Prefix)
		})
	}
	t.Run("icon from suggested list is valid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		p, err := s.Create(context.Background(), "ws-1", "Frontend", "FE", ProjectIconRocket)
		require.NoError(t, err)
		assert.Equal(t, ProjectIconRocket, p.Icon)
	})
}

func TestList(t *testing.T) {
	t.Run("lists a workspace's projects", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", WorkspaceID: "ws-1"}
		repo.projects["p-2"] = &Project{ID: "p-2", Name: "Frontend", WorkspaceID: "ws-1"}
		repo.projects["p-3"] = &Project{ID: "p-3", Name: "Other", WorkspaceID: "ws-other"}
		projects, err := s.List(context.Background(), "ws-1")
		require.NoError(t, err)
		require.Len(t, projects, 2)
	})
	t.Run("leaves out the projects the caller can't open", func(t *testing.T) {
		repo := newFakeRepo()
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", WorkspaceID: "ws-1"}
		repo.projects["p-2"] = &Project{ID: "p-2", Name: "Frontend", WorkspaceID: "ws-1"}
		s := newTestService(repo, &projectsGate{open: []string{"p-2"}})
		projects, err := s.List(context.Background(), "ws-1")
		require.NoError(t, err)
		require.Len(t, projects, 1)
		assert.Equal(t, "p-2", projects[0].ID)
	})
	t.Run("a failed access read propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", WorkspaceID: "ws-1"}
		gate := &projectsGate{err: errors.New("database is locked")}
		_, err := newTestService(repo, gate).List(context.Background(), "ws-1")
		assert.ErrorIs(t, err, gate.err)
	})
}

// projectsGate lets the caller open only the projects in open, or fails to list them with err.
type projectsGate struct {
	fakeGate
	open []string
	err  error
}

func (g *projectsGate) CallerProjectsIn(context.Context, string, permissions.Action) ([]string, bool, error) {
	return g.open, false, g.err
}

// onlyGate holds membership and one action.
type onlyGate struct{ action permissions.Action }

func (g onlyGate) Require(_ context.Context, _ string, action permissions.Action) error {
	if action == permissions.Member || action == g.action {
		return nil
	}
	return apperrs.ErrForbidden
}

func (g onlyGate) RequireProject(ctx context.Context, _ string, action permissions.Action) error {
	return g.Require(ctx, "", action)
}

func (g onlyGate) CallerProjectsIn(context.Context, string, permissions.Action) ([]string, bool, error) {
	return nil, true, nil
}

func TestRename(t *testing.T) {
	t.Run("renames a project", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Old", Position: 0}
		p, err := s.Rename(context.Background(), "p-1", "New name", nil)
		require.NoError(t, err)
		assert.Equal(t, "New name", p.Name)
		assert.Equal(t, wsFixedNow, p.UpdatedAt)
	})
	t.Run("sets and clears icon", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Old", Icon: ProjectIconRocket}
		globe := ProjectIconGlobe
		p, err := s.Rename(context.Background(), "p-1", "New name", &globe)
		require.NoError(t, err)
		assert.Equal(t, ProjectIconGlobe, p.Icon)

		empty := ProjectIcon("")
		p, err = s.Rename(context.Background(), "p-1", "New name", &empty)
		require.NoError(t, err)
		assert.Equal(t, ProjectIcon(""), p.Icon)
	})
	t.Run("nil icon keeps the current icon", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Old", Icon: ProjectIconRocket}
		p, err := s.Rename(context.Background(), "p-1", "New name", nil)
		require.NoError(t, err)
		assert.Equal(t, ProjectIconRocket, p.Icon)
	})
}

func TestSetPrefix(t *testing.T) {
	t.Run("setting the same prefix again is a no-op instead of a conflict", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "General", WorkspaceID: "ws-1", Prefix: "GEN"}
		p, err := s.SetPrefix(context.Background(), "p-1", "gen")
		require.NoError(t, err)
		assert.Equal(t, "GEN", p.Prefix)
	})
	t.Run("sets the prefix on a project with an empty prefix", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "General", WorkspaceID: "ws-1"}
		p, err := s.SetPrefix(context.Background(), "p-1", "gen")
		require.NoError(t, err)
		assert.Equal(t, "GEN", p.Prefix)
		assert.Equal(t, wsFixedNow, p.UpdatedAt)
		assert.Equal(t, "GEN", repo.projects["p-1"].Prefix)
	})
	t.Run("already-set prefix is refused", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", Prefix: "BE", WorkspaceID: "ws-1"}
		_, err := s.SetPrefix(context.Background(), "p-1", "BEX")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
		assert.Equal(t, "BE", repo.projects["p-1"].Prefix)
	})
	t.Run("duplicate prefix in another workspace is allowed", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "General", WorkspaceID: "ws-1"}
		repo.projects["p-2"] = &Project{ID: "p-2", Name: "Other", Prefix: "GEN", WorkspaceID: "ws-other"}
		p, err := s.SetPrefix(context.Background(), "p-1", "GEN")
		require.NoError(t, err)
		assert.Equal(t, "GEN", p.Prefix)
	})
}

func TestReorder(t *testing.T) {
	tests := []struct {
		name    string
		ids     []string
		wantErr bool
	}{
		{"empty list", nil, true},
		{"blank id", []string{" "}, true},
		{"duplicate id", []string{"p-1", "p-1"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo, _ := newOwnerRepo(t, true)
			repo.projects["p-1"] = &Project{ID: "p-1", Name: "A", WorkspaceID: "ws-1"}
			err := s.Reorder(context.Background(), "ws-1", tt.ids)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, apperrs.ErrInvalid))
				return
			}
			require.NoError(t, err)
		})
	}
	t.Run("missing ids must list every project", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A", WorkspaceID: "ws-1"}
		repo.projects["p-2"] = &Project{ID: "p-2", Name: "B", WorkspaceID: "ws-1"}
		err := s.Reorder(context.Background(), "ws-1", []string{"p-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("takes projects:create, not projects:write", func(t *testing.T) {
		repo := newFakeRepo()
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A", WorkspaceID: "ws-1"}
		err := newTestService(repo, onlyGate{permissions.ProjectsWrite}).Reorder(context.Background(), "ws-1", []string{"p-1"})
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		require.NoError(t, newTestService(repo, onlyGate{permissions.ProjectsCreate}).Reorder(context.Background(), "ws-1", []string{"p-1"}))
	})
	t.Run("orders the projects the caller is in and leaves a hidden one in its place", func(t *testing.T) {
		repo := newFakeRepo()
		for i, id := range []string{"p-1", "p-2", "p-3"} {
			repo.projects[id] = &Project{ID: id, Name: id, WorkspaceID: "ws-1", Position: i}
		}
		s := newTestService(repo, &projectsGate{fakeGate: fakeGate{allow: true}, open: []string{"p-1", "p-3"}})
		require.ErrorIs(t, s.Reorder(context.Background(), "ws-1", []string{"p-2", "p-3", "p-1"}), apperrs.ErrInvalid)
		require.NoError(t, s.Reorder(context.Background(), "ws-1", []string{"p-3", "p-1"}))
		assert.Equal(t, []int{2, 1, 0}, []int{repo.projects["p-1"].Position, repo.projects["p-2"].Position, repo.projects["p-3"].Position})
	})
	t.Run("reorders positions", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A", WorkspaceID: "ws-1"}
		repo.projects["p-2"] = &Project{ID: "p-2", Name: "B", WorkspaceID: "ws-1"}
		require.NoError(t, s.Reorder(context.Background(), "ws-1", []string{"p-2", "p-1"}))
		assert.Equal(t, 0, repo.projects["p-2"].Position)
		assert.Equal(t, 1, repo.projects["p-1"].Position)
	})
}

func TestDeleteImpact(t *testing.T) {
	t.Run("counts tickets, repos and services", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.ticketPro["t-1"] = "p-1"
		repo.ticketPro["t-2"] = "p-1"
		repo.repos["p-1"] = []RepoRef{{Owner: "acme", Name: "app"}}
		repo.services["p-1"] = []string{"svc-1"}
		impact, err := s.DeleteImpact(context.Background(), "p-1")
		require.NoError(t, err)
		assert.Equal(t, DeleteImpact{Tickets: 2, Repos: 1, Services: 1, RestrictedMembers: []RestrictedMember{}}, impact)
	})
}

func TestDelete(t *testing.T) {
	t.Run("project with services is refused", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.services["p-1"] = []string{"svc-1"}
		err := s.Delete(context.Background(), "p-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
		assert.Contains(t, err.Error(), "service")
	})
	t.Run("empty project deletes", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		require.NoError(t, s.Delete(context.Background(), "p-1"))
		_, err := s.Get(context.Background(), "p-1")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestAddRepo(t *testing.T) {
	t.Run("associates a repo with a project", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		require.NoError(t, s.AddRepo(context.Background(), "p-1", "acme", "app", "", ""))
		repos, err := s.ListRepos(context.Background(), "p-1")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "acme/app", repos[0].FullName)
	})
	t.Run("empty connector id defaults to github", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		require.NoError(t, s.AddRepo(context.Background(), "p-1", "acme", "app", "  ", ""))
		repos, err := s.ListRepos(context.Background(), "p-1")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "github", repos[0].ConnectorID)
	})
	t.Run("persists the given connector id", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		require.NoError(t, s.AddRepo(context.Background(), "p-1", "acme", "app", "gitlab-self-hosted", ""))
		repos, err := s.ListRepos(context.Background(), "p-1")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "gitlab-self-hosted", repos[0].ConnectorID)
	})
}

func TestRemoveRepo(t *testing.T) {
	t.Run("dissociates a repo", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.repos["p-1"] = []RepoRef{{Owner: "acme", Name: "app"}}
		require.NoError(t, s.RemoveRepo(context.Background(), "acme", "app"))
		repos, err := s.ListRepos(context.Background(), "p-1")
		require.NoError(t, err)
		assert.Empty(t, repos)
	})
}

func TestListRepos(t *testing.T) {
	t.Run("lists repos for a project", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.repos["p-1"] = []RepoRef{{Owner: "acme", Name: "app"}}
		repos, err := s.ListRepos(context.Background(), "p-1")
		require.NoError(t, err)
		require.Len(t, repos, 1)
	})
}

func TestMoveTicket(t *testing.T) {
	t.Run("moves a ticket into a project", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.ticketPro["t-1"] = "old"
		require.NoError(t, s.MoveTicket(context.Background(), "t-1", "p-1"))
		assert.Equal(t, "p-1", repo.ticketPro["t-1"])
	})
}

func TestProjectUsecases_Refusals(t *testing.T) {
	ctx := context.Background()
	errDown := errors.New("db down")
	seedProject := func(_ *Service, repo *fakeRepo, _ *fakeGate) {
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A", WorkspaceID: "ws-1"}
	}
	tests := []struct {
		name  string
		owner bool
		seed  func(s *Service, repo *fakeRepo, owner *fakeGate)
		call  func(s *Service) error
		want  error
	}{
		{"create without a workspace", true, nil, func(s *Service) error { _, err := s.Create(ctx, "  ", "Backend", "BE", ""); return err }, apperrs.ErrInvalid},
		{"create without a name", true, nil, func(s *Service) error { _, err := s.Create(ctx, "ws-1", "   ", "BE", ""); return err }, apperrs.ErrInvalid},
		{"create as a non-owner", false, nil, func(s *Service) error { _, err := s.Create(ctx, "ws-1", "Backend", "BE", ""); return err }, apperrs.ErrForbidden},
		{"create when the owner gate fails", true, func(_ *Service, _ *fakeRepo, owner *fakeGate) { owner.err = errDown }, func(s *Service) error {
			_, err := s.Create(ctx, "ws-1", "Backend", "BE", "")
			return err
		}, errDown},
		{"create when the repo fails", true, func(_ *Service, repo *fakeRepo, _ *fakeGate) { repo.createErr = errDown }, func(s *Service) error {
			_, err := s.Create(ctx, "ws-1", "Backend", "BE", "")
			return err
		}, errDown},
		{"create with a prefix another project uses", true, func(_ *Service, repo *fakeRepo, _ *fakeGate) {
			repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", Prefix: "BE", WorkspaceID: "ws-1"}
		}, func(s *Service) error { _, err := s.Create(ctx, "ws-1", "Frontend", "be", ""); return err }, apperrs.ErrInvalid},
		{"create with a bogus icon", true, nil, func(s *Service) error {
			_, err := s.Create(ctx, "ws-1", "Frontend", "FE", ProjectIcon("bogus"))
			return err
		}, apperrs.ErrInvalid},
		{"get without an id", true, nil, func(s *Service) error { _, err := s.Get(ctx, "  "); return err }, apperrs.ErrInvalid},
		{"get a missing project", true, nil, func(s *Service) error { _, err := s.Get(ctx, "nope"); return err }, apperrs.ErrNotFound},
		{"get when the repo fails", true, func(_ *Service, repo *fakeRepo, _ *fakeGate) { repo.getErr = errDown }, func(s *Service) error { _, err := s.Get(ctx, "p-1"); return err }, errDown},
		{"list without a workspace", true, nil, func(s *Service) error { _, err := s.List(ctx, "  "); return err }, apperrs.ErrInvalid},
		{"list when the repo fails", true, func(_ *Service, repo *fakeRepo, _ *fakeGate) { repo.listErr = errDown }, func(s *Service) error { _, err := s.List(ctx, "ws-1"); return err }, errDown},
		{"rename as a non-owner", false, nil, func(s *Service) error { _, err := s.Rename(ctx, "p-1", "New name", nil); return err }, apperrs.ErrForbidden},
		{"rename without a name", true, nil, func(s *Service) error { _, err := s.Rename(ctx, "p-1", " ", nil); return err }, apperrs.ErrInvalid},
		{"rename a missing project", true, nil, func(s *Service) error { _, err := s.Rename(ctx, "nope", "New name", nil); return err }, apperrs.ErrNotFound},
		{"rename with a bogus icon", true, seedProject, func(s *Service) error {
			bogus := ProjectIcon("bogus")
			_, err := s.Rename(ctx, "p-1", "New name", &bogus)
			return err
		}, apperrs.ErrInvalid},
		{"set a prefix as a non-owner", false, nil, func(s *Service) error { _, err := s.SetPrefix(ctx, "p-1", "GEN"); return err }, apperrs.ErrForbidden},
		{"set a prefix of the wrong shape", true, seedProject, func(s *Service) error { _, err := s.SetPrefix(ctx, "p-1", "1gen"); return err }, apperrs.ErrInvalid},
		{"set a prefix on a missing project", true, nil, func(s *Service) error { _, err := s.SetPrefix(ctx, "nope", "GEN"); return err }, apperrs.ErrNotFound},
		{"set a prefix another project in the workspace uses", true, func(s *Service, repo *fakeRepo, owner *fakeGate) {
			seedProject(s, repo, owner)
			repo.projects["p-2"] = &Project{ID: "p-2", Name: "Backend", Prefix: "BE", WorkspaceID: "ws-1"}
		}, func(s *Service) error { _, err := s.SetPrefix(ctx, "p-1", "be"); return err }, apperrs.ErrInvalid},
		{"reorder as a non-owner", false, nil, func(s *Service) error { return s.Reorder(ctx, "ws-1", []string{"p-1"}) }, apperrs.ErrForbidden},
		{"reorder without a workspace", true, nil, func(s *Service) error { return s.Reorder(ctx, "  ", []string{"p-1"}) }, apperrs.ErrInvalid},
		{"delete impact of a missing project", true, nil, func(s *Service) error { _, err := s.DeleteImpact(ctx, "nope"); return err }, apperrs.ErrNotFound},
		{"delete impact when counting fails", true, func(s *Service, repo *fakeRepo, owner *fakeGate) {
			seedProject(s, repo, owner)
			repo.countErr = errDown
		}, func(s *Service) error { _, err := s.DeleteImpact(ctx, "p-1"); return err }, errDown},
		{"delete as a non-owner", false, nil, func(s *Service) error { return s.Delete(ctx, "p-1") }, apperrs.ErrForbidden},
		{"delete a missing project", true, nil, func(s *Service) error { return s.Delete(ctx, "nope") }, apperrs.ErrNotFound},
		{"delete a project with tickets", true, func(s *Service, repo *fakeRepo, owner *fakeGate) {
			seedProject(s, repo, owner)
			repo.ticketPro["t-1"] = "p-1"
		}, func(s *Service) error { return s.Delete(ctx, "p-1") }, apperrs.ErrConflict},
		{"delete a project with repos", true, func(s *Service, repo *fakeRepo, owner *fakeGate) {
			seedProject(s, repo, owner)
			repo.repos["p-1"] = []RepoRef{{Owner: "acme", Name: "app"}}
		}, func(s *Service) error { return s.Delete(ctx, "p-1") }, apperrs.ErrConflict},
		{"add a repo as a non-owner", false, nil, func(s *Service) error { return s.AddRepo(ctx, "p-1", "acme", "app", "", "") }, apperrs.ErrForbidden},
		{"add a repo without an owner", true, nil, func(s *Service) error { return s.AddRepo(ctx, "p-1", "  ", "app", "", "") }, apperrs.ErrInvalid},
		{"add a repo to a missing project", true, nil, func(s *Service) error { return s.AddRepo(ctx, "nope", "acme", "app", "", "") }, apperrs.ErrNotFound},
		{"add a repo when the repo fails", true, func(s *Service, repo *fakeRepo, owner *fakeGate) {
			seedProject(s, repo, owner)
			repo.repoErr = errDown
		}, func(s *Service) error { return s.AddRepo(ctx, "p-1", "acme", "app", "", "") }, errDown},
		{"remove a repo as a non-owner", false, func(s *Service, repo *fakeRepo, owner *fakeGate) {
			seedProject(s, repo, owner)
			repo.repos["p-1"] = []RepoRef{{Owner: "acme", Name: "app"}}
		}, func(s *Service) error { return s.RemoveRepo(ctx, "acme", "app") }, apperrs.ErrForbidden},
		{"remove a repo without an owner", true, nil, func(s *Service) error { return s.RemoveRepo(ctx, "", "app") }, apperrs.ErrInvalid},
		{"remove a missing repo", true, nil, func(s *Service) error { return s.RemoveRepo(ctx, "acme", "app") }, apperrs.ErrNotFound},
		{"list repos without a project", true, nil, func(s *Service) error { _, err := s.ListRepos(ctx, " "); return err }, apperrs.ErrInvalid},
		{"move a ticket without an id", true, nil, func(s *Service) error { return s.MoveTicket(ctx, " ", "p-1") }, apperrs.ErrInvalid},
		{"move a ticket into a missing project", true, nil, func(s *Service) error { return s.MoveTicket(ctx, "t-1", "nope") }, apperrs.ErrNotFound},
		{"move a missing ticket", true, seedProject, func(s *Service) error { return s.MoveTicket(ctx, "nope", "p-1") }, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo, owner := newOwnerRepo(t, tt.owner)
			if tt.seed != nil {
				tt.seed(s, repo, owner)
			}
			require.ErrorIs(t, tt.call(s), tt.want)
		})
	}
}

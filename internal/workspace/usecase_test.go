package workspace

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

var wsFixedNow = time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)

type fakeOwner struct {
	mu          sync.Mutex
	allowCreate bool
	err         error
}

func (f *fakeOwner) CanCreateWorkspace(_ context.Context, _ string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.allowCreate, f.err
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
	moveErr   error
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
	for _, existing := range f.repos[projectID] {
		if existing.Owner == r.Owner && existing.Name == r.Name {
			return apperrs.ErrConflict
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
	if f.repoErr != nil {
		return RepoRef{}, f.repoErr
	}
	for _, repos := range f.repos {
		for _, r := range repos {
			if r.Owner == owner && r.Name == name {
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

func newTestService(repo *fakeRepo, owner InstanceAdminGate) *Service {
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

func (f *fakeCategoryRepo) SetTicketCategory(_ context.Context, ticketID, _ string, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.moveErr != nil {
		return f.moveErr
	}
	if ticketID == "missing" {
		return apperrs.ErrNotFound
	}
	return nil
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

func (f *fakeStatusRepo) Update(_ context.Context, s *Status, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
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

func newOwnerRepo(t *testing.T, allowCreate bool) (*Service, *fakeRepo, *fakeOwner) {
	t.Helper()
	repo := newFakeRepo()
	owner := &fakeOwner{allowCreate: allowCreate}
	return newTestService(repo, owner), repo, owner
}

func TestCreate(t *testing.T) {
	t.Run("empty workspace id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.Create(context.Background(), "u-1", "  ", "Backend", "BE", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unknown workspace is not found", func(t *testing.T) {
		repo := newFakeRepo()
		owner := &fakeOwner{allowCreate: true}
		s := NewService(repo, newFakeCategoryRepo(), newFakeTicketTypeRepo(), newFakeStatusRepo(), owner, &fakeWorkspaceGate{exists: false})
		_, err := s.Create(context.Background(), "u-1", "ws-1", "Backend", "BE", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("workspace gate error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		owner := &fakeOwner{allowCreate: true}
		wsGate := &fakeWorkspaceGate{err: errors.New("db down")}
		s := NewService(repo, newFakeCategoryRepo(), newFakeTicketTypeRepo(), newFakeStatusRepo(), owner, wsGate)
		_, err := s.Create(context.Background(), "u-1", "ws-1", "Backend", "BE", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, wsGate.err)
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.Create(context.Background(), "u-1", "ws-1", "   ", "BE", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		_, err := s.Create(context.Background(), "u-1", "ws-1", "Backend", "BE", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("owner gate error propagates", func(t *testing.T) {
		s, _, owner := newOwnerRepo(t, true)
		owner.err = errors.New("db down")
		_, err := s.Create(context.Background(), "u-1", "ws-1", "Backend", "BE", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, owner.err)
	})
	t.Run("repo error propagates", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.createErr = errors.New("db down")
		_, err := s.Create(context.Background(), "u-1", "ws-1", "Backend", "BE", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.createErr)
	})
	t.Run("creates with the next position", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", Prefix: "BE", Position: 0, WorkspaceID: "ws-1"}
		p, err := s.Create(context.Background(), "u-1", "ws-1", "Frontend", "FE", "")
		require.NoError(t, err)
		assert.Equal(t, "Frontend", p.Name)
		assert.Equal(t, "ws-1", p.WorkspaceID)
		assert.Equal(t, 1, p.Position)
		assert.Equal(t, wsFixedNow, p.CreatedAt)
	})
	t.Run("scopes the next position and prefix uniqueness to the workspace", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", Prefix: "BE", Position: 0, WorkspaceID: "ws-other"}
		p, err := s.Create(context.Background(), "u-1", "ws-1", "Frontend", "BE", "")
		require.NoError(t, err)
		assert.Equal(t, 0, p.Position)
	})
	t.Run("lowercases and trims are normalized to uppercase", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		p, err := s.Create(context.Background(), "u-1", "ws-1", "Frontend", " fe ", "")
		require.NoError(t, err)
		assert.Equal(t, "FE", p.Prefix)
	})
	t.Run("rejects a prefix shorter than 2 letters", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.Create(context.Background(), "u-1", "ws-1", "Frontend", "F", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("rejects a prefix longer than 5 letters", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.Create(context.Background(), "u-1", "ws-1", "Frontend", "FRONTS", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("rejects a prefix with non-letters", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.Create(context.Background(), "u-1", "ws-1", "Frontend", "FE2", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("rejects a prefix already used by another project", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", Prefix: "BE", WorkspaceID: "ws-1"}
		_, err := s.Create(context.Background(), "u-1", "ws-1", "Frontend", "be", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("invalid icon is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.Create(context.Background(), "u-1", "ws-1", "Frontend", "FE", ProjectIcon("bogus"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty icon is valid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		p, err := s.Create(context.Background(), "u-1", "ws-1", "Frontend", "FE", "")
		require.NoError(t, err)
		assert.Equal(t, ProjectIcon(""), p.Icon)
	})
	t.Run("icon from suggested list is valid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		p, err := s.Create(context.Background(), "u-1", "ws-1", "Frontend", "FE", ProjectIconRocket)
		require.NoError(t, err)
		assert.Equal(t, ProjectIconRocket, p.Icon)
	})
}

func TestGet(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.Get(context.Background(), "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing project is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.Get(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.getErr = errors.New("db down")
		_, err := s.Get(context.Background(), "p-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.getErr)
	})
}

func TestList(t *testing.T) {
	t.Run("empty workspace id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.List(context.Background(), "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.listErr = errors.New("db down")
		_, err := s.List(context.Background(), "ws-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.listErr)
	})
	t.Run("lists a workspace's projects", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", WorkspaceID: "ws-1"}
		repo.projects["p-2"] = &Project{ID: "p-2", Name: "Frontend", WorkspaceID: "ws-1"}
		repo.projects["p-3"] = &Project{ID: "p-3", Name: "Other", WorkspaceID: "ws-other"}
		projects, err := s.List(context.Background(), "ws-1")
		require.NoError(t, err)
		require.Len(t, projects, 2)
	})
}

func TestRename(t *testing.T) {
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		_, err := s.Rename(context.Background(), "u-1", "p-1", "New name", nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.Rename(context.Background(), "u-1", "p-1", " ", nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing project is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.Rename(context.Background(), "u-1", "nope", "New name", nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("renames a project", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Old", Position: 0}
		p, err := s.Rename(context.Background(), "u-1", "p-1", "New name", nil)
		require.NoError(t, err)
		assert.Equal(t, "New name", p.Name)
		assert.Equal(t, wsFixedNow, p.UpdatedAt)
	})
	t.Run("invalid icon is invalid", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Old"}
		bogus := ProjectIcon("bogus")
		_, err := s.Rename(context.Background(), "u-1", "p-1", "New name", &bogus)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("sets and clears icon", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Old", Icon: ProjectIconRocket}
		globe := ProjectIconGlobe
		p, err := s.Rename(context.Background(), "u-1", "p-1", "New name", &globe)
		require.NoError(t, err)
		assert.Equal(t, ProjectIconGlobe, p.Icon)

		empty := ProjectIcon("")
		p, err = s.Rename(context.Background(), "u-1", "p-1", "New name", &empty)
		require.NoError(t, err)
		assert.Equal(t, ProjectIcon(""), p.Icon)
	})
	t.Run("nil icon keeps the current icon", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Old", Icon: ProjectIconRocket}
		p, err := s.Rename(context.Background(), "u-1", "p-1", "New name", nil)
		require.NoError(t, err)
		assert.Equal(t, ProjectIconRocket, p.Icon)
	})
}

func TestSetPrefix(t *testing.T) {
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		_, err := s.SetPrefix(context.Background(), "u-1", "p-1", "GEN")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("invalid format prefix is rejected", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "General", WorkspaceID: "ws-1"}
		_, err := s.SetPrefix(context.Background(), "u-1", "p-1", "gen1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing project is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.SetPrefix(context.Background(), "u-1", "nope", "GEN")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("setting the same prefix again is a no-op instead of a conflict", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "General", WorkspaceID: "ws-1", Prefix: "GEN"}
		p, err := s.SetPrefix(context.Background(), "u-1", "p-1", "gen")
		require.NoError(t, err)
		assert.Equal(t, "GEN", p.Prefix)
	})
	t.Run("sets the prefix on a project with an empty prefix", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "General", WorkspaceID: "ws-1"}
		p, err := s.SetPrefix(context.Background(), "u-1", "p-1", "gen")
		require.NoError(t, err)
		assert.Equal(t, "GEN", p.Prefix)
		assert.Equal(t, wsFixedNow, p.UpdatedAt)
		assert.Equal(t, "GEN", repo.projects["p-1"].Prefix)
	})
	t.Run("already-set prefix is refused", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", Prefix: "BE", WorkspaceID: "ws-1"}
		_, err := s.SetPrefix(context.Background(), "u-1", "p-1", "BEX")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
		assert.Equal(t, "BE", repo.projects["p-1"].Prefix)
	})
	t.Run("rejects a prefix already used by another project in the same workspace", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "General", WorkspaceID: "ws-1"}
		repo.projects["p-2"] = &Project{ID: "p-2", Name: "Backend", Prefix: "BE", WorkspaceID: "ws-1"}
		_, err := s.SetPrefix(context.Background(), "u-1", "p-1", "be")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("duplicate prefix in another workspace is allowed", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "General", WorkspaceID: "ws-1"}
		repo.projects["p-2"] = &Project{ID: "p-2", Name: "Other", Prefix: "GEN", WorkspaceID: "ws-other"}
		p, err := s.SetPrefix(context.Background(), "u-1", "p-1", "GEN")
		require.NoError(t, err)
		assert.Equal(t, "GEN", p.Prefix)
	})
}

func TestReorder(t *testing.T) {
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		err := s.Reorder(context.Background(), "u-1", "ws-1", []string{"p-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("empty workspace id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.Reorder(context.Background(), "u-1", "  ", []string{"p-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
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
			err := s.Reorder(context.Background(), "u-1", "ws-1", tt.ids)
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
		err := s.Reorder(context.Background(), "u-1", "ws-1", []string{"p-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("reorders positions", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A", WorkspaceID: "ws-1"}
		repo.projects["p-2"] = &Project{ID: "p-2", Name: "B", WorkspaceID: "ws-1"}
		require.NoError(t, s.Reorder(context.Background(), "u-1", "ws-1", []string{"p-2", "p-1"}))
		assert.Equal(t, 0, repo.projects["p-2"].Position)
		assert.Equal(t, 1, repo.projects["p-1"].Position)
	})
}

func TestDeleteImpact(t *testing.T) {
	t.Run("missing project is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.DeleteImpact(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("counts tickets, repos and services", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.ticketPro["t-1"] = "p-1"
		repo.ticketPro["t-2"] = "p-1"
		repo.repos["p-1"] = []RepoRef{{Owner: "acme", Name: "app"}}
		repo.services["p-1"] = []string{"svc-1"}
		impact, err := s.DeleteImpact(context.Background(), "p-1")
		require.NoError(t, err)
		assert.Equal(t, DeleteImpact{Tickets: 2, Repos: 1, Services: 1}, impact)
	})
	t.Run("count error propagates", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.countErr = errors.New("db down")
		_, err := s.DeleteImpact(context.Background(), "p-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.countErr)
	})
}

func TestDelete(t *testing.T) {
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		err := s.Delete(context.Background(), "u-1", "p-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("missing project is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.Delete(context.Background(), "u-1", "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("project with tickets is refused", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.ticketPro["t-1"] = "p-1"
		err := s.Delete(context.Background(), "u-1", "p-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})
	t.Run("project with repos is refused", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.repos["p-1"] = []RepoRef{{Owner: "acme", Name: "app"}}
		err := s.Delete(context.Background(), "u-1", "p-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})
	t.Run("project with services is refused", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.services["p-1"] = []string{"svc-1"}
		err := s.Delete(context.Background(), "u-1", "p-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
		assert.Contains(t, err.Error(), "service")
	})
	t.Run("empty project deletes", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		require.NoError(t, s.Delete(context.Background(), "u-1", "p-1"))
		_, err := s.Get(context.Background(), "p-1")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestAddRepo(t *testing.T) {
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		err := s.AddRepo(context.Background(), "u-1", "p-1", "acme", "app", "", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("empty owner or name is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.AddRepo(context.Background(), "u-1", "p-1", "  ", "app", "", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing project is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.AddRepo(context.Background(), "u-1", "nope", "acme", "app", "", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.repoErr = errors.New("db down")
		err := s.AddRepo(context.Background(), "u-1", "p-1", "acme", "app", "", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.repoErr)
	})
	t.Run("associates a repo with a project", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		require.NoError(t, s.AddRepo(context.Background(), "u-1", "p-1", "acme", "app", "", ""))
		repos, err := s.ListRepos(context.Background(), "p-1")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "acme/app", repos[0].FullName)
	})
	t.Run("empty connector id defaults to github", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		require.NoError(t, s.AddRepo(context.Background(), "u-1", "p-1", "acme", "app", "  ", ""))
		repos, err := s.ListRepos(context.Background(), "p-1")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "github", repos[0].ConnectorID)
	})
	t.Run("persists the given connector id", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		require.NoError(t, s.AddRepo(context.Background(), "u-1", "p-1", "acme", "app", "gitlab-self-hosted", ""))
		repos, err := s.ListRepos(context.Background(), "p-1")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "gitlab-self-hosted", repos[0].ConnectorID)
	})
}

func TestRemoveRepo(t *testing.T) {
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		err := s.RemoveRepo(context.Background(), "u-1", "acme", "app")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("empty owner or name is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.RemoveRepo(context.Background(), "u-1", "", "app")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing repo is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.RemoveRepo(context.Background(), "u-1", "acme", "app")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("dissociates a repo", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.repos["p-1"] = []RepoRef{{Owner: "acme", Name: "app"}}
		require.NoError(t, s.RemoveRepo(context.Background(), "u-1", "acme", "app"))
		repos, err := s.ListRepos(context.Background(), "p-1")
		require.NoError(t, err)
		assert.Empty(t, repos)
	})
}

func TestListRepos(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.ListRepos(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
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
	t.Run("empty ticket id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.MoveTicket(context.Background(), " ", "p-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing project is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.MoveTicket(context.Background(), "t-1", "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		err := s.MoveTicket(context.Background(), "nope", "p-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("moves a ticket into a project", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.ticketPro["t-1"] = "old"
		require.NoError(t, s.MoveTicket(context.Background(), "t-1", "p-1"))
		assert.Equal(t, "p-1", repo.ticketPro["t-1"])
	})
}

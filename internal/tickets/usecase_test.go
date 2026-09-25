package tickets

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

var fixedNow = time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)

// fakeRepo is an in-memory tickets.Repo for use-case tests.
type fakeRepo struct {
	mu            sync.Mutex
	tickets       map[string]*Ticket
	prLinks       map[string][]PRLink
	branchLinks   map[string][]BranchLink
	labelColors   map[string]map[string]colors.Color
	events        []eventbus.OutboxEvent
	createErr     error
	getErr        error
	listErr       error
	statusErr     error
	deleteErr     error
	searchErr     error
	linkErr       error
	prLinksErr    error
	positionErr   error
	labelColorErr error
	updateErr     error
	ticketLinks   []TicketLink
	ticketLinkErr error
	doneStatuses  map[Status]bool
	prefixes      map[string]string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{tickets: map[string]*Ticket{}, prLinks: map[string][]PRLink{}, branchLinks: map[string][]BranchLink{}, labelColors: map[string]map[string]colors.Color{}}
}

func (f *fakeRepo) Create(_ context.Context, t *Ticket, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if _, ok := f.tickets[t.ID]; ok {
		return apperrs.ErrConflict
	}
	t.Position = f.nextPositionLocked(t.Status, t.CategoryID)
	f.tickets[t.ID] = t
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id string) (*Ticket, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	t, ok := f.tickets[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return t, nil
}

// GetByPrefixAndNumber reads the project prefix from prefixes, since tickets alone never store it.
func (f *fakeRepo) GetByPrefixAndNumber(_ context.Context, prefix string, number int) (*Ticket, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	for _, t := range f.tickets {
		if f.prefixes[t.ProjectID] == prefix && t.Number == number {
			return t, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) List(_ context.Context) ([]*Ticket, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*Ticket, 0, len(f.tickets))
	for _, t := range f.tickets {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeRepo) ListByDoc(_ context.Context, docID string) ([]*Ticket, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Ticket
	for _, t := range f.tickets {
		if t.DocID == docID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListByProject(_ context.Context, projectID string) ([]*Ticket, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Ticket
	for _, t := range f.tickets {
		if t.ProjectID == projectID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, id string, status Status, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.statusErr != nil {
		return f.statusErr
	}
	t, ok := f.tickets[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	// Mirrors the real repo: position is computed against the ticket's still-current old status/category, so the moved row never counts toward its own new pair's max.
	t.Position = f.nextPositionLocked(status, t.CategoryID)
	t.Status = status
	f.events = append(f.events, evts...)
	return nil
}

// nextPositionLocked mirrors the real repo's append-to-end-of-pair semantics (ADR 0002); callers must hold f.mu.
func (f *fakeRepo) nextPositionLocked(status Status, categoryID string) int {
	max, found := -1, false
	for _, t := range f.tickets {
		if t.Status == status && t.CategoryID == categoryID {
			found = true
			if t.Position > max {
				max = t.Position
			}
		}
	}
	if !found {
		return 0
	}
	return max + 1
}

func (f *fakeRepo) Delete(_ context.Context, id string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.tickets[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.tickets, id)
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
	for _, t := range f.tickets {
		if strings.Contains(strings.ToLower(t.Title), strings.ToLower(query)) || strings.Contains(strings.ToLower(t.Body), strings.ToLower(query)) {
			out = append(out, SearchResult{ID: t.ID, Title: t.Title})
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeRepo) LinkPR(_ context.Context, id string, ref PRRef, state PRState) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.linkErr != nil {
		return f.linkErr
	}
	if _, ok := f.tickets[id]; !ok {
		return apperrs.ErrNotFound
	}
	for i, existing := range f.prLinks[id] {
		if existing.Owner == ref.Owner && existing.Repo == ref.Repo && existing.Number == ref.Number {
			f.prLinks[id][i].State = state
			return nil
		}
	}
	f.prLinks[id] = append(f.prLinks[id], PRLink{PRRef: ref, State: state})
	return nil
}

func (f *fakeRepo) ListPRLinks(_ context.Context, id string) ([]PRLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.prLinksErr != nil {
		return nil, f.prLinksErr
	}
	return append([]PRLink(nil), f.prLinks[id]...), nil
}

func (f *fakeRepo) ListPRLinksBatch(_ context.Context, ids []string) (map[string][]PRLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.prLinksErr != nil {
		return nil, f.prLinksErr
	}
	out := make(map[string][]PRLink, len(ids))
	for _, id := range ids {
		out[id] = append([]PRLink(nil), f.prLinks[id]...)
	}
	return out, nil
}

func (f *fakeRepo) MarkPRState(_ context.Context, owner, repo string, number int, state PRState) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var affected []string
	for id, links := range f.prLinks {
		for i := range links {
			if links[i].Owner == owner && links[i].Repo == repo && links[i].Number == number {
				f.prLinks[id][i].State = state
				if !contains(affected, id) {
					affected = append(affected, id)
				}
			}
		}
	}
	return affected, nil
}

func (f *fakeRepo) ListIDsByPR(_ context.Context, owner, repo string, number int) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.prLinksErr != nil {
		return nil, f.prLinksErr
	}
	var ids []string
	for id, links := range f.prLinks {
		for _, l := range links {
			if l.Owner == owner && l.Repo == repo && l.Number == number && !contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (f *fakeRepo) SetFinishedAt(_ context.Context, id string, at time.Time, evts ...eventbus.OutboxEvent) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tickets[id]
	if !ok {
		return false, apperrs.ErrNotFound
	}
	if t.FinishedAt != nil {
		return false, nil
	}
	ft := at
	t.FinishedAt = &ft
	f.events = append(f.events, evts...)
	return true, nil
}

func (f *fakeRepo) LinkBranch(_ context.Context, id string, link BranchLink) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.linkErr != nil {
		return f.linkErr
	}
	if _, ok := f.tickets[id]; !ok {
		return apperrs.ErrNotFound
	}
	for _, existing := range f.branchLinks[id] {
		if existing == link {
			return nil
		}
	}
	f.branchLinks[id] = append(f.branchLinks[id], link)
	return nil
}

func (f *fakeRepo) ListBranchLinks(_ context.Context, id string) ([]BranchLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]BranchLink(nil), f.branchLinks[id]...), nil
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
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
	s := NewService(repo, fakeStatusStore{}, nil)
	s.now = func() time.Time { return fixedNow }
	return s
}

// fakeStatusStore accepts the seeded status columns; a non-existent status is rejected.
type fakeStatusStore struct {
	known map[Status]bool
	err   error
}

func (f fakeStatusStore) Exists(_ context.Context, id string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	if len(f.known) == 0 {
		return id == string(StatusOpen) || id == string(StatusInProgress) || id == string(StatusDone) || id == string(StatusClosed), nil
	}
	return f.known[Status(id)], nil
}

func (f *fakeRepo) UpdateType(_ context.Context, id, typeID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tickets[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	t.TypeID = typeID
	return nil
}

func (f *fakeRepo) UpdatePerson(_ context.Context, id string, role Role, login string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tickets[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	if role == RoleTester {
		t.Tester = login
	}
	if role == RoleDeveloper {
		t.Developer = login
	}
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) UpdateTicket(_ context.Context, id, title, body string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	t, ok := f.tickets[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	t.Title = title
	t.Body = body
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) SetPosition(_ context.Context, id string, position int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.positionErr != nil {
		return f.positionErr
	}
	t, ok := f.tickets[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	t.Position = position
	return nil
}

func (f *fakeRepo) AddLabel(_ context.Context, id, label string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tickets[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	for _, l := range t.Labels {
		if l == label {
			return nil
		}
	}
	t.Labels = append(t.Labels, label)
	return nil
}

func (f *fakeRepo) RemoveLabel(_ context.Context, id, label string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tickets[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	out := t.Labels[:0]
	for _, l := range t.Labels {
		if l != label {
			out = append(out, l)
		}
	}
	t.Labels = out
	return nil
}

func (f *fakeRepo) ListLabels(_ context.Context, id string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tickets[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return append([]string(nil), t.Labels...), nil
}

func (f *fakeRepo) ListAllLabels(_ context.Context) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	seen := map[string]bool{}
	var out []string
	for _, t := range f.tickets {
		for _, l := range t.Labels {
			if !seen[l] {
				seen[l] = true
				out = append(out, l)
			}
		}
	}
	return out, nil
}

func (f *fakeRepo) SetLabelColor(_ context.Context, projectID, label string, color colors.Color) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.labelColorErr != nil {
		return f.labelColorErr
	}
	if f.labelColors[projectID] == nil {
		f.labelColors[projectID] = map[string]colors.Color{}
	}
	f.labelColors[projectID][label] = color
	return nil
}

func (f *fakeRepo) LabelColors(_ context.Context, projectID string, labels []string) (map[string]colors.Color, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.labelColorErr != nil {
		return nil, f.labelColorErr
	}
	out := make(map[string]colors.Color, len(labels))
	for _, l := range labels {
		if c, ok := f.labelColors[projectID][l]; ok {
			out[l] = c
		}
	}
	return out, nil
}

func validPRRef() PRRef {
	return PRRef{Owner: "acme", Repo: "app", Number: 42, Title: "Fix login", SHA: "abc123"}
}

func TestCreate(t *testing.T) {
	t.Run("empty title is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Create(context.Background(), "p-1", "  ", "body", "", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.createErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.Create(context.Background(), "p-1", "title", "body", "", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.createErr)
	})
	t.Run("creates open ticket linked to doc and enqueues ticket.created", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		tk, err := s.Create(context.Background(), "p-1", "Fix storage", "write migrations", "doc-1", "onik97")
		require.NoError(t, err)
		assert.Equal(t, StatusOpen, tk.Status)
		assert.Equal(t, "doc-1", tk.DocID)
		assert.Equal(t, fixedNow, tk.CreatedAt)
		assert.NotEmpty(t, repo.eventsFor(TopicCreated))
	})
	t.Run("blank doc_id is stored empty", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		tk, err := s.Create(context.Background(), "p-1", "Standalone", "body", "   ", "")
		require.NoError(t, err)
		assert.Equal(t, "", tk.DocID)
	})
	t.Run("post-create fetch error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.getErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.Create(context.Background(), "p-1", "title", "body", "", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.getErr)
	})
	t.Run("a second ticket in the same (open, category) pair appends rather than colliding", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		first, err := s.Create(context.Background(), "p-1", "First", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, 0, first.Position, "first ticket in an empty pair")

		second, err := s.Create(context.Background(), "p-1", "Second", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, 1, second.Position, "appended after the first, not defaulted to 0")
	})
}

func TestGet(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Get(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Get(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.getErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.Get(context.Background(), "t-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.getErr)
	})
}

func TestResolve(t *testing.T) {
	repo := newFakeRepo()
	repo.prefixes = map[string]string{"p-1": "REF"}
	repo.tickets["t-1"] = &Ticket{ID: "t-1", ProjectID: "p-1", Number: 102}
	s := newTestService(repo)
	tests := []struct {
		name    string
		in      string
		wantErr error
	}{
		{"empty is invalid", " ", apperrs.ErrInvalid},
		{"a key past the int range is invalid", "REF-99999999999999999999", apperrs.ErrInvalid},
		{"a missing key is not found", "REF-7", apperrs.ErrNotFound},
		{"a missing id is not found", "nope", apperrs.ErrNotFound},
		{"by id", "t-1", nil},
		{"by key", "REF-102", nil},
		{"by a lowercase key", " ref-102 ", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.Resolve(t.Context(), tt.in)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "t-1", got.ID)
		})
	}
}

func TestListAndListByDoc(t *testing.T) {
	t.Run("empty doc id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.ListByDoc(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("lists and scopes by doc", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		_, err := s.Create(context.Background(), "p-1", "A", "", "doc-1", "")
		require.NoError(t, err)
		_, err = s.Create(context.Background(), "p-1", "B", "", "doc-1", "")
		require.NoError(t, err)
		_, err = s.Create(context.Background(), "p-1", "C", "", "doc-2", "")
		require.NoError(t, err)

		all, err := s.List(context.Background())
		require.NoError(t, err)
		assert.Len(t, all, 3)

		forDoc, err := s.ListByDoc(context.Background(), "doc-1")
		require.NoError(t, err)
		assert.Len(t, forDoc, 2)
	})
}

func TestUpdateStatus(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.UpdateStatus(context.Background(), " ", StatusDone)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.UpdateStatus(context.Background(), "nope", StatusDone)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("unconfigured status is rejected", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		// Statuses are configurable columns: any configured status is a valid target, a status that does not exist is rejected.
		_, err = s.UpdateStatus(context.Background(), created.ID, Status("nope"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		repo.statusErr = errors.New("db down")
		_, err = s.UpdateStatus(context.Background(), created.ID, StatusDone)
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.statusErr)
	})
	t.Run("same status is a no-op with no event", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		got, err := s.UpdateStatus(context.Background(), created.ID, StatusOpen)
		require.NoError(t, err)
		assert.Equal(t, StatusOpen, got.Status)
		assert.Empty(t, repo.eventsFor(TopicStatusChanged))
	})
	t.Run("valid transition persists and enqueues ticket.status_changed", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)

		got, err := s.UpdateStatus(context.Background(), created.ID, StatusInProgress)
		require.NoError(t, err)
		assert.Equal(t, StatusInProgress, got.Status)

		evts := repo.eventsFor(TopicStatusChanged)
		require.Len(t, evts, 1)
	})
	t.Run("moving into an occupied pair appends after the existing ticket, reflected in the returned ticket", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		already, err := s.Create(context.Background(), "p-1", "already in_progress", "", "", "")
		require.NoError(t, err)
		_, err = s.UpdateStatus(context.Background(), already.ID, StatusInProgress)
		require.NoError(t, err)

		moving, err := s.Create(context.Background(), "p-1", "moving", "", "", "")
		require.NoError(t, err)

		got, err := s.UpdateStatus(context.Background(), moving.ID, StatusInProgress)
		require.NoError(t, err)
		assert.Equal(t, 1, got.Position, "returned ticket is re-fetched, so it carries the persisted position, not the pre-write snapshot")

		persisted, err := s.Get(context.Background(), moving.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, persisted.Position)
	})
}

func TestSetPosition(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.SetPosition(context.Background(), " ", 0)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("negative position is invalid", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		_, err = s.SetPosition(context.Background(), created.ID, -1)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.SetPosition(context.Background(), "nope", 1)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		repo.positionErr = errors.New("db down")
		_, err = s.SetPosition(context.Background(), created.ID, 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.positionErr)
	})
	t.Run("sets position", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		got, err := s.SetPosition(context.Background(), created.ID, 3)
		require.NoError(t, err)
		assert.Equal(t, 3, got.Position)
	})
}

func TestUpdateTicket(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.UpdateTicket(context.Background(), " ", "New title", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty title is invalid", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "old body", "", "")
		require.NoError(t, err)
		_, err = s.UpdateTicket(context.Background(), created.ID, " ", "new body")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.UpdateTicket(context.Background(), "nope", "New title", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		repo.updateErr = errors.New("db down")
		_, err = s.UpdateTicket(context.Background(), created.ID, "New title", "New body")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.updateErr)
	})
	t.Run("updates title and body", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "old body", "", "")
		require.NoError(t, err)
		got, err := s.UpdateTicket(context.Background(), created.ID, "New title", "New body")
		require.NoError(t, err)
		assert.Equal(t, "New title", got.Title)
		assert.Equal(t, "New body", got.Body)
	})
	t.Run("enqueues ticket.updated with the post-edit ticket", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "old body", "", "")
		require.NoError(t, err)
		_, err = s.UpdateTicket(context.Background(), created.ID, "New title", "New body")
		require.NoError(t, err)
		evts := repo.eventsFor(TopicUpdated)
		require.Len(t, evts, 1)
		payload, ok := evts[0].Payload.(UpdatedEvent)
		require.True(t, ok)
		assert.Equal(t, "New title", payload.Ticket.Title)
		assert.Equal(t, "New body", payload.Ticket.Body)
	})
}

func TestDelete(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.Delete(context.Background(), "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.Delete(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		repo.deleteErr = errors.New("db down")
		err = s.Delete(context.Background(), created.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.deleteErr)
	})
	t.Run("removes ticket", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		require.NoError(t, s.Delete(context.Background(), created.ID))
		_, err = s.Get(context.Background(), created.ID)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("publishes ticket.deleted", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		require.NoError(t, s.Delete(context.Background(), created.ID))
		evts := repo.eventsFor(TopicDeleted)
		require.Len(t, evts, 1)
		payload, ok := evts[0].Payload.(DeletedEvent)
		require.True(t, ok)
		assert.Equal(t, created.ID, payload.ID)
		assert.Equal(t, "ticket", payload.Title)
	})
}

func TestSearch(t *testing.T) {
	t.Run("empty query is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.Search(context.Background(), "  ", 10)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.searchErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.Search(context.Background(), "sqlite", 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.searchErr)
	})
	t.Run("finds matching tickets", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		_, err := s.Create(context.Background(), "p-1", "Fix storage", "migrations", "", "")
		require.NoError(t, err)
		results, err := s.Search(context.Background(), "migrations", 0)
		require.NoError(t, err)
		require.Len(t, results, 1)
	})
}

func TestLinkPR(t *testing.T) {
	t.Run("empty ticket id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.LinkPR(context.Background(), "", validPRRef())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("incomplete pr ref is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.LinkPR(context.Background(), "t-1", PRRef{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.LinkPR(context.Background(), "nope", validPRRef())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("links a pr", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		require.NoError(t, s.LinkPR(context.Background(), created.ID, validPRRef()))
		links, err := repo.ListPRLinks(context.Background(), created.ID)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, 42, links[0].Number)
	})
}

func TestDevStatus(t *testing.T) {
	t.Run("counts open and merged prs per ticket", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		a, err := s.Create(context.Background(), "p-1", "A", "", "", "")
		require.NoError(t, err)
		b, err := s.Create(context.Background(), "p-1", "B", "", "", "")
		require.NoError(t, err)
		require.NoError(t, s.LinkPR(context.Background(), a.ID, validPRRef()))
		require.NoError(t, s.LinkPR(context.Background(), a.ID, PRRef{Owner: "acme", Repo: "app", Number: 43}))
		_, err = repo.MarkPRState(context.Background(), "acme", "app", 43, PRStateMerged)
		require.NoError(t, err)

		got, err := s.DevStatus(context.Background(), []string{a.ID, b.ID})
		require.NoError(t, err)
		assert.Equal(t, DevStatusCounts{Open: 1, Merged: 1}, got[a.ID])
		assert.Equal(t, DevStatusCounts{}, got[b.ID])
	})
	t.Run("unknown ticket id gets zero counts, not an error", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		got, err := s.DevStatus(context.Background(), []string{"nope"})
		require.NoError(t, err)
		assert.Equal(t, DevStatusCounts{}, got["nope"])
	})
	t.Run("blank ids are skipped", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		got, err := s.DevStatus(context.Background(), []string{" ", ""})
		require.NoError(t, err)
		assert.Empty(t, got)
	})
	t.Run("repo error is passed through", func(t *testing.T) {
		repo := newFakeRepo()
		repo.prLinksErr = errors.New("boom")
		s := newTestService(repo)
		_, err := s.DevStatus(context.Background(), []string{"t-1"})
		require.Error(t, err)
	})
}

func TestCompletion(t *testing.T) {
	merged := func() eventbus.Event {
		return eventbus.Event{Payload: json.RawMessage(`{"owner":"acme","repo":"app","pr":{"number":42,"title":"Fix login","head_sha":"abc123","linked_ticket_ids":[]}}`)}
	}
	closed := func() eventbus.Event {
		return eventbus.Event{Payload: json.RawMessage(`{"owner":"acme","repo":"app","pr":{"number":42,"title":"Fix login","head_sha":"abc123","linked_ticket_ids":[]}}`)}
	}
	open := func() eventbus.Event {
		return eventbus.Event{Payload: json.RawMessage(`{"owner":"acme","repo":"app","pr":{"number":43,"title":"Second","head_sha":"x","linked_ticket_ids":[]}}`)}
	}

	t.Run("merged single pr publishes ticket.finished once", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		require.NoError(t, s.LinkPR(context.Background(), created.ID, validPRRef()))

		require.NoError(t, s.HandlePRMerged(context.Background(), merged()))
		require.Len(t, repo.eventsFor(TopicFinished), 1)
		got, err := s.Get(context.Background(), created.ID)
		require.NoError(t, err)
		require.NotNil(t, got.FinishedAt)

		// Redelivery and repeat merges stay exactly-once.
		require.NoError(t, s.HandlePRMerged(context.Background(), merged()))
		require.Len(t, repo.eventsFor(TopicFinished), 1)
	})

	t.Run("an open pr blocks completion", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		require.NoError(t, s.LinkPR(context.Background(), created.ID, validPRRef()))
		require.NoError(t, s.LinkPR(context.Background(), created.ID, PRRef{Owner: "acme", Repo: "app", Number: 43}))

		require.NoError(t, s.HandlePRMerged(context.Background(), merged()))
		require.Empty(t, repo.eventsFor(TopicFinished))

		// A closed-unmerged PR doesn't unblock completion since the still-open PR keeps it blocked.
		require.NoError(t, s.HandlePRClosed(context.Background(), closed()))
		require.Empty(t, repo.eventsFor(TopicFinished))
	})

	t.Run("all closed unmerged never finishes", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		require.NoError(t, s.LinkPR(context.Background(), created.ID, validPRRef()))

		require.NoError(t, s.HandlePRClosed(context.Background(), closed()))
		require.Empty(t, repo.eventsFor(TopicFinished))
		got, err := s.Get(context.Background(), created.ID)
		require.NoError(t, err)
		require.Nil(t, got.FinishedAt)
	})

	t.Run("second pr merging finishes an open-linked ticket", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		require.NoError(t, s.LinkPR(context.Background(), created.ID, validPRRef()))

		require.NoError(t, s.HandlePRMerged(context.Background(), open()))
		require.Empty(t, repo.eventsFor(TopicFinished))
		require.NoError(t, s.HandlePRMerged(context.Background(), merged()))
		require.Len(t, repo.eventsFor(TopicFinished), 1)
	})

	t.Run("malformed payload is fatal", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.HandlePRMerged(context.Background(), eventbus.Event{Payload: json.RawMessage("{")})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})

	t.Run("repo error is retryable", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		repo.linkErr = errors.New("db down")
		ev := eventbus.Event{Payload: json.RawMessage(`{"owner":"acme","repo":"app","pr":{"number":42,"linked_ticket_ids":["t-1"]}}`)}
		err := s.HandlePRMerged(context.Background(), ev)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
}

func TestUpdateStatus_AutomationTokenRequestRecordsAutomationActor(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
	require.NoError(t, err)

	ctx := identity.WithActor(context.Background(), identity.Actor{
		ID:         "owner",
		Automation: &identity.AutomationRef{ID: "auto-1", Name: "Ticket finished"},
	})
	_, err = s.UpdateStatus(ctx, created.ID, StatusDone)
	require.NoError(t, err)

	evts := repo.eventsFor(TopicStatusChanged)
	require.Len(t, evts, 1)
	e, ok := evts[0].Payload.(StatusChangedEvent)
	require.True(t, ok, "payload should be a StatusChangedEvent")
	assert.Equal(t, Actor{Kind: ActorKindAutomation, AutomationID: "auto-1", AutomationName: "Ticket finished"}, e.Actor)
}

func TestSetStatusAs(t *testing.T) {
	t.Run("records actor and run id on the event", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)

		actor := Actor{Kind: ActorKindAutomation, AutomationID: "auto-1", AutomationName: "default"}
		_, err = s.SetStatusAs(context.Background(), created.ID, StatusDone, actor, "run-1")
		require.NoError(t, err)

		evts := repo.eventsFor(TopicStatusChanged)
		require.Len(t, evts, 1)
		e, ok := evts[0].Payload.(StatusChangedEvent)
		require.True(t, ok, "payload should be a StatusChangedEvent")
		assert.Equal(t, actor, e.Actor)
		assert.Equal(t, "run-1", e.RunID)
		assert.Equal(t, StatusDone, e.To)
	})
	t.Run("records a play actor with its label and trail id", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)

		actor := Actor{Kind: ActorKindPlay + ":mcp", PlayLabel: "Fix with AI", TrailID: "trail-1"}
		_, err = s.SetStatusAs(context.Background(), created.ID, StatusDone, actor, "")
		require.NoError(t, err)

		evts := repo.eventsFor(TopicStatusChanged)
		require.Len(t, evts, 1)
		e := evts[0].Payload.(StatusChangedEvent)
		assert.Equal(t, actor, e.Actor)
		assert.Empty(t, e.RunID)
	})
	t.Run("domain invariants still apply", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		_, err = s.UpdateStatus(context.Background(), created.ID, StatusClosed)
		require.NoError(t, err)

		_, err = s.SetStatusAs(context.Background(), created.ID, Status("nope"), Actor{Kind: ActorKindAutomation}, "run-2")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestLinkBranch(t *testing.T) {
	t.Run("links a branch", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		require.NoError(t, s.LinkBranch(context.Background(), created.ID, "acme", "app", "ticket/1"))
		_, branches, err := s.ListLinks(context.Background(), created.ID)
		require.NoError(t, err)
		require.Len(t, branches, 1)
		assert.Equal(t, "ticket/1", branches[0].Branch)
	})
	t.Run("incomplete link is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.LinkBranch(context.Background(), "t-1", "", "app", "branch")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.LinkBranch(context.Background(), "nope", "acme", "app", "branch")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
		want bool
	}{
		{"open to in_progress", StatusOpen, StatusInProgress, true},
		{"open to done", StatusOpen, StatusDone, true},
		{"open to closed", StatusOpen, StatusClosed, true},
		{"closed to in_progress", StatusClosed, StatusInProgress, true},
		{"open to open is a no-op", StatusOpen, StatusOpen, true},
		{"empty target is not a transition", StatusOpen, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CanTransition(tt.from, tt.to))
		})
	}
}

func TestUpdateStatus_UserRequestRecordsTheMoversUserID(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
	require.NoError(t, err)

	_, err = s.UpdateStatus(identity.WithActor(context.Background(), identity.Actor{ID: "u-7"}), created.ID, StatusDone)
	require.NoError(t, err)

	evts := repo.eventsFor(TopicStatusChanged)
	require.Len(t, evts, 1)
	e, ok := evts[0].Payload.(StatusChangedEvent)
	require.True(t, ok)
	assert.Equal(t, Actor{Kind: ActorKindUser, UserID: "u-7"}, e.Actor)
}

func TestListByPR(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	a, err := s.Create(context.Background(), "p-1", "first", "", "", "")
	require.NoError(t, err)
	b, err := s.Create(context.Background(), "p-1", "second", "", "", "")
	require.NoError(t, err)
	ref := PRRef{Owner: "o", Repo: "r", Number: 7}
	require.NoError(t, s.LinkPR(context.Background(), a.ID, ref))
	require.NoError(t, s.LinkPR(context.Background(), b.ID, ref))
	require.NoError(t, s.LinkPR(context.Background(), b.ID, PRRef{Owner: "o", Repo: "r", Number: 8}))

	got, err := s.ListByPR(context.Background(), "o", "r", 7)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{a.ID, b.ID}, []string{got[0].ID, got[1].ID})

	none, err := s.ListByPR(context.Background(), "o", "r", 9)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestListByPR_Errors(t *testing.T) {
	t.Run("invalid input", func(t *testing.T) {
		_, err := newTestService(newFakeRepo()).ListByPR(context.Background(), "o", "", 1)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		_, err = newTestService(newFakeRepo()).ListByPR(context.Background(), "o", "r", 0)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("link lookup fails", func(t *testing.T) {
		repo := newFakeRepo()
		repo.prLinksErr = errors.New("disk")
		_, err := newTestService(repo).ListByPR(context.Background(), "o", "r", 1)
		require.Error(t, err)
	})
	t.Run("a linked ticket is gone", func(t *testing.T) {
		repo := newFakeRepo()
		repo.prLinks["t-gone"] = []PRLink{{PRRef: PRRef{Owner: "o", Repo: "r", Number: 1}}}
		_, err := newTestService(repo).ListByPR(context.Background(), "o", "r", 1)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

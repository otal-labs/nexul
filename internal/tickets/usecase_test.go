package tickets

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
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
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
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
	unblocked     UnblockedEvents
	// workspaces maps a project to its workspace's id and slug; unlisted projects sit in workspace-default.
	workspaces map[string][2]string
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

// ListByKey reads the project prefix from prefixes, since tickets alone never store it.
func (f *fakeRepo) ListByKey(_ context.Context, prefix string, number int) ([]KeyMatch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	var out []KeyMatch
	for _, t := range f.tickets {
		if f.prefixes[t.ProjectID] == prefix && t.Number == number {
			ws := cmp.Or(f.workspaces[t.ProjectID], [2]string{"workspace-default", "default"})
			out = append(out, KeyMatch{Ticket: t, WorkspaceID: ws[0], WorkspaceSlug: ws[1]})
		}
	}
	slices.SortFunc(out, func(a, b KeyMatch) int {
		return cmp.Or(cmp.Compare(a.WorkspaceSlug, b.WorkspaceSlug), cmp.Compare(a.Ticket.ID, b.Ticket.ID))
	})
	return out, nil
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

func (f *fakeRepo) Page(_ context.Context, filter TicketFilter, _ TicketScope, w paging.Window) ([]*Ticket, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Ticket
	for _, t := range f.tickets {
		if (filter.ProjectID == "" || t.ProjectID == filter.ProjectID) && (filter.DocID == "" || t.DocID == filter.DocID) {
			out = append(out, t)
		}
	}
	return out[min(w.Offset, len(out)):min(w.Offset+w.Limit, len(out))], len(out), nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, id string, status Status, unblocked UnblockedEvents, _ []Change, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unblocked = unblocked
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

func (f *fakeRepo) Delete(_ context.Context, id string, unblocked UnblockedEvents, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unblocked = unblocked
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

func (f *fakeRepo) SetFinishedAt(_ context.Context, id string, at time.Time, unblocked UnblockedEvents, _ []Change, evts ...eventbus.OutboxEvent) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unblocked = unblocked
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

func (f *fakeRepo) UnlinkPR(_ context.Context, id string, ref PRRef) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prLinks[id] = slices.DeleteFunc(f.prLinks[id], func(l PRLink) bool {
		return l.Owner == ref.Owner && l.Repo == ref.Repo && l.Number == ref.Number
	})
	return nil
}

func (f *fakeRepo) UnlinkBranch(_ context.Context, id string, link BranchLink) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.branchLinks[id] = slices.DeleteFunc(f.branchLinks[id], func(l BranchLink) bool { return l == link })
	return nil
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
	s.SetGate(allowGate{})
	s.now = func() time.Time { return fixedNow }
	return s
}

// allowGate lets every caller through, for tests about what happens past the permission check.
type allowGate struct{}

func (allowGate) RequireProject(context.Context, string, permissions.Action) error { return nil }

func (allowGate) Require(context.Context, string, permissions.Action) error { return nil }

func (allowGate) CallerProjectsIn(context.Context, string, permissions.Action) ([]string, bool, error) {
	return nil, true, nil
}

func (allowGate) CallerProjects(context.Context, permissions.Action) ([]string, bool, error) {
	return nil, true, nil
}

type fakePRStates struct {
	state PRState
	err   error
}

func (f fakePRStates) PRState(context.Context, string, string, int) (PRState, error) {
	return f.state, f.err
}

// fakeStatusStore accepts the seeded status columns; a non-existent status is rejected.
type fakeStatusStore struct {
	known    map[Status]bool
	err      error
	first    string
	firstErr error
}

func (f fakeStatusStore) Name(_ context.Context, id string) (string, error) { return id, nil }

func (f fakeStatusStore) FirstStatus(context.Context, string) (string, error) {
	return f.first, f.firstErr
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

func (f *fakeRepo) UpdateType(_ context.Context, id, typeID string, _ []Change) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tickets[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	t.TypeID = typeID
	return nil
}

func (f *fakeRepo) UpdatePerson(_ context.Context, id string, role Role, p Person, _ []Change, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tickets[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	if role == RoleTester {
		t.Tester, t.TesterID = p.Login, p.ID
	}
	if role == RoleDeveloper {
		t.Developer, t.DeveloperID = p.Login, p.ID
	}
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) UpdateTicket(_ context.Context, id, title, body string, _ []Change, _ *Credit, evts ...eventbus.OutboxEvent) error {
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

func (f *fakeRepo) UpdateDoc(_ context.Context, id, docID string, _ []Change, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	t, ok := f.tickets[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	t.DocID = docID
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

func (f *fakeRepo) AddLabel(_ context.Context, id, label string, _ []Change) error {
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

func (f *fakeRepo) RemoveLabel(_ context.Context, id, label string, _ []Change) error {
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

func (f *fakeRepo) ColoredLabels(_ context.Context, projectID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.labelColorErr != nil {
		return nil, f.labelColorErr
	}
	return slices.Collect(maps.Keys(f.labelColors[projectID])), nil
}

func (f *fakeRepo) DeleteUnusedLabel(_ context.Context, projectID, label string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.labelColorErr != nil {
		return false, f.labelColorErr
	}
	for _, t := range f.tickets {
		if t.ProjectID == projectID && slices.Contains(t.Labels, label) {
			return false, nil
		}
	}
	_, ok := f.labelColors[projectID][label]
	delete(f.labelColors[projectID], label)
	return ok, nil
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
	t.Run("first column lookup error propagates", func(t *testing.T) {
		boom := errors.New("db down")
		s := NewService(newFakeRepo(), fakeStatusStore{firstErr: boom}, nil)
		_, err := s.Create(context.Background(), "p-1", "title", "body", "", "")
		require.ErrorIs(t, err, boom)
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

func TestResolve(t *testing.T) {
	repo := newFakeRepo()
	repo.prefixes = map[string]string{"p-1": "REF", "p-2": "P1", "p-3": "1P", "p-4": "PHASE1", "p-nor": "WEB", "p-otal": "WEB"}
	repo.workspaces = map[string][2]string{"p-nor": {"ws-nor", "norwood"}, "p-otal": {"ws-otal", "otal"}}
	repo.tickets["t-nor"] = &Ticket{ID: "t-nor", ProjectID: "p-nor", Number: 1}
	repo.tickets["t-otal"] = &Ticket{ID: "t-otal", ProjectID: "p-otal", Number: 1}
	// A ticket moved into p-otal keeps its number, so WEB-2 names two tickets there.
	repo.tickets["t-otal-2"] = &Ticket{ID: "t-otal-2", ProjectID: "p-otal", Number: 2}
	repo.tickets["t-moved-in"] = &Ticket{ID: "t-moved-in", ProjectID: "p-otal", Number: 2}
	repo.tickets["t-1"] = &Ticket{ID: "t-1", ProjectID: "p-1", Number: 102}
	repo.tickets["t-2"] = &Ticket{ID: "t-2", ProjectID: "p-2", Number: 12}
	repo.tickets["t-3"] = &Ticket{ID: "t-3", ProjectID: "p-3", Number: 12}
	repo.tickets["t-4"] = &Ticket{ID: "t-4", ProjectID: "p-4", Number: 12}
	s := newTestService(repo)
	tests := []struct {
		name      string
		workspace string
		in        string
		wantID    string
		wantErr   error
		wantMsg   string
	}{
		{name: "empty is invalid", in: " ", wantErr: apperrs.ErrInvalid},
		{name: "a key past the int range is invalid", in: "REF-99999999999999999999", wantErr: apperrs.ErrInvalid},
		{name: "a missing key is not found", in: "REF-7", wantErr: apperrs.ErrNotFound},
		{name: "a missing id is not found", in: "nope", wantErr: apperrs.ErrNotFound},
		{name: "by id", in: "t-1", wantID: "t-1"},
		{name: "by key", in: "REF-102", wantID: "t-1"},
		{name: "by a lowercase key", in: " ref-102 ", wantID: "t-1"},
		{name: "by a key whose prefix has a digit", in: "P1-12", wantID: "t-2"},
		{name: "by a lowercase key whose prefix has a digit", in: "p1-12", wantID: "t-2"},
		{name: "a prefix starting with a digit is not a key", in: "1P-12", wantErr: apperrs.ErrNotFound},
		{name: "a six character prefix is not a key", in: "PHASE1-12", wantErr: apperrs.ErrNotFound},
		{name: "a key in two workspaces names both", in: "WEB-1", wantErr: apperrs.ErrConflict, wantMsg: "WEB-1 exists in norwood and otal; pass workspace"},
		{name: "a workspace slug picks one", workspace: "norwood", in: "web-1", wantID: "t-nor"},
		{name: "a workspace id picks one", workspace: "ws-otal", in: "WEB-1", wantID: "t-otal"},
		{name: "a workspace without the key is not found", workspace: "default", in: "WEB-1", wantErr: apperrs.ErrNotFound},
		{name: "a key twice in one workspace asks for the id", workspace: "otal", in: "WEB-2", wantErr: apperrs.ErrConflict, wantMsg: "WEB-2 matches more than one ticket in otal; use the ticket's id"},
		{name: "an id ignores the workspace", workspace: "norwood", in: "t-otal", wantID: "t-otal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.Resolve(t.Context(), tt.workspace, tt.in)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.ErrorContains(t, err, tt.wantMsg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, got.ID)
		})
	}
}

func TestListAndListByDoc(t *testing.T) {
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
	t.Run("unconfigured status is rejected", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		// Statuses are configurable columns: any configured status is a valid target, a status that does not exist is rejected.
		for _, to := range []Status{"nope", ""} {
			_, err = s.UpdateStatus(context.Background(), created.ID, to)
			require.ErrorIs(t, err, apperrs.ErrInvalid, "status %q", to)
		}
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

	linkedAfter := func(state PRState, lookupErr error) (*fakeRepo, *Service, string) {
		repo := newFakeRepo()
		s := newTestService(repo)
		s.SetPRStates(fakePRStates{state: state, err: lookupErr})
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		require.NoError(t, s.LinkPR(context.Background(), created.ID, validPRRef()))
		return repo, s, created.ID
	}

	t.Run("a pr linked after it merged finishes the ticket", func(t *testing.T) {
		repo, _, id := linkedAfter(PRStateMerged, nil)
		links, err := repo.ListPRLinks(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, PRStateMerged, links[0].State)
		require.Len(t, repo.eventsFor(TopicFinished), 1)
	})

	t.Run("a pr linked after it closed unmerged is recorded closed and never finishes", func(t *testing.T) {
		repo, _, id := linkedAfter(PRStateClosed, nil)
		links, err := repo.ListPRLinks(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, PRStateClosed, links[0].State)
		require.Empty(t, repo.eventsFor(TopicFinished))
	})

	t.Run("a failed state lookup links the pr as open", func(t *testing.T) {
		repo, _, id := linkedAfter(PRStateMerged, errors.New("github down"))
		links, err := repo.ListPRLinks(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, PRStateOpen, links[0].State)
		require.Empty(t, repo.eventsFor(TopicFinished))
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
}

func TestService_Refusals_ReturnTheirNamedError(t *testing.T) {
	ctx := context.Background()
	errDown := errors.New("db down")
	tests := []struct {
		name string
		seed func(repo *fakeRepo)
		call func(s *Service, id string) error
		want error
	}{
		{"create without a title", nil, func(s *Service, _ string) error { _, err := s.Create(ctx, "p-1", "  ", "body", "", ""); return err }, apperrs.ErrInvalid},
		{"create when the repo fails", func(r *fakeRepo) { r.createErr = errDown }, func(s *Service, _ string) error {
			_, err := s.Create(ctx, "p-1", "title", "body", "", "")
			return err
		}, errDown},
		{"create when the fetch after it fails", func(r *fakeRepo) { r.getErr = errDown }, func(s *Service, _ string) error {
			_, err := s.Create(ctx, "p-1", "title", "body", "", "")
			return err
		}, errDown},
		{"get without an id", nil, func(s *Service, _ string) error { _, err := s.Get(ctx, " "); return err }, apperrs.ErrInvalid},
		{"get a missing ticket", nil, func(s *Service, _ string) error { _, err := s.Get(ctx, "nope"); return err }, apperrs.ErrNotFound},
		{"get when the repo fails", func(r *fakeRepo) { r.getErr = errDown }, func(s *Service, id string) error { _, err := s.Get(ctx, id); return err }, errDown},
		{"list by a blank doc", nil, func(s *Service, _ string) error { _, err := s.ListByDoc(ctx, " "); return err }, apperrs.ErrInvalid},
		{"move status without an id", nil, func(s *Service, _ string) error { _, err := s.UpdateStatus(ctx, " ", StatusDone); return err }, apperrs.ErrInvalid},
		{"move status of a missing ticket", nil, func(s *Service, _ string) error { _, err := s.UpdateStatus(ctx, "nope", StatusDone); return err }, apperrs.ErrNotFound},
		{"move status when the repo fails", func(r *fakeRepo) { r.statusErr = errDown }, func(s *Service, id string) error { _, err := s.UpdateStatus(ctx, id, StatusDone); return err }, errDown},
		{"set a position without an id", nil, func(s *Service, _ string) error { _, err := s.SetPosition(ctx, " ", 0); return err }, apperrs.ErrInvalid},
		{"set a negative position", nil, func(s *Service, id string) error { _, err := s.SetPosition(ctx, id, -1); return err }, apperrs.ErrInvalid},
		{"set the position of a missing ticket", nil, func(s *Service, _ string) error { _, err := s.SetPosition(ctx, "nope", 1); return err }, apperrs.ErrNotFound},
		{"set a position when the repo fails", func(r *fakeRepo) { r.positionErr = errDown }, func(s *Service, id string) error { _, err := s.SetPosition(ctx, id, 1); return err }, errDown},
		{"edit without an id", nil, func(s *Service, _ string) error { _, err := s.UpdateTicket(ctx, " ", "New title", ""); return err }, apperrs.ErrInvalid},
		{"edit without a title", nil, func(s *Service, id string) error { _, err := s.UpdateTicket(ctx, id, " ", "new body"); return err }, apperrs.ErrInvalid},
		{"edit a missing ticket", nil, func(s *Service, _ string) error { _, err := s.UpdateTicket(ctx, "nope", "New title", ""); return err }, apperrs.ErrNotFound},
		{"edit when the repo fails", func(r *fakeRepo) { r.updateErr = errDown }, func(s *Service, id string) error {
			_, err := s.UpdateTicket(ctx, id, "New title", "New body")
			return err
		}, errDown},
		{"delete without an id", nil, func(s *Service, _ string) error { return s.Delete(ctx, "") }, apperrs.ErrInvalid},
		{"delete a missing ticket", nil, func(s *Service, _ string) error { return s.Delete(ctx, "nope") }, apperrs.ErrNotFound},
		{"delete when the repo fails", func(r *fakeRepo) { r.deleteErr = errDown }, func(s *Service, id string) error { return s.Delete(ctx, id) }, errDown},
		{"search for a blank query", nil, func(s *Service, _ string) error { _, err := s.Search(ctx, "  ", 10); return err }, apperrs.ErrInvalid},
		{"search when the repo fails", func(r *fakeRepo) { r.searchErr = errDown }, func(s *Service, _ string) error { _, err := s.Search(ctx, "sqlite", 10); return err }, errDown},
		{"link a pr without a ticket", nil, func(s *Service, _ string) error { return s.LinkPR(ctx, "", validPRRef()) }, apperrs.ErrInvalid},
		{"link an incomplete pr", nil, func(s *Service, _ string) error { return s.LinkPR(ctx, "t-1", PRRef{}) }, apperrs.ErrInvalid},
		{"link a pr to a missing ticket", nil, func(s *Service, _ string) error { return s.LinkPR(ctx, "nope", validPRRef()) }, apperrs.ErrNotFound},
		{"link an incomplete branch", nil, func(s *Service, _ string) error { return s.LinkBranch(ctx, "t-1", "", "app", "branch") }, apperrs.ErrInvalid},
		{"link a branch to a missing ticket", nil, func(s *Service, _ string) error { return s.LinkBranch(ctx, "nope", "acme", "app", "branch") }, apperrs.ErrNotFound},
		{"unlink a pr of a missing ticket", nil, func(s *Service, _ string) error { return s.UnlinkPR(ctx, "nope", validPRRef()) }, apperrs.ErrNotFound},
		{"unlink a branch of a missing ticket", nil, func(s *Service, _ string) error { return s.UnlinkBranch(ctx, "nope", "acme", "app", "b") }, apperrs.ErrNotFound},
		{"unlink an incomplete pr", nil, func(s *Service, id string) error { return s.UnlinkPR(ctx, id, PRRef{Owner: "acme"}) }, apperrs.ErrInvalid},
		{"unlink an incomplete branch", nil, func(s *Service, id string) error { return s.UnlinkBranch(ctx, id, "acme", "app", " ") }, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			s := newTestService(repo)
			created, err := s.Create(ctx, "p-1", "ticket", "", "", "")
			require.NoError(t, err)
			if tt.seed != nil {
				tt.seed(repo)
			}
			require.ErrorIs(t, tt.call(s, created.ID), tt.want)
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

func TestUpdateStatus_ThroughMCPPublishesUserMCPOnEveryEvent(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	ts := seedTickets(t, s, "blocked", "blocker")
	_, err := s.AddBlocker(t.Context(), ts[0].ID, ts[1].ID)
	require.NoError(t, err)
	want := Actor{Kind: ActorKindUserMCP, UserID: "u-7"}

	_, err = s.UpdateStatus(WithViaMCP(identity.WithActor(t.Context(), identity.Actor{ID: "u-7"})), ts[1].ID, StatusDone)
	require.NoError(t, err)

	evts := repo.eventsFor(TopicStatusChanged)
	require.Len(t, evts, 1)
	assert.Equal(t, want, evts[0].Payload.(StatusChangedEvent).Actor)
	unblocked := repo.unblocked([]Unblocked{{TicketID: ts[0].ID, ProjectID: ts[0].ProjectID}})
	require.Len(t, unblocked, 1)
	assert.Equal(t, want, unblocked[0].Payload.(UnblockedEvent).Actor)
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

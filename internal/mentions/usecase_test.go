package mentions

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// fakeTicketSource is an in-memory TicketSource for tests.
type fakeTicketSource struct {
	tickets map[string]Ticket
	// byKey maps "PREFIX-NUMBER" to a ticket, mirroring GetByPrefixAndNumber.
	byKey     map[string]Ticket
	search    []SearchHit
	searchErr error
}

func (f *fakeTicketSource) GetByID(_ context.Context, id string) (*Ticket, error) {
	t, ok := f.tickets[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &t, nil
}

func (f *fakeTicketSource) Search(_ context.Context, _ string, _ int) ([]SearchHit, error) {
	return f.search, f.searchErr
}

func (f *fakeTicketSource) GetByKey(_ context.Context, prefix string, number int) (*Ticket, error) {
	t, ok := f.byKey[fmt.Sprintf("%s-%d", prefix, number)]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &t, nil
}

// fakeDocSource is an in-memory DocSource for tests.
type fakeDocSource struct {
	docs      map[string]Doc
	search    []SearchHit
	searchErr error
}

func (f *fakeDocSource) GetByID(_ context.Context, id string) (*Doc, error) {
	d, ok := f.docs[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &d, nil
}

func (f *fakeDocSource) Search(_ context.Context, _ string, _ int) ([]SearchHit, error) {
	return f.search, f.searchErr
}

// fakeStatusSource is an in-memory StatusSource for tests.
type fakeStatusSource struct {
	statuses map[string]Status
}

func (f *fakeStatusSource) Get(_ context.Context, id string) (*Status, error) {
	s, ok := f.statuses[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &s, nil
}

// fakeProjectSource is an in-memory ProjectSource for tests.
type fakeProjectSource struct {
	projects map[string]Project
}

func (f *fakeProjectSource) Get(_ context.Context, id string) (*Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &p, nil
}

// fakeTicketTypeSource is an in-memory TicketTypeSource for tests.
type fakeTicketTypeSource struct {
	types map[string]TicketType
}

func (f *fakeTicketTypeSource) Get(_ context.Context, id string) (*TicketType, error) {
	tt, ok := f.types[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &tt, nil
}

// fakeAccessChecker records doc reads and returns the canned decision.
type fakeAccessChecker struct {
	canOpen map[string]bool
	err     error
}

func (f *fakeAccessChecker) Can(_ context.Context, _, docID string, action permissions.Action) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	if action != permissions.DocsRead {
		return false, nil
	}
	ok, found := f.canOpen[docID]
	if !found {
		return false, nil
	}
	return ok, nil
}

func newTestService(t *testing.T, tickets TicketSource, docs DocSource, statuses StatusSource, access AccessChecker) *Service {
	t.Helper()
	if tickets == nil {
		tickets = &fakeTicketSource{tickets: map[string]Ticket{}}
	}
	if docs == nil {
		docs = &fakeDocSource{docs: map[string]Doc{}}
	}
	if statuses == nil {
		statuses = &fakeStatusSource{statuses: map[string]Status{}}
	}
	if access == nil {
		access = &fakeAccessChecker{canOpen: map[string]bool{}}
	}
	return New(Config{
		Tickets:     tickets,
		Docs:        docs,
		Statuses:    statuses,
		Access:      access,
		Projects:    &fakeProjectSource{projects: map[string]Project{}},
		TicketTypes: &fakeTicketTypeSource{types: map[string]TicketType{}},
	})
}

func actorCtx(id string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: id})
}

func TestResolve_RequiresActor(t *testing.T) {
	svc := newTestService(t, nil, nil, nil, nil)
	_, err := svc.Resolve(context.Background(), []Ref{{Type: "ticket", ID: "t-1"}})
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestResolve_RequiresSources(t *testing.T) {
	svc := New(Config{})
	_, err := svc.Resolve(actorCtx("u-1"), []Ref{{Type: "ticket", ID: "t-1"}})
	require.Error(t, err)
}

func TestResolve_TicketChip(t *testing.T) {
	statuses := &fakeStatusSource{statuses: map[string]Status{
		"open": {ID: "open", Name: "Open"},
	}}
	tickets := &fakeTicketSource{tickets: map[string]Ticket{
		"t-1": {ID: "t-1", Title: "Fix the bug", Status: "open"},
	}}
	svc := newTestService(t, tickets, nil, statuses, nil)

	chips, err := svc.Resolve(actorCtx("u-1"), []Ref{{Type: "ticket", ID: "t-1"}})
	require.NoError(t, err)
	require.Len(t, chips, 1)
	assert.Equal(t, Chip{Type: "ticket", ID: "t-1", Title: "Fix the bug", Status: "open", StatusLabel: "Open", CanOpen: true}, chips[0])
}

// TestResolve_TicketChip_ExtraFields proves the chip template's extra fields (spec.md section 6, ticket 05) are populated from the ticket plus its project and type.
func TestResolve_TicketChip_ExtraFields(t *testing.T) {
	tickets := &fakeTicketSource{tickets: map[string]Ticket{
		"t-1": {ID: "t-1", Title: "Fix the bug", Status: "open", Number: 7, ProjectID: "p-1", TypeID: "type-bug", Developer: "onik97"},
	}}
	statuses := &fakeStatusSource{statuses: map[string]Status{"open": {ID: "open", Name: "Open"}}}
	projects := &fakeProjectSource{projects: map[string]Project{"p-1": {ID: "p-1", Prefix: "ERF"}}}
	types := &fakeTicketTypeSource{types: map[string]TicketType{"type-bug": {ID: "type-bug", Name: "Bug"}}}
	svc := New(Config{
		Tickets: tickets, Docs: &fakeDocSource{docs: map[string]Doc{}}, Statuses: statuses,
		Access: &fakeAccessChecker{canOpen: map[string]bool{}}, Projects: projects, TicketTypes: types,
	})

	chips, err := svc.Resolve(actorCtx("u-1"), []Ref{{Type: "ticket", ID: "t-1"}})
	require.NoError(t, err)
	require.Len(t, chips, 1)
	assert.Equal(t, "ERF", chips[0].ProjectPrefix)
	assert.Equal(t, 7, chips[0].ProjectNumber)
	assert.Equal(t, "Bug", chips[0].TypeLabel)
	assert.Equal(t, "onik97", chips[0].DeveloperLabel)
	assert.Equal(t, "", chips[0].DueLabel, "no due-date concept exists yet (ticket 05 flagged decision)")
}

// TestResolve_TicketChip_ProjectAndTypeFallback proves a ticket referencing a deleted project or no-longer-configured type degrades gracefully, mirroring TestResolve_StatusLabelFallback below.
func TestResolve_TicketChip_ProjectAndTypeFallback(t *testing.T) {
	tickets := &fakeTicketSource{tickets: map[string]Ticket{
		"t-1": {ID: "t-1", Title: "Ghost refs", Status: "open", ProjectID: "gone", TypeID: "gone-type"},
	}}
	statuses := &fakeStatusSource{statuses: map[string]Status{"open": {ID: "open", Name: "Open"}}}
	svc := New(Config{
		Tickets: tickets, Docs: &fakeDocSource{docs: map[string]Doc{}}, Statuses: statuses,
		Access:      &fakeAccessChecker{canOpen: map[string]bool{}},
		Projects:    &fakeProjectSource{projects: map[string]Project{}},
		TicketTypes: &fakeTicketTypeSource{types: map[string]TicketType{}},
	})

	chips, err := svc.Resolve(actorCtx("u-1"), []Ref{{Type: "ticket", ID: "t-1"}})
	require.NoError(t, err)
	require.Len(t, chips, 1)
	assert.Equal(t, "", chips[0].ProjectPrefix, "deleted project falls back to an empty prefix")
	assert.Equal(t, "gone-type", chips[0].TypeLabel, "falls back to the type id when it no longer resolves")
}

func TestResolve_DocChip_AccessAware(t *testing.T) {
	docs := &fakeDocSource{docs: map[string]Doc{
		"d-1": {ID: "d-1", Title: "Architecture"},
		"d-2": {ID: "d-2", Title: "Secrets"},
	}}
	access := &fakeAccessChecker{canOpen: map[string]bool{"d-1": true, "d-2": false}}
	svc := newTestService(t, nil, docs, nil, access)

	chips, err := svc.Resolve(actorCtx("u-1"), []Ref{{Type: "doc", ID: "d-1"}, {Type: "doc", ID: "d-2"}})
	require.NoError(t, err)
	require.Len(t, chips, 2)
	assert.True(t, chips[0].CanOpen)
	assert.False(t, chips[1].CanOpen)
	assert.Equal(t, "Secrets", chips[1].Title, "inaccessible targets still disclose title")
	assert.Equal(t, "d-2", chips[1].ID)
}

func TestResolve_StatusLabelFallback(t *testing.T) {
	tickets := &fakeTicketSource{tickets: map[string]Ticket{
		"t-1": {ID: "t-1", Title: "Done ticket", Status: "done"},
	}}
	statuses := &fakeStatusSource{statuses: map[string]Status{}} // done is not configured
	svc := newTestService(t, tickets, nil, statuses, nil)

	chips, err := svc.Resolve(actorCtx("u-1"), []Ref{{Type: "ticket", ID: "t-1"}})
	require.NoError(t, err)
	require.Len(t, chips, 1)
	assert.Equal(t, "done", chips[0].StatusLabel, "falls back to the status id when the column is gone")
}

func TestResolve_MissingTargetsOmitted(t *testing.T) {
	tickets := &fakeTicketSource{tickets: map[string]Ticket{}}
	docs := &fakeDocSource{docs: map[string]Doc{}}
	svc := newTestService(t, tickets, docs, nil, nil)

	chips, err := svc.Resolve(actorCtx("u-1"), []Ref{
		{Type: "ticket", ID: "missing"},
		{Type: "doc", ID: "gone"},
	})
	require.NoError(t, err)
	assert.Len(t, chips, 0)
}

func TestResolve_DedupesAndSkipsInvalidRefs(t *testing.T) {
	tickets := &fakeTicketSource{tickets: map[string]Ticket{
		"t-1": {ID: "t-1", Title: "Only once", Status: "open"},
	}}
	statuses := &fakeStatusSource{statuses: map[string]Status{"open": {ID: "open", Name: "Open"}}}
	svc := newTestService(t, tickets, nil, statuses, nil)

	chips, err := svc.Resolve(actorCtx("u-1"), []Ref{
		{Type: "ticket", ID: "t-1"},
		{Type: "ticket", ID: "t-1"},
		{Type: "", ID: "x"},
		{Type: "widget", ID: "y"},
		{Type: "doc", ID: ""},
	})
	require.NoError(t, err)
	require.Len(t, chips, 1)
	assert.Equal(t, "t-1", chips[0].ID)
}

func TestSearch_CombinesAndFiltersDocs(t *testing.T) {
	tickets := &fakeTicketSource{
		tickets: map[string]Ticket{
			"t-1": {ID: "t-1", Title: "Fix login", Status: "in_progress"},
		},
		search: []SearchHit{{ID: "t-1", Title: "Fix login"}},
	}
	docs := &fakeDocSource{
		docs: map[string]Doc{
			"d-1": {ID: "d-1", Title: "Auth notes"},
			"d-2": {ID: "d-2", Title: "Secret vault"},
		},
		search: []SearchHit{{ID: "d-1", Title: "Auth notes"}, {ID: "d-2", Title: "Secret vault"}},
	}
	statuses := &fakeStatusSource{statuses: map[string]Status{"in_progress": {ID: "in_progress", Name: "In progress"}}}
	access := &fakeAccessChecker{canOpen: map[string]bool{"d-1": true, "d-2": false}}
	svc := newTestService(t, tickets, docs, statuses, access)

	results, err := svc.Search(actorCtx("u-1"), "auth", 20)
	require.NoError(t, err)

	ids := make([]string, 0, len(results))
	for _, r := range results {
		ids = append(ids, r.Type+":"+r.ID)
	}
	assert.Equal(t, []string{"ticket:t-1", "doc:d-1"}, ids, "inaccessible docs are excluded")
	assert.Equal(t, "In progress", results[0].StatusLabel)
	assert.True(t, results[0].CanOpen)
}

func TestSearch_Errors(t *testing.T) {
	svc := newTestService(t, nil, nil, nil, nil)

	t.Run("requires actor", func(t *testing.T) {
		_, err := svc.Search(context.Background(), "auth", 20)
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})

	t.Run("requires query", func(t *testing.T) {
		_, err := svc.Search(actorCtx("u-1"), "  ", 20)
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("propagates source errors", func(t *testing.T) {
		boom := errors.New("search exploded")
		svc := newTestService(t,
			&fakeTicketSource{searchErr: boom},
			&fakeDocSource{searchErr: boom},
			nil, nil)
		_, err := svc.Search(actorCtx("u-1"), "auth", 20)
		require.Error(t, err)
	})
}

func TestSearch_KeyMatch_ExactPrefixNumber(t *testing.T) {
	tickets := &fakeTicketSource{
		tickets: map[string]Ticket{},
		byKey:   map[string]Ticket{"ERF-1": {ID: "t-1", Title: "Fix the router", Status: "open"}},
	}
	statuses := &fakeStatusSource{statuses: map[string]Status{"open": {ID: "open", Name: "Open"}}}
	svc := newTestService(t, tickets, nil, statuses, nil)

	results, err := svc.Search(actorCtx("u-1"), "ERF-1", 20)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, SearchResult{Type: "ticket", ID: "t-1", Title: "Fix the router", StatusLabel: "Open", CanOpen: true}, results[0])
}

func TestSearch_KeyMatch_WrongCaseFallsThroughToTitleSearch(t *testing.T) {
	tickets := &fakeTicketSource{
		tickets: map[string]Ticket{"t-1": {ID: "t-1", Title: "erf-1 something", Status: "open"}},
		byKey:   map[string]Ticket{"ERF-1": {ID: "t-2", Title: "Fix the router", Status: "open"}},
		search:  []SearchHit{{ID: "t-1", Title: "erf-1 something"}},
	}
	svc := newTestService(t, tickets, nil, nil, nil)

	// Lowercase query never matches the uppercase-only key regex, so it's treated as a plain title search; the key match for "ERF-1" is not surfaced even though it exists.
	results, err := svc.Search(actorCtx("u-1"), "erf-1", 20)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "t-1", results[0].ID)
}

func TestSearch_KeyMatch_NonexistentPrefixFallsThroughToTitleSearchOnly(t *testing.T) {
	tickets := &fakeTicketSource{
		tickets: map[string]Ticket{"t-1": {ID: "t-1", Title: "ZZZ-1 mentioned in title", Status: "open"}},
		byKey:   map[string]Ticket{}, // no project has prefix ZZZ
		search:  []SearchHit{{ID: "t-1", Title: "ZZZ-1 mentioned in title"}},
	}
	svc := newTestService(t, tickets, nil, nil, nil)

	results, err := svc.Search(actorCtx("u-1"), "ZZZ-1", 20)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "t-1", results[0].ID, "key regex matched but no ticket resolved, so title search still runs")
}

func TestSearch_KeyMatch_SortsFirstAndDedupsTitleHit(t *testing.T) {
	tickets := &fakeTicketSource{
		tickets: map[string]Ticket{"t-2": {ID: "t-2", Title: "Some other ticket"}},
		byKey:   map[string]Ticket{"ERF-1": {ID: "t-1", Title: "Fix the router", Status: "open"}},
		// The key match's ticket also happens to surface via title search; must be deduped, not listed twice.
		search: []SearchHit{{ID: "t-1", Title: "Fix the router"}, {ID: "t-2", Title: "Some other ticket"}},
	}
	docs := &fakeDocSource{
		docs:   map[string]Doc{"d-1": {ID: "d-1", Title: "ERF-1 design notes"}},
		search: []SearchHit{{ID: "d-1", Title: "ERF-1 design notes"}},
	}
	access := &fakeAccessChecker{canOpen: map[string]bool{"d-1": true}}
	svc := newTestService(t, tickets, docs, nil, access)

	results, err := svc.Search(actorCtx("u-1"), "ERF-1", 20)
	require.NoError(t, err)

	ids := make([]string, 0, len(results))
	for _, r := range results {
		ids = append(ids, r.Type+":"+r.ID)
	}
	assert.Equal(t, []string{"ticket:t-1", "ticket:t-2", "doc:d-1"}, ids, "key match first, deduped title hit, then docs")
}

func TestSearch_LimitClamping(t *testing.T) {
	tickets := &fakeTicketSource{
		search: []SearchHit{{ID: "t-1", Title: "a"}, {ID: "t-2", Title: "b"}, {ID: "t-3", Title: "c"}},
	}
	docs := &fakeDocSource{
		docs:   map[string]Doc{"d-1": {ID: "d-1", Title: "x"}},
		search: []SearchHit{{ID: "d-1", Title: "x"}},
	}
	access := &fakeAccessChecker{canOpen: map[string]bool{"d-1": true}}
	svc := newTestService(t, tickets, docs, nil, access)

	results, err := svc.Search(actorCtx("u-1"), "q", 2)
	require.NoError(t, err)
	require.Len(t, results, 2)
}

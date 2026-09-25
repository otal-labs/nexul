package composite

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// rows is an ordered in-memory table; its order stands in for the stored position.
type rows[T any] struct {
	items []T
	id    func(T) string
}

func (r *rows[T]) get(id string) (T, error) {
	for _, it := range r.items {
		if r.id(it) == id {
			return it, nil
		}
	}
	var zero T
	return zero, apperrs.ErrNotFound
}

func (r *rows[T]) put(v T) error {
	for i, it := range r.items {
		if r.id(it) == r.id(v) {
			r.items[i] = v
			return nil
		}
	}
	r.items = append(r.items, v)
	return nil
}

func (r *rows[T]) remove(id string) error {
	n := len(r.items)
	r.items = slices.DeleteFunc(r.items, func(it T) bool { return r.id(it) == id })
	if len(r.items) == n {
		return apperrs.ErrNotFound
	}
	return nil
}

func (r *rows[T]) where(keep func(T) bool) []T {
	var out []T
	for _, it := range r.items {
		if keep(it) {
			out = append(out, it)
		}
	}
	return out
}

func (r *rows[T]) reorder(ids []string) error {
	slices.SortStableFunc(r.items, func(a, b T) int { return slices.Index(ids, r.id(a)) - slices.Index(ids, r.id(b)) })
	return nil
}

// world is the one store behind every fake repo, so a move in one domain shows in the other, as in SQLite.
type world struct {
	tickets    map[string]*tickets.Ticket
	order      []string
	links      []tickets.TicketLink
	prs        map[string][]tickets.PRLink
	branches   map[string][]tickets.BranchLink
	labelColor map[string]colors.Color
	projects   rows[*workspace.Project]
	repos      map[string][]workspace.RepoRef
	statuses   rows[*workspace.Status]
	categories rows[*workspace.Category]
	types      rows[*workspace.TicketType]
	reviews    map[string][]*codereview.CodeReview
	owner      bool
	target     tickets.TestTarget
	fail       string
}

var errBoom = errors.New("storage is down")

// check fails the fake method named by fail, so a test can reach each storage error path.
func (w *world) check(method string) error {
	if w.fail == method {
		return errBoom
	}
	return nil
}

type fixture struct {
	w        *world
	tickets  *tickets.Service
	projects *workspace.Service
	reviews  *codereview.Service
}

// newFixture seeds project REF with three columns, two types, a category, and tickets REF-1 (todo) and REF-2 (done).
func newFixture(t *testing.T) fixture {
	t.Helper()
	w := &world{
		tickets: map[string]*tickets.Ticket{}, prs: map[string][]tickets.PRLink{}, branches: map[string][]tickets.BranchLink{},
		labelColor: map[string]colors.Color{}, repos: map[string][]workspace.RepoRef{}, reviews: map[string][]*codereview.CodeReview{},
		projects:   rows[*workspace.Project]{id: func(p *workspace.Project) string { return p.ID }},
		statuses:   rows[*workspace.Status]{id: func(s *workspace.Status) string { return s.ID }},
		categories: rows[*workspace.Category]{id: func(c *workspace.Category) string { return c.ID }},
		types:      rows[*workspace.TicketType]{id: func(tt *workspace.TicketType) string { return tt.ID }},
		owner:      true,
	}
	must(t, w.projects.put(&workspace.Project{ID: "p-1", Name: "Backend", Prefix: "REF", WorkspaceID: "ws-1", Icon: workspace.ProjectIconServer, TestsLocation: workspace.TestsLocationSame}))
	must(t, w.projects.put(&workspace.Project{ID: "p-2", Name: "Frontend", Prefix: "WEB", WorkspaceID: "ws-1"}))
	for _, s := range []*workspace.Status{
		{ID: "st-todo", ProjectID: "p-1", Name: "Todo", Kind: workspace.StatusKindBacklog, Icon: workspace.StatusIconTodo},
		{ID: "st-doing", ProjectID: "p-1", Name: "Doing", Kind: workspace.StatusKindProgress},
		{ID: "st-done", ProjectID: "p-1", Name: "Shipped", Kind: workspace.StatusKindDone},
		{ID: "st-web", ProjectID: "p-2", Name: "Web todo", Kind: workspace.StatusKindBacklog},
	} {
		must(t, w.statuses.put(s))
	}
	must(t, w.types.put(&workspace.TicketType{ID: "tt-task", ProjectID: "p-1", Name: "task", Color: colors.Cyan, BodyTemplate: "## What needs doing\n"}))
	must(t, w.types.put(&workspace.TicketType{ID: "tt-bug", ProjectID: "p-1", Name: "bug"}))
	must(t, w.categories.put(&workspace.Category{ID: "c-1", ProjectID: "p-1", Name: "Sprint 1", Color: colors.Lime}))
	w.addTicket(&tickets.Ticket{ID: "t-1", ProjectID: "p-1", Title: "Login times out", Body: "On slow networks.", Status: "st-todo", TypeID: "tt-task", CategoryID: "c-1", Developer: "onik97", Labels: []string{"urgent"}})
	w.addTicket(&tickets.Ticket{ID: "t-2", ProjectID: "p-1", Title: "Sign-in page", Status: "st-done"})

	ts := tickets.NewService(ticketRepo{w: w}, statusStore{w}, nil)
	ts.SetTicketTypes(typeLookup{w})
	ts.SetTesting(tickets.Testing{Targets: testTargets{w}})
	ws := workspace.NewService(projectRepo{w: w}, categoryRepo{w: w}, typeRepo{w}, statusRepo{w}, ownerGate{w}, workspaceGate{})
	return fixture{w: w, tickets: ts, projects: ws, reviews: codereview.NewService(reviewRepo{w: w})}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func (w *world) addTicket(t *tickets.Ticket) {
	n := 1
	for _, other := range w.tickets {
		if other.ProjectID == t.ProjectID {
			n++
		}
	}
	t.Number = n
	w.tickets[t.ID] = t
	w.order = append(w.order, t.ID)
}

func (w *world) linked(id string) *tickets.LinkedTicket {
	t, ok := w.tickets[id]
	if !ok {
		return nil
	}
	l := &tickets.LinkedTicket{ID: t.ID, ProjectID: t.ProjectID, Number: t.Number, Title: t.Title, Status: t.Status}
	if p, err := w.projects.get(t.ProjectID); err == nil {
		l.Prefix = p.Prefix
	}
	if s, err := w.statuses.get(string(t.Status)); err == nil {
		l.Done = s.Kind == workspace.StatusKindDone
	}
	return l
}

func (f fixture) ticketTools() []mcptool.Tool {
	return TicketTools(f.tickets, f.projects, f.reviews)
}

func (f fixture) projectTools() []mcptool.Tool {
	return ProjectTools(f.projects, f.tickets)
}

func call(t *testing.T, ctx context.Context, tools []mcptool.Tool, name, args string) (any, error) {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

func asUser(ctx context.Context) context.Context {
	return identity.WithActor(ctx, identity.Actor{ID: "u-1"})
}

func anonymous(ctx context.Context) context.Context { return ctx }

type ticketRepo struct {
	tickets.Repo
	w *world
}

func (r ticketRepo) Create(_ context.Context, t *tickets.Ticket, _ ...eventbus.OutboxEvent) error {
	r.w.addTicket(t)
	return nil
}

func (r ticketRepo) CreateWithLink(ctx context.Context, t *tickets.Ticket, link tickets.TicketLink, _ ...eventbus.OutboxEvent) error {
	r.w.addTicket(t)
	r.w.links = append(r.w.links, link)
	return nil
}

func (r ticketRepo) GetByID(_ context.Context, id string) (*tickets.Ticket, error) {
	t, ok := r.w.tickets[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	c := *t
	c.Labels = slices.Clone(t.Labels)
	return &c, nil
}

func (r ticketRepo) GetByPrefixAndNumber(ctx context.Context, prefix string, number int) (*tickets.Ticket, error) {
	for _, id := range r.w.order {
		l := r.w.linked(id)
		if l != nil && l.Prefix == prefix && l.Number == number {
			return r.GetByID(ctx, id)
		}
	}
	return nil, apperrs.ErrNotFound
}

func (r ticketRepo) filter(keep func(*tickets.Ticket) bool) []*tickets.Ticket {
	var out []*tickets.Ticket
	for _, id := range r.w.order {
		if t, ok := r.w.tickets[id]; ok && keep(t) {
			out = append(out, t)
		}
	}
	return out
}

func (r ticketRepo) List(context.Context) ([]*tickets.Ticket, error) {
	if err := r.w.check("List"); err != nil {
		return nil, err
	}
	return r.filter(func(*tickets.Ticket) bool { return true }), nil
}

func (r ticketRepo) ListByProject(_ context.Context, projectID string) ([]*tickets.Ticket, error) {
	return r.filter(func(t *tickets.Ticket) bool { return t.ProjectID == projectID }), nil
}

func (r ticketRepo) ListByDoc(_ context.Context, docID string) ([]*tickets.Ticket, error) {
	return r.filter(func(t *tickets.Ticket) bool { return t.DocID == docID }), nil
}

func (r ticketRepo) Search(_ context.Context, query string, limit int) ([]tickets.SearchResult, error) {
	if err := r.w.check("Search"); err != nil {
		return nil, err
	}
	var out []tickets.SearchResult
	for _, t := range r.filter(func(t *tickets.Ticket) bool {
		return strings.Contains(strings.ToLower(t.Title+" "+t.Body), strings.ToLower(query))
	}) {
		out = append(out, tickets.SearchResult{ID: t.ID, Title: t.Title})
	}
	return out[:min(limit, len(out))], nil
}

func (r ticketRepo) edit(id string, change func(*tickets.Ticket)) error {
	t, ok := r.w.tickets[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	change(t)
	return nil
}

func (r ticketRepo) UpdateTicket(_ context.Context, id, title, body string, _ ...eventbus.OutboxEvent) error {
	if err := r.w.check("UpdateTicket"); err != nil {
		return err
	}
	return r.edit(id, func(t *tickets.Ticket) { t.Title, t.Body = title, body })
}

func (r ticketRepo) UpdateStatus(_ context.Context, id string, status tickets.Status, _ ...eventbus.OutboxEvent) error {
	return r.edit(id, func(t *tickets.Ticket) { t.Status = status })
}

func (r ticketRepo) SetPosition(_ context.Context, id string, position int) error {
	return r.edit(id, func(t *tickets.Ticket) { t.Position = position })
}

func (r ticketRepo) UpdateType(_ context.Context, id, typeID string) error {
	return r.edit(id, func(t *tickets.Ticket) { t.TypeID = typeID })
}

func (r ticketRepo) UpdatePerson(_ context.Context, id string, role tickets.Role, login string, _ ...eventbus.OutboxEvent) error {
	return r.edit(id, func(t *tickets.Ticket) {
		if role == tickets.RoleTester {
			t.Tester = login
			return
		}
		t.Developer = login
	})
}

func (r ticketRepo) AddLabel(_ context.Context, id, label string) error {
	return r.edit(id, func(t *tickets.Ticket) {
		if !slices.Contains(t.Labels, label) {
			t.Labels = append(t.Labels, label)
		}
	})
}

func (r ticketRepo) RemoveLabel(_ context.Context, id, label string) error {
	return r.edit(id, func(t *tickets.Ticket) {
		t.Labels = slices.DeleteFunc(t.Labels, func(l string) bool { return l == label })
	})
}

func (r ticketRepo) ListAllLabels(context.Context) ([]string, error) {
	if err := r.w.check("ListAllLabels"); err != nil {
		return nil, err
	}
	var all []string
	for _, t := range r.w.tickets {
		for _, l := range t.Labels {
			if !slices.Contains(all, l) {
				all = append(all, l)
			}
		}
	}
	slices.Sort(all)
	return all, nil
}

func (r ticketRepo) SetLabelColor(_ context.Context, projectID, label string, color colors.Color) error {
	r.w.labelColor[projectID+"/"+label] = color
	return nil
}

func (r ticketRepo) LabelColors(_ context.Context, projectID string, labels []string) (map[string]colors.Color, error) {
	if err := r.w.check("LabelColors"); err != nil {
		return nil, err
	}
	out := map[string]colors.Color{}
	for _, l := range labels {
		if c, ok := r.w.labelColor[projectID+"/"+l]; ok {
			out[l] = c
		}
	}
	return out, nil
}

func (r ticketRepo) LinkPR(_ context.Context, id string, ref tickets.PRRef, state tickets.PRState) error {
	r.w.prs[id] = append(r.w.prs[id], tickets.PRLink{PRRef: ref, State: state})
	return nil
}

func (r ticketRepo) ListPRLinks(_ context.Context, id string) ([]tickets.PRLink, error) {
	if err := r.w.check("ListPRLinks"); err != nil {
		return nil, err
	}
	return r.w.prs[id], nil
}

func (r ticketRepo) LinkBranch(_ context.Context, id string, link tickets.BranchLink) error {
	r.w.branches[id] = append(r.w.branches[id], link)
	return nil
}

func (r ticketRepo) ListBranchLinks(_ context.Context, id string) ([]tickets.BranchLink, error) {
	return r.w.branches[id], nil
}

func (r ticketRepo) ListLinkEnds(_ context.Context, id string) (from, to []tickets.LinkEnd, err error) {
	if err := r.w.check("ListLinkEnds"); err != nil {
		return nil, nil, err
	}
	for _, l := range r.w.links {
		if l.TicketID == id {
			from = append(from, tickets.LinkEnd{Kind: l.Kind, Ticket: r.w.linked(l.TargetID)})
		}
		if l.TargetID == id {
			to = append(to, tickets.LinkEnd{Kind: l.Kind, Ticket: r.w.linked(l.TicketID)})
		}
	}
	return from, to, nil
}

func (r ticketRepo) BlockerIDs(_ context.Context, id string) ([]string, error) {
	var out []string
	for _, l := range r.w.links {
		if l.TicketID == id && l.Kind == tickets.LinkBlockedBy {
			out = append(out, l.TargetID)
		}
	}
	return out, nil
}

func sameLink(a, b tickets.TicketLink) bool {
	return a.TicketID == b.TicketID && a.Kind == b.Kind && (a.Kind == tickets.LinkFoundIn || a.TargetID == b.TargetID)
}

func (r ticketRepo) PutLink(_ context.Context, link tickets.TicketLink, _ ...eventbus.OutboxEvent) error {
	r.w.links = slices.DeleteFunc(r.w.links, func(l tickets.TicketLink) bool { return sameLink(l, link) })
	r.w.links = append(r.w.links, link)
	return nil
}

func (r ticketRepo) DeleteLink(_ context.Context, link tickets.TicketLink, _ ...eventbus.OutboxEvent) (bool, error) {
	n := len(r.w.links)
	r.w.links = slices.DeleteFunc(r.w.links, func(l tickets.TicketLink) bool { return sameLink(l, link) })
	return len(r.w.links) < n, nil
}

func (r ticketRepo) UnclearedBlockers(context.Context) (map[string][]tickets.LinkedTicket, error) {
	if err := r.w.check("UnclearedBlockers"); err != nil {
		return nil, err
	}
	out := map[string][]tickets.LinkedTicket{}
	for _, l := range r.w.links {
		if b := r.w.linked(l.TargetID); l.Kind == tickets.LinkBlockedBy && b != nil && !b.Done {
			out[l.TicketID] = append(out[l.TicketID], *b)
		}
	}
	return out, nil
}

type statusStore struct{ w *world }

func (s statusStore) Exists(_ context.Context, id string) (bool, error) {
	_, err := s.w.statuses.get(id)
	return err == nil, nil
}

type typeLookup struct{ w *world }

func (l typeLookup) TypeName(_ context.Context, id string) (string, error) {
	tt, err := l.w.types.get(id)
	if err != nil {
		return "", err
	}
	return tt.Name, nil
}

func (l typeLookup) BodyTemplate(_ context.Context, id string) (string, error) {
	tt, err := l.w.types.get(id)
	if err != nil {
		return "", err
	}
	return tt.BodyTemplate, nil
}

type testTargets struct{ w *world }

func (t testTargets) ResolveTestTarget(context.Context, string, []tickets.BranchLink) (tickets.TestTarget, error) {
	return t.w.target, nil
}

type reviewRepo struct {
	codereview.Repo
	w *world
}

func (r reviewRepo) ListByTicket(_ context.Context, ticketID string) ([]*codereview.CodeReview, error) {
	if err := r.w.check("ListReviews"); err != nil {
		return nil, err
	}
	return r.w.reviews[ticketID], nil
}

type projectRepo struct {
	workspace.Repo
	w *world
}

func (r projectRepo) Get(_ context.Context, id string) (*workspace.Project, error) {
	p, err := r.w.projects.get(id)
	if err != nil {
		return nil, err
	}
	c := *p
	return &c, nil
}

func (r projectRepo) List(_ context.Context, workspaceID string) ([]*workspace.Project, error) {
	if err := r.w.check("ListProjects"); err != nil {
		return nil, err
	}
	return r.w.projects.where(func(p *workspace.Project) bool { return p.WorkspaceID == workspaceID }), nil
}

func (r projectRepo) Update(_ context.Context, p *workspace.Project) error {
	return r.w.projects.put(p)
}

func (r projectRepo) Reorder(_ context.Context, ids []string) error { return r.w.projects.reorder(ids) }

func (r projectRepo) CountTickets(_ context.Context, projectID string) (int, error) {
	if err := r.w.check("CountTickets"); err != nil {
		return 0, err
	}
	n := 0
	for _, t := range r.w.tickets {
		if t.ProjectID == projectID {
			n++
		}
	}
	return n, nil
}

func (r projectRepo) CountRepos(_ context.Context, projectID string) (int, error) {
	return len(r.w.repos[projectID]), nil
}

func (r projectRepo) CountServices(context.Context, string) (int, error) { return 0, nil }

func (r projectRepo) AddRepo(_ context.Context, projectID string, ref workspace.RepoRef) error {
	r.w.repos[projectID] = append(r.w.repos[projectID], ref)
	return nil
}

func (r projectRepo) RemoveRepo(_ context.Context, owner, name string) error {
	for id, refs := range r.w.repos {
		r.w.repos[id] = slices.DeleteFunc(refs, func(ref workspace.RepoRef) bool { return ref.Owner == owner && ref.Name == name })
	}
	return nil
}

func (r projectRepo) ListRepos(_ context.Context, projectID string) ([]workspace.RepoRef, error) {
	if err := r.w.check("ListRepos"); err != nil {
		return nil, err
	}
	return r.w.repos[projectID], nil
}

func (r projectRepo) MoveTicket(_ context.Context, ticketID, projectID string) error {
	return ticketRepo{w: r.w}.edit(ticketID, func(t *tickets.Ticket) { t.ProjectID = projectID })
}

type statusRepo struct{ w *world }

func (r statusRepo) Create(_ context.Context, s *workspace.Status, _ ...eventbus.OutboxEvent) error {
	return r.w.statuses.put(s)
}

func (r statusRepo) Get(_ context.Context, id string) (*workspace.Status, error) {
	return r.w.statuses.get(id)
}

func (r statusRepo) ListByProject(_ context.Context, projectID string) ([]*workspace.Status, error) {
	if err := r.w.check("ListStatuses"); err != nil {
		return nil, err
	}
	return r.w.statuses.where(func(s *workspace.Status) bool { return s.ProjectID == projectID }), nil
}

func (r statusRepo) Update(_ context.Context, s *workspace.Status, _ ...eventbus.OutboxEvent) error {
	return r.w.statuses.put(s)
}

func (r statusRepo) Delete(_ context.Context, id string, _ ...eventbus.OutboxEvent) error {
	return r.w.statuses.remove(id)
}

func (r statusRepo) Reorder(_ context.Context, _ string, ids []string) error {
	return r.w.statuses.reorder(ids)
}

func (r statusRepo) CountTickets(_ context.Context, id string) (int, error) {
	return len(ticketRepo{w: r.w}.filter(func(t *tickets.Ticket) bool { return string(t.Status) == id })), nil
}

type categoryRepo struct {
	workspace.CategoryRepo
	w *world
}

func (r categoryRepo) Create(_ context.Context, c *workspace.Category, _ ...eventbus.OutboxEvent) error {
	return r.w.categories.put(c)
}

func (r categoryRepo) Get(_ context.Context, id string) (*workspace.Category, error) {
	return r.w.categories.get(id)
}

func (r categoryRepo) ListByProject(_ context.Context, projectID string) ([]*workspace.Category, error) {
	if err := r.w.check("ListCategories"); err != nil {
		return nil, err
	}
	return r.w.categories.where(func(c *workspace.Category) bool { return c.ProjectID == projectID }), nil
}

func (r categoryRepo) Update(_ context.Context, c *workspace.Category, _ ...eventbus.OutboxEvent) error {
	return r.w.categories.put(c)
}

func (r categoryRepo) Delete(_ context.Context, id string, _ ...eventbus.OutboxEvent) error {
	return r.w.categories.remove(id)
}

func (r categoryRepo) Reorder(_ context.Context, _ string, ids []string) error {
	return r.w.categories.reorder(ids)
}

func (r categoryRepo) SetTicketCategory(_ context.Context, ticketID, categoryID string, _ ...eventbus.OutboxEvent) error {
	return ticketRepo{w: r.w}.edit(ticketID, func(t *tickets.Ticket) { t.CategoryID = categoryID })
}

type typeRepo struct{ w *world }

func (r typeRepo) Create(_ context.Context, tt *workspace.TicketType, _ ...eventbus.OutboxEvent) error {
	return r.w.types.put(tt)
}

func (r typeRepo) Get(_ context.Context, id string) (*workspace.TicketType, error) {
	return r.w.types.get(id)
}

func (r typeRepo) ListByProject(_ context.Context, projectID string) ([]*workspace.TicketType, error) {
	if err := r.w.check("ListTypes"); err != nil {
		return nil, err
	}
	return r.w.types.where(func(tt *workspace.TicketType) bool { return tt.ProjectID == projectID }), nil
}

func (r typeRepo) Update(_ context.Context, tt *workspace.TicketType, _ ...eventbus.OutboxEvent) error {
	return r.w.types.put(tt)
}

func (r typeRepo) Delete(_ context.Context, id string, _ ...eventbus.OutboxEvent) error {
	return r.w.types.remove(id)
}

func (r typeRepo) Reorder(_ context.Context, _ string, ids []string) error {
	return r.w.types.reorder(ids)
}

func (r typeRepo) CountTickets(_ context.Context, id string) (int, error) {
	return len(ticketRepo{w: r.w}.filter(func(t *tickets.Ticket) bool { return t.TypeID == id })), nil
}

type ownerGate struct{ w *world }

func (g ownerGate) CanCreateWorkspace(context.Context, string) (bool, error) { return g.w.owner, nil }

type workspaceGate struct{}

func (workspaceGate) WorkspaceExists(context.Context, string) (bool, error) { return true, nil }

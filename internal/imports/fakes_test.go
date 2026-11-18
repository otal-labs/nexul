package imports

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// fakeRepo keeps imports in memory with the stored rules the SQLite repo has: one open import per workspace, and a
// compare-and-set on state.
type fakeRepo struct {
	mu      sync.Mutex
	imports map[string]Import
	items   map[string]Item
	ids     map[string]string
	events  []eventbus.OutboxEvent
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{imports: map[string]Import{}, items: map[string]Item{}, ids: map[string]string{}}
}

func (r *fakeRepo) Create(_ context.Context, imp *Import, evts ...eventbus.OutboxEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, other := range r.imports {
		if other.WorkspaceID == imp.WorkspaceID && !other.State.Final() {
			return apperrs.ErrConflict
		}
	}
	r.imports[imp.ID] = *imp
	r.events = append(r.events, evts...)
	return nil
}

func (r *fakeRepo) Get(_ context.Context, id string) (*Import, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	imp, ok := r.imports[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &imp, nil
}

func (r *fakeRepo) ListByWorkspace(_ context.Context, workspaceID string, _ int) ([]*Import, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*Import
	for _, imp := range r.imports {
		if imp.WorkspaceID == workspaceID {
			out = append(out, &imp)
		}
	}
	return out, nil
}

func (r *fakeRepo) Open(_ context.Context) ([]*Import, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*Import
	for _, imp := range r.imports {
		if !imp.State.Final() {
			out = append(out, &imp)
		}
	}
	return out, nil
}

func (r *fakeRepo) Update(_ context.Context, imp *Import, from State, evts ...eventbus.OutboxEvent) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.imports[imp.ID].State != from {
		return false, nil
	}
	stored := *imp
	if stored.State.Final() {
		stored.Credential = ""
	}
	r.imports[imp.ID] = stored
	r.events = append(r.events, evts...)
	return true, nil
}

func (r *fakeRepo) Move(_ context.Context, id string, from, to State, reason string, at time.Time, event func(*Import) eventbus.OutboxEvent) (*Import, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.imports[id]
	if !ok || stored.State != from {
		return nil, false, nil
	}
	stored.State, stored.UpdatedAt = to, at
	if reason != "" {
		stored.Report.Error = reason
	}
	if to.Final() {
		stored.Credential = ""
	}
	r.imports[id] = stored
	r.events = append(r.events, event(&stored))
	return &stored, true, nil
}

func (r *fakeRepo) Item(_ context.Context, importID, kind, sourceID string) (Item, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	it, ok := r.items[importID+"/"+kind+"/"+sourceID]
	return it, ok, nil
}

func (r *fakeRepo) Imported(_ context.Context, workspaceID, source, kind, sourceID string) (string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.ids[workspaceID+"/"+source+"/"+kind+"/"+sourceID]
	return id, ok, nil
}

func (r *fakeRepo) Record(_ context.Context, workspaceID, source string, it Item) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[it.ImportID+"/"+it.Kind+"/"+it.SourceID] = it
	if it.State != ItemFailed {
		r.ids[workspaceID+"/"+source+"/"+it.Kind+"/"+it.SourceID] = it.NexulID
	}
	return nil
}

func (r *fakeRepo) Unfilled(_ context.Context, importID, kind string) ([]Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Item
	for _, it := range r.items {
		if it.ImportID == importID && it.Kind == kind && it.State == ItemCreated {
			out = append(out, it)
		}
	}
	return out, nil
}

func (r *fakeRepo) Forget(_ context.Context, workspaceID, source string, it Item) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[it.ImportID+"/"+it.Kind+"/"+it.SourceID] = it
	delete(r.ids, workspaceID+"/"+source+"/"+it.Kind+"/"+it.SourceID)
	return nil
}

func (r *fakeRepo) stored(id string) Import {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.imports[id]
}

// fakeOwners makes owner the Owner of every workspace.
type fakeOwners struct{ owner string }

func (o fakeOwners) IsOwner(_ context.Context, userID, _ string) (bool, error) {
	return userID == o.owner, nil
}

type fakeProjects map[string]string

func (p fakeProjects) WorkspaceOf(_ context.Context, projectID string) (string, error) {
	ws, ok := p[projectID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return ws, nil
}

type fakeBoard struct {
	cols []Column
	cats []Category
}

func (b *fakeBoard) Columns(context.Context, string) ([]Column, error) { return b.cols, nil }

func (b *fakeBoard) AddColumn(_ context.Context, _, name string, kind StatusKind) (Column, error) {
	c := Column{ID: "col-" + name, Name: name, Kind: kind}
	b.cols = append(b.cols, c)
	return c, nil
}

func (b *fakeBoard) Categories(context.Context, string) ([]Category, error) { return b.cats, nil }

func (b *fakeBoard) AddCategory(_ context.Context, _, name string) (Category, error) {
	c := Category{ID: "cat-" + name, Name: name}
	b.cats = append(b.cats, c)
	return c, nil
}

type fakePeople []Member

func (p fakePeople) Members(context.Context, string) ([]Member, error) { return p, nil }

// fakeSource scans to a fixed result and hands over its tickets.
type fakeSource struct {
	scan    Scan
	scanErr error
	tickets []Ticket
}

func (s fakeSource) Scan(context.Context, Job) (Scan, error) { return s.scan, s.scanErr }

func (s fakeSource) Import(ctx context.Context, _ Job, sink Sink) error {
	for _, t := range s.tickets {
		if err := sink.Ticket(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

func (s fakeSource) Info() Info { return Info{Title: "Fake", FileHosts: []string{"files.example.com"}} }

// fakeDesk keeps what an import wrote; fail refuses the ticket titled with it.
type fakeDesk struct {
	mu       sync.Mutex
	tickets  map[string]NewTicket
	comments map[string][]NewComment
	files    map[string]string
	fail     string
	onFile   func()
	// filed counts each title filed under a new id, so a ticket filed twice shows as 2.
	filed map[string]int
	// lostAfterFile files the ticket of that title, then answers as if the write failed, as a crash before its record does.
	lostAfterFile string
	// refuseComments answers every comment as invalid.
	refuseComments bool
	// blocks is each ticket id and the ids it waits on.
	blocks map[string][]string
}

func newFakeDesk() *fakeDesk {
	return &fakeDesk{tickets: map[string]NewTicket{}, comments: map[string][]NewComment{}, files: map[string]string{}, filed: map[string]int{}, blocks: map[string][]string{}}
}

func (d *fakeDesk) FileTicket(_ context.Context, t NewTicket) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if t.Title == d.fail {
		return fmt.Errorf("%w: title refused", apperrs.ErrInvalid)
	}
	if _, ok := d.tickets[t.ID]; ok {
		return apperrs.ErrConflict
	}
	d.tickets[t.ID] = t
	d.filed[t.Title]++
	if d.onFile != nil {
		d.onFile()
	}
	if t.Title == d.lostAfterFile {
		d.lostAfterFile = ""
		return errors.New("connection reset after the commit")
	}
	return nil
}

func (d *fakeDesk) SetBody(_ context.Context, ticketID, _, body string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	t := d.tickets[ticketID]
	t.Body = body
	d.tickets[ticketID] = t
	return nil
}

func (d *fakeDesk) Comment(_ context.Context, ticketID string, c NewComment) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.refuseComments {
		return fmt.Errorf("%w: comment refused", apperrs.ErrInvalid)
	}
	d.comments[ticketID] = append(d.comments[ticketID], c)
	return nil
}

func (d *fakeDesk) Attach(_ context.Context, _, name string, _ []byte) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := "att-" + name
	d.files[id] = name
	return id, nil
}

func (d *fakeDesk) Block(_ context.Context, ticketID, blockerID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !slices.Contains(d.blocks[ticketID], blockerID) {
		d.blocks[ticketID] = append(d.blocks[ticketID], blockerID)
	}
	return nil
}

// fakeFiles serves every URL under files.example.com, and refuses anything over its size.
type fakeFiles struct{}

func (fakeFiles) Fetch(_ context.Context, rawURL string, hosts []string) (string, []byte, error) {
	if strings.Contains(rawURL, "huge") {
		return "", nil, ErrTooLarge
	}
	if !strings.HasPrefix(rawURL, "https://"+hosts[0]+"/") {
		return "", nil, errHost
	}
	return path.Base(rawURL), []byte("bytes"), nil
}

const (
	testOwner     = "u-owner"
	testWorkspace = "ws-1"
	testProject   = "p-1"
)

func newTestService(src Source) (*Service, *fakeRepo) {
	svc, repo, _ := newDeskService(src, newFakeDesk())
	return svc, repo
}

func newDeskService(src Source, desk *fakeDesk) (*Service, *fakeRepo, *fakeDesk) {
	repo := newFakeRepo()
	return NewService(Deps{
		Repo: repo, Owners: fakeOwners{owner: testOwner}, Projects: fakeProjects{testProject: testWorkspace},
		Board: &fakeBoard{}, People: fakePeople{{Login: "alice", GitHubLogins: []string{"alice"}}}, Desk: desk,
		Files: fakeFiles{}, Sources: map[string]Source{"fake": src},
	}), repo, desk
}

func as(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

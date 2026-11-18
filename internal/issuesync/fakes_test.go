package issuesync

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

type fakeRepo struct {
	mu       sync.Mutex
	syncs    map[string]*Sync
	links    map[string]*Link
	comments map[int64]CommentLink
	err      error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{syncs: map[string]*Sync{}, links: map[string]*Link{}, comments: map[int64]CommentLink{}}
}

func linkKey(owner, repo string, n int) string { return fmt.Sprintf("%s/%s#%d", owner, repo, n) }

func (r *fakeRepo) GetSync(_ context.Context, owner, repo string) (*Sync, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	s, ok := r.syncs[owner+"/"+repo]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *fakeRepo) SyncsOfProject(_ context.Context, projectID string) ([]*Sync, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*Sync
	for _, s := range r.syncs {
		if s.ProjectID == projectID {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *fakeRepo) EnabledSyncs(_ context.Context) ([]*Sync, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	var out []*Sync
	for _, s := range r.syncs {
		if s.Enabled {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *fakeRepo) SaveSync(_ context.Context, s *Sync) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *s
	r.syncs[s.Owner+"/"+s.Repo] = &cp
	return nil
}

func (r *fakeRepo) MarkSwept(_ context.Context, owner, repo string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.syncs[owner+"/"+repo].LastSweptAt = &at
	return nil
}

func (r *fakeRepo) GetLink(_ context.Context, owner, repo string, n int) (*Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.links[linkKey(owner, repo, n)]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *l
	return &cp, nil
}

func (r *fakeRepo) SaveLink(_ context.Context, l *Link) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *l
	r.links[linkKey(l.Owner, l.Repo, l.Number)] = &cp
	return nil
}

func (r *fakeRepo) DeleteLink(_ context.Context, owner, repo string, n int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.links, linkKey(owner, repo, n))
	return nil
}

func (r *fakeRepo) GetCommentLink(_ context.Context, id int64) (*CommentLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.comments[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &l, nil
}

func (r *fakeRepo) SaveCommentLink(_ context.Context, l CommentLink) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.comments[l.CommentID] = l
	return nil
}

func (r *fakeRepo) DeleteCommentLink(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.comments, id)
	return nil
}

// fakeTickets is the tickets side; writes counts every field write, so a no-op can be told from a rewrite.
type fakeTickets struct {
	mu        sync.Mutex
	tickets   map[string]*Ticket
	reporters map[string]string
	moves     []Stage
	writes    int
	createErr error
}

func newFakeTickets() *fakeTickets {
	return &fakeTickets{tickets: map[string]*Ticket{}, reporters: map[string]string{}}
}

func (f *fakeTickets) Get(_ context.Context, id string) (Ticket, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tickets[id]
	if !ok {
		return Ticket{}, apperrs.ErrNotFound
	}
	cp := *t
	cp.Labels = slices.Clone(t.Labels)
	return cp, nil
}

func (f *fakeTickets) Create(_ context.Context, projectID string, n NewTicket) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return "", f.createErr
	}
	id := fmt.Sprintf("t%d", len(f.tickets)+1)
	f.tickets[id] = &Ticket{ID: id, ProjectID: projectID, Title: n.Title, Body: n.Body}
	f.reporters[id] = n.Reporter
	return id, nil
}

func (f *fakeTickets) Move(_ context.Context, id string, stage Stage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.moves = append(f.moves, stage)
	f.tickets[id].Done = stage == StageDone
	return nil
}

func (f *fakeTickets) SetText(_ context.Context, id, title, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	f.tickets[id].Title, f.tickets[id].Body = title, body
	return nil
}

func (f *fakeTickets) SetLabels(_ context.Context, id string, labels []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	f.tickets[id].Labels = slices.Clone(labels)
	return nil
}

func (f *fakeTickets) SetDeveloper(_ context.Context, id, login string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	f.tickets[id].Developer = login
	return nil
}

func (f *fakeTickets) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.tickets)
}

type fakeThread struct {
	mu    sync.Mutex
	notes map[string]string
	order []string
}

func newFakeThread() *fakeThread { return &fakeThread{notes: map[string]string{}} }

func (t *fakeThread) Note(_ context.Context, _ string, key, body string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.notes[key]; ok {
		return nil
	}
	t.notes[key] = body
	t.order = append(t.order, body)
	return nil
}

type fakeComments struct {
	mu       sync.Mutex
	messages map[string]string
	deleted  []string
}

func newFakeComments() *fakeComments { return &fakeComments{messages: map[string]string{}} }

func (c *fakeComments) Post(_ context.Context, _ string, cm Comment) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := fmt.Sprintf("m%d", cm.ID)
	c.messages[id] = cm.Body
	return id, nil
}

func (c *fakeComments) Edit(_ context.Context, id string, cm Comment) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages[id] = cm.Body
	return nil
}

func (c *fakeComments) Delete(_ context.Context, id, _ string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.messages, id)
	c.deleted = append(c.deleted, id)
	return nil
}

// fakeMembers maps GitHub logins to member logins.
type fakeMembers map[string]string

func (m fakeMembers) MemberLogin(_ context.Context, _ string, gh string) (string, bool, error) {
	login, ok := m[gh]
	return login, ok, nil
}

type fakeHost struct {
	mu      sync.Mutex
	bot     string
	pages   []IssuePage
	err     error
	queries []IssueQuery
}

func (h *fakeHost) BotLogin(context.Context) (string, error) { return h.bot, nil }

func (h *fakeHost) ListIssues(_ context.Context, _, _ string, q IssueQuery) (IssuePage, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.queries = append(h.queries, q)
	if h.err != nil {
		return IssuePage{}, h.err
	}
	if q.Page > len(h.pages) {
		return IssuePage{}, nil
	}
	return h.pages[q.Page-1], nil
}

type fakeGate struct{ err error }

func (g fakeGate) RequireProject(context.Context, string, permissions.Action) error { return g.err }

type fakeRepos map[string]bool

func (r fakeRepos) HasRepo(_ context.Context, projectID, owner, repo string) (bool, error) {
	return r[projectID+":"+owner+"/"+repo], nil
}

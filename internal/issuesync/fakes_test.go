package issuesync

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type linkKey struct {
	owner, repo string
	number      int
}

type fakeRepo struct {
	mu       sync.Mutex
	syncs    map[string]Sync
	links    map[linkKey]Link
	comments map[int64]CommentLink
	swept    []string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{syncs: map[string]Sync{}, links: map[linkKey]Link{}, comments: map[int64]CommentLink{}}
}

func (r *fakeRepo) GetSync(_ context.Context, owner, repo string) (Sync, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.syncs[owner+"/"+repo]
	if !ok {
		return Sync{}, apperrs.ErrNotFound
	}
	return s, nil
}

func (r *fakeRepo) ListSyncsByProject(_ context.Context, projectID string) ([]Sync, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Sync
	for _, s := range r.syncs {
		if s.ProjectID == projectID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *fakeRepo) ListEnabledSyncs(context.Context) ([]Sync, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Sync
	for _, s := range r.syncs {
		if s.Enabled {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b Sync) int { return cmp.Compare(a.Owner+a.Repo, b.Owner+b.Repo) })
	return out, nil
}

func (r *fakeRepo) PutSync(_ context.Context, s Sync) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.syncs[s.Owner+"/"+s.Repo] = s
	return nil
}

func (r *fakeRepo) SetSwept(_ context.Context, owner, repo string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.syncs[owner+"/"+repo]
	s.LastSweptAt = &at
	r.syncs[owner+"/"+repo] = s
	r.swept = append(r.swept, owner+"/"+repo)
	return nil
}

func (r *fakeRepo) GetLink(_ context.Context, owner, repo string, number int) (Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.links[linkKey{owner, repo, number}]
	if !ok {
		return Link{}, apperrs.ErrNotFound
	}
	return l, nil
}

func (r *fakeRepo) PutLink(_ context.Context, l Link) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.links[linkKey{l.Owner, l.Repo, l.IssueNumber}] = l
	return nil
}

func (r *fakeRepo) DeleteLink(_ context.Context, l Link) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.links, linkKey{l.Owner, l.Repo, l.IssueNumber})
	for id, c := range r.comments {
		if c.TicketID == l.TicketID {
			delete(r.comments, id)
		}
	}
	return nil
}

func (r *fakeRepo) GetCommentLink(_ context.Context, id int64) (CommentLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.comments[id]
	if !ok {
		return CommentLink{}, apperrs.ErrNotFound
	}
	return c, nil
}

func (r *fakeRepo) PutCommentLink(_ context.Context, c CommentLink) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.comments[c.GitHubCommentID] = c
	return nil
}

func (r *fakeRepo) DeleteCommentLink(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.comments, id)
	return nil
}

type fakeTicket struct {
	Fields    TicketFields
	Developer string
	Reporter  string
	Stage     Stage
}

// fakeDesk is the tickets side: Desk, Board, Notes and Comments in one.
type fakeDesk struct {
	mu       sync.Mutex
	tickets  map[string]*fakeTicket
	applied  int
	notes    map[string]string
	messages map[string]string
	edits    int
	deleted  []string
	fail     error
}

func newFakeDesk() *fakeDesk {
	return &fakeDesk{tickets: map[string]*fakeTicket{}, notes: map[string]string{}, messages: map[string]string{}}
}

func (d *fakeDesk) File(_ context.Context, _ string, f TicketFields, developer, reporter string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fail != nil {
		return "", d.fail
	}
	id := fmt.Sprintf("t-%d", len(d.tickets)+1)
	d.tickets[id] = &fakeTicket{Fields: f, Developer: developer, Reporter: reporter, Stage: StageBacklog}
	return id, nil
}

func (d *fakeDesk) Apply(_ context.Context, id string, f TicketFields, developer string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fail != nil {
		return d.fail
	}
	d.applied++
	t := d.tickets[id]
	t.Fields = f
	if developer != "" {
		t.Developer = developer
	}
	return nil
}

func (d *fakeDesk) Done(_ context.Context, id string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.tickets[id].Stage == StageDone, nil
}

func (d *fakeDesk) MoveTo(_ context.Context, id string, stage Stage) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tickets[id].Stage = stage
	return nil
}

func (d *fakeDesk) Note(_ context.Context, id, key, body string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.notes[key] = id + ": " + body
	return nil
}

func (d *fakeDesk) Post(_ context.Context, _ string, c Comment) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := fmt.Sprintf("m-%d", c.ID)
	d.messages[id] = c.Body
	return id, nil
}

func (d *fakeDesk) Edit(_ context.Context, id string, c Comment) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.edits++
	d.messages[id] = c.Body
	return nil
}

func (d *fakeDesk) Delete(_ context.Context, id string, _ Comment) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.deleted = append(d.deleted, id)
	delete(d.messages, id)
	return nil
}

func (d *fakeDesk) only() *fakeTicket {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.tickets["t-1"]
}

type fakePeople map[string]string

func (p fakePeople) Member(_ context.Context, _, login string) (string, error) { return p[login], nil }

type fakeIssues struct {
	mu       sync.Mutex
	pages    map[int][]Issue
	next     map[int]int
	queries  []IssueQuery
	comments map[int][]Comment
	rate     Rate
	listErr  error
}

func (g *fakeIssues) ListIssues(_ context.Context, _, _ string, q IssueQuery) ([]Issue, int, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.queries = append(g.queries, q)
	if g.listErr != nil {
		return nil, 0, g.listErr
	}
	return g.pages[q.Page], g.next[q.Page], nil
}

func (g *fakeIssues) listed() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.queries)
}

func (g *fakeIssues) ListComments(_ context.Context, _, _ string, number int, _ time.Time) ([]Comment, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.comments[number], nil
}

func (g *fakeIssues) RateLimit(context.Context, string, string) (Rate, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.rate, nil
}

type fakeBot string

func (b fakeBot) BotLogin(context.Context) (string, error) { return string(b), nil }

type fakeRepos map[string]string

func (r fakeRepos) ProjectOf(_ context.Context, owner, repo string) (string, error) {
	p, ok := r[owner+"/"+repo]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return p, nil
}

type fixture struct {
	svc    *Service
	repo   *fakeRepo
	desk   *fakeDesk
	issues *fakeIssues
	now    time.Time
}

func newFixture(syncs ...Sync) *fixture {
	f := &fixture{
		repo: newFakeRepo(), desk: newFakeDesk(),
		issues: &fakeIssues{pages: map[int][]Issue{}, next: map[int]int{}, comments: map[int][]Comment{}, rate: Rate{Remaining: 5000}},
		now:    time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC),
	}
	for _, s := range syncs {
		f.repo.syncs[s.Owner+"/"+s.Repo] = s
	}
	f.svc = New(Deps{
		Repo: f.repo, Desk: f.desk, Board: f.desk, Notes: f.desk, Comments: f.desk,
		People: fakePeople{"bob-gh": "bob"}, Issues: f.issues, Bot: fakeBot("nexul-app[bot]"),
		Repos: fakeRepos{"acme/app": "p-1", "acme/other": "p-2"}, Now: func() time.Time { return f.now },
	})
	return f
}

func onSync() Sync {
	return Sync{ProjectID: "p-1", Owner: "acme", Repo: "app", Label: DefaultLabel, Mode: ModeLabel, Enabled: true}
}

var issueTime = time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)

func newIssue(opts ...func(*Issue)) Issue {
	i := Issue{ID: 9001, Number: 7, Title: "Crash on save", Body: "Steps", State: "open", Labels: []string{"nexul", "bug"}, Author: "alice-gh", UpdatedAt: issueTime}
	for _, o := range opts {
		o(&i)
	}
	return i
}

func withTitle(t string) func(*Issue) { return func(i *Issue) { i.Title = t } }

func withLabels(l ...string) func(*Issue) { return func(i *Issue) { i.Labels = l } }

func later(d time.Duration) func(*Issue) { return func(i *Issue) { i.UpdatedAt = issueTime.Add(d) } }

func closed(reason string) func(*Issue) {
	return func(i *Issue) { i.State, i.StateReason = "closed", reason }
}

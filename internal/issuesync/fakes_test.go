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
	mu            sync.Mutex
	syncs         map[string]Sync
	links         map[linkKey]Link
	comments      map[int64]CommentLink
	swept         []string
	ticketPushes  map[string]TicketPush
	commentPushes map[string]CommentPush
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		syncs: map[string]Sync{}, links: map[linkKey]Link{}, comments: map[int64]CommentLink{},
		ticketPushes: map[string]TicketPush{}, commentPushes: map[string]CommentPush{},
	}
}

func (r *fakeRepo) GetLinkByTicket(_ context.Context, ticketID string) (Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.links {
		if l.TicketID == ticketID {
			return l, nil
		}
	}
	return Link{}, apperrs.ErrNotFound
}

func (r *fakeRepo) GetCommentLinkByMessage(_ context.Context, messageID string) (CommentLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.comments {
		if c.MessageID == messageID {
			return c, nil
		}
	}
	return CommentLink{}, apperrs.ErrNotFound
}

func (r *fakeRepo) QueueTicketPush(_ context.Context, p TicketPush) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	old, ok := r.ticketPushes[p.TicketID]
	if ok {
		p.Create = p.Create || old.Create
		p.Tried = old.Tried
	}
	r.ticketPushes[p.TicketID] = p
	return nil
}

func (r *fakeRepo) QueueCommentPush(_ context.Context, p CommentPush) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.commentPushes[p.MessageID]; ok {
		p.Tried = old.Tried
	}
	r.commentPushes[p.MessageID] = p
	return nil
}

func (r *fakeRepo) DueTicketPushes(_ context.Context, now time.Time, limit int) ([]TicketPush, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []TicketPush
	for _, p := range r.ticketPushes {
		if !p.Due.After(now) {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b TicketPush) int { return a.Due.Compare(b.Due) })
	return out[:min(limit, len(out))], nil
}

func (r *fakeRepo) DueCommentPushes(_ context.Context, now time.Time, limit int) ([]CommentPush, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []CommentPush
	for _, p := range r.commentPushes {
		if !p.Due.After(now) {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b CommentPush) int { return a.Due.Compare(b.Due) })
	return out[:min(limit, len(out))], nil
}

func (r *fakeRepo) NextPushDue(context.Context) (time.Time, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var next time.Time
	for _, p := range r.ticketPushes {
		if next.IsZero() || p.Due.Before(next) {
			next = p.Due
		}
	}
	for _, p := range r.commentPushes {
		if next.IsZero() || p.Due.Before(next) {
			next = p.Due
		}
	}
	return next, !next.IsZero(), nil
}

func (r *fakeRepo) DeferTicketPush(_ context.Context, p TicketPush, was time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.ticketPushes[p.TicketID]
	if !ok {
		return nil
	}
	if !cur.Due.Equal(was) {
		p.Due = cur.Due
	}
	if !cur.Tried.IsZero() {
		p.Tried = cur.Tried
	}
	p.Attempts = max(p.Attempts, cur.Attempts)
	r.ticketPushes[p.TicketID] = p
	return nil
}

func (r *fakeRepo) DeferCommentPush(_ context.Context, p CommentPush, was time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.commentPushes[p.MessageID]
	if !ok {
		return nil
	}
	if !cur.Due.Equal(was) {
		p.Due = cur.Due
	}
	if !cur.Tried.IsZero() {
		p.Tried = cur.Tried
	}
	p.Attempts = max(p.Attempts, cur.Attempts)
	r.commentPushes[p.MessageID] = p
	return nil
}

func (r *fakeRepo) DoneTicketPush(_ context.Context, p TicketPush) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if cur, ok := r.ticketPushes[p.TicketID]; ok && cur.Due.Equal(p.Due) {
		delete(r.ticketPushes, p.TicketID)
	}
	return nil
}

func (r *fakeRepo) DoneCommentPush(_ context.Context, p CommentPush) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if cur, ok := r.commentPushes[p.MessageID]; ok && cur.Due.Equal(p.Due) {
		delete(r.commentPushes, p.MessageID)
	}
	return nil
}

func (r *fakeRepo) waiting() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.ticketPushes) + len(r.commentPushes)
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

// fakeDesk is the tickets side: Desk, Board, Notes, Comments and Source in one.
type fakeDesk struct {
	mu       sync.Mutex
	tickets  map[string]*fakeTicket
	applied  int
	notes    map[string]string
	messages map[string]string
	edits    int
	deleted  []string
	fail     error
	// thread holds the messages written in Nexul, as Source reads them; threads maps a conversation to its ticket.
	thread  map[string]Mirror
	threads map[string]string
}

func newFakeDesk() *fakeDesk {
	return &fakeDesk{
		tickets: map[string]*fakeTicket{}, notes: map[string]string{}, messages: map[string]string{},
		thread: map[string]Mirror{}, threads: map[string]string{"c-1": "t-1"},
	}
}

// githubLogins maps a member's login to the GitHub login they sign in with.
var githubLogins = map[string]string{"bob": "bob-gh"}

func (d *fakeDesk) Ticket(_ context.Context, id string) (Outbound, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	t, ok := d.tickets[id]
	if !ok {
		return Outbound{}, apperrs.ErrNotFound
	}
	f := TicketFields{Title: t.Fields.Title, Body: t.Fields.Body, Closed: t.Stage == StageDone, Labels: slices.Clone(t.Fields.Labels)}
	if gh := githubLogins[t.Developer]; gh != "" {
		f.Assignees = []string{gh}
	}
	return Outbound{ProjectID: "p-1", Fields: f}, nil
}

func (d *fakeDesk) Message(_ context.Context, id string) (Mirror, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	m, ok := d.thread[id]
	if !ok {
		return Mirror{}, apperrs.ErrNotFound
	}
	return m, nil
}

func (d *fakeDesk) ThreadOf(_ context.Context, conversationID string) (string, error) {
	return d.threads[conversationID], nil
}

// edit changes a ticket as a person does in Nexul.
func (d *fakeDesk) edit(id string, change func(*fakeTicket)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	change(d.tickets[id])
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
	github *fakeGitHub
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
	f.github = &fakeGitHub{issues: f.issues, now: func() time.Time { return f.now }}
	f.svc = New(Deps{
		Repo: f.repo, Desk: f.desk, Board: f.desk, Notes: f.desk, Comments: f.desk,
		People: fakePeople{"bob-gh": "bob"}, Issues: f.issues, Bot: fakeBot("nexul-app[bot]"),
		Repos: fakeRepos{"acme/app": "p-1", "acme/other": "p-2"}, Source: f.desk, IssueWriter: f.github, CommentWriter: f.github,
		Now: func() time.Time { return f.now },
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

// fakeGitHub records every write; fail answers the next writes with an error, and accept makes a failed create land
// anyway, as a timed-out request GitHub accepted does.
type fakeGitHub struct {
	mu      sync.Mutex
	issues  *fakeIssues
	now     func() time.Time
	updates []IssueWrite
	creates []IssueWrite
	posted  []string
	edited  []string
	removed []int64
	fail    []error
	accept  bool
	nextID  int64
}

func (g *fakeGitHub) failure() error {
	if len(g.fail) == 0 {
		return nil
	}
	err := g.fail[0]
	g.fail = g.fail[1:]
	return err
}

func (g *fakeGitHub) issue(number int, w IssueWrite) Issue {
	state := "open"
	if w.Closed {
		state = "closed"
	}
	return Issue{ID: 9000 + int64(number), Number: number, Title: w.Title, Body: w.Body, State: state, Labels: w.Labels, Assignees: w.Assignees, UpdatedAt: g.now()}
}

func (g *fakeGitHub) CreateIssue(_ context.Context, _, _ string, w IssueWrite) (Issue, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	err := g.failure()
	if err != nil && !g.accept {
		return Issue{}, err
	}
	g.creates = append(g.creates, w)
	i := g.issue(40+len(g.creates), w)
	i.State = "open"
	g.issues.mu.Lock()
	g.issues.pages[1] = append(g.issues.pages[1], i)
	g.issues.mu.Unlock()
	return i, err
}

func (g *fakeGitHub) UpdateIssue(_ context.Context, _, _ string, number int, w IssueWrite) (Issue, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.failure(); err != nil {
		return Issue{}, err
	}
	g.updates = append(g.updates, w)
	return g.issue(number, w), nil
}

func (g *fakeGitHub) CreateComment(_ context.Context, _, _ string, number int, body string) (Comment, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	err := g.failure()
	if err != nil && !g.accept {
		return Comment{}, err
	}
	g.nextID++
	c := Comment{ID: 7000 + g.nextID, Body: body, Author: "nexul-app[bot]", UpdatedAt: g.now()}
	g.posted = append(g.posted, body)
	g.issues.mu.Lock()
	g.issues.comments[number] = append(g.issues.comments[number], c)
	g.issues.mu.Unlock()
	return c, err
}

func (g *fakeGitHub) UpdateComment(_ context.Context, _, _ string, _ int64, body string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.failure(); err != nil {
		return err
	}
	g.edited = append(g.edited, body)
	return nil
}

func (g *fakeGitHub) DeleteComment(_ context.Context, _, _ string, id int64) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.failure(); err != nil {
		return err
	}
	g.removed = append(g.removed, id)
	return nil
}

func (g *fakeGitHub) writes() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.updates) + len(g.creates) + len(g.posted) + len(g.edited) + len(g.removed)
}

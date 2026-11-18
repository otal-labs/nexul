package issuesync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Tickets is the tickets side of the sync (ADR 0017).
type Tickets interface {
	Get(ctx context.Context, id string) (Ticket, error)
	// Create files the ticket in the project's first backlog column, or its first column when it has none; an id returned
	// with an error is a ticket that exists.
	Create(ctx context.Context, projectID string, t NewTicket) (string, error)
	// Move puts the ticket in the project's first column of stage.
	Move(ctx context.Context, id string, stage Stage) error
}

// TicketFields writes the fields a ticket follows from its issue.
type TicketFields interface {
	// SetText writes the title and Markdown body through the ticket's update path, which resets an open editor.
	SetText(ctx context.Context, id, title, body string) error
	SetLabels(ctx context.Context, id string, labels []string) error
	SetDeveloper(ctx context.Context, id, login string) error
}

// Thread posts on a ticket's thread.
type Thread interface {
	// Note posts a system note once per key.
	Note(ctx context.Context, ticketID, key, body string) error
}

// Comments mirrors GitHub comments as ticket-thread messages by an external author.
type Comments interface {
	// Post returns the message's id; posting the same comment again returns the same message.
	Post(ctx context.Context, ticketID string, c Comment) (string, error)
	Edit(ctx context.Context, messageID string, c Comment) error
	Delete(ctx context.Context, messageID, author string) error
}

// Members maps a GitHub login to the login of a project member who signed in with it.
type Members interface {
	MemberLogin(ctx context.Context, projectID, githubLogin string) (string, bool, error)
}

// IssueQuery is one page of a repository's issues.
type IssueQuery struct {
	// State is open or all.
	State  string
	Labels []string
	Since  time.Time
	Page   int
}

// IssuePage is one page of issues and the rate limit left after reading it.
type IssuePage struct {
	Issues []Issue
	// NextPage is 0 after the last page.
	NextPage      int
	RateRemaining int
	// RateReset is when the limit refills; zero when the host did not say.
	RateReset time.Time
}

// Host is the git host the issues live on.
type Host interface {
	// BotLogin is the sender login of the App's own changes, such as nexul[bot]; empty while there is no App.
	BotLogin(ctx context.Context) (string, error)
	ListIssues(ctx context.Context, owner, repo string, q IssueQuery) (IssuePage, error)
}

// Gate checks the caller's permission in a project.
type Gate interface {
	RequireProject(ctx context.Context, projectID string, action permissions.Action) error
}

// ProjectRepos says whether a repository is attached to a project.
type ProjectRepos interface {
	HasRepo(ctx context.Context, projectID, owner, repo string) (bool, error)
}

// Deps are the sync's collaborators.
type Deps struct {
	Repo     Repo
	Tickets  Tickets
	Fields   TicketFields
	Thread   Thread
	Comments Comments
	Members  Members
	Host     Host
	Gate     Gate
	Repos    ProjectRepos
	Log      *slog.Logger
}

// SweepInterval is how often every enabled sync lists its recently updated issues.
const SweepInterval = 15 * time.Minute

// minRateRemaining is the GitHub rate limit the sweep leaves for everything else.
const minRateRemaining = 200

// Service is the issue sync.
type Service struct {
	Deps
	now  func() time.Time
	kick chan struct{}
	// shortcut: one lock for every issue, so a webhook and a sweep never file two tickets; per-issue locks if volume grows.
	mu sync.Mutex
	// pausedUntil is written only by the sweep, which runs on one goroutine.
	pausedUntil time.Time
}

// NewService wires the sync.
func NewService(d Deps) *Service {
	if d.Log == nil {
		d.Log = slog.Default()
	}
	return &Service{Deps: d, now: time.Now, kick: make(chan struct{}, 1)}
}

// Syncs lists a project's issue syncs.
func (s *Service) Syncs(ctx context.Context, projectID string) ([]*Sync, error) {
	if err := s.Gate.RequireProject(ctx, projectID, permissions.ProjectsRead); err != nil {
		return nil, err
	}
	return s.Repo.SyncsOfProject(ctx, projectID)
}

// Set creates or changes the sync of one of the project's repositories; turning it on links its open issues soon after.
func (s *Service) Set(ctx context.Context, projectID string, in Input) (*Sync, error) {
	if err := s.Gate.RequireProject(ctx, projectID, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	owner, repo := strings.TrimSpace(in.Owner), strings.TrimSpace(in.Repo)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("%w: an issue sync needs the repository's owner and name", apperrs.ErrInvalid)
	}
	has, err := s.Repos.HasRepo(ctx, projectID, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("issue sync for %s/%s: %w", owner, repo, err)
	}
	if !has {
		return nil, fmt.Errorf("%w: %s/%s is not a repository of this project; project_get lists them", apperrs.ErrInvalid, owner, repo)
	}
	cur, err := s.Repo.GetSync(ctx, owner, repo)
	if errors.Is(err, apperrs.ErrNotFound) {
		cur, err = &Sync{ProjectID: projectID, Owner: owner, Repo: repo, Label: DefaultLabel, Mode: ModeLabel}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("issue sync for %s/%s: %w", owner, repo, err)
	}
	next, err := patched(*cur, in)
	if err != nil {
		return nil, err
	}
	if err := s.Repo.SaveSync(ctx, &next); err != nil {
		return nil, fmt.Errorf("save issue sync for %s/%s: %w", owner, repo, err)
	}
	if next.Enabled {
		s.wake()
	}
	return &next, nil
}

func patched(sync Sync, in Input) (Sync, error) {
	if in.Label != nil {
		sync.Label = strings.TrimSpace(*in.Label)
	}
	if sync.Label == "" {
		return Sync{}, fmt.Errorf("%w: an issue sync's label cannot be empty", apperrs.ErrInvalid)
	}
	if in.Mode != nil {
		sync.Mode = Mode(strings.TrimSpace(*in.Mode))
	}
	if sync.Mode != ModeLabel && sync.Mode != ModeAll {
		return Sync{}, fmt.Errorf("%w: an issue sync's mode is label or all", apperrs.ErrInvalid)
	}
	if in.Enabled != nil {
		sync.Enabled = *in.Enabled
	}
	return sync, nil
}

func (s *Service) wake() {
	select {
	case s.kick <- struct{}{}:
	default:
	}
}

// receive applies one issue's state; action is the webhook's, empty for a sweep. sender is empty for a sweep.
func (s *Service) receive(ctx context.Context, owner, repo, action, sender string, issue Issue) error {
	sync, ok, err := s.enabledSync(ctx, owner, repo)
	if err != nil || !ok {
		return err
	}
	if bot, err := s.fromBot(ctx, sender); err != nil || bot {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	link, err := s.link(ctx, owner, repo, issue.Number)
	if err != nil {
		return err
	}
	if action == "deleted" || action == "transferred" {
		return s.unlink(ctx, link, action)
	}
	if link == nil {
		return s.file(ctx, sync, action, issue)
	}
	return s.update(ctx, sync, link, action, issue)
}

func (s *Service) enabledSync(ctx context.Context, owner, repo string) (*Sync, bool, error) {
	sync, err := s.Repo.GetSync(ctx, owner, repo)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("issue sync for %s/%s: %w", owner, repo, err)
	}
	return sync, sync.Enabled, nil
}

// fromBot reports a change the App made itself, which the outbound half already wrote.
func (s *Service) fromBot(ctx context.Context, sender string) (bool, error) {
	if sender == "" {
		return false, nil
	}
	bot, err := s.Host.BotLogin(ctx)
	if err != nil {
		return false, fmt.Errorf("read the App's bot login: %w", err)
	}
	return bot != "" && strings.EqualFold(bot, sender), nil
}

func (s *Service) link(ctx context.Context, owner, repo string, number int) (*Link, error) {
	l, err := s.Repo.GetLink(ctx, owner, repo, number)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("link of %s/%s#%d: %w", owner, repo, number, err)
	}
	return l, nil
}

// unlink drops the link of a deleted or transferred issue and keeps its ticket.
func (s *Service) unlink(ctx context.Context, link *Link, action string) error {
	if link == nil {
		return nil
	}
	what := "deleted"
	if action == "transferred" {
		what = "moved to another repository"
	}
	body := fmt.Sprintf("GitHub issue %s/%s#%d was %s, so this ticket no longer follows it.", link.Owner, link.Repo, link.Number, what)
	if err := s.Thread.Note(ctx, link.TicketID, noteKey(link.Owner, link.Repo, link.Number, action), body); err != nil {
		return err
	}
	return s.Repo.DeleteLink(ctx, link.Owner, link.Repo, link.Number)
}

// file creates the ticket of a newly matching issue and links it before anything else can fail, so a retry finds the
// link and never files a second one.
func (s *Service) file(ctx context.Context, sync *Sync, action string, issue Issue) error {
	if !sync.Matches(issue.Labels) {
		return nil
	}
	id, createErr := s.Tickets.Create(ctx, sync.ProjectID, NewTicket{Title: issue.Title, Body: issue.Body, Reporter: issue.Author})
	if id == "" {
		return fmt.Errorf("file a ticket for %s/%s#%d: %w", sync.Owner, sync.Repo, issue.Number, createErr)
	}
	link := &Link{Owner: sync.Owner, Repo: sync.Repo, Number: issue.Number, IssueID: issue.ID, TicketID: id, LinkedAt: s.now().UTC()}
	if err := s.Repo.SaveLink(ctx, link); err != nil {
		return fmt.Errorf("link %s/%s#%d: %w", sync.Owner, sync.Repo, issue.Number, err)
	}
	if createErr != nil {
		return fmt.Errorf("file a ticket for %s/%s#%d: %w", sync.Owner, sync.Repo, issue.Number, createErr)
	}
	return s.follow(ctx, sync, link, action, issue)
}

func (s *Service) update(ctx context.Context, sync *Sync, link *Link, action string, issue Issue) error {
	if issue.UpdatedAt.Before(link.IssueUpdatedAt) {
		return nil
	}
	// Every state a delivery leaves records its time, so an older one arriving later never undoes a pause or resume.
	if !sync.Matches(issue.Labels) {
		if link.Paused {
			return nil
		}
		link.Paused, link.IssueUpdatedAt = true, issue.UpdatedAt.UTC()
		return s.Repo.SaveLink(ctx, link)
	}
	resumed := link.Paused
	link.Paused = false
	if hash := issue.Hash(); hash == link.InboundHash || hash == link.OutboundHash {
		if !resumed && !issue.UpdatedAt.After(link.IssueUpdatedAt) {
			return nil
		}
		link.IssueUpdatedAt = issue.UpdatedAt.UTC()
		return s.Repo.SaveLink(ctx, link)
	}
	return s.follow(ctx, sync, link, action, issue)
}

// follow writes the issue onto its ticket, each field only when it differs, then records what it followed.
func (s *Service) follow(ctx context.Context, sync *Sync, link *Link, action string, issue Issue) error {
	t, err := s.Tickets.Get(ctx, link.TicketID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read ticket %s: %w", link.TicketID, err)
	}
	if err := s.followFields(ctx, sync, t, issue); err != nil {
		return fmt.Errorf("follow %s/%s#%d onto ticket %s: %w", sync.Owner, sync.Repo, issue.Number, t.ID, err)
	}
	if err := s.followState(ctx, t, link, action, issue); err != nil {
		return fmt.Errorf("follow the state of %s/%s#%d onto ticket %s: %w", sync.Owner, sync.Repo, issue.Number, t.ID, err)
	}
	link.InboundHash, link.IssueUpdatedAt = issue.Hash(), issue.UpdatedAt.UTC()
	return s.Repo.SaveLink(ctx, link)
}

func (s *Service) followFields(ctx context.Context, sync *Sync, t Ticket, issue Issue) error {
	if t.Title != issue.Title || t.Body != issue.Body {
		if err := s.Fields.SetText(ctx, t.ID, issue.Title, issue.Body); err != nil {
			return err
		}
	}
	labels := ticketLabels(sync, issue.Labels)
	if !sameLabels(t.Labels, labels) {
		if err := s.Fields.SetLabels(ctx, t.ID, labels); err != nil {
			return err
		}
	}
	dev, ok, err := s.developer(ctx, t.ProjectID, issue.Assignees)
	if err != nil || !ok || dev == t.Developer {
		return err
	}
	return s.Fields.SetDeveloper(ctx, t.ID, dev)
}

// ticketLabels are the issue's labels but the one that links it.
func ticketLabels(sync *Sync, labels []string) []string {
	out := make([]string, 0, len(labels))
	for _, l := range labels {
		if sync.Mode == ModeLabel && strings.EqualFold(l, sync.Label) {
			continue
		}
		out = append(out, l)
	}
	return out
}

func sameLabels(a, b []string) bool {
	fold := func(ls []string) []string {
		out := make([]string, len(ls))
		for i, l := range ls {
			out[i] = strings.ToLower(l)
		}
		slices.Sort(out)
		return slices.Compact(out)
	}
	return slices.Equal(fold(a), fold(b))
}

// developer is the first assignee who maps to a project member; the rest are ignored.
func (s *Service) developer(ctx context.Context, projectID string, assignees []string) (string, bool, error) {
	for _, a := range assignees {
		login, ok, err := s.Members.MemberLogin(ctx, projectID, a)
		if err != nil {
			return "", false, fmt.Errorf("map assignee %s: %w", a, err)
		}
		if ok {
			return login, true, nil
		}
	}
	return "", false, nil
}

// followState closes a ticket with its issue, and never takes a done ticket out of done (ADR 0064).
func (s *Service) followState(ctx context.Context, t Ticket, link *Link, action string, issue Issue) error {
	key := noteKey(link.Owner, link.Repo, link.Number, action+"\x00"+issue.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if issue.Closed() {
		if t.Done {
			return nil
		}
		if err := s.Thread.Note(ctx, t.ID, key, closedNote(issue.StateReason)); err != nil {
			return err
		}
		return s.Tickets.Move(ctx, t.ID, StageDone)
	}
	if action != "reopened" {
		return nil
	}
	if t.Done {
		return s.Thread.Note(ctx, t.ID, key, "Reopened on GitHub. Done tickets stay done; report a bug to follow up.")
	}
	return s.Tickets.Move(ctx, t.ID, StageBacklog)
}

func closedNote(reason string) string {
	if reason == "" {
		return "Closed on GitHub."
	}
	return "Closed on GitHub as " + strings.ReplaceAll(reason, "_", " ") + "."
}

func noteKey(owner, repo string, number int, what string) string {
	return fmt.Sprintf("github-issue\x00%s/%s#%d\x00%s", owner, repo, number, what)
}

// receiveComment mirrors one comment onto a linked, unpaused issue's ticket thread.
func (s *Service) receiveComment(ctx context.Context, owner, repo string, number int, action, sender string, c Comment) error {
	_, ok, err := s.enabledSync(ctx, owner, repo)
	if err != nil || !ok {
		return err
	}
	if bot, err := s.fromBot(ctx, sender); err != nil || bot {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	link, err := s.link(ctx, owner, repo, number)
	if err != nil || link == nil || link.Paused {
		return err
	}
	cl, err := s.Repo.GetCommentLink(ctx, c.ID)
	if errors.Is(err, apperrs.ErrNotFound) {
		if action != "created" {
			return nil
		}
		return s.postComment(ctx, link.TicketID, c)
	}
	if err != nil {
		return fmt.Errorf("comment link %d: %w", c.ID, err)
	}
	return s.followComment(ctx, cl, action, c)
}

// followComment carries an edit or delete onto the message a comment already became; a repeated create changes nothing.
func (s *Service) followComment(ctx context.Context, cl *CommentLink, action string, c Comment) error {
	if action == "edited" {
		return s.Comments.Edit(ctx, cl.MessageID, c)
	}
	if action != "deleted" {
		return nil
	}
	if err := s.Comments.Delete(ctx, cl.MessageID, c.Author); err != nil {
		return err
	}
	return s.Repo.DeleteCommentLink(ctx, c.ID)
}

func (s *Service) postComment(ctx context.Context, ticketID string, c Comment) error {
	id, err := s.Comments.Post(ctx, ticketID, c)
	if err != nil {
		return fmt.Errorf("post comment %d: %w", c.ID, err)
	}
	return s.Repo.SaveCommentLink(ctx, CommentLink{CommentID: c.ID, MessageID: id, TicketID: ticketID})
}

// Run sweeps at once, then every SweepInterval and whenever a sync is turned on, until ctx ends.
func (s *Service) Run(ctx context.Context) {
	// GitHub never redelivers a missed webhook, so listing the issues is the only way to catch one.
	tick := time.NewTicker(SweepInterval)
	defer tick.Stop()
	for {
		if err := s.Sweep(ctx); err != nil && ctx.Err() == nil {
			s.Log.WarnContext(ctx, "issue sync sweep failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		case <-s.kick:
		}
	}
}

// Sweep lists each due sync's issues: a new sync's open matching ones, else every issue updated since its last sweep.
// It stops while the host's rate limit is low, until it refills.
func (s *Service) Sweep(ctx context.Context) error {
	if s.now().Before(s.pausedUntil) {
		return nil
	}
	syncs, err := s.Repo.EnabledSyncs(ctx)
	if err != nil {
		return fmt.Errorf("list enabled issue syncs: %w", err)
	}
	var errs []error
	for _, sync := range syncs {
		if !s.due(sync) {
			continue
		}
		err := s.sweep(ctx, sync)
		if errors.Is(err, errRateLow) {
			return nil
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// due is a sync never swept, or one last swept at least half an interval ago, so a wake on turning one on reads only it.
func (s *Service) due(sync *Sync) bool {
	return sync.LastSweptAt == nil || s.now().Sub(*sync.LastSweptAt) >= SweepInterval/2
}

var errRateLow = errors.New("rate limit low")

func (s *Service) sweep(ctx context.Context, sync *Sync) error {
	started := s.now().UTC()
	q := IssueQuery{State: "all"}
	if sync.LastSweptAt != nil {
		q.Since = *sync.LastSweptAt
	}
	if sync.LastSweptAt == nil {
		q.State = "open"
	}
	if sync.LastSweptAt == nil && sync.Mode == ModeLabel {
		q.Labels = []string{sync.Label}
	}
	for page := 1; page != 0; {
		q.Page = page
		p, err := s.Host.ListIssues(ctx, sync.Owner, sync.Repo, q)
		if wait, ok := retryAfter(err); ok {
			s.pause(ctx, s.now().Add(wait))
			return errRateLow
		}
		if err != nil {
			return fmt.Errorf("sweep %s/%s: %w", sync.Owner, sync.Repo, err)
		}
		for _, issue := range p.Issues {
			if err := s.receive(ctx, sync.Owner, sync.Repo, "", "", issue); err != nil {
				return fmt.Errorf("sweep %s/%s#%d: %w", sync.Owner, sync.Repo, issue.Number, err)
			}
		}
		if !p.RateReset.IsZero() && p.RateRemaining < minRateRemaining {
			s.pause(ctx, p.RateReset)
			return errRateLow
		}
		page = p.NextPage
	}
	return s.Repo.MarkSwept(ctx, sync.Owner, sync.Repo, started)
}

func (s *Service) pause(ctx context.Context, until time.Time) {
	s.pausedUntil = until
	s.Log.InfoContext(ctx, "issue sync sweep paused for the GitHub rate limit", "until", until)
}

func retryAfter(err error) (time.Duration, bool) {
	var ra *apperrs.RetryAfterError
	if errors.As(err, &ra) {
		return ra.Wait, true
	}
	return 0, false
}

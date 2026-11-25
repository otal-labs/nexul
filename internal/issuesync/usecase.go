package issuesync

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Stage names the board stage a ticket moves to.
type Stage string

const (
	StageBacklog Stage = "backlog"
	StageDone    Stage = "done"
)

// Desk is the tickets side of the sync (ADR 0017).
type Desk interface {
	// File creates the ticket in the project's first backlog column, else its first, reported by the GitHub login.
	File(ctx context.Context, projectID string, f TicketFields, developer, reporter string) (string, error)
	// Apply sets the ticket's title, body and labels to f's, and its developer when developer is set.
	Apply(ctx context.Context, ticketID string, f TicketFields, developer string) error
}

// Board reads and moves a ticket's stage.
type Board interface {
	Done(ctx context.Context, ticketID string) (bool, error)
	// MoveTo moves the ticket to the first column of stage; a project without one leaves it where it is.
	MoveTo(ctx context.Context, ticketID string, stage Stage) error
}

// Notes posts system notes on a ticket's thread.
type Notes interface {
	// Note posts body once per key.
	Note(ctx context.Context, ticketID, key, body string) error
}

// Comments mirrors GitHub comments as external messages on a ticket's thread.
type Comments interface {
	// Post returns the new message's id; posting the same comment again returns the same id.
	Post(ctx context.Context, ticketID string, c Comment) (string, error)
	Edit(ctx context.Context, messageID string, c Comment) error
	Delete(ctx context.Context, messageID string, c Comment) error
}

// People maps a GitHub login to the login of a member who may open the project, empty when none does.
type People interface {
	Member(ctx context.Context, projectID, githubLogin string) (string, error)
}

// IssueQuery filters a listing; Page starts at 1.
type IssueQuery struct {
	State  string
	Labels []string
	Since  time.Time
	Page   int
}

// Rate is what is left of the token's GitHub rate limit, and when it resets.
type Rate struct {
	Remaining int
	Reset     time.Time
}

// Issues reads a repository's issues from GitHub for the backfill and the sweep.
type Issues interface {
	// ListIssues returns one page and the next page number, 0 after the last.
	ListIssues(ctx context.Context, owner, repo string, q IssueQuery) ([]Issue, int, error)
	ListComments(ctx context.Context, owner, repo string, number int, since time.Time) ([]Comment, error)
	RateLimit(ctx context.Context, owner, repo string) (Rate, error)
}

// Bot names the GitHub App's own account, "<app_slug>[bot]", empty before an App is set up.
type Bot interface {
	BotLogin(ctx context.Context) (string, error)
}

// Repos answers which project a repository is attached to.
type Repos interface {
	ProjectOf(ctx context.Context, owner, repo string) (string, error)
}

// Gate checks the caller's project permissions.
type Gate interface {
	RequireProject(ctx context.Context, projectID string, action permissions.Action) error
}

// Deps are the Service's seams; Gate may be nil in tests, which leaves the calls ungated.
type Deps struct {
	Repo          Repo
	Desk          Desk
	Board         Board
	Notes         Notes
	Comments      Comments
	People        People
	Issues        Issues
	Bot           Bot
	Repos         Repos
	Gate          Gate
	Source        Source
	IssueWriter   IssueWriter
	CommentWriter CommentWriter
	Now           func() time.Time
	// Quiet is how long a ticket's changes rest before they are written; zero is the default, Quiet.
	Quiet time.Duration
	// Retry is the wait after GitHub first fails a write, doubling with each failure; zero is 30 seconds.
	Retry time.Duration
}

// rateFloor is the remaining requests under which the sweep stops until the limit resets, leaving the rest to
// webhooks' follow-ups and the people sharing the token.
const rateFloor = 200

// Service applies GitHub issues to tickets.
type Service struct {
	d Deps
	// locks serialize applying and writing each issue, so a webhook, the sweep and a write never interleave on one.
	locks keyedLocks
	pace  *pacer
}

// New builds the Service.
func New(d Deps) *Service {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Quiet == 0 {
		d.Quiet = Quiet
	}
	if d.Retry == 0 {
		d.Retry = pushRetry
	}
	return &Service{d: d, pace: newPacer()}
}

func (s *Service) require(ctx context.Context, projectID string, action permissions.Action) error {
	if s.d.Gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.d.Gate.RequireProject(ctx, projectID, action)
}

// Syncs lists a project's issue syncs, one per repository sync was ever turned on for.
func (s *Service) Syncs(ctx context.Context, projectID string) ([]Sync, error) {
	if err := s.require(ctx, projectID, permissions.ProjectsRead); err != nil {
		return nil, err
	}
	return s.d.Repo.ListSyncsByProject(ctx, projectID)
}

// Set changes the sync of one of the project's repositories; turning it on links the open issues it matches, in the
// background loop, which the stored change wakes.
func (s *Service) Set(ctx context.Context, projectID string, c Change) (Sync, error) {
	if err := s.require(ctx, projectID, permissions.ProjectsWrite); err != nil {
		return Sync{}, err
	}
	c.Owner, c.Repo = strings.TrimSpace(c.Owner), strings.TrimSpace(c.Repo)
	attachedTo, err := s.d.Repos.ProjectOf(ctx, c.Owner, c.Repo)
	if errors.Is(err, apperrs.ErrNotFound) || (err == nil && attachedTo != projectID) {
		return Sync{}, fmt.Errorf("%w: %s/%s is not a repository of this project; project_get lists them", apperrs.ErrInvalid, c.Owner, c.Repo)
	}
	if err != nil {
		return Sync{}, err
	}
	current, err := s.d.Repo.GetSync(ctx, c.Owner, c.Repo)
	if errors.Is(err, apperrs.ErrNotFound) {
		current, err = Sync{Owner: c.Owner, Repo: c.Repo, Label: DefaultLabel, Mode: ModeLabel}, nil
	}
	if err != nil {
		return Sync{}, err
	}
	wasOn := current.Enabled
	next, err := current.apply(c)
	if err != nil {
		return Sync{}, err
	}
	next.ProjectID = projectID
	if next.Enabled && (!wasOn || next.Label != current.Label || next.Mode != current.Mode) {
		next.LastSweptAt = nil
	}
	if err := s.d.Repo.PutSync(ctx, next); err != nil {
		return Sync{}, err
	}
	return next, nil
}

// enabled is the repository's sync while it is on; a repository without one, or with it off, changes nothing.
func (s *Service) enabled(ctx context.Context, owner, repo string) (Sync, bool, error) {
	sy, err := s.d.Repo.GetSync(ctx, owner, repo)
	if errors.Is(err, apperrs.ErrNotFound) {
		return Sync{}, false, nil
	}
	if err != nil {
		return Sync{}, false, err
	}
	return sy, sy.Enabled, nil
}

// fromBot reports whether sender is the App's own account, whose changes are Nexul's own echoed back.
func (s *Service) fromBot(ctx context.Context, sender string) (bool, error) {
	if s.d.Bot == nil || sender == "" {
		return false, nil
	}
	bot, err := s.d.Bot.BotLogin(ctx)
	if err != nil {
		return false, err
	}
	return bot != "" && strings.EqualFold(sender, bot), nil
}

// HandleIssue applies one git.issue delivery.
func (s *Service) HandleIssue(ctx context.Context, owner, repo, action string, i Issue, sender string) error {
	sy, on, err := s.enabled(ctx, owner, repo)
	if err != nil || !on {
		return err
	}
	bot, err := s.fromBot(ctx, sender)
	if err != nil || bot {
		return err
	}
	defer s.lockIssue(owner, repo, i)()
	return s.apply(ctx, sy, i, action)
}

// apply brings the issue's ticket in step with it, filing one for an issue the sync newly covers; callers hold its lock.
func (s *Service) apply(ctx context.Context, sy Sync, i Issue, action string) error {
	link, err := s.d.Repo.GetLink(ctx, sy.Owner, sy.Repo, i.Number)
	linked := err == nil
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return err
	}
	if action == "deleted" || action == "transferred" {
		if !linked {
			return nil
		}
		return s.unlink(ctx, link, action)
	}
	covered := sy.syncs(i)
	if !linked && !covered {
		return nil
	}
	if !linked {
		return s.fileOrClaim(ctx, sy, i)
	}
	return s.follow(ctx, sy, link, i, action, covered)
}

// fileOrClaim links an issue Nexul made to the ticket its marker names, when its own link was lost before it was
// stored; any other issue files a new ticket.
func (s *Service) fileOrClaim(ctx context.Context, sy Sync, i Issue) error {
	ticketID := markedTicket(i.Body)
	if ticketID == "" {
		return s.file(ctx, sy, i)
	}
	out, err := s.d.Source.Ticket(ctx, ticketID)
	if errors.Is(err, apperrs.ErrNotFound) || (err == nil && out.ProjectID != sy.ProjectID) {
		return s.file(ctx, sy, i)
	}
	if err != nil {
		return err
	}
	_, err = s.d.Repo.GetLinkByTicket(ctx, ticketID)
	if err == nil {
		return s.file(ctx, sy, i)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return err
	}
	link := Link{Owner: sy.Owner, Repo: sy.Repo, IssueNumber: i.Number, IssueID: i.ID, TicketID: ticketID, IssueUpdatedAt: i.UpdatedAt, LinkedAt: s.d.Now().UTC()}
	link.InboundHash = Hash(FieldsFromIssue(i, sy.trigger()))
	return s.inStep(ctx, sy, link)
}

// keepTrigger is f with the trigger label added back when the ticket carries it, as a ticket that made its issue does,
// so an edit on GitHub never takes from the ticket the label that says why it syncs.
func (s *Service) keepTrigger(ctx context.Context, sy Sync, ticketID string, f TicketFields) (TicketFields, error) {
	t := sy.trigger()
	if t == "" {
		return f, nil
	}
	cur, err := s.d.Source.Ticket(ctx, ticketID)
	if err != nil {
		return TicketFields{}, err
	}
	i := slices.IndexFunc(cur.Fields.Labels, func(l string) bool { return strings.EqualFold(l, t) })
	if i < 0 {
		return f, nil
	}
	f.Labels = append(slices.Clone(f.Labels), cur.Fields.Labels[i])
	return f, nil
}

// inStep stores link with the ticket as it stands, so the changes just applied to it are never written back.
func (s *Service) inStep(ctx context.Context, sy Sync, link Link) error {
	f, err := s.outbound(ctx, link.TicketID, sy)
	if err != nil {
		return err
	}
	link.TicketHash = Hash(f)
	return s.d.Repo.PutLink(ctx, link)
}

// follow applies a linked issue: a label-mode issue without its label pauses, and one with it again resumes.
func (s *Service) follow(ctx context.Context, sy Sync, link Link, i Issue, action string, covered bool) error {
	if !covered {
		if link.Paused {
			return nil
		}
		link.Paused = true
		return s.d.Repo.PutLink(ctx, link)
	}
	resumed := link.Paused
	link.Paused = false
	if !resumed && i.UpdatedAt.Before(link.IssueUpdatedAt) {
		return nil
	}
	f := FieldsFromIssue(i, sy.trigger())
	h := Hash(f)
	if h == link.InboundHash || h == link.OutboundHash {
		if !resumed {
			return nil
		}
		return s.d.Repo.PutLink(ctx, link)
	}
	dev, err := s.developer(ctx, sy.ProjectID, f.Assignees)
	if err != nil {
		return err
	}
	apply, err := s.keepTrigger(ctx, sy, link.TicketID, f)
	if err != nil {
		return err
	}
	if err := s.d.Desk.Apply(ctx, link.TicketID, apply, dev); err != nil {
		return err
	}
	if err := s.state(ctx, link, f, action, i.UpdatedAt); err != nil {
		return err
	}
	link.InboundHash, link.IssueUpdatedAt = h, i.UpdatedAt
	return s.inStep(ctx, sy, link)
}

func (s *Service) file(ctx context.Context, sy Sync, i Issue) error {
	f := FieldsFromIssue(i, sy.trigger())
	dev, err := s.developer(ctx, sy.ProjectID, f.Assignees)
	if err != nil {
		return err
	}
	ticketID, err := s.d.Desk.File(ctx, sy.ProjectID, f, dev, i.Author)
	if err != nil {
		return err
	}
	link := Link{
		Owner: sy.Owner, Repo: sy.Repo, IssueNumber: i.Number, IssueID: i.ID, TicketID: ticketID,
		InboundHash: Hash(f), IssueUpdatedAt: i.UpdatedAt, LinkedAt: s.d.Now().UTC(),
	}
	if err := s.d.Repo.PutLink(ctx, link); err != nil {
		return err
	}
	if f.Closed {
		if err := s.state(ctx, link, f, "", i.UpdatedAt); err != nil {
			return err
		}
	}
	return s.inStep(ctx, sy, link)
}

// developer is the first assignee a project member holds; the rest, and logins nobody holds, are ignored.
func (s *Service) developer(ctx context.Context, projectID string, assignees []string) (string, error) {
	for _, a := range assignees {
		login, err := s.d.People.Member(ctx, projectID, a)
		if err != nil {
			return "", err
		}
		if login != "" {
			return login, nil
		}
	}
	return "", nil
}

// state moves a closed issue's ticket to done, and a reopened one's back to backlog unless it is done, which it
// never leaves (ADR 0064); either way the thread says why.
func (s *Service) state(ctx context.Context, link Link, f TicketFields, action string, at time.Time) error {
	done, err := s.d.Board.Done(ctx, link.TicketID)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("github-issue %s/%s#%d %s %d", link.Owner, link.Repo, link.IssueNumber, action, at.Unix())
	if f.Closed && !done {
		if err := s.d.Board.MoveTo(ctx, link.TicketID, StageDone); err != nil {
			return err
		}
		return s.d.Notes.Note(ctx, link.TicketID, key, closedNote(link, f.StateReason))
	}
	if f.Closed || action != "reopened" {
		return nil
	}
	if done {
		return s.d.Notes.Note(ctx, link.TicketID, key, fmt.Sprintf("%s was reopened on GitHub. Done tickets stay done, so report a bug to follow up.", issueRef(link)))
	}
	return s.d.Board.MoveTo(ctx, link.TicketID, StageBacklog)
}

func closedNote(link Link, reason string) string {
	why := strings.ReplaceAll(reason, "_", " ")
	if why == "" {
		return fmt.Sprintf("%s was closed on GitHub, so this ticket moved to done.", issueRef(link))
	}
	return fmt.Sprintf("%s was closed on GitHub as %s, so this ticket moved to done.", issueRef(link), why)
}

func issueRef(link Link) string {
	return fmt.Sprintf("GitHub issue %s/%s#%d", link.Owner, link.Repo, link.IssueNumber)
}

// unlink drops a deleted or transferred issue's link; the ticket stays, and its thread says why it stopped syncing.
func (s *Service) unlink(ctx context.Context, link Link, action string) error {
	if err := s.d.Repo.DeleteLink(ctx, link); err != nil {
		return err
	}
	key := fmt.Sprintf("github-issue %s/%s#%d %s", link.Owner, link.Repo, link.IssueNumber, action)
	return s.d.Notes.Note(ctx, link.TicketID, key, fmt.Sprintf("%s was %s, so it no longer syncs. The ticket stays.", issueRef(link), action))
}

// HandleComment applies one git.issue_comment delivery to the linked ticket's thread.
func (s *Service) HandleComment(ctx context.Context, owner, repo string, number int, action string, c Comment, sender string) error {
	_, on, err := s.enabled(ctx, owner, repo)
	if err != nil || !on {
		return err
	}
	bot, err := s.fromBot(ctx, sender)
	if err != nil || bot {
		return err
	}
	defer s.locks.lock(issueKey(owner, repo, number))()
	link, err := s.d.Repo.GetLink(ctx, owner, repo, number)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil || link.Paused {
		return err
	}
	return s.comment(ctx, link, action, c)
}

// comment posts a new comment, edits or deletes a known one, and posts an edit of a comment it never saw; callers hold the issue's lock.
func (s *Service) comment(ctx context.Context, link Link, action string, c Comment) error {
	cl, err := s.d.Repo.GetCommentLink(ctx, c.ID)
	known := err == nil
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return err
	}
	if action == "deleted" {
		if !known {
			return nil
		}
		if err := s.d.Comments.Delete(ctx, cl.MessageID, c); err != nil {
			return err
		}
		return s.d.Repo.DeleteCommentLink(ctx, c.ID)
	}
	if known && action == "edited" {
		return s.d.Comments.Edit(ctx, cl.MessageID, c)
	}
	if known {
		return nil
	}
	messageID, err := s.d.Comments.Post(ctx, link.TicketID, c)
	if err != nil {
		return err
	}
	return s.d.Repo.PutCommentLink(ctx, CommentLink{GitHubCommentID: c.ID, MessageID: messageID, TicketID: link.TicketID})
}

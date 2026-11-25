package issuesync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Quiet is how long a ticket's changes rest before they are written to its issue, so a burst of edits makes one write.
const Quiet = 10 * time.Second

const (
	pushBatch = 50
	// pushRetry is the loop's wait after the store failed it, and the default first retry of a write GitHub failed.
	pushRetry = 30 * time.Second
	// maxPushAttempts drops a write GitHub keeps failing; the ticket's next change queues it again.
	maxPushAttempts = 10
	// markerSlack is how far before a first attempt a retry looks for what that attempt made, for clock skew.
	markerSlack = time.Minute
)

// Outbound is a ticket as its issue takes it.
type Outbound struct {
	ProjectID string
	// Fields are the title, the body as markdown for GitHub, done as closed, the labels, and the developer's GitHub
	// login as the one assignee, none when they have no GitHub login.
	Fields TicketFields
}

// Mirror is a ticket-thread message as its issue's comment takes it.
type Mirror struct {
	// Person is a message a person wrote; an agent's, a system note and a mirrored GitHub comment never go back.
	Person  bool
	Deleted bool
	// Body is the comment as GitHub shows it, naming who wrote it.
	Body string
}

// Source reads tickets and messages as GitHub takes them (ADR 0017).
type Source interface {
	// Ticket is ErrNotFound for a ticket that is gone.
	Ticket(ctx context.Context, ticketID string) (Outbound, error)
	// Message is ErrNotFound for a message that is gone.
	Message(ctx context.Context, messageID string) (Mirror, error)
	// ThreadOf is the ticket a conversation is the thread of, empty for any other conversation.
	ThreadOf(ctx context.Context, conversationID string) (string, error)
}

// IssueWrite is what Nexul sets on an issue; nil Assignees leave the issue's as they are.
type IssueWrite struct {
	Title     string
	Body      string
	Closed    bool
	Labels    []string
	Assignees []string
}

// IssueWriter writes issues as the GitHub App's installation.
type IssueWriter interface {
	CreateIssue(ctx context.Context, owner, repo string, w IssueWrite) (Issue, error)
	UpdateIssue(ctx context.Context, owner, repo string, number int, w IssueWrite) (Issue, error)
}

// CommentWriter writes issue comments as the GitHub App's installation.
type CommentWriter interface {
	CreateComment(ctx context.Context, owner, repo string, number int, body string) (Comment, error)
	UpdateComment(ctx context.Context, owner, repo string, commentID int64, body string) error
	DeleteComment(ctx context.Context, owner, repo string, commentID int64) error
}

// ticketMarker hides a ticket's id in the issue made for it, so a retry finds that issue instead of making another.
var ticketMarker = regexp.MustCompile(`\n*<!-- nexul:ticket:([\w-]+) -->`)

func ticketMarkerFor(ticketID string) string   { return "<!-- nexul:ticket:" + ticketID + " -->" }
func messageMarkerFor(messageID string) string { return "<!-- nexul:message:" + messageID + " -->" }

// markedTicket is the ticket id an issue body was made for, empty when Nexul made none.
func markedTicket(body string) string {
	m := ticketMarker.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return m[1]
}

// withoutMarker is an issue body without the hidden ticket marker Nexul wrote into it.
func withoutMarker(body string) string {
	if markedTicket(body) == "" {
		return body
	}
	return strings.TrimRight(ticketMarker.ReplaceAllString(body, ""), "\n")
}

// TicketChanged queues a linked ticket's fields to be written to its issue once its changes rest for the quiet spell;
// added are the labels the change added, and a synced repository's trigger label among them queues an unlinked
// ticket's issue to be made.
func (s *Service) TicketChanged(ctx context.Context, ticketID, projectID string, added []string) error {
	due := s.d.Now().Add(s.d.Quiet)
	link, err := s.d.Repo.GetLinkByTicket(ctx, ticketID)
	if err == nil {
		_, on, err := s.enabled(ctx, link.Owner, link.Repo)
		if err != nil || !on || link.Paused {
			return err
		}
		return s.d.Repo.QueueTicketPush(ctx, TicketPush{TicketID: ticketID, Owner: link.Owner, Repo: link.Repo, Due: due})
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return err
	}
	if len(added) == 0 {
		return nil
	}
	syncs, err := s.d.Repo.ListSyncsByProject(ctx, projectID)
	if err != nil {
		return err
	}
	for _, sy := range syncs {
		if sy.Enabled && slices.ContainsFunc(added, func(l string) bool { return strings.EqualFold(l, sy.Label) }) {
			return s.d.Repo.QueueTicketPush(ctx, TicketPush{TicketID: ticketID, Owner: sy.Owner, Repo: sy.Repo, Create: true, Due: due})
		}
	}
	return nil
}

// MessageChanged queues a message on a linked ticket's thread to be mirrored to its issue: posted, edited or deleted.
func (s *Service) MessageChanged(ctx context.Context, conversationID, messageID string) error {
	ticketID, err := s.d.Source.ThreadOf(ctx, conversationID)
	if err != nil || ticketID == "" {
		return err
	}
	link, err := s.d.Repo.GetLinkByTicket(ctx, ticketID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil || link.Paused {
		return err
	}
	return s.d.Repo.QueueCommentPush(ctx, CommentPush{MessageID: messageID, TicketID: ticketID, Due: s.d.Now()})
}

// RunPushLoop writes what waits for GitHub once it is due, woken by the commit that queues it and by the next due.
func (s *Service) RunPushLoop(ctx context.Context, wake func() <-chan struct{}, logger *slog.Logger) {
	for {
		committed := wake()
		wait, err := s.Push(ctx)
		if err != nil {
			logger.Warn("github issue push", "error", err)
		}
		if !sleep(ctx, wait, committed) {
			return
		}
	}
}

// sleep waits out wait (forever when negative) or a commit, and reports false once ctx ends.
func sleep(ctx context.Context, wait time.Duration, committed <-chan struct{}) bool {
	var timeout <-chan time.Time
	if wait >= 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		timeout = timer.C
	}
	select {
	case <-ctx.Done():
		return false
	case <-timeout:
	case <-committed:
	}
	return true
}

// Push writes every due ticket and comment and returns how long until the next is due, -1 when nothing waits; one
// write's failure leaves the others to run, and one that failed before it settled waits a retry, so a failing store
// never spins the loop. A push already done or deferred is left as it is.
func (s *Service) Push(ctx context.Context) (time.Duration, error) {
	var errs []error
	pushes, err := s.d.Repo.DueTicketPushes(ctx, s.d.Now(), pushBatch)
	if err != nil {
		return pushRetry, err
	}
	for _, p := range pushes {
		if err := s.pushTicket(ctx, p); err != nil {
			errs = append(errs, fmt.Errorf("ticket %s: %w", p.TicketID, errors.Join(err, s.retryLater(ctx, p))))
		}
	}
	comments, err := s.d.Repo.DueCommentPushes(ctx, s.d.Now(), pushBatch)
	if err != nil {
		return pushRetry, errors.Join(append(errs, err)...)
	}
	for _, p := range comments {
		if err := s.pushComment(ctx, p); err != nil {
			errs = append(errs, fmt.Errorf("message %s: %w", p.MessageID, errors.Join(err, s.retryCommentLater(ctx, p))))
		}
	}
	next, ok, err := s.d.Repo.NextPushDue(ctx)
	if err != nil {
		return pushRetry, errors.Join(append(errs, err)...)
	}
	if !ok {
		return -1, errors.Join(errs...)
	}
	return max(next.Sub(s.d.Now()), 0), errors.Join(errs...)
}

// retryLater moves a push that failed before it settled a retry away; it takes no attempts from p, so a newer change
// queued meanwhile keeps its own count.
func (s *Service) retryLater(ctx context.Context, p TicketPush) error {
	p.Attempts = 0
	return s.deferTicket(ctx, p, s.d.Retry, false)
}

func (s *Service) retryCommentLater(ctx context.Context, p CommentPush) error {
	p.Attempts = 0
	return s.deferComment(ctx, p, s.d.Retry, false)
}

// outbound is the ticket as sy's issue takes it, labels sorted and without the trigger label.
func (s *Service) outbound(ctx context.Context, ticketID string, sy Sync) (TicketFields, error) {
	out, err := s.d.Source.Ticket(ctx, ticketID)
	if err != nil {
		return TicketFields{}, err
	}
	f := out.Fields
	f.Labels = slices.DeleteFunc(slices.Clone(f.Labels), func(l string) bool { return sy.trigger() != "" && strings.EqualFold(l, sy.trigger()) })
	slices.Sort(f.Labels)
	return f, nil
}

// issueWrite is f as sy writes it: labels in full plus the trigger label, and the hidden marker of the ticket.
func (s Sync) issueWrite(f TicketFields, ticketID string) IssueWrite {
	labels := slices.Clone(f.Labels)
	if t := s.trigger(); t != "" {
		labels = append(labels, t)
	}
	return IssueWrite{Title: f.Title, Body: f.Body + "\n\n" + ticketMarkerFor(ticketID), Closed: f.Closed, Labels: labels, Assignees: f.Assignees}
}

// pushTicket writes one ticket to its issue, or makes the issue a trigger label asked for; it holds the issue's lock so
// a webhook never applies an issue halfway through being written.
func (s *Service) pushTicket(ctx context.Context, p TicketPush) error {
	unlock, link, linked, err := s.lockTicket(ctx, p.TicketID)
	if err != nil {
		return err
	}
	defer unlock()
	if linked {
		p.Owner, p.Repo = link.Owner, link.Repo
	}
	sy, on, err := s.enabled(ctx, p.Owner, p.Repo)
	if err != nil {
		return err
	}
	if !on || (linked && link.Paused) || (!linked && !p.Create) {
		return s.d.Repo.DoneTicketPush(ctx, p)
	}
	f, err := s.outbound(ctx, p.TicketID, sy)
	if errors.Is(err, apperrs.ErrNotFound) {
		return s.d.Repo.DoneTicketPush(ctx, p)
	}
	if err != nil {
		return err
	}
	if !linked {
		return s.createIssue(ctx, sy, p, f)
	}
	return s.updateIssue(ctx, sy, p, link, f)
}

// updateIssue writes f to the linked issue, unless the ticket is as it was when the two were last in step. Only that
// hash says so: a ticket put back as it was before the last sync still differs from its issue now.
func (s *Service) updateIssue(ctx context.Context, sy Sync, p TicketPush, link Link, f TicketFields) error {
	h := Hash(f)
	inStep := h == link.TicketHash
	if link.TicketHash == "" {
		inStep = h == link.InboundHash || h == link.OutboundHash
	}
	if inStep {
		return s.d.Repo.DoneTicketPush(ctx, p)
	}
	if wait := s.pace.wait(p.Owner, s.d.Now(), false); wait > 0 {
		return s.deferTicket(ctx, p, wait, false)
	}
	s.pace.took(p.Owner, s.d.Now(), false)
	issue, err := s.d.IssueWriter.UpdateIssue(ctx, p.Owner, p.Repo, link.IssueNumber, sy.issueWrite(f, p.TicketID))
	if errors.Is(err, apperrs.ErrNotFound) {
		if err := s.lost(ctx, link); err != nil {
			return err
		}
		return s.d.Repo.DoneTicketPush(ctx, p)
	}
	if err != nil {
		return s.ticketFailed(ctx, p, err)
	}
	return s.wrote(ctx, sy, p, link, issue, h)
}

// wrote records what Nexul wrote, so neither its own webhook nor the ticket's unchanged fields write again.
func (s *Service) wrote(ctx context.Context, sy Sync, p TicketPush, link Link, issue Issue, ticketHash string) error {
	link.OutboundHash, link.TicketHash = Hash(FieldsFromIssue(issue, sy.trigger())), ticketHash
	if issue.UpdatedAt.After(link.IssueUpdatedAt) {
		link.IssueUpdatedAt = issue.UpdatedAt
	}
	if err := s.d.Repo.PutLink(ctx, link); err != nil {
		return err
	}
	return s.d.Repo.DoneTicketPush(ctx, p)
}

// createIssue makes the issue for a ticket its trigger label was added to; a retry first looks for the issue an
// earlier attempt may have made before GitHub's answer was lost.
func (s *Service) createIssue(ctx context.Context, sy Sync, p TicketPush, f TicketFields) error {
	if !p.Tried.IsZero() {
		found, ok, err := s.findIssue(ctx, sy, p)
		if err != nil {
			return s.ticketFailed(ctx, p, err)
		}
		if ok {
			return s.linkCreated(ctx, sy, p, f, found)
		}
	}
	if wait := s.pace.wait(p.Owner, s.d.Now(), true); wait > 0 {
		return s.deferTicket(ctx, p, wait, false)
	}
	if p.Tried.IsZero() {
		p.Tried = s.d.Now()
	}
	if err := s.d.Repo.DeferTicketPush(ctx, p, p.Due); err != nil {
		return err
	}
	s.pace.took(p.Owner, s.d.Now(), true)
	issue, err := s.d.IssueWriter.CreateIssue(ctx, p.Owner, p.Repo, sy.issueWrite(f, p.TicketID))
	if err != nil {
		return s.ticketFailed(ctx, p, err)
	}
	return s.linkCreated(ctx, sy, p, f, issue)
}

// linkCreated links the issue made for p's ticket; GitHub opens every new issue, so a ticket already done is closed by
// the next pass.
func (s *Service) linkCreated(ctx context.Context, sy Sync, p TicketPush, f TicketFields, issue Issue) error {
	link := Link{Owner: p.Owner, Repo: p.Repo, IssueNumber: issue.Number, IssueID: issue.ID, TicketID: p.TicketID, LinkedAt: s.d.Now().UTC()}
	if !f.Closed {
		return s.wrote(ctx, sy, p, link, issue, Hash(f))
	}
	link.OutboundHash, link.IssueUpdatedAt = Hash(FieldsFromIssue(issue, sy.trigger())), issue.UpdatedAt
	if err := s.d.Repo.PutLink(ctx, link); err != nil {
		return err
	}
	return s.deferTicket(ctx, p, 0, false)
}

// findIssue looks for the issue carrying p's ticket marker among those changed since its first attempt.
func (s *Service) findIssue(ctx context.Context, sy Sync, p TicketPush) (Issue, bool, error) {
	q := IssueQuery{State: "all", Since: p.Tried.Add(-markerSlack)}
	for q.Page = 1; q.Page != 0; {
		issues, next, err := s.d.Issues.ListIssues(ctx, sy.Owner, sy.Repo, q)
		if err != nil {
			return Issue{}, false, err
		}
		for _, i := range issues {
			if markedTicket(i.Body) == p.TicketID {
				return i, true, nil
			}
		}
		q.Page = next
	}
	return Issue{}, false, nil
}

// ticketFailed retries a write GitHub failed for a while, after its Retry-After when it named one, and drops one it
// refused, returning why so the loop logs it.
func (s *Service) ticketFailed(ctx context.Context, p TicketPush, err error) error {
	var limited *apperrs.RetryAfterError
	if errors.As(err, &limited) {
		s.restOnLimit(p.Owner, err)
		return s.deferTicket(ctx, p, limited.Wait, true)
	}
	if errors.Is(err, apperrs.ErrRetryable) && p.Attempts+1 < maxPushAttempts {
		return s.deferTicket(ctx, p, s.backoff(p.Attempts), true)
	}
	if derr := s.d.Repo.DoneTicketPush(ctx, p); derr != nil {
		return errors.Join(err, derr)
	}
	return fmt.Errorf("write to %s/%s dropped: %w", p.Owner, p.Repo, err)
}

func (s *Service) deferTicket(ctx context.Context, p TicketPush, wait time.Duration, failed bool) error {
	was := p.Due
	p.Due = s.d.Now().Add(wait)
	if failed {
		p.Attempts++
	}
	return s.d.Repo.DeferTicketPush(ctx, p, was)
}

// backoff doubles the first retry's wait with each attempt, up to half an hour.
func (s *Service) backoff(attempts int) time.Duration {
	return min(s.d.Retry<<min(attempts, 6), 30*time.Minute)
}

// lost drops the link of an issue GitHub no longer has; the ticket stays, and its thread says why it stopped syncing.
func (s *Service) lost(ctx context.Context, link Link) error {
	if err := s.d.Repo.DeleteLink(ctx, link); err != nil {
		return err
	}
	key := fmt.Sprintf("github-issue %s/%s#%d gone", link.Owner, link.Repo, link.IssueNumber)
	return s.d.Notes.Note(ctx, link.TicketID, key, fmt.Sprintf("%s is no longer on GitHub, so it no longer syncs. The ticket stays.", issueRef(link)))
}

// pushComment mirrors one message to its ticket's issue: a new one is posted once however often it is retried, an
// edit edits the comment, and a delete deletes it.
func (s *Service) pushComment(ctx context.Context, p CommentPush) error {
	unlock, link, linked, err := s.lockTicket(ctx, p.TicketID)
	if err != nil {
		return err
	}
	defer unlock()
	if !linked {
		return s.d.Repo.DoneCommentPush(ctx, p)
	}
	_, on, err := s.enabled(ctx, link.Owner, link.Repo)
	if err != nil {
		return err
	}
	m, err := s.d.Source.Message(ctx, p.MessageID)
	if errors.Is(err, apperrs.ErrNotFound) || (err == nil && (!on || link.Paused || !m.Person)) {
		return s.d.Repo.DoneCommentPush(ctx, p)
	}
	if err != nil {
		return err
	}
	cl, err := s.d.Repo.GetCommentLinkByMessage(ctx, p.MessageID)
	known := err == nil
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return err
	}
	if !known && m.Deleted {
		return s.d.Repo.DoneCommentPush(ctx, p)
	}
	body := m.Body + "\n\n" + messageMarkerFor(p.MessageID)
	if !known {
		return s.createComment(ctx, link, p, body)
	}
	return s.followComment(ctx, link, p, cl, m, body)
}

// followComment edits a linked comment to body, or deletes it with its link for a deleted message; a comment already
// gone from GitHub only loses its link.
func (s *Service) followComment(ctx context.Context, link Link, p CommentPush, cl CommentLink, m Mirror, body string) error {
	if wait := s.pace.wait(link.Owner, s.d.Now(), false); wait > 0 {
		return s.deferComment(ctx, p, wait, false)
	}
	s.pace.took(link.Owner, s.d.Now(), false)
	err := s.editComment(ctx, link, cl, m, body)
	if errors.Is(err, apperrs.ErrNotFound) || (err == nil && m.Deleted) {
		if err := s.d.Repo.DeleteCommentLink(ctx, cl.GitHubCommentID); err != nil {
			return err
		}
		return s.d.Repo.DoneCommentPush(ctx, p)
	}
	if err != nil {
		return s.commentFailed(ctx, p, link.Owner, err)
	}
	return s.d.Repo.DoneCommentPush(ctx, p)
}

// editComment deletes the comment of a deleted message, else sets its body.
func (s *Service) editComment(ctx context.Context, link Link, cl CommentLink, m Mirror, body string) error {
	if m.Deleted {
		return s.d.CommentWriter.DeleteComment(ctx, link.Owner, link.Repo, cl.GitHubCommentID)
	}
	return s.d.CommentWriter.UpdateComment(ctx, link.Owner, link.Repo, cl.GitHubCommentID, body)
}

// createComment posts the comment, first looking for one an earlier attempt may have posted before GitHub's answer
// was lost, so a retry never posts it twice.
func (s *Service) createComment(ctx context.Context, link Link, p CommentPush, body string) error {
	if !p.Tried.IsZero() {
		comments, err := s.d.Issues.ListComments(ctx, link.Owner, link.Repo, link.IssueNumber, p.Tried.Add(-markerSlack))
		if err != nil {
			return s.commentFailed(ctx, p, link.Owner, err)
		}
		marker := messageMarkerFor(p.MessageID)
		for _, c := range comments {
			if strings.Contains(c.Body, marker) {
				return s.commentPosted(ctx, p, c)
			}
		}
	}
	if wait := s.pace.wait(link.Owner, s.d.Now(), true); wait > 0 {
		return s.deferComment(ctx, p, wait, false)
	}
	if p.Tried.IsZero() {
		p.Tried = s.d.Now()
	}
	if err := s.d.Repo.DeferCommentPush(ctx, p, p.Due); err != nil {
		return err
	}
	s.pace.took(link.Owner, s.d.Now(), true)
	c, err := s.d.CommentWriter.CreateComment(ctx, link.Owner, link.Repo, link.IssueNumber, body)
	if errors.Is(err, apperrs.ErrNotFound) {
		if err := s.lost(ctx, link); err != nil {
			return err
		}
		return s.d.Repo.DoneCommentPush(ctx, p)
	}
	if err != nil {
		return s.commentFailed(ctx, p, link.Owner, err)
	}
	return s.commentPosted(ctx, p, c)
}

func (s *Service) commentPosted(ctx context.Context, p CommentPush, c Comment) error {
	if err := s.d.Repo.PutCommentLink(ctx, CommentLink{GitHubCommentID: c.ID, MessageID: p.MessageID, TicketID: p.TicketID}); err != nil {
		return err
	}
	return s.d.Repo.DoneCommentPush(ctx, p)
}

func (s *Service) commentFailed(ctx context.Context, p CommentPush, owner string, err error) error {
	var limited *apperrs.RetryAfterError
	if errors.As(err, &limited) {
		s.restOnLimit(owner, err)
		return s.deferComment(ctx, p, limited.Wait, true)
	}
	if errors.Is(err, apperrs.ErrRetryable) && p.Attempts+1 < maxPushAttempts {
		return s.deferComment(ctx, p, s.backoff(p.Attempts), true)
	}
	if derr := s.d.Repo.DoneCommentPush(ctx, p); derr != nil {
		return errors.Join(err, derr)
	}
	return fmt.Errorf("comment on %s dropped: %w", owner, err)
}

func (s *Service) deferComment(ctx context.Context, p CommentPush, wait time.Duration, failed bool) error {
	was := p.Due
	p.Due = s.d.Now().Add(wait)
	if failed {
		p.Attempts++
	}
	return s.d.Repo.DeferCommentPush(ctx, p, was)
}

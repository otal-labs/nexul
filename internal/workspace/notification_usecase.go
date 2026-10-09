package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// NotificationService is the notifications use-case layer (ADR 0019): the inbox plus v1's fan-out generation rules.
type NotificationService struct {
	repo     NotificationRepo
	users    UserStore
	members  WorkspaceMemberStore
	access   PermissionChecker
	projects ProjectReader
	tickets  TicketProjects
	watchers DocWatchers
	now      func() time.Time
}

// SetTicketProjects wires the ticket lookup a ticket notice whose event names no project is checked through.
func (s *NotificationService) SetTicketProjects(t TicketProjects) { s.tickets = t }

// NewNotificationService wires the notification use-cases; delivery goes through the outbox, not a direct
// publish. members and access back memory.updated's permission-gated fan-out to a workspace's members, and
// projects resolves a ticket's or doc's workspace.
func NewNotificationService(repo NotificationRepo, users UserStore, members WorkspaceMemberStore, access PermissionChecker, projects ProjectReader) *NotificationService {
	return &NotificationService{repo: repo, users: users, members: members, access: access, projects: projects, now: time.Now}
}

// WithDocWatchers sets who a doc's changes notify; without it a doc save notifies only the people it mentions.
func (s *NotificationService) WithDocWatchers(w DocWatchers) *NotificationService {
	s.watchers = w
	return s
}

// Page returns one window of a user's inbox, newest first, and how many notifications it shows in all. It shows only
// the workspaces they still belong to and, for a notice about a project, the projects they may open; SQL leaves the
// rest out, so the total counts only what the pages hold (ADR 0140).
func (s *NotificationService) Page(ctx context.Context, userID string, f InboxFilter, w paging.Window) ([]*Notification, int, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, 0, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	f.WorkspaceID = strings.TrimSpace(f.WorkspaceID)
	if f.WorkspaceID != "" {
		if err := s.requireMember(ctx, userID, f.WorkspaceID); err != nil {
			return nil, 0, err
		}
	}
	scope, err := s.inboxScope(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	ns, total, err := s.repo.Page(ctx, userID, f, scope, w.Clamped())
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	return ns, total, nil
}

// inboxScope is what userID's inbox may show; nil, showing everything, only when no access checker is wired.
func (s *NotificationService) inboxScope(ctx context.Context, userID string) (*InboxScope, error) {
	if s.access == nil {
		return nil, nil
	}
	workspaceIDs, projectIDs, err := s.access.ProjectsAnywhere(ctx, userID, permissions.Member)
	if err != nil {
		return nil, fmt.Errorf("inbox access of %s: %w", userID, err)
	}
	return &InboxScope{WorkspaceIDs: workspaceIDs, ProjectIDs: projectIDs}, nil
}

// opensProject keeps a notice about a project its reader can no longer open out of their inbox; the row stays, so
// access given back brings it back.
func (s *NotificationService) opensProject(ctx context.Context, userID, projectID string) bool {
	if projectID == "" || s.access == nil {
		return true
	}
	return s.access.CanInProject(ctx, userID, projectID, permissions.Member)
}

// UnreadCount returns how many of the user's notifications in one workspace (or every workspace) are unread.
func (s *NotificationService) UnreadCount(ctx context.Context, userID, workspaceID string) (int, error) {
	byWorkspace, err := s.UnreadByWorkspace(ctx, userID, workspaceID)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, c := range byWorkspace {
		n += c
	}
	return n, nil
}

// UnreadByWorkspace returns the user's unread notification count per workspace that has any, the switcher's badges.
func (s *NotificationService) UnreadByWorkspace(ctx context.Context, userID, workspaceID string) (map[string]int, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	groups, err := s.repo.UnreadByProject(ctx, userID, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("unread count: %w", err)
	}
	out := map[string]int{}
	for _, g := range groups {
		if s.opensProject(ctx, userID, g.ProjectID) {
			out[g.WorkspaceID] += g.Unread
		}
	}
	return out, nil
}

// MarkRead marks one of the user's notifications as read.
func (s *NotificationService) MarkRead(ctx context.Context, userID, id string) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: notification id is required", apperrs.ErrInvalid)
	}
	if err := s.repo.MarkRead(ctx, userID, id, s.now().UTC()); err != nil {
		return fmt.Errorf("mark notification %s read: %w", id, err)
	}
	return nil
}

// MarkAllRead marks every notification of the user in one workspace (or every workspace) as read.
func (s *NotificationService) MarkAllRead(ctx context.Context, userID, workspaceID string) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	if err := s.repo.MarkAllRead(ctx, userID, strings.TrimSpace(workspaceID), s.now().UTC()); err != nil {
		return fmt.Errorf("mark all read: %w", err)
	}
	return nil
}

// Retention keeps the inbox bounded, since nothing else ever deletes a notification (ADR 0100).
const (
	readNotificationRetention   = 90 * 24 * time.Hour
	notificationRetention       = 180 * 24 * time.Hour
	notificationCleanupDelay    = time.Minute
	notificationCleanupInterval = 24 * time.Hour
)

// CleanupExpired applies the retention rule (ADR 0100), unchecked because it runs from a background loop, not a request.
func (s *NotificationService) CleanupExpired(ctx context.Context) (read, old int64, err error) {
	now := s.now()
	read, old, err = s.repo.DeleteExpired(ctx, now.Add(-readNotificationRetention), now.Add(-notificationRetention))
	if err != nil {
		return 0, 0, fmt.Errorf("clean up expired notifications: %w", err)
	}
	return read, old, nil
}

// RunCleanupLoop runs CleanupExpired a minute after start and daily after that, until ctx is cancelled.
func (s *NotificationService) RunCleanupLoop(ctx context.Context, log *slog.Logger) {
	timer := time.NewTimer(notificationCleanupDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		timer.Reset(notificationCleanupInterval)
		read, old, err := s.CleanupExpired(ctx)
		if err != nil {
			log.Warn("notification retention cleanup failed", "error", err)
			continue
		}
		if read+old > 0 {
			log.Info("notification retention cleanup", "read_deleted", read, "old_deleted", old)
		}
	}
}

// requireMember keeps a person's inbox to the workspaces they still belong to; one they left reads as not found.
func (s *NotificationService) requireMember(ctx context.Context, userID, workspaceID string) error {
	if s.members == nil {
		return nil
	}
	ids, err := s.members.ListMemberUserIDs(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("list members of workspace %s: %w", workspaceID, err)
	}
	if !slices.Contains(ids, userID) {
		return fmt.Errorf("%w: workspace %s", apperrs.ErrNotFound, workspaceID)
	}
	return nil
}

// canOpen keeps a notice from naming a ticket or doc its recipient may not read.
func (s *NotificationService) canOpen(ctx context.Context, n notice, userID string) bool {
	if s.access == nil {
		return true
	}
	switch n.subjectType {
	case SubjectTicket:
		if projectID := s.ticketProject(ctx, n); projectID != "" {
			return s.access.CanInProject(ctx, userID, projectID, permissions.TicketsRead)
		}
		return s.access.HasPermission(ctx, userID, n.workspaceID, permissions.TicketsRead)
	case SubjectDoc:
		return s.access.CanReadDoc(ctx, userID, n.subjectID)
	}
	return true
}

// ticketProject is the project a ticket notice is about: named by its event, else read from the ticket itself.
func (s *NotificationService) ticketProject(ctx context.Context, n notice) string {
	if n.projectID != "" || s.tickets == nil {
		return n.projectID
	}
	projectID, err := s.tickets.ProjectOfTicket(ctx, n.subjectID)
	if err != nil {
		return ""
	}
	return projectID
}

// onTicketCreated fans out ticket.assigned to the developer and tester and ticket.mentioned to every @-mentioned user.
func (s *NotificationService) onTicketCreated(ctx context.Context, t ticketRef, mentionedIDs []string) error {
	actorID, err := s.userIDForLogin(ctx, t.Reporter.Login)
	if err != nil {
		return err
	}
	n, err := s.notice(ctx, SubjectTicket, t.ID, t.Title, t.ProjectID, actorID)
	if err != nil {
		return err
	}
	mentionTitle := s.mentionTitle(ctx, n)
	recipients := []recipient{{Login: t.Developer, Kind: KindTicketAssigned}, {Login: t.Tester, Kind: KindTicketAssigned}}
	for _, login := range extractMentions(t.Title + " " + t.Body) {
		recipients = append(recipients, recipient{Login: login, Kind: KindTicketMentioned, Title: mentionTitle})
	}
	for _, id := range mentionedIDs {
		recipients = append(recipients, recipient{UserID: id, Kind: KindTicketMentioned, Title: mentionTitle})
	}
	return s.fanOut(ctx, evtKey(ctx), n, recipients)
}

// onTicketUpdated fans out ticket.mentioned to the people an edit newly @-mentions.
func (s *NotificationService) onTicketUpdated(ctx context.Context, e ticketUpdatedEvent) error {
	if len(e.MentionedUserIDs) == 0 {
		return nil
	}
	n, err := s.notice(ctx, SubjectTicket, e.Ticket.ID, e.Ticket.Title, e.Ticket.ProjectID, e.ActorID)
	if err != nil {
		return err
	}
	mentionTitle := s.mentionTitle(ctx, n)
	recipients := make([]recipient, 0, len(e.MentionedUserIDs))
	for _, id := range e.MentionedUserIDs {
		recipients = append(recipients, recipient{UserID: id, Kind: KindTicketMentioned, Title: mentionTitle})
	}
	return s.fanOut(ctx, evtKey(ctx), n, recipients)
}

// onTicketStatusChanged fans out ticket.status_changed to the developer, tester, and @-mentioned users of the ticket.
func (s *NotificationService) onTicketStatusChanged(ctx context.Context, t ticketRef, actorID string) error {
	recipients := []recipient{{Login: t.Developer, Kind: KindTicketStatus}, {Login: t.Tester, Kind: KindTicketStatus}}
	for _, login := range extractMentions(t.Title + " " + t.Body) {
		if strings.EqualFold(login, t.Developer) || strings.EqualFold(login, t.Tester) {
			continue
		}
		recipients = append(recipients, recipient{Login: login, Kind: KindTicketStatus})
	}
	n, err := s.notice(ctx, SubjectTicket, t.ID, t.Title, t.ProjectID, actorID)
	if err != nil {
		return err
	}
	return s.fanOut(ctx, evtKey(ctx), n, recipients)
}

// onDocActivity tells the doc's watchers but the actor, and newly mentioned members with doc.mentioned (ADR 0101).
func (s *NotificationService) onDocActivity(ctx context.Context, e docEvent, kind Kind) error {
	n, err := s.notice(ctx, SubjectDoc, e.Doc.ID, e.Doc.Title, e.Doc.ProjectID, e.ActorID)
	if err != nil {
		return err
	}
	if n.workspaceID == "" {
		return nil
	}
	recipients, err := s.docMentionRecipients(ctx, n, e.MentionedUserIDs)
	if err != nil {
		return err
	}
	if s.watchers != nil {
		ids, err := s.watchers.ListDocWatcherIDs(ctx, e.Doc.ID)
		if err != nil {
			return fmt.Errorf("list watchers of doc %s: %w", e.Doc.ID, err)
		}
		for _, id := range ids {
			recipients = append(recipients, recipient{UserID: id, Kind: kind})
		}
	}
	return s.fanOut(ctx, evtKey(ctx), n, recipients)
}

// onDocQuestionsPosted tells the doc's watchers but the round's starter; the starter's own post is no news to them.
func (s *NotificationService) onDocQuestionsPosted(ctx context.Context, e docQuestionsPostedEvent) error {
	if e.NoGaps || e.QuestionCount <= 0 || s.watchers == nil {
		return nil
	}
	n, err := s.notice(ctx, SubjectDoc, e.Doc.ID, "New questions on "+e.Doc.Title, e.Doc.ProjectID, e.StartedBy)
	if err != nil || n.workspaceID == "" {
		return err
	}
	ids, err := s.watchers.ListDocWatcherIDs(ctx, e.Doc.ID)
	if err != nil {
		return fmt.Errorf("list watchers of doc %s: %w", e.Doc.ID, err)
	}
	return s.fanOutByUserID(ctx, evtKey(ctx), n, KindDocQuestionsAsked, ids)
}

// onDocRoundAnswered tells the round's starter, unless the answer that finished it was their own.
func (s *NotificationService) onDocRoundAnswered(ctx context.Context, e docRoundAnsweredEvent) error {
	n, err := s.notice(ctx, SubjectDoc, e.Doc.ID, "Questions answered on "+e.Doc.Title, e.Doc.ProjectID, e.ActorID)
	if err != nil || n.workspaceID == "" {
		return err
	}
	return s.fanOutByUserID(ctx, evtKey(ctx), n, KindDocQuestionsAnswered, []string{e.StartedBy})
}

// docMentionRecipients keeps the mentioned people who are members of the doc's workspace.
func (s *NotificationService) docMentionRecipients(ctx context.Context, n notice, mentioned []string) ([]recipient, error) {
	if len(mentioned) == 0 || s.members == nil {
		return nil, nil
	}
	members, err := s.members.ListMemberUserIDs(ctx, n.workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list members for doc mentions: %w", err)
	}
	mentionTitle := s.mentionTitle(ctx, n)
	var out []recipient
	for _, id := range mentioned {
		if slices.Contains(members, id) {
			out = append(out, recipient{UserID: id, Kind: KindDocMentioned, Title: mentionTitle})
		}
	}
	return out, nil
}

// mentionTitle words a mention as "<author> mentioned you in <title>"; with no known author it stays the title.
func (s *NotificationService) mentionTitle(ctx context.Context, n notice) string {
	if n.actorID == "" {
		return n.subjectTitle
	}
	name, err := s.users.NameForUserID(ctx, n.actorID)
	if err != nil || name == "" {
		return n.subjectTitle
	}
	return fmt.Sprintf("%s mentioned you in %s", name, n.subjectTitle)
}

// onMemoryUpdated fans out to every member of the memory's workspace who may read memories in its project, excluding
// the author; membership and Project access decide, not every registered user.
func (s *NotificationService) onMemoryUpdated(ctx context.Context, m memoryRef, authorID, authorVia string) error {
	if s.members == nil || s.access == nil {
		return nil
	}
	userIDs, err := s.members.ListMemberUserIDs(ctx, m.WorkspaceID)
	if err != nil {
		return fmt.Errorf("list members for workspace %s: %w", m.WorkspaceID, err)
	}
	authorLabel := authorID
	if login, err := s.users.LoginForUserID(ctx, authorID); err == nil && login != "" {
		authorLabel = login
	}
	if authorVia == "mcp" {
		authorLabel = "Agent via " + authorLabel
	}
	subjectTitle := fmt.Sprintf("%s — v%d by %s", m.Title, m.Version, authorLabel)
	var recipients []string
	for _, uid := range userIDs {
		if !s.access.CanInProject(ctx, uid, m.ProjectID, permissions.MemoriesRead) {
			continue
		}
		recipients = append(recipients, uid)
	}
	n := notice{subjectType: SubjectMemory, subjectID: m.ID, subjectTitle: subjectTitle, workspaceID: m.WorkspaceID, actorID: authorID}
	return s.fanOutByUserID(ctx, evtKey(ctx), n, KindMemoryUpdated, recipients)
}

// onPlayRunFinished tells the starter how their run ended; the subject is the target so the inbox opens it.
func (s *NotificationService) onPlayRunFinished(ctx context.Context, e playRunFinishedEvent) error {
	outcome := "finished"
	if e.Outcome == "failed" || e.Outcome == "interrupted" {
		outcome = e.Outcome
	}
	return s.notifyPlayStarter(ctx, e, outcome, KindPlayRunFinished)
}

// onPlayRunWaiting tells the starter their run needs an answer before it can go on.
func (s *NotificationService) onPlayRunWaiting(ctx context.Context, e playRunFinishedEvent) error {
	return s.notifyPlayStarter(ctx, e, "needs your answer", KindPlayRunWaiting)
}

// notifyPlayStarter has no actor to exclude: the run, not the starter, is what finished or asked.
func (s *NotificationService) notifyPlayStarter(ctx context.Context, e playRunFinishedEvent, outcome string, kind Kind) error {
	subjectType := SubjectTicket
	if e.TargetType == string(SubjectDoc) {
		subjectType = SubjectDoc
	}
	title := e.TargetTitle
	if title == "" {
		title = e.TargetID
	}
	n := notice{subjectType: subjectType, subjectID: e.TargetID, subjectTitle: fmt.Sprintf("%s %s on %s", e.PlayLabel, outcome, title), workspaceID: e.WorkspaceID}
	return s.fanOutByUserID(ctx, evtKey(ctx), n, kind, []string{e.StarterID})
}

// notice is what one source event is about: its subject, the subject's workspace, and whose action it was.
type notice struct {
	subjectType  SubjectType
	subjectID    string
	subjectTitle string
	workspaceID  string
	projectID    string
	actorID      string
}

// notice resolves the subject's project to its workspace; a project deleted since the event leaves it unscoped.
func (s *NotificationService) notice(ctx context.Context, subjectType SubjectType, subjectID, subjectTitle, projectID, actorID string) (notice, error) {
	n := notice{subjectType: subjectType, subjectID: subjectID, subjectTitle: subjectTitle, projectID: projectID, actorID: actorID}
	if s.projects == nil || strings.TrimSpace(projectID) == "" {
		return n, nil
	}
	p, err := s.projects.Get(ctx, projectID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return n, nil
	}
	if err != nil {
		return n, fmt.Errorf("resolve workspace of project %s: %w", projectID, err)
	}
	n.workspaceID = p.WorkspaceID
	return n, nil
}

// userIDForLogin resolves an event's actor login; an unknown login falls back to itself, since the tickets domain
// records the raw user id when its own lookup failed.
func (s *NotificationService) userIDForLogin(ctx context.Context, login string) (string, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return "", nil
	}
	u, err := s.users.GetUserByLogin(ctx, login)
	if errors.Is(err, apperrs.ErrNotFound) || (err == nil && u == nil) {
		return login, nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve actor %s: %w", login, err)
	}
	return u.ID, nil
}

// recipient is one pending fan-out row: who by login or user id, the kind to create, and a title replacing the subject's.
type recipient struct {
	Login  string
	UserID string
	Kind   Kind
	Title  string
}

// fanOut creates one notification per known recipient in a single outbox transaction, idempotent per event; a
// person named twice keeps their first row.
func (s *NotificationService) fanOut(ctx context.Context, key string, n notice, recipients []recipient) error {
	now := s.now().UTC()
	seen := map[string]bool{}
	var toCreate []*Notification
	for _, r := range recipients {
		userID, err := s.recipientID(ctx, r)
		if err != nil {
			return err
		}
		if userID == "" || seen[userID] {
			continue
		}
		seen[userID] = true
		row := n.row(key, userID, r.Kind, now)
		if r.Title != "" {
			row.SubjectTitle = r.Title
		}
		toCreate = append(toCreate, row)
	}
	return s.create(ctx, n, toCreate)
}

// recipientID resolves a recipient to a user id; an unknown login is nobody to notify.
func (s *NotificationService) recipientID(ctx context.Context, r recipient) (string, error) {
	if id := strings.TrimSpace(r.UserID); id != "" {
		return id, nil
	}
	login := strings.TrimSpace(r.Login)
	if login == "" {
		return "", nil
	}
	user, err := s.users.GetUserByLogin(ctx, login)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve recipient %s: %w", login, err)
	}
	if user == nil {
		return "", nil
	}
	return strings.TrimSpace(user.ID), nil
}

// fanOutByUserID creates one notification per already-resolved user id, skipping fanOut's login lookup for
// recipient lists that came from a workspace-membership scan rather than a login.
func (s *NotificationService) fanOutByUserID(ctx context.Context, key string, n notice, kind Kind, userIDs []string) error {
	now := s.now().UTC()
	var toCreate []*Notification
	for _, uid := range userIDs {
		uid = strings.TrimSpace(uid)
		if uid == "" {
			continue
		}
		toCreate = append(toCreate, n.row(key, uid, kind, now))
	}
	return s.create(ctx, n, toCreate)
}

func (n notice) row(key, userID string, kind Kind, now time.Time) *Notification {
	return &Notification{
		ID:           key + ":" + userID,
		UserID:       userID,
		WorkspaceID:  n.workspaceID,
		Kind:         kind,
		SubjectType:  n.subjectType,
		SubjectID:    n.subjectID,
		SubjectTitle: n.subjectTitle,
		CreatedAt:    now,
	}
}

// create writes the rows with both outbox events; every fan-out passes through here, so the actor is dropped once for all kinds.
func (s *NotificationService) create(ctx context.Context, n notice, toCreate []*Notification) error {
	toCreate = slices.DeleteFunc(toCreate, func(row *Notification) bool {
		return row.UserID == n.actorID || !s.canOpen(ctx, n, row.UserID)
	})
	if len(toCreate) == 0 {
		return nil
	}
	items := make([]NotificationPushItem, 0, len(toCreate))
	userIDs := make([]string, 0, len(toCreate))
	for _, row := range toCreate {
		items = append(items, NotificationPushItem{ID: row.ID, UserID: row.UserID, WorkspaceID: row.WorkspaceID})
		userIDs = append(userIDs, row.UserID)
	}
	created := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicNotificationCreated, Payload: NotificationCreatedEvent{UserIDs: userIDs, WorkspaceID: n.workspaceID, ProjectID: n.projectID}}
	push := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicNotificationPushRequested, Payload: NotificationPushRequestedEvent{Notifications: items}}
	if err := s.repo.CreateMany(ctx, toCreate, created, push); err != nil {
		return fmt.Errorf("create notifications: %w", err)
	}
	return nil
}

// evtKey returns a stable per-event key for idempotent fan-out, injected by the composition root via the context.
func evtKey(ctx context.Context) string {
	if k, ok := ctx.Value(eventKeyCtx{}).(string); ok && k != "" {
		return k
	}
	return ids.New()
}

type eventKeyCtx struct{}

// CtxWithEventKey carries the source event ID the fan-out uses to derive idempotent notification IDs.
func CtxWithEventKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, eventKeyCtx{}, key)
}

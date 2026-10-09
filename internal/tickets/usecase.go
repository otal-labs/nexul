package tickets

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Service is the tickets use-case layer (ADR 0019); mutations enqueue events into the transactional outbox.
type Service struct {
	repo     Repo
	statuses StatusStore
	users    UserLogins
	types    TicketTypes
	testing  Testing
	gate     Gate
	docs     SourceDocs
	people   ProjectPeople
	live     LiveSessions
	now      func() time.Time
}

// LiveSessions fences a body write against the ticket's open editing room and resets the room after it, so the
// write wins over open editors (ADR 0109); tickets never imports collab (ADR 0017).
type LiveSessions interface {
	Reset(ctx context.Context, roomID string, write func(context.Context) error) error
}

// SetLiveSessions wires the editing rooms a body write must win over; without it a write touches only the ticket.
func (s *Service) SetLiveSessions(live LiveSessions) { s.live = live }

// Gate is the permission check a ticket passes through before the caller touches it (the access domain, ADR 0042).
type Gate interface {
	RequireProject(ctx context.Context, projectID string, action permissions.Action) error
}

// NewService wires the tickets use-cases; users resolves the reporter's login and may be nil, recording the user id.
func NewService(repo Repo, statuses StatusStore, users UserLogins) *Service {
	return &Service{repo: repo, statuses: statuses, users: users, now: time.Now}
}

// SetGate wires the permission check; unset, only the server's own calls pass.
func (s *Service) SetGate(g Gate) { s.gate = g }

// ProjectPeople says whether the person behind a login may open a project, so a ticket's developer or tester is never
// someone its project is hidden from (ADR 0097).
type ProjectPeople interface {
	MayOpen(ctx context.Context, login, projectID string) (bool, error)
}

// SetProjectPeople wires the developer and tester check; unset, any login is accepted.
func (s *Service) SetProjectPeople(p ProjectPeople) { s.people = p }

// requirePerson refuses, as invalid, a developer or tester who may not open projectID.
func (s *Service) requirePerson(ctx context.Context, projectID string, role Role, login string) error {
	if login == "" || s.people == nil {
		return nil
	}
	ok, err := s.people.MayOpen(ctx, login, projectID)
	if err != nil {
		return fmt.Errorf("check %s %s: %w", role, login, err)
	}
	if !ok {
		return fmt.Errorf("%w: %s cannot open this ticket's project, so they cannot be its %s", apperrs.ErrInvalid, login, role)
	}
	return nil
}

func (s *Service) require(ctx context.Context, projectID string, action permissions.Action) error {
	if s.gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.gate.RequireProject(ctx, projectID, action)
}

// load fetches a ticket the caller may act on with action.
func (s *Service) load(ctx context.Context, id string, action permissions.Action) (*Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.require(ctx, t.ProjectID, action); err != nil {
		return nil, err
	}
	return t, nil
}

// readable keeps the tickets the caller may read.
func (s *Service) readable(ctx context.Context, ts []*Ticket) ([]*Ticket, error) {
	return permissions.Filter(ts, func(t *Ticket) string { return t.ProjectID }, func(projectID string) error {
		return s.require(ctx, projectID, permissions.TicketsRead)
	})
}

// readableIDs keeps the ids of tickets the caller may read; an id naming no ticket is left out too.
func (s *Service) readableIDs(ctx context.Context, ids []string) ([]string, error) {
	ts := make([]*Ticket, 0, len(ids))
	for _, id := range ids {
		t, err := s.repo.GetByID(ctx, id)
		if errors.Is(err, apperrs.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	ts, err := s.readable(ctx, ts)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.ID
	}
	return out, nil
}

// SetTicketTypes wires the type lookup; unset, an MCP-filed ticket keeps the body it was given and no type counts as a bug.
func (s *Service) SetTicketTypes(t TicketTypes) { s.types = t }

// CreateOptions carries a new ticket's optional metadata; a bug needs OriginID (where it was found) or OriginUnknown.
type CreateOptions struct {
	CategoryID    string
	TypeID        string
	Tester        string
	ViaMCP        bool
	OriginID      string
	OriginUnknown bool
}

// Create persists a new open ticket and enqueues ticket.created; a workspace needs a project first.
func (s *Service) Create(ctx context.Context, projectID, title, body, docID, developer string, opts ...CreateOptions) (*Ticket, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required — create a project before creating tickets", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, projectID, permissions.TicketsWrite); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", apperrs.ErrInvalid)
	}
	var opt CreateOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	opt.OriginID = strings.TrimSpace(opt.OriginID)
	typeID, err := s.typeOrDefault(ctx, projectID, opt.TypeID)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	opt.TypeID = typeID
	status, err := s.firstColumn(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	if err := s.checkFoundIn(ctx, opt); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	if err := s.requirePerson(ctx, projectID, RoleDeveloper, strings.TrimSpace(developer)); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	if err := s.requirePerson(ctx, projectID, RoleTester, strings.TrimSpace(opt.Tester)); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	docID = strings.TrimSpace(docID)
	if err := s.requireSource(ctx, docID); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	body, err = s.defaultBody(ctx, body, opt)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	now := s.now().UTC()
	t := &Ticket{
		ID:         ids.New(),
		ProjectID:  projectID,
		CategoryID: strings.TrimSpace(opt.CategoryID),
		TypeID:     opt.TypeID,
		Title:      title,
		Body:       body,
		Status:     status,
		DocID:      docID,
		Developer:  strings.TrimSpace(developer),
		Tester:     strings.TrimSpace(opt.Tester),
		Reporter:   s.reporter(ctx, opt.ViaMCP),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.persist(ctx, t, opt); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	// Position is assigned atomically in its own transaction (ADR 0002); re-fetch to return the persisted value.
	created, err := s.repo.GetByID(ctx, t.ID)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	return created, nil
}

// typeOrDefault gives a ticket filed without a type the project's first one, as the web's create form preselects.
func (s *Service) typeOrDefault(ctx context.Context, projectID, typeID string) (string, error) {
	typeID = strings.TrimSpace(typeID)
	if typeID != "" || s.types == nil {
		return typeID, nil
	}
	first, err := s.types.FirstType(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("default type of project %s: %w", projectID, err)
	}
	return first, nil
}

// firstColumn is where a new ticket lands, so every board shows it; a project without columns keeps the open status.
func (s *Service) firstColumn(ctx context.Context, projectID string) (Status, error) {
	first, err := s.statuses.FirstStatus(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("first column of project %s: %w", projectID, err)
	}
	if first == "" {
		return StatusOpen, nil
	}
	return Status(first), nil
}

// persist writes the ticket, and its found-in link in the same transaction when it was filed with one.
func (s *Service) persist(ctx context.Context, t *Ticket, opt CreateOptions) error {
	created := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicCreated, Payload: CreatedEvent{Ticket: *t, MentionedUserIDs: richtext.PersonMentions(t.Body)}}
	if opt.OriginID == "" && !opt.OriginUnknown {
		return s.repo.Create(ctx, t, created)
	}
	link := TicketLink{TicketID: t.ID, Kind: LinkFoundIn, TargetID: opt.OriginID, CreatedAt: t.CreatedAt}
	return s.repo.CreateWithLink(ctx, t, link, created, linkEvent(TopicLinkCreated, link))
}

// checkFoundIn holds a new bug to ADR 0064: it names the ticket it was found in, or its reporter marks the origin unknown.
func (s *Service) checkFoundIn(ctx context.Context, opt CreateOptions) error {
	if opt.OriginID != "" && opt.OriginUnknown {
		return fmt.Errorf("%w: give an origin_id or mark the origin unknown, not both", apperrs.ErrInvalid)
	}
	if opt.OriginID != "" {
		return s.originExists(ctx, opt.OriginID)
	}
	if opt.OriginUnknown {
		return nil
	}
	bug, err := s.isBug(ctx, opt.TypeID)
	if err != nil {
		return err
	}
	if bug {
		return fmt.Errorf("%w: a bug needs the ticket it was found in (origin_id), or origin_unknown when nobody knows", apperrs.ErrInvalid)
	}
	return nil
}

// originExists reports a missing origin as invalid input, not a missing ticket being created.
func (s *Service) originExists(ctx context.Context, originID string) error {
	_, err := s.repo.GetByID(ctx, originID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("%w: found-in ticket %s does not exist", apperrs.ErrInvalid, originID)
	}
	return err
}

func (s *Service) isBug(ctx context.Context, typeID string) (bool, error) {
	typeID = strings.TrimSpace(typeID)
	if s.types == nil || typeID == "" {
		return false, nil
	}
	name, err := s.types.TypeName(ctx, typeID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return false, fmt.Errorf("%w: ticket type %s does not exist", apperrs.ErrInvalid, typeID)
	}
	if err != nil {
		return false, fmt.Errorf("ticket type %s: %w", typeID, err)
	}
	return IsBugType(name), nil
}

// defaultBody hands an agent's empty MCP ticket its type's template, so it carries the same sections a person's would.
func (s *Service) defaultBody(ctx context.Context, body string, opt CreateOptions) (string, error) {
	typeID := strings.TrimSpace(opt.TypeID)
	if !opt.ViaMCP || s.types == nil || typeID == "" || strings.TrimSpace(body) != "" {
		return body, nil
	}
	template, err := s.types.BodyTemplate(ctx, typeID)
	if err != nil {
		return "", fmt.Errorf("body template for type %s: %w", typeID, err)
	}
	return template, nil
}

// reporter prefers the automation behind an automation token; a person acting through MCP is recorded as user:mcp.
func (s *Service) reporter(ctx context.Context, viaMCP bool) Reporter {
	actor, ok := identity.ActorFromCtx(ctx)
	if ok && actor.Automation != nil {
		return Reporter{Kind: ActorKindAutomation, AutomationID: actor.Automation.ID, AutomationName: actor.Automation.Name}
	}
	kind := ActorKindUser
	if viaMCP {
		kind = ActorKindUserMCP
	}
	if !ok || actor.ID == "" {
		return Reporter{Kind: kind}
	}
	return Reporter{Kind: kind, Login: s.login(ctx, actor.ID)}
}

// login falls back to the user id when no lookup is wired or it fails, so a reporter is never lost.
func (s *Service) login(ctx context.Context, userID string) string {
	if s.users == nil {
		return userID
	}
	login, err := s.users.LoginForUserID(ctx, userID)
	if err != nil || login == "" {
		return userID
	}
	return login
}

// Get returns a ticket by id.
func (s *Service) Get(ctx context.Context, id string) (*Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	t, err := s.load(ctx, id, permissions.TicketsRead)
	if err != nil {
		return nil, fmt.Errorf("get ticket %s: %w", id, err)
	}
	return t, nil
}

// ticketKey matches a ticket's human key, PREFIX-NUMBER (ADR 0004); a ticket id is a UUID and never matches.
var ticketKey = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9]{1,4})-([0-9]+)$`)

// Resolve returns a ticket by its id or by its key, such as REF-102. A key is unique only within a workspace, so
// workspace (an id or a slug) picks one; without it the key resolves among the tickets the caller can read, and
// matches in several workspaces are refused with their slugs rather than guessed (ADR 0089).
func (s *Service) Resolve(ctx context.Context, workspace, idOrKey string) (*Ticket, error) {
	idOrKey = strings.TrimSpace(idOrKey)
	m := ticketKey.FindStringSubmatch(idOrKey)
	if m == nil {
		return s.Get(ctx, idOrKey)
	}
	number, err := strconv.Atoi(m[2])
	if err != nil {
		return nil, fmt.Errorf("%w: ticket key %s has an out-of-range number", apperrs.ErrInvalid, idOrKey)
	}
	key := strings.ToUpper(m[1]) + "-" + m[2]
	matches, err := s.repo.ListByKey(ctx, strings.ToUpper(m[1]), number)
	if err != nil {
		return nil, fmt.Errorf("get ticket %s: %w", key, err)
	}
	workspace = strings.TrimSpace(workspace)
	var readable []KeyMatch
	var refused error
	for _, km := range matches {
		if workspace != "" && workspace != km.WorkspaceID && workspace != km.WorkspaceSlug {
			continue
		}
		if err := s.require(ctx, km.Ticket.ProjectID, permissions.TicketsRead); err != nil {
			refused = cmp.Or(refused, err)
			continue
		}
		readable = append(readable, km)
	}
	if len(readable) == 1 {
		return readable[0].Ticket, nil
	}
	if len(readable) > 1 {
		return nil, ambiguousKey(key, readable)
	}
	if refused != nil {
		return nil, fmt.Errorf("get ticket %s: %w", key, refused)
	}
	return nil, fmt.Errorf("get ticket %s: %w", key, apperrs.ErrNotFound)
}

// ambiguousKey names the workspaces a key was found in, so the caller can pass one of them.
func ambiguousKey(key string, matches []KeyMatch) error {
	var slugs []string
	for _, km := range matches {
		if !slices.Contains(slugs, km.WorkspaceSlug) {
			slugs = append(slugs, km.WorkspaceSlug)
		}
	}
	if len(slugs) == 1 {
		return fmt.Errorf("%w: %s matches more than one ticket in %s; use the ticket's id", apperrs.ErrConflict, key, slugs[0])
	}
	last := len(slugs) - 1
	return fmt.Errorf("%w: %s exists in %s and %s; pass workspace", apperrs.ErrConflict, key, strings.Join(slugs[:last], ", "), slugs[last])
}

// List returns every ticket the caller may read, oldest first.
func (s *Service) List(ctx context.Context) ([]*Ticket, error) {
	ts, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	return s.readable(ctx, ts)
}

// ListByDoc returns the tickets derived from a given doc.
func (s *Service) ListByDoc(ctx context.Context, docID string) ([]*Ticket, error) {
	if strings.TrimSpace(docID) == "" {
		return nil, fmt.Errorf("%w: doc id is required", apperrs.ErrInvalid)
	}
	ts, err := s.repo.ListByDoc(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("list tickets for doc %s: %w", docID, err)
	}
	return s.readable(ctx, ts)
}

// ListByProject returns the tickets in a given project.
func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*Ticket, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, projectID, permissions.TicketsRead); err != nil {
		return nil, fmt.Errorf("list tickets for project %s: %w", projectID, err)
	}
	ts, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list tickets for project %s: %w", projectID, err)
	}
	return ts, nil
}

// SetType changes a ticket's type id in place, keeping its identity; the ticket must exist.
func (s *Service) SetType(ctx context.Context, id, typeID string) (*Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	typeID = strings.TrimSpace(typeID)
	if typeID == "" {
		return nil, fmt.Errorf("%w: type id is required", apperrs.ErrInvalid)
	}
	current, err := s.load(ctx, id, permissions.TicketsWrite)
	if err != nil {
		return nil, fmt.Errorf("set type on ticket %s: %w", id, err)
	}
	if err := s.repo.UpdateType(ctx, id, typeID); err != nil {
		return nil, fmt.Errorf("set type on ticket %s: %w", id, err)
	}
	updated := *current
	updated.TypeID = typeID
	updated.UpdatedAt = s.now().UTC()
	return &updated, nil
}

// SetPerson sets a ticket's developer or tester to a member login in place; an empty login clears the role.
func (s *Service) SetPerson(ctx context.Context, id string, role Role, login string) (*Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	if role != RoleDeveloper && role != RoleTester {
		return nil, fmt.Errorf("%w: role must be developer or tester", apperrs.ErrInvalid)
	}
	login = strings.TrimSpace(login)
	current, err := s.load(ctx, id, permissions.TicketsWrite)
	if err != nil {
		return nil, fmt.Errorf("set %s on ticket %s: %w", role, id, err)
	}
	updated := *current
	field := &updated.Developer
	if role == RoleTester {
		field = &updated.Tester
	}
	previous := *field
	if previous == login {
		return current, nil
	}
	if err := s.requirePerson(ctx, current.ProjectID, role, login); err != nil {
		return nil, fmt.Errorf("set %s on ticket %s: %w", role, id, err)
	}
	*field = login
	updated.UpdatedAt = s.now().UTC()
	evts := []eventbus.OutboxEvent{{ID: ids.New(), Topic: personTopic(role), Payload: PersonChangedEvent{Ticket: updated, From: previous, To: login}}}
	if role == RoleDeveloper {
		evts = append(evts, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicAssigneeChanged, Payload: AssigneeChangedEvent{Ticket: updated, From: previous, To: login}})
	}
	if err := s.repo.UpdatePerson(ctx, id, role, login, evts...); err != nil {
		return nil, fmt.Errorf("set %s on ticket %s: %w", role, id, err)
	}
	return &updated, nil
}

// UpdateTicket keeps title non-empty like Create; body may be cleared.
func (s *Service) UpdateTicket(ctx context.Context, id, title, body string) (*Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", apperrs.ErrInvalid)
	}
	current, err := s.load(ctx, id, permissions.TicketsWrite)
	if err != nil {
		return nil, fmt.Errorf("update ticket %s: %w", id, err)
	}
	updated := *current
	updated.Title = title
	updated.Body = body
	updated.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicUpdated, Payload: UpdatedEvent{
		Ticket: updated, ActorID: statusActor(ctx).UserID, MentionedUserIDs: richtext.AddedPersonMentions(current.Body, body),
	}}
	write := func(ctx context.Context) error { return s.repo.UpdateTicket(ctx, id, title, body, evt) }
	if err := s.fence(ctx, id, body != current.Body, write); err != nil {
		return nil, fmt.Errorf("update ticket %s: %w", id, err)
	}
	return &updated, nil
}

// fence runs a body change through the ticket's live room when one is wired; a title-only write leaves the room be.
func (s *Service) fence(ctx context.Context, id string, bodyChanged bool, write func(context.Context) error) error {
	if s.live == nil || !bodyChanged {
		return write(ctx)
	}
	return s.live.Reset(ctx, id, write)
}

// CommitCollab writes a ticket's live room into its title and body; an empty title means unchanged, since only
// the client that renamed sends one.
func (s *Service) CommitCollab(ctx context.Context, id, title, body string) error {
	current, err := s.load(ctx, id, permissions.TicketsWrite)
	if err != nil {
		return fmt.Errorf("commit ticket %s: %w", id, err)
	}
	updated := *current
	if title = strings.TrimSpace(title); title != "" {
		updated.Title = title
	}
	updated.Body = body
	updated.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicUpdated, Payload: UpdatedEvent{
		Ticket: updated, ActorID: statusActor(ctx).UserID, MentionedUserIDs: richtext.AddedPersonMentions(current.Body, body),
	}}
	if err := s.repo.UpdateTicket(ctx, id, updated.Title, body, evt); err != nil {
		return fmt.Errorf("commit ticket %s: %w", id, err)
	}
	return nil
}

// CanJoin reports whether the caller may join a ticket's live room under action; the hub's context carries them.
func (s *Service) CanJoin(ctx context.Context, id string, action permissions.Action) (bool, error) {
	if _, err := s.load(ctx, id, action); err != nil {
		return false, err
	}
	return true, nil
}

// Locked never refuses a ticket's live edits: tickets have no lock, and a deleted one fails the read instead.
func (s *Service) Locked(ctx context.Context, id string) (bool, error) {
	_, err := s.repo.GetByID(ctx, id)
	return false, err
}

// AddLabel attaches a cross-cutting tag to a ticket; the ticket must exist, a duplicate label is a no-op.
func (s *Service) AddLabel(ctx context.Context, id, label string) (*Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	label = strings.TrimSpace(label)
	if label == "" {
		return nil, fmt.Errorf("%w: label is required", apperrs.ErrInvalid)
	}
	current, err := s.load(ctx, id, permissions.TicketsWrite)
	if err != nil {
		return nil, fmt.Errorf("add label to ticket %s: %w", id, err)
	}
	if err := s.repo.AddLabel(ctx, id, label); err != nil {
		return nil, fmt.Errorf("add label to ticket %s: %w", id, err)
	}
	updated := *current
	updated.Labels = append(updated.Labels, label)
	updated.UpdatedAt = s.now().UTC()
	return &updated, nil
}

// RemoveLabel detaches a cross-cutting tag from a ticket; the ticket must exist, a missing label is a no-op.
func (s *Service) RemoveLabel(ctx context.Context, id, label string) (*Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	label = strings.TrimSpace(label)
	if label == "" {
		return nil, fmt.Errorf("%w: label is required", apperrs.ErrInvalid)
	}
	current, err := s.load(ctx, id, permissions.TicketsWrite)
	if err != nil {
		return nil, fmt.Errorf("remove label from ticket %s: %w", id, err)
	}
	if err := s.repo.RemoveLabel(ctx, id, label); err != nil {
		return nil, fmt.Errorf("remove label from ticket %s: %w", id, err)
	}
	labels := make([]string, 0, len(current.Labels))
	for _, l := range current.Labels {
		if l != label {
			labels = append(labels, l)
		}
	}
	updated := *current
	updated.Labels = labels
	updated.UpdatedAt = s.now().UTC()
	return &updated, nil
}

// ListLabels returns the labels attached to a ticket.
func (s *Service) ListLabels(ctx context.Context, id string) ([]string, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	if _, err := s.load(ctx, id, permissions.TicketsRead); err != nil {
		return nil, fmt.Errorf("list labels for ticket %s: %w", id, err)
	}
	labels, err := s.repo.ListLabels(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list labels for ticket %s: %w", id, err)
	}
	return labels, nil
}

// ListAllLabels returns the distinct labels across the tickets the caller may read, ordered, for the board filter bar.
func (s *Service) ListAllLabels(ctx context.Context) ([]string, error) {
	ts, err := s.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all labels: %w", err)
	}
	var labels []string
	for _, t := range ts {
		labels = append(labels, t.Labels...)
	}
	slices.Sort(labels)
	return slices.Compact(labels), nil
}

// SetLabelColor works even for a label no ticket has used yet, with no separate registration step.
func (s *Service) SetLabelColor(ctx context.Context, projectID, label string, color colors.Color) (*LabelColor, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	label = strings.TrimSpace(label)
	if label == "" {
		return nil, fmt.Errorf("%w: label is required", apperrs.ErrInvalid)
	}
	if !colors.Valid(color) {
		return nil, fmt.Errorf("%w: color must be one of the suggested palette colors", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, projectID, permissions.TicketsWrite); err != nil {
		return nil, fmt.Errorf("set color for label %q: %w", label, err)
	}
	if err := s.repo.SetLabelColor(ctx, projectID, label, color); err != nil {
		return nil, fmt.Errorf("set color for label %q: %w", label, err)
	}
	return &LabelColor{Label: label, Color: color}, nil
}

// LabelColors batches lookups in one call since the board renders many badges per screen.
func (s *Service) LabelColors(ctx context.Context, projectID string, labels []string) (map[string]colors.Color, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, projectID, permissions.TicketsRead); err != nil {
		return nil, fmt.Errorf("label colors: %w", err)
	}
	cleaned := make([]string, 0, len(labels))
	for _, l := range labels {
		if l = strings.TrimSpace(l); l != "" {
			cleaned = append(cleaned, l)
		}
	}
	out, err := s.repo.LabelColors(ctx, projectID, cleaned)
	if err != nil {
		return nil, fmt.Errorf("label colors: %w", err)
	}
	return out, nil
}

// UpdateStatus records the automation as actor for an automation-token request, else the user.
func (s *Service) UpdateStatus(ctx context.Context, id string, to Status) (*Ticket, error) {
	if _, err := s.load(ctx, id, permissions.TicketsWrite); err != nil {
		return nil, fmt.Errorf("update ticket %s status: %w", id, err)
	}
	return s.transition(ctx, id, to, statusActor(ctx), "")
}

func statusActor(ctx context.Context) Actor {
	actor, _ := identity.ActorFromCtx(ctx)
	if actor.Automation != nil {
		return Actor{Kind: ActorKindAutomation, AutomationID: actor.Automation.ID, AutomationName: actor.Automation.Name}
	}
	return Actor{Kind: ActorKindUser, UserID: actor.ID}
}

// SetStatusAs records the given actor (an automation with its run id, or a play with its trail id on the actor).
func (s *Service) SetStatusAs(ctx context.Context, id string, to Status, actor Actor, runID string) (*Ticket, error) {
	return s.transition(ctx, id, to, actor, runID)
}

// transition publishes each extra event, built from the moved ticket, in the same transaction as the move.
func (s *Service) transition(ctx context.Context, id string, to Status, actor Actor, runID string, extra ...func(Ticket) eventbus.OutboxEvent) (*Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update ticket %s status: %w", id, err)
	}
	if current.Status == to {
		return current, nil
	}
	if !CanTransition(current.Status, to) {
		return nil, fmt.Errorf("%w: cannot move ticket from %s to %s", apperrs.ErrInvalid, current.Status, to)
	}
	exists, err := s.statuses.Exists(ctx, string(to))
	if err != nil {
		return nil, fmt.Errorf("check status %s: %w", to, err)
	}
	if !exists {
		return nil, fmt.Errorf("%w: status %q is not a configured board column", apperrs.ErrInvalid, to)
	}
	updated := *current
	updated.Status = to
	updated.UpdatedAt = s.now().UTC()
	evts := []eventbus.OutboxEvent{{ID: ids.New(), Topic: TopicStatusChanged, Payload: StatusChangedEvent{Ticket: updated, From: current.Status, To: to, Actor: actor, RunID: runID}}}
	for _, build := range extra {
		evts = append(evts, build(updated))
	}
	// Position is assigned atomically in its own transaction (ADR 0002); re-fetch to return the persisted value.
	if err := s.repo.UpdateStatus(ctx, id, to, evts...); err != nil {
		return nil, fmt.Errorf("update ticket %s status: %w", id, err)
	}
	fresh, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update ticket %s status: %w", id, err)
	}
	return fresh, nil
}

// SetPosition orders a ticket within its (status, category) pair (ADR 0002).
func (s *Service) SetPosition(ctx context.Context, id string, position int) (*Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	if position < 0 {
		return nil, fmt.Errorf("%w: position must be non-negative", apperrs.ErrInvalid)
	}
	current, err := s.load(ctx, id, permissions.TicketsWrite)
	if err != nil {
		return nil, fmt.Errorf("set position on ticket %s: %w", id, err)
	}
	if err := s.repo.SetPosition(ctx, id, position); err != nil {
		return nil, fmt.Errorf("set position on ticket %s: %w", id, err)
	}
	updated := *current
	updated.Position = position
	updated.UpdatedAt = s.now().UTC()
	return &updated, nil
}

// Delete publishes ticket.deleted via the outbox so consumers can react.
func (s *Service) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	t, err := s.load(ctx, id, permissions.TicketsDelete)
	if err != nil {
		return fmt.Errorf("delete ticket %s: %w", id, err)
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicDeleted, Payload: DeletedEvent{ID: t.ID, Title: t.Title, ProjectID: t.ProjectID}}
	if err := s.repo.Delete(ctx, id, evt); err != nil {
		return fmt.Errorf("delete ticket %s: %w", id, err)
	}
	return nil
}

// Search runs an FTS5 query over ticket titles and bodies, then the text of their notes.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("%w: query is required", apperrs.ErrInvalid)
	}
	if limit < 1 {
		limit = 20
	}
	results, err := s.repo.Search(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search tickets: %w", err)
	}
	out := make([]SearchResult, 0, len(results))
	for _, r := range results {
		t, err := s.load(ctx, r.ID, permissions.TicketsRead)
		if permissions.Refused(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		r.Ticket = t
		out = append(out, r)
	}
	return out, nil
}

// LinkPR is a no-op on a duplicate link; the ticket must already exist.
func (s *Service) LinkPR(ctx context.Context, id string, ref PRRef) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: ticket id is required", apperrs.ErrInvalid)
	}
	if ref.Owner == "" || ref.Repo == "" || ref.Number < 1 {
		return fmt.Errorf("%w: pr ref is incomplete", apperrs.ErrInvalid)
	}
	if _, err := s.load(ctx, id, permissions.TicketsWrite); err != nil {
		return fmt.Errorf("link pr to ticket %s: %w", id, err)
	}
	if err := s.repo.LinkPR(ctx, id, ref, PRStateOpen); err != nil {
		return fmt.Errorf("link pr to ticket %s: %w", id, err)
	}
	return nil
}

// LinkBranch allows a branch to link to multiple tickets; a duplicate link is a no-op.
func (s *Service) LinkBranch(ctx context.Context, id, owner, repo, branch string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: ticket id is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" || strings.TrimSpace(branch) == "" {
		return fmt.Errorf("%w: branch link is incomplete", apperrs.ErrInvalid)
	}
	if _, err := s.load(ctx, id, permissions.TicketsWrite); err != nil {
		return fmt.Errorf("link branch to ticket %s: %w", id, err)
	}
	if err := s.repo.LinkBranch(ctx, id, BranchLink{Owner: owner, Repo: repo, Branch: branch}); err != nil {
		return fmt.Errorf("link branch to ticket %s: %w", id, err)
	}
	return nil
}

// ListByPR returns the tickets a pull request is linked to.
func (s *Service) ListByPR(ctx context.Context, owner, repo string, number int) ([]*Ticket, error) {
	if owner == "" || repo == "" || number < 1 {
		return nil, fmt.Errorf("%w: owner, repo, and a positive PR number are required", apperrs.ErrInvalid)
	}
	ids, err := s.repo.ListIDsByPR(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("list tickets for PR %s/%s#%d: %w", owner, repo, number, err)
	}
	out := make([]*Ticket, 0, len(ids))
	for _, id := range ids {
		t, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("list tickets for PR %s/%s#%d: %w", owner, repo, number, err)
		}
		out = append(out, t)
	}
	return s.readable(ctx, out)
}

// ListLinks returns the PR and branch links recorded against a ticket (development section).
func (s *Service) ListLinks(ctx context.Context, id string) ([]PRLink, []BranchLink, error) {
	if strings.TrimSpace(id) == "" {
		return nil, nil, fmt.Errorf("%w: ticket id is required", apperrs.ErrInvalid)
	}
	if _, err := s.load(ctx, id, permissions.TicketsRead); err != nil {
		return nil, nil, fmt.Errorf("list links for ticket %s: %w", id, err)
	}
	prs, err := s.repo.ListPRLinks(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("list pr links for ticket %s: %w", id, err)
	}
	branches, err := s.repo.ListBranchLinks(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("list branch links for ticket %s: %w", id, err)
	}
	return prs, branches, nil
}

// DevStatus batches PR counts in one query so the board avoids a request per card; a ticket the caller may not
// read gets no entry.
func (s *Service) DevStatus(ctx context.Context, ids []string) (map[string]DevStatusCounts, error) {
	cleaned := make([]string, 0, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			cleaned = append(cleaned, id)
		}
	}
	cleaned, err := s.readableIDs(ctx, cleaned)
	if err != nil {
		return nil, fmt.Errorf("dev status: %w", err)
	}
	linksByTicket, err := s.repo.ListPRLinksBatch(ctx, cleaned)
	if err != nil {
		return nil, fmt.Errorf("dev status: %w", err)
	}
	out := make(map[string]DevStatusCounts, len(cleaned))
	for _, id := range cleaned {
		var counts DevStatusCounts
		for _, l := range linksByTicket[id] {
			switch l.State {
			case PRStateOpen:
				counts.Open++
			case PRStateMerged:
				counts.Merged++
			}
		}
		out[id] = counts
	}
	return out, nil
}

// evaluateCompletion finishes a ticket once a PR merges and none remain open; closed-unmerged PRs never block it (ADR 0021).
func (s *Service) evaluateCompletion(ctx context.Context, id string) error {
	links, err := s.repo.ListPRLinks(ctx, id)
	if err != nil {
		return fmt.Errorf("evaluate completion for ticket %s: %w", id, err)
	}
	merged, open := false, false
	for _, l := range links {
		if l.State == PRStateMerged {
			merged = true
		}
		if l.State == PRStateOpen {
			open = true
		}
	}
	if !merged || open {
		return nil
	}
	return s.finishTicket(ctx, id)
}

// finishTicket's conditional finished_at update ensures only one writer enqueues, even across redeliveries.
func (s *Service) finishTicket(ctx context.Context, id string) error {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("finish ticket %s: %w", id, err)
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicFinished, Payload: FinishedEvent{Ticket: *t}}
	if _, err := s.repo.SetFinishedAt(ctx, id, s.now().UTC(), evt); err != nil {
		return fmt.Errorf("finish ticket %s: %w", id, err)
	}
	return nil
}

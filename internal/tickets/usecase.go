package tickets

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// Service is the tickets use-case layer (ADR 0019); mutations enqueue events into the transactional outbox.
type Service struct {
	repo     Repo
	statuses StatusStore
	users    UserLogins
	types    TicketTypes
	now      func() time.Time
}

// NewService wires the tickets use-cases; users resolves the reporter's login and may be nil, recording the user id.
func NewService(repo Repo, statuses StatusStore, users UserLogins) *Service {
	return &Service{repo: repo, statuses: statuses, users: users, now: time.Now}
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
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", apperrs.ErrInvalid)
	}
	var opt CreateOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	opt.OriginID = strings.TrimSpace(opt.OriginID)
	if err := s.checkFoundIn(ctx, opt); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	body, err := s.defaultBody(ctx, body, opt)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	now := s.now().UTC()
	t := &Ticket{
		ID:         ids.New(),
		ProjectID:  projectID,
		CategoryID: strings.TrimSpace(opt.CategoryID),
		TypeID:     strings.TrimSpace(opt.TypeID),
		Title:      title,
		Body:       body,
		Status:     StatusOpen,
		DocID:      strings.TrimSpace(docID),
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

// persist writes the ticket, and its found-in link in the same transaction when it was filed with one.
func (s *Service) persist(ctx context.Context, t *Ticket, opt CreateOptions) error {
	created := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicCreated, Payload: CreatedEvent{Ticket: *t}}
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
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get ticket %s: %w", id, err)
	}
	return t, nil
}

// List returns all tickets, oldest first.
func (s *Service) List(ctx context.Context) ([]*Ticket, error) {
	ts, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	return ts, nil
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
	return ts, nil
}

// ListByProject returns the tickets in a given project.
func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*Ticket, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
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
	current, err := s.repo.GetByID(ctx, id)
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
	current, err := s.repo.GetByID(ctx, id)
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
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update ticket %s: %w", id, err)
	}
	updated := *current
	updated.Title = title
	updated.Body = body
	updated.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicUpdated, Payload: UpdatedEvent{Ticket: updated}}
	if err := s.repo.UpdateTicket(ctx, id, title, body, evt); err != nil {
		return nil, fmt.Errorf("update ticket %s: %w", id, err)
	}
	return &updated, nil
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
	current, err := s.repo.GetByID(ctx, id)
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
	current, err := s.repo.GetByID(ctx, id)
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
	labels, err := s.repo.ListLabels(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list labels for ticket %s: %w", id, err)
	}
	return labels, nil
}

// ListAllLabels returns the distinct labels across all tickets, ordered, for the board filter bar.
func (s *Service) ListAllLabels(ctx context.Context) ([]string, error) {
	labels, err := s.repo.ListAllLabels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all labels: %w", err)
	}
	return labels, nil
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
	if actor, ok := identity.ActorFromCtx(ctx); ok && actor.Automation != nil {
		return s.transition(ctx, id, to, Actor{
			Kind:           ActorKindAutomation,
			AutomationID:   actor.Automation.ID,
			AutomationName: actor.Automation.Name,
		}, "")
	}
	return s.transition(ctx, id, to, Actor{Kind: ActorKindUser}, "")
}

// SetStatusAs records the given actor (an automation with its run id, or a play with its trail id on the actor).
func (s *Service) SetStatusAs(ctx context.Context, id string, to Status, actor Actor, runID string) (*Ticket, error) {
	return s.transition(ctx, id, to, actor, runID)
}

func (s *Service) transition(ctx context.Context, id string, to Status, actor Actor, runID string) (*Ticket, error) {
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
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicStatusChanged, Payload: StatusChangedEvent{Ticket: updated, From: current.Status, To: to, Actor: actor, RunID: runID}}
	// Position is assigned atomically in its own transaction (ADR 0002); re-fetch to return the persisted value.
	if err := s.repo.UpdateStatus(ctx, id, to, evt); err != nil {
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
	current, err := s.repo.GetByID(ctx, id)
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
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete ticket %s: %w", id, err)
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicDeleted, Payload: DeletedEvent{ID: t.ID, Title: t.Title}}
	if err := s.repo.Delete(ctx, id, evt); err != nil {
		return fmt.Errorf("delete ticket %s: %w", id, err)
	}
	return nil
}

// Search runs an FTS5 query over ticket titles and bodies.
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
	return results, nil
}

// LinkPR is a no-op on a duplicate link; the ticket must already exist.
func (s *Service) LinkPR(ctx context.Context, id string, ref PRRef) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: ticket id is required", apperrs.ErrInvalid)
	}
	if ref.Owner == "" || ref.Repo == "" || ref.Number < 1 {
		return fmt.Errorf("%w: pr ref is incomplete", apperrs.ErrInvalid)
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
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
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return fmt.Errorf("link branch to ticket %s: %w", id, err)
	}
	if err := s.repo.LinkBranch(ctx, id, BranchLink{Owner: owner, Repo: repo, Branch: branch}); err != nil {
		return fmt.Errorf("link branch to ticket %s: %w", id, err)
	}
	return nil
}

// ListLinks returns the PR and branch links recorded against a ticket (development section).
func (s *Service) ListLinks(ctx context.Context, id string) ([]PRLink, []BranchLink, error) {
	if strings.TrimSpace(id) == "" {
		return nil, nil, fmt.Errorf("%w: ticket id is required", apperrs.ErrInvalid)
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
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

// DevStatus batches PR counts in one query so the board avoids a request per card.
func (s *Service) DevStatus(ctx context.Context, ids []string) (map[string]DevStatusCounts, error) {
	cleaned := make([]string, 0, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			cleaned = append(cleaned, id)
		}
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

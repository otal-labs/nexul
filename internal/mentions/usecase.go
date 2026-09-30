package mentions

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// mentionKeyRe matches a PREFIX-NUMBER display id (ADR 0004); uppercase-only, mixed-case falls through to search.
var mentionKeyRe = regexp.MustCompile(`^([A-Z][A-Z0-9]{1,4})-(\d+)$`)

// Config wires the resolution seam; a nil source makes Resolve/Search fail closed.
type Config struct {
	Tickets  TicketSource
	Docs     DocSource
	Statuses StatusSource
	Access   AccessChecker
	// Projects and TicketTypes resolve the chip template's tokens (spec.md 6); required like the sources above.
	Projects    ProjectSource
	TicketTypes TicketTypeSource
	// People finds and resolves person mentions; without it neither search nor resolve returns people.
	People PeopleSource
}

// Service resolves mention references and search targets for the @ picker.
type Service struct {
	cfg Config
}

// New wires the mentions service over the given sources.
func New(cfg Config) *Service {
	return &Service{cfg: cfg}
}

// SetPeople wires People after construction, since it is built after this service.
func (s *Service) SetPeople(p PeopleSource) {
	s.cfg.People = p
}

// Resolve renders a batch of refs as chips; missing targets are omitted, unopenable ones get can_open=false.
func (s *Service) Resolve(ctx context.Context, refs []Ref) ([]Chip, error) {
	if s.cfg.Tickets == nil || s.cfg.Docs == nil || s.cfg.Statuses == nil || s.cfg.Projects == nil || s.cfg.TicketTypes == nil {
		return nil, errors.New("mentions: resolution sources are not wired")
	}
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	people, err := s.peopleFor(ctx, actor.ID, refs)
	if err != nil {
		return nil, err
	}
	chips := make([]Chip, 0, len(refs))
	seen := map[string]bool{}
	for _, ref := range refs {
		key := ref.Type + ":" + ref.ID
		if ref.Type == "" || ref.ID == "" || seen[key] {
			continue
		}
		seen[key] = true
		chip, ok, err := s.resolveRef(ctx, actor.ID, people, ref)
		if err != nil {
			return nil, err
		}
		if ok {
			chips = append(chips, chip)
		}
	}
	return chips, nil
}

// resolveRef resolves one ref to a chip; ok is false when the target was not found.
func (s *Service) resolveRef(ctx context.Context, actorID string, people map[string]Person, ref Ref) (Chip, bool, error) {
	switch ref.Type {
	case string(KindPerson):
		p, ok := people[ref.ID]
		return personChip(p), ok, nil
	case string(KindTicket):
		chip, err := s.resolveTicket(ctx, ref.ID)
		if err != nil {
			if errors.Is(err, apperrs.ErrNotFound) {
				return Chip{}, false, nil
			}
			return Chip{}, false, err
		}
		return chip, true, nil
	case string(KindDoc):
		chip, err := s.resolveDoc(ctx, actorID, ref.ID)
		if err != nil {
			if errors.Is(err, apperrs.ErrNotFound) {
				return Chip{}, false, nil
			}
			return Chip{}, false, err
		}
		return chip, true, nil
	}
	return Chip{}, false, nil
}

// Search returns autocomplete entries: people, tickets, and docs, filtered to what the actor can see. People come
// from workspaceID's People, or from every workspace the actor shares when it is empty. A ticket key sorts first,
// then people whose login or display name starts with the query, then tickets, docs, and the remaining people.
func (s *Service) Search(ctx context.Context, query, workspaceID string, limit int) ([]SearchResult, error) {
	if s.cfg.Tickets == nil || s.cfg.Docs == nil || s.cfg.Statuses == nil {
		return nil, errors.New("mentions: resolution sources are not wired")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("%w: query is required", apperrs.ErrInvalid)
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	ticketHits, err := s.cfg.Tickets.Search(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search mention tickets: %w", err)
	}
	docHits, err := s.cfg.Docs.Search(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search mention docs: %w", err)
	}
	people, err := s.people(ctx, actor.ID, workspaceID)
	if err != nil {
		return nil, err
	}
	startsWith, contains := matchPeople(people, query)

	results := make([]SearchResult, 0, len(ticketHits)+len(docHits)+1)
	seenTickets := map[string]bool{}

	keyResults, err := s.resolveMentionKey(ctx, query, workspaceID)
	if err != nil {
		return nil, err
	}
	for _, r := range keyResults {
		results = append(results, r)
		seenTickets[r.ID] = true
	}

	results = append(results, startsWith...)
	results = append(results, s.ticketSearchResults(ctx, ticketHits, seenTickets)...)
	results = append(results, s.docSearchResults(ctx, actor.ID, docHits)...)
	results = append(results, contains...)
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// resolveMentionKey returns the openable tickets an exact PREFIX-NUMBER names (spec.md 7), sorted first.
func (s *Service) resolveMentionKey(ctx context.Context, query, workspaceID string) ([]SearchResult, error) {
	m := mentionKeyRe.FindStringSubmatch(query)
	if m == nil {
		return nil, nil
	}
	number, convErr := strconv.Atoi(m[2])
	if convErr != nil {
		return nil, nil
	}
	ts, err := s.cfg.Tickets.ListByKey(ctx, strings.TrimSpace(workspaceID), m[1], number)
	if err != nil {
		return nil, fmt.Errorf("resolve mention key %q: %w", query, err)
	}
	var out []SearchResult
	for _, t := range ts {
		if s.canOpenTicket(ctx, t.ProjectID) {
			out = append(out, SearchResult{Type: string(KindTicket), ID: t.ID, Title: t.Title, StatusLabel: s.statusLabel(ctx, t.Status), CanOpen: true})
		}
	}
	return out, nil
}

func (s *Service) ticketSearchResults(ctx context.Context, hits []SearchHit, seen map[string]bool) []SearchResult {
	results := make([]SearchResult, 0, len(hits))
	for _, h := range hits {
		if seen[h.ID] {
			continue
		}
		seen[h.ID] = true
		ticket, err := s.cfg.Tickets.GetByID(ctx, h.ID)
		if err != nil || !s.canOpenTicket(ctx, ticket.ProjectID) {
			continue
		}
		results = append(results, SearchResult{Type: string(KindTicket), ID: h.ID, Title: h.Title, StatusLabel: s.statusLabel(ctx, ticket.Status), CanOpen: true})
	}
	return results
}

func (s *Service) docSearchResults(ctx context.Context, actorID string, hits []SearchHit) []SearchResult {
	results := make([]SearchResult, 0, len(hits))
	for _, h := range hits {
		if !s.canOpenDoc(ctx, actorID, h.ID) {
			continue
		}
		results = append(results, SearchResult{Type: string(KindDoc), ID: h.ID, Title: h.Title, CanOpen: true})
	}
	return results
}

// people reads the People the actor sees; a missing source means no people rather than an error.
func (s *Service) people(ctx context.Context, actorID, workspaceID string) ([]Person, error) {
	if s.cfg.People == nil {
		return nil, nil
	}
	people, err := s.cfg.People.People(ctx, actorID, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("list people for mentions: %w", err)
	}
	return people, nil
}

// peopleFor indexes everyone the actor shares a workspace with by user id, reading them only when refs name a person.
func (s *Service) peopleFor(ctx context.Context, actorID string, refs []Ref) (map[string]Person, error) {
	if !slices.ContainsFunc(refs, func(r Ref) bool { return r.Type == string(KindPerson) }) {
		return nil, nil
	}
	people, err := s.people(ctx, actorID, "")
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Person, len(people))
	for _, p := range people {
		byID[p.UserID] = p
	}
	return byID, nil
}

// matchPeople splits case-insensitive login and display-name matches into prefix and substring hits, exact first.
func matchPeople(people []Person, query string) (startsWith, contains []SearchResult) {
	q := strings.ToLower(query)
	var exact []SearchResult
	for _, p := range people {
		login, name := strings.ToLower(p.Login), strings.ToLower(p.DisplayName)
		switch {
		case login == q || name == q:
			exact = append(exact, personResult(p))
		case strings.HasPrefix(login, q) || strings.HasPrefix(name, q):
			startsWith = append(startsWith, personResult(p))
		case strings.Contains(login, q) || strings.Contains(name, q):
			contains = append(contains, personResult(p))
		}
	}
	return append(exact, startsWith...), contains
}

func personLabel(p Person) string {
	if p.DisplayName != "" {
		return p.DisplayName
	}
	return p.Login
}

func personResult(p Person) SearchResult {
	return SearchResult{Type: string(KindPerson), ID: p.UserID, Title: personLabel(p), Login: p.Login, AvatarURL: p.AvatarURL, CanOpen: true}
}

func personChip(p Person) Chip {
	return Chip{Type: string(KindPerson), ID: p.UserID, Title: personLabel(p), Login: p.Login, AvatarURL: p.AvatarURL, CanOpen: true}
}

func (s *Service) resolveTicket(ctx context.Context, id string) (Chip, error) {
	t, err := s.cfg.Tickets.GetByID(ctx, id)
	if err != nil {
		return Chip{}, err
	}
	prefix := s.projectPrefix(ctx, t.ProjectID)
	return Chip{
		Type:           string(KindTicket),
		ID:             t.ID,
		Title:          t.Title,
		Status:         t.Status,
		StatusLabel:    s.statusLabel(ctx, t.Status),
		CanOpen:        s.canOpenTicket(ctx, t.ProjectID),
		ProjectPrefix:  prefix,
		ProjectNumber:  t.Number,
		TypeLabel:      s.typeLabel(ctx, t.TypeID),
		DeveloperLabel: t.Developer,
	}, nil
}

func (s *Service) resolveDoc(ctx context.Context, actorID, id string) (Chip, error) {
	d, err := s.cfg.Docs.GetByID(ctx, id)
	if err != nil {
		return Chip{}, err
	}
	return Chip{
		Type:    string(KindDoc),
		ID:      d.ID,
		Title:   d.Title,
		CanOpen: s.canOpenDoc(ctx, actorID, d.ID),
	}, nil
}

// statusLabel resolves a status id's column name; a since-deleted status falls back to its id.
func (s *Service) statusLabel(ctx context.Context, id string) string {
	if id == "" {
		return ""
	}
	st, err := s.cfg.Statuses.Get(ctx, id)
	if err != nil || st == nil || st.Name == "" {
		return id
	}
	return st.Name
}

// projectPrefix resolves a project to its Prefix (ADR 0004); a deleted or unset project degrades to "".
func (s *Service) projectPrefix(ctx context.Context, projectID string) string {
	if projectID == "" {
		return ""
	}
	p, err := s.cfg.Projects.Get(ctx, projectID)
	if err != nil || p == nil {
		return ""
	}
	return p.Prefix
}

// typeLabel resolves a ticket type id to its display name, falling back to the id itself.
func (s *Service) typeLabel(ctx context.Context, id string) string {
	if id == "" {
		return ""
	}
	tt, err := s.cfg.TicketTypes.Get(ctx, id)
	if err != nil || tt == nil || tt.Name == "" {
		return id
	}
	return tt.Name
}

func (s *Service) canOpenTicket(ctx context.Context, projectID string) bool {
	if s.cfg.Access == nil {
		return false
	}
	return s.cfg.Access.RequireProject(ctx, projectID, permissions.TicketsRead) == nil
}

func (s *Service) canOpenDoc(ctx context.Context, actorID, docID string) bool {
	if s.cfg.Access == nil {
		return false
	}
	ok, err := s.cfg.Access.Can(ctx, actorID, docID, permissions.DocsRead)
	if err != nil {
		return false
	}
	return ok
}

func (s *Service) actor(ctx context.Context) (identity.Actor, error) {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return identity.Actor{}, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	return actor, nil
}

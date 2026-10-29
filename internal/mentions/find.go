package mentions

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Kinds the search box finds beside tickets and docs; each is listed by its own domain through a NamedSource.
const (
	KindPullRequest Kind = "pull_request"
	KindStack       Kind = "stack"
	KindAutomation  Kind = "automation"
	KindChannel     Kind = "channel"
	KindMemory      Kind = "memory"
)

// findKinds is the order the box shows its kinds in.
var findKinds = []Kind{KindTicket, KindDoc, KindPullRequest, KindStack, KindAutomation, KindChannel, KindMemory}

// keyTier ranks a ticket its key names above any title match.
const keyTier = 4

// findScan is how many full-text candidates a ticket or doc search ranks before cutting to the limit.
const findScan = 50

// Named is one entity a NamedSource lists, found by its title alone.
type Named struct {
	ID        string
	Title     string
	Hint      string
	ProjectID string
	// TicketID is the ticket a pull request opens, since pull requests have no page of their own.
	TicketID  string
	UpdatedAt time.Time
}

// NamedSource lists one kind's entities in a workspace that the actor on ctx can open. It may narrow by the query's
// words in SQL, but the service decides what matches, so listing everything is correct too.
type NamedSource interface {
	Find(ctx context.Context, workspaceID string, words []string) ([]Named, error)
}

// Hit is one result of the workspace search box; InBody marks a ticket or doc found only through its body or notes.
type Hit struct {
	Type        string    `json:"type"`
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Hint        string    `json:"hint,omitempty"`
	StatusLabel string    `json:"status_label,omitempty"`
	StatusKind  string    `json:"status_kind,omitempty"`
	ProjectID   string    `json:"project_id,omitempty"`
	TicketID    string    `json:"ticket_id,omitempty"`
	InBody      bool      `json:"in_body"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Found is one answer of the search box: hits in kind order, title hits before body hits, and the kinds cut short.
type Found struct {
	Hits []Hit    `json:"hits"`
	More []string `json:"more"`
}

type rankedHit struct {
	hit  Hit
	tier int
}

// SetNamed wires the named sources after construction, since their domains are built after this service.
func (s *Service) SetNamed(named map[Kind]NamedSource) {
	s.cfg.Named = named
}

// Find answers the workspace search box: every kind the actor can open in workspaceID, or only kind when it is set,
// at most limit title hits and limit body hits per kind. Within a kind a title equal to the query ranks first, then
// one holding each word whole, then one holding each word as a prefix; ties go to the most recently updated.
func (s *Service) Find(ctx context.Context, query, workspaceID, kind string, limit int) (Found, error) {
	if s.cfg.Tickets == nil || s.cfg.Docs == nil || s.cfg.Statuses == nil || s.cfg.Projects == nil {
		return Found{}, errors.New("mentions: resolution sources are not wired")
	}
	words, workspaceID := searchWords(query), strings.TrimSpace(workspaceID)
	if err := checkFind(words, workspaceID, kind); err != nil {
		return Found{}, err
	}
	if limit < 1 || limit > 50 {
		limit = 5
	}
	actor, err := s.actor(ctx)
	if err != nil {
		return Found{}, err
	}
	found := Found{Hits: []Hit{}, More: []string{}}
	for _, k := range findKinds {
		if kind != "" && Kind(kind) != k {
			continue
		}
		ranked, err := s.findKind(ctx, actor.ID, workspaceID, strings.TrimSpace(query), words, k)
		if err != nil {
			return Found{}, err
		}
		hits, more := cutRanked(ranked, limit)
		found.Hits = append(found.Hits, hits...)
		if more {
			found.More = append(found.More, string(k))
		}
	}
	return found, nil
}

func checkFind(words []string, workspaceID, kind string) error {
	if len(words) == 0 {
		return fmt.Errorf("%w: query is required", apperrs.ErrInvalid)
	}
	if workspaceID == "" {
		return fmt.Errorf("%w: workspace_id is required", apperrs.ErrInvalid)
	}
	if kind != "" && !slices.Contains(findKinds, Kind(kind)) {
		return fmt.Errorf("%w: kind %q must be one of ticket, doc, pull_request, stack, automation, channel, or memory", apperrs.ErrInvalid, kind)
	}
	return nil
}

func (s *Service) findKind(ctx context.Context, actorID, workspaceID, query string, words []string, kind Kind) ([]rankedHit, error) {
	switch kind {
	case KindTicket:
		return s.findTickets(ctx, workspaceID, query, words)
	case KindDoc:
		return s.findDocs(ctx, actorID, workspaceID, query, words)
	}
	src := s.cfg.Named[kind]
	if src == nil {
		return nil, nil
	}
	named, err := src.Find(ctx, workspaceID, words)
	if permissions.Refused(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find %ss: %w", kind, err)
	}
	out := make([]rankedHit, 0, len(named))
	for _, n := range named {
		tier := titleTier(n.Title, words)
		if tier < 0 {
			continue
		}
		out = append(out, rankedHit{tier: tier, hit: Hit{
			Type: string(kind), ID: n.ID, Title: n.Title, Hint: n.Hint, ProjectID: n.ProjectID, TicketID: n.TicketID, UpdatedAt: n.UpdatedAt,
		}})
	}
	return out, nil
}

// findTickets ranks the tickets full-text search finds, a ticket its key names first of all.
func (s *Service) findTickets(ctx context.Context, workspaceID, query string, words []string) ([]rankedHit, error) {
	keyed, err := s.resolveMentionKey(ctx, query, workspaceID)
	if err != nil {
		return nil, err
	}
	hits, err := s.cfg.Tickets.Search(ctx, workspaceID, query, findScan)
	if err != nil {
		return nil, fmt.Errorf("find tickets: %w", err)
	}
	seen := map[string]bool{}
	out := make([]rankedHit, 0, len(keyed)+len(hits))
	add := func(id string, keyed bool) {
		if seen[id] {
			return
		}
		seen[id] = true
		t, err := s.cfg.Tickets.GetByID(ctx, id)
		if err != nil || !s.canOpenTicket(ctx, t.ProjectID) {
			return
		}
		label, kind := s.status(ctx, t.Status)
		hit := Hit{Type: string(KindTicket), ID: t.ID, Title: t.Title, StatusLabel: label, StatusKind: kind, ProjectID: t.ProjectID, UpdatedAt: t.UpdatedAt}
		if prefix := s.projectPrefix(ctx, t.ProjectID); prefix != "" {
			hit.Hint = fmt.Sprintf("%s-%d", prefix, t.Number)
		}
		tier := keyTier
		if !keyed {
			tier = titleTier(t.Title, words)
		}
		hit.InBody = tier < 0
		out = append(out, rankedHit{hit: hit, tier: tier})
	}
	for _, k := range keyed {
		add(k.ID, true)
	}
	for _, h := range hits {
		add(h.ID, false)
	}
	return out, nil
}

func (s *Service) findDocs(ctx context.Context, actorID, workspaceID, query string, words []string) ([]rankedHit, error) {
	hits, err := s.cfg.Docs.Search(ctx, workspaceID, query, findScan)
	if err != nil {
		return nil, fmt.Errorf("find docs: %w", err)
	}
	out := make([]rankedHit, 0, len(hits))
	for _, h := range hits {
		if !s.canOpenDoc(ctx, actorID, h.ID) {
			continue
		}
		d, err := s.cfg.Docs.GetByID(ctx, h.ID)
		if err != nil {
			continue
		}
		tier := titleTier(d.Title, words)
		out = append(out, rankedHit{tier: tier, hit: Hit{Type: string(KindDoc), ID: d.ID, Title: d.Title, ProjectID: d.ProjectID, InBody: tier < 0, UpdatedAt: d.UpdatedAt}})
	}
	return out, nil
}

// cutRanked orders a kind's hits, title hits by tier then recency and body hits by recency, and keeps limit of each.
func cutRanked(ranked []rankedHit, limit int) ([]Hit, bool) {
	slices.SortStableFunc(ranked, func(a, b rankedHit) int {
		return cmp.Or(
			cmp.Compare(b.tier, a.tier),
			b.hit.UpdatedAt.Compare(a.hit.UpdatedAt),
			cmp.Compare(a.hit.Title, b.hit.Title),
			cmp.Compare(a.hit.ID, b.hit.ID),
		)
	})
	var title, body []Hit
	for _, r := range ranked {
		if r.hit.InBody {
			body = append(body, r.hit)
			continue
		}
		title = append(title, r.hit)
	}
	more := len(title) > limit || len(body) > limit
	return append(title[:min(len(title), limit)], body[:min(len(body), limit)]...), more
}

// searchWords splits text the way the full-text index does: lowercase runs of letters and digits.
func searchWords(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// titleTier scores how a title holds the query's words: 3 equal, 2 each word whole, 1 each word as a prefix, -1 not.
func titleTier(title string, query []string) int {
	words := searchWords(title)
	if slices.Equal(words, query) {
		return 3
	}
	tier := 2
	for _, q := range query {
		if slices.Contains(words, q) {
			continue
		}
		if !slices.ContainsFunc(words, func(w string) bool { return strings.HasPrefix(w, q) }) {
			return -1
		}
		tier = 1
	}
	return tier
}

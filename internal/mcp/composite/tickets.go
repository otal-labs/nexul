package composite

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/codereview"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// TicketTools are the ticket tools that show project keys and names, or reach into the workspace and code review domains.
func TicketTools(t *tickets.Service, w *workspace.Service, r *codereview.Service) []mcptool.Tool {
	return []mcptool.Tool{ticketListTool(t, w), ticketGetTool(t, w, r), ticketCreateTool(t, w), ticketUpdateTool(t, w)}
}

// ponytail: a query reads the top 200 matches and pages over them in memory; raise it if agents need deeper search.
const searchScan = 200

const (
	idHint     = "project_get lists the project's status, ticket type, and category ids"
	ticketHint = "ticket_list finds tickets by text or project"
)

// ticketResult is a ticket as an agent reads it: ids with their human names and the key beside them.
type ticketResult struct {
	ID         string           `json:"id"`
	Key        string           `json:"key,omitempty"`
	Title      string           `json:"title"`
	Body       string           `json:"body,omitempty"`
	ProjectID  string           `json:"project_id"`
	StatusID   string           `json:"status_id"`
	Status     string           `json:"status,omitempty"`
	TypeID     string           `json:"type_id,omitempty"`
	Type       string           `json:"type,omitempty"`
	CategoryID string           `json:"category_id,omitempty"`
	Category   string           `json:"category,omitempty"`
	Developer  string           `json:"developer,omitempty"`
	Tester     string           `json:"tester,omitempty"`
	Reporter   tickets.Reporter `json:"reporter"`
	Labels     []string         `json:"labels"`
	DocID      string           `json:"doc_id,omitempty"`
	WaitingOn  []string         `json:"waiting_on,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
	FinishedAt *time.Time       `json:"finished_at,omitempty"`
}

// names caches each project's prefix and column, type, and category names for one call.
type names struct {
	w        *workspace.Service
	projects map[string]*projectNames
}

type projectNames struct {
	prefix                      string
	statuses, types, categories map[string]string
}

func newNames(w *workspace.Service) *names {
	return &names{w: w, projects: map[string]*projectNames{}}
}

func (n *names) of(ctx context.Context, projectID string) (*projectNames, error) {
	if pn, ok := n.projects[projectID]; ok {
		return pn, nil
	}
	p, err := n.w.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	pn := &projectNames{prefix: p.Prefix, statuses: map[string]string{}, types: map[string]string{}, categories: map[string]string{}}
	statuses, err := n.w.ListStatusesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, s := range statuses {
		pn.statuses[s.ID] = s.Name
	}
	types, err := n.w.ListTicketTypesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, tt := range types {
		pn.types[tt.ID] = tt.Name
	}
	cats, err := n.w.ListCategoriesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, c := range cats {
		pn.categories[c.ID] = c.Name
	}
	n.projects[projectID] = pn
	return pn, nil
}

func (n *names) ticket(ctx context.Context, t *tickets.Ticket) (ticketResult, error) {
	pn, err := n.of(ctx, t.ProjectID)
	if err != nil {
		return ticketResult{}, err
	}
	labels := t.Labels
	if labels == nil {
		labels = []string{}
	}
	return ticketResult{
		ID: t.ID, Key: ticketKey(pn.prefix, t.Number), Title: t.Title, Body: t.Body, ProjectID: t.ProjectID,
		StatusID: string(t.Status), Status: pn.statuses[string(t.Status)],
		TypeID: t.TypeID, Type: pn.types[t.TypeID], CategoryID: t.CategoryID, Category: pn.categories[t.CategoryID],
		Developer: t.Developer, Tester: t.Tester, Reporter: t.Reporter, Labels: labels, DocID: t.DocID,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, FinishedAt: t.FinishedAt,
	}, nil
}

// ticketKey renders PREFIX-NUMBER (ADR 0004); a project without a prefix yet has no keys.
func ticketKey(prefix string, number int) string {
	if prefix == "" || number < 1 {
		return ""
	}
	return prefix + "-" + strconv.Itoa(number)
}

type ticketListIn struct {
	ProjectID   string `json:"project_id,omitempty" jsonschema:"Only tickets in this project, by its id from project_list."`
	DocID       string `json:"doc_id,omitempty" jsonschema:"Only tickets filed from this doc, by its id."`
	Query       string `json:"query,omitempty" jsonschema:"Full-text search over titles and bodies, for example login timeout. Results are ordered by relevance."`
	BlockedOnly bool   `json:"blocked_only,omitzero" jsonschema:"Only tickets still waiting on a blocker that has not reached a done-stage column."`
	mcptool.PageArgs
}

func ticketListTool(t *tickets.Service, w *workspace.Service) mcptool.Tool {
	return mcptool.New("ticket_list", "List tickets",
		"Lists tickets without their bodies: each with its id, key (such as REF-102), title, status column, type, "+
			"category, people, labels, and, when it is blocked, the keys of the blockers it still waits on (waiting_on). "+
			"Filter by project, by source doc, by a full-text query, or to blocked tickets only; filters combine. "+
			"Use ticket_get for one ticket's body, links, pull requests, and reviews. Without a query tickets come "+
			"oldest first; a query returns at most its 200 best matches. Returns at most 100 tickets per page.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in ticketListIn) (any, error) {
			found, err := findTickets(ctx, t, in)
			if err != nil {
				return nil, err
			}
			blockers, err := t.UnclearedBlockers(ctx)
			if err != nil {
				return nil, err
			}
			kept := make([]*tickets.Ticket, 0, len(found))
			for _, tk := range found {
				if in.keeps(tk, blockers) {
					kept = append(kept, tk)
				}
			}
			return shapePage(ctx, newNames(w), mcptool.Paginate(kept, in.PageArgs), blockers)
		})
}

func findTickets(ctx context.Context, t *tickets.Service, in ticketListIn) ([]*tickets.Ticket, error) {
	if strings.TrimSpace(in.Query) != "" {
		return searchTickets(ctx, t, in.Query)
	}
	if in.DocID != "" {
		return t.ListByDoc(ctx, in.DocID)
	}
	if in.ProjectID != "" {
		return t.ListByProject(ctx, in.ProjectID)
	}
	return t.List(ctx)
}

func searchTickets(ctx context.Context, t *tickets.Service, query string) ([]*tickets.Ticket, error) {
	hits, err := t.Search(ctx, query, searchScan)
	if err != nil {
		return nil, err
	}
	out := make([]*tickets.Ticket, 0, len(hits))
	for _, h := range hits {
		tk, err := t.Get(ctx, h.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, tk)
	}
	return out, nil
}

func (in ticketListIn) keeps(t *tickets.Ticket, blockers map[string][]tickets.LinkedTicket) bool {
	if in.ProjectID != "" && t.ProjectID != in.ProjectID {
		return false
	}
	if in.DocID != "" && t.DocID != in.DocID {
		return false
	}
	return !in.BlockedOnly || len(blockers[t.ID]) > 0
}

func shapePage(ctx context.Context, n *names, page mcptool.Page[*tickets.Ticket], blockers map[string][]tickets.LinkedTicket) (mcptool.Page[ticketResult], error) {
	out := mcptool.Page[ticketResult]{Items: make([]ticketResult, 0, len(page.Items)), Total: page.Total, HasMore: page.HasMore, NextOffset: page.NextOffset}
	for _, tk := range page.Items {
		r, err := n.ticket(ctx, tk)
		if err != nil {
			return out, err
		}
		r.Body = ""
		for _, b := range blockers[tk.ID] {
			r.WaitingOn = append(r.WaitingOn, linkedKey(b))
		}
		out.Items = append(out.Items, r)
	}
	return out, nil
}

// linkedKey names a linked ticket by its key, or by its id when its project has no prefix.
func linkedKey(l tickets.LinkedTicket) string {
	if k := ticketKey(l.Prefix, l.Number); k != "" {
		return k
	}
	return l.ID
}

type ticketGetIn struct {
	ID string `json:"id" jsonschema:"The ticket's id (a UUID) or its key, for example REF-102."`
}

type ticketDetail struct {
	ticketResult
	PullRequests  []pullRequestResult  `json:"pull_requests"`
	Branches      []tickets.BranchLink `json:"branches"`
	FoundIn       *linkedResult        `json:"found_in,omitempty"`
	OriginUnknown bool                 `json:"origin_unknown,omitempty"`
	BugsFound     []linkedResult       `json:"bugs_found"`
	BlockedBy     []linkedResult       `json:"blocked_by"`
	Blocks        []linkedResult       `json:"blocks"`
	Blocked       bool                 `json:"blocked"`
	Reviews       []reviewResult       `json:"reviews"`
	TestTarget    tickets.TestTarget   `json:"test_target"`
}

type pullRequestResult struct {
	Owner  string          `json:"owner"`
	Repo   string          `json:"repo"`
	Number int             `json:"number"`
	Title  string          `json:"title,omitempty"`
	State  tickets.PRState `json:"state"`
}

type linkedResult struct {
	ID       string `json:"id"`
	Key      string `json:"key,omitempty"`
	Title    string `json:"title"`
	StatusID string `json:"status_id"`
	Done     bool   `json:"done"`
}

type reviewResult struct {
	ID       string            `json:"id"`
	Repo     string            `json:"repo"`
	PRNumber int               `json:"pr_number"`
	Status   codereview.Status `json:"status"`
	Reviewer string            `json:"reviewer,omitempty"`
}

func ticketGetTool(t *tickets.Service, w *workspace.Service, r *codereview.Service) mcptool.Tool {
	return mcptool.New("ticket_get", "Get ticket",
		"Returns one ticket with its markdown body, status column, type, category, people, and labels, plus its "+
			"linked pull requests and branches, the ticket it was found in and the bugs found in it, what blocks it "+
			"and what it blocks (each with done read from its column's stage), the code reviews of its pull requests, "+
			"and its test_target. Use it before changing a ticket with ticket_update, and before testing one: "+
			"test_target is its branch's preview (kind preview) or a shared test environment (kind shared), never "+
			"production, and an empty url means nowhere is safe to test. To find tickets by text or project, use "+
			"ticket_list.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in ticketGetIn) (any, error) {
			tk, err := t.Resolve(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			base, err := newNames(w).ticket(ctx, tk)
			if err != nil {
				return nil, err
			}
			d := ticketDetail{ticketResult: base}
			if err := d.addLinks(ctx, t, tk.ID); err != nil {
				return nil, err
			}
			if err := d.addReviews(ctx, r, tk.ID); err != nil {
				return nil, err
			}
			if d.TestTarget, err = t.TestTarget(ctx, tk.ID); err != nil {
				return nil, err
			}
			return d, nil
		})
}

func (d *ticketDetail) addLinks(ctx context.Context, t *tickets.Service, id string) error {
	prs, branches, err := t.ListLinks(ctx, id)
	if err != nil {
		return err
	}
	d.PullRequests = make([]pullRequestResult, 0, len(prs))
	for _, p := range prs {
		d.PullRequests = append(d.PullRequests, pullRequestResult{Owner: p.Owner, Repo: p.Repo, Number: p.Number, Title: p.Title, State: p.State})
	}
	d.Branches = branches
	if d.Branches == nil {
		d.Branches = []tickets.BranchLink{}
	}
	set, err := t.Links(ctx, id)
	if err != nil {
		return err
	}
	if set.FoundIn != nil {
		found := linked(*set.FoundIn)
		d.FoundIn = &found
	}
	d.OriginUnknown, d.Blocked = set.OriginUnknown, set.Blocked
	d.BugsFound, d.BlockedBy, d.Blocks = linkedAll(set.BugsFound), linkedAll(set.BlockedBy), linkedAll(set.Blocks)
	return nil
}

func (d *ticketDetail) addReviews(ctx context.Context, r *codereview.Service, id string) error {
	reviews, err := r.ListByTicket(ctx, id)
	if err != nil {
		return err
	}
	d.Reviews = make([]reviewResult, 0, len(reviews))
	for _, rv := range reviews {
		d.Reviews = append(d.Reviews, reviewResult{ID: rv.ID, Repo: rv.Repo, PRNumber: rv.PRNumber, Status: rv.Status, Reviewer: rv.Reviewer})
	}
	return nil
}

func linked(l tickets.LinkedTicket) linkedResult {
	return linkedResult{ID: l.ID, Key: ticketKey(l.Prefix, l.Number), Title: l.Title, StatusID: string(l.Status), Done: l.Done}
}

func linkedAll(ls []tickets.LinkedTicket) []linkedResult {
	out := make([]linkedResult, 0, len(ls))
	for _, l := range ls {
		out = append(out, linked(l))
	}
	return out
}

type ticketCreateIn struct {
	ProjectID     string `json:"project_id" jsonschema:"The project's id (a UUID), from project_list."`
	Title         string `json:"title" jsonschema:"A short summary, for example Login times out on slow networks."`
	Body          string `json:"body,omitempty" jsonschema:"The body in markdown. Fill the type's body_template from project_get; omitted with a type_id, the ticket starts from that template."`
	TypeID        string `json:"type_id,omitempty" jsonschema:"The ticket type's id, from project_get."`
	CategoryID    string `json:"category_id,omitempty" jsonschema:"The category's id, from project_get. Omit to leave it uncategorized."`
	Developer     string `json:"developer,omitempty" jsonschema:"The member login who builds it, for example onik97."`
	Tester        string `json:"tester,omitempty" jsonschema:"The member login who tests it in its testing stage, for example lena."`
	DocID         string `json:"doc_id,omitempty" jsonschema:"The id of the doc this ticket is derived from."`
	OriginID      string `json:"origin_id,omitempty" jsonschema:"For a bug: the id or key of the ticket it was found in, for example REF-98."`
	OriginUnknown bool   `json:"origin_unknown,omitzero" jsonschema:"For a bug: true when nobody knows which ticket it was found in."`
}

func ticketCreateTool(t *tickets.Service, w *workspace.Service) mcptool.Tool {
	return mcptool.New("ticket_create", "Create ticket",
		"Creates a ticket in a project and returns it with its key (such as REF-102). The reporter is recorded as "+
			"Nexul on behalf of you; developer and tester are member logins. project_get lists the project's ticket "+
			"types with their body_template: fill that template as the body, or omit the body with a type_id to start "+
			"from it. A bug (the type named bug) needs origin_id, the ticket it was found in, or origin_unknown true "+
			"when nobody knows; never guess an origin. Set its status column, labels, and blockers afterwards with "+
			"ticket_update.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in ticketCreateIn) (any, error) {
			originID, err := originOf(ctx, t, in.OriginID)
			if err != nil {
				return nil, err
			}
			created, err := t.Create(ctx, in.ProjectID, in.Title, in.Body, in.DocID, in.Developer, tickets.CreateOptions{
				CategoryID: in.CategoryID, TypeID: in.TypeID, Tester: in.Tester, ViaMCP: true,
				OriginID: originID, OriginUnknown: in.OriginUnknown,
			})
			if err != nil {
				return nil, err
			}
			return newNames(w).ticket(ctx, created)
		})
}

// originOf turns an origin key into its id; a missing origin is bad input to the new ticket, not a missing ticket.
func originOf(ctx context.Context, t *tickets.Service, idOrKey string) (string, error) {
	if strings.TrimSpace(idOrKey) == "" {
		return "", nil
	}
	origin, err := t.Resolve(ctx, idOrKey)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", fmt.Errorf("%w: origin_id %s is not a ticket; %s", apperrs.ErrInvalid, idOrKey, ticketHint)
	}
	if err != nil {
		return "", err
	}
	return origin.ID, nil
}

type ticketUpdateIn struct {
	ID               string        `json:"id" jsonschema:"The ticket's id (a UUID) or its key, for example REF-102."`
	Title            *string       `json:"title,omitempty" jsonschema:"A new title; it cannot be empty."`
	Body             *string       `json:"body,omitempty" jsonschema:"A new body in markdown; an empty string clears it."`
	ProjectID        *string       `json:"project_id,omitempty" jsonschema:"Move the ticket to this project, by its id from project_list; set status_id to one of its columns too."`
	StatusID         *string       `json:"status_id,omitempty" jsonschema:"Move the ticket to this status column, by its id from project_get. It lands at the bottom of the column."`
	Position         *int          `json:"position,omitempty" jsonschema:"The ticket's place in its column, 0 for the top."`
	TypeID           *string       `json:"type_id,omitempty" jsonschema:"The ticket type's id, from project_get."`
	CategoryID       *string       `json:"category_id,omitempty" jsonschema:"The category's id, from project_get; an empty string uncategorizes the ticket."`
	Developer        *string       `json:"developer,omitempty" jsonschema:"The member login who builds it; an empty string clears it."`
	Tester           *string       `json:"tester,omitempty" jsonschema:"The member login who tests it; an empty string clears it."`
	AddLabels        []string      `json:"add_labels,omitempty" jsonschema:"Labels to attach, for example urgent."`
	RemoveLabels     []string      `json:"remove_labels,omitempty" jsonschema:"Labels to detach."`
	FoundInID        string        `json:"found_in_id,omitempty" jsonschema:"The id or key of the ticket this bug was found in; replaces any earlier found-in."`
	FoundInUnknown   bool          `json:"found_in_unknown,omitzero" jsonschema:"True marks that nobody knows which ticket this bug was found in."`
	ClearFoundIn     bool          `json:"clear_found_in,omitzero" jsonschema:"True removes the found-in link or the unknown marker."`
	AddBlockerIDs    []string      `json:"add_blocker_ids,omitempty" jsonschema:"Ids or keys of tickets that must reach a done-stage column first. A link that would form a cycle is refused."`
	RemoveBlockerIDs []string      `json:"remove_blocker_ids,omitempty" jsonschema:"Ids or keys of blockers to remove."`
	LinkPR           *prLinkIn     `json:"link_pr,omitempty" jsonschema:"A pull request to link to the ticket."`
	LinkBranch       *branchLinkIn `json:"link_branch,omitempty" jsonschema:"A branch to link to the ticket."`
}

type prLinkIn struct {
	Owner  string `json:"owner" jsonschema:"The repository owner, for example otal-labs."`
	Repo   string `json:"repo" jsonschema:"The repository name, for example nexul."`
	Number int    `json:"number" jsonschema:"The pull request number, for example 42."`
	Title  string `json:"title,omitempty" jsonschema:"The pull request's title."`
	SHA    string `json:"sha,omitempty" jsonschema:"The pull request's head commit."`
}

type branchLinkIn struct {
	Owner  string `json:"owner" jsonschema:"The repository owner, for example otal-labs."`
	Repo   string `json:"repo" jsonschema:"The repository name, for example nexul."`
	Branch string `json:"branch" jsonschema:"The branch name, for example fix/login-timeout."`
}

type ticketUpdateResult struct {
	Applied []string     `json:"applied"`
	Ticket  ticketResult `json:"ticket"`
}

func ticketUpdateTool(t *tickets.Service, w *workspace.Service) mcptool.Tool {
	return mcptool.New("ticket_update", "Update ticket",
		"Changes a ticket: its title and body, project, status column and place in it, type, category, developer "+
			"and tester, labels, found-in link, blockers, and linked pull request or branch. Only the fields you send "+
			"change; an omitted field keeps its value. The changes apply in the order the fields are listed here and "+
			"stop at the first failure, whose message names the field and the ones already applied. To record a test "+
			"result, which also moves the ticket, use ticket_test_report; to remove a ticket, ticket_delete. Returns "+
			"the updated ticket and the list of fields applied.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in ticketUpdateIn) (any, error) {
			if err := in.checkFoundIn(); err != nil {
				return nil, err
			}
			tk, err := t.Resolve(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			u := ticketUpdate{ctx: ctx, t: t, w: w, id: tk.ID}
			applied, err := runSteps(u.steps(in))
			if err != nil {
				return nil, err
			}
			fresh, err := t.Get(ctx, tk.ID)
			if err != nil {
				return nil, err
			}
			shaped, err := newNames(w).ticket(ctx, fresh)
			if err != nil {
				return nil, err
			}
			return ticketUpdateResult{Applied: applied, Ticket: shaped}, nil
		})
}

func (in ticketUpdateIn) checkFoundIn() error {
	given := 0
	for _, set := range []bool{in.FoundInID != "", in.FoundInUnknown, in.ClearFoundIn} {
		if set {
			given++
		}
	}
	if given > 1 {
		return fmt.Errorf("%w: give only one of found_in_id, found_in_unknown, and clear_found_in", apperrs.ErrInvalid)
	}
	return nil
}

// ticketUpdate builds a ticket_update's steps; each closure reads its field only when the field was sent.
type ticketUpdate struct {
	ctx context.Context
	t   *tickets.Service
	w   *workspace.Service
	id  string
}

func (u ticketUpdate) steps(in ticketUpdateIn) []step {
	var steps []step
	if in.Title != nil || in.Body != nil {
		steps = append(steps, step{field: pairLabel("title", in.Title != nil, "body", in.Body != nil), run: func() error { return u.content(in.Title, in.Body) }})
	}
	fields := []struct {
		sent bool
		step step
	}{
		{in.ProjectID != nil, step{"project_id", "project_list lists the projects", func() error { return u.w.MoveTicket(u.ctx, u.id, *in.ProjectID) }}},
		{in.StatusID != nil, step{"status_id", idHint, func() error { return discard(u.t.UpdateStatus(u.ctx, u.id, tickets.Status(*in.StatusID))) }}},
		{in.Position != nil, step{"position", "", func() error { return discard(u.t.SetPosition(u.ctx, u.id, *in.Position)) }}},
		{in.TypeID != nil, step{"type_id", idHint, func() error { return discard(u.t.SetType(u.ctx, u.id, *in.TypeID)) }}},
		{in.CategoryID != nil, step{"category_id", idHint, func() error { return u.w.MoveTicketToCategory(u.ctx, u.id, *in.CategoryID) }}},
		{in.Developer != nil, step{"developer", "", func() error { return discard(u.t.SetPerson(u.ctx, u.id, tickets.RoleDeveloper, *in.Developer)) }}},
		{in.Tester != nil, step{"tester", "", func() error { return discard(u.t.SetPerson(u.ctx, u.id, tickets.RoleTester, *in.Tester)) }}},
	}
	for _, f := range fields {
		if f.sent {
			steps = append(steps, f.step)
		}
	}
	steps = append(steps, u.labelSteps(in)...)
	steps = append(steps, u.linkSteps(in)...)
	return steps
}

// content overlays the sent title and body on the stored ones, since the use-case replaces both.
func (u ticketUpdate) content(title, body *string) error {
	current, err := u.t.Get(u.ctx, u.id)
	if err != nil {
		return err
	}
	return discard(u.t.UpdateTicket(u.ctx, u.id, valueOr(title, current.Title), valueOr(body, current.Body)))
}

func (u ticketUpdate) labelSteps(in ticketUpdateIn) []step {
	var steps []step
	for i, label := range in.AddLabels {
		steps = append(steps, step{fmt.Sprintf("add_labels[%d]", i), "", func() error { return discard(u.t.AddLabel(u.ctx, u.id, label)) }})
	}
	for i, label := range in.RemoveLabels {
		steps = append(steps, step{fmt.Sprintf("remove_labels[%d]", i), "", func() error { return discard(u.t.RemoveLabel(u.ctx, u.id, label)) }})
	}
	return steps
}

func (u ticketUpdate) linkSteps(in ticketUpdateIn) []step {
	var steps []step
	if in.FoundInID != "" || in.FoundInUnknown {
		steps = append(steps, step{"found_in", ticketHint, func() error { return u.foundIn(in.FoundInID, in.FoundInUnknown) }})
	}
	if in.ClearFoundIn {
		steps = append(steps, step{"clear_found_in", "", func() error { return discard(u.t.RemoveFoundIn(u.ctx, u.id)) }})
	}
	for i, ref := range in.AddBlockerIDs {
		steps = append(steps, step{fmt.Sprintf("add_blocker_ids[%d]", i), ticketHint, func() error { return u.blocker(ref, u.t.AddBlocker) }})
	}
	for i, ref := range in.RemoveBlockerIDs {
		steps = append(steps, step{fmt.Sprintf("remove_blocker_ids[%d]", i), ticketHint, func() error { return u.blocker(ref, u.t.RemoveBlocker) }})
	}
	if pr := in.LinkPR; pr != nil {
		steps = append(steps, step{"link_pr", "", func() error {
			return u.t.LinkPR(u.ctx, u.id, tickets.PRRef{Owner: pr.Owner, Repo: pr.Repo, Number: pr.Number, Title: pr.Title, SHA: pr.SHA})
		}})
	}
	if b := in.LinkBranch; b != nil {
		steps = append(steps, step{"link_branch", "", func() error { return u.t.LinkBranch(u.ctx, u.id, b.Owner, b.Repo, b.Branch) }})
	}
	return steps
}

func (u ticketUpdate) foundIn(originRef string, unknown bool) error {
	originID := ""
	if originRef != "" {
		origin, err := u.t.Resolve(u.ctx, originRef)
		if err != nil {
			return err
		}
		originID = origin.ID
	}
	return discard(u.t.SetFoundIn(u.ctx, u.id, originID, unknown))
}

func (u ticketUpdate) blocker(ref string, apply func(ctx context.Context, id, blockerID string) (*tickets.LinkSet, error)) error {
	blocker, err := u.t.Resolve(u.ctx, ref)
	if err != nil {
		return err
	}
	return discard(apply(u.ctx, u.id, blocker.ID))
}

// discard keeps a use-case's error and drops the value an update step does not need.
func discard[T any](_ T, err error) error {
	return err
}

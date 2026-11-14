package composite

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/textedit"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// TicketTools are the ticket tools that show project keys and names, or reach into the workspace and code review domains.
func TicketTools(t *tickets.Service, w *workspace.Service, r *codereview.Service) []mcptool.Tool {
	return []mcptool.Tool{ticketListTool(t, w), ticketGetTool(t, w, r), ticketCreateTool(t, w), ticketUpdateTool(t, w)}
}

const (
	idHint     = "project_get lists the project's status, ticket type, and category ids"
	ticketHint = "ticket_list finds tickets by text or project"
	createHint = "project_list lists the projects; project_get lists a project's ticket types and categories"
)

// meToLogin swaps me in a people field for the caller's login, so an agent need not look its own login up first.
func meToLogin(ctx context.Context, t *tickets.Service, logins ...*string) error {
	for _, login := range logins {
		if login == nil || *login != tickets.Me {
			continue
		}
		me, err := t.CallerLogin(ctx)
		if err != nil {
			return err
		}
		*login = me
	}
	return nil
}

// resolveTicket names the tool that finds tickets when the one asked for is missing.
func resolveTicket(ctx context.Context, t *tickets.Service, workspace, idOrKey string) (*tickets.Ticket, error) {
	tk, err := t.Resolve(ctx, workspace, idOrKey)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("%w; %s", err, ticketHint)
	}
	return tk, err
}

// ticketResult is a ticket as an agent reads it: ids with their human names and the key beside them.
type ticketResult struct {
	ID          string           `json:"id"`
	Key         string           `json:"key,omitempty"`
	Title       string           `json:"title"`
	Body        string           `json:"body,omitempty"`
	ProjectID   string           `json:"project_id"`
	StatusID    string           `json:"status_id"`
	Status      string           `json:"status,omitempty"`
	TypeID      string           `json:"type_id,omitempty"`
	Type        string           `json:"type,omitempty"`
	CategoryID  string           `json:"category_id,omitempty"`
	Category    string           `json:"category,omitempty"`
	Developer   string           `json:"developer,omitempty"`
	DeveloperID string           `json:"developer_id,omitempty"`
	Tester      string           `json:"tester,omitempty"`
	TesterID    string           `json:"tester_id,omitempty"`
	DueOn       string           `json:"due_on"`
	Reporter    tickets.Reporter `json:"reporter"`
	Labels      []string         `json:"labels"`
	DocID       string           `json:"doc_id,omitempty"`
	WaitingOn   []string         `json:"waiting_on,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	FinishedAt  *time.Time       `json:"finished_at,omitempty"`
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
	body, err := richtext.ToMarkdown(t.Body)
	if err != nil {
		body = t.Body
	}
	return ticketResult{
		ID: t.ID, Key: ticketKey(pn.prefix, t.Number), Title: t.Title, Body: body, ProjectID: t.ProjectID,
		StatusID: string(t.Status), Status: pn.statuses[string(t.Status)],
		TypeID: t.TypeID, Type: pn.types[t.TypeID], CategoryID: t.CategoryID, Category: pn.categories[t.CategoryID],
		Developer: t.Developer, DeveloperID: t.DeveloperID, Tester: t.Tester, TesterID: t.TesterID, DueOn: t.DueOn, Reporter: t.Reporter, Labels: t.Labels, DocID: t.DocID,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, FinishedAt: t.FinishedAt,
	}, nil
}

// key is the ticket's key, or its id when its project has no prefix.
func (n *names) key(ctx context.Context, t *tickets.Ticket) (string, error) {
	pn, err := n.of(ctx, t.ProjectID)
	if err != nil {
		return "", err
	}
	return cmp.Or(ticketKey(pn.prefix, t.Number), t.ID), nil
}

// ticketKey renders PREFIX-NUMBER (ADR 0004); a project without a prefix yet has no keys.
func ticketKey(prefix string, number int) string {
	if prefix == "" || number < 1 {
		return ""
	}
	return prefix + "-" + strconv.Itoa(number)
}

type ticketListIn struct {
	ProjectID   string  `json:"project_id,omitempty" jsonschema:"Only tickets in this project, from project_list."`
	DocID       string  `json:"doc_id,omitempty" jsonschema:"Only tickets filed from this doc, from doc_list."`
	Query       string  `json:"query,omitempty" jsonschema:"Full-text search over titles, bodies, and thread notes, such as login timeout; note-only matches last."`
	BlockedOnly bool    `json:"blocked_only,omitzero" jsonschema:"Only tickets waiting on a blocker."`
	DueFrom     string  `json:"due_from,omitempty" jsonschema:"Only tickets due on or after this date (YYYY-MM-DD, such as 2026-10-01); either date leaves out tickets with none."`
	DueTo       string  `json:"due_to,omitempty" jsonschema:"Only tickets due on or before this date (YYYY-MM-DD)."`
	CategoryID  *string `json:"category_id,omitempty" jsonschema:"Only this category's tickets, from project_get; an empty string for uncategorized ones."`
	StatusID    string  `json:"status_id,omitempty" jsonschema:"Only this status column's tickets, from project_get."`
	Developer   *string `json:"developer,omitempty" jsonschema:"Only this developer's tickets, by login or me; empty for unassigned."`
	mcptool.PageArgs
}

func ticketListTool(t *tickets.Service, w *workspace.Service) mcptool.Tool {
	return mcptool.New("ticket_list", "List tickets",
		"Lists tickets without their bodies: each with its id, key, title, status column, type, "+
			"category, people, due_on, labels, and, when blocked, the keys of the blockers it still waits on (waiting_on). "+
			"Filters combine: project, category, column, developer, source doc, full-text query, due date range (inclusive), or blocked tickets only. "+
			"Use ticket_get for one ticket's body, links, pull requests, and reviews. Tickets come oldest first, "+
			"by due date when a due date filter is given, and by relevance with a query.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in ticketListIn) (any, error) {
			if err := meToLogin(ctx, t, in.Developer); err != nil {
				return nil, err
			}
			if in.ProjectID != "" {
				// A mistyped project would otherwise read as a project with no tickets.
				if _, err := getProject(ctx, w, in.ProjectID); err != nil {
					return nil, err
				}
			}
			f := tickets.TicketFilter{
				ProjectID: in.ProjectID, DocID: in.DocID, Query: in.Query, BlockedOnly: in.BlockedOnly, DueFrom: in.DueFrom, DueTo: in.DueTo,
				CategoryID: in.CategoryID, StatusID: in.StatusID, Developer: in.Developer,
			}
			found, total, err := t.PageTickets(ctx, f, in.Window())
			if err != nil {
				return nil, err
			}
			blockers, err := t.UnclearedBlockersOf(ctx, found)
			if err != nil {
				return nil, err
			}
			return shapePage(ctx, newNames(w), mcptool.PageOf(found, total, in.Window()), blockers)
		})
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
	ID        string `json:"id" jsonschema:"The ticket's id or key, such as REF-102."`
	Workspace string `json:"workspace,omitempty" jsonschema:"Workspace id or slug, when the key is in more than one of yours."`
	Version   int    `json:"version,omitzero" jsonschema:"Read this version, not the current one."`
}

// recentChanges caps the field changes ticket_get lists, newest first.
const recentChanges = 20

type ticketVersionResult struct {
	Version   int       `json:"version"`
	EditorIDs []string  `json:"editor_ids"`
	Via       string    `json:"via,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ticketChangeResult struct {
	Field     string        `json:"field"`
	From      string        `json:"from"`
	To        string        `json:"to"`
	Actor     tickets.Actor `json:"actor"`
	CreatedAt time.Time     `json:"created_at"`
}

type ticketDetail struct {
	ticketResult
	// CurrentVersion is the newest description version, set when version asked for an older body.
	CurrentVersion int                   `json:"current_version,omitempty"`
	Versions       []ticketVersionResult `json:"versions"`
	Changes        []ticketChangeResult  `json:"changes"`
	PullRequests   []pullRequestResult   `json:"pull_requests"`
	Branches       []tickets.BranchLink  `json:"branches"`
	FoundIn        *linkedResult         `json:"found_in,omitempty"`
	OriginUnknown  bool                  `json:"origin_unknown,omitempty"`
	BugsFound      []linkedResult        `json:"bugs_found"`
	BlockedBy      []linkedResult        `json:"blocked_by"`
	Blocks         []linkedResult        `json:"blocks"`
	Blocked        bool                  `json:"blocked"`
	Reviews        []reviewResult        `json:"reviews"`
	TestTarget     tickets.TestTarget    `json:"test_target"`
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
	Finished bool   `json:"finished"`
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
		"Returns one ticket with its markdown body and fields, plus its "+
			"linked pull requests and branches, the ticket it was found in and the bugs found in it, what blocks it "+
			"and what it blocks (each with done read from its column's stage), the code reviews of its pull requests, "+
			"its versions, latest 20 changes, and test_target. Use it before changing a ticket with ticket_update, and before testing one: "+
			"test_target is its branch's preview (kind preview) or a shared test environment (kind shared), never "+
			"production, and an empty url means nowhere is safe to test. To find tickets by text or project, use "+
			"ticket_list.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in ticketGetIn) (any, error) {
			tk, err := resolveTicket(ctx, t, in.Workspace, in.ID)
			if err != nil {
				return nil, err
			}
			base, err := newNames(w).ticket(ctx, tk)
			if err != nil {
				return nil, err
			}
			d := ticketDetail{ticketResult: base}
			if err := d.addHistory(ctx, t, tk.ID, in.Version); err != nil {
				return nil, err
			}
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

// addHistory lists the kept description versions and the recent field changes, and swaps in an older body when asked.
func (d *ticketDetail) addHistory(ctx context.Context, t *tickets.Service, id string, version int) error {
	h, err := t.History(ctx, id)
	if err != nil {
		return err
	}
	d.Versions = make([]ticketVersionResult, 0, len(h.Versions))
	for _, v := range h.Versions {
		d.Versions = append(d.Versions, ticketVersionResult{Version: v.Version, EditorIDs: v.EditorIDs, Via: v.Via, CreatedAt: v.CreatedAt})
	}
	d.Changes = make([]ticketChangeResult, 0, min(len(h.Changes), recentChanges))
	for _, c := range h.Changes[:min(len(h.Changes), recentChanges)] {
		d.Changes = append(d.Changes, ticketChangeResult{Field: c.Field, From: c.From, To: c.To, Actor: c.Actor, CreatedAt: c.CreatedAt})
	}
	if version == 0 {
		return nil
	}
	v, err := t.GetVersion(ctx, id, version)
	if err != nil {
		return err
	}
	body, err := richtext.ToMarkdown(v.Body)
	if err != nil {
		body = v.Body
	}
	d.Body = body
	if len(h.Versions) > 0 {
		d.CurrentVersion = h.Versions[0].Version
	}
	return nil
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
	return linkedResult{ID: l.ID, Key: ticketKey(l.Prefix, l.Number), Title: l.Title, StatusID: string(l.Status), Done: l.Done, Finished: l.Finished}
}

func linkedAll(ls []tickets.LinkedTicket) []linkedResult {
	out := make([]linkedResult, 0, len(ls))
	for _, l := range ls {
		out = append(out, linked(l))
	}
	return out
}

type ticketCreateIn struct {
	ProjectID     string        `json:"project_id" jsonschema:"The project's id, from project_list."`
	Title         string        `json:"title,omitempty" jsonschema:"A short summary."`
	Tickets       []newTicketIn `json:"tickets,omitempty" jsonschema:"Instead of title and body, up to 100 tickets sharing every other field."`
	Body          string        `json:"body,omitempty" jsonschema:"The body in markdown. Fill the type's body_template from project_get; omit it with a type_id to start from that template."`
	TypeID        string        `json:"type_id,omitempty" jsonschema:"The ticket type's id, from project_get. Omit for the project's first type."`
	CategoryID    string        `json:"category_id,omitempty" jsonschema:"The category's id, from project_get."`
	Developer     string        `json:"developer,omitempty" jsonschema:"The member login who builds it, or me."`
	Tester        string        `json:"tester,omitempty" jsonschema:"The member login who tests it, or me."`
	DocID         string        `json:"doc_id,omitempty" jsonschema:"The doc this ticket is derived from, from doc_list."`
	OriginID      string        `json:"origin_id,omitempty" jsonschema:"For a bug: the id or key of the ticket it was found in, such as REF-98."`
	OriginUnknown bool          `json:"origin_unknown,omitzero" jsonschema:"For a bug: true when nobody knows which ticket it was found in."`
}

type newTicketIn struct {
	Title string `json:"title" jsonschema:"A short summary."`
	Body  string `json:"body,omitempty" jsonschema:"The body in markdown."`
}

// ticketsCreateResult is a batch of new tickets by key, in the order they were sent.
type ticketsCreateResult struct {
	Tickets []createdTicket `json:"tickets"`
}

type createdTicket struct {
	ID    string `json:"id"`
	Key   string `json:"key,omitempty"`
	Title string `json:"title"`
}

func ticketCreateTool(t *tickets.Service, w *workspace.Service) mcptool.Tool {
	return mcptool.New("ticket_create", "Create ticket",
		"Creates a ticket in a project and returns it with its key. The reporter is recorded as "+
			"Nexul on behalf of you. project_get lists the project's ticket types with their body_template. "+
			"A bug (the type named bug) needs origin_id, the ticket it was found in, or origin_unknown true "+
			"when nobody knows; never guess an origin. It starts in the project's first backlog column; set its "+
			"status, labels, and blockers afterwards with ticket_update. With tickets it creates each in order, stops at the first "+
			"that fails, and returns their keys.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in ticketCreateIn) (any, error) {
			if err := meToLogin(ctx, t, &in.Developer, &in.Tester); err != nil {
				return nil, err
			}
			if err := in.checkBatch(); err != nil {
				return nil, err
			}
			originID, err := originOf(ctx, t, w, in.ProjectID, in.OriginID)
			if err != nil {
				return nil, err
			}
			if len(in.Tickets) > 0 {
				return createMany(ctx, t, w, in, originID)
			}
			created, err := createOne(ctx, t, in, originID, newTicketIn{Title: in.Title, Body: in.Body})
			if err != nil {
				return nil, err
			}
			return newNames(w).ticket(ctx, created)
		})
}

func createOne(ctx context.Context, t *tickets.Service, in ticketCreateIn, originID string, one newTicketIn) (*tickets.Ticket, error) {
	created, err := t.Create(ctx, in.ProjectID, one.Title, one.Body, in.DocID, in.Developer, tickets.CreateOptions{
		CategoryID: in.CategoryID, TypeID: in.TypeID, Tester: in.Tester, ViaMCP: true,
		OriginID: originID, OriginUnknown: in.OriginUnknown,
	})
	if err != nil {
		return nil, fmt.Errorf("%w (%s)", err, createHint)
	}
	return created, nil
}

// checkBatch refuses a call with no title and no tickets, both at once, a batch past one page, or a blank title in it,
// so a batch that is bound to fail creates nothing.
func (in ticketCreateIn) checkBatch() error {
	if len(in.Tickets) == 0 {
		if in.Title == "" {
			return fmt.Errorf("%w: title or tickets is required", apperrs.ErrInvalid)
		}
		return nil
	}
	if in.Title != "" || in.Body != "" {
		return fmt.Errorf("%w: send title and body, or tickets, not both", apperrs.ErrInvalid)
	}
	if len(in.Tickets) > paging.MaxLimit {
		return fmt.Errorf("%w: tickets takes at most %d, got %d; send the rest in another call", apperrs.ErrInvalid, paging.MaxLimit, len(in.Tickets))
	}
	for i, one := range in.Tickets {
		if strings.TrimSpace(one.Title) == "" {
			return fmt.Errorf("%w: tickets[%d] has no title; nothing was created", apperrs.ErrInvalid, i)
		}
	}
	return nil
}

// createMany creates the batch in order and stops at the first failure, naming the tickets already created.
func createMany(ctx context.Context, t *tickets.Service, w *workspace.Service, in ticketCreateIn, originID string) (ticketsCreateResult, error) {
	n := newNames(w)
	out := ticketsCreateResult{Tickets: make([]createdTicket, 0, len(in.Tickets))}
	done := []string{}
	for i, one := range in.Tickets {
		created, err := createOne(ctx, t, in, originID, one)
		if err != nil {
			return out, batchStopped(fmt.Sprintf("tickets[%d]", i), done, err)
		}
		key, err := n.key(ctx, created)
		if err != nil {
			return out, batchStopped(fmt.Sprintf("tickets[%d]", i), append(done, created.ID), err)
		}
		done = append(done, key)
		out.Tickets = append(out.Tickets, createdTicket{ID: created.ID, Key: key, Title: created.Title})
	}
	return out, nil
}

// batchStopped says which item failed and which were created before it.
func batchStopped(item string, done []string, err error) error {
	if len(done) == 0 {
		return fmt.Errorf("%s: %w; nothing was created", item, err)
	}
	return &mcptool.PartialError{Applied: done, Err: fmt.Errorf("%s: %w; the tickets after it were not tried", item, err)}
}

// keyScope is where related ticket keys resolve: the workspace the caller named, else the project's own.
func keyScope(ctx context.Context, w *workspace.Service, named, projectID string) (string, error) {
	if named != "" {
		return named, nil
	}
	p, err := w.Get(ctx, projectID)
	if err != nil {
		return "", err
	}
	return p.WorkspaceID, nil
}

// originOf turns an origin key into its id; a missing origin is bad input to the new ticket, not a missing ticket.
func originOf(ctx context.Context, t *tickets.Service, w *workspace.Service, projectID, idOrKey string) (string, error) {
	if strings.TrimSpace(idOrKey) == "" {
		return "", nil
	}
	scope, err := keyScope(ctx, w, "", projectID)
	if err != nil {
		return "", fmt.Errorf("%w (%s)", err, createHint)
	}
	origin, err := t.Resolve(ctx, scope, idOrKey)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", fmt.Errorf("%w: origin_id %s is not a ticket; %s", apperrs.ErrInvalid, idOrKey, ticketHint)
	}
	if err != nil {
		return "", err
	}
	return origin.ID, nil
}

type ticketUpdateIn struct {
	ID               string        `json:"id,omitempty" jsonschema:"The ticket's id or key, such as REF-102."`
	IDs              []string      `json:"ids,omitempty" jsonschema:"Instead of id, up to 100 ids or keys to change alike."`
	Workspace        string        `json:"workspace,omitempty" jsonschema:"Workspace id or slug, when a key is in more than one of yours; found_in_id and blocker keys resolve there, or else in the ticket's own."`
	Title            *string       `json:"title,omitempty" jsonschema:"A new title."`
	Body             *string       `json:"body,omitempty" jsonschema:"The ticket's spec, as markdown; an empty string clears it. Change it only when a person asks; add anything you learn afterwards as a note with message_post's file, and if the spec looks wrong, say so and suggest the person edit it."`
	ProjectID        *string       `json:"project_id,omitempty" jsonschema:"Move the ticket to this project, by its id from project_list; set status_id to one of its columns too."`
	StatusID         *string       `json:"status_id,omitempty" jsonschema:"Move the ticket to this status column, by its id from project_get. It lands at the bottom of the column."`
	Position         *int          `json:"position,omitempty" jsonschema:"The ticket's place in its column, 0 for the top."`
	TypeID           *string       `json:"type_id,omitempty" jsonschema:"The ticket type's id, from project_get."`
	CategoryID       *string       `json:"category_id,omitempty" jsonschema:"The category's id, from project_get; an empty string uncategorizes the ticket."`
	Developer        *string       `json:"developer,omitempty" jsonschema:"The member login who builds it, or me; an empty string clears it."`
	Tester           *string       `json:"tester,omitempty" jsonschema:"The member login who tests it, or me; an empty string clears it."`
	DueOn            *string       `json:"due_on,omitempty" jsonschema:"The due date, YYYY-MM-DD such as 2026-10-30; an empty string clears it."`
	DocID            *string       `json:"doc_id,omitempty" jsonschema:"The doc this ticket is derived from, from doc_list; an empty string clears it."`
	AddLabels        []string      `json:"add_labels,omitempty" jsonschema:"Labels to attach."`
	RemoveLabels     []string      `json:"remove_labels,omitempty" jsonschema:"Labels to detach."`
	FoundInID        string        `json:"found_in_id,omitempty" jsonschema:"The id or key of the ticket this bug was found in; replaces any earlier found-in."`
	FoundInUnknown   bool          `json:"found_in_unknown,omitzero" jsonschema:"true marks the found-in ticket as unknown."`
	ClearFoundIn     bool          `json:"clear_found_in,omitzero" jsonschema:"true removes the found-in link or unknown marker."`
	AddBlockerIDs    []string      `json:"add_blocker_ids,omitempty" jsonschema:"Ids or keys of tickets that must merge a pull request or reach a done-stage column first."`
	RemoveBlockerIDs []string      `json:"remove_blocker_ids,omitempty" jsonschema:"Ids or keys of blockers to remove."`
	LinkPR           *prLinkIn     `json:"link_pr,omitempty" jsonschema:"A pull request to link to the ticket."`
	LinkBranch       *branchLinkIn `json:"link_branch,omitempty" jsonschema:"A branch to link to the ticket."`
	UnlinkPR         *prRefIn      `json:"unlink_pr,omitempty" jsonschema:"A linked pull request to unlink."`
	UnlinkBranch     *branchLinkIn `json:"unlink_branch,omitempty" jsonschema:"A linked branch to unlink."`
	RevertToVersion  *int          `json:"revert_to_version,omitempty" jsonschema:"Restore this version as a new one. Send it alone."`

	Edits []textedit.Edit `json:"edits,omitempty" jsonschema:"Replace text in the body instead of resending it."`
}

// revertsAlone reports a revert sent with no other field, which would otherwise be half applied.
func (in ticketUpdateIn) revertsAlone() bool {
	rest := in
	rest.ID, rest.IDs, rest.Workspace, rest.RevertToVersion = "", nil, "", nil
	return reflect.ValueOf(rest).IsZero()
}

type prLinkIn struct {
	Owner  string `json:"owner" jsonschema:"The repository owner."`
	Repo   string `json:"repo" jsonschema:"The repository name."`
	Number int    `json:"number" jsonschema:"The pull request number."`
	Title  string `json:"title,omitempty" jsonschema:"The pull request's title."`
	SHA    string `json:"sha,omitempty" jsonschema:"The pull request's head commit."`
}

type prRefIn struct {
	Owner  string `json:"owner" jsonschema:"The repository owner."`
	Repo   string `json:"repo" jsonschema:"The repository name."`
	Number int    `json:"number" jsonschema:"The pull request number."`
}

type branchLinkIn struct {
	Owner  string `json:"owner" jsonschema:"The repository owner."`
	Repo   string `json:"repo" jsonschema:"The repository name."`
	Branch string `json:"branch" jsonschema:"The branch name."`
}

type ticketUpdateResult struct {
	Applied []string     `json:"applied"`
	Ticket  ticketResult `json:"ticket"`
}

// ticketsUpdateResult is a change made to many tickets: the fields applied to each, and the tickets by key.
type ticketsUpdateResult struct {
	Applied []string `json:"applied"`
	Tickets []string `json:"tickets"`
}

func ticketUpdateTool(t *tickets.Service, w *workspace.Service) mcptool.Tool {
	return mcptool.New("ticket_update", "Update ticket",
		"Changes a ticket's title, body, project, status column and position, type, category, developer "+
			"and tester, due date, source doc, labels, found-in link, blockers, and linked pull requests and branches. "+
			"The changes apply in the order the fields are listed here and "+
			"stop at the first failure; with ids, each ticket in turn, returning the keys changed. revert_to_version restores a version, listed by ticket_get. To record a test "+
			"result, which also moves the ticket, use ticket_test_report; to remove a ticket, ticket_delete.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in ticketUpdateIn) (any, error) {
			if err := in.checkFoundIn(); err != nil {
				return nil, err
			}
			if err := in.checkTargets(); err != nil {
				return nil, err
			}
			if err := meToLogin(ctx, t, in.Developer, in.Tester); err != nil {
				return nil, err
			}
			if len(in.IDs) > 0 {
				return updateMany(ctx, t, w, in)
			}
			if in.RevertToVersion != nil && !in.revertsAlone() {
				return nil, fmt.Errorf("%w: revert_to_version restores that version's whole body; send it alone, then update again", apperrs.ErrInvalid)
			}
			if len(in.Edits) > 0 && in.Body != nil {
				return nil, fmt.Errorf("%w: edits change the current body; send edits or body, not both", apperrs.ErrInvalid)
			}
			tk, err := resolveTicket(ctx, t, in.Workspace, in.ID)
			if err != nil {
				return nil, err
			}
			scope, err := keyScope(ctx, w, in.Workspace, tk.ProjectID)
			if err != nil {
				return nil, err
			}
			u := ticketUpdate{t: t, w: w, id: tk.ID, workspace: scope}
			applied, err := runSteps(t.WithViaMCPOn(ctx, tk.ID), u.allSteps(in))
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

// checkTargets refuses a call naming no ticket, or naming many with a field that only fits one.
func (in ticketUpdateIn) checkTargets() error {
	if in.ID != "" && len(in.IDs) > 0 {
		return fmt.Errorf("%w: send id or ids, not both", apperrs.ErrInvalid)
	}
	if len(in.IDs) == 0 {
		return requireID(in.ID)
	}
	if len(in.IDs) > paging.MaxLimit {
		return fmt.Errorf("%w: ids takes at most %d tickets, got %d; send the rest in another call", apperrs.ErrInvalid, paging.MaxLimit, len(in.IDs))
	}
	for _, f := range []struct {
		name string
		sent bool
	}{
		{"title", in.Title != nil}, {"body", in.Body != nil}, {"edits", len(in.Edits) > 0},
		{"position", in.Position != nil}, {"revert_to_version", in.RevertToVersion != nil},
	} {
		if f.sent {
			return fmt.Errorf("%w: %s changes one ticket; send it with id, not ids", apperrs.ErrInvalid, f.name)
		}
	}
	return nil
}

func requireID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id or ids is required; %s", apperrs.ErrInvalid, ticketHint)
	}
	return nil
}

// updateMany finds every ticket before changing any, so a mistyped key changes nothing, then updates them in order.
func updateMany(ctx context.Context, t *tickets.Service, w *workspace.Service, in ticketUpdateIn) (ticketsUpdateResult, error) {
	targets, err := resolveTickets(ctx, t, in.Workspace, in.IDs)
	if err != nil {
		return ticketsUpdateResult{}, fmt.Errorf("%w; nothing was changed", err)
	}
	n := newNames(w)
	out := ticketsUpdateResult{Applied: []string{}, Tickets: []string{}}
	for _, tk := range targets {
		label, err := n.key(ctx, tk)
		if err != nil {
			return out, stoppedAt(tk.ID, out.Tickets, nil, err)
		}
		scope, err := keyScope(ctx, w, in.Workspace, tk.ProjectID)
		if err != nil {
			return out, stoppedAt(label, out.Tickets, nil, err)
		}
		u := ticketUpdate{t: t, w: w, id: tk.ID, workspace: scope}
		applied, err := runSteps(t.WithViaMCPOn(ctx, tk.ID), u.steps(in))
		if err != nil {
			return out, stoppedAt(label, out.Tickets, applied, err)
		}
		out.Applied = applied
		out.Tickets = append(out.Tickets, label)
	}
	return out, nil
}

// resolveTickets finds each ticket once, however many times ids names it.
func resolveTickets(ctx context.Context, t *tickets.Service, workspace string, refs []string) ([]*tickets.Ticket, error) {
	seen := map[string]bool{}
	out := make([]*tickets.Ticket, 0, len(refs))
	for _, ref := range refs {
		tk, err := resolveTicket(ctx, t, workspace, ref)
		if err != nil {
			return nil, fmt.Errorf("ids: %s: %w", ref, err)
		}
		if seen[tk.ID] {
			continue
		}
		seen[tk.ID] = true
		out = append(out, tk)
	}
	return out, nil
}

// stoppedAt says which ticket failed and which were changed before it, with the fields it got when it got some.
func stoppedAt(label string, done, applied []string, err error) error {
	var partial *mcptool.PartialError
	if errors.As(err, &partial) {
		err = partial.Err
	}
	if len(applied) > 0 {
		done = append(slices.Clone(done), label+" ("+strings.Join(applied, ", ")+")")
	}
	if len(done) == 0 {
		return fmt.Errorf("%s: %w", label, err)
	}
	return &mcptool.PartialError{Applied: done, Err: fmt.Errorf("%s: %w; tickets after %s were not tried", label, err, label)}
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
	t  *tickets.Service
	w  *workspace.Service
	id string
	// workspace is where the keys in the link fields resolve.
	workspace string
}

// allSteps is the one restore a revert makes, or the steps of the fields sent.
func (u ticketUpdate) allSteps(in ticketUpdateIn) []step {
	if in.RevertToVersion == nil {
		return u.steps(in)
	}
	return []step{{"revert_to_version", "ticket_get lists the versions", func(ctx context.Context) error {
		return discard(u.t.RestoreVersion(ctx, u.id, *in.RevertToVersion))
	}}}
}

func (u ticketUpdate) steps(in ticketUpdateIn) []step {
	var steps []step
	edits := len(in.Edits) > 0
	if in.Title != nil || in.Body != nil || edits {
		steps = append(steps, step{field: pairLabel("title", in.Title != nil, "body", in.Body != nil || edits), run: func(ctx context.Context) error { return u.content(ctx, in.Title, in.Body, in.Edits) }})
	}
	fields := []struct {
		sent bool
		step step
	}{
		{in.ProjectID != nil, step{"project_id", "project_list lists the projects", func(ctx context.Context) error { return u.w.MoveTicket(ctx, u.id, *in.ProjectID) }}},
		{in.StatusID != nil, step{"status_id", idHint, func(ctx context.Context) error {
			return discard(u.t.UpdateStatus(ctx, u.id, tickets.Status(*in.StatusID)))
		}}},
		{in.TypeID != nil, step{"type_id", idHint, func(ctx context.Context) error { return discard(u.t.SetType(ctx, u.id, *in.TypeID)) }}},
		{in.CategoryID != nil, step{"category_id", idHint, func(ctx context.Context) error { return u.w.MoveTicketToCategory(ctx, u.id, *in.CategoryID) }}},
		// Last of the placement steps: a status or category move sends the ticket to the bottom first.
		{in.Position != nil, step{"position", "", func(ctx context.Context) error { return u.place(ctx, *in.Position) }}},
		{in.Developer != nil, step{"developer", "", func(ctx context.Context) error {
			return discard(u.t.SetPerson(ctx, u.id, tickets.RoleDeveloper, *in.Developer))
		}}},
		{in.Tester != nil, step{"tester", "", func(ctx context.Context) error {
			return discard(u.t.SetPerson(ctx, u.id, tickets.RoleTester, *in.Tester))
		}}},
		{in.DueOn != nil, step{"due_on", "", func(ctx context.Context) error { return discard(u.t.SetDueOn(ctx, u.id, *in.DueOn)) }}},
		{in.DocID != nil, step{"doc_id", "doc_list lists the docs", func(ctx context.Context) error { return discard(u.t.SetSource(ctx, u.id, *in.DocID)) }}},
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

// place puts the ticket at index in its column cell and renumbers the cell, as the board does when a card is dragged.
func (u ticketUpdate) place(ctx context.Context, index int) error {
	current, err := u.t.Get(ctx, u.id)
	if err != nil {
		return err
	}
	all, err := u.t.ListByProject(ctx, current.ProjectID)
	if err != nil {
		return err
	}
	cell := slices.DeleteFunc(slices.Clone(all), func(t *tickets.Ticket) bool {
		return t.Status != current.Status || t.CategoryID != current.CategoryID
	})
	slices.SortStableFunc(cell, func(a, b *tickets.Ticket) int {
		if a.Position != b.Position {
			return a.Position - b.Position
		}
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	ids := make([]string, 0, len(cell))
	positions := map[string]int{}
	for _, t := range cell {
		ids = append(ids, t.ID)
		positions[t.ID] = t.Position
	}
	for i, id := range moveTo(ids, u.id, index) {
		if positions[id] == i {
			continue
		}
		if _, err := u.t.SetPosition(ctx, id, i); err != nil {
			return err
		}
	}
	return nil
}

// content overlays the sent title and body (or the body with edits applied) on the stored ones, since the use-case replaces both.
func (u ticketUpdate) content(ctx context.Context, title, body *string, edits []textedit.Edit) error {
	current, err := u.t.Get(ctx, u.id)
	if err != nil {
		return err
	}
	if len(edits) > 0 {
		edited, err := u.t.ApplyEdits(ctx, u.id, edits)
		if err != nil {
			return err
		}
		body = &edited
	}
	return discard(u.t.UpdateTicket(ctx, u.id, valueOr(title, current.Title), valueOr(body, current.Body)))
}

func (u ticketUpdate) labelSteps(in ticketUpdateIn) []step {
	var steps []step
	for i, label := range in.AddLabels {
		steps = append(steps, step{fmt.Sprintf("add_labels[%d]", i), "", func(ctx context.Context) error { return discard(u.t.AddLabel(ctx, u.id, label)) }})
	}
	for i, label := range in.RemoveLabels {
		steps = append(steps, step{fmt.Sprintf("remove_labels[%d]", i), "", func(ctx context.Context) error { return discard(u.t.RemoveLabel(ctx, u.id, label)) }})
	}
	return steps
}

func (u ticketUpdate) linkSteps(in ticketUpdateIn) []step {
	var steps []step
	if in.FoundInID != "" || in.FoundInUnknown {
		steps = append(steps, step{"found_in", ticketHint, func(ctx context.Context) error { return u.foundIn(ctx, in.FoundInID, in.FoundInUnknown) }})
	}
	if in.ClearFoundIn {
		steps = append(steps, step{"clear_found_in", "", func(ctx context.Context) error { return discard(u.t.RemoveFoundIn(ctx, u.id)) }})
	}
	for i, ref := range in.AddBlockerIDs {
		steps = append(steps, step{fmt.Sprintf("add_blocker_ids[%d]", i), ticketHint, func(ctx context.Context) error { return u.blocker(ctx, ref, u.t.AddBlocker) }})
	}
	for i, ref := range in.RemoveBlockerIDs {
		steps = append(steps, step{fmt.Sprintf("remove_blocker_ids[%d]", i), ticketHint, func(ctx context.Context) error { return u.blocker(ctx, ref, u.t.RemoveBlocker) }})
	}
	if pr := in.LinkPR; pr != nil {
		steps = append(steps, step{"link_pr", "", func(ctx context.Context) error {
			return u.t.LinkPR(ctx, u.id, tickets.PRRef{Owner: pr.Owner, Repo: pr.Repo, Number: pr.Number, Title: pr.Title, SHA: pr.SHA})
		}})
	}
	if b := in.LinkBranch; b != nil {
		steps = append(steps, step{"link_branch", "", func(ctx context.Context) error { return u.t.LinkBranch(ctx, u.id, b.Owner, b.Repo, b.Branch) }})
	}
	if pr := in.UnlinkPR; pr != nil {
		steps = append(steps, step{"unlink_pr", "", func(ctx context.Context) error {
			return u.t.UnlinkPR(ctx, u.id, tickets.PRRef{Owner: pr.Owner, Repo: pr.Repo, Number: pr.Number})
		}})
	}
	if b := in.UnlinkBranch; b != nil {
		steps = append(steps, step{"unlink_branch", "", func(ctx context.Context) error { return u.t.UnlinkBranch(ctx, u.id, b.Owner, b.Repo, b.Branch) }})
	}
	return steps
}

func (u ticketUpdate) foundIn(ctx context.Context, originRef string, unknown bool) error {
	originID := ""
	if originRef != "" {
		origin, err := u.t.Resolve(ctx, u.workspace, originRef)
		if err != nil {
			return err
		}
		originID = origin.ID
	}
	return discard(u.t.SetFoundIn(ctx, u.id, originID, unknown))
}

func (u ticketUpdate) blocker(ctx context.Context, ref string, apply func(ctx context.Context, id, blockerID string) (*tickets.LinkSet, error)) error {
	blocker, err := u.t.Resolve(ctx, u.workspace, ref)
	if err != nil {
		return err
	}
	return discard(apply(ctx, u.id, blocker.ID))
}

// discard keeps a use-case's error and drops the value an update step does not need.
func discard[T any](_ T, err error) error {
	return err
}

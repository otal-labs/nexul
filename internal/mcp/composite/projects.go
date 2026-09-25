package composite

import (
	"context"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// ProjectTools are the project tools that also read and set label colors, which the tickets domain owns.
func ProjectTools(w *workspace.Service, t *tickets.Service) []mcptool.Tool {
	return []mcptool.Tool{projectGetTool(w, t), projectUpdateTool(w, t)}
}

// projectDetail is everything an agent needs before filing or moving a ticket in a project.
type projectDetail struct {
	*workspace.Project
	Repositories []workspace.RepoRef    `json:"repositories"`
	Statuses     []statusResult         `json:"statuses"`
	Categories   []categoryResult       `json:"categories"`
	TicketTypes  []ticketTypeResult     `json:"ticket_types"`
	Labels       []labelResult          `json:"labels"`
	DeleteImpact workspace.DeleteImpact `json:"delete_impact"`
}

type statusResult struct {
	ID       string               `json:"id"`
	Name     string               `json:"name"`
	Stage    workspace.StatusKind `json:"stage"`
	Icon     workspace.StatusIcon `json:"icon,omitempty"`
	Position int                  `json:"position"`
}

type categoryResult struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Color    colors.Color `json:"color,omitempty"`
	Position int          `json:"position"`
}

type ticketTypeResult struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Color        colors.Color `json:"color,omitempty"`
	BodyTemplate string       `json:"body_template"`
	Position     int          `json:"position"`
}

type labelResult struct {
	Name  string       `json:"name"`
	Color colors.Color `json:"color,omitempty"`
}

type projectGetIn struct {
	ID string `json:"id" jsonschema:"The project's id (a UUID), from project_list."`
}

func projectGetTool(w *workspace.Service, t *tickets.Service) mcptool.Tool {
	return mcptool.New("project_get", "Get project",
		"Returns one project with everything needed to file and move its tickets: its status columns in board "+
			"order (each with its stage: backlog, progress, review, testing, or done), categories, ticket types "+
			"(each with the body_template a new ticket fills), labels with this project's colors, repositories "+
			"(each with its role, app or tests), where its tests live, and delete_impact, the tickets, repositories, "+
			"and services that block deleting it. Call it before ticket_create or ticket_update to get valid "+
			"status, type, and category ids; change any of these with project_update. Use project_list to find a "+
			"project's id.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in projectGetIn) (any, error) {
			return projectView(ctx, w, t, in.ID)
		})
}

func projectView(ctx context.Context, w *workspace.Service, t *tickets.Service, id string) (projectDetail, error) {
	p, err := w.Get(ctx, id)
	if err != nil {
		return projectDetail{}, err
	}
	d := projectDetail{Project: p}
	if d.Repositories, err = w.ListRepos(ctx, id); err != nil {
		return d, err
	}
	if d.Repositories == nil {
		d.Repositories = []workspace.RepoRef{}
	}
	if err := d.addBoard(ctx, w, id); err != nil {
		return d, err
	}
	if d.Labels, err = projectLabels(ctx, t, id); err != nil {
		return d, err
	}
	d.DeleteImpact, err = w.DeleteImpact(ctx, id)
	return d, err
}

func (d *projectDetail) addBoard(ctx context.Context, w *workspace.Service, id string) error {
	statuses, err := w.ListStatusesByProject(ctx, id)
	if err != nil {
		return err
	}
	d.Statuses = make([]statusResult, 0, len(statuses))
	for _, s := range statuses {
		d.Statuses = append(d.Statuses, statusResult{ID: s.ID, Name: s.Name, Stage: s.Kind, Icon: s.Icon, Position: s.Position})
	}
	cats, err := w.ListCategoriesByProject(ctx, id)
	if err != nil {
		return err
	}
	d.Categories = make([]categoryResult, 0, len(cats))
	for _, c := range cats {
		d.Categories = append(d.Categories, categoryResult{ID: c.ID, Name: c.Name, Color: c.Color, Position: c.Position})
	}
	types, err := w.ListTicketTypesByProject(ctx, id)
	if err != nil {
		return err
	}
	d.TicketTypes = make([]ticketTypeResult, 0, len(types))
	for _, tt := range types {
		d.TicketTypes = append(d.TicketTypes, ticketTypeResult{ID: tt.ID, Name: tt.Name, Color: tt.Color, BodyTemplate: tt.BodyTemplate, Position: tt.Position})
	}
	return nil
}

// projectLabels mirrors the board's filter bar: every label in use, with this project's color for each.
func projectLabels(ctx context.Context, t *tickets.Service, projectID string) ([]labelResult, error) {
	all, err := t.ListAllLabels(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]labelResult, 0, len(all))
	if len(all) == 0 {
		return out, nil
	}
	byLabel, err := t.LabelColors(ctx, projectID, all)
	if err != nil {
		return nil, err
	}
	for _, l := range all {
		out = append(out, labelResult{Name: l, Color: byLabel[l]})
	}
	return out, nil
}

type projectUpdateIn struct {
	ID            string             `json:"id" jsonschema:"The project's id (a UUID), from project_list."`
	Name          *string            `json:"name,omitempty" jsonschema:"A new display name."`
	Icon          *string            `json:"icon,omitempty" jsonschema:"A display icon: Box, Rocket, Server, Globe, Database, Layers, Terminal, Shield, Zap, Package, Cpu, or Cloud. An empty string clears it."`
	Prefix        *string            `json:"prefix,omitempty" jsonschema:"2 to 5 letters for a project that has no prefix yet, for example REF. An existing prefix never changes, since ticket keys are built from it."`
	Position      *int               `json:"position,omitempty" jsonschema:"The project's place in the workspace's order, 0 for first."`
	TestsLocation *string            `json:"tests_location,omitempty" jsonschema:"Where the tests live: same (in the repository that deploys), separate (in a tests repository), or an empty string to withdraw the answer."`
	AddRepos      []repoAddIn        `json:"add_repos,omitempty" jsonschema:"Repositories to attach; a repository belongs to one project only."`
	RemoveRepos   []repoRefIn        `json:"remove_repos,omitempty" jsonschema:"Repositories of this project to detach."`
	Statuses      *statusChanges     `json:"statuses,omitempty" jsonschema:"Status columns to create, change, or delete."`
	Categories    *categoryChanges   `json:"categories,omitempty" jsonschema:"Categories to create, change, or delete."`
	TicketTypes   *ticketTypeChanges `json:"ticket_types,omitempty" jsonschema:"Ticket types to create, change, or delete."`
	LabelColors   []labelColorIn     `json:"label_colors,omitempty" jsonschema:"Colors for labels in this project, even for a label no ticket uses yet."`
}

type repoAddIn struct {
	Owner       string `json:"owner" jsonschema:"The repository owner, for example otal-labs."`
	Name        string `json:"name" jsonschema:"The repository name, for example nexul."`
	ConnectorID string `json:"connector_id,omitempty" jsonschema:"The git connector that hosts it. Defaults to github."`
	Role        string `json:"role,omitempty" jsonschema:"app (the default, the repository stacks build from) or tests (a tests repository, never deployed; attaching one records the tests location as separate)."`
}

type repoRefIn struct {
	Owner string `json:"owner" jsonschema:"The repository owner, for example otal-labs."`
	Name  string `json:"name" jsonschema:"The repository name, for example nexul."`
}

type statusChanges struct {
	Create []statusCreateIn `json:"create,omitempty" jsonschema:"New columns, added at the end of the board."`
	Update []statusUpdateIn `json:"update,omitempty" jsonschema:"Changes to existing columns; omitted fields keep their values."`
	Delete []string         `json:"delete,omitempty" jsonschema:"Ids of columns to delete; a column still holding tickets is refused."`
}

type statusCreateIn struct {
	Name  string `json:"name" jsonschema:"The column's name, for example In review."`
	Stage string `json:"stage,omitempty" jsonschema:"The stage rules read: backlog, progress, review, testing, or done. Defaults to backlog."`
	Icon  string `json:"icon,omitempty" jsonschema:"A display icon: CircleDashed, Circle, CircleDot, CircleEllipsis, CircleCheckBig, or CircleX."`
}

type statusUpdateIn struct {
	ID       string  `json:"id" jsonschema:"The column's id, from project_get."`
	Name     *string `json:"name,omitempty" jsonschema:"A new name; tickets keep their column."`
	Stage    *string `json:"stage,omitempty" jsonschema:"A new stage: backlog, progress, review, testing, or done."`
	Icon     *string `json:"icon,omitempty" jsonschema:"A new icon from the list statuses.create gives; an empty string clears it."`
	Position *int    `json:"position,omitempty" jsonschema:"The column's place on the board, 0 for first."`
}

type categoryChanges struct {
	Create []categoryCreateIn `json:"create,omitempty" jsonschema:"New categories, added last."`
	Update []categoryUpdateIn `json:"update,omitempty" jsonschema:"Changes to existing categories; omitted fields keep their values."`
	Delete []string           `json:"delete,omitempty" jsonschema:"Ids of categories to delete; their tickets become uncategorized, never deleted."`
}

type categoryCreateIn struct {
	Name  string `json:"name" jsonschema:"The category's name, for example Sprint 1."`
	Color string `json:"color,omitempty" jsonschema:"A display color: cyan, emerald, orange, fuchsia, or lime."`
}

type categoryUpdateIn struct {
	ID       string  `json:"id" jsonschema:"The category's id, from project_get."`
	Name     *string `json:"name,omitempty" jsonschema:"A new name."`
	Color    *string `json:"color,omitempty" jsonschema:"cyan, emerald, orange, fuchsia, or lime; an empty string clears it."`
	Position *int    `json:"position,omitempty" jsonschema:"The category's place in the project's order, 0 for first."`
}

type ticketTypeChanges struct {
	Create []ticketTypeCreateIn `json:"create,omitempty" jsonschema:"New ticket types, added last."`
	Update []ticketTypeUpdateIn `json:"update,omitempty" jsonschema:"Changes to existing ticket types; omitted fields keep their values."`
	Delete []string             `json:"delete,omitempty" jsonschema:"Ids of ticket types to delete; a type still used by a ticket is refused."`
}

type ticketTypeCreateIn struct {
	Name         string `json:"name" jsonschema:"The type's name, for example chore. A type named bug requires a found-in on its tickets."`
	Color        string `json:"color,omitempty" jsonschema:"A display color: cyan, emerald, orange, fuchsia, or lime."`
	BodyTemplate string `json:"body_template,omitempty" jsonschema:"Markdown sections that pre-fill a new ticket of this type."`
}

type ticketTypeUpdateIn struct {
	ID           string  `json:"id" jsonschema:"The ticket type's id, from project_get."`
	Name         *string `json:"name,omitempty" jsonschema:"A new name; tickets keep their type."`
	Color        *string `json:"color,omitempty" jsonschema:"cyan, emerald, orange, fuchsia, or lime; an empty string clears it."`
	BodyTemplate *string `json:"body_template,omitempty" jsonschema:"A new body template in markdown; existing tickets are never rewritten, and an empty string clears it."`
	Position     *int    `json:"position,omitempty" jsonschema:"The type's place in the project's order, 0 for first."`
}

type labelColorIn struct {
	Label string `json:"label" jsonschema:"The label, for example urgent."`
	Color string `json:"color" jsonschema:"cyan, emerald, orange, fuchsia, or lime."`
}

type projectUpdateResult struct {
	Applied []string      `json:"applied"`
	Project projectDetail `json:"project"`
}

func projectUpdateTool(w *workspace.Service, t *tickets.Service) mcptool.Tool {
	return mcptool.New("project_update", "Update project",
		"Changes a project: its name, icon, prefix (only when it has none), place in the workspace, and tests "+
			"location; attaches and detaches repositories; creates, changes, reorders (position), and deletes its "+
			"status columns, categories, and ticket types; and sets label colors. Only the fields you send change; "+
			"an omitted field keeps its value. The changes apply in the order the fields are listed here, creates "+
			"before updates before deletes, and stop at the first failure, whose message names the field and the "+
			"ones already applied. Owners only, except label colors. Returns the updated project as project_get "+
			"shows it, with new ids, and the list of fields applied.",
		mcptool.Hints{Local: true},
		func(ctx context.Context, in projectUpdateIn) (any, error) {
			actor, err := ownerActor(ctx)
			if err != nil {
				return nil, err
			}
			if _, err := w.Get(ctx, in.ID); err != nil {
				return nil, err
			}
			u := projectUpdate{ctx: ctx, w: w, t: t, actor: actor, id: in.ID}
			applied, err := runSteps(u.steps(in))
			if err != nil {
				return nil, err
			}
			view, err := projectView(ctx, w, t, in.ID)
			if err != nil {
				return nil, err
			}
			return projectUpdateResult{Applied: applied, Project: view}, nil
		})
}

// projectUpdate builds a project_update's steps; each closure reads its field only when the field was sent.
type projectUpdate struct {
	ctx       context.Context
	w         *workspace.Service
	t         *tickets.Service
	actor, id string
}

func (u projectUpdate) steps(in projectUpdateIn) []step {
	var steps []step
	if in.Name != nil || in.Icon != nil {
		steps = append(steps, step{field: pairLabel("name", in.Name != nil, "icon", in.Icon != nil), run: func() error { return u.rename(in.Name, in.Icon) }})
	}
	if in.Prefix != nil {
		steps = append(steps, step{"prefix", "", func() error { return discard(u.w.SetPrefix(u.ctx, u.actor, u.id, *in.Prefix)) }})
	}
	if in.Position != nil {
		steps = append(steps, step{"position", "", func() error { return u.moveProject(*in.Position) }})
	}
	if in.TestsLocation != nil {
		steps = append(steps, step{"tests_location", "", func() error {
			return discard(u.w.SetTestsLocation(u.ctx, u.actor, u.id, workspace.TestsLocation(*in.TestsLocation)))
		}})
	}
	steps = append(steps, u.repoSteps(in)...)
	steps = append(steps, u.statusSteps(in.Statuses)...)
	steps = append(steps, u.categorySteps(in.Categories)...)
	steps = append(steps, u.ticketTypeSteps(in.TicketTypes)...)
	for i, lc := range in.LabelColors {
		steps = append(steps, step{fmt.Sprintf("label_colors[%d]", i), "", func() error {
			return discard(u.t.SetLabelColor(u.ctx, u.id, lc.Label, colors.Color(lc.Color)))
		}})
	}
	return steps
}

func (u projectUpdate) rename(name, icon *string) error {
	current, err := u.w.Get(u.ctx, u.id)
	if err != nil {
		return err
	}
	return discard(u.w.Rename(u.ctx, u.actor, u.id, valueOr(name, current.Name), (*workspace.ProjectIcon)(icon)))
}

func (u projectUpdate) moveProject(position int) error {
	p, err := u.w.Get(u.ctx, u.id)
	if err != nil {
		return err
	}
	projects, err := u.w.List(u.ctx, p.WorkspaceID)
	if err != nil {
		return err
	}
	return u.w.Reorder(u.ctx, u.actor, p.WorkspaceID, moveTo(idsOf(projects, func(p *workspace.Project) string { return p.ID }), u.id, position))
}

func (u projectUpdate) repoSteps(in projectUpdateIn) []step {
	var steps []step
	for i, r := range in.AddRepos {
		steps = append(steps, step{fmt.Sprintf("add_repos[%d]", i), "", func() error {
			return u.w.AddRepo(u.ctx, u.actor, u.id, r.Owner, r.Name, r.ConnectorID, workspace.RepoRole(r.Role))
		}})
	}
	for i, r := range in.RemoveRepos {
		steps = append(steps, step{fmt.Sprintf("remove_repos[%d]", i), "project_get lists the project's repositories", func() error { return u.removeRepo(r) }})
	}
	return steps
}

// removeRepo detaches only this project's repository; the use-case finds a repository by name in any project.
func (u projectUpdate) removeRepo(r repoRefIn) error {
	repos, err := u.w.ListRepos(u.ctx, u.id)
	if err != nil {
		return err
	}
	for _, have := range repos {
		if strings.EqualFold(have.Owner, r.Owner) && strings.EqualFold(have.Name, r.Name) {
			return u.w.RemoveRepo(u.ctx, u.actor, have.Owner, have.Name)
		}
	}
	return fmt.Errorf("%w: %s/%s is not a repository of this project", apperrs.ErrInvalid, r.Owner, r.Name)
}

func (u projectUpdate) statusSteps(c *statusChanges) []step {
	if c == nil {
		return nil
	}
	var steps []step
	for i, s := range c.Create {
		steps = append(steps, step{fmt.Sprintf("statuses.create[%d]", i), "", func() error {
			return discard(u.w.CreateStatus(u.ctx, u.actor, u.id, s.Name, workspace.StatusKind(s.Stage), workspace.StatusIcon(s.Icon)))
		}})
	}
	for i, s := range c.Update {
		steps = append(steps, step{fmt.Sprintf("statuses.update[%d]", i), idHint, func() error { return u.updateStatus(s) }})
	}
	for i, id := range c.Delete {
		steps = append(steps, step{fmt.Sprintf("statuses.delete[%d]", i), idHint, func() error {
			current, err := u.w.GetStatus(u.ctx, id)
			if err != nil {
				return err
			}
			if err := u.owns("status column", id, current.ProjectID); err != nil {
				return err
			}
			return u.w.DeleteStatus(u.ctx, u.actor, id)
		}})
	}
	return steps
}

func (u projectUpdate) updateStatus(in statusUpdateIn) error {
	current, err := u.w.GetStatus(u.ctx, in.ID)
	if err != nil {
		return err
	}
	if err := u.owns("status column", in.ID, current.ProjectID); err != nil {
		return err
	}
	if in.Name != nil || in.Stage != nil || in.Icon != nil {
		stage := workspace.StatusKind(valueOr(in.Stage, string(current.Kind)))
		icon := workspace.StatusIcon(valueOr(in.Icon, string(current.Icon)))
		if _, err := u.w.RenameStatus(u.ctx, u.actor, in.ID, valueOr(in.Name, current.Name), stage, icon); err != nil {
			return err
		}
	}
	if in.Position == nil {
		return nil
	}
	statuses, err := u.w.ListStatusesByProject(u.ctx, u.id)
	if err != nil {
		return err
	}
	order := moveTo(idsOf(statuses, func(s *workspace.Status) string { return s.ID }), in.ID, *in.Position)
	return u.w.ReorderStatuses(u.ctx, u.actor, u.id, order)
}

func (u projectUpdate) categorySteps(c *categoryChanges) []step {
	if c == nil {
		return nil
	}
	var steps []step
	for i, cat := range c.Create {
		steps = append(steps, step{fmt.Sprintf("categories.create[%d]", i), "", func() error {
			return discard(u.w.CreateCategory(u.ctx, u.actor, u.id, cat.Name, colors.Color(cat.Color)))
		}})
	}
	for i, cat := range c.Update {
		steps = append(steps, step{fmt.Sprintf("categories.update[%d]", i), idHint, func() error { return u.updateCategory(cat) }})
	}
	for i, id := range c.Delete {
		steps = append(steps, step{fmt.Sprintf("categories.delete[%d]", i), idHint, func() error {
			current, err := u.w.GetCategory(u.ctx, id)
			if err != nil {
				return err
			}
			if err := u.owns("category", id, current.ProjectID); err != nil {
				return err
			}
			return u.w.DeleteCategory(u.ctx, u.actor, id)
		}})
	}
	return steps
}

func (u projectUpdate) updateCategory(in categoryUpdateIn) error {
	current, err := u.w.GetCategory(u.ctx, in.ID)
	if err != nil {
		return err
	}
	if err := u.owns("category", in.ID, current.ProjectID); err != nil {
		return err
	}
	if in.Name != nil || in.Color != nil {
		color := colors.Color(valueOr(in.Color, string(current.Color)))
		if _, err := u.w.RenameCategory(u.ctx, u.actor, in.ID, valueOr(in.Name, current.Name), color); err != nil {
			return err
		}
	}
	if in.Position == nil {
		return nil
	}
	cats, err := u.w.ListCategoriesByProject(u.ctx, u.id)
	if err != nil {
		return err
	}
	order := moveTo(idsOf(cats, func(c *workspace.Category) string { return c.ID }), in.ID, *in.Position)
	return u.w.ReorderCategories(u.ctx, u.actor, u.id, order)
}

func (u projectUpdate) ticketTypeSteps(c *ticketTypeChanges) []step {
	if c == nil {
		return nil
	}
	var steps []step
	for i, tt := range c.Create {
		steps = append(steps, step{fmt.Sprintf("ticket_types.create[%d]", i), "", func() error { return u.createTicketType(tt) }})
	}
	for i, tt := range c.Update {
		steps = append(steps, step{fmt.Sprintf("ticket_types.update[%d]", i), idHint, func() error { return u.updateTicketType(tt) }})
	}
	for i, id := range c.Delete {
		steps = append(steps, step{fmt.Sprintf("ticket_types.delete[%d]", i), idHint, func() error {
			current, err := u.w.GetTicketType(u.ctx, id)
			if err != nil {
				return err
			}
			if err := u.owns("ticket type", id, current.ProjectID); err != nil {
				return err
			}
			return u.w.DeleteTicketType(u.ctx, u.actor, id)
		}})
	}
	return steps
}

func (u projectUpdate) createTicketType(in ticketTypeCreateIn) error {
	created, err := u.w.CreateTicketType(u.ctx, u.actor, u.id, in.Name, colors.Color(in.Color))
	if err != nil || in.BodyTemplate == "" {
		return err
	}
	return discard(u.w.SetTicketTypeTemplate(u.ctx, u.actor, created.ID, in.BodyTemplate))
}

func (u projectUpdate) updateTicketType(in ticketTypeUpdateIn) error {
	current, err := u.w.GetTicketType(u.ctx, in.ID)
	if err != nil {
		return err
	}
	if err := u.owns("ticket type", in.ID, current.ProjectID); err != nil {
		return err
	}
	if in.Name != nil || in.Color != nil {
		color := colors.Color(valueOr(in.Color, string(current.Color)))
		if _, err := u.w.RenameTicketType(u.ctx, u.actor, in.ID, valueOr(in.Name, current.Name), color); err != nil {
			return err
		}
	}
	if in.BodyTemplate != nil {
		if _, err := u.w.SetTicketTypeTemplate(u.ctx, u.actor, in.ID, *in.BodyTemplate); err != nil {
			return err
		}
	}
	if in.Position == nil {
		return nil
	}
	types, err := u.w.ListTicketTypesByProject(u.ctx, u.id)
	if err != nil {
		return err
	}
	order := moveTo(idsOf(types, func(tt *workspace.TicketType) string { return tt.ID }), in.ID, *in.Position)
	return u.w.ReorderTicketTypes(u.ctx, u.actor, u.id, order)
}

// owns refuses a column, category, or type of another project, which the use-cases would change by id alone.
func (u projectUpdate) owns(kind, id, projectID string) error {
	if projectID == u.id {
		return nil
	}
	return fmt.Errorf("%w: %s %s is not in this project", apperrs.ErrInvalid, kind, id)
}

// valueOr overlays a sent patch field on the stored value.
func valueOr[T any](sent *T, current T) T {
	if sent != nil {
		return *sent
	}
	return current
}

func idsOf[T any](items []T, id func(T) string) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = id(item)
	}
	return out
}

// pairLabel names which of a step's two fields were sent.
func pairLabel(first string, firstSent bool, second string, secondSent bool) string {
	if firstSent && secondSent {
		return first + ", " + second
	}
	if firstSent {
		return first
	}
	return second
}

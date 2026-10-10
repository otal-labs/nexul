package composite

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// DocFolders is the slice of the docs use-cases project_get and project_update read and change doc folders through.
type DocFolders interface {
	ListFolders(ctx context.Context, projectID string) ([]*docs.Folder, error)
	CreateFolder(ctx context.Context, projectID, name string) (*docs.Folder, error)
	RenameFolder(ctx context.Context, id, name string) (*docs.Folder, error)
	DeleteFolder(ctx context.Context, id string) error
}

// ProjectTools are the project tools that also read and set label colors and doc folders, which tickets and docs own.
func ProjectTools(w *workspace.Service, t *tickets.Service, d DocFolders) []mcptool.Tool {
	return []mcptool.Tool{projectGetTool(w, t, d), projectUpdateTool(w, t, d)}
}

// projectDetail is everything an agent needs before filing or moving a ticket in a project.
type projectDetail struct {
	*workspace.Project
	Repositories []workspace.RepoRef            `json:"repositories"`
	Statuses     []statusResult                 `json:"statuses"`
	Categories   []categoryResult               `json:"categories"`
	TicketTypes  []ticketTypeResult             `json:"ticket_types"`
	Labels       []labelResult                  `json:"labels"`
	DocFolders   []docFolderResult              `json:"doc_folders"`
	DeleteImpact workspace.DeleteImpact         `json:"delete_impact"`
	Access       []workspace.ProjectAccessEntry `json:"access"`
}

type docFolderResult struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
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

func projectGetTool(w *workspace.Service, t *tickets.Service, d DocFolders) mcptool.Tool {
	return mcptool.New("project_get", "Get project",
		"Returns one project with everything needed to file and move its tickets: its status columns in board "+
			"order (each with its stage: backlog, progress, review, testing, or done), categories, ticket types "+
			"(each with the body_template a new ticket fills), labels with this project's colors, repositories "+
			"(each with its role, app or tests), where its tests live, its doc folders (the one-level groups every "+
			"doc lives in exactly one of, the default Main first, where a new doc goes unless doc_create names "+
			"another), delete_impact, the tickets, repositories, "+
			"and services that block deleting it and the Restricted members who lose access with it, and access, the "+
			"Restricted members who may open it with the actions they hold, filled only when you hold members:write in "+
			"its workspace and empty otherwise; account_update changes that access. Its setup says whether the project "+
			"wizard was finished and which steps were done or skipped; an unfinished project still works in full. Call it before ticket_create or ticket_update to get valid "+
			"status, type, and category ids, and before doc_create or doc_update to get folder ids; change any of "+
			"these with project_update. Use project_list to find a project's id.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in projectGetIn) (any, error) {
			return projectView(ctx, w, t, d, in.ID)
		})
}

func projectView(ctx context.Context, w *workspace.Service, t *tickets.Service, folders DocFolders, id string) (projectDetail, error) {
	p, err := getProject(ctx, w, id)
	if err != nil {
		return projectDetail{}, err
	}
	d := projectDetail{Project: p}
	if d.Repositories, err = w.ListRepos(ctx, id); err != nil {
		return d, err
	}
	if err := d.addBoard(ctx, w, id); err != nil {
		return d, err
	}
	if d.Labels, err = projectLabels(ctx, t, id); err != nil {
		return d, err
	}
	if d.DocFolders, err = projectDocFolders(ctx, folders, id); err != nil {
		return d, err
	}
	if d.DeleteImpact, err = w.DeleteImpact(ctx, id); err != nil {
		return d, err
	}
	d.Access, err = projectAccess(ctx, w, id)
	return d, err
}

// projectAccess is empty rather than an error for a caller without members:write, so project_get still answers.
func projectAccess(ctx context.Context, w *workspace.Service, id string) ([]workspace.ProjectAccessEntry, error) {
	entries, err := w.ProjectAccess(ctx, id)
	if errors.Is(err, apperrs.ErrForbidden) {
		return []workspace.ProjectAccessEntry{}, nil
	}
	return entries, err
}

// getProject names the tool that lists projects when the one asked for is missing.
func getProject(ctx context.Context, w *workspace.Service, id string) (*workspace.Project, error) {
	p, err := w.Get(ctx, id)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("%w; project_list lists a workspace's projects", err)
	}
	return p, err
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

// projectDocFolders lists the folders the caller may see: every one with docs:write, else those holding a doc they can open.
func projectDocFolders(ctx context.Context, d DocFolders, projectID string) ([]docFolderResult, error) {
	folders, err := d.ListFolders(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]docFolderResult, 0, len(folders))
	for _, f := range folders {
		out = append(out, docFolderResult{ID: f.ID, Name: f.Name, IsDefault: f.IsDefault})
	}
	return out, nil
}

type projectUpdateIn struct {
	ID            string             `json:"id" jsonschema:"The project's id (a UUID), from project_list."`
	Name          *string            `json:"name,omitempty" jsonschema:"A new display name."`
	Icon          *string            `json:"icon,omitempty" jsonschema:"A display icon: Box, Rocket, Server, Globe, Database, Layers, Terminal, Shield, Zap, Package, Cpu, or Cloud. An empty string clears it."`
	Prefix        *string            `json:"prefix,omitempty" jsonschema:"2 to 5 letters or digits, starting with a letter, for a project that has no prefix yet, for example REF or P1. An existing prefix never changes, since ticket keys are built from it."`
	Position      *int               `json:"position,omitempty" jsonschema:"The project's place in the workspace's order, 0 for first."`
	TestsLocation *string            `json:"tests_location,omitempty" jsonschema:"Where the tests live: same (in the repository that deploys), separate (in a tests repository), or an empty string to withdraw the answer."`
	AddRepos      []repoAddIn        `json:"add_repos,omitempty" jsonschema:"Repositories to attach; a repository belongs to one project only."`
	RemoveRepos   []repoRefIn        `json:"remove_repos,omitempty" jsonschema:"Repositories of this project to detach."`
	Statuses      *statusChanges     `json:"statuses,omitempty" jsonschema:"Status columns to create, change, or delete."`
	Categories    *categoryChanges   `json:"categories,omitempty" jsonschema:"Categories to create, change, or delete."`
	TicketTypes   *ticketTypeChanges `json:"ticket_types,omitempty" jsonschema:"Ticket types to create, change, or delete."`
	LabelColors   []labelColorIn     `json:"label_colors,omitempty" jsonschema:"Colors for labels in this project, even for a label no ticket uses yet."`
	DocFolders    *docFolderChanges  `json:"doc_folders,omitempty" jsonschema:"Doc folders to create, rename, or delete. A folder groups the project's docs one level deep and every doc lives in exactly one; the default folder, Main, takes new docs and can be renamed but never deleted. Needs docs:write."`
	Setup         *setupIn           `json:"setup,omitempty" jsonschema:"The project wizard's record of this project, as project_get shows it under setup."`
}

type setupIn struct {
	Finished *bool             `json:"finished,omitempty" jsonschema:"true finishes setup, so the sidebar lists the project's pages instead of Continue setup; false reopens it."`
	Steps    map[string]string `json:"steps,omitempty" jsonschema:"Wizard steps to mark, each done or skipped, keyed by step: project, repository, service, env, reach, or branches. A step already done stays done when marked skipped."`
}

type docFolderChanges struct {
	Create []docFolderCreateIn `json:"create,omitempty" jsonschema:"New folders, listed after the existing ones."`
	Update []docFolderUpdateIn `json:"update,omitempty" jsonschema:"Folders to rename, the default one included."`
	Delete []string            `json:"delete,omitempty" jsonschema:"Ids of folders to delete; their docs move to the default folder, never deleted, and the default folder itself is refused."`
}

type docFolderCreateIn struct {
	Name string `json:"name" jsonschema:"The folder's name, unique in the project ignoring case, for example GetSource."`
}

type docFolderUpdateIn struct {
	ID   string `json:"id" jsonschema:"The folder's id, from project_get's doc_folders."`
	Name string `json:"name" jsonschema:"The folder's new name, unique in the project ignoring case."`
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

func projectUpdateTool(w *workspace.Service, t *tickets.Service, d DocFolders) mcptool.Tool {
	return mcptool.New("project_update", "Update project",
		"Changes a project: its name, icon, prefix (only when it has none), place in the workspace, and tests "+
			"location; attaches and detaches repositories; creates, changes, reorders (position), and deletes its "+
			"status columns, categories, and ticket types; sets label colors; and creates, renames, and deletes doc "+
			"folders, the groups a project's docs live in (doc_update moves a doc between them); and finishes or reopens its setup "+
			"and marks project wizard steps done or skipped. Only the fields you send change; "+
			"an omitted field keeps its value. The changes apply in the order the fields are listed here, creates "+
			"before updates before deletes, and stop at the first failure, whose message names the field and the "+
			"ones already applied. Owners only, except label colors and doc folders, which take docs:write. Returns the updated project as project_get "+
			"shows it, with new ids, and the list of fields applied.",
		mcptool.Hints{Local: true},
		func(ctx context.Context, in projectUpdateIn) (any, error) {
			if err := requireOwnerActor(ctx); err != nil {
				return nil, err
			}
			if _, err := getProject(ctx, w, in.ID); err != nil {
				return nil, err
			}
			u := projectUpdate{w: w, t: t, folders: d, id: in.ID}
			applied, err := runSteps(ctx, u.steps(in))
			if err != nil {
				return nil, err
			}
			view, err := projectView(ctx, w, t, d, in.ID)
			if err != nil {
				return nil, err
			}
			return projectUpdateResult{Applied: applied, Project: view}, nil
		})
}

// projectUpdate builds a project_update's steps; each closure reads its field only when the field was sent.
type projectUpdate struct {
	w       *workspace.Service
	t       *tickets.Service
	folders DocFolders
	id      string
}

func (u projectUpdate) steps(in projectUpdateIn) []step {
	var steps []step
	if in.Name != nil || in.Icon != nil {
		steps = append(steps, step{field: pairLabel("name", in.Name != nil, "icon", in.Icon != nil), run: func(ctx context.Context) error { return u.rename(ctx, in.Name, in.Icon) }})
	}
	if in.Prefix != nil {
		steps = append(steps, step{"prefix", "", func(ctx context.Context) error {
			return discard(u.w.SetPrefix(ctx, u.id, *in.Prefix))
		}})
	}
	if in.Position != nil {
		steps = append(steps, step{"position", "", func(ctx context.Context) error { return u.moveProject(ctx, *in.Position) }})
	}
	if in.TestsLocation != nil {
		steps = append(steps, step{"tests_location", "", func(ctx context.Context) error {
			return discard(u.w.SetTestsLocation(ctx, u.id, workspace.TestsLocation(*in.TestsLocation)))
		}})
	}
	steps = append(steps, u.repoSteps(in)...)
	steps = append(steps, u.statusSteps(in.Statuses)...)
	steps = append(steps, u.categorySteps(in.Categories)...)
	steps = append(steps, u.ticketTypeSteps(in.TicketTypes)...)
	for i, lc := range in.LabelColors {
		steps = append(steps, step{fmt.Sprintf("label_colors[%d]", i), "", func(ctx context.Context) error {
			return discard(u.t.SetLabelColor(ctx, u.id, lc.Label, colors.Color(lc.Color)))
		}})
	}
	steps = append(steps, u.docFolderSteps(in.DocFolders)...)
	if in.Setup != nil {
		steps = append(steps, step{"setup", "", func(ctx context.Context) error {
			change := workspace.SetupChange{Finished: in.Setup.Finished, Steps: map[workspace.SetupStep]workspace.SetupMark{}}
			for step, mark := range in.Setup.Steps {
				change.Steps[workspace.SetupStep(step)] = workspace.SetupMark(mark)
			}
			return discard(u.w.ChangeSetup(ctx, u.id, change))
		}})
	}
	return steps
}

const docFolderHint = "project_get lists the project's doc folders"

func (u projectUpdate) docFolderSteps(c *docFolderChanges) []step {
	if c == nil {
		return nil
	}
	var steps []step
	for i, f := range c.Create {
		steps = append(steps, step{fmt.Sprintf("doc_folders.create[%d]", i), "", func(ctx context.Context) error {
			return discard(u.folders.CreateFolder(ctx, u.id, f.Name))
		}})
	}
	for i, f := range c.Update {
		steps = append(steps, step{fmt.Sprintf("doc_folders.update[%d]", i), docFolderHint, func(ctx context.Context) error {
			if err := u.ownsFolder(ctx, f.ID); err != nil {
				return err
			}
			return discard(u.folders.RenameFolder(ctx, f.ID, f.Name))
		}})
	}
	for i, id := range c.Delete {
		steps = append(steps, step{fmt.Sprintf("doc_folders.delete[%d]", i), docFolderHint, func(ctx context.Context) error {
			if err := u.ownsFolder(ctx, id); err != nil {
				return err
			}
			return u.folders.DeleteFolder(ctx, id)
		}})
	}
	return steps
}

// ownsFolder refuses a folder of another project, which the use-cases would change by id alone.
func (u projectUpdate) ownsFolder(ctx context.Context, id string) error {
	folders, err := u.folders.ListFolders(ctx, u.id)
	if err != nil {
		return err
	}
	for _, f := range folders {
		if f.ID == id {
			return nil
		}
	}
	return fmt.Errorf("%w: doc folder %s is not in this project", apperrs.ErrInvalid, id)
}

func (u projectUpdate) rename(ctx context.Context, name, icon *string) error {
	current, err := u.w.Get(ctx, u.id)
	if err != nil {
		return err
	}
	return discard(u.w.Rename(ctx, u.id, valueOr(name, current.Name), (*workspace.ProjectIcon)(icon)))
}

func (u projectUpdate) moveProject(ctx context.Context, position int) error {
	p, err := u.w.Get(ctx, u.id)
	if err != nil {
		return err
	}
	projects, err := u.w.List(ctx, p.WorkspaceID)
	if err != nil {
		return err
	}
	return u.w.Reorder(ctx, p.WorkspaceID, moveTo(idsOf(projects, func(p *workspace.Project) string {
		return p.ID
	}), u.id, position))
}

func (u projectUpdate) repoSteps(in projectUpdateIn) []step {
	var steps []step
	for i, r := range in.AddRepos {
		steps = append(steps, step{fmt.Sprintf("add_repos[%d]", i), "", func(ctx context.Context) error {
			return u.w.AddRepo(ctx, u.id, r.Owner, r.Name, r.ConnectorID, workspace.RepoRole(r.Role))
		}})
	}
	for i, r := range in.RemoveRepos {
		steps = append(steps, step{fmt.Sprintf("remove_repos[%d]", i), "project_get lists the project's repositories", func(ctx context.Context) error { return u.removeRepo(ctx, r) }})
	}
	return steps
}

// removeRepo detaches only this project's repository; the use-case finds a repository by name in any project.
func (u projectUpdate) removeRepo(ctx context.Context, r repoRefIn) error {
	repos, err := u.w.ListRepos(ctx, u.id)
	if err != nil {
		return err
	}
	for _, have := range repos {
		if strings.EqualFold(have.Owner, r.Owner) && strings.EqualFold(have.Name, r.Name) {
			return u.w.RemoveRepo(ctx, have.Owner, have.Name)
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
		steps = append(steps, step{fmt.Sprintf("statuses.create[%d]", i), "", func(ctx context.Context) error {
			return discard(u.w.CreateStatus(ctx, u.id, s.Name, workspace.StatusKind(s.Stage), workspace.StatusIcon(s.Icon)))
		}})
	}
	for i, s := range c.Update {
		steps = append(steps, step{fmt.Sprintf("statuses.update[%d]", i), idHint, func(ctx context.Context) error { return u.updateStatus(ctx, s) }})
	}
	for i, id := range c.Delete {
		steps = append(steps, step{fmt.Sprintf("statuses.delete[%d]", i), idHint, func(ctx context.Context) error {
			current, err := u.w.GetStatus(ctx, id)
			if err != nil {
				return err
			}
			if err := u.owns("status column", id, current.ProjectID); err != nil {
				return err
			}
			return u.w.DeleteStatus(ctx, id)
		}})
	}
	return steps
}

func (u projectUpdate) updateStatus(ctx context.Context, in statusUpdateIn) error {
	current, err := u.w.GetStatus(ctx, in.ID)
	if err != nil {
		return err
	}
	if err := u.owns("status column", in.ID, current.ProjectID); err != nil {
		return err
	}
	if in.Name != nil || in.Stage != nil || in.Icon != nil {
		stage := workspace.StatusKind(valueOr(in.Stage, string(current.Kind)))
		icon := workspace.StatusIcon(valueOr(in.Icon, string(current.Icon)))
		if _, err := u.w.RenameStatus(ctx, in.ID, valueOr(in.Name, current.Name), stage, icon); err != nil {
			return err
		}
	}
	if in.Position == nil {
		return nil
	}
	statuses, err := u.w.ListStatusesByProject(ctx, u.id)
	if err != nil {
		return err
	}
	order := moveTo(idsOf(statuses, func(s *workspace.Status) string { return s.ID }), in.ID, *in.Position)
	return u.w.ReorderStatuses(ctx, u.id, order)
}

func (u projectUpdate) categorySteps(c *categoryChanges) []step {
	if c == nil {
		return nil
	}
	var steps []step
	for i, cat := range c.Create {
		steps = append(steps, step{fmt.Sprintf("categories.create[%d]", i), "", func(ctx context.Context) error {
			return discard(u.w.CreateCategory(ctx, u.id, cat.Name, colors.Color(cat.Color)))
		}})
	}
	for i, cat := range c.Update {
		steps = append(steps, step{fmt.Sprintf("categories.update[%d]", i), idHint, func(ctx context.Context) error { return u.updateCategory(ctx, cat) }})
	}
	for i, id := range c.Delete {
		steps = append(steps, step{fmt.Sprintf("categories.delete[%d]", i), idHint, func(ctx context.Context) error {
			current, err := u.w.GetCategory(ctx, id)
			if err != nil {
				return err
			}
			if err := u.owns("category", id, current.ProjectID); err != nil {
				return err
			}
			return u.w.DeleteCategory(ctx, id)
		}})
	}
	return steps
}

func (u projectUpdate) updateCategory(ctx context.Context, in categoryUpdateIn) error {
	current, err := u.w.GetCategory(ctx, in.ID)
	if err != nil {
		return err
	}
	if err := u.owns("category", in.ID, current.ProjectID); err != nil {
		return err
	}
	if in.Name != nil || in.Color != nil {
		color := colors.Color(valueOr(in.Color, string(current.Color)))
		if _, err := u.w.RenameCategory(ctx, in.ID, valueOr(in.Name, current.Name), color); err != nil {
			return err
		}
	}
	if in.Position == nil {
		return nil
	}
	cats, err := u.w.ListCategoriesByProject(ctx, u.id)
	if err != nil {
		return err
	}
	order := moveTo(idsOf(cats, func(c *workspace.Category) string { return c.ID }), in.ID, *in.Position)
	return u.w.ReorderCategories(ctx, u.id, order)
}

func (u projectUpdate) ticketTypeSteps(c *ticketTypeChanges) []step {
	if c == nil {
		return nil
	}
	var steps []step
	for i, tt := range c.Create {
		steps = append(steps, step{fmt.Sprintf("ticket_types.create[%d]", i), "", func(ctx context.Context) error { return u.createTicketType(ctx, tt) }})
	}
	for i, tt := range c.Update {
		steps = append(steps, step{fmt.Sprintf("ticket_types.update[%d]", i), idHint, func(ctx context.Context) error { return u.updateTicketType(ctx, tt) }})
	}
	for i, id := range c.Delete {
		steps = append(steps, step{fmt.Sprintf("ticket_types.delete[%d]", i), idHint, func(ctx context.Context) error {
			current, err := u.w.GetTicketType(ctx, id)
			if err != nil {
				return err
			}
			if err := u.owns("ticket type", id, current.ProjectID); err != nil {
				return err
			}
			return u.w.DeleteTicketType(ctx, id)
		}})
	}
	return steps
}

func (u projectUpdate) createTicketType(ctx context.Context, in ticketTypeCreateIn) error {
	created, err := u.w.CreateTicketType(ctx, u.id, in.Name, colors.Color(in.Color))
	if err != nil || in.BodyTemplate == "" {
		return err
	}
	return discard(u.w.SetTicketTypeTemplate(ctx, created.ID, in.BodyTemplate))
}

func (u projectUpdate) updateTicketType(ctx context.Context, in ticketTypeUpdateIn) error {
	current, err := u.w.GetTicketType(ctx, in.ID)
	if err != nil {
		return err
	}
	if err := u.owns("ticket type", in.ID, current.ProjectID); err != nil {
		return err
	}
	if in.Name != nil || in.Color != nil {
		color := colors.Color(valueOr(in.Color, string(current.Color)))
		if _, err := u.w.RenameTicketType(ctx, in.ID, valueOr(in.Name, current.Name), color); err != nil {
			return err
		}
	}
	if in.BodyTemplate != nil {
		if _, err := u.w.SetTicketTypeTemplate(ctx, in.ID, *in.BodyTemplate); err != nil {
			return err
		}
	}
	if in.Position == nil {
		return nil
	}
	types, err := u.w.ListTicketTypesByProject(ctx, u.id)
	if err != nil {
		return err
	}
	order := moveTo(idsOf(types, func(tt *workspace.TicketType) string { return tt.ID }), in.ID, *in.Position)
	return u.w.ReorderTicketTypes(ctx, u.id, order)
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

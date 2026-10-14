package roles

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/welcomestep"
)

// TemplateKind is the instance template kind a Template role is kept under.
const TemplateKind = "role"

// TemplateBody is a Template role's text: the role every new workspace starts with a copy of.
type TemplateBody struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Permissions  permissions.Set `json:"permissions"`
	WelcomeSteps []string        `json:"welcome_steps"`
	// ShowLockedActions is the switch the copy starts with (ADR 0152).
	ShowLockedActions bool `json:"show_locked_actions"`
}

// ShippedTemplate is one Template role Nexul ships, under its stable key.
type ShippedTemplate struct {
	Key  string
	Body TemplateBody
}

// JSON is the body as stored.
func (b TemplateBody) JSON() string {
	out, _ := json.Marshal(b) // strings and string lists always encode
	return string(out)
}

// ParseTemplate reads a Template role's body, refusing a missing name, an action off the grid, or an unknown step.
func ParseTemplate(body string) (TemplateBody, error) {
	var raw struct {
		Name              string   `json:"name"`
		Description       string   `json:"description"`
		Permissions       []string `json:"permissions"`
		WelcomeSteps      []string `json:"welcome_steps"`
		ShowLockedActions bool     `json:"show_locked_actions"`
	}
	dec := json.NewDecoder(strings.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return TemplateBody{}, fmt.Errorf(`%w: a role template is JSON {"name", "description", "permissions", "welcome_steps", "show_locked_actions"}: %v`, apperrs.ErrInvalid, err)
	}
	b := TemplateBody{Name: strings.TrimSpace(raw.Name), ShowLockedActions: raw.ShowLockedActions}
	if b.Name == "" {
		return TemplateBody{}, fmt.Errorf("%w: a role template needs a name", apperrs.ErrInvalid)
	}
	description, err := checkDescription(raw.Description)
	if err != nil {
		return TemplateBody{}, err
	}
	b.Description = description
	for _, p := range raw.Permissions {
		action, ok := permissions.ParseAction(p)
		if !ok {
			return TemplateBody{}, fmt.Errorf("%w: %q is not a permission; the catalog lists them", apperrs.ErrInvalid, p)
		}
		b.Permissions = append(b.Permissions, action)
	}
	b.Permissions = permissions.SetOf(b.Permissions...)
	steps, err := welcomestep.Parse(raw.WelcomeSteps)
	if err != nil {
		return TemplateBody{}, err
	}
	b.WelcomeSteps = steps
	return b, nil
}

// CheckTemplate validates a Template role's body before the instance stores it.
func CheckTemplate(body string) error {
	_, err := ParseTemplate(body)
	return err
}

// TemplateName is the role name a Template role's body carries, "" for one that doesn't parse.
func TemplateName(body string) string {
	t, _ := ParseTemplate(body) // an unparsable body has no name to show
	return t.Name
}

// TemplateRoles reads every Template role's text, shipped and added, in the order a new workspace lists them.
type TemplateRoles interface {
	Bodies(ctx context.Context, kind string) ([]string, error)
}

// SetTemplateRoles wires the instance layer a new workspace's roles are copied from.
func (s *Service) SetTemplateRoles(t TemplateRoles) { s.templates = t }

// SeedTemplateRoles gives a new workspace a copy of every Template role beside its Owner; it never rewrites a copy, and
// skips a name the workspace already has, so a retried seed adds nothing twice.
func (s *Service) SeedTemplateRoles(ctx context.Context, workspaceID string) error {
	if s.templates == nil {
		return nil
	}
	bodies, err := s.templates.Bodies(ctx, TemplateKind)
	if err != nil {
		return fmt.Errorf("read the template roles: %w", err)
	}
	existing, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("list roles for workspace %s: %w", workspaceID, err)
	}
	taken := map[string]bool{}
	for _, r := range existing {
		taken[strings.ToLower(r.Name)] = true
	}
	now := s.now().UTC()
	for _, body := range bodies {
		t, err := ParseTemplate(body)
		if err != nil {
			return fmt.Errorf("template role: %w", err)
		}
		if taken[strings.ToLower(t.Name)] {
			continue
		}
		taken[strings.ToLower(t.Name)] = true
		r := &Role{ID: ids.New(), WorkspaceID: workspaceID, Name: t.Name, Description: t.Description, Permissions: t.Permissions, WelcomeSteps: t.WelcomeSteps, ShowLockedActions: t.ShowLockedActions, CreatedAt: now, UpdatedAt: now}
		if err := s.repo.Create(ctx, r); err != nil {
			return fmt.Errorf("copy template role %q into workspace %s: %w", t.Name, workspaceID, err)
		}
	}
	return nil
}

// ShippedTemplates are the five Template roles Nexul ships; a project reaches none of them until it lists the role.
func ShippedTemplates() []ShippedTemplate {
	chat := []permissions.Action{
		"chat:read", "chat:write", "channels:read", "channels:write", "voice:read", "voice:write",
		"mentions:read", "mentions:write", "notifications:read", "notifications:write",
	}
	set := func(actions ...permissions.Action) permissions.Set {
		return permissions.SetOf(append(actions, chat...)...)
	}
	devops := append([]permissions.Action{"tickets:read", "tickets:write", "docs:read", "docs:write",
		"attachments:read", "attachments:write", "memories:read", "repos:read", "reviews:read",
		"repositories:read", "repositories:write",
		"stacks:read", "stacks:write", "stacks:delete", "stacks:logs", "deploys:read", "deploys:write",
		"projects:read", "plays:read"}, deployInfrastructure...)
	return []ShippedTemplate{
		{"client", TemplateBody{
			Name:        "Client",
			Description: "Reads and writes the tickets and docs of the projects you're added to.",
			Permissions: set("tickets:read", "tickets:write", "docs:read", "docs:write", "docs:thread",
				"attachments:read", "attachments:write", "memories:read"),
			WelcomeSteps: []string{welcomestep.Projects},
		}},
		{"junior_developer", TemplateBody{
			Name:        "Junior Developer",
			Description: "Works tickets and docs, runs plays and reads deploys in the projects you're added to.",
			Permissions: set("tickets:read", "tickets:write", "docs:read", "docs:write", "docs:thread",
				"attachments:read", "attachments:write", "memories:read", "repos:read", "reviews:read", "repositories:read",
				"stacks:read", "stacks:logs", "deploys:read", "plays:read", "plays:run"),
			WelcomeSteps: []string{welcomestep.Computer, welcomestep.Projects}, ShowLockedActions: true,
		}},
		{"senior_developer", TemplateBody{
			Name:        "Senior Developer",
			Description: "Runs the tickets, docs, plays and deploys of the projects you're added to, and creates projects.",
			Permissions: set("tickets:read", "tickets:write", "tickets:delete",
				"docs:read", "docs:write", "docs:delete", "docs:thread", "docs:clone", "docs:lock",
				"attachments:read", "attachments:write", "attachments:delete",
				"memories:read", "memories:write", "memories:delete", "memories:template",
				"repos:read", "reviews:read", "repositories:read", "repositories:write",
				"stacks:read", "stacks:write", "stacks:logs", "deploys:read", "deploys:write",
				"projects:read", "projects:write", "projects:create",
				"plays:read", "plays:write", "plays:delete", "plays:run", "autoplays:read", "autoplays:write",
				"topology:read"),
			WelcomeSteps: []string{welcomestep.GitHub, welcomestep.Computer}, ShowLockedActions: true,
		}},
		{"devops", TemplateBody{
			Name:         "DevOps",
			Description:  "Deploys the stacks of the projects you're added to and looks after machines, runners, DNS and gateways.",
			Permissions:  set(devops...),
			WelcomeSteps: []string{welcomestep.DeployServer}, ShowLockedActions: true,
		}},
		{"administrator", TemplateBody{
			Name:              "Administrator",
			Description:       "Does everything in the workspace except what stays with the Owner.",
			Permissions:       administrator(),
			WelcomeSteps:      []string{welcomestep.GitHub},
			ShowLockedActions: true,
		}},
	}
}

// deployInfrastructure is deploying to, managing and deleting machines and the workspace's topology canvas; seeing a
// machine takes its rules, and adding one and the instance's DNS stay with the instance Owner (ADR 0151).
var deployInfrastructure = []permissions.Action{
	"machines:deploy", "machines:write", "machines:delete", "topology:read", "topology:write", "topology:delete",
}

// ownerOnly stays with Owner even though its area is the workspace: making or removing a workspace is as strong as Owner.
var ownerOnly = []permissions.Action{permissions.WorkspacesCreate, permissions.WorkspacesDelete}

// administrator holds every project and workspace action and the deploy infrastructure, none of the instance's own
// administration, which stays with Owner.
func administrator() permissions.Set {
	out := slices.Clone(deployInfrastructure)
	for _, info := range permissions.Catalog() {
		if info.Area != permissions.AreaInstance && !slices.Contains(ownerOnly, info.Value) {
			out = append(out, info.Value)
		}
	}
	return permissions.SetOf(out...)
}

package tickets

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the tickets tool definitions (named domain_action).
func MCPTools(s *Service) []mcptool.Tool {
	var tools []mcptool.Tool
	tools = append(tools, ticketCoreMCPTools(s)...)
	tools = append(tools, ticketLabelMCPTools(s)...)
	tools = append(tools, ticketSearchLinkMCPTools(s)...)
	tools = append(tools, ticketLinkMCPTools(s)...)
	return tools
}

func ticketCoreMCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "ticket_create",
			Description: "Create a ticket in a project, optionally derived from a doc, and return it. developer and tester are member logins; the reporter is recorded as Nexul on behalf of the calling user. Fill the type's body_template (from ticket_type_list) as the body; an empty body with a type_id is created from that template.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id":  map[string]any{"type": "string"},
					"title":       map[string]any{"type": "string"},
					"body":        map[string]any{"type": "string"},
					"doc_id":      map[string]any{"type": "string"},
					"developer":   map[string]any{"type": "string"},
					"tester":      map[string]any{"type": "string"},
					"category_id": map[string]any{"type": "string"},
					"type_id":     map[string]any{"type": "string"},
				},
				"required": []string{"project_id", "title"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "project_id", "title")
				if err != nil {
					return nil, err
				}
				projectID, title := vals[0], vals[1]
				return s.Create(ctx, projectID, title, mcptool.OptionalString(args["body"]), mcptool.OptionalString(args["doc_id"]), mcptool.OptionalString(args["developer"]), CreateOptions{
					CategoryID: mcptool.OptionalString(args["category_id"]),
					TypeID:     mcptool.OptionalString(args["type_id"]),
					Tester:     mcptool.OptionalString(args["tester"]),
					ViaMCP:     true,
				})
			},
		},
		{
			Name:        "ticket_get",
			Description: "Fetch a single ticket by id.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.Get(ctx, id)
			},
		},
		{
			Name:        "ticket_update",
			Description: "Edit a ticket's title and body in place. Body may be empty to clear it.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":    map[string]any{"type": "string"},
					"title": map[string]any{"type": "string"},
					"body":  map[string]any{"type": "string"},
				},
				"required": []string{"id", "title"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "title")
				if err != nil {
					return nil, err
				}
				body, _ := args["body"].(string)
				return s.UpdateTicket(ctx, vals[0], vals[1], body)
			},
		},
		{
			Name:        "ticket_update_status",
			Description: "Move a ticket to a status column id (list configured columns with status_list).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string"},
					"status": map[string]any{"type": "string"},
				},
				"required": []string{"id", "status"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "status")
				if err != nil {
					return nil, err
				}
				id, status := vals[0], vals[1]
				return s.UpdateStatus(ctx, id, Status(status))
			},
		},
		ticketSetPersonTool(s, RoleDeveloper, "ticket_set_developer", "Set the member login who builds a ticket; an empty login clears it."),
		ticketSetPersonTool(s, RoleTester, "ticket_set_tester", "Set the member login who tests a ticket in its testing stage; an empty login clears it."),
		{
			Name:        "ticket_set_type",
			Description: "Set a ticket's type id (list configured types with ticket_type_list).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":      map[string]any{"type": "string"},
					"type_id": map[string]any{"type": "string"},
				},
				"required": []string{"id", "type_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "type_id")
				if err != nil {
					return nil, err
				}
				id, typeID := vals[0], vals[1]
				return s.SetType(ctx, id, typeID)
			},
		},
	}
}

func ticketSetPersonTool(s *Service, role Role, name, description string) mcptool.Tool {
	return mcptool.Tool{
		Name:        name,
		Description: description,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":    map[string]any{"type": "string"},
				"login": map[string]any{"type": "string"},
			},
			"required": []string{"id"},
		},
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			id, err := mcptool.RequiredString(args, "id")
			if err != nil {
				return nil, err
			}
			return s.SetPerson(ctx, id, role, mcptool.OptionalString(args["login"]))
		},
	}
}

func ticketLabelMCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "ticket_add_label",
			Description: "Attach a cross-cutting label (e.g. bug, urgent) to a ticket.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":    map[string]any{"type": "string"},
					"label": map[string]any{"type": "string"},
				},
				"required": []string{"id", "label"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "label")
				if err != nil {
					return nil, err
				}
				id, label := vals[0], vals[1]
				return s.AddLabel(ctx, id, label)
			},
		},
		{
			Name:        "ticket_remove_label",
			Description: "Remove a label from a ticket.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":    map[string]any{"type": "string"},
					"label": map[string]any{"type": "string"},
				},
				"required": []string{"id", "label"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "label")
				if err != nil {
					return nil, err
				}
				id, label := vals[0], vals[1]
				return s.RemoveLabel(ctx, id, label)
			},
		},
		{
			Name:        "ticket_list_labels",
			Description: "List the labels attached to a ticket.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.ListLabels(ctx, id)
			},
		},
		{
			Name:        "ticket_list_all_labels",
			Description: "List the distinct labels across all tickets, ordered.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.ListAllLabels(ctx)
			},
		},
		{
			Name:        "ticket_set_label_color",
			Description: "Set or update a label's suggested-palette color within a project. Works even for a label no ticket has used yet (create-or-update).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
					"label":      map[string]any{"type": "string"},
					"color":      map[string]any{"type": "string", "enum": colorNames()},
				},
				"required": []string{"project_id", "label", "color"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "project_id", "label", "color")
				if err != nil {
					return nil, err
				}
				projectID, label, color := vals[0], vals[1], vals[2]
				return s.SetLabelColor(ctx, projectID, label, colors.Color(color))
			},
		},
		{
			Name:        "ticket_label_colors",
			Description: "Fetch a project's configured colors for a set of labels in one batched call.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
					"labels":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
				"required": []string{"project_id", "labels"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				projectID, err := mcptool.RequiredString(args, "project_id")
				if err != nil {
					return nil, err
				}
				labels, err := stringSliceArg(args, "labels")
				if err != nil {
					return nil, err
				}
				return s.LabelColors(ctx, projectID, labels)
			},
		},
	}
}

func ticketSearchLinkMCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "ticket_search",
			Description: "Full-text search over ticket titles and bodies.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"limit": map[string]any{"type": "integer"},
				},
				"required": []string{"query"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				query, err := mcptool.RequiredString(args, "query")
				if err != nil {
					return nil, err
				}
				return s.Search(ctx, query, intArg(args["limit"]))
			},
		},
		{
			Name:        "ticket_link_pr",
			Description: "Link an existing pull request to a ticket.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string"},
					"owner":  map[string]any{"type": "string"},
					"repo":   map[string]any{"type": "string"},
					"number": map[string]any{"type": "integer"},
					"title":  map[string]any{"type": "string"},
					"sha":    map[string]any{"type": "string"},
				},
				"required": []string{"id", "owner", "repo", "number"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "owner", "repo")
				if err != nil {
					return nil, err
				}
				id, owner, repo := vals[0], vals[1], vals[2]
				number := intArg(args["number"])
				if number < 1 {
					return nil, fmt.Errorf("%w: number is required", apperrs.ErrInvalid)
				}
				if err := s.LinkPR(ctx, id, PRRef{Owner: owner, Repo: repo, Number: number, Title: mcptool.OptionalString(args["title"]), SHA: mcptool.OptionalString(args["sha"])}); err != nil {
					return nil, err
				}
				return s.getLinksResult(ctx, id)
			},
		},
		{
			Name:        "ticket_link_branch",
			Description: "Link an existing branch to a ticket.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string"},
					"owner":  map[string]any{"type": "string"},
					"repo":   map[string]any{"type": "string"},
					"branch": map[string]any{"type": "string"},
				},
				"required": []string{"id", "owner", "repo", "branch"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "owner", "repo", "branch")
				if err != nil {
					return nil, err
				}
				id, owner, repo, branch := vals[0], vals[1], vals[2], vals[3]
				if err := s.LinkBranch(ctx, id, owner, repo, branch); err != nil {
					return nil, err
				}
				return s.getLinksResult(ctx, id)
			},
		},
		{
			Name:        "ticket_get_links",
			Description: "List the branches and pull requests linked to a ticket.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.getLinksResult(ctx, id)
			},
		},
	}
}

func (s *Service) getLinksResult(ctx context.Context, id string) (any, error) {
	prs, branches, err := s.ListLinks(ctx, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"prs": prs, "branches": branches}, nil
}

func colorNames() []string {
	all := colors.All()
	names := make([]string, len(all))
	for i, c := range all {
		names[i] = string(c)
	}
	return names
}

func stringSliceArg(args map[string]any, key string) ([]string, error) {
	raw, ok := args[key].([]any)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("%w: %s is required", apperrs.ErrInvalid, key)
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		if !ok || s == "" {
			return nil, fmt.Errorf("%w: %s must be a list of strings", apperrs.ErrInvalid, key)
		}
		out = append(out, s)
	}
	return out, nil
}

func intArg(v any) int {
	switch n := v.(type) {
	case float64:
		if n > 0 {
			return int(n)
		}
	case int:
		if n > 0 {
			return n
		}
	}
	return 0
}

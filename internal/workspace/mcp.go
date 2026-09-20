package workspace

import (
	"context"
	"fmt"
	"sort"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the workspace tool definitions (named domain_action).
func MCPTools(s *Service) []mcptool.Tool {
	var tools []mcptool.Tool
	tools = append(tools, projectMCPTools(s)...)
	tools = append(tools, projectRepoMCPTools(s)...)
	tools = append(tools, categoryMCPTools(s)...)
	tools = append(tools, ticketTypeMCPTools(s)...)
	tools = append(tools, statusMCPTools(s)...)
	return tools
}

func projectMCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "project_create",
			Description: "Create a project (the organizational grouping tickets belong to) inside a workspace, and return it. Prefix is an immutable 2-5 letter tag (e.g. \"REF\") used to render human-readable ticket ids like REF-102. Icon is an optional display choice from the suggested list.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"workspace_id": map[string]any{"type": "string"},
					"name":         map[string]any{"type": "string"},
					"prefix":       map[string]any{"type": "string"},
					"icon":         map[string]any{"type": "string", "enum": projectIconNames()},
				},
				"required": []string{"workspace_id", "name", "prefix"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "workspace_id", "name", "prefix")
				if err != nil {
					return nil, err
				}
				workspaceID, name, prefix := vals[0], vals[1], vals[2]
				icon := mcptool.OptionalString(args["icon"])
				return s.Create(ctx, "", workspaceID, name, prefix, ProjectIcon(icon))
			},
		},
		{
			Name:        "project_get",
			Description: "Fetch a single project by id.",
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
			Name:        "project_list",
			Description: "List all projects in a workspace.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"workspace_id": map[string]any{"type": "string"},
				},
				"required": []string{"workspace_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				workspaceID, err := mcptool.RequiredString(args, "workspace_id")
				if err != nil {
					return nil, err
				}
				return s.List(ctx, workspaceID)
			},
		},
		{
			Name:        "project_rename",
			Description: "Rename a project and/or change its icon.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":   map[string]any{"type": "string"},
					"name": map[string]any{"type": "string"},
					"icon": map[string]any{"type": "string", "enum": projectIconNames()},
				},
				"required": []string{"id", "name"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "name")
				if err != nil {
					return nil, err
				}
				id, name := vals[0], vals[1]
				var icon *ProjectIcon
				if _, ok := args["icon"]; ok {
					v := ProjectIcon(mcptool.OptionalString(args["icon"]))
					icon = &v
				}
				return s.Rename(ctx, "", id, name, icon)
			},
		},
		{
			Name:        "project_delete",
			Description: "Delete an empty project (one with no tickets or repositories).",
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
				if err := s.Delete(ctx, "", id); err != nil {
					return nil, err
				}
				return map[string]string{"id": id, "status": "deleted"}, nil
			},
		},
		{
			Name:        "project_reorder",
			Description: "Set the display order of a workspace's projects (every id must appear exactly once).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"workspace_id": map[string]any{"type": "string"},
					"ids":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
				"required": []string{"workspace_id", "ids"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				workspaceID, err := mcptool.RequiredString(args, "workspace_id")
				if err != nil {
					return nil, err
				}
				ids, err := stringSliceArg(args, "ids")
				if err != nil {
					return nil, err
				}
				if err := s.Reorder(ctx, "", workspaceID, ids); err != nil {
					return nil, err
				}
				return map[string]string{"status": "reordered"}, nil
			},
		},
		{
			Name:        "project_delete_impact",
			Description: "Report how many tickets and repositories a project holds (what deleting it would affect).",
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
				return s.DeleteImpact(ctx, id)
			},
		},
	}
}

func projectRepoMCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "project_add_repo",
			Description: "Associate a repository with a project (a repository belongs to exactly one project). connector_id names which connected git connector hosts it; omit it to default to \"github\".",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id":   map[string]any{"type": "string"},
					"owner":        map[string]any{"type": "string"},
					"name":         map[string]any{"type": "string"},
					"connector_id": map[string]any{"type": "string"},
				},
				"required": []string{"project_id", "owner", "name"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "project_id", "owner", "name")
				if err != nil {
					return nil, err
				}
				projectID, owner, name := vals[0], vals[1], vals[2]
				connectorID := mcptool.OptionalString(args["connector_id"])
				if err := s.AddRepo(ctx, "", projectID, owner, name, connectorID); err != nil {
					return nil, err
				}
				return map[string]string{"project_id": projectID, "owner": owner, "name": name}, nil
			},
		},
		{
			Name:        "project_remove_repo",
			Description: "Dissociate a repository from its project.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"owner": map[string]any{"type": "string"},
					"name":  map[string]any{"type": "string"},
				},
				"required": []string{"owner", "name"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "owner", "name")
				if err != nil {
					return nil, err
				}
				owner, name := vals[0], vals[1]
				if err := s.RemoveRepo(ctx, "", owner, name); err != nil {
					return nil, err
				}
				return map[string]string{"owner": owner, "name": name, "status": "removed"}, nil
			},
		},
		{
			Name:        "project_list_repos",
			Description: "List the repositories associated with a project.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
				},
				"required": []string{"project_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				projectID, err := mcptool.RequiredString(args, "project_id")
				if err != nil {
					return nil, err
				}
				return s.ListRepos(ctx, projectID)
			},
		},
		{
			Name:        "project_move_ticket",
			Description: "Move a ticket into a project without changing its identity.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ticket_id":  map[string]any{"type": "string"},
					"project_id": map[string]any{"type": "string"},
				},
				"required": []string{"ticket_id", "project_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "ticket_id", "project_id")
				if err != nil {
					return nil, err
				}
				ticketID, projectID := vals[0], vals[1]
				if err := s.MoveTicket(ctx, ticketID, projectID); err != nil {
					return nil, err
				}
				return map[string]string{"ticket_id": ticketID, "project_id": projectID, "status": "moved"}, nil
			},
		},
	}
}

func categoryMCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "category_create",
			Description: "Create a category (structural grouping, e.g. Sprint 1) in a project; color is an optional display choice from the suggested list.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
					"name":       map[string]any{"type": "string"},
					"color":      map[string]any{"type": "string", "enum": ticketTypeColorNames()},
				},
				"required": []string{"project_id", "name"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "project_id", "name")
				if err != nil {
					return nil, err
				}
				projectID, name := vals[0], vals[1]
				color := mcptool.OptionalString(args["color"])
				return s.CreateCategory(ctx, "", projectID, name, colors.Color(color))
			},
		},
		{
			Name:        "category_get",
			Description: "Fetch a single category by id.",
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
				return s.GetCategory(ctx, id)
			},
		},
		{
			Name:        "category_list",
			Description: "List all categories in the workspace (optionally for one project).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
				},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				if pid := mcptool.OptionalString(args["project_id"]); pid != "" {
					return s.ListCategoriesByProject(ctx, pid)
				}
				return s.ListCategories(ctx)
			},
		},
		{
			Name:        "category_rename",
			Description: "Rename a category and/or change its color.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":    map[string]any{"type": "string"},
					"name":  map[string]any{"type": "string"},
					"color": map[string]any{"type": "string", "enum": ticketTypeColorNames()},
				},
				"required": []string{"id", "name"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "name")
				if err != nil {
					return nil, err
				}
				id, name := vals[0], vals[1]
				color := mcptool.OptionalString(args["color"])
				return s.RenameCategory(ctx, "", id, name, colors.Color(color))
			},
		},
		{
			Name:        "category_delete",
			Description: "Delete a category. Its tickets become uncategorized — they are never deleted.",
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
				if err := s.DeleteCategory(ctx, "", id); err != nil {
					return nil, err
				}
				return map[string]string{"id": id, "status": "deleted"}, nil
			},
		},
		{
			Name:        "category_reorder",
			Description: "Set the display order of a project's categories (every id must appear exactly once).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
					"ids":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
				"required": []string{"project_id", "ids"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				projectID, err := mcptool.RequiredString(args, "project_id")
				if err != nil {
					return nil, err
				}
				ids, err := stringSliceArg(args, "ids")
				if err != nil {
					return nil, err
				}
				if err := s.ReorderCategories(ctx, "", projectID, ids); err != nil {
					return nil, err
				}
				return map[string]string{"status": "reordered"}, nil
			},
		},
		{
			Name:        "category_move_ticket",
			Description: "Move a ticket into a category without changing its identity.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ticket_id":   map[string]any{"type": "string"},
					"category_id": map[string]any{"type": "string"},
				},
				"required": []string{"ticket_id", "category_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "ticket_id", "category_id")
				if err != nil {
					return nil, err
				}
				ticketID, categoryID := vals[0], vals[1]
				if err := s.MoveTicketToCategory(ctx, ticketID, categoryID); err != nil {
					return nil, err
				}
				return map[string]string{"ticket_id": ticketID, "category_id": categoryID, "status": "moved"}, nil
			},
		},
		{
			Name:        "category_clear_ticket",
			Description: "Uncategorize a ticket (it stays on the board in the uncategorized swimlane).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ticket_id": map[string]any{"type": "string"},
				},
				"required": []string{"ticket_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				ticketID, err := mcptool.RequiredString(args, "ticket_id")
				if err != nil {
					return nil, err
				}
				if err := s.MoveTicketToCategory(ctx, ticketID, ""); err != nil {
					return nil, err
				}
				return map[string]string{"ticket_id": ticketID, "status": "uncategorized"}, nil
			},
		},
	}
}

func ticketTypeMCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "ticket_type_create",
			Description: "Create a ticket type (bug, feature, task, ...) in a project; color is an optional display choice from the suggested list.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
					"name":       map[string]any{"type": "string"},
					"color":      map[string]any{"type": "string", "enum": ticketTypeColorNames()},
				},
				"required": []string{"project_id", "name"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "project_id", "name")
				if err != nil {
					return nil, err
				}
				projectID, name := vals[0], vals[1]
				color := mcptool.OptionalString(args["color"])
				return s.CreateTicketType(ctx, "", projectID, name, colors.Color(color))
			},
		},
		{
			Name:        "ticket_type_list",
			Description: "List a project's ticket types.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
				},
				"required": []string{"project_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				projectID, err := mcptool.RequiredString(args, "project_id")
				if err != nil {
					return nil, err
				}
				return s.ListTicketTypesByProject(ctx, projectID)
			},
		},
		{
			Name:        "ticket_type_rename",
			Description: "Rename a ticket type and/or change its color.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":    map[string]any{"type": "string"},
					"name":  map[string]any{"type": "string"},
					"color": map[string]any{"type": "string", "enum": ticketTypeColorNames()},
				},
				"required": []string{"id", "name"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "name")
				if err != nil {
					return nil, err
				}
				id, name := vals[0], vals[1]
				color := mcptool.OptionalString(args["color"])
				return s.RenameTicketType(ctx, "", id, name, colors.Color(color))
			},
		},
		{
			Name:        "ticket_type_delete",
			Description: "Delete an unused ticket type (one with no tickets assigned).",
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
				if err := s.DeleteTicketType(ctx, "", id); err != nil {
					return nil, err
				}
				return map[string]string{"id": id, "status": "deleted"}, nil
			},
		},
	}
}

func statusMCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "status_create",
			Description: "Create a status column in a project under one of the board stages (kind: backlog, progress, review, testing, done); icon is an optional display choice from the suggested list.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
					"name":       map[string]any{"type": "string"},
					"kind":       map[string]any{"type": "string", "enum": statusKindNames()},
					"icon":       map[string]any{"type": "string", "enum": statusIconNames()},
				},
				"required": []string{"project_id", "name", "kind"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "project_id", "name", "kind")
				if err != nil {
					return nil, err
				}
				projectID, name, kind := vals[0], vals[1], vals[2]
				icon := mcptool.OptionalString(args["icon"])
				return s.CreateStatus(ctx, "", projectID, name, StatusKind(kind), StatusIcon(icon))
			},
		},
		{
			Name:        "status_list",
			Description: "List a project's status columns (the board's columns).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
				},
				"required": []string{"project_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				projectID, err := mcptool.RequiredString(args, "project_id")
				if err != nil {
					return nil, err
				}
				return s.ListStatusesByProject(ctx, projectID)
			},
		},
		{
			Name:        "status_rename",
			Description: "Rename a status column and/or change its stage (kind) or icon. Tickets keep their status identity.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":   map[string]any{"type": "string"},
					"name": map[string]any{"type": "string"},
					"kind": map[string]any{"type": "string", "enum": statusKindNames()},
					"icon": map[string]any{"type": "string", "enum": statusIconNames()},
				},
				"required": []string{"id", "name", "kind"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "name", "kind")
				if err != nil {
					return nil, err
				}
				id, name, kind := vals[0], vals[1], vals[2]
				icon := mcptool.OptionalString(args["icon"])
				return s.RenameStatus(ctx, "", id, name, StatusKind(kind), StatusIcon(icon))
			},
		},
		{
			Name:        "status_reorder",
			Description: "Set the display order of a project's status columns.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
					"ids":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
				"required": []string{"project_id", "ids"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				projectID, err := mcptool.RequiredString(args, "project_id")
				if err != nil {
					return nil, err
				}
				ids, err := stringSliceArg(args, "ids")
				if err != nil {
					return nil, err
				}
				if err := s.ReorderStatuses(ctx, "", projectID, ids); err != nil {
					return nil, err
				}
				return map[string]string{"status": "reordered"}, nil
			},
		},
		{
			Name:        "status_delete",
			Description: "Delete an unused status column (one holding no tickets).",
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
				if err := s.DeleteStatus(ctx, "", id); err != nil {
					return nil, err
				}
				return map[string]string{"id": id, "status": "deleted"}, nil
			},
		},
	}
}

func projectIconNames() []string {
	names := make([]string, 0, len(validProjectIcons))
	for icon := range validProjectIcons {
		names = append(names, string(icon))
	}
	sort.Strings(names)
	return names
}

func statusKindNames() []string {
	names := make([]string, len(StatusKinds))
	for i, k := range StatusKinds {
		names[i] = string(k)
	}
	return names
}

func statusIconNames() []string {
	names := make([]string, 0, len(validStatusIcons))
	for icon := range validStatusIcons {
		names = append(names, string(icon))
	}
	sort.Strings(names)
	return names
}

func ticketTypeColorNames() []string {
	all := colors.All()
	names := make([]string, 0, len(all))
	for _, color := range all {
		names = append(names, string(color))
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
			return nil, fmt.Errorf("%w: %s must be a list of ids", apperrs.ErrInvalid, key)
		}
		out = append(out, s)
	}
	return out, nil
}

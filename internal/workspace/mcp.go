package workspace

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools are the project tools that need only this domain; project_get and project_update live in internal/mcp/composite.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{projectListTool(s), projectCreateTool(s), projectDeleteTool(s)}
}

type projectListIn struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace's id (a UUID); every project and memory carries it as workspace_id."`
	mcptool.PageArgs
}

func projectListTool(s *Service) mcptool.Tool {
	return mcptool.New("project_list", "List projects",
		"Lists a workspace's projects in board order, each with its id, name, and prefix (the tag in ticket keys "+
			"such as REF-102). Use it to find a project's id; project_get then returns its statuses, categories, "+
			"ticket types, labels, and repositories, which you need before filing or moving tickets. Returns at most "+
			"100 projects per page.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in projectListIn) (any, error) {
			projects, err := s.List(ctx, in.WorkspaceID)
			if err != nil {
				return nil, err
			}
			return mcptool.Paginate(projects, in.PageArgs), nil
		})
}

type projectCreateIn struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace's id (a UUID) the project belongs to."`
	Name        string `json:"name" jsonschema:"The project's display name, for example Backend."`
	Prefix      string `json:"prefix" jsonschema:"2 to 5 letters, unique in the workspace, that start the project's ticket keys, for example REF for REF-102."`
	Icon        string `json:"icon,omitempty" jsonschema:"A display icon: Box, Rocket, Server, Globe, Database, Layers, Terminal, Shield, Zap, Package, Cpu, or Cloud. Omit for none."`
}

func projectCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("project_create", "Create project",
		"Creates a project, the grouping tickets, repositories, and stacks belong to, with the default status "+
			"columns and ticket types (task, bug, feature). Owners only. Use project_update afterwards to rename it, "+
			"attach repositories, or change its columns, categories, and ticket types. Returns the new project.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in projectCreateIn) (any, error) {
			actorID, err := ownerActor(ctx)
			if err != nil {
				return nil, err
			}
			return s.Create(ctx, actorID, in.WorkspaceID, in.Name, in.Prefix, ProjectIcon(in.Icon))
		})
}

type projectDeleteIn struct {
	ID string `json:"id" jsonschema:"The project's id (a UUID), from project_list."`
}

func projectDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("project_delete", "Delete project",
		"Deletes an empty project for good. Owners only. It is refused while the project still holds tickets, "+
			"repositories, or services; project_get shows those counts under delete_impact, and ticket_update "+
			"(project_id) and project_update (remove_repos) move them out. Returns the deleted project's id.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in projectDeleteIn) (any, error) {
			actorID, err := ownerActor(ctx)
			if err != nil {
				return nil, err
			}
			if err := s.Delete(ctx, actorID, in.ID); err != nil {
				return nil, err
			}
			return mcptool.Gone(in.ID), nil
		})
}

// ownerActor is the caller the owner gate checks; an empty id would read as a trusted adapter and skip the gate.
func ownerActor(ctx context.Context) (string, error) {
	a, ok := identity.ActorFromCtx(ctx)
	if !ok || a.ID == "" {
		return "", fmt.Errorf("%w: changing projects needs a signed-in owner", apperrs.ErrUnauthorized)
	}
	return a.ID, nil
}

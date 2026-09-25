package tenancy

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

type workspaceListIn struct {
	mcptool.PageArgs
}

type workspaceResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// WorkspaceMCPTools lists the caller's workspaces, the ids the project, play, memory, and invitation tools are scoped by.
func WorkspaceMCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{mcptool.New("workspace_list", "List workspaces",
		"Lists the workspaces you belong to, with each one's id, name, and your role in it. Start here when a tool "+
			"needs a workspace_id: project_list, play_list, and the memory and invitation tools are scoped by "+
			"workspace. It returns only workspaces you are a member of.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in workspaceListIn) (any, error) {
			a, ok := identity.ActorFromCtx(ctx)
			if !ok || a.ID == "" {
				return nil, fmt.Errorf("%w: listing workspaces needs a signed-in user", apperrs.ErrUnauthorized)
			}
			ws, err := s.ListForUser(ctx, a.ID)
			if err != nil {
				return nil, err
			}
			out := make([]workspaceResult, 0, len(ws))
			for _, w := range ws {
				role, err := s.MemberRoleName(ctx, w.ID, a.ID)
				if err != nil {
					return nil, err
				}
				out = append(out, workspaceResult{ID: w.ID, Name: w.Name, Role: role})
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})}
}

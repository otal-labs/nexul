package access

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// MCPTools' acting user comes from the identity context the MCP server injects.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{accessGrantListTool(s), accessGrantUpdateTool(s)}
}

type accessGrantListIn struct {
	ResourceType string `json:"resource_type" jsonschema:"doc or play."`
	ResourceID   string `json:"resource_id" jsonschema:"The doc's or play's id."`
	mcptool.PageArgs
}

type accessGrantUpdateIn struct {
	ResourceType string   `json:"resource_type" jsonschema:"doc or play."`
	ResourceIDs  []string `json:"resource_ids" jsonschema:"The docs' or plays' ids; every one gets the same change."`
	UserIDs      []string `json:"user_ids" jsonschema:"The account ids the change applies to."`
	Actions      []string `json:"actions" jsonschema:"Permissions to grant or revoke, each <domain>:<action>, for example [\"docs:read\", \"docs:write\"]. On a play only plays:run."`
	Grant        bool     `json:"grant" jsonschema:"true grants the actions (on a play, lifts the users' exclusion); false revokes them (on a play, excludes the users from running it)."`
}

// grantResult is one user's overwrite on the resource that was asked for.
type grantResult struct {
	UserID string          `json:"user_id"`
	Allow  permissions.Set `json:"allow"`
	Deny   permissions.Set `json:"deny"`
}

func accessGrantListTool(s *Service) mcptool.Tool {
	return mcptool.New("access_grant_list", "List access grants",
		"Lists the per-user permission overwrites on one doc or play: on a doc, who was granted which docs "+
			"actions; on a play, which users are excluded from running it (plays:run in deny). Needs "+
			"permissions:write on the doc, or plays:write on the play. Change them with access_grant_update.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in accessGrantListIn) (any, error) {
			grants, err := listGrants(ctx, s, in)
			if err != nil {
				return nil, err
			}
			out := make([]grantResult, 0, len(grants))
			for _, g := range grants {
				out = append(out, grantResult{UserID: g.UserID, Allow: g.Allow, Deny: g.Deny})
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})
}

func listGrants(ctx context.Context, s *Service, in accessGrantListIn) ([]*Overwrite, error) {
	if in.ResourceType == resourceTypeDoc {
		return s.ListGrants(ctx, actorIDFromCtx(ctx), in.ResourceID)
	}
	if in.ResourceType == resourceTypePlay {
		return s.ListPlayGrants(ctx, actorIDFromCtx(ctx), in.ResourceID)
	}
	return nil, unknownResourceType(in.ResourceType)
}

func accessGrantUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("access_grant_update", "Update access grants",
		"Grants or revokes permission actions for every listed user on every listed doc, or excludes users from "+
			"running plays and lifts that exclusion, in one operation. Only the named actions change: every other "+
			"action a user already holds on the resource stays. Needs permissions:write on each doc, or plays:write "+
			"on each play, and changes nothing unless all of them pass; access_grant_list shows the result.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in accessGrantUpdateIn) (any, error) {
			if err := setGrants(ctx, s, in); err != nil {
				return nil, err
			}
			return in, nil
		})
}

func setGrants(ctx context.Context, s *Service, in accessGrantUpdateIn) error {
	actions, err := parseActions(in.Actions)
	if err != nil {
		return err
	}
	if in.ResourceType == resourceTypeDoc {
		return s.SetGrants(ctx, actorIDFromCtx(ctx), in.ResourceIDs, in.UserIDs, actions, in.Grant)
	}
	if in.ResourceType == resourceTypePlay {
		return s.SetPlayGrants(ctx, actorIDFromCtx(ctx), in.ResourceIDs, in.UserIDs, actions, in.Grant)
	}
	return unknownResourceType(in.ResourceType)
}

func unknownResourceType(t string) error {
	return fmt.Errorf("%w: resource_type %q is not doc or play", apperrs.ErrInvalid, t)
}

func actorIDFromCtx(ctx context.Context) string {
	if a, ok := identity.ActorFromCtx(ctx); ok {
		return a.ID
	}
	return ""
}

package auth

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools reads and removes accounts; account_list and account_update carry workspace access, so they are composite tools.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{accountGetTool(s), accountDeleteTool(s)}
}

type accountGetIn struct {
	ID string `json:"id,omitempty" jsonschema:"The account's id, from account_list. Omit it to get the account this connection acts as."`
}

type accountDeleteIn struct {
	ID string `json:"id" jsonschema:"The account's id, from account_list."`
}

// accountResult is an account without its sign-in identities and onboarding bookkeeping.
type accountResult struct {
	ID          string        `json:"id"`
	Login       string        `json:"login"`
	Name        string        `json:"name"`
	DisplayName string        `json:"display_name,omitempty"`
	AvatarURL   string        `json:"avatar_url,omitempty"`
	Status      AccountStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
}

func toAccountResult(u *User) accountResult {
	status := u.AccountStatus
	if status == "" {
		status = AccountActive
	}
	r := accountResult{
		ID: u.ID, Login: u.Login, Name: u.Name, AvatarURL: u.AvatarURL,
		Status: status, CreatedAt: u.CreatedAt,
	}
	if u.DisplayName != nil {
		r.DisplayName = *u.DisplayName
	}
	return r
}

func accountGetTool(s *Service) mcptool.Tool {
	return mcptool.New("account_get", "Get account",
		"Returns one account: without id, the account this MCP connection acts as, which also confirms Nexul is "+
			"reachable; call it first in a session. With another account's id it needs accounts:read in any "+
			"workspace, the same as seeing everyone in account_list, which lists every account with its workspace "+
			"access. Returns the login, name, the display name the person chose if any, and status (active, "+
			"disabled, or removed).",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in accountGetIn) (any, error) {
			u, err := s.GetAccount(ctx, actorID(ctx), in.ID)
			if err != nil {
				return nil, err
			}
			return toAccountResult(u), nil
		})
}

func accountDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("account_delete", "Remove account",
		"Removes an account: it can no longer sign in, and its credentials and workspace memberships are deleted, "+
			"while everything it authored stays. The account remains listed as removed; account_update with status "+
			"active restores it, without the deleted access. Needs accounts:delete in any workspace, and the last "+
			"active Owner cannot be removed; returns {id, deleted: true}.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in accountDeleteIn) (any, error) {
			if err := s.RemoveAccount(ctx, actorID(ctx), in.ID); err != nil {
				return nil, err
			}
			return mcptool.Gone(in.ID), nil
		})
}

func actorID(ctx context.Context) string {
	if actor, ok := identity.ActorFromCtx(ctx); ok {
		return actor.ID
	}
	return ""
}

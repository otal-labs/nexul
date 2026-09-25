package auth

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools exposes the caller's own account and account administration through the same use-cases as HTTP.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{accountGetTool(s), accountListTool(s), accountUpdateTool(s), accountDeleteTool(s)}
}

type accountGetIn struct {
	ID string `json:"id,omitempty" jsonschema:"The account's id, from account_list. Omit it to get the account this connection acts as."`
}

type accountListIn struct {
	mcptool.PageArgs
}

type accountUpdateIn struct {
	ID     string        `json:"id" jsonschema:"The account's id, from account_list."`
	Status AccountStatus `json:"status" jsonschema:"active or disabled. active reactivates a disabled account and restores a removed one."`
}

type accountDeleteIn struct {
	ID string `json:"id" jsonschema:"The account's id, from account_list."`
}

// accountResult is an account without its provider-side ids and onboarding bookkeeping.
type accountResult struct {
	ID                 string        `json:"id"`
	Login              string        `json:"login"`
	Name               string        `json:"name"`
	DisplayName        string        `json:"display_name,omitempty"`
	Provider           Provider      `json:"provider"`
	AvatarURL          string        `json:"avatar_url,omitempty"`
	Status             AccountStatus `json:"status"`
	CanCreateWorkspace bool          `json:"can_create_workspace"`
	CreatedAt          time.Time     `json:"created_at"`
}

func toAccountResult(u *User) accountResult {
	status := u.AccountStatus
	if status == "" {
		status = AccountActive
	}
	r := accountResult{
		ID: u.ID, Login: u.Login, Name: u.Name, Provider: u.Provider, AvatarURL: u.AvatarURL,
		Status: status, CanCreateWorkspace: u.CanCreateWorkspace, CreatedAt: u.CreatedAt,
	}
	if u.DisplayName != nil {
		r.DisplayName = *u.DisplayName
	}
	return r
}

func accountGetTool(s *Service) mcptool.Tool {
	return mcptool.New("account_get", "Get account",
		"Returns one account: without id, the account this MCP connection acts as, which also confirms Nexul is "+
			"reachable; call it first in a session. With another account's id it needs an instance administrator, "+
			"the same as account_list, which lists every account. Returns the login, name, the display name the "+
			"person chose if any, sign-in provider, status (active, disabled, or removed), and whether the account "+
			"administers the instance.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in accountGetIn) (any, error) {
			u, err := s.GetAccount(ctx, actorID(ctx), in.ID)
			if err != nil {
				return nil, err
			}
			return toAccountResult(u), nil
		})
}

func accountListTool(s *Service) mcptool.Tool {
	return mcptool.New("account_list", "List accounts",
		"Lists every account registered on the instance, with its status (active, disabled, or removed). "+
			"Instance administrators only. Use account_get for one account or your own, and account_update or "+
			"account_delete to change one.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in accountListIn) (any, error) {
			users, err := s.ListAccounts(ctx, actorID(ctx))
			if err != nil {
				return nil, err
			}
			out := make([]accountResult, 0, len(users))
			for _, u := range users {
				out = append(out, toAccountResult(u))
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})
}

func accountUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("account_update", "Update account status",
		"Sets an account's status. disabled blocks sign-in while keeping the account's memberships and "+
			"credentials; active reactivates a disabled account, or restores a removed one without restoring the "+
			"access account_delete took away. Instance administrators only; returns the account with its new "+
			"status, and setting the status it already has changes nothing.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in accountUpdateIn) (any, error) {
			actor := actorID(ctx)
			if err := s.UpdateAccountStatus(ctx, actor, in.ID, in.Status); err != nil {
				return nil, err
			}
			u, err := s.GetAccount(ctx, actor, in.ID)
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
			"active restores it, without the deleted access. Instance administrators only; returns {id, deleted: true}.",
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

package automations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// automationResult is an automation as the model reads it: no token material, no audit fields.
type automationResult struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Kind          Kind            `json:"kind"`
	Enabled       bool            `json:"enabled"`
	Subscriptions []string        `json:"subscriptions"`
	ConfigSchema  json.RawMessage `json:"config_schema"`
	ConfigValues  json.RawMessage `json:"config_values"`
	Scopes        []string        `json:"scopes"`
	TokenRevoked  bool            `json:"token_revoked"`
}

// mintedTokenResult is the only result that carries a token, returned once by the tools that mint one.
type mintedTokenResult struct {
	Automation automationResult `json:"automation"`
	Token      string           `json:"token"`
}

func toAutomationResult(a *Automation) automationResult {
	return automationResult{
		ID: a.ID, Name: a.Name, Description: a.Description, Kind: a.Kind, Enabled: a.Enabled,
		Subscriptions: a.Subscriptions, ConfigSchema: a.ConfigSchema, ConfigValues: a.ConfigValues, Scopes: a.Scopes,
		TokenRevoked: a.TokenRevokedAt != nil,
	}
}

type automationListIn struct {
	ID string `json:"id,omitempty" jsonschema:"An automation's id. Returns that one automation instead of a list."`
	mcptool.PageArgs
}

type automationCreateIn struct {
	Name   string   `json:"name" jsonschema:"The automation's name, for example Close stale tickets. Its code overwrites it at the first dial-in."`
	Scopes []string `json:"scopes" jsonschema:"The permissions its token may use, as domain:action, for example tickets:read or tickets:write. At least one."`
}

type automationUpdateIn struct {
	ID           string         `json:"id" jsonschema:"The automation's id, from automation_list."`
	ConfigValues map[string]any `json:"config_values,omitempty" jsonschema:"The owner-set config as a JSON object, replacing the current values whole and checked by the automation against its config_schema; omit to keep them."`
	Enabled      *bool          `json:"enabled,omitempty" jsonschema:"true delivers events to the automation and runs its worker, false stops both; omit to keep it."`
	RevokeToken  bool           `json:"revoke_token,omitempty" jsonschema:"true revokes the automation's token at once and drops its live connection; automation_token_create mints a new one to restore access."`
}

type automationIDIn struct {
	ID string `json:"id" jsonschema:"The automation's id, from automation_list."`
}

// MCPTools returns the automations tools.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("automation_list", "List automations",
			"Lists every automation, defaults and custom ones alike, with its subscriptions, config schema and "+
				"values, scopes, enabled switch, and whether its token is revoked. With id it returns that one "+
				"automation instead of a list. Tokens never appear here; automation_token_create mints a new one. "+
				"Paged, 50 per page by default. Needs automations:read.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in automationListIn) (any, error) {
				if in.ID != "" {
					a, err := s.Get(ctx, actorIDFromCtx(ctx), in.ID)
					if err != nil {
						return nil, notFoundHint(err)
					}
					return toAutomationResult(a), nil
				}
				list, err := s.List(ctx, actorIDFromCtx(ctx))
				if err != nil {
					return nil, err
				}
				out := make([]automationResult, 0, len(list))
				for i := range list {
					out = append(out, toAutomationResult(&list[i]))
				}
				return mcptool.Paginate(out, in.PageArgs), nil
			}),
		mcptool.New("automation_create", "Create automation",
			"Creates a custom automation shell, disabled, and mints its scoped token. The result carries the "+
				"token this one time only; it is never shown again, so hand it to the automation's code straight "+
				"away, and use automation_token_create to replace a lost one. Enable it with automation_update once "+
				"its code has dialed in. Needs automations:write.",
			mcptool.Hints{Additive: true, Local: true},
			func(ctx context.Context, in automationCreateIn) (any, error) {
				a, token, err := s.Create(ctx, actorIDFromCtx(ctx), in.Name, in.Scopes)
				if err != nil {
					return nil, err
				}
				return mintedTokenResult{Automation: toAutomationResult(a), Token: token}, nil
			}),
		mcptool.New("automation_update", "Update automation",
			"Changes an automation's config values, enables or disables it, or revokes its token. Only the "+
				"fields you pass change, applied in the order config_values, enabled, revoke_token; it stops at the "+
				"first failure and the error says which fields already took effect. Returns the updated automation. "+
				"To rotate the token use automation_token_create, and to remove the automation automation_delete. "+
				"Needs automations:write.",
			mcptool.Hints{Idempotent: true, Local: true},
			func(ctx context.Context, in automationUpdateIn) (any, error) {
				a, err := updateAutomation(ctx, s, in)
				if err != nil {
					return nil, notFoundHint(err)
				}
				return toAutomationResult(a), nil
			}),
		mcptool.New("automation_delete", "Delete automation",
			"Deletes an automation for good and revokes its token, dropping its live connection. To pause one "+
				"instead, use automation_update with enabled false. Needs automations:delete.",
			mcptool.Hints{Idempotent: true, Local: true},
			func(ctx context.Context, in automationIDIn) (any, error) {
				if err := s.Delete(ctx, actorIDFromCtx(ctx), in.ID); err != nil {
					return nil, notFoundHint(err)
				}
				return mcptool.Gone(in.ID), nil
			}),
		mcptool.New("automation_token_create", "Rotate automation token",
			"Mints a new token for an automation, replacing its current or revoked one: the old token stops "+
				"working at once and its live connection drops. The result carries the new token this one time "+
				"only, so hand it to the automation's code straight away. To cut access without a replacement, "+
				"use automation_update with revoke_token instead. Needs automations:write.",
			mcptool.Hints{Local: true},
			func(ctx context.Context, in automationIDIn) (any, error) {
				a, token, err := s.MintToken(ctx, actorIDFromCtx(ctx), in.ID)
				if err != nil {
					return nil, notFoundHint(err)
				}
				return mintedTokenResult{Automation: toAutomationResult(a), Token: token}, nil
			}),
	}
}

// updateAutomation runs one use-case per given field; each commits on its own, so a failure names what already landed.
func updateAutomation(ctx context.Context, s *Service, in automationUpdateIn) (*Automation, error) {
	actor := actorIDFromCtx(ctx)
	var a *Automation
	var applied []string
	partial := func(err error) error {
		if len(applied) == 0 {
			return err
		}
		return fmt.Errorf("%w (already applied: %s)", err, strings.Join(applied, ", "))
	}
	if in.ConfigValues != nil {
		raw, _ := json.Marshal(in.ConfigValues) // re-encoding decoded JSON cannot fail
		var err error
		if a, err = s.UpdateConfigValues(ctx, actor, in.ID, raw); err != nil {
			return nil, err
		}
		applied = append(applied, "config_values")
	}
	if in.Enabled != nil {
		var err error
		if a, err = s.SetEnabled(ctx, actor, in.ID, *in.Enabled); err != nil {
			return nil, partial(err)
		}
		applied = append(applied, "enabled")
	}
	if in.RevokeToken {
		var err error
		if a, err = s.RevokeToken(ctx, actor, in.ID); err != nil {
			return nil, partial(err)
		}
	}
	if a == nil {
		return nil, fmt.Errorf("%w: nothing to change; pass config_values, enabled, or revoke_token", apperrs.ErrInvalid)
	}
	return a, nil
}

func notFoundHint(err error) error {
	if errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("%w; automation_list shows every automation's id", err)
	}
	return err
}

func actorIDFromCtx(ctx context.Context) string {
	if a, ok := identity.ActorFromCtx(ctx); ok {
		return a.ID
	}
	return ""
}

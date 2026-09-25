package pairing

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the computer tools: pairing, the tunnel token, setup runs and confirmations (ADR 0063), and MCP tokens.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		computerListTool(s), computerCreateTool(s), computerPairTool(s), computerDeleteTool(s), computerTunnelTokenGetTool(s),
		computerSetupRunTool(s), computerSetupUpdateTool(s), computerMCPTokenCreateTool(s), computerMCPTokenDeleteTool(s),
	}
}

type computerListIn struct {
	ID string `json:"id,omitempty" jsonschema:"One computer's id, to read it alone with its tunnel's live status. Omit to list every computer you own."`
	mcptool.PageArgs
}

type computerCreateIn struct {
	Name string `json:"name" jsonschema:"The computer's name, for example Onik Laptop; its tunnel hostname is made from it."`
	Port int    `json:"port,omitzero" jsonschema:"The local port T3 Code serves on, 1 to 65535. Defaults to 3773."`
}

type computerPairIn struct {
	Token     string `json:"token" jsonschema:"The one-time token t3 pair prints on the computer."`
	ID        string `json:"id,omitempty" jsonschema:"One of your computers to pair or re-pair, for example the one computer_create returned. Omit it to pair a new computer by name and server_url."`
	Name      string `json:"name,omitempty" jsonschema:"The computer's name, for example Onik Laptop. Required without id; with id it renames the computer, and omitting it keeps the name."`
	ServerURL string `json:"server_url,omitempty" jsonschema:"The T3 Code server URL this server reaches, for example https://vps.example.com:3773. Required without id; with id it moves a computer paired by URL, and omitting it keeps the address. A tunnel computer always pairs over its own hostname."`
}

type computerDeleteIn struct {
	ID string `json:"id" jsonschema:"The computer's id, from computer_list."`
}

type computerIDIn struct {
	ComputerID string `json:"computer_id" jsonschema:"The computer's id, from computer_list."`
}

type computerSetupRunIn struct {
	ComputerID string            `json:"computer_id" jsonschema:"The paired computer's id, from computer_list."`
	Provider   string            `json:"provider,omitempty" jsonschema:"One provider's driver kind to run alone, for example codex, after its turn failed. Omit to run every provider the harness lists."`
	Model      string            `json:"model,omitempty" jsonschema:"With provider only: the model slug its turn runs on, as the harness lists it. Omit for the provider's own default."`
	Models     map[string]string `json:"models,omitempty" jsonschema:"Without provider only: the model slug each provider's turn runs on, keyed by driver kind, for example {\"codex\": \"gpt-5-mini\"}. A provider left out runs on its own default."`
}

type computerSetupUpdateIn struct {
	ComputerID string   `json:"computer_id" jsonschema:"The paired computer's id, from computer_list."`
	Provider   string   `json:"provider,omitempty" jsonschema:"A provider's driver kind, for example claude, codex, or opencode, to change that provider's confirmation. Omit to change the computer's overall confirmation."`
	Confirmed  bool     `json:"confirmed" jsonschema:"true confirms, false withdraws the confirmation."`
	Skills     []string `json:"skills,omitempty" jsonschema:"With provider and confirmed true, required: the skill names this session's harness reported as discovered, for example [\"tdd\", \"nexul-memory\"]."`
}

// computerResult is a computer as an agent reads it: no bearer token, and its setup and MCP token only on a list.
type computerResult struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Kind             harness.Kind    `json:"kind"`
	ServerURL        string          `json:"server_url"`
	Paired           bool            `json:"paired"`
	SessionExpiresAt *time.Time      `json:"session_expires_at,omitempty"`
	HarnessVersion   string          `json:"harness_version,omitempty"`
	Tunnel           *ComputerTunnel `json:"tunnel,omitempty"`
	TunnelStatus     *TunnelStatus   `json:"tunnel_status,omitempty"`
	Setup            *Setup          `json:"setup,omitempty"`
	MCPToken         *MCPToken       `json:"mcp_token,omitempty"`
}

func toComputerResult(c Computer) computerResult {
	r := computerResult{ID: c.ID, Name: c.Name, Kind: c.Kind, ServerURL: c.ServerURL, Paired: c.Paired(), HarnessVersion: c.HarnessVersion, Tunnel: c.Tunnel}
	if r.Paired {
		r.SessionExpiresAt = &c.TokenExpiresAt
	}
	return r
}

func computerListTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_list", "List paired computers",
		"Lists the computers you own, each with its tunnel, its setup confirmation (overall and per provider, plus each "+
			"provider's newest setup turn), and its MCP token's metadata, never the token itself. Pass id to read one "+
			"computer alone with its tunnel's live status: tunnel is Cloudflare's connector state (inactive, healthy, "+
			"degraded, or down) and harness_reachable says whether T3 Code answers through the hostname; pairing can "+
			"continue with computer_pair once both pass. A computer with paired false has a tunnel but no harness "+
			"session yet, and one without setup.confirmed_at needs computer_setup_run before agent work can use it.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in computerListIn) (any, error) {
			userID := mcpActorID(ctx)
			computers, err := s.ListComputers(ctx, userID)
			if err != nil {
				return nil, err
			}
			if in.ID != "" {
				computers = slices.DeleteFunc(computers, func(c Computer) bool { return c.ID != in.ID })
			}
			if in.ID != "" && len(computers) == 0 {
				return nil, fmt.Errorf("%w: computer %s is not one of yours; computer_list without id lists them", apperrs.ErrNotFound, in.ID)
			}
			out := make([]computerResult, 0, len(computers))
			for _, c := range computers {
				r, err := listedComputer(ctx, s, userID, c, in.ID != "")
				if err != nil {
					return nil, err
				}
				out = append(out, r)
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})
}

// listedComputer adds the setup and MCP token reads, and the live tunnel status only when one computer was asked for.
func listedComputer(ctx context.Context, s *Service, userID string, c Computer, live bool) (computerResult, error) {
	r := toComputerResult(c)
	setup, err := s.GetSetup(ctx, userID, c.ID)
	if err != nil {
		return r, err
	}
	r.Setup = &setup
	if r.MCPToken, err = s.GetMCPToken(ctx, userID, c.ID); err != nil {
		return r, err
	}
	if !live || c.Tunnel == nil {
		return r, nil
	}
	status, err := s.ComputerTunnelStatus(ctx, userID, c.ID)
	if err != nil {
		return r, err
	}
	r.TunnelStatus = &status
	return r, nil
}

func computerCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_create", "Create computer",
		"Starts pairing a computer through its own tunnel on the instance's Cloudflare: creates the computer, its tunnel, "+
			"and a hostname closed to everything but this server. Next, install cloudflared on the computer with the "+
			"token computer_tunnel_token_get reveals, wait for computer_list with this id to report the tunnel healthy "+
			"and the harness reachable, then call computer_pair. For a machine this server can already reach by URL, "+
			"skip this and call computer_pair with name and server_url. Returns the new, still unpaired computer.",
		mcptool.Hints{Additive: true},
		func(ctx context.Context, in computerCreateIn) (any, error) {
			port := in.Port
			if port == 0 {
				port = DefaultT3CodePort
			}
			c, err := s.CreateComputerTunnel(ctx, mcpActorID(ctx), harness.KindT3Code, in.Name, port)
			if err != nil {
				return nil, err
			}
			return toComputerResult(*c), nil
		})
}

func computerPairTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_pair", "Pair computer",
		"Pairs T3 Code on a computer with the one-time token t3 pair prints there, giving Nexul a harness session on it. "+
			"Pass id to pair a computer computer_create made, over its tunnel hostname, or to re-pair one of your "+
			"computers after its session expired; with id, name renames it and server_url moves a computer paired "+
			"by URL, and an omitted one keeps its value. Without id, name and server_url pair a new machine this "+
			"server can already reach. Returns the paired computer; run computer_setup_run next so agent work can use it.",
		mcptool.Hints{},
		func(ctx context.Context, in computerPairIn) (any, error) {
			c, err := pairComputer(ctx, s, in)
			if err != nil {
				return nil, err
			}
			return toComputerResult(*c), nil
		})
}

func pairComputer(ctx context.Context, s *Service, in computerPairIn) (*Computer, error) {
	userID := mcpActorID(ctx)
	if in.ID == "" {
		return s.Pair(ctx, userID, harness.KindT3Code, in.Name, in.ServerURL, in.Token)
	}
	current, err := s.ownComputer(ctx, userID, in.ID)
	if err != nil {
		return nil, err
	}
	return s.Repair(ctx, userID, current.ID, cmp.Or(in.Name, current.Name), cmp.Or(in.ServerURL, current.address()), in.Token)
}

func computerDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_delete", "Delete computer",
		"Removes one of your computers: revokes its MCP token, tears down its tunnel, DNS record, and Access app on "+
			"Cloudflare, then deletes it. Agent work can no longer run there; pair it again with computer_create or "+
			"computer_pair. A failed Cloudflare teardown keeps the computer, so calling this again retries it. "+
			"Returns {id, deleted: true}.",
		mcptool.Hints{Idempotent: true},
		func(ctx context.Context, in computerDeleteIn) (any, error) {
			if err := s.DeleteComputer(ctx, mcpActorID(ctx), in.ID); err != nil {
				return nil, err
			}
			return mcptool.Gone(in.ID), nil
		})
}

func computerTunnelTokenGetTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_tunnel_token_get", "Reveal tunnel token",
		"Reveals the secret connector token one of your computers installs cloudflared with (cloudflared service "+
			"install <token>); this tool exists only to hand it over, so show it to the computer's owner and nobody "+
			"else. Only a computer made by computer_create has one; a computer paired by URL has no tunnel. "+
			"computer_list shows the tunnel itself without its token.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in computerIDIn) (any, error) {
			token, err := s.ComputerTunnelToken(ctx, mcpActorID(ctx), in.ComputerID)
			if err != nil {
				return nil, err
			}
			return map[string]string{"computer_id": in.ComputerID, "token": token}, nil
		})
}

func computerSetupRunTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_setup_run", "Run computer setup",
		"Starts setup on one of your paired computers: Nexul mints or reuses the computer's MCP token, then runs one "+
			"setup turn per provider its harness lists, one after another, each connecting Nexul's MCP server, "+
			"installing the default skills and nexul-memory, and confirming through computer_setup_update. Pass "+
			"provider to run only that provider's turn again after it failed; on a confirmed provider it re-verifies "+
			"without undoing anything. Returns at once with the run and its providers; computer_list shows each "+
			"provider's turn state as it progresses. Only one setup runs on a computer at a time.",
		mcptool.Hints{},
		func(ctx context.Context, in computerSetupRunIn) (any, error) {
			if in.Provider == "" && in.Model != "" {
				return nil, fmt.Errorf("%w: model applies to one provider; pass provider too, or models to pick per provider", apperrs.ErrInvalid)
			}
			if in.Provider != "" && len(in.Models) > 0 {
				return nil, fmt.Errorf("%w: models applies to a run of every provider; with provider pass model instead", apperrs.ErrInvalid)
			}
			if in.Provider == "" {
				return s.StartSetup(ctx, mcpActorID(ctx), in.ComputerID, in.Models)
			}
			return s.RetrySetupProvider(ctx, mcpActorID(ctx), in.ComputerID, in.Provider, in.Model)
		})
}

func computerSetupUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_setup_update", "Update setup confirmation",
		"Confirms or withdraws a setup confirmation on one of your paired computers; agent work runs there only while "+
			"both the computer's overall confirmation and its provider's are set. With provider and confirmed true, "+
			"it records that Nexul's MCP server is connected to that provider and its harness reports the given "+
			"skills; without provider it confirms the computer overall, the last step of its setup. Confirmed false "+
			"withdraws the one named, and withdrawing the overall confirmation also revokes the computer's MCP token. "+
			"Only this tool writes a confirmation, and repeating a call changes nothing more; returns the computer's "+
			"setup, which computer_list also shows.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in computerSetupUpdateIn) (any, error) {
			if len(in.Skills) > 0 && (in.Provider == "" || !in.Confirmed) {
				return nil, fmt.Errorf("%w: skills go only with provider and confirmed true", apperrs.ErrInvalid)
			}
			return updateSetup(ctx, s, mcpActorID(ctx), in)
		})
}

func updateSetup(ctx context.Context, s *Service, userID string, in computerSetupUpdateIn) (Setup, error) {
	if in.Provider == "" && in.Confirmed {
		return s.ConfirmSetup(ctx, userID, in.ComputerID)
	}
	if in.Provider == "" {
		return s.UnconfirmSetup(ctx, userID, in.ComputerID)
	}
	if in.Confirmed {
		return s.ConfirmProviderSetup(ctx, userID, in.ComputerID, in.Provider, in.Skills)
	}
	return s.UnconfirmProviderSetup(ctx, userID, in.ComputerID, in.Provider)
}

func computerMCPTokenCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_mcp_token_create", "Create computer MCP token",
		"Mints one of your paired computers its own personal access token, \"Nexul MCP on <computer>\", for its "+
			"providers' MCP configs, revoking the one it had. The token is returned only in this response and is "+
			"hidden in saved transcripts, so show it only to the computer's owner; computer_list shows its metadata "+
			"afterwards. computer_setup_run mints one on its own, so call this only to rotate the token by hand.",
		mcptool.Hints{Local: true},
		func(ctx context.Context, in computerIDIn) (any, error) {
			return s.MintMCPToken(ctx, mcpActorID(ctx), in.ComputerID)
		})
}

func computerMCPTokenDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_mcp_token_delete", "Delete computer MCP token",
		"Revokes one of your paired computers' MCP token, so its providers lose Nexul's MCP server until "+
			"computer_mcp_token_create or computer_setup_run mints a new one. Returns the revoked token's id. "+
			"A computer without a token is not found; computer_list shows which computers have one.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in computerIDIn) (any, error) {
			userID := mcpActorID(ctx)
			token, err := s.GetMCPToken(ctx, userID, in.ComputerID)
			if err != nil {
				return nil, err
			}
			if token == nil {
				return nil, fmt.Errorf("%w: computer %s has no MCP token; computer_mcp_token_create mints one", apperrs.ErrNotFound, in.ComputerID)
			}
			if err := s.RevokeMCPToken(ctx, userID, in.ComputerID); err != nil {
				return nil, err
			}
			return mcptool.Gone(token.ID), nil
		})
}

func mcpActorID(ctx context.Context) string {
	if a, ok := identity.ActorFromCtx(ctx); ok {
		return a.ID
	}
	return ""
}

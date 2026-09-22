package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/mentions"
	"github.com/otal-labs/nexul/internal/platform/eventbus/deadletter"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/repository"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/topology"
	"github.com/otal-labs/nexul/internal/workspace"
)

// RegistryOptions wires every domain use-case layer the MCP adapter exposes (ADR 0019); each entry is optional.
type RegistryOptions struct {
	Docs          *docs.Service
	Memories      *memories.Service
	Tickets       *tickets.Service
	Topology      *topology.Service
	Deploy        *deploy.Service
	Reviews       *codereview.Service
	Workspace     *workspace.Service
	Notifications *workspace.NotificationService
	Git           gitprovider.GitProvider
	Repository    repository.Scanner
	Runner        *runner.Service
	DNS           *dns.Service
	Automations   *automations.Service
	Access        *access.Service
	Auth          *auth.Service
	Invitations   *tenancy.InvitationService
	Mentions      *mentions.Service
	Chat          *chat.Service
	Plays         *plays.Service
	PlayRuns      *plays.Runner
	DeadLetter    deadletter.Storer
	Publisher     Publisher
	Logger        *slog.Logger
	// Actor resolves the acting user; when nil, tool calls carry no identity and permission checks deny.
	Actor func(context.Context) identity.Actor
}

// Publisher is the slice of the event bus the adapter needs; replay reuses the dead letter's ID.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
	PublishWithID(ctx context.Context, id, topic string, payload any) error
}

// New assembles the default MCP server: every domain's MCPTools, day-1 search/list/replay tools, and resources.
func New(opts RegistryOptions) *Server {
	s := &Server{}
	registerDocsTools(s, opts.Docs)
	registerTicketsTools(s, opts.Tickets)
	if opts.Topology != nil {
		s.tools = append(s.tools, topology.MCPTools(opts.Topology)...)
		s.resources = append(s.resources, topologyResource(opts.Topology))
	}
	registerDomainTools(s, opts)
	if opts.DeadLetter != nil && opts.Publisher != nil {
		s.tools = append(s.tools, listDeadLettersTool(opts.DeadLetter), replayDeadLetterTool(opts.DeadLetter, opts.Publisher))
	}
	s.actor = opts.Actor
	s.prompts = defaultPrompts()
	return s
}

func registerDocsTools(s *Server, svc *docs.Service) {
	if svc == nil {
		return
	}
	s.tools = append(s.tools, docs.MCPTools(svc)...)
	s.tools = append(s.tools, searchDocsTool(svc))
	s.templates = append(s.templates, docResource(svc))
}

func registerTicketsTools(s *Server, svc *tickets.Service) {
	if svc == nil {
		return
	}
	s.tools = append(s.tools, tickets.MCPTools(svc)...)
	s.tools = append(s.tools, searchTicketsTool(svc))
	s.templates = append(s.templates, ticketResource(svc))
}

// registerDomainTools wires the domains that contribute tools only (no resources).
func registerDomainTools(s *Server, opts RegistryOptions) {
	if opts.Deploy != nil {
		s.tools = append(s.tools, deploy.MCPTools(opts.Deploy)...)
	}
	if opts.Runner != nil {
		s.tools = append(s.tools, runner.MCPTools(opts.Runner)...)
	}
	if opts.Reviews != nil {
		s.tools = append(s.tools, codereview.MCPTools(opts.Reviews)...)
	}
	if opts.Workspace != nil {
		s.tools = append(s.tools, workspace.MCPTools(opts.Workspace)...)
	}
	if opts.Notifications != nil {
		s.tools = append(s.tools, workspace.NotificationMCPTools(opts.Notifications)...)
	}
	if opts.Git != nil {
		s.tools = append(s.tools, gitprovider.MCPTools(opts.Git)...)
	}
	if opts.Repository != nil {
		s.tools = append(s.tools, repository.MCPTools(opts.Repository)...)
	}
	if opts.DNS != nil {
		s.tools = append(s.tools, dns.MCPTools(opts.DNS)...)
	}
	if opts.Automations != nil {
		s.tools = append(s.tools, automations.MCPTools(opts.Automations)...)
	}
	if opts.Access != nil {
		s.tools = append(s.tools, access.MCPTools(opts.Access)...)
	}
	registerIdentityTools(s, opts)
	registerPlaysTools(s, opts)
	if opts.Memories != nil {
		s.tools = append(s.tools, memories.MCPTools(opts.Memories)...)
	}
}

func registerIdentityTools(s *Server, opts RegistryOptions) {
	if opts.Auth != nil {
		s.tools = append(s.tools, auth.MCPTools(opts.Auth)...)
	}
	if opts.Invitations != nil {
		s.tools = append(s.tools, tenancy.MCPTools(opts.Invitations)...)
	}
	if opts.Mentions != nil {
		s.tools = append(s.tools, mentions.MCPTools(opts.Mentions)...)
	}
	if opts.Chat != nil {
		s.tools = append(s.tools, chat.MCPTools(opts.Chat)...)
	}
}

func registerPlaysTools(s *Server, opts RegistryOptions) {
	if opts.Plays != nil {
		s.tools = append(s.tools, plays.MCPTools(opts.Plays)...)
	}
	if opts.PlayRuns != nil {
		s.tools = append(s.tools, plays.RunMCPTools(opts.PlayRuns)...)
	}
}

// searchDocsTool is the ws-08 day-1 tool (AC list) over the docs use-case.
func searchDocsTool(s *docs.Service) Tool {
	return Tool{
		Name:        "search_docs",
		Description: "Full-text search over doc titles and bodies.",
		InputSchema: objectSchema(map[string]any{
			"query": map[string]any{"type": "string"},
			"limit": map[string]any{"type": "integer"},
		}, "query"),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			q, err := mcptool.RequiredString(args, "query")
			if err != nil {
				return nil, err
			}
			return s.Search(ctx, q, intArg(args["limit"]))
		},
	}
}

func searchTicketsTool(s *tickets.Service) Tool {
	return Tool{
		Name:        "search_tickets",
		Description: "Full-text search over ticket titles and bodies.",
		InputSchema: objectSchema(map[string]any{
			"query": map[string]any{"type": "string"},
			"limit": map[string]any{"type": "integer"},
		}, "query"),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			q, err := mcptool.RequiredString(args, "query")
			if err != nil {
				return nil, err
			}
			return s.Search(ctx, q, intArg(args["limit"]))
		},
	}
}

func listDeadLettersTool(store deadletter.Storer) Tool {
	return Tool{
		Name:        "list_dead_letters",
		Description: "List events that exhausted retries or failed permanently.",
		InputSchema: objectSchema(map[string]any{
			"limit":  map[string]any{"type": "integer"},
			"offset": map[string]any{"type": "integer"},
		}),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			limit := intArg(args["limit"])
			if limit < 1 {
				limit = 50
			}
			return store.List(ctx, limit, intArg(args["offset"]))
		},
	}
}

func replayDeadLetterTool(store deadletter.Storer, p Publisher) Tool {
	return Tool{
		Name:        "replay_dead_letter",
		Description: "Republish a dead letter to its original topic and remove it from the store.",
		InputSchema: objectSchema(map[string]any{
			"id": map[string]any{"type": "string"},
		}, "id"),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			id, err := mcptool.RequiredString(args, "id")
			if err != nil {
				return nil, err
			}
			if err := deadletter.Replay(ctx, store, p, id); err != nil {
				return nil, err
			}
			return map[string]string{"id": id, "status": "replayed"}, nil
		},
	}
}

func docResource(s *docs.Service) ResourceTemplate {
	return ResourceTemplate{
		URITemplate: "docs://{id}",
		Name:        "Doc",
		Description: "A document's full text.",
		MIMEType:    "text/markdown",
		Read: func(ctx context.Context, vars map[string]string) (string, error) {
			md, err := s.ExportMarkdown(ctx, vars["id"])
			if err != nil {
				return "", err
			}
			d, err := s.Get(ctx, vars["id"])
			if err != nil {
				return "", err
			}
			return "# " + d.Title + "\n\n" + md, nil
		},
	}
}

func ticketResource(s *tickets.Service) ResourceTemplate {
	return ResourceTemplate{
		URITemplate: "tickets://{id}",
		Name:        "Ticket",
		Description: "A ticket's full text.",
		MIMEType:    "text/markdown",
		Read: func(ctx context.Context, vars map[string]string) (string, error) {
			t, err := s.Get(ctx, vars["id"])
			if err != nil {
				return "", err
			}
			return "# " + t.Title + "\n\nstatus: " + string(t.Status) + "\n\n" + t.Body, nil
		},
	}
}

func topologyResource(s *topology.Service) Resource {
	return Resource{
		URI:         "topology://current",
		Name:        "Current topology",
		Description: "The topology canvas for the default environment.",
		MIMEType:    "application/json",
		Read: func(ctx context.Context) (string, error) {
			c, err := s.Get(ctx, topology.DefaultEnvironment)
			if err != nil {
				return "", err
			}
			b, err := json.Marshal(c)
			if err != nil {
				return "", fmt.Errorf("marshal topology: %w", err)
			}
			return string(b), nil
		},
	}
}

func defaultPrompts() []Prompt {
	return []Prompt{
		{
			Name:        "create_ticket_from_doc",
			Description: "Turn a doc into an isolated, scoped ticket.",
			Arguments:   []PromptArgument{{Name: "doc_id", Description: "The source doc id", Required: true}},
			Messages: func(args map[string]string) []PromptMessage {
				text := interpolate("Read doc {{doc_id}}, then create one ticket scoped to the work it describes: propose a title, a body summarizing the task, and keep the ticket limited to that doc.", args)
				return []PromptMessage{{Role: "user", Content: text}}
			},
		},
		{
			Name:        "deploy_stack",
			Description: "Deploy a stack and watch its containers come up.",
			Arguments:   []PromptArgument{{Name: "stack_id", Description: "The stack id (stack_list/stack_get)", Required: true}},
			Messages: func(args map[string]string) []PromptMessage {
				text := interpolate("Deploy stack {{stack_id}}: call stack_deploy (build from its ref, or redeploy its last image), then poll deploy_get and service_list until every container is healthy or the deploy fails.", args)
				return []PromptMessage{{Role: "user", Content: text}}
			},
		},
		{
			Name:        "investigate_failure",
			Description: "Investigate a failed deploy end to end.",
			Arguments:   []PromptArgument{{Name: "deploy_id", Description: "The failed deploy id", Required: true}},
			Messages: func(args map[string]string) []PromptMessage {
				text := interpolate("Investigate failed deploy {{deploy_id}}: read it with deploy_get and its output with deploy_log, list the stack's containers with service_list, check the topology, list related dead letters, and propose a fix as a new ticket.", args)
				return []PromptMessage{{Role: "user", Content: text}}
			},
		},
		{
			Name:        "ship_repository",
			Description: "Take a repository from zero to a reachable, deployed stack: the project wizard's own steps, in order.",
			Arguments: []PromptArgument{
				{Name: "owner", Description: "Repository owner", Required: true},
				{Name: "name", Description: "Repository name", Required: true},
				{Name: "project_id", Description: "Workspace project to create the stack in", Required: true},
				{Name: "machine", Description: "Machine to deploy the stack on", Required: true},
				{Name: "hostname", Description: "Hostname to expose the stack at, if it should be reachable", Required: false},
			},
			Messages: func(args map[string]string) []PromptMessage {
				text := interpolate(
					"Ship {{owner}}/{{name}} to project {{project_id}} on machine {{machine}}: "+
						"call repository_list to confirm the repository is visible, then repository_scan {owner: \"{{owner}}\", name: \"{{name}}\"} "+
						"to get its candidates and default branch. Pick the best candidate (compose over a standalone Dockerfile when both exist) "+
						"and call stack_create with project_id \"{{project_id}}\", machine \"{{machine}}\", that candidate, "+
						"build_source set to this repo and branch, and deploy: true. Poll deploy_get and service_list until every "+
						"container is healthy. If a hostname argument was given, once healthy call exposure_create to route it to the "+
						"reachable service's container and port from the scan result.",
					args)
				return []PromptMessage{{Role: "user", Content: text}}
			},
		},
	}
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func intArg(v any) int {
	if f, ok := v.(float64); ok && f > 0 {
		return int(f)
	}
	return 0
}

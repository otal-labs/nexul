package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/docs/richtext"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/mcp/composite"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/mentions"
	"github.com/otal-labs/nexul/internal/pairing"
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

// RegistryOptions wires every domain use-case layer the MCP adapter exposes (ADR 0019).
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
	ChangeContext gitprovider.ChangeContextReader
	Repository    repository.Scanner
	Runner        *runner.Service
	DNS           *dns.Service
	Automations   *automations.Service
	Access        *access.Service
	Auth          *auth.Service
	Invitations   *tenancy.InvitationService
	Workspaces    *tenancy.Service
	Mentions      *mentions.Service
	Chat          *chat.Service
	Plays         *plays.Service
	PlayRuns      *plays.Runner
	Pairing       *pairing.Service
	DeadLetter    deadletter.Storer
	Publisher     deadletter.Publisher
	// InstanceAdmin gates the dead-letter tools, which read and replay every domain's failed events.
	InstanceAdmin identity.InstanceAdmin
	// InstanceURL is the configured public URL; a browser request from any other origin is refused.
	InstanceURL func(context.Context) (string, error)
	Logger      *slog.Logger
}

// New assembles the MCP endpoint: every domain's tools, the resources and prompts, behind the SDK's stateless handler.
// Mount it behind authentication; tools act as the identity the request context carries.
func New(opts RegistryOptions) http.Handler {
	return server{
		tools:        registryTools(opts),
		resources:    []resource{docResource(opts.Docs), ticketResource(opts.Tickets), topologyResource(opts.Topology)},
		prompts:      workflowPrompts(),
		instructions: instructions,
		logger:       opts.Logger,
	}.handler(opts.InstanceURL)
}

// registryTools lists the tools in a fixed order, so tools/list is byte-stable across processes.
func registryTools(opts RegistryOptions) []mcptool.Tool {
	return slices.Concat(
		docs.MCPTools(opts.Docs),
		memories.MCPTools(opts.Memories),
		composite.TicketTools(opts.Tickets, opts.Workspace, opts.Reviews),
		tickets.MCPTools(opts.Tickets),
		composite.ProjectTools(opts.Workspace, opts.Tickets),
		workspace.MCPTools(opts.Workspace),
		workspace.NotificationMCPTools(opts.Notifications),
		topology.MCPTools(opts.Topology),
		deploy.MCPTools(opts.Deploy),
		runner.MCPTools(opts.Runner),
		gitprovider.MCPTools(opts.Git, opts.ChangeContext),
		repository.MCPTools(opts.Repository),
		dns.MCPTools(opts.DNS),
		automations.MCPTools(opts.Automations),
		access.MCPTools(opts.Access),
		auth.MCPTools(opts.Auth),
		tenancy.WorkspaceMCPTools(opts.Workspaces),
		tenancy.MCPTools(opts.Invitations),
		mentions.MCPTools(opts.Mentions),
		chat.MCPTools(opts.Chat),
		pairing.MCPTools(opts.Pairing),
		plays.MCPTools(opts.Plays),
		plays.RunMCPTools(opts.PlayRuns),
		deadLetterTools(opts.DeadLetter, opts.Publisher, opts.InstanceAdmin),
	)
}

func docResource(s *docs.Service) resource {
	return resource{
		uri: "docs://{id}", name: "doc", title: "Doc", mimeType: "text/markdown",
		description: "A doc's title and full body as markdown, the same content doc_get returns.",
		read: func(ctx context.Context, id string) (string, error) {
			d, err := s.Get(ctx, id)
			if err != nil {
				return "", err
			}
			md, err := richtext.ToMarkdown(d.Body)
			if err != nil {
				return "", fmt.Errorf("render doc %s: %w", id, err)
			}
			return "# " + d.Title + "\n\n" + md, nil
		},
	}
}

func ticketResource(s *tickets.Service) resource {
	return resource{
		uri: "tickets://{id}", name: "ticket", title: "Ticket", mimeType: "text/markdown",
		description: "A ticket's title, status, and body as markdown, by id or key (REF-102); ticket_get returns the full record.",
		read: func(ctx context.Context, id string) (string, error) {
			t, err := s.Resolve(ctx, id)
			if err != nil {
				return "", err
			}
			return "# " + t.Title + "\n\nstatus: " + string(t.Status) + "\n\n" + t.Body, nil
		},
	}
}

func topologyResource(s *topology.Service) resource {
	return resource{
		uri: "topology://current", name: "topology", title: "Current topology", mimeType: "application/json",
		description: "The topology canvas of the default environment, the same content topology_get returns.",
		read: func(ctx context.Context, _ string) (string, error) {
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

package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/botwebhook"
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
	"github.com/otal-labs/nexul/internal/platform/jsonx"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/repository"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/templates"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/topology"
	"github.com/otal-labs/nexul/internal/workspace"
)

// RegistryOptions wires every domain use-case layer the MCP adapter exposes (ADR 0019).
type RegistryOptions struct {
	Docs          *docs.Service
	Attachments   *attachments.Service
	Memories      *memories.Service
	Templates     *templates.Service
	Tickets       *tickets.Service
	Topology      *topology.Service
	Deploy        *deploy.Service
	Reviews       *codereview.Service
	Workspace     *workspace.Service
	Notifications *workspace.NotificationService
	Git           gitprovider.GitProvider
	ChangeContext gitprovider.ChangeContextReader
	GitGate       gitprovider.Gate
	Repository    repository.Scanner
	// RepositoryInstallations answers repository_list's installations flag.
	RepositoryInstallations repository.InstallationLister
	RepositoryGate          repository.Gate
	Runner                  *runner.Service
	// Hosts adapts each host kind (runner, automations host) to the shared host_create and host_delete tools.
	Hosts       map[string]composite.HostKind
	DNS         *dns.Service
	Automations *automations.Service
	Access      *access.Service
	Auth        *auth.Service
	Invitations *tenancy.InvitationService
	Workspaces  *tenancy.Service
	Roles       *roles.Service
	Mentions    *mentions.Service
	Chat        *chat.Service
	Botwebhooks *botwebhook.Service
	Plays       *plays.Service
	PlayRuns    *plays.Runner
	Pairing     *pairing.Service
	DeadLetter  DeadLetterStore
	Publisher   deadletter.Publisher
	// InstanceURL is the configured public URL; a browser request from any other origin is refused.
	InstanceURL func(context.Context) (string, error)
	// Audit records a call to a tool that changes state.
	Audit  func(ctx context.Context, tool string)
	Logger *slog.Logger
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
		memo:         accessMemo(opts.Access),
		audit:        opts.Audit,
	}.handler(opts.InstanceURL)
}

// accessMemo starts a fresh access memo per request; with no access service wired there is nothing to remember.
func accessMemo(a *access.Service) func(context.Context) context.Context {
	if a == nil {
		return nil
	}
	return func(ctx context.Context) context.Context {
		return access.WithMemo(ctx, a.NewMemo())
	}
}

// registryTools lists the tools in a fixed order, so tools/list is byte-stable across processes.
func registryTools(opts RegistryOptions) []mcptool.Tool {
	return slices.Concat(
		docs.MCPTools(opts.Docs),
		composite.AttachmentTools(opts.Attachments, opts.Tickets),
		memories.MCPTools(opts.Memories),
		templates.MCPTools(opts.Templates),
		composite.TicketTools(opts.Tickets, opts.Workspace, opts.Reviews),
		tickets.MCPTools(opts.Tickets),
		composite.ProjectTools(opts.Workspace, opts.Tickets, opts.Docs),
		workspace.MCPTools(opts.Workspace),
		workspace.NotificationMCPTools(opts.Notifications),
		topology.MCPTools(opts.Topology),
		deploy.MCPTools(opts.Deploy),
		runner.MCPTools(opts.Runner),
		composite.HostTools(opts.Hosts),
		gitprovider.MCPTools(opts.Git, opts.ChangeContext, opts.GitGate),
		repository.MCPTools(opts.Repository, opts.RepositoryInstallations, opts.RepositoryGate),
		dns.MCPTools(opts.DNS),
		automations.MCPTools(opts.Automations),
		access.MCPTools(opts.Access),
		auth.MCPTools(opts.Auth),
		composite.AccountTools(opts.Auth, opts.Workspaces),
		composite.WorkspaceTools(opts.Workspaces, opts.Roles),
		roles.MCPTools(opts.Roles),
		tenancy.MCPTools(opts.Invitations),
		mentions.MCPTools(opts.Mentions),
		chat.MCPTools(opts.Chat),
		botwebhook.MCPTools(opts.Botwebhooks),
		pairing.MCPTools(opts.Pairing),
		plays.MCPTools(opts.Plays),
		plays.RunMCPTools(opts.PlayRuns),
		deadLetterTools(opts.DeadLetter, opts.Publisher, opts.Access),
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
		description: "A ticket's title, status, and body as markdown, by id or key (REF-102); a key that two of your workspaces share needs the id instead, and ticket_get returns the full record.",
		read: func(ctx context.Context, id string) (string, error) {
			t, err := s.Resolve(ctx, "", id)
			if err != nil {
				return "", err
			}
			md, err := richtext.ToMarkdown(t.Body)
			if err != nil {
				return "", fmt.Errorf("render ticket %s: %w", id, err)
			}
			return "# " + t.Title + "\n\nstatus: " + string(t.Status) + "\n\n" + md, nil
		},
	}
}

func topologyResource(s *topology.Service) resource {
	return resource{
		uri: "topology://{id}", name: "topology", title: "Topology", mimeType: "application/json",
		description: "A workspace's topology canvas, by the workspace's id, the same content topology_get returns.",
		read: func(ctx context.Context, workspaceID string) (string, error) {
			c, err := s.Get(ctx, workspaceID)
			if err != nil {
				return "", err
			}
			b, err := jsonx.Marshal(c)
			if err != nil {
				return "", fmt.Errorf("marshal topology: %w", err)
			}
			return string(b), nil
		},
	}
}

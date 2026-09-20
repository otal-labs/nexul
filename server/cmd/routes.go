package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/mcp"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/mentions"
	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/live"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/openapi"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/presence"
	"github.com/otal-labs/nexul/internal/repository"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/topology"
	"github.com/otal-labs/nexul/internal/voice"
	"github.com/otal-labs/nexul/internal/workspace"
	"github.com/otal-labs/nexul/server/webui"
)

// pairingPresenceHandler reads the keeper's held-session state for the settings row's status dot — no extra probing.
func pairingPresenceHandler(presenceKeeper *presence.Keeper) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := ""
		if a, ok := identity.ActorFromCtx(r.Context()); ok {
			userID = a.ID
		}
		states := presenceKeeper.Status(userID)
		if states == nil {
			states = map[string]string{}
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"computers": states})
	}
}

// liveEventsHandler's socket doubles as the user's harness presence signal for as long as it stays open.
func liveEventsHandler(presenceKeeper *presence.Keeper, liveHub *live.Hub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userID := currentUserID(r); userID != "" {
			presenceKeeper.Connected(userID)
			defer presenceKeeper.Disconnected(userID)
		}
		liveHub.ServeHTTP(w, r)
	})
}

// buildRoutes wires the runner WS listener, the HTTP gateway (ADR 0019), and the MCP HTTP transport.
func buildRoutes(cfg *config.Config, bus *inprocess.Bus, store *storage.Store, svc *coreServices, wsHandler *runner.Handler, runnerSvc *runner.Service, runnerHTTP *runner.HTTPHandler, automationsDialin *automations.DialinHandler, liveHub *live.Hub, agentHandler *agent.Handler, logger *slog.Logger) (wsServer, httpServer, mcpServerHTTP *http.Server) {
	mux := http.NewServeMux()
	mux.Handle("/ws/runner", wsHandler)
	wsServer = &http.Server{Addr: cfg.WSAddr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	apiMux := http.NewServeMux()
	mountGateway(apiMux, "/api/docs", docs.NewHandler(svc.docsSvc).Routes())
	mountGateway(apiMux, "/api/memories", memories.NewHandler(svc.memoriesSvc).Routes())
	mountGateway(apiMux, "/api/attachments", attachments.NewHandler(svc.attachmentsSvc).Routes())
	mountGateway(apiMux, "/api/tickets", tickets.NewHandler(svc.ticketsSvc).Routes())
	mountGateway(apiMux, "/api/topology", topology.NewHandler(svc.topoSvc).Routes())
	deployHandler := deploy.NewHandler(svc.deploySvc).Routes()
	mountGateway(apiMux, "/api/deploys", deployHandler)
	mountGateway(apiMux, "/api/stacks", deployHandler)
	// One-release alias while the web moves off the old name.
	mountGateway(apiMux, "/api/services", deployHandler)
	mountGateway(apiMux, "/api/reviews", codereview.NewHandler(svc.reviewSvc).Routes())
	mountGateway(apiMux, "/api/repos", gitprovider.NewHandler(svc.gitRouter).Routes())
	mountGateway(apiMux, "/api/repositories", repository.NewHandler(svc.repositoryScanner).Routes())
	mountGateway(apiMux, "/api/permissions", access.NewHandler(svc.accessSvc).Routes())
	mountGateway(apiMux, "/api/mentions", mentions.NewHandler(svc.mentionsSvc).Routes())
	mountGateway(apiMux, "/api/projects", withUserID(workspace.WithUserID)(workspace.NewHandler(svc.workspaceSvc).Routes()))
	mountGateway(apiMux, "/api/categories", withUserID(workspace.WithUserID)(workspace.NewHandler(svc.workspaceSvc).Routes()))
	mountGateway(apiMux, "/api/ticket-types", withUserID(workspace.WithUserID)(workspace.NewHandler(svc.workspaceSvc).Routes()))
	mountGateway(apiMux, "/api/statuses", withUserID(workspace.WithUserID)(workspace.NewHandler(svc.workspaceSvc).Routes()))
	mountGateway(apiMux, "/api/workspaces", withUserID(tenancy.WithUserID)(tenancy.NewHandler(svc.tenancySvc).Routes()))
	mountGateway(apiMux, "/api/workspaces/{workspaceID}/roles", withUserID(roles.WithUserID)(roles.NewHandler(svc.rolesSvc).Routes()))
	mountGateway(apiMux, "/api/workspaces/{workspaceID}/plays", plays.NewHandler(svc.playsSvc).Routes())
	mountGateway(apiMux, "/api/plays", plays.NewRunHandler(svc.playsRunner).Routes())
	mountGateway(apiMux, "/api/automations", automations.NewHandler(svc.automationsSvc).WithVersions(svc.automationVersionsSvc).Routes())
	mountGateway(apiMux, "/api/automation-secrets", automations.NewSecretsHandler(svc.automationSecretsSvc).Routes())
	// Registered as exact patterns, more specific than the "/api/automations/" subtree mountGateway claimed above.
	automationRunsRoutes := automations.NewRunsHandler(svc.automationRunsSvc).Routes()
	apiMux.Handle("GET /api/automations/{id}/runs", automationRunsRoutes)
	apiMux.Handle("GET /api/automations/{id}/runs/{runID}", automationRunsRoutes)
	mountGateway(apiMux, "/api/auth", svc.authHandler.ProtectedRoutes())
	runnerRoutes := runnerHTTP.Routes()
	mountGateway(apiMux, "/api/runners", runnerRoutes)
	// Machines live in the runner domain (issue 05); registered as exact patterns since /api/machines/{id}/import
	// (deploy domain, below) shares the prefix.
	apiMux.Handle("GET /api/machines", runnerRoutes)
	apiMux.Handle("PATCH /api/machines/{id}", runnerRoutes)
	apiMux.Handle("POST /api/machines/{id}/discover", runnerRoutes)
	apiMux.Handle("POST /api/machines/{id}/import", deployHandler)
	mountGateway(apiMux, "/api/dns", svc.dnsHandler.Routes())
	mountGateway(apiMux, "/api/pairing", svc.pairingHandler.Routes())
	apiMux.HandleFunc("GET /api/pairing/presence", pairingPresenceHandler(svc.presenceKeeper))
	mountGateway(apiMux, "/api/connectors", withUserID(connectors.WithUserID)(svc.connectorsHandler.Routes()))
	mountGateway(apiMux, "/api/notifications", workspace.NewNotificationHandler(svc.notifSvc, currentUserID).Routes())
	mountGateway(apiMux, "/api/chat", withUserID(chat.WithUserID)(chat.NewHandler(svc.chatSvc).Routes()))
	mountGateway(apiMux, "/api/voice", withUserID(voice.WithUserID)(svc.voiceHandler.Routes()))
	mountGateway(apiMux, "/api/agent", agentHandler.Routes())
	integrationsRoutes := integrations.NewHandler(svc.integrationsSvc).Routes()
	mountGateway(apiMux, "/api/integrations", integrationsRoutes)
	mountGateway(apiMux, "/api/events", integrationsRoutes)
	mountGateway(apiMux, "/api/audit", integrationsRoutes)
	// The web UI's console errors and uncaught exceptions ride the server's logger into the same sinks (ADR 0008).
	apiMux.Handle("POST /api/logs", logging.BrowserHandler(logger, currentUserID))
	apiMux.HandleFunc("GET /api/version", versionHandler(runnerSvc))
	instanceAdmin := instanceAdminGate{svc: svc.authSvc}
	apiMux.HandleFunc("GET /api/instance/upgrade", instanceUpgradeGetHandler(runnerSvc, instanceAdmin))
	apiMux.HandleFunc("POST /api/instance/upgrade", instanceUpgradePostHandler(runnerSvc, instanceAdmin))

	// Generated from the gateway by registering the mounted routes, never hand-written.
	spec := openapi.New(openapi.Info{Title: "Nexul API", Version: "v1"})
	spec.AddSecuritySchemes()
	registerOpenAPIRoutes(spec)

	mcpServer := mcp.New(mcp.RegistryOptions{
		Docs:          svc.docsSvc,
		Memories:      svc.memoriesSvc,
		Tickets:       svc.ticketsSvc,
		Topology:      svc.topoSvc,
		Deploy:        svc.deploySvc,
		Reviews:       svc.reviewSvc,
		Workspace:     svc.workspaceSvc,
		Notifications: svc.notifSvc,
		Git:           svc.gitRouter,
		Repository:    svc.repositoryScanner,
		Runner:        runnerSvc,
		DNS:           svc.dnsSvc,
		Automations:   svc.automationsSvc,
		Access:        svc.accessSvc,
		Mentions:      svc.mentionsSvc,
		Chat:          svc.chatSvc,
		Plays:         svc.playsSvc,
		PlayRuns:      svc.playsRunner,
		DeadLetter:    store.DeadLetters,
		Publisher:     bus,
		Logger:        logger,
		Actor:         mcpActor,
	})

	httpMux := http.NewServeMux()
	mountGateway(httpMux, "/auth", svc.authHandler.Routes())
	httpMux.Handle("/auth/connectors/", svc.connectorsHandler.PublicRoutes())
	// Authenticates with the runner secret, not the browser session — a fresh machine's curl carries no session token.
	httpMux.Handle("GET /api/runners/download/{target}", runnerHTTP.PublicRoutes())
	// Without this, these fall through to the /api/ catch-all below and 401 before reaching the handler.
	httpMux.Handle("GET /api/auth/bootstrap-status", svc.authHandler.Routes())
	httpMux.Handle("POST /api/auth/bootstrap", svc.authHandler.Routes())
	httpMux.Handle("POST /api/auth/bootstrap/verify", svc.authHandler.Routes())
	// Automation tokens, then integration tokens, then session/PAT — one audit-logged handler underneath all three.
	userAuth := func(h http.Handler) http.Handler { return svc.authSvc.RequireAuth(withIdentity(h)) }
	httpMux.Handle("/api/", svc.automationsSvc.RequireAutomation(
		func(h http.Handler) http.Handler { return svc.integrationsSvc.RequireIntegration(userAuth, h) },
		svc.integrationsSvc.AuditLog(resolveAuditActor, apiMux),
	))
	httpMux.Handle("/ws/events", svc.authSvc.RequireWS(liveEventsHandler(svc.presenceKeeper, liveHub)))
	httpMux.Handle("GET /ws/collab/{docID}", svc.authSvc.RequireWS(withIdentity(svc.collabHub)))
	// The SDK derives this from the instance URL, so it must share the API's origin, not the runner's WS listener.
	httpMux.Handle("/ws/automations", automationsDialin)
	// Also on the main listener: the install command and the MCP URL are derived from the instance URL, so a proxy
	// (or the single binary) only has to forward one origin; the dedicated listeners below stay for direct access.
	httpMux.Handle("/ws/runner", wsHandler)
	httpMux.Handle("/mcp", svc.authSvc.RequireAuth(mcpServer))
	httpMux.Handle("/hooks/github", gitprovider.NewWebhookHandler(cfg.AuthSecret, bus))
	httpMux.Handle("/hooks/livekit", svc.voiceWebhookHandler)
	httpMux.Handle("/openapi.json", spec.Handler())
	httpMux.Handle("/swagger", spec.Handler())
	httpMux.Handle("/", webui.Handler(webui.Assets()))
	httpServer = &http.Server{Addr: cfg.HTTPAddr, Handler: httpMux, ReadHeaderTimeout: 10 * time.Second}

	// The MCP server is a plain http.Handler, so the same gateway auth middleware guards it.
	mcpServerHTTP = &http.Server{Addr: cfg.MCPAddr, Handler: svc.authSvc.RequireAuth(mcpServer), ReadHeaderTimeout: 10 * time.Second}

	return wsServer, httpServer, mcpServerHTTP
}

// shutdownServers gives the WS, MCP, and HTTP servers a bounded window to drain (graceful shutdown).
func shutdownServers(wsServer, mcpServerHTTP, httpServer *http.Server) {
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = wsServer.Shutdown(shutdown)
	_ = mcpServerHTTP.Shutdown(shutdown)
	_ = httpServer.Shutdown(shutdown)
}

// registerOpenAPIRoutes mirrors the mounted gateway into the OpenAPI spec; new routes get a Register line here too.
func registerOpenAPIRoutes(spec *openapi.Spec) {
	spec.SetTagDescription("auth", "Identity, onboarding, settings, members")
	spec.SetTagDescription("docs", "Documents and version history")
	spec.SetTagDescription("attachments", "Files attached to docs and tickets")
	spec.SetTagDescription("tickets", "Tickets and status transitions")
	spec.SetTagDescription("deploys", "Deploy requests and history")
	spec.SetTagDescription("topology", "Topology canvas")
	spec.SetTagDescription("reviews", "Code review records")
	spec.SetTagDescription("repos", "Git provider repos and PRs")
	spec.SetTagDescription("repositories", "Repository scan: deployable candidates from a repo's tree")
	spec.SetTagDescription("permissions", "Document permission grants")
	spec.SetTagDescription("mentions", "Cross-reference mentions and live chips")
	spec.SetTagDescription("projects", "Workspace projects")
	spec.SetTagDescription("workspaces", "Workspaces and membership")
	spec.SetTagDescription("roles", "Workspace roles and permission masks")
	spec.SetTagDescription("automations", "First-party event-driven automations: identity, config, and scoped tokens")
	spec.SetTagDescription("runners", "Runner fleet visibility")
	spec.SetTagDescription("machines", "Machines runners belong to, discovery, and import")
	spec.SetTagDescription("dns", "DNS zones, records, service hostnames, tunnels")
	spec.SetTagDescription("notifications", "Notification inbox")
	spec.SetTagDescription("chat", "Workspace conversations, messages, and unread state")
	spec.SetTagDescription("voice", "Voice channel join tokens and live occupancy")
	spec.SetTagDescription("agent", "The @Agent chat pipeline's stop control")
	spec.SetTagDescription("integrations", "Integration installs, subscriptions, deliveries")
	spec.SetTagDescription("events", "Published event-schema catalog")
	spec.SetTagDescription("audit", "Audit log")
	spec.SetTagDescription("logs", "Browser log relay into the server's log sinks")
	spec.SetTagDescription("version", "Build version, channel, and update check")

	spec.Register("GET", "/api/auth/me", "Current user + onboarding state", "auth")
	spec.Register("POST", "/api/logs", "Relay a batch of browser log records (console errors, uncaught exceptions)", "logs")
	spec.Register("GET", "/api/version", "This build's version, channel, and the channel's latest release", "version")
	spec.Register("GET", "/api/instance/upgrade", "Instance-admin: running version, latest release, and whether an upgrade can start now", "version")
	spec.Register("POST", "/api/instance/upgrade", "Instance-admin: upgrade the instance to the channel's newest release", "version")
	spec.Register("POST", "/api/auth/tokens", "Mint a personal access token", "auth")
	spec.Register("GET", "/api/auth/tokens", "List personal access tokens", "auth")
	spec.Register("DELETE", "/api/auth/tokens/{id}", "Revoke a personal access token", "auth")
	spec.Register("GET", "/api/docs", "List docs", "docs")
	spec.Register("POST", "/api/docs", "Create a doc", "docs")
	spec.Register("GET", "/api/docs/search", "Search docs", "docs")
	spec.Register("GET", "/api/docs/{id}", "Get a doc", "docs")
	spec.Register("PUT", "/api/docs/{id}", "Update a doc", "docs")
	spec.Register("DELETE", "/api/docs/{id}", "Delete a doc", "docs")
	spec.Register("POST", "/api/docs/{id}/versions", "Snapshot the doc as a named milestone version", "docs")
	spec.Register("POST", "/api/attachments", "Upload a file to a doc or ticket (multipart: file, doc_id|ticket_id)", "attachments")
	spec.Register("GET", "/api/attachments", "List a doc or ticket's attachments", "attachments")
	spec.Register("GET", "/api/attachments/{id}", "Download an attachment", "attachments")
	spec.Register("DELETE", "/api/attachments/{id}", "Delete an attachment", "attachments")
	spec.Register("GET", "/api/tickets", "List tickets", "tickets")
	spec.Register("POST", "/api/tickets", "Create a ticket", "tickets")
	spec.Register("GET", "/api/tickets/{id}", "Get a ticket", "tickets")
	spec.Register("PATCH", "/api/tickets/{id}/status", "Transition a ticket's status", "tickets")
	spec.Register("GET", "/api/deploys", "List deploys", "deploys")
	spec.Register("POST", "/api/deploys", "Trigger a deploy", "deploys")
	spec.Register("GET", "/api/deploys/{id}", "Get a deploy", "deploys")
	spec.Register("POST", "/api/deploys/{id}/cancel", "Cancel a pending/running deploy", "deploys")
	spec.Register("GET", "/api/topology", "Get the topology canvas", "topology")
	spec.Register("PUT", "/api/topology", "Replace the topology canvas", "topology")
	spec.Register("GET", "/api/reviews", "List code reviews", "reviews")
	spec.Register("GET", "/api/reviews/{id}", "Get a code review", "reviews")
	spec.Register("GET", "/api/repos/{owner}/{repo}/prs", "List pull requests", "repos")
	spec.Register("POST", "/api/repositories/scan", "Scan a repository's tree for deployable candidates", "repositories")
	spec.Register("GET", "/api/repositories", "List repositories the connected GitHub App installation grants", "repositories")
	spec.Register("GET", "/api/permissions", "List document permission grants", "permissions")
	spec.Register("GET", "/api/permissions/catalog", "List the permission grid roles, tokens, and grants share", "permissions")
	spec.Register("GET", "/api/mentions/search", "Search mention targets for the @ picker", "mentions")
	spec.Register("POST", "/api/mentions/resolve", "Resolve mention references to live chips", "mentions")
	spec.Register("GET", "/api/projects", "List projects", "projects")
	spec.Register("POST", "/api/projects", "Create a project", "projects")
	spec.Register("GET", "/api/workspaces", "List the caller's workspaces", "workspaces")
	spec.Register("POST", "/api/workspaces", "Create a workspace", "workspaces")
	spec.Register("GET", "/api/workspaces/{workspaceID}/roles", "List a workspace's roles", "roles")
	spec.Register("POST", "/api/workspaces/{workspaceID}/roles", "Create a custom role", "roles")
	spec.Register("GET", "/api/workspaces/{workspaceID}/roles/{roleID}", "Get a role", "roles")
	spec.Register("PATCH", "/api/workspaces/{workspaceID}/roles/{roleID}", "Rename a role or change its permission mask", "roles")
	spec.Register("DELETE", "/api/workspaces/{workspaceID}/roles/{roleID}", "Delete a custom role", "roles")
	spec.Register("GET", "/api/automations", "List automations", "automations")
	spec.Register("POST", "/api/automations", "Create a Custom automation and mint its scoped token", "automations")
	spec.Register("GET", "/api/automations/{id}", "Get an automation", "automations")
	spec.Register("PATCH", "/api/automations/{id}/config", "Replace an automation's config values", "automations")
	spec.Register("PATCH", "/api/automations/{id}/enabled", "Enable or disable an automation", "automations")
	spec.Register("DELETE", "/api/automations/{id}", "Delete an automation", "automations")
	spec.Register("POST", "/api/automations/{id}/token", "Rotate an automation's token", "automations")
	spec.Register("DELETE", "/api/automations/{id}/token", "Revoke an automation's token", "automations")
	spec.Register("GET", "/api/runners", "List runners", "runners")
	spec.Register("GET", "/api/runners/install", "Install info for a new runner (WS URL + shared runner secret)", "runners")
	spec.Register("GET", "/api/runners/download/{target}", "Download the runner binary for a target (runner-secret auth)", "runners")
	spec.Register("GET", "/api/runners/latest-version", "Latest published runner version", "runners")
	spec.Register("GET", "/api/machines", "List machines", "machines")
	spec.Register("PATCH", "/api/machines/{id}", "Rename a machine or change its stack root", "machines")
	spec.Register("POST", "/api/machines/{id}/discover", "Scan a machine's containers and networks", "machines")
	spec.Register("POST", "/api/machines/{id}/import", "Adopt ticked discovery selections as unmanaged stacks", "machines")
	spec.Register("GET", "/api/dns/zones", "List DNS zones", "dns")
	spec.Register("GET", "/api/dns/service-hostnames", "List service hostnames", "dns")
	spec.Register("POST", "/api/dns/tunnels", "Create a Cloudflare tunnel", "dns")
	spec.Register("GET", "/api/dns/tunnels", "List Cloudflare tunnels", "dns")
	spec.Register("POST", "/api/dns/tunnels/{tunnelID}/route", "Route a hostname into a tunnel", "dns")
	spec.Register("POST", "/api/dns/tunnels/{tunnelID}/rotate", "Rotate tunnel credentials", "dns")
	spec.Register("DELETE", "/api/dns/tunnels/{tunnelID}", "Delete a tunnel", "dns")
	spec.Register("POST", "/api/dns/gateways", "Create a gateway (tunnel or proxy)", "dns")
	spec.Register("GET", "/api/dns/gateways", "List gateways", "dns")
	spec.Register("DELETE", "/api/dns/gateways/{gatewayID}", "Delete a gateway", "dns")
	spec.Register("POST", "/api/dns/exposures", "Route a hostname through a gateway to a container", "dns")
	spec.Register("GET", "/api/dns/exposures", "List exposures", "dns")
	spec.Register("DELETE", "/api/dns/exposures/{exposureID}", "Remove an exposure", "dns")
	spec.Register("GET", "/api/connectors", "List connectors and their connection status", "connectors")
	spec.Register("GET", "/api/connectors/{id}/oauth/start", "Start a connector's OAuth consent flow", "connectors")
	spec.Register("POST", "/api/connectors/{id}/disconnect", "Disconnect a connector", "connectors")
	spec.Register("GET", "/api/notifications", "List the notification inbox", "notifications")
	spec.Register("GET", "/api/chat/conversations", "List the caller's conversations in a workspace", "chat")
	spec.Register("POST", "/api/chat/channels", "Create a channel", "chat")
	spec.Register("POST", "/api/chat/dms", "Create a direct-message conversation", "chat")
	spec.Register("POST", "/api/chat/tickets/{ticketID}/thread", "Get or lazily create a ticket's thread", "chat")
	spec.Register("GET", "/api/chat/tickets/thread-status", "Batch-check which tickets have a thread", "chat")
	spec.Register("GET", "/api/chat/conversations/{id}/messages", "List a conversation's messages", "chat")
	spec.Register("POST", "/api/chat/conversations/{id}/messages", "Post a message", "chat")
	spec.Register("PATCH", "/api/chat/messages/{id}", "Edit a message (author only)", "chat")
	spec.Register("DELETE", "/api/chat/messages/{id}", "Delete a message (author only)", "chat")
	spec.Register("POST", "/api/chat/conversations/{id}/read", "Mark a conversation read", "chat")
	spec.Register("GET", "/api/chat/unread", "Per-conversation unread counts", "chat")
	spec.Register("POST", "/api/chat/voice-channels", "Create a voice channel", "chat")
	spec.Register("POST", "/api/voice/{conversationID}/token", "Mint a LiveKit join token for a voice channel", "voice")
	spec.Register("POST", "/api/voice/{conversationID}/leave", "Leave a voice channel (removes optimistic presence)", "voice")
	spec.Register("GET", "/api/voice/occupancy", "Current voice channel occupancy", "voice")
	spec.Register("POST", "/api/agent/conversations/{id}/interrupt", "Stop the in-flight agent turn on a conversation", "agent")
	spec.Register("POST", "/api/agent/conversations/{id}/answer", "Answer the question the agent turn on a conversation is waiting on", "agent")
	spec.Register("POST", "/api/integrations", "Install an integration (mint scoped token)", "integrations")
	spec.Register("GET", "/api/integrations", "List integration installs", "integrations")
	spec.Register("GET", "/api/integrations/scopes", "List grantable token scopes with labels", "integrations")
	spec.Register("GET", "/api/integrations/{id}", "Get an integration install", "integrations")
	spec.Register("DELETE", "/api/integrations/{id}", "Revoke an integration install", "integrations")
	spec.Register("POST", "/api/integrations/{id}/subscriptions", "Subscribe to a topic", "integrations")
	spec.Register("GET", "/api/integrations/{id}/subscriptions", "List subscriptions", "integrations")
	spec.Register("DELETE", "/api/integrations/{id}/subscriptions/{topic}", "Unsubscribe from a topic", "integrations")
	spec.Register("GET", "/api/integrations/{id}/deliveries", "List webhook deliveries", "integrations")
	spec.Register("GET", "/api/events/catalog", "Published event-schema catalog", "events")
	spec.Register("GET", "/api/audit", "Audit log", "audit")
}

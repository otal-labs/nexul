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
	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/calendar"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/crashintake"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/mcp"
	"github.com/otal-labs/nexul/internal/mcp/composite"
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
	"github.com/otal-labs/nexul/internal/templates"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/topology"
	"github.com/otal-labs/nexul/internal/voice"
	"github.com/otal-labs/nexul/internal/welcome"
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

// buildRoutes wires the HTTP gateway (ADR 0019), the runner and browser WebSockets, the MCP HTTP transport, and the web UI on one listener.
func buildRoutes(cfg *config.Config, bus *inprocess.Bus, store *storage.Store, svc *coreServices, wsHandler *runner.Handler, runnerSvc *runner.Service, runnerHTTP *runner.HTTPHandler, automationsDialin *automations.DialinHandler, liveHub *live.Hub, agentHandler *agent.Handler, logger *slog.Logger) *http.Server {
	apiMux := httpx.NewServeMux()
	mountGateway(apiMux, "/api/docs", docs.NewHandler(svc.docsSvc).Routes())
	mountGateway(apiMux, "/api/memories", memories.NewHandler(svc.memoriesSvc).Routes())
	mountGateway(apiMux, "/api/templates", templates.NewHandler(svc.templatesSvc).Routes())
	mountGateway(apiMux, "/api/attachments", attachments.NewHandler(svc.attachmentsSvc).Routes())
	mountGateway(apiMux, "/api/tickets", tickets.NewHandler(svc.ticketsSvc).Routes())
	mountGateway(apiMux, "/api/crash-reports", crashintake.NewHandler(svc.crashIntakeSvc).Routes())
	mountGateway(apiMux, "/api/topology", topology.NewHandler(svc.topoSvc).Routes())
	deployHandler := deploy.NewHandler(svc.deploySvc).Routes()
	mountGateway(apiMux, "/api/deploys", deployHandler)
	mountGateway(apiMux, "/api/stacks", deployHandler)
	// One-release alias while the web moves off the old name.
	mountGateway(apiMux, "/api/services", deployHandler)
	mountGateway(apiMux, "/api/reviews", codereview.NewHandler(svc.reviewSvc).Routes())
	repoGate := projectEntityGate{access: svc.accessSvc, projects: store.Projects, tickets: store.Tickets}
	changeContext := changeContextReader{tickets: svc.ticketsSvc, docs: svc.docsSvc, workspace: svc.workspaceSvc, memories: svc.memoriesSvc}
	mountGateway(apiMux, "/api/repos", gitprovider.NewHandler(svc.gitRouter, repoGate).WithChangeContext(changeContext).Routes())
	mountGateway(apiMux, "/api/repositories", repository.NewHandler(svc.repositorySvc).Routes())
	mountGateway(apiMux, "/api/permissions", access.NewHandler(svc.accessSvc).Routes())
	mountGateway(apiMux, "/api/mentions", mentions.NewHandler(svc.mentionsSvc).Routes())
	mountGateway(apiMux, "/api/projects", withUserID(workspace.WithUserID)(workspace.NewHandler(svc.workspaceSvc).Routes()))
	mountGateway(apiMux, "/api/categories", withUserID(workspace.WithUserID)(workspace.NewHandler(svc.workspaceSvc).Routes()))
	mountGateway(apiMux, "/api/ticket-types", withUserID(workspace.WithUserID)(workspace.NewHandler(svc.workspaceSvc).Routes()))
	mountGateway(apiMux, "/api/statuses", withUserID(workspace.WithUserID)(workspace.NewHandler(svc.workspaceSvc).Routes()))
	mountGateway(apiMux, "/api/workspaces", withUserID(tenancy.WithUserID)(tenancy.NewHandler(svc.tenancySvc).Routes()))
	mountGateway(apiMux, "/api/team", withUserID(tenancy.WithUserID)(tenancy.NewHandler(svc.tenancySvc).TeamRoutes()))
	mountGateway(apiMux, "/api/people", withUserID(tenancy.WithUserID)(tenancy.NewHandler(svc.tenancySvc).PeopleRoutes()))
	// An exact pattern, more specific than the "/api/projects/" subtree the workspace domain claimed above.
	apiMux.Handle("GET /api/projects/{projectID}/people", withUserID(tenancy.WithUserID)(tenancy.NewHandler(svc.tenancySvc).ProjectPeopleRoutes()))
	mountGateway(apiMux, "/api/invitations", withUserID(tenancy.WithUserID)(svc.invitationHandler.Routes()))
	mountGateway(apiMux, "/api/workspaces/{workspaceID}/roles", withUserID(roles.WithUserID)(roles.NewHandler(svc.rolesSvc).Routes()))
	mountGateway(apiMux, "/api/workspaces/{workspaceID}/access", access.NewHandler(svc.accessSvc).GridRoutes())
	mountGateway(apiMux, "/api/workspaces/{workspaceID}/plays", plays.NewHandler(svc.playsSvc).Routes())
	mountGateway(apiMux, "/api/plays", plays.NewRunHandler(svc.playsRunner, projectTargets{tickets: svc.ticketsSvc, docs: svc.docsSvc}).Routes())
	mountGateway(apiMux, "/api/automations", automations.NewHandler(svc.automationsSvc).WithVersions(svc.automationVersionsSvc).Routes())
	mountGateway(apiMux, "/api/automation-secrets", automations.NewSecretsHandler(svc.automationSecretsSvc).Routes())
	automationHostsHTTP := automations.NewHostsHandler(svc.automationHostsSvc)
	mountGateway(apiMux, "/api/automation-hosts", automationHostsHTTP.Routes())
	// Registered as exact patterns, more specific than the "/api/automations/" subtree mountGateway claimed above.
	automationRunsRoutes := automations.NewRunsHandler(svc.automationRunsSvc).Routes()
	apiMux.Handle("GET /api/automations/{id}/runs", automationRunsRoutes)
	apiMux.Handle("GET /api/automations/{id}/runs/{runID}", automationRunsRoutes)
	mountGateway(apiMux, "/api/auth", svc.authHandler.ProtectedRoutes())
	mountGateway(apiMux, "/api/setup", svc.authHandler.SetupRoutes())
	runnerRoutes := runnerHTTP.Routes()
	mountGateway(apiMux, "/api/runners", runnerRoutes)
	// Machines live in the runner domain (issue 05); registered as exact patterns since /api/machines/{id}/import
	// (deploy domain, below) shares the prefix.
	apiMux.Handle("GET /api/machines", runnerRoutes)
	apiMux.Handle("PATCH /api/machines/{id}", runnerRoutes)
	apiMux.Handle("DELETE /api/machines/{id}", runnerRoutes)
	apiMux.Handle("POST /api/machines/{id}/discover", runnerRoutes)
	apiMux.Handle("POST /api/machines/{id}/import", deployHandler)
	apiMux.Handle("GET /api/machines/deployable", deployHandler)
	mountGateway(apiMux, "/api/dns", svc.dnsHandler.Routes())
	mountGateway(apiMux, "/api/pairing", svc.pairingHandler.Routes())
	mountGateway(apiMux, "/api/me", welcome.NewHandler(svc.welcomeSvc).Routes())
	apiMux.HandleFunc("GET /api/pairing/presence", pairingPresenceHandler(svc.presenceKeeper))
	mountGateway(apiMux, "/api/connectors", withUserID(connectors.WithUserID)(svc.connectorsHandler.Routes()))
	mountGateway(apiMux, "/api/notifications", workspace.NewNotificationHandler(svc.notifSvc, currentUserID).Routes())
	mountGateway(apiMux, "/api/chat", withUserID(chat.WithUserID)(chat.NewHandler(svc.chatSvc, projectTargets{tickets: svc.ticketsSvc, docs: svc.docsSvc}).Routes()))
	mountGateway(apiMux, "/api/voice", withUserID(voice.WithUserID)(svc.voiceHandler.Routes()))
	botwebhookRoutes := botwebhook.NewHandler(svc.botwebhookSvc).Routes()
	mountGateway(apiMux, "/api/conversations", botwebhookRoutes)
	mountGateway(apiMux, "/api/botwebhooks", botwebhookRoutes)
	calendarRoutes := calendar.NewHandler(svc.calendarSvc).Routes()
	mountGateway(apiMux, "/api/workspaces/{workspaceID}/meetings", calendarRoutes)
	mountGateway(apiMux, "/api/meetings", calendarRoutes)
	mountGateway(apiMux, "/api/agent", agentHandler.Routes())
	integrationsRoutes := integrations.NewHandler(svc.integrationsSvc).Routes()
	mountGateway(apiMux, "/api/integrations", integrationsRoutes)
	mountGateway(apiMux, "/api/events", integrationsRoutes)
	mountGateway(apiMux, "/api/audit", integrationsRoutes)
	// The web UI's console errors and uncaught exceptions ride the server's logger into the same sinks (ADR 0008).
	apiMux.Handle("POST /api/logs", logging.BrowserHandler(logger, currentUserID))
	apiMux.HandleFunc("GET /api/version", versionHandler(runnerSvc))
	apiMux.HandleFunc("GET /api/instance/upgrade", instanceUpgradeGetHandler(runnerSvc))
	apiMux.HandleFunc("POST /api/instance/upgrade", instanceUpgradePostHandler(runnerSvc))
	apiMux.HandleFunc("GET /api/instance/logs", instanceLogsHandler(runnerSvc))

	spec := openapi.New(openapi.Info{Title: "Nexul API", Version: "v1"})
	spec.AddSecuritySchemes()

	mcpServer := mcp.New(mcp.RegistryOptions{
		Docs:          svc.docsSvc,
		Attachments:   svc.attachmentsSvc,
		Memories:      svc.memoriesSvc,
		Templates:     svc.templatesSvc,
		Tickets:       svc.ticketsSvc,
		Topology:      svc.topoSvc,
		Deploy:        svc.deploySvc,
		Reviews:       svc.reviewSvc,
		Workspace:     svc.workspaceSvc,
		Notifications: svc.notifSvc,
		Git:           svc.gitRouter,
		ChangeContext: changeContext,
		GitGate:       repoGate,
		Repository:    svc.repositorySvc,
		Runner:        runnerSvc,
		Hosts: map[string]composite.HostKind{
			"runner":      runnerHostKind{svc: runnerSvc},
			"automations": automationsHostKind{svc: svc.automationHostsSvc},
			"machine":     machineHostKind{svc: runnerSvc},
		},
		DNS:         svc.dnsSvc,
		Automations: svc.automationsSvc,
		Access:      svc.accessSvc,
		Auth:        svc.authSvc,
		Invitations: svc.invitationSvc,
		Workspaces:  svc.tenancySvc,
		Roles:       svc.rolesSvc,
		Mentions:    svc.mentionsSvc,
		Chat:        svc.chatSvc,
		Botwebhooks: svc.botwebhookSvc,
		Calendar:    svc.calendarSvc,
		Plays:       svc.playsSvc,
		PlayRuns:    svc.playsRunner,
		Pairing:     svc.pairingSvc,
		DeadLetter:  store.DeadLetters,
		Publisher:   bus,
		Logger:      logger,
		InstanceURL: dnsSettingsAdapter{store.Settings}.GetInstanceURL,
		Audit:       svc.integrationsSvc.AuditTool(resolveAuditActor),
	})

	httpMux := httpx.NewServeMux()
	mountGateway(httpMux, "/auth", svc.authHandler.Routes())
	httpMux.Handle("/auth/connectors/", svc.connectorsHandler.PublicRoutes())
	// A machine holds no session: these authenticate with an enrollment code or the runner's own credential.
	runnerPublic := runnerHTTP.PublicRoutes()
	httpMux.Handle("GET /api/runners/download/{target}", runnerPublic)
	httpMux.Handle("POST /api/runners/enroll", runnerPublic)
	httpMux.Handle("POST /api/runners/self/remove", runnerPublic)
	automationHostsPublic := automationHostsHTTP.PublicRoutes()
	httpMux.Handle("POST /api/automation-hosts/enroll", automationHostsPublic)
	httpMux.Handle("POST /api/automation-hosts/self/remove", automationHostsPublic)
	httpMux.Handle("GET /api/automation-hosts/self/assignments", automationHostsPublic)
	httpMux.Handle("POST /api/automation-hosts/self/logs", automationHostsPublic)
	viewAs := withViewAs(svc.tenancySvc, svc.authSvc)
	userAuth := func(h http.Handler) http.Handler { return svc.authSvc.RequireAuth(viewAs(withIdentity(h))) }
	// Without this, these fall through to the /api/ catch-all below and 401 before reaching the handler.
	httpMux.HandleFunc("GET /api/about", aboutHandler)
	httpMux.Handle("GET /api/auth/bootstrap-status", svc.authHandler.Routes())
	httpMux.Handle("POST /api/setup/unlock", svc.authHandler.Routes())
	httpMux.Handle("POST /api/auth/connect-codes/exchange", svc.authHandler.Routes())
	// Bootstrap lives on the public mux but needs the setup pass, so only session/PAT/pass auth wraps it.
	httpMux.Handle("POST /api/auth/bootstrap", userAuth(svc.authHandler.Routes()))
	httpMux.Handle("POST /api/auth/bootstrap/verify", userAuth(svc.authHandler.Routes()))
	httpMux.Handle("POST /api/invitations/preview", svc.invitationHandler.PublicRoutes())
	httpMux.Handle("POST /api/invitations/oauth", svc.authHandler.Routes())
	httpMux.Handle("POST /api/invitations/acceptance", svc.authHandler.Routes())
	httpMux.Handle("POST /api/invitations/redeem", svc.authHandler.Routes())
	// Automation tokens, then integration tokens, then session/PAT/setup pass — one audit-logged handler underneath all.
	httpMux.Handle("/api/", withAccessMemo(svc.accessSvc, svc.automationsSvc.RequireAutomation(
		func(h http.Handler) http.Handler { return svc.integrationsSvc.RequireIntegration(userAuth, h) },
		rejectViewAsOffSession(svc.integrationsSvc.AuditLog(resolveAuditActor, apiMux)),
	)))
	httpMux.Handle("/ws/events", svc.authSvc.RequireWS(withIdentity(liveEventsHandler(svc.presenceKeeper, liveHub))))
	httpMux.Handle("GET /ws/collab/{id}", svc.authSvc.RequireWS(withIdentity(svc.collabHub)))
	httpMux.Handle("GET /ws/collab/notes/{id}", svc.authSvc.RequireWS(withIdentity(svc.notesHub)))
	httpMux.Handle("GET /ws/collab/tickets/{id}", svc.authSvc.RequireWS(withIdentity(svc.ticketsHub)))
	httpMux.Handle("GET /ws/stacks/{id}/services/{name}/logs", svc.authSvc.RequireWS(withIdentity(http.HandlerFunc(deploy.NewHandler(svc.deploySvc).LogsSocket))))
	httpMux.Handle("GET /ws/pairing/computers/{id}/containers/{name}/logs", svc.authSvc.RequireWS(withIdentity(http.HandlerFunc(svc.pairingHandler.ContainerLogsSocket))))
	httpMux.Handle("GET /ws/pairing/computers/{id}/runner/logs", svc.authSvc.RequireWS(withIdentity(http.HandlerFunc(svc.pairingHandler.RunnerLogsSocket))))
	httpMux.Handle("/ws/automations", automationsDialin)
	httpMux.Handle("/ws/runner", wsHandler)
	// A computer's stream authenticates with the runner's own credential, like /ws/runner.
	httpMux.HandleFunc("GET /api/runners/streams/{id}", wsHandler.ServeStream)
	httpMux.Handle("/mcp", svc.authSvc.RequireAuth(withIdentity(mcpServer)))
	// No OAuth authorization server: a client probing OAuth discovery gets a clean 404, not the web app's HTML.
	httpMux.Handle("/.well-known/", http.NotFoundHandler())
	httpMux.Handle("/hooks/github", gitprovider.NewWebhookHandler(githubWebhookSecret(cfg.AuthSecret), bus))
	httpMux.Handle("/hooks/livekit", svc.voiceWebhookHandler)
	// The token in the path is the credential, so the route stays outside the audit middleware, which records raw paths.
	httpMux.Handle("POST /api/botwebhooks/{id}/{token}", botwebhook.NewExecuteHandler(svc.botwebhookSvc, logger).Routes())
	mountLogsProxy(httpMux, cfg.LogsURL, logger)
	routes := append(httpx.RoutesOf(apiMux), httpx.RoutesOf(httpMux)...)
	registerOpenAPIRoutes(spec, routes)
	httpMux.Handle("/openapi.json", spec.Handler())
	httpMux.Handle("/swagger", spec.Handler())
	httpMux.Handle("/", webui.Handler(webui.Assets()))
	return &http.Server{Addr: cfg.HTTPAddr, Handler: httpMux, ReadHeaderTimeout: 10 * time.Second}
}

// shutdownServer gives the HTTP server a bounded window to drain (graceful shutdown).
func shutdownServer(httpServer *http.Server) {
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdown)
}

// registerOpenAPIRoutes adds mounted operations and applies the maintained summaries.
func registerOpenAPIRoutes(spec *openapi.Spec, routes []httpx.Route) {
	spec.RegisterMountedRoutes(routes)
	spec.SetTagDescription("auth", "Identity, onboarding, settings, members")
	spec.SetTagDescription("setup", "First run: setup code unlock, instance URL, public address")
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
	spec.SetTagDescription("runners", "Runner fleet: enrollment, visibility, and removal")
	spec.SetTagDescription("automation-hosts", "Automations hosts: enrollment, assignments, and removal")
	spec.SetTagDescription("machines", "Machines runners belong to, discovery, and import")
	spec.SetTagDescription("dns", "DNS zones, records, service hostnames, tunnels")
	spec.SetTagDescription("notifications", "Notification inbox")
	spec.SetTagDescription("chat", "Workspace conversations, messages, and unread state")
	spec.SetTagDescription("voice", "Voice channel join tokens and live occupancy")
	spec.SetTagDescription("botwebhooks", "Bots: a conversation's webhook posters, their URLs, deletion, and restore")
	spec.SetTagDescription("calendar", "The workspace calendar: meetings, their invitees and replies, repeats, cancelling and restoring")
	spec.SetTagDescription("agent", "The @Agent chat pipeline's stop control")
	spec.SetTagDescription("integrations", "Integration installs, subscriptions, deliveries")
	spec.SetTagDescription("events", "Published event-schema catalog")
	spec.SetTagDescription("audit", "Audit log")
	spec.SetTagDescription("logs", "The instance's logs: the browser log relay into the server's sinks, and reading them back")
	spec.SetTagDescription("version", "Build version, channel, and update check")
	spec.SetTagDescription("templates", "Instance templates, and cloning or resetting a template at any layer")

	spec.Register("GET", "/api/auth/me", "Current user + onboarding state", "auth")
	spec.Register("POST", "/api/logs", "Relay a batch of browser log records (console errors, uncaught exceptions)", "logs")
	spec.Register("GET", "/api/about", "Public: the product name and this build's version, so a client can confirm the address is a Nexul server new enough for it", "version")
	spec.Register("GET", "/api/version", "This build's version, channel, the channel's latest release, and the release notes since this build; ?refresh=1 skips the release cache", "version")
	spec.Register("GET", "/api/instance/upgrade", "Needs instance:read: running version, latest release, and whether an upgrade can start now; ?refresh=1 skips the release cache", "version")
	spec.Register("POST", "/api/instance/upgrade", "Needs instance:write: upgrade the instance to the channel's newest release", "version")
	spec.Register("GET", "/api/instance/logs", "Needs logs:read: the instance's own logs, newest first, filtered by ?source, ?level, ?since, ?until, ?trace_id, ?name and ?text; ?limit (1 to 500, default 100) and ?offset page them, and the until a page returns pins the next one", "logs")
	spec.Register("POST", "/api/auth/tokens", "Mint a personal access token", "auth")
	spec.Register("GET", "/api/auth/tokens", "List personal access tokens", "auth")
	spec.Register("DELETE", "/api/auth/tokens/{id}", "Revoke a personal access token", "auth")
	spec.Register("GET", "/api/auth/github-link", "Whether the caller's own GitHub sign-in lists their repositories: connected, reconnect, or none", "auth")
	spec.Register("DELETE", "/api/auth/github-link", "Forget the caller's GitHub token, so no repositories list for them; session only", "auth")
	spec.Register("POST", "/api/auth/connect-codes", "Issue a two-minute connect code for the phone app; session only, a personal access token is refused", "auth")
	spec.Register("POST", "/api/auth/connect-codes/exchange", "Public: trade a connect code and the phone's model for a phone session and the server version; 429 after five wrong codes from one address in ten minutes", "auth")
	spec.Register("GET", "/api/docs", "List docs", "docs")
	spec.Register("POST", "/api/docs", "Create a doc", "docs")
	spec.Register("GET", "/api/docs/search", "Search docs", "docs")
	spec.Register("GET", "/api/docs/{id}", "Get a doc", "docs")
	spec.Register("PUT", "/api/docs/{id}", "Update a doc", "docs")
	spec.Register("DELETE", "/api/docs/{id}", "Delete a doc", "docs")
	spec.Register("POST", "/api/docs/{id}/versions", "Snapshot the doc as a named milestone version", "docs")
	spec.Register("GET", "/api/docs/{id}/watchers", "List the people who get the doc's change notifications, and whether you are one", "docs")
	spec.Register("PUT", "/api/docs/{id}/watchers/me", "Watch the doc: get its change notifications (docs:read)", "docs")
	spec.Register("DELETE", "/api/docs/{id}/watchers/me", "Stop watching the doc; your own later edits do not start it again (docs:read)", "docs")
	spec.Register("GET", "/api/docs/{id}/clarification", "The doc's clarification: its rounds of questions with their answers, whether a round is running, and whether it is closed (docs:read)", "docs")
	spec.Register("PUT", "/api/docs/{id}/clarification/questions/{questionId}", "Answer or skip one of the doc's questions; a locked doc still takes answers (docs:write)", "docs")
	spec.Register("DELETE", "/api/docs/{id}/clarification/questions/{questionId}", "Make one of the doc's questions unanswered again (docs:write)", "docs")
	spec.Register("PUT", "/api/docs/{id}/clarification/rounds/{round}/anything-else", "Save or clear a round's Anything else? text (docs:write)", "docs")
	spec.Register("POST", "/api/docs/{id}/clarification/close", "Close the doc's clarification; a new Clarify via AI round reopens it (docs:write and plays:run on that play)", "docs")
	spec.Register("POST", "/api/docs/{id}/lock", "Lock the doc read-only: its title and body refuse edits until unlocked (docs:write)", "docs")
	spec.Register("POST", "/api/docs/{id}/unlock", "Unlock the doc so its title and body can be edited again (docs:write)", "docs")
	spec.Register("POST", "/api/attachments", "Upload a file to a doc or ticket (multipart: file, doc_id|ticket_id)", "attachments")
	spec.Register("GET", "/api/attachments", "List a doc or ticket's attachments", "attachments")
	spec.Register("GET", "/api/attachments/{id}", "Download an attachment", "attachments")
	spec.Register("DELETE", "/api/attachments/{id}", "Delete an attachment", "attachments")
	spec.Register("GET", "/api/tickets", "List tickets", "tickets")
	spec.Register("POST", "/api/tickets", "Create a ticket", "tickets")
	spec.Register("GET", "/api/tickets/{id}", "Get a ticket by id, or by key with ?workspace= naming its workspace", "tickets")
	spec.Register("PATCH", "/api/tickets/{id}/status", "Transition a ticket's status", "tickets")
	spec.Register("POST", "/api/crash-reports", "File or note a crash report a bot posted", "tickets")
	spec.Register("GET", "/api/deploys", "List deploys", "deploys")
	spec.Register("POST", "/api/deploys", "Trigger a deploy", "deploys")
	spec.Register("GET", "/api/deploys/{id}", "Get a deploy", "deploys")
	spec.Register("GET", "/api/deploys/{id}/log", "Get a deploy's timestamped output lines", "deploys")
	spec.Register("POST", "/api/deploys/{id}/cancel", "Cancel a pending/running deploy", "deploys")
	spec.Register("POST", "/api/deploys/{id}/retry", "Start a failed deploy again on the commit it built, with the stack's current settings", "deploys")
	spec.Register("GET", "/api/topology", "Get a workspace's topology canvas; ?workspace= names it, the first workspace when absent", "topology")
	spec.Register("PUT", "/api/topology", "Replace a workspace's topology canvas; ?workspace= names it, the first workspace when absent", "topology")
	spec.Register("GET", "/api/reviews", "List code reviews", "reviews")
	spec.Register("GET", "/api/reviews/{id}", "Get a code review", "reviews")
	spec.Register("GET", "/api/repos/{owner}/{repo}/prs", "List pull requests", "repos")
	spec.Register("POST", "/api/repositories/scan", "Scan a repository's tree for deployable candidates", "repositories")
	spec.Register("GET", "/api/repositories", "List the repositories the caller's own GitHub account can open where the App is installed; ?workspace_id= checks projects:create there, ?q= (3 or more characters) keeps those whose owner/name contains it, ?refresh=1 skips the one-minute cache; 403 github_not_connected or github_reconnect without a live GitHub sign-in", "repositories")
	spec.Register("GET", "/api/repositories/installations", "List the GitHub App's installations the caller's own GitHub account sees, with the caller's workspaces that use each", "repositories")
	spec.Register("GET", "/api/repositories/install-url", "GitHub's page for installing the App on another account; installing links it to no workspace", "repositories")
	spec.Register("PUT", "/api/repositories/installations/{account}/workspaces/{workspaceID}", "Refused since installations are linked by attaching a repository; kept for older clients", "repositories")
	spec.Register("DELETE", "/api/repositories/installations/{account}/workspaces/{workspaceID}", "Detach an account from a workspace, so background work there stops reading its repositories as the App", "repositories")
	spec.Register("GET", "/api/permissions", "List document permission grants", "permissions")
	spec.Register("GET", "/api/permissions/catalog", "List the permission grid roles, tokens, and grants share", "permissions")
	spec.Register("GET", "/api/permissions/explain", "Explain a person's, or with role_id a role's, permissions on a doc, play, or project and what decided each", "permissions")
	spec.Register("GET", "/api/workspaces/{workspaceID}/access", "Read a workspace's Access page: the roles and people each project lists, the roles each public channel is hidden from (Owner only), and its one-time setup state", "permissions")
	spec.Register("PUT", "/api/workspaces/{workspaceID}/access", "Save the Access page in one write: who each named project lists, which roles each named public channel is hidden from, and finish to end the one-time setup", "permissions")
	spec.Register("GET", "/api/mentions/search", "Search mention targets for the @ picker", "mentions")
	spec.Register("POST", "/api/mentions/resolve", "Resolve mention references to live chips", "mentions")
	spec.Register("GET", "/api/projects", "List projects", "projects")
	spec.Register("POST", "/api/projects", "Create a project", "projects")
	spec.Register("GET", "/api/workspaces", "List the caller's workspaces", "workspaces")
	spec.Register("POST", "/api/workspaces", "Create a workspace; needs workspaces:create in any workspace", "workspaces")
	spec.Register("GET", "/api/workspaces/archived", "List the archived workspaces, newest archive first; needs instance:write, the instance Owner's", "workspaces")
	spec.Register("POST", "/api/workspaces/{workspaceID}/archive", "Archive a workspace: it leaves every list, search and live frame for its members until the instance Owner restores it; needs workspaces:delete there", "workspaces")
	spec.Register("POST", "/api/workspaces/{workspaceID}/restore", "Restore an archived workspace for everyone who was in it; needs instance:write, the instance Owner's", "workspaces")
	spec.Register("POST", "/api/workspaces/{workspaceID}/members/{userID}/view-as", "Check the caller may view Nexul as this member, read only, and name them: needs members:view_as, a member who is not an Owner, and roles that give nothing the caller could not give. Other requests then carry X-View-As-User and X-View-As-Workspace; the audit row of this call records who viewed as whom", "workspaces")
	spec.Register("GET", "/api/workspaces/{workspaceID}/people", "List a workspace's people: id, login, display name, picture; any member may read it", "workspaces")
	spec.Register("GET", "/api/people/{userID}/avatar", "Get a person's uploaded picture; the URL comes from the people list", "workspaces")
	spec.Register("GET", "/api/workspaces/{workspaceID}/roles", "List a workspace's roles", "roles")
	spec.Register("POST", "/api/workspaces/{workspaceID}/roles", "Create a custom role", "roles")
	spec.Register("GET", "/api/workspaces/{workspaceID}/roles/{roleID}", "Get a role", "roles")
	spec.Register("PATCH", "/api/workspaces/{workspaceID}/roles/{roleID}", "Rename a role or change its permission mask", "roles")
	spec.Register("DELETE", "/api/workspaces/{workspaceID}/roles/{roleID}", "Delete a custom role", "roles")
	spec.Register("POST", "/api/workspaces/{workspaceID}/roles/{roleID}/clone", "Clone a custom role into another workspace", "roles")
	spec.Register("GET", "/api/automations", "List automations", "automations")
	spec.Register("POST", "/api/automations", "Create a Custom automation and mint its scoped token", "automations")
	spec.Register("GET", "/api/automations/{id}", "Get an automation", "automations")
	spec.Register("PATCH", "/api/automations/{id}/config", "Replace an automation's config values", "automations")
	spec.Register("PATCH", "/api/automations/{id}/enabled", "Enable or disable an automation", "automations")
	spec.Register("DELETE", "/api/automations/{id}", "Delete an automation", "automations")
	spec.Register("POST", "/api/automations/{id}/token", "Rotate an automation's token", "automations")
	spec.Register("DELETE", "/api/automations/{id}/token", "Revoke an automation's token", "automations")
	spec.Register("PATCH", "/api/automations/{id}/host", "Place an automation on an automations host, which takes Deploy to on its machine (null: the instance host)", "automations")
	spec.Register("GET", "/api/automation-hosts", "List the automations hosts on the machines the caller sees", "automation-hosts")
	spec.Register("POST", "/api/automation-hosts/enrollments", "Mint a one-time automations host enrollment code and its install commands; Manage on the machine it names, and the instance Owner for a new machine or none", "automation-hosts")
	spec.Register("POST", "/api/automation-hosts/enroll", "Trade an enrollment code for the automations host's own credential (public)", "automation-hosts")
	spec.Register("DELETE", "/api/automation-hosts/{id}", "Needs Manage on the host's machine: remove an automations host, revoking its credential", "automation-hosts")
	spec.Register("POST", "/api/automation-hosts/self/remove", "Remove the automations host whose credential is the bearer token", "automation-hosts")
	spec.Register("GET", "/api/automation-hosts/self/assignments", "The enabled automations placed on the calling host, each with its worker's token (host credential)", "automation-hosts")
	spec.Register("POST", "/api/automation-hosts/self/logs", "Relay a batch of the calling host's own log records to the log store, stamped with the host's name, machine and version (host credential)", "automation-hosts")
	spec.Register("GET", "/api/runners", "List the runners on the machines the caller sees", "runners")
	spec.Register("POST", "/api/runners/enrollments", "Mint a one-time runner enrollment code and its install commands; Manage on the machine it names, or Add machines for a new one", "runners")
	spec.Register("GET", "/api/runners/enrollments", "List the unused enrollment codes you may revoke: your own, those of machines you manage, and every one for the instance Owner", "runners")
	spec.Register("DELETE", "/api/runners/enrollments/{id}", "Revoke an unused enrollment code you made or manage the machine of; one you may not revoke is not found", "runners")
	spec.Register("POST", "/api/runners/enroll", "Trade an enrollment code for the runner's own credential (public)", "runners")
	spec.Register("DELETE", "/api/runners/{id}", "Needs Manage on the runner's machine: remove a runner, revoking its credential", "runners")
	spec.Register("POST", "/api/runners/self/remove", "Remove the runner whose credential is the bearer token", "runners")
	spec.Register("GET", "/api/runners/download/{target}", "Download the runner binary for a target (runner credential)", "runners")
	spec.Register("GET", "/api/runners/latest-version", "Latest published runner version", "runners")
	spec.Register("GET", "/api/machines", "List the machines the caller sees", "machines")
	spec.Register("PATCH", "/api/machines/{id}", "Needs Manage on the machine: rename it or change its stack root", "machines")
	spec.Register("DELETE", "/api/machines/{id}", "Needs Delete on the machine: refused, naming each, while a stack or gateway targets it or a runner on it is connected", "machines")
	spec.Register("POST", "/api/machines/{id}/discover", "Needs Manage on the machine: scan its containers and networks", "machines")
	spec.Register("POST", "/api/machines/{id}/import", "Adopt ticked discovery selections as unmanaged stacks", "machines")
	spec.Register("GET", "/api/machines/deployable", "List the machines the caller may Deploy to, the choices for where a stack runs", "machines")
	spec.Register("GET", "/api/dns/zones", "List DNS zones", "dns")
	spec.Register("GET", "/api/dns/service-hostnames", "List service hostnames", "dns")
	spec.Register("POST", "/api/dns/tunnels", "Create a Cloudflare tunnel", "dns")
	spec.Register("GET", "/api/dns/tunnels", "List Cloudflare tunnels", "dns")
	spec.Register("POST", "/api/dns/tunnels/{tunnelID}/route", "Route a hostname into a tunnel", "dns")
	spec.Register("POST", "/api/dns/tunnels/{tunnelID}/rotate", "Rotate tunnel credentials", "dns")
	spec.Register("DELETE", "/api/dns/tunnels/{tunnelID}", "Delete a tunnel", "dns")
	spec.Register("POST", "/api/dns/gateways", "Create a gateway (tunnel or proxy)", "dns")
	spec.Register("POST", "/api/dns/instance-proxy", "Route a domain to this server through Traefik with Let's Encrypt", "dns")
	spec.Register("GET", "/api/dns/resolve", "Resolve a domain's current addresses", "dns")
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
	spec.Register("POST", "/api/chat/channels", "Create a channel, public or private", "chat")
	spec.Register("POST", "/api/chat/dms", "Create a direct-message conversation", "chat")
	spec.Register("POST", "/api/chat/tickets/{ticketID}/thread", "Get or lazily create a ticket's thread", "chat")
	spec.Register("GET", "/api/chat/tickets/thread-status", "Batch-check which tickets have a thread", "chat")
	spec.Register("GET", "/api/chat/conversations/{id}/messages", "List a conversation's messages", "chat")
	spec.Register("POST", "/api/chat/conversations/{id}/messages", "Post a message", "chat")
	spec.Register("PATCH", "/api/chat/messages/{id}", "Edit a message (author only)", "chat")
	spec.Register("DELETE", "/api/chat/messages/{id}", "Delete a message (author only)", "chat")
	spec.Register("PUT", "/api/chat/messages/{id}/note", "Replace a note's markdown file, resetting its live editing room", "chat")
	spec.Register("POST", "/api/chat/conversations/{id}/read", "Mark a conversation read", "chat")
	spec.Register("GET", "/api/chat/unread", "Per-conversation unread counts", "chat")
	spec.Register("POST", "/api/chat/voice-channels", "Create a voice channel", "chat")
	spec.Register("PATCH", "/api/chat/conversations/{id}", "Rename a channel or voice channel", "chat")
	spec.Register("DELETE", "/api/chat/conversations/{id}", "Delete a channel or voice channel and its messages", "chat")
	spec.Register("PUT", "/api/chat/conversations/{id}/private", "Make a channel private, keeping the named members, or public", "chat")
	spec.Register("POST", "/api/chat/conversations/{id}/members", "Add members to a private channel", "chat")
	spec.Register("DELETE", "/api/chat/conversations/{id}/members/{userID}", "Remove a member from a private channel", "chat")
	spec.Register("POST", "/api/chat/conversations/{id}/leave", "Leave a private channel", "chat")
	spec.Register("GET", "/api/workspaces/{workspaceID}/meetings", "The caller's meetings between ?from and ?to (RFC 3339, at most 62 days apart) and each time they take place", "calendar")
	spec.Register("POST", "/api/workspaces/{workspaceID}/meetings", "Create a meeting the caller organises (calendar:write); its invitees are told", "calendar")
	spec.Register("GET", "/api/meetings/{id}", "One meeting the caller organises or is invited to, with its invitees and their replies", "calendar")
	spec.Register("PATCH", "/api/meetings/{id}", "Change, cancel or restore a meeting, or with slot and scope one occurrence or it and every later one (the organiser with calendar:write, or calendar:delete)", "calendar")
	spec.Register("PUT", "/api/meetings/{id}/reply", "Accept or decline a meeting the caller is invited to", "calendar")
	spec.Register("GET", "/api/conversations/{id}/botwebhooks", "List a conversation's bots (botwebhook:read); URLs only with botwebhook:write; ?deleted=true lists the deleted ones (botwebhook:write)", "botwebhooks")
	spec.Register("POST", "/api/conversations/{id}/botwebhooks", "Create a bot with a name and an avatar (botwebhook:write); ten live bots per conversation", "botwebhooks")
	spec.Register("PATCH", "/api/botwebhooks/{id}", "Rename a bot, change or clear its avatar, regenerate its URL, or restore it with a new URL via deleted false (botwebhook:write)", "botwebhooks")
	spec.Register("DELETE", "/api/botwebhooks/{id}", "Delete a bot: its URL stops working at once and its messages stay (botwebhook:delete)", "botwebhooks")
	spec.Register("GET", "/api/botwebhooks/{id}/avatar", "A bot's own avatar, to anyone who reads its conversation", "botwebhooks")
	spec.Register("POST", "/api/botwebhooks/{id}/{token}", "Post as a bot with Discord's execute-webhook JSON; the token is the credential; ?wait=true returns the message", "botwebhooks")
	spec.Register("POST", "/api/voice/{conversationID}/token", "Mint a LiveKit join token for a voice channel", "voice")
	spec.Register("POST", "/api/voice/{conversationID}/leave", "Leave a voice channel (removes optimistic presence)", "voice")
	spec.Register("GET", "/api/voice/occupancy", "Current voice channel occupancy", "voice")
	spec.Register("POST", "/api/agent/conversations/{id}/interrupt", "Stop the in-flight agent turn on a conversation", "agent")
	spec.Register("POST", "/api/agent/conversations/{id}/answer", "Answer the question the agent turn on a conversation is waiting on", "agent")
	spec.Register("POST", "/api/agent/conversations/{id}/secret", "Save or decline the secret the agent turn on a conversation asked for", "agent")
	spec.Register("POST", "/api/agent/conversations/{id}/steer", "Hand a message whose agent turn waits in the queue to the turn under way", "agent")
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
	spec.Register("GET", "/api/templates", "List every instance template, the code default where none was edited; any signed-in member", "templates")
	spec.Register("GET", "/api/templates/{kind}", "Get one template; ?key= names a play or ticket type, ?scope= with workspace_id or project_id reads it below the instance", "templates")
	spec.Register("PUT", "/api/templates/{kind}", "Needs templates:write: replace an instance template's text; a role body with no key adds a template role", "templates")
	spec.Register("DELETE", "/api/templates/{kind}", "Needs templates:write: reset an instance template to its code default; ?key= names a play or ticket type", "templates")
	spec.Register("POST", "/api/templates/clone", "Copy a template's text from one layer over another; read where the source lives, write where the target does", "templates")
	spec.Register("POST", "/api/templates/reset", "Reset a template at a layer: the instance to its code default, a workspace or project to the instance's text", "templates")
	spec.Register("POST", "/api/templates/delete", "Needs templates:write: delete a role template the instance added; the shipped ones only reset", "templates")
	spec.Register("GET", "/api/audit", "Audit log: changes made through the API and over MCP, kept 45 days; reads are not recorded", "audit")
}

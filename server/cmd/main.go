package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/version"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/voice"
	"github.com/otal-labs/nexul/internal/workspace"
)

func main() {
	cfg := mustLoadConfig()
	logger, flushLogs, err := logging.NewWithOTLP(context.Background(), cfg.LogLevel, logging.OTLP{
		Endpoint: cfg.OTLPEndpoint,
		User:     cfg.OTLPUser,
		Token:    cfg.OTLPToken,
		Stream:   "nexul",
		Service:  "nexul-server",
	})
	if err != nil {
		fail(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := flushLogs(ctx); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "flush shipped logs:", err) // best-effort diagnostic; process is exiting regardless
		}
	}()
	logger.Info("server starting", "version", version.Version, "http_addr", cfg.HTTPAddr, "ws_addr", cfg.WSAddr, "mcp_addr", cfg.MCPAddr, "otlp_endpoint", cfg.OTLPEndpoint)

	store, encKey := bootstrapStore(cfg)
	defer func() {
		if err := store.Close(); err != nil {
			logger.Error("close store", "error", err)
		}
	}()

	bus := bootstrapBus(logger, store)
	defer func() {
		if err := bus.Close(); err != nil {
			logger.Error("close bus", "error", err)
		}
	}()

	svc := wireCoreServices(cfg, store, encKey, bus, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := svc.automationSeeder.Seed(ctx); err != nil {
		fail(fmt.Errorf("seed default automations: %w", err))
	}

	wireDomainEventSubscriptions(ctx, bus, svc, logger)
	wireIntegrationFanout(ctx, bus, store, svc, logger)
	wsHandler, runnerSvc, runnerHTTP, automationsDialin := startBackgroundWorkers(ctx, cfg, store, bus, svc, logger)
	// runner is constructed above, after deploy (wireCoreServices); closing the machine-import cycle here
	// mirrors deploy's own dns seams (SetGatewayLookup/SetExposureManager), just wired one phase later.
	svc.deploySvc.SetMachineLookup(deployMachineLookupAdapter{runner: runnerSvc})
	svc.deploySvc.SetMachineDiscoverer(deployMachineDiscovererAdapter{runner: runnerSvc})
	liveHub, agentHandler := wireLiveHubAndAgent(ctx, bus, store, svc, logger)

	wsServer, httpServer, mcpServerHTTP := buildRoutes(cfg, bus, store, svc, wsHandler, runnerSvc, runnerHTTP, automationsDialin, liveHub, agentHandler, logger)

	go serveHTTP(mcpServerHTTP, logger, stop)
	go serveHTTP(wsServer, logger, stop)
	go serveHTTP(httpServer, logger, stop)

	if err := wsHandler.Run(ctx); err != nil {
		fail(fmt.Errorf("ws handler: %w", err))
	}
	automationsDialin.CloseAll("shutdown")

	shutdownServers(wsServer, mcpServerHTTP, httpServer)
	logger.Info("server stopped")
}

// requestUserID prefers the session/PAT user over an integration-scoped actor; an empty id fails the caller's check.
func requestUserID(r *http.Request) string {
	if u := auth.UserFromCtx(r.Context()); u != nil {
		return u.ID
	}
	if a, ok := identity.ActorFromCtx(r.Context()); ok {
		return a.ID
	}
	return ""
}

// withUserID injects the user id via setCtx (each domain's own WithUserID) so writes attribute without importing auth.
func withUserID(setCtx func(context.Context, string) context.Context) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := requestUserID(r)
			if userID == "" {
				httpx.WriteError(w, apperrs.ErrUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(setCtx(r.Context(), userID)))
		})
		return httpx.WithRoutes(wrapped, httpx.RoutesOf(next))
	}
}

// resolveAuditActor prefers a token identity over a user id; unauthenticated requests aren't audited.
func resolveAuditActor(ctx context.Context) (actorType, actorID, tokenID string) {
	if aa := automations.AutomationFromCtx(ctx); aa != nil {
		return "automation", aa.Automation.ID, aa.Automation.TokenPrefix
	}
	if ia := integrations.IntegrationFromCtx(ctx); ia != nil {
		return "integration", ia.Install.ID, ia.Token.ID
	}
	if u := auth.UserFromCtx(ctx); u != nil {
		return "user", u.ID, ""
	}
	return "", "", ""
}

// currentUserID reads the user id RequireAuth already resolved into the request context.
func currentUserID(r *http.Request) string {
	if u := auth.UserFromCtx(r.Context()); u != nil {
		return u.ID
	}
	return ""
}

func serveHTTP(srv *http.Server, logger *slog.Logger, stop context.CancelFunc) {
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("http server failed", "addr", srv.Addr, "error", err)
		stop()
	}
}

// mountGateway registers both the exact prefix and its trailing-slash subtree; a bare path alone 307s to the slash.
func mountGateway(mux *httpx.ServeMux, prefix string, h http.Handler) {
	if prefix == "/api" || strings.HasPrefix(prefix, "/api/") {
		if httpx.RoutesOf(h) == nil {
			panic("API gateway handlers must expose tracked routes")
		}
	}
	mux.Mount(prefix, h)
}

// withIdentity attaches the user as the acting identity so permission checks (docs, access) see who is calling.
func withIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := auth.UserFromCtx(r.Context()); u != nil {
			r = r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: u.ID, CanCreateWorkspace: u.CanCreateWorkspace}))
		}
		next.ServeHTTP(w, r)
	})
}

// mcpActor resolves the acting user for MCP tool calls; RequireAuth guarantees every request carries one.
func mcpActor(ctx context.Context) identity.Actor {
	if u := auth.UserFromCtx(ctx); u != nil {
		return identity.Actor{ID: u.ID, CanCreateWorkspace: u.CanCreateWorkspace}
	}
	return identity.Actor{}
}

// livePushTopics are the bus topics the browser socket bridges onto. Append-only: new streams add topics here.
var livePushTopics = []string{
	docs.TopicCreated,
	docs.TopicUpdated,
	runner.TopicRunnerConnected,
	runner.TopicRunnerDisconnected,
	runner.TopicDeployBuildStarted,
	runner.TopicDeployBuildProgress,
	runner.TopicDeployBuildCompleted,
	runner.TopicDeployDeployProgress,
	runner.TopicDeployStatusChanged,
	deploy.TopicDeployUpdated,
	deploy.TopicStackCreated,
	deploy.TopicStackUpdated,
	deploy.TopicStackDeleted,
	dns.TopicRecordChanged,
	dns.TopicTunnelChanged,
	dns.TopicGatewayChanged,
	dns.TopicExposureChanged,
	workspace.TopicNotificationCreated,
	workspace.TopicCategoryCreated,
	workspace.TopicCategoryUpdated,
	workspace.TopicCategoryDeleted,
	workspace.TopicTicketCategoryChanged,
	tickets.TopicCreated,
	tickets.TopicUpdated,
	tickets.TopicStatusChanged,
	tickets.TopicAssigneeChanged,
	tickets.TopicFinished,
	workspace.TopicTicketTypeCreated,
	workspace.TopicTicketTypeUpdated,
	workspace.TopicTicketTypeDeleted,
	workspace.TopicStatusCreated,
	workspace.TopicStatusUpdated,
	workspace.TopicStatusDeleted,
	chat.TopicConversationCreated,
	chat.TopicMessageCreated,
	chat.TopicMessageUpdated,
	chat.TopicMessageDeleted,
	voice.TopicOccupancyChanged,
	pairing.TopicSetupConfirmed,
	pairing.TopicSetupUnconfirmed,
}

func fail(err error) {
	_, _ = fmt.Fprintln(os.Stderr, "server:", err) // best-effort diagnostic; exit code carries the real result
	os.Exit(1)
}

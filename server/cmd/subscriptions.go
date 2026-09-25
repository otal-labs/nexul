package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/eventcatalog"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/mcp"
	"github.com/otal-labs/nexul/internal/memories"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/live"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/topology"
	"github.com/otal-labs/nexul/internal/workspace"
)

// mustSubscribe registers a fan-out handler, exiting on failure like every other startup wiring step.
func mustSubscribe(ctx context.Context, bus *inprocess.Bus, consumer, topic, note string, h eventbus.Handler) {
	if err := bus.SubscribeWithConsumer(ctx, consumer, topic, h); err != nil {
		fail(fmt.Errorf("subscribe %s%s: %w", topic, note, err))
	}
}

// topologyStackCreatedHandler anchors one topology node per container the event carries — pending ones for a
// stack the wizard just created, observed ones for containers adopted by machine import — with the row's own
// status and address; a redelivered event on an already-present node is a no-op. Renamed from
// topologyServiceCreatedHandler; still subscribed on the "service.created" topic (deploy.TopicStackCreated).
func topologyStackCreatedHandler(topoSvc *topology.Service) eventbus.Handler {
	return func(ctx context.Context, ev eventbus.Event) error {
		var e deploy.StackEvent
		if err := json.Unmarshal(ev.Payload, &e); err != nil {
			return apperrs.Fatal(fmt.Errorf("parse %s: %w", deploy.TopicStackCreated, err))
		}
		for _, svc := range e.Services {
			name := e.Stack.Name
			if len(e.Services) > 1 {
				name = e.Stack.Name + "/" + svc.Name
			}
			status, address := containerNodeStatus(&svc, deploy.StatusPending)
			_, err := topoSvc.AddNode(ctx, topology.DefaultEnvironment, topology.Node{
				ID:   svc.ID,
				Type: topology.NodeService,
				Data: topology.NodeData{
					ServiceID: svc.ID,
					Name:      name,
					Runtime:   "docker",
					Status:    status,
					Address:   address,
				},
			})
			if err != nil && !errors.Is(err, apperrs.ErrConflict) {
				return err
			}
		}
		return nil
	}
}

// topologyStackUpdatedHandler propagates a stack rename onto every one of its auto-managed container nodes.
// Renamed from topologyServiceUpdatedHandler.
func topologyStackUpdatedHandler(deploySvc *deploy.Service, topoSvc *topology.Service) eventbus.Handler {
	return func(ctx context.Context, ev eventbus.Event) error {
		var e deploy.StackEvent
		if err := json.Unmarshal(ev.Payload, &e); err != nil {
			return apperrs.Fatal(fmt.Errorf("parse %s: %w", deploy.TopicStackUpdated, err))
		}
		services, err := deploySvc.ListServices(ctx, e.Stack.ID)
		if err != nil {
			return err
		}
		for _, svc := range services {
			name := e.Stack.Name
			if len(services) > 1 {
				name = e.Stack.Name + "/" + svc.Name
			}
			if _, err := topoSvc.RenameServiceNode(ctx, topology.DefaultEnvironment, svc.ID, name); err != nil {
				return err
			}
		}
		return nil
	}
}

// topologyStackDeletedHandler drops every one of the deleted stack's container nodes/edges; a redelivered
// event on an already-missing node is a no-op. Renamed from topologyServiceDeletedHandler.
func topologyStackDeletedHandler(topoSvc *topology.Service) eventbus.Handler {
	return func(ctx context.Context, ev eventbus.Event) error {
		var e deploy.StackDeletedEvent
		if err := json.Unmarshal(ev.Payload, &e); err != nil {
			return apperrs.Fatal(fmt.Errorf("parse %s: %w", deploy.TopicStackDeleted, err))
		}
		for _, id := range e.ServiceIDs {
			if _, err := topoSvc.RemoveServiceNode(ctx, topology.DefaultEnvironment, id); err != nil && !errors.Is(err, apperrs.ErrNotFound) {
				return err
			}
		}
		return nil
	}
}

// topologyDeployStatusHandler drives every one of the deploying stack's container nodes' badges. HandleStatusChanged
// reconciles the runner's observation report into the services table first, so each node gets its own container's
// status and address; a container the report never reached (e.g. a failed deploy with
// no report at all) falls back to the deploy's own terminal status, same as before per-container observation.
func topologyDeployStatusHandler(deploySvc *deploy.Service, topoSvc *topology.Service) eventbus.Handler {
	return func(ctx context.Context, ev eventbus.Event) error {
		if err := deploySvc.HandleStatusChanged(ctx, ev); err != nil {
			return err
		}
		var e deploy.DeployStatusChangedEvent
		if err := json.Unmarshal(ev.Payload, &e); err != nil {
			return apperrs.Fatal(fmt.Errorf("parse %s: %w", deploy.TopicDeployStatusChanged, err))
		}
		d, err := deploySvc.Get(ctx, e.ID)
		if err != nil {
			return err
		}
		services, err := deploySvc.ListServices(ctx, d.StackID)
		if err != nil {
			return err
		}
		for _, svc := range services {
			status, address := containerNodeStatus(svc, deploy.Status(d.Status))
			if _, err := topoSvc.SetServiceNodeStatus(ctx, topology.DefaultEnvironment, svc.ID, status, address); err != nil {
				return err
			}
		}
		return nil
	}
}

// containerNodeStatus maps one observed container onto its node's badge and address.
func containerNodeStatus(svc *deploy.Container, deployStatus deploy.Status) (topology.ServiceStatus, string) {
	address := ""
	if len(svc.Networks) > 0 {
		address = svc.Networks[0].Address
	}
	switch svc.Status {
	case deploy.ServiceStatusHealthy:
		return topology.ServiceHealthy, address
	case deploy.ServiceStatusRunning:
		return topology.ServiceRunning, address
	case deploy.ServiceStatusStopped:
		return topology.ServiceStopped, address
	case deploy.ServiceStatusExited:
		return topology.ServiceFailed, address
	default: // pending: the report never reached this container, so reflect the deploy's own terminal status.
		if deployStatus == deploy.StatusFailed {
			return topology.ServiceFailed, address
		}
		return topology.ServiceRunning, address
	}
}

// wireDomainEventSubscriptions must run before wireIntegrationFanout starts the outbox relay.
func wireDomainEventSubscriptions(ctx context.Context, bus *inprocess.Bus, svc *coreServices, logger *slog.Logger) {
	// These dedupe via processed_events; PR merge/close completion is owned by tickets, which evaluates ticket.finished.
	linker := ticketLinker{svc: svc.ticketsSvc}
	mustSubscribe(ctx, bus, "tickets.link_pr", gitprovider.TopicPROpened, "", func(ctx context.Context, ev eventbus.Event) error {
		return gitprovider.HandlePROpened(ctx, linker, ev)
	})
	mustSubscribe(ctx, bus, "codereview.mirror", gitprovider.TopicPROpened, "", func(ctx context.Context, ev eventbus.Event) error {
		return codereview.HandlePROpened(ctx, svc.reviewSvc, ev)
	})
	mustSubscribe(ctx, bus, "codereview.mirror", gitprovider.TopicPRReviewSubmitted, "", func(ctx context.Context, ev eventbus.Event) error {
		return codereview.HandlePRReviewSubmitted(ctx, svc.reviewSvc, ev)
	})
	mustSubscribe(ctx, bus, "tickets.complete", gitprovider.TopicPRMerged, "", svc.ticketsSvc.HandlePRMerged)
	mustSubscribe(ctx, bus, "codereview.mirror", gitprovider.TopicPRMerged, "", func(ctx context.Context, ev eventbus.Event) error {
		return codereview.HandlePRMerged(ctx, svc.reviewSvc, ev)
	})
	mustSubscribe(ctx, bus, "tickets.close", gitprovider.TopicPRClosed, "", svc.ticketsSvc.HandlePRClosed)
	mustSubscribe(ctx, bus, "codereview.mirror", gitprovider.TopicPRClosed, "", func(ctx context.Context, ev eventbus.Event) error {
		return codereview.HandlePRClosed(ctx, svc.reviewSvc, ev)
	})

	// deploy owns these directly: branch deploy rules are service config, not automation code.
	mustSubscribe(ctx, bus, "deploy.branch_push", gitprovider.TopicPush, "", svc.deploySvc.HandlePush)
	mustSubscribe(ctx, bus, "deploy.branch_teardown", gitprovider.TopicBranchDeleted, "", svc.deploySvc.HandleBranchDeleted)

	// Stack lifecycle events drive the auto-managed nodes; consumers are idempotent (see the handlers' own docs above).
	mustSubscribe(ctx, bus, "topology.auto_manage", deploy.TopicStackCreated, "", topologyStackCreatedHandler(svc.topoSvc))
	mustSubscribe(ctx, bus, "topology.auto_manage", deploy.TopicStackUpdated, "", topologyStackUpdatedHandler(svc.deploySvc, svc.topoSvc))
	mustSubscribe(ctx, bus, "topology.auto_manage", deploy.TopicStackDeleted, "", topologyStackDeletedHandler(svc.topoSvc))

	// Deploy terminal states drive the anchored node's badge.
	mustSubscribe(ctx, bus, "deploy.live_status", deploy.TopicDeployStatusChanged, "", topologyDeployStatusHandler(svc.deploySvc, svc.topoSvc))
	mustSubscribe(ctx, bus, "deploy.log", deploy.TopicDeployLog, "", svc.deploySvc.HandleLog)

	// Notification IDs derive from the source event, so redeliveries are idempotent.
	mustSubscribe(ctx, bus, "notifications", tickets.TopicCreated, "", func(ctx context.Context, ev eventbus.Event) error {
		return workspace.HandleTicketCreated(ctx, svc.notifSvc, ev)
	})
	mustSubscribe(ctx, bus, "notifications", tickets.TopicStatusChanged, "", func(ctx context.Context, ev eventbus.Event) error {
		return workspace.HandleTicketStatusChanged(ctx, svc.notifSvc, ev)
	})
	mustSubscribe(ctx, bus, "notifications", docs.TopicCreated, "", func(ctx context.Context, ev eventbus.Event) error {
		return workspace.HandleDocCreated(ctx, svc.notifSvc, ev)
	})
	mustSubscribe(ctx, bus, "notifications", docs.TopicUpdated, "", func(ctx context.Context, ev eventbus.Event) error {
		return workspace.HandleDocUpdated(ctx, svc.notifSvc, ev)
	})
	mustSubscribe(ctx, bus, "notifications", memories.TopicUpdated, "", func(ctx context.Context, ev eventbus.Event) error {
		return workspace.HandleMemoryUpdated(ctx, svc.notifSvc, ev)
	})
	mustSubscribe(ctx, bus, "notifications", plays.TopicRunFinished, "", func(ctx context.Context, ev eventbus.Event) error {
		return workspace.HandlePlayRunFinished(ctx, svc.notifSvc, ev)
	})
	mustSubscribe(ctx, bus, "notifications", plays.TopicRunWaiting, "", func(ctx context.Context, ev eventbus.Event) error {
		return workspace.HandlePlayRunWaiting(ctx, svc.notifSvc, ev)
	})
}

// wireIntegrationFanout subscribes the fan-out per topic, starts the delivery relay, and publishes the catalog.
func wireIntegrationFanout(ctx context.Context, bus *inprocess.Bus, store *storage.Store, svc *coreServices, logger *slog.Logger) {
	fanout := integrations.NewFanoutHandler(svc.integrationsSvc)
	for _, topic := range eventcatalog.AllTopics() {
		mustSubscribe(ctx, bus, "integrations.fanout", topic, "", fanout.HandleEvent)
	}
	deliveryRelay := integrations.NewRelay(store.IntegrationDeliveries, integrations.RelayConfig{Logger: logger})
	go func() {
		if err := deliveryRelay.Run(ctx); err != nil {
			logger.Error("integration delivery relay stopped", "error", err)
		}
	}()
	if err := svc.integrationsSvc.PublishCatalog(ctx); err != nil {
		fail(fmt.Errorf("publish event schema catalog: %w", err))
	}
}

// wireLiveHubAndAgent builds the browser live-events hub and wires the @Agent chat pipeline that publishes to it.
func wireLiveHubAndAgent(ctx context.Context, bus *inprocess.Bus, store *storage.Store, svc *coreServices, logger *slog.Logger) (*live.Hub, *agent.Handler) {
	// The hub is generic; future streams add their own topics to livePushTopics.
	liveHub := live.New(logger)
	for _, topic := range livePushTopics {
		mustSubscribe(ctx, bus, "live.push", topic, " for live push", func(ctx context.Context, ev eventbus.Event) error {
			return liveHub.Publish(ctx, ev.Topic, ev.Payload)
		})
	}

	// In-progress reply text pushes to liveHub as ephemeral frames; only the final reply is durable, via PostAgentReply.
	agentSvc := agent.NewService(agent.Config{
		Conversations: agentConversations{svc: svc.chatSvc, projects: svc.workspaceSvc},
		Targets:       svc.pairingSvc,
		Harnesses:     svc.harnesses,
		Tickets:       agentTicketReader{svc: svc.ticketsSvc},
		Docs:          agentDocReader{svc: svc.docsSvc},
		Users:         agentUserReader{users: store.Users},
		Memories:      agentMemories{svc: svc.memoriesSvc},
		Attachments:   agentAttachmentReader{svc: svc.attachmentsSvc},
		Live:          liveHub,
		Logger:        logger,
	})
	mustSubscribe(ctx, bus, "agent.mention", chat.TopicMessageCreated, " for agent pipeline", agentSvc.HandleMessageCreated)
	agentHandler := agent.NewHandler(agentSvc)
	// A play run is an Agent turn with a trail (ADR 0055), so the runner sits on the same pipeline the mentions use.
	svc.playsRunner = plays.NewRunner(plays.RunnerConfig{
		Plays:       store.Plays,
		Trails:      store.PlayTrails,
		Perm:        playsPermissionGate{svc: svc.accessSvc},
		Targets:     playsTargetReader{tickets: svc.ticketsSvc, docs: svc.docsSvc, workspace: svc.workspaceSvc},
		Projects:    playsProjectLookup{memoriesProjectLookup{svc: svc.workspaceSvc}},
		Harness:     playsHarnessResolver{svc: svc.pairingSvc},
		Memories:    playsMemoryReader{svc: svc.memoriesSvc},
		Threads:     playsThreads{svc: svc.chatSvc},
		Turns:       agentSvc,
		Tickets:     playsStatusMover{svc: svc.ticketsSvc},
		Live:        liveHub,
		Users:       agentUserReader{users: store.Users},
		Links:       playsLinkReader{tickets: svc.ticketsSvc, docs: agentDocReader{svc: svc.docsSvc}},
		Attachments: agentAttachmentReader{svc: svc.attachmentsSvc},
		Logger:      logger,
	})

	// A ticket entering done fires the built-in decisions check on the mover's or the developer's harness.
	mustSubscribe(ctx, bus, "plays.decisions_check", tickets.TopicStatusChanged, "", svc.playsRunner.HandleTicketStatusChanged)

	// A bad frame here only costs one live patch, so missing/wrong-shaped payloads are dropped rather than fatal.
	mustSubscribe(ctx, bus, "topology.live_canvas", topology.TopicUpdated, " for live push", func(ctx context.Context, ev eventbus.Event) error {
		var e topology.UpdatedEvent
		if err := json.Unmarshal(ev.Payload, &e); err != nil {
			return apperrs.Fatal(fmt.Errorf("parse %s: %w", topology.TopicUpdated, err))
		}
		return liveHub.Publish(ctx, "topology", e.Canvas)
	})

	return liveHub, agentHandler
}

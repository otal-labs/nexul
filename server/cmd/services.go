package main

import (
	"context"
	"log/slog"

	"net/http"
	"time"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/collab"
	"github.com/otal-labs/nexul/internal/connectors"
	cfoauth "github.com/otal-labs/nexul/internal/connectors/cloudflare"
	githuboauth "github.com/otal-labs/nexul/internal/connectors/github"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/dns/cloudflare"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/mentions"
	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/presence"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/t3client"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/topology"
	"github.com/otal-labs/nexul/internal/voice"
	"github.com/otal-labs/nexul/internal/workspace"
)

// coreServices bundles every wired domain service so later startup phases take one param instead of dozens.
type coreServices struct {
	accessSvc      *access.Service
	docsSvc        *docs.Service
	memoriesSvc    *memories.Service
	attachmentsSvc *attachments.Service
	collabHub      *collab.Hub

	ticketsSvc  *tickets.Service
	mentionsSvc *mentions.Service
	topoSvc     *topology.Service
	deploySvc   *deploy.Service
	reviewSvc   *codereview.Service

	automationsSvc        *automations.Service
	automationVersionsSvc *automations.VersionsService
	automationSecretsSvc  *automations.SecretsService
	automationSeeder      *automations.Seeder
	automationRunsSvc     *automations.RunsService

	authSvc           *auth.Service
	authHandler       *auth.Handler
	invitationSvc     *tenancy.InvitationService
	invitationHandler *tenancy.InvitationHandler

	connectorsSvc     *connectors.Service
	connectorsHandler *connectors.Handler

	dnsSvc     *dns.Service
	dnsHandler *dns.Handler

	pairingSvc     *pairing.Service
	harnesses      harness.Registry
	presenceKeeper *presence.Keeper
	pairingHandler *pairing.Handler

	rolesSvc *roles.Service
	playsSvc *plays.Service
	// playsRunner is wired in wireLiveHubAndAgent: it needs the agent pipeline, which needs the live hub.
	playsRunner *plays.Runner

	chatSvc             *chat.Service
	voiceSvc            *voice.Service
	voiceHandler        *voice.Handler
	voiceWebhookHandler *voice.WebhookHandler

	tenancySvc   *tenancy.Service
	workspaceSvc *workspace.Service

	gitRouter         gitProviderRouter
	repositoryScanner repositoryScanner

	integrationsSvc *integrations.Service

	notifSvc *workspace.NotificationService
}

// wireCoreServices constructs every domain service; order matters since some close cycles via Set*/gateway calls.
func wireCoreServices(cfg *config.Config, store *storage.Store, encKey []byte, bus *inprocess.Bus, logger *slog.Logger) *coreServices {
	accessSvc := access.NewService(store.Access, accessUsers{store.Users})
	docsSvc := docs.NewService(store.Docs, accessSvc)
	attachmentsSvc := attachments.NewService(store.Attachments, accessSvc, memoryAttachmentsAccessGate{memories: store.Memories, access: accessSvc})

	// The hub relays/persists Y.js updates and commits via the docs use-case layer (ADR 0017 seam, collab never imports docs).
	collabHub := collab.NewHub(logger, store.Collab, accessSvc, collabDocWriter{docsSvc})
	ticketsSvc := tickets.NewService(store.Tickets, store.Statuses, workspaceUserStore{users: store.Users})
	mentionsSvc := mentions.New(mentions.Config{
		Tickets:     mentionTicketSource{repo: store.Tickets},
		Docs:        mentionDocSource{repo: store.Docs},
		Statuses:    mentionStatusSource{repo: store.Statuses},
		Access:      accessSvc,
		Projects:    mentionProjectSource{repo: store.Projects},
		TicketTypes: mentionTicketTypeSource{repo: store.TicketTypes},
	})
	topoSvc := topology.NewService(store.Topology)
	deploySvc := deploy.NewService(store.Deploys, store.Stacks, store.Services, deployProjectStore{projects: store.Projects})
	reviewSvc := codereview.NewService(store.CodeReviews)
	automationsSvc := automations.NewService(store.Automations, automationPermissionGate{svc: accessSvc})
	// DefaultDefinitions supplies the board pair's bundled default automation code.
	automationVersionsSvc := automations.NewVersionsService(store.AutomationVersions, store.Automations, automationPermissionGate{svc: accessSvc})
	automationSecretsSvc := automations.NewSecretsService(store.AutomationSecrets, automationPermissionGate{svc: accessSvc})
	automationSeeder := automations.NewSeeder(store.Automations, automationVersionsSvc, automations.DefaultDefinitions(), logger)
	automationRunsSvc := automations.NewRunsService(store.AutomationRuns, automationPermissionGate{svc: accessSvc})
	if cfg.DevLogin {
		logger.Warn("DEV AUTH BYPASS ENABLED — /auth/dev-login mints sessions with no GitHub round trip; never set NEXUL_DEV_LOGIN in production")
	}
	authSvc := auth.NewService(auth.Config{
		Secret:        []byte(cfg.AuthSecret),
		SPAOrigin:     cfg.SPAOrigin,
		Users:         store.Users,
		OAuthHandoffs: store.OAuthHandoffs,
		Allowlist:     store.Allowlist,
		Settings:      store.Settings,
		PATs:          store.PATs,
		MentionLayout: mentionLayoutGate{svc: accessSvc},
		ConnectorApps: connectorAppSeederGate{store: store.ConnectorAppConfig},
		GitHubApp:     githubAppVerifierGate{hc: &http.Client{Timeout: 15 * time.Second}},
		DevLogin:      cfg.DevLogin,
	})
	authHandler := auth.NewHandler(authSvc)
	invitationSvc := tenancy.NewInvitationService(store.Invitations, authSvc)
	invitationHandler := tenancy.NewInvitationHandler(invitationSvc)
	authSvc.SetInvitationGate(invitationAuthGate{svc: invitationSvc})
	// Built before dnsSvc since dns's Cloudflare token comes from connectorsSvc; livekit gets a Verifier, not OAuth.
	connectorsRegistry := connectors.Registry()
	wireConnectorOAuth(connectorsRegistry, store)
	connectorsSvc := connectors.NewService(connectors.Config{
		Store:          store.Connectors,
		AppConfigStore: store.ConnectorAppConfig,
		Owner:          instanceAdminGate{svc: authSvc},
		Settings:       dnsSettingsAdapter{store.Settings},
		Registry:       connectorsRegistry,
	})
	connectorsHandler := connectors.NewHandler(connectorsSvc)
	dnsSvc := dns.NewService(dns.Config{
		Repo: store.DNS,
		NewProvider: func(_ context.Context, token string) (dns.DNSProvider, error) {
			return cloudflare.New(token), nil
		},
		NewTunnelProvider: func(_ context.Context, token string) (dns.TunnelProvider, error) {
			return cloudflare.New(token), nil
		},
		NewAccessProvider: func(_ context.Context, token string) (dns.AccessProvider, error) {
			return cloudflare.New(token), nil
		},
		Tokens:        dnsCloudflareTokenAdapter{connectors: connectorsSvc},
		EncryptionKey: encKey,
		Settings:      dnsSettingsAdapter{store.Settings},
		Provisioner:   dnsProvisioner{deploy: deploySvc},
		Containers:    dnsContainerLookupAdapter{deploy: deploySvc},
	})
	dnsHandler := dns.NewHandler(dnsSvc)
	// deploy needs dns, dns needs deploy's Containers/Provisioner, so neither builds the other in its constructor.
	deploySvc.SetGatewayLookup(deployGatewayLookupAdapter{dns: dnsSvc})
	deploySvc.SetExposureManager(deployExposureAdapter{dns: dnsSvc})
	deploySvc.SetGatewayJoin(deployGatewayJoinAdapter{dns: dnsSvc})
	deploySvc.SetGatewayAdopter(deployGatewayAdopterAdapter{dns: dnsSvc})
	// One client per harness kind; the real ones talk to the user's own machines, never reachable in tests.
	harnesses := harness.Registry{harness.KindT3Code: t3client.NewHarness(t3client.Options{Logger: logger})}
	var presenceKeeper *presence.Keeper // constructed below; pairing only fires the callback after requests start flowing
	pairingSvc := pairing.NewService(pairing.Config{
		Repo:               store.Pairing,
		Harnesses:          harnesses,
		EncryptionKey:      encKey,
		OnComputersChanged: func(userID string) { presenceKeeper.Refresh(userID) },
	})
	presenceKeeper = presence.New(presence.Config{
		Sessions:  pairingSvc.ActiveSessions,
		Harnesses: harnesses,
		Logger:    logger,
	})
	pairingHandler := pairing.NewHandler(pairingSvc)
	// tenancy and roles need each other, so roles starts with a nil member gate, closed via SetMemberGate below.
	rolesSvc := roles.NewService(store.Roles, nil)
	chatSvc := chat.NewService(store.Chat)
	// voice never imports chat or connectors directly (ADR 0017); ephemeral occupancy publishes onto bus, not the outbox.
	voiceSvc := voice.NewService(voice.Config{
		Conversations: voiceConversations{svc: chatSvc},
		Credentials:   voiceCredentials{svc: connectorsSvc},
		Users:         voiceUserNames{users: store.Users},
		Bus:           bus,
	})
	voiceHandler := voice.NewHandler(voiceSvc)
	voiceWebhookHandler := voice.NewWebhookHandler(voiceSvc, voiceCredentials{svc: connectorsSvc}, logger)
	playsSvc := plays.NewService(store.Plays, playsPermissionGate{svc: accessSvc})
	tenancySvc := tenancy.NewService(store.Workspaces, store.WorkspaceMembers, store.WorkspaceInvites, roleGate{svc: rolesSvc}, instanceAdminGate{svc: authSvc}, roleNameGate{svc: rolesSvc}, workspacePermissionGate{svc: accessSvc}, allowlistGate{svc: authSvc}, userLookupGate{svc: authSvc}, channelGate{svc: chatSvc}, playsGate{svc: playsSvc})
	rolesSvc.SetMemberGate(roleMemberGate{svc: tenancySvc})
	authSvc.SetDefaultWorkspace(defaultWorkspaceGate{svc: tenancySvc})
	authSvc.SetPendingInviteResolver(pendingInviteResolverGate{svc: tenancySvc})
	workspaceSvc := workspace.NewService(store.Projects, store.Categories, store.TicketTypes, store.Statuses, instanceAdminGate{svc: authSvc}, workspaceGate{svc: tenancySvc})
	memoriesSvc := memories.NewService(store.Memories, memoriesPermissionGate{svc: accessSvc}, memoriesProjectLookup{svc: workspaceSvc}, memoriesAttachmentsGate{svc: attachmentsSvc}, memoriesMembershipGate{members: store.WorkspaceMembers})
	// HasPermission's role-mask layer needs both roles and tenancy, wired only after the cycle above closes.
	accessSvc.SetRoles(accessRoleResolver{tenancy: tenancySvc, roles: rolesSvc})
	// Docs are project-scoped, so this resolver needs workspaceSvc to exist first.
	accessSvc.SetDocWorkspaces(accessDocWorkspaceResolver{docs: store.Docs, projects: workspaceSvc})
	accessSvc.SetPlayWorkspaces(accessPlayWorkspaceResolver{plays: store.Plays})
	// accessSvc.Can already matches chat.DocAccess's shape (ADR 0017 seam), so it wires in directly.
	chatSvc.SetDocAccess(accessSvc)
	// gitRouter resolves per-repo since different projects' repos can live on different git hosts.
	gitRouter := gitProviderRouter{workspace: workspaceSvc, connectors: connectorsSvc, appConfigs: store.ConnectorAppConfig}
	repoScanner := repositoryScanner{git: gitRouter, appConfigs: store.ConnectorAppConfig}
	integrationsSvc := integrations.NewService(integrations.Config{
		Installs:   store.IntegrationInstalls,
		Tokens:     store.IntegrationTokens,
		Subs:       store.IntegrationSubs,
		Deliveries: store.IntegrationDeliveries,
		Schemas:    store.EventSchemas,
		Audit:      store.Audit,
		Owner:      instanceAdminGate{svc: authSvc},
	})
	// ScopeAllows is injected as a function value rather than automations importing integrations (ADR 0017 seam rule).
	automationsSvc.SetGateway(integrations.ScopeAllows, integrations.ResolveScopes, instanceAdminGate{svc: authSvc})

	notifSvc := workspace.NewNotificationService(store.Notifications, workspaceUserStore{users: store.Users}, workspaceMembersStore{members: store.WorkspaceMembers}, notificationPermissionGate{svc: accessSvc})

	return &coreServices{
		accessSvc:      accessSvc,
		docsSvc:        docsSvc,
		memoriesSvc:    memoriesSvc,
		attachmentsSvc: attachmentsSvc,
		collabHub:      collabHub,

		ticketsSvc:  ticketsSvc,
		mentionsSvc: mentionsSvc,
		topoSvc:     topoSvc,
		deploySvc:   deploySvc,
		reviewSvc:   reviewSvc,

		automationsSvc:        automationsSvc,
		automationVersionsSvc: automationVersionsSvc,
		automationSecretsSvc:  automationSecretsSvc,
		automationSeeder:      automationSeeder,
		automationRunsSvc:     automationRunsSvc,

		authSvc:           authSvc,
		authHandler:       authHandler,
		invitationSvc:     invitationSvc,
		invitationHandler: invitationHandler,

		connectorsSvc:     connectorsSvc,
		connectorsHandler: connectorsHandler,

		dnsSvc:     dnsSvc,
		dnsHandler: dnsHandler,

		pairingSvc:     pairingSvc,
		harnesses:      harnesses,
		presenceKeeper: presenceKeeper,
		pairingHandler: pairingHandler,

		rolesSvc: rolesSvc,
		playsSvc: playsSvc,

		chatSvc:             chatSvc,
		voiceSvc:            voiceSvc,
		voiceHandler:        voiceHandler,
		voiceWebhookHandler: voiceWebhookHandler,

		tenancySvc:   tenancySvc,
		workspaceSvc: workspaceSvc,

		gitRouter:         gitRouter,
		repositoryScanner: repoScanner,

		integrationsSvc: integrationsSvc,

		notifSvc: notifSvc,
	}
}

// wireConnectorOAuth attaches OAuth clients/verifiers to registry entries that need one; others keep the nil default.
func wireConnectorOAuth(registry []connectors.Connector, store *storage.Store) {
	for i, c := range registry {
		if c.ID == "github" {
			registry[i].OAuth = githuboauth.New("github", store.ConnectorAppConfig, dnsSettingsAdapter{store.Settings}, nil)
		}
		if c.ID == "cloudflare" {
			registry[i].OAuth = cfoauth.New("cloudflare", store.ConnectorAppConfig, dnsSettingsAdapter{store.Settings}, nil)
			registry[i].Verify = cfoauth.NewTokenVerifier(nil)
		}
		if c.ID == "livekit" {
			registry[i].Verify = livekitVerifier{}
		}
	}
}

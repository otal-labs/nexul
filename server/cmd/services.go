package main

import (
	"context"
	"log/slog"

	"net/http"
	"path/filepath"
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
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/presence"
	"github.com/otal-labs/nexul/internal/push"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/t3client"
	"github.com/otal-labs/nexul/internal/t3clientv2"
	"github.com/otal-labs/nexul/internal/t3rpc"
	"github.com/otal-labs/nexul/internal/templates"
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
	notesHub       *collab.Hub

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
	automationHostsSvc    *automations.HostsService

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
	templatesSvc *templates.Service

	gitRouter         gitProviderRouter
	repoHooks         repoWebhooks
	repositoryScanner repositoryScanner

	integrationsSvc *integrations.Service

	notifSvc   *workspace.NotificationService
	pushSender *push.Sender
}

// wireCoreServices constructs every domain service; order matters since some close cycles via Set*/gateway calls.
func wireCoreServices(cfg *config.Config, store *storage.Store, encKey []byte, bus *inprocess.Bus, logger *slog.Logger) *coreServices {
	accessSvc := access.NewService(store.Access, accessUsers{store.Users})
	attachmentsSvc := attachments.NewService(store.Attachments, accessSvc, memoryAttachmentsAccessGate{memories: store.Memories, access: accessSvc})
	attachmentsSvc.SetTicketAccess(projectEntityGate{access: accessSvc, projects: store.Projects, tickets: store.Tickets})
	docsSvc := docs.NewService(store.Docs, accessSvc, docsAttachmentsGate{svc: attachmentsSvc})

	// The hub relays/persists Y.js updates and commits via the docs use-case layer (ADR 0017 seam, collab never imports docs).
	collabHub := collab.NewHub(logger, store.Collab, accessSvc, collabDocWriter{docsSvc}, permissions.DocsWrite, permissions.DocsRead)
	docsSvc.SetLiveSessions(collabHub)
	ticketsSvc := tickets.NewService(store.Tickets, store.Statuses, workspaceUserStore{users: store.Users})
	ticketsSvc.SetTicketTypes(store.TicketTypes)
	ticketsSvc.SetGate(accessSvc)
	ticketsSvc.SetSourceDocs(ticketSourceDocs{svc: docsSvc})
	mentionsSvc := mentions.New(mentions.Config{
		Tickets:     mentionTicketSource{repo: store.Tickets},
		Docs:        mentionDocSource{repo: store.Docs},
		Statuses:    mentionStatusSource{repo: store.Statuses},
		Access:      accessSvc,
		Projects:    mentionProjectSource{repo: store.Projects},
		TicketTypes: mentionTicketTypeSource{repo: store.TicketTypes},
	})
	// Built before dnsSvc since dns's Cloudflare token comes from connectorsSvc; livekit gets a Verifier, not OAuth.
	connectorsRegistry := connectors.Registry()
	wireConnectorOAuth(connectorsRegistry, store)
	connectorsSvc := connectors.NewService(connectors.Config{
		Store:          store.Connectors,
		AppConfigStore: store.ConnectorAppConfig,
		Gate:           accessSvc,
		Settings:       dnsSettingsAdapter{store.Settings},
		Registry:       connectorsRegistry,
	})
	connectorsHandler := connectors.NewHandler(connectorsSvc)
	repoHooks := repoWebhooks{
		git:         gitProviderRouter{workspace: store.Projects, connectors: connectorsSvc, appConfigs: store.ConnectorAppConfig},
		instanceURL: dnsSettingsAdapter{store.Settings}.GetInstanceURL,
		secret:      githubWebhookSecret(cfg.AuthSecret),
	}
	projects := hookedProjects{Repo: store.Projects, hooks: repoHooks}
	topoSvc := topology.NewService(store.Topology)
	deploySvc := deploy.NewService(store.Deploys, store.Stacks, store.Services, deployProjectStore{projects: projects})
	topoSvc.SetGate(accessSvc)
	deploySvc.SetGate(accessSvc)
	reviewSvc := codereview.NewService(store.CodeReviews)
	reviewSvc.SetGate(projectEntityGate{access: accessSvc, projects: store.Projects, tickets: store.Tickets})
	automationsSvc := automations.NewService(store.Automations, accessSvc)
	// DefaultDefinitions supplies the board pair's bundled default automation code.
	automationVersionsSvc := automations.NewVersionsService(store.AutomationVersions, store.Automations, accessSvc)
	automationSecretsSvc := automations.NewSecretsService(store.AutomationSecrets, accessSvc)
	automationSeeder := automations.NewSeeder(store.Automations, automationVersionsSvc, automations.DefaultDefinitions(), store.EventWorkspaces, tenancy.DefaultWorkspaceID, logger)
	automationRunsSvc := automations.NewRunsService(store.AutomationRuns, store.Automations, accessSvc)
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
		Sessions:      store.Sessions,
		ConnectorApps: connectorAppSeederGate{store: store.ConnectorAppConfig},
		GitHubApp:     githubAppVerifierGate{hc: &http.Client{Timeout: 15 * time.Second}},
		DevLogin:      cfg.DevLogin,
		SetupCodes:    store.SetupCodes,
		ConnectCodes:  store.ConnectCodes,
		EnrollDir:     filepath.Join(filepath.Dir(cfg.DBPath), "enroll"),
		Local:         cfg.Local,
		Permissions:   accessSvc,
	})
	authHandler := auth.NewHandler(authSvc)
	invitationSvc := tenancy.NewInvitationService(store.Invitations, authSvc)
	invitationHandler := tenancy.NewInvitationHandler(invitationSvc)
	authSvc.SetInvitationGate(invitationAuthGate{svc: invitationSvc})
	dnsSvc := dns.NewService(dns.Config{
		Repo: store.DNS,
		NewProvider: func(_ context.Context, token string) (dns.DNSProvider, error) {
			return cloudflare.New(token), nil
		},
		NewTunnelProvider: func(_ context.Context, token, accountID string) (dns.TunnelProvider, error) {
			return cloudflare.New(token, cloudflare.WithAccountID(accountID)), nil
		},
		NewAccessProvider: func(_ context.Context, token, accountID string) (dns.AccessProvider, error) {
			return cloudflare.New(token, cloudflare.WithAccountID(accountID)), nil
		},
		Tokens:         dnsCloudflareTokenAdapter{connectors: connectorsSvc},
		EncryptionKey:  encKey,
		Settings:       dnsSettingsAdapter{store.Settings},
		Provisioner:    dnsProvisioner{deploy: deploySvc},
		Containers:     dnsContainerLookupAdapter{deploy: deploySvc},
		InstanceOrigin: instanceOrigin(cfg.HTTPAddr),
		Placement:      dnsInstancePlacement{runners: store.Runners, machines: store.Machines},
	})
	dnsSvc.SetGate(accessSvc)
	dnsHandler := dns.NewHandler(dnsSvc)
	// deploy needs dns, dns needs deploy's Containers/Provisioner, so neither builds the other in its constructor.
	deploySvc.SetGatewayLookup(deployGatewayLookupAdapter{dns: dnsSvc})
	deploySvc.SetExposureManager(deployExposureAdapter{dns: dnsSvc})
	deploySvc.SetGatewayJoin(deployGatewayJoinAdapter{dns: dnsSvc})
	deploySvc.SetGatewayAdopter(deployGatewayAdopterAdapter{dns: dnsSvc})
	// One client per harness kind; the real ones talk to the user's own machines, never reachable in tests.
	harnessHTTP := &http.Client{Transport: &cloudflare.AccessTransport{
		Credentials: computerTunnelAccess{hosts: store.Pairing, dns: dnsSvc}.Credentials,
	}}
	harnessOpts := t3rpc.Options{Logger: logger, HTTPClient: harnessHTTP}
	harnesses := harness.Registry{
		harness.KindT3Code:   t3client.NewHarness(harnessOpts),
		harness.KindT3CodeV2: t3clientv2.NewHarness(harnessOpts),
	}
	var presenceKeeper *presence.Keeper // constructed below; pairing only fires the callback after requests start flowing
	pairingSvc := pairing.NewService(pairing.Config{
		Repo:               store.Pairing,
		Harnesses:          harnesses,
		EncryptionKey:      encKey,
		OnComputersChanged: func(userID string) { presenceKeeper.Refresh(userID) },
		Tunnels:            pairingTunnels{dns: dnsSvc},
		Bus:                bus,
		Tokens:             pairingMCPTokens{auth: authSvc},
		Instance:           dnsSettingsAdapter{store.Settings},
		Projects:           accessSvc,
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
	tenancySvc := tenancy.NewService(store.Workspaces, store.WorkspaceMembers, store.WorkspaceInvites, roleGate{svc: rolesSvc}, accessSvc, roleNameGate{svc: rolesSvc}, workspacePermissionGate{svc: accessSvc}, allowlistGate{svc: authSvc}, userLookupGate{svc: authSvc}, channelGate{svc: chatSvc}, workspaceDefaultsGate{plays: playsSvc, automations: automationSeeder}, accountGate{svc: authSvc, presence: presenceKeeper})
	tenancySvc.SetProjects(tenancyProjectGate{projects: store.Projects, access: accessSvc})
	rolesSvc.SetMemberGate(roleMemberGate{svc: tenancySvc})
	mentionsSvc.SetPeople(mentionPeopleSource{svc: tenancySvc})
	rolesSvc.SetPermissionGate(workspacePermissionGate{svc: accessSvc})
	authSvc.SetDefaultWorkspace(defaultWorkspaceGate{svc: tenancySvc})
	authSvc.SetPendingInviteResolver(pendingInviteResolverGate{svc: tenancySvc})
	workspaceSvc := workspace.NewService(projects, store.Categories, store.TicketTypes, store.Statuses, accessSvc, workspaceGate{svc: tenancySvc})
	workspaceSvc.SetTicketProjects(workspaceTicketProjects{tickets: store.Tickets})
	memoriesSvc := memories.NewService(store.Memories, accessSvc, memoriesProjectLookup{projects: store.Projects}, memoriesAttachmentsGate{svc: attachmentsSvc})
	templatesSvc := wireTemplates(store, accessSvc, memoriesSvc, tenancySvc, playsSvc, workspaceSvc)
	// HasPermission's role-mask layer needs both roles and tenancy, wired only after the cycle above closes.
	accessSvc.SetRoles(accessRoleResolver{tenancy: tenancySvc, roles: rolesSvc})
	accessSvc.SetDocWorkspaces(accessDocWorkspaceResolver{docs: store.Docs, projects: store.Projects})
	accessSvc.SetScopes(accessScopes{projects: store.Projects, members: store.WorkspaceMembers})
	accessSvc.SetPlayWorkspaces(accessPlayWorkspaceResolver{plays: store.Plays})
	// accessSvc.Can already matches chat.DocAccess's shape (ADR 0017 seam), so it wires in directly.
	chatSvc.SetDocAccess(accessSvc)
	chatSvc.SetGate(accessSvc)
	chatSvc.SetMembership(membershipGate{members: store.WorkspaceMembers})
	chatSvc.SetThreadGate(chatThreadGate{projectEntityGate{access: accessSvc, projects: store.Projects, tickets: store.Tickets}})
	chatSvc.SetStanding(chatStanding{roles: accessRoleResolver{tenancy: tenancySvc, roles: rolesSvc}})
	attachmentsSvc.SetConversations(chatAttachmentConversations{svc: chatSvc})
	// A note's file is its only stored state, so its live rooms keep theirs in memory; readers never join (ADR 0110).
	notesHub := collab.NewHub(logger, collab.NewMemoryStore(), collabNoteRooms{chatSvc}, collabNoteRooms{chatSvc}, permissions.TicketsWrite, permissions.TicketsWrite)
	chatSvc.SetNoteLive(notesHub)
	ticketsSvc.SetProjectPeople(ticketProjectPeople{users: userLookupGate{svc: authSvc}, access: accessSvc})
	ticketsSvc.SetTesting(tickets.Testing{
		Stages:  ticketStages{statuses: store.Statuses},
		Threads: ticketThreads{chat: chatSvc, projects: store.Projects},
		Targets: ticketTestTargets{deploy: deploySvc},
	})
	// gitRouter resolves per-repo since different projects' repos can live on different git hosts.
	gitRouter := gitProviderRouter{workspace: workspaceSvc, connectors: connectorsSvc, appConfigs: store.ConnectorAppConfig}
	repoScanner := newRepositoryScanner(gitRouter, store.ConnectorAppConfig)
	integrationsSvc := integrations.NewService(integrations.Config{
		Installs:   store.IntegrationInstalls,
		Tokens:     store.IntegrationTokens,
		Subs:       store.IntegrationSubs,
		Deliveries: store.IntegrationDeliveries,
		Schemas:    store.EventSchemas,
		Audit:      store.Audit,
		Perms:      accessSvc,
	})
	// ScopeAllows is injected as a function value rather than automations importing integrations (ADR 0017 seam rule).
	automationsSvc.SetGateway(integrations.ScopeAllows, integrations.ResolveScopes)
	// Its own key, derived from the auth secret, signs the tokens host workers dial in with.
	automationHostsSvc := automations.NewHostsService(store.AutomationHosts, store.Automations, crypto.DeriveKey("nexul automations host token key:"+cfg.AuthSecret)).
		WithGate(accessSvc).
		WithInstanceURL(dnsSettingsAdapter{store.Settings}).
		WithEnrollDir(filepath.Join(filepath.Dir(cfg.DBPath), "enroll"))
	automationsSvc.SetHosts(automationHostsSvc)

	notifSvc := workspace.NewNotificationService(store.Notifications, workspaceUserStore{users: store.Users}, workspaceMembersStore{members: store.WorkspaceMembers}, notificationPermissionGate{svc: accessSvc}, store.Projects).
		WithDocWatchers(docWatchersAdapter{repo: store.Docs})
	notifSvc.SetTicketProjects(workspaceTicketProjects{tickets: store.Tickets})
	pushSender := push.New(push.Config{
		Tokens:     store.Sessions,
		Workspaces: pushWorkspaceNamer{workspaces: store.Workspaces},
		Instance:   dnsSettingsAdapter{store.Settings},
		Logger:     logger,
	})

	return &coreServices{
		accessSvc:      accessSvc,
		docsSvc:        docsSvc,
		memoriesSvc:    memoriesSvc,
		attachmentsSvc: attachmentsSvc,
		collabHub:      collabHub,
		notesHub:       notesHub,

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
		automationHostsSvc:    automationHostsSvc,

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
		templatesSvc: templatesSvc,

		gitRouter:         gitRouter,
		repoHooks:         repoHooks,
		repositoryScanner: repoScanner,

		integrationsSvc: integrationsSvc,

		notifSvc:   notifSvc,
		pushSender: pushSender,
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

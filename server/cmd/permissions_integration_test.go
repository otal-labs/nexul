package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/platform/config"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/release"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/repository"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// The people every case is asked about, members of the default workspace except the outsider.
const (
	uOwner     = "u-owner"
	uReader    = "u-reader"
	uWriter    = "u-writer"
	uPlain     = "u-plain"
	uOverwrite = "u-overwrite"
	uOutsider  = "u-outsider"
	// uSteward holds every instance-level permission through a role, uClerk only accounts:read and accounts:write,
	// and uManager may edit roles and members but holds no instance-level permission.
	uSteward = "u-steward"
	uClerk   = "u-clerk"
	uManager = "u-manager"
	// uCloner may clone docs but not write them, uEditor may write but not clone, and uCopier may do both.
	uCloner = "u-cloner"
	uEditor = "u-editor"
	uCopier = "u-copier"
)

var people = []string{uOwner, uReader, uWriter, uPlain, uOverwrite, uOutsider, uSteward, uClerk, uManager, uCloner, uEditor, uCopier}

// instanceBits is every permission that used to take the instance administrator flag (ADR 0088).
var instanceBits = []string{"instance:read", "instance:write", "accounts:read", "accounts:write", "accounts:delete",
	"workspaces:create", "runners:write", "runners:delete", "automations:write", "automations:delete",
	"connectors:write", "integrations:read", "integrations:write", "integrations:delete", "audit:read"}

// outcomes a case can have: allowed, refused as forbidden, refused as not found, or left out of a list.
const (
	ok        = "ok"
	invalid   = "400"
	forbidden = "403"
	notFound  = "404"
	conflict  = "409"
	hidden    = "hidden"
)

var errHidden = errors.New("left out of the list")

func outcome(err error) string {
	switch {
	case err == nil:
		return ok
	case errors.Is(err, errHidden):
		return hidden
	case errors.Is(err, apperrs.ErrNotFound):
		return notFound
	case errors.Is(err, apperrs.ErrForbidden):
		return forbidden
	case errors.Is(err, apperrs.ErrConflict):
		return conflict
	case errors.Is(err, apperrs.ErrInvalid):
		return invalid
	}
	return err.Error()
}

func as(userID string) context.Context {
	return withTestMemo(identity.WithActor(context.Background(), identity.Actor{ID: userID}))
}

func contains[T any](items []T, err error, match func(T) bool) error {
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(items, match) {
		return errHidden
	}
	return nil
}

// present reports a batch read's entry for id, or errHidden when the read left it out.
func present[V any](m map[string]V, err error, id string) error {
	if err != nil {
		return err
	}
	if _, ok := m[id]; !ok {
		return errHidden
	}
	return nil
}

// noPRs and noRepos stand in for the git host, so an allowed call ends at an empty answer instead of the network.
type noPRs struct{ gitprovider.GitProvider }

func (noPRs) ListPRs(context.Context, string, string, gitprovider.PROpts) ([]*gitprovider.PR, error) {
	return nil, nil
}

type noRepos struct{ repository.Scanner }

func (noRepos) ListInstallationRepos(context.Context, bool) ([]repository.Repo, error) {
	return nil, nil
}

func grant(actions ...string) permissions.Set {
	out := make([]permissions.Action, len(actions))
	for i, a := range actions {
		out[i] = permissions.Action(a)
	}
	return permissions.SetOf(out...)
}

type permFixture struct {
	svc     *coreServices
	store   *storage.Store
	ticket  *tickets.Ticket
	doc     string
	channel *chat.Conversation
	dm      *chat.Conversation
	stack   string
	infra   string
	deploy  string
	review  string
}

// newPermFixture wires the real composition root over a fresh database and seeds one of everything in the default
// workspace, plus the people above: the Owner, a reader and a writer role, a member with no grants, a member whose
// workspace-wide overwrite grants the reads, and someone outside the workspace.
// newWired wires the real composition root over a fresh database, before anyone has signed in.
func newWired(t *testing.T) (*coreServices, *storage.Store) {
	t.Helper()
	return wiredOver(t, mentionsTestDB(t))
}

// wiredOver wires the composition root over db, already migrated and seeded with the general project.
func wiredOver(t *testing.T, db *sql.DB) (*coreServices, *storage.Store) {
	t.Helper()
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: store.ProcessedEvents, DeadLetterStore: store.DeadLetters})
	t.Cleanup(func() { _ = bus.Close() })
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "nexul.db"), HTTPAddr: "127.0.0.1:0", AuthSecret: "0123456789abcdef0123456789abcdef"}
	svc := wireCoreServices(cfg, store, []byte("0123456789abcdef0123456789abcdef"), bus, logger)
	useTestMemoOf(t, svc.accessSvc)
	return svc, store
}

func newPermFixture(t *testing.T) permFixture {
	t.Helper()
	svc, store := newWired(t)
	return seedPermFixture(t, svc, store)
}

func seedPermFixture(t *testing.T, svc *coreServices, store *storage.Store) permFixture {
	t.Helper()
	ctx := context.Background()

	for _, id := range people {
		_, _, err := store.Users.UpsertUser(ctx, &auth.Identity{UserID: id, Provider: auth.ProviderGitHub, ProviderUserID: id, Login: id})
		require.NoError(t, err)
	}
	reads := []string{"tickets:read", "stacks:read", "deploys:read", "topology:read", "docs:read", "memories:read", "dns:read",
		"machines:read", "connectors:read", "reviews:read", "repos:read"}
	now := time.Now()
	for _, r := range []*roles.Role{
		{ID: "role-owner", WorkspaceID: "workspace-default", Name: "Owner", IsOwnerRole: true},
		{ID: "role-reader", WorkspaceID: "workspace-default", Name: "Reader", Permissions: grant(reads...)},
		{ID: "role-writer", WorkspaceID: "workspace-default", Name: "Writer", Permissions: grant("tickets:read", "tickets:write", "projects:write", "docs:write", "chat:write", "channels:write")},
		{ID: "role-plain", WorkspaceID: "workspace-default", Name: "Member"},
		{ID: "role-steward", WorkspaceID: "workspace-default", Name: "Steward", Permissions: grant(instanceBits...)},
		{ID: "role-clerk", WorkspaceID: "workspace-default", Name: "Clerk", Permissions: grant("accounts:read", "accounts:write")},
		{ID: "role-manager", WorkspaceID: "workspace-default", Name: "Manager", Permissions: grant("roles:write", "members:write")},
		{ID: "role-cloner", WorkspaceID: "workspace-default", Name: "Cloner", Permissions: grant("docs:read", "docs:clone")},
		{ID: "role-editor", WorkspaceID: "workspace-default", Name: "Editor", Permissions: grant("docs:read", "docs:write")},
		{ID: "role-copier", WorkspaceID: "workspace-default", Name: "Copier", Permissions: grant("docs:read", "docs:clone", "docs:write")},
	} {
		r.CreatedAt, r.UpdatedAt = now, now
		require.NoError(t, store.Roles.Create(ctx, r))
	}
	for user, role := range map[string]string{uOwner: "role-owner", uReader: "role-reader", uWriter: "role-writer", uPlain: "role-plain", uOverwrite: "role-plain",
		uSteward: "role-steward", uClerk: "role-clerk", uManager: "role-manager", uCloner: "role-cloner", uEditor: "role-editor", uCopier: "role-copier"} {
		require.NoError(t, store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: "workspace-default", RoleID: role, CreatedAt: now}))
	}
	require.NoError(t, store.Access.Set(ctx, "workspace", "workspace-default", uOverwrite, grant(reads...), nil))

	f := permFixture{svc: svc, store: store, doc: "doc-1", stack: "stack-1", infra: "stack-gateway", deploy: "deploy-1", review: "review-1"}
	var err error
	f.ticket, err = svc.ticketsSvc.Create(ctx, "project-general", "Login times out", "", "", "")
	require.NoError(t, err)
	require.NoError(t, store.Docs.Create(ctx, &docs.Doc{ID: f.doc, ProjectID: "project-general", Title: "Spec", Body: `{"type":"doc","content":[]}`, Version: 1, CreatedAt: now, UpdatedAt: now}))
	f.channel, err = svc.chatSvc.CreateChannel(ctx, "workspace-default", uOwner, "eng")
	require.NoError(t, err)
	f.dm, err = svc.chatSvc.CreateDM(ctx, "workspace-default", uOwner, []string{uWriter})
	require.NoError(t, err)
	require.NoError(t, store.Stacks.Create(ctx, &deploy.Stack{ID: f.stack, ProjectID: "project-general", Name: "web", Slug: "web", Machine: "m1", Strategy: deploy.StrategyRun, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, store.Stacks.Create(ctx, &deploy.Stack{ID: f.infra, Name: "cloudflared", Slug: "cloudflared", Machine: "m1", Strategy: deploy.StrategyRun, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, store.Deploys.Create(ctx, &deploy.Deploy{ID: f.deploy, StackID: f.stack, Service: "web", Status: deploy.StatusHealthy, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, store.Projects.AddRepo(ctx, "project-general", workspace.RepoRef{Owner: "acme", Name: "app", FullName: "acme/app", ConnectorID: "github", Role: workspace.RepoRoleApp}))
	require.NoError(t, store.Tickets.LinkPR(ctx, f.ticket.ID, tickets.PRRef{Owner: "acme", Repo: "app", Number: 7}, tickets.PRStateOpen))
	require.NoError(t, store.CodeReviews.Create(ctx, &codereview.CodeReview{ID: f.review, Repo: "acme/app", PRNumber: 7, Status: codereview.StatusPending, CreatedAt: now}))
	_, err = svc.chatSvc.GetOrCreateTicketThread(ctx, "workspace-default", f.ticket.ID, uOwner)
	require.NoError(t, err)
	return f
}

// TestIntegration_PermissionTable walks each domain's reads, lists, and creates as every kind of caller, through the
// wired use-cases the HTTP gateway and MCP tools both call.
func TestIntegration_PermissionTable(t *testing.T) {
	f := newPermFixture(t)
	s := f.svc
	machines := runner.NewService(f.store.Runners, nil).WithMachines(f.store.Machines).WithGate(s.accessSvc)
	upgrades := runner.NewService(noopRunnerRepo{}, fakeUpgradeDispatch{}).
		WithInstall(runner.InstallConfig{Release: release.New(release.Config{APIBase: fakeReleaseServer(t, "v0.2.1").URL})}).
		WithUpgrades(newMemUpgradeRepo()).WithBus(noopPublisher{}).WithGate(s.accessSvc)
	entities := projectEntityGate{access: s.accessSvc, projects: f.store.Projects, tickets: f.store.Tickets}
	instance := map[string]string{uOwner: ok, uSteward: ok, uPlain: forbidden, uManager: forbidden, uOutsider: forbidden}
	cases := []struct {
		name string
		call func(ctx context.Context) error
		want map[string]string
	}{
		{"tickets: get", func(ctx context.Context) error {
			_, err := s.ticketsSvc.Get(ctx, f.ticket.ID)
			return err
		}, map[string]string{uOwner: ok, uReader: ok, uOverwrite: ok, uPlain: forbidden, uOutsider: notFound}},
		{"tickets: list", func(ctx context.Context) error {
			ts, err := s.ticketsSvc.List(ctx)
			return contains(ts, err, func(t *tickets.Ticket) bool { return t.ID == f.ticket.ID })
		}, map[string]string{uOwner: ok, uReader: ok, uOverwrite: ok, uPlain: hidden, uOutsider: hidden}},
		{"tickets: search", func(ctx context.Context) error {
			rs, err := s.ticketsSvc.Search(ctx, "login", 10)
			return contains(rs, err, func(r tickets.SearchResult) bool { return r.ID == f.ticket.ID })
		}, map[string]string{uOwner: ok, uReader: ok, uPlain: hidden, uOutsider: hidden}},
		{"tickets: create", func(ctx context.Context) error {
			_, err := s.ticketsSvc.Create(ctx, "project-general", "New", "", "", "")
			return err
		}, map[string]string{uOwner: ok, uWriter: ok, uReader: forbidden, uOutsider: notFound}},
		{"tickets: delete", func(ctx context.Context) error {
			return s.ticketsSvc.Delete(ctx, f.ticket.ID)
		}, map[string]string{uWriter: forbidden, uOutsider: notFound}},
		{"projects: create", func(ctx context.Context) error {
			actor, _ := identity.ActorFromCtx(ctx)
			_, err := s.workspaceSvc.Create(ctx, "workspace-default", "Mobile", "MOB"+actor.ID[2:4], workspace.ProjectIcon(""))
			return err
		}, map[string]string{uOwner: ok, uWriter: ok, uReader: forbidden, uPlain: forbidden, uOutsider: notFound}},
		{"projects: list", func(ctx context.Context) error {
			_, err := s.workspaceSvc.List(ctx, "workspace-default")
			return err
		}, map[string]string{uPlain: ok, uOutsider: notFound}},
		{"docs: create", func(ctx context.Context) error {
			_, err := s.docsSvc.Create(ctx, "project-general", "Plan", "")
			return err
		}, map[string]string{uOwner: ok, uWriter: ok, uPlain: forbidden, uOutsider: notFound}},
		{"docs: list", func(ctx context.Context) error {
			ds, err := s.docsSvc.List(ctx)
			return contains(ds, err, func(d *docs.DocListItem) bool { return d.ID == f.doc })
		}, map[string]string{uPlain: ok, uOutsider: hidden}},
		{"docs: clone", func(ctx context.Context) error {
			_, err := s.docsSvc.Clone(ctx, f.doc, "project-general")
			return err
		}, map[string]string{uOwner: ok, uCopier: ok, uEditor: forbidden, uCloner: forbidden, uWriter: forbidden, uOutsider: notFound}},
		{"docs: create a folder", func(ctx context.Context) error {
			_, err := s.docsSvc.CreateFolder(ctx, "project-general", "Folder "+actorID(ctx))
			return err
		}, map[string]string{uOwner: ok, uEditor: ok, uReader: forbidden, uOutsider: notFound}},
		{"docs: delete a folder", func(ctx context.Context) error {
			folder, err := s.docsSvc.CreateFolder(context.Background(), "project-general", "Doomed "+actorID(ctx))
			require.NoError(t, err)
			return s.docsSvc.DeleteFolder(ctx, folder.ID)
		}, map[string]string{uOwner: ok, uEditor: ok, uReader: forbidden, uOutsider: notFound}},
		{"docs: move a doc to a folder", func(ctx context.Context) error {
			folder, err := s.docsSvc.CreateFolder(context.Background(), "project-general", "Target "+actorID(ctx))
			require.NoError(t, err)
			_, err = s.docsSvc.MoveToFolder(ctx, f.doc, folder.ID)
			return err
		}, map[string]string{uOwner: ok, uEditor: ok, uReader: forbidden, uOutsider: notFound}},
		{"channels: create one", func(ctx context.Context) error {
			actor, _ := identity.ActorFromCtx(ctx)
			_, err := s.chatSvc.CreateChannel(ctx, "workspace-default", actor.ID, "room-"+actor.ID)
			return err
		}, map[string]string{uOwner: ok, uWriter: ok, uPlain: forbidden, uOutsider: notFound}},
		{"channels: rename one", func(ctx context.Context) error {
			c, err := s.chatSvc.CreateVoiceChannel(context.Background(), "workspace-default", uOwner, "rename-"+actorID(ctx))
			require.NoError(t, err)
			_, err = s.chatSvc.RenameChannel(ctx, c.ID, "renamed-"+actorID(ctx))
			return err
		}, map[string]string{uOwner: ok, uWriter: ok, uPlain: forbidden, uOutsider: notFound}},
		{"channels: delete one", func(ctx context.Context) error {
			c, err := s.chatSvc.CreateChannel(context.Background(), "workspace-default", uOwner, "delete-"+actorID(ctx))
			require.NoError(t, err)
			_, err = s.chatSvc.DeleteChannel(ctx, c.ID)
			return err
		}, map[string]string{uOwner: ok, uWriter: forbidden, uPlain: forbidden, uOutsider: notFound}},
		{"channels: the workspace's #general is never deleted", func(ctx context.Context) error {
			require.NoError(t, s.chatSvc.EnsureGeneralChannel(context.Background(), "workspace-default", uOwner))
			general, err := f.store.Chat.GetChannelByName(context.Background(), "workspace-default", chat.GeneralChannelName)
			require.NoError(t, err)
			_, err = s.chatSvc.DeleteChannel(ctx, general.ID)
			return err
		}, map[string]string{uOwner: invalid, uWriter: forbidden, uOutsider: notFound}},
		{"channels: a DM is not renamed", func(ctx context.Context) error {
			_, err := s.chatSvc.RenameChannel(ctx, f.dm.ID, "secret")
			return err
		}, map[string]string{uOwner: invalid, uWriter: invalid, uPlain: notFound, uOutsider: notFound}},
		{"chat: read a channel", func(ctx context.Context) error {
			_, err := s.chatSvc.GetConversation(ctx, f.channel.ID)
			return err
		}, map[string]string{uPlain: ok, uOutsider: notFound}},
		{"chat: read a DM", func(ctx context.Context) error {
			_, err := s.chatSvc.GetConversation(ctx, f.dm.ID)
			return err
		}, map[string]string{uOwner: ok, uWriter: ok, uPlain: notFound, uOutsider: notFound}},
		{"stacks: get", func(ctx context.Context) error {
			_, err := s.deploySvc.GetStack(ctx, f.stack)
			return err
		}, map[string]string{uOwner: ok, uReader: ok, uOverwrite: ok, uPlain: forbidden, uOutsider: notFound}},
		{"stacks: list", func(ctx context.Context) error {
			st, err := s.deploySvc.ListStacks(ctx, "")
			return contains(st, err, func(st *deploy.Stack) bool { return st.ID == f.stack })
		}, map[string]string{uReader: ok, uPlain: hidden, uOutsider: hidden}},
		{"stacks: get one outside every project", func(ctx context.Context) error {
			_, err := s.deploySvc.GetStack(ctx, f.infra)
			return err
		}, map[string]string{uOwner: ok, uReader: ok, uPlain: forbidden, uOutsider: forbidden}},
		{"deploys: get", func(ctx context.Context) error {
			_, err := s.deploySvc.Get(ctx, f.deploy)
			return err
		}, map[string]string{uReader: ok, uPlain: forbidden, uOutsider: notFound}},
		{"topology: get a workspace's canvas", func(ctx context.Context) error {
			_, err := s.topoSvc.Get(ctx, "workspace-default")
			return err
		}, map[string]string{uOwner: ok, uReader: ok, uOverwrite: ok, uPlain: forbidden, uOutsider: notFound}},
		{"notifications: list a workspace's inbox", func(ctx context.Context) error {
			actor, _ := identity.ActorFromCtx(ctx)
			_, err := s.notifSvc.List(ctx, actor.ID, "workspace-default", 10)
			return err
		}, map[string]string{uPlain: ok, uOutsider: notFound}},
		{"dns: list gateways", func(ctx context.Context) error {
			_, err := s.dnsSvc.ListGateways(ctx)
			return err
		}, map[string]string{uOwner: ok, uReader: ok, uOverwrite: ok, uPlain: forbidden, uOutsider: forbidden}},
		{"dns: delete an exposure (the owner passes the gate to a missing id)", func(ctx context.Context) error {
			return s.dnsSvc.DeleteExposure(ctx, "exposure-missing")
		}, map[string]string{uOwner: notFound, uReader: forbidden, uWriter: forbidden, uOutsider: forbidden}},
		{"dns: create a record", func(ctx context.Context) error {
			_, err := s.dnsSvc.CreateRecord(ctx, "zone-1", dns.RecordInput{Type: dns.RecordA, Name: "app", Content: "192.0.2.1"})
			return err
		}, map[string]string{uReader: forbidden, uPlain: forbidden, uOutsider: forbidden}},
		{"machines: list", func(ctx context.Context) error {
			_, err := machines.ListMachines(ctx)
			return err
		}, map[string]string{uOwner: ok, uReader: ok, uPlain: forbidden, uOutsider: forbidden}},
		{"automations hosts: list", func(ctx context.Context) error {
			_, err := s.automationHostsSvc.List(ctx)
			return err
		}, map[string]string{uOwner: ok, uReader: forbidden, uOutsider: forbidden}},
		{"connectors: list", func(ctx context.Context) error {
			_, err := s.connectorsSvc.List(ctx)
			return err
		}, map[string]string{uOwner: ok, uReader: ok, uPlain: forbidden, uOutsider: forbidden}},
		{"connectors: disconnect", func(ctx context.Context) error {
			return s.connectorsSvc.Disconnect(ctx, "cloudflare")
		}, map[string]string{uOwner: ok, uReader: forbidden, uWriter: forbidden, uOutsider: forbidden}},
		{"repositories: list the installation's", func(ctx context.Context) error {
			_, err := repository.ListRepos(ctx, s.accessSvc, noRepos{}, "", false)
			return err
		}, map[string]string{uOwner: ok, uWriter: ok, uReader: forbidden, uOutsider: forbidden}},
		{"pull requests: list a project repository's", func(ctx context.Context) error {
			_, err := gitprovider.ListPRs(ctx, entities, noPRs{}, "acme", "app", gitprovider.PROpts{})
			return err
		}, map[string]string{uOwner: ok, uReader: ok, uPlain: forbidden, uOutsider: notFound}},
		{"reviews: get", func(ctx context.Context) error {
			_, err := s.reviewSvc.Get(ctx, f.review)
			return err
		}, map[string]string{uOwner: ok, uReader: ok, uPlain: forbidden, uOutsider: notFound}},
		{"reviews: list a ticket's", func(ctx context.Context) error {
			rs, err := s.reviewSvc.ListByTicket(ctx, f.ticket.ID)
			return contains(rs, err, func(r *codereview.CodeReview) bool { return r.ID == f.review })
		}, map[string]string{uOwner: ok, uReader: ok, uWriter: hidden, uPlain: forbidden, uOutsider: notFound}},
		{"board: PR counts per card", func(ctx context.Context) error {
			counts, err := s.ticketsSvc.DevStatus(ctx, []string{f.ticket.ID})
			return present(counts, err, f.ticket.ID)
		}, map[string]string{uOwner: ok, uReader: ok, uPlain: hidden, uOutsider: hidden}},
		{"board: ticket-thread markers", func(ctx context.Context) error {
			marks, err := s.chatSvc.HasTicketThreads(ctx, []string{f.ticket.ID})
			return present(marks, err, f.ticket.ID)
		}, map[string]string{uOwner: ok, uReader: ok, uPlain: hidden, uOutsider: hidden}},
		{"chat: a DM with someone outside the workspace", func(ctx context.Context) error {
			actor, _ := identity.ActorFromCtx(ctx)
			_, err := s.chatSvc.CreateDM(ctx, "workspace-default", actor.ID, []string{uOutsider})
			return err
		}, map[string]string{uOwner: notFound, uWriter: notFound}},
		{"chat: a DM with nobody by that id reads the same", func(ctx context.Context) error {
			actor, _ := identity.ActorFromCtx(ctx)
			_, err := s.chatSvc.CreateDM(ctx, "workspace-default", actor.ID, []string{"u-nobody"})
			return err
		}, map[string]string{uOwner: notFound}},
		// Everything below took the instance administrator flag before ADR 0088; the gate runs before any lookup, so
		// a holder reaching a missing id or an unset instance URL proves the gate let them through.
		{"instance: change the instance URL", func(ctx context.Context) error {
			_, err := s.authSvc.UpdateInstanceURL(ctx, actorID(ctx), "https://nexul.example.com")
			return err
		}, instance},
		{"instance: turn a sign-in provider off", func(ctx context.Context) error {
			_, err := s.authSvc.SetProviderOAuth(ctx, actorID(ctx), auth.ProviderGoogle, "", "")
			return err
		}, instance},
		{"instance: mint a connection token", func(ctx context.Context) error {
			_, err := s.authSvc.GenerateConnectionToken(ctx, actorID(ctx))
			return err
		}, instance},
		{"instance: read the version and upgrade facts", func(ctx context.Context) error {
			_, err := upgrades.UpgradeStatus(ctx)
			return err
		}, instance},
		{"instance: start an upgrade (a dev build is refused past the gate)", func(ctx context.Context) error {
			_, err := upgrades.RequestUpgrade(ctx, actorID(ctx))
			return err
		}, map[string]string{uOwner: conflict, uSteward: conflict, uPlain: forbidden, uOutsider: forbidden}},
		{"accounts: list every account", func(ctx context.Context) error {
			_, err := s.authSvc.ListAccounts(ctx, actorID(ctx))
			return err
		}, map[string]string{uOwner: ok, uSteward: ok, uClerk: ok, uPlain: forbidden, uManager: forbidden, uOutsider: forbidden}},
		{"accounts: read someone else's", func(ctx context.Context) error {
			_, err := s.authSvc.GetAccount(ctx, actorID(ctx), uReader)
			return err
		}, map[string]string{uOwner: ok, uClerk: ok, uPlain: forbidden, uOutsider: forbidden}},
		{"accounts: the user directory", func(ctx context.Context) error {
			_, err := s.accessSvc.ListUsers(ctx, actorID(ctx))
			return err
		}, map[string]string{uOwner: ok, uClerk: ok, uPlain: forbidden, uOutsider: forbidden}},
		{"accounts: disable one", func(ctx context.Context) error {
			return s.authSvc.UpdateAccountStatus(ctx, actorID(ctx), "u-ghost", auth.AccountDisabled)
		}, map[string]string{uOwner: notFound, uSteward: notFound, uClerk: notFound, uPlain: forbidden, uOutsider: forbidden}},
		{"accounts: remove one", func(ctx context.Context) error {
			return s.authSvc.RemoveAccount(ctx, actorID(ctx), "u-ghost")
		}, map[string]string{uOwner: notFound, uSteward: notFound, uClerk: forbidden, uPlain: forbidden, uOutsider: forbidden}},
		{"accounts: the whole Team, or the workspaces one manages", func(ctx context.Context) error {
			_, err := s.tenancySvc.ListTeam(ctx, actorID(ctx))
			return err
		}, map[string]string{uOwner: ok, uClerk: ok, uManager: ok, uPlain: forbidden, uOutsider: forbidden}},
		{"workspaces: create one", func(ctx context.Context) error {
			_, err := s.tenancySvc.Create(ctx, actorID(ctx), "Made by "+actorID(ctx))
			return err
		}, instance},
		{"workspaces: rename one (where it lives)", func(ctx context.Context) error {
			name := "Default"
			_, err := s.tenancySvc.Rename(ctx, actorID(ctx), "workspace-default", &name, nil)
			return err
		}, map[string]string{uOwner: ok, uSteward: forbidden, uPlain: forbidden, uOutsider: forbidden}},
		{"runners: enroll one", func(ctx context.Context) error {
			_, err := machines.CreateEnrollment(ctx, "box-"+actorID(ctx), "")
			return err
		}, map[string]string{uOwner: conflict, uSteward: conflict, uPlain: forbidden, uManager: forbidden, uOutsider: forbidden}},
		{"runners: remove one", func(ctx context.Context) error {
			return machines.RemoveRunner(ctx, "runner-ghost")
		}, map[string]string{uOwner: notFound, uSteward: notFound, uPlain: forbidden, uOutsider: forbidden}},
		{"automations hosts: enroll one", func(ctx context.Context) error {
			_, err := s.automationHostsSvc.CreateEnrollment(ctx, "jobs-"+actorID(ctx), "")
			return err
		}, instance},
		{"automations hosts: remove one", func(ctx context.Context) error {
			return s.automationHostsSvc.Remove(ctx, "host-ghost")
		}, map[string]string{uOwner: notFound, uSteward: notFound, uPlain: forbidden, uOutsider: forbidden}},
		{"connectors: register an app", func(ctx context.Context) error {
			_, err := s.connectorsSvc.SetAppConfig(ctx, actorID(ctx), "connector-ghost", "id", "secret", "", "")
			return err
		}, map[string]string{uOwner: notFound, uSteward: notFound, uReader: forbidden, uPlain: forbidden, uOutsider: forbidden}},
		{"integrations: list installs", func(ctx context.Context) error {
			_, err := s.integrationsSvc.ListInstalls(ctx, actorID(ctx))
			return err
		}, instance},
		{"integrations: revoke one", func(ctx context.Context) error {
			return s.integrationsSvc.RevokeInstall(ctx, actorID(ctx), "install-ghost")
		}, map[string]string{uOwner: notFound, uSteward: notFound, uPlain: forbidden, uOutsider: forbidden}},
		{"audit: read the log", func(ctx context.Context) error {
			_, err := s.integrationsSvc.ListAudit(ctx, actorID(ctx), 10)
			return err
		}, instance},
	}
	for _, tc := range cases {
		for user, want := range tc.want {
			t.Run(tc.name+" as "+user, func(t *testing.T) {
				require.Equal(t, want, outcome(tc.call(as(user))))
			})
		}
	}
}

// TestIntegration_GrantsNeverExceedTheGiver is ADR 0088's escalation rule: someone who may edit roles, members, or
// invitations in a workspace hands out only what they hold there, and only the Owner holds everything.
func TestIntegration_GrantsNeverExceedTheGiver(t *testing.T) {
	f := newPermFixture(t)
	s := f.svc
	ctx := context.Background()
	_, err := f.store.Settings.Set(ctx, "https://nexul.example.com")
	require.NoError(t, err)
	instanceWrite := grant("instance:write")
	cases := []struct {
		name string
		call func(ctx context.Context) error
	}{
		{"a role carrying instance:write", func(ctx context.Context) error {
			_, err := s.rolesSvc.Create(ctx, "workspace-default", actorID(ctx), "Admins "+actorID(ctx), instanceWrite)
			return err
		}},
		{"instance:write added to an existing role", func(ctx context.Context) error {
			_, err := s.rolesSvc.Update(ctx, "workspace-default", "role-plain", actorID(ctx), "Member", instanceWrite)
			return err
		}},
		{"a member given a role carrying instance:write", func(ctx context.Context) error {
			return s.tenancySvc.ChangeMemberRole(ctx, actorID(ctx), "workspace-default", uPlain, "role-steward")
		}},
		{"an allow override of instance:write", func(ctx context.Context) error {
			return s.tenancySvc.SetMemberOverrides(ctx, actorID(ctx), "workspace-default", uReader, &instanceWrite, nil)
		}},
		{"an invitation whose override allows instance:write", func(ctx context.Context) error {
			_, err := s.invitationSvc.Create(ctx, actorID(ctx), tenancy.CreateInvitationInput{ExpiresInDays: 1,
				Grants: []*tenancy.InvitationGrant{{WorkspaceID: "workspace-default", RoleID: "role-plain", Allow: instanceWrite}}})
			return err
		}},
		{"an invitation into a role carrying instance:write", func(ctx context.Context) error {
			_, err := s.invitationSvc.Create(ctx, actorID(ctx), tenancy.CreateInvitationInput{ExpiresInDays: 1,
				Grants: []*tenancy.InvitationGrant{{WorkspaceID: "workspace-default", RoleID: "role-steward"}}})
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name+" is refused to a roles and members manager without it", func(t *testing.T) {
			err := tc.call(as(uManager))
			require.ErrorIs(t, err, apperrs.ErrForbidden)
			require.ErrorContains(t, err, "you can only grant permissions you hold", "refused for the grant, not for managing roles or members")
		})
	}
	for _, tc := range cases {
		t.Run(tc.name+" is the Owner's to give", func(t *testing.T) {
			require.Equal(t, ok, outcome(tc.call(as(uOwner))))
		})
	}
}

// TestIntegration_FirstUserReachesEveryFormerAdminCapability: the owner wizard makes the first user the default
// workspace's Owner, and nothing else is needed for every instance-level permission.
func TestIntegration_FirstUserReachesEveryFormerAdminCapability(t *testing.T) {
	svc, store := newWired(t)
	ctx := context.Background()
	first, err := store.Users.CreateFirstUser(ctx, &auth.Identity{UserID: "u-first", Provider: auth.ProviderGitHub, ProviderUserID: "1", Login: "first"})
	require.NoError(t, err)

	before, err := svc.authSvc.Me(ctx, first.ID)
	require.NoError(t, err)
	require.True(t, before.NeedsOwnerWizard)
	require.NoError(t, svc.authSvc.CompleteOwnerWizard(as(first.ID), first.ID, "https://nexul.example.com"))

	after, err := svc.authSvc.Me(ctx, first.ID)
	require.NoError(t, err)
	assert.False(t, after.NeedsOwnerWizard)
	assert.Subset(t, after.InstancePermissions, instanceBits)
	for _, action := range instanceBits {
		held, err := svc.accessSvc.HoldsAnywhere(ctx, first.ID, permissions.Action(action))
		require.NoError(t, err)
		assert.True(t, held, action)
	}
	_, err = svc.tenancySvc.Create(as(first.ID), first.ID, "Second")
	require.NoError(t, err)
}

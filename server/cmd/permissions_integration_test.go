package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/platform/config"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/roles"
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
)

var people = []string{uOwner, uReader, uWriter, uPlain, uOverwrite, uOutsider}

// outcomes a case can have: allowed, refused as forbidden, refused as not found, or left out of a list.
const (
	ok        = "ok"
	forbidden = "403"
	notFound  = "404"
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
	}
	return err.Error()
}

func as(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
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
}

// newPermFixture wires the real composition root over a fresh database and seeds one of everything in the default
// workspace, plus the people above: the Owner, a reader and a writer role, a member with no grants, a member whose
// workspace-wide overwrite grants the reads, and someone outside the workspace.
func newPermFixture(t *testing.T) permFixture {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	db := mentionsTestDB(t)
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: store.ProcessedEvents, DeadLetterStore: store.DeadLetters})
	t.Cleanup(func() { _ = bus.Close() })
	cfg := &config.Config{DBPath: filepath.Join(dir, "nexul.db"), HTTPAddr: "127.0.0.1:0", AuthSecret: "0123456789abcdef0123456789abcdef"}
	svc := wireCoreServices(cfg, store, []byte("0123456789abcdef0123456789abcdef"), bus, logger)

	for _, id := range people {
		_, _, err := store.Users.UpsertUser(ctx, &auth.Identity{UserID: id, Provider: auth.ProviderGitHub, ProviderUserID: id, Login: id})
		require.NoError(t, err)
	}
	reads := []string{"tickets:read", "stacks:read", "deploys:read", "topology:read", "docs:read", "memories:read"}
	now := time.Now()
	for _, r := range []*roles.Role{
		{ID: "role-owner", WorkspaceID: "workspace-default", Name: "Owner", IsOwnerRole: true},
		{ID: "role-reader", WorkspaceID: "workspace-default", Name: "Reader", Permissions: grant(reads...)},
		{ID: "role-writer", WorkspaceID: "workspace-default", Name: "Writer", Permissions: grant("tickets:read", "tickets:write", "projects:write", "docs:write", "chat:write")},
		{ID: "role-plain", WorkspaceID: "workspace-default", Name: "Member"},
	} {
		r.CreatedAt, r.UpdatedAt = now, now
		require.NoError(t, store.Roles.Create(ctx, r))
	}
	for user, role := range map[string]string{uOwner: "role-owner", uReader: "role-reader", uWriter: "role-writer", uPlain: "role-plain", uOverwrite: "role-plain"} {
		require.NoError(t, store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: "workspace-default", RoleID: role, CreatedAt: now}))
	}
	require.NoError(t, store.Access.Set(ctx, "workspace", "workspace-default", uOverwrite, grant(reads...), nil))

	f := permFixture{svc: svc, store: store, doc: "doc-1", stack: "stack-1", infra: "stack-gateway", deploy: "deploy-1"}
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
	return f
}

// TestIntegration_PermissionTable walks each domain's reads, lists, and creates as every kind of caller, through the
// wired use-cases the HTTP gateway and MCP tools both call.
func TestIntegration_PermissionTable(t *testing.T) {
	f := newPermFixture(t)
	s := f.svc
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
			_, err := s.workspaceSvc.Create(ctx, actor.ID, "workspace-default", "Mobile", "MOB"+actor.ID[2:4], workspace.ProjectIcon(""))
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
		{"chat: create channel", func(ctx context.Context) error {
			actor, _ := identity.ActorFromCtx(ctx)
			_, err := s.chatSvc.CreateChannel(ctx, "workspace-default", actor.ID, "room-"+actor.ID)
			return err
		}, map[string]string{uOwner: ok, uWriter: ok, uPlain: forbidden, uOutsider: notFound}},
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
		{"topology: get", func(ctx context.Context) error {
			_, err := s.topoSvc.Get(ctx, "production")
			return err
		}, map[string]string{uOwner: ok, uReader: ok, uOverwrite: ok, uPlain: forbidden, uOutsider: forbidden}},
		{"notifications: list a workspace's inbox", func(ctx context.Context) error {
			actor, _ := identity.ActorFromCtx(ctx)
			_, err := s.notifSvc.List(ctx, actor.ID, "workspace-default", 10)
			return err
		}, map[string]string{uPlain: ok, uOutsider: notFound}},
	}
	for _, tc := range cases {
		for user, want := range tc.want {
			t.Run(tc.name+" as "+user, func(t *testing.T) {
				require.Equal(t, want, outcome(tc.call(as(user))))
			})
		}
	}
}

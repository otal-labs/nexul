package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus/deadletter"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/repository"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/topology"
	"github.com/otal-labs/nexul/internal/workspace"
)

// fakeGitProvider stubs the gitprovider seam for registry tests.
type fakeGitProvider struct{}

func (fakeGitProvider) GetRepo(context.Context, string, string) (*gitprovider.Repo, error) {
	return &gitprovider.Repo{FullName: "a/b"}, nil
}
func (fakeGitProvider) ListPRs(context.Context, string, string, gitprovider.PROpts) ([]*gitprovider.PR, error) {
	return []*gitprovider.PR{{Number: 1}}, nil
}
func (fakeGitProvider) GetPR(context.Context, string, string, int) (*gitprovider.PR, error) {
	return &gitprovider.PR{Number: 1}, nil
}
func (fakeGitProvider) CreateWebhook(context.Context, string, string, gitprovider.WebhookConfig) (string, error) {
	return "h1", nil
}
func (fakeGitProvider) DeleteWebhook(context.Context, string, string, string) error {
	return nil
}
func (fakeGitProvider) ListInstallationRepos(context.Context) ([]*gitprovider.Repo, error) {
	return nil, nil
}
func (fakeGitProvider) GetTree(context.Context, string, string, string) ([]gitprovider.TreeEntry, error) {
	return nil, nil
}
func (fakeGitProvider) GetFile(context.Context, string, string, string, string) ([]byte, error) {
	return nil, nil
}

// fakeScanner stubs the repository seam for registry tests.
type fakeScanner struct{}

func (fakeScanner) ListInstallationRepos(context.Context) ([]repository.Repo, error) { return nil, nil }
func (fakeScanner) GetTree(_ context.Context, _, _, ref string) (string, []repository.TreeEntry, error) {
	return ref, nil, nil
}
func (fakeScanner) GetFile(context.Context, string, string, string, string) ([]byte, error) {
	return nil, nil
}

// newAccess wires a real access.Service over the given store's repos with a
// users adapter, so docs enforcement tests exercise the real access path.
func newAccess(t *testing.T, store *storage.Store) *access.Service {
	t.Helper()
	return access.NewService(store.Access, testUsers{store.Users})
}

// seedOwner creates an owner user and returns its ID, so the real access
// service resolves owner status through the store.
func seedOwner(t *testing.T, store *storage.Store) string {
	t.Helper()
	owner, _, err := store.Users.UpsertUser(context.Background(), &auth.User{
		ID:             "owner-user",
		Provider:       auth.ProviderGitHub,
		ProviderUserID: "1",
		Login:          "owner",
		Name:           "Owner",
	})
	require.NoError(t, err)
	require.NoError(t, store.Users.SetCanCreateWorkspace(context.Background(), owner.ID, true))
	return owner.ID
}

// testUsers adapts the storage UsersRepo to access.Users (mirror of the
// server/cmd adapter).
type testUsers struct {
	repo *storage.UsersRepo
}

func (u testUsers) GetUserByID(ctx context.Context, id string) (*access.User, error) {
	user, err := u.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &access.User{ID: user.ID, Login: user.Login, Name: user.Name, CanCreateWorkspace: user.CanCreateWorkspace}, nil
}

func (u testUsers) ListUsers(ctx context.Context) ([]*access.User, error) {
	users, err := u.repo.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*access.User, 0, len(users))
	for _, user := range users {
		out = append(out, &access.User{ID: user.ID, Login: user.Login, Name: user.Name, CanCreateWorkspace: user.CanCreateWorkspace})
	}
	return out, nil
}

// testDeployProjects adapts the storage ProjectsRepo to deploy.ProjectStore
// for the mcp tests (mirror of the server/cmd adapter).
type testDeployProjects struct {
	repo *storage.ProjectsRepo
}

func (p testDeployProjects) ProjectExists(ctx context.Context, projectID string) (bool, error) {
	_, err := p.repo.Get(ctx, projectID)
	return err == nil, nil
}

func (p testDeployProjects) LinkRepo(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (p testDeployProjects) RepoInProject(_ context.Context, _ string, _ string, _ string) (bool, error) {
	return true, nil
}

// testMemoriesPermission adapts access's HasPermission to memories' PermissionGate seam (mirror of the server/cmd adapter).
type testMemoriesPermission struct {
	svc *access.Service
}

func (g testMemoriesPermission) HasPermission(ctx context.Context, userID, workspaceID string, action permissions.Action) bool {
	return g.svc.HasPermission(ctx, userID, workspaceID, action, "", "")
}

// testProjectLookup adapts the storage ProjectsRepo to memories' ProjectLookup seam (mirror of the server/cmd adapter).
type testProjectLookup struct {
	repo *storage.ProjectsRepo
}

func (p testProjectLookup) WorkspaceForProject(ctx context.Context, projectID string) (string, error) {
	proj, err := p.repo.Get(ctx, projectID)
	if err != nil {
		return "", err
	}
	return proj.WorkspaceID, nil
}

// fakePublisher records publishes for topology/deploy/replay wiring.
type fakePublisher struct {
	published []string
}

func (f *fakePublisher) Publish(_ context.Context, topic string, _ any) error {
	f.published = append(f.published, topic)
	return nil
}

func (f *fakePublisher) PublishWithID(_ context.Context, _ string, topic string, _ any) error {
	return f.Publish(context.TODO(), topic, nil)
}

// fakeDispatch stubs the runner use-case seam for registry tests.
type fakeDispatch struct{}

func (fakeDispatch) Runners() []runner.RunnerStatus       { return nil }
func (fakeDispatch) Queue() []runner.QueuedJob            { return nil }
func (fakeDispatch) Cancel(context.Context, string) error { return nil }
func (fakeDispatch) Discover(context.Context, string, time.Duration) (runner.DiscoverReport, error) {
	return runner.DiscoverReport{}, nil
}

// workspaceUserStoreFake is a no-op UserStore for registry wiring tests.
type workspaceUserStoreFake struct{}

func (workspaceUserStoreFake) GetUserByLogin(context.Context, string) (*workspace.User, error) {
	return nil, apperrs.ErrNotFound
}

func (workspaceUserStoreFake) ListUsers(context.Context) ([]*workspace.User, error) {
	return nil, nil
}

func (workspaceUserStoreFake) LoginForUserID(context.Context, string) (string, error) {
	return "", apperrs.ErrNotFound
}

// workspaceMembersStoreFake is a no-op WorkspaceMemberStore for registry wiring tests.
type workspaceMembersStoreFake struct{}

func (workspaceMembersStoreFake) ListMemberUserIDs(context.Context, string) ([]string, error) {
	return nil, nil
}

type registryInstanceURL struct{}

func (registryInstanceURL) InstanceURL(context.Context) string { return "https://nexul.example" }

// registryNotificationPermGate mirrors server/cmd/wire_gates.go's notificationPermissionGate adapter.
type registryNotificationPermGate struct {
	svc *access.Service
}

func (g registryNotificationPermGate) HasPermission(ctx context.Context, userID, workspaceID string, action permissions.Action) bool {
	return g.svc.HasPermission(ctx, userID, workspaceID, action, "", "")
}

// registryPlaysPermGate mirrors server/cmd/wire_gates.go's playsPermissionGate adapter, workspace-scoped
// unlike automations' instance-wide one.
type registryPlaysPermGate struct {
	svc *access.Service
}

func (g registryPlaysPermGate) HasPermission(ctx context.Context, userID, workspaceID string, action permissions.Action, resourceType, resourceID string) bool {
	return g.svc.HasPermission(ctx, userID, workspaceID, action, resourceType, resourceID)
}

func newRegistryServer(t *testing.T) (*Server, *storage.Store, *fakePublisher) {
	t.Helper()
	store := testutil.NewStore(t)
	pub := &fakePublisher{}
	accessSvc := newAccess(t, store)
	ownerID := seedOwner(t, store)
	authSvc := auth.NewService(auth.Config{Secret: []byte("registry-test"), Users: store.Users})
	invitationSvc := tenancy.NewInvitationService(store.Invitations, registryInstanceURL{})
	return New(RegistryOptions{
		Docs:          docs.NewService(store.Docs, accessSvc),
		Memories:      memories.NewService(store.Memories, testMemoriesPermission{accessSvc}, testProjectLookup{store.Projects}, nil, nil),
		Tickets:       tickets.NewService(store.Tickets, store.Statuses, nil),
		Topology:      topology.NewService(store.Topology),
		Deploy:        deploy.NewService(store.Deploys, store.Stacks, store.Services, testDeployProjects{store.Projects}),
		Reviews:       codereview.NewService(store.CodeReviews),
		Workspace:     workspace.NewService(store.Projects, store.Categories, store.TicketTypes, store.Statuses, nil, nil),
		Notifications: workspace.NewNotificationService(store.Notifications, workspaceUserStoreFake{}, workspaceMembersStoreFake{}, registryNotificationPermGate{svc: accessSvc}),
		Git:           fakeGitProvider{},
		Repository:    fakeScanner{},
		Runner:        runner.NewService(store.Runners, fakeDispatch{}),
		Automations:   automations.NewService(store.Automations, nil),
		Access:        accessSvc,
		Auth:          authSvc,
		Invitations:   invitationSvc,
		Plays:         plays.NewService(store.Plays, registryPlaysPermGate{svc: accessSvc}),
		PlayRuns:      plays.NewRunner(plays.RunnerConfig{Plays: store.Plays, Trails: store.PlayTrails, Perm: registryPlaysPermGate{svc: accessSvc}}),
		Pairing:       pairing.NewService(pairing.Config{Repo: store.Pairing}),
		DeadLetter:    store.DeadLetters,
		Publisher:     pub,
		Actor: func(context.Context) identity.Actor {
			return identity.Actor{ID: ownerID, CanCreateWorkspace: true}
		},
	}), store, pub
}

func TestRegistry_ToolsComplete(t *testing.T) {
	srv, _, _ := newRegistryServer(t)
	require.Len(t, srv.tools, 131)
	names := make(map[string]bool)
	for _, tool := range srv.tools {
		require.NotEmpty(t, tool.Name, "every tool must be named")
		require.False(t, names[tool.Name], "duplicate tool %s", tool.Name)
		names[tool.Name] = true
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
	expected := []string{
		"search_docs", "search_tickets", "list_dead_letters", "replay_dead_letter",
		"doc_create", "doc_get", "doc_search", "doc_update", "doc_archive", "doc_restore",
		"ticket_create", "ticket_get", "ticket_update", "ticket_update_status", "ticket_set_type", "ticket_set_developer", "ticket_set_tester",
		"ticket_add_label", "ticket_remove_label", "ticket_list_labels", "ticket_list_all_labels",
		"ticket_search",
		"topology_get", "topology_add_node", "topology_remove_node", "topology_add_edge", "topology_remove_edge",
		"deploy_get", "deploy_log", "deploy_list", "deploy_list_by_service", "deploy_list_by_status", "deploy_cancel",
		"service_list", "stack_create", "stack_deploy", "stack_get", "stack_list", "stack_update",
		"stack_delete", "stack_rollback", "machine_import",
		"runner_list", "runner_queue", "machine_list", "machine_discover",
		"instance_upgrade_status", "instance_upgrade",
		"repository_list", "repository_scan",
		"review_list_by_ticket", "review_get",
		"git_list_prs", "git_get_pr",
		"project_create", "project_get", "project_list", "project_rename", "project_delete",
		"project_reorder", "project_delete_impact", "project_add_repo", "project_remove_repo",
		"project_list_repos", "project_move_ticket",
		"category_create", "category_get", "category_list", "category_rename", "category_delete",
		"category_reorder", "category_move_ticket", "category_clear_ticket",
		"ticket_type_create", "ticket_type_list", "ticket_type_rename", "ticket_type_delete",
		"status_create", "status_list", "status_rename", "status_reorder", "status_delete",
		"notification_list", "notification_mark_read", "notification_mark_all_read",
		"access_list_grants", "access_set_grants",
		"play_list", "play_create", "play_update", "play_delete",
		"play_run", "play_run_get", "play_run_stop", "play_run_answer", "play_list_runs",
		"memory_list", "memory_get", "memory_create", "memory_update", "memory_delete",
		"create_invitation", "list_invitations", "revoke_invitation",
		"account_whoami", "list_accounts", "disable_account", "reactivate_account", "remove_account", "restore_account",
		"computer_setup_get", "computer_setup_confirm_provider", "computer_setup_unconfirm_provider",
		"computer_setup_confirm", "computer_setup_unconfirm",
	}
	for _, name := range expected {
		assert.True(t, names[name], "missing tool %s", name)
	}
}

func TestRegistry_ResourcesAndPrompts(t *testing.T) {
	srv, _, _ := newRegistryServer(t)
	require.Len(t, srv.resources, 1)
	assert.Equal(t, "topology://current", srv.resources[0].URI)
	require.Len(t, srv.templates, 2)
	var templates []string
	for _, tmpl := range srv.templates {
		templates = append(templates, tmpl.URITemplate)
	}
	assert.ElementsMatch(t, []string{"docs://{id}", "tickets://{id}"}, templates)
	require.Len(t, srv.prompts, 4)
	var prompts []string
	for _, p := range srv.prompts {
		prompts = append(prompts, p.Name)
	}
	assert.ElementsMatch(t, []string{"create_ticket_from_doc", "deploy_stack", "investigate_failure", "ship_repository"}, prompts)
}

func TestRegistry_SearchDocsTool(t *testing.T) {
	srv, store, _ := newRegistryServer(t)
	ownerID := seedOwner(t, store)
	ds := docs.NewService(store.Docs, newAccess(t, store))
	ctx := identity.WithActor(context.Background(), identity.Actor{ID: ownerID, CanCreateWorkspace: true})
	_, err := ds.Create(ctx, "project-general", "Storage Spine", "SQLite migrations and FTS5")
	require.NoError(t, err)

	resp := dispatch(t, srv, 1, MethodToolsCall, map[string]any{"name": "search_docs", "arguments": map[string]any{"query": "sqlite"}})
	require.Nil(t, resp.Error)
	result := decodeResult[toolCallResult](t, resp)
	require.Len(t, result.Content, 1)
	assert.Contains(t, result.Content[0].Text, "Storage Spine")
}

func TestRegistry_DocResource(t *testing.T) {
	srv, store, _ := newRegistryServer(t)
	ownerID := seedOwner(t, store)
	ds := docs.NewService(store.Docs, newAccess(t, store))
	ctx := identity.WithActor(context.Background(), identity.Actor{ID: ownerID, CanCreateWorkspace: true})
	doc, err := ds.Create(ctx, "project-general", "Spec", "body text")
	require.NoError(t, err)

	resp := dispatch(t, srv, 1, MethodResourcesRead, map[string]any{"uri": "docs://" + doc.ID})
	require.Nil(t, resp.Error)
	result := decodeResult[resourceReadResult](t, resp)
	assert.Equal(t, "# Spec\n\nbody text", result.Contents[0].Text)
}

func TestRegistry_TopologyResource(t *testing.T) {
	srv, store, _ := newRegistryServer(t)
	ts := topology.NewService(store.Topology)
	_, err := ts.Update(context.Background(), topology.DefaultEnvironment, &topology.Canvas{
		SchemaVersion: topology.CurrentSchemaVersion,
		Nodes: []topology.Node{{
			ID: "net-main", Type: topology.NodeNetwork, Position: topology.Position{X: 1, Y: 1}, Data: topology.NodeData{Name: "main"},
		}},
	})
	require.NoError(t, err)

	resp := dispatch(t, srv, 1, MethodResourcesRead, map[string]any{"uri": "topology://current"})
	require.Nil(t, resp.Error)
	result := decodeResult[resourceReadResult](t, resp)
	assert.Contains(t, result.Contents[0].Text, `"name":"main"`)
	assert.Equal(t, "application/json", result.Contents[0].MIMEType)
}

func TestRegistry_ReplayDeadLetter(t *testing.T) {
	srv, store, pub := newRegistryServer(t)
	adapter := store.DeadLetters
	require.NoError(t, adapter.Put(context.Background(), deadletter.DeadLetter{
		ID: "dl-1", Topic: "doc.created", Payload: []byte(`{"doc":{"id":"x"}}`), Error: "boom", Attempts: 3,
	}))

	listResp := dispatch(t, srv, 1, MethodToolsCall, map[string]any{"name": "list_dead_letters"})
	require.Nil(t, listResp.Error)
	listResult := decodeResult[toolCallResult](t, listResp)
	assert.Contains(t, listResult.Content[0].Text, "dl-1")

	replayResp := dispatch(t, srv, 2, MethodToolsCall, map[string]any{"name": "replay_dead_letter", "arguments": map[string]any{"id": "dl-1"}})
	require.Nil(t, replayResp.Error)
	replayResult := decodeResult[toolCallResult](t, replayResp)
	assert.Contains(t, replayResult.Content[0].Text, "replayed")
	assert.Equal(t, []string{"doc.created"}, pub.published)

	_, err := adapter.Get(context.Background(), "dl-1")
	require.Error(t, err)

	replayMissing := dispatch(t, srv, 3, MethodToolsCall, map[string]any{"name": "replay_dead_letter", "arguments": map[string]any{"id": "dl-1"}})
	require.NotNil(t, replayMissing.Error)
	assert.Equal(t, CodeNotFound, replayMissing.Error.Code)
}

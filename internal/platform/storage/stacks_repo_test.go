package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func newTestStack(id string) *deploy.Stack {
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	return &deploy.Stack{
		ID:          id,
		ProjectID:   "project-general",
		Name:        "api",
		Slug:        "api",
		Machine:     "10.0.0.1:22",
		Strategy:    deploy.StrategyCompose,
		ComposePath: "/srv/api/docker-compose.yml",
		Env:         map[string]string{"PORT": "8080"},
		Managed:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestStacksRepo_Create_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestStack("svc-1")
	require.NoError(t, s.Stacks.Create(context.Background(), want))

	got, err := s.Stacks.GetByID(context.Background(), "svc-1")
	require.NoError(t, err)
	assert.Equal(t, "api", got.Name)
	assert.Equal(t, "api", got.Slug)
	assert.Equal(t, "10.0.0.1:22", got.Machine)
	assert.Equal(t, deploy.StrategyCompose, got.Strategy)
	assert.Equal(t, map[string]string{"PORT": "8080"}, got.Env)
	assert.Equal(t, "/srv/api/docker-compose.yml", got.ComposePath)
	assert.True(t, got.Managed)
}

func TestStacksRepo_Create_RunStrategyRoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	stack := newTestStack("svc-1")
	stack.Strategy = deploy.StrategyRun
	stack.ComposePath = ""
	stack.DockerNetwork = "net1"
	stack.Ports = []string{"80:80", "443:443"}
	stack.Command = []string{"tunnel", "--no-autoupdate", "run"}
	stack.BuildSource = &deploy.BuildSource{RepoOwner: "acme", RepoName: "api", Branch: "main", Dockerfile: "Dockerfile"}
	require.NoError(t, s.Stacks.Create(context.Background(), stack))

	got, err := s.Stacks.GetByID(context.Background(), "svc-1")
	require.NoError(t, err)
	assert.Equal(t, "net1", got.DockerNetwork)
	assert.Equal(t, []string{"80:80", "443:443"}, got.Ports)
	assert.Equal(t, []string{"tunnel", "--no-autoupdate", "run"}, got.Command)
	require.NotNil(t, got.BuildSource)
	assert.Equal(t, "acme", got.BuildSource.RepoOwner)
	assert.Equal(t, "Dockerfile", got.BuildSource.Dockerfile)
}

func TestStacksRepo_Create_WritesOutboxInSameTx(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	stack := newTestStack("svc-1")
	evt := eventbus.OutboxEvent{ID: "evt-1", Topic: deploy.TopicStackCreated, Payload: deploy.StackEvent{Stack: *stack}}
	require.NoError(t, s.Stacks.Create(context.Background(), stack, evt))

	var raw []byte
	require.NoError(t, s.db.QueryRowContext(context.Background(), `SELECT payload FROM outbox WHERE id = 'evt-1'`).Scan(&raw))
	var e deploy.StackEvent
	require.NoError(t, json.Unmarshal(raw, &e))
	assert.Equal(t, "api", e.Stack.Name)
}

func TestStacksRepo_DuplicateSlugPerMachine_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Stacks.Create(context.Background(), newTestStack("svc-1")))
	dup := newTestStack("svc-2")
	dup.Name, dup.Slug = "web", "web"
	require.NoError(t, s.Stacks.Create(context.Background(), dup))
	third := newTestStack("svc-3") // same slug+machine as svc-1
	err := s.Stacks.Create(context.Background(), third)
	require.ErrorIs(t, err, apperrs.ErrConflict, "the (machine, slug) unique index rejects duplicates")
}

func TestStacksRepo_UnknownProject_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	stack := newTestStack("svc-1")
	stack.ProjectID = "no-such-project"
	err := s.Stacks.Create(context.Background(), stack)
	require.ErrorIs(t, err, apperrs.ErrConflict, "the project FK rejects unknown project ids")
}

func TestStacksRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Stacks.GetByID(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestStacksRepo_GetBySlugAndMachine(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Stacks.Create(context.Background(), newTestStack("svc-1")))

	got, err := s.Stacks.GetBySlugAndMachine(context.Background(), "api", "10.0.0.1:22")
	require.NoError(t, err)
	assert.Equal(t, "svc-1", got.ID)

	_, err = s.Stacks.GetBySlugAndMachine(context.Background(), "api", "other:22")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestStacksRepo_ListByProject(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Stacks.Create(context.Background(), newTestStack("svc-1")))
	other := newTestStack("svc-2")
	other.Name, other.Slug = "web", "web"
	require.NoError(t, s.Stacks.Create(context.Background(), other))

	got, err := s.Stacks.ListByProject(context.Background(), "project-general")
	require.NoError(t, err)
	require.Len(t, got, 2)

	all, err := s.Stacks.ListByProject(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, all, 2)
	none, err := s.Stacks.ListByProject(context.Background(), "project-other")
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestStacksRepo_Update_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	created := newTestStack("svc-1")
	require.NoError(t, s.Stacks.Create(context.Background(), created))

	created.Name = "api-v2"
	created.Managed = false
	created.UpdatedAt = time.Now().UTC()
	require.NoError(t, s.Stacks.Update(context.Background(), created))

	got, err := s.Stacks.GetByID(context.Background(), "svc-1")
	require.NoError(t, err)
	assert.Equal(t, "api-v2", got.Name)
	assert.False(t, got.Managed)
}

func TestStacksRepo_Update_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Stacks.Update(context.Background(), newTestStack("svc-1"))
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestStacksRepo_Delete_WritesOutboxAndRemoves(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Stacks.Create(context.Background(), newTestStack("svc-1")))
	evt := eventbus.OutboxEvent{ID: "evt-2", Topic: deploy.TopicStackDeleted, Payload: deploy.StackDeletedEvent{ID: "svc-1", Name: "api"}}
	require.NoError(t, s.Stacks.Delete(context.Background(), "svc-1", evt))

	_, err := s.Stacks.GetByID(context.Background(), "svc-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	var raw []byte
	require.NoError(t, s.db.QueryRowContext(context.Background(), `SELECT payload FROM outbox WHERE id = 'evt-2'`).Scan(&raw))
	var e deploy.StackDeletedEvent
	require.NoError(t, json.Unmarshal(raw, &e))
	assert.Equal(t, "svc-1", e.ID)
}

func TestStacksRepo_BranchDeployRules_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	stack := newTestStack("svc-1")
	stack.Strategy = deploy.StrategyRun
	stack.ComposePath = ""
	stack.DockerNetwork = "app-net"
	stack.BuildSource = &deploy.BuildSource{RepoOwner: "acme", RepoName: "api", Dockerfile: "Dockerfile"}
	stack.BranchDeployRules = []deploy.BranchDeployRule{
		{Pattern: "main", DockerNetwork: "app-net"},
		{Pattern: "feature/*", DockerNetwork: "app-net", HostnameTemplate: "{branch}.example.com", Port: 8080, Overrides: map[string]string{"DATABASE_URL": "postgres://qa"}},
	}
	require.NoError(t, s.Stacks.Create(context.Background(), stack))

	got, err := s.Stacks.GetByID(context.Background(), "svc-1")
	require.NoError(t, err)
	require.Len(t, got.BranchDeployRules, 2)
	assert.Equal(t, "main", got.BranchDeployRules[0].Pattern)
	assert.Equal(t, "feature/*", got.BranchDeployRules[1].Pattern)
	assert.Equal(t, "{branch}.example.com", got.BranchDeployRules[1].HostnameTemplate)
	assert.Equal(t, 8080, got.BranchDeployRules[1].Port)
	assert.Equal(t, map[string]string{"DATABASE_URL": "postgres://qa"}, got.BranchDeployRules[1].Overrides)
}

func TestStacksRepo_DerivedFromAndBranch_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	base := newTestStack("svc-1")
	require.NoError(t, s.Stacks.Create(context.Background(), base))

	derived := newTestStack("svc-2")
	derived.Name, derived.Slug = "api-feature-x", "api-feature-x"
	derived.DerivedFrom = "svc-1"
	derived.Branch = "feature/x"
	require.NoError(t, s.Stacks.Create(context.Background(), derived))

	got, err := s.Stacks.GetByID(context.Background(), "svc-2")
	require.NoError(t, err)
	assert.Equal(t, "svc-1", got.DerivedFrom)
	assert.Equal(t, "feature/x", got.Branch)
}

func TestStacksRepo_ListByBuildRepo(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	base := newTestStack("svc-1")
	base.BuildSource = &deploy.BuildSource{RepoOwner: "acme", RepoName: "api", Dockerfile: "Dockerfile"}
	require.NoError(t, s.Stacks.Create(context.Background(), base))

	derived := newTestStack("svc-2")
	derived.Name, derived.Slug = "api-feature-x", "api-feature-x"
	derived.BuildSource = &deploy.BuildSource{RepoOwner: "acme", RepoName: "api", Dockerfile: "Dockerfile"}
	derived.DerivedFrom = "svc-1"
	require.NoError(t, s.Stacks.Create(context.Background(), derived))

	other := newTestStack("svc-3")
	other.Name, other.Slug = "web", "web"
	other.BuildSource = &deploy.BuildSource{RepoOwner: "acme", RepoName: "web", Dockerfile: "Dockerfile"}
	require.NoError(t, s.Stacks.Create(context.Background(), other))

	got, err := s.Stacks.ListByBuildRepo(context.Background(), "acme", "api")
	require.NoError(t, err)
	require.Len(t, got, 1, "excludes the derived clone even though it shares the repo")
	assert.Equal(t, "svc-1", got[0].ID)
}

func TestStacksRepo_ListByDerivedFrom(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	base := newTestStack("svc-1")
	require.NoError(t, s.Stacks.Create(context.Background(), base))

	derived := newTestStack("svc-2")
	derived.Name, derived.Slug = "api-feature-x", "api-feature-x"
	derived.DerivedFrom = "svc-1"
	derived.Branch = "feature/x"
	require.NoError(t, s.Stacks.Create(context.Background(), derived))

	other := newTestStack("svc-3")
	other.Name, other.Slug = "web", "web"
	require.NoError(t, s.Stacks.Create(context.Background(), other))

	got, err := s.Stacks.ListByDerivedFrom(context.Background(), "svc-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "svc-2", got[0].ID)
}

func TestProjectsRepo_CountServices(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Stacks.Create(context.Background(), newTestStack("svc-1")))
	other := newTestStack("svc-2")
	other.Name, other.Slug = "web", "web"
	require.NoError(t, s.Stacks.Create(context.Background(), other))

	n, err := s.Projects.CountServices(context.Background(), "project-general")
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

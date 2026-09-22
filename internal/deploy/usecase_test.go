package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func validRequest() DeployRequest {
	return DeployRequest{
		StackID: "svc-1",
		Image:   "ghcr.io/onik/api:v1",
	}
}

func validStack() Stack {
	return Stack{
		ID:          "svc-1",
		ProjectID:   "proj-1",
		Name:        "api",
		Slug:        "api",
		Machine:     "10.0.0.1:22",
		Strategy:    StrategyCompose,
		ComposePath: "docker-compose.yml",
		Env:         map[string]string{"PORT": "8080"},
		Managed:     true,
	}
}

func newTestService(repo *fakeRepo, bus *fakeBus) *Service {
	stacks := newFakeStackRepo()
	stack := validStack()
	if err := stacks.Create(context.Background(), &stack); err != nil {
		panic(err)
	}
	return newTestServiceWith(repo, stacks, newFakeContainerRepo(), newFakeProjects(), bus)
}

// bus is accepted for call-site compatibility across the package's tests but
// is no longer wired into the service: deploy.requested / cancel_requested
// reach the bus via the transactional outbox on repo, not a direct Publish.
// Tests assert on repo.of(...)/repo.topics() instead.
func newTestServiceWith(repo *fakeRepo, stacks *fakeStackRepo, containers *fakeContainerRepo, projects *fakeProjects, bus *fakeBus) *Service {
	s := NewService(repo, stacks, containers, projects)
	s.now = func() time.Time { return time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC) }
	return s
}

func requireStack(t *testing.T, stacks *fakeStackRepo, stack Stack) {
	t.Helper()
	if stack.ID == "" {
		stack.ID = "svc-1"
	}
	if stack.Slug == "" {
		stack.Slug = Slug(stack.Name, dnsLabelMaxLen)
	}
	if err := stacks.Create(context.Background(), &stack); err != nil {
		t.Fatal(err)
	}
}

func TestDeploy_ValidationError(t *testing.T) {
	t.Run("missing stack is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		req := validRequest()
		req.StackID = ""
		_, err := s.Deploy(context.Background(), req)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("neither image nor ref is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		req := validRequest()
		req.Image = ""
		_, err := s.Deploy(context.Background(), req)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unknown stack is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		req := validRequest()
		req.StackID = "nope"
		_, err := s.Deploy(context.Background(), req)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("deprecated service_id alias still works", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		req := DeployRequest{ServiceID: "svc-1", Image: "ghcr.io/onik/api:v1"}
		got, err := s.Deploy(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "svc-1", got.StackID)
	})
}

func TestDeploy_RepoCreateError(t *testing.T) {
	repo := newFakeRepo()
	repo.createErr = errors.New("disk full")
	s := newTestService(repo, newFakeBus())

	_, err := s.Deploy(context.Background(), validRequest())
	require.Error(t, err)
	assert.ErrorIs(t, err, repo.createErr)
}

func TestDeploy_OneActivePerStack(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, newFakeBus())
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d0", StackID: "svc-1", Service: "api", Status: StatusPending, CreatedAt: now, UpdatedAt: now}))

	_, err := s.Deploy(context.Background(), validRequest())
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrConflict))
	assert.Contains(t, err.Error(), "active deploy")
}

func TestDeploy_PublishOnly(t *testing.T) {
	repo := newFakeRepo()
	bus := newFakeBus()
	s := newTestService(repo, bus)

	got, err := s.Deploy(context.Background(), validRequest())
	require.NoError(t, err)
	assert.Equal(t, StatusPending, got.Status, "publish-only deploy stays pending; the runner owns the rest")
	assert.Equal(t, "svc-1", got.StackID)
	assert.Equal(t, "svc-1", got.ServiceID, "deprecated alias mirrors StackID")
	assert.Equal(t, "api", got.Service)
	assert.Equal(t, "10.0.0.1:22", got.Target)
	assert.Equal(t, StrategyCompose, got.Strategy)

	reqEv := repo.of(TopicDeployRequested)
	require.NotNil(t, reqEv.Payload, "deploy.requested was enqueued via the outbox")
	payload := deployRequestedPayload(t, reqEv)
	assert.Equal(t, "deploy", payload.Kind)
	assert.Equal(t, "api", payload.Service)
	assert.Equal(t, "10.0.0.1:22", payload.Target)
	assert.Equal(t, "ghcr.io/onik/api:v1", payload.Image)
	assert.Equal(t, "compose", payload.Strategy)
	assert.Equal(t, map[string]string{"PORT": ""}, payload.Env, "env values are redacted to empty strings before the event leaves the deploy domain; only the keys travel")
	assert.Equal(t, "docker-compose.yml", payload.ComposePath)
	assert.Empty(t, repo.of(TopicDeployStatusChanged).Payload, "the deploy domain never publishes terminal states")
}

// TestDeploy_PublishRedactsEnvValues is ticket 14's guard: a secret placed in
// a stack's Env must never appear in the deploy.requested payload, however
// it is marshaled onto the wire.
func TestDeploy_PublishRedactsEnvValues(t *testing.T) {
	repo := newFakeRepo()
	bus := newFakeBus()
	stacks := newFakeStackRepo()
	stack := validStack()
	stack.Env = map[string]string{"DB_PASSWORD": "sup3rSecretValue", "PORT": "8080"}
	require.NoError(t, stacks.Create(context.Background(), &stack))
	s := newTestServiceWith(repo, stacks, newFakeContainerRepo(), newFakeProjects(), bus)

	_, err := s.Deploy(context.Background(), validRequest())
	require.NoError(t, err)

	raw, err := json.Marshal(repo.of(TopicDeployRequested).Payload)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "sup3rSecretValue", "the secret value must never reach the published payload")
	assert.Contains(t, string(raw), "DB_PASSWORD", "the key still travels so consumers know which env vars exist")

	payload := deployRequestedPayload(t, repo.of(TopicDeployRequested))
	assert.Equal(t, map[string]string{"DB_PASSWORD": "", "PORT": ""}, payload.Env)
}

// deployRequestedPayload decodes an outbox event's payload as
// DeployRequestedEvent, round-tripping through JSON the way the real relay
// does (repo.of returns the Go value the use-case handed to the repo, not
// the wire bytes).
func deployRequestedPayload(t *testing.T, evt eventbus.OutboxEvent) DeployRequestedEvent {
	t.Helper()
	b, err := json.Marshal(evt.Payload)
	require.NoError(t, err)
	var payload DeployRequestedEvent
	require.NoError(t, json.Unmarshal(b, &payload))
	return payload
}

// TestDeploy_GatewayJoinLastStep proves a stack whose network already has a
// gateway watching it carries gateway_container/join_networks on the deploy.requested payload, so the
// runner's last deploy step rejoins it (a compose deploy can recreate its own network on every `up -d`).
func TestDeploy_GatewayJoinLastStep(t *testing.T) {
	repo := newFakeRepo()
	bus := newFakeBus()
	s := newTestService(repo, bus)
	s.SetGatewayJoin(fakeGatewayJoin{containers: map[string]string{"api_default": "gateway-container"}})

	_, err := s.Deploy(context.Background(), validRequest())
	require.NoError(t, err)

	payload := deployRequestedPayload(t, repo.of(TopicDeployRequested))
	assert.Equal(t, "gateway-container", payload.GatewayContainer)
	assert.Equal(t, []string{"api_default"}, payload.JoinNetworks)
}

// TestDeploy_NoGatewayJoinWhenNoneServesTheNetwork covers the common case: no gateway watches this stack's
// network yet, so the deploy carries no join step at all.
func TestDeploy_NoGatewayJoinWhenNoneServesTheNetwork(t *testing.T) {
	repo := newFakeRepo()
	bus := newFakeBus()
	s := newTestService(repo, bus)
	s.SetGatewayJoin(fakeGatewayJoin{containers: map[string]string{}})

	_, err := s.Deploy(context.Background(), validRequest())
	require.NoError(t, err)

	payload := deployRequestedPayload(t, repo.of(TopicDeployRequested))
	assert.Empty(t, payload.GatewayContainer)
	assert.Empty(t, payload.JoinNetworks)
}

func TestDeploy_PortsPassThrough(t *testing.T) {
	repo := newFakeRepo()
	bus := newFakeBus()
	stacks := newFakeStackRepo()
	stack := validStack()
	stack.Strategy = StrategyRun
	stack.ComposePath = ""
	stack.DockerNetwork = "net1"
	stack.Ports = []string{"80:80", "443:443"}
	require.NoError(t, stacks.Create(context.Background(), &stack))
	s := newTestServiceWith(repo, stacks, newFakeContainerRepo(), newFakeProjects(), bus)

	req := DeployRequest{StackID: "svc-1", Image: "traefik:v3"}
	_, err := s.Deploy(context.Background(), req)
	require.NoError(t, err)

	payload := deployRequestedPayload(t, repo.of(TopicDeployRequested))
	assert.Equal(t, []string{"80:80", "443:443"}, payload.Ports)
	assert.Equal(t, "net1", payload.Network)
}

func TestDeploy_RefBuildKind(t *testing.T) {
	repo := newFakeRepo()
	bus := newFakeBus()
	stacks := newFakeStackRepo()
	stack := validStack()
	stack.BuildSource = &BuildSource{RepoOwner: "org", RepoName: "app", Branch: "main", Dockerfile: "Dockerfile"}
	requireStack(t, stacks, stack)
	s := newTestServiceWith(repo, stacks, newFakeContainerRepo(), newFakeProjects(), bus)

	req := DeployRequest{StackID: "svc-1", Ref: "main"}
	got, err := s.Deploy(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, StatusPending, got.Status)
	assert.Equal(t, KindBuild, got.Kind, "a ref trigger records a build-kind deploy")

	payload := deployRequestedPayload(t, repo.of(TopicDeployRequested))
	assert.Equal(t, "build", payload.Kind)
	assert.Equal(t, "main", payload.Ref)
	assert.Equal(t, "org/app", payload.Repo)
	assert.Equal(t, "Dockerfile", payload.Dockerfile)
}

func TestDeploy_RefBuildRequiresBuildSource(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, newFakeBus())

	_, err := s.Deploy(context.Background(), DeployRequest{StackID: "svc-1", Ref: "main"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	assert.Contains(t, err.Error(), "build source")
	assert.Empty(t, repo.of(TopicDeployRequested).Payload, "nothing is enqueued for an unbuildable stack")
}

func TestDeploy_Provenance(t *testing.T) {
	repo := newFakeRepo()
	bus := newFakeBus()
	s := newTestService(repo, bus)

	req := validRequest()
	req.TriggeredBy = "user-1"
	req.RuleID = "rule-1"
	req.RuleName = "auto deploy"
	req.TicketID = "T-42"
	req.PRNumber = 7
	got, err := s.Deploy(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "user-1", got.TriggeredBy)
	assert.Equal(t, "rule-1", got.RuleID)
	assert.Equal(t, "T-42", got.TicketID)
	assert.Equal(t, 7, got.PRNumber)
}

func TestRollback(t *testing.T) {
	t.Run("rolls back to last healthy image", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", StackID: "svc-1", Service: "api", Image: "img:v1", Status: StatusHealthy, CreatedAt: now, UpdatedAt: now}))
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d2", StackID: "svc-1", Service: "api", Image: "img:v2", Status: StatusHealthy, CreatedAt: now.Add(time.Hour), UpdatedAt: now.Add(time.Hour)}))
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d3", StackID: "svc-1", Service: "api", Image: "img:v3", Status: StatusFailed, CreatedAt: now.Add(2 * time.Hour), UpdatedAt: now.Add(2 * time.Hour)}))

		got, err := s.Rollback(context.Background(), "svc-1", "user-1")
		require.NoError(t, err)
		assert.Equal(t, "img:v2", got.Image, "rollback redeploys the last healthy image, not the failed one")
		assert.Equal(t, "user-1", got.TriggeredBy)

		payload := deployRequestedPayload(t, repo.of(TopicDeployRequested))
		assert.Equal(t, "img:v2", payload.Image)
	})
	t.Run("no healthy deploy is a conflict", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.Rollback(context.Background(), "svc-1", "user-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("blank stack is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.Rollback(context.Background(), " ", "user-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestStackCRUD(t *testing.T) {
	t.Run("create derives the slug, persists, and publishes stack.created", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())

		input := validStack()
		input.ID = ""
		input.Slug = ""
		got, err := s.CreateStack(context.Background(), input, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, got.ID)
		assert.Equal(t, "api", got.Slug)
		assert.True(t, got.Managed)
		assert.Contains(t, stacks.topics(), TopicStackCreated)

		evt := stacks.outbox[len(stacks.outbox)-1]
		require.Equal(t, TopicStackCreated, evt.Topic)
		b, err := json.Marshal(evt.Payload)
		require.NoError(t, err)
		var e StackEvent
		require.NoError(t, json.Unmarshal(b, &e))
		assert.Equal(t, "api", e.Stack.Name)
	})
	t.Run("a compose stack with no explicit compose path defaults to docker-compose.yml", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())

		input := validStack()
		input.ID = ""
		input.ComposePath = ""
		got, err := s.CreateStack(context.Background(), input, nil)
		require.NoError(t, err)
		assert.Equal(t, "docker-compose.yml", got.ComposePath)
	})
	t.Run("a run stack with no declared list gets exactly one pending service named after the slug", func(t *testing.T) {
		stacks := newFakeStackRepo()
		containers := newFakeContainerRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, containers, projects, newFakeBus())

		input := validStack()
		input.ID = ""
		input.Strategy = StrategyRun
		input.ComposePath = ""
		input.DockerNetwork = "net1"
		got, err := s.CreateStack(context.Background(), input, nil)
		require.NoError(t, err)

		svcs, err := s.ListServices(context.Background(), got.ID)
		require.NoError(t, err)
		require.Len(t, svcs, 1)
		assert.Equal(t, got.Slug, svcs[0].Name)
		assert.Equal(t, ServiceStatusPending, svcs[0].Status)
	})
	t.Run("a compose stack gets one pending service per declared entry", func(t *testing.T) {
		stacks := newFakeStackRepo()
		containers := newFakeContainerRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, containers, projects, newFakeBus())

		input := validStack()
		input.ID = ""
		got, err := s.CreateStack(context.Background(), input, map[string]Declared{
			"web": {Image: "nginx"},
			"db":  {Image: "postgres"},
		})
		require.NoError(t, err)

		svcs, err := s.ListServices(context.Background(), got.ID)
		require.NoError(t, err)
		require.Len(t, svcs, 2)
		var names []string
		for _, svc := range svcs {
			names = append(names, svc.Name)
			assert.Equal(t, ServiceStatusPending, svc.Status)
		}
		assert.ElementsMatch(t, []string{"web", "db"}, names)
	})
	t.Run("duplicate slug on same machine is a conflict", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		first := validStack()
		first.ID = ""
		_, err := s.CreateStack(context.Background(), first, nil)
		require.NoError(t, err)

		second := validStack()
		second.ID = ""
		_, err = s.CreateStack(context.Background(), second, nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
		assert.Contains(t, err.Error(), "already exists on machine")
	})
	t.Run("same name on different machine is allowed", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		first := validStack()
		first.ID = ""
		_, err := s.CreateStack(context.Background(), first, nil)
		require.NoError(t, err)

		dup := validStack()
		dup.ID = ""
		dup.Machine = "other:22"
		_, err = s.CreateStack(context.Background(), dup, nil)
		require.NoError(t, err)
	})
	t.Run("unknown project is invalid", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		input := validStack()
		input.ID = ""
		_, err := s.CreateStack(context.Background(), input, nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
		assert.Contains(t, err.Error(), "does not exist")
	})
	t.Run("build source must reference a project repo", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		projects.repos["proj-1"] = []string{"acme/api"}
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())

		stack := validStack()
		stack.ID = ""
		stack.BuildSource = &BuildSource{RepoOwner: "acme", RepoName: "api"}
		_, err := s.CreateStack(context.Background(), stack, nil)
		require.NoError(t, err)

		bad := validStack()
		bad.ID = ""
		bad.Machine = "other:22"
		bad.BuildSource = &BuildSource{RepoOwner: "other", RepoName: "thing"}
		_, err = s.CreateStack(context.Background(), bad, nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
		assert.Contains(t, err.Error(), "not in project")
	})
	t.Run("get requires id", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.GetStack(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("get returns the stack", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		got, err := s.GetStack(context.Background(), "svc-1")
		require.NoError(t, err)
		assert.Equal(t, "api", got.Name)
	})
	t.Run("list without a project spans every project", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		projects.exists["proj-2"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		first := validStack()
		first.ID = ""
		_, err := s.CreateStack(context.Background(), first, nil)
		require.NoError(t, err)
		other := validStack()
		other.ID = ""
		other.Name = "worker"
		other.ProjectID = "proj-2"
		_, err = s.CreateStack(context.Background(), other, nil)
		require.NoError(t, err)

		got, err := s.ListStacks(context.Background(), "")
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
	t.Run("list scopes to project", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		input := validStack()
		input.ID = ""
		_, err := s.CreateStack(context.Background(), input, nil)
		require.NoError(t, err)

		got, err := s.ListStacks(context.Background(), "proj-1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		other, err := s.ListStacks(context.Background(), "proj-2")
		require.NoError(t, err)
		assert.Empty(t, other)
	})
	t.Run("update publishes stack.updated and keeps the slug fixed", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		input := validStack()
		input.ID = ""
		created, err := s.CreateStack(context.Background(), input, nil)
		require.NoError(t, err)

		updated := *created
		updated.Name = "api-v2"
		got, err := s.UpdateStack(context.Background(), updated)
		require.NoError(t, err)
		assert.Equal(t, "api-v2", got.Name)
		assert.Equal(t, "api", got.Slug, "slug is derived once at creation and never re-derived")
		assert.Contains(t, stacks.topics(), TopicStackUpdated)

		persisted, err := s.GetStack(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "api-v2", persisted.Name)
	})
	t.Run("update requires id", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.UpdateStack(context.Background(), validStack())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("update with a build source links the repo and flips managed", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		unmanaged := validStack()
		unmanaged.Managed = false
		requireStack(t, stacks, unmanaged)

		updated := unmanaged
		updated.BuildSource = &BuildSource{RepoOwner: "onik97", RepoName: "app", Branch: "main", ComposePath: "docker-compose.yml"}
		got, err := s.UpdateStack(context.Background(), updated)
		require.NoError(t, err)
		assert.True(t, got.Managed, "attaching a build source flips an unmanaged stack to managed")
		assert.Contains(t, projects.repos["proj-1"], "onik97/app")
	})
	t.Run("update on an already-managed stack stays managed with no build source", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		managed := validStack()
		managed.BuildSource = &BuildSource{RepoOwner: "onik97", RepoName: "app"}
		requireStack(t, stacks, managed)
		projects.repos["proj-1"] = []string{"onik97/app"}

		updated := managed
		updated.BuildSource = nil
		got, err := s.UpdateStack(context.Background(), updated)
		require.NoError(t, err)
		assert.True(t, got.Managed, "an update never flips a managed stack back to unmanaged")
	})
	t.Run("update with no build source on either side leaves managed untouched and never links", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		unmanaged := validStack()
		unmanaged.Managed = false
		requireStack(t, stacks, unmanaged)

		updated := unmanaged
		updated.Name = "api-renamed"
		got, err := s.UpdateStack(context.Background(), updated)
		require.NoError(t, err)
		assert.False(t, got.Managed)
		assert.Empty(t, projects.repos["proj-1"])
	})
	t.Run("delete publishes stack.deleted, drops services, and keeps deploys", func(t *testing.T) {
		repo := newFakeRepo()
		stacks := newFakeStackRepo()
		containers := newFakeContainerRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(repo, stacks, containers, projects, newFakeBus())
		input := validStack()
		input.ID = ""
		input.Strategy = StrategyRun
		input.ComposePath = ""
		input.DockerNetwork = "net1"
		created, err := s.CreateStack(context.Background(), input, nil)
		require.NoError(t, err)
		now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", StackID: created.ID, Service: "api", Status: StatusHealthy, CreatedAt: now, UpdatedAt: now}))

		require.NoError(t, s.DeleteStack(context.Background(), created.ID))
		assert.Contains(t, stacks.topics(), TopicStackDeleted)
		_, err = s.GetStack(context.Background(), created.ID)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
		svcs, err := s.ListServices(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Empty(t, svcs, "services are dropped with the stack")
		deploys, err := s.ListByStackID(context.Background(), created.ID)
		require.NoError(t, err)
		require.Len(t, deploys, 1, "deploy history is kept on stack delete")
	})
	t.Run("delete releases every hostname via the exposure manager, by name and by container id", func(t *testing.T) {
		repo := newFakeRepo()
		stacks := newFakeStackRepo()
		containers := newFakeContainerRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(repo, stacks, containers, projects, newFakeBus())
		exposures := newFakeExposureManager()
		s.SetExposureManager(exposures)
		input := validStack()
		input.ID = ""
		input.Strategy = StrategyRun
		input.ComposePath = ""
		input.DockerNetwork = "net1"
		created, err := s.CreateStack(context.Background(), input, nil)
		require.NoError(t, err)
		svcs, err := s.ListServices(context.Background(), created.ID)
		require.NoError(t, err)
		require.Len(t, svcs, 1, "a run stack with no declared services defaults to one container")

		require.NoError(t, s.DeleteStack(context.Background(), created.ID))

		require.Len(t, exposures.removes, 1)
		assert.Equal(t, created.Name, exposures.removes[0].stackName)
		assert.Equal(t, []string{svcs[0].ID}, exposures.removes[0].containerIDs)
	})
	t.Run("delete requires id", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := s.DeleteStack(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("delete missing is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := s.DeleteStack(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestHandleStatusChanged(t *testing.T) {
	t.Run("malformed payload is fatal", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := s.HandleStatusChanged(context.Background(), eventbus.Event{Payload: json.RawMessage(`{not json`)})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("unknown status is fatal", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		ev := eventbus.Event{Payload: mustMarshal(t, DeployStatusChangedEvent{ID: "d1", Status: "exploded"})}
		err := s.HandleStatusChanged(context.Background(), ev)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("healthy lands the record and enqueues deploy.updated with the status", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Status: StatusPending}))
		require.NoError(t, s.HandleStatusChanged(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployStatusChangedEvent{ID: "d1", Status: string(StatusHealthy)})}))

		got, err := repo.GetByID(context.Background(), "d1")
		require.NoError(t, err)
		assert.Equal(t, StatusHealthy, got.Status)
		assert.Equal(t, []DeployUpdatedEvent{{ID: "d1", Status: "healthy"}}, repo.updatedEvents())
	})
	t.Run("failed appends the error as a phase-less log line", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Status: StatusPending}))
		require.NoError(t, s.HandleStatusChanged(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployStatusChangedEvent{ID: "d1", Status: string(StatusFailed), Error: "oom-killed"})}))

		got, err := repo.GetByID(context.Background(), "d1")
		require.NoError(t, err)
		assert.Equal(t, StatusFailed, got.Status)
		lines, err := s.Log(context.Background(), "d1")
		require.NoError(t, err)
		require.Len(t, lines, 1)
		assert.Equal(t, LogLine{Seq: 1, TS: s.now().UnixMilli(), Phase: "", Text: "deploy failed: oom-killed"}, lines[0])
	})
	t.Run("failed with a log write error is reported", func(t *testing.T) {
		repo := newFakeRepo()
		repo.appendErr = errors.New("disk full")
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Status: StatusPending}))
		err := s.HandleStatusChanged(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployStatusChangedEvent{ID: "d1", Status: string(StatusFailed), Error: "oom-killed"})})
		require.ErrorContains(t, err, "disk full")
	})
	t.Run("invalid transition is rejected", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		// healthy is terminal; nothing may move a healthy deploy.
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Status: StatusHealthy}))
		err := s.HandleStatusChanged(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployStatusChangedEvent{ID: "d1", Status: string(StatusFailed)})})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("redelivered terminal event is a no-op (idempotent)", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Status: StatusFailed}))
		// at-least-once redelivery of the same terminal event must not error.
		require.NoError(t, s.HandleStatusChanged(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployStatusChangedEvent{ID: "d1", Status: string(StatusFailed), Error: "cancelled"})}))
		got, err := repo.GetByID(context.Background(), "d1")
		require.NoError(t, err)
		assert.Equal(t, StatusFailed, got.Status)
		assert.Empty(t, repo.updatedEvents(), "an unchanged status announces nothing")
	})
	t.Run("missing deploy is an error", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := s.HandleStatusChanged(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployStatusChangedEvent{ID: "nope", Status: string(StatusHealthy)})})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("healthy with an address stores it on the record", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Status: StatusPending}))
		require.NoError(t, s.HandleStatusChanged(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployStatusChangedEvent{ID: "d1", Status: string(StatusHealthy), Address: "172.18.0.4"})}))

		got, err := repo.GetByID(context.Background(), "d1")
		require.NoError(t, err)
		assert.Equal(t, "172.18.0.4", got.Address)
	})
	t.Run("an event without an address leaves the record untouched", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Status: StatusPending, Address: "172.18.0.4"}))
		require.NoError(t, s.HandleStatusChanged(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployStatusChangedEvent{ID: "d1", Status: string(StatusHealthy)})}))

		got, err := repo.GetByID(context.Background(), "d1")
		require.NoError(t, err)
		assert.Equal(t, "172.18.0.4", got.Address, "an event with no address must not clear a previously reported one")
	})

	t.Run("reconcile upserts observed services and stops missing ones", func(t *testing.T) {
		repo := newFakeRepo()
		containers := newFakeContainerRepo()
		stacks := newFakeStackRepo()
		require.NoError(t, stacks.Create(context.Background(), &Stack{ID: "svc-1", Name: "api", Slug: "api"}))
		s := newTestServiceWith(repo, stacks, containers, newFakeProjects(), newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", StackID: "svc-1", Status: StatusPending}))
		require.NoError(t, containers.Create(context.Background(), &Container{
			ID: "c-web", StackID: "svc-1", Name: "web", Declared: Declared{Image: "nginx"}, Status: ServiceStatusPending,
		}))
		require.NoError(t, containers.Create(context.Background(), &Container{
			ID: "c-worker", StackID: "svc-1", Name: "worker", Status: ServiceStatusPending,
		}))

		ev := DeployStatusChangedEvent{
			ID: "d1", Status: string(StatusHealthy),
			Services: []ObservedService{
				{
					Name: "web", ContainerName: "api-web-1", Image: "nginx:1", Status: "healthy",
					Networks: []Network{{Name: "api_default", Address: "172.18.0.2"}},
					Ports:    []string{"8080:80/tcp"},
				},
				{Name: "cache", ContainerName: "api-cache-1", Image: "redis:7", Status: "running"},
			},
		}
		require.NoError(t, s.HandleStatusChanged(context.Background(), eventbus.Event{Payload: mustMarshal(t, ev)}))

		services, err := containers.ListByStack(context.Background(), "svc-1")
		require.NoError(t, err)
		byName := map[string]*Container{}
		for _, svc := range services {
			byName[svc.Name] = svc
		}
		require.Len(t, byName, 3)

		web := byName["web"]
		assert.Equal(t, "c-web", web.ID, "an observed service matching an existing row keeps its id")
		assert.Equal(t, ServiceStatusHealthy, web.Status)
		assert.Equal(t, "nginx", web.Declared.Image, "the compose parse's declared data survives an observation upsert")
		assert.Equal(t, "api-web-1", web.ContainerName)
		assert.Equal(t, []Network{{Name: "api_default", Address: "172.18.0.2"}}, web.Networks)
		assert.Equal(t, []string{"8080:80/tcp"}, web.Ports)

		worker := byName["worker"]
		assert.Equal(t, ServiceStatusStopped, worker.Status, "a service missing from the report is marked stopped, never deleted")

		cache := byName["cache"]
		assert.NotEmpty(t, cache.ID, "a container the report finds with no matching row gets a fresh id")
		assert.Equal(t, ServiceStatusRunning, cache.Status)
	})

	t.Run("a failed deploy with no report leaves existing services untouched", func(t *testing.T) {
		repo := newFakeRepo()
		containers := newFakeContainerRepo()
		stacks := newFakeStackRepo()
		require.NoError(t, stacks.Create(context.Background(), &Stack{ID: "svc-1", Name: "api", Slug: "api"}))
		s := newTestServiceWith(repo, stacks, containers, newFakeProjects(), newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", StackID: "svc-1", Status: StatusPending}))
		require.NoError(t, containers.Create(context.Background(), &Container{ID: "c-web", StackID: "svc-1", Name: "web", Status: ServiceStatusPending}))

		require.NoError(t, s.HandleStatusChanged(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployStatusChangedEvent{ID: "d1", Status: string(StatusFailed), Error: "boom"})}))

		svc, err := containers.Get(context.Background(), "c-web")
		require.NoError(t, err)
		assert.Equal(t, ServiceStatusPending, svc.Status, "no report means no reconcile; a failed deploy never touches services")
	})
}

func TestService_Reads(t *testing.T) {
	repo := newFakeRepo()
	bus := newFakeBus()
	s := newTestService(repo, bus)
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Service: "api", Status: StatusHealthy, CreatedAt: now}))
	require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d2", Service: "worker", Status: StatusPending, CreatedAt: now}))

	t.Run("get requires id", func(t *testing.T) {
		_, err := s.Get(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("get returns the record", func(t *testing.T) {
		got, err := s.Get(context.Background(), "d1")
		require.NoError(t, err)
		assert.Equal(t, "api", got.Service)
	})
	t.Run("get missing is not found", func(t *testing.T) {
		_, err := s.Get(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("list returns all", func(t *testing.T) {
		got, err := s.List(context.Background())
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
	t.Run("list by service requires service", func(t *testing.T) {
		_, err := s.ListByService(context.Background(), "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("list by service scopes", func(t *testing.T) {
		got, err := s.ListByService(context.Background(), "api")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "d1", got[0].ID)
	})
	t.Run("list by stack id scopes", func(t *testing.T) {
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d3", StackID: "svc-1", Service: "api", Status: StatusHealthy, CreatedAt: now}))
		got, err := s.ListByStackID(context.Background(), "svc-1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "d3", got[0].ID)
	})
	t.Run("list by status scopes", func(t *testing.T) {
		got, err := s.ListByStatus(context.Background(), StatusPending)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "d2", got[0].ID)
	})
	t.Run("list by unknown status is invalid", func(t *testing.T) {
		_, err := s.ListByStatus(context.Background(), Status("bogus"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestHandleLog(t *testing.T) {
	t.Run("splits the batch into one row per line and drops the trailing newline", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Status: StatusRunning}))
		ev := DeployLogEvent{ID: "d1", Phase: "checkout", Log: "clone org/app@main\nCloning into '/data/repo'...\n", TS: 1695379028112}
		require.NoError(t, s.HandleLog(context.Background(), eventbus.Event{Payload: mustMarshal(t, ev)}))

		lines, err := s.Log(context.Background(), "d1")
		require.NoError(t, err)
		assert.Equal(t, []LogLine{
			{Seq: 1, TS: 1695379028112, Phase: "checkout", Text: "clone org/app@main"},
			{Seq: 2, TS: 1695379028112, Phase: "checkout", Text: "Cloning into '/data/repo'..."},
		}, lines)
		assert.Equal(t, []DeployUpdatedEvent{{ID: "d1", Status: "running"}}, repo.updatedEvents(), "the batch commits with its own deploy.updated row")
	})
	t.Run("a blank line inside the batch is kept, only the trailing one is dropped", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Status: StatusRunning}))
		ev := DeployLogEvent{ID: "d1", Phase: "build", Log: "a\n\nb", TS: 5}
		require.NoError(t, s.HandleLog(context.Background(), eventbus.Event{Payload: mustMarshal(t, ev)}))

		lines, err := s.Log(context.Background(), "d1")
		require.NoError(t, err)
		require.Len(t, lines, 3)
		assert.Equal(t, "", lines[1].Text)
	})
	t.Run("an empty batch writes nothing", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Status: StatusRunning}))
		require.NoError(t, s.HandleLog(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployLogEvent{ID: "d1", Phase: "build", Log: "\n"})}))
		lines, err := s.Log(context.Background(), "d1")
		require.NoError(t, err)
		assert.Empty(t, lines)
		assert.Empty(t, repo.updatedEvents())
	})
	t.Run("a read error before the append is returned for retry", func(t *testing.T) {
		repo := newFakeRepo()
		repo.getErr = errors.New("locked")
		s := newTestService(repo, newFakeBus())
		err := s.HandleLog(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployLogEvent{ID: "d1", Phase: "build", Log: "x"})})
		require.ErrorContains(t, err, "locked")
		assert.False(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("malformed payload is fatal", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := s.HandleLog(context.Background(), eventbus.Event{Payload: json.RawMessage(`{`)})
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("missing id is fatal", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := s.HandleLog(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployLogEvent{Log: "x"})})
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("unknown deploy is fatal, not retried", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := s.HandleLog(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployLogEvent{ID: "nope", Phase: "build", Log: "x"})})
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("a transient write error is returned for retry", func(t *testing.T) {
		repo := newFakeRepo()
		repo.appendErr = errors.New("locked")
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Status: StatusRunning}))
		err := s.HandleLog(context.Background(), eventbus.Event{Payload: mustMarshal(t, DeployLogEvent{ID: "d1", Phase: "build", Log: "x"})})
		require.ErrorContains(t, err, "locked")
		assert.False(t, errors.Is(err, apperrs.ErrFatal))
	})
}

func TestLog(t *testing.T) {
	t.Run("unknown deploy is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.Log(context.Background(), "nope")
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("a deploy with no output yields an empty, non-nil slice", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", Status: StatusPending}))
		lines, err := s.Log(context.Background(), "d1")
		require.NoError(t, err)
		assert.NotNil(t, lines)
		assert.Empty(t, lines)
	})
}

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func TestService_Cancel(t *testing.T) {
	t.Run("queued deploy publishes cancel_requested", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		seedDeploy(t, repo, &Deploy{ID: "d1", Service: "api", Status: StatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		s := newTestService(repo, bus)

		require.NoError(t, s.Cancel(context.Background(), "d1"))
		b, err := json.Marshal(repo.of(TopicDeployCancelRequested).Payload)
		require.NoError(t, err)
		var ev DeployCancelRequestedEvent
		require.NoError(t, json.Unmarshal(b, &ev))
		assert.Equal(t, "d1", ev.ID)
	})

	t.Run("running deploy publishes cancel_requested", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		seedDeploy(t, repo, &Deploy{ID: "d1", Service: "api", Status: StatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		s := newTestService(repo, bus)

		require.NoError(t, s.Cancel(context.Background(), "d1"))
		assert.NotEmpty(t, repo.of(TopicDeployCancelRequested).Topic)
	})

	t.Run("terminal deploy is a conflict", func(t *testing.T) {
		for _, st := range []Status{StatusHealthy, StatusFailed} {
			repo := newFakeRepo()
			seedDeploy(t, repo, &Deploy{ID: "d1", Service: "api", Status: st, CreatedAt: time.Now(), UpdatedAt: time.Now()})
			s := newTestService(repo, newFakeBus())

			err := s.Cancel(context.Background(), "d1")
			require.Error(t, err)
			assert.True(t, errors.Is(err, apperrs.ErrConflict))
		}
	})

	t.Run("missing deploy is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := s.Cancel(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})

	t.Run("blank id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := s.Cancel(context.Background(), "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})

	t.Run("outbox enqueue failure propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.cancelErr = assert.AnError
		seedDeploy(t, repo, &Deploy{ID: "d1", Service: "api", Status: StatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		s := newTestService(repo, newFakeBus())

		require.Error(t, s.Cancel(context.Background(), "d1"))
	})
}

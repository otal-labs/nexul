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
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func TestMCPTools_Shape(t *testing.T) {
	tools := MCPTools(newTestService(newFakeRepo(), newFakeBus()))
	require.Len(t, tools, 15)
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
	assert.ElementsMatch(t, []string{
		"deploy_get", "deploy_log", "deploy_list", "deploy_list_by_service", "deploy_list_by_status", "deploy_cancel",
		"service_list", "stack_create", "stack_deploy", "stack_get", "stack_list", "stack_update",
		"stack_delete", "stack_rollback", "machine_import",
	}, names)
}

func seedDeploy(t *testing.T, repo *fakeRepo, d *Deploy) {
	t.Helper()
	require.NoError(t, repo.Create(context.Background(), d))
}

func TestMCPTools_Get(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newFakeRepo()
		d := &Deploy{ID: "d1", Service: "api", Status: StatusHealthy, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		seedDeploy(t, repo, d)
		call := toolByName(t, MCPTools(newTestService(repo, newFakeBus())), "deploy_get").Call
		got, err := call(context.Background(), map[string]any{"id": "d1"})
		require.NoError(t, err)
		assert.Equal(t, "api", got.(*Deploy).Service)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "deploy_get").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unknown deploy is not found", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "deploy_get").Call
		_, err := call(context.Background(), map[string]any{"id": "nope"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestMCPTools_Log(t *testing.T) {
	t.Run("happy path returns the lines oldest first", func(t *testing.T) {
		repo := newFakeRepo()
		seedDeploy(t, repo, &Deploy{ID: "d1", Service: "api", Status: StatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		require.NoError(t, repo.AppendLogLines(context.Background(), "d1", []LogLine{
			{TS: 10, Phase: "checkout", Text: "clone org/app@main"},
			{TS: 20, Phase: "build", Text: "Step 1/3"},
		}))
		call := toolByName(t, MCPTools(newTestService(repo, newFakeBus())), "deploy_log").Call
		got, err := call(context.Background(), map[string]any{"id": "d1"})
		require.NoError(t, err)
		assert.Equal(t, []LogLine{
			{Seq: 1, TS: 10, Phase: "checkout", Text: "clone org/app@main"},
			{Seq: 2, TS: 20, Phase: "build", Text: "Step 1/3"},
		}, got)
	})
	t.Run("a deploy without output returns an empty list, not null", func(t *testing.T) {
		repo := newFakeRepo()
		seedDeploy(t, repo, &Deploy{ID: "d1", Status: StatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		call := toolByName(t, MCPTools(newTestService(repo, newFakeBus())), "deploy_log").Call
		got, err := call(context.Background(), map[string]any{"id": "d1"})
		require.NoError(t, err)
		assert.Equal(t, []LogLine{}, got)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "deploy_log").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unknown deploy is not found", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "deploy_log").Call
		_, err := call(context.Background(), map[string]any{"id": "nope"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestMCPTools_List(t *testing.T) {
	repo := newFakeRepo()
	seedDeploy(t, repo, &Deploy{ID: "d1", Service: "api", Status: StatusHealthy, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	call := toolByName(t, MCPTools(newTestService(repo, newFakeBus())), "deploy_list").Call
	got, err := call(context.Background(), map[string]any{})
	require.NoError(t, err)
	assert.Len(t, got.([]*Deploy), 1)
}

func TestMCPTools_ListByService(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newFakeRepo()
		seedDeploy(t, repo, &Deploy{ID: "d1", Service: "api", Status: StatusHealthy, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		seedDeploy(t, repo, &Deploy{ID: "d2", Service: "web", Status: StatusFailed, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		call := toolByName(t, MCPTools(newTestService(repo, newFakeBus())), "deploy_list_by_service").Call
		got, err := call(context.Background(), map[string]any{"service": "api"})
		require.NoError(t, err)
		ds := got.([]*Deploy)
		require.Len(t, ds, 1)
		assert.Equal(t, "d1", ds[0].ID)
	})
	t.Run("missing service is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "deploy_list_by_service").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_ListByStatus(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newFakeRepo()
		seedDeploy(t, repo, &Deploy{ID: "d1", Service: "api", Status: StatusFailed, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		seedDeploy(t, repo, &Deploy{ID: "d2", Service: "web", Status: StatusHealthy, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		call := toolByName(t, MCPTools(newTestService(repo, newFakeBus())), "deploy_list_by_status").Call
		got, err := call(context.Background(), map[string]any{"status": "failed"})
		require.NoError(t, err)
		ds := got.([]*Deploy)
		require.Len(t, ds, 1)
		assert.Equal(t, "d1", ds[0].ID)
	})
	t.Run("missing status is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "deploy_list_by_status").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_Cancel(t *testing.T) {
	t.Run("queued deploy publishes cancel_requested", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		seedDeploy(t, repo, &Deploy{ID: "d1", Service: "api", Status: StatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		call := toolByName(t, MCPTools(newTestService(repo, bus)), "deploy_cancel").Call
		got, err := call(context.Background(), map[string]any{"id": "d1"})
		require.NoError(t, err)
		body := got.(map[string]string)
		assert.Equal(t, "d1", body["id"])
		assert.Equal(t, "cancelling", body["status"])
		b, err := json.Marshal(repo.of(TopicDeployCancelRequested).Payload)
		require.NoError(t, err)
		var ev DeployCancelRequestedEvent
		require.NoError(t, json.Unmarshal(b, &ev))
		assert.Equal(t, "d1", ev.ID)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "deploy_cancel").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("terminal deploy is a conflict", func(t *testing.T) {
		repo := newFakeRepo()
		seedDeploy(t, repo, &Deploy{ID: "d1", Service: "api", Status: StatusHealthy, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		call := toolByName(t, MCPTools(newTestService(repo, newFakeBus())), "deploy_cancel").Call
		_, err := call(context.Background(), map[string]any{"id": "d1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})
}

func TestMCPTools_StackCreate(t *testing.T) {
	t.Run("creates a stack", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		projects.repos["proj-1"] = []string{"acme/api"}
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_create").Call
		got, err := call(context.Background(), map[string]any{
			"project_id": "proj-1", "name": "api", "machine": "10.0.0.1:22",
			"strategy": "compose", "compose_path": "/srv/api/docker-compose.yml",
			"env":          map[string]any{"PORT": "8080"},
			"build_source": map[string]any{"repo_owner": "acme", "repo_name": "api", "branch": "main", "dockerfile": "Dockerfile"},
		})
		require.NoError(t, err)
		gotStack := got.(*Stack)
		assert.Equal(t, "api", gotStack.Name)
		assert.Equal(t, "api", gotStack.Slug)
		assert.Equal(t, map[string]string{"PORT": "8080"}, gotStack.Env)
		require.NotNil(t, gotStack.BuildSource)
		assert.Equal(t, "acme", gotStack.BuildSource.RepoOwner)
		assert.Equal(t, "Dockerfile", gotStack.BuildSource.Dockerfile)
	})
	t.Run("run strategy with docker network", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_create").Call
		got, err := call(context.Background(), map[string]any{
			"project_id": "proj-1", "name": "worker", "machine": "10.0.0.2:22",
			"strategy": "run", "docker_network": "net1",
		})
		require.NoError(t, err)
		assert.Equal(t, StrategyRun, got.(*Stack).Strategy)
		assert.Equal(t, "net1", got.(*Stack).DockerNetwork)
	})
	t.Run("missing project is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_create").Call
		_, err := call(context.Background(), map[string]any{"name": "api", "machine": "h:22", "strategy": "run", "docker_network": "net"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		projects.repoErr = errors.New("db down")
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_create").Call
		_, err := call(context.Background(), map[string]any{
			"project_id": "proj-1", "name": "api", "machine": "10.0.0.1:22",
			"strategy": "compose", "compose_path": "/srv/api/docker-compose.yml",
			"build_source": map[string]any{"repo_owner": "acme", "repo_name": "api"},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, projects.repoErr)
	})
}

func TestMCPTools_StackGet(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		input := validStack()
		input.ID = ""
		created, err := s.CreateStack(context.Background(), input, nil)
		require.NoError(t, err)
		call := toolByName(t, MCPTools(s), "stack_get").Call
		got, err := call(context.Background(), map[string]any{"id": created.ID})
		require.NoError(t, err)
		assert.Equal(t, "api", got.(*Stack).Name)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "stack_get").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_StackList(t *testing.T) {
	t.Run("scopes to project", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		input := validStack()
		input.ID = ""
		_, err := s.CreateStack(context.Background(), input, nil)
		require.NoError(t, err)
		call := toolByName(t, MCPTools(s), "stack_list").Call
		got, err := call(context.Background(), map[string]any{"project_id": "proj-1"})
		require.NoError(t, err)
		require.Len(t, got.([]*Stack), 1)
	})
	t.Run("missing project is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "stack_list").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_StackUpdate(t *testing.T) {
	t.Run("updates the stack", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		input := validStack()
		input.ID = ""
		created, err := s.CreateStack(context.Background(), input, nil)
		require.NoError(t, err)
		call := toolByName(t, MCPTools(s), "stack_update").Call
		got, err := call(context.Background(), map[string]any{
			"id": created.ID, "project_id": "proj-1", "name": "api-v2", "machine": "10.0.0.1:22",
			"strategy": "compose", "compose_path": "/srv/api/docker-compose.yml",
		})
		require.NoError(t, err)
		assert.Equal(t, "api-v2", got.(*Stack).Name)
	})
}

func TestMCPTools_StackDelete(t *testing.T) {
	t.Run("deletes the stack", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		input := validStack()
		input.ID = ""
		created, err := s.CreateStack(context.Background(), input, nil)
		require.NoError(t, err)
		call := toolByName(t, MCPTools(s), "stack_delete").Call
		_, err = call(context.Background(), map[string]any{"id": created.ID})
		require.NoError(t, err)
		_, err = s.GetStack(context.Background(), created.ID)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestMCPTools_StackRollback(t *testing.T) {
	t.Run("rolls back to last healthy image", func(t *testing.T) {
		repo := newFakeRepo()
		now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
		seedDeploy(t, repo, &Deploy{ID: "d1", StackID: "svc-1", Service: "api", Image: "img:v1", Status: StatusHealthy, CreatedAt: now, UpdatedAt: now})
		s := newTestService(repo, newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_rollback").Call
		got, err := call(context.Background(), map[string]any{"stack_id": "svc-1"})
		require.NoError(t, err)
		assert.Equal(t, "img:v1", got.(*Deploy).Image)
	})
	t.Run("missing stack is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "stack_rollback").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("provenance records the mcp source", func(t *testing.T) {
		repo := newFakeRepo()
		now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
		seedDeploy(t, repo, &Deploy{ID: "d1", StackID: "svc-1", Service: "api", Image: "img:v1", Status: StatusHealthy, CreatedAt: now, UpdatedAt: now})
		s := newTestService(repo, newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_rollback").Call
		ctx := identity.WithActor(context.Background(), identity.Actor{ID: "user-1"})
		got, err := call(ctx, map[string]any{"stack_id": "svc-1"})
		require.NoError(t, err)
		assert.Equal(t, "user-1:mcp", got.(*Deploy).TriggeredBy)
	})
}

func TestMCPTools_StackDeploy(t *testing.T) {
	t.Run("builds from a ref", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		projects.repos["proj-1"] = []string{"acme/api"}
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		input := validStack()
		input.ID = ""
		input.BuildSource = &BuildSource{RepoOwner: "acme", RepoName: "api", Dockerfile: "Dockerfile"}
		created, err := s.CreateStack(context.Background(), input, nil)
		require.NoError(t, err)
		call := toolByName(t, MCPTools(s), "stack_deploy").Call
		ctx := identity.WithActor(context.Background(), identity.Actor{ID: "user-1"})
		got, err := call(ctx, map[string]any{"stack_id": created.ID, "ref": "main"})
		require.NoError(t, err)
		d := got.(*Deploy)
		assert.Equal(t, KindBuild, d.Kind)
		assert.Equal(t, "user-1:mcp", d.TriggeredBy)
	})
	t.Run("redeploys an image", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_deploy").Call
		got, err := call(context.Background(), map[string]any{"stack_id": "svc-1", "image": "img:v2"})
		require.NoError(t, err)
		assert.Equal(t, KindDeploy, got.(*Deploy).Kind)
	})
	t.Run("missing stack id is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "stack_deploy").Call
		_, err := call(context.Background(), map[string]any{"image": "img:v2"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_StackCreate_FromCandidate(t *testing.T) {
	t.Run("compose candidate", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		projects.repos["proj-1"] = []string{"acme/api"}
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_create").Call
		got, err := call(context.Background(), map[string]any{
			"project_id": "proj-1", "machine": "10.0.0.1:22",
			"build_source": map[string]any{"repo_owner": "acme", "repo_name": "api", "branch": "main"},
			"candidate": map[string]any{
				"kind": "compose", "path": "docker-compose.yml",
				"services": []any{
					map[string]any{"name": "web", "image": "nginx", "ports": []any{float64(80)}},
				},
			},
		})
		require.NoError(t, err)
		stack := got.(*Stack)
		assert.Equal(t, "api", stack.Name)
		assert.Equal(t, StrategyCompose, stack.Strategy)
		assert.Equal(t, "docker-compose.yml", stack.ComposePath)
		svcs, err := s.ListServices(context.Background(), stack.ID)
		require.NoError(t, err)
		require.Len(t, svcs, 1)
		assert.Equal(t, "web", svcs[0].Name)
		assert.Equal(t, "nginx", svcs[0].Declared.Image)
	})
	t.Run("dockerfile candidate is a stack of one, named after the slug", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_create").Call
		got, err := call(context.Background(), map[string]any{
			"project_id": "proj-1", "machine": "10.0.0.1:22", "name": "worker", "docker_network": "net1",
			"candidate": map[string]any{
				"kind": "dockerfile", "path": "Dockerfile",
				"services": []any{map[string]any{"name": "worker", "build": map[string]any{"dockerfile": "Dockerfile"}}},
			},
		})
		require.NoError(t, err)
		stack := got.(*Stack)
		assert.Equal(t, StrategyRun, stack.Strategy)
		svcs, err := s.ListServices(context.Background(), stack.ID)
		require.NoError(t, err)
		require.Len(t, svcs, 1)
		assert.Equal(t, stack.Slug, svcs[0].Name)
		assert.Equal(t, "Dockerfile", svcs[0].Declared.Build)
	})
	t.Run("deploy true enqueues the first deploy", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		projects.repos["proj-1"] = []string{"acme/api"}
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_create").Call
		got, err := call(context.Background(), map[string]any{
			"project_id": "proj-1", "machine": "10.0.0.1:22", "deploy": true,
			"build_source": map[string]any{"repo_owner": "acme", "repo_name": "api", "branch": "main", "dockerfile": "Dockerfile"},
			"candidate": map[string]any{
				"kind": "compose", "path": "docker-compose.yml",
				"services": []any{map[string]any{"name": "web", "image": "nginx"}},
			},
		})
		require.NoError(t, err)
		stack := got.(*Stack)
		deploys, err := s.ListByStackID(context.Background(), stack.ID)
		require.NoError(t, err)
		require.Len(t, deploys, 1)
		assert.Equal(t, KindBuild, deploys[0].Kind)
		assert.Equal(t, stack.ID, deploys[0].StackID)
	})
	t.Run("deploy true without a ref is invalid", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_create").Call
		_, err := call(context.Background(), map[string]any{
			"project_id": "proj-1", "machine": "10.0.0.1:22", "deploy": true,
			"candidate": map[string]any{
				"kind": "compose", "path": "docker-compose.yml",
				"services": []any{map[string]any{"name": "web", "image": "nginx"}},
			},
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("no name and no build source is invalid", func(t *testing.T) {
		s := newTestServiceWith(newFakeRepo(), newFakeStackRepo(), newFakeContainerRepo(), newFakeProjects(), newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_create").Call
		_, err := call(context.Background(), map[string]any{
			"project_id": "proj-1", "machine": "10.0.0.1:22",
			"candidate": map[string]any{"kind": "compose", "path": "docker-compose.yml"},
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unknown candidate kind is invalid", func(t *testing.T) {
		s := newTestServiceWith(newFakeRepo(), newFakeStackRepo(), newFakeContainerRepo(), newFakeProjects(), newFakeBus())
		call := toolByName(t, MCPTools(s), "stack_create").Call
		_, err := call(context.Background(), map[string]any{
			"project_id": "proj-1", "machine": "10.0.0.1:22", "name": "x",
			"candidate": map[string]any{"kind": "helm", "path": "chart"},
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_ServiceList(t *testing.T) {
	stacks := newFakeStackRepo()
	projects := newFakeProjects()
	projects.exists["proj-1"] = true
	containers := newFakeContainerRepo()
	s := newTestServiceWith(newFakeRepo(), stacks, containers, projects, newFakeBus())
	input := validStack()
	input.ID = ""
	created, err := s.CreateStack(context.Background(), input, map[string]Declared{"api": {Image: "img"}})
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "service_list").Call
	got, err := call(context.Background(), map[string]any{"stack_id": created.ID})
	require.NoError(t, err)
	require.Len(t, got.([]*Container), 1)
	assert.Equal(t, "api", got.([]*Container)[0].Name)
}

func TestMCPTools_MachineImport(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		s, stacks, _ := newImportTestService([]DiscoveredContainer{
			{Name: "myapp-web-1", Image: "nginx:latest", Status: "running"},
		})
		call := toolByName(t, MCPTools(s), "machine_import").Call
		got, err := call(context.Background(), map[string]any{
			"machine_id": "m-1", "project_id": "proj-1",
			"stacks": []any{
				map[string]any{"project": "myapp", "containers": []any{"myapp-web-1"}},
			},
		})
		require.NoError(t, err)
		result := got.(*ImportResult)
		require.Len(t, result.Stacks, 1)
		assert.Equal(t, "myapp", result.Stacks[0].Name)
		require.Len(t, stacks.stored, 1)
	})
	t.Run("missing machine id is invalid", func(t *testing.T) {
		s, _, _ := newImportTestService(nil)
		call := toolByName(t, MCPTools(s), "machine_import").Call
		_, err := call(context.Background(), map[string]any{"project_id": "proj-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing project id is invalid", func(t *testing.T) {
		s, _, _ := newImportTestService(nil)
		call := toolByName(t, MCPTools(s), "machine_import").Call
		_, err := call(context.Background(), map[string]any{"machine_id": "m-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func toolByName(t *testing.T, tools []mcptool.Tool, name string) mcptool.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found", name)
	return mcptool.Tool{}
}

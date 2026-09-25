package deploy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func call(t *testing.T, s *Service, name, args string) (any, error) {
	t.Helper()
	return callAs(t, t.Context(), s, name, args)
}

func callAs(t *testing.T, ctx context.Context, s *Service, name, args string) (any, error) {
	t.Helper()
	for _, tool := range MCPTools(s) {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

// asJSON renders a result the way the adapter sends it, so a test can assert on what the model reads.
func asJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func at(minute int) time.Time { return time.Date(2026, 9, 1, 12, minute, 0, 0, time.UTC) }

// stackFixture is a service with project proj-1 (repository acme/api) and one created compose stack.
func stackFixture(t *testing.T) (*Service, *Stack, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	projects := newFakeProjects()
	projects.exists["proj-1"] = true
	projects.repos["proj-1"] = []string{"acme/api"}
	s := newTestServiceWith(repo, newFakeStackRepo(), newFakeContainerRepo(), projects, newFakeBus())
	in := validStack()
	in.ID = ""
	in.Env = map[string]string{"PORT": "8080", "TUNNEL_TOKEN": "secret-token"}
	in.Mounts = []string{"/srv:/srv"}
	in.Command = []string{"serve"}
	in.BuildSource = &BuildSource{RepoOwner: "acme", RepoName: "api", Branch: "main", Dockerfile: "Dockerfile"}
	in.BranchDeployRules = []BranchDeployRule{{Pattern: "dev", DockerNetwork: "qa", NameSuffix: "qa", Overrides: map[string]string{"DB": "postgres://qa"}}}
	created, err := s.CreateStack(t.Context(), in, map[string]Declared{"api": {Image: "img"}})
	require.NoError(t, err)
	return s, created, repo
}

func TestMCPTools_Surface(t *testing.T) {
	var names []string
	for _, tool := range MCPTools(newTestService(newFakeRepo(), newFakeBus())) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
		assert.NotEmpty(t, tool.Description, tool.Name)
		assert.Equal(t, "object", tool.InputSchema.Type, tool.Name)
	}
	assert.Equal(t, []string{
		"deploy_list", "deploy_get", "deploy_cancel", "stack_list", "stack_get", "stack_create", "stack_update",
		"stack_delete", "stack_deploy", "machine_import",
	}, names)
}

func TestMCPTools_Errors(t *testing.T) {
	s, stack, repo := stackFixture(t)
	seedDeploy(t, repo, &Deploy{ID: "done", StackID: stack.ID, Status: StatusHealthy, CreatedAt: at(1)})
	tests := []struct {
		name, tool, args string
		want             error
		msg              string
	}{
		{"deploy_list rejects an unknown status", "deploy_list", `{"status":"queued"}`, apperrs.ErrInvalid, "pending, running, healthy, failed"},
		{"deploy_list rejects an unknown key", "deploy_list", `{"service":"api"}`, apperrs.ErrInvalid, ""},
		{"deploy_get needs an id", "deploy_get", `{}`, apperrs.ErrInvalid, ""},
		{"deploy_get of a missing deploy", "deploy_get", `{"id":"nope"}`, apperrs.ErrNotFound, "deploy_list"},
		{"deploy_cancel of a missing deploy", "deploy_cancel", `{"id":"nope"}`, apperrs.ErrNotFound, "deploy_list"},
		{"deploy_cancel of a finished deploy", "deploy_cancel", `{"id":"done"}`, apperrs.ErrConflict, ""},
		{"stack_get of a missing stack", "stack_get", `{"id":"nope"}`, apperrs.ErrNotFound, "stack_list"},
		{"stack_update of a missing stack", "stack_update", `{"id":"nope","name":"x"}`, apperrs.ErrNotFound, "stack_list"},
		{"stack_update rejects an unknown strategy", "stack_update", `{"id":"` + stack.ID + `","strategy":"helm"}`, apperrs.ErrInvalid, ""},
		{"stack_update rejects a wrongly typed field", "stack_update", `{"id":"` + stack.ID + `","ports":"8080:80"}`, apperrs.ErrInvalid, ""},
		{"stack_update rejects an in-place rule with overrides", "stack_update", `{"id":"` + stack.ID + `","branch_deploy_rules":[{"pattern":"main","docker_network":"n","overrides":{"A":"b"}}]}`, apperrs.ErrInvalid, ""},
		{"stack_delete of a missing stack", "stack_delete", `{"id":"nope"}`, apperrs.ErrNotFound, "stack_list"},
		{"stack_deploy needs ref, image, or rollback", "stack_deploy", `{"id":"` + stack.ID + `"}`, apperrs.ErrInvalid, "rollback"},
		{"stack_deploy rollback takes no ref", "stack_deploy", `{"id":"` + stack.ID + `","rollback":true,"ref":"main"}`, apperrs.ErrInvalid, ""},
		{"stack_deploy of a missing stack", "stack_deploy", `{"id":"nope","image":"img:1"}`, apperrs.ErrNotFound, "stack_list"},
		{"stack_deploy rollback with no healthy deploy", "stack_deploy", `{"id":"nope","rollback":true}`, apperrs.ErrNotFound, "deploy_list"},
		{"stack_deploy rollback with no healthy image", "stack_deploy", `{"id":"` + stack.ID + `","rollback":true}`, apperrs.ErrConflict, "roll back"},
		{"stack_create needs a strategy without a candidate", "stack_create", `{"project_id":"proj-1","machine":"m","name":"x"}`, apperrs.ErrInvalid, "compose or run"},
		{"stack_create needs a machine", "stack_create", `{"project_id":"proj-1","name":"x","strategy":"compose"}`, apperrs.ErrInvalid, ""},
		{"stack_create of an unknown project", "stack_create", `{"project_id":"ghost","machine":"m","name":"x","strategy":"compose"}`, apperrs.ErrInvalid, ""},
		{"stack_create rejects an unknown candidate kind", "stack_create", `{"project_id":"proj-1","machine":"m","name":"x","candidate":{"kind":"helm","path":"chart"}}`, apperrs.ErrInvalid, "compose or dockerfile"},
		{"stack_create from a candidate needs a name", "stack_create", `{"project_id":"proj-1","machine":"m","candidate":{"kind":"compose","path":"docker-compose.yml"}}`, apperrs.ErrInvalid, "name"},
		{"machine_import needs a project", "machine_import", `{"id":"m-1"}`, apperrs.ErrInvalid, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := call(t, s, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.want)
			assert.Contains(t, err.Error(), tt.msg)
		})
	}
}

func TestDeployList(t *testing.T) {
	repo := newFakeRepo()
	seedDeploy(t, repo, &Deploy{ID: "d1", StackID: "s1", Status: StatusHealthy, CreatedAt: at(1)})
	seedDeploy(t, repo, &Deploy{ID: "d2", StackID: "s2", Status: StatusFailed, CreatedAt: at(2)})
	seedDeploy(t, repo, &Deploy{ID: "d3", StackID: "s1", Status: StatusFailed, CreatedAt: at(3)})
	s := newTestService(repo, newFakeBus())
	ids := func(page mcptool.Page[deployResult]) []string {
		var out []string
		for _, d := range page.Items {
			out = append(out, d.ID)
		}
		return out
	}
	tests := []struct {
		name string
		args string
		want []string
	}{
		{"everything, newest first", `{}`, []string{"d3", "d2", "d1"}},
		{"one stack", `{"stack_id":"s1"}`, []string{"d1", "d3"}},
		{"one status", `{"status":"healthy"}`, []string{"d1"}},
		{"a stack and a status", `{"stack_id":"s1","status":"failed"}`, []string{"d3"}},
		{"a page", `{"limit":1,"offset":1}`, []string{"d2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := call(t, s, "deploy_list", tt.args)
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.want, ids(got.(mcptool.Page[deployResult])))
		})
	}
	got, err := call(t, s, "deploy_list", `{}`)
	require.NoError(t, err)
	assert.Equal(t, []string{"d3", "d2", "d1"}, ids(got.(mcptool.Page[deployResult])), "newest first")
}

func TestDeployGet_ReturnsTheLogTail(t *testing.T) {
	repo := newFakeRepo()
	seedDeploy(t, repo, &Deploy{ID: "d1", StackID: "s1", Service: "api", Target: "prod", Status: StatusRunning, CreatedAt: at(1)})
	require.NoError(t, repo.AppendLogLines(t.Context(), "d1", []LogLine{
		{TS: 10, Phase: "checkout", Text: "clone"}, {TS: 20, Phase: "build", Text: "step 1"}, {TS: 30, Phase: "build", Text: "step 2"},
	}))
	s := newTestService(repo, newFakeBus())

	got, err := call(t, s, "deploy_get", `{"id":"d1","log_lines":2}`)
	require.NoError(t, err)
	d := got.(deployDetail)
	assert.Equal(t, "api", d.Stack)
	assert.Equal(t, "prod", d.Machine)
	assert.Equal(t, 3, d.LogTotal)
	require.Len(t, d.Log, 2)
	assert.Equal(t, "step 2", d.Log[1].Text)

	got, err = call(t, s, "deploy_get", `{"id":"d1"}`)
	require.NoError(t, err)
	assert.Len(t, got.(deployDetail).Log, 3, "the default tail covers a short log")
}

func TestDeployCancel_EnqueuesTheCancel(t *testing.T) {
	repo := newFakeRepo()
	seedDeploy(t, repo, &Deploy{ID: "d1", Status: StatusPending, CreatedAt: at(1)})
	got, err := call(t, newTestService(repo, newFakeBus()), "deploy_cancel", `{"id":"d1"}`)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"id": "d1", "status": "cancelling"}, got)
	assert.Equal(t, TopicDeployCancelRequested, repo.of(TopicDeployCancelRequested).Topic)
}

func TestStackList(t *testing.T) {
	s, stack, _ := stackFixture(t)
	got, err := call(t, s, "stack_list", `{"project_id":"proj-1"}`)
	require.NoError(t, err)
	page := got.(mcptool.Page[stackSummary])
	require.Len(t, page.Items, 1)
	assert.Equal(t, stack.ID, page.Items[0].ID)
	assert.NotContains(t, asJSON(t, got), "secret-token")

	got, err = call(t, s, "stack_list", `{"project_id":"other"}`)
	require.NoError(t, err)
	assert.Empty(t, got.(mcptool.Page[stackSummary]).Items)
}

func TestStackGet_CarriesServicesDeploysAndOnlyEnvKeys(t *testing.T) {
	s, stack, repo := stackFixture(t)
	for i := range 12 {
		seedDeploy(t, repo, &Deploy{ID: "d" + string(rune('a'+i)), StackID: stack.ID, Status: StatusHealthy, CreatedAt: at(i)})
	}
	got, err := call(t, s, "stack_get", `{"id":"`+stack.ID+`"}`)
	require.NoError(t, err)
	d := got.(stackDetail)
	assert.Equal(t, "api", d.Name)
	require.Len(t, d.Services, 1)
	assert.Equal(t, "api", d.Services[0].Name)
	assert.Len(t, d.RecentDeploys, recentDeploys)
	assert.Equal(t, []string{"PORT", "TUNNEL_TOKEN"}, d.EnvKeys)
	require.Len(t, d.BranchDeployRules, 1)
	assert.Equal(t, []string{"DB"}, d.BranchDeployRules[0].OverrideKeys)
	out := asJSON(t, got)
	assert.NotContains(t, out, "secret-token")
	assert.NotContains(t, out, "postgres://qa")
}

func TestStackCreate(t *testing.T) {
	newService := func() (*Service, *fakeProjects) {
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		projects.repos["proj-1"] = []string{"acme/api"}
		return newTestServiceWith(newFakeRepo(), newFakeStackRepo(), newFakeContainerRepo(), projects, newFakeBus()), projects
	}

	t.Run("explicit fields, env values withheld from the result", func(t *testing.T) {
		s, _ := newService()
		got, err := call(t, s, "stack_create", `{"project_id":"proj-1","machine":"prod","name":"worker","strategy":"run",
			"docker_network":"net1","mounts":["/a:/a"],"command":["tunnel","run"],"env":{"TOKEN":"secret"}}`)
		require.NoError(t, err)
		created := got.(stackCreated)
		assert.Equal(t, StrategyRun, created.Stack.Strategy)
		assert.Equal(t, []string{"tunnel", "run"}, created.Stack.Command)
		assert.Equal(t, []string{"TOKEN"}, created.Stack.EnvKeys)
		assert.Nil(t, created.Deploy)
		assert.NotContains(t, asJSON(t, got), "secret")
	})
	t.Run("a repository_scan candidate passes unchanged and deploys", func(t *testing.T) {
		s, _ := newService()
		ctx := identity.WithActor(t.Context(), identity.Actor{ID: "user-1"})
		args := `{"project_id":"proj-1","machine":"prod","deploy":true,
			"build_source":{"repo_owner":"acme","repo_name":"api","branch":"main","dockerfile":"Dockerfile"},
			"candidate":{"kind":"compose","path":"docker-compose.yml","name":"api",
				"services":[{"name":"web","image":"nginx","ports":[80],"expose":[],"env_keys":["PORT"]}],
				"reachable":{"service":"web","port":80}}}`
		got, err := callAs(t, ctx, s, "stack_create", args)
		require.NoError(t, err)
		created := got.(stackCreated)
		assert.Equal(t, "api", created.Stack.Name, "the name falls back to the repository")
		assert.Equal(t, StrategyCompose, created.Stack.Strategy)
		require.NotNil(t, created.Deploy)
		assert.Equal(t, KindBuild, created.Deploy.Kind)
		assert.Equal(t, "user-1:mcp", created.Deploy.TriggeredBy)
		svcs, err := s.ListServices(t.Context(), created.Stack.ID)
		require.NoError(t, err)
		require.Len(t, svcs, 1)
		assert.Equal(t, Declared{Image: "nginx", Ports: []string{"80"}, EnvKeys: []string{"PORT"}}, svcs[0].Declared)
	})
	t.Run("a dockerfile candidate is a stack of one named after the slug", func(t *testing.T) {
		s, _ := newService()
		got, err := call(t, s, "stack_create", `{"project_id":"proj-1","machine":"prod","name":"Worker","docker_network":"net1",
			"candidate":{"kind":"dockerfile","path":"Dockerfile","services":[{"name":"worker","build":{"context":".","dockerfile":"Dockerfile"}}]}}`)
		require.NoError(t, err)
		created := got.(stackCreated)
		svcs, err := s.ListServices(t.Context(), created.Stack.ID)
		require.NoError(t, err)
		require.Len(t, svcs, 1)
		assert.Equal(t, "worker", svcs[0].Name)
		assert.Equal(t, "Dockerfile", svcs[0].Declared.Build)
	})
	t.Run("link_repository attaches a repository the project lacks", func(t *testing.T) {
		s, projects := newService()
		_, err := call(t, s, "stack_create", `{"project_id":"proj-1","machine":"prod","name":"web","strategy":"compose",
			"link_repository":true,"build_source":{"repo_owner":"acme","repo_name":"web"}}`)
		require.NoError(t, err)
		assert.Contains(t, projects.repos["proj-1"], "acme/web")
	})
	t.Run("a deploy that cannot start still reports the created stack", func(t *testing.T) {
		s, _ := newService()
		_, err := call(t, s, "stack_create", `{"project_id":"proj-1","machine":"prod","name":"web","strategy":"compose","deploy":true}`)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		assert.Contains(t, err.Error(), "already applied: created stack web (id ")
		stacks, err := s.ListStacks(t.Context(), "proj-1")
		require.NoError(t, err)
		assert.Len(t, stacks, 1)
	})
}

func TestStackUpdate_OmittedFieldsKeepTheirValues(t *testing.T) {
	s, stack, _ := stackFixture(t)
	got, err := call(t, s, "stack_update", `{"id":"`+stack.ID+`","name":"api-v2"}`)
	require.NoError(t, err)
	assert.Equal(t, "api-v2", got.(stackResult).Name)

	after, err := s.GetStack(t.Context(), stack.ID)
	require.NoError(t, err)
	assert.Equal(t, "api-v2", after.Name)
	assert.Equal(t, stack.Slug, after.Slug)
	assert.Equal(t, stack.Env, after.Env)
	assert.Equal(t, stack.Mounts, after.Mounts)
	assert.Equal(t, stack.Command, after.Command)
	assert.Equal(t, stack.BuildSource, after.BuildSource)
	assert.Equal(t, stack.BranchDeployRules, after.BranchDeployRules)
}

func TestStackUpdate_Patches(t *testing.T) {
	tests := []struct {
		name  string
		patch string
		check func(t *testing.T, before, after *Stack)
	}{
		{"env set and unset keep the other keys", `"env":{"set":{"LOG":"debug"},"unset":["PORT"]}`, func(t *testing.T, _, after *Stack) {
			assert.Equal(t, map[string]string{"LOG": "debug", "TUNNEL_TOKEN": "secret-token"}, after.Env)
		}},
		{"an empty list clears", `"mounts":[]`, func(t *testing.T, before, after *Stack) {
			assert.Empty(t, after.Mounts)
			assert.Equal(t, before.Command, after.Command)
		}},
		{"build source fields patch one by one", `"build_source":{"branch":"release","dockerfile":""}`, func(t *testing.T, _, after *Stack) {
			assert.Equal(t, &BuildSource{RepoOwner: "acme", RepoName: "api", Branch: "release"}, after.BuildSource)
		}},
		{"a rule without overrides keeps them", `"branch_deploy_rules":[{"pattern":"dev","docker_network":"qa2","name_suffix":"qa"}]`, func(t *testing.T, _, after *Stack) {
			require.Len(t, after.BranchDeployRules, 1)
			assert.Equal(t, "qa2", after.BranchDeployRules[0].DockerNetwork)
			assert.Equal(t, map[string]string{"DB": "postgres://qa"}, after.BranchDeployRules[0].Overrides)
		}},
		{"empty overrides clear them", `"branch_deploy_rules":[{"pattern":"dev","docker_network":"qa","name_suffix":"qa","overrides":{}}]`, func(t *testing.T, _, after *Stack) {
			assert.Empty(t, after.BranchDeployRules[0].Overrides)
		}},
		{"a clone rule turned in-place drops its overrides", `"branch_deploy_rules":[{"pattern":"dev","docker_network":"qa"}]`, func(t *testing.T, _, after *Stack) {
			require.Len(t, after.BranchDeployRules, 1)
			assert.False(t, after.BranchDeployRules[0].DerivesClone())
			assert.Empty(t, after.BranchDeployRules[0].Overrides)
		}},
		{"an empty rule list removes every rule", `"branch_deploy_rules":[]`, func(t *testing.T, _, after *Stack) {
			assert.Empty(t, after.BranchDeployRules)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, stack, _ := stackFixture(t)
			_, err := call(t, s, "stack_update", `{"id":"`+stack.ID+`",`+tt.patch+`}`)
			require.NoError(t, err)
			after, err := s.GetStack(t.Context(), stack.ID)
			require.NoError(t, err)
			tt.check(t, stack, after)
		})
	}
}

func TestStackDelete_ReturnsGone(t *testing.T) {
	s, stack, _ := stackFixture(t)
	got, err := call(t, s, "stack_delete", `{"id":"`+stack.ID+`"}`)
	require.NoError(t, err)
	assert.Equal(t, mcptool.Gone(stack.ID), got)
	_, err = s.GetStack(t.Context(), stack.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestStackDeploy(t *testing.T) {
	tests := []struct {
		name     string
		args     string
		wantKind Kind
		image    string
	}{
		{"a ref builds", `"ref":"main"`, KindBuild, ""},
		{"an image redeploys", `"image":"img:2"`, KindDeploy, "img:2"},
		{"a rollback redeploys the last healthy image", `"rollback":true`, KindDeploy, "img:1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, stack, repo := stackFixture(t)
			seedDeploy(t, repo, &Deploy{ID: "old", StackID: stack.ID, Image: "img:1", Status: StatusHealthy, CreatedAt: at(1)})
			ctx := identity.WithActor(t.Context(), identity.Actor{ID: "user-1"})
			got, err := callAs(t, ctx, s, "stack_deploy", `{"id":"`+stack.ID+`",`+tt.args+`}`)
			require.NoError(t, err)
			d := got.(deployResult)
			assert.Equal(t, tt.wantKind, d.Kind)
			assert.Equal(t, tt.image, d.Image)
			assert.Equal(t, "user-1:mcp", d.TriggeredBy)
		})
	}
	t.Run("a second deploy while one is active is a conflict", func(t *testing.T) {
		s, stack, _ := stackFixture(t)
		_, err := call(t, s, "stack_deploy", `{"id":"`+stack.ID+`","ref":"main"}`)
		require.NoError(t, err)
		_, err = call(t, s, "stack_deploy", `{"id":"`+stack.ID+`","ref":"main"}`)
		require.ErrorIs(t, err, apperrs.ErrConflict)
	})
}

func TestMachineImport(t *testing.T) {
	t.Run("an unknown machine is not found", func(t *testing.T) {
		s, _, _ := newImportTestService(nil)
		_, err := call(t, s, "machine_import", `{"id":"ghost","project_id":"proj-1"}`)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		assert.Contains(t, err.Error(), "machine_list")
	})
	t.Run("adopts a compose project as one stack", func(t *testing.T) {
		s, stacks, _ := newImportTestService([]DiscoveredContainer{{Name: "myapp-web-1", Image: "nginx:latest", Status: "running"}})
		got, err := call(t, s, "machine_import", `{"id":"m-1","project_id":"proj-1","stacks":[{"project":"myapp","containers":["myapp-web-1"]}]}`)
		require.NoError(t, err)
		result := got.(importResult)
		require.Len(t, result.Stacks, 1)
		assert.Equal(t, "myapp", result.Stacks[0].Name)
		assert.False(t, result.Stacks[0].Managed)
		assert.Len(t, stacks.stored, 1)
	})
}

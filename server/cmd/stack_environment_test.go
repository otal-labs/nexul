package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

const (
	uOps = "u-ops"
	// stackEnv is a stack with secrets in its env and in a rule's overrides; stackEnvFeat is its branch deployment.
	stackEnv     = "stack-env"
	stackEnvFeat = "stack-env-feat"
	prodDatabase = "postgres://acme:hunter2@db.example.com/app"
	qaDatabase   = "postgres://acme:qa-pass@qa-db.example.com/app"
)

// withStackEnvironment adds uOps, who reads and writes stacks, and stackEnv with its branch deployment.
func withStackEnvironment(t *testing.T, f permFixture) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	role := &roles.Role{ID: "role-ops", WorkspaceID: wsDefault, Name: "Ops", Permissions: grant("stacks:read", "stacks:write", "deploys:read"), CreatedAt: now, UpdatedAt: now}
	require.NoError(t, f.store.Roles.Create(ctx, role))
	_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: uOps, Provider: auth.ProviderGitHub, ProviderUserID: uOps, Login: uOps})
	require.NoError(t, err)
	require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: uOps, WorkspaceID: wsDefault, RoleID: role.ID, CreatedAt: now}))
	listRolesIn(t, f.db, pGeneral)

	build := &deploy.BuildSource{RepoOwner: "acme", RepoName: "app", Branch: "main", Dockerfile: "Dockerfile"}
	rule := deploy.BranchDeployRule{Pattern: "feat/*", DockerNetwork: "net1", Overrides: map[string]string{"DATABASE_URL": qaDatabase}}
	require.NoError(t, f.store.Stacks.Create(ctx, &deploy.Stack{ID: stackEnv, ProjectID: pGeneral, Name: "production", Slug: "production", Machine: "m1",
		Strategy: deploy.StrategyRun, DockerNetwork: "net1", BuildSource: build, BranchDeployRules: []deploy.BranchDeployRule{rule},
		Env: map[string]string{"DATABASE_URL": prodDatabase, "LOG_LEVEL": "info", "OLD_FLAG": "1"}, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Stacks.Create(ctx, &deploy.Stack{ID: stackEnvFeat, ProjectID: pGeneral, Name: "production-feat-login", Slug: "production-feat-login", Machine: "m1",
		Strategy: deploy.StrategyRun, DockerNetwork: "net1", BuildSource: build, DerivedFrom: stackEnv, Branch: "feat/login",
		Env: map[string]string{"DATABASE_URL": qaDatabase, "LOG_LEVEL": "info", "OLD_FLAG": "1"}, CreatedAt: now, UpdatedAt: now}))
}

func stackRoutes(f permFixture) http.Handler { return deploy.NewHandler(f.svc.deploySvc).Routes() }

// storedEnv is the env the database holds for stackID.
func storedEnv(t *testing.T, f permFixture, stackID string) map[string]string {
	t.Helper()
	stack, err := f.store.Stacks.GetByID(context.Background(), stackID)
	require.NoError(t, err)
	return stack.Env
}

// TestStackEnvironment_AReadersReadsCarryNoSecretValue: a reader's stack, list and branch deployments carry keys only, overrides included.
func TestStackEnvironment_AReadersReadsCarryNoSecretValue(t *testing.T) {
	f := newPermFixture(t)
	withStackEnvironment(t, f)
	routes := stackRoutes(f)

	for _, path := range []string{"/api/stacks/" + stackEnv, "/api/stacks?workspace_id=" + wsDefault} {
		rec := serveAs(routes, uReader, http.MethodGet, path, "")
		require.Equal(t, http.StatusOK, rec.Code, path)
		for _, secret := range []string{prodDatabase, qaDatabase, "hunter2", "qa-pass"} {
			assert.NotContains(t, rec.Body.String(), secret, "%s as a reader", path)
		}
	}

	var got struct {
		Env               map[string]string `json:"env"`
		EnvMasked         bool              `json:"env_masked"`
		BranchDeployments []struct {
			Env map[string]string `json:"env"`
		} `json:"branch_deployments"`
	}
	rec := serveAs(routes, uReader, http.MethodGet, "/api/stacks/"+stackEnv, "")
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, map[string]string{"DATABASE_URL": "", "LOG_LEVEL": "", "OLD_FLAG": ""}, got.Env, "the keys stay")
	assert.True(t, got.EnvMasked)
	require.Len(t, got.BranchDeployments, 1)
	assert.Equal(t, map[string]string{"DATABASE_URL": "", "LOG_LEVEL": "", "OLD_FLAG": ""}, got.BranchDeployments[0].Env)
}

// TestStackEnvironment_AWriterRevealsAndEditsTheValues: a writer is sent the values and can send the body back with one changed.
func TestStackEnvironment_AWriterRevealsAndEditsTheValues(t *testing.T) {
	f := newPermFixture(t)
	withStackEnvironment(t, f)
	routes := stackRoutes(f)

	rec := serveAs(routes, uOps, http.MethodGet, "/api/stacks/"+stackEnv, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var stack map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &stack))
	assert.Equal(t, map[string]any{"DATABASE_URL": prodDatabase, "LOG_LEVEL": "info", "OLD_FLAG": "1"}, stack["env"])
	assert.NotContains(t, stack, "env_masked")

	stack["env"].(map[string]any)["LOG_LEVEL"] = "debug"
	body, err := json.Marshal(stack)
	require.NoError(t, err)
	rec = serveAs(routes, uOps, http.MethodPatch, "/api/stacks/"+stackEnv, string(body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, map[string]string{"DATABASE_URL": prodDatabase, "LOG_LEVEL": "debug", "OLD_FLAG": "1"}, storedEnv(t, f, stackEnv))
	rules, err := f.store.Stacks.GetByID(context.Background(), stackEnv)
	require.NoError(t, err)
	assert.Equal(t, qaDatabase, rules.BranchDeployRules[0].Overrides["DATABASE_URL"], "the rule's overrides survive the round trip")
}

// TestStackEnvironment_RemovingAKeyOrBlankingAValueSticks: an edit replaces the env as sent, so a key left out is gone.
func TestStackEnvironment_RemovingAKeyOrBlankingAValueSticks(t *testing.T) {
	f := newPermFixture(t)
	withStackEnvironment(t, f)
	routes := stackRoutes(f)

	var stack map[string]any
	rec := serveAs(routes, uOps, http.MethodGet, "/api/stacks/"+stackEnv, "")
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &stack))
	stack["env"] = map[string]any{"DATABASE_URL": "", "LOG_LEVEL": "info"}
	body, err := json.Marshal(stack)
	require.NoError(t, err)
	rec = serveAs(routes, uOps, http.MethodPatch, "/api/stacks/"+stackEnv, string(body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	assert.Equal(t, map[string]string{"DATABASE_URL": "", "LOG_LEVEL": "info"}, storedEnv(t, f, stackEnv))
}

// TestStackEnvironment_AStackRuleTurnsWhoSeesTheValues: a rule on one stack decides who is sent its values, in the read and the list.
func TestStackEnvironment_AStackRuleTurnsWhoSeesTheValues(t *testing.T) {
	f := newPermFixture(t)
	withStackEnvironment(t, f)
	routes := stackRoutes(f)
	write := []permissions.Action{permissions.StacksWrite}
	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "stack", []string{stackEnv}, access.Targets{RoleIDs: []string{"role-ops"}}, write, access.StateDeny))
	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "stack", []string{stackEnv}, access.Targets{UserIDs: []string{uReader}}, write, access.StateAllow))

	for user, sees := range map[string]bool{uOps: false, uReader: true, uOwner: true} {
		for _, path := range []string{"/api/stacks/" + stackEnv, "/api/stacks?workspace_id=" + wsDefault} {
			rec := serveAs(routes, user, http.MethodGet, path, "")
			require.Equal(t, http.StatusOK, rec.Code, "%s %s", user, path)
			assert.Equal(t, sees, strings.Contains(rec.Body.String(), prodDatabase), "%s: %s", user, path)
		}
	}
}

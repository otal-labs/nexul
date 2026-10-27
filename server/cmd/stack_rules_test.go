package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

const (
	uDeveloper = "u-developer"
	uBoth      = "u-both"
	// stackProd builds acme/app; stackFeat is its branch deployment of feat/login.
	stackProd = "stack-prod"
	stackFeat = "stack-prod-feat"
)

// withDeployRoles adds Developer and Frontend, both listed in the fixture's project, uDeveloper holding Developer and uBoth
// both, and stackProd with its branch deployment stackFeat.
func withDeployRoles(t *testing.T, f permFixture) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	for _, r := range []*roles.Role{
		{ID: "role-dev", WorkspaceID: wsDefault, Name: "Developer", Permissions: grant("stacks:read", "deploys:read", "deploys:write")},
		{ID: "role-front", WorkspaceID: wsDefault, Name: "Frontend", Permissions: grant("stacks:read", "deploys:read")},
	} {
		r.CreatedAt, r.UpdatedAt = now, now
		require.NoError(t, f.store.Roles.Create(ctx, r))
	}
	for _, user := range []string{uDeveloper, uBoth} {
		_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: user, Provider: auth.ProviderGitHub, ProviderUserID: user, Login: user})
		require.NoError(t, err)
		require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: wsDefault, RoleID: "role-dev", CreatedAt: now}))
	}
	require.NoError(t, f.store.WorkspaceMembers.ChangeRoles(ctx, wsDefault, uBoth, []string{"role-front"}, nil))
	listRolesIn(t, f.db, pGeneral)
	withMachineM1(t, f)
	for _, role := range []string{"role-dev", "role-front"} {
		require.NoError(t, f.store.Access.SetRole(ctx, "machine", machineM1, role, permissions.SetOf(permissions.MachinesSee, permissions.MachinesDeploy), permissions.Set{}))
	}
	build := &deploy.BuildSource{RepoOwner: "acme", RepoName: "app", Branch: "main", Dockerfile: "Dockerfile"}
	require.NoError(t, f.store.Stacks.Create(ctx, &deploy.Stack{ID: stackProd, ProjectID: pGeneral, Name: "production", Slug: "production", Machine: "m1",
		Strategy: deploy.StrategyRun, BuildSource: build, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Stacks.Create(ctx, &deploy.Stack{ID: stackFeat, ProjectID: pGeneral, Name: "production-feat-login", Slug: "production-feat-login",
		Machine: "m1", Strategy: deploy.StrategyRun, BuildSource: build, DerivedFrom: stackProd, Branch: "feat/login", CreatedAt: now, UpdatedAt: now}))
}

// deployOverHTTP starts a deploy through the gateway as user and returns the status and body.
func deployOverHTTP(t *testing.T, f permFixture, user, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/deploys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(as(user))
	rec := httptest.NewRecorder()
	deploy.NewHandler(f.svc.deploySvc).Routes().ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// TestStackRules_OnlyTheRoleDeploys: role Developer Deny deploys:write and role Frontend Allow on the production stack;
// a Developer without Frontend is refused with the rule that refused it, over HTTP, MCP and a branch's deploy, while a
// Developer who is also Frontend deploys, Allow winning between their roles.
func TestStackRules_OnlyTheRoleDeploys(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	withDeployRoles(t, f)
	write := []permissions.Action{permissions.DeploysWrite}
	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "stack", []string{stackProd}, access.Targets{RoleIDs: []string{"role-dev"}}, write, access.StateDeny))
	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "stack", []string{stackProd}, access.Targets{RoleIDs: []string{"role-front"}}, write, access.StateAllow))
	const source = "deploys:write required (Denied on stack production for role Developer)"

	status, body := deployOverHTTP(t, f, uDeveloper, `{"stack_id":"`+stackProd+`","ref":"main"}`)
	assert.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, body, source, "over HTTP")

	stackDeploy := tool(t, deploy.MCPTools(f.svc.deploySvc), "stack_deploy")
	_, err := stackDeploy.Call(as(uDeveloper), json.RawMessage(`{"id":"`+stackProd+`","ref":"main"}`))
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), source, "over MCP")

	_, err = f.svc.deploySvc.Deploy(as(uDeveloper), deploy.DeployRequest{StackID: stackFeat, Ref: "feat/login"})
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), source, "a branch deployment answers to its base stack's rules")

	held, src, err := f.svc.accessSvc.Decide(as(uOwner), uBoth, "stack", stackProd, permissions.DeploysWrite)
	require.NoError(t, err)
	assert.True(t, held)
	assert.Equal(t, access.Source{Kind: access.SourceRule, EntityType: "stack", EntityID: stackProd, TargetType: access.TargetRole, TargetID: "role-front", State: access.StateAllow}, src)
	status, body = deployOverHTTP(t, f, uBoth, `{"stack_id":"`+stackFeat+`","ref":"feat/login"}`)
	assert.Equal(t, http.StatusCreated, status, body)

	status, _ = deployOverHTTP(t, f, uPlain, `{"stack_id":"`+stackProd+`","ref":"main"}`)
	assert.Equal(t, http.StatusNotFound, status, "a stack they can't read is not found, never forbidden")
	_, err = f.svc.accessSvc.ListRules(as(uDeveloper), uDeveloper, "stack", stackProd)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "a stack's rules take stacks:write on it")
	_, err = f.svc.accessSvc.ListRules(as(uOwner), uOwner, "stack", stackFeat)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a branch deployment's rules are its base stack's")
	_, err = f.svc.accessSvc.ListRules(as(uOwner), uOwner, "stack", f.infra)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "an instance stack carries no rules")
}

// TestStackRules_ADeniedStackLeavesTheListAndFrames: a role's Deny of stacks:read hides the stack from its holders on
// every path, its deploys and branch deployment included, though the role reads deploys, while the Owner keeps it.
func TestStackRules_ADeniedStackLeavesTheListAndFrames(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	withDeployRoles(t, f)
	ctx := context.Background()
	now := time.Now()
	for _, stackID := range []string{stackProd, stackFeat} {
		require.NoError(t, f.store.Deploys.Create(ctx, &deploy.Deploy{ID: "deploy-" + stackID, StackID: stackID, Service: stackID, Status: deploy.StatusHealthy, CreatedAt: now, UpdatedAt: now}))
	}
	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "stack", []string{stackProd}, access.Targets{RoleIDs: []string{"role-reader"}},
		[]permissions.Action{permissions.StacksRead}, access.StateDeny))
	stackList := tool(t, deploy.MCPTools(f.svc.deploySvc), "stack_list")
	deployList := tool(t, deploy.MCPTools(f.svc.deploySvc), "deploy_list")
	a := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	prod, err := f.store.Stacks.GetByID(ctx, stackProd)
	require.NoError(t, err)
	feat, err := f.store.Stacks.GetByID(ctx, stackFeat)
	require.NoError(t, err)

	for user, sees := range map[string]bool{uReader: false, uOwner: true, uDeveloper: true} {
		stacks, _ := callPage(t, f.svc.accessSvc, stackList, user, `{}`)
		assert.Equal(t, sees, slices.Contains(stacks, stackProd), "%s: stack_list", user)
		assert.True(t, slices.Contains(stacks, f.stack), "%s: the project's other stack stays", user)
		canvas, err := f.svc.deploySvc.ListScopedStacks(as(user), "", wsDefault)
		require.NoError(t, err)
		assert.Equal(t, sees, slices.ContainsFunc(canvas, func(st *deploy.Stack) bool { return st.ID == stackProd }), "%s: the topology canvas's stacks", user)
		deploys, _ := callPage(t, f.svc.accessSvc, deployList, user, `{}`)
		assert.Equal(t, sees, slices.Contains(deploys, "deploy-"+stackProd), "%s: deploy_list", user)
		assert.Equal(t, sees, slices.Contains(deploys, "deploy-"+stackFeat), "%s: deploy_list, the branch deployment's", user)
		_, err = f.svc.deploySvc.GetStack(as(user), stackFeat)
		assert.Equal(t, sees, err == nil, "%s: the branch deployment's page", user)
		if !sees {
			require.ErrorIs(t, err, apperrs.ErrNotFound)
		}
		_, err = f.svc.deploySvc.Get(as(user), "deploy-"+stackProd)
		assert.Equal(t, sees, err == nil, "%s: deploy_list with an id", user)
		assert.Equal(t, sees, a.allows(as(user), deploy.TopicStackUpdated, deploy.StackEvent{Stack: *prod}), "%s: the stack's frame", user)
		assert.Equal(t, sees, a.allows(as(user), deploy.TopicStackUpdated, deploy.StackEvent{Stack: *feat}), "%s: its branch deployment's frame", user)
		assert.Equal(t, sees, a.allows(as(user), deploy.TopicDeployUpdated, deploy.DeployUpdatedEvent{ID: "deploy-" + stackProd, StackID: stackProd}), "%s: a deploy's frame", user)
	}

	require.NoError(t, f.svc.deploySvc.DeleteStack(as(uOwner), stackProd))
	rules, err := f.store.Access.ListByResource(ctx, "stack", stackProd)
	require.NoError(t, err)
	assert.Empty(t, rules, "a deleted stack's rules go with it")
}

// TestStackRules_FramesReachWhomTheyName: a stack rule's frame reaches whom it names and who manages the stack.
func TestStackRules_FramesReachWhomTheyName(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	withDeployRoles(t, f)
	a := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	person := access.GrantEvent{ResourceType: "stack", ResourceID: stackProd, Target: access.TargetPerson, UserID: uPlain, ActorID: uOwner}
	assert.True(t, a.allows(as(uPlain), access.TopicGrantChanged, person), "the person themselves")
	assert.True(t, a.allows(as(uOwner), access.TopicGrantChanged, person), "who manages the stack's rules")
	assert.False(t, a.allows(as(uReader), access.TopicGrantChanged, person))
	role := access.GrantEvent{ResourceType: "stack", ResourceID: stackProd, Target: access.TargetRole, RoleID: "role-dev", ActorID: uOwner}
	assert.True(t, a.allows(as(uDeveloper), access.TopicGrantChanged, role), "a holder of the role")
	assert.False(t, a.allows(as(uReader), access.TopicGrantChanged, role))
}

// TestStackRules_ExplainAndHoldWhatYouGive: explain covers a stack's actions, and nobody allows on a stack what they
// don't hold there.
func TestStackRules_ExplainAndHoldWhatYouGive(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	withDeployRoles(t, f)
	a := f.svc.accessSvc
	require.NoError(t, a.SetRules(as(uOwner), uOwner, "stack", []string{stackProd}, access.Targets{UserIDs: []string{uDeveloper}}, []permissions.Action{permissions.StacksWrite}, access.StateAllow))

	e, err := a.Explain(as(uDeveloper), uDeveloper, "stack", stackProd, "")
	require.NoError(t, err)
	var actions []permissions.Action
	for _, answer := range e.Effective {
		actions = append(actions, answer.Action)
		if answer.Action == permissions.StacksWrite {
			assert.True(t, answer.Allowed)
			assert.Equal(t, "Allowed on stack production for u-developer", answer.Source.Sentence)
			assert.False(t, answer.Fallback.Allowed, "at Fallback the roles answer")
		}
	}
	assert.Equal(t, []permissions.Action{"stacks:read", "stacks:write", "stacks:delete", "stacks:logs", "deploys:read", "deploys:write"}, actions)

	err = a.SetRules(as(uDeveloper), uDeveloper, "stack", []string{stackProd}, access.Targets{RoleIDs: []string{"role-front"}}, []permissions.Action{permissions.StacksDelete}, access.StateAllow)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "stacks:delete is not theirs to give")
	require.NoError(t, a.SetRules(as(uDeveloper), uDeveloper, "stack", []string{stackProd}, access.Targets{RoleIDs: []string{"role-front"}}, []permissions.Action{permissions.DeploysWrite}, access.StateAllow))
	err = a.SetRules(as(uOwner), uOwner, "stack", []string{stackProd}, access.Targets{UserIDs: []string{uDeveloper}}, []permissions.Action{permissions.DocsRead}, access.StateAllow)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a stack's rules set only stacks and deploys")
	_, err = a.Explain(as(uOwner), uOwner, "stack", "stack-gone", "")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

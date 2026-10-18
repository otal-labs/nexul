package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// machineM1 is the id of the machine named m1, where the fixture's stacks run.
const machineM1 = "machine-m1"

// withMachineM1 adds the machine the fixture's stacks run on, seen by the given roles and by no one else but the
// instance Owner.
func withMachineM1(t *testing.T, f permFixture, seenBy ...string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	require.NoError(t, f.store.Machines.Create(ctx, &runner.Machine{ID: machineM1, Name: "m1", FirstSeen: now, LastSeen: now}))
	for _, role := range seenBy {
		require.NoError(t, f.store.Access.SetRole(ctx, "machine", machineM1, role, permissions.SetOf(permissions.MachinesSee), permissions.Set{}))
	}
}

const (
	uShipper  = "u-shipper"
	uHosting  = "u-hosting"
	uNoDNS    = "u-nodns"
	machineM2 = "machine-m2"
	machineM3 = "machine-m3"
)

// machinesFixture is the permission fixture with machines m1, m2 and m3 and three members of the default workspace:
// u-shipper edits stacks, deploys and DNS and holds Deploy to on m1 and only See on m2; u-hosting is the same but reads
// DNS; u-nodns reads stacks and sees m1 with no DNS access. Nobody but the instance Owner sees m3.
func machinesFixture(t *testing.T) permFixture {
	t.Helper()
	f := newPermFixture(t)
	ctx := context.Background()
	now := time.Now()
	for _, r := range []struct {
		id, name string
		bits     []string
	}{
		{"role-shipper", "Shipper", []string{"stacks:read", "stacks:write", "deploys:read", "deploys:write", "dns:read", "dns:write"}},
		{"role-hosting", "Hosting", []string{"stacks:read", "stacks:write", "deploys:read", "deploys:write", "dns:read"}},
		{"role-nodns", "No DNS", []string{"stacks:read", "deploys:read"}},
	} {
		require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: r.id, WorkspaceID: wsDefault, Name: r.name, Permissions: grant(r.bits...), CreatedAt: now, UpdatedAt: now}))
	}
	for user, role := range map[string]string{uShipper: "role-shipper", uHosting: "role-hosting", uNoDNS: "role-nodns"} {
		_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: user, Provider: auth.ProviderGitHub, ProviderUserID: user, Login: user})
		require.NoError(t, err)
		require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: wsDefault, RoleID: role, CreatedAt: now}))
	}
	listRolesIn(t, f.db, pGeneral)
	for id, name := range map[string]string{machineM1: "m1", machineM2: "m2", machineM3: "m3"} {
		require.NoError(t, f.store.Machines.Create(ctx, &runner.Machine{ID: id, Name: name, FirstSeen: now, LastSeen: now}))
	}
	rule := func(machine, role string, actions ...permissions.Action) {
		require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "machine", []string{machine}, access.Targets{RoleIDs: []string{role}}, actions, access.StateAllow))
	}
	see, deployTo := permissions.MachinesSee, permissions.MachinesDeploy
	rule(machineM1, "role-shipper", see, deployTo)
	rule(machineM2, "role-shipper", see)
	rule(machineM1, "role-hosting", see, deployTo)
	rule(machineM1, "role-nodns", see)
	return f
}

// stackOn is a run stack in the default project on the named machine, as stack_create takes it.
func stackOn(machine string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"project_id":%q,"machine":%q,"name":"api-%s","strategy":"run","docker_network":"edge"}`, pGeneral, machine, machine))
}

// stacksNamed counts the stacks called name, to show a refused create left nothing behind.
func stacksNamed(t *testing.T, f permFixture, name string) int {
	t.Helper()
	var n int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM stacks WHERE name = ?`, name).Scan(&n))
	return n
}

// TestMachines_StackCreateRefusesAMachineYouCantDeployTo: a person who edits stacks in a project still can't aim one at a
// machine they hold no Deploy to on. Typing the machine's name into stack_create refuses one they see, hides one they
// don't, and rejects a name no machine bears, and the gateway's POST does the same, leaving no stack behind.
func TestMachines_StackCreateRefusesAMachineYouCantDeployTo(t *testing.T) {
	t.Parallel()
	f := machinesFixture(t)
	create := tool(t, deploy.MCPTools(f.svc.deploySvc), "stack_create")

	_, err := create.Call(as(uShipper), stackOn("m1"))
	require.NoError(t, err, "Deploy to on m1")
	assert.Equal(t, 1, stacksNamed(t, f, "api-m1"))

	_, err = create.Call(as(uShipper), stackOn("m2"))
	require.ErrorIs(t, err, apperrs.ErrForbidden, "m2 is seen but not deployed to")
	assert.Contains(t, err.Error(), "machines:deploy")
	_, err = create.Call(as(uShipper), stackOn("m3"))
	require.ErrorIs(t, err, apperrs.ErrInvalid, "m3 is not seen, so it answers as a name no machine bears")
	_, err = create.Call(as(uShipper), stackOn("m9"))
	require.ErrorIs(t, err, apperrs.ErrInvalid, "no machine bears the name")
	assert.Zero(t, stacksNamed(t, f, "api-m2")+stacksNamed(t, f, "api-m3")+stacksNamed(t, f, "api-m9"))

	_, err = create.Call(as(uOwner), stackOn("m3"))
	require.NoError(t, err, "the instance Owner deploys to every machine")

	post := func(machine string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/stacks", strings.NewReader(string(stackOn(machine))))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		deploy.NewHandler(f.svc.deploySvc).Routes().ServeHTTP(rec, req.WithContext(as(uShipper)))
		return rec.Code
	}
	assert.Equal(t, http.StatusForbidden, post("m2"))
	assert.Equal(t, http.StatusBadRequest, post("m3"))
}

// TestMachines_AHiddenMachineTypedByNameAnswersAsAnUnknownName: a member who edits stacks can't find out which machine
// names exist by typing them. A machine they don't see and a name no machine bears answer with the same status and the
// same body, naming only what they typed, over HTTP and over stack_create.
func TestMachines_AHiddenMachineTypedByNameAnswersAsAnUnknownName(t *testing.T) {
	t.Parallel()
	f := machinesFixture(t)
	create := tool(t, deploy.MCPTools(f.svc.deploySvc), "stack_create")

	// asked is what a call answered with the typed name replaced, so only the shape is compared.
	httpAsked := func(machine string) string {
		req := httptest.NewRequest(http.MethodPost, "/api/stacks", strings.NewReader(string(stackOn(machine))))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		deploy.NewHandler(f.svc.deploySvc).Routes().ServeHTTP(rec, req.WithContext(as(uShipper)))
		return fmt.Sprintf("%d %s", rec.Code, strings.ReplaceAll(rec.Body.String(), machine, "<typed>"))
	}
	mcpAsked := func(machine string) string {
		_, err := create.Call(as(uShipper), stackOn(machine))
		require.Error(t, err)
		return strings.ReplaceAll(err.Error(), machine, "<typed>")
	}

	assert.Equal(t, httpAsked("m9"), httpAsked("m3"), "POST /api/stacks")
	assert.Equal(t, mcpAsked("m9"), mcpAsked("m3"), "stack_create")
}

// TestMachines_MovingAStackNeedsDeployToOnTheNewMachine: an update that changes a stack's machine takes Deploy to on the
// machine it moves to, while an edit that leaves the machine alone does not.
func TestMachines_MovingAStackNeedsDeployToOnTheNewMachine(t *testing.T) {
	t.Parallel()
	f := machinesFixture(t)
	update := tool(t, deploy.MCPTools(f.svc.deploySvc), "stack_update")
	_, err := tool(t, deploy.MCPTools(f.svc.deploySvc), "stack_create").Call(as(uShipper), stackOn("m1"))
	require.NoError(t, err)
	var id string
	require.NoError(t, f.db.QueryRow(`SELECT id FROM stacks WHERE name = 'api-m1'`).Scan(&id))

	_, err = update.Call(as(uShipper), json.RawMessage(fmt.Sprintf(`{"id":%q,"machine":"m2"}`, id)))
	require.ErrorIs(t, err, apperrs.ErrForbidden, "m2 is seen but not deployed to")
	_, err = update.Call(as(uShipper), json.RawMessage(fmt.Sprintf(`{"id":%q,"machine":"m3"}`, id)))
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = update.Call(as(uShipper), json.RawMessage(fmt.Sprintf(`{"id":%q,"docker_network":"core"}`, id)))
	require.NoError(t, err)

	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "machine", []string{machineM1}, access.Targets{RoleIDs: []string{"role-shipper"}}, []permissions.Action{permissions.MachinesDeploy}, access.StateDeny))
	_, err = update.Call(as(uShipper), json.RawMessage(fmt.Sprintf(`{"id":%q,"docker_network":"edge"}`, id)))
	require.NoError(t, err, "an edit that keeps the machine is the stack's own rules' business, not the machine's")
}

// deploysOf counts the deploys recorded for a stack, to show a refused deploy left no job behind.
func deploysOf(t *testing.T, f permFixture, stackID string) int {
	t.Helper()
	var n int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM deploys WHERE stack_id = ?`, stackID).Scan(&n))
	return n
}

// TestMachines_TakenOffStopsTheNextDeploy: a stack made on a machine is redeployed by its maker, over HTTP and MCP, until
// the Owner denies Deploy to or takes them off the machine, and the very next deploy is refused, no job recorded.
func TestMachines_TakenOffStopsTheNextDeploy(t *testing.T) {
	t.Parallel()
	f := machinesFixture(t)
	stack, _, err := f.svc.deploySvc.CreateStackWithOptions(as(uShipper), deploy.Stack{ProjectID: pGeneral, Name: "api", Machine: "m1", Strategy: deploy.StrategyRun, DockerNetwork: "edge"}, nil, deploy.CreateStackOptions{})
	require.NoError(t, err)
	stackDeploy := tool(t, deploy.MCPTools(f.svc.deploySvc), "stack_deploy")
	body := fmt.Sprintf(`{"id":%q,"image":"api:1"}`, stack.ID)

	status, _ := deployOverHTTP(t, f, uShipper, fmt.Sprintf(`{"stack_id":%q,"image":"api:1"}`, stack.ID))
	require.Equal(t, http.StatusCreated, status, "on the machine")
	_, err = f.db.Exec(`UPDATE deploys SET status = 'healthy' WHERE stack_id = ?`, stack.ID)
	require.NoError(t, err)

	machine := []string{machineM1}
	roleShipper := access.Targets{RoleIDs: []string{"role-shipper"}}
	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "machine", machine, roleShipper, []permissions.Action{permissions.MachinesDeploy}, access.StateDeny))
	before := deploysOf(t, f, stack.ID)
	_, err = stackDeploy.Call(as(uShipper), json.RawMessage(body))
	require.ErrorIs(t, err, apperrs.ErrForbidden, "denied Deploy to")
	assert.Contains(t, err.Error(), "machines:deploy")

	both := []permissions.Action{permissions.MachinesSee, permissions.MachinesDeploy}
	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "machine", machine, roleShipper, both, access.StateFallback))
	status, _ = deployOverHTTP(t, f, uShipper, fmt.Sprintf(`{"stack_id":%q,"image":"api:1"}`, stack.ID))
	assert.Equal(t, http.StatusNotFound, status, "taken off the machine, it is gone")
	_, err = f.svc.deploySvc.Rollback(as(uShipper), stack.ID, uShipper)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Equal(t, before, deploysOf(t, f, stack.ID), "no deploy was recorded after the machine was taken away")

	_, err = stackDeploy.Call(as(uOwner), json.RawMessage(body))
	require.NoError(t, err, "the instance Owner still deploys there")
}

// TestMachines_APushDeployStillRuns: a push to a branch with a deploy rule creates the branch deployment and deploys it on
// a machine no member sees, and so does a plain deploy with no one signed in: the server's own deploys pass as the server.
func TestMachines_APushDeployStillRuns(t *testing.T) {
	t.Parallel()
	f := machinesFixture(t)
	ctx := context.Background()
	now := time.Now()
	build := &deploy.BuildSource{RepoOwner: "acme", RepoName: "app", Branch: "main", Dockerfile: "Dockerfile"}
	rules := []deploy.BranchDeployRule{{Pattern: "feat/*", DockerNetwork: "edge"}}
	require.NoError(t, f.store.Stacks.Create(ctx, &deploy.Stack{ID: "stack-hidden", ProjectID: pGeneral, Name: "hidden", Slug: "hidden", Machine: "m3", Strategy: deploy.StrategyRun,
		BuildSource: build, BranchDeployRules: rules, CreatedAt: now, UpdatedAt: now}))
	payload, err := json.Marshal(deploy.PushTrigger{Owner: "acme", Repo: "app", Branch: "feat/login", SHA: "abc123"})
	require.NoError(t, err)

	require.NoError(t, f.svc.deploySvc.HandlePush(ctx, eventbus.Event{Payload: payload}))

	var derived string
	require.NoError(t, f.db.QueryRow(`SELECT id FROM stacks WHERE derived_from = 'stack-hidden'`).Scan(&derived), "the branch deployment was created on m3")
	assert.Equal(t, 1, deploysOf(t, f, derived), "and deployed")
	_, err = f.svc.deploySvc.Deploy(ctx, deploy.DeployRequest{StackID: "stack-hidden", Ref: "main"})
	require.NoError(t, err, "a merge's redeploy has no caller")
	_, err = f.svc.deploySvc.Deploy(as(uShipper), deploy.DeployRequest{StackID: "stack-hidden", Ref: "main"})
	require.ErrorIs(t, err, apperrs.ErrNotFound, "while a member who doesn't see m3 can't")
}

// TestMachines_AGatewayTakesDeployToAndDNSWrite: creating a gateway, which deploys the proxy's own stack on a machine,
// takes dns:write and Deploy to on that machine; either alone is refused, and a machine not seen is not found, on the
// gateway and on the proxy step the DNS page runs.
func TestMachines_AGatewayTakesDeployToAndDNSWrite(t *testing.T) {
	t.Parallel()
	f := machinesFixture(t)
	gateway := func(network, machine string) dns.CreateGatewayInput {
		return dns.CreateGatewayInput{Kind: dns.GatewayProxy, DockerNetwork: network, ZoneID: "zone-1", Zone: "example.com", ServerAddress: "203.0.113.9", Target: machine}
	}

	_, err := f.svc.dnsSvc.CreateGateway(as(uHosting), gateway("edge-1", "m1"))
	require.ErrorIs(t, err, apperrs.ErrForbidden, "Deploy to on m1 without dns:write")
	_, err = f.svc.dnsSvc.CreateGateway(as(uShipper), gateway("edge-2", "m2"))
	require.ErrorIs(t, err, apperrs.ErrForbidden, "dns:write with only See on m2")
	assert.Contains(t, err.Error(), "machines:deploy")
	_, err = f.svc.dnsSvc.CreateGateway(as(uShipper), gateway("edge-3", "m3"))
	require.ErrorIs(t, err, apperrs.ErrInvalid, "m3 is not seen, so it answers as a name no machine bears")
	_, err = f.svc.dnsSvc.ProvisionReverseProxy(as(uShipper), dns.AgentSpec{Target: "m2"})
	require.ErrorIs(t, err, apperrs.ErrForbidden, "the DNS page's proxy step")
	var stacks int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM stacks WHERE project_id IS NULL AND machine IN ('m1', 'm2', 'm3') AND id != ?`, f.infra).Scan(&stacks))
	assert.Zero(t, stacks, "no refused attempt deployed anything")

	g, err := f.svc.dnsSvc.CreateGateway(as(uShipper), gateway("edge-4", "m1"))
	require.NoError(t, err)
	assert.Equal(t, "m1", g.Machine)
	stack, err := f.svc.deploySvc.GetStack(as(uShipper), g.ServiceID)
	require.NoError(t, err)
	assert.Empty(t, stack.ProjectID, "an instance stack, belonging to no project")
}

// TestMachines_AnInstanceStackIsNotFoundWithoutSeeAndDNSRead: a stack outside every project is found only by someone who
// sees its machine and reads DNS, on a read, in both lists and on its live frame, and a deploy of it needs Deploy to
// and dns:write beside the stack's own bits.
func TestMachines_AnInstanceStackIsNotFoundWithoutSeeAndDNSRead(t *testing.T) {
	t.Parallel()
	f := machinesFixture(t)
	ctx := context.Background()
	now := time.Now()
	require.NoError(t, f.store.Stacks.Create(ctx, &deploy.Stack{ID: "stack-proxy-m3", Name: "proxy-m3", Slug: "proxy-m3", Machine: "m3", Strategy: deploy.StrategyRun, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Deploys.Create(ctx, &deploy.Deploy{ID: "deploy-proxy-m3", StackID: "stack-proxy-m3", Service: "proxy-m3", Status: deploy.StatusHealthy, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Deploys.Create(ctx, &deploy.Deploy{ID: "deploy-cloudflared", StackID: f.infra, Service: "cloudflared", Status: deploy.StatusHealthy, CreatedAt: now, UpdatedAt: now}))
	stackList := tool(t, deploy.MCPTools(f.svc.deploySvc), "stack_list")
	deployList := tool(t, deploy.MCPTools(f.svc.deploySvc), "deploy_list")
	audience := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	frame := func(user, stackID, machine string) bool {
		return audience.allows(as(user), deploy.TopicStackUpdated, deploy.StackEvent{Stack: deploy.Stack{ID: stackID, Machine: machine}})
	}
	// f.infra, the gateway stack on m1.
	type view struct{ get, listed, listedDeploy, live bool }
	seen := func(user string) view {
		_, err := f.svc.deploySvc.GetStack(as(user), f.infra)
		ids, _ := callPage(t, f.svc.accessSvc, stackList, user, `{}`)
		deploys, _ := callPage(t, f.svc.accessSvc, deployList, user, `{}`)
		return view{err == nil, slices.Contains(ids, f.infra), slices.Contains(deploys, "deploy-cloudflared"), frame(user, f.infra, "m1")}
	}
	everything, nothing := view{true, true, true, true}, view{}
	assert.Equal(t, everything, seen(uShipper), "sees m1 and reads DNS")
	assert.Equal(t, nothing, seen(uNoDNS), "sees m1 and reads no DNS")
	assert.Equal(t, everything, seen(uOwner))

	_, err := f.svc.deploySvc.GetStack(as(uNoDNS), f.infra)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = f.svc.deploySvc.GetStack(as(uShipper), "stack-proxy-m3")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "reads DNS but doesn't see m3")
	listedStacks, _ := callPage(t, f.svc.accessSvc, stackList, uShipper, `{}`)
	assert.NotContains(t, listedStacks, "stack-proxy-m3")
	listedDeploys, _ := callPage(t, f.svc.accessSvc, deployList, uShipper, `{}`)
	assert.NotContains(t, listedDeploys, "deploy-proxy-m3")
	assert.False(t, frame(uShipper, "stack-proxy-m3", "m3"))
	assert.True(t, frame(uOwner, "stack-proxy-m3", "m3"))

	_, err = f.svc.deploySvc.Deploy(as(uHosting), deploy.DeployRequest{StackID: f.infra, Image: "proxy:1"})
	require.ErrorIs(t, err, apperrs.ErrForbidden, "sees m1 and holds Deploy to there but writes no DNS")
	_, err = f.svc.deploySvc.Deploy(as(uShipper), deploy.DeployRequest{StackID: f.infra, Image: "proxy:1"})
	require.NoError(t, err)
}

// TestMachines_ThePickerListsWhereYouMayDeploy: the machines offered for a stack are those the caller holds Deploy to on,
// not those merely seen, and the instance Owner is offered all of them.
func TestMachines_ThePickerListsWhereYouMayDeploy(t *testing.T) {
	t.Parallel()
	f := machinesFixture(t)
	f.svc.deploySvc.SetMachineLookup(deployMachineLookupAdapter{machines: f.store.Machines})
	offered := func(user string) []string {
		req := httptest.NewRequest(http.MethodGet, "/api/machines/deployable", nil)
		rec := httptest.NewRecorder()
		deploy.NewHandler(f.svc.deploySvc).Routes().ServeHTTP(rec, req.WithContext(as(user)))
		require.Equal(t, http.StatusOK, rec.Code)
		var machines []deploy.MachineRef
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &machines))
		names := []string{}
		for _, m := range machines {
			names = append(names, m.Name)
		}
		slices.Sort(names)
		return names
	}

	assert.Equal(t, []string{"m1"}, offered(uShipper), "m2 is seen but not deployed to, m3 is not seen")
	assert.Equal(t, []string{"m1", "m2", "m3"}, offered(uOwner))
	assert.Equal(t, []string{}, offered(uPlain))
}

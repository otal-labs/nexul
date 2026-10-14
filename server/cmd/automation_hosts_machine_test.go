package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// hostsCast is edge-box and core-box with the instance URL set, and five people who all edit automations in the
// default workspace and differ only in what the edge machine gives them: the instance Owner (bob), a Manage holder
// (dana), a Deploy to holder (gail), someone who only sees it (erin), and someone with no rule on it (frank).
type hostsCast struct {
	privacyCast
	owner, manager, deployer, seer, nobody string
}

func newHostsCast(t *testing.T) hostsCast {
	t.Helper()
	c := enrollmentCast(t)
	h := hostsCast{privacyCast: c, owner: c.owner}
	add := func(user string, bits ...string) string {
		role := &roles.Role{ID: "role-" + user, WorkspaceID: "workspace-default", Name: user, Permissions: grant(append(bits, "automations:read", "automations:write")...), CreatedAt: time.Now(), UpdatedAt: time.Now()}
		require.NoError(t, c.store.Roles.Create(t.Context(), role))
		_, _, err := c.store.Users.UpsertUser(t.Context(), &auth.Identity{UserID: user, Provider: auth.ProviderGitHub, ProviderUserID: user, Login: user[2:]})
		require.NoError(t, err)
		require.NoError(t, c.store.WorkspaceMembers.AddMember(t.Context(), &tenancy.Member{UserID: user, WorkspaceID: "workspace-default", RoleID: role.ID, CreatedAt: time.Now()}))
		token, _, err := c.svc.authSvc.MintPAT(t.Context(), user, "hosts test")
		require.NoError(t, err)
		return token
	}
	h.manager = add("u-dana", "machines:write")
	h.deployer = add("u-gail", "machines:deploy")
	h.seer = add("u-erin")
	h.nobody = add("u-frank", "machines:write", "machines:deploy")
	for _, user := range []string{"u-dana", "u-gail", "u-erin"} {
		_, err := c.db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at)
VALUES ('machine', 'm-edge', 'person', ?, '["machines:see"]', '[]', 1, 1)`, user)
		require.NoError(t, err)
	}
	return h
}

// seedHost files a host under machine the way enrollment leaves it.
func seedHost(t *testing.T, db *sql.DB, id, name, machine string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO automation_hosts (id, name, machine, os, arch, version, last_seen, created_at) VALUES (?, ?, ?, 'linux', 'amd64', 'v1', 1, 1)`, id, name, machine)
	require.NoError(t, err)
}

// TestAutomationHosts_NotOnTheMachineIsNotListed: a host follows its machine, so a person who can't see the machine
// doesn't see its hosts, over HTTP and MCP, and a host on a machine Nexul doesn't know is the instance Owner's alone.
// Having automations:read in a workspace, which is all they used to need, lists nothing.
func TestAutomationHosts_NotOnTheMachineIsNotListed(t *testing.T) {
	c := newHostsCast(t)
	seedHost(t, c.db, "h-edge", "jobs-edge", "edge-box")
	seedHost(t, c.db, "h-core", "jobs-core", "core-box")
	seedHost(t, c.db, "h-lost", "jobs-lost", "unknown-box")

	people := []struct {
		name, token string
		sees        []string
	}{
		{"the instance Owner", c.owner, []string{"jobs-edge", "jobs-core", "jobs-lost"}},
		{"a person on edge", c.seer, []string{"jobs-edge"}},
		{"a person on no machine", c.nobody, nil},
	}
	for _, p := range people {
		t.Run(p.name, func(t *testing.T) {
			rec := callAPI(t, c.h, http.MethodGet, "/api/automation-hosts", p.token, "")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			over := callTool(t, c.h, p.token, "machine_list", `{}`)
			for _, host := range []string{"jobs-edge", "jobs-core", "jobs-lost"} {
				want := containsString(p.sees, host)
				assert.Equal(t, want, strings.Contains(rec.Body.String(), host), "GET /api/automation-hosts lists %s", host)
				assert.Equal(t, want, strings.Contains(over, host), "machine_list lists %s", host)
			}
		})
	}
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

// TestAutomationHosts_AddingAndRemovingTakeManageOnTheMachine: a code for a host on an existing machine and removing a
// host both take Manage on its machine; a machine the person can't see answers as no such machine or host, and a host
// on a machine Nexul doesn't know yet, or none named, is the instance Owner's.
func TestAutomationHosts_AddingAndRemovingTakeManageOnTheMachine(t *testing.T) {
	c := newHostsCast(t)
	people := []struct {
		name, user, token                                 string
		onEdge, onNew, onNone, removeOnEdge, removeOnCore int
	}{
		{"the instance Owner", "owner", c.owner, 201, 201, 201, 204, 204},
		{"a Manage holder", "manager", c.manager, 201, 400, 403, 204, 404},
		{"a Deploy to holder", "deployer", c.deployer, 403, 400, 403, 403, 404},
		{"someone who only sees the machine", "seer", c.seer, 403, 400, 403, 403, 404},
		{"someone with no rule on it", "nobody", c.nobody, 400, 400, 403, 404, 404},
	}
	for _, p := range people {
		t.Run(p.name, func(t *testing.T) {
			mint := func(machine string) *httptest.ResponseRecorder {
				return callAPI(t, c.h, http.MethodPost, "/api/automation-hosts/enrollments", p.token, `{"name":"jobs-`+p.user+`-`+strings.TrimSuffix(machine, "-box")+`","machine":"`+machine+`"}`)
			}
			assert.Equal(t, p.onEdge, mint("edge-box").Code, "a code for a host on edge-box")
			assert.Equal(t, p.onNew, mint("new-box").Code, "a code for a host on a machine Nexul doesn't know")
			assert.Equal(t, p.onNone, callAPI(t, c.h, http.MethodPost, "/api/automation-hosts/enrollments", p.token, `{"name":"jobs-`+p.user+`-none"}`).Code, "a code for a host on no machine")

			seedHost(t, c.db, "h-edge-"+p.user, "edge-"+p.user, "edge-box")
			assert.Equal(t, p.removeOnEdge, callAPI(t, c.h, http.MethodDelete, "/api/automation-hosts/h-edge-"+p.user, p.token, "").Code, "removing a host on edge-box")
			seedHost(t, c.db, "h-core-"+p.user, "core-"+p.user, "core-box")
			assert.Equal(t, p.removeOnCore, callAPI(t, c.h, http.MethodDelete, "/api/automation-hosts/h-core-"+p.user, p.token, "").Code, "removing a host on core-box")
		})
	}
}

// TestAutomationHosts_AMachineYouCantSeeIsNeverNamed: asking about a machine nobody has and one the person can't see
// answers the same way, so trying names finds out nothing; over MCP too.
func TestAutomationHosts_AMachineYouCantSeeIsNeverNamed(t *testing.T) {
	c := newHostsCast(t)
	seedHost(t, c.db, "h-core", "jobs-core", "core-box")

	unseen := callAPI(t, c.h, http.MethodPost, "/api/automation-hosts/enrollments", c.manager, `{"name":"jobs-a","machine":"core-box"}`)
	absent := callAPI(t, c.h, http.MethodPost, "/api/automation-hosts/enrollments", c.manager, `{"name":"jobs-a","machine":"nowhere-box"}`)
	assert.Equal(t, absent.Code, unseen.Code)
	assert.Equal(t, strings.ReplaceAll(absent.Body.String(), "nowhere-box", "X"), strings.ReplaceAll(unseen.Body.String(), "core-box", "X"))

	over := callToolResult(t, c.h, c.manager, "host_create", `{"kind":"automations","name":"jobs-b","machine":"core-box"}`)
	assert.Contains(t, over, `"isError":true`)
	assert.Contains(t, over, "does not exist")
	over = callToolResult(t, c.h, c.manager, "host_delete", `{"kind":"automations","id":"h-core"}`)
	assert.Contains(t, over, `"isError":true`)
	assert.Contains(t, over, "not found")
	assert.Contains(t, callTool(t, c.h, c.owner, "machine_list", `{}`), "jobs-core", "and the host is still there")
}

// automationOn creates an automation in the default workspace as the instance Owner and returns its id.
func automationOn(t *testing.T, c hostsCast, name string) string {
	t.Helper()
	rec := callAPI(t, c.h, http.MethodPost, "/api/automations", c.owner, `{"workspace_id":"workspace-default","name":"`+name+`","scopes":["tickets:read"]}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var got struct {
		Automation struct {
			ID string `json:"id"`
		} `json:"automation"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return got.Automation.ID
}

// TestAutomations_SetHostRefusesAMachineYouCantDeployTo: placing an automation on a host takes Deploy to on the host's
// machine beside automations:write in its workspace; a machine the person can't see is no host at all, and the
// bundled instance host stays open. Over HTTP and MCP.
func TestAutomations_SetHostRefusesAMachineYouCantDeployTo(t *testing.T) {
	c := newHostsCast(t)
	seedHost(t, c.db, "h-edge", "jobs-edge", "edge-box")
	seedHost(t, c.db, "h-core", "jobs-core", "core-box")
	id := automationOn(t, c, "nightly")
	reset := func() {
		_, err := c.db.Exec(`UPDATE automations SET host_id = NULL WHERE id = ?`, id)
		require.NoError(t, err)
	}

	people := []struct {
		name, token         string
		edge, core, instant int
	}{
		{"the instance Owner", c.owner, 200, 200, 200},
		{"a Deploy to holder", c.deployer, 200, 404, 200},
		{"a Manage holder", c.manager, 403, 404, 200},
		{"someone who only sees the machine", c.seer, 403, 404, 200},
		{"someone with no rule on it", c.nobody, 404, 404, 200},
	}
	for _, p := range people {
		t.Run(p.name, func(t *testing.T) {
			place := func(host string) int {
				reset()
				return callAPI(t, c.h, http.MethodPatch, "/api/automations/"+id+"/host", p.token, `{"host_id":`+host+`}`).Code
			}
			assert.Equal(t, p.edge, place(`"h-edge"`), "a host on edge-box")
			assert.Equal(t, p.core, place(`"h-core"`), "a host on core-box")
			assert.Equal(t, p.instant, place(`null`), "the instance host")

			reset()
			over := callToolResult(t, c.h, p.token, "automation_update", `{"id":"`+id+`","host_id":"h-edge"}`)
			assert.Equal(t, p.edge != 200, strings.Contains(over, `"isError":true`), "automation_update over MCP: %s", over)
		})
	}

	reset()
	rec := callAPI(t, c.h, http.MethodPatch, "/api/automations/"+id+"/host", c.manager, `{"host_id":"h-edge"}`)
	assert.Contains(t, rec.Body.String(), "machines:deploy", "the refusal names what was missing")
	var host sql.NullString
	require.NoError(t, c.db.QueryRow(`SELECT host_id FROM automations WHERE id = ?`, id).Scan(&host))
	assert.False(t, host.Valid, "a refused move leaves the automation where it was")
}

// enrollHost trades code for a host as its installer would, with no sign-in.
func enrollHost(t *testing.T, c hostsCast, code, name string) *httptest.ResponseRecorder {
	t.Helper()
	return callAPI(t, c.h, http.MethodPost, "/api/automation-hosts/enroll", "", `{"code":"`+code+`","name":"`+name+`","os":"linux","arch":"amd64","version":"v1"}`)
}

func codeOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var got struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return got.Code
}

// TestAutomationHosts_ACodeIsCheckedAgainWhenItIsUsed: a code reads its maker's rights when it is used, so a host code
// made while its maker managed the machine is refused once they don't, and one made by the instance Owner for a
// machine Nexul doesn't know lands wherever the host says it is.
func TestAutomationHosts_ACodeIsCheckedAgainWhenItIsUsed(t *testing.T) {
	c := newHostsCast(t)

	code := codeOf(t, callAPI(t, c.h, http.MethodPost, "/api/automation-hosts/enrollments", c.manager, `{"name":"jobs-late","machine":"edge-box"}`))
	_, err := c.db.Exec(`DELETE FROM permission_overwrites WHERE resource_id = 'm-edge' AND user_id = 'u-dana'`)
	require.NoError(t, err)
	rec := enrollHost(t, c, code, "jobs-late")
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "machine_taken")
	var hosts int
	require.NoError(t, c.db.QueryRow(`SELECT COUNT(*) FROM automation_hosts`).Scan(&hosts))
	assert.Zero(t, hosts, "no host landed on the machine")

	_, err = c.db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at)
VALUES ('machine', 'm-edge', 'person', 'u-dana', '["machines:see","machines:write"]', '[]', 1, 1)`)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, enrollHost(t, c, code, "jobs-late").Code, "once its maker manages the machine again the same code lands")

	code = codeOf(t, callAPI(t, c.h, http.MethodPost, "/api/automation-hosts/enrollments", c.owner, `{"name":"jobs-new"}`))
	assert.Equal(t, http.StatusCreated, enrollHost(t, c, code, "jobs-new").Code, "the instance Owner's code lands on a machine Nexul doesn't know")

	code = codeOf(t, callAPI(t, c.h, http.MethodPost, "/api/automation-hosts/enrollments", c.owner, `{"name":"jobs-unowned"}`))
	_, err = c.db.Exec(`UPDATE member_roles SET role_id = 'role-member' WHERE user_id = 'u-bob'`)
	require.NoError(t, err)
	rec = enrollHost(t, c, code, "jobs-unowned")
	assert.Equal(t, http.StatusForbidden, rec.Code, "a code with no machine is the instance Owner's, and its maker no longer is: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "code_cant_add")
}

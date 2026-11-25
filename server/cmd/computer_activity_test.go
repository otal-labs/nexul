package main

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// activityCast is alice's computer with a runner, a rename and two commands on record, one over MCP.
func activityCast(t *testing.T) (privacyCast, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the commands are POSIX shell")
	}
	c, _ := newPrivacyCast(t)
	srv := httptest.NewServer(c.h)
	t.Cleanup(srv.Close)
	id, enrollToken := addComputer(t, c, c.alice, "Alice laptop")
	startShellRunner(t, c, srv, id, enrollToken, true)
	require.Equal(t, http.StatusOK, callAPI(t, c.h, http.MethodPatch, "/api/pairing/computers/"+id, c.alice, `{"name":"Laptop"}`).Code)
	assert.Contains(t, callToolResult(t, c.h, c.alice, "command_run", `{"computer_id":"`+id+`","command":"echo secret-output"}`), "succeeded")
	require.Equal(t, http.StatusAccepted, callAPI(t, c.h, http.MethodPost, "/api/pairing/computers/"+id+"/commands", c.alice, `{"command":"true"}`).Code)
	return c, id
}

func TestComputerActivity_AuditReadAloneSeesNone(t *testing.T) {
	t.Parallel()
	c, id := activityCast(t)
	ctx := t.Context()
	_, _, err := c.store.Users.UpsertUser(ctx, &auth.Identity{UserID: "u-dave", Provider: auth.ProviderGitHub, ProviderUserID: "u-dave", Login: "dave"})
	require.NoError(t, err)
	require.NoError(t, c.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "u-dave", WorkspaceID: "workspace-default", RoleID: "role-member", CreatedAt: time.Now()}))
	_, err = c.db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at)
VALUES ('instance_area', 'audit', 'person', 'u-dave', '["audit:read"]', '[]', 1, 1)`)
	require.NoError(t, err)
	dave, _, err := c.svc.authSvc.MintPAT(ctx, "u-dave", "activity test")
	require.NoError(t, err)
	_, err = c.db.Exec(`INSERT INTO audit_log (id, actor_type, actor_id, action, created_at) VALUES ('a-plain', 'user', 'u-bob', 'PATCH /api/workspaces/workspace-default', 1)`)
	require.NoError(t, err)

	rec := callAPI(t, c.h, http.MethodGet, "/api/audit?limit=100", dave, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "PATCH /api/workspaces/workspace-default", "the audit log itself still answers")
	assert.NotContains(t, rec.Body.String(), "/api/pairing/computers", "no action on a computer reaches audit:read")
	assert.NotContains(t, rec.Body.String(), "command_run")
	assert.NotContains(t, rec.Body.String(), id)
	assert.Equal(t, http.StatusForbidden, callAPI(t, c.h, http.MethodGet, "/api/audit/computer-activity", dave, "").Code)
}

func TestComputerActivity_OwnerRoleReadsCommandsAndOutput(t *testing.T) {
	t.Parallel()
	c, id := activityCast(t)

	rec := callAPI(t, c.h, http.MethodGet, "/api/audit/computer-activity", c.owner, "")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := rec.Body.String()
	assert.Contains(t, body, `"computer_id":"`+id+`"`)
	assert.Contains(t, body, `"computer_name":"Laptop"`)
	assert.Contains(t, body, `"owner_name":"alice"`)
	assert.Contains(t, body, `"action":"mcp command_run"`)
	assert.Contains(t, body, `"command":"echo secret-output"`)
	assert.Contains(t, body, `"exit_code":0`)
	assert.Contains(t, body, `"output":"secret-output\n"`)
	assert.Contains(t, body, `"action":"PATCH /api/pairing/computers/`+id+`"`, "a rename is on the record too")
	assert.Contains(t, body, `"action":"POST /api/pairing/computers/enrollments"`, "and adding the computer")
}

func TestComputerActivity_DoesNotShowFactsOrComputers(t *testing.T) {
	t.Parallel()
	c, id := activityCast(t)

	rec := callAPI(t, c.h, http.MethodGet, "/api/audit/computer-activity", c.owner, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "facts")
	assert.NotContains(t, rec.Body.String(), "hostname")

	rec = callAPI(t, c.h, http.MethodGet, "/api/pairing/computers", c.owner, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), id, "the permission lists no computer")
	assert.Equal(t, http.StatusNotFound, callAPI(t, c.h, http.MethodGet, "/api/pairing/computers/"+id+"/commands", c.owner, "").Code)
	assert.Contains(t, callToolResult(t, c.h, c.owner, "computer_list", `{"id":"`+id+`"}`), "not one of yours")
}

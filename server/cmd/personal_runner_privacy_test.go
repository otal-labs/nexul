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
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// privacyCast is alice, who owns a computer, and two people who hold every permission there is: bob through the
// workspace's Owner role and carol through a role granted every bit.
type privacyCast struct {
	h                       http.Handler
	store                   *storage.Store
	svc                     *coreServices
	alice, owner, everyBits string
}

func newPrivacyCast(t *testing.T) (privacyCast, *sql.DB) {
	t.Helper()
	db := mentionsTestDB(t)
	h, store, svc := routerOver(t, db)
	ctx := t.Context()
	now := time.Now()
	every := make([]string, 0)
	for _, a := range permissions.AllActions() {
		every = append(every, string(a))
	}
	for _, r := range []*roles.Role{
		{ID: "role-owner", WorkspaceID: "workspace-default", Name: "Owner", IsOwnerRole: true},
		{ID: "role-everything", WorkspaceID: "workspace-default", Name: "Everything", Permissions: grant(every...)},
		{ID: "role-member", WorkspaceID: "workspace-default", Name: "Member"},
	} {
		r.CreatedAt, r.UpdatedAt = now, now
		require.NoError(t, store.Roles.Create(ctx, r))
	}
	tokens := map[string]string{}
	for user, role := range map[string]string{"u-alice": "role-member", "u-bob": "role-owner", "u-carol": "role-everything"} {
		_, _, err := store.Users.UpsertUser(ctx, &auth.Identity{UserID: user, Provider: auth.ProviderGitHub, ProviderUserID: user, Login: user[2:]})
		require.NoError(t, err)
		require.NoError(t, store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: "workspace-default", RoleID: role, CreatedAt: now}))
		tokens[user], _, err = svc.authSvc.MintPAT(ctx, user, "privacy test")
		require.NoError(t, err)
	}
	return privacyCast{h: h, store: store, svc: svc, alice: tokens["u-alice"], owner: tokens["u-bob"], everyBits: tokens["u-carol"]}, db
}

// seedRunners writes a deploy runner on machine m1 and alice's personal runner, the way enrollment leaves them.
func seedRunners(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO machines (id, name, stack_root, reported_hostname, first_seen, last_seen) VALUES ('m-1', 'm1', '/data/nexul', 'm1', 1, 1);
INSERT INTO runners (id, name, last_seen, connected, created_at, version, machine_id) VALUES ('r-deploy', 'edge', 1, 1, 1, 'v1', 'm-1');
INSERT INTO runners (id, name, last_seen, connected, created_at, version, machine_id, owner_user_id, computer_id)
VALUES ('r-laptop', 'computer-ab12cd34', 1, 1, 1, 'v1', '', 'u-alice', 'c-laptop');`)
	require.NoError(t, err)
}

// TestRunners_ListNeverShowsAPersonalRunner: runners:read and machines:read open the deploy runners and machines,
// never a person's computer, even for someone holding every bit or the workspace Owner's bypass.
func TestRunners_ListNeverShowsAPersonalRunner(t *testing.T) {
	c, db := newPrivacyCast(t)
	seedRunners(t, db)

	for who, token := range map[string]string{"workspace Owner": c.owner, "every bit": c.everyBits} {
		rec := callAPI(t, c.h, http.MethodGet, "/api/runners", token, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Body.String(), "r-deploy", who)
		assert.NotContains(t, rec.Body.String(), "r-laptop", "GET /api/runners as %s", who)

		rec = callAPI(t, c.h, http.MethodGet, "/api/machines", token, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.NotContains(t, rec.Body.String(), "computer-", "GET /api/machines as %s", who)

		body := callTool(t, c.h, token, "machine_list", `{}`)
		assert.Contains(t, body, "r-deploy", who)
		assert.NotContains(t, body, "r-laptop", "machine_list as %s", who)
	}
}

// addComputer adds a computer for the caller through the HTTP gateway and returns its id and enrollment token.
func addComputer(t *testing.T, c privacyCast, token, name string) (id, enrollToken string) {
	t.Helper()
	_, err := c.store.Settings.Set(t.Context(), "https://nexul.example.com")
	require.NoError(t, err)
	rec := callAPI(t, c.h, http.MethodPost, "/api/pairing/computers/enrollments", token, `{"name":"`+name+`"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var got struct {
		Computer struct {
			ID string `json:"id"`
		} `json:"computer"`
		Token    string `json:"token"`
		Commands struct {
			Unix string `json:"unix"`
		} `json:"commands"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- "+got.Token, got.Commands.Unix)
	return got.Computer.ID, got.Token
}

// callToolResult sends one tools/call and returns the tool's result text, error or not.
func callToolResult(t *testing.T, h http.Handler, token, tool, args string) string {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + tool + `","arguments":` + args + `,"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28",` +
		`"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},"io.modelcontextprotocol/clientCapabilities":{}}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2026-07-28")
	req.Header.Set("Mcp-Method", "tools/call")
	req.Header.Set("Mcp-Name", tool)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	return rec.Body.String()
}

// TestComputers_AnotherPersonsComputerIsNotFound: alice's computer answers not found, never forbidden, to the
// workspace Owner and to a role holding every bit, over HTTP and MCP, and nothing they try changes it.
func TestComputers_AnotherPersonsComputerIsNotFound(t *testing.T) {
	c, _ := newPrivacyCast(t)
	id, _ := addComputer(t, c, c.alice, "Alice laptop")

	for who, token := range map[string]string{"workspace Owner": c.owner, "every bit": c.everyBits} {
		rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers", token, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.NotContains(t, rec.Body.String(), id, "list as %s", who)
		for _, call := range []struct{ method, path, body string }{
			{http.MethodGet, "/api/pairing/computers/" + id + "/setup", ""},
			{http.MethodPatch, "/api/pairing/computers/" + id, `{"name":"Taken"}`},
			{http.MethodPost, "/api/pairing/computers/enrollments", `{"id":"` + id + `"}`},
		} {
			rec := callAPI(t, c.h, call.method, call.path, token, call.body)
			assert.Equal(t, http.StatusNotFound, rec.Code, "%s %s as %s: %s", call.method, call.path, who, rec.Body.String())
		}
		for tool, args := range map[string]string{
			"computer_list":   `{"id":"` + id + `"}`,
			"computer_pair":   `{"id":"` + id + `","name":"Taken"}`,
			"computer_create": `{"id":"` + id + `"}`,
		} {
			body := callToolResult(t, c.h, token, tool, args)
			assert.Contains(t, body, `"isError":true`, "%s as %s", tool, who)
			assert.Contains(t, body, "not found", "%s as %s: %s", tool, who, body)
		}
		assert.NotContains(t, callToolResult(t, c.h, token, "computer_list", `{}`), id, "computer_list as %s", who)
	}

	rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers", c.alice, "")
	assert.Contains(t, rec.Body.String(), `"name":"Alice laptop"`, "nobody else renamed it")
	body := callTool(t, c.h, c.alice, "computer_pair", `{"id":"`+id+`","name":"Alice desk"}`)
	assert.Contains(t, body, "Alice desk", "its owner renames it over MCP")
	body = callTool(t, c.h, c.alice, "computer_create", `{}`)
	assert.Contains(t, body, "computer.sh", "a member holding no bit adds a computer over MCP")
}

// TestAddComputer_ThroughTheRouter_EnrollsItsRunnerOnNoMachine: the command's code enrolls a personal runner for
// alice's computer, which the row then shows and names after the hostname; no machine appears for it, and a second
// code minted for the same computer enrolls nothing.
func TestAddComputer_ThroughTheRouter_EnrollsItsRunnerOnNoMachine(t *testing.T) {
	c, _ := newPrivacyCast(t)
	id, token := addComputer(t, c, c.alice, "")
	rec := callAPI(t, c.h, http.MethodPost, "/api/pairing/computers/enrollments", c.alice, `{"id":"`+id+`"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var second struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &second))

	rec = callAPI(t, c.h, http.MethodPost, "/api/runners/enroll", "", `{"token":"`+token+`","machine":"alice-laptop","os":"linux","arch":"amd64"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"credential":"nxr_`)
	rec = callAPI(t, c.h, http.MethodPost, "/api/runners/enroll", "", `{"token":"`+second.Token+`","machine":"alice-laptop"}`)
	assert.Equal(t, http.StatusConflict, rec.Code, "a computer has one runner: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "already has its runner")

	assert.Eventually(t, func() bool {
		rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers", c.alice, "")
		return strings.Contains(rec.Body.String(), `"name":"alice-laptop"`) && strings.Contains(rec.Body.String(), `"runner":{"connected":false`)
	}, 5*time.Second, 10*time.Millisecond, "the row shows its runner and the hostname it reported")
	rec = callAPI(t, c.h, http.MethodGet, "/api/machines", c.owner, "")
	assert.NotContains(t, rec.Body.String(), "alice-laptop")
	r, err := c.store.Runners.GetByComputer(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, "u-alice", r.OwnerUserID)
}

// TestLiveAudience_PersonalRunnerChangesReachOnlyTheOwner: a personal runner's frame reaches its owner's sockets
// only, never the workspace Owner's or a role holding every bit, and no integration or automation hears it.
func TestLiveAudience_PersonalRunnerChangesReachOnlyTheOwner(t *testing.T) {
	c, _ := newPrivacyCast(t)
	a := liveAudience{access: c.svc.accessSvc}
	raw, err := json.Marshal(runner.PersonalChangedEvent{RunnerID: "r-laptop", ComputerID: "c-laptop", UserID: "u-alice", State: runner.PersonalConnected, MembersOnly: true})
	require.NoError(t, err)

	for user, want := range map[string]bool{"u-alice": true, "u-bob": false, "u-carol": false} {
		assert.Equal(t, want, a.allows(as(user), runner.TopicPersonalChanged, json.RawMessage(raw)), "as %s", user)
	}
	assert.True(t, eventbus.MembersOnly(raw), "integrations and automations skip it")
}

// TestRunners_RemoveNeverReachesAPersonalRunner: runners:delete removes deploy runners; a person's computer runner
// is not found to it, over HTTP and MCP, and stays enrolled.
func TestRunners_RemoveNeverReachesAPersonalRunner(t *testing.T) {
	c, db := newPrivacyCast(t)
	seedRunners(t, db)

	for who, token := range map[string]string{"workspace Owner": c.owner, "every bit": c.everyBits} {
		rec := callAPI(t, c.h, http.MethodDelete, "/api/runners/r-laptop", token, "")
		assert.Equal(t, http.StatusNotFound, rec.Code, "DELETE as %s: %s", who, rec.Body.String())
		body := callToolResult(t, c.h, token, "host_delete", `{"kind":"runner","id":"r-laptop"}`)
		assert.Contains(t, body, `"isError":true`, "host_delete as %s", who)
		assert.Contains(t, body, "not found", "host_delete as %s", who)
	}
	_, err := c.store.Runners.GetByComputer(t.Context(), "c-laptop")
	require.NoError(t, err, "alice's runner is still enrolled")
}

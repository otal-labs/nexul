package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// sharingCast adds dave, a member with no bit, and erin, who shares no workspace with alice, the laptop's owner.
type sharingCast struct {
	privacyCast
	db         *sql.DB
	dave, erin string
	laptop     string
}

func newSharingCast(t *testing.T) sharingCast {
	t.Helper()
	c, db := newPrivacyCast(t)
	ctx := t.Context()
	now := time.Now()
	tokens := map[string]string{}
	for _, user := range []string{"u-dave", "u-erin"} {
		_, _, err := c.store.Users.UpsertUser(ctx, &auth.Identity{UserID: user, Provider: auth.ProviderGitHub, ProviderUserID: user, Login: user[2:]})
		require.NoError(t, err)
		tokens[user], _, err = c.svc.authSvc.MintPAT(ctx, user, "sharing test")
		require.NoError(t, err)
	}
	require.NoError(t, c.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "u-dave", WorkspaceID: "workspace-default", RoleID: "role-member", CreatedAt: now}))
	laptop, _ := addComputer(t, c, c.alice, "Alice laptop")
	return sharingCast{privacyCast: c, db: db, dave: tokens["u-dave"], erin: tokens["u-erin"], laptop: laptop}
}

// share writes alice's rule on her laptop through the HTTP gateway, as token, and returns the response.
func (c sharingCast) share(t *testing.T, token, body string) (int, string) {
	t.Helper()
	rec := callAPI(t, c.h, http.MethodPut, "/api/permissions", token, `{"resource_type":"computer","resource_ids":["`+c.laptop+`"],`+body+`}`)
	return rec.Code, rec.Body.String()
}

// held is what userID holds on the laptop now, read the way every computer check reads it.
func (c sharingCast) held(t *testing.T, userID string) (permissions.Set, error) {
	t.Helper()
	_, held, err := c.svc.accessSvc.ComputerPermissions(as(userID), userID, c.laptop)
	return held, err
}

// Carol's role holds every catalog bit, so computer_activity:read too once it is built.
func TestComputerRules_OwnerOrEveryBitRoleWithoutARuleGetsNotFound(t *testing.T) {
	c := newSharingCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:run_commands"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)

	for who, token := range map[string]string{"workspace Owner": c.owner, "every bit": c.everyBits} {
		for _, call := range []struct{ method, path, body string }{
			{http.MethodGet, "/api/permissions?resource_type=computer&resource_id=" + c.laptop, ""},
			{http.MethodGet, "/api/permissions/explain?resource_type=computer&resource_id=" + c.laptop, ""},
			{http.MethodGet, "/api/permissions/explain?resource_type=computer&resource_id=" + c.laptop + "&user_id=u-dave", ""},
			{http.MethodPut, "/api/permissions", `{"resource_type":"computer","resource_ids":["` + c.laptop + `"],"user_ids":["u-dave"],"actions":["computers:see"],"state":"fallback"}`},
		} {
			rec := callAPI(t, c.h, call.method, call.path, token, call.body)
			assert.Equal(t, http.StatusNotFound, rec.Code, "%s %s as %s: %s", call.method, call.path, who, rec.Body.String())
		}
		rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers", token, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.NotContains(t, rec.Body.String(), c.laptop, "GET /api/pairing/computers as %s", who)
		assert.NotContains(t, callToolResult(t, c.h, token, "computer_list", `{}`), c.laptop, "computer_list as %s", who)
		for tool, args := range map[string]string{
			"computer_list":               `{"id":"` + c.laptop + `"}`,
			"permission_overwrite_list":   `{"resource_type":"computer","resource_id":"` + c.laptop + `"}`,
			"permission_overwrite_update": `{"resource_type":"computer","resource_ids":["` + c.laptop + `"],"user_ids":["u-bob"],"actions":["computers:see"],"state":"allow"}`,
		} {
			body := callToolResult(t, c.h, token, tool, args)
			assert.Contains(t, body, `"isError":true`, "%s as %s", tool, who)
			assert.Contains(t, body, "not found", "%s as %s: %s", tool, who, body)
		}
	}
	for _, user := range []string{"u-bob", "u-carol"} {
		_, err := c.held(t, user)
		require.ErrorIs(t, err, apperrs.ErrNotFound, user)
	}

	frame := outboxPayload(t, c, access.TopicComputerGrantChanged)
	audience := liveAudience{access: c.svc.accessSvc}
	for user, want := range map[string]bool{"u-alice": true, "u-dave": true, "u-bob": false, "u-carol": false, "u-erin": false} {
		assert.Equal(t, want, audience.allows(as(user), access.TopicComputerGrantChanged, frame), "the share's frame as %s", user)
	}
	assert.True(t, eventbus.MembersOnly(frame), "no integration or automation hears it")
	assert.NotContains(t, string(frame), "Alice laptop", "an event never names the computer")
}

func TestComputerRules_ANamedPersonHoldsExactlyWhatIsGranted(t *testing.T) {
	c := newSharingCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave","u-bob"],"actions":["computers:run_commands"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)

	for _, user := range []string{"u-dave", "u-bob"} {
		held, err := c.held(t, user)
		require.NoError(t, err, user)
		assert.Equal(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersCommands), held, "%s holds See and Run commands, not Run agents", user)
	}
	held, err := c.held(t, "u-alice")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.ComputerActions...), held, "the owner holds everything")

	rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers", c.dave, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var listed struct {
		Computers []map[string]any `json:"computers"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed.Computers, 1)
	shared := listed.Computers[0]
	assert.Equal(t, "Alice laptop", shared["name"])
	assert.Equal(t, map[string]any{"owner_id": "u-alice", "owner_name": "alice", "permissions": []any{"computers:run_commands", "computers:see"}}, shared["shared"])
	for _, private := range []string{"server_url", "harness_version", "kind", "facts", "tunnel", "pair_error"} {
		assert.Empty(t, shared[private], "a shared computer never shows its %s", private)
	}
	assert.Contains(t, callTool(t, c.h, c.dave, "computer_list", `{}`), `alice's Alice laptop (shared)`)

	rec = callAPI(t, c.h, http.MethodGet, "/api/permissions/explain?resource_type=computer&resource_id="+c.laptop, c.dave, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "Shared with dave by alice")
	assert.Contains(t, rec.Body.String(), "Not shared with dave")
	rec = callAPI(t, c.h, http.MethodGet, "/api/permissions/explain?resource_type=computer&resource_id="+c.laptop+"&user_id=u-bob", c.dave, "")
	assert.Equal(t, http.StatusNotFound, rec.Code, "a grantee explains only themselves")
	rec = callAPI(t, c.h, http.MethodGet, "/api/permissions/explain?resource_type=computer&resource_id="+c.laptop+"&user_id=u-carol", c.alice, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "Not shared with carol", "the owner explains anyone")

	for _, path := range []string{"/api/pairing/computers/" + c.laptop + "/setup", "/api/pairing/computers/" + c.laptop + "/mcp-token"} {
		rec := callAPI(t, c.h, http.MethodGet, path, c.dave, "")
		assert.Equal(t, http.StatusNotFound, rec.Code, "GET %s as the grantee", path)
	}
}

func TestComputerRules_OnlyTheOwnerWrites(t *testing.T) {
	c := newSharingCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:see"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)

	for who, token := range map[string]string{"grantee": c.dave, "workspace Owner": c.owner, "every bit": c.everyBits} {
		for _, write := range []string{
			`"user_ids":["u-carol"],"actions":["computers:see"],"state":"allow"`,
			`"user_ids":["u-dave"],"actions":["computers:run_agents"],"state":"allow"`,
			`"user_ids":["u-dave"],"actions":["computers:see"],"state":"fallback"`,
		} {
			code, body := c.share(t, token, write)
			assert.Equal(t, http.StatusNotFound, code, "%s writing %s: %s", who, write, body)
		}
		result := callToolResult(t, c.h, token, "permission_overwrite_update",
			`{"resource_type":"computer","resource_ids":["`+c.laptop+`"],"user_ids":["u-dave"],"actions":["computers:run_agents"],"state":"allow"}`)
		assert.Contains(t, result, "not found", "%s over MCP", who)
	}
	held, err := c.held(t, "u-dave")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.ComputersSee), held, "nothing they tried changed the rule")

	callTool(t, c.h, c.alice, "permission_overwrite_update",
		`{"resource_type":"computer","resource_ids":["`+c.laptop+`"],"user_ids":["u-dave"],"actions":["computers:run_agents"],"state":"allow"}`)
	held, err = c.held(t, "u-dave")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersAgents), held, "the owner raised it over MCP")
	assert.Contains(t, callTool(t, c.h, c.alice, "permission_overwrite_list", `{"resource_type":"computer","resource_id":"`+c.laptop+`"}`), "u-dave")
}

func TestComputerRules_TargetsAreNamedPeopleOnly(t *testing.T) {
	c := newSharingCast(t)
	for name, write := range map[string]string{
		"a role":              `"role_ids":["role-member"],"actions":["computers:see"],"state":"allow"`,
		"a deny":              `"user_ids":["u-dave"],"actions":["computers:see"],"state":"deny"`,
		"the owner":           `"user_ids":["u-alice"],"actions":["computers:see"],"state":"allow"`,
		"no shared workspace": `"user_ids":["u-erin"],"actions":["computers:see"],"state":"allow"`,
		"a grid permission":   `"user_ids":["u-dave"],"actions":["docs:read"],"state":"allow"`,
	} {
		code, body := c.share(t, c.alice, write)
		assert.Equal(t, http.StatusBadRequest, code, "%s: %s", name, body)
	}
	result := callToolResult(t, c.h, c.alice, "permission_overwrite_update",
		`{"resource_type":"computer","resource_ids":["`+c.laptop+`"],"role_ids":["role-member"],"actions":["computers:see"],"state":"allow"}`)
	assert.Contains(t, result, "never a role")
	rec := callAPI(t, c.h, http.MethodPut, "/api/permissions", c.owner, `{"resource_type":"project","resource_ids":["project-general"],"user_ids":["u-dave"],"actions":["computers:see"],"state":"allow"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "a project rule never sets a computer permission: %s", rec.Body.String())

	rec = callAPI(t, c.h, http.MethodGet, "/api/permissions?resource_type=computer&resource_id="+c.laptop, c.alice, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"grants":[]`, "nothing refused was stored")
}

// The second check runs under the same memo as the first, so only a fresh read sees the revoke.
func TestComputerRules_RevokeIsImmediate(t *testing.T) {
	c := newSharingCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:run_agents"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)
	ctx := as("u-dave")
	_, held, err := c.svc.accessSvc.ComputerPermissions(ctx, "u-dave", c.laptop)
	require.NoError(t, err)
	assert.True(t, held.Has(permissions.ComputersAgents))

	code, body = c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:see"],"state":"fallback"`)
	require.Equal(t, http.StatusNoContent, code, body)
	_, _, err = c.svc.accessSvc.ComputerPermissions(ctx, "u-dave", c.laptop)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "taking See off takes both Runs with it, at once")
	rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers", c.dave, "")
	assert.NotContains(t, rec.Body.String(), c.laptop)
}

func TestComputerRules_EndWhenNoWorkspaceIsSharedOrAnAccountIsDisabled(t *testing.T) {
	c := newSharingCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave","u-carol"],"actions":["computers:see"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)

	rec := callAPI(t, c.h, http.MethodDelete, "/api/workspaces/workspace-default/members/u-dave", c.owner, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	_, err := c.held(t, "u-dave")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "no shared workspace, no computer, before any consumer runs")
	assert.Eventually(t, func() bool { return computerRules(t, c) == 1 }, 5*time.Second, 10*time.Millisecond, "dave's rule is deleted")
	require.NoError(t, c.store.WorkspaceMembers.AddMember(t.Context(), &tenancy.Member{UserID: "u-dave", WorkspaceID: "workspace-default", RoleID: "role-member", CreatedAt: time.Now()}))
	_, err = c.held(t, "u-dave")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "rejoining brings no rule back")

	rec = callAPI(t, c.h, http.MethodPost, "/api/auth/accounts/u-carol/disable", c.owner, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	assert.Eventually(t, func() bool { return computerRules(t, c) == 0 }, 5*time.Second, 10*time.Millisecond, "carol's rule is deleted")
	rec = callAPI(t, c.h, http.MethodPost, "/api/auth/accounts/u-carol/reactivate", c.owner, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	_, err = c.held(t, "u-carol")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "reactivating brings no rule back")

	code, body = c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:see"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)
	rec = callAPI(t, c.h, http.MethodPost, "/api/auth/accounts/u-alice/disable", c.owner, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	_, err = c.held(t, "u-dave")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "an owner's disabled account shares nothing")
	assert.Eventually(t, func() bool { return computerRules(t, c) == 0 }, 5*time.Second, 10*time.Millisecond, "and the rule is deleted")

	rec = callAPI(t, c.h, http.MethodPost, "/api/auth/accounts/u-alice/reactivate", c.owner, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	code, body = c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:see"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)
	rec = callAPI(t, c.h, http.MethodDelete, "/api/pairing/computers/"+c.laptop, c.alice, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Zero(t, computerRules(t, c), "removing the computer deletes its rules")
}

func TestComputerRules_EventsReachOnlyTheOwnerAndTheGrantee(t *testing.T) {
	c := newSharingCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:run_commands"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)
	rec := callAPI(t, c.h, http.MethodDelete, "/api/workspaces/workspace-default/members/u-dave", c.owner, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	assert.Eventually(t, func() bool { return computerRules(t, c) == 0 }, 5*time.Second, 10*time.Millisecond)

	rows, err := c.db.QueryContext(t.Context(), `SELECT payload FROM outbox WHERE topic = ? ORDER BY created_at, rowid`, access.TopicComputerGrantChanged)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })
	var frames []access.ComputerGrantEvent
	audience := liveAudience{access: c.svc.accessSvc}
	for rows.Next() {
		var raw []byte
		require.NoError(t, rows.Scan(&raw))
		var e access.ComputerGrantEvent
		require.NoError(t, json.Unmarshal(raw, &e))
		frames = append(frames, e)
		assert.True(t, eventbus.MembersOnly(raw))
		assert.NotContains(t, string(raw), "Alice laptop")
		for user, want := range map[string]bool{"u-alice": true, "u-dave": true, "u-bob": false, "u-carol": false} {
			assert.Equal(t, want, audience.allows(as(user), access.TopicComputerGrantChanged, json.RawMessage(raw)), "as %s", user)
		}
	}
	require.NoError(t, rows.Err())
	require.Len(t, frames, 2)
	assert.Equal(t, access.ComputerGrantEvent{ComputerID: c.laptop, OwnerID: "u-alice", UserID: "u-dave", Permissions: []string{"computers:run_commands", "computers:see"}, ActorID: "u-alice", MembersOnly: true}, frames[0])
	assert.Equal(t, access.ComputerGrantEvent{ComputerID: c.laptop, OwnerID: "u-alice", UserID: "u-dave", Permissions: []string{}, MembersOnly: true}, frames[1], "the rule's end")
}

func computerRules(t *testing.T, c sharingCast) int {
	t.Helper()
	var n int
	require.NoError(t, c.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM permission_overwrites WHERE resource_type = 'computer'`).Scan(&n))
	return n
}

// outboxPayload is the newest outbox payload on topic.
func outboxPayload(t *testing.T, c sharingCast, topic string) json.RawMessage {
	t.Helper()
	var raw string
	require.NoError(t, c.db.QueryRowContext(t.Context(), `SELECT CAST(payload AS TEXT) FROM outbox WHERE topic = ? ORDER BY created_at DESC, rowid DESC LIMIT 1`, topic).Scan(&raw))
	require.True(t, strings.HasPrefix(raw, "{"))
	return json.RawMessage(raw)
}

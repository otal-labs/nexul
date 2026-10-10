package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TestComputerFacts_NeverReachAnotherUser: alice reads her computer's facts over HTTP and MCP; the workspace Owner, who
// holds every instance permission, and a role holding every bit read none of them on any path, and the live frame
// that announces a change reaches only alice and carries no fact. Holders of Read computer activity join this test
// with the permission (ticket 15).
func TestComputerFacts_NeverReachAnotherUser(t *testing.T) {
	c, db := newPrivacyCast(t)
	id, _ := addComputer(t, c, c.alice, "Alice laptop")
	report, err := json.Marshal(map[string]any{
		"runner_id": "r-laptop", "computer_id": id, "user_id": "u-alice", "members_only": true,
		"facts": map[string]any{"hostname": "alice-laptop", "git_email": "alice@example.com", "t3": map[string]any{"state": "not_running"}},
	})
	require.NoError(t, err)
	require.NoError(t, c.svc.pairingSvc.HandleFactsReported(t.Context(), eventbus.Event{Topic: "runner.facts_reported", Payload: report}))

	rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers", c.alice, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"git_email":"alice@example.com"`, "alice reads her facts over HTTP")
	assert.Contains(t, callTool(t, c.h, c.alice, "computer_list", `{"id":"`+id+`"}`), "alice@example.com", "and over MCP")

	for who, token := range map[string]string{"workspace Owner": c.owner, "every bit": c.everyBits} {
		rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers", token, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.NotContains(t, rec.Body.String(), "alice-laptop", "GET /api/pairing/computers as %s", who)
		assert.NotContains(t, callToolResult(t, c.h, token, "computer_list", `{}`), "alice-laptop", "computer_list as %s", who)
		body := callToolResult(t, c.h, token, "computer_list", `{"id":"`+id+`"}`)
		assert.Contains(t, body, "not found", "computer_list with alice's id as %s", who)
		assert.NotContains(t, body, "alice-laptop", "computer_list with alice's id as %s", who)
	}

	var payload string
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT payload FROM outbox WHERE topic = ?`, pairing.TopicFactsChanged).Scan(&payload))
	assert.NotContains(t, payload, "alice@example.com", "the event names the computer and carries no fact")
	assert.True(t, eventbus.MembersOnly(json.RawMessage(payload)), "integrations and automations skip it")
	a := liveAudience{access: c.svc.accessSvc}
	for user, want := range map[string]bool{"u-alice": true, "u-bob": false, "u-carol": false} {
		assert.Equal(t, want, a.allows(as(user), pairing.TopicFactsChanged, json.RawMessage(payload)), "live frame as %s", user)
	}
}

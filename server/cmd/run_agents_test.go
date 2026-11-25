package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// guestClient is a harness client whose computer gives each run an instance of its own, recording the token it got.
type guestClient struct {
	harness.Client
	token  string
	closed []string
}

func (g *guestClient) OpenGuest(_ context.Context, _ harness.Session, base, _, token, _ string) (string, error) {
	g.token = token
	return "nexul-guest-" + base, nil
}

func (g *guestClient) CloseGuest(_ context.Context, _ harness.Session, id string) error {
	g.closed = append(g.closed, id)
	return nil
}

func (c sharingCast) openGuestRun(t *testing.T, userID string, client *guestClient) (*harness.Target, func(context.Context), error) {
	t.Helper()
	_, err := c.store.Settings.Set(t.Context(), "https://nexul.example.com")
	require.NoError(t, err)
	computer, err := c.store.Pairing.GetComputer(t.Context(), "u-alice", c.laptop)
	require.NoError(t, err)
	target := &harness.Target{Provider: "claudeAgent"}
	closeRun, err := c.svc.pairingSvc.OpenGuestRun(as(userID), userID, *computer, client, target)
	return target, closeRun, err
}

func TestRunAgents_RunTokenIsTheStartersAndExpires(t *testing.T) {
	t.Parallel()
	c := newSharingCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:run_agents"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)
	client := &guestClient{}

	target, closeRun, err := c.openGuestRun(t, "u-dave", client)
	require.NoError(t, err)

	assert.Equal(t, "nexul-guest-claudeAgent", target.Provider, "the turn runs on the run's own instance")
	assert.Contains(t, callToolResult(t, c.h, client.token, "account_get", `{}`), `\"login\":\"dave\"`, "the agent acts as the starter")

	closeRun(t.Context())
	assert.Equal(t, []string{"nexul-guest-claudeAgent"}, client.closed)
	rec := callAPI(t, c.h, http.MethodGet, "/api/auth/me", client.token, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "the token stops working when the run ends")
}

func TestRunAgents_NeedsThePermission(t *testing.T) {
	t.Parallel()
	c := newSharingCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:see","computers:run_commands"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)

	_, _, err := c.openGuestRun(t, "u-dave", &guestClient{})
	require.ErrorIs(t, err, apperrs.ErrNotFound, "See and Run commands without Run agents start no agent")
	_, _, err = c.openGuestRun(t, "u-bob", &guestClient{})
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a workspace Owner no rule names")
}

func TestRunAgents_AnOwnersOwnRunIsLeftAsItIs(t *testing.T) {
	t.Parallel()
	c := newSharingCast(t)
	client := &guestClient{}

	target, closeRun, err := c.openGuestRun(t, "u-alice", client)

	require.NoError(t, err)
	closeRun(t.Context())
	assert.Equal(t, "claudeAgent", target.Provider)
	assert.Empty(t, client.token, "no token is minted for the owner")
}

func TestRunAgents_DoesNotShowTheOwnerTheTranscript(t *testing.T) {
	t.Parallel()
	c := newSharingCast(t)
	_, err := c.db.Exec(`INSERT INTO play_trails (id, workspace_id, play_id, play_label, target_type, target_id, project_id, conversation_id, starter_id, via, harness_session_id, state, started_at, activity, computer_id)
VALUES ('tr-dave', 'workspace-default', 'p1', 'Implement with AI', 'ticket', 'tk1', '', 'conv-1', 'u-dave', 'button', 'th1', 'done', 1, '[{"summary":"read the private notes file"}]', ?),
       ('tr-alice', 'workspace-default', 'p1', 'Implement with AI', 'ticket', 'tk2', '', 'conv-2', 'u-alice', 'button', 'th2', 'done', 2, '[]', ?)`, c.laptop, c.laptop)
	require.NoError(t, err)

	rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers/"+c.laptop+"/shared-runs", c.alice, "")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"trail_id":"tr-dave"`)
	assert.Contains(t, rec.Body.String(), `"starter_name":"dave"`)
	assert.NotContains(t, rec.Body.String(), "tr-alice", "the owner's own runs are not in it")
	assert.NotContains(t, rec.Body.String(), "private notes", "never what the run did")
	assert.Equal(t, http.StatusNotFound, callAPI(t, c.h, http.MethodGet, "/api/pairing/computers/"+c.laptop+"/shared-runs", c.dave, "").Code)
}

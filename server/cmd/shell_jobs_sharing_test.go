package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sharedShellCast is alice's laptop with its runner up and commands on, for dave to be shared.
func sharedShellCast(t *testing.T) sharingCast {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the commands are POSIX shell")
	}
	c := newSharingCast(t)
	srv := httptest.NewServer(c.h)
	t.Cleanup(srv.Close)
	startShellRunner(t, c.privacyCast, srv, c.laptop, c.laptopEnroll, true)
	return c
}

func (c sharingCast) runCommand(t *testing.T, token, command string) *httptest.ResponseRecorder {
	t.Helper()
	return callAPI(t, c.h, http.MethodPost, "/api/pairing/computers/"+c.laptop+"/commands", token, `{"command":"`+command+`"}`)
}

func TestShellJobs_GranteeNeedsRunCommands(t *testing.T) {
	t.Parallel()
	c := sharedShellCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:see"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)
	assert.Equal(t, http.StatusNotFound, c.runCommand(t, c.dave, "id").Code, "See alone runs nothing")

	code, body = c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:run_commands"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)
	got := callToolResult(t, c.h, c.dave, "command_run", `{"computer_id":"`+c.laptop+`","command":"echo as-dave"}`)
	assert.Contains(t, got, `\"status\":\"succeeded\"`)
	assert.Contains(t, got, "as-dave")

	rec := callAPI(t, c.h, http.MethodPatch, "/api/pairing/computers/"+c.laptop, c.alice, `{"shell_enabled":false}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, http.StatusConflict, c.runCommand(t, c.dave, "id").Code, "the owner's switch off refuses the grantee too")
	assert.Equal(t, http.StatusNotFound, callAPI(t, c.h, http.MethodPatch, "/api/pairing/computers/"+c.laptop, c.dave, `{"shell_enabled":true}`).Code,
		"only the owner holds the switch")
}

func TestShellJobs_FollowTheGranteesIdentity(t *testing.T) {
	t.Parallel()
	c := sharedShellCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:run_commands"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)

	rec := c.runCommand(t, c.dave, "true")
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	var job struct {
		RequesterID string `json:"requester_id"`
		OwnerID     string `json:"owner_id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &job))
	assert.Equal(t, "u-dave", job.RequesterID, "the job is dave's, not alice's")
	assert.Equal(t, "u-alice", job.OwnerID)

	for who, token := range map[string]string{"a third person": c.erin, "a workspace Owner no rule names": c.owner} {
		assert.Equal(t, http.StatusNotFound, c.runCommand(t, token, "id").Code, who)
		assert.Contains(t, callToolResult(t, c.h, token, "command_run", `{"computer_id":"`+c.laptop+`","command":"id"}`), "not found", who)
	}
}

func TestShellJobs_RevokeDropsQueuedAndCancelsRunningJobs(t *testing.T) {
	t.Parallel()
	c := sharedShellCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:run_commands"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)
	var ids []string
	for range 5 {
		rec := c.runCommand(t, c.dave, "sleep 60")
		require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
		var job struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &job))
		ids = append(ids, job.ID)
	}
	alice := c.runCommand(t, c.alice, "sleep 60")
	require.Equal(t, http.StatusAccepted, alice.Code)

	code, body = c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:see"],"state":"fallback"`)
	require.Equal(t, http.StatusNoContent, code, body)

	start := time.Now()
	for _, id := range ids {
		require.Eventually(t, func() bool {
			rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/commands/"+id, c.alice, "")
			var job struct {
				Status string `json:"status"`
				Error  string `json:"error"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &job)
			return job.Status == "cancelled" && job.Error == "access revoked"
		}, 10*time.Second, 50*time.Millisecond, id)
	}
	assert.Less(t, time.Since(start), 10*time.Second, "no restart or cache wait")
	assert.Equal(t, http.StatusNotFound, c.runCommand(t, c.dave, "id").Code, "the next start is refused")
	rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers/"+c.laptop+"/commands", c.alice, "")
	assert.Contains(t, rec.Body.String(), `"status":"running"`, "alice's own command keeps running")
}

func TestGrantee_SeesNoFactsOrOtherPeoplesJobs(t *testing.T) {
	t.Parallel()
	c := sharedShellCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:run_commands"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)
	require.Equal(t, http.StatusAccepted, c.runCommand(t, c.alice, "echo alices-secret").Code)
	require.Equal(t, http.StatusAccepted, c.runCommand(t, c.dave, "echo daves-own").Code)

	rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers/"+c.laptop+"/commands", c.dave, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"total":1`)
	assert.Contains(t, rec.Body.String(), "daves-own")
	assert.NotContains(t, rec.Body.String(), "alices-secret")
	listed := callToolResult(t, c.h, c.dave, "computer_list", `{"id":"`+c.laptop+`","commands":{}}`)
	assert.NotContains(t, listed, "alices-secret")
	assert.NotContains(t, listed, "facts")

	rec = callAPI(t, c.h, http.MethodGet, "/api/pairing/computers/"+c.laptop+"/commands", c.alice, "")
	assert.Contains(t, rec.Body.String(), `"total":2`, "the owner's run log lists what grantees ran")
	assert.Contains(t, rec.Body.String(), `"requester_name":"dave"`)
	assert.NotContains(t, rec.Body.String(), "transcript")
}

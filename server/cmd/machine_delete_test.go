package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/runner"
)

func machineCount(t *testing.T, c privacyCast, id string) int {
	t.Helper()
	var n int
	require.NoError(t, c.db.QueryRow(`SELECT COUNT(*) FROM machines WHERE id = ?`, id).Scan(&n))
	return n
}

// TestMachineDelete_RefusalsLeaveTheMachine: Delete is its own action on the machine. Someone who can't see it gets not
// found, someone who sees it and manages it but is denied Delete gets forbidden, and a stack aimed at it names itself in
// a conflict, over HTTP and over host_delete. The machine is still there after each.
func TestMachineDelete_RefusalsLeaveTheMachine(t *testing.T) {
	c, db := newPrivacyCast(t)
	seedEdgeAndCore(t, db)

	rec := callAPI(t, c.h, http.MethodDelete, "/api/machines/m-edge", c.alice, "")
	assert.Equal(t, http.StatusNotFound, rec.Code, "a machine alice can't see: %s", rec.Body.String())

	_, err := db.Exec(`UPDATE permission_overwrites SET deny = '["machines:delete"]' WHERE resource_id = 'm-edge' AND user_id = 'u-carol'`)
	require.NoError(t, err)
	rec = callAPI(t, c.h, http.MethodDelete, "/api/machines/m-edge", c.everyBits, "")
	assert.Equal(t, http.StatusForbidden, rec.Code, "carol holds Manage but is denied Delete: %s", rec.Body.String())
	assert.Contains(t, callToolResult(t, c.h, c.everyBits, "host_delete", `{"kind":"machine","id":"m-edge"}`), `"isError":true`)

	_, err = db.Exec(`UPDATE permission_overwrites SET deny = '[]' WHERE resource_id = 'm-edge' AND user_id = 'u-carol';
INSERT INTO stacks (id, name, slug, machine, strategy, created_at, updated_at) VALUES ('s-api', 'api', 'api', 'edge-box', 'run', 1, 1)`)
	require.NoError(t, err)
	rec = callAPI(t, c.h, http.MethodDelete, "/api/machines/m-edge", c.everyBits, "")
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `machine \"edge-box\" is still used by stack api`)
	body := callToolResult(t, c.h, c.everyBits, "host_delete", `{"kind":"machine","id":"m-edge"}`)
	assert.Contains(t, body, `"isError":true`)
	assert.Contains(t, body, "stack api")

	assert.Equal(t, 1, machineCount(t, c, "m-edge"))
	assert.Equal(t, 1, machineCount(t, c, "m-core"), "the other machine was never touched")
}

// TestMachineDelete_FreeMachineGoesForEveryoneWhoSawIt: with nothing aimed at it, the person holding Delete removes the
// machine over HTTP, the instance Owner another over host_delete; the machine's rules go with it, and the one
// machine.deleted frame reaches the people who saw it (the instance Owner and the person its rule named) and nobody else.
func TestMachineDelete_FreeMachineGoesForEveryoneWhoSawIt(t *testing.T) {
	c, db := newPrivacyCast(t)
	seedEdgeAndCore(t, db)

	rec := callAPI(t, c.h, http.MethodDelete, "/api/machines/m-edge", c.everyBits, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())

	assert.Zero(t, machineCount(t, c, "m-edge"))
	var rules, runners int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM permission_overwrites WHERE resource_type = 'machine' AND resource_id = 'm-edge'`).Scan(&rules))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM runners WHERE machine_id = 'm-edge'`).Scan(&runners))
	assert.Zero(t, rules, "its rules went with it")
	assert.Zero(t, runners, "its offline runner went with it")
	assert.NotContains(t, callAPI(t, c.h, http.MethodGet, "/api/machines", c.owner, "").Body.String(), "edge-box")

	var payload string
	require.NoError(t, db.QueryRow(`SELECT payload FROM outbox WHERE topic = ?`, runner.TopicMachineDeleted).Scan(&payload))
	var evt runner.MachineDeletedEvent
	require.NoError(t, json.Unmarshal([]byte(payload), &evt))
	assert.Equal(t, "edge-box", evt.Name)
	a := liveAudience{access: c.svc.accessSvc}
	for user, sees := range map[string]bool{"u-bob": true, "u-carol": true, "u-alice": false} {
		assert.Equal(t, sees, a.allows(as(user), runner.TopicMachineDeleted, json.RawMessage(payload)), user)
	}

	body := callTool(t, c.h, c.owner, "host_delete", `{"kind":"machine","id":"m-core"}`)
	assert.Contains(t, body, `deleted`)
	assert.Zero(t, machineCount(t, c, "m-core"))
}
